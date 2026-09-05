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
			name: "another path never matches",
			rows: []ActionableThreadSummary{selectRow("couch-elsewhere", ThreadDetached, "/other", newer)},
			want: "",
		},
		{
			name: "a live row is never selected -- this couch already hosts it",
			rows: []ActionableThreadSummary{selectRow("couch-live", ThreadLive, path, newer)},
			want: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			address, ok := SelectResumableRoot(tc.rows, "scope", path)
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
	address, ok := SelectResumableRoot(rows, "scope", path)
	if !ok || address.Tag != "couch-4" {
		t.Fatalf("selected %+v (ok=%v), want the newest detached row", address, ok)
	}
}

// The occupancy predicates answer different questions, so they are not one
// function -- but they must not disagree where they overlap. A state that holds
// a path has to be one the operator can actually reach, or couch refuses a
// start for a thread it will not offer.
func TestOccupancyPredicatesAgreeWhereTheyOverlap(t *testing.T) {
	const path = "/repo"
	for _, state := range []ActionableThreadState{
		ThreadLive, ThreadDetached, ThreadParked, ThreadBusy, ThreadUnusable,
	} {
		row := selectRow("couch-0000000000000001", state, path, time.Unix(1000, 0).UTC())
		_, holds := PathHoldsUsableThread([]ActionableThreadSummary{row}, "scope", path)
		reachable := row.Live() || row.Resumable()
		if holds != reachable {
			t.Fatalf("state %q: holds path = %v, reachable by the operator = %v -- "+
				"couch would refuse a start for a thread it will not offer", state, holds, reachable)
		}
	}
}
