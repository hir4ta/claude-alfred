package hookhandler

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
)

// codeHeuristic represents a single code quality check.
type codeHeuristic struct {
	Name     string
	Language string // file extension trigger (e.g. "go", "py"), "" for all languages
	Check    func(filePath, content string) string
}

var codeHeuristics = []codeHeuristic{
	{Name: "go_unchecked_error", Language: "go", Check: checkGoUncheckedError},
	{Name: "go_debug_print", Language: "go", Check: checkGoDebugPrint},
	{Name: "todo_without_ticket", Language: "", Check: checkTODOWithoutTicket},
	{Name: "py_bare_except", Language: "py", Check: checkPyBareExcept},
	{Name: "js_console_log", Language: "js", Check: checkJSConsoleLog},
	{Name: "hardcoded_secret", Language: "", Check: checkHardcodedSecret},
}

// runCodeHeuristics checks edited/written content against code quality patterns.
// Returns an observation string, or "" if no issues found.
func runCodeHeuristics(filePath string, toolInput json.RawMessage) string {
	ext := fileExtFromPath(filePath)
	content := extractWriteContent(toolInput)
	if content == "" {
		return ""
	}

	for _, h := range codeHeuristics {
		if h.Language != "" && h.Language != ext {
			continue
		}
		if suggestion := h.Check(filePath, content); suggestion != "" {
			return suggestion
		}
	}
	return ""
}

// --- Individual checks ---

var goUncheckedErrPattern = regexp.MustCompile(`_\s*(?:,\s*_\s*)?=\s*\w+\.?\w+\(`)

func checkGoUncheckedError(_, content string) string {
	if !goUncheckedErrPattern.MatchString(content) {
		return ""
	}
	return "Discarded error with `_ =` — consider handling or adding justification comment"
}

var goDebugPrintPattern = regexp.MustCompile(`\bfmt\.Print(ln|f)?\(`)

func checkGoDebugPrint(filePath, content string) string {
	if strings.HasSuffix(filePath, "_test.go") {
		return ""
	}
	if !goDebugPrintPattern.MatchString(content) {
		return ""
	}
	return "fmt.Println detected in non-test file — remove debug prints before committing"
}

var todoPattern = regexp.MustCompile(`(?i)\bTODO\b`)
var todoWithTicket = regexp.MustCompile(`(?i)\bTODO\s*[\(:]?\s*[A-Z]+-\d+`)

func checkTODOWithoutTicket(_, content string) string {
	if !todoPattern.MatchString(content) {
		return ""
	}
	if todoWithTicket.MatchString(content) {
		return ""
	}
	return "TODO without ticket reference — consider linking to an issue"
}

var pyBareExceptPattern = regexp.MustCompile(`\bexcept\s*:`)

func checkPyBareExcept(_, content string) string {
	if !pyBareExceptPattern.MatchString(content) {
		return ""
	}
	return "Bare `except:` catches all exceptions including KeyboardInterrupt — specify the exception type"
}

var jsConsoleLogPattern = regexp.MustCompile(`\bconsole\.log\(`)

func checkJSConsoleLog(filePath, content string) string {
	base := filepath.Base(filePath)
	if strings.Contains(base, ".test.") || strings.Contains(base, "_test.") || strings.Contains(base, ".spec.") {
		return ""
	}
	if !jsConsoleLogPattern.MatchString(content) {
		return ""
	}
	return "console.log detected — remove debug logs before committing"
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|secret|api_key|apikey|api_secret)\s*[:=]\s*["'][^"']{8,}`),
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-._~+/]{20,}`),
}

func checkHardcodedSecret(_, content string) string {
	for _, p := range secretPatterns {
		if p.MatchString(content) {
			return "Potential hardcoded secret detected — consider using environment variables"
		}
	}
	return ""
}

// --- Helpers ---

// extractWriteContent extracts the new content from Edit or Write tool input.
func extractWriteContent(toolInput json.RawMessage) string {
	var edit struct {
		NewString string `json:"new_string"`
	}
	if json.Unmarshal(toolInput, &edit) == nil && edit.NewString != "" {
		return edit.NewString
	}
	var write struct {
		Content string `json:"content"`
	}
	if json.Unmarshal(toolInput, &write) == nil && write.Content != "" {
		return write.Content
	}
	return ""
}

// fileExtFromPath returns the file extension without the dot.
func fileExtFromPath(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return ""
	}
	ext = ext[1:] // remove leading dot
	switch ext {
	case "tsx", "jsx":
		return "js"
	case "ts":
		return "js"
	}
	return ext
}
