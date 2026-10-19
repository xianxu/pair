package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type continuationFixture struct {
	env        *relaunchEnv
	source     ThreadRecord
	status     ContinuationStatus
	registered map[string]bool
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
			env.Artifacts.SetPairSession(source.Address, session, false)
		}
		return err
	}
	env.Couch.FreshRegistration = func(_ context.Context, _ ThreadAddress, _ string, attempt string) (bool, error) {
		return f.registered[attempt], nil
	}
	env.Runner.AfterAcknowledge = func(id string) error {
		f.launches++
		record, err := env.Couch.Threads.GetThread(source.Address)
		if err != nil {
			return err
		}
		inc := record.Incarnations[0]
		env.Proc.Set(inc.PID, inc.Identity)
		env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
		if record.Continuation != nil && inc.Start != nil && inc.Start.Nonce == record.Continuation.Attempt {
			f.registered[record.Continuation.Attempt] = true
		}
		return nil
	}
	return f
}

func TestContinuationRegistrationCrashReconcilesWithoutSpawn(t *testing.T) {
	t.Parallel()

	f := newContinuationFixture(t)
	c := f.env.Couch
	first, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	record, _ := c.Threads.GetThread(f.source.Address)
	if _, err := c.Threads.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error { unrecordTarget(next); return nil }); err != nil {
		t.Fatal(err)
	}
	again, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Status.Phase != checkpoint.Complete || again.Handle != nil || f.launches != 1 || again.Status.Attempt != first.Status.Attempt {
		t.Fatalf("recovery %+v", again)
	}
}
func TestContinuationForkFailureLeavesTheLineAndResumesTheOldConversation(t *testing.T) {
	t.Parallel()

	f := newContinuationFixture(t)
	c := f.env.Couch
	f.env.Runner.FailNextStart(errors.New("fork unavailable"))
	result, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err == nil || result.Missed == nil || !strings.Contains(result.Missed.Reason, "fork unavailable") || result.Missed.TargetMayExist {
		t.Fatalf("fork failure = %+v, %v", result.Missed, err)
	}
	record, err := c.Threads.GetThread(f.source.Address)
	if err != nil {
		t.Fatal(err)
	}
	// The park stays the resume authority; the line names the continuation
	// and blocks nothing.
	if record.VerifiedPark == nil || len(record.Incarnations) != 0 || record.Continuation != nil || record.ContinuationMissed == nil || record.ContinuationMissed.Path != record.PreviousSession.Continuation && record.ContinuationMissed.Path == "" {
		t.Fatalf("failure lost state %+v", record)
	}
	if err := continuationGuard(record); err != nil {
		t.Fatalf("the line blocks: %v", err)
	}
}

func TestContinuationWarmTargetRecoveryAfterOwnerDeath(t *testing.T) {
	t.Parallel()

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
	if _, err := c.Threads.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error {
		next.Incarnations[0].State = IncarnationUnknown
		unrecordTarget(next)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	f.env.Artifacts.SetDetachedSession(record.Address, record.SessionBinding.Name)
	recovered, err := c.Continue(context.Background(), record.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Status.Phase != checkpoint.Complete || recovered.Handle == nil || recovered.Record.Shape != StartWarmReattach || recovered.Status.Attempt != first.Status.Attempt {
		t.Fatalf("warm recovery %+v", recovered)
	}
	child := f.env.Runner.Child(recovered.Handle.ID())
	for _, arg := range child.Env {
		if len(arg) > len(launcher.CouchLaunchProfileEnv) && arg[:len(launcher.CouchLaunchProfileEnv)] == launcher.CouchLaunchProfileEnv {
			t.Fatal("target recovery started another fresh conversation")
		}
	}
}

func TestContinuationWarmSourceRecoveryHandsOffHelperBeforePark(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	f := newContinuationFixture(t)
	c := f.env.Couch
	if _, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID); err != nil {
		t.Fatal(err)
	}
	record, _ := c.Threads.GetThread(f.source.Address)
	if _, err := c.Threads.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error { next.Incarnations = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := c.Threads.ArchiveThread(record.Address, retireAuthorityForTest(t, c.Threads, record.Address)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mustContinuationPath(t, c, record.Address)); !os.IsNotExist(err) {
		t.Fatalf("derived snapshot survived archive: %v", err)
	}
	archived, err := c.Threads.ArchivedThreads()
	if err != nil || len(archived) != 1 || archived[0].Continuation.Checkpoint.Body != record.Continuation.Checkpoint.Body {
		t.Fatalf("archive lost snapshot %+v %v", archived, err)
	}
}

func TestRegisteredUnknownTargetRecoveryInterruptionAndProofRefusals(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"cancel-after-retirement", "receipt-lost", "helper-live-again", "revision-changed"} {
		t.Run(mode, func(t *testing.T) {
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
			record, err = c.Threads.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error {
				next.Incarnations[0].State = IncarnationUnknown
				unrecordTarget(next)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			f.env.Artifacts.SetDetachedSession(record.Address, record.SessionBinding.Name)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			originalStore := c.Threads
			switch mode {
			case "cancel-after-retirement":
				c.Threads = newThreadStoreWithHooks(c.Namespace, threadStoreHooks{AfterTarget: func(int) error { cancel(); return nil }})
			case "receipt-lost":
				c.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return false, nil }
			case "helper-live-again":
				c.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) {
					inc := record.Incarnations[0]
					f.env.Proc.Set(inc.PID, inc.Identity)
					return true, nil
				}
			case "revision-changed":
				c.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) {
					_, err := originalStore.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error { next.PublishedSummary = "concurrent"; return nil })
					return true, err
				}
			}
			_, _, _, err = c.ensureContinuationAttached(ctx, record, true)
			if err == nil {
				t.Fatal("recovery unexpectedly attached")
			}
			c.Threads = originalStore
			after, readErr := c.Threads.GetThread(record.Address)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if f.launches != 1 {
				t.Fatalf("spawned before proof: %d", f.launches)
			}
			if mode != "cancel-after-retirement" {
				expected := cloneThreadRecord(record)
				if mode == "revision-changed" {
					expected.Revision++
					expected.PublishedSummary = "concurrent"
				}
				if !reflect.DeepEqual(after, expected) {
					t.Fatalf("refusal changed durable record: %+v", after)
				}
				return
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("lost cancellation: %v", err)
			}
			expected := cloneThreadRecord(record)
			expected.Revision++
			expected.Incarnations = nil
			if len(after.Incarnations) != 0 {
				t.Fatalf("retired helper survived: %+v", after.Incarnations)
			}
			after.Incarnations = nil // persisted decoding normalizes an empty collection
			if !reflect.DeepEqual(after, expected) {
				t.Fatalf("interrupted retirement resurrected helper or changed history: %+v", after)
			}
			again, err := c.Continue(context.Background(), record.Address, f.status.RequestID)
			if err != nil {
				t.Fatal(err)
			}
			if again.Record.Shape != StartWarmReattach || again.Status.Attempt != first.Status.Attempt {
				t.Fatalf("retry changed conversation: %+v", again)
			}
		})
	}
}

