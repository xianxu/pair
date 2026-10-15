package couchcore

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
)

func assertWarmClientRepair(t *testing.T, env *testEnv, before ThreadRecord, actor ActorRecord) {
	t.Helper()
	after, err := env.Couch.Threads.GetThread(before.Address)
	if err != nil {
		t.Fatal(err)
	}
	expected := cloneThreadRecord(before)
	expected.Revision++
	expected.Incarnations[0].PID = actor.PID
	expected.Incarnations[0].Identity = actor.Identity
	expected.Incarnations[0].State = IncarnationLive
	if !reflect.DeepEqual(expected, after) {
		t.Fatalf("repair changed more than client incarnation: before=%+v after=%+v", before, after)
	}
	if actor.AttachmentNotice != "" {
		t.Fatalf("repaired helper reported borrowed: %s", actor.AttachmentNotice)
	}
}

func TestWarmAttachmentRepairsOnlyDeadManagedClient(t *testing.T) {
	env, _, before := liveSlotFixture(t)
	env.Proc.Kill(before.Incarnations[0].PID)
	env.Artifacts.SetPairSession(before.Address, "listed-slot-one", true)
	result, err := env.Couch.ResumeTarget(context.Background(), ResumeTarget{Path: before.StartingPath})
	if err != nil {
		t.Fatal(err)
	}
	start, ok := result.(StartResult)
	if !ok || start.Handle == nil {
		t.Fatalf("result %T", result)
	}
	assertWarmClientRepair(t, env, before, start.Record)
	if len(env.Proc.Signals[before.Incarnations[0].PID]) != 0 {
		t.Fatal("repair signalled old helper")
	}
}

func TestWarmAttachmentBorrowsWithClaimsOrUnprovenOldClient(t *testing.T) {
	for _, reason := range []string{"live", "unknown", "start claim"} {
		t.Run(reason, func(t *testing.T) {
			record := warmDetachedRecord(t)
			record.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "old-client", State: IncarnationLive, LaunchProfile: record.LatestLaunchProfile}}
			owner := Dead
			if reason == "live" {
				owner = Live
			}
			if reason == "unknown" {
				owner = Unknown
			}
			if reason == "start claim" {
				record.Incarnations[0].State = IncarnationCreating
				record.Incarnations[0].Start = &ThreadStartClaim{Nonce: "start-0123456789abcdef", OwnerPID: 4242, OwnerIdentity: "supervisor", LaunchProfile: record.LatestLaunchProfile}
				record.Incarnations[0].LaunchProfile = nil
			}
			env, before, session := readonlyWarmFixture(t, record, owner)
			actor, h, err := env.Couch.attachListedTerminal(context.Background(), before, session, ResumeOptions{})
			if err != nil || h == nil {
				t.Fatalf("h=%v err=%v", h, err)
			}
			if !strings.Contains(actor.AttachmentNotice, "BORROWED ATTACH") {
				t.Fatalf("missing borrowed reason: %+v", actor)
			}
			assertWarmRecordUnchanged(t, env, before)
		})
	}
}

func TestWarmAttachmentLeavesRecordWhenOldClientProofChangesAtEffect(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	before := recordAtCheckout(t, local.slot.WorktreeRoot, local.slot.WorktreeRoot, "couch-0123456789abcdef")
	before.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "old-client", State: IncarnationLive}}
	var err error
	before, err = local.CreateThread(before)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetPairSession(before.Address, "listed-slot-one", true)
	// The preflight sees Dead; changing at the staged write proves the final
	// guard must run immediately before atomic publication of the replacement.
	env.Couch.Threads = local
	local.hooks.AfterPublicationWrite = func(string) error { env.Proc.Set(5151, "old-client"); return nil }
	actor, h, err := env.Couch.attachListedTerminal(context.Background(), before, "listed-slot-one", ResumeOptions{})
	if err != nil || h == nil {
		t.Fatalf("h=%v err=%v", h, err)
	}
	if !strings.Contains(actor.AttachmentNotice, "BORROWED ATTACH") {
		t.Fatalf("missing changed-proof notice: %s", actor.AttachmentNotice)
	}
	assertWarmRecordUnchanged(t, env, before)
}

