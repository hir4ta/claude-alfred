package hookhandler

import (
	"regexp"
	"strings"
)

// multiAnalyzer dispatches to language-specific CodeAnalyzers.
type multiAnalyzer struct {
	analyzers map[string]CodeAnalyzer
}

// NewMultiAnalyzer creates a CodeAnalyzer that delegates to per-language analyzers.
// Go uses go/ast; Python, JS/TS, and Rust use gotreesitter (pure Go tree-sitter).
func NewMultiAnalyzer() CodeAnalyzer {
	ts := NewTreeSitterAnalyzer()
	return &multiAnalyzer{
		analyzers: map[string]CodeAnalyzer{
			"go":  NewGoAnalyzer(),
			"py":  ts,
			"js":  ts,
			"ts":  ts,
			"tsx": ts,
			"jsx": ts,
			"rs":  ts,
		},
	}
}

func (m *multiAnalyzer) Analyze(filePath string, content []byte) []Finding {
	ext := rawFileExt(filePath)
	if a, ok := m.analyzers[ext]; ok {
		return a.Analyze(filePath, content)
	}
	// Fallback: try normalized extension (e.g. "rust" → "rs").
	norm := fileExtFromPath(filePath)
	if norm != ext {
		if a, ok := m.analyzers[norm]; ok {
			return a.Analyze(filePath, content)
		}
	}
	return nil
}

func (m *multiAnalyzer) SupportedLanguages() []string {
	langs := make([]string, 0, len(m.analyzers))
	for lang := range m.analyzers {
		langs = append(langs, lang)
	}
	return langs
}

// Rust patterns used by heuristic check functions in code_heuristics.go.
var (
	rsUnwrapPattern = regexp.MustCompile(`\.unwrap\(\)`)
	rsTodoPattern   = regexp.MustCompile(`\btodo!\s*\(`)
	rsUnsafePattern = regexp.MustCompile(`\bunsafe\s*\{`)
)

// --- Rust heuristic check functions (for codeHeuristics table) ---

func checkRustUnwrap(filePath, content string) string {
	if strings.Contains(content, "#[cfg(test)]") || strings.HasSuffix(filePath, "_test.rs") {
		return ""
	}
	if !rsUnwrapPattern.MatchString(content) {
		return ""
	}
	return "`.unwrap()` on Result/Option — use `?` operator or handle the error explicitly"
}

func checkRustTodoMacro(filePath, content string) string {
	if strings.Contains(content, "#[cfg(test)]") || strings.HasSuffix(filePath, "_test.rs") {
		return ""
	}
	if !rsTodoPattern.MatchString(content) {
		return ""
	}
	return "`todo!()` macro in non-test code — will panic at runtime"
}

func checkRustUnsafeNoComment(_, content string) string {
	locs := rsUnsafePattern.FindAllStringIndex(content, -1)
	for _, loc := range locs {
		start := max(0, loc[0]-100)
		nearby := content[start:loc[0]]
		if !strings.Contains(strings.ToUpper(nearby), "SAFETY") {
			return "`unsafe` block without `// SAFETY:` comment — document the invariants"
		}
	}
	return ""
}
