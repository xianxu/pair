package couchtty

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Console code reaches Presenter.Input ONLY through its door (pair#265).
//
// The crash this pins was one call site, added with the #255 migration, that
// forwarded key release, focus and blur straight to the presenter without the
// panel check every neighbouring arm honours. The presenter answered correctly
// -- the panel holds no endpoint -- and the console turned that answer into a
// fatal exit. A second such site is a one-line edit away, so the rule is a test
// rather than a comment.
var presenterInputCallersAllowed = map[string]bool{
	"deliverPresenterInput": true, // the door: classifies the answer
}

func TestConsoleReachesPresenterInputOnlyThroughItsDoor(t *testing.T) {
	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var violations []string
	fset := token.NewFileSet()
	for _, path := range sources {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, path, raw, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || presenterInputCallersAllowed[fn.Name.Name] {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "Input" {
					return true
				}
				inner, ok := selector.X.(*ast.SelectorExpr)
				if !ok || inner.Sel.Name != "presenter" {
					return true
				}
				violations = append(violations, fset.Position(call.Pos()).String()+" in "+fn.Name.Name)
				return true
			})
		}
	}
	if len(violations) > 0 {
		t.Fatalf("Presenter.Input called outside its door:\n  %s", strings.Join(violations, "\n  "))
	}
}
