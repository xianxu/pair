package repolint

import (
	"go/ast"
	"go/token"
	"strconv"
)

// This syntax guard follows direct constructors, lexical variable aliases and
// explicit local types/fields (including range elements). It is not a type
// checker: cross-file declarations, dot imports, method values, function-return
// aliases and arbitrary reassignment/control-flow analysis are outside its scope.
func discardedControlledZellijCloses(file *ast.File) []ast.Node {
	imports := map[string]bool{}
	for _, spec := range file.Imports {
		path, _ := strconv.Unquote(spec.Path.Value)
		if path == "github.com/xianxu/pair/cmd/internal/pairlifecycletest" {
			name := "pairlifecycletest"
			if spec.Name != nil {
				name = spec.Name.Name
			}
			imports[name] = true
		}
	}
	member := func(expr ast.Expr, name string) bool {
		switch n := ast.Unparen(expr).(type) {
		case *ast.SelectorExpr:
			pkg, ok := ast.Unparen(n.X).(*ast.Ident)
			return ok && pkg.Obj == nil && imports[pkg.Name] && n.Sel.Name == name
		case *ast.Ident:
			return file.Name.Name == "pairlifecycletest" && n.Name == name
		}
		return false
	}
	// Resolve local named types and pointers without importing packages.
	underlying := func(typ ast.Expr) ast.Expr {
		for depth := 0; typ != nil && depth < 16; depth++ {
			typ = ast.Unparen(typ)
			if ptr, ok := typ.(*ast.StarExpr); ok {
				typ = ptr.X
				continue
			}
			if id, ok := typ.(*ast.Ident); ok && id.Obj != nil {
				if decl, ok := id.Obj.Decl.(*ast.TypeSpec); ok {
					typ = decl.Type
					continue
				}
			}
			return typ
		}
		return nil
	}
	fixtureType := &ast.Ident{Name: "controlled-zellij-provenance"}
	rangeSources := map[*ast.Object]ast.Expr{}
	ast.Inspect(file, func(node ast.Node) bool {
		if loop, ok := node.(*ast.RangeStmt); ok && loop.Value != nil {
			if id, ok := loop.Value.(*ast.Ident); ok && id.Obj != nil && loop.Tok == token.DEFINE {
				rangeSources[id.Obj] = loop.X
			}
		}
		return true
	})
	var typeOf func(ast.Expr, map[*ast.Object]bool) ast.Expr
	typeOf = func(expr ast.Expr, seen map[*ast.Object]bool) ast.Expr {
		if expr == nil {
			return nil
		}
		switch n := ast.Unparen(expr).(type) {
		case *ast.CallExpr:
			if member(n.Fun, "StartControlledZellij") || member(n.Fun, "StartControlledZellijWithOptions") {
				return fixtureType
			}
		case *ast.CompositeLit:
			return n.Type
		case *ast.UnaryExpr:
			if n.Op == token.AND {
				return typeOf(n.X, seen)
			}
		case *ast.Ident:
			if n.Obj == nil || seen[n.Obj] {
				return nil
			}
			seen[n.Obj] = true
			defer delete(seen, n.Obj)
			if source := rangeSources[n.Obj]; source != nil {
				switch typ := underlying(typeOf(source, seen)).(type) {
				case *ast.ArrayType:
					return typ.Elt
				case *ast.MapType:
					return typ.Value
				}
			}
			switch decl := n.Obj.Decl.(type) {
			case *ast.Field:
				return decl.Type
			case *ast.ValueSpec:
				if decl.Type != nil {
					return decl.Type
				}
				for i, id := range decl.Names {
					if id.Obj == n.Obj && i < len(decl.Values) {
						return typeOf(decl.Values[i], seen)
					}
				}
			case *ast.AssignStmt:
				for i, lhs := range decl.Lhs {
					if id, ok := lhs.(*ast.Ident); ok && id.Obj == n.Obj && i < len(decl.Rhs) {
						return typeOf(decl.Rhs[i], seen)
					}
				}
			}
		case *ast.SelectorExpr:
			if typ, ok := underlying(typeOf(n.X, seen)).(*ast.StructType); ok {
				for _, field := range typ.Fields.List {
					for _, name := range field.Names {
						if name.Name == n.Sel.Name {
							return field.Type
						}
					}
				}
			}
		case *ast.IndexExpr:
			switch typ := underlying(typeOf(n.X, seen)).(type) {
			case *ast.ArrayType:
				return typ.Elt
			case *ast.MapType:
				return typ.Value
			}
		}
		return nil
	}
	closeCall := func(expr ast.Expr) *ast.CallExpr {
		call, ok := ast.Unparen(expr).(*ast.CallExpr)
		if !ok {
			return nil
		}
		sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Close" {
			return nil
		}
		typ := underlying(typeOf(sel.X, map[*ast.Object]bool{}))
		if typ == fixtureType || typ != nil && member(typ, "ControlledZellij") {
			return call
		}
		return nil
	}
	var matches []ast.Node
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.ExprStmt:
			if call := closeCall(n.X); call != nil {
				matches = append(matches, call)
			}
		case *ast.DeferStmt:
			if call := closeCall(n.Call); call != nil {
				matches = append(matches, call)
			}
		case *ast.GoStmt:
			if call := closeCall(n.Call); call != nil {
				matches = append(matches, call)
			}
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				if id, ok := lhs.(*ast.Ident); ok && id.Name == "_" && i < len(n.Rhs) {
					if call := closeCall(n.Rhs[i]); call != nil {
						matches = append(matches, call)
					}
				}
			}
		case *ast.IfStmt:
			assign, ok := n.Init.(*ast.AssignStmt)
			if !ok || len(assign.Rhs) != 1 || len(assign.Lhs) != 1 {
				break
			}
			call := closeCall(assign.Rhs[0])
			id, ok := assign.Lhs[0].(*ast.Ident)
			body := n.Body
			if cond, ok := ast.Unparen(n.Cond).(*ast.BinaryExpr); ok && cond.Op == token.EQL {
				// In the conventional err == nil guard, failure is the else arm.
				body, _ = n.Else.(*ast.BlockStmt)
			}
			if call != nil && ok && !controlledCloseErrorHandled(body, id) {
				matches = append(matches, call)
			}
		}
		return true
	})
	return matches
}

