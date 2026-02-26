package hookhandler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

// goFixer generates patches for Go code quality findings using go/ast.
type goFixer struct{}

func (g *goFixer) Fix(finding Finding, content []byte) *CodeFix {
	switch {
	case finding.Rule == "go_defer_in_loop" || strings.Contains(finding.Message, "defer` inside loop"):
		return g.fixDeferInLoop(finding, content)
	case finding.Rule == "go_nil_error_wrap" || strings.Contains(finding.Message, "wrapping nil"):
		return g.fixNilErrorWrap(finding, content)
	case finding.Rule == "go_empty_error_return" || strings.Contains(finding.Message, "swallows the error"):
		return g.fixEmptyErrorReturn(finding, content)
	case strings.Contains(finding.Message, "Error variable shadowed"):
		return g.fixErrorShadow(finding, content)
	}
	return nil
}

// fixDeferInLoop wraps a defer inside a loop with an immediately-invoked closure.
// Before: for ... { defer f.Close() }
// After:  for ... { func() { defer f.Close() }() }
func (g *goFixer) fixDeferInLoop(finding Finding, content []byte) *CodeFix {
	src := string(content)

	// Find defer statements inside for loops using AST.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, finding.File, src, 0)
	if err != nil {
		return nil
	}

	var fix *CodeFix
	ast.Inspect(file, func(n ast.Node) bool {
		if fix != nil {
			return false
		}
		forStmt, isFor := n.(*ast.ForStmt)
		rangeStmt, isRange := n.(*ast.RangeStmt)

		var body *ast.BlockStmt
		if isFor {
			body = forStmt.Body
		} else if isRange {
			body = rangeStmt.Body
		} else {
			return true
		}
		if body == nil {
			return true
		}

		for _, stmt := range body.List {
			deferStmt, ok := stmt.(*ast.DeferStmt)
			if !ok {
				continue
			}
			start := fset.Position(deferStmt.Pos()).Offset
			end := fset.Position(deferStmt.End()).Offset
			if start < 0 || end > len(src) {
				continue
			}

			before := src[start:end]
			// Preserve the defer call, wrap in closure.
			after := "func() { " + before + " }()"

			fix = &CodeFix{
				Finding:     finding,
				Before:      before,
				After:       after,
				Confidence:  0.9,
				Explanation: "Wrap defer in closure so it executes per iteration, not at function exit",
			}
			return false
		}
		return true
	})
	return fix
}

// fixNilErrorWrap removes %w wrapping of nil errors.
// Before: fmt.Errorf("failed: %w", nil)
// After:  fmt.Errorf("failed")
var nilWrapFixPattern = regexp.MustCompile(`(fmt\.Errorf\([^)]*?):\s*%w([^)]*?),\s*nil\s*\)`)

func (g *goFixer) fixNilErrorWrap(finding Finding, content []byte) *CodeFix {
	src := string(content)
	loc := nilWrapFixPattern.FindStringIndex(src)
	if loc == nil {
		return nil
	}
	before := src[loc[0]:loc[1]]
	after := nilWrapFixPattern.ReplaceAllString(before, `${1}${2})`)

	return &CodeFix{
		Finding:     finding,
		Before:      before,
		After:       after,
		Confidence:  0.95,
		Explanation: "Remove %w wrapping of nil — fmt.Errorf with %w and nil creates a non-nil error containing nil",
	}
}

// fixEmptyErrorReturn changes `return nil` to `return err` inside `if err != nil`.
// Before: if err != nil { return nil }
// After:  if err != nil { return err }
var emptyErrReturnFixPattern = regexp.MustCompile(`(if\s+err\s*!=\s*nil\s*\{\s*return\s+)nil(\s*\})`)

func (g *goFixer) fixEmptyErrorReturn(finding Finding, content []byte) *CodeFix {
	src := string(content)
	loc := emptyErrReturnFixPattern.FindStringIndex(src)
	if loc == nil {
		return nil
	}
	before := src[loc[0]:loc[1]]
	after := emptyErrReturnFixPattern.ReplaceAllString(before, `${1}err${2}`)

	return &CodeFix{
		Finding:     finding,
		Before:      before,
		After:       after,
		Confidence:  0.85,
		Explanation: "Return the error instead of swallowing it — callers need to know about failures",
	}
}

// fixErrorShadow changes `:=` to `=` for `err` re-declarations inside if err != nil blocks.
// Before: result, err := doSomething()
// After:  result, err = doSomething()
func (g *goFixer) fixErrorShadow(finding Finding, content []byte) *CodeFix {
	if finding.Line <= 0 {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	if finding.Line > len(lines) {
		return nil
	}
	line := lines[finding.Line-1]

	// Only fix if the line has `:=` with `err` on the left side.
	if !strings.Contains(line, ":=") || !strings.Contains(line, "err") {
		return nil
	}

	before := strings.TrimSpace(line)
	after := strings.Replace(before, ":=", "=", 1)

	return &CodeFix{
		Finding:     finding,
		Before:      before,
		After:       after,
		Confidence:  0.7, // lower confidence — may need var declaration elsewhere
		Explanation: "Use `=` instead of `:=` to avoid shadowing the outer `err` variable",
	}
}
