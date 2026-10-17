package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"testing"
)

func TestRecoverParkMenuConfirmsAndBindsExactTransaction(t *testing.T) {
	for _, row := range []couchcore.ActionableThreadSummary{{Address: menuAddress("ordinary"), State: couchcore.ThreadUnusable}, menuSlotRow(2, "slot-thread")} {
		row.HasOpenPark = true
		row.ParkNonce = "original-park"
		items := menuActionItems(row)
		if len(items) != 1 || items[0] != "recover-park" {
			t.Fatalf("actions %v", items)
		}
		state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
		state.InventoryReady = true
		state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
		state.Frames[len(state.Frames)-1].SelectedItem = "recover-park"
		next, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
		if len(effects) != 0 || next.CurrentFrame().Kind != MenuFrameConfirmation {
			t.Fatalf("must confirm: %+v %v", next.CurrentFrame(), effects)
		}
		next.Frames[len(next.Frames)-1].SelectedItem = "recover-park"
		_, effects = reduceKey(next, PanelKey{Kind: KeyEnter})
		if len(effects) != 1 {
			t.Fatalf("effects %v", effects)
		}
		args := effects[0].Args
		if effects[0].Operation != "recover-park" || args["tag"] != string(row.Address.Tag) || args["repo-scope"] != row.Address.RepoScope || args["park-nonce"] != "original-park" || args["path"] != "" {
			t.Fatalf("unbound recovery: %+v", effects[0])
		}
	}
	if got := menuItemLabel("recover-park"); got != "Recover park" {
		t.Fatalf("label %q", got)
	}
}

func TestRecoverParkConfirmationRefusesReplacementTransaction(t *testing.T) {
	row := couchcore.ActionableThreadSummary{Address: menuAddress("ordinary"), State: couchcore.ThreadUnusable, HasOpenPark: true, ParkNonce: "first"}
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state.InventoryReady = true
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "recover-park"
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	for i := range state.Inventory {
		state.Inventory[i].ParkNonce = "replacement"
	}
	state.Frames[len(state.Frames)-1].SelectedItem = "recover-park"
	_, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 {
		t.Fatalf("confirmed replacement transaction: %+v", effects)
	}
}