// A checked cleanup must affect the test result or propagate the bound error.
// This checks syntax, not whether every nested execution path reaches the action.
// Testing failure methods are recognized on conventional t/tb identifiers.
func controlledCloseErrorHandled(body *ast.BlockStmt, bound *ast.Ident) bool {
	if body == nil {
		return false
	}
	referencesError := func(node ast.Node) bool {
		found := false
		ast.Inspect(node, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Obj == bound.Obj {
				found = true
			}
			return true
		})
		return found
	}
	handled := false
	ast.Inspect(body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.ReturnStmt:
			for _, result := range n.Results {
				if referencesError(result) {
					handled = true
				}
			}
		case *ast.AssignStmt:
			for i, rhs := range n.Rhs {
				if i >= len(n.Lhs) {
					continue
				}
				if id, ok := n.Lhs[i].(*ast.Ident); ok && id.Name == "_" {
					continue
				}
				if call, ok := ast.Unparen(rhs).(*ast.CallExpr); ok {
					if sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok && sel.Sel.Name == "Join" && referencesError(call) {
						pkg, ok := ast.Unparen(sel.X).(*ast.Ident)
						if !ok || pkg.Name != "errors" {
							continue
						}
						handled = true
					}
				}
			}
		case *ast.CallExpr:
			if sel, ok := ast.Unparen(n.Fun).(*ast.SelectorExpr); ok {
				if receiver, ok := ast.Unparen(sel.X).(*ast.Ident); ok && (receiver.Name == "t" || receiver.Name == "tb") {
					switch sel.Sel.Name {
					case "Error", "Errorf", "Fatal", "Fatalf", "Fail", "FailNow":
						handled = true
					}
				}
			}
		}
		return true
	})
	return handled
}
