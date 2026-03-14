package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// handlePostToolUse provides contextual feedback after Edit/Write operations.
// Tracks modified file patterns and suggests relevant actions.
func handlePostToolUse(ctx context.Context, ev *hookEvent) {
	if ctx.Err() != nil || ev.ToolName == "" {
		return
	}

	if ev.ToolName != "Edit" && ev.ToolName != "Write" {
		return
	}

	// Extract file path from tool input.
	var input struct {
		FilePath string `json:"file_path"`
	}
	if err := json.Unmarshal(ev.ToolInput, &input); err != nil || input.FilePath == "" {
		return
	}

	ext := strings.ToLower(filepath.Ext(input.FilePath))
	basename := filepath.Base(input.FilePath)

	var hints []string

	// Test file modified — suggest running tests.
	if isTestFile(basename) {
		hints = append(hints, "Test file modified. Consider running tests to verify.")
	}

	// Config file changes.
	for _, cf := range []string{"CLAUDE.md", "hooks.json", ".mcp.json", "settings.json"} {
		if basename == cf {
			hints = append(hints, fmt.Sprintf("Config file %s modified. Verify consistency with project setup.", cf))
			break
		}
	}

	// Go files: suggest go vet/test.
	if ext == ".go" && !isTestFile(basename) {
		hints = append(hints, "Go source modified. Run 'go test' and 'go vet' for the affected package.")
	}

	// Schema/migration files: high-impact change.
	if strings.Contains(basename, "schema") || strings.Contains(basename, "migration") {
		hints = append(hints, "Schema/migration file modified. Verify migration compatibility and test with existing data.")
	}

	if len(hints) == 0 {
		return
	}

	output := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "PostToolUse",
			"additionalContext": strings.Join(hints, "\n"),
		},
	}
	data, _ := json.Marshal(output)
	fmt.Fprintln(os.Stdout, string(data))
	debugf("PostToolUse: %d hints for %s", len(hints), basename)
}

// isTestFile checks if a filename looks like a test file.
func isTestFile(basename string) bool {
	lower := strings.ToLower(basename)
	return strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".test.ts") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".spec.ts") ||
		strings.HasSuffix(lower, ".spec.js") ||
		strings.HasSuffix(lower, "_test.py") ||
		strings.HasPrefix(lower, "test_")
}
