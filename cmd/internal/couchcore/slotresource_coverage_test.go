package couchcore

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
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
	{"saved work", regexp.MustCompile(`(^|/)saved-work($|/)`)},
}

// slotTokenAllowlist holds literals that match a token but are not slot
// state, each with its reason. Every entry must still match something
// (TestSlotTokenAllowlistIsCurrent), so a stale exemption cannot linger.
var slotTokenAllowlist = map[string]string{
	"move-slot":    "a hub ownership operation name, not a slot environment path",
	"--move-slot":  "a public ownership CLI flag, not a slot environment path",
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

// These two exact joins address native Git administrative metadata, not Pair
// slot containers. Source, function, base expression, arity and component all
// participate; the inventory below requires exactly one occurrence per entry.
type slotNativeGitJoinSite struct{ source, function, base, component string }

var slotNativeGitJoinAllowlist = map[slotNativeGitJoinSite]string{
	{"cmd/internal/sessioninventory/codex_git_scope.go", "codexRepositoryIdentity", "common", "worktrees"}: "validates a registered native Git administrative directory",
	{"cmd/internal/sessioninventory/codex_git_scope.go", "codexWorktreeCWDs", "id.common", "worktrees"}:    "enumerates native Git registrations for corresponding checkout directories",
}

func slotNativeGitJoin(src slotSourceFile, call *ast.CallExpr) (slotNativeGitJoinSite, bool) {
	if !isFilepathJoin(call) || len(call.Args) != 2 {
		return slotNativeGitJoinSite{}, false
	}
	component, ok := stringLit(call.Args[1])
	if !ok {
		return slotNativeGitJoinSite{}, false
	}
	var base bytes.Buffer
	if err := format.Node(&base, src.fset, call.Args[0]); err != nil {
		return slotNativeGitJoinSite{}, false
	}
	site := slotNativeGitJoinSite{src.rel, funcAt(src.file, call.Pos()), base.String(), component}
	_, allowed := slotNativeGitJoinAllowlist[site]
	return site, allowed
}
func slotNativeGitJoinInventoryViolations(sources []slotSourceFile) []string {
	counts := map[slotNativeGitJoinSite]int{}
	for _, src := range sources {
		ast.Inspect(src.file, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if site, allowed := slotNativeGitJoin(src, call); allowed {
					counts[site]++
				}
			}
			return true
		})
	}
	var violations []string
	for site, reason := range slotNativeGitJoinAllowlist {
		if counts[site] != 1 {
			violations = append(violations, fmt.Sprintf("native Git join exemption %s:%s base=%s component=%q (%s): occurrences=%d, want exactly 1", site.source, site.function, site.base, site.component, reason, counts[site]))
		}
	}
	sort.Strings(violations)
	return violations
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
	t.Parallel()

	for _, v := range slotStateViolations(productionGoFiles(t), slotLayoutMethodNames()) {
		t.Error(v)
	}
}

func slotStateViolations(sources []slotSourceFile, layoutMethods map[string]bool) []string {
	var violations []string
	for _, src := range sources {
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
							if _, allowed := slotNativeGitJoin(src, n); allowed {
								continue
							}
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
	return violations
}

// TestSlotTokenAllowlistIsCurrent keeps every exemption earning its place: an
// allowlisted literal must exist in production code and must match a token.
func TestSlotTokenAllowlistIsCurrent(t *testing.T) {
	t.Parallel()

	sources := productionGoFiles(t)
	for _, v := range slotNativeGitJoinInventoryViolations(sources) {
		t.Error(v)
	}
	seen := map[string]bool{}
	funcs := map[string]bool{}
	for _, src := range sources {
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

func slotCoverageFixture(t *testing.T, rel, body string) slotSourceFile {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, "package fixture\n"+body, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	return slotSourceFile{rel: rel, fset: fset, file: file}
}
func TestNativeGitContainerJoinClassification(t *testing.T) {
	native := slotCoverageFixture(t, "cmd/internal/sessioninventory/codex_git_scope.go", `func codexRepositoryIdentity(){filepath.Join(common,"worktrees")}
func codexWorktreeCWDs(){filepath.Join(id.common,"worktrees")}`)
	if violations := slotStateViolations([]slotSourceFile{native}, slotLayoutMethodNames()); len(violations) != 0 {
		t.Fatalf("native Git administrative directories classified as Pair slots: %v", violations)
	}
	for _, tc := range []struct{ name, rel, body string }{
		{"different source", "cmd/internal/couchcore/example.go", `func codexRepositoryIdentity(){filepath.Join(common,"worktrees")}`},
		{"different function", native.rel, `func createSlot(){filepath.Join(common,"worktrees")}`},
		{"different base", native.rel, `func codexRepositoryIdentity(){filepath.Join(fleet,"worktrees")}`},
		{"different component", native.rel, `func codexRepositoryIdentity(){filepath.Join(common,"worktree")}`},
		{"extra child", native.rel, `func codexRepositoryIdentity(){filepath.Join(common,"worktrees","slot")}`},
		{"slot root", native.rel, `func codexWorktreeCWDs(){filepath.Join(id.WorktreeRoot,"worktrees")}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := slotCoverageFixture(t, tc.rel, tc.body)
			violations := slotStateViolations([]slotSourceFile{src}, slotLayoutMethodNames())
			found := false
			for _, v := range violations {
				found = found || strings.Contains(v, "(a) slot container directory")
			}
			if !found {
				t.Fatalf("unrelated slot construction escaped guard: %v", violations)
			}
		})
	}
}

func TestNativeGitJoinInventoryRejectsStaleAndDuplicateExemptions(t *testing.T) {
	source := "cmd/internal/sessioninventory/codex_git_scope.go"
	first := `func codexRepositoryIdentity(){filepath.Join(common,"worktrees")}`
	second := `func codexWorktreeCWDs(){filepath.Join(id.common,"worktrees")}`
	for _, tc := range []struct {
		name, body string
		want       int
		fragment   string
	}{
		{"exact", first + "\n" + second, 0, ""},
		{"missing", first, 1, "occurrences=0"},
		{"changed base", first + "\n" + `func codexWorktreeCWDs(){filepath.Join(other.common,"worktrees")}`, 1, "occurrences=0"},
		{"duplicate", `func codexRepositoryIdentity(){filepath.Join(common,"worktrees");filepath.Join(common,"worktrees")}` + "\n" + second, 1, "occurrences=2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := slotCoverageFixture(t, source, tc.body)
			violations := slotNativeGitJoinInventoryViolations([]slotSourceFile{src})
			if len(violations) != tc.want {
				t.Fatalf("%v: want %d violations", violations, tc.want)
			}
			if tc.want > 0 && !strings.Contains(violations[0], tc.fragment) {
				t.Fatalf("wrong failure: %v", violations)
			}
		})
	}
}
