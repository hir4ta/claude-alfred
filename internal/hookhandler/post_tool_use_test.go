package hookhandler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hir4ta/claude-alfred/internal/sessiondb"
)

func openPostToolTestDB(t *testing.T) *sessiondb.SessionDB {
	t.Helper()
	id := "test-post-" + strings.ReplaceAll(t.Name(), "/", "-")
	sdb, err := sessiondb.Open(id)
	if err != nil {
		t.Fatalf("sessiondb.Open(%q) = %v", id, err)
	}
	t.Cleanup(func() { _ = sdb.Destroy() })
	return sdb
}

func TestClassifyPhase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		toolName  string
		toolInput string
		want      string
	}{
		{"read", "Read", `{"file_path":"main.go"}`, "read"},
		{"grep", "Grep", `{"pattern":"foo"}`, "read"},
		{"glob", "Glob", `{"pattern":"*.go"}`, "read"},
		{"edit", "Edit", `{"file_path":"main.go"}`, "write"},
		{"write", "Write", `{"file_path":"main.go"}`, "write"},
		{"notebook", "NotebookEdit", `{}`, "write"},
		{"plan", "EnterPlanMode", `{}`, "plan"},
		{"go test", "Bash", `{"command":"go test ./..."}`, "test"},
		{"npm test", "Bash", `{"command":"npm test"}`, "test"},
		{"go build", "Bash", `{"command":"go build ./..."}`, "compile"},
		{"make", "Bash", `{"command":"make all"}`, "compile"},
		{"ls", "Bash", `{"command":"ls -la"}`, ""},
		{"agent", "Agent", `{}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyPhase(tt.toolName, json.RawMessage(tt.toolInput))
			if got != tt.want {
				t.Errorf("classifyPhase(%q, ...) = %q, want %q", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestIsGitCommitCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cmd  string
		want bool
	}{
		{`git commit -m "fix"`, true},
		{`git merge main`, true},
		{`git status`, false},
		{`git push`, false},
		{`echo "git commit"`, true}, // substring match (acceptable)
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			t.Parallel()
			if got := isGitCommitCommand(tt.cmd); got != tt.want {
				t.Errorf("isGitCommitCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestRecordFeaturePreference(t *testing.T) {
	t.Parallel()
	tests := []struct {
		toolName string
		key      string
	}{
		{"EnterPlanMode", "pref:plan_mode_count"},
		{"EnterWorktree", "pref:worktree_count"},
		{"Agent", "pref:agent_count"},
		{"Skill", "pref:skill_count"},
		{"TeamCreate", "pref:team_count"},
	}
	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			t.Parallel()
			sdb := openPostToolTestDB(t)
			recordFeaturePreference(sdb, tt.toolName)
			recordFeaturePreference(sdb, tt.toolName)
			val, _ := sdb.GetContext(tt.key)
			if val != "2" {
				t.Errorf("recordFeaturePreference(%q) x2: got %q, want \"2\"", tt.toolName, val)
			}
		})
	}
}

func TestRecordFeaturePreference_Ignored(t *testing.T) {
	t.Parallel()
	sdb := openPostToolTestDB(t)
	recordFeaturePreference(sdb, "Read")
	recordFeaturePreference(sdb, "Bash")
	// No pref: keys should be set.
	for _, key := range []string{"plan_mode", "worktree", "agent", "skill", "team"} {
		val, _ := sdb.GetContext("pref:" + key + "_count")
		if val != "" {
			t.Errorf("unexpected pref key %q = %q after recording Read/Bash", key, val)
		}
	}
}

func TestBuildMCPEnrichment(t *testing.T) {
	t.Parallel()

	t.Run("empty session", func(t *testing.T) {
		t.Parallel()
		sdb := openPostToolTestDB(t)
		if got := buildMCPEnrichment(sdb); got != "" {
			t.Errorf("buildMCPEnrichment(empty) = %q, want empty", got)
		}
	})

	t.Run("with context", func(t *testing.T) {
		t.Parallel()
		sdb := openPostToolTestDB(t)
		_ = sdb.SetWorkingSet("intent", "refactor auth")
		_ = sdb.SetContext("task_type", "refactor")
		_ = sdb.SetWorkingSet("git_branch", "feature/auth")

		got := buildMCPEnrichment(sdb)
		if got == "" {
			t.Fatal("buildMCPEnrichment() = empty, want non-empty")
		}
		if !strings.Contains(got, "refactor auth") {
			t.Errorf("missing intent in %q", got)
		}
		if !strings.Contains(got, "refactor") {
			t.Errorf("missing task type in %q", got)
		}
		if !strings.Contains(got, "feature/auth") {
			t.Errorf("missing branch in %q", got)
		}
	})
}
