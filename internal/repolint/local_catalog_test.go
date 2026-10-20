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
	Name:     "drain-state-reducer",
	Incident: "pair#722",
	Message:  "change drain phase and error through couchcore.ReduceDrain; direct coordinator state writes bypass reducer authority",
	Match:    drainStateOverrides,
})

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
