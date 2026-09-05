package couchtty

import "github.com/xianxu/pair/cmd/internal/couchcore"

// SwitchTracker is the whole of couch's `previous` behaviour: one slot, and one
// boolean that keeps a notification hop from spending it.
//
// One slot, not a stack, deliberately: a stack the operator cannot see is a
// stack they lose track of. The cost of that choice is that `ctrl+backspace`
// out of a notification actor lands home with previous == current, so the next
// press is a no-op -- which is the right answer, because the operator is home
// and there is nowhere to bounce to.
//
// Keyed by ThreadAddress rather than the console-local actor id: a park/resume
// or detach/reattach cycle mints a new actor id for the same durable thread,
// and `previous` has to survive that.
type SwitchTracker struct {
	current  couchcore.ThreadAddress
	previous couchcore.ThreadAddress

	// currentViaNotification is the Spec's `entered_via_notification`, carried
	// on the CURRENT actor rather than passed at return time. That placement is
	// the whole trick: the decision "does leaving here cost the operator their
	// place" belongs to the actor being left, and is known when it was entered.
	currentViaNotification bool
}

// Switch records a landing.
//
// viaNotification is true only when the operator arrived by ctrl-space + Return
// on an actor that had a pending notification. Such an actor never becomes
// `previous`, so chasing two pages -- or detouring manually to spot-check a
// third actor -- still leaves ctrl+backspace pointing at the actor the operator
// was actually working in.
//
// Every consequence in the issue Spec falls out of these three lines; none is
// special-cased. Callers must route EVERY landing through here, whatever the
// mechanism, or the invariant is only true for the paths that remembered.
func (t *SwitchTracker) Switch(target couchcore.ThreadAddress, viaNotification bool) {
	if target == t.current {
		// Landing on the actor already current is a repaint, not a switch --
		// returning from the panel to where you were, or a redundant Enter.
		// Recording it would copy `current` into `previous` and spend the slot
		// on a no-op, losing the actor the operator could actually go back to.
		return
	}
	if !t.currentViaNotification {
		t.previous = t.current
	}
	t.current = target
	t.currentViaNotification = viaNotification
}

// Drop forgets a thread that is gone -- an exited child, whose pane the console
// has just removed.
//
// It is not a Switch: on exit the operator lands on the panel, not on another
// actor, so recording a landing would make the dead thread the return target --
// the one place ctrl+backspace can never usefully go. Dropping `current`
// deliberately does not promote it to `previous` either, for the same reason.
func (t *SwitchTracker) Drop(address couchcore.ThreadAddress) {
	if t.previous == address {
		t.previous = couchcore.ThreadAddress{}
	}
	if t.current == address {
		t.current = couchcore.ThreadAddress{}
		// A dead actor cannot be a notification hop worth protecting, and
		// leaving the flag set would silently swallow the NEXT switch's
		// previous.
		t.currentViaNotification = false
	}
}

// Previous names the actor ctrl+backspace returns to, if there is one.
func (t *SwitchTracker) Previous() (couchcore.ThreadAddress, bool) {
	return t.previous, t.previous != couchcore.ThreadAddress{}
}

// Current names the actor the operator is on, empty when that is nothing.
func (t *SwitchTracker) Current() couchcore.ThreadAddress { return t.current }
