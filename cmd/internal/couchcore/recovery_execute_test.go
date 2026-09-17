package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestRecoverThreadWarmPreservesSessionWithoutNativeBinding(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	env.Proc.Kill(source.Incarnations[0].PID)
	env.Artifacts.SetDetachedSession(source.Address, "pair-exact")
	env.Artifacts.SetNativeBinding(source.Address, "claude", sessioninventory.BindingProvisional, "")
	result, err := env.Couch.RecoverThread(context.Background(), source.Address, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Handle == nil || result.Record.Shape != StartWarmReattach {
		t.Fatalf("not warm: %+v", result)
	}
	record, err := env.Couch.Threads.GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if record.VerifiedPark != nil || record.Continuation != nil || record.LastActiveAt != source.LastActiveAt {
		t.Fatalf("recovery fabricated park/continuation/activity: %+v", record)
	}
	if len(env.Artifacts.quiesced) != 0 {
		t.Fatal("warm recovery stopped session")
	}
}

func TestRecoverThreadAbsentSourceRetainsExactSnapshot(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	before, err := c.Threads.GetThread(f.source.Address)
	if err != nil {
		t.Fatal(err)
	}
	f.env.Proc.Kill(f.source.Incarnations[0].PID)
	f.env.Artifacts.SetPairSession(f.source.Address, "pair-exact", false)
	if err := os.Remove(before.Continuation.Checkpoint.SourcePath); err != nil {
		t.Fatal(err)
	}
	result, err := c.RecoverThread(context.Background(), f.source.Address, "")
	if err != nil {
		t.Fatal(err)
	}
	record, err := c.Threads.GetThread(f.source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if record.VerifiedPark != nil || len(record.ParkHistory) != 0 || record.Continuation.SourceAbsence == nil || record.Continuation.Checkpoint != before.Continuation.Checkpoint || f.launches != 1 || result.Orientation == nil {
		t.Fatalf("bad absence recovery: %+v launches=%d", record, f.launches)
	}
}

func TestRecoverThreadImportsLegacyCheckpointAfterRetirement(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	old, _ := c.Threads.GetThread(f.source.Address)
	path := old.Continuation.Checkpoint.SourcePath
	cp := old.Continuation.Checkpoint
	f.env.Proc.Kill(f.source.Incarnations[0].PID)
	f.env.Artifacts.SetPairSession(f.source.Address, "pair-exact", false)
	_, err := c.Threads.updateExistingThread(old.Address, old.Revision, func(r *ThreadRecord) error { r.Continuation = nil; r.Incarnations = nil; return nil })
	if err != nil {
		t.Fatal(err)
	}
	result, err := c.RecoverThread(context.Background(), f.source.Address, path)
	if err != nil {
		t.Fatal(err)
	}
	record, _ := c.Threads.GetThread(f.source.Address)
	if result.Handle == nil || record.Continuation.Source.Helper != (checkpoint.Process{}) || record.Continuation.SourceAbsence == nil || record.Continuation.Checkpoint != cp {
		t.Fatalf("legacy import %+v", record)
	}
}

func TestRecoverThreadRefusesUnprovedOwnershipBeforeMutation(t *testing.T) {
	for _, kind := range []string{"live-helper", "active-client", "observer-error", "missing-checkpoint", "unreadable-checkpoint", "wrong-agent-checkpoint"} {
		t.Run(kind, func(t *testing.T) {
			f := newContinuationFixture(t)
			c := f.env.Couch
			if kind != "live-helper" {
				f.env.Proc.Kill(f.source.Incarnations[0].PID)
			}
			if kind == "observer-error" {
				f.env.Artifacts.BeforePairSession = func(ThreadAddress) error { return errors.New("cannot inspect") }
			}
			path := ""
			if strings.HasSuffix(kind, "checkpoint") {
				f.env.Artifacts.SetPairSession(f.source.Address, "pair-exact", false)
				path = "/absent/checkpoint.md"
				if kind == "unreadable-checkpoint" {
					path = t.TempDir()
				} // A directory is never a readable checkpoint file.
				if kind == "wrong-agent-checkpoint" {
					path = filepath.Join(t.TempDir(), "different-agent.md")
					if err := os.WriteFile(path, []byte("---\ntype: continuation\nagent: codex\n---\n## NEXT ACTION\nContinue the selected task.\n"), 0600); err != nil {
						t.Fatal(err)
					}
					existing, err := c.Threads.GetThread(f.source.Address)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := c.Threads.updateExistingThread(existing.Address, existing.Revision, func(next *ThreadRecord) error { next.Continuation = nil; return nil }); err != nil {
						t.Fatal(err)
					}
				}
			}
			before, _ := c.Threads.GetThread(f.source.Address)
			_, err := c.RecoverThread(context.Background(), f.source.Address, path)
			if err == nil {
				t.Fatal("unproved recovery accepted")
			}
			if kind == "wrong-agent-checkpoint" && !strings.Contains(err.Error(), "checkpoint agent") {
				t.Fatalf("wrong-agent import failed before exercising agent match: %v", err)
			}
			if kind == "unreadable-checkpoint" && !strings.Contains(err.Error(), "regular file") {
				t.Fatalf("unreadable import did not reach checkpoint reader: %v", err)
			}
			after, _ := c.Threads.GetThread(f.source.Address)
			if !reflect.DeepEqual(before, after) || f.launches != 0 {
				t.Fatalf("refusal mutated source: %v", err)
			}
		})
	}
}

func TestAdmitRecoveryGenerationRejectsUnownedAdvancement(t *testing.T) {
	f := newContinuationFixture(t)
	r, _ := f.env.Couch.Threads.GetThread(f.source.Address)
	r.Continuation.PreviousTargetGeneration = &checkpoint.TargetGeneration{Agent: "claude", Session: "pair-exact", Attempt: "previous", LaunchOrdinal: 4}
	for _, ordinal := range []uint64{2, 3, 4, 5} {
		err := AdmitRecoveryGeneration(*r.Continuation, ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: ordinal})
		if (err == nil) != (ordinal == 2 || ordinal == 4) {
			t.Fatalf("ordinal%d: %v", ordinal, err)
		}
	}
	if err := AdmitRecoveryGeneration(*r.Continuation, ContinuationSource{Agent: "codex", Session: "pair-exact", LaunchOrdinal: 4}); err == nil || !strings.Contains(err.Error(), "generation") {
		t.Fatalf("foreign agent admitted: %v", err)
	}
}

func TestRecoverContinuationRetryDistinguishesOwnTargetGeneration(t *testing.T) {
	for _, scenario := range []string{"owned", "foreign", "registration-crash"} {
		t.Run(scenario, func(t *testing.T) {
			f := newContinuationFixture(t)
			c := f.env.Couch
			ordinal := uint64(2)
			generations := map[string]checkpoint.TargetGeneration{}
			c.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) {
				return ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: ordinal}, nil
			}
			c.ContinuationGeneration = func(_ context.Context, _ ThreadAddress, _ string, attempt string) (*checkpoint.TargetGeneration, error) {
				g, ok := generations[attempt]
				if !ok {
					return nil, nil
				}
				return &g, nil
			}
			prior := f.env.Runner.AfterAcknowledge
			f.env.Runner.AfterAcknowledge = func(id string) error {
				if err := prior(id); err != nil {
					return err
				}
				ordinal++
				r, _ := c.Threads.GetThread(f.source.Address)
				generations[r.Continuation.Attempt] = checkpoint.TargetGeneration{Agent: "claude", Session: "pair-exact", Attempt: r.Continuation.Attempt, LaunchOrdinal: ordinal}
				return nil
			}
			f.env.Proc.Kill(f.source.Incarnations[0].PID)
			f.env.Artifacts.SetPairSession(f.source.Address, "pair-exact", false)
			first, err := c.RecoverThread(context.Background(), f.source.Address, "")
			if err != nil {
				t.Fatal(err)
			}
			f.env.Proc.Kill(first.Record.PID)
			f.env.Runner.SetExited(first.Handle.ID(), 0)
			f.env.Artifacts.SetPairSession(f.source.Address, "pair-exact", false)
			f.registered[first.Status.Attempt] = false
			_, err = c.ReconcileContinuation(context.Background(), f.source.Address, first.Status.RequestID, first.Status.Attempt)
			if err == nil {
				t.Fatal("target death not detected")
			}
			if scenario == "registration-crash" {
				record, _ := c.Threads.GetThread(f.source.Address)
				_, err = c.Threads.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error { next.Continuation.Target = nil; return nil })
				if err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "foreign" {
				ordinal++
			}
			second, err := c.RetryContinuation(context.Background(), f.source.Address, first.Status.RequestID)
			if scenario == "foreign" {
				if err == nil || f.launches != 1 {
					t.Fatalf("foreign generation duplicated target: %+v %v launches%d", second, err, f.launches)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			record, _ := c.Threads.GetThread(f.source.Address)
			if f.launches != 2 || second.Status.Attempt == first.Status.Attempt || record.Continuation.Source.LaunchOrdinal != 2 || record.Continuation.PreviousTargetGeneration == nil || record.Continuation.PreviousTargetGeneration.LaunchOrdinal != 3 {
				t.Fatalf("wrong retry %+v launches%d", record.Continuation, f.launches)
			}
		})
	}
}

func TestRecoveryRetirementBoundedConflictsAndCancellation(t *testing.T) {
	for _, conflicts := range []int{7, 8} {
		t.Run(fmt.Sprint(conflicts), func(t *testing.T) {
			env, source := switchEnvWithLiveThread(t)
			c := env.Couch
			env.Proc.Kill(source.Incarnations[0].PID)
			env.Artifacts.SetPairSession(source.Address, "pair-exact", false)
			plain := NewThreadStore(c.Namespace)
			reads := 0
			c.Threads.hooks.AfterGetThread = func(address ThreadAddress) error {
				reads++
				if reads > conflicts {
					return nil
				}
				r, err := plain.GetThread(address)
				if err != nil {
					return err
				}
				_, err = plain.updateExistingThread(address, r.Revision, func(next *ThreadRecord) error { next.Description = "concurrent metadata"; return nil })
				return err
			}
			_, _, err := c.reconcileRecoveryHelper(context.Background(), source.Address)
			if (err == nil) != (conflicts == 7) || reads != 8 {
				t.Fatalf("conflicts%d reads%d err%v", conflicts, reads, err)
			}
			actual, _ := plain.GetThread(source.Address)
			want := 0
			if conflicts == 8 {
				want = 1
			}
			if len(actual.Incarnations) != want {
				t.Fatalf("retired despite conflict budget: %+v", actual)
			}
		})
	}
	t.Run("cancel before retirement", func(t *testing.T) {
		env, source := switchEnvWithLiveThread(t)
		env.Proc.Kill(source.Incarnations[0].PID)
		env.Artifacts.SetPairSession(source.Address, "pair-exact", false)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		env.Artifacts.BeforePairSession = func(ThreadAddress) error { cancel(); return nil }
		_, _, err := env.Couch.reconcileRecoveryHelper(ctx, source.Address)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation=%v", err)
		}
		actual, _ := env.Couch.Threads.GetThread(source.Address)
		if len(actual.Incarnations) != 1 {
			t.Fatal("canceled observation retired source")
		}
	})
}

