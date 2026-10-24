package repolint

import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"

	"github.com/xianxu/ariadne/pkg/golint"
)

// Copy the shared catalog before extending it: Pair rules must not alter the
// upstream catalog used by sleepSites to identify existing sleep debt.
var pairCatalog = append(append([]golint.Rule(nil), golint.Catalog...), golint.Rule{
	Name:     "test-fifo-empty-handoff",
	Incident: "pair#679",
	Message:  "write a nonempty FIFO payload before closing a test writer; an empty open/close can strand Darwin readers",
	Match:    func(*ast.File, *golint.CheckContext) []ast.Node { return nil }, // bound to test filenames by pairCatalogFor
}, golint.Rule{
	Name:     "slash-command-submit",
	Incident: "pair#162",
	Message:  "execute authored slash commands with KeyEnter and exact command echo proof; KeySubmit is prose submission (Alt+Return)",
	Match:    func(*ast.File, *golint.CheckContext) []ast.Node { return nil }, // bound to the filename by pairCatalogFor
}, golint.Rule{
	Name:     "quit-trigger-private-home",
	Incident: "pair#797",
	Message:  "set a private HOME before in-process TriggerQuit/TriggerBoundQuit tests; a private data directory does not isolate compatibility quit markers",
	Match:    syntaxMatch(quitTriggersWithoutPrivateHome),
}, golint.Rule{
	Name:     "fixture-cli-group-completion",
	Incident: "pair#744",
	Message:  "observe CLI process-group quiescence through cliGroupError after Wait; leader completion alone cannot acknowledge descendants",
	Match:    syntaxMatch(fixtureCLIUnobservedGroups),
}, golint.Rule{
	Name:     "wrapper-child-release",
	Incident: "pair#715",
	Message:  "retain wrapper child ownership and join it with Wait before returning; Process.Release permits writes after wrapper shutdown",
	Match:    syntaxMatch(wrapperChildReleases),
}, golint.Rule{
	Name:     "lifecycle-failure-address",
	Incident: "pair#714",
	Message:  "compose diagnostic addresses atomically with completeFailureAddress; partial target evidence must not replace an exact captured address",
	Match:    syntaxMatch(lifecycleFailureAddressOverrides),
}, golint.Rule{
	Name:     "terminal-registry-owner",
	Incident: "pair#731",
	Message:  "use LiveOwnerIDs for native terminal ownership; kill-only LiveIDs admits a shell after exec",
	Match:    syntaxMatch(terminalRegistryOwnerCalls),
}, golint.Rule{
	Name:     "zellij-read-attempt-bypass",
	Incident: "pair#748",
	Message:  "call zellijquery.Source.ReadContext to preserve bounded read recovery, the shared deadline and per-attempt evidence; readAttempt belongs only to that entry point",
	Match:    syntaxMatch(zellijReadAttemptBypasses),
}, golint.Rule{
	Name:     "zellij-query-boundary",
	Incident: "pair#803",
	Message:  "use zellijquery.Source.ReadContext or Output for decision reads; raw collectors belong only to the shared attempt and qualified diagnostics",
	Match:    func(*ast.File, *golint.CheckContext) []ast.Node { return nil }, // bound to the filename by pairCatalogFor
}, golint.Rule{
	Name:     "controlled-zellij-close",
	Incident: "pair#764",
	Message:  "report or propagate ControlledZellij.Close cleanup errors; discarded cleanup can leave a controlled session alive",
	Match:    syntaxMatch(discardedControlledZellijCloses),
}, golint.Rule{
	Name:     "fd-path-permission",
	Incident: "pair#767",
	Message:  "prove inherited descriptor usability directly (read/fstat/fcntl), not through a /dev/fd pathname permission check; macOS can report a false negative",
	Match:    syntaxMatch(fdPathPermissionChecks),
}, golint.Rule{
	Name:     "scribe-pty-printf",
	Incident: "pair#766",
	Message:  "use a terminal-aware producer and acknowledge its readiness before final output; direct printf can fail before producing the test marker",
	Match:    syntaxMatch(scribePTYPrintfCalls),
}, golint.Rule{
	Name:     "startup-admission-budget",
	Incident: "pair#773",
	Message:  "birthBound is a probe cadence, not a terminal admission budget; preserve the caller-owned startup deadline",
	Match:    syntaxMatch(birthProbeAdmissionBudgets),
}, golint.Rule{
	Name:     "manual-clock-rendezvous",
	Incident: "pair#771",
	Message:  "manualClock timer delivery must rendezvous with its receiver before a request barrier proves completion",
	Match:    syntaxMatch(bufferedManualClockTimers),
}, golint.Rule{
	Name:     "fixture-couch-identity",
	Incident: "pair#778",
	Message:  "seed fixture roots with testpair.SeedCouchIdentity so the production census recognizes their Couch identity",
	Match:    syntaxMatch(fixtureConstructorsWithoutCouchIdentity),
})

