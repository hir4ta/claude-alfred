package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/hir4ta/claude-buddy/internal/embedder"
	"github.com/hir4ta/claude-buddy/internal/hookhandler"
	"github.com/hir4ta/claude-buddy/internal/sessiondb"
	"github.com/hir4ta/claude-buddy/internal/store"
)

// withBuddyTracker wraps a handler to reset the silence-as-signal counter
// whenever a buddy MCP tool is invoked. This signals to the Thompson Sampling
// system that the user is actively engaging with buddy suggestions.
func withBuddyTracker(st *store.Store, h server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resetBuddyTracker(st)
		return h(ctx, req)
	}
}

func resetBuddyTracker(st *store.Store) {
	if st == nil {
		return
	}
	sid := latestSessionID(st)
	if sid == "" || sid == "unknown" {
		return
	}
	sdb, err := sessiondb.Open(sid)
	if err != nil {
		return
	}
	defer sdb.Close()
	hookhandler.ResetBuddyCallTracker(sdb)
}

const serverInstructions = `claude-buddy is a real-time session advisor for Claude Code.
Hook-based briefings are delivered automatically every turn.

## When you need more detail from a briefing:
  buddy_knowledge — Search past patterns, decisions, and solutions
  buddy_guidance — Get alerts, recommendations, next steps
  buddy_diagnose — Root cause analysis + fix patches for errors

## Session management:
  buddy_state — Session health, statistics, and predictions
  buddy_resume — Restore context from a previous session
  buddy_recall — Search pre-compact conversation details

## Planning & feedback:
  buddy_plan — Task estimation, progress tracking, strategic plans
  buddy_feedback — Rate suggestion quality (improves future relevance)
  buddy_skill_context — Aggregated context for skills
`

