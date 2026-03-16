package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hir4ta/claude-alfred/internal/spec"
)

func TestParseDesignFileRefs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    map[string]string // path → component
	}{
		{
			name:    "empty content",
			content: "",
			want:    map[string]string{},
		},
		{
			name: "single component with file",
			content: `### Component: SpecDriftDetector
- **File**: ` + "`cmd/alfred/hooks_posttool.go`" + `
- **Responsibility**: Detect spec drift`,
			want: map[string]string{
				"cmd/alfred/hooks_posttool.go": "SpecDriftDetector",
			},
		},
		{
			name: "multiple components",
			content: `### Component: DriftDetector
- **File**: ` + "`cmd/alfred/hooks_posttool.go`" + `

### Component: ConventionAuditor
- **File**: ` + "`internal/mcpserver/handlers_recall.go`",
			want: map[string]string{
				"cmd/alfred/hooks_posttool.go":           "DriftDetector",
				"internal/mcpserver/handlers_recall.go": "ConventionAuditor",
			},
		},
		{
			name: "file without backticks",
			content: `### Component: Foo
- **File**: internal/store/docs.go`,
			want: map[string]string{
				"internal/store/docs.go": "Foo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			refs := make(map[string]string)
			parseDesignFileRefs(tt.content, refs)
			if len(refs) != len(tt.want) {
				t.Errorf("got %d refs, want %d: %v", len(refs), len(tt.want), refs)
				return
			}
			for path, comp := range tt.want {
				if refs[path] != comp {
					t.Errorf("refs[%q] = %q, want %q", path, refs[path], comp)
				}
			}
		})
	}
}

func TestParseTasksFileRefs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "empty content",
			content: "",
			want:    nil,
		},
		{
			name:    "single file reference",
			content: `  _Requirements: FR-1 | Files: cmd/alfred/hooks_posttool.go_`,
			want:    []string{"cmd/alfred/hooks_posttool.go"},
		},
		{
			name:    "multiple files",
			content: `  _Requirements: FR-7, FR-8 | Files: internal/tui/diff_viewer.go, internal/tui/keys.go, internal/tui/datasource.go_`,
			want:    []string{"internal/tui/diff_viewer.go", "internal/tui/keys.go", "internal/tui/datasource.go"},
		},
		{
			name:    "no files field",
			content: `  _Requirements: FR-1 | Risk: low_`,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			refs := make(map[string]string)
			parseTasksFileRefs(tt.content, refs)
			if len(refs) != len(tt.want) {
				t.Errorf("got %d refs, want %d", len(refs), len(tt.want))
				return
			}
			for _, path := range tt.want {
				if _, ok := refs[path]; !ok {
					t.Errorf("missing ref for %q", path)
				}
			}
		})
	}
}

func TestClassifyDriftSeverity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path        string
		isComponent bool
		want        string
	}{
		{"internal/store/docs_test.go", false, "info"},
		{"internal/store/docs.go", false, "warning"},
		{"cmd/alfred/hooks.go", true, "critical"},
		{"README.md", false, "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			got := classifyDriftSeverity(tt.path, tt.isComponent)
			if got != tt.want {
				t.Errorf("classifyDriftSeverity(%q, %v) = %q, want %q", tt.path, tt.isComponent, got, tt.want)
			}
		})
	}
}

func TestParseSpecFileRefs(t *testing.T) {
	t.Parallel()

	// Create a temporary spec directory with design.md and tasks.md.
	tmp := t.TempDir()
	taskSlug := "test-task"
	specsDir := filepath.Join(tmp, ".alfred", "specs", taskSlug)
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	designContent := `# Design: test-task
### Component: StoreExtension
- **File**: ` + "`internal/store/docs.go`" + `
### Component: DiffViewer
- **File**: ` + "`internal/tui/diff_viewer.go`"

	tasksContent := `# Tasks: test-task
- [ ] T-1.1 Implement X
  _Requirements: FR-1 | Files: cmd/alfred/hooks_posttool.go_
- [ ] T-1.2 Implement Y
  _Requirements: FR-2 | Files: internal/mcpserver/handlers_recall.go_`

	os.WriteFile(filepath.Join(specsDir, "design.md"), []byte(designContent), 0o644)
	os.WriteFile(filepath.Join(specsDir, "tasks.md"), []byte(tasksContent), 0o644)

	refs := parseSpecFileRefs(tmp, taskSlug)

	expectedPaths := []string{
		"internal/store/docs.go",
		"internal/tui/diff_viewer.go",
		"cmd/alfred/hooks_posttool.go",
		"internal/mcpserver/handlers_recall.go",
	}

	for _, path := range expectedPaths {
		if _, ok := refs[path]; !ok {
			t.Errorf("missing ref for %q in %v", path, refs)
		}
	}

	// Check component association.
	if refs["internal/store/docs.go"] != "StoreExtension" {
		t.Errorf("expected component StoreExtension for docs.go, got %q", refs["internal/store/docs.go"])
	}
}

func TestTryDetectSpecDrift_NoActiveTask(t *testing.T) {
	t.Parallel()
	// Should not panic when there is no active task.
	tmp := t.TempDir()
	ctx := context.Background()
	tryDetectSpecDrift(ctx, tmp, "git commit -m 'test'")
	// No crash = pass.
}

func TestTryDetectSpecDrift_NoGitCommit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Non-commit commands should be ignored.
	tryDetectSpecDrift(ctx, "/tmp", "go test ./...")
}

func TestLogDriftEvent(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	alfredDir := filepath.Join(tmp, ".alfred")
	os.MkdirAll(alfredDir, 0o755)

	logDriftEvent(tmp, driftActionSpec, "test-task", map[string]any{
		"type":     "spec-drift",
		"severity": "warning",
		"files":    []string{"new_file.go"},
	})

	entries, err := spec.ReadAuditLog(tmp, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Action != driftActionSpec {
		t.Errorf("action = %q, want %q", entries[0].Action, driftActionSpec)
	}
	if entries[0].Target != "test-task" {
		t.Errorf("target = %q, want test-task", entries[0].Target)
	}
}

func TestHighestSeverity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		files       []string
		isComponent bool
		want        string
	}{
		{"all test files", []string{"a_test.go", "b_test.go"}, false, "info"},
		{"mixed", []string{"a_test.go", "b.go"}, false, "warning"},
		{"component", []string{"a.go"}, true, "critical"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := highestSeverity(tt.files, tt.isComponent)
			if got != tt.want {
				t.Errorf("highestSeverity(%v, %v) = %q, want %q", tt.files, tt.isComponent, got, tt.want)
			}
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	t.Parallel()
	got := uniqueStrings([]string{"a", "b", "a", "c", "b"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("got[%d] = %q, want %q", i, got[i], v)
		}
	}
}
