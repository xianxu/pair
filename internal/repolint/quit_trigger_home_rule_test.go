package repolint

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// This syntax guard covers direct quit-trigger calls in Couch tests, including
// nested subtests. Helper methods inherit fixture isolation from their callers;
// this rule deliberately does not attempt interprocedural or type analysis.
func quitTriggersWithoutPrivateHome(file *ast.File) []ast.Node {
	if file.Name.Name != "couchcore" {
		return nil
	}
	var matches []ast.Node
	var block func(*ast.BlockStmt, bool, map[string]bool)
	block = func(body *ast.BlockStmt, isolated bool, inherited map[string]bool) {
		if body == nil {
			return
		}
		private := map[string]bool{}
		for name, ok := range inherited {
			private[name] = ok
		}
		isTemp := func(expr ast.Expr) bool {
			if name, ok := ast.Unparen(expr).(*ast.Ident); ok {
				return private[name.Name]
			}
			call, ok := ast.Unparen(expr).(*ast.CallExpr)
			if !ok {
				return false
			}
			sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
			return ok && sel.Sel.Name == "TempDir" && len(call.Args) == 0
		}
		for _, stmt := range body.List {
			ast.Inspect(stmt, func(node ast.Node) bool {
				if nested, ok := node.(*ast.BlockStmt); ok {
					block(nested, isolated, private)
					return false
				}
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
				if ok && (sel.Sel.Name == "TriggerQuit" || sel.Sel.Name == "TriggerBoundQuit") && !isolated {
					matches = append(matches, call)
				}
				return true
			})
			// Only unconditional statements in this lexical block establish HOME for
			// subsequent statements. A child's setup never establishes its parent's.
			if expr, ok := stmt.(*ast.ExprStmt); ok {
				if call, ok := ast.Unparen(expr.X).(*ast.CallExpr); ok && len(call.Args) == 2 {
					if sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok && sel.Sel.Name == "Setenv" {
						if key, ok := call.Args[0].(*ast.BasicLit); ok && key.Kind == token.STRING {
							name, _ := strconv.Unquote(key.Value)
							if name == "HOME" {
								isolated = isTemp(call.Args[1])
							}
						}
					}
				}
			}
			if assign, ok := stmt.(*ast.AssignStmt); ok {
				for i, lhs := range assign.Lhs {
					if name, ok := lhs.(*ast.Ident); ok {
						private[name.Name] = len(assign.Rhs) == len(assign.Lhs) && isTemp(assign.Rhs[i])
					}
				}
				// sandboxedChecker is the existing in-process fixture boundary: it sets
				// private HOME itself before returning its checker. Dynamic helpers are
				// intentionally not guessed from their names or return values.
				for _, rhs := range assign.Rhs {
					if call, ok := ast.Unparen(rhs).(*ast.CallExpr); ok {
						if name, ok := call.Fun.(*ast.Ident); ok && name.Name == "sandboxedChecker" {
							isolated = true
						}
					}
				}
			}
		}
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") {
			block(fn.Body, false, nil)
		}
	}
	return matches
}