// New creates a new MCP server with all tools registered.
// emb may be nil if VOYAGE_API_KEY is not set.
func New(claudeHome string, st *store.Store, emb *embedder.Embedder) *server.MCPServer {
	s := server.NewMCPServer(
		"claude-buddy",
		"0.3.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithInstructions(serverInstructions),
		server.WithLogging(),
	)

	s.AddTools(
		// 1. buddy_state: Consolidated session state (stats + current_state + session_outlook).
		server.ServerTool{
			Tool: mcp.NewTool("buddy_state",
				mcp.WithDescription("Get session health, statistics, burst state, and trend predictions. Use when you need to assess current session progress, check health metrics, or predict potential issues. The 'detail' parameter controls depth: 'brief' returns stats only, 'standard' (default) returns a full snapshot including burst state and phase, 'outlook' adds health trends and risk predictions."),
				mcp.WithTitleAnnotation("Session State"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithIdempotentHintAnnotation(true),
				mcp.WithOpenWorldHintAnnotation(false),
				mcp.WithString("detail",
					mcp.Description("Level of detail: brief (stats), standard (default, full snapshot), outlook (health + phase + risk)"),
				),
				mcp.WithString("session_id",
					mcp.Description("Session ID (optional, defaults to latest)"),
				),
				mcp.WithNumber("limit",
					mcp.Description("Number of sessions for brief mode (default: 1)"),
				),
			),
			Handler: withBuddyTracker(st, stateConsolidatedHandler(claudeHome, st)),
		},

		// 2. buddy_sessions: List recent sessions (unchanged).
		server.ServerTool{
			Tool: mcp.NewTool("buddy_sessions",
				mcp.WithDescription("List recent Claude Code sessions with timestamps, durations, and tool counts. Use when browsing past sessions to find one to resume or compare. The 'limit' parameter controls how many sessions to return (default: 10). Results are sorted by recency; only sessions from the current Claude home directory are included."),
				mcp.WithTitleAnnotation("Recent Sessions"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithIdempotentHintAnnotation(true),
				mcp.WithOpenWorldHintAnnotation(false),
				mcp.WithNumber("limit",
					mcp.Description("Maximum sessions to return (default: 10)"),
				),
			),
			Handler: withBuddyTracker(st, sessionsHandler(claudeHome)),
		},

		// 3. buddy_resume: Restore context from a previous session (unchanged).
		server.ServerTool{
			Tool: mcp.NewTool("buddy_resume",
				mcp.WithDescription("Restore full context from a previous session including summary, key events, decisions made, and files modified. Use when starting a new session that continues earlier work, or when you need to recall what happened in a past session. Specify 'session_id' for a specific session or 'project' to filter by project path; defaults to the most recent session."),
				mcp.WithTitleAnnotation("Resume Context"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithIdempotentHintAnnotation(true),
				mcp.WithOpenWorldHintAnnotation(false),
				mcp.WithString("session_id",
					mcp.Description("Session ID to resume from (optional, defaults to most recent)"),
				),
				mcp.WithString("project",
					mcp.Description("Project name or path to filter sessions"),
				),
			),
			Handler: withBuddyTracker(st, resumeHandler(st)),
		},

		// 4. buddy_recall: Search pre-compact conversation history (unchanged).
		server.ServerTool{
			Tool: mcp.NewTool("buddy_recall",
				mcp.WithDescription("Search pre-compact conversation history to recover details lost after context compaction. Use when important information from earlier in the session is no longer visible after a compact event. The required 'query' parameter accepts keyword searches; 'segment' selects which compact segment to search (0 = pre-compact)."),
				mcp.WithTitleAnnotation("Recall Details"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithIdempotentHintAnnotation(true),
				mcp.WithOpenWorldHintAnnotation(false),
				mcp.WithString("query",
					mcp.Description("Search query"),
					mcp.Required(),
				),
				mcp.WithString("session_id",
					mcp.Description("Session ID to search in"),
				),
				mcp.WithNumber("segment",
					mcp.Description("Compact segment (0=pre-compact)"),
				),
				mcp.WithNumber("limit",
					mcp.Description("Maximum results (default: 10)"),
				),
			),
			Handler: withBuddyTracker(st, recallHandler(st)),
		},

		// 5. buddy_knowledge: Consolidated knowledge search (patterns + decisions + cross_project).
		server.ServerTool{
			Tool: mcp.NewTool("buddy_knowledge",
				mcp.WithDescription("Search accumulated knowledge including error solutions, architecture patterns, design decisions, and cross-project insights. Use when encountering a problem that may have been solved before, or when checking past architectural decisions. Set 'scope' to 'global' for cross-project search or 'project' (default) for current project; 'type' filters by kind (error_solution, architecture, decision, tool_usage)."),
				mcp.WithTitleAnnotation("Knowledge Search"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithIdempotentHintAnnotation(true),
				mcp.WithOpenWorldHintAnnotation(false),
				mcp.WithString("query",
					mcp.Description("Search query (required)"),
					mcp.Required(),
				),
				mcp.WithString("scope",
					mcp.Description("Search scope: project (default) or global (cross-project)"),
				),
				mcp.WithString("type",
					mcp.Description("Filter: all (default), error_solution, architecture, decision, tool_usage"),
				),
				mcp.WithString("session_id",
					mcp.Description("Session ID to filter decisions"),
				),
				mcp.WithString("project",
					mcp.Description("Project name to filter"),
				),
				mcp.WithNumber("limit",
					mcp.Description("Maximum results (default: 5)"),
				),
			),
			Handler: withBuddyTracker(st, knowledgeConsolidatedHandler(st, emb)),
		},

		// 6. buddy_guidance: Consolidated guidance (suggest + alerts + next_step + pending_nudges).
		server.ServerTool{
			Tool: mcp.NewTool("buddy_guidance",
				mcp.WithDescription("Get workflow guidance including active alerts, actionable recommendations, suggested next steps, and pending nudges. Use when you want to check what buddy recommends doing next, review outstanding warnings, or see queued suggestions. The 'focus' parameter narrows results: 'alerts' for warnings only, 'recommendations' for suggestions, 'next_steps' for prioritized actions, 'pending' for queued nudges."),
				mcp.WithTitleAnnotation("Workflow Guidance"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithIdempotentHintAnnotation(true),
				mcp.WithOpenWorldHintAnnotation(false),
				mcp.WithString("focus",
					mcp.Description("Focus area: all (default), alerts, recommendations, next_steps, pending"),
				),
				mcp.WithString("session_id",
					mcp.Description("Session ID (optional, defaults to latest)"),
				),
				mcp.WithString("context",
					mcp.Description("Additional context for next_steps recommendations"),
				),
			),
			Handler: withBuddyTracker(st, guidanceConsolidatedHandler(claudeHome, st)),
		},

		// 7. buddy_plan: Consolidated planning (estimate + task_progress + strategic_plan).
		server.ServerTool{
			Tool: mcp.NewTool("buddy_plan",
				mcp.WithDescription("Plan and estimate tasks using historical session data for complexity estimation, multi-session progress tracking, and strategic workflow planning. Use when starting a new task to get effort estimates, checking progress on ongoing work, or generating an optimal phase sequence. The 'mode' parameter selects the function: 'estimate' returns median tool counts, 'progress' shows tracking, 'strategy' generates a phase-sequenced plan."),
				mcp.WithTitleAnnotation("Planning"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithIdempotentHintAnnotation(true),
				mcp.WithOpenWorldHintAnnotation(false),
				mcp.WithString("mode",
					mcp.Description("Mode: estimate (task complexity), progress (multi-session tracking), strategy (optimal workflow), all"),
				),
				mcp.WithString("task_type",
					mcp.Description("Task type: bugfix, feature, refactor, test, explore, debug"),
				),
				mcp.WithString("project",
					mcp.Description("Project path"),
				),
				mcp.WithString("session_id",
					mcp.Description("Session ID for progress tracking"),
				),
			),
			Handler: withBuddyTracker(st, planConsolidatedHandler(st)),
		},

		// 8. buddy_diagnose: Consolidated diagnosis (diagnose + fix).
		server.ServerTool{
			Tool: mcp.NewTool("buddy_diagnose",
				mcp.WithDescription("Diagnose errors and generate concrete fix patches with before/after code and verification commands. Use when a tool produces an error you need to understand, or when code quality findings need automated fixes. Provide 'error_output' for error diagnosis, or 'file_path' with 'finding_rule' for code fix generation. Supports Go compile errors, test failure correlation, and AST-based code fixes."),
				mcp.WithTitleAnnotation("Error Diagnosis & Fix"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithIdempotentHintAnnotation(true),
				mcp.WithOpenWorldHintAnnotation(false),
				mcp.WithString("error_output",
					mcp.Description("Error message or command output to diagnose"),
				),
				mcp.WithString("tool_name",
					mcp.Description("Tool that produced the error (e.g., 'Bash', 'Edit')"),
				),
				mcp.WithString("file_path",
					mcp.Description("File path related to the error or containing the finding"),
				),
				mcp.WithString("finding_rule",
					mcp.Description("Code quality rule identifier for fix generation"),
				),
				mcp.WithString("message",
					mcp.Description("Finding message text (for fix generation when rule is not known)"),
				),
				mcp.WithNumber("line",
					mcp.Description("Line number of the finding"),
				),
			),
			Handler: withBuddyTracker(st, diagnoseConsolidatedHandler(st)),
		},

		// 9. buddy_feedback: Provide suggestion feedback (unchanged).
		server.ServerTool{
			Tool: mcp.NewTool("buddy_feedback",
				mcp.WithDescription("Rate the quality of a buddy suggestion to improve future relevance via Thompson Sampling. Use when a suggestion was particularly helpful or unhelpful and you want to provide explicit signal. The required 'pattern' identifies which suggestion to rate; 'rating' must be helpful, partially_helpful, not_helpful, or misleading. Explicit feedback overrides automatic inference and has lasting impact on prioritization."),
				mcp.WithTitleAnnotation("Suggestion Feedback"),
				mcp.WithReadOnlyHintAnnotation(false),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithIdempotentHintAnnotation(false),
				mcp.WithOpenWorldHintAnnotation(false),
				mcp.WithString("pattern",
					mcp.Required(),
					mcp.Description("The suggestion pattern name"),
				),
				mcp.WithString("rating",
					mcp.Required(),
					mcp.Description("Rating: helpful, partially_helpful, not_helpful, or misleading"),
				),
				mcp.WithNumber("suggestion_id",
					mcp.Description("Specific suggestion outcome ID"),
				),
				mcp.WithString("comment",
					mcp.Description("Additional feedback details"),
				),
			),
			Handler: withBuddyTracker(st, feedbackHandler(st)),
		},

		// 10. buddy_skill_context: Skill-specific context (unchanged).
		server.ServerTool{
			Tool: mcp.NewTool("buddy_skill_context",
				mcp.WithDescription("Get aggregated session context tailored for a specific skill, including relevant files, health status, test results, patterns, and alerts. Use when a skill needs session-aware context before executing its workflow. The required 'skill_name' determines which context facets are included. Returns a compact summary designed to fit within skill context budgets."),
				mcp.WithTitleAnnotation("Skill Context"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithIdempotentHintAnnotation(true),
				mcp.WithOpenWorldHintAnnotation(false),
				mcp.WithString("skill_name",
					mcp.Required(),
					mcp.Description("Name of the skill requesting context"),
				),
				mcp.WithString("session_id",
					mcp.Description("Session ID (optional, defaults to latest)"),
				),
			),
			Handler: withBuddyTracker(st, skillContextHandler(claudeHome)),
		},
	)

	// Register resources and prompts.
	registerResources(s, claudeHome, st)
	registerPrompts(s, claudeHome, st)

	return s
}