func TestWarmAttachmentRepairPreservesProducingLaunchAndParkHistory(t *testing.T) {
	record := warmDetachedRecord(t)
	markActionableParked(&record, time.Unix(100, 0).UTC())
	record.VerifiedPark = nil
	record.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "dead-client", State: IncarnationLive, StartedAt: time.Unix(110, 0).UTC(), LaunchProfile: record.LatestLaunchProfile}}
	record.SessionBinding = &couchidentity.SessionBinding{Legacy: true, Name: "pair-producing-launch", ScopeKey: record.Address.RepoScope, Tag: string(record.Address.Tag), StartNonce: "start-0123456789abcdef"}
	env, before, session := readonlyWarmFixture(t, record, Dead)
	actor, h, err := env.Couch.attachListedTerminal(context.Background(), before, session, ResumeOptions{})
	if err != nil || h == nil {
		t.Fatalf("h=%v err=%v", h, err)
	}
	assertWarmClientRepair(t, env, before, actor)
	if env.Artifacts.BindingResolutions() != 0 {
		t.Fatal("repair re-derived native launch identity")
	}
}

func TestWarmAttachmentRepeatedClientRemainsBorrowedWhileFirstClientLives(t *testing.T) {
	env, _, before := liveSlotFixture(t)
	env.Proc.Kill(before.Incarnations[0].PID)
	env.Artifacts.SetPairSession(before.Address, "listed-slot-one", true)
	first, h, err := env.Couch.attachListedTerminal(context.Background(), before, "listed-slot-one", ResumeOptions{})
	if err != nil || h == nil {
		t.Fatal(err)
	}
	env.Proc.Set(first.PID, first.Identity)
	repaired, err := env.Couch.Threads.GetThread(before.Address)
	if err != nil {
		t.Fatal(err)
	}
	second, h2, err := env.Couch.attachListedTerminal(context.Background(), repaired, "listed-slot-one", ResumeOptions{})
	if err != nil || h2 == nil || !strings.Contains(second.AttachmentNotice, "still alive") {
		t.Fatalf("second=%+v h=%v err=%v", second, h2, err)
	}
	assertWarmRecordUnchanged(t, env, repaired)
}

func TestDeadClientRepairDoesNotClaimPublicationIntoMissingStore(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	before := recordAtCheckout(t, local.slot.WorktreeRoot, local.slot.WorktreeRoot, "couch-0123456789abcdef")
	before.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "dead-client", State: IncarnationLive}}
	var err error
	before, err = local.CreateThread(before)
	if err != nil {
		t.Fatal(err)
	}
	h, err := env.Runner.Start(local.slot.WorktreeRoot, []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := ActorRecord{Thread: before.Address, PID: h.PID(), Identity: h.Identity(), Shape: StartWarmReattach}
	if err := os.Rename(local.root, local.root+"-moved"); err != nil {
		t.Fatal(err)
	}
	err = local.RepairDeadClientHelper(context.Background(), before, client, h, env.Proc)
	if err == nil {
		t.Fatal("missing store reported a client repair without publication")
	}
	if _, err := os.Stat(local.root); !os.IsNotExist(err) {
		t.Fatalf("repair manufactured missing store: %v", err)
	}
}

func TestWarmAttachmentBorrowsVerifiedParkWithoutChangingReceipt(t *testing.T) {
	record := warmDetachedRecord(t)
	markActionableParked(&record, time.Unix(100, 0).UTC())
	record.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "dead-client", State: IncarnationUnknown, LaunchProfile: record.LatestLaunchProfile}}
	env, before, session := readonlyWarmFixture(t, record, Dead)
	actor, h, err := env.Couch.attachListedTerminal(context.Background(), before, session, ResumeOptions{})
	if err != nil || h == nil || !strings.Contains(actor.AttachmentNotice, "BORROWED ATTACH") {
		t.Fatalf("actor=%+v h=%v err=%v", actor, h, err)
	}
	assertWarmRecordUnchanged(t, env, before)
}

