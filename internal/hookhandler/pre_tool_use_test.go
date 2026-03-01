package hookhandler

import (
	"strings"
	"testing"

	"github.com/hir4ta/claude-alfred/internal/sessiondb"
)

func TestExtractCmdSignature(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cmd  string
		want string
	}{
		{"go test ./internal/store/...", "go test"},
		{"npm install lodash", "npm install"},
		{"ls", "ls"},
		{"", ""},
		{"git commit -m 'fix'", "git commit"},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			t.Parallel()
			if got := extractCmdSignature(tt.cmd); got != tt.want {
				t.Errorf("extractCmdSignature(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestIsCompileCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cmd  string
		want bool
	}{
		{"go build ./...", true},
		{"go install ./cmd/app", true},
		{"make", true},
		{"gcc -o main main.c", true},
		{"cargo build", true},
		{"npm run build", true},
		{"tsc --noEmit", true},
		{"go test ./...", false},
		{"npm install", false},
		{"ls -la", false},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			t.Parallel()
			if got := isCompileCommand(tt.cmd); got != tt.want {
				t.Errorf("isCompileCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestFixForcePush(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cmd         string
		wantFixed   bool
		wantContain string
	}{
		{"force", "git push --force origin main", true, "--force-with-lease"},
		{"already safe", "git push --force-with-lease", false, ""},
		{"no force", "git push origin main", false, ""},
		{"not git push", "echo --force", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			corrected, ctx := fixForcePush(tt.cmd)
			if tt.wantFixed {
				if corrected == "" {
					t.Error("fixForcePush() returned empty, want corrected command")
				}
				if !strings.Contains(corrected, tt.wantContain) {
					t.Errorf("corrected = %q, want to contain %q", corrected, tt.wantContain)
				}
				if ctx == "" {
					t.Error("context should not be empty for corrected commands")
				}
			} else {
				if corrected != "" {
					t.Errorf("fixForcePush(%q) = %q, want empty", tt.cmd, corrected)
				}
			}
		})
	}
}

func TestFixGoTestRaceCache(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		cmd       string
		wantFixed bool
	}{
		{"race without count", "go test -race ./...", true},
		{"race with count", "go test -race -count=1 ./...", false},
		{"no race", "go test ./...", false},
		{"not go test", "echo -race", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			corrected, _ := fixGoTestRaceCache(tt.cmd)
			if tt.wantFixed && corrected == "" {
				t.Error("fixGoTestRaceCache() returned empty, want corrected command")
			}
			if !tt.wantFixed && corrected != "" {
				t.Errorf("fixGoTestRaceCache(%q) = %q, want empty", tt.cmd, corrected)
			}
			if tt.wantFixed && !strings.Contains(corrected, "-count=1") {
				t.Errorf("corrected = %q, want -count=1", corrected)
			}
		})
	}
}

func TestNarrowTestScope(t *testing.T) {
	t.Parallel()

	t.Run("no files", func(t *testing.T) {
		t.Parallel()
		id := "test-narrow-empty-" + strings.ReplaceAll(t.Name(), "/", "-")
		sdb, err := sessiondb.Open(id)
		if err != nil {
			t.Fatalf("sessiondb.Open() = %v", err)
		}
		t.Cleanup(func() { _ = sdb.Destroy() })

		suggested, _ := narrowTestScope(sdb, "go test ./...", "/project")
		if suggested != "" {
			t.Errorf("narrowTestScope(empty) = %q, want empty", suggested)
		}
	})

	t.Run("with files", func(t *testing.T) {
		t.Parallel()
		id := "test-narrow-files-" + strings.ReplaceAll(t.Name(), "/", "-")
		sdb, err := sessiondb.Open(id)
		if err != nil {
			t.Fatalf("sessiondb.Open() = %v", err)
		}
		t.Cleanup(func() { _ = sdb.Destroy() })

		_ = sdb.AddWorkingSetFile("/project/internal/store/store.go")
		suggested, ctx := narrowTestScope(sdb, "go test ./...", "/project")
		if suggested == "" {
			t.Error("narrowTestScope() returned empty, want narrowed command")
		}
		if !strings.Contains(suggested, "internal/store") {
			t.Errorf("suggested = %q, want to contain 'internal/store'", suggested)
		}
		if ctx == "" {
			t.Error("context should not be empty")
		}
	})

	t.Run("not go test", func(t *testing.T) {
		t.Parallel()
		id := "test-narrow-notgotest-" + strings.ReplaceAll(t.Name(), "/", "-")
		sdb, err := sessiondb.Open(id)
		if err != nil {
			t.Fatalf("sessiondb.Open() = %v", err)
		}
		t.Cleanup(func() { _ = sdb.Destroy() })

		suggested, _ := narrowTestScope(sdb, "npm test", "/project")
		if suggested != "" {
			t.Errorf("narrowTestScope(npm test) = %q, want empty", suggested)
		}
	})
}

func TestScopeGitAdd(t *testing.T) {
	t.Parallel()

	t.Run("git add dot", func(t *testing.T) {
		t.Parallel()
		id := "test-scope-dot-" + strings.ReplaceAll(t.Name(), "/", "-")
		sdb, err := sessiondb.Open(id)
		if err != nil {
			t.Fatalf("sessiondb.Open() = %v", err)
		}
		t.Cleanup(func() { _ = sdb.Destroy() })

		_ = sdb.AddWorkingSetFile("/project/main.go")
		_ = sdb.AddWorkingSetFile("/project/handler.go")

		suggested, ctx := scopeGitAdd(sdb, "git add .", "/project")
		if suggested == "" {
			t.Error("scopeGitAdd() returned empty, want scoped command")
		}
		if !strings.Contains(suggested, "main.go") {
			t.Errorf("suggested = %q, want to contain 'main.go'", suggested)
		}
		if ctx == "" {
			t.Error("context should not be empty")
		}
	})

	t.Run("not git add", func(t *testing.T) {
		t.Parallel()
		id := "test-scope-notgit-" + strings.ReplaceAll(t.Name(), "/", "-")
		sdb, err := sessiondb.Open(id)
		if err != nil {
			t.Fatalf("sessiondb.Open() = %v", err)
		}
		t.Cleanup(func() { _ = sdb.Destroy() })

		suggested, _ := scopeGitAdd(sdb, "git status", "/project")
		if suggested != "" {
			t.Errorf("scopeGitAdd(git status) = %q, want empty", suggested)
		}
	})

	t.Run("no files", func(t *testing.T) {
		t.Parallel()
		id := "test-scope-nofiles-" + strings.ReplaceAll(t.Name(), "/", "-")
		sdb, err := sessiondb.Open(id)
		if err != nil {
			t.Fatalf("sessiondb.Open() = %v", err)
		}
		t.Cleanup(func() { _ = sdb.Destroy() })

		suggested, _ := scopeGitAdd(sdb, "git add .", "/project")
		if suggested != "" {
			t.Errorf("scopeGitAdd(no files) = %q, want empty", suggested)
		}
	})
}

func TestContainsAny(t *testing.T) {
	t.Parallel()
	if !containsAny("hello world", "world", "foo") {
		t.Error("containsAny should match 'world'")
	}
	if containsAny("hello", "foo", "bar") {
		t.Error("containsAny should not match")
	}
	if containsAny("", "foo") {
		t.Error("containsAny should not match empty string")
	}
}
