package couchcore

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func TestRecoveryLaunchFailedResumeIsImmediatelyRetryableAndRecorded(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.Couch.RecoveryEvidence = nil
	f.Couch.LaunchFailures = NewLaunchFailureStore(f.Couch.Namespace)
	f.Artifacts.SetNativeBinding(f.record.Address, "codex", sessioninventory.BindingEstablished, "proven-native")
	success := f.Runner.AfterAcknowledge
	f.Runner.AfterAcknowledge = func(id string) error {
		f.Runner.SetExited(id, 23)
		return nil
	}
	resume := func() (any, error) {
		return DispatchOperation(OperationExecutors{
			Attempt: f.Couch.OperationAttemptForCall, LiveOwner: CouchLiveOwnerExecutor(f.Couch),
		}, OperationCall{Name: "resume", Context: t.Context(), Implicit: true,
			Args: map[string]string{"repo-scope": f.record.Address.RepoScope, "ref": string(f.record.Address.Tag)}})
	}
	_, err := resume()
	if err == nil {
		t.Fatal("early helper exit became a successful resume")
	}
	var uncertain *RecoveryLaunchUnconfirmed
	if errors.As(err, &uncertain) {
		t.Errorf("proved absent failed launch remained unconfirmed: %v", err)
	}
	failed, readErr := f.Couch.Threads.GetThread(f.record.Address)
	if readErr != nil || len(failed.Incarnations) != 0 || failed.RecoveryIntent == nil || failed.RecoveryIntent.NativeID != "proven-native" {
		t.Errorf("failed claim still occupies slot or lost choice: %+v %v", failed, readErr)
	}
	// Reopen the real diagnostic store: a Save attempt alone is not persistence.
	history, readErr := NewLaunchFailureStore(f.Couch.Namespace).Read()
	if readErr != nil || len(history) != 1 || history[0].Address != f.record.Address || history[0].LaunchNonce == "" || history[0].BeforeExit == nil || *history[0].BeforeExit != 23 || history[0].Outcome != "failed" {
		t.Errorf("scope-only outer dispatch lost exact durable launch diagnostics: %+v %v", history, readErr)
	}
	f.Runner.AfterAcknowledge = success
	_, err = resume()
	if err != nil {
		t.Fatalf("second resume without clock advance refused: %v", err)
	}
	current, err := f.Couch.Threads.GetThread(f.record.Address)
	if err != nil || len(current.Incarnations) != 1 || current.Incarnations[0].State != IncarnationLive {
		t.Fatalf("retry did not register same conversation: %+v %v", current, err)
	}
}

