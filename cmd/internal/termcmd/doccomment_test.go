package termcmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// ONE DECLARATION, ONE GODOC.
//
// Three findings in this family across M3 (`paneTitleLocked`, `childSizeLocked`,
// `redrawTab`, plus `ptychild.Screen.HoldsCursorSave` whose doc was inserted into
// the MIDDLE of `TakeRowDirty`'s). The shape is always the same and always
// well-intentioned: behaviour changes, a new opening sentence is written, and the
// old one is left above it. godoc then prints a function's own opposite, and the
// reader has no way to tell which paragraph is current.
//
// The tell is mechanical, which is why this can be a test rather than a habit: a
// doc comment that starts its declaration's name at the beginning of a line more
// than once has been restarted. Named in the M3 review as "a vet-style check for
// two comment blocks on one declaration catches the class"; this is that check.
func TestNoDeclarationCarriesTwoStackedGodocs(t *testing.T) {
	roots := []string{
		filepath.Join("..", "termcmd"),
		filepath.Join("..", "hostty"),
		filepath.Join("..", "ptychild"),
		filepath.Join("..", "rowtext"),
	}
	checked := 0
	for _, root := range roots {
		files, err := filepath.Glob(filepath.Join(root, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			for _, decl := range file.Decls {
				for _, d := range documented(decl) {
					checked++
					if n := countOpeners(d.doc, d.name); n > 1 {
						t.Errorf("%s: %s's doc comment opens %d times — a superseded paragraph was "+
							"left above its replacement, so godoc prints both. Rewrite the sentence "+
							"that is now wrong; do not append a correction below it.",
							fset.Position(d.doc.Pos()), d.name, n)
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no documented declaration was inspected; the guard proved nothing")
	}
}

// countOpeners counts the lines in a doc comment that START its declaration's
// name — i.e. how many times the godoc begins. One is correct; more means an
// older opening paragraph was left standing above its replacement.
func countOpeners(doc *ast.CommentGroup, name string) int {
	n := 0
	for _, c := range doc.List {
		body := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
		if body == name || strings.HasPrefix(body, name+" ") {
			n++
		}
	}
	return n
}

// documented is every (name, doc comment) pair a declaration carries.
//
// GROUPED declarations count, and that is the whole point of this shape. The
// first version returned nothing when a GenDecl had more than one Spec, which
// silently excluded `hostty/control.go`'s single `const (…)` block -- the file
// where this very milestone rewrote two doc comments, and therefore the most
// likely place for the defect the guard is named after.
func documented(decl ast.Decl) []struct {
	name string
	doc  *ast.CommentGroup
} {
	type pair = struct {
		name string
		doc  *ast.CommentGroup
	}
	var out []pair
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Doc != nil {
			out = append(out, pair{d.Name.Name, d.Doc})
		}
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			var name string
			var doc *ast.CommentGroup
			switch sp := spec.(type) {
			case *ast.TypeSpec:
				name, doc = sp.Name.Name, sp.Doc
			case *ast.ValueSpec:
				if len(sp.Names) == 0 {
					continue
				}
				name, doc = sp.Names[0].Name, sp.Doc
			default:
				continue
			}
			// An ungrouped declaration carries its doc on the GenDecl, a
			// grouped one on each Spec. Both reach a reader as that
			// declaration's godoc, so both are checked.
			if doc == nil && len(d.Specs) == 1 {
				doc = d.Doc
			}
			if doc != nil {
				out = append(out, pair{name, doc})
			}
		}
	}
	return out
}
