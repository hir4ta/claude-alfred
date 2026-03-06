package mcpserver

import (
	"testing"
)

func TestDeduplicateFindings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    []reviewFinding
		wantLen  int
		wantSev  []string // expected severities in order
	}{
		{
			name:    "empty",
			input:   []reviewFinding{},
			wantLen: 0,
		},
		{
			name: "no duplicates",
			input: []reviewFinding{
				{Layer: "spec", Severity: "info", Message: "msg A", Source: "a"},
				{Layer: "knowledge", Severity: "warning", Message: "msg B", Source: "b"},
			},
			wantLen: 2,
			wantSev: []string{"info", "warning"},
		},
		{
			name: "duplicate keeps higher severity",
			input: []reviewFinding{
				{Layer: "spec", Severity: "info", Message: "same message here", Source: "src"},
				{Layer: "knowledge", Severity: "warning", Message: "same message here", Source: "src"},
			},
			wantLen: 1,
			wantSev: []string{"warning"},
		},
		{
			name: "different sources not deduped",
			input: []reviewFinding{
				{Layer: "spec", Severity: "info", Message: "same message", Source: "src1"},
				{Layer: "knowledge", Severity: "info", Message: "same message", Source: "src2"},
			},
			wantLen: 2,
		},
		{
			name: "critical beats warning",
			input: []reviewFinding{
				{Layer: "spec", Severity: "warning", Message: "dead end found", Source: "s"},
				{Layer: "spec", Severity: "critical", Message: "dead end found", Source: "s"},
				{Layer: "spec", Severity: "info", Message: "dead end found", Source: "s"},
			},
			wantLen: 1,
			wantSev: []string{"critical"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deduplicateFindings(tt.input)
			if len(got) != tt.wantLen {
				t.Fatalf("deduplicateFindings() = %d findings, want %d", len(got), tt.wantLen)
			}
			for i, sev := range tt.wantSev {
				if i < len(got) && got[i].Severity != sev {
					t.Errorf("finding[%d].Severity = %q, want %q", i, got[i].Severity, sev)
				}
			}
		})
	}
}

func TestSeverityRank(t *testing.T) {
	t.Parallel()
	if severityRank("critical") <= severityRank("warning") {
		t.Error("critical should rank higher than warning")
	}
	if severityRank("warning") <= severityRank("info") {
		t.Error("warning should rank higher than info")
	}
	if severityRank("info") != severityRank("unknown") {
		t.Error("info and unknown should have same rank")
	}
}

func TestExtractOutOfScopeItems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantLen int
		want    []string
	}{
		{
			name:    "no out of scope section",
			input:   "## Goal\nBuild something\n",
			wantLen: 0,
		},
		{
			name:    "with items",
			input:   "## Goal\nBuild\n\n## Out of Scope\n- subagent\n- LLM summary\n- cross-task search\n\n## Notes\nfoo\n",
			wantLen: 3,
			want:    []string{"subagent", "LLM summary", "cross-task search"},
		},
		{
			name:    "empty section",
			input:   "## Out of Scope\n\n## Next\nstuff\n",
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractOutOfScopeItems(tt.input)
			if len(got) != tt.wantLen {
				t.Fatalf("extractOutOfScopeItems() = %d items, want %d (got %v)", len(got), tt.wantLen, got)
			}
			for i, w := range tt.want {
				if i < len(got) && got[i] != w {
					t.Errorf("item[%d] = %q, want %q", i, got[i], w)
				}
			}
		})
	}
}

func TestExtractDiffContent(t *testing.T) {
	t.Parallel()
	diff := "+++ b/main.go\n+package main\n+\n+func hello() {\n+}\n+import \"fmt\"\n"
	got := extractDiffContent(diff, 100)
	if got == "" {
		t.Error("extractDiffContent() returned empty")
	}
	// Should skip +++ header, empty lines, and lone braces. "func hello() {" is kept (not a lone brace).
	if got != "package main func hello() { import \"fmt\"" {
		t.Errorf("extractDiffContent() = %q", got)
	}
}

func TestExtractDiffKeywords(t *testing.T) {
	t.Parallel()
	diff := "+++ b/internal/mcpserver/server.go\n+++ b/cmd/alfred/main.go\n"
	got := extractDiffKeywords(diff)
	if got == "" {
		t.Error("extractDiffKeywords() returned empty")
	}
}

func TestExtractDecisionExcerpts(t *testing.T) {
	t.Parallel()
	decisions := "# Decisions\n\n## Use YAML for active state\nWe chose YAML.\n\n## Server auth approach\nJWT tokens.\n"
	diff := "+++ b/internal/spec/active.go\n--- a/internal/spec/active.go\n+yaml parsing\n"

	got := extractDecisionExcerpts(decisions, diff)
	// "active" in the file path should match "Use YAML for active state" heading.
	found := false
	for _, e := range got {
		if e == "Use YAML for active state" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected decision excerpt about 'active state', got %v", got)
	}
}
