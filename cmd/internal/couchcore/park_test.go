package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

type fakeControllerLifecycle struct {
	model       *pairlifecycletest.Fake
	store       *ThreadStore
	trace       []string
	completion  *pairlifecycle.QuitCompletion
	completions map[uint64]pairlifecycle.QuitCompletion
	publishErr  error
	observeErr  error
	cleanupErr  error
	onPublish   func(pairlifecycle.QuitRequest)
	onObserve   func(pairlifecycle.QuitRequest)
	onCleanup   func(pairlifecycle.QuitRequest)
	lastRequest pairlifecycle.QuitRequest
}

type constructorLifecycleArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	lifecycle LifecycleIO
	dataDir   string
}

func (a constructorLifecycleArtifacts) PairLifecycleIO() LifecycleIO { return a.lifecycle }
func (a constructorLifecycleArtifacts) PairLifecycleDataDir() string { return a.dataDir }

func (f *fakeControllerLifecycle) PublishRequest(_ artifactpath.LifecyclePaths, request pairlifecycle.QuitRequest) error {
	f.trace = append(f.trace, "publish-request")
	f.lastRequest = request
	if f.onPublish != nil {
		f.onPublish(request)
	}
	if f.publishErr != nil {
		return f.publishErr
	}
	return f.model.CommitRequest(request)
}

func (f *fakeControllerLifecycle) ObserveCompletion(_ artifactpath.LifecyclePaths, request pairlifecycle.QuitRequest) (pairlifecycle.QuitCompletion, bool, error) {
	f.trace = append(f.trace, "observe-completion")
	if f.onObserve != nil {
		f.onObserve(request)
	}
	if f.observeErr != nil {
		return pairlifecycle.QuitCompletion{}, false, f.observeErr
	}
	if completion, ok := f.completions[request.Attempt]; ok {
		return completion, true, nil
	}
	if f.completion == nil {
		return pairlifecycle.QuitCompletion{}, false, nil
	}
	return *f.completion, true, nil
}

func (f *fakeControllerLifecycle) CleanupAttempt(_ artifactpath.LifecyclePaths, request pairlifecycle.QuitRequest) error {
	f.trace = append(f.trace, "cleanup-attempt")
	if f.onCleanup != nil {
		f.onCleanup(request)
	}
	return f.cleanupErr
}

func createControllerThread(t *testing.T) (*ThreadStore, CouchNamespace, ThreadRecord) {
	t.Helper()
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record.Reservation = false
	profile := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
	record.Incarnations = []ThreadIncarnation{{
		PID: 42, Identity: "pair-helper", State: IncarnationLive, LaunchProfile: &profile,
	}}
	record.LatestLaunchProfile = &profile
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	return store, ns, created
}

func successCompletion(request pairlifecycle.QuitRequest, at time.Time) pairlifecycle.QuitCompletion {
	return pairlifecycle.QuitCompletion{
		SchemaVersion: request.SchemaVersion, Identity: request.Identity, Attempt: request.Attempt,
		Session: request.Session, Mode: request.Mode, CompletionKey: request.CompletionKey,
		Outcome: pairlifecycle.CompletionSuccess, CompletedAt: at,
	}
}

func cleanupFailureCompletion(request pairlifecycle.QuitRequest, at time.Time) pairlifecycle.QuitCompletion {
	completion := successCompletion(request, at)
	completion.Outcome = pairlifecycle.CompletionFailure
	completion.FailureCode = pairlifecycle.FailureCleanupFailed
	return completion
}

func TestParkCoordinatorOrdering(t *testing.T) {
	store, _, thread := createControllerThread(t)
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	model := pairlifecycletest.New(now)
	model.SetSession("pair-exact", true)
	lifecycle := &fakeControllerLifecycle{model: model, store: store}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(thread.Address, "pair-exact", true)
	artifacts.TriggerQuitHook = func(session string, intent launcher.QuitIntent) error {
		lifecycle.trace = append(lifecycle.trace, "trigger")
		if session != "pair-exact" || intent.Kind != launcher.QuitIntentCouch {
			t.Fatalf("trigger = %q %+v", session, intent)
		}
		if err := model.DeliverTrigger(lifecycle.lastRequest); err != nil {
			return err
		}
		completion := successCompletion(lifecycle.lastRequest, now)
		lifecycle.completion = &completion
		return model.CommitCompletion(lifecycle.lastRequest, pairlifecycle.CleanupResult{
			Outcome: pairlifecycle.CompletionSuccess, CompletedAt: now,
		})
	}
	lifecycle.onPublish = func(request pairlifecycle.QuitRequest) {
		persisted, err := store.GetThread(thread.Address)
		if err != nil || persisted.Park == nil || persisted.Park.Phase != ParkRequested || persisted.Park.Attempts[0].Number != request.Attempt {
			t.Fatalf("publication preceded requested CAS: %+v, %v", persisted, err)
		}
	}
	lifecycle.onCleanup = func(request pairlifecycle.QuitRequest) {
		persisted, err := store.GetThread(thread.Address)
		if err != nil || persisted.VerifiedPark == nil || persisted.Park != nil {
			t.Fatalf("completion cleanup preceded final CAS: %+v, %v", persisted, err)
		}
		if model.CompletionState(request) != pairlifecycletest.Committed {
			t.Fatal("completion authority disappeared before final CAS")
		}
	}

	controller := PairLifecycleController{
		Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
		Proc:  NewFakeProcOps(),
		Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "park-0123456789abcdef", nil },
	}
	result, err := controller.Park(context.Background(), thread.Address)
	if err != nil {
		t.Fatalf("Park: %v", err)
	}
	if result.Thread.VerifiedPark == nil || len(result.Thread.Incarnations) != 0 {
		t.Fatalf("result = %+v", result)
	}
	if !reflect.DeepEqual(lifecycle.trace, []string{"publish-request", "observe-completion", "trigger", "observe-completion", "cleanup-attempt"}) {
		t.Fatalf("trace = %v", lifecycle.trace)
	}
	if got := artifacts.Quiesces(); len(got) != 0 {
		t.Fatalf("coordinator called Artifacts.Quiesce: %v", got)
	}
	if got := artifacts.TriggeredQuits(); len(got) != 1 {
		t.Fatalf("typed triggers = %v", got)
	}
}

