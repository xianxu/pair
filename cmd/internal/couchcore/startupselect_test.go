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

// The rule the operator chose: detached before parked, most recent first, and a
// new thread only when there is nothing to return to.
//
// It replaces an EXACTNESS rule that was a ratchet -- two resumable rows at one
// path created a third, which guaranteed the next startup created a fourth. Six
// threads in one repo is what that produced.
func TestSelectResumableRootPrefersWarmThenRecent(t *testing.T) {
	const path = "/repo"
	older := time.Unix(1000, 0).UTC()
	newer := time.Unix(2000, 0).UTC()

	for _, tc := range []struct {
		name string
		rows []ActionableThreadSummary
		want string
	}{
		{
			name: "detached beats parked, however old",
			rows: []ActionableThreadSummary{
				selectRow("couch-parked", ThreadParked, path, newer),
				selectRow("couch-detached", ThreadDetached, path, older),
			},
			want: "couch-detached",
		},
		{
			name: "within a class, most recently active wins",
			rows: []ActionableThreadSummary{
				selectRow("couch-old", ThreadDetached, path, older),
				selectRow("couch-new", ThreadDetached, path, newer),
			},
			want: "couch-new",
		},
		{
			name: "parked when nothing is detached",
			rows: []ActionableThreadSummary{
				selectRow("couch-parked", ThreadParked, path, older),
				selectRow("couch-broken", ThreadUnusable, path, newer),
			},
			want: "couch-parked",
		},
		{
			name: "debris alone selects nothing, so a new thread starts",
			rows: []ActionableThreadSummary{
				selectRow("couch-broken", ThreadUnusable, path, newer),
				selectRow("couch-broken2", ThreadUnusable, path, older),
			},
			want: "",
		},
		{
			// pair#363: the repository, not the path, is what a row holds, so
			// a row elsewhere in the same scope is the one to return to.
			name: "another path in the same repository matches",
			rows: []ActionableThreadSummary{selectRow("couch-elsewhere", ThreadDetached, "/other", newer)},
			want: "couch-elsewhere",
		},
		{
			name: "a live row is never selected -- this couch already hosts it",
			rows: []ActionableThreadSummary{selectRow("couch-live", ThreadLive, path, newer)},
			want: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			address, ok := SelectResumableRoot(tc.rows, "scope")
			if tc.want == "" {
				if ok {
					t.Fatalf("selected %+v, want nothing", address)
				}
				return
			}
			if !ok || string(address.Tag) != tc.want {
				t.Fatalf("selected %+v (ok=%v), want %s", address, ok, tc.want)
			}
		})
	}
}

// Six rows at one path is the operator's real store, and it must not be a
// refusal: refusing is what minted a seventh thread.
func TestSelectResumableRootHandlesACrowdedPath(t *testing.T) {
	const path = "/repo"
	rows := []ActionableThreadSummary{
		selectRow("couch-1", ThreadUnusable, path, time.Unix(5000, 0).UTC()),
		selectRow("couch-2", ThreadParked, path, time.Unix(4000, 0).UTC()),
		selectRow("couch-3", ThreadDetached, path, time.Unix(1000, 0).UTC()),
		selectRow("couch-4", ThreadDetached, path, time.Unix(3000, 0).UTC()),
		selectRow("couch-5", ThreadUnusable, path, time.Unix(2000, 0).UTC()),
	}
	address, ok := SelectResumableRoot(rows, "scope")
	if !ok || address.Tag != "couch-4" {
		t.Fatalf("selected %+v (ok=%v), want the newest detached row", address, ok)
	}
}

// The occupancy predicates answer different questions, so they are not one
// function -- but they must not disagree where they overlap. A state that holds
// a repository's primary has to be one the operator can actually reach, or
// couch refuses a start for a thread it will not offer. Read from a
// subdirectory, because since pair#363 the scope, not the path, is what a row
// holds.
func TestOccupancyPredicatesAgreeWhereTheyOverlap(t *testing.T) {
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
func TestSelectResumableRootMatchesTheRepositoryNotThePath(t *testing.T) {
	at := time.Unix(1000, 0).UTC()
	parked := selectRow("couch-0000000000000001", ThreadParked, "/w/repo", at)
	if address, ok := SelectResumableRoot([]ActionableThreadSummary{parked}, "scope"); !ok || address != parked.Address {
		t.Fatalf("a parked primary at the repository root was not selected from its subdirectory: (%+v, %v)", address, ok)
	}
	foreign := parked
	foreign.Address.RepoScope = "other-scope"
	if address, ok := SelectResumableRoot([]ActionableThreadSummary{foreign}, "scope"); ok {
		t.Fatalf("a row in another repository scope was selected: %+v", address)
	}
	// A slot row is never the primary, even if a scope key were shared.
	slot := parked
	slot.Target = ThreadTarget{Kind: ThreadTargetSlot}
	if address, ok := SelectResumableRoot([]ActionableThreadSummary{slot}, "scope"); ok {
		t.Fatalf("a slot row was selected as the primary: %+v", address)
	}
}

// Live, detached and parked rows hold the repository's primary from any
// subdirectory; unusable rows do not, or a corrupted record would lock its
// repository out (resolved ambiguity 9).
func TestScopeHoldsUsableThreadFromASubdirectory(t *testing.T) {
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
