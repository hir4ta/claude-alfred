package mcpserver

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/hir4ta/claude-alfred/internal/embedder"
	"github.com/hir4ta/claude-alfred/internal/spec"
	"github.com/hir4ta/claude-alfred/internal/store"
)

type reviewFinding struct {
	Layer    string `json:"layer"`              // "spec" | "knowledge" | "best_practice"
	Severity string `json:"severity"`           // "info" | "warning"
	Message  string `json:"message"`
	Source   string `json:"source,omitempty"`
}

// butlerReviewHandler performs a 3-layer knowledge-powered code review.
func butlerReviewHandler(st *store.Store, emb *embedder.Embedder) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projectPath := req.GetString("project_path", "")
		focus := req.GetString("focus", "")

		if projectPath == "" {
			return mcp.NewToolResultError("project_path is required"), nil
		}

		diff := getReviewDiff(projectPath)
		if diff == "" {
			return marshalResult(map[string]any{
				"findings":      []reviewFinding{},
				"finding_count": 0,
				"message":       "no changes to review",
			})
		}

		var findings []reviewFinding

		// Layer 1: Spec-Aware Review
		taskSlug, err := spec.ReadActive(projectPath)
		if err == nil {
			sd := &spec.SpecDir{ProjectPath: projectPath, TaskSlug: taskSlug}
			if sd.Exists() {
				findings = append(findings, reviewAgainstSpec(sd, diff)...)
			}
		}

		// Layer 2: Knowledge-Powered Review (semantic search for related knowledge)
		if emb != nil {
			findings = append(findings, reviewAgainstKnowledge(ctx, st, emb, diff, focus)...)
		}

		// Layer 3: Best Practices Review (FTS search)
		findings = append(findings, reviewAgainstBestPractices(st, diff, focus)...)

		return marshalResult(map[string]any{
			"diff_lines":     len(strings.Split(diff, "\n")),
			"findings":       findings,
			"finding_count":  len(findings),
			"layers_checked": []string{"spec", "knowledge", "best_practice"},
		})
	}
}

// getReviewDiff tries staged, unstaged, then recent 3 commits.
func getReviewDiff(projectPath string) string {
	for _, args := range [][]string{
		{"diff", "--cached"},
		{"diff"},
		{"diff", "HEAD~3..HEAD"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = projectPath
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			s := string(out)
			if len(s) > 32*1024 {
				s = s[:32*1024]
			}
			return s
		}
	}
	return ""
}

// reviewAgainstSpec checks changes against decisions.md, knowledge.md, and requirements.md.
func reviewAgainstSpec(sd *spec.SpecDir, _ string) []reviewFinding {
	var findings []reviewFinding

	// Check against decisions
	decisions, err := sd.ReadFile(spec.FileDecisions)
	if err == nil && decisions != "" {
		decisionCount := max(strings.Count(decisions, "## ")-1, 0) // exclude header
		if decisionCount > 0 {
			findings = append(findings, reviewFinding{
				Layer:    "spec",
				Severity: "info",
				Message:  fmt.Sprintf("Review against %d recorded decisions in spec '%s'.", decisionCount, sd.TaskSlug),
				Source:   sd.FilePath(spec.FileDecisions),
			})
		}
	}

	// Check knowledge for dead ends
	knowledge, err := sd.ReadFile(spec.FileKnowledge)
	if err == nil && knowledge != "" {
		discoveryCount := max(strings.Count(knowledge, "## ")-1, 0)
		if discoveryCount > 0 {
			findings = append(findings, reviewFinding{
				Layer:    "spec",
				Severity: "info",
				Message:  fmt.Sprintf("Review against %d knowledge entries (including dead ends) in spec.", discoveryCount),
				Source:   sd.FilePath(spec.FileKnowledge),
			})
		}
	}

	// Check out-of-scope
	requirements, err := sd.ReadFile(spec.FileRequirements)
	if err == nil && strings.Contains(requirements, "## Out of Scope") {
		findings = append(findings, reviewFinding{
			Layer:    "spec",
			Severity: "info",
			Message:  "Out of Scope section exists in requirements. Verify changes respect scope boundaries.",
			Source:   sd.FilePath(spec.FileRequirements),
		})
	}

	return findings
}

// reviewAgainstKnowledge performs semantic search for related spec knowledge.
func reviewAgainstKnowledge(ctx context.Context, st *store.Store, emb *embedder.Embedder, diff, focus string) []reviewFinding {
	var findings []reviewFinding

	query := focus
	if query == "" {
		query = diff
		if len(query) > 500 {
			query = query[:500]
		}
	}

	queryVec, err := emb.EmbedForSearch(ctx, query)
	if err != nil {
		return findings
	}

	matches, err := st.HybridSearch(queryVec, query, "spec", 3, 12)
	if err != nil || len(matches) == 0 {
		return findings
	}

	ids := make([]int64, len(matches))
	for i, m := range matches {
		ids[i] = m.DocID
	}
	docs, err := st.GetDocsByIDs(ids)
	if err != nil {
		return findings
	}

	for _, doc := range docs {
		findings = append(findings, reviewFinding{
			Layer:    "knowledge",
			Severity: "info",
			Message:  fmt.Sprintf("Related spec knowledge: %s", truncate(doc.SectionPath, 100)),
			Source:   doc.URL,
		})
	}

	return findings
}

// reviewAgainstBestPractices performs FTS search for relevant documentation.
func reviewAgainstBestPractices(st *store.Store, _ string, focus string) []reviewFinding {
	var findings []reviewFinding

	query := focus
	if query == "" {
		query = "code review best practices"
	}

	snippets := queryKB(st, query, 3)
	for _, s := range snippets {
		findings = append(findings, reviewFinding{
			Layer:    "best_practice",
			Severity: "info",
			Message:  fmt.Sprintf("Related best practice: %s", truncate(s.SectionPath, 100)),
			Source:   s.URL,
		})
	}

	return findings
}