func TestParkWaitsForCompletionAndExactChildDeath(t *testing.T) {
	store, _, thread := createControllerThread(t)
	now := time.Date(2026, 8, 30, 12, 30, 0, 0, time.UTC)
	lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(now)}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(thread.Address, "pair-exact", true)
	proc := NewFakeProcOps()
	proc.Set(42, "pair-helper")
	waits := 0
	controller := PairLifecycleController{
		Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
		Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "park-delayed-proof", nil },
		Proc: proc, CompletionTimeout: time.Second, PollInterval: time.Millisecond,
		Wait: func(context.Context, time.Duration) error {
			waits++
			switch waits {
			case 1:
				completion := successCompletion(lifecycle.lastRequest, now)
				lifecycle.completion = &completion
			case 2:
				proc.Kill(42)
			}
			return nil
		},
	}

	result, err := controller.Park(context.Background(), thread.Address)
	if err != nil {
		t.Fatalf("Park: %v", err)
	}
	if waits != 2 {
		t.Fatalf("waits = %d, want completion wait then child-death wait", waits)
	}
	if result.Thread.VerifiedPark == nil || result.Thread.Park != nil || len(result.Thread.Incarnations) != 0 {
		t.Fatalf("result = %+v", result.Thread)
	}
}

func TestParkRetainsOccupiedTransactionWhenCompletionPrecedesChildDeathDeadline(t *testing.T) {
	store, _, thread := createControllerThread(t)
	now := time.Date(2026, 8, 30, 12, 45, 0, 0, time.UTC)
	lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(now)}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(thread.Address, "pair-exact", true)
	artifacts.TriggerQuitHook = func(_ string, _ launcher.QuitIntent) error {
		completion := successCompletion(lifecycle.lastRequest, now)
		lifecycle.completion = &completion
		return nil
	}
	proc := NewFakeProcOps()
	proc.Set(42, "pair-helper")
	controller := PairLifecycleController{
		Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
		Proc: proc, Clock: FixedClock{T: now},
		Nonce:             func() (string, error) { return "park-child-still-live", nil },
		CompletionTimeout: time.Second, PollInterval: time.Millisecond,
		Wait: func(context.Context, time.Duration) error { return context.DeadlineExceeded },
	}

	result, err := controller.Park(context.Background(), thread.Address)
	if err == nil {
		t.Fatal("Park returned success while the exact child remained live")
	}
	if result.Thread.Park == nil || result.Thread.VerifiedPark != nil || !result.Thread.Park.Attempts[0].TimedOut || len(result.Thread.Incarnations) != 1 {
		t.Fatalf("result = %+v", result.Thread)
	}
	for _, event := range lifecycle.trace {
		if event == "cleanup-attempt" {
			t.Fatalf("completion evidence was deleted before child death: %v", lifecycle.trace)
		}
	}
}

// Leave DETACHES rather than parks: quitting couch must not kill every agent,
// including any mid-turn. The partial-failure property is unchanged -- a failure
// mid-sweep preserves what already succeeded and leaves the rest occupied.
func TestLeaveDetachesLiveThreadsSequentiallyAndRetainsPartialFailure(t *testing.T) {
	store, ns, first := createControllerThread(t)
	second := validThreadRecord(t)
	second.Address.Tag = "couch-fedcba9876543210"
	second.StartingPath, second.WorkingPath = ns.Dir(), ns.Dir()
	second.Reservation = false
	profile := LaunchProfile{Agent: "codex"}
	second.Incarnations = []ThreadIncarnation{{PID: 43, Identity: "pair-second", State: IncarnationLive, LaunchProfile: &profile}}
	second.LatestLaunchProfile = &profile
	second, err := store.CreateThread(second)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(first.Address, "pair-first", true)
	artifacts.SetPairSession(second.Address, "pair-second", true)

	proc := NewFakeProcOps()
	firstIncarnation := first.Incarnations[0]
	proc.Set(firstIncarnation.PID, firstIncarnation.Identity)
	proc.DiesOn = map[int]os.Signal{firstIncarnation.PID: syscall.SIGTERM}
	// The second client ignores SIGTERM, so its detach fails.
	proc.Set(43, "pair-second")

	couch := &Couch{Threads: store, Proc: proc, Artifacts: artifacts, Clock: FixedClock{T: time.Unix(100, 0).UTC()}, sleep: func(time.Duration) {}}

	result, err := couch.Leave(context.Background(), LeaveDetach)
	if err == nil {
		t.Fatalf("Leave = %+v, want the second detach to fail", result)
	}
	if len(result.Detached) != 1 || result.Detached[0] != first.Address {
		t.Fatalf("Leave = %+v, want the first thread detached", result)
	}
	detached, _ := store.GetThread(first.Address)
	occupied, _ := store.GetThread(second.Address)
	if len(detached.Incarnations) != 0 || detached.VerifiedPark != nil {
		t.Fatalf("first = %+v, want a retired incarnation and no verified park", detached)
	}
	if len(occupied.Incarnations) != 1 {
		t.Fatalf("second = %+v, want it left occupied", occupied)
	}
	// Nothing was torn down: leaving couch keeps every agent alive.
	if got := artifacts.Quiesces(); len(got) != 0 {
		t.Fatalf("leave quiesced sessions: %+v", got)
	}
	if got := artifacts.TriggeredQuits(); len(got) != 0 {
		t.Fatalf("leave triggered Pair quits: %+v", got)
	}
}

