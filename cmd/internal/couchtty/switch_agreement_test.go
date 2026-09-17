package couchtty

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// TestSwitchAgentOfferedImpliesPermitted is the class guard for #256 M2's BR-33,
// in the shape M3 Task 8 plans for archive.
//
// The defect it exists to catch: the switcher offers an action on a set of
// states, a guard permits it on a different set, and the two are kept in
// agreement by two authors remembering to. That drifted twice in one milestone —
// first when `parked` gained a producer the guard's receipt check refused, then
// when a replacement guard admitted a `live` row couch was hosting.
//
// The domain is DERIVED from the vocabularies (`AllThreadStates` x
// `AllThreadReasons`), not hand-listed, so a new state or reason lands here
// automatically. Offered must imply permitted; permitted may be wider, because
// `SwitchableState` also answers callers that reach the API without going
// through a row.
func TestSwitchAgentOfferedImpliesPermitted(t *testing.T) {
	for _, state := range couchcore.AllThreadStates() {
		for _, reason := range append(couchcore.AllThreadReasons(), "") {
			row := couchcore.ActionableThreadSummary{
				Address: couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000001"},
				State:   state, Reason: reason,
			}
			// The projection sets a reason exactly when the state is unusable;
			// anything else is not a row the classifier can produce.
			if (state == couchcore.ThreadUnusable) != (reason != "") {
				continue
			}
			offered := false
			for _, item := range menuActionItems(row) {
				if item == "switch-agent" {
					offered = true
				}
			}
			permitted := couchcore.SwitchableState(state, reason)
			if offered && !permitted {
				t.Errorf("%s/%s: the switcher offers switch-agent and the guard refuses it — "+
					"an action that always fails is how a switcher teaches an operator to distrust it",
					state, reason)
			}
		}
	}
}
