package hookhandler

import (
	"strings"
	"testing"

	"github.com/hir4ta/claude-alfred/internal/sessiondb"
)

func openPromptTestDB(t *testing.T) *sessiondb.SessionDB {
	t.Helper()
	id := "test-prompt-" + strings.ReplaceAll(t.Name(), "/", "-")
	sdb, err := sessiondb.Open(id)
	if err != nil {
		t.Fatalf("sessiondb.Open(%q) = %v", id, err)
	}
	t.Cleanup(func() { _ = sdb.Destroy() })
	return sdb
}

func TestIncrementContextInt(t *testing.T) {
	t.Parallel()
	sdb := openPromptTestDB(t)

	// First increment: 0 → 1.
	v := incrementContextInt(sdb, "test_counter")
	if v != "1" {
		t.Errorf("incrementContextInt() = %q, want \"1\"", v)
	}
	_ = sdb.SetContext("test_counter", v)

	// Second increment: 1 → 2.
	v = incrementContextInt(sdb, "test_counter")
	if v != "2" {
		t.Errorf("incrementContextInt() = %q, want \"2\"", v)
	}
}

func TestBuildSessionContextSummary_MinParts(t *testing.T) {
	t.Parallel()
	sdb := openPromptTestDB(t)
	pc := newPromptContext(sdb)

	// Empty session → too few parts → empty summary.
	got := buildSessionContextSummary(pc)
	if got != "" {
		t.Errorf("buildSessionContextSummary(empty) = %q, want empty", got)
	}
}

func TestBuildSessionContextSummary_WithContext(t *testing.T) {
	t.Parallel()
	sdb := openPromptTestDB(t)

	_ = sdb.AddWorkingSetFile("/src/main.go")
	_ = sdb.AddWorkingSetFile("/src/handler.go")
	_ = sdb.SetContext("task_type", "bugfix")
	_ = sdb.SetWorkingSet("git_branch", "fix/auth")

	pc := newPromptContext(sdb)
	got := buildSessionContextSummary(pc)
	if got == "" {
		t.Fatal("buildSessionContextSummary() = empty, want context summary")
	}
	if !strings.Contains(got, "main.go") {
		t.Errorf("summary missing file name: %q", got)
	}
	if !strings.Contains(got, "fix/auth") {
		t.Errorf("summary missing branch: %q", got)
	}
}

func TestBuildSessionContextSummary_Cooldown(t *testing.T) {
	t.Parallel()
	sdb := openPromptTestDB(t)

	_ = sdb.AddWorkingSetFile("/src/main.go")
	_ = sdb.SetContext("task_type", "bugfix")
	_ = sdb.SetWorkingSet("git_branch", "main")

	pc1 := newPromptContext(sdb)
	first := buildSessionContextSummary(pc1)
	if first == "" {
		t.Fatal("first call should return summary")
	}

	// Second call should be suppressed by cooldown.
	pc2 := newPromptContext(sdb)
	second := buildSessionContextSummary(pc2)
	if second != "" {
		t.Errorf("second call should be suppressed by cooldown, got %q", second)
	}
}

func TestBuildSessionContextSummary_TestStatus(t *testing.T) {
	t.Parallel()

	t.Run("failing", func(t *testing.T) {
		t.Parallel()
		sdb := openPromptTestDB(t)
		_ = sdb.AddWorkingSetFile("/src/main.go")
		_ = sdb.SetContext("task_type", "bugfix")
		_ = sdb.SetContext("has_test_run", "true")
		_ = sdb.SetContext("last_test_passed", "false")

		pc := newPromptContext(sdb)
		got := buildSessionContextSummary(pc)
		if !strings.Contains(got, "FAILING") {
			t.Errorf("expected FAILING in summary: %q", got)
		}
	})

	t.Run("passing", func(t *testing.T) {
		t.Parallel()
		sdb := openPromptTestDB(t)
		_ = sdb.AddWorkingSetFile("/src/main.go")
		_ = sdb.SetContext("task_type", "bugfix")
		_ = sdb.SetContext("has_test_run", "true")
		_ = sdb.SetContext("last_test_passed", "true")

		pc := newPromptContext(sdb)
		got := buildSessionContextSummary(pc)
		if !strings.Contains(got, "passing") {
			t.Errorf("expected 'passing' in summary: %q", got)
		}
	})

	t.Run("not run", func(t *testing.T) {
		t.Parallel()
		sdb := openPromptTestDB(t)
		_ = sdb.AddWorkingSetFile("/src/main.go")
		_ = sdb.SetContext("task_type", "bugfix")

		pc := newPromptContext(sdb)
		got := buildSessionContextSummary(pc)
		if !strings.Contains(got, "not run") {
			t.Errorf("expected 'not run' in summary: %q", got)
		}
	})
}

func TestPromptContext_Caching(t *testing.T) {
	t.Parallel()
	sdb := openPromptTestDB(t)

	_ = sdb.AddWorkingSetFile("/src/main.go")

	pc := newPromptContext(sdb)

	// First call.
	files1 := pc.WorkingSetFiles()
	// Second call should return cached result.
	files2 := pc.WorkingSetFiles()

	if len(files1) != len(files2) {
		t.Errorf("cached files differ: %d vs %d", len(files1), len(files2))
	}
}
