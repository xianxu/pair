package couchtty

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// TestActionOfferedImpliesPermitted is the class guard for #256: the switcher
// offers an action on one set of states, a guard permits it on a different set,
// and the two are kept in agreement by two authors remembering to.
//
// It began as switch-agent's own table (M2, BR-33) and M3 widened it to every
// action that has a classification-level admission rule, because the drift was
// never specific to switching. The defects it exists to catch, all three
// observed in this issue: `parked` gained a producer the switch guard's receipt
// check refused; a replacement guard admitted a `live` row couch was hosting;
// and an unresolved probe put an archive item on a row whose own archive path
// refuses every time it is pressed.
//
// The domain is DERIVED from the vocabularies (`AllThreadStates` x
// `AllThreadReasons`), not hand-listed, so a new state or reason lands here
// automatically. Offered must imply permitted; permitted may be wider, because
// these predicates also answer callers that reach the API without a row.
func TestActionOfferedImpliesPermitted(t *testing.T) {
	// resumeAdmitted is resume's admission as ResumeTarget's route states it:
	// ResumableState, plus every unusable row, because the route hands those
	// to OpenSlot, RecoverThread or RetryContinuation, each with its own
	// refusal -- and an unknown classification is refused by the table, not
	// here.
	resumeAdmitted := func(state couchcore.ActionableThreadState, reason couchcore.ThreadReason) bool {
		return couchcore.ResumableState(state, reason) || state == couchcore.ThreadUnusable
	}
	actions := []struct {
		item      string
		permitted func(couchcore.ActionableThreadState, couchcore.ThreadReason) bool
	}{
		{"switch-agent", couchcore.SwitchableState},
		{"resume", resumeAdmitted},
		{"reboot", couchcore.RebootableState},
	}
	everOffered := make([]bool, len(actions))
	// The domain is the action table's own (everyMenuRowShape): derived from
	// AllThreadStates x AllThreadReasons x AllPhases over both kinds, so a new
	// state, reason or phase lands here automatically. `archived` is left out
	// there because ProjectActionableThreads cannot produce it (couchcore's
	// TestProjectionNeverProducesArchived).
	for _, shape := range everyMenuRowShape(t) {
		offered := map[string]bool{}
		for _, item := range menuActionItems(shape.row) {
			offered[item] = true
		}
		for i, action := range actions {
			if !offered[action.item] {
				continue
			}
			everOffered[i] = true
			if !action.permitted(shape.state, shape.reason) {
				t.Errorf("%s: the switcher offers %s and the guard refuses it — "+
					"an action that always fails is how a switcher teaches an operator to distrust it",
					shape.name, action.item)
			}
		}
	}
	// NON-VACUOUS. Offered-implies-permitted is satisfied by offering nothing,
	// so a menu change that dropped an item everywhere would leave this table
	// green while deleting the behaviour it exists to constrain.
	for i, action := range actions {
		if !everOffered[i] {
			t.Errorf("no row offers %s at all; this table is passing by vacuity", action.item)
		}
	}
}
