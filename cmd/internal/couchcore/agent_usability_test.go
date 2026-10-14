package couchcore

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestListedLiveIsContinueAuthority(t *testing.T) {
	r := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	for _, hosted := range []bool{false, true} {
		e := ThreadEvidence{Session: SessionObservation{State: SessionPresent}}
		want := ThreadDetached
		if hosted {
			e.Live = []ProcessIdentity{{PID: 42, Identity: "helper"}}
			want = ThreadLive
		}
		got, reason := ClassifyThread(r, e)
		if got != want || reason != "" {
			t.Fatalf("hosted=%v got %s/%s", hosted, got, reason)
		}
	}
}
func TestAttributedOwnerCannotResumeDisk(t *testing.T) {
	r := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	r.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	for _, state := range []SessionState{SessionAbsent, SessionUnresolved, SessionOrphaned} {
		for _, unknown := range []bool{false, true} {
			e := ThreadEvidence{Session: SessionObservation{State: state}, ParkedStatus: ProofResolved, Parked: []ParkedResumeObservation{{Address: r.Address, Agent: "claude", NativeID: "native"}}}
			if unknown {
				e.Unproven = []ProcessIdentity{{PID: 42, Identity: "owner"}}
			} else {
				e.Live = []ProcessIdentity{{PID: 42, Identity: "owner"}}
			}
			got, _ := ClassifyThread(r, e)
			if got == ThreadParked || got == ThreadDetached || got == ThreadLive {
				t.Fatalf("owner unknown=%v session=%v got %s", unknown, state, got)
			}
		}
	}
}

func TestInventoryRetriesOneSharedListing(t *testing.T) {
	for _, recover := range []bool{false, true} {
		t.Run(fmt.Sprint(recover), func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			actor, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
			calls := 0
			env.Artifacts.SessionPresenceHook = func([]ThreadAddress) error {
				calls++
				if !recover || calls == 1 {
					return errors.New("listing unavailable")
				}
				return nil
			}
			_, evidence, err := env.Couch.gatherThreadEvidence(context.Background(), nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatalf("shared listing calls=%d", calls)
			}
			if got := evidence[actor.Thread].Session.Present(); got != recover {
				t.Fatalf("presence=%v recovered=%v", got, recover)
			}
		})
	}
}

func TestContinuityOutcomeReusesProofClassification(t *testing.T) {
	for _, tc := range []struct {
		state ActionableThreadState
		want  ContinuityAction
	}{
		{ThreadLive, Continue}, {ThreadDetached, Continue}, {ThreadParked, ResumeDisk},
		{ThreadUnusable, Fresh}, {ThreadBusy, Fresh},
	} {
		if got := ContinuityOutcome(ActionableThreadSummary{State: tc.state}); got != tc.want {
			t.Fatalf("%s: %s want %s", tc.state, got, tc.want)
		}
	}
}

func TestListedLiveOverridesStartMarkersWithoutChangingThem(t *testing.T) {
	for _, tc := range everyThreadShape(t) {
		if len(tc.record.Incarnations) == 0 {
			continue
		}
		evidence := tc.evidence
		evidence.Session = SessionObservation{State: SessionPresent}
		evidence.StartOwner = Live
		state, reason := ClassifyThread(tc.record, evidence)
		if state != ThreadLive && state != ThreadDetached {
			t.Fatalf("%s: %s/%s", tc.name, state, reason)
		}
	}
}

func TestInventoryListingBudgetExpiryStillProjectsFresh(t *testing.T) {
	env := newTestEnv(t, "/repo")
	actor, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	env.Artifacts.SessionPresenceHook = func([]ThreadAddress) error { calls++; cancel(); return context.DeadlineExceeded }
	rows, err := env.Couch.ActionableThreadInventoryContext(ctx, nil)
	if err != nil {
		t.Fatalf("known inventory discarded on listing expiry: %v", err)
	}
	for _, row := range rows {
		if row.Address == actor.Thread {
			if ContinuityOutcome(row) != Fresh {
				t.Fatalf("expired listing offered %s", ContinuityOutcome(row))
			}
			if calls != 1 {
				t.Fatalf("retried after cancellation: %d", calls)
			}
			return
		}
	}
	t.Fatal("known thread disappeared after listing timeout")
}

func TestContinueProofNeedsNoSavedLaunchProfile(t *testing.T) {
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	observed := []DetachedSessionObservation{{Address: record.Address, SessionName: "listed-session"}}
	for _, profile := range []*LaunchProfile{nil, {Agent: "legacy-unknown"}, {Agent: "claude", Argv: nil}} {
		record.LatestLaunchProfile = profile
		if !detachedResumeProofMatches(record, observed) {
			t.Fatalf("listed Continue depended on saved profile: %+v", profile)
		}
	}
	observed[0].Address.Tag = "different"
	if detachedResumeProofMatches(record, observed) {
		t.Fatal("foreign address accepted")
	}
}
