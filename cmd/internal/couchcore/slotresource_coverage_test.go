package couchcore

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The slot-state coverage audit (pair#387 Done-when 1). The resource table is
// the single source only if no slot state is located anywhere else, so this
// test reads the code itself: the expected set of sites is never written
// down, it is whatever the AST contains.

// slotLayoutFile is the only production file allowed to spell slot state.
const slotLayoutFile = "cmd/internal/couchcore/slotlayout.go"

// slotStateTokens name KINDS of slot state. A literal matching one outside
// slotLayoutFile is a slot-state site that bypasses the table. Matching is at
// a path or ref boundary, so "add-slot" (a menu action) is not "-slot<N>".
var slotStateTokens = []struct {
	name string
	re   *regexp.Regexp
}{
	{"slot environment name (-slot<N>)", regexp.MustCompile(`-slot(\d|$)`)},
	{"resting branch (main-slot)", regexp.MustCompile(`main-slot`)},
	{"slot store (.couch)", regexp.MustCompile(`(^|/)\.couch($|/)`)},
	{"setup marker", regexp.MustCompile(`couch-setup-success\.json`)},
	{"setup attempt memo", regexp.MustCompile(`couch-setup-attempt\.json`)},
	{"weave setup lock", regexp.MustCompile(`\.weave-setup\.lock`)},
	{"Couch's git-common directory", regexp.MustCompile(`couch-workspaces`)},
	{"creation intent", regexp.MustCompile(`creation\.json`)},
	{"saved work", regexp.MustCompile(`saved-work`)},
}

// slotTokenAllowlist holds literals that match a token but are not slot
// state, each with its reason. Every entry must still match something
// (TestSlotTokenAllowlistIsCurrent), so a stale exemption cannot linger.
var slotTokenAllowlist = map[string]string{
	"add-slot":     "a switcher menu action name, not a path",
	"unknown-slot": "a slot-operation receipt code, not a path",
	"reuse-slot":   "a start-resolution fingerprint field, not a path",
}

// slotPathRootFields are identity fields whose value is a slot or fleet root:
// joining a path onto one outside slotLayoutFile builds slot state by hand.
var slotPathRootFields = map[string]bool{"EnvironmentRoot": true, "WorktreeRoot": true, "FleetRoot": true, "PrimaryRoot": true, "RepoIdentity": true}

// slotPathRootAllowlist names functions (file:func) whose joins onto those
// fields are not slot state, each with its reason.
var slotPathRootAllowlist = map[string]string{
	"cmd/internal/couchcore/workspace_identity.go:validate":          "a fleet's primary checkout <fleet>/<repo>, not slot state",
	"cmd/internal/couchcore/repository_family.go:ResolveFamilyStart": "a family's start directory inside a checkout, not slot state",
}

// slotGitFiles may run git verbs that change slot state. provision.go is
// listed until Task 2.3 replaces Ensure's body with the reconciler.
var slotGitFiles = map[string]string{
	"cmd/internal/couchcore/slotobserve.go":  "observes slot resources",
	"cmd/internal/couchcore/slotconverge.go": "converges slot resources",
	"cmd/internal/couchcore/slotcatalog.go":  "discovers slots (git worktree list)",
	"cmd/internal/couchcore/slotgit.go":      "reads slot git status",
	"cmd/internal/couchcore/provision.go":    "Ensure, until Task 2.3 replaces it with Reconcile",
}

var gitWorktreeSubcommands = map[string]bool{"add": true, "remove": true, "list": true, "prune": true, "repair": true, "move": true, "lock": true, "unlock": true}

type slotSourceFile struct {
	rel  string
	fset *token.FileSet
	file *ast.File
}

func productionGoFiles(t *testing.T) []slotSourceFile {
	t.Helper()
	root := repoRootFrom(t)
	var out []slotSourceFile
	err := filepath.WalkDir(filepath.Join(root, "cmd"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "testdata" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		fset := token.NewFileSet()
		file, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if perr != nil {
			return perr
		}
		out = append(out, slotSourceFile{rel: filepath.ToSlash(rel), fset: fset, file: file})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 100 {
		t.Fatalf("found only %d production Go files; the walk is not reading the tree", len(out))
	}
	return out
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(lit.Value)
	return v, err == nil
}

func isFilepathJoin(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Join" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "filepath"
}

// enclosingFunc maps each node position to the name of the function declaring it.
func funcAt(file *ast.File, pos token.Pos) string {
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Pos() <= pos && pos <= fn.End() {
			return fn.Name.Name
		}
	}
	return ""
}

func slotLayoutMethodNames() map[string]bool {
	out := map[string]bool{}
	typ := reflect.TypeOf(SlotLayout{})
	for i := 0; i < typ.NumMethod(); i++ {
		out[typ.Method(i).Name] = true
	}
	return out
}

func TestEverySlotStateSiteIsAResource(t *testing.T) {
	layoutMethods := slotLayoutMethodNames()
	var violations []string
	for _, src := range productionGoFiles(t) {
		if src.rel == slotLayoutFile {
			continue
		}
		at := func(n ast.Node) string { return src.fset.Position(n.Pos()).String() }
		ast.Inspect(src.file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.BasicLit:
				v, ok := stringLit(n)
				if !ok {
					return true
				}
				if _, allowed := slotTokenAllowlist[v]; allowed {
					return true
				}
				for _, tok := range slotStateTokens {
					if tok.re.MatchString(v) {
						violations = append(violations, at(n)+": (a) "+tok.name+" spelled outside SlotLayout: "+strconv.Quote(v))
					}
				}
				if v == "update-ref" || strings.HasPrefix(v, "branch.") {
					if _, ok := slotGitFiles[src.rel]; !ok {
						violations = append(violations, at(n)+": (d) git ref/config verb outside the slot git files: "+strconv.Quote(v))
					}
				}
			case *ast.CallExpr:
				if isFilepathJoin(n) && len(n.Args) > 0 {
					for _, arg := range n.Args[1:] {
						if v, ok := stringLit(arg); ok && (v == "worktree" || v == "worktrees") {
							violations = append(violations, at(n)+": (a) slot container directory "+strconv.Quote(v)+" joined outside SlotLayout")
						}
					}
					switch first := n.Args[0].(type) {
					case *ast.SelectorExpr:
						if slotPathRootFields[first.Sel.Name] {
							if _, ok := slotPathRootAllowlist[src.rel+":"+funcAt(src.file, n.Pos())]; !ok {
								violations = append(violations, at(n)+": (b) path joined onto ."+first.Sel.Name+" outside SlotLayout")
							}
						}
					case *ast.CallExpr:
						if sel, ok := first.Fun.(*ast.SelectorExpr); ok && layoutMethods[sel.Sel.Name] && len(first.Args) == 0 {
							violations = append(violations, at(n)+": (b) path joined onto SlotLayout."+sel.Sel.Name+"() outside SlotLayout")
						}
					}
				}
				for i, arg := range n.Args {
					if v, ok := stringLit(arg); ok && v == "worktree" && i+1 < len(n.Args) {
						if next, ok := stringLit(n.Args[i+1]); ok && gitWorktreeSubcommands[next] {
							if _, ok := slotGitFiles[src.rel]; !ok {
								violations = append(violations, at(n)+": (d) git worktree "+next+" outside the slot git files")
							}
						}
					}
				}
			}
			return true
		})
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Error(v)
	}
}

// TestSlotTokenAllowlistIsCurrent keeps every exemption earning its place: an
// allowlisted literal must exist in production code and must match a token.
func TestSlotTokenAllowlistIsCurrent(t *testing.T) {
	seen := map[string]bool{}
	funcs := map[string]bool{}
	for _, src := range productionGoFiles(t) {
		ast.Inspect(src.file, func(n ast.Node) bool {
			if v, ok := stringLit(asExpr(n)); ok {
				seen[v] = true
			}
			if fn, ok := n.(*ast.FuncDecl); ok {
				funcs[src.rel+":"+fn.Name.Name] = true
			}
			return true
		})
	}
	for lit, reason := range slotTokenAllowlist {
		matched := false
		for _, tok := range slotStateTokens {
			matched = matched || tok.re.MatchString(lit)
		}
		if !seen[lit] || !matched {
			t.Errorf("stale token exemption %q (%s): present=%v, matches a token=%v", lit, reason, seen[lit], matched)
		}
	}
	for site, reason := range slotPathRootAllowlist {
		if !funcs[site] {
			t.Errorf("stale path-root exemption %s (%s): no such function", site, reason)
		}
	}
}

func asExpr(n ast.Node) ast.Expr {
	e, _ := n.(ast.Expr)
	return e
}