// Recorded-live is a claim to recheck, not proof that a client can detach.
func TestLeaveRechecksRecordedLiveBeforeDetaching(t *testing.T) {
	for _, tc := range []struct {
		name    string
		process string
		session bool
		retired bool
	}{
		{"dead-absent", "dead", false, true},
		{"dead-surviving-session", "dead", true, true},
		{"reused-pid-absent", "reused", false, true},
		{"live-absent", "live", false, false},
		{"unknown-absent", "unknown", false, false},
		{"identity-unreadable-absent", "identity-error", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, ns, first := createControllerThread(t)
			second := validThreadRecord(t)
			second.Address.Tag = "couch-fedcba9876543210"
			second.StartingPath, second.WorkingPath = ns.Dir(), ns.Dir()
			second.Reservation = false
			second.Incarnations = []ThreadIncarnation{{PID: 43, Identity: "pair-second", State: IncarnationLive}}
			second, err := store.CreateThread(second)
			if err != nil {
				t.Fatal(err)
			}
			artifacts := NewFakeThreadArtifactCollisionChecker()
			artifacts.SetPairSession(first.Address, "pair-first", tc.session)
			artifacts.SetPairSession(second.Address, "pair-second", true)
			proc := NewFakeProcOps()
			switch tc.process {
			case "live":
				proc.Set(42, "pair-helper")
			case "reused":
				proc.Set(42, "unrelated-process")
			case "unknown":
				proc.SetUnknown(42)
			case "identity-error":
				proc.Set(42, "pair-helper")
				proc.IdentityErr[42] = true
			}
			proc.Set(43, "pair-second")
			proc.DiesOn[43] = syscall.SIGTERM
			couch := &Couch{Threads: store, Proc: proc, Artifacts: artifacts, Clock: FixedClock{T: time.Unix(100, 0).UTC()}, sleep: func(time.Duration) {}}
			result, err := couch.Leave(context.Background(), LeaveDetach)
			if err != nil {
				t.Fatalf("Leave = %+v, %v; stale first thread must not block healthy second", result, err)
			}
			if !reflect.DeepEqual(result.Detached, []ThreadAddress{second.Address}) || len(result.Parked) != 0 {
				t.Fatalf("Leave = %+v; only healthy second thread was detached", result)
			}
			wantSkipped := []ThreadAddress(nil)
			if !tc.retired {
				wantSkipped = []ThreadAddress{first.Address}
			}
			if !reflect.DeepEqual(result.Skipped, wantSkipped) {
				t.Fatalf("Skipped = %+v, want %+v", result.Skipped, wantSkipped)
			}
			after, err := store.GetThread(first.Address)
			if err != nil {
				t.Fatal(err)
			}
			if tc.retired {
				if len(after.Incarnations) != 0 || after.VerifiedPark != nil || after.LatestLaunchProfile == nil {
					t.Fatalf("retired thread lost resume history or gained park authority: %+v", after)
				}
			} else if !reflect.DeepEqual(after, first) {
				t.Fatalf("unproved thread changed: before %+v, after %+v", first, after)
			}
			if len(proc.Signals[42]) != 0 || len(proc.GroupSignals[42]) != 0 {
				t.Fatalf("signalled first thread: %+v / %+v", proc.Signals, proc.GroupSignals)
			}
			if !reflect.DeepEqual(proc.GroupSignals[43], []os.Signal{syscall.SIGTERM}) {
				t.Fatalf("healthy thread signals = %+v", proc.GroupSignals)
			}
			detached, err := store.GetThread(second.Address)
			if err != nil || len(detached.Incarnations) != 0 || detached.VerifiedPark != nil {
				t.Fatalf("healthy thread was not detached: %+v, %v", detached, err)
			}
			binding, err := artifacts.PairSession(first.Address)
			if err != nil || binding.Present != tc.session {
				t.Fatalf("first session changed: %+v, %v", binding, err)
			}
			if len(artifacts.Quiesces()) != 0 || len(artifacts.TriggeredQuits()) != 0 {
				t.Fatal("leave tore down an agent session")
			}
		})
	}
}

func TestLeaveObservationErrorPreservesRecordedLive(t *testing.T) {
	store, _, thread := createControllerThread(t)
	proc := NewFakeProcOps()
	proc.Set(42, "pair-helper")
	artifacts := NewFakeThreadArtifactCollisionChecker()
	observationErr := errors.New("session inventory unreadable")
	artifacts.BeforePairSession = func(ThreadAddress) error { return observationErr }
	couch := &Couch{Threads: store, Proc: proc, Artifacts: artifacts, Clock: FixedClock{T: time.Unix(100, 0).UTC()}}
	result, err := couch.Leave(context.Background(), LeaveDetach)
	if !errors.Is(err, observationErr) {
		t.Fatalf("Leave = %+v, %v; want observation failure", result, err)
	}
	after, getErr := store.GetThread(thread.Address)
	if getErr != nil || !reflect.DeepEqual(after, thread) || len(proc.Signals) != 0 || len(result.Detached) != 0 || len(result.Skipped) != 0 {
		t.Fatalf("failed observation mutated thread: %+v, %v; signals %+v; result %+v", after, getErr, proc.Signals, result)
	}
}

