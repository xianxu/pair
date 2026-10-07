package couchcore

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

// Every couchcore entry that changes a thread's lifecycle refuses a thread
// another operation holds (pair#205 M1). This table is the enumeration that
// keeps a future entry from skipping the gate: add the entry here when adding
// it to couchcore.
func TestEveryLifecycleEntryRefusesAHeldThread(t *testing.T) {
	env, live := envWithLiveThread(t)
	c := env.Couch
	c.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	address := live.Address
	_, release, err := c.hold(context.Background(), address, "holder")
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	bg := context.Background()
	entries := []struct {
		name string
		call func() error
	}{
		{"resume", func() error { _, _, err := c.ResumeContext(bg, address); return err }},
		{"warm-only resume", func() error {
			_, _, err := c.ResumeContextWith(bg, address, ResumeOptions{WarmOnly: true})
			return err
		}},
		{"relaunch", func() error { _, err := c.Relaunch(bg, address); return err }},
		{"detach", func() error { _, err := c.Detach(bg, address); return err }},
		{"park", func() error { _, err := c.Park(bg, address, ""); return err }},
		{"park retry", func() error { _, err := c.Park(bg, address, "retry"); return err }},
		{"switch-agent", func() error {
			_, err := c.SwitchAgent(bg, SwitchAgentRequest{Address: address, Agent: "codex", Argv: []string{}, AcceptedFingerprint: "x"})
			return err
		}},
		{"continue-thread", func() error { _, err := c.Continue(bg, address, "req"); return err }},
		{"retry-continuation", func() error { _, err := c.RetryContinuation(bg, address, "req"); return err }},
		{"continuation-status", func() error { _, err := c.ReconcileContinuation(bg, address, "req", ""); return err }},
		{"reboot", func() error { _, err := c.Reboot(bg, RebootTarget{Address: address}); return err }},
		{"recover", func() error { _, err := c.RecoverThread(bg, address, ""); return err }},
		{"stop", func() error { _, err := c.Stop(ActorRecord{ID: "a", Thread: address}); return err }},
	}
	for _, entry := range entries {
		err := entry.call()
		var busy *ThreadBusyError
		if !errors.As(err, &busy) || busy.Running != "holder" || busy.Address != address {
			t.Errorf("%s on a held thread = %v, want ThreadBusyError naming the holder", entry.name, err)
		}
	}
}

// The 2026-09-08 incident (pair#214) in today's code: a resume arriving while
// a relaunch is mid-park is refused naming the relaunch, and the relaunch
// completes undisturbed.
func TestAResumeDuringARelaunchIsRefusedAndTheThreadStaysResumable(t *testing.T) {
	env, live := envWithLiveThread(t)
	c := env.Couch
	reached, proceed := make(chan struct{}), make(chan struct{})
	completePark := env.Artifacts.TriggerQuitHook
	env.Artifacts.TriggerQuitHook = func(session string, intent launcher.QuitIntent) error {
		close(reached)
		<-proceed
		return completePark(session, intent)
	}
	type relaunched struct {
		result RelaunchResult
		err    error
	}
	done := make(chan relaunched, 1)
	go func() {
		result, err := c.Relaunch(context.Background(), live.Address)
		done <- relaunched{result, err}
	}()
	<-reached

	_, _, err := c.ResumeContext(context.Background(), live.Address)
	var busy *ThreadBusyError
	if !errors.As(err, &busy) || busy.Running != "relaunch" {
		t.Fatalf("resume during relaunch = %v, want ThreadBusyError naming relaunch", err)
	}
	close(proceed)
	got := <-done
	if got.err != nil || got.result.Outcome != Relaunched {
		t.Fatalf("relaunch = %+v, %v; want Relaunched", got.result, got.err)
	}
	// Released: the next operation is judged on the thread's state, not refused.
	if _, err := c.Detach(context.Background(), live.Address); IsThreadBusy(err) {
		t.Fatalf("detach after the relaunch finished was refused busy: %v", err)
	}
}

// The resume-then-relaunch order the Done-when names: a relaunch on a thread a
// resume holds is refused naming the resume.
func TestARelaunchDuringAResumeIsRefused(t *testing.T) {
	env, live := envWithLiveThread(t)
	c := env.Couch
	_, release, err := c.hold(context.Background(), live.Address, "resume")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	_, err = c.Relaunch(context.Background(), live.Address)
	var busy *ThreadBusyError
	if !errors.As(err, &busy) || busy.Running != "resume" {
		t.Fatalf("relaunch during resume = %v, want ThreadBusyError naming resume", err)
	}
	thread, getErr := c.Threads.GetThread(live.Address)
	if getErr != nil || len(thread.Incarnations) != 1 || thread.Park != nil {
		t.Fatalf("a refused relaunch changed the thread: %+v, %v", thread, getErr)
	}
}

