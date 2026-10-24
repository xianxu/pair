package repolint

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"strconv"
)

// emptyFIFOHandoffs is a function-local syntax guard for FIFO test helpers.
// It pairs direct Mkfifo/OpenFile paths and tracks direct writer variables in
// each block. It does not follow aliases, reassignment through other handles,
// helper calls, or control flow. Conditional/nested writes do not prove payload;
// dynamically computed payloads are left for review, rather than called empty.
// The catalog owner limits this matcher to test files.
func emptyFIFOHandoffs(file *ast.File) []ast.Node {
	var matches []ast.Node
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		paths := map[string]bool{}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if _, closure := node.(*ast.FuncLit); closure {
				return false
			}
			call, ok := node.(*ast.CallExpr)
			if ok && len(call.Args) > 0 && (importedZellijReference(file, ast.Unparen(call.Fun), "golang.org/x/sys/unix", "Mkfifo") || importedZellijReference(file, ast.Unparen(call.Fun), "syscall", "Mkfifo")) {
				paths[fifoHandoffPath(call.Args[0])] = true
			}
			return true
		})
		if len(paths) == 0 {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if _, closure := node.(*ast.FuncLit); closure {
				return false
			}
			block, ok := node.(*ast.BlockStmt)
			if !ok {
				return true
			}
			writers := map[*ast.Object]bool{}
			for _, statement := range block.List {
				// Inspect only this statement's immediate events. A nested block has its
				// own scan; its payload cannot acknowledge an outer unconditional close.
				ast.Inspect(statement, func(node ast.Node) bool {
					switch n := node.(type) {
					case *ast.BlockStmt, *ast.FuncLit, *ast.GoStmt, *ast.DeferStmt:
						return false
					case *ast.AssignStmt:
						for _, lhs := range n.Lhs {
							if id, ok := lhs.(*ast.Ident); ok {
								delete(writers, id.Obj)
							}
						}
						if len(n.Lhs) == 0 || len(n.Rhs) != 1 {
							return true
						}
						id, ok := n.Lhs[0].(*ast.Ident)
						if !ok || id.Obj == nil {
							return true
						}
						call, ok := ast.Unparen(n.Rhs[0]).(*ast.CallExpr)
						if !ok || len(call.Args) < 2 || !importedZellijReference(file, ast.Unparen(call.Fun), "os", "OpenFile") {
							return true
						}
						if paths[fifoHandoffPath(call.Args[0])] && fifoHandoffFlag(file, call.Args[1], "O_WRONLY") && !fifoHandoffFlag(file, call.Args[1], "O_RDWR") {
							writers[id.Obj] = false
						}
					case *ast.CallExpr:
						method, ok := ast.Unparen(n.Fun).(*ast.SelectorExpr)
						if !ok {
							return true
						}
						id, ok := ast.Unparen(method.X).(*ast.Ident)
						if !ok || id.Obj == nil {
							return true
						}
						payload, tracked := writers[id.Obj]
						if !tracked {
							return true
						}
						switch method.Sel.Name {
						case "Write", "WriteString":
							if len(n.Args) == 1 && !fifoHandoffEmptyPayload(n.Args[0]) {
								writers[id.Obj] = true
							}
						case "Close":
							if !payload {
								matches = append(matches, n)
							}
							delete(writers, id.Obj)
						}
					}
					return true
				})
			}
			return true
		})
	}
	return matches
}

// Keep direct path syntax and parser-resolved local identities together so a
// same-spelling shadow in another block does not become the FIFO path.
func fifoHandoffPath(expr ast.Expr) string {
	var text bytes.Buffer
	_ = format.Node(&text, token.NewFileSet(), ast.Unparen(expr))
	ast.Inspect(expr, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && id.Obj != nil {
			fmt.Fprintf(&text, "|%p", id.Obj)
		}
		return true
	})
	return text.String()
}

func fifoHandoffFlag(file *ast.File, expr ast.Expr, flag string) bool {
	expr = ast.Unparen(expr)
	if binary, ok := expr.(*ast.BinaryExpr); ok && binary.Op == token.OR {
		return fifoHandoffFlag(file, binary.X, flag) || fifoHandoffFlag(file, binary.Y, flag)
	}
	return importedZellijReference(file, expr, "os", flag) || importedZellijReference(file, expr, "syscall", flag) || importedZellijReference(file, expr, "golang.org/x/sys/unix", flag)
}

// Only statically empty literals are classified empty. Unknown payloads may be
// nonempty and therefore fall outside this narrowly mechanical guard.
func fifoHandoffEmptyPayload(expr ast.Expr) bool {
	expr = ast.Unparen(expr)
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name == "nil" && id.Obj == nil
	}
	if literal, ok := expr.(*ast.BasicLit); ok && literal.Kind == token.STRING {
		text, err := strconv.Unquote(literal.Value)
		return err == nil && text == ""
	}
	if literal, ok := expr.(*ast.CompositeLit); ok {
		slice, ok := literal.Type.(*ast.ArrayType)
		if !ok || slice.Len != nil {
			return false
		}
		element, ok := slice.Elt.(*ast.Ident)
		return ok && (element.Name == "byte" || element.Name == "uint8") && len(literal.Elts) == 0
	}
	if call, ok := expr.(*ast.CallExpr); ok && len(call.Args) == 1 {
		slice, ok := call.Fun.(*ast.ArrayType)
		if !ok || slice.Len != nil {
			return false
		}
		element, ok := slice.Elt.(*ast.Ident)
		return ok && (element.Name == "byte" || element.Name == "uint8") && fifoHandoffEmptyPayload(call.Args[0])
	}
	return false
}