func TestLeaveCanceledDoesNotRetireDeadIncarnation(t *testing.T) {
	store, _, thread := createControllerThread(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	couch := &Couch{Threads: store, Proc: NewFakeProcOps(), Artifacts: NewFakeThreadArtifactCollisionChecker(), Clock: FixedClock{T: time.Unix(100, 0).UTC()}}
	if _, err := couch.Leave(ctx, LeaveDetach); !errors.Is(err, context.Canceled) {
		t.Fatalf("Leave error = %v, want cancellation", err)
	}
	after, err := store.GetThread(thread.Address)
	if err != nil || !reflect.DeepEqual(after, thread) {
		t.Fatalf("canceled leave mutated thread: %+v, %v", after, err)
	}
}

// An unknown incarnation is skipped, not parked. Parking is the destructive
// option and Couch cannot vouch for what that thread is doing.
func TestLeaveSkipsUnknownIncarnationsRatherThanParkingThem(t *testing.T) {
	ns := testCouchNamespace(t)
	store := NewThreadStore(ns)
	unknown := validThreadRecord(t)
	unknown.Address.Tag = "couch-aaaabbbbccccdddd"
	unknown.StartingPath, unknown.WorkingPath = ns.Dir(), ns.Dir()
	unknown.Reservation = false
	unknown.Incarnations = []ThreadIncarnation{{PID: 77, Identity: "gone", State: IncarnationUnknown}}
	unknown, err := store.CreateThread(unknown)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	proc := NewFakeProcOps()
	couch := &Couch{Threads: store, Proc: proc, Artifacts: artifacts, Clock: FixedClock{T: time.Unix(100, 0).UTC()}, sleep: func(time.Duration) {}}

	result, _ := couch.Leave(context.Background(), LeaveDetach)
	found := false
	for _, address := range result.Skipped {
		if address == unknown.Address {
			found = true
		}
	}
	if !found {
		t.Fatalf("Leave = %+v, want the unknown-incarnation thread reported as skipped", result)
	}
	after, _ := store.GetThread(unknown.Address)
	if len(after.Incarnations) != 1 || after.VerifiedPark != nil {
		t.Fatalf("unknown thread = %+v, want it untouched", after)
	}
}

func TestParkCoordinatorTransitionMatrix(t *testing.T) {
	t.Run("request-publish-failed", func(t *testing.T) {
		store, _, thread := createControllerThread(t)
		now := time.Unix(100, 0).UTC()
		lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(now), publishErr: errors.New("disk full")}
		artifacts := NewFakeThreadArtifactCollisionChecker()
		artifacts.SetPairSession(thread.Address, "pair-exact", true)
		controller := PairLifecycleController{
			Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
			Proc:  NewFakeProcOps(),
			Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "park-1111111111111111", nil },
		}
		if _, err := controller.Park(context.Background(), thread.Address); err == nil {
			t.Fatal("publication failure returned success")
		}
		kept, _ := store.GetThread(thread.Address)
		if kept.Park == nil || kept.Park.Phase != ParkRequested || len(kept.Incarnations) != 1 || len(artifacts.TriggeredQuits()) != 0 {
			t.Fatalf("publication failure state = %+v", kept)
		}
	})

	t.Run("session-absent-completion-missing", func(t *testing.T) {
		store, _, thread := createControllerThread(t)
		now := time.Unix(100, 0).UTC()
		model := pairlifecycletest.New(now)
		lifecycle := &fakeControllerLifecycle{model: model}
		artifacts := NewFakeThreadArtifactCollisionChecker()
		artifacts.SetPairSession(thread.Address, "pair-exact", true)
		artifacts.TriggerQuitHook = func(_ string, _ launcher.QuitIntent) error {
			artifacts.SetPairSession(thread.Address, "pair-exact", false)
			return nil
		}
		controller := PairLifecycleController{
			Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
			Proc:  NewFakeProcOps(),
			Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "park-2222222222222222", nil },
		}
		if _, err := controller.Park(context.Background(), thread.Address); err == nil {
			t.Fatal("missing completion returned success")
		}
		kept, _ := store.GetThread(thread.Address)
		if kept.Park == nil || kept.Park.Phase != ParkAwaitingCompletion || !kept.Park.Attempts[0].TimedOut ||
			kept.Park.Attempts[0].Failure == nil || kept.Park.Attempts[0].Failure.Code != pairlifecycle.FailureTimeout || len(kept.Incarnations) != 1 {
			t.Fatalf("missing completion state = %+v", kept)
		}
	})

	t.Run("stale-completion", func(t *testing.T) {
		store, _, thread := createControllerThread(t)
		now := time.Unix(250, 0).UTC()
		lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(now)}
		artifacts := NewFakeThreadArtifactCollisionChecker()
		artifacts.SetPairSession(thread.Address, "pair-exact", true)
		artifacts.TriggerQuitHook = func(_ string, _ launcher.QuitIntent) error {
			completion := successCompletion(lifecycle.lastRequest, now)
			completion.Session = "pair-wrong"
			lifecycle.completion = &completion
			return nil
		}
		controller := PairLifecycleController{
			Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
			Proc:  NewFakeProcOps(),
			Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "park-stale-completion", nil },
		}
		if _, err := controller.Park(context.Background(), thread.Address); err == nil {
			t.Fatal("stale completion returned success")
		}
		kept, err := store.GetThread(thread.Address)
		if err != nil {
			t.Fatal(err)
		}
		if kept.Park == nil || kept.Park.Phase != ParkAwaitingCompletion || kept.Park.Attempts[0].Failure == nil || kept.Park.Attempts[0].Failure.Code != pairlifecycle.FailureStaleCompletion {
			t.Fatalf("stale completion state = %+v", kept)
		}
	})

	t.Run("lock-sync-indeterminate", func(t *testing.T) {
		store, _, thread := createControllerThread(t)
		now := time.Unix(260, 0).UTC()
		lifecycle := &fakeControllerLifecycle{
			model: pairlifecycletest.New(now),
			observeErr: &pairlifecycle.PublicationError{
				Outcome: pairlifecycle.Indeterminate,
				Err:     errors.New("directory sync unavailable"),
			},
		}
		artifacts := NewFakeThreadArtifactCollisionChecker()
		artifacts.SetPairSession(thread.Address, "pair-exact", true)
		controller := PairLifecycleController{
			Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
			Proc:  NewFakeProcOps(),
			Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "park-indeterminate", nil },
		}
		if _, err := controller.Park(context.Background(), thread.Address); err == nil {
			t.Fatal("indeterminate lifecycle authority returned success")
		}
		kept, err := store.GetThread(thread.Address)
		if err != nil {
			t.Fatal(err)
		}
		if kept.Park == nil || kept.Park.Phase != ParkUnknown || kept.Park.Attempts[0].Failure == nil || kept.Park.Attempts[0].Failure.Code != pairlifecycle.FailureCompletionMissing || len(artifacts.TriggeredQuits()) != 0 {
			t.Fatalf("indeterminate authority state = %+v triggers=%v", kept, artifacts.TriggeredQuits())
		}
	})

	for _, test := range []struct {
		name           string
		sessionPresent bool
		retry          bool
	}{
		{name: "cleanup-failed-retry-eligible", sessionPresent: true, retry: true},
		{name: "cleanup-failed-recover-required", sessionPresent: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, _, thread := createControllerThread(t)
			now := time.Unix(275, 0).UTC()
			lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(now)}
			artifacts := NewFakeThreadArtifactCollisionChecker()
			artifacts.SetPairSession(thread.Address, "pair-exact", true)
			artifacts.TriggerQuitHook = func(_ string, _ launcher.QuitIntent) error {
				completion := cleanupFailureCompletion(lifecycle.lastRequest, now)
				lifecycle.completion = &completion
				return nil
			}
			controller := PairLifecycleController{
				Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
				Proc:  NewFakeProcOps(),
				Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "park-cleanup-failed", nil },
			}
			if _, err := controller.Park(context.Background(), thread.Address); err == nil {
				t.Fatal("cleanup failure returned success")
			}
			failed, err := store.GetThread(thread.Address)
			if err != nil || failed.Park == nil || failed.Park.Phase != ParkUnknown || !failed.Park.Attempts[0].Closed {
				t.Fatalf("cleanup failure state = %+v, %v", failed, err)
			}
			lifecycle.completion = nil
			artifacts.TriggerQuitHook = nil
			artifacts.SetPairSession(thread.Address, "pair-exact", test.sessionPresent)
			if test.retry {
				result, err := controller.Retry(context.Background(), thread.Address)
				if err == nil || result.Thread.Park == nil || len(result.Thread.Park.Attempts) != 2 || result.Thread.Park.Phase != ParkAwaitingCompletion || !result.Thread.Park.Attempts[1].TimedOut {
					t.Fatalf("Retry = %+v, %v", result, err)
				}
				return
			}
			if _, err := controller.Retry(context.Background(), thread.Address); err == nil {
				t.Fatal("Retry accepted an absent exact session")
			}
			result, err := controller.Recover(context.Background(), thread.Address)
			if err == nil || result.Thread.Park == nil || len(result.Thread.Park.Attempts) != 1 {
				t.Fatalf("Recover = %+v, %v", result, err)
			}
		})
	}

	for _, test := range []struct {
		name        string
		replacement bool
		wantCode    pairlifecycle.FailureCode
	}{
		{name: "revision-conflict", wantCode: pairlifecycle.FailureRevisionConflict},
		{name: "replacement-incarnation", replacement: true, wantCode: pairlifecycle.FailureReplacementIncarnation},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, _, thread := createControllerThread(t)
			now := time.Unix(290, 0).UTC()
			lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(now)}
			artifacts := NewFakeThreadArtifactCollisionChecker()
			artifacts.SetPairSession(thread.Address, "pair-exact", true)
			artifacts.TriggerQuitHook = func(_ string, _ launcher.QuitIntent) error {
				completion := successCompletion(lifecycle.lastRequest, now)
				lifecycle.completion = &completion
				return nil
			}
			mutated := false
			lifecycle.onObserve = func(request pairlifecycle.QuitRequest) {
				if lifecycle.completion == nil || mutated || request.Attempt != lifecycle.lastRequest.Attempt {
					return
				}
				mutated = true
				current, err := store.GetThread(thread.Address)
				if err != nil {
					t.Fatal(err)
				}
				if test.replacement {
					abandoned, abandonErr := store.AbandonPark(thread.Address, current.Revision, current.Park.Identity)
					if abandonErr != nil {
						t.Fatal(abandonErr)
					}
					replacement := ParkIdentity{Nonce: "park-replacement", Address: thread.Address, PID: 42, ProcessIdentity: "pair-helper"}
					if _, beginErr := store.BeginPark(thread.Address, abandoned.Revision, replacement); beginErr != nil {
						t.Fatal(beginErr)
					}
					return
				}
				_, err = store.updateExistingThread(thread.Address, current.Revision, func(next *ThreadRecord) error {
					next.LastActiveAt = now.Add(time.Second)
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			controller := PairLifecycleController{
				Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
				Proc:  NewFakeProcOps(),
				Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "park-cas-conflict", nil },
			}
			if _, err := controller.Park(context.Background(), thread.Address); err == nil {
				t.Fatal("conflicted finalization returned success")
			}
			kept, err := store.GetThread(thread.Address)
			if err != nil {
				t.Fatal(err)
			}
			if test.replacement {
				if kept.Park == nil || kept.Park.Identity.Nonce != "park-replacement" || kept.Park.Phase != ParkRequested || len(kept.ParkHistory) != 1 || !kept.ParkHistory[0].Tombstoned || len(kept.Incarnations) != 1 {
					t.Fatalf("replacement transaction changed = %+v", kept)
				}
				return
			}
			if kept.Park == nil || kept.Park.Phase != ParkUnknown || kept.Park.Attempts[0].Failure == nil || kept.Park.Attempts[0].Failure.Code != test.wantCode || len(kept.Incarnations) != 1 {
				t.Fatalf("conflicted finalization = %+v", kept)
			}
		})
	}

	t.Run("older-success-closes-transaction", func(t *testing.T) {
		store, _, thread := createControllerThread(t)
		now := time.Unix(300, 0).UTC()
		identity := ParkIdentity{Nonce: "park-older-success", Address: thread.Address, PID: 42, ProcessIdentity: "pair-helper"}
		current, err := store.BeginPark(thread.Address, thread.Revision, identity)
		if err != nil {
			t.Fatal(err)
		}
		current, err = store.AdvancePark(thread.Address, current.Revision, ParkEvent{Kind: ParkRequestCommitted, Identity: identity, Attempt: 1})
		if err != nil {
			t.Fatal(err)
		}
		current, err = store.AdvancePark(thread.Address, current.Revision, ParkEvent{
			Kind: ParkFailureObserved, Identity: identity, Attempt: 1,
			Failure: &ParkFailure{Code: pairlifecycle.FailureTimeout, Diagnostic: "deadline"},
		})
		if err != nil {
			t.Fatal(err)
		}
		current, err = store.AppendParkAttempt(thread.Address, current.Revision, identity)
		if err != nil {
			t.Fatal(err)
		}
		artifacts := NewFakeThreadArtifactCollisionChecker()
		artifacts.SetPairSession(thread.Address, "pair-exact", true)
		lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(now), completions: map[uint64]pairlifecycle.QuitCompletion{}}
		controller := PairLifecycleController{
			Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
			Proc:  NewFakeProcOps(),
			Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "unused", nil },
		}
		request, _, err := controller.requestForAttempt(current, "pair-exact", 1)
		if err != nil {
			t.Fatal(err)
		}
		lifecycle.completions[1] = successCompletion(request, now)

		result, err := controller.Recover(context.Background(), thread.Address)
		if err != nil {
			t.Fatalf("Recover: %v", err)
		}
		if result.Thread.VerifiedPark == nil || result.Thread.VerifiedPark.Attempt != 1 || result.Thread.Park != nil {
			t.Fatalf("late older success result = %+v", result.Thread)
		}
		if got := artifacts.TriggeredQuits(); len(got) != 0 {
			t.Fatalf("newer attempt triggered after older success: %v", got)
		}
	})

	t.Run("abandon-late-success-noop", func(t *testing.T) {
		store, _, thread := createControllerThread(t)
		identity := ParkIdentity{Nonce: "park-abandoned", Address: thread.Address, PID: 42, ProcessIdentity: "pair-helper"}
		current, err := store.BeginPark(thread.Address, thread.Revision, identity)
		if err != nil {
			t.Fatal(err)
		}
		abandoned, err := store.AbandonPark(thread.Address, current.Revision, identity)
		if err != nil {
			t.Fatal(err)
		}
		artifacts := NewFakeThreadArtifactCollisionChecker()
		lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(time.Unix(400, 0).UTC())}
		controller := PairLifecycleController{
			Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
			Proc:  NewFakeProcOps(),
			Clock: FixedClock{T: time.Unix(400, 0).UTC()}, Nonce: func() (string, error) { return "unused", nil },
		}
		if err := controller.ReconcileActive(context.Background()); err != nil {
			t.Fatalf("ReconcileActive: %v", err)
		}
		got, err := store.GetThread(thread.Address)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, abandoned) || len(lifecycle.trace) != 0 || len(artifacts.TriggeredQuits()) != 0 {
			t.Fatalf("abandoned transaction changed: got=%+v want=%+v trace=%v triggers=%v", got, abandoned, lifecycle.trace, artifacts.TriggeredQuits())
		}
	})
}

