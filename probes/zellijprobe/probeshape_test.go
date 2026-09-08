package zellijprobe_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A PROBE MUST NOT EXIT PAST ITS OWN CLEANUP.
//
// `os.Exit` SKIPS DEFERS. Every probe here registers cleanup as a defer —
// deleting the zellij session it created, removing a temp layout or data dir —
// and then used to leave through `os.Exit` on three or four paths each,
// INCLUDING the likeliest one of all: `PROBE-INCONCLUSIVE: the session never
// appeared`. Measured during pair#199 M3: six `couchnestedrows-<pid>` sessions
// left alive by failing runs, each name carrying a pid nothing later reclaims.
//
// pair#199 BR-51 made "a probe can only destroy a session it made" structural.
// This is the other half, and it needs to be structural for the same reason:
// "remember not to os.Exit below a defer" is exactly the kind of rule that holds
// until the next diagnostic path is added.
//
// THE SHAPE: `func main() { os.Exit(run()) }`, every path RETURNS a code, and
// the defers live in `run`.
func TestNoProbeExitsPastItsOwnCleanup(t *testing.T) {
	root := filepath.Join("..", "..")
	dirs := []string{
		filepath.Join(root, "probes"),
		filepath.Join(root, "cmd", "probes"),
	}

	checked := 0
	for _, base := range dirs {
		entries, err := os.ReadDir(base)
		if err != nil {
			t.Fatalf("read %s: %v", base, err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			files, err := filepath.Glob(filepath.Join(base, e.Name(), "*.go"))
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range files {
				if strings.HasSuffix(path, "_test.go") {
					continue
				}
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, path, nil, 0)
				if err != nil {
					t.Fatalf("parse %s: %v", path, err)
				}
				checked++
				for _, decl := range file.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						continue
					}
					var hasDefer bool
					var exits []token.Pos
					ast.Inspect(fn.Body, func(n ast.Node) bool {
						switch node := n.(type) {
						case *ast.DeferStmt:
							hasDefer = true
						case *ast.CallExpr:
							if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
								if pkg, ok := sel.X.(*ast.Ident); ok &&
									pkg.Name == "os" && sel.Sel.Name == "Exit" {
									exits = append(exits, node.Pos())
								}
							}
						}
						return true
					})
					if !hasDefer || len(exits) == 0 {
						continue
					}
					for _, pos := range exits {
						t.Errorf("%s: %s registers a defer and then calls os.Exit at %s — "+
							"os.Exit skips defers, so this leaks whatever the defer was cleaning up. "+
							"Return a code instead and let main be `func main() { os.Exit(run()) }`.",
							path, fn.Name.Name, fset.Position(pos))
					}
				}
			}
		}
	}
	if checked < 5 {
		t.Fatalf("inspected %d probe source files; there are more, so the guard is "+
			"reading less than it claims", checked)
	}
}
