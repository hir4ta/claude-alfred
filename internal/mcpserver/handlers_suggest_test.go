package mcpserver

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSuggestHandler_MissingProjectPath(t *testing.T) {
	t.Parallel()
	handler := suggestHandler(t.TempDir())

	res, err := handler(context.Background(), newRequest(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for missing project_path")
	}
}

func TestSuggestHandler_NoGitRepo(t *testing.T) {
	t.Parallel()
	handler := suggestHandler(t.TempDir())

	res, err := handler(context.Background(), newRequest(map[string]any{
		"project_path": t.TempDir(),
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", resultText(t, res))
	}

	m := resultJSON(t, res)
	if summary, _ := m["summary"].(string); summary == "" {
		t.Error("expected summary for no-changes result")
	}
}

func TestSuggestHandler_WithGitChanges(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	claudeHome := t.TempDir()
	handler := suggestHandler(claudeHome)

	gitRun := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}

	gitRun("init")
	gitRun("config", "user.name", "test")
	gitRun("config", "user.email", "test@test.com")
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun("add", ".")
	gitRun("commit", "-m", "initial")

	// Add a new file and commit.
	if err := os.WriteFile(filepath.Join(dir, "new.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun("add", "new.go")
	gitRun("commit", "-m", "add new.go")

	res, err := handler(context.Background(), newRequest(map[string]any{
		"project_path": dir,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", resultText(t, res))
	}

	m := resultJSON(t, res)
	changedFiles, _ := m["changed_files"].(float64)
	if changedFiles == 0 {
		t.Error("expected changed_files > 0")
	}
	if _, ok := m["suggestions"]; !ok {
		t.Error("expected suggestions key")
	}
}
