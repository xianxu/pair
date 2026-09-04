package couchtty

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// Relaunch parks before it resumes, so a refresh landing mid-operation sees the
// thread NOT live -- because of the very operation in flight. Judging the
// confirmation by that reading reported "thread action is no longer applicable"
// over a relaunch that went on to succeed, and the operator watched their thread
// vanish from the status bar and come back under an error.
func TestInFlightOperationKeepsItsConfirmationWhileItsThreadIsNotLive(t *testing.T) {
	address := menuAddress("brain")
	live := []couchcore.ActionableThreadSummary{{
		Address: address, WorkingPath: "/w/brain", Name: "brain", State: couchcore.ThreadLive,
	}}
	// Mid-relaunch: parked, which is exactly "not live".
	parked := []couchcore.ActionableThreadSummary{{
		Address: address, WorkingPath: "/w/brain", Name: "brain", State: couchcore.ThreadParked,
	}}

	openConfirmation := func() MenuState {
		state := NewMenuState(live, address)
		state, _ = reduceParkHotkey(state, MenuEvent{
			Kind: MenuEventParkHotkey, Operation: "relaunch", Address: address,
		})
		state, _ = reduceConfirmationKey(state, PanelKey{Kind: KeyDown})
		state, _ = reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
		if state.InFlight.Operation != "relaunch" {
			t.Fatalf("relaunch did not go in flight: %+v", state.InFlight)
		}
		return state
	}

	t.Run("in flight, the confirmation survives", func(t *testing.T) {
		state := openConfirmation()
		state.Inventory = parked
		state = reconcileMenuFrames(state, live)
		if frame := state.CurrentFrame(); frame.Kind != MenuFrameConfirmation || frame.Action != "relaunch" {
			t.Fatalf("mid-relaunch refresh discarded the confirmation: frame = %v/%q notice = %q",
				frame.Kind, frame.Action, state.Notice.Text)
		}
	})

	// The guard still has to work: once nothing is in flight, a confirmation
	// for a thread that stopped being live IS stale.
	t.Run("not in flight, a dead thread's confirmation is discarded", func(t *testing.T) {
		state := openConfirmation()
		state.InFlight = MenuOperationOrigin{}
		state.Inventory = parked
		state = reconcileMenuFrames(state, live)
		if frame := state.CurrentFrame(); frame.Kind == MenuFrameConfirmation {
			t.Fatalf("stale confirmation survived with nothing in flight: %+v", frame)
		}
		if state.Notice.Text != "thread action is no longer applicable" {
			t.Fatalf("notice = %q", state.Notice.Text)
		}
	})
}

// The exemption must be no wider than its rationale. reduceActionKey has no
// in-flight guard, so a confirmation for a DIFFERENT action can be open on the
// same thread while an operation runs; matching on address alone exempted that
// one too, which is broader than "the frame whose own operation is running".
func TestTheInFlightExemptionDoesNotCoverAnotherActionsConfirmation(t *testing.T) {
	address := menuAddress("brain")
	live := []couchcore.ActionableThreadSummary{{
		Address: address, WorkingPath: "/w/brain", Name: "brain", State: couchcore.ThreadLive,
	}}
	parked := []couchcore.ActionableThreadSummary{{
		Address: address, WorkingPath: "/w/brain", Name: "brain", State: couchcore.ThreadParked,
	}}
	state := NewMenuState(live, address)
	state, _ = reduceParkHotkey(state, MenuEvent{
		Kind: MenuEventParkHotkey, Operation: "relaunch", Address: address,
	})
	state, _ = reduceConfirmationKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
	if state.InFlight.Operation != "relaunch" {
		t.Fatalf("relaunch did not go in flight: %+v", state.InFlight)
	}
	// A park confirmation on the SAME thread, while relaunch is the operation
	// actually running.
	state.Frames = state.Frames[:1]
	appendMenuFrame(&state, MenuFrame{
		Kind: MenuFrameConfirmation, Action: "park", Thread: address, SelectedItem: "cancel",
	})

	state.Inventory = parked
	state = reconcileMenuFrames(state, live)

	if frame := state.CurrentFrame(); frame.Kind == MenuFrameConfirmation && frame.Action == "park" {
		t.Fatal("a park confirmation was exempted by relaunch being in flight; the exemption is wider than its rationale")
	}
}

// The fallback exists so a third operation joining the guard cannot produce
// "only a running thread can be " with nothing after it.
func TestPastParticipleHasAWordForAnOperationItWasNotWrittenFor(t *testing.T) {
	for operation, want := range map[string]string{
		"park":     "parked",
		"relaunch": "relaunched",
		"archive":  "archiveed",
	} {
		if got := pastParticiple(operation); got != want {
			t.Errorf("pastParticiple(%q) = %q, want %q", operation, got, want)
		}
	}
	if pastParticiple("") == "" {
		t.Error("an unknown operation yielded an empty word, which is the silence the fallback exists to prevent")
	}
}
