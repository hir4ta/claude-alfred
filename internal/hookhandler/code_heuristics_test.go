package hookhandler

import (
	"encoding/json"
	"testing"
)

func TestCheckGoUncheckedError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"discarded error", `_ = sdb.SetContext("key", "val")`, true},
		{"handled error", `if err := sdb.SetContext("key", "val"); err != nil {`, false},
		{"comment about errors", `// we discard errors here`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := checkGoUncheckedError("test.go", tt.content)
			if (got != "") != tt.want {
				t.Errorf("checkGoUncheckedError(%q) = %q, wantMatch=%v", tt.content, got, tt.want)
			}
		})
	}
}

func TestCheckGoDebugPrint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		content  string
		want     bool
	}{
		{"println in source", "main.go", `fmt.Println("debug")`, true},
		{"printf in source", "handler.go", `fmt.Printf("val: %d", x)`, true},
		{"println in test", "main_test.go", `fmt.Println("debug")`, false},
		{"no debug", "main.go", `log.Info("starting")`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := checkGoDebugPrint(tt.filePath, tt.content)
			if (got != "") != tt.want {
				t.Errorf("checkGoDebugPrint(%q, %q) = %q, wantMatch=%v", tt.filePath, tt.content, got, tt.want)
			}
		})
	}
}

func TestCheckTODOWithoutTicket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"bare todo", "// TODO fix this later", true},
		{"todo with ticket", "// TODO(AUTH-123) fix this later", false},
		{"todo with colon ticket", "// TODO: AUTH-456 handle edge case", false},
		{"no todo", "// handle edge case", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := checkTODOWithoutTicket("test.go", tt.content)
			if (got != "") != tt.want {
				t.Errorf("checkTODOWithoutTicket(%q) = %q, wantMatch=%v", tt.content, got, tt.want)
			}
		})
	}
}

func TestCheckPyBareExcept(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"bare except", "except:", true},
		{"typed except", "except ValueError:", false},
		{"except as", "except Exception as e:", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := checkPyBareExcept("test.py", tt.content)
			if (got != "") != tt.want {
				t.Errorf("checkPyBareExcept(%q) = %q, wantMatch=%v", tt.content, got, tt.want)
			}
		})
	}
}

func TestCheckJSConsoleLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		content  string
		want     bool
	}{
		{"console.log in source", "app.js", `console.log("debug")`, true},
		{"console.log in test", "app.test.js", `console.log("debug")`, false},
		{"console.log in spec", "app.spec.ts", `console.log("debug")`, false},
		{"no console", "app.js", `logger.info("debug")`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := checkJSConsoleLog(tt.filePath, tt.content)
			if (got != "") != tt.want {
				t.Errorf("checkJSConsoleLog(%q, %q) = %q, wantMatch=%v", tt.filePath, tt.content, got, tt.want)
			}
		})
	}
}

func TestCheckHardcodedSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"password literal", `password = "my_super_secret_123"`, true},
		{"api key", `api_key: "sk-1234567890abcdef"`, true},
		{"bearer token", `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9`, true},
		{"env var", `password = os.Getenv("DB_PASSWORD")`, false},
		{"short value", `api_key = "short"`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := checkHardcodedSecret("config.go", tt.content)
			if (got != "") != tt.want {
				t.Errorf("checkHardcodedSecret(%q) = %q, wantMatch=%v", tt.content, got, tt.want)
			}
		})
	}
}

func TestExtractWriteContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input json.RawMessage
		want  string
	}{
		{
			"edit tool",
			json.RawMessage(`{"file_path":"/a.go","old_string":"foo","new_string":"bar"}`),
			"bar",
		},
		{
			"write tool",
			json.RawMessage(`{"file_path":"/a.go","content":"package main"}`),
			"package main",
		},
		{
			"empty",
			json.RawMessage(`{"file_path":"/a.go"}`),
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractWriteContent(tt.input)
			if got != tt.want {
				t.Errorf("extractWriteContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileExtFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{"/src/main.go", "go"},
		{"/src/app.py", "py"},
		{"/src/app.tsx", "js"},
		{"/src/app.ts", "js"},
		{"/src/app.jsx", "js"},
		{"/src/app.js", "js"},
		{"/Makefile", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			got := fileExtFromPath(tt.path)
			if got != tt.want {
				t.Errorf("fileExtFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestRunCodeHeuristics_Integration(t *testing.T) {
	t.Parallel()

	// Go file with unchecked error.
	input := json.RawMessage(`{"file_path":"/src/main.go","new_string":"_ = db.Close()"}`)
	got := runCodeHeuristics("/src/main.go", input)
	if got == "" {
		t.Error("runCodeHeuristics() should detect unchecked error in .go file")
	}

	// Python file should not trigger Go checks.
	input = json.RawMessage(`{"file_path":"/src/app.py","new_string":"_ = db.Close()"}`)
	got = runCodeHeuristics("/src/app.py", input)
	if got != "" {
		t.Errorf("runCodeHeuristics() should not trigger Go check for .py file, got: %q", got)
	}
}
