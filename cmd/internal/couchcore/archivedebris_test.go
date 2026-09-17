package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

// The two rows the operator could not archive, and the rule that frees them.
//
// Traced live 2026-09-16. Archive refused them for BOOKKEEPING -- an open park
// transaction, a start claim -- while the processes both described had been gone
// for hours. #256's rule says an untidy record is not a reason to refuse: the
// only thing that may refuse an irreversible act is a process that cannot be
// proved gone.

// archiveDebrisCouch wires a store to fakes that answer "no session, no client".
func archiveDebrisCouch(t *testing.T, store *ThreadStore, proc *FakeProcOps, address ThreadAddress) *Couch {
	t.Helper()
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(address, "pair-"+string(address.Tag), false)
	return &Couch{Threads: store, Artifacts: artifacts, Proc: proc, Path: NewFakePathOps(nil)}
}

// TestArchiveClearsAnOrphanedPark is #271's record, archived.
//
// couch-e1a31510b7033d08 rev 293: park phase awaiting_completion since
// 2026-09-15 22:53, pid 64734 confirmed gone (ESRCH). DecideRecovery:33 refused
// before any evidence was read -- "a park transaction is still open; let
// lifecycle recovery finish" -- naming a recovery that had nothing left to run.
func TestArchiveClearsAnOrphanedPark(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-e1a31510b7033d08", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{
		PID: 64734, Identity: "1789535173.46673", State: IncarnationLive,
	}}
	record.Park = wedgedParkFixture(record.Address)
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	// AbandonPark requires the record revision to have advanced past the
	// transaction's, and the live record had -- it was the operator's named
	// `brain` thread. Naming it here reproduces that, through the real store.
	label := "brain"
	if created, err = store.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{Name: &label}); err != nil {
		t.Fatal(err)
	}

	// Nothing in the fake pid table: both the park owner and the helper are gone.
	couch := archiveDebrisCouch(t, store, NewFakeProcOps(), created.Address)
	if _, err := couch.ArchiveThread(context.Background(), created.Address); err != nil {
		t.Fatalf("a thread whose park owner and helper are both provably dead is unarchivable: %v", err)
	}
	archived, err := store.ArchivedThreads()
	if err != nil || len(archived) != 1 {
		t.Fatalf("archive = %+v, %v", archived, err)
	}
}

// TestArchiveRefusesAParkItCannotProveOrphaned is the fail-closed half. Quiesce
// stops the session, so it must not run on a park that may still be mid-flight.
//
// Two rows, because one would pass for the wrong reason. In the operator's
// record the park owner IS the incarnation, so the incarnation screen refuses
// first and the park-owner screen is never reached -- a mutation that removed it
// entirely left this test green. The foreign-owned row is the one that pins it:
// the park names a process that is not an incarnation at all, which
// validateLifecycle permits through its `replacementUnknown` escape.
func TestArchiveRefusesAParkItCannotProveOrphaned(t *testing.T) {
	const helperPID, foreignOwnerPID = 64734, 42
	for _, tc := range []struct {
		name    string
		foreign bool
		unknown int
	}{
		{name: "its own helper cannot be probed", unknown: helperPID},
		{name: "a foreign owner cannot be probed", foreign: true, unknown: foreignOwnerPID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			record := actionableTestThread("couch-e1a31510b7033d08", time.Unix(100, 0).UTC())
			record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
			record.Incarnations = []ThreadIncarnation{{
				PID: helperPID, Identity: "1789535173.46673", State: IncarnationLive,
			}}
			record.Park = wedgedParkFixture(record.Address)
			if tc.foreign {
				record.Park.Identity.PID = foreignOwnerPID
				record.Park.Identity.ProcessIdentity = "original-owner"
				record.Park.Phase = ParkUnknown
				record.Park.Attempts = []ParkAttempt{{
					Number: 1,
					Failure: &ParkFailure{
						Code:       pairlifecycle.FailureReplacementIncarnation,
						Diagnostic: "a replacement incarnation appeared",
					},
				}}
			}
			created, err := store.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			label := "brain"
			if created, err = store.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{Name: &label}); err != nil {
				t.Fatal(err)
			}

			proc := NewFakeProcOps()
			proc.SetUnknown(tc.unknown)
			couch := archiveDebrisCouch(t, store, proc, created.Address)

			if _, err := couch.ArchiveThread(context.Background(), created.Address); err == nil {
				t.Fatal("archived a thread whose park owner could not be proved dead")
			}
			after, err := store.GetThread(created.Address)
			if err != nil {
				t.Fatal(err)
			}
			if after.Park == nil {
				t.Fatal("the refusal still abandoned the park; that tombstone is permanent")
			}
		})
	}
}

