package repolint

import (
	"go/ast"
	"strings"
)

func terminalRegistryOwnerCalls(file *ast.File) []ast.Node {
	switch file.Name.Name {
	case "termcmd", "layoutcmd", "clipcmd", "couchcmd":
	default:
		return nil
	}
	var matches []ast.Node
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || strings.HasPrefix(fn.Name.Name, "Test") || strings.HasPrefix(fn.Name.Name, "Benchmark") || strings.HasPrefix(fn.Name.Name, "Fuzz") {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
			if ok && selector.Sel.Name == "LiveIDs" {
				matches = append(matches, call)
			}
			return true
		})
	}
	return matches
}
