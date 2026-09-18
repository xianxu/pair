package couchcore

import (
	"testing"
	"time"
)

// A recorded process couch could not ask about is not a recorded process that
// is gone, and the difference has to survive the projection.
//
// `ObserveRecordedProcesses` produced POSITIVE live proof and dropped everything
// else with one `continue`, so an unanswerable probe and a proved-dead one
// reached the classifier identically. Since M1 absence of live proof falls
// through to the session -- which is #272's whole fix -- so the unanswerable
// case could reach `session-gone`, an archive-eligible row, while the recorded
// helper may still have been running. Press archive on it and observeRecovery
// probes the same process, gets Unknown, and DecideRecovery refuses every time:
// an action the switcher offers and the guard always refuses, which is the
// invariant the action tables exist to forbid.
//
// The distinction is narrow ON PURPOSE. A probe that answers Dead, and an
// identity token that READS and differs (a recycled pid), are both CONFIRMED
// answers and must keep falling through, or #272 comes back.
func TestAnUnprovableRecordedProcessIsNotConfirmedAbsence(t *testing.T) {
	cases := []struct {
		name           string
		arrange        func(*FakeProcOps)
		wantState      ActionableThreadState
		wantReason     ThreadReason
		wantArchivable bool
	}{
		{
			name:       "the probe cannot answer",
			arrange:    func(p *FakeProcOps) { p.SetUnknown(42) },
			wantState:  ThreadUnusable,
			wantReason: ReasonUnknown,
		},
		{
			name: "the process is there but its identity token cannot be read",
			arrange: func(p *FakeProcOps) {
				p.Set(42, "pair-x")
				p.IdentityErr[42] = true
			},
			wantState:  ThreadUnusable,
			wantReason: ReasonUnknown,
		},
		{
			name:       "proved dead -- the confirmed answer still falls through to the session",
			arrange:    func(p *FakeProcOps) {},
			wantState:  ThreadUnusable,
			wantReason: ReasonSessionGone, wantArchivable: true,
		},
		{
			name: "the pid was recycled -- a token that reads and differs is confirmed too",
			arrange: func(p *FakeProcOps) {
				p.Set(42, "somebody-else")
			},
			wantState:  ThreadUnusable,
			wantReason: ReasonSessionGone, wantArchivable: true,
		},
		{
			name: "alive and the token matches -- still live",
			arrange: func(p *FakeProcOps) {
				p.Set(42, "pair-x")
			},
			wantState: ThreadLive,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
			record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
			created, err := store.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			withIncarnation, err := store.updateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
				next.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-x", State: IncarnationLive}}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			proc := NewFakeProcOps()
			tc.arrange(proc)
			couch := &Couch{Threads: store, Artifacts: NewFakeThreadArtifactCollisionChecker(), Proc: proc, Path: NewFakePathOps(nil)}

			rows, err := couch.ActionableThreadInventory(nil)
			if err != nil || len(rows) != 1 || rows[0].Address != withIncarnation.Address {
				t.Fatalf("inventory = %+v, %v", rows, err)
			}
			if rows[0].State != tc.wantState || rows[0].Reason != tc.wantReason {
				t.Fatalf("= (%q, %q), want (%q, %q)", rows[0].State, rows[0].Reason, tc.wantState, tc.wantReason)
			}
			// The consequence, not just the label: `unknown` is the one
			// unusable reason archive refuses, and `live` is refused because
			// couch hosts it. Asserting the reason alone would let the
			// classification move without the guard noticing.
			if got := ArchivableState(rows[0].State, rows[0].Reason); got != tc.wantArchivable {
				t.Fatalf("archive permitted=%v for (%q, %q), want %v",
					got, rows[0].State, rows[0].Reason, tc.wantArchivable)
			}
		})
	}
}