func TestParkCoordinatorRestartInspectionAndRecoveryAreSeparate(t *testing.T) {
	store, ns, active := createControllerThread(t)
	identity := ParkIdentity{
		Nonce: "park-restart", Address: active.Address, PID: 42, ProcessIdentity: "pair-helper",
	}
	active, err := store.BeginPark(active.Address, active.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	inactive := validThreadRecord(t)
	inactive.Address.Tag = "couch-fedcba9876543210"
	inactive.StartingPath, inactive.WorkingPath = ns.Dir(), ns.Dir()
	if _, err := store.CreateThread(inactive); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(200, 0).UTC()
	lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(now)}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(active.Address, "pair-exact", true)
	controller := PairLifecycleController{
		Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
		Proc:  NewFakeProcOps(),
		Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "unused", nil },
	}
	couch := &Couch{Threads: store, PairLifecycle: &controller}
	if err := couch.ReconcileActiveParks(context.Background()); err != nil {
		t.Fatalf("ReconcileActive: %v", err)
	}
	reconciled, err := store.GetThread(active.Address)
	if err != nil || reconciled.Park == nil || reconciled.Park.Phase != ParkAwaitingCompletion || len(reconciled.Park.Attempts) != 1 {
		t.Fatalf("reconciled active = %+v, %v", reconciled, err)
	}
	if len(artifacts.TriggeredQuits()) != 0 || lifecycle.lastRequest.Attempt != 1 {
		t.Fatalf("inspection effects: triggers=%v request=%+v", artifacts.TriggeredQuits(), lifecycle.lastRequest)
	}
	controller.CompletionTimeout = 0
	recoveryErr := couch.RecoverActiveParks(context.Background())
	if recoveryErr == nil {
		t.Fatal("recovery without completion returned success")
	}
	if len(artifacts.TriggeredQuits()) != 1 {
		t.Fatalf("worker recovery triggers = %v, err=%v", artifacts.TriggeredQuits(), recoveryErr)
	}
}

