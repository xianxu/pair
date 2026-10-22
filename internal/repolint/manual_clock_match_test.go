package repolint

import "go/ast"

// The manual clock's unbuffered channel is its delivery acknowledgment. Scope
// this syntax guard to its constructor and after method, not real timers.
func bufferedManualClockTimers(file *ast.File) []ast.Node {
	var matches []ast.Node
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		owned := fn.Recv == nil && fn.Name.Name == "newManualClock"
		if fn.Recv != nil && fn.Name.Name == "after" {
			for _, receiver := range fn.Recv.List {
				typ := receiver.Type
				if ptr, ok := typ.(*ast.StarExpr); ok {
					typ = ptr.X
				}
				if id, ok := typ.(*ast.Ident); ok && id.Name == "manualClock" {
					owned = true
				}
			}
		}
		if !owned || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				return true
			}
			name, ok := call.Fun.(*ast.Ident)
			if !ok || name.Name != "make" {
				return true
			}
			ch, ok := call.Args[0].(*ast.ChanType)
			if !ok {
				return true
			}
			value, ok := ch.Value.(*ast.SelectorExpr)
			if !ok || value.Sel.Name != "Time" {
				return true
			}
			pkg, ok := value.X.(*ast.Ident)
			if !ok || pkg.Name != "time" {
				return true
			}
			if size, ok := call.Args[1].(*ast.BasicLit); !ok || size.Value != "0" {
				matches = append(matches, call)
			}
			return true
		})
	}
	return matches
}