func TestTheGateDoesNotSerialiseDifferentThreads(t *testing.T) {
	env, live := envWithLiveThread(t)
	c := env.Couch
	_, release, err := c.hold(context.Background(), live.Address, "relaunch")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	other := ThreadAddress{RepoScope: live.Address.RepoScope, Tag: "couch-fedcba9876543210"}
	if _, err := c.Detach(context.Background(), other); err == nil || IsThreadBusy(err) {
		t.Fatalf("detach of a different thread = %v, want its own (not-found) answer, not busy", err)
	}
}

// leaveEnv is one live, detachable thread under a bare Couch.
func leaveEnv(t *testing.T) (*Couch, ThreadRecord) {
	t.Helper()
	store, _, thread := createControllerThread(t)
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(thread.Address, "pair-exact", true)
	proc := NewFakeProcOps()
	incarnation := thread.Incarnations[0]
	proc.Set(incarnation.PID, incarnation.Identity)
	proc.DiesOn = map[int]os.Signal{incarnation.PID: syscall.SIGTERM}
	return &Couch{Threads: store, Proc: proc, Artifacts: artifacts, Clock: FixedClock{T: time.Unix(100, 0).UTC()}, sleep: func(time.Duration) {}}, thread
}

// Leave is a drain: it waits for a thread's holder instead of refusing or
// skipping it.
func TestLeaveWaitsForAHolder(t *testing.T) {
	c, thread := leaveEnv(t)
	_, release, err := c.hold(context.Background(), thread.Address, "resume")
	if err != nil {
		t.Fatal(err)
	}
	type left struct {
		result LeaveResult
		err    error
	}
	done := make(chan left, 1)
	go func() {
		result, err := c.Leave(context.Background(), LeaveDetach)
		done <- left{result, err}
	}()
	select {
	case got := <-done:
		t.Fatalf("Leave returned while the thread was held: %+v, %v", got.result, got.err)
	case <-time.After(150 * time.Millisecond):
	}
	release()
	got := <-done
	if got.err != nil || len(got.result.Detached) != 1 || got.result.Detached[0] != thread.Address {
		t.Fatalf("Leave = %+v, %v; want the thread detached after the holder released", got.result, got.err)
	}
}

// Leave decides from the record as it is after the wait, not from its
// snapshot: a thread archived while Leave waited is skipped, not acted on.
func TestLeaveDecidesFromTheRecordAfterWaiting(t *testing.T) {
	c, thread := leaveEnv(t)
	_, release, err := c.hold(context.Background(), thread.Address, "archive")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	var result LeaveResult
	go func() {
		var err error
		result, err = c.Leave(context.Background(), LeaveDetach)
		done <- err
	}()
	time.Sleep(20 * time.Millisecond) // Leave has taken its snapshot and is waiting
	if err := c.Threads.ArchiveThread(thread.Address); err != nil {
		t.Fatal(err)
	}
	release()
	if err := <-done; err != nil {
		t.Fatalf("Leave = %v, want an archived thread skipped silently", err)
	}
	if len(result.Detached)+len(result.Parked)+len(result.Skipped) != 0 {
		t.Fatalf("Leave acted on a thread archived while it waited: %+v", result)
	}
	if got := c.Proc.(*FakeProcOps).Signals; len(got) != 0 {
		t.Fatalf("Leave signalled an archived thread's process: %+v", got)
	}
}

