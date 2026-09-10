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
	// SCOPE IS DERIVED, NOT LISTED. This guard replaced a remembered habit with
	// a check; hand-listing four package roots put the remembering back, one
	// level out. It was measured: adding `couchtty` to the old list failed
	// immediately on a declaration carrying another function's doc -- inside a
	// paragraph explaining that this exact mistake had been caught before.
	roots, err := filepath.Glob(filepath.Join("..", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) < 4 {
		t.Fatalf("globbed %d package roots under cmd/internal; the scope went blind", len(roots))
	}
	checked := 0
	for _, root := range roots {
		files, globErr := filepath.Glob(filepath.Join(root, "*.go"))
		if globErr != nil {
			t.Fatal(globErr)
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
			// The names declared in THIS file, so the second check below can
			// tell "opens with another declaration's name" from "opens with an
			// ordinary word".
			declared := map[string]bool{}
			for _, decl := range file.Decls {
				for _, d := range documented(decl) {
					declared[d.name] = true
				}
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
					if stolen := stolenFrom(d.doc, d.name, declared); stolen != "" {
						t.Errorf("%s: %s's doc comment opens with %s's — a new declaration was "+
							"inserted between %s's doc and its func, so godoc attributes one "+
							"function's paragraph to another and leaves %s undocumented. Move the "+
							"paragraph back down to the declaration it describes.",
							fset.Position(d.doc.Pos()), d.name, stolen, stolen, stolen)
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

// stolenFrom names the declaration whose godoc this one opens with, or "" when
// the doc opens with its own name or with an ordinary word.
//
// The OTHER half of this family, and the half countOpeners cannot see. Its tell
// is one doc comment restarted; this one's is a new declaration inserted
// BETWEEN an existing doc and the func it described. The doc then reads as the
// new declaration's, and the old one is left bare — godoc prints one function's
// paragraph above another's body, which is worse than no doc because it is
// confidently wrong.
//
// Both had already happened when the guard was written (`redrawTab` and
// `ptychild.Screen.HoldsCursorSave`, whose doc landed in the MIDDLE of
// `TakeRowDirty`'s) and the guard caught only the first shape; it recurred in
// #209, where `RequestRepaint` was inserted between `Child.Resize`'s doc and
// its func. So this is the class's own lesson applied to the check for it.
//
// Deliberately narrow: it fires only when the leading identifier is ANOTHER
// declaration's name in the same file. A doc opening with an ordinary word is
// not the convention but is also not this bug, and flagging it would make the
// guard a style rule nobody keeps.
func stolenFrom(doc *ast.CommentGroup, name string, declared map[string]bool) string {
	if len(doc.List) == 0 {
		return ""
	}
	body := strings.TrimSpace(strings.TrimPrefix(doc.List[0].Text, "//"))
	lead := body
	if i := strings.IndexAny(lead, " \t"); i >= 0 {
		lead = lead[:i]
	}
	if lead == name || !declared[lead] {
		return ""
	}
	return lead
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