// TestArchiveClearsADriverlessStartClaim is Task 4a: the escape the busy row
// needs. #256 M2 released such a row from `busy` so the menu would offer
// archive; without this the offer is an action that always fails, which is the
// anti-pattern the menu's own comments name.
func TestArchiveClearsADriverlessStartClaim(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-3b82bfd593cac896", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{
		State: IncarnationCreating,
		Start: &ThreadStartClaim{
			Nonce: "start-0123456789abcdef", OwnerPID: 4242, OwnerIdentity: "supervisor",
		},
	}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}

	couch := archiveDebrisCouch(t, store, NewFakeProcOps(), created.Address)
	if _, err := couch.ArchiveThread(context.Background(), created.Address); err != nil {
		t.Fatalf("a thread whose starting couch is gone is unarchivable: %v", err)
	}
	archived, err := store.ArchivedThreads()
	if err != nil || len(archived) != 1 {
		t.Fatalf("archive = %+v, %v", archived, err)
	}
}

// TestArchiveRefusesAStartItCannotProveAbandoned keeps the destructive act
// behind proof: a couch that may still be starting this thread must not have
// its session quiesced out from under it.
func TestArchiveRefusesAStartItCannotProveAbandoned(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*FakeProcOps)
	}{
		{name: "owner alive", setup: func(p *FakeProcOps) { p.Set(4242, "supervisor") }},
		{name: "owner unprovable", setup: func(p *FakeProcOps) { p.SetUnknown(4242) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			record := actionableTestThread("couch-3b82bfd593cac896", time.Unix(100, 0).UTC())
			record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
			record.Incarnations = []ThreadIncarnation{{
				State: IncarnationCreating,
				Start: &ThreadStartClaim{
					Nonce: "start-0123456789abcdef", OwnerPID: 4242, OwnerIdentity: "supervisor",
				},
			}}
			created, err := store.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			proc := NewFakeProcOps()
			tc.setup(proc)
			couch := archiveDebrisCouch(t, store, proc, created.Address)

			if _, err := couch.ArchiveThread(context.Background(), created.Address); err == nil {
				t.Fatal("archived a thread another couch may still be starting")
			}
			after, err := store.GetThread(created.Address)
			if err != nil {
				t.Fatal(err)
			}
			if len(after.Incarnations) != 1 || after.Incarnations[0].Start == nil {
				t.Fatalf("the refusal still rolled the claim back: %+v", after.Incarnations)
			}
		})
	}
}

