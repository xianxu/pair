package couchcore

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func staleStartRecord(started time.Time) ThreadRecord {
	r := actionableTestThread("couch-0000000000000001", started)
	r.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	r.Incarnations = []ThreadIncarnation{{State: IncarnationCreating, StartedAt: started, Start: &ThreadStartClaim{Nonce: "start-0123456789abcdef", OwnerPID: 4242, OwnerIdentity: "supervisor"}}}
	return r
}

func TestDecideStaleStartUsesObservedClockAndPositiveListing(t *testing.T) {
	now := time.Unix(10000, 0).UTC()
	for _, tc := range []struct {
		name                               string
		owner                              Liveness
		age                                time.Duration
		session                            SessionState
		missingTime, missingClock, noClaim bool
		want                               bool
	}{
		{name: "live-younger", owner: Live, age: 5*time.Minute - time.Nanosecond},
		{name: "unknown-younger", owner: Unknown, age: time.Minute},
		{name: "live-at-bound", owner: Live, age: 5 * time.Minute, want: true},
		{name: "unknown-at-bound", owner: Unknown, age: 5 * time.Minute, want: true},
		{name: "live-over-bound", owner: Live, age: 5*time.Minute + time.Nanosecond, want: true},
		{name: "dead-younger", owner: Dead, age: time.Second, want: true},
		{name: "dead-no-time", owner: Dead, missingTime: true, missingClock: true, want: true},
		{name: "future-claim", owner: Live, age: -time.Hour},
		{name: "missing-claim-time", owner: Unknown, missingTime: true},
		{name: "missing-observation-clock", owner: Live, age: time.Hour, missingClock: true},
		{name: "listed-live-old", owner: Live, age: time.Hour, session: SessionPresent},
		{name: "listed-dead-old", owner: Dead, age: time.Hour, session: SessionPresent},
		{name: "listed-unknown-old", owner: Unknown, age: time.Hour, session: SessionPresent},
		{name: "no-start-claim", owner: Dead, age: time.Hour, noClaim: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := staleStartRecord(now.Add(-tc.age))
			if tc.missingTime {
				r.Incarnations[0].StartedAt = time.Time{}
			}
			if tc.noClaim {
				r.Incarnations = nil
			}
			e := ThreadEvidence{ObservedAt: now, StartOwner: tc.owner, Session: SessionObservation{State: tc.session}}
			if tc.missingClock {
				e.ObservedAt = time.Time{}
			}
			before, _ := json.Marshal(r)
			if got := DecideStaleStart(r, e); got != tc.want {
				t.Fatalf("stale=%v want %v", got, tc.want)
			}
			after, _ := json.Marshal(r)
			if string(before) != string(after) {
				t.Fatal("projection mutated start claim")
			}
		})
	}
}

func TestStaleStartClassifierProjectsResumeOrFreshAndPreservesContinue(t *testing.T) {
	now := time.Unix(10000, 0).UTC()
	for _, bound := range []bool{false, true} {
		for _, owner := range []Liveness{Live, Unknown, Dead} {
			for _, session := range []SessionState{SessionPresent, SessionAbsent, SessionUnresolved} {
				r := staleStartRecord(now.Add(-5 * time.Minute))
				e := ThreadEvidence{ObservedAt: now, StartOwner: owner, Session: SessionObservation{State: session}, ParkedStatus: ProofResolved}
				if bound {
					e.Parked = []ParkedResumeObservation{{Address: r.Address, Agent: "claude", NativeID: "saved-native"}}
				}
				state, reason := ClassifyThread(r, e)
				got := ContinuityOutcome(ActionableThreadSummary{State: state, Reason: reason})
				want := Fresh
				if session == SessionPresent {
					want = Continue
				} else if bound && session == SessionAbsent {
					want = ResumeDisk
				}
				if got != want {
					t.Fatalf("bound=%v owner=%v session=%v: %s want %s", bound, owner, session, got, want)
				}
			}
		}
	}
}

type staleInventoryClock struct {
	now   time.Time
	calls int
}

func (c *staleInventoryClock) Now() time.Time { c.calls++; return c.now }

func TestStaleStartInventoryContextsUseInjectedObservationTimeWithoutWrites(t *testing.T) {
	start := time.Unix(10000, 0).UTC()
	store, _ := newTestThreadStore(t)
	record, err := store.CreateThread(staleStartRecord(start))
	if err != nil {
		t.Fatal(err)
	}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetSessionPresence(record.Address, SessionObservation{State: SessionAbsent})
	artifacts.SetNativeBinding(record.Address, "claude", sessioninventory.BindingEstablished, "saved-native")
	proc := NewFakeProcOps()
	proc.Set(4242, "supervisor")
	clock := &staleInventoryClock{now: start.Add(5*time.Minute - time.Nanosecond)}
	c := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil), Proc: proc, Clock: clock}
	younger, err := c.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil || len(younger) != 1 || ContinuityOutcome(younger[0]) != ContinuityStarting {
		t.Fatalf("young=%+v err=%v", younger, err)
	}
	clock.now = start.Add(5 * time.Minute)
	stale, err := c.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil || len(stale) != 1 || ContinuityOutcome(stale[0]) != ResumeDisk {
		t.Fatalf("stale=%+v err=%v", stale, err)
	}
	diagnostic, err := c.ThreadInventoryContext(context.Background())
	if err != nil || len(diagnostic) != 1 || diagnostic[0].State != ThreadParked {
		t.Fatalf("diagnostic=%+v err=%v", diagnostic, err)
	}
	if clock.calls != 3 {
		t.Fatalf("observation clock calls=%d want 3", clock.calls)
	}
	current, err := store.GetThread(record.Address)
	if err != nil || !reflect.DeepEqual(record, current) {
		t.Fatalf("inventory wrote lifecycle: before=%+v after=%+v err=%v", record, current, err)
	}
	if len(proc.Signals) != 0 || len(proc.GroupSignals) != 0 {
		t.Fatal("projection signalled supervisor")
	}
}
