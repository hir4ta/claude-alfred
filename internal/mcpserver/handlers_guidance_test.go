package mcpserver

import (
	"path/filepath"
	"testing"

	"github.com/hir4ta/claude-alfred/internal/analyzer"
	"github.com/hir4ta/claude-alfred/internal/store"
)

func TestBuildBriefing_NilStore(t *testing.T) {
	t.Parallel()
	items := buildBriefing(nil, "/some/project")
	if len(items) != 0 {
		t.Errorf("buildBriefing(nil) returned %d items, want 0", len(items))
	}
}

func TestBuildBriefing_EmptyProject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("store.Open() = %v", err)
	}
	defer st.Close()

	items := buildBriefing(st, "")
	if len(items) != 0 {
		t.Errorf("buildBriefing(st, \"\") returned %d items, want 0", len(items))
	}
}

func TestBuildBriefing_NoSessions(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("store.Open() = %v", err)
	}
	defer st.Close()

	items := buildBriefing(st, "/some/project")
	if len(items) != 0 {
		t.Errorf("buildBriefing(fresh store) returned %d items, want 0", len(items))
	}
}

func TestLevelString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		level analyzer.FeedbackLevel
		want  string
	}{
		{"low", analyzer.LevelLow, "low"},
		{"insight", analyzer.LevelInsight, "insight"},
		{"warning", analyzer.LevelWarning, "warning"},
		{"action", analyzer.LevelAction, "action"},
		{"unknown", 99, "info"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := levelString(tt.level)
			if got != tt.want {
				t.Errorf("levelString(%d) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestTruncateHelper(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		max   int
		want  string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"truncated", "hello world", 5, "hello..."},
		{"empty", "", 5, ""},
		{"cjk", "日本語テスト文字列", 4, "日本語テ..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncate(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}