func TestRecoveryLaunchFailedTransitionRetainsExactChoice(t *testing.T) {
	for _, mode := range []threadrecord.RecoveryMode{threadrecord.RecoveryFresh, threadrecord.RecoveryProven, threadrecord.RecoveryInferred} {
		t.Run(string(mode), func(t *testing.T) {
			record := admittedStartRecord(t)
			intent := recoveryStartIntent(record)
			if mode != threadrecord.RecoveryInferred {
				intent = taggedRecoveryIntent(record, mode)
			}
			binding := storedBinding(3)
			claimed, err := AdvanceStartTransaction(record, StartEvent{Kind: StartClaimed, Nonce: "failed-attempt", Owner: SupervisorOwner{PID: 7, Identity: "live-owner"}, RecoveryIntent: intent, Transcript: binding})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(claimed.Transcript, binding) {
				t.Fatal("recovery claim lost its transcript binding")
			}
			claimed, err = AdvanceStartTransaction(claimed, StartEvent{Kind: StartHelperRecorded, Nonce: "failed-attempt", Helper: ProcessIdentity{PID: 8, Identity: "exited-helper"}})
			if err != nil {
				t.Fatal(err)
			}
			before := cloneThreadRecord(claimed)
			next, err := AdvanceStartTransaction(claimed, StartEvent{Kind: StartEventKind("failed-confirmed"), Nonce: "failed-attempt"})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, claimed) || len(next.Incarnations) != 0 || !reflect.DeepEqual(next.RecoveryIntent, intent) || !reflect.DeepEqual(next.VerifiedPark, claimed.VerifiedPark) || !reflect.DeepEqual(next.Transcript, binding) || next.Address != claimed.Address {
				t.Fatalf("failure transition changed source or lost recovery: %+v", next)
			}
			for _, event := range []StartEvent{{Kind: StartEventKind("failed-confirmed"), Nonce: "failed-attempt"}, {Kind: StartRegistered, Nonce: "failed-attempt"}, {Kind: StartHelperRecorded, Nonce: "failed-attempt", Helper: ProcessIdentity{PID: 9, Identity: "late-helper"}}} {
				if _, err := AdvanceStartTransaction(next, event); err == nil {
					t.Fatalf("late/duplicate event resurrected failed attempt: %+v", event)
				}
			}
			if _, err := AdvanceStartTransaction(claimed, StartEvent{Kind: StartEventKind("failed-confirmed"), Nonce: "other"}); err == nil {
				t.Fatal("failure event released another nonce")
			}
			registered, err := AdvanceStartTransaction(claimed, StartEvent{Kind: StartRegistered, Nonce: "failed-attempt"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := AdvanceStartTransaction(registered, StartEvent{Kind: StartEventKind("failed-confirmed"), Nonce: "failed-attempt"}); err == nil {
				t.Fatal("late failure cleared registered incarnation")
			}
		})
	}
}

func TestRecoveryLaunchFailedDiagnosticRefusalKeepsRetryAndVisibleWarning(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.Couch.RecoveryEvidence = nil
	store := NewFakeLaunchFailureStore()
	saveFailure := errors.New("fixture failure storage unavailable")
	store.SaveError = saveFailure
	f.Couch.LaunchFailures = store
	f.Artifacts.SetNativeBinding(f.record.Address, "codex", sessioninventory.BindingEstablished, "proven-native")
	f.Runner.AfterAcknowledge = func(id string) error { f.Runner.SetExited(id, 23); return nil }
	_, err := DispatchOperation(OperationExecutors{Attempt: f.Couch.OperationAttemptForCall, LiveOwner: CouchLiveOwnerExecutor(f.Couch)},
		OperationCall{Name: "resume", Context: t.Context(), Implicit: true, Args: map[string]string{"repo-scope": f.record.Address.RepoScope, "ref": string(f.record.Address.Tag)}})
	if !errors.Is(err, saveFailure) {
		t.Fatalf("storage refusal lost its cause: %v", err)
	}
	if line := OperatorError(err); !strings.Contains(line, "diagnostics could not be saved") || !strings.Contains(line, "exit 23") {
		t.Errorf("operator line hid failed publication or original exit: %q", line)
	}
	current, readErr := f.Couch.Threads.GetThread(f.record.Address)
	if readErr != nil || len(current.Incarnations) != 0 || current.RecoveryIntent == nil || len(store.Writes) != 1 {
		t.Fatalf("diagnostic refusal changed failure disposition or duplicated publication: %+v %v", current, readErr)
	}
}

func TestRecoveryLaunchExitedChildWithLiveGroupRetainsClaim(t *testing.T) {
	env, in := recoveryTrackedFixture(t)
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Runner.SetChildExited(id)
		return nil
	}
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) {
		return false, errors.New("no registration")
	}
	_, h, err := env.Couch.launchTrackedThread(in)
	var uncertain *RecoveryLaunchUnconfirmed
	if !errors.As(err, &uncertain) || h == nil || !h.Alive() {
		t.Fatalf("direct child exit discarded live owned group: %v", err)
	}
	if line := OperatorError(err); !strings.Contains(line, "Launch unconfirmed") {
		t.Errorf("operator line hid uncertain launch outcome: %q", line)
	}
	record, readErr := env.Couch.Threads.GetThread(in.Thread.Address)
	if readErr != nil || len(record.Incarnations) != 1 || record.Incarnations[0].Start == nil || record.Incarnations[0].Start.Nonce != in.Nonce {
		t.Fatalf("owned group lost exact claim: %+v %v", record, readErr)
	}
	if len(env.Runner.Signals(h.ID())) != 0 || len(env.Artifacts.TriggeredQuits()) != 0 {
		t.Fatal("uncertain recovery performed cleanup")
	}
}

func failedRecoveryClaimFixture(t *testing.T) (*testEnv, ThreadRecord, Handle) {
	t.Helper()
	env, in := recoveryTrackedFixture(t)
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) {
		return false, errors.New("no registration")
	}
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Runner.SetExited(id, 23)
		env.Artifacts.BeforePairSession = func(ThreadAddress) error { return errors.New("terminal unavailable") }
		return nil
	}
	_, h, err := env.Couch.launchTrackedThread(in)
	var uncertain *RecoveryLaunchUnconfirmed
	if !errors.As(err, &uncertain) || h == nil {
		t.Fatalf("fixture did not preserve uncertain claim: %v", err)
	}
	env.Artifacts.BeforePairSession = nil
	record, err := env.Couch.Threads.GetThread(in.Thread.Address)
	if err != nil {
		t.Fatal(err)
	}
	return env, record, h
}