// Observe the actual terminal identity emitted by the launch transaction. The
// source fixture is legacy, but each replacement gets a separately allocated M.
func continuationChildSession(t *testing.T, runner *FakeRunner, id string) string {
	t.Helper()
	child := runner.Child(id)
	for _, entry := range child.Env {
		if raw, ok := strings.CutPrefix(entry, launcher.CouchSessionIntentEnv+"="); ok {
			var intent launcher.CouchSessionIntent
			if err := json.Unmarshal([]byte(raw), &intent); err != nil {
				t.Fatal(err)
			}
			if err := intent.Validate(); err != nil {
				t.Fatal(err)
			}
			return intent.Name
		}
	}
	t.Fatal("continuation child has no managed terminal intent")
	return ""
}
func (f *continuationFixture) sessionName(t *testing.T) string {
	t.Helper()
	record, err := f.env.Couch.Threads.GetThread(f.source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if record.SessionBinding != nil {
		return record.SessionBinding.Name
	}
	return "pair-exact"
}

func TestContinuationRegisteredTargetUsesNewTerminalBinding(t *testing.T) {
	t.Parallel()

	f := newContinuationFixture(t)
	first, err := f.env.Couch.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	record, err := f.env.Couch.Threads.GetThread(f.source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if record.SessionBinding == nil || record.SessionBinding.Name == record.Continuation.Source.Session {
		t.Fatal("fixture did not allocate a distinct target terminal")
	}
	record.Incarnations[0].State = IncarnationUnknown
	unrecordTarget(&record)
	proof := RegisteredTargetProof{RequestID: record.Continuation.ID, Agent: record.Continuation.Source.Agent, Session: record.Continuation.Source.Session, Attempt: first.Status.Attempt, Helper: ProcessIdentity{PID: first.Record.PID, Identity: first.Record.Identity}}
	if _, err := ReconcileRegisteredTarget(record, proof); err == nil {
		t.Fatal("previous source session authorized target retirement")
	}
	proof.Session = record.SessionBinding.Name
	if _, err := ReconcileRegisteredTarget(record, proof); err != nil {
		t.Fatalf("exact promoted target refused: %v", err)
	}
	for _, session := range []string{record.Continuation.Source.Session, "📁999-999", record.SessionBinding.Name} {
		f.env.Couch.ContinuationGeneration = func(context.Context, ThreadAddress, string, string) (*checkpoint.TargetGeneration, error) {
			return &checkpoint.TargetGeneration{Agent: proof.Agent, Session: session, Attempt: proof.Attempt, LaunchOrdinal: record.Continuation.Source.LaunchOrdinal + 1}, nil
		}
		_, err := f.env.Couch.continuationTargetGeneration(context.Background(), record)
		if (err == nil) != (session == record.SessionBinding.Name) {
			t.Fatalf("target generation session=%q err=%v", session, err)
		}
	}
}

// unrecordTarget models Couch dying after the target registered and before it
// recorded that: the request is still running and names no target.
func unrecordTarget(next *ThreadRecord) {
	next.Continuation.Phase = checkpoint.Running
	next.Continuation.Target = nil
}

// BR-1 (#522 M1 review): when the compaction's previous-session write fails
// after the source was parked, the request still ends as the thread's line --
// the error path keeps its record rather than replacing it.
func TestAFailedPreviousSessionWriteLeavesTheLine(t *testing.T) {
	t.Parallel()

	f := newContinuationFixture(t)
	c := f.env.Couch
	c.Threads.hooks.BeforeUpdate = func(current, next ThreadRecord) error {
		if next.PreviousSession != nil && next.PreviousSession.Continuation != "" && (current.PreviousSession == nil || current.PreviousSession.Continuation == "") {
			return errors.New("injected previous-session write failure")
		}
		return nil
	}
	result, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err == nil || result.Missed == nil || !strings.Contains(result.Missed.Reason, "injected previous-session write failure") {
		t.Fatalf("= %+v, %v; want the line", result.Missed, err)
	}
	if f.launches != 0 {
		t.Fatalf("a target launched after the failure: %d", f.launches)
	}
	record, err := c.Threads.GetThread(f.source.Address)
	if err != nil || record.Continuation != nil || record.ContinuationMissed == nil || record.VerifiedPark == nil {
		t.Fatalf("after: %+v / %+v, %v", record.Continuation, record.ContinuationMissed, err)
	}
}