func TestParkCoordinatorConstructorDoesNotQueryPairSession(t *testing.T) {
	ns := testCouchNamespace(t)
	store := NewThreadStore(ns)
	active := validThreadRecord(t)
	active.StartingPath, active.WorkingPath = ns.Dir(), ns.Dir()
	active.Reservation = false
	profile := LaunchProfile{Agent: "codex", Argv: []string{"--sandbox", "workspace-write"}}
	active.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-helper", State: IncarnationLive, LaunchProfile: &profile}}
	active.LatestLaunchProfile = &profile
	active, err := store.CreateThread(active)
	if err != nil {
		t.Fatal(err)
	}
	identity := ParkIdentity{Nonce: "park-constructor", Address: active.Address, PID: 42, ProcessIdentity: "pair-helper"}
	if _, err := store.BeginPark(active.Address, active.Revision, identity); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(500, 0).UTC()
	lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(now)}
	artifacts := constructorLifecycleArtifacts{
		FakeThreadArtifactCollisionChecker: NewFakeThreadArtifactCollisionChecker(),
		lifecycle:                          lifecycle,
		dataDir:                            t.TempDir(),
	}
	artifacts.SetPairSession(active.Address, "pair-exact", true)
	queryEntered := make(chan struct{})
	releaseQuery := make(chan struct{})
	artifacts.BeforePairSession = func(ThreadAddress) error {
		close(queryEntered)
		<-releaseQuery
		return nil
	}
	type newResult struct {
		couch *Couch
		err   error
	}
	constructed := make(chan newResult, 1)
	go func() {
		couch, err := New(
			ns, NewFakeRunner(), NewFakePathOps(nil), NewFakeGit(nil), NewFakeProcOps(), NewStore(ns.Dir()),
			FixedClock{T: now}, NewFixedIDGen("id"), newIncrementingEntropy(), artifacts,
		)
		constructed <- newResult{couch: couch, err: err}
	}()
	var couch *Couch
	select {
	case result := <-constructed:
		couch, err = result.couch, result.err
	case <-queryEntered:
		close(releaseQuery)
		result := <-constructed
		t.Fatalf("New queried Pair/Zellij before returning: %v", result.err)
	case <-time.After(5 * time.Second):
		// A LIVENESS backstop, not the assertion. What this test actually
		// proves is that New never queries Pair/Zellij, and the queryEntered
		// channel above proves that exactly -- a query blocks forever, so the
		// budget only has to exceed scheduling latency. At 100ms it fired twice
		// under a loaded parallel run of a 160-second package, both times
		// reporting a phantom regression with err=<nil>; New reaches no session
		// query at all (it calls reconcileInterruptedStarts, which reads
		// Registration). Raised so the false signal cannot recur while the real
		// one -- a constructor that blocks -- still fails here.
		close(releaseQuery)
		result := <-constructed
		t.Fatalf("New blocked before returning: %v", result.err)
	}
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if couch.PairLifecycle == nil {
		t.Fatal("New did not install the Pair lifecycle controller")
	}
	reconciled, err := couch.Threads.GetThread(active.Address)
	if err != nil || reconciled.Park == nil || reconciled.Park.Phase != ParkRequested {
		t.Fatalf("constructor durable state = %+v, %v", reconciled, err)
	}
	if len(lifecycle.trace) != 0 || len(artifacts.TriggeredQuits()) != 0 {
		t.Fatalf("constructor external effects: lifecycle=%v triggers=%v", lifecycle.trace, artifacts.TriggeredQuits())
	}

	couch.PairLifecycle.CompletionTimeout = 0
	recoveryDone := make(chan error, 1)
	go func() { recoveryDone <- couch.RecoverActiveParks(context.Background()) }()
	select {
	case <-queryEntered:
	case err := <-recoveryDone:
		t.Fatalf("recovery returned before querying Pair session: %v", err)
	case <-time.After(time.Second):
		t.Fatal("recovery did not query Pair session")
	}
	close(releaseQuery)
	if err := <-recoveryDone; err == nil {
		t.Fatal("recovery without completion returned success")
	}
	if len(artifacts.TriggeredQuits()) != 1 {
		t.Fatalf("worker recovery triggers = %v", artifacts.TriggeredQuits())
	}
}

