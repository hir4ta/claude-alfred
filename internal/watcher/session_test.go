package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDecodeProjectName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"-Users-user-Projects-myapp", "myapp"},
		{"-Users-user-cli", "cli"},
		{"singleword", "singleword"},
		{"", ""},
		{"-", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := decodeProjectName(tt.input)
			if got != tt.want {
				t.Errorf("decodeProjectName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultClaudeHome(t *testing.T) {
	t.Parallel()
	got := DefaultClaudeHome()
	if !strings.HasSuffix(got, "/.claude") {
		t.Errorf("DefaultClaudeHome() = %q, want suffix /.claude", got)
	}
}

func TestExtractFirstPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "valid user message",
			content: `{"type":"user","message":{"role":"user","content":"hello world"}}`,
			want:    "hello world",
		},
		{
			name:    "long message truncated",
			content: `{"type":"user","message":{"role":"user","content":"あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみ"}}`,
			want:    "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほ...",
		},
		{
			name:    "empty file",
			content: "",
			want:    "",
		},
		{
			name:    "no user message",
			content: `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}`,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "test.jsonl")
			os.WriteFile(path, []byte(tt.content), 0o644)

			got := extractFirstPrompt(path)
			if got != tt.want {
				t.Errorf("extractFirstPrompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFirstPrompt_NonexistentFile(t *testing.T) {
	t.Parallel()
	got := extractFirstPrompt("/nonexistent/path/file.jsonl")
	if got != "" {
		t.Errorf("extractFirstPrompt(nonexistent) = %q, want empty", got)
	}
}

func TestListSessions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	projDir := filepath.Join(dir, "projects", "-Users-user-Projects-myapp")
	os.MkdirAll(projDir, 0o755)

	// Create session files
	os.WriteFile(filepath.Join(projDir, "session-a.jsonl"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(projDir, "session-b.jsonl"), []byte("{}"), 0o644)
	// Non-JSONL should be ignored
	os.WriteFile(filepath.Join(projDir, "readme.txt"), []byte("not a session"), 0o644)

	sessions, err := ListSessions(dir)
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("ListSessions() returned %d sessions, want 2", len(sessions))
	}
	for _, s := range sessions {
		if s.Project != "myapp" {
			t.Errorf("session.Project = %q, want %q", s.Project, "myapp")
		}
	}
}

func TestListSessions_EmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "projects"), 0o755)

	sessions, err := ListSessions(dir)
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("ListSessions() returned %d sessions, want 0", len(sessions))
	}
}

func TestFindRecentSessions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	projDir := filepath.Join(dir, "projects", "-Users-user-Projects-app")
	os.MkdirAll(projDir, 0o755)

	// Each session needs a unique prompt to avoid dedup.
	for i := 0; i < 5; i++ {
		prompt := fmt.Sprintf("task %d", i)
		content := fmt.Sprintf(`{"type":"user","message":{"role":"user","content":"%s"}}`, prompt)
		path := filepath.Join(projDir, fmt.Sprintf("session-%c.jsonl", rune('a'+i)))
		os.WriteFile(path, []byte(content), 0o644)
	}

	sessions, err := FindRecentSessions(dir, 3)
	if err != nil {
		t.Fatalf("FindRecentSessions() error: %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("FindRecentSessions(3) returned %d sessions, want 3", len(sessions))
	}
}

func TestDeduplicateByPrompt(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    []RecentSession
		wantIDs  []string
	}{
		{
			name: "resume pair within seconds — keep newest only",
			input: []RecentSession{
				{SessionID: "b3a87513", Project: "alfred", FirstPrompt: "このclaude-alfred...", ModTime: now},
				{SessionID: "30df70a9", Project: "alfred", FirstPrompt: "このclaude-alfred...", ModTime: now.Add(-30 * time.Second)},
			},
			wantIDs: []string{"b3a87513"},
		},
		{
			name: "same prompt hours apart — keep both (separate sessions)",
			input: []RecentSession{
				{SessionID: "c4d0832b", Project: "alfred", FirstPrompt: "このclaude-alfred...", ModTime: now.Add(-8 * time.Hour)},
				{SessionID: "a5e82f6d", Project: "alfred", FirstPrompt: "このclaude-alfred...", ModTime: now.Add(-9 * time.Hour)},
			},
			wantIDs: []string{"c4d0832b", "a5e82f6d"},
		},
		{
			name: "full user scenario — 4 sessions, 1 resume pair + 2 separate",
			input: []RecentSession{
				{SessionID: "b3a87513", Project: "alfred", FirstPrompt: "このclaude-alfred...", ModTime: now},
				{SessionID: "30df70a9", Project: "alfred", FirstPrompt: "このclaude-alfred...", ModTime: now.Add(-1 * time.Minute)},
				{SessionID: "c4d0832b", Project: "alfred", FirstPrompt: "このclaude-alfred...", ModTime: now.Add(-8 * time.Hour)},
				{SessionID: "a5e82f6d", Project: "alfred", FirstPrompt: "このclaude-alfred...", ModTime: now.Add(-9 * time.Hour)},
			},
			wantIDs: []string{"b3a87513", "c4d0832b", "a5e82f6d"},
		},
		{
			name: "different projects same prompt — keep both",
			input: []RecentSession{
				{SessionID: "aaa", Project: "proj-a", FirstPrompt: "same prompt", ModTime: now},
				{SessionID: "bbb", Project: "proj-b", FirstPrompt: "same prompt", ModTime: now.Add(-1 * time.Minute)},
			},
			wantIDs: []string{"aaa", "bbb"},
		},
		{
			name: "different prompts same project — keep both",
			input: []RecentSession{
				{SessionID: "aaa", Project: "alfred", FirstPrompt: "first task", ModTime: now},
				{SessionID: "bbb", Project: "alfred", FirstPrompt: "second task", ModTime: now.Add(-1 * time.Minute)},
			},
			wantIDs: []string{"aaa", "bbb"},
		},
		{
			name: "empty prompt sessions — always kept",
			input: []RecentSession{
				{SessionID: "aaa", Project: "alfred", FirstPrompt: "", ModTime: now},
				{SessionID: "bbb", Project: "alfred", FirstPrompt: "", ModTime: now.Add(-10 * time.Second)},
			},
			wantIDs: []string{"aaa", "bbb"},
		},
		{
			name: "triple resume chain — all within 2 minutes",
			input: []RecentSession{
				{SessionID: "aaa", Project: "alfred", FirstPrompt: "fix bug", ModTime: now},
				{SessionID: "bbb", Project: "alfred", FirstPrompt: "fix bug", ModTime: now.Add(-1 * time.Minute)},
				{SessionID: "ccc", Project: "alfred", FirstPrompt: "fix bug", ModTime: now.Add(-2 * time.Minute)},
			},
			wantIDs: []string{"aaa"},
		},
		{
			name: "genuine session 6 min after resume — kept",
			input: []RecentSession{
				{SessionID: "aaa", Project: "alfred", FirstPrompt: "task X", ModTime: now},
				{SessionID: "bbb", Project: "alfred", FirstPrompt: "task X", ModTime: now.Add(-1 * time.Minute)},
				{SessionID: "ccc", Project: "alfred", FirstPrompt: "task X", ModTime: now.Add(-6 * time.Minute)},
			},
			wantIDs: []string{"aaa", "ccc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicateByPrompt(tt.input)
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("got %d sessions, want %d", len(got), len(tt.wantIDs))
			}
			for i, want := range tt.wantIDs {
				if got[i].SessionID != want {
					t.Errorf("session[%d] = %s, want %s", i, got[i].SessionID, want)
				}
			}
		})
	}
}