// TestSpawnedButNeverBoundThreadIsArchivable is the operator's second wedged
// row, traced live 2026-09-16.
//
// couch-3b82bfd593cac896: one incarnation {pid 87309, state live, start nil},
// pid confirmed gone, and NO row in the scope's session-names.jsonl --
// `repos/2e51fcf9799b1d8f/session-names.jsonl` held only couch-dbc88727c6378a0f.
// #273's fresh-spawn shape: the launch failed before it ever published a
// binding.
//
// The retirement that fixes it already existed and was correct
// (reconcileRecoveryHelper re-probes the exact {PID, Identity}, requires Dead,
// retires). It was unreachable: observeRecoverySession errored on the absent
// binding fifteen lines earlier, and ArchiveThread's escape hatch admits a
// record only when it carries no incarnation at all.
//
// The rule: an absent session binding is not a reason to skip retiring a
// provably-dead incarnation -- it is corroborating evidence the thread is gone.
func TestSpawnedButNeverBoundThreadIsArchivable(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-3b82bfd593cac896", time.Time{})
	record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{
		PID: 87309, Identity: "never-bound", State: IncarnationLive,
	}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}

	// No SetPairSession call at all: the index has no row for this address.
	artifacts := NewFakeThreadArtifactCollisionChecker()
	couch := &Couch{Threads: store, Artifacts: artifacts, Proc: NewFakeProcOps(), Path: NewFakePathOps(nil)}

	if _, err := couch.ArchiveThread(context.Background(), created.Address); err != nil {
		t.Fatalf("a thread with a dead helper and no binding is unarchivable: %v", err)
	}
	archived, err := store.ArchivedThreads()
	if err != nil || len(archived) != 1 {
		t.Fatalf("archive = %+v, %v", archived, err)
	}
}

// TestAbsentBindingProvesNothingAboutALiveHelper is the other half of the same
// rule, and the reason it is written as a predicate rather than as "no binding
// means gone". A binding can be absent because the launch has not published one
// YET, and a process that is alive or unprovable may still publish it.
func TestAbsentBindingProvesNothingAboutALiveHelper(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*FakeProcOps)
	}{
		{name: "helper alive", setup: func(p *FakeProcOps) { p.Set(87309, "never-bound") }},
		{name: "helper unprovable", setup: func(p *FakeProcOps) { p.SetUnknown(87309) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			record := actionableTestThread("couch-3b82bfd593cac896", time.Time{})
			record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
			record.Incarnations = []ThreadIncarnation{{
				PID: 87309, Identity: "never-bound", State: IncarnationLive,
			}}
			created, err := store.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			proc := NewFakeProcOps()
			tc.setup(proc)
			couch := &Couch{
				Threads: store, Artifacts: NewFakeThreadArtifactCollisionChecker(),
				Proc: proc, Path: NewFakePathOps(nil),
			}

			if _, err := couch.ArchiveThread(context.Background(), created.Address); err == nil {
				t.Fatal("archived a thread whose helper may still be publishing its binding")
			}
			if after, err := store.GetThread(created.Address); err != nil || len(after.Incarnations) != 1 {
				t.Fatalf("the refusal still retired the incarnation: %+v %v", after, err)
			}
		})
	}
}

// TestArchivingAHuskReportsTheRollback covers the one exit that had no test:
// ErrThreadRolledBack.
//
// A start claim rolls back through DeleteStart, and DeleteStart's last branch
// DELETES the record rather than emptying it -- when the thread carries nothing
// else worth keeping: no launch profile, no park, no metadata. That is the husk
// a launch leaves when it fails before it ever registers. The row is gone, which
// is the whole of what archive promises, so archive reports success. Resume
// never reaches this exit -- DecideResume refuses a profile-less record first --
// so archive is the only consumer there is.
//
// Without this the sentinel was a branch nothing reached -- the shape the M1
// review kept finding (a vocabulary value with no producer is a claim no test
// can check).
func TestArchivingAHuskReportsTheRollback(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-00000000000000d1", time.Unix(100, 0).UTC())
	// No LatestLaunchProfile, no name, no park: nothing durable to preserve.
	record.Incarnations = []ThreadIncarnation{{
		State: IncarnationCreating,
		Start: &ThreadStartClaim{
			Nonce: "start-0123456789abcdef", OwnerPID: 4242, OwnerIdentity: "supervisor",
		},
	}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}

	couch := archiveDebrisCouch(t, store, NewFakeProcOps(), created.Address)
	result, err := couch.ArchiveThread(context.Background(), created.Address)
	if err != nil {
		t.Fatalf("archiving a husk refused: %v", err)
	}
	if result.Record.Address != created.Address {
		t.Fatalf("archive reported %+v, want the record it removed", result.Record.Address)
	}
	if _, err := store.GetThread(created.Address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("the husk survived its rollback: %v", err)
	}

	// The producer itself, so the sentinel is pinned rather than inferred from
	// archive's success.
	second := actionableTestThread("couch-00000000000000d2", time.Unix(100, 0).UTC())
	second.Incarnations = record.Incarnations
	createdSecond, err := store.CreateThread(second)
	if err != nil {
		t.Fatal(err)
	}
	bare := &Couch{Threads: store, Artifacts: NewFakeThreadArtifactCollisionChecker(), Proc: NewFakeProcOps()}
	if _, err := bare.clearLifecycleDebris(createdSecond); !errors.Is(err, ErrThreadRolledBack) {
		t.Fatalf("clearLifecycleDebris on a husk = %v, want ErrThreadRolledBack", err)
	}
}

