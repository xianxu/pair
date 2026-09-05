package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// park-ok-resume-failed is the outcome whose whole value is its sentence: the
// thread is parked, nothing is lost, and Enter brings it back. Relaunch's park
// makes the thread non-live, which invalidates the very frame whose operation
// just failed -- so the refresh that followed replaced those recovery
// instructions with "thread action is no longer applicable", on the one outcome
// that most needs them.
func TestAFailedRelaunchKeepsItsRecoveryMessageAcrossARefresh(t *testing.T) {
	address := menuAddress("brain")
	live := []couchcore.ActionableThreadSummary{{
		Address: address, WorkingPath: "/w/brain", Name: "brain", State: couchcore.ThreadLive,
	}}
	parked := []couchcore.ActionableThreadSummary{{
		Address: address, WorkingPath: "/w/brain", Name: "brain", State: couchcore.ThreadParked,
	}}
	state := NewMenuState(live, address)
	state.InventoryReady = true
	state, _ = reduceParkHotkey(state, MenuEvent{
		Kind: MenuEventParkHotkey, Operation: "relaunch", Address: address,
	})
	state, _ = reduceConfirmationKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})

	const recovery = "brain is parked and was not resumed; Enter resumes it"
	// Through ReduceMenu, which is the production seam. Driving
	// reconcileMenuFrames directly hid the real cause entirely: ReduceMenu zeroes
	// the notice near the top of the transition, so the guard downstream was
	// never reached and a direct test of it passed while the operator still lost
	// the message.
	state, _ = ReduceMenu(state, MenuEvent{
		Kind: MenuEventOperationResult, Operation: "relaunch", Attempt: state.InFlight.Attempt,
		Address: address, Success: false, Error: recovery,
	})
	if !strings.Contains(state.Notice.Text, "Enter resumes it") {
		t.Fatalf("notice after the failure = %q", state.Notice.Text)
	}

	// The refresh that follows: the thread is parked now, exactly because the
	// relaunch got that far. Two of them, because one is not a pattern.
	for round := 1; round <= 2; round++ {
		state, _ = ReduceMenu(state, MenuEvent{
			Kind: MenuEventInventory, Inventory: parked, Generation: uint64(round),
		})
		if !strings.Contains(state.Notice.Text, "Enter resumes it") {
			t.Fatalf("refresh %d erased the recovery instructions: %q", round, state.Notice.Text)
		}
	}

	// And the operator's next keystroke DOES retire it -- a message answers the
	// last thing they did, and they have now done something else.
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyDown}})
	if strings.Contains(state.Notice.Text, "Enter resumes it") {
		t.Fatalf("a keystroke did not retire the message: %q", state.Notice.Text)
	}
}

// The guard must still write bookkeeping when nothing owns the notice, or a
// genuinely stale frame goes unexplained.
func TestBookkeepingStillSpeaksWhenNoOperationOwnsTheNotice(t *testing.T) {
	state := MenuState{}
	setBookkeepingNotice(&state, "thread action is no longer applicable")
	if state.Notice.Text != "thread action is no longer applicable" {
		t.Fatalf("notice = %q", state.Notice.Text)
	}
}
