package mcpserver

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/hir4ta/claude-alfred/internal/store"
)

func TestBuildDiagnosis_EditMismatch(t *testing.T) {
	t.Parallel()
	d := buildDiagnosis(nil, "old_string not found in file /src/main.go", "Edit", "/src/main.go")
	if d.FailureType != "edit_mismatch" {
		t.Errorf("FailureType = %q, want %q", d.FailureType, "edit_mismatch")
	}
	if d.Confidence < 0.5 {
		t.Errorf("Confidence = %v, want >= 0.5", d.Confidence)
	}
	if len(d.Actions) == 0 {
		t.Error("Actions should not be empty for edit_mismatch")
	}
}

func TestBuildDiagnosis_CompileError(t *testing.T) {
	t.Parallel()
	errorMsg := `# github.com/example/pkg
./main.go:42:10: undefined: FooBar`
	d := buildDiagnosis(nil, errorMsg, "Bash", "")
	if d.FailureType != "compile_error" {
		t.Errorf("FailureType = %q, want %q", d.FailureType, "compile_error")
	}
	if d.Location == "" {
		t.Error("Location should be extracted from compile error")
	}
	if !strings.Contains(d.RootCause, "FooBar") {
		t.Errorf("RootCause = %q, want to mention FooBar", d.RootCause)
	}
}

func TestBuildDiagnosis_TestFailure(t *testing.T) {
	t.Parallel()
	errorMsg := `--- FAIL: TestSomething (0.01s)
    thing_test.go:42: got 1, want 2
FAIL	github.com/example/pkg	0.015s`
	d := buildDiagnosis(nil, errorMsg, "Bash", "")
	if d.FailureType != "test_failure" {
		t.Errorf("FailureType = %q, want %q", d.FailureType, "test_failure")
	}
	if d.Confidence == 0 {
		t.Error("Confidence should not be 0")
	}
}

func TestBuildDiagnosis_WithCoChanges(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("store.Open() = %v", err)
	}
	defer st.Close()

	// Record co-changes across multiple sessions.
	for i := 0; i < 3; i++ {
		_ = st.RecordCoChanges([]string{"/src/main.go", "/src/handler.go"})
	}

	d := buildDiagnosis(st, "old_string not found", "Edit", "/src/main.go")
	if len(d.CoChangedWith) == 0 {
		t.Error("CoChangedWith should include handler.go")
	}
}

func TestBuildDiagnosis_FileNotFound(t *testing.T) {
	t.Parallel()
	d := buildDiagnosis(nil, "no such file or directory: /foo/bar.go", "Read", "/foo/bar.go")
	if d.FailureType != "file_not_found" {
		t.Errorf("FailureType = %q, want %q", d.FailureType, "file_not_found")
	}
	if d.Confidence < 0.9 {
		t.Errorf("Confidence = %v, want >= 0.9 for file_not_found", d.Confidence)
	}
}

func TestBuildDiagnosis_Permission(t *testing.T) {
	t.Parallel()
	d := buildDiagnosis(nil, "permission denied: /etc/passwd", "Write", "/etc/passwd")
	if d.FailureType != "permission" {
		t.Errorf("FailureType = %q, want %q", d.FailureType, "permission")
	}
}

func TestBuildDiagnosis_RuntimeError(t *testing.T) {
	t.Parallel()
	d := buildDiagnosis(nil, "command exited with code 127", "Bash", "")
	if d.FailureType != "runtime_error" {
		t.Errorf("FailureType = %q, want %q", d.FailureType, "runtime_error")
	}
}

func TestDiagnoseConsolidatedHandler_MissingError(t *testing.T) {
	t.Parallel()
	handler := diagnoseConsolidatedHandler(nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("diagnoseConsolidatedHandler() error = %v", err)
	}
	if !result.IsError {
		t.Error("diagnoseConsolidatedHandler() without error_output should return error")
	}
}

func TestDiagnoseConsolidatedHandler_WithError(t *testing.T) {
	t.Parallel()
	handler := diagnoseConsolidatedHandler(nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"error_output": "undefined: myFunc",
		"tool_name":    "Bash",
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("diagnoseConsolidatedHandler() error = %v", err)
	}
	if result.IsError {
		t.Errorf("diagnoseConsolidatedHandler() returned unexpected error: %v", result.Content)
	}
}

func TestExtractLocation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"go file", "main.go:42: undefined", "main.go:42"},
		{"python file", "File app.py:10 error", "app.py:10"},
		{"js file", "at server.js:15:3", "server.js:15"},
		{"no match", "just an error", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractLocation(tt.input)
			if got != tt.want {
				t.Errorf("extractLocation(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetermineRootCause(t *testing.T) {
	t.Parallel()
	tests := []struct {
		failureType string
		errorMsg    string
		wantMinConf float64
	}{
		{"edit_mismatch", "old_string not found", 0.9},
		{"file_not_found", "no such file", 0.95},
		{"permission", "permission denied", 0.95},
		{"compile_error", "undefined: foo", 0.9},
		{"compile_error", "some compile issue", 0.6},
		{"test_failure", "FAIL TestFoo", 0.7},
		{"runtime_error", "exit code 1", 0.4},
	}
	for _, tt := range tests {
		t.Run(tt.failureType, func(t *testing.T) {
			t.Parallel()
			cause, conf := determineRootCause(tt.failureType, tt.errorMsg, "")
			if cause == "" {
				t.Error("root cause should not be empty")
			}
			if conf < tt.wantMinConf {
				t.Errorf("confidence = %v, want >= %v", conf, tt.wantMinConf)
			}
		})
	}
}