// The whole-couch form of PARK. Alt+x in the switcher means the operator asked
// to stop every agent, so Leave has to perform the same verified teardown the
// per-thread park does -- not a detach wearing park's name.
func TestLeaveParkStopsEveryLiveThreadWithVerifiedParks(t *testing.T) {
	store, _, thread := createControllerThread(t)
	now := time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC)
	model := pairlifecycletest.New(now)
	model.SetSession("pair-exact", true)
	lifecycle := &fakeControllerLifecycle{model: model, store: store}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(thread.Address, "pair-exact", true)
	artifacts.TriggerQuitHook = func(_ string, _ launcher.QuitIntent) error {
		if err := model.DeliverTrigger(lifecycle.lastRequest); err != nil {
			return err
		}
		completion := successCompletion(lifecycle.lastRequest, now)
		lifecycle.completion = &completion
		return model.CommitCompletion(lifecycle.lastRequest, pairlifecycle.CleanupResult{
			Outcome: pairlifecycle.CompletionSuccess, CompletedAt: now,
		})
	}
	controller := PairLifecycleController{
		Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
		Proc: NewFakeProcOps(), Clock: FixedClock{T: now},
		Nonce: func() (string, error) { return "park-leave-0123456789", nil },
	}
	couch := &Couch{
		Threads: store, Proc: NewFakeProcOps(), Artifacts: artifacts,
		PairLifecycle: &controller, sleep: func(time.Duration) {},
	}

	result, err := couch.Leave(context.Background(), LeavePark)
	if err != nil {
		t.Fatalf("Leave: %v", err)
	}
	if len(result.Parked) != 1 || result.Parked[0] != thread.Address || len(result.Detached) != 0 {
		t.Fatalf("Leave = %+v, want the live thread parked and nothing detached", result)
	}
	if result.Disposition != LeavePark {
		t.Fatalf("Leave disposition = %q, want park -- the exit report has to say which ran", result.Disposition)
	}
	parked, err := store.GetThread(thread.Address)
	if err != nil || parked.VerifiedPark == nil || parked.Park != nil || len(parked.Incarnations) != 0 {
		t.Fatalf("thread = %+v, %v; want a verified park with no live incarnation", parked, err)
	}
	if got := artifacts.TriggeredQuits(); len(got) != 1 {
		t.Fatalf("typed Pair quit triggers = %v, want exactly one", got)
	}
}

