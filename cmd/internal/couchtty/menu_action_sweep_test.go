package couchtty

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// EVERY action a row offers must do something when the operator presses Enter
// on it. This is the sweep, not a case: relaunch was added to menuActionItems
// and to no arm of reduceActionKey's switch, so it appeared in the list and
// Enter on it fell through and did nothing at all -- silently, which is the way
// an operator loses trust in a menu. Adding a sixth action to the list without
// making it reachable now fails here rather than in a smoke test.
func TestEveryOfferedActionIsReachableFromEnter(t *testing.T) {
	everOffered := 0
	for _, shape := range everyMenuRowShape(t) {
		thread, address := shape.row, shape.row.Address
		t.Run(shape.name, func(t *testing.T) {
			offered := menuActionItems(thread)
			everOffered += len(offered)
			for _, action := range offered {
				t.Run(action, func(t *testing.T) {
					state := NewMenuState([]couchcore.ActionableThreadSummary{thread}, address)
					state, _ = reduceRootKey(state, PanelKey{Kind: KeyTab})
					if state.CurrentFrame().Kind != MenuFrameActions {
						t.Fatalf("Tab did not open the action list: %v", state.CurrentFrame().Kind)
					}
					frame := &state.Frames[len(state.Frames)-1]
					frame.SelectedItem = action

					next, effects := reduceActionKey(state, PanelKey{Kind: KeyEnter})
					// Reachable means one of exactly three things happened: it
					// dispatched, it opened a confirmation, or it opened a text
					// form. Anything else is the silent fall-through.
					dispatched := len(effects) > 0 || next.InFlight.Operation != ""
					descended := len(next.Frames) > len(state.Frames) &&
						(next.CurrentFrame().Kind == MenuFrameConfirmation || next.CurrentFrame().Kind == MenuFrameText || next.CurrentFrame().Kind == MenuFrameSwitchAgent)
					if !dispatched && !descended {
						t.Fatalf("Enter on %q did nothing: frames %d→%d, notice %q",
							action, len(state.Frames), len(next.Frames), next.Notice.Text)
					}
					// And what it did must match the DECLARATION, so the two
					// cannot drift apart again.
					if confirms, declared := couchcore.OperationConfirms(action); declared && confirms {
						if next.CurrentFrame().Kind != MenuFrameConfirmation && next.CurrentFrame().Kind != MenuFrameSwitchAgent {
							t.Errorf("%q declares ConfirmRequired but Enter did not confirm: %v",
								action, next.CurrentFrame().Kind)
						}
					}
				})
			}
		})
	}
	if everOffered == 0 {
		t.Fatal("no row offers anything; this sweep is passing by vacuity")
	}
}

// The direction the plan actually asked for, and the one the sweep above cannot
// give. Offered-implies-reachable catches "this row offers an action that does
// nothing"; it is blind to "this operation is declared and no row offers it",
// which is the failure the guard was written for. Both directions, so membership
// has one source of truth (Operation.RowAction) instead of two lists that agree
// until someone adds to one.
func TestRowActionDeclarationsAndTheMenuAgreeInBothDirections(t *testing.T) {
	// The SAME derived domain the action table is tested over, not a
	// hand-picked set of rows: a row shape this sweep never built is a row
	// whose offer it could not check.
	offered := map[string]bool{}
	for _, shape := range everyMenuRowShape(t) {
		for _, action := range menuActionItems(shape.row) {
			offered[action] = true
		}
	}
	// Read straight off the declaration. A helper here would need a production
	// caller to survive the dead-symbol guard, and the only honest one -- having
	// menuActionItems filter through it -- is exactly what made this test
	// unfalsifiable last round.
	declared := map[string]bool{}
	for _, op := range couchcore.Operations() {
		if op.RowAction {
			declared[op.Name] = true
		}
	}

	for name := range declared {
		if !offered[name] {
			t.Errorf("%q declares RowAction but no row state offers it — declared and unreachable", name)
		}
	}
	// add-slot is the switcher's own affordance, not an operation: it opens
	// the start form, whose submit is the declared `start`. It is reachable
	// (TestEveryOfferedActionIsReachableFromEnter) and has nothing to declare.
	menuLocal := map[string]bool{"add-slot": true}
	for name := range offered {
		if !declared[name] && !menuLocal[name] {
			t.Errorf("the switcher offers %q on a row, but it does not declare RowAction", name)
		}
	}
}

// endsItsOwnChild names the operations whose child exit is EXPECTED, so the two
// sites that need the answer cannot disagree. It shipped in pair#182 with its
// only _test.go occurrence being the concept inventory's own literal, which is
// why the coverage assertion was passing vacuously.
func TestEndsItsOwnChildNamesTheDeliberateOnes(t *testing.T) {
	for _, operation := range []string{"park", "detach", "relaunch", "reap", "recover"} {
		if !endsItsOwnChild(operation) {
			t.Errorf("%q deliberately ends its child but is not named, so its exit raises a spurious notice", operation)
		}
	}
	for _, operation := range []string{"switch", "resume", "reboot", "alias", "leave", ""} {
		if endsItsOwnChild(operation) {
			t.Errorf("%q does not end its own child, so marking its exit expected would SWALLOW a real one", operation)
		}
	}
}

// Manual marks a dispatch that HAPPENED. dispatchThreadOperation refuses while
// another operation is in flight and returns the state unchanged, so marking
// unconditionally would set Manual on somebody else's operation -- and that
// operation would then skip its attention capture and be misclassified as an
// ordinary landing.
func TestAClickRefusedMidOperationDoesNotMarkSomeoneElsesWork(t *testing.T) {
	address := menuAddress("one")
	state := NewMenuState([]couchcore.ActionableThreadSummary{{
		Address: address, WorkingPath: "/w/one", Name: "one", State: couchcore.ThreadLive,
	}}, address)

	// Something else is already in flight, so the click's dispatch is refused.
	state.InFlight = MenuOperationOrigin{Operation: "park", Attempt: 7, Address: address}
	next, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventMouseSwitch, Address: address})

	if len(effects) != 0 {
		t.Fatalf("the click dispatched %d effect(s) while park was in flight", len(effects))
	}
	if next.InFlight.Operation != "park" || next.InFlight.Attempt != 7 {
		t.Fatalf("the click displaced the in-flight operation: %+v", next.InFlight)
	}
	if next.InFlight.Manual {
		t.Error("the refused click marked the in-flight park as manual, so it will skip its attention capture")
	}
}
