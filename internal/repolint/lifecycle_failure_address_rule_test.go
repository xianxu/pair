package repolint

import "go/ast"

// These three owners compose untrusted dispatch evidence with exact captured
// launch evidence. Ordinary record copying elsewhere is outside this guard.
func lifecycleFailureAddressOverrides(file *ast.File) []ast.Node {
	if file.Name.Name != "couchcore" {
		return nil
	}
	var matches []ast.Node
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !failureAddressCompositionOwner(fn) {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.AssignStmt:
				for i, lhs := range n.Lhs {
					field, ok := ast.Unparen(lhs).(*ast.SelectorExpr)
					if !ok {
						continue
					}
					if field.Sel.Name == "Address" {
						if len(n.Rhs) != len(n.Lhs) || !safeFailureAddress(n.Rhs[i]) {
							matches = append(matches, lhs)
						}
					} else if parent, ok := ast.Unparen(field.X).(*ast.SelectorExpr); ok && parent.Sel.Name == "Address" {
						matches = append(matches, lhs)
					}
				}
			case *ast.KeyValueExpr:
				if key, ok := n.Key.(*ast.Ident); ok && key.Name == "Address" && !safeFailureAddress(n.Value) {
					matches = append(matches, n)
				}
			}
			return true
		})
	}
	return matches
}

func failureAddressCompositionOwner(fn *ast.FuncDecl) bool {
	switch fn.Name.Name {
	case "lifecycleFailureTargetForCall", "lifecycleFailureForAttempt":
		return fn.Recv == nil
	case "ObserveCall":
		if fn.Recv == nil || len(fn.Recv.List) != 1 {
			return false
		}
		receiver := ast.Unparen(fn.Recv.List[0].Type)
		if pointer, ok := receiver.(*ast.StarExpr); ok {
			receiver = ast.Unparen(pointer.X)
		}
		name, ok := receiver.(*ast.Ident)
		return ok && name.Name == "OperationAttempt"
	}
	return false
}

func safeFailureAddress(expr ast.Expr) bool {
	expr = ast.Unparen(expr)
	if call, ok := expr.(*ast.CallExpr); ok {
		name, ok := ast.Unparen(call.Fun).(*ast.Ident)
		return ok && name.Name == "completeFailureAddress" && len(call.Args) == 2
	}
	return false
}