func TestRecoveryLaunchFailedDispositionKeepsPresentOrUnknownTerminal(t *testing.T) {
	for _, mode := range []string{"present", "unknown", "canceled", "revision changed", "different helper"} {
		t.Run(mode, func(t *testing.T) {
			env, record, h := failedRecoveryClaimFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch mode {
			case "present":
				env.Artifacts.SetPairSession(record.Address, recordSessionName(record), true)
			case "unknown":
				env.Artifacts.BeforePairSession = func(ThreadAddress) error { return errors.New("terminal observation failed") }
			case "canceled":
				cancel()
			case "revision changed":
				env.Artifacts.BeforePairSession = func(address ThreadAddress) error {
					_, err := env.Couch.Threads.updateExistingThread(address, record.Revision, func(next *ThreadRecord) error { next.Name = "concurrent activity"; return nil })
					return err
				}
			case "different helper":
				other, err := env.Runner.Start("/repo", []string{"pair"}, nil)
				if err != nil {
					t.Fatal(err)
				}
				env.Runner.SetExited(other.ID(), 23)
				h = other
			}
			released, err := env.Couch.finalizeFailedRecoveryLaunch(ctx, record, record.Incarnations[0].Start.Nonce, h)
			if released || mode != "present" && err == nil {
				t.Fatalf("unproved or changed attempt released: %v %v", released, err)
			}
			current, err := env.Couch.Threads.GetThread(record.Address)
			if err != nil || len(current.Incarnations) != 1 || !reflect.DeepEqual(current.Incarnations, record.Incarnations) {
				t.Fatalf("uncertain disposition discarded ownership: %+v %v", current, err)
			}
			if len(env.Runner.Signals(h.ID())) != 0 || len(env.Artifacts.TriggeredQuits()) != 0 {
				t.Fatal("failure disposition performed cleanup")
			}
		})
	}
}

func TestRecoveryLaunchFailedDispositionCountsOneObservationAndCommit(t *testing.T) {
	for _, n := range []int{8, 16} {
		observations := 0
		for i := 0; i < n; i++ {
			env, record, h := failedRecoveryClaimFixture(t)
			env.Artifacts.BeforePairSession = func(ThreadAddress) error { observations++; return nil }
			released, err := env.Couch.finalizeFailedRecoveryLaunch(t.Context(), record, record.Incarnations[0].Start.Nonce, h)
			if !released || err != nil {
				t.Fatalf("failure disposition: %v %v", released, err)
			}
			current, err := env.Couch.Threads.GetThread(record.Address)
			if err != nil || current.Revision != record.Revision+1 || len(current.Incarnations) != 0 {
				t.Fatalf("disposition did not use one commit: %+v %v", current, err)
			}
		}
		if observations != n {
			t.Fatalf("%d failed launches used %d terminal observations", n, observations)
		}
	}
}

func TestRecoveryLaunchFailedDispositionHonorsHeldStoreDeadline(t *testing.T) {
	env, record, h := failedRecoveryClaimFixture(t)
	release := holdStoreLock(t, env.Couch.Threads)
	releasedLock := false
	defer func() {
		if !releasedLock {
			release()
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		released, err := env.Couch.finalizeFailedRecoveryLaunch(ctx, record, record.Incarnations[0].Start.Nonce, h)
		if released {
			err = errors.New("released claim behind held store lock")
		}
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, syscall.EWOULDBLOCK) {
			t.Fatalf("held store did not produce a bounded refusal: %v", err)
		}
	case <-time.After(sessionCleanupReturnBudget):
		release()
		releasedLock = true
		<-done
		t.Fatal("disposition waited for held metadata lock after its deadline")
	}
	release()
	releasedLock = true
	current, err := env.Couch.Threads.GetThread(record.Address)
	if err != nil || !reflect.DeepEqual(current, record) {
		t.Fatalf("timed-out disposition changed occupied claim: %+v %v", current, err)
	}
}

type stalledFailedLaunchObservation struct {
	*FakeThreadArtifactCollisionChecker
}

func (a stalledFailedLaunchObservation) NamedPairSessionContext(ctx context.Context, _ ThreadAddress, _ string) (PairSessionBinding, error) {
	<-ctx.Done()
	return PairSessionBinding{}, ctx.Err()
}

func TestRecoveryLaunchFailedDispositionBoundsTerminalObservation(t *testing.T) {
	env, record, h := failedRecoveryClaimFixture(t)
	env.Couch.Artifacts = stalledFailedLaunchObservation{env.Artifacts}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	released, err := env.Couch.finalizeFailedRecoveryLaunch(ctx, record, record.Incarnations[0].Start.Nonce, h)
	if released || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stalled observation lost deadline or released claim: %v %v", released, err)
	}
	current, readErr := env.Couch.Threads.GetThread(record.Address)
	if readErr != nil || !reflect.DeepEqual(current, record) {
		t.Fatalf("failed observation changed retained claim: %+v %v", current, readErr)
	}
}
