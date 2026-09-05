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
	address := menuAddress("brain")
	for _, row := range []struct {
		name  string
		state couchcore.ActionableThreadState
	}{
		{"live", couchcore.ThreadLive},
		{"parked", couchcore.ThreadParked},
		{"busy", couchcore.ThreadBusy},
		{"unusable", couchcore.ThreadUnusable},
	} {
		t.Run(row.name, func(t *testing.T) {
			thread := couchcore.ActionableThreadSummary{
				Address: address, WorkingPath: "/w/brain", Name: "brain", State: row.state,
			}
			offered := menuActionItems(thread)
			if len(offered) == 0 {
				t.Fatalf("%s row offers no actions", row.name)
			}
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
						(next.CurrentFrame().Kind == MenuFrameConfirmation || next.CurrentFrame().Kind == MenuFrameText)
					if !dispatched && !descended {
						t.Fatalf("Enter on %q did nothing: frames %d→%d, notice %q",
							action, len(state.Frames), len(next.Frames), next.Notice.Text)
					}
					// And what it did must match the DECLARATION, so the two
					// cannot drift apart again.
					if confirms, declared := couchcore.OperationConfirms(action); declared && confirms {
						if next.CurrentFrame().Kind != MenuFrameConfirmation {
							t.Errorf("%q declares ConfirmRequired but Enter did not confirm: %v",
								action, next.CurrentFrame().Kind)
						}
					}
				})
			}
		})
	}
}

// The direction the plan actually asked for, and the one the sweep above cannot
// give. Offered-implies-reachable catches "this row offers an action that does
// nothing"; it is blind to "this operation is declared and no row offers it",
// which is the failure the guard was written for. Both directions, so membership
// has one source of truth (Operation.RowAction) instead of two lists that agree
// until someone adds to one.
func TestRowActionDeclarationsAndTheMenuAgreeInBothDirections(t *testing.T) {
	offered := map[string]bool{}
	for _, state := range []couchcore.ActionableThreadState{
		couchcore.ThreadLive, couchcore.ThreadParked, couchcore.ThreadBusy, couchcore.ThreadUnusable,
		couchcore.ThreadDetached,
	} {
		for _, action := range menuActionItems(couchcore.ActionableThreadSummary{
			Address: menuAddress("brain"), WorkingPath: "/w/brain", Name: "brain", State: state,
		}) {
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
	for name := range offered {
		if !declared[name] {
			t.Errorf("the switcher offers %q on a row, but it does not declare RowAction", name)
		}
	}
}
