package couchcore

import (
	"strings"
	"testing"
)

func TestHubMoveBatchIsAtomic(t *testing.T) {
	state, _ := NewHubOwnership(hubFixture(t, "", "pair:7\n", "pair:8\n"))
	slots := []HubSlot{{"/work/pair", 7}, {"/work/pair", 8}}
	next, err := state.Apply(HubMove{Slots: slots, Caller: "/work/bench", Target: "/work/ops", Revision: state.Revision()})
	if err == nil {
		t.Fatal("mixed authority batch accepted")
	}
	if next.Resolve(slots[0]).Hub != "/work/bench" || next.Revision() != state.Revision() {
		t.Fatal("partial mutation")
	}
}
func TestHubClaimsSatisfiedEditPreservesPosition(t *testing.T) {
	hubs := hubFixture(t, "", "pair:7\n# keep next\nparley.nvim\n")
	edits, err := PlanHubClaimEdits(hubs, []HubSlot{{"/work/pair", 7}}, "/work/bench")
	if err != nil || len(edits) != 0 {
		t.Fatalf("already satisfied file changed: %+v %v", edits, err)
	}
}
func TestHubClaimCopiesCannotMutateOwnership(t *testing.T) {
	state, _ := NewHubOwnership(hubFixture(t, "", "pair\n"))
	hubs := state.Hubs()
	claims := hubs[1].Claims.Claims()
	claims[0].Repository = "evil"
	hubs[1].Root = "/evil"
	raw := state.Hubs()[1].Claims.Bytes()
	raw[0] = 'x'
	if got := state.Resolve(HubSlot{"/work/pair", 1}).Hub; got != "/work/bench" {
		t.Fatal(got)
	}
}
func TestHubClaimsRevisionPreservesRuntimeMoves(t *testing.T) {
	state, _ := NewHubOwnership(hubFixture(t, "", "", ""))
	state, err := state.Apply(HubMove{Slots: []HubSlot{{"/work/pair", 7}}, Caller: "/work/bench", Target: "/work/bench", Revision: state.Revision()})
	if err != nil {
		t.Fatal(err)
	}
	hubs := state.Hubs()
	hubs[2].Claims, _ = ParseHubClaims([]byte("pair\n"))
	next, err := state.WithClaims(hubs, state.Revision())
	if err != nil {
		t.Fatal(err)
	}
	if next.Resolve(HubSlot{"/work/pair", 7}).Hub != "/work/bench" || next.Resolve(HubSlot{"/work/pair", 8}).Hub != "/work/other" {
		t.Fatal("lost runtime or durable claims")
	}
	if _, err := next.WithClaims(hubs, state.Revision()); err == nil {
		t.Fatal("stale claims accepted")
	}
}

func TestHubGeneratedClaimEditsConvergeWithoutChangingOtherSlots(t *testing.T) {
	variants := []string{"# comment\n", "# comment\npair\n", "# comment\npair:7\n"}
	for _, a := range variants {
		for _, b := range variants {
			for _, c := range variants {
				for target := 0; target < 3; target++ {
					defs := hubFixture(t, a, b, c)
					before, err := NewHubOwnership(defs)
					if err != nil {
						t.Fatal(err)
					}
					selected := HubSlot{Repository: "/work/pair", Number: 7}
					other := HubSlot{Repository: "/work/pair", Number: 8}
					wantOther := before.Resolve(other).Hub
					edits, err := PlanHubClaimEdits(defs, []HubSlot{selected}, defs[target].Root)
					if err != nil {
						t.Fatal(err)
					}
					for _, edit := range edits {
						for i := range defs {
							if defs[i].Root == edit.Root {
								defs[i].Claims, err = ParseHubClaims(edit.After)
								if err != nil {
									t.Fatal(err)
								}
							}
						}
					}
					after, err := NewHubOwnership(defs)
					if err != nil {
						t.Fatal(err)
					}
					if after.Resolve(selected).Hub != defs[target].Root || after.Resolve(other).Hub != wantOther {
						t.Fatalf("claims %q %q %q target %d changed unrelated slot", a, b, c, target)
					}
					again, err := PlanHubClaimEdits(defs, []HubSlot{selected}, defs[target].Root)
					if err != nil || len(again) != 0 {
						t.Fatalf("not convergent: %+v %v", again, err)
					}
				}
			}
		}
	}
}

func TestHubMoveCapacityRefusesWholeBatchAndNamesRecovery(t *testing.T) {
	state, err := NewHubOwnership(hubFixture(t, "", ""))
	if err != nil {
		t.Fatal(err)
	}
	slots := make([]HubSlot, 128)
	for i := range slots {
		slots[i] = HubSlot{Repository: "/work/pair", Number: i}
	}
	state, err = state.Apply(HubMove{Caller: "/work/bench", Target: "/work/bench", Revision: state.Revision(), Slots: slots})
	if err != nil {
		t.Fatal(err)
	}
	extra := HubSlot{Repository: "/work/pair", Number: 128}
	next, err := state.Apply(HubMove{Caller: "/work/bench", Target: "/work/bench", Revision: state.Revision(), Slots: []HubSlot{extra}})
	if err == nil || !strings.Contains(err.Error(), "restart Couch") {
		t.Fatalf("capacity recovery: %v", err)
	}
	if next.Revision() != state.Revision() || next.Resolve(extra).Hub != "/work/ops" {
		t.Fatal("capacity refusal partly moved slot")
	}
}
