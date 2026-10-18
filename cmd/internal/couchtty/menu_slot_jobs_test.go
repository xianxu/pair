package couchtty

import (
	"strings"
	"testing"
)

func TestMenuBackgroundOperationAllowsOtherSlot(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, first := dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	if len(first) != 1 {
		t.Fatal("first operation not admitted")
	}
	_, second := dispatchThreadOperation(state, "switch", menuAddress("couch-two"))
	if len(second) != 1 || second[0].Operation != "switch" {
		t.Fatalf("running relaunch blocked another slot: %+v", second)
	}
}

func TestMenuBusyOperationExplainsSameSlotRefusal(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	next, effects := dispatchThreadOperation(state, "detach", menuAddress("couch-one"))
	if len(effects) != 0 || !strings.Contains(next.Notice.Text, "busy: relaunch") {
		t.Fatalf("same-slot refusal must name the owner: effects=%+v notice=%+v", effects, next.Notice)
	}
}

func TestMenuBackgroundOperationReturnsToRootImmediately(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	appendMenuFrame(&state, MenuFrame{Kind: MenuFrameConfirmation, Thread: menuAddress("couch-one"), Action: "relaunch", SelectedItem: "confirm"})
	next, effects := dispatchThreadOperation(state, "relaunch", menuAddress("couch-one"))
	if len(effects) != 1 || next.CurrentFrame().Kind != MenuFrameRoot {
		t.Fatalf("admitted lifecycle action must return to usable root: frame=%+v effects=%+v", next.CurrentFrame(), effects)
	}
}