// syntaxMatch adapts Pair's AST-only rules to the contextual upstream catalog.
// Contextual upstream rules retain their original context and metadata.
func syntaxMatch(match func(*ast.File) []ast.Node) func(*ast.File, *golint.CheckContext) []ast.Node {
	return func(file *ast.File, _ *golint.CheckContext) []ast.Node { return match(file) }
}

// These enumerated fixture constructors must preserve the production identity
// contract. This syntax guard checks a direct imported helper call, not runtime
// execution or every possible fixture factory; constructor renames must update it.
func fixtureConstructorsWithoutCouchIdentity(file *ast.File) []ast.Node {
	var constructor string
	switch file.Name.Name {
	case "main":
		constructor = "installedEnv"
	case "testcouch":
		constructor = "allocateRoots"
	default:
		return nil
	}
	imports := map[string]bool{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != "github.com/xianxu/pair/cmd/internal/testpair" {
			continue
		}
		name := "testpair"
		if spec.Name != nil {
			name = spec.Name.Name
		}
		imports[name] = true
	}
	var matches []ast.Node
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != constructor {
			continue
		}
		seeded := false
		if fn.Body != nil {
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "SeedCouchIdentity" {
					return true
				}
				pkg, ok := ast.Unparen(selector.X).(*ast.Ident)
				if ok && pkg.Obj == nil && imports[pkg.Name] {
					seeded = true
				}
				return true
			})
		}
		if !seeded {
			matches = append(matches, fn)
		}
	}
	return matches
}

// This lexical guard covers literal shell test/[ /[[ permission probes. It
// intentionally does not infer dynamically assembled shell commands.
var fdPathPermission = regexp.MustCompile(`(?:\btest|\[\[?)\s+-[rwx]\s+["']?/dev/fd/[0-9]+`)

func fdPathPermissionChecks(file *ast.File) []ast.Node {
	var matches []ast.Node
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if ok && literal.Kind == token.STRING {
			value, err := strconv.Unquote(literal.Value)
			if err == nil && fdPathPermission.MatchString(value) {
				matches = append(matches, literal)
			}
		}
		return true
	})
	return matches
}

// The bounded entry owns individual native attempts. New callers must not
// independently recreate its attempt budget, deadline or observation policy.
func zellijReadAttemptBypasses(file *ast.File) []ast.Node {
	if file.Name.Name != "zellijquery" {
		return nil
	}
	var matches []ast.Node
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && zellijReadAttemptOwner(fn) {
			continue
		}
		ast.Inspect(decl, func(node ast.Node) bool {
			if fn, ok := decl.(*ast.FuncDecl); ok && node == fn.Name {
				return false
			}
			switch target := node.(type) {
			case *ast.SelectorExpr:
				if target.Sel.Name == "readAttempt" {
					matches = append(matches, target)
					return false // selector identifiers are not a second reference
				}
			case *ast.Ident:
				if target.Name == "readAttempt" && target.Obj == nil {
					matches = append(matches, target)
				}
			}
			return true
		})
	}
	return matches
}

func zellijReadAttemptOwner(fn *ast.FuncDecl) bool {
	if fn.Name.Name != "ReadContext" || fn.Recv == nil || len(fn.Recv.List) != 1 {
		return false
	}
	return zellijReceiver(fn, "Source")
}

func wrapperChildReleases(file *ast.File) []ast.Node {
	if file.Name.Name != "wrapcmd" {
		return nil
	}
	var matches []ast.Node
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		release, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		if !ok || release.Sel.Name != "Release" {
			return true
		}
		process, ok := ast.Unparen(release.X).(*ast.SelectorExpr)
		if ok && process.Sel.Name == "Process" {
			matches = append(matches, call)
		}
		return true
	})
	return matches
}

// This is a package-local syntax guard, not symbol or dataflow analysis. The
// named admission argument must be the direct birthBound identifier (allowing
// parentheses). Local aliases, conversions and computed bounds are out of scope.
func birthProbeAdmissionBudgets(file *ast.File) []ast.Node {
	if file.Name.Name != "launcher" {
		return nil
	}
	contextImports := map[string]bool{}
	for _, spec := range file.Imports {
		imported, err := strconv.Unquote(spec.Path.Value)
		if err != nil || imported != "context" {
			continue
		}
		name := "context"
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name != "." && name != "_" {
			contextImports[name] = true
		}
	}
	var matches []ast.Node
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		index := -1
		switch target := ast.Unparen(call.Fun).(type) {
		case *ast.Ident:
			switch target.Name {
			case "coordinateTerminalHandoff":
				index = 3
			case "CoordinateTerminalHandoff":
				index = 2
			}
		case *ast.SelectorExpr:
			qualifier, ok := ast.Unparen(target.X).(*ast.Ident)
			if ok && contextImports[qualifier.Name] && target.Sel.Name == "WithTimeout" {
				index = 1
			}
		}
		if index >= 0 && len(call.Args) > index {
			if bound, ok := ast.Unparen(call.Args[index]).(*ast.Ident); ok && bound.Name == "birthBound" {
				matches = append(matches, call)
			}
		}
		return true
	})
	return matches
}