// An unknown disposition is a refusal, not a default. Defaulting to detach
// would silently keep alive the agents a caller asked to stop, and defaulting
// to park would kill the ones it asked to keep.
func TestLeaveRefusesAnUnknownDisposition(t *testing.T) {
	store, _, _ := createControllerThread(t)
	couch := &Couch{Threads: store, Proc: NewFakeProcOps(), Artifacts: NewFakeThreadArtifactCollisionChecker()}
	result, err := couch.Leave(context.Background(), LeaveDisposition("evict"))
	if err == nil {
		t.Fatalf("Leave accepted an unknown disposition: %+v", result)
	}
	if len(result.Detached) != 0 || len(result.Parked) != 0 {
		t.Fatalf("refused Leave still acted: %+v", result)
	}
}

// Captures with the real Pair archive writer while keeping session effects in
// the lifecycle state machine fake.
type capturedParkOps struct {
	*pairlifecycletest.Fake
	rt         *launcher.OSRuntime
	paths      artifactpath.Paths
	agent      string
	descriptor *pairlifecycle.PreservedScrollback
}

func (o *capturedParkOps) PreserveScrollback(ctx context.Context, intent pairlifecycle.CleanupIntent) error {
	if err := o.Fake.PreserveScrollback(ctx, intent); err != nil {
		return err
	}
	base, ok := o.rt.ParkScrollback(o.paths.Tag(), o.agent, true)
	if !ok {
		return errors.New("archive failed")
	}
	template, _ := o.paths.ParkedScrollbackArtifacts("TOKEN")
	token := strings.TrimPrefix(base, strings.TrimSuffix(template.Base, "TOKEN"))
	archived, _ := o.paths.ParkedScrollbackArtifacts(token)
	_, err := os.Stat(archived.Events)
	o.descriptor = &pairlifecycle.PreservedScrollback{Agent: o.agent, Token: token, Events: err == nil}
	return nil
}
func (o *capturedParkOps) PreservedScrollback() *pairlifecycle.PreservedScrollback {
	return o.descriptor
}

func TestExactParkCaptureSurvivesCompletionAndReopen(t *testing.T) {
	store, namespace, thread := createControllerThread(t)
	dataDir := t.TempDir()
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: dataDir, RepoScope: thread.Address.RepoScope, Tag: string(thread.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	live, _ := paths.ScrollbackArtifacts("codex")
	if err := os.MkdirAll(filepath.Dir(live.Raw), 0700); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{live.Raw: "exact outgoing transcript", live.Events: "{\"offset\":0}\n"} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	model := pairlifecycletest.New(now)
	model.SetSession("pair-exact", true)
	lifecycle := &fakeControllerLifecycle{model: model, store: store}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(thread.Address, "pair-exact", true)
	var want *pairlifecycle.PreservedScrollback
	artifacts.TriggerQuitHook = func(_ string, intent launcher.QuitIntent) error {
		request := lifecycle.lastRequest
		if err := model.DeliverTrigger(request); err != nil {
			return err
		}
		lifecyclePaths, err := paths.Lifecycle(request.Identity.Nonce)
		if err != nil {
			return err
		}
		durable := pairlifecycle.Store{Runtime: pairlifecycle.OSRuntime{}}
		if err := durable.PublishRequest(lifecyclePaths, request); err != nil {
			return err
		}
		ops := &capturedParkOps{Fake: model, rt: launcher.NewOSRuntime(filepath.Dir(live.Raw), "/pair"), paths: paths, agent: "codex"}
		result, err := launcher.ConsumeCouchAttempt(context.Background(), durable, lifecyclePaths, *intent.Request, request.Session, ops)
		if err != nil {
			return err
		}
		completionPath, _ := lifecyclePaths.Completion(request.Attempt)
		raw, err := os.ReadFile(completionPath)
		if err != nil {
			return err
		}
		var completion pairlifecycle.QuitCompletion
		if err := json.Unmarshal(raw, &completion); err != nil {
			return err
		}
		if err := pairlifecycle.MatchQuitCompletion(request, completion); err != nil {
			return fmt.Errorf("completion mismatch: %w", err)
		}
		if !reflect.DeepEqual(completion.Scrollback, result.Scrollback) {
			return errors.New("completion lost capture")
		}
		want = pairlifecycle.ClonePreservedScrollback(result.Scrollback)
		lifecycle.completion = &completion
		return model.CommitCompletion(request, result)
	}
	controller := PairLifecycleController{Threads: store, DataDir: dataDir, Lifecycle: lifecycle, Sessions: artifacts, Proc: NewFakeProcOps(), Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "capture-nonce", nil }}
	result, err := controller.Park(context.Background(), thread.Address)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := NewThreadStore(namespace).GetThread(thread.Address)
	if err != nil {
		t.Fatal(err)
	}
	if want == nil || !want.Events || reopened.VerifiedPark == nil || !reflect.DeepEqual(reopened.VerifiedPark.Scrollback, want) {
		t.Fatalf("capture lost after reopen: %+v want %+v", reopened.VerifiedPark, want)
	}
	archived, _ := paths.ParkedScrollbackArtifacts(want.Token)
	raw, err := os.ReadFile(archived.Raw)
	if err != nil || string(raw) != "exact outgoing transcript" {
		t.Fatalf("archive = %q, %v", raw, err)
	}
	result.Thread.VerifiedPark.Scrollback.Token = "mutated"
	again, err := store.GetThread(thread.Address)
	if err != nil || !reflect.DeepEqual(again.VerifiedPark.Scrollback, want) {
		t.Fatal("returned descriptor aliases stored record", err)
	}
}
