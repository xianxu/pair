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
	Name:     "remote-console-reset",
	Incident: "pair#759",
	Message:  "remote reset authority belongs only to the owned host KeyPress branch; do not export, alias or invoke it from IPC, paste or another caller",
	Match:    matchRemoteConsoleReset,
}, golint.Rule{
	Name:     "wrapper-child-release",
	Incident: "pair#715",
	Message:  "retain wrapper child ownership and join it with Wait before returning; Process.Release permits writes after wrapper shutdown",
	Match:    wrapperChildReleases,
}, golint.Rule{
	Name:     "lifecycle-failure-address",
	Incident: "pair#714",
	Message:  "compose diagnostic addresses atomically with completeFailureAddress; partial target evidence must not replace an exact captured address",
	Match:    lifecycleFailureAddressOverrides,
}, golint.Rule{
	Name:     "terminal-registry-owner",
	Incident: "pair#731",
	Message:  "use LiveOwnerIDs for native terminal ownership; kill-only LiveIDs admits a shell after exec",
	Match:    terminalRegistryOwnerCalls,
}, golint.Rule{
	Name:     "zellij-read-attempt-bypass",
	Incident: "pair#748",
	Message:  "call ZellijSource.runContext to preserve bounded read recovery, the shared deadline and per-attempt evidence; runQueryAttempt belongs only to that entry point",
	Match:    zellijReadAttemptBypasses,
}, golint.Rule{
	Name:     "controlled-zellij-close",
	Incident: "pair#764",
	Message:  "report or propagate ControlledZellij.Close cleanup errors; discarded cleanup can leave a controlled session alive",
	Match:    discardedControlledZellijCloses,
}, golint.Rule{
	Name:     "fd-path-permission",
	Incident: "pair#767",
	Message:  "prove inherited descriptor usability directly (read/fstat/fcntl), not through a /dev/fd pathname permission check; macOS can report a false negative",
	Match:    fdPathPermissionChecks,
}, golint.Rule{
	Name:     "scribe-pty-printf",
	Incident: "pair#766",
	Message:  "use a terminal-aware producer and acknowledge its readiness before final output; direct printf can fail before producing the test marker",
	Match:    scribePTYPrintfCalls,
})

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
	if file.Name.Name != "launcher" {
		return nil
	}
	var matches []ast.Node
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && zellijReadAttemptOwner(fn) {
			continue
		}
		ast.Inspect(decl, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			var name string
			switch target := ast.Unparen(call.Fun).(type) {
			case *ast.SelectorExpr:
				name = target.Sel.Name
			case *ast.Ident:
				name = target.Name
			}
			if name == "runQueryAttempt" {
				matches = append(matches, call)
			}
			return true
		})
	}
	return matches
}

func zellijReadAttemptOwner(fn *ast.FuncDecl) bool {
	if fn.Name.Name != "runContext" || fn.Recv == nil || len(fn.Recv.List) != 1 {
		return false
	}
	receiver := ast.Unparen(fn.Recv.List[0].Type)
	if pointer, ok := receiver.(*ast.StarExpr); ok {
		receiver = ast.Unparen(pointer.X)
	}
	name, ok := receiver.(*ast.Ident)
	return ok && name.Name == "ZellijSource"
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
