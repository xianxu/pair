package couchcore

import (
	"strings"
	"testing"
)

// pair#847: only a non-main hub, acting on a slot it holds by an explicit
// claim, may drive it as a bench slot. One row per ownership source and caller
// role; every cell but (other hub, claim) refuses with its own reason.
func TestHubOwnershipProvisions(t *testing.T) {
	// /work/ops is main; /work/bench claims pair:2 exactly and the whole of
	// parley.nvim; /work/other claims pair:4, which bench also claims below.
	base, err := NewHubOwnership(hubFixture(t, "", "pair:2\nparley.nvim\npair:4\n", "pair:4\n"))
	if err != nil {
		t.Fatal(err)
	}
	moved, err := base.Apply(HubMove{Slots: []HubSlot{{"/work/pair", 6}}, Caller: "/work/ops", Target: "/work/bench", Revision: base.Revision()})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, caller string
		target       HubSlot
		refusal      string
	}{
		{"exact claim", "/work/bench", HubSlot{"/work/pair", 2}, ""},
		{"whole-repo claim", "/work/bench", HubSlot{"/work/parley.nvim", 3}, ""},
		{"main hub", "/work/ops", HubSlot{"/work/pair", 2}, "main hub"},
		{"default ownership", "/work/bench", HubSlot{"/work/pair", 3}, "not claimed"},
		{"own hub slot", "/work/bench", HubSlot{"/work/bench", 0}, "its own"},
		{"runtime move", "/work/bench", HubSlot{"/work/pair", 6}, "moved"},
		{"conflicting claims", "/work/bench", HubSlot{"/work/pair", 4}, "conflict"},
		{"not a hub", "/work/pair-slot1", HubSlot{"/work/pair", 2}, "not a hub"},
		{"another hub's claim", "/work/other", HubSlot{"/work/pair", 2}, "not claimed"},
	} {
		t.Run(c.name, func(t *testing.T) {
			state := base
			if c.name == "runtime move" {
				state = moved
			}
			err := state.Provisions(c.caller, c.target)
			if c.refusal == "" {
				if err != nil {
					t.Fatalf("refused: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.refusal) {
				t.Fatalf("want refusal %q, got %v", c.refusal, err)
			}
		})
	}
}
