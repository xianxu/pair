package repolint

import (
	"go/ast"
	"path"
	"strconv"
)

// This guard checks only direct helper calls with inline []string arguments and
// a literal executable after --. Variables, aliases, computed argv and shell
// programs are outside its syntax scope; it does not prove a producer's protocol.
func scribePTYPrintfCalls(file *ast.File) []ast.Node {
	if file.Name.Name != "scribecmd" {
		return nil
	}
	literal := func(expr ast.Expr) string {
		if lit, ok := ast.Unparen(expr).(*ast.BasicLit); ok {
			value, _ := strconv.Unquote(lit.Value)
			return value
		}
		return ""
	}
	var matches []ast.Node
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		name, ok := ast.Unparen(call.Fun).(*ast.Ident)
		if !ok || name.Name != "runWithPTY" && name.Name != "runWithPTYReply" {
			return true
		}
		args, ok := ast.Unparen(call.Args[1]).(*ast.CompositeLit)
		if !ok {
			return true
		}
		typ, ok := args.Type.(*ast.ArrayType)
		if !ok || typ.Len != nil {
			return true
		}
		elt, ok := typ.Elt.(*ast.Ident)
		if !ok || elt.Name != "string" {
			return true
		}
		for i, arg := range args.Elts {
			if literal(arg) == "--" && i+1 < len(args.Elts) {
				if path.Base(literal(args.Elts[i+1])) == "printf" {
					matches = append(matches, call)
				}
				break
			}
		}
		return true
	})
	return matches
}
