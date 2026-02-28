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
				mcp.WithDescription("Get session state: health, statistics, burst state, and predictions. Use 'detail' to control depth: brief (stats only), standard (full snapshot), outlook (strategic view)."),
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
				mcp.WithDescription("List recent Claude Code sessions with basic metadata."),
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
				mcp.WithDescription("Resume context from a previous session. Returns summary, events, decisions, and files modified."),
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
				mcp.WithDescription("Search pre-compact conversation history for lost details."),
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
				mcp.WithDescription("Search knowledge: past patterns, design decisions, and cross-project solutions. Use 'scope' for project vs global, 'type' to filter by pattern kind."),
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
				mcp.WithDescription("Get workflow guidance: alerts, recommendations, next steps, and pending nudges. Use 'focus' to narrow: all (default), alerts, recommendations, next_steps, pending."),
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
				mcp.WithDescription("Planning tools: task estimation, progress tracking, and strategic plans. Use 'mode' to select: estimate, progress, strategy, or all."),
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
				mcp.WithDescription("Diagnose errors and generate fix patches. Provide error_output for diagnosis, or file_path with finding_rule for code fixes. Set fix=true to include a patch alongside diagnosis."),
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
				mcp.WithDescription("Rate suggestion quality: helpful, partially_helpful, not_helpful, or misleading."),
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
				mcp.WithDescription("Get aggregated session context for a specific skill."),
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