// A graceful old Couch can retire dead client bookkeeping before the next
// process starts. Continue then owns a new local client without inventing a
// durable incarnation that narrow replacement repair has no authority to add.
func TestWarmAttachmentAfterLeaveRetiresDeadClientRemainsBorrowed(t *testing.T) {
	record := warmDetachedRecord(t)
	markActionableParked(&record, time.Unix(100, 0).UTC())
	record.VerifiedPark = nil
	record.Incarnations = []ThreadIncarnation{{PID: 27163, Identity: "old-dead-client", State: IncarnationLive, StartedAt: time.Unix(110, 0).UTC(), LaunchProfile: record.LatestLaunchProfile}}
	record.SessionBinding = &couchidentity.SessionBinding{Legacy: true, Name: "pair-surviving-launch", ScopeKey: record.Address.RepoScope, Tag: string(record.Address.Tag), StartNonce: "start-0123456789abcdef"}
	env, before, _ := readonlyWarmFixture(t, record, Dead)
	old := env.Couch
	if _, err := old.Leave(context.Background(), LeaveDetach); err != nil {
		t.Fatal(err)
	}
	retired, err := old.Threads.GetThread(before.Address)
	if err != nil {
		t.Fatal(err)
	}
	expected := cloneThreadRecord(before)
	expected.Revision++
	expected.Incarnations = []ThreadIncarnation{} // persisted decoding normalizes empty collections
	if !reflect.DeepEqual(expected, retired) {
		t.Fatalf("old shutdown changed more than dead-client retirement: expected=%+v got=%+v", expected, retired)
	}
	if len(env.Proc.Signals) != 0 || len(env.Artifacts.Quiesces()) != 0 {
		t.Fatal("retiring dead bookkeeping signalled a process or stopped the surviving session")
	}

	restarted, err := New(old.Namespace, env.Runner, old.Path, env.Git, env.Proc, NewStore(env.Dir), old.Clock, NewFixedIDGen("restart-client"), newIncrementingEntropy(), env.Artifacts)
	if err != nil {
		t.Fatal(err)
	}
	actor, handle, err := restarted.ResumeContextWith(context.Background(), before.Address, ResumeOptions{WarmOnly: true})
	if err != nil || handle == nil || !handle.Alive() {
		t.Fatalf("warm attachment failed: actor=%+v handle=%v err=%v", actor, handle, err)
	}
	t.Cleanup(func() { env.Runner.SetExited(handle.ID(), 0) })
	if actor.Shape != StartWarmReattach || actor.Thread != before.Address || actor.PID != handle.PID() || actor.Identity != handle.Identity() || restarted.warmClients[actor.ID] != handle {
		t.Fatalf("attachment is not a locally owned client: %+v", actor)
	}
	if !strings.Contains(actor.AttachmentNotice, "BORROWED ATTACH: no unique retained client helper") {
		t.Fatalf("missing truthful borrowed reason: %s", actor.AttachmentNotice)
	}
	after, err := restarted.Threads.GetThread(before.Address)
	if err != nil || !reflect.DeepEqual(retired, after) {
		t.Fatalf("restart attachment invented durable state: before=%+v after=%+v err=%v", retired, after, err)
	}
	if len(env.Proc.Signals) != 0 || len(env.Artifacts.Quiesces()) != 0 || env.Artifacts.BindingResolutions() != 0 {
		t.Fatal("warm attachment signalled, tore down, or resolved a native conversation")
	}
}

func TestWarmOnlyAttachmentRepairRejectsFIFOAfterAcknowledgment(t *testing.T) {
	record := warmDetachedRecord(t)
	record.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "dead-client", State: IncarnationLive, LaunchProfile: record.LatestLaunchProfile}}
	env, before, session := readonlyWarmFixture(t, record, Dead)
	path := env.Couch.Threads.recordPath(before.Address)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var done chan struct{}
	defer func() {
		if done == nil {
			return
		}
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("FIFO fixture writer did not finish")
		}
	}()
	env.Runner.AfterAcknowledge = func(string) error {
		env.Artifacts.SetPairSession(before.Address, session, true)
		if err := os.Remove(path); err != nil {
			return err
		}
		if err := unix.Mkfifo(path, 0600); err != nil {
			return err
		}
		done = make(chan struct{})
		go func() {
			defer close(done)
			time.Sleep(200 * time.Millisecond)
			fd, err := unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK, 0)
			if err == nil {
				_, _ = unix.Write(fd, original)
				_ = unix.Close(fd)
			}
		}()
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	actor, h, err := env.Couch.ResumeContextWith(ctx, before.Address, ResumeOptions{WarmOnly: true})
	elapsed := time.Since(started)
	if h != nil {
		defer env.Runner.SetExited(h.ID(), 0)
	}
	if elapsed >= 150*time.Millisecond {
		t.Fatalf("automatic repair waited for FIFO writer: %v", elapsed)
	}
	if err != nil || h == nil || !strings.Contains(actor.AttachmentNotice, "BORROWED ATTACH") {
		t.Fatalf("attachment=%+v handle=%v err=%v", actor, h, err)
	}
	if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("repair changed FIFO: %v", err)
	}
}

