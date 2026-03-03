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

  knowledge   — Search Claude Code docs and best practices
  review      — Analyze your project's Claude Code utilization
  suggest     — Suggest .claude/ config changes based on recent code changes

When to use alfred tools:
- Reviewing or auditing .claude/ configuration (agents, skills, rules, hooks, MCP) → call review first
- Creating or modifying .claude/ configuration files → call knowledge for best practices first
- Looking up how a Claude Code feature works → call knowledge
- After code changes, check if .claude/ config needs updating → call suggest

Do NOT review or create .claude/ configuration by only reading files.
alfred's knowledge base has current best practices not in your training data.
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

	s.AddTools(
		server.ServerTool{
			Tool: mcp.NewTool("knowledge",
				mcp.WithDescription("Search Claude Code documentation and best practices. Uses hybrid vector + FTS5 search with Voyage AI reranking."),
				mcp.WithTitleAnnotation("Knowledge Search"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithString("query", mcp.Description("Search query"), mcp.Required()),
				mcp.WithNumber("limit", mcp.Description("Maximum results (default: 5)")),
			),
			Handler: docsSearchHandler(st, emb),
		},

		server.ServerTool{
			Tool: mcp.NewTool("review",
				mcp.WithDescription("Analyze your project's Claude Code utilization. Checks CLAUDE.md, skills, rules, hooks, MCP servers, and session history. Returns improvement suggestions."),
				mcp.WithTitleAnnotation("Project Review"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithString("project_path", mcp.Description("Project root path (cwd)")),
			),
			Handler: reviewHandler(defaultClaudeHome()),
		},

		server.ServerTool{
			Tool: mcp.NewTool("suggest",
				mcp.WithDescription("Suggest .claude/ configuration changes based on recent code changes. Analyzes git diff and cross-references with current project setup."),
				mcp.WithTitleAnnotation("Config Suggestions"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithString("project_path", mcp.Description("Project root path (cwd)")),
			),
			Handler: suggestHandler(defaultClaudeHome()),
		},
	)

	return s
}
