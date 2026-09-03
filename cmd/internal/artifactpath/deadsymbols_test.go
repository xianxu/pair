package artifactpath

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// deadSymbolScope is the package this guard watches. It is deliberately one
// package rather than the tree: couchcore is where a deletion milestone shed
// five subsystems, and a guard that fires everywhere on day one gets an
// allowlist instead of a fix.
const deadSymbolScope = "cmd/internal/couchcore"

// deadSymbolAllowlist names production symbols that legitimately have no
// production caller. Each needs a reason -- an entry without one is how this
// guard degrades into a list of things nobody wanted to think about.
var deadSymbolAllowlist = map[string]string{
	// Seams and non-context wrappers: production takes the Context form, the
	// bare one exists so a test can call it without threading a context.
	"Spawn":                     "the test seam over the start path; documented as such at couch.go",
	"Resume":                    "non-context wrapper; production calls ResumeContext",
	"ActionableThreadInventory": "non-context wrapper; production calls ActionableThreadInventoryContext",
	"RecoverStoreJournal":       "the explicit recovery entry point; withLock recovers implicitly on the real path",

	// Fakes and hooks. A fake's only job is to be used by tests; these live
	// outside a _fake.go file, which is the only reason the scan sees them.
	"NewFakeProcOps":          "fake constructor; procops.go rather than a _fake.go file",
	"SetUnknown":              "fake helper for the unanswerable-probe case",
	"newThreadStoreWithHooks": "crash-injection constructor for journal recovery tests",

	// Genuinely unreferenced, and NOT dispositioned here. Deleting each means
	// deleting its tests, which is a judgement call per symbol rather than part
	// of one deletion sweep -- pair#173 owns that. Listed rather than silently
	// tolerated so the debt is countable.
	"PublishDescription":            "pair#173: superseded by ApplyThreadMetadata, which the publish-description op calls directly",
	"ReconcileActiveParks":          "pair#173: explicit reconciliation pass with no caller",
	"OperationNames":                "pair#173: the CLI resolves operations by name without it",
	"Unregister":                    "pair#173: registry mutation with no caller",
	"ResumeDiagnosticOf":            "pair#173: diagnostic accessor with no caller",
	"ClassifyThreadReferenceFields": "pair#173: per-row rule reached only through MatchThreadReferenceFields",
}

// pair#170 M4 deleted ~3,900 lines and left seven orphans, because deletion was
// checked by "the compiler is happy". Go does not error on an unused exported
// symbol, or on an unused unexported method -- so a subsystem's remnants sit
// there looking load-bearing.
//
// The fix round for that finding mechanised the DOCS half and left the Go half
// to recall, and it regressed in the same commit: deleting rollbackUnforkedStart
// orphaned ThreadStore.DeleteUnstartedThread. That is the argument for this
// being a test rather than a habit.
//
// Name-based on purpose. Resolving types would catch more, but the failure mode
// that matters is a whole cluster going cold at once, and a shared name only
// ever HIDES an orphan (a false negative) -- it never invents one.
func TestNoProductionSymbolIsReferencedOnlyByTests(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	declarations := productionDeclarations(t, filepath.Join(repoRoot, deadSymbolScope))
	references := productionIdentifierCounts(t, filepath.Join(repoRoot, "cmd"))

	var orphans []string
	for name, position := range declarations {
		if reason, allowed := deadSymbolAllowlist[name]; allowed {
			_ = reason
			continue
		}
		// One occurrence is the declaration itself.
		if references[name] <= 1 {
			orphans = append(orphans, position+": "+name)
		}
	}
	sort.Strings(orphans)
	for _, orphan := range orphans {
		t.Errorf("%s has no production reference outside its own declaration.\n"+
			"Delete it, or add it to deadSymbolAllowlist with the reason it survives.", orphan)
	}
}

// productionDeclarations collects top-level funcs, methods and types from the
// package's non-test files.
func productionDeclarations(t *testing.T, packageDir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		t.Fatal(err)
	}
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		// A _fake.go file exists to be used by tests; that is its whole job.
		if strings.HasSuffix(name, "_fake.go") {
			continue
		}
		file, err := parser.ParseFile(fileSet, filepath.Join(packageDir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			switch typed := declaration.(type) {
			case *ast.FuncDecl:
				// Methods satisfying an interface are reached through the
				// interface's own method name, which this counts.
				out[typed.Name.Name] = name
			case *ast.GenDecl:
				if typed.Tok != token.TYPE {
					continue
				}
				for _, spec := range typed.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						out[typeSpec.Name.Name] = name
					}
				}
			}
		}
	}
	return out
}

var identifierPattern = regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*\b`)

// productionIdentifierCounts counts identifier occurrences across every
// non-test Go file under root. Tests are excluded deliberately: a symbol only
// tests mention is exactly what this looks for.
func productionIdentifierCounts(t *testing.T, root string) map[string]int {
	t.Helper()
	counts := map[string]int{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, identifier := range identifierPattern.FindAllString(stripComments(string(raw)), -1) {
			counts[identifier]++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return counts
}

// stripComments keeps a symbol named only in prose from counting as a use --
// a doc comment describing a function is not a caller.
func stripComments(source string) string {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "src.go", source, 0)
	if err != nil {
		return source
	}
	var builder strings.Builder
	ast.Inspect(file, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok {
			builder.WriteString(identifier.Name)
			builder.WriteString(" ")
		}
		return true
	})
	return builder.String()
}
