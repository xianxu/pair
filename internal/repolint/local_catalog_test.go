package repolint

import (
	"go/ast"

	"github.com/xianxu/ariadne/pkg/golint"
)

// Copy the shared catalog before extending it: Pair rules must not alter the
// upstream catalog used by sleepSites to identify existing sleep debt.
var pairCatalog = append(append([]golint.Rule(nil), golint.Catalog...), golint.Rule{
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
})

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
