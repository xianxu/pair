package couchcore

import "testing"

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
