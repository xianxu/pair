package storagegc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func TestReduceTransactionPhaseEventMatrix(t *testing.T) {
	// Independent contract: detachment may only advance prepared authority;
	// owner retirement requires detached authority. Repeated completed steps are
	// idempotent, but a finalized transaction may never reauthorize source work.
	allowed := map[string]map[CollectionEvent]string{
		"prepared":  {CollectionDetachmentProved: "detached"},
		"detached":  {CollectionDetachmentProved: "detached", CollectionOwnerRetired: "finalized"},
		"finalized": {CollectionOwnerRetired: "finalized"},
	}
	for _, phase := range []string{"", "prepared", "detached", "finalized", "detaching", "cleaned"} {
		for _, event := range []CollectionEvent{CollectionDetachmentProved, CollectionOwnerRetired, "unknown"} {
			before := CollectionTransaction{Version: 1, ID: "fixed-authority", Owner: artifactpath.StorageOwner{DataDir: "/fixture", Tag: "tag"}, Incarnation: "incarnation-a", Bucket: artifactpath.SessionRetention, Phase: phase, Entries: []CollectionEntry{{Source: "original", Destination: "0", Top: true}}, Archives: []ArchiveReference{{Store: "/other-filesystem"}}}
			next, err := ReduceTransaction(before, event)
			want, ok := allowed[phase][event]
			if (err == nil) != ok {
				t.Fatalf("phase=%s event=%s: %v", phase, event, err)
			}
			if !ok {
				if !reflect.DeepEqual(before, next) {
					t.Fatal("rejected event changed authority")
				}
				continue
			}
			if next.Phase != want {
				t.Fatalf("phase=%s event=%s => %s want %s", phase, event, next.Phase, want)
			}
			next.Phase = before.Phase
			if !reflect.DeepEqual(before, next) {
				t.Fatal("transition changed frozen deletion authority")
			}
		}
	}
}

func TestReduceTransactionSequencesNeverRegress(t *testing.T) {
	rank := map[string]int{"prepared": 0, "detached": 1, "finalized": 2}
	var visit func(CollectionTransaction, int)
	visit = func(current CollectionTransaction, remaining int) {
		if remaining == 0 {
			return
		}
		for _, event := range []CollectionEvent{CollectionDetachmentProved, CollectionOwnerRetired, "invalid"} {
			next, err := ReduceTransaction(current, event)
			if err != nil {
				if !reflect.DeepEqual(current, next) {
					t.Fatal("rejection mutated state")
				}
				continue
			}
			if rank[next.Phase] < rank[current.Phase] {
				t.Fatal("transition can revisit source names")
			}
			visit(next, remaining-1)
		}
	}
	visit(CollectionTransaction{Phase: "prepared"}, 8)
}

// Phase mutation outside the reducer would turn the pure model into decoration.
// This AST guard covers every production source, including future adapters.
func TestCollectionPhaseWritesOnlyThroughReducer(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	transitions := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		tree, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(tree, func(node ast.Node) bool {
			if path != "transaction_model.go" {
				if assignment, ok := node.(*ast.AssignStmt); ok {
					for _, lhs := range assignment.Lhs {
						if field, ok := lhs.(*ast.SelectorExpr); ok && field.Sel.Name == "Phase" {
							t.Errorf("%s directly mutates transaction phase", path)
						}
					}
				}
			}
			if call, ok := node.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "ReduceTransaction" {
					transitions++
				}
			}
			return true
		})
	}
	if transitions != 1 {
		t.Fatalf("expected one enforced reducer adapter, got %d", transitions)
	}
}
