package terminal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

// The ErrNoDestination contract, enforced on the METHOD SET rather than on one
// method (pair#265 BR-9).
//
// Three gates caught the same mistake in a row -- the plan gate found
// UpdateChrome, the close gate found resizeLayout, then found that the AST door
// guard pinned only Input. Each time the fix was applied to the instance, so the
// next instance was free to appear. The rule is: every console or mux call to a
// presenter method that can answer ErrNoDestination must classify it.
//
// producerEntryPoints is the one hand-written fact, and it is the MAPPING, not
// the set: which exported methods a consumer can call to reach each producer.
// The producer set itself is derived from the source below, so a fifth producer
// fails this test rather than silently widening the answer surface.
var producerEntryPoints = map[string][]string{
	"Input":        {"Input"},
	"mouseInput":   {"Input"}, // reachable only through Input
	"UpdateChrome": {"UpdateChrome"},
	"resizeLayout": {"Resize", "ResizeLayout"},
}

// consumerPackages are the packages that hold a presenter and must classify its
// answer. Both had an unclassified site at some point in pair#265.
var consumerPackages = []string{"../couchtty", "../termcmd"}

func parseDir(t *testing.T, dir string) ([]*ast.File, *token.FileSet) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var out []*ast.File
	fset := token.NewFileSet()
	for _, path := range paths {
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
		out = append(out, file)
	}
	if len(out) == 0 {
		t.Fatalf("no production sources under %s", dir)
	}
	return out, fset
}

// Every function that can answer ErrNoDestination is declared in the mapping.
func TestNoDestinationProducersAreEnumerated(t *testing.T) {
	var found []string
	files, _ := parseDir(t, ".")
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Name.Name == "noDestination" {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "noDestination" {
					if !slices.Contains(found, fn.Name.Name) {
						found = append(found, fn.Name.Name)
					}
				}
				return true
			})
		}
	}
	var declared []string
	for name := range producerEntryPoints {
		declared = append(declared, name)
	}
	sort.Strings(found)
	sort.Strings(declared)
	if !slices.Equal(found, declared) {
		t.Fatalf("ErrNoDestination producers drifted.\n  in source:   %v\n  in mapping:  %v\n"+
			"Add the new producer to producerEntryPoints with the exported methods that reach it, "+
			"then make sure every consumer of those methods classifies the answer.", found, declared)
	}
}

// Every consumer call to a reachable entry point classifies the answer.
func TestConsumersOfNoDestinationMethodsClassifyIt(t *testing.T) {
	guarded := map[string]bool{}
	for _, entries := range producerEntryPoints {
		for _, name := range entries {
			guarded[name] = true
		}
	}
	var violations []string
	for _, dir := range consumerPackages {
		files, fset := parseDir(t, dir)
		for _, file := range files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				classifies := false
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					if sel, ok := node.(*ast.SelectorExpr); ok && sel.Sel.Name == "ErrNoDestination" {
						classifies = true
					}
					return true
				})
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || !guarded[sel.Sel.Name] {
						return true
					}
					inner, ok := sel.X.(*ast.SelectorExpr)
					if !ok || inner.Sel.Name != "presenter" {
						return true
					}
					if !classifies {
						violations = append(violations,
							fset.Position(call.Pos()).String()+": "+fn.Name.Name+" calls presenter."+sel.Sel.Name+" without classifying ErrNoDestination")
					}
					return true
				})
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("presenter methods that can answer ErrNoDestination, reached without classifying it:\n  %s",
			strings.Join(violations, "\n  "))
	}
}
