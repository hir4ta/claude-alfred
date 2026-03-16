package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGoMod(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	goModPath := filepath.Join(tmp, "go.mod")

	content := `module github.com/example/myproject

go 1.25.0

require (
	github.com/foo/bar v1.2.3
	github.com/baz/qux v0.1.0
)

require (
	github.com/indirect/dep v0.0.1 // indirect
)
`
	if err := os.WriteFile(goModPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	module, deps, err := parseGoMod(goModPath)
	if err != nil {
		t.Fatalf("parseGoMod() error: %v", err)
	}

	if module != "github.com/example/myproject" {
		t.Errorf("parseGoMod() module = %q, want %q", module, "github.com/example/myproject")
	}

	if len(deps) != 3 {
		t.Errorf("parseGoMod() deps count = %d, want 3", len(deps))
	}

	wantDeps := []string{"github.com/foo/bar", "github.com/baz/qux", "github.com/indirect/dep"}
	for i, want := range wantDeps {
		if i >= len(deps) {
			break
		}
		if deps[i] != want {
			t.Errorf("parseGoMod() deps[%d] = %q, want %q", i, deps[i], want)
		}
	}
}

func TestParseGoModMissing(t *testing.T) {
	t.Parallel()

	_, _, err := parseGoMod("/nonexistent/go.mod")
	if err == nil {
		t.Error("parseGoMod() should fail for missing file")
	}
}

func TestParsePackageJSON(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	pkgPath := filepath.Join(tmp, "package.json")

	content := `{
  "name": "my-node-app",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.0",
    "lodash": "^4.17.21"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  }
}
`
	if err := os.WriteFile(pkgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	name, deps, err := parsePackageJSON(pkgPath)
	if err != nil {
		t.Fatalf("parsePackageJSON() error: %v", err)
	}

	if name != "my-node-app" {
		t.Errorf("parsePackageJSON() name = %q, want %q", name, "my-node-app")
	}

	if len(deps) != 3 {
		t.Errorf("parsePackageJSON() deps count = %d, want 3", len(deps))
	}
}

func TestParsePackageJSONMissing(t *testing.T) {
	t.Parallel()

	_, _, err := parsePackageJSON("/nonexistent/package.json")
	if err == nil {
		t.Error("parsePackageJSON() should fail for missing file")
	}
}

func TestScanTopDirs(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Create directories.
	for _, dir := range []string{"cmd", "internal", "pkg", ".git", "node_modules", "vendor"} {
		if err := os.MkdirAll(filepath.Join(tmp, dir), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}
	// Create a file (should be excluded).
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	dirs, err := scanTopDirs(tmp)
	if err != nil {
		t.Fatalf("scanTopDirs() error: %v", err)
	}

	// Should include cmd/, internal/, pkg/ but not .git, node_modules, vendor.
	expected := map[string]bool{"cmd/": true, "internal/": true, "pkg/": true}
	excluded := map[string]bool{".git/": true, "node_modules/": true, "vendor/": true}

	for _, d := range dirs {
		if excluded[d] {
			t.Errorf("scanTopDirs() should exclude %q", d)
		}
		delete(expected, d)
	}

	for d := range expected {
		t.Errorf("scanTopDirs() should include %q", d)
	}
}

func TestExtractCLAUDEMDConventions(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	claudePath := filepath.Join(tmp, "CLAUDE.md")

	content := `# My Project

## Stack
Go 1.25

## Rules
- Always use gofmt
- Error strings lowercase
- No trailing punctuation in errors

## Other Section
Some other content.
`
	if err := os.WriteFile(claudePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	conventions, err := extractCLAUDEMDConventions(claudePath)
	if err != nil {
		t.Fatalf("extractCLAUDEMDConventions() error: %v", err)
	}

	if len(conventions) != 3 {
		t.Errorf("extractCLAUDEMDConventions() returned %d conventions, want 3", len(conventions))
	}

	if len(conventions) > 0 && conventions[0] != "Always use gofmt" {
		t.Errorf("extractCLAUDEMDConventions()[0] = %q, want %q", conventions[0], "Always use gofmt")
	}
}

func TestExtractREADMEDescription(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	readmePath := filepath.Join(tmp, "README.md")

	content := `# My Cool Project

This is a project that does amazing things.

## Installation
...
`
	if err := os.WriteFile(readmePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	desc := extractREADMEDescription(readmePath)
	if desc != "This is a project that does amazing things." {
		t.Errorf("extractREADMEDescription() = %q, want %q", desc, "This is a project that does amazing things.")
	}
}

func TestExtractREADMEDescriptionWithBadges(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	readmePath := filepath.Join(tmp, "README.md")

	content := `# Project

![badge](https://example.com/badge.svg)
[![CI](https://example.com/ci.svg)](https://example.com/ci)

The actual description starts here.

## More
`
	if err := os.WriteFile(readmePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	desc := extractREADMEDescription(readmePath)
	if desc != "The actual description starts here." {
		t.Errorf("extractREADMEDescription() = %q, want %q", desc, "The actual description starts here.")
	}
}

func TestAnalyzeProject(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Create go.mod.
	goMod := `module github.com/test/myapp

go 1.25.0

require (
	github.com/gin-gonic/gin v1.9.0
)
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create README.md.
	readme := "# My App\n\nA web application.\n"
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create directories.
	os.MkdirAll(filepath.Join(tmp, "cmd"), 0o755)
	os.MkdirAll(filepath.Join(tmp, "internal"), 0o755)

	data, err := analyzeProject(tmp)
	if err != nil {
		t.Fatalf("analyzeProject() error: %v", err)
	}

	if data.ProjectName != "myapp" {
		t.Errorf("analyzeProject() ProjectName = %q, want %q", data.ProjectName, "myapp")
	}
	if data.Description != "A web application." {
		t.Errorf("analyzeProject() Description = %q, want %q", data.Description, "A web application.")
	}
	if data.TechStack != "Go 1.25.0" {
		t.Errorf("analyzeProject() TechStack = %q, want %q", data.TechStack, "Go 1.25.0")
	}
	if len(data.Dependencies) == 0 {
		t.Error("analyzeProject() should find dependencies")
	}
	if len(data.Packages) == 0 {
		t.Error("analyzeProject() should find packages")
	}
}
