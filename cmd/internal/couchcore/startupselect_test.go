package couchcore

import (
	"testing"
	"time"
)

func selectRow(tag string, state ActionableThreadState, path string, active time.Time) ActionableThreadSummary {
	return ActionableThreadSummary{
		Address:      ThreadAddress{RepoScope: "scope", Tag: ThreadTag(tag)},
		WorkingPath:  path,
		State:        state,
		LastActiveAt: active,
	}
}

func TestOccupancyPredicatesAgreeWhereTheyOverlap(t *testing.T) {
	t.Parallel()

	for _, state := range AllThreadStates() {
		for _, reason := range append(AllThreadReasons(), "") {
			if (state == ThreadUnusable) != (reason != "") {
				continue
			}
			row := selectRow("couch-0000000000000001", state, "/repo", time.Unix(1000, 0).UTC())
			row.Reason = reason
			_, holds := ScopeHoldsUsableThread([]ActionableThreadSummary{row}, "scope")
			reachable := row.Live() || row.Resumable()
			if holds != reachable {
				t.Fatalf("state %q/%q: holds the repository = %v, reachable by the operator = %v -- "+
					"couch would refuse a start for a thread it will not offer", state, reason, holds, reachable)
			}
		}
	}
}

// pair#363: one primary per repository. A start in a subdirectory of a
// repository returns to the primary thread wherever in that repository it
// sits; a thread in another repository scope is not this one's.
func TestScopeHoldsUsableThreadFromASubdirectory(t *testing.T) {
	t.Parallel()

	at := time.Unix(1000, 0).UTC()
	for _, state := range []ActionableThreadState{ThreadLive, ThreadDetached, ThreadParked} {
		row := selectRow("couch-0000000000000001", state, "/w/repo", at)
		if held, ok := ScopeHoldsUsableThread([]ActionableThreadSummary{row}, "scope"); !ok || held.Address != row.Address {
			t.Errorf("%s row at the root does not hold the repository: (%+v, %v)", state, held.Address, ok)
		}
		slot := row
		slot.Target = ThreadTarget{Kind: ThreadTargetSlot}
		if _, ok := ScopeHoldsUsableThread([]ActionableThreadSummary{slot}, "scope"); ok {
			t.Errorf("a %s slot row holds the primary", state)
		}
	}
	for _, reason := range AllThreadReasons() {
		row := selectRow("couch-0000000000000001", ThreadUnusable, "/w/repo", at)
		row.Reason = reason
		if _, ok := ScopeHoldsUsableThread([]ActionableThreadSummary{row}, "scope"); ok {
			t.Errorf("unusable/%s holds the repository; debris must not lock it out", reason)
		}
	}
}