// TestClearingDebrisResumesSafelyAfterACrashBetweenItsWrites pins the claim
// lifecycledebris.go makes in a comment and nothing checked: "Independent
// writes, not one transaction. Each is independently correct ... so a crash
// between them leaves a record the next attempt repeats safely."
//
// That is a partial-progress path, which ARCH-ORDER says is exactly the kind of
// claim that ships with a sample size of zero. The store's AfterJournal seam can
// fail the run between AbandonPark and RetireIncarnation, so the interleaving is
// reproducible rather than argued.
func TestClearingDebrisResumesSafelyAfterACrashBetweenItsWrites(t *testing.T) {
	_, ns := newTestThreadStore(t)
	writes := 0
	crashing := newThreadStoreWithHooks(ns, threadStoreHooks{
		AfterTarget: func(int) error {
			writes++
			// Let CreateThread and the metadata write through, let AbandonPark
			// commit, then die before the incarnation is retired.
			if writes == 4 {
				return errInjectedStoreCrash
			}
			return nil
		},
	})

	record := actionableTestThread("couch-00000000000000d8", time.Unix(100, 0).UTC())
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{
		PID: 64734, Identity: "1789535173.46673", State: IncarnationLive,
	}}
	record.Park = wedgedParkFixture(record.Address)
	created, err := crashing.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	label := "brain"
	if created, err = crashing.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{Name: &label}); err != nil {
		t.Fatal(err)
	}

	crashed := &Couch{Threads: crashing, Proc: NewFakeProcOps()}
	if _, err := crashed.clearLifecycleDebris(created); err == nil {
		t.Fatal("the injected crash did not stop the second write")
	}

	// The park is gone and the incarnation is not: a genuinely partial record.
	partial, err := crashing.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if partial.Park != nil {
		t.Fatalf("the crash landed before AbandonPark (%d writes); this seam no longer splits the two, so the partial-progress path is unexercised", writes)
	}
	if len(partial.Incarnations) != 1 {
		t.Fatalf("expected a half-cleared record, got %+v", partial.Incarnations)
	}

	// The next attempt, on a store that is not crashing, must finish the job
	// rather than refuse what the first attempt already did.
	restarted := NewThreadStore(ns)
	if err := restarted.RecoverStoreJournal(); err != nil {
		t.Fatal(err)
	}
	current, err := restarted.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	retried := &Couch{Threads: restarted, Proc: NewFakeProcOps()}
	cleared, err := retried.clearLifecycleDebris(current)
	if err != nil {
		t.Fatalf("the retry refused a record its own earlier attempt half-cleared: %v", err)
	}
	if cleared == nil || len(cleared.Incarnations) != 0 || cleared.Park != nil {
		t.Fatalf("retry left debris: %+v", cleared)
	}
}
