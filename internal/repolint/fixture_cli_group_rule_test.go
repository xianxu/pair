package repolint

import "go/ast"

// This owner-specific syntax guard prevents dropping the mandatory observation
// when changing the fixture runner. Runtime completion-path tests prove ordering
// and behavior; this is not a control-flow or exec.Cmd alias analysis.
func fixtureCLIUnobservedGroups(file *ast.File) []ast.Node {
	if file.Name.Name != "agentfixture" {
		return nil
	}
	var findings []ast.Node
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != "runCLI" {
			continue
		}
		var waits []ast.Node
		observed := false
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch target := ast.Unparen(call.Fun).(type) {
			case *ast.SelectorExpr:
				if target.Sel.Name == "Wait" {
					waits = append(waits, call)
				}
			case *ast.Ident:
				if target.Name == "cliGroupError" {
					observed = true
				}
			}
			return true
		})
		if !observed {
			findings = append(findings, waits...)
		}
	}
	return findings
}
