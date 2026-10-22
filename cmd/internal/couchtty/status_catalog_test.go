package couchtty

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// Inventory the source of built-in chrome messages, not a second list of their
// wording. Dynamic labels/errors remain full diagnostics and are display-capped.
func statusCatalog(src string) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "status.go", src, 0)
	if err != nil {
		return nil, err
	}
	var out []string
	globals := map[string][]ast.Expr{}
	for _, decl := range f.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gen.Specs {
				if v, ok := spec.(*ast.ValueSpec); ok {
					for i, id := range v.Names {
						if i < len(v.Values) {
							globals[id.Name] = append(globals[id.Name], v.Values[i])
						}
					}
				}
			}
		}
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		// This is a full diagnostic report, shared unchanged with the generated
		// hub event and desktop notification (sequence, fence and epoch included).
		// It remains recoverable from the feed; only its chrome projection is capped.
		if fn.Name.Name == "handleAutomaticInputStall" {
			continue
		}
		defs := map[string][]ast.Expr{}
		for key, value := range globals {
			defs[key] = append([]ast.Expr(nil), value...)
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if r, ok := n.(*ast.RangeStmt); ok {
				if id, ok := r.Value.(*ast.Ident); ok {
					defs[id.Name] = append(defs[id.Name], r.X)
				}
			}
			if v, ok := n.(*ast.ValueSpec); ok {
				for i, id := range v.Names {
					if i < len(v.Values) {
						defs[id.Name] = append(defs[id.Name], v.Values[i])
					}
				}
			}
			if a, ok := n.(*ast.AssignStmt); ok {
				for i, lhs := range a.Lhs {
					if id, ok := lhs.(*ast.Ident); ok && i < len(a.Rhs) {
						defs[id.Name] = append(defs[id.Name], a.Rhs[i])
					}
				}
			}
			return true
		})
		var literals func(ast.Expr, map[string]bool) []string
		literals = func(e ast.Expr, seen map[string]bool) []string {
			switch e := e.(type) {
			case *ast.BasicLit:
				if e.Kind == token.STRING {
					s, _ := strconv.Unquote(e.Value)
					return []string{s}
				}
			case *ast.Ident:
				if seen[e.Name] {
					return nil
				}
				next := map[string]bool{}
				for k, v := range seen {
					next[k] = v
				}
				next[e.Name] = true
				var parts []string
				for _, rhs := range defs[e.Name] {
					parts = append(parts, literals(rhs, next)...)
				}
				return parts
			case *ast.BinaryExpr:
				if e.Op == token.ADD {
					a, b := literals(e.X, seen), literals(e.Y, seen)
					if len(a) == 0 {
						return b
					}
					if len(b) == 0 {
						return a
					}
					var sums []string
					for _, x := range a {
						for _, y := range b {
							sums = append(sums, x+y)
						}
					}
					return sums
				}
			case *ast.CallExpr:
				if id, ok := e.Fun.(*ast.Ident); ok && id.Name == "append" {
					var parts []string
					for _, arg := range e.Args[1:] {
						parts = append(parts, literals(arg, seen)...)
					}
					return parts
				}
				if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
					if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "fmt" && sel.Sel.Name == "Sprintf" && len(e.Args) > 0 {
						parts := literals(e.Args[0], seen)
						for i := range parts {
							parts[i] = statusFormatVerb.ReplaceAllString(parts[i], "")
						}
						return parts
					}
				}
			}
			return nil
		}
		add := func(e ast.Expr) { out = append(out, literals(e, map[string]bool{})...) }
		helper := fn.Name.Name == "capabilityNotice" || fn.Name.Name == "driveNotices" || fn.Name.Name == "driveFailure"
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				for i, lhs := range n.Lhs {
					if ix, ok := lhs.(*ast.IndexExpr); ok {
						if field, ok := ix.X.(*ast.SelectorExpr); ok && field.Sel.Name == "diagnostics" && i < len(n.Rhs) {
							add(n.Rhs[i])
						}
					}
				}
			case *ast.CallExpr:
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "setNotice" || sel.Sel.Name == "pointerOff" || sel.Sel.Name == "reportPrevious" || sel.Sel.Name == "notify" || sel.Sel.Name == "notice") && len(n.Args) > 0 {
					add(n.Args[0])
				}
				if helper {
					if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "append" {
						for _, arg := range n.Args[1:] {
							add(arg)
						}
					}
				}
			case *ast.CompositeLit:
				name := ""
				switch typ := n.Type.(type) {
				case *ast.Ident:
					name = typ.Name
				case *ast.SelectorExpr:
					name = typ.Sel.Name
				}
				if name == "Notice" {
					for _, elt := range n.Elts {
						if kv, ok := elt.(*ast.KeyValueExpr); ok {
							if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Body" {
								add(kv.Value)
							}
						}
					}
				}
			case *ast.ReturnStmt:
				if helper {
					for _, e := range n.Results {
						add(e)
					}
				}
			}
			return true
		})
	}
	return out, nil
}

var statusFormatVerb = regexp.MustCompile(`%[-+# 0]*[0-9]*(?:\.[0-9]+)?[a-zA-Z%]`)

func TestBuiltInChromeStatusCatalog(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	upstream, err := filepath.Glob("../couchcmd/*.go")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, upstream...)
	count := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		messages, err := statusCatalog(string(src))
		if err != nil {
			t.Fatal(err)
		}
		count += len(messages)
		for _, message := range messages {
			if width := textwidth.Width(message); width > ChromeStatusCells {
				t.Errorf("%s: built-in status template has %d cells: %q", path, width, message)
			}
		}
	}
	if count == 0 {
		t.Fatal("empty built-in chrome status catalog")
	}
}

func TestStatusCatalogDiscoversActualProducerTemplates(t *testing.T) {
	long := strings.Repeat("x", ChromeStatusCells+1)
	for _, body := range []string{
		`c.setNotice("` + long + `")`,
		`const message="` + long + `"; c.setNotice(message)`,
		`c.pointerOff("` + long + `")`,
		`c.notice("` + long + `")`,
		`var message="` + long + `"; c.notify(message)`,
		`say=append(say,"` + long + `"); for _,message:=range say { c.notify(message) }`,
		`p.diagnostics[""]="` + long + `"`,
		`c.setNotice(fmt.Sprintf("` + long + ` %s",label))`,
		`notice:="` + long + `"; c.publishNotice(Notice{Body:notice})`,
		`return "` + long + `"+err.Error()`,
		`out=append(out,label+"` + long + `")`,
	} {
		name := "catalogFixture"
		if strings.HasPrefix(body, "return ") || strings.HasPrefix(body, "out=append") {
			name = "driveNotices"
		}
		got, err := statusCatalog("package p; func " + name + "(){" + body + "}")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, s := range got {
			found = found || textwidth.Width(s) > ChromeStatusCells
		}
		if !found {
			t.Fatalf("missed overlong producer: %s; got %q", body, got)
		}
	}
	got, err := statusCatalog(`package p; func unrelated(){ c.showMenu("` + long + `"); _=Detail{Body:"` + long + `"} }`)
	if err != nil || len(got) != 0 {
		t.Fatalf("catalog included unrelated diagnostics: %q, %v", got, err)
	}
}
