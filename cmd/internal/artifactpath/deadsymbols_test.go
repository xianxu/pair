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

// deadSymbolScopes are the packages this guard watches, each with its own
// allowlist. Deliberately a list rather than the tree: a guard that fires
// everywhere on day one gets an allowlist instead of a fix. A package joins once
// its orphans have been dispositioned.
//
// couchcore is where a deletion milestone shed five subsystems. hostty lost its
// production writer in #255 M3, when every parent-terminal write moved to
// terminal.Presenter. Its escape-sequence constants then outlived their consumer
// for a week, reading as live policy, which is how #279's regression hid (#289).
var deadSymbolScopes = []struct {
	dir       string
	allowlist map[string]string
}{
	{"cmd/internal/couchcore", couchcoreDeadSymbolAllowlist},
	{"cmd/internal/hostty", hosttyDeadSymbolAllowlist},
}

// Each allowlist names production symbols that legitimately have no production
// caller. Each needs a reason -- an entry without one is how this guard
// degrades into a list of things nobody wanted to think about.
var hosttyDeadSymbolAllowlist = map[string]string{
	"EdgeTop": "represented and REFUSED: NewReservation's refusal is tested against it, and its doc carries the measured reason a top strip fails (#223)",
}

var couchcoreDeadSymbolAllowlist = map[string]string{
	"ReadStoreRetention": "pair#239 M2 Task4/5: read-only registered-store adapter; remove exemption when production GC preview is wired",
	"RestoreThread":      "pair#239: explicitly supported typed archive restoration transaction; no new UI is in scope",
	// Seams and non-context wrappers: production takes the Context form, the
	// bare one exists so a test can call it without threading a context.
	"Spawn":                     "the test seam over the start path; documented as such at couch.go",
	"Resume":                    "non-context wrapper; production calls ResumeContext",
	"ActionableThreadInventory": "non-context wrapper; production calls ActionableThreadInventoryContext",
	"ThreadInventory":           "non-context wrapper; production calls ThreadInventoryContext",
	"RecoverStoreJournal":       "the explicit recovery entry point; withLock recovers implicitly on the real path",

	// Fakes and hooks. A fake's only job is to be used by tests; these live
	// outside a _fake.go file, which is the only reason the scan sees them.
	"NewFakeProcOps":          "fake constructor; procops.go rather than a _fake.go file",
	"SetUnknown":              "fake helper for the unanswerable-probe case",
	"newThreadStoreWithHooks": "crash-injection constructor for journal recovery tests",

	// Vocabulary enumerations. Their job is to BE iterated by exhaustiveness
	// guards -- Go cannot check a switch for exhaustiveness, so the enumeration
	// is what does. A production caller would be the tail wagging the dog.
	"AllThreadReasons": "the ThreadReason vocabulary; iterated by the guards that prove every reason is produced and rendered",
	"AllThreadStates":  "the ActionableThreadState vocabulary; iterated by the offered-implies-permitted guards, which derive their domain from it rather than hand-listing states (pair#256 M2, BR-33)",

	// Genuinely unreferenced, and NOT dispositioned here. Deleting each means
	// deleting its tests, which is a judgement call per symbol rather than part
	// of one deletion sweep -- pair#192 owns that, and it is a filed issue:
	// workshop/issues/000192-disposition-six-production-symbols-reachable-only-from-tests.md.
	// A deferral whose ticket does not exist is not deferred, it is exempted,
	// which is what the M4 review caught this list doing.
	"PublishDescription":            "pair#192: superseded by ApplyThreadMetadata, which the publish-description op calls directly",
	"ReconcileActiveParks":          "pair#192: explicit reconciliation pass with no caller",
	"OperationNames":                "pair#192: the CLI resolves operations by name without it",
	"Unregister":                    "pair#192: registry mutation with no caller",
	"ResumeDiagnosticOf":            "pair#192: diagnostic accessor with no caller",
	"ClassifyThreadReferenceFields": "pair#192: per-row rule reached only through MatchThreadReferenceFields",
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
	references := productionIdentifierCounts(t, filepath.Join(repoRoot, "cmd"))
	for _, scope := range deadSymbolScopes {
		t.Run(scope.dir, func(t *testing.T) {
			declarations := productionDeclarations(t, filepath.Join(repoRoot, scope.dir))
			var orphans []string
			for name, position := range declarations {
				if _, allowed := scope.allowlist[name]; allowed {
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
					"Delete it, or add it to %s's allowlist with the reason it survives.", orphan, scope.dir)
			}
		})
	}
}

// productionDeclarations collects top-level funcs, methods, types, consts and
// vars from the package's non-test files.
//
// Consts and vars joined in #289: hostty's stranded surface was escape-sequence
// constants, which a funcs-and-types scan cannot see.
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
		// A fake file exists to be used by tests; that is its whole job.
		if name == "fake.go" || strings.HasSuffix(name, "_fake.go") {
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
				for _, spec := range typed.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						out[spec.Name.Name] = name
					case *ast.ValueSpec:
						if isIotaZero(spec) {
							continue
						}
						for _, identifier := range spec.Names {
							if identifier.Name != "_" {
								out[identifier.Name] = name
							}
						}
					}
				}
			}
		}
	}
	return out
}

// isIotaZero reports an enum's `X T = iota`. That value is reached by leaving a
// field unset, never by name, so a name count would call every one of them dead
// (couchcore's five *Unknown values were the measured cases).
func isIotaZero(spec *ast.ValueSpec) bool {
	if len(spec.Values) != 1 {
		return false
	}
	identifier, ok := spec.Values[0].(*ast.Ident)
	return ok && identifier.Name == "iota"
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
