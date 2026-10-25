package repolint

import (
	"go/ast"
	"go/token"
	"path"
	"strings"
)

const moduleCacheHelperImport = "github.com/xianxu/pair/cmd/internal/testpair"

type moduleCachePath struct {
	allocation ast.Expr
	suffix     string
	managed    bool
}

// unmanagedModuleCaches is a function-local syntax guard, not a type checker or
// control-flow proof. It follows direct local initializers and literal Join
// descendants of testing.T/B.TempDir, os.MkdirTemp and CacheTempDir. It checks
// literal environment concatenations, Setenv calls and keyed GOMODCACHE fields.
// Only deferred or inline testing cleanup callbacks establish explicit removal;
// arbitrary helpers, computed paths, cross-function aliases and reassignment
// are outside its scope. Parser object identity keeps same-named shadows apart.
func unmanagedModuleCaches(file *ast.File) []ast.Node {
	var matches []ast.Node
	var bodies []*ast.BlockStmt
	ast.Inspect(file, func(node ast.Node) bool {
		switch fn := node.(type) {
		case *ast.FuncDecl:
			if fn.Body != nil {
				bodies = append(bodies, fn.Body)
			}
		case *ast.FuncLit:
			bodies = append(bodies, fn.Body)
		}
		return true
	})
	for _, body := range bodies {
		var resolve func(ast.Expr, map[*ast.Object]bool) moduleCachePath
		resolve = func(expr ast.Expr, seen map[*ast.Object]bool) moduleCachePath {
			if expr == nil {
				return moduleCachePath{}
			}
			switch n := ast.Unparen(expr).(type) {
			case *ast.Ident:
				if n.Obj == nil || seen[n.Obj] {
					return moduleCachePath{}
				}
				seen[n.Obj] = true
				switch definition := n.Obj.Decl.(type) {
				case *ast.AssignStmt:
					for i, lhs := range definition.Lhs {
						id, ok := lhs.(*ast.Ident)
						if ok && id.Obj == n.Obj && (len(definition.Rhs) == len(definition.Lhs) || i == 0 && len(definition.Rhs) == 1) {
							return resolve(definition.Rhs[i], seen)
						}
					}
				case *ast.ValueSpec:
					for i, id := range definition.Names {
						if id.Obj == n.Obj && i < len(definition.Values) {
							return resolve(definition.Values[i], seen)
						}
					}
				}
			case *ast.CallExpr:
				if importedZellijReference(file, ast.Unparen(n.Fun), moduleCacheHelperImport, "CacheTempDir") {
					return moduleCachePath{allocation: n, managed: true}
				}
				if importedZellijReference(file, ast.Unparen(n.Fun), "os", "MkdirTemp") || moduleCacheTestingMethod(file, n, "TempDir") {
					return moduleCachePath{allocation: n}
				}
				if importedZellijReference(file, ast.Unparen(n.Fun), "path/filepath", "Join") && len(n.Args) > 0 {
					root := resolve(n.Args[0], seen)
					for _, arg := range n.Args[1:] {
						segment := zellijString(arg)
						if segment == "" || strings.HasPrefix(segment, "/") || segment == ".." || strings.HasPrefix(segment, "../") {
							return moduleCachePath{}
						}
						root.suffix = path.Join(root.suffix, segment)
					}
					return root
				}
			}
			return moduleCachePath{}
		}
		rootOf := func(expr ast.Expr) moduleCachePath { return resolve(expr, map[*ast.Object]bool{}) }
		var cleanupRoots []moduleCachePath
		var rawRemovals []*ast.CallExpr
		collectCleanup := func(node ast.Node) {
			ast.Inspect(node, func(node ast.Node) bool {
				if _, nested := node.(*ast.FuncLit); nested {
					return false
				}
				if call, ok := node.(*ast.CallExpr); ok && len(call.Args) == 1 && importedZellijReference(file, ast.Unparen(call.Fun), moduleCacheHelperImport, "RemoveCacheTree") {
					cleanupRoots = append(cleanupRoots, rootOf(call.Args[0]))
				}
				if call, ok := node.(*ast.CallExpr); ok && len(call.Args) == 1 && importedZellijReference(file, ast.Unparen(call.Fun), "os", "RemoveAll") {
					rawRemovals = append(rawRemovals, call)
				}
				return true
			})
		}
		ast.Inspect(body, func(node ast.Node) bool {
			if _, nested := node.(*ast.FuncLit); nested {
				return false
			}
			if deferred, ok := node.(*ast.DeferStmt); ok {
				if closure, ok := ast.Unparen(deferred.Call.Fun).(*ast.FuncLit); ok {
					collectCleanup(closure.Body)
				} else {
					collectCleanup(deferred.Call)
				}
			}
			if call, ok := node.(*ast.CallExpr); ok && moduleCacheTestingMethod(file, call, "Cleanup") && len(call.Args) == 1 {
				if closure, ok := ast.Unparen(call.Args[0]).(*ast.FuncLit); ok {
					collectCleanup(closure.Body)
				}
			}
			return true
		})
		owned := func(root moduleCachePath) bool {
			if root.managed {
				return true
			}
			for _, cleanup := range cleanupRoots {
				if root.allocation != nil && root.allocation == cleanup.allocation && (cleanup.suffix == "" || root.suffix == cleanup.suffix || strings.HasPrefix(root.suffix, cleanup.suffix+"/")) {
					return true
				}
			}
			return false
		}
		var homes []moduleCachePath
		ast.Inspect(body, func(node ast.Node) bool {
			if _, nested := node.(*ast.FuncLit); nested {
				return false
			}
			var cache ast.Expr
			switch n := node.(type) {
			case *ast.BinaryExpr:
				if n.Op == token.ADD && zellijString(n.X) == "GOMODCACHE=" {
					cache = n.Y
				}
			case *ast.KeyValueExpr:
				if id, ok := n.Key.(*ast.Ident); ok && id.Name == "GOMODCACHE" {
					cache = n.Value
				}
			case *ast.CallExpr:
				if len(n.Args) == 2 && (moduleCacheTestingMethod(file, n, "Setenv") || importedZellijReference(file, ast.Unparen(n.Fun), "os", "Setenv")) {
					switch zellijString(n.Args[0]) {
					case "GOMODCACHE":
						cache = n.Args[1]
					case "HOME":
						homes = append(homes, rootOf(n.Args[1]))
					}
				}
			}
			if cache != nil {
				if root := rootOf(cache); root.allocation != nil && !owned(root) {
					matches = append(matches, node)
				}
			}
			return true
		})
		ast.Inspect(body, func(node ast.Node) bool {
			if _, nested := node.(*ast.FuncLit); nested {
				return false
			}
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 || !importedZellijReference(file, ast.Unparen(call.Fun), "os", "RemoveAll") {
				return true
			}
			rawRemovals = append(rawRemovals, call)
			return true
		})
		seenRemovals := map[*ast.CallExpr]bool{}
		for _, call := range rawRemovals {
			if seenRemovals[call] {
				continue
			}
			seenRemovals[call] = true
			root := rootOf(call.Args[0])
			for _, home := range homes {
				if root.allocation != nil && root.allocation == home.allocation && root.suffix == home.suffix {
					matches = append(matches, call)
					break
				}
			}
		}
	}
	return matches
}

func moduleCacheTestingMethod(file *ast.File, call *ast.CallExpr, method string) bool {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != method {
		return false
	}
	id, ok := ast.Unparen(selector.X).(*ast.Ident)
	if !ok || id.Obj == nil {
		return false
	}
	field, ok := id.Obj.Decl.(*ast.Field)
	if !ok {
		return false
	}
	typ := ast.Unparen(field.Type)
	if pointer, ok := typ.(*ast.StarExpr); ok {
		typ = ast.Unparen(pointer.X)
	}
	return importedZellijReference(file, typ, "testing", "T") || importedZellijReference(file, typ, "testing", "B") || importedZellijReference(file, typ, "testing", "TB")
}