func TestRecoveryProcessIdentityRejectsUnknownAndAcceptsReusedPID(t *testing.T) {
	for _, unknown := range []bool{true, false} {
		t.Run(fmt.Sprint(unknown), func(t *testing.T) {
			env, source := switchEnvWithLiveThread(t)
			inc := source.Incarnations[0]
			if unknown {
				env.Proc.SetUnknown(inc.PID)
			} else {
				env.Proc.Set(inc.PID, "unrelated process start")
			}
			env.Artifacts.SetPairSession(source.Address, "pair-exact", false)
			_, _, err := env.Couch.reconcileRecoveryHelper(context.Background(), source.Address)
			if (err != nil) != unknown {
				t.Fatalf("unknown%v err%v", unknown, err)
			}
			if len(env.Proc.Signals) != 0 || len(env.Proc.GroupSignals) != 0 {
				t.Fatal("reconciliation signaled process")
			}
		})
	}
}

func TestRecoverCheckpointRechecksAbsenceAfterSourceRead(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	f.env.Proc.Kill(f.source.Incarnations[0].PID)
	f.env.Artifacts.SetPairSession(f.source.Address, "pair-exact", false)
	before, _ := c.Threads.GetThread(f.source.Address)
	c.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) {
		f.env.Artifacts.SetPairSession(f.source.Address, "pair-exact", true)
		return ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 2}, nil
	}
	_, err := c.RecoverThread(context.Background(), f.source.Address, "")
	if err == nil {
		t.Fatal("new source session did not invalidate recovery")
	}
	after, _ := c.Threads.GetThread(f.source.Address)
	if !reflect.DeepEqual(before, after) || f.launches != 0 {
		t.Fatal("recorded source absence after source appeared")
	}
}

