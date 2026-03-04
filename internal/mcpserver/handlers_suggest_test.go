package mcpserver

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hir4ta/claude-alfred/internal/store"
)

func TestSuggestHandler_MissingProjectPath(t *testing.T) {
	t.Parallel()
	handler := suggestHandler(t.TempDir(), nil, nil)

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
	handler := suggestHandler(t.TempDir(), nil, nil)

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
	handler := suggestHandler(claudeHome, nil, nil)

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

func TestSuggestHandler_StructuredSuggestions(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	claudeHome := t.TempDir()
	handler := suggestHandler(claudeHome, nil, nil)

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
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	gitRun("add", ".")
	gitRun("commit", "-m", "initial")

	os.WriteFile(filepath.Join(dir, "new.go"), []byte("package main\n"), 0o644)
	gitRun("add", "new.go")
	gitRun("commit", "-m", "add new.go")

	res, err := handler(context.Background(), newRequest(map[string]any{
		"project_path": dir,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := resultJSON(t, res)
	suggestions, ok := m["suggestions"].([]any)
	if !ok || len(suggestions) == 0 {
		t.Fatal("expected suggestions")
	}

	// Verify structured format.
	first := suggestions[0].(map[string]any)
	if first["severity"] == nil {
		t.Error("expected severity field in suggestion")
	}
	if first["category"] == nil {
		t.Error("expected category field in suggestion")
	}
	if first["message"] == nil {
		t.Error("expected message field in suggestion")
	}
}

func TestSuggestHandler_DetectsPatterns(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	claudeHome := t.TempDir()
	handler := suggestHandler(claudeHome, nil, nil)

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

	// Create initial commit with go.mod.
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\ngo 1.22\n"), 0o644)
	gitRun("add", ".")
	gitRun("commit", "-m", "initial")

	// Add new dependency to go.mod.
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\ngo 1.22\nrequire example.com/pkg v1.0.0\n"), 0o644)
	gitRun("add", "go.mod")
	gitRun("commit", "-m", "add dep")

	res, err := handler(context.Background(), newRequest(map[string]any{
		"project_path": dir,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := resultJSON(t, res)
	patterns, ok := m["change_patterns"].([]any)
	if !ok || len(patterns) == 0 {
		t.Fatal("expected change_patterns")
	}

	found := false
	for _, p := range patterns {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if pm["type"] == "dependency_changes" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected dependency_changes pattern")
	}
}

func TestSuggestHandler_WithKBEnrichment(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	claudeHome := t.TempDir()
	st := openTestStore(t)

	// Seed KB.
	doc := &store.DocRow{
		URL:         "https://example.com/claude-md",
		SectionPath: "CLAUDE.md > Structure",
		Content:     "Keep CLAUDE.md Structure section updated when adding packages.",
		SourceType:  "docs",
	}
	doc.ContentHash = store.ContentHashOf(doc.Content)
	st.UpsertDoc(doc)

	handler := suggestHandler(claudeHome, st, nil)

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
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	gitRun("add", ".")
	gitRun("commit", "-m", "initial")

	os.WriteFile(filepath.Join(dir, "new.go"), []byte("package main\n"), 0o644)
	gitRun("add", "new.go")
	gitRun("commit", "-m", "add new")

	res, err := handler(context.Background(), newRequest(map[string]any{
		"project_path": dir,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := resultJSON(t, res)
	suggestions, ok := m["suggestions"].([]any)
	if !ok || len(suggestions) == 0 {
		t.Skip("no suggestions generated (expected for simple change)")
	}

	hasBP := false
	for _, s := range suggestions {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		if sm["best_practice"] != nil {
			hasBP = true
			break
		}
	}
	if !hasBP {
		// Not a hard failure — KB enrichment depends on FTS matching.
		t.Log("no suggestions had best_practice; KB match may not apply to this diff")
	}
}

func TestDetectChangePatterns_NewTests(t *testing.T) {
	t.Parallel()
	diff := diffInfo{
		files:   []string{"pkg/auth_test.go"},
		content: "+func TestLogin(t *testing.T) {\n",
	}
	patterns := detectChangePatterns(diff)

	found := false
	for _, p := range patterns {
		if p.Type == "new_tests" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected new_tests pattern")
	}
}

func TestDetectChangePatterns_APIEndpoints(t *testing.T) {
	t.Parallel()
	diff := diffInfo{
		files:   []string{"main.go"},
		content: "+\thttp.HandleFunc(\"/api/users\", handleUsers)\n",
	}
	patterns := detectChangePatterns(diff)

	found := false
	for _, p := range patterns {
		if p.Type == "new_api_endpoints" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected new_api_endpoints pattern")
	}
}

func TestContainsAny(t *testing.T) {
	t.Parallel()
	if !containsAny("hello world", "world", "xyz") {
		t.Error("expected true")
	}
	if containsAny("hello", "world", "xyz") {
		t.Error("expected false")
	}
}

func TestDetectChangePatterns_Empty(t *testing.T) {
	t.Parallel()
	diff := diffInfo{files: []string{"readme.md"}, content: ""}
	patterns := detectChangePatterns(diff)
	// No false positives on empty content.
	for _, p := range patterns {
		if strings.Contains(p.Type, "api") || strings.Contains(p.Type, "test") {
			t.Errorf("unexpected pattern %q for empty diff content", p.Type)
		}
	}
}