// An abort whose thread now carries another incarnation ends only its own
// helper and terminal; the newer session and record are left alone.
func TestAMismatchedAbortStillClosesItsOwnHandle(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record, handle, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	// Another operation replaced the incarnation after this start's hold
	// was released.
	thread, err := env.Couch.Threads.GetThread(record.Thread)
	if err != nil {
		t.Fatal(err)
	}
	replaced, err := env.Couch.Threads.updateExistingThread(record.Thread, thread.Revision, func(next *ThreadRecord) error {
		next.Incarnations = []ThreadIncarnation{{PID: 9999, Identity: "newer-holder", State: IncarnationLive}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	cause := errors.New("attach failed")
	if err := env.Couch.AbortStarted(context.Background(), StartResult{Record: record, Handle: handle}, cause); !errors.Is(err, cause) {
		t.Fatalf("AbortStarted = %v, want the cause", err)
	}
	if handle.Alive() {
		t.Fatal("the aborted start's own helper survived")
	}
	if got := env.Artifacts.Quiesces(); len(got) != 0 {
		t.Fatalf("a mismatched abort quiesced the thread's session: %+v", got)
	}
	after, err := env.Couch.Threads.GetThread(record.Thread)
	if err != nil || after.Revision != replaced.Revision {
		t.Fatalf("a mismatched abort touched the newer holder's record: %+v, %v", after, err)
	}
	if got := env.Couch.reg.Records(); len(got) != 0 {
		t.Fatalf("the aborted actor stayed registered: %+v", got)
	}
}

// Inside a composite that already holds the thread, AbortStarted re-enters
// through the caller's context instead of waiting on its own caller.
func TestAbortStartedInsideAComposite(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record, handle, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	held, release, err := env.Couch.hold(context.Background(), record.Thread, "continue-thread")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	done := make(chan error, 1)
	go func() {
		done <- env.Couch.AbortStarted(held, StartResult{Record: record, Handle: handle}, errors.New("attach failed"))
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("AbortStarted inside its holder's context waited on its own caller")
	}
}

// Joining an open park transaction without the gate may only drive that
// transaction: a mode that could begin a new one becomes retry (BR-1).
func TestParkJoinModeNeverBegins(t *testing.T) {
	for mode, want := range map[string]string{"": "retry", "normal": "retry", "retry": "retry", "recover": "recover", "abandon": "abandon"} {
		if got := parkJoinMode(mode); got != want {
			t.Errorf("parkJoinMode(%q) = %q, want %q", mode, got, want)
		}
	}
}

// RecoverActiveParks is a drain: it waits for a thread's holder, then
// recovers the open park (BR-4).
func TestParkRecoveryWaitsForAHolderThenRecovers(t *testing.T) {
	store, _, thread := createControllerThread(t)
	identity := ParkIdentity{Nonce: "park-wait", Address: thread.Address, PID: 42, ProcessIdentity: "pair-helper"}
	thread, err := store.BeginPark(thread.Address, thread.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(300, 0).UTC()
	model := pairlifecycletest.New(now)
	model.SetSession("pair-exact", true)
	lifecycle := &fakeControllerLifecycle{model: model}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(thread.Address, "pair-exact", true)
	artifacts.TriggerQuitHook = func(string, launcher.QuitIntent) error {
		completion := successCompletion(lifecycle.lastRequest, now)
		lifecycle.completion = &completion
		return nil
	}
	controller := &PairLifecycleController{
		Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
		Proc: NewFakeProcOps(), Clock: FixedClock{T: now},
		Nonce: func() (string, error) { return "unused", nil },
	}
	c := &Couch{Threads: store, PairLifecycle: controller}
	_, release, err := c.hold(context.Background(), thread.Address, "resume")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- c.RecoverActiveParks(context.Background()) }()
	select {
	case err := <-done:
		t.Fatalf("park recovery returned while the thread was held: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if got := len(artifacts.TriggeredQuits()); got != 0 {
		t.Fatalf("park recovery acted on a held thread: %d quits", got)
	}
	release()
	if err := <-done; err != nil {
		t.Fatalf("park recovery after release = %v", err)
	}
	recovered, err := store.GetThread(thread.Address)
	if err != nil || recovered.Park != nil || recovered.VerifiedPark == nil {
		t.Fatalf("recovered thread = %+v, %v; want the park closed and verified", recovered, err)
	}
}

// AbortStarted waits for a holder; once released, a start that still owns its
// thread is cleaned up in full, session included (BR-4).
func TestAbortStartedWaitsForAHolderThenQuiesces(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record, handle, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	_, release, err := env.Couch.hold(context.Background(), record.Thread, "detach")
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("attach failed")
	done := make(chan error, 1)
	go func() {
		done <- env.Couch.AbortStarted(context.Background(), StartResult{Record: record, Handle: handle}, cause)
	}()
	select {
	case err := <-done:
		t.Fatalf("AbortStarted returned while the thread was held: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if got := env.Artifacts.Quiesces(); len(got) != 0 {
		t.Fatalf("AbortStarted quiesced a held thread's session: %+v", got)
	}
	release()
	if err := <-done; !errors.Is(err, cause) {
		t.Fatalf("AbortStarted = %v, want the cause", err)
	}
	if got := env.Artifacts.Quiesces(); len(got) != 1 || got[0] != record.Thread {
		t.Fatalf("quiesces = %+v, want the owned session after the wait", got)
	}
}

// A cancelled abort wait still ends the start's own helper and terminal, and
// touches nothing address-scoped (BR-4).
func TestACancelledAbortWaitEndsOnlyItsOwnHelper(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record, handle, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	_, release, err := env.Couch.hold(context.Background(), record.Thread, "detach")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- env.Couch.AbortStarted(ctx, StartResult{Record: record, Handle: handle}, errors.New("attach failed"))
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("AbortStarted = %v, want the cancellation", err)
	}
	if handle.Alive() {
		t.Fatal("a cancelled abort left its own helper running")
	}
	if got := env.Artifacts.Quiesces(); len(got) != 0 {
		t.Fatalf("a cancelled abort quiesced the session without the hold: %+v", got)
	}
	if got := env.Couch.actorRegistry().Records(); len(got) != 0 {
		t.Fatalf("a cancelled abort left its actor registered: %+v", got)
	}
}
