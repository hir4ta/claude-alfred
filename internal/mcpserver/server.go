package mcpserver

import (
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/hir4ta/claude-alfred/internal/embedder"
	"github.com/hir4ta/claude-alfred/internal/store"
)

const serverInstructions = `alfred is your silent butler for Claude Code.

He never interrupts your work. When you need him, he's ready:

  knowledge    — Search Claude Code docs and best practices
  review       — Deep audit of .claude/ config: reads file contents, checks sizes, cross-references with best practices
  suggest      — Reads git diff content, detects change patterns, suggests specific config updates with best practices
  butler-init   — Initialize spec for a new development task (creates .alfred/specs/ files + DB sync)
  butler-update — Record decisions, knowledge, task progress to active spec (auto DB sync)
  butler-status — Get current task state for context restoration after compact/new session
  butler-review — 3-layer knowledge-powered code review (spec + knowledge + best practices)

When to use alfred tools:
- Reviewing or auditing .claude/ configuration → call review first (reads file contents, checks skill sizes and structure, validates rules, cross-references with knowledge base)
- Creating or modifying .claude/ configuration files → call knowledge for best practices first
- Looking up how a Claude Code feature works → call knowledge
- After code changes, check if .claude/ config needs updating → call suggest (reads diff content, detects patterns like new APIs/deps/tests)
- Starting a new development task → call butler-init to create spec
- Making design decisions or discovering knowledge → call butler-update to record
- Starting/resuming a session → call butler-status to check active task
- Reviewing code changes against spec, knowledge, and best practices → call butler-review

Do NOT review or create .claude/ configuration by only reading files.
review and suggest cross-reference your config against best practices from the knowledge base — information not in your training data.
Always: alfred tools first → then read/edit files.
`

// defaultClaudeHome returns the default Claude Code configuration directory.
func defaultClaudeHome() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// New creates a new MCP server with all tools registered.
func New(st *store.Store, emb *embedder.Embedder) *server.MCPServer {
	s := server.NewMCPServer(
		"alfred",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithInstructions(serverInstructions),
		server.WithLogging(),
	)

	ar := newAutoRefresher(st, emb)

	s.AddTools(
		server.ServerTool{
			Tool: mcp.NewTool("knowledge",
				mcp.WithDescription("Search Claude Code documentation and best practices. Uses hybrid vector + FTS5 search with Voyage AI reranking."),
				mcp.WithTitleAnnotation("Knowledge Search"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithString("query", mcp.Description("Search query"), mcp.Required()),
				mcp.WithNumber("limit", mcp.Description("Maximum results (default: 5)")),
			),
			Handler: docsSearchHandler(st, emb, ar),
		},

		server.ServerTool{
			Tool: mcp.NewTool("review",
				mcp.WithDescription("Deep audit of .claude/ configuration against best practices. Reads file contents, checks skill sizes and structure, validates rules, and cross-references findings with the knowledge base. Returns structured suggestions with severity levels and documentation references."),
				mcp.WithTitleAnnotation("Project Review"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithString("project_path", mcp.Description("Project root path (cwd)")),
			),
			Handler: reviewHandler(defaultClaudeHome(), st, emb),
		},

		server.ServerTool{
			Tool: mcp.NewTool("suggest",
				mcp.WithDescription("Analyze recent code changes and suggest specific .claude/ configuration updates. Reads git diff content to detect change patterns (new APIs, dependencies, tests, migrations), cross-references with current config and best practices from the knowledge base."),
				mcp.WithTitleAnnotation("Config Suggestions"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithString("project_path", mcp.Description("Project root path (cwd)")),
			),
			Handler: suggestHandler(defaultClaudeHome(), st, emb),
		},

		server.ServerTool{
			Tool: mcp.NewTool("butler-init",
				mcp.WithDescription("Initialize a new spec for a development task. Creates .alfred/specs/{task_slug}/ with 6 template files (requirements, design, tasks, decisions, knowledge, session) and syncs to the knowledge DB for semantic search."),
				mcp.WithString("project_path", mcp.Description("Absolute path to the project root"), mcp.Required()),
				mcp.WithString("task_slug", mcp.Description("URL-safe task identifier (e.g., 'add-auth', 'fix-memory-leak')"), mcp.Required()),
				mcp.WithString("description", mcp.Description("Brief description of the task goal")),
			),
			Handler: butlerInitHandler(st, emb),
		},

		server.ServerTool{
			Tool: mcp.NewTool("butler-update",
				mcp.WithDescription("Update a spec file for the active task. Appends or replaces content, then syncs to the knowledge DB. Use for recording decisions, knowledge discoveries, task progress, and session state."),
				mcp.WithString("project_path", mcp.Description("Absolute path to the project root"), mcp.Required()),
				mcp.WithString("file", mcp.Description("Spec file to update: requirements.md, design.md, tasks.md, decisions.md, knowledge.md, session.md"), mcp.Required()),
				mcp.WithString("content", mcp.Description("Content to write"), mcp.Required()),
				mcp.WithString("mode", mcp.Description("Write mode: 'append' (default) or 'replace'")),
			),
			Handler: butlerUpdateHandler(st, emb),
		},

		server.ServerTool{
			Tool: mcp.NewTool("butler-status",
				mcp.WithDescription("Get the current spec status for a project. Returns the active task's session state, task list, and requirements. Use at session start to restore context after compact or new session."),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithString("project_path", mcp.Description("Absolute path to the project root"), mcp.Required()),
			),
			Handler: butlerStatusHandler(),
		},

		server.ServerTool{
			Tool: mcp.NewTool("butler-review",
				mcp.WithDescription("3-layer knowledge-powered code review. Layer 1: checks changes against active spec (decisions, requirements scope). Layer 2: semantic search against accumulated knowledge (past bugs, dead ends). Layer 3: best practices from documentation sources."),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithString("project_path", mcp.Description("Absolute path to the project root"), mcp.Required()),
				mcp.WithString("focus", mcp.Description("Optional focus area for the review (e.g., 'auth logic', 'error handling')")),
			),
			Handler: butlerReviewHandler(st, emb),
		},
	)

	return s
}