func TestWarmOnlyResumePreflightHonorsCallerDeadline(t *testing.T) {
	env, before, _ := readonlyWarmFixture(t, warmDetachedRecord(t), Dead)
	lock, err := env.Couch.Threads.retentionReadLock(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, h, err := env.Couch.ResumeContextWith(ctx, before.Address, ResumeOptions{WarmOnly: true})
	if elapsed := time.Since(started); elapsed >= 150*time.Millisecond || !errors.Is(err, context.DeadlineExceeded) || h != nil {
		t.Fatalf("preflight ignored deadline: elapsed=%v err=%v handle=%v", elapsed, err, h)
	}
}

func TestWarmOnlyAbsentTerminalPreservesJournalArrivingAfterPreflight(t *testing.T) {
	env, before, _ := readonlyWarmFixture(t, warmDetachedRecord(t), Dead)
	store := env.Couch.Threads
	env.Artifacts.SetDetachedSession(before.Address, "")
	var pending []byte
	store.hooks.AfterGetThread = func(ThreadAddress) error {
		store.hooks.AfterGetThread = nil
		interrupted := errors.New("pending journal fixture")
		store.hooks.AfterJournal = func() error { return interrupted }
		if _, err := store.updateExistingThread(before.Address, before.Revision, func(next *ThreadRecord) error { next.Description = "pending must not replay"; return nil }); !errors.Is(err, interrupted) {
			t.Fatalf("fixture=%v", err)
		}
		store.hooks.AfterJournal = nil
		var err error
		pending, err = os.ReadFile(store.journalPath())
		if err != nil {
			t.Fatal(err)
		}
		return nil
	}
	original, err := os.ReadFile(store.recordPath(before.Address))
	if err != nil {
		t.Fatal(err)
	}
	_, h, err := env.Couch.ResumeContextWith(context.Background(), before.Address, ResumeOptions{WarmOnly: true})
	if err == nil || h != nil {
		t.Fatalf("absent warm attachment err=%v handle=%v", err, h)
	}
	after, readErr := os.ReadFile(store.journalPath())
	if readErr != nil || string(after) != string(pending) {
		t.Fatalf("warm-only refusal replayed pending journal: %v", readErr)
	}
	after, readErr = os.ReadFile(store.recordPath(before.Address))
	if readErr != nil || string(after) != string(original) {
		t.Fatalf("warm-only refusal changed record: %v", readErr)
	}
}

func TestWarmOnlyRepairCancellationBeforePublicationPreservesRecord(t *testing.T) {
	record := warmDetachedRecord(t)
	record.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "dead-client", State: IncarnationLive, LaunchProfile: record.LatestLaunchProfile}}
	env, before, _ := readonlyWarmFixture(t, record, Dead)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	env.Couch.Threads.hooks.BeforeUpdate = func(ThreadRecord, ThreadRecord) error { cancel(); return nil }
	actor, h, err := env.Couch.ResumeContextWith(ctx, before.Address, ResumeOptions{WarmOnly: true})
	if h != nil {
		defer env.Runner.SetExited(h.ID(), 0)
	}
	if err != nil || h == nil || !strings.Contains(actor.AttachmentNotice, "BORROWED ATTACH") {
		t.Fatalf("canceled repair attachment=%+v handle=%v err=%v", actor, h, err)
	}
	assertWarmRecordUnchanged(t, env, before)
	if _, err := os.Lstat(env.Couch.Threads.publicationPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled repair left publication: %v", err)
	}
}

func TestAutomaticInventoryPreviewHonorsCallerDeadline(t *testing.T) {
	for _, kind := range []string{"global", "local", "routed-local"} {
		t.Run(kind, func(t *testing.T) {
			store, backend, _ := previewThreadFixture(t, kind)
			env := newTestEnv(t, "/repo")
			env.Couch.Threads = store
			lock, err := backend.retentionReadLock(context.Background(), 0)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
			defer cancel()
			started := time.Now()
			_, _, err = env.Couch.gatherThreadEvidence(ctx, nil, func(ThreadRecord) bool { return false })
			if elapsed := time.Since(started); elapsed >= 150*time.Millisecond || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("inventory ignored deadline: elapsed=%v err=%v", elapsed, err)
			}
		})
	}
}