func TestRecoveryInventoryUsesExistingEvidenceWithoutSourceReads(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	f.env.Proc.Kill(f.source.Incarnations[0].PID)
	record, _ := c.Threads.GetThread(f.source.Address)
	if err := os.Remove(record.Continuation.Checkpoint.SourcePath); err != nil {
		t.Fatal(err)
	}
	c.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) {
		t.Fatal("inventory read recovery source ledger")
		return ContinuationSource{}, nil
	}
	wanted := map[ThreadAddress]bool{record.Address: true}
	for i := 0; i < 7; i++ {
		other := cloneThreadRecord(record)
		other.Address.Tag = ThreadTag(fmt.Sprintf("couch-%016x", 1000+i))
		other.Continuation.ID = checkpoint.RequestID(other.Address.RepoScope, string(other.Address.Tag), other.Continuation.Source.LaunchOrdinal, other.Continuation.Checkpoint.Digest)
		created, err := c.Threads.CreateThread(other)
		if err != nil {
			t.Fatal(err)
		}
		wanted[created.Address] = true
	}
	before := f.env.Artifacts.DetachedQueries()
	for i := 0; i < 3; i++ {
		rows, err := c.ActionableThreadInventoryContext(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		found := map[ThreadAddress]bool{}
		for _, row := range rows {
			if wanted[row.Address] {
				found[row.Address] = true
				if row.Recovery == nil || row.Recovery.CheckpointDigest != record.Continuation.Checkpoint.Digest {
					t.Fatalf("missing embedded recovery projection: %+v", row)
				}
			}
		}
		if len(found) != len(wanted) {
			t.Fatalf("stale sources hidden: got%d want%d", len(found), len(wanted))
		}
	}
	if got := f.env.Artifacts.DetachedQueries(); got != before {
		t.Fatalf("recovery added session queries: before%d after%d", before, got)
	}
}

