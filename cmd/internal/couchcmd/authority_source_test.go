package couchcmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// Published authority is immutable. Enumerate direct fixture writes so a rare
// admission interleaving cannot hide a new instance from the race suite (#860).
func TestFixtureAuthorityConfiguredBeforeReaders(t *testing.T) {
	// These exact writes have no connected wrappers: a no-loop service literal,
	// a cold-start/prewarm contrast, pre-wrapper lifecycle setup, and an injected
	// in-memory registry. Each exception admits exactly one write, not a file.
	cold := map[string]string{
		"reap_proven_test.go:TestProvenOrphanSignalHoldsHubOwnershipThroughSingleEffect:hubs":    "no-loop service",
		"hub_resume_batch_acceptance_test.go:TestHubResumeBatchSocketAcceptance:hubs":            "cold startup contrast",
		"lifecycle_notices_test.go:TestLifecyclePublisherDropsMovedOwnership:hubs":               "pre-wrapper setup",
		"worker_cue_coordinator_test.go:TestWorkerCueCoordinatorCountedTwentyFortyWorkers:child": "in-memory registry",
	}
	seen := map[string]int{}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		positions := token.NewFileSet()
		file, err := parser.ParseFile(positions, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			connected := false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						switch sel.Sel.Name {
						case "wrapper", "v2wrapper", "connect", "waitConnected":
							connected = true
						}
					}
				}
				assign, ok := n.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for _, lhs := range assign.Lhs {
					field, ok := lhs.(*ast.SelectorExpr)
					if !ok || (field.Sel.Name != "child" && field.Sel.Name != "hub" && field.Sel.Name != "hubs") {
						continue
					}
					owner, ok := field.X.(*ast.SelectorExpr)
					if !ok || owner.Sel.Name != "authority" {
						continue // Local authority values compose safely before construction.
					}
					key := name + ":" + fn.Name.Name + ":" + field.Sel.Name
					seen[key]++
					if _, allowed := cold[key]; !allowed || connected || seen[key] > 1 {
						t.Errorf("%s: configure %s through decorateAuthority before service construction", positions.Position(lhs.Pos()), field.Sel.Name)
					}
				}
				return true
			})
		}
	}
	for key := range cold {
		if seen[key] != 1 {
			t.Errorf("stale cold-fixture exception %s: writes=%d, want 1", key, seen[key])
		}
	}
}
