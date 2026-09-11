package couchtty

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Menu code reads the inventory ONLY through the lookups that apply the
// reattach pass's view (pair#206).
//
// While the pass owns a row, the inventory is behind it: a thread mid-reattach
// reads detached, then stale, then busy, and a just-attached one reads detached
// until the next refresh. A reader that goes around the view reads a stale row.
// Three plan-review rounds each found another reader doing exactly that, so the
// rule is a test rather than a comment: any `.Inventory` read outside the
// functions below fails it.
//
// The allowlist is the view itself, plus the reducer's own bookkeeping, which
// installs and copies inventories rather than interpreting a row. `event` is
// excluded because MenuEvent.Inventory is the INCOMING data, not a row read.
var inventoryReadersAllowed = map[string]bool{
	"menuRows":             true, // the view: every row
	"menuThread":           true, // the view: one row
	"replaceMenuInventory": true, // installs a new inventory
	"cloneMenuState":       true, // copies state by value
	"reconcileMenuFrames":  true, // compares the prior inventory against the new one
}

func TestMenuCodeReadsTheInventoryOnlyThroughTheViewedLookups(t *testing.T) {
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
			if !ok || fn.Body == nil || inventoryReadersAllowed[fn.Name.Name] {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "Inventory" {
					return true
				}
				if ident, isIdent := selector.X.(*ast.Ident); isIdent && ident.Name == "event" {
					return true // MenuEvent.Inventory: the incoming data, not a row read
				}
				violations = append(violations, fset.Position(selector.Pos()).String()+" in "+fn.Name.Name)
				return true
			})
		}
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Errorf("%s reads the inventory around the reattach pass's view; use menuRows, menuThread or visibleMenuRows", v)
	}
}
