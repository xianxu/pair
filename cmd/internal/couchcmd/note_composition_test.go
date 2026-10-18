package couchcmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// Hosted frames have no native tab provenance. Production composition must
// not attach the old asynchronous native geometry adapter to screen notes.
func TestCouchCompositionHasNoNativeNoteAdapter(t *testing.T) {
	source, err := parser.ParseFile(token.NewFileSet(), "run.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, decl := range source.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "wireResolver" {
			continue
		}
		found = true
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if method, ok := call.Fun.(*ast.SelectorExpr); ok && method.Sel.Name == "SetPaneRects" {
				t.Error("production composition still installs native note attribution")
			}
			return true
		})
	}
	if !found {
		t.Fatal("production composition moved; update the zero-native-note wiring guard")
	}
}
