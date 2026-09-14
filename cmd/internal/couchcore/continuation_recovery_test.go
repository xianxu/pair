package couchcore

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type continuationFixture struct {
	env        *relaunchEnv
	source     ThreadRecord
	status     ContinuationStatus
	registered map[string]bool
	delivery   orientation.DeliveryState
	launches   int
}

func newContinuationFixture(t *testing.T) *continuationFixture {
	t.Helper()
	env, source := switchEnvWithLiveThread(t)
	f := &continuationFixture{env: env, source: source, registered: map[string]bool{}}
	proof := ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 2}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) { return proof, nil }
	path := filepath.Join(t.TempDir(), "checkpoint.md")
	if err := os.WriteFile(path, []byte("---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nContinue exact work.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	status, err := env.Couch.RequestContinuation(context.Background(), source.Address, proof, path)
	if err != nil {
		t.Fatal(err)
	}
	f.status = status
	priorQuit := env.Artifacts.TriggerQuitHook
	env.Artifacts.TriggerQuitHook = func(session string, intent launcher.QuitIntent) error {
		err := priorQuit(session, intent)
		if err == nil {
			env.Artifacts.SetPairSession(source.Address, "pair-exact", false)
		}
		return err
	}
	env.Couch.FreshRegistration = func(_ context.Context, _ ThreadAddress, _ string, attempt string) (bool, error) {
		return f.registered[attempt], nil
	}
	env.Couch.OrientationStatus = func(context.Context, ThreadAddress, string, string) (orientation.DeliveryState, error) {
		return f.delivery, nil
	}
	env.Runner.AfterAcknowledge = func(id string) error {
		f.launches++
		record, err := env.Couch.Threads.GetThread(source.Address)
		if err != nil {
			return err
		}
		inc := record.Incarnations[0]
		env.Proc.Set(inc.PID, inc.Identity)
		env.Artifacts.SetPairSession(source.Address, "pair-exact", true)
		if inc.Start != nil && inc.Start.Nonce == record.Continuation.Attempt {
			f.registered[record.Continuation.Attempt] = true
		}
		return nil
	}
	return f
}
func TestContinuationRetryAfterProvedTargetDeath(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	first, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	f.delivery = orientation.DeliveryState{Phase: orientation.DeliveryIndeterminate, BodyWritten: true, Reason: "interrupted"}
	failed, err := c.ReconcileContinuation(context.Background(), f.source.Address, f.status.RequestID, first.Status.Attempt)
	if err == nil || failed.Phase != checkpoint.Failed {
		t.Fatalf("failure %+v %v", failed, err)
	}
	// Retry while the target is live reobserves only; uncertain input is never
	// pasted again and a second helper is not authorized.
	if _, err := c.RetryContinuation(context.Background(), f.source.Address, f.status.RequestID); err == nil {
		t.Fatal("uncertain target reported complete")
	}
	if f.launches != 1 {
		t.Fatalf("live target duplicated %d", f.launches)
	}
	f.registered[first.Status.Attempt] = false
	f.env.Proc.Kill(first.Record.PID)
	f.env.Runner.SetExited(first.Handle.ID(), 0)
	f.env.Artifacts.SetPairSession(f.source.Address, "pair-exact", false)
	f.delivery = orientation.DeliveryState{}
	second, err := c.RetryContinuation(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if f.launches != 2 || second.Status.Attempt == first.Status.Attempt || second.Record.Thread != first.Record.Thread {
		t.Fatalf("retry %+v launches=%d", second, f.launches)
	}
}
func TestContinuationRegistrationCrashReconcilesWithoutSpawn(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	first, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	record, _ := c.Threads.GetThread(f.source.Address)
	if _, err := c.Threads.UpdateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error { next.Continuation.Target = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	f.delivery = orientation.DeliveryState{Phase: orientation.DeliverySubmitted}
	again, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Status.Phase != checkpoint.Complete || again.Handle != nil || f.launches != 1 || again.Status.Attempt != first.Status.Attempt {
		t.Fatalf("recovery %+v", again)
	}
}
func TestContinuationForkFailureRetainsParkAndSnapshot(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	f.env.Runner.FailNextStart(errors.New("fork unavailable"))
	if _, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID); err == nil {
		t.Fatal("fork failure ignored")
	}
	record, err := c.Threads.GetThread(f.source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if record.VerifiedPark == nil || len(record.Incarnations) != 0 || record.Continuation.Phase != checkpoint.Failed || record.Continuation.Checkpoint.Body == "" {
		t.Fatalf("failure lost state %+v", record)
	}
	if _, err := c.RetryContinuation(context.Background(), f.source.Address, f.status.RequestID); err != nil {
		t.Fatal(err)
	}
	if f.launches != 1 {
		t.Fatalf("retry launches %d", f.launches)
	}
}

func TestContinuationWarmTargetRecoveryAfterOwnerDeath(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	first, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	f.env.Proc.Kill(first.Record.PID)
	f.env.Runner.SetExited(first.Handle.ID(), 0)
	c.reg = c.reg.RemoveActor(first.Record.Args.Worktree, first.Record.ID)
	record, _ := c.Threads.GetThread(f.source.Address)
	if _, err := c.Threads.UpdateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error {
		next.Incarnations[0].State = IncarnationUnknown
		next.Continuation.Target = nil
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	f.env.Artifacts.SetDetachedSession(record.Address, "pair-exact")
	f.delivery = orientation.DeliveryState{Phase: orientation.DeliverySubmitted}
	recovered, err := c.Continue(context.Background(), record.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Status.Phase != checkpoint.Running || recovered.Handle == nil || recovered.Record.Shape != StartWarmReattach || recovered.Status.Attempt != first.Status.Attempt {
		t.Fatalf("warm recovery %+v", recovered)
	}
	completed, err := c.ReconcileContinuation(context.Background(), record.Address, f.status.RequestID, first.Status.Attempt)
	if err != nil || completed.Phase != checkpoint.Complete {
		t.Fatalf("receipt after adoption %+v %v", completed, err)
	}
	child := f.env.Runner.Child(recovered.Handle.ID())
	for _, arg := range child.Env {
		if len(arg) > len(launcher.CouchLaunchProfileEnv) && arg[:len(launcher.CouchLaunchProfileEnv)] == launcher.CouchLaunchProfileEnv {
			t.Fatal("target recovery started another fresh conversation")
		}
	}
}

func TestContinuationWarmSourceRecoveryHandsOffHelperBeforePark(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	f.env.Proc.Kill(42)
	c.reg = c.reg.RemoveActor(Worktree(f.source.StartingPath), "source-actor")
	f.env.Artifacts.SetDetachedSession(f.source.Address, "pair-exact")
	first, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Handle == nil || first.Record.Shape != StartWarmReattach || first.Status.Phase != checkpoint.Running {
		t.Fatalf("source warm attach %+v", first)
	}
	record, _ := c.Threads.GetThread(f.source.Address)
	if record.Park != nil || record.Continuation.Source.Helper.PID != first.Record.PID {
		t.Fatal("source was parked before its helper was adopted")
	}
	// The existing lifecycle fixture kills its original helper. Warm ownership
	// changed, so model the quit effect on the freshly recorded exact helper.
	prior := f.env.Artifacts.TriggerQuitHook
	f.env.Artifacts.TriggerQuitHook = func(session string, intent launcher.QuitIntent) error {
		err := prior(session, intent)
		if err == nil {
			f.env.Proc.Kill(first.Record.PID)
			f.env.Runner.SetExited(first.Handle.ID(), 0)
		}
		return err
	}
	second, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if second.Handle == nil || second.Record.Shape != StartFreshExisting || second.Record.Thread != first.Record.Thread || second.Status.Attempt != first.Status.Attempt {
		t.Fatalf("fresh after warm %+v", second)
	}
}

func TestContinuationArchivePreservesSnapshotRemovesDerivedCopy(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	started, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	f.delivery = orientation.DeliveryState{Phase: orientation.DeliverySubmitted}
	if _, err := c.ReconcileContinuation(context.Background(), f.source.Address, f.status.RequestID, started.Status.Attempt); err != nil {
		t.Fatal(err)
	}
	record, _ := c.Threads.GetThread(f.source.Address)
	if _, err := c.Threads.UpdateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error { next.Incarnations = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := c.Threads.ArchiveThread(record.Address); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(c.continuationPath(record.Address)); !os.IsNotExist(err) {
		t.Fatalf("derived snapshot survived archive: %v", err)
	}
	archived, err := c.Threads.ArchivedThreads()
	if err != nil || len(archived) != 1 || archived[0].Continuation.Checkpoint.Body != record.Continuation.Checkpoint.Body {
		t.Fatalf("archive lost snapshot %+v %v", archived, err)
	}
}

func TestContinuationRetryDoesNotStealUnrecordedHelperFromLiveOwner(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	f.env.Runner.FailNextStart(errors.New("fork unavailable"))
	if _, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID); err == nil {
		t.Fatal("expected fork failure")
	}
	record, _ := c.Threads.GetThread(f.source.Address)
	attempt := "start-1234567890abcdef"
	record, err := c.Threads.AdvanceContinuation(record.Address, record.Revision, checkpoint.Event{Kind: checkpoint.RetryAbsent, RequestID: f.status.RequestID, Attempt: attempt})
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := c.Proc.Current()
	record, err = c.Threads.CommitStartClaim(record.Address, record.Revision, "repo", c.Clock.Now(), StartEvent{Kind: StartClaimed, Nonce: attempt, Owner: SupervisorOwner{PID: owner.PID, Identity: owner.Identity}, Profile: record.LatestLaunchProfile, Shape: StartFreshExisting})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Threads.AdvanceContinuation(record.Address, record.Revision, checkpoint.Event{Kind: checkpoint.Fail, RequestID: f.status.RequestID, Attempt: attempt, Failure: "interrupted before helper record"}); err != nil {
		t.Fatal(err)
	}
	if _, err = c.RetryContinuation(context.Background(), record.Address, f.status.RequestID); err == nil {
		t.Fatal("stole claim while owner could still fork")
	}
	if f.launches != 0 {
		t.Fatal("launched duplicate owner")
	}
}

func TestContinuationUnconfirmedSubmissionTimesOutDurably(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	result, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	c.Clock = FixedClock{T: f.env.Now.Add(31 * time.Second)}
	status, err := c.ReconcileContinuation(context.Background(), f.source.Address, f.status.RequestID, result.Status.Attempt)
	if err == nil || status.Phase != checkpoint.Failed {
		t.Fatalf("unconfirmed receipt never timed out: %+v %v", status, err)
	}
	record, _ := c.Threads.GetThread(f.source.Address)
	if record.Continuation.Target == nil || record.Continuation.Checkpoint.Body == "" {
		t.Fatal("timeout lost recovery evidence")
	}
}