func TestContinuationRecoveryRefusesMultipleDeadIncarnations(t *testing.T) {
	f := newContinuationFixture(t)
	c := f.env.Couch
	first, err := c.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	f.env.Proc.Kill(first.Record.PID)
	f.env.Runner.SetExited(first.Handle.ID(), 0)
	f.env.Artifacts.SetPairSession(f.source.Address, "pair-exact", false)
	f.registered[first.Status.Attempt] = false
	_, err = c.ReconcileContinuation(context.Background(), f.source.Address, first.Status.RequestID, first.Status.Attempt)
	if err == nil {
		t.Fatal("target death ignored")
	}
	record, _ := c.Threads.GetThread(f.source.Address)
	before, err := c.Threads.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error {
		next.Incarnations = append(next.Incarnations, ThreadIncarnation{PID: 778, Identity: "unrelated", State: IncarnationLive})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.RetryContinuation(context.Background(), record.Address, first.Status.RequestID)
	if err == nil {
		t.Fatal("ambiguous helpers allowed retry")
	}
	after, _ := c.Threads.GetThread(record.Address)
	if !reflect.DeepEqual(before, after) || f.launches != 1 {
		t.Fatal("retry mutated ambiguous owners")
	}
}

// The optional context seam models a session service that honors cancellation;
// its other state and effects remain the production-shaped artifact fake.
type recoveryContextArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	observe func(context.Context, ThreadAddress) (PairSessionBinding, error)
}

func (a recoveryContextArtifacts) PairSessionContext(ctx context.Context, address ThreadAddress) (PairSessionBinding, error) {
	return a.observe(ctx, address)
}

