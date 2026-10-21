package repolint

import (
	"go/ast"
	"strconv"
	"strings"
)

// matchRemoteConsoleReset is a call-site/escape guard, not an adversarial Go
// sandbox. Same-UID arbitrary code is outside the remote-session boundary.
// Test exemptions require an actual testing entrypoint, not a Test-like name.
func matchRemoteConsoleReset(file *ast.File) []ast.Node {
	var matches []ast.Node
	for _, decl := range file.Decls {
		fn, _ := decl.(*ast.FuncDecl)
		if remoteResetTestEntrypoint(file, fn) {
			continue
		}
		owner := func(name, receiver string) bool {
			return fn != nil && fn.Name.Name == name && remoteResetReceiver(fn) == receiver
		}
		keyOwner := file.Name.Name == "couchtty" && owner("routeRemoteSessionKey", "Console") && remoteResetKeyParameter(fn)
		var stack []ast.Node
		ast.Inspect(decl, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			stack = append(stack, node)
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			allowed := true
			switch selector.Sel.Name {
			case "remoteConsoleReset":
				allowed = file.Name.Name == "couchtty" && owner("SetRemoteSession", "Console")
				if keyOwner {
					allowed = remoteResetUseStaysLocal(fn, selector)
				}
			case "routeRemoteSessionKey":
				allowed = file.Name.Name == "couchtty" && owner("routeInputEvent", "Console") && remoteResetInKeyCase(stack)
			case "SetRemoteSession":
				allowed = file.Name.Name == "couchcmd" && owner("configureRemoteSession", "messageService")
			case "NewSupervisor":
				qualifier, ok := selector.X.(*ast.Ident)
				if !ok || !pairImportIs(file, qualifier.Name, "github.com/xianxu/pair/cmd/internal/remotesession") {
					return true
				}
				allowed = file.Name.Name == "couchcmd" && owner("configureRemoteSession", "messageService")
			case "consoleReset":
				allowed = file.Name.Name == "remotesession" && (owner("consoleReset", "Supervisor") || owner("NewSupervisor", ""))
			default:
				return true
			}
			if !allowed {
				matches = append(matches, selector)
			}
			return true
		})
	}
	return matches
}

func pairImportIs(file *ast.File, qualifier, path string) bool {
	for _, imp := range file.Imports {
		value, err := strconv.Unquote(imp.Path.Value)
		if err != nil || value != path {
			continue
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if name == qualifier {
			return true
		}
	}
	return false
}
func remoteResetReceiver(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return ""
	}
	t := ast.Unparen(fn.Recv.List[0].Type)
	if pointer, ok := t.(*ast.StarExpr); ok {
		t = ast.Unparen(pointer.X)
	}
	if name, ok := t.(*ast.Ident); ok {
		return name.Name
	}
	return ""
}
func remoteResetKeyParameter(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}
	t, ok := fn.Type.Params.List[0].Type.(*ast.SelectorExpr)
	return ok && t.Sel.Name == "KeyPressEvent"
}
func remoteResetInKeyCase(stack []ast.Node) bool {
	for _, node := range stack {
		if clause, ok := node.(*ast.CaseClause); ok && len(clause.List) == 1 {
			if selector, ok := clause.List[0].(*ast.SelectorExpr); ok && selector.Sel.Name == "KeyPressEvent" {
				return true
			}
		}
	}
	return false
}
func remoteResetTestEntrypoint(file *ast.File, fn *ast.FuncDecl) bool {
	if fn == nil || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") || fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}
	pointer, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	selector, ok := pointer.X.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "T" {
		return false
	}
	qualifier, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil || path != "testing" {
			continue
		}
		name := "testing"
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if qualifier.Name == name {
			return true
		}
	}
	return false
}
func remoteResetUseStaysLocal(fn *ast.FuncDecl, selector *ast.SelectorExpr) bool {
	var alias *ast.Object
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if ok && len(assignment.Rhs) == 1 && assignment.Rhs[0] == selector && len(assignment.Lhs) == 1 {
			if name, ok := assignment.Lhs[0].(*ast.Ident); ok {
				alias = name.Obj
			}
		}
		return true
	})
	if alias == nil {
		return false
	}
	valid := true
	var stack []ast.Node
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		stack = append(stack, node)
		name, ok := node.(*ast.Ident)
		if !ok || name.Obj != alias {
			return true
		}
		if assignment, ok := alias.Decl.(*ast.AssignStmt); ok {
			for _, lhs := range assignment.Lhs {
				if lhs == name {
					return true
				}
			}
		}
		// A nil comparison is fine; a return, argument, asynchronous or nested
		// closure capture would let reset authority leave this host key event.
		parent := stack[len(stack)-2]
		if comparison, ok := parent.(*ast.BinaryExpr); ok {
			other := comparison.X
			if other == name {
				other = comparison.Y
			}
			if ident, ok := other.(*ast.Ident); ok && ident.Name == "nil" {
				return true
			}
		}
		call, ok := parent.(*ast.CallExpr)
		if !ok || call.Fun != name {
			valid = false
			return true
		}
		for _, ancestor := range stack {
			switch ancestor.(type) {
			case *ast.GoStmt, *ast.DeferStmt, *ast.FuncLit:
				valid = false
			}
		}
		return true
	})
	return valid
}
