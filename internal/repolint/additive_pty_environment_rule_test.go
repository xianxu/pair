package repolint

import "go/ast"

// An empty-base environment slice is not isolation when the launcher overlays
// it onto os.Environ. Pin the explicit process boundary for this Pair-specific
// additive API. Runtime poisoned-identity tests defend the actual child values.
func matchAdditivePTYEnvironment(file *ast.File) []ast.Node {
	var matches []ast.Node
	importsCall := func(call *ast.CallExpr, path, name string) bool {
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			return false
		}
		qualifier, ok := selector.X.(*ast.Ident)
		return ok && pairImportIs(file, qualifier.Name, path)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		var starts []*ast.CallExpr
		emptySlice := false
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if importsCall(call, "github.com/xianxu/pair/cmd/internal/testpair", "Environment") {
				emptySlice = true
			}
			if importsCall(call, "github.com/xianxu/pair/cmd/internal/ptychild", "Start") {
				starts = append(starts, call)
			}
			return true
		})
		if !emptySlice {
			continue
		}
		for _, start := range starts {
			cleared := false
			for _, statement := range fn.Body.List {
				if statement.Pos() >= start.Pos() {
					break
				}
				expression, ok := statement.(*ast.ExprStmt)
				if !ok {
					continue
				}
				call, ok := expression.X.(*ast.CallExpr)
				if ok && importsCall(call, "github.com/xianxu/pair/cmd/internal/testpair", "ApplyEnvironment") {
					cleared = true
				}
			}
			if !cleared {
				matches = append(matches, start)
			}
		}
	}
	return matches
}