func TestRecoveryObservationDeadlineCancelsBlockingObserver(t *testing.T) {
	for _, parentBudget := range []time.Duration{30 * time.Second, 30 * time.Millisecond} {
		t.Run(parentBudget.String(), func(t *testing.T) {
			env, source := switchEnvWithLiveThread(t)
			env.Proc.Kill(source.Incarnations[0].PID)
			parent, cancel := context.WithTimeout(context.Background(), parentBudget)
			defer cancel()
			parentDeadline, _ := parent.Deadline()
			observations := 0
			env.Couch.Artifacts = recoveryContextArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, observe: func(ctx context.Context, _ ThreadAddress) (PairSessionBinding, error) {
				observations++
				deadline, ok := ctx.Deadline()
				if !ok {
					t.Fatal("observer has no deadline")
				}
				if parentBudget > 5*time.Second {
					remaining := time.Until(deadline)
					if remaining > 5*time.Second || remaining < 4*time.Second {
						t.Fatalf("recovery observation budget=%v, want five seconds", remaining)
					}
					if !deadline.Before(parentDeadline) {
						t.Fatal("recovery inherited unbounded parent budget")
					}
				} else if !deadline.Equal(parentDeadline) {
					t.Fatal("recovery extended caller deadline")
				}
				<-ctx.Done()
				return PairSessionBinding{}, ctx.Err()
			}}
			before, _ := env.Couch.Threads.GetThread(source.Address)
			_, err := env.Couch.RecoverThread(parent, source.Address, "")
			if !errors.Is(err, context.DeadlineExceeded) || observations != 1 {
				t.Fatalf("blocked observer did not stop exactly once: %v calls%d", err, observations)
			}
			if parentBudget > 5*time.Second && parent.Err() != nil {
				t.Fatal("caller deadline, not recovery budget, stopped observation")
			}
			after, _ := env.Couch.Threads.GetThread(source.Address)
			if !reflect.DeepEqual(before, after) || len(env.Artifacts.quiesced) != 0 || len(env.Proc.Signals) != 0 {
				t.Fatal("timed-out observation performed recovery effects")
			}
		})
	}
}

func TestRecoveryCancellationBetweenRevisionAttemptsStopsReobservation(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	c := env.Couch
	env.Proc.Kill(source.Incarnations[0].PID)
	env.Artifacts.SetPairSession(source.Address, "pair-exact", false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	plain := NewThreadStore(c.Namespace)
	reads, observations := 0, 0
	env.Artifacts.BeforePairSession = func(ThreadAddress) error { observations++; return nil }
	var expected ThreadRecord
	c.Threads.hooks.AfterGetThread = func(address ThreadAddress) error {
		reads++
		if reads == 1 {
			original, err := plain.GetThread(address)
			if err != nil {
				return err
			}
			expected, err = plain.updateExistingThread(address, original.Revision, func(next *ThreadRecord) error {
				next.Description = "concurrent metadata forced revision conflict"
				return nil
			})
			return err
		}
		// The second read is reachable only after RetireIncarnation's first CAS
		// conflicted. Cancellation here must stop a second observation/CAS.
		cancel()
		return nil
	}
	_, _, err := c.reconcileRecoveryHelper(ctx, source.Address)
	if !errors.Is(err, context.Canceled) || reads != 2 || observations != 1 {
		t.Fatalf("canceled retry: %v reads%d observations%d", err, reads, observations)
	}
	actual, _ := plain.GetThread(source.Address)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatal("canceled retry changed the concurrent writer's record")
	}
}

func TestCanceledParkAwaitStillBlocksRecoveryAndArchiveUntilWorkerSettles(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	c := env.Couch
	reached, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer unblock()
	env.Lifecycle.onPublish = func(pairlifecycle.QuitRequest) { close(reached); <-release }
	returned := make(chan error, 1)
	go func() { _, err := c.PairLifecycle.Park(ctx, source.Address); returned <- err }()
	select {
	case <-reached:
	case <-time.After(5 * time.Second):
		t.Fatal("park did not enter blocked publication")
	}
	worker := c.PairLifecycle.worker
	worker.mu.Lock()
	future := worker.active[source.Address].future
	worker.mu.Unlock()
	if future == nil {
		t.Fatal("park worker not active after publication began")
	}
	defer func() {
		unblock()
		waitCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_, _ = future.Await(waitCtx)
	}()
	cancel()
	select {
	case err := <-returned:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("park Await=%v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("park Await ignored cancellation")
	}
	select {
	case <-future.done:
		t.Fatal("worker settled while publication was blocked")
	default:
	}
	before, err := c.Threads.GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if before.Park == nil {
		t.Fatal("canceled Await discarded active park transaction")
	}
	if _, err := c.RecoverThread(context.Background(), source.Address, ""); err == nil {
		t.Fatal("recovery entered while park worker continued")
	}
	if _, err := c.ArchiveThread(context.Background(), source.Address); err == nil {
		t.Fatal("archive entered while park worker continued")
	}
	after, err := c.Threads.GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) || len(env.Artifacts.quiesced) != 0 || len(env.Proc.Signals) != 0 {
		t.Fatal("later operation disturbed active park worker")
	}
	unblock()
	waitCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	_, _ = future.Await(waitCtx)
	select {
	case <-future.done:
	default:
		t.Fatal("released worker never settled")
	}
}
