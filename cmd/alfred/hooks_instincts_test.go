package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestClassifyInstinctPatterns(t *testing.T) {
	t.Parallel()

	session := `## Currently Working On
全5項目の実装完了。テスト全パス。

## Recent Decisions (last 3)
1. detectSettingsConflicts テストは []settingsSource を直接構築して渡す方式に（reviewPermissions経由ではなくユニットテストとして適切）
2. Recency floor=0.5 (0.75では半減期が実質無効になるため)
3. Agent maturity scoring: 50+25(desc)+25(no bypass) の3段階

## Next Steps
1. セルフレビュー実行
2. README.md 更新
`
	patterns := classifyInstinctPatterns(session)
	if len(patterns) == 0 {
		t.Fatal("expected at least one pattern extracted")
	}

	// Should extract from decisions section.
	found := false
	for _, p := range patterns {
		if p.trigger != "" && p.action != "" {
			found = true
			t.Logf("pattern: trigger=%q action=%q domain=%s confidence=%.2f", p.trigger, p.action, p.domain, p.confidence)
		}
	}
	if !found {
		t.Error("no valid trigger/action pair extracted")
	}
}

func TestSplitDecisionToInstinct(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		input      string
		wantT, wantA string
	}{
		{
			"arrow separator",
			"large package → split by concern",
			"large package", "split by concern",
		},
		{
			"colon separator",
			"Error handling style: always use fmt.Errorf with %w",
			"Error handling style", "always use fmt.Errorf with %w",
		},
		{
			"use prefix",
			"Use table-driven tests for all Go tests",
			"関連する作業時", "Use table-driven tests for all Go tests",
		},
		{
			"japanese は pattern",
			"テスト は table-driven で書く",
			"テスト", "table-driven で書く",
		},
		{
			"too short",
			"short",
			"", "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			trigger, action := splitDecisionToInstinct(tt.input)
			if trigger != tt.wantT {
				t.Errorf("trigger = %q, want %q", trigger, tt.wantT)
			}
			if action != tt.wantA {
				t.Errorf("action = %q, want %q", action, tt.wantA)
			}
		})
	}
}

func TestClassifyDomain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		trigger, action string
		want            string
	}{
		{"when writing tests", "use t.Parallel()", "testing"},
		{"git branch strategy", "use feature branches", "git"},
		{"error handling", "wrap with fmt.Errorf", "debugging"},
		{"naming convention", "use camelCase", "code-style"},
		{"deploy pipeline", "run CI/CD first", "workflow"},
		{"something generic", "do generic thing", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.trigger, func(t *testing.T) {
			t.Parallel()
			got := classifyDomain(tt.trigger, tt.action)
			if got != tt.want {
				t.Errorf("classifyDomain(%q, %q) = %q, want %q", tt.trigger, tt.action, got, tt.want)
			}
		})
	}
}

func TestExtractCorrectionPatterns(t *testing.T) {
	t.Parallel()

	content := "mockではなく実際のDBを使ったテスト"
	patterns := extractCorrectionPatterns(content)
	if len(patterns) == 0 {
		t.Fatal("expected correction pattern to be detected")
	}
	if patterns[0].confidence < 0.7 {
		t.Errorf("correction confidence = %.2f, want >= 0.7", patterns[0].confidence)
	}
	if patterns[0].domain != "preferences" {
		t.Errorf("domain = %q, want preferences", patterns[0].domain)
	}
}

func TestProjectHashFromPath(t *testing.T) {
	t.Parallel()
	// Use a non-git directory — should return local-<hash>.
	hash := projectHashFromPath(t.TempDir())
	if hash == "" {
		t.Error("projectHashFromPath returned empty")
	}
	if len(hash) < 10 {
		t.Errorf("hash too short: %q", hash)
	}
}

func TestClassifyInstinctPatterns_Empty(t *testing.T) {
	t.Parallel()
	patterns := classifyInstinctPatterns("")
	if len(patterns) != 0 {
		t.Errorf("expected 0 patterns from empty session, got %d", len(patterns))
	}
}

func TestClassifyInstinctPatterns_Cap(t *testing.T) {
	t.Parallel()
	// Session with many decisions should be capped at 5.
	var decisions strings.Builder
	decisions.WriteString("## Recent Decisions\n")
	for i := 0; i < 20; i++ {
		decisions.WriteString(fmt.Sprintf("%d. pattern-%d → action-%d のように実装する\n", i+1, i, i))
	}
	patterns := classifyInstinctPatterns(decisions.String())
	if len(patterns) > 5 {
		t.Errorf("patterns should be capped at 5, got %d", len(patterns))
	}
}
