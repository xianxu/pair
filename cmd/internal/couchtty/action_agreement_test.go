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
	actions := []struct {
		item      string
		permitted func(couchcore.ActionableThreadState, couchcore.ThreadReason) bool
	}{
		{"switch-agent", couchcore.SwitchableState},
		{"archive", couchcore.ArchivableState},
		{"resume", couchcore.ResumableState},
	}
	for _, state := range couchcore.AllThreadStates() {
		// `archived` is a state of the ARCHIVED inventory, which the switcher
		// does not render: ProjectActionableThreads cannot produce it, pinned
		// by couchcore's TestProjectionNeverProducesArchived. Asking what a row
		// offers for a row that cannot exist would force every predicate to
		// carry an answer no operator can reach.
		if state == couchcore.ThreadArchived {
			continue
		}
		for _, reason := range append(couchcore.AllThreadReasons(), "") {
			// The projection sets a reason exactly when the state is unusable;
			// anything else is not a row the classifier can produce.
			if (state == couchcore.ThreadUnusable) != (reason != "") {
				continue
			}
			row := couchcore.ActionableThreadSummary{
				Address: couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000001"},
				State:   state, Reason: reason,
			}
			offered := map[string]bool{}
			for _, item := range menuActionItems(row) {
				offered[item] = true
			}
			for _, action := range actions {
				if offered[action.item] && !action.permitted(state, reason) {
					t.Errorf("%s/%s: the switcher offers %s and the guard refuses it — "+
						"an action that always fails is how a switcher teaches an operator to distrust it",
						state, reason, action.item)
				}
			}
		}
	}
}

// A row carrying a recovery offer is the OTHER shape menuActionItems answers
// for, and it is the one that reaches an unusable row: ProjectRecoveryChoices
// fills it in for `session-gone` and for any record with a retained
// continuation, so the two branches offer archive for different reasons and
// only one of them was swept when the unknown case was found.
func TestRecoveryRowArchiveOfferedImpliesPermitted(t *testing.T) {
	for _, reason := range couchcore.AllThreadReasons() {
		row := couchcore.ActionableThreadSummary{
			Address:  couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000001"},
			State:    couchcore.ThreadUnusable,
			Reason:   reason,
			Recovery: &couchcore.RecoveryDecision{Recover: true, Archive: true},
		}
		offered := false
		for _, item := range menuActionItems(row) {
			if item == "archive" {
				offered = true
			}
		}
		if offered && !couchcore.ArchivableState(row.State, reason) {
			t.Errorf("unusable/%s with a recovery offer: the switcher offers archive and the guard refuses it", reason)
		}
	}
}
