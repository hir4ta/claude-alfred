package hookhandler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// DetectChangedGoFunctions returns the names of Go functions whose bodies
// contain the edited region. For Edit operations, it searches the new_string
// content within the full file to identify which functions were touched.
// For Write operations, it returns all functions in the file.
func DetectChangedGoFunctions(filePath string, fullContent []byte, editSnippet string) []string {
	if filepath.Ext(filePath) != ".go" || strings.HasSuffix(filePath, "_test.go") {
		return nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, fullContent, 0)
	if err != nil {
		return nil
	}

	src := string(fullContent)

	// If no edit snippet, return all function names (Write: whole file replaced).
	if editSnippet == "" {
		return allFuncNames(file)
	}

	// Find the byte offset of the edit snippet in the full file.
	editStart := strings.Index(src, editSnippet)
	if editStart < 0 {
		// Snippet not found — may have been applied already. Return all funcs.
		return allFuncNames(file)
	}
	editEnd := editStart + len(editSnippet)

	// Find functions whose body spans overlap the edit region.
	var changed []string
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil {
			continue
		}

		funcStart := fset.Position(funcDecl.Pos()).Offset
		funcEnd := fset.Position(funcDecl.End()).Offset

		if funcStart <= editEnd && funcEnd >= editStart {
			name := funcName(funcDecl)
			if name != "" {
				changed = append(changed, name)
			}
		}
	}

	return changed
}

// funcName returns the qualified function name (receiver.Method or just FuncName).
func funcName(fn *ast.FuncDecl) string {
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recvType := receiverTypeName(fn.Recv.List[0].Type)
		if recvType != "" {
			return recvType + "." + fn.Name.Name
		}
	}
	return fn.Name.Name
}

// receiverTypeName extracts the type name from a receiver field.
func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// allFuncNames returns all function names in a file.
func allFuncNames(file *ast.File) []string {
	var names []string
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			name := funcName(fn)
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}
