package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"os"
	"testing"
	"time"
)

type durableNameObserver struct {
	*FakeThreadArtifactCollisionChecker
	named   []string
	binding PairSessionBinding
	err     error
}

func (o *durableNameObserver) NamedPairSessionContext(ctx context.Context, address ThreadAddress, name string) (PairSessionBinding, error) {
	o.named = append(o.named, name)
	if name != o.binding.Name {
		return PairSessionBinding{Name: name}, nil
	}
	return o.binding, o.err
}
func bindAuthorityThread(t *testing.T, s *ThreadStore, r ThreadRecord) ThreadRecord {
	t.Helper()
	next, err := s.updateExistingThread(r.Address, r.Revision, func(n *ThreadRecord) error {
		n.SessionBinding = &couchidentity.SessionBinding{C: 9, M: 4, Name: "📁9-4", ScopeKey: r.Address.RepoScope, Tag: string(r.Address.Tag), StartNonce: "original"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return next
}
func TestParkUsesDurableSessionInsteadOfStaleIndex(t *testing.T) {
	store, _, thread := createControllerThread(t)
	thread = bindAuthorityThread(t, store, thread)
	now := time.Unix(100, 0)
	boom := errors.New("stop after request publication")
	lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(now), store: store, publishErr: boom}
	fake := NewFakeThreadArtifactCollisionChecker()
	fake.SetPairSession(thread.Address, "pair-stale", true)
	observer := &durableNameObserver{FakeThreadArtifactCollisionChecker: fake, binding: PairSessionBinding{Name: thread.SessionBinding.Name, Present: true}}
	controller := PairLifecycleController{Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: observer, Proc: NewFakeProcOps(), Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "park-0123456789abcdef", nil }}
	_, _ = controller.Park(context.Background(), thread.Address)
	if lifecycle.lastRequest.Session != thread.SessionBinding.Name {
		t.Fatalf("request used %q, want durable %q", lifecycle.lastRequest.Session, thread.SessionBinding.Name)
	}
	if len(observer.named) == 0 {
		t.Fatal("no durable owner query")
	}
}
func TestDetachUsesDurableSessionWithoutIndex(t *testing.T) {
	f := newDetachFixture(t)
	thread, err := f.store.GetThread(f.address)
	if err != nil {
		t.Fatal(err)
	}
	thread = bindAuthorityThread(t, f.store, thread)
	f.artifact.SetPairSession(f.address, "pair-stale", false)
	observer := &durableNameObserver{FakeThreadArtifactCollisionChecker: f.artifact, binding: PairSessionBinding{Name: thread.SessionBinding.Name, Present: true}}
	f.couch.Artifacts = observer
	if _, err = f.couch.Detach(context.Background(), f.address); err != nil {
		t.Fatal(err)
	}
	if len(observer.named) < 2 {
		t.Fatalf("expected before/after durable queries, got %v", observer.named)
	}
}
func TestDetachedProofRejectsStaleDurableName(t *testing.T) {
	r := validThreadRecord(t)
	r.LatestLaunchProfile = &LaunchProfile{Agent: "codex", Argv: []string{}}
	r.SessionBinding = &couchidentity.SessionBinding{C: 9, M: 4, Name: "📁9-4", ScopeKey: r.Address.RepoScope, Tag: string(r.Address.Tag), StartNonce: "original"}
	proof := []DetachedSessionObservation{{Address: r.Address, Agent: "codex", SessionName: "pair-stale"}}
	if detachedResumeProofMatches(r, proof) {
		t.Fatal("stale index selected different terminal")
	}
	proof[0].SessionName = r.SessionBinding.Name
	if !detachedResumeProofMatches(r, proof) {
		t.Fatal("matching bound terminal refused")
	}
}

func TestSlotAdmissionProbesDurableNameEvenWhenIndexSaysAbsent(t *testing.T) {
	s := testLocalThreadStore(t)
	env := newTestEnv(t, s.slot.WorktreeRoot)
	env.Couch.Threads = s
	scope, _ := launcher.ResolveRepoScope(s.slot.WorktreeRoot)
	r := validThreadRecord(t)
	r.Address.RepoScope = scope.Key
	r.StartingPath = s.slot.WorktreeRoot
	r.WorkingPath = r.StartingPath
	r, err := s.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	r = bindAuthorityThread(t, s, r)
	env.Artifacts.SetSessionPresence(r.Address, SessionObservation{State: SessionAbsent})
	env.Artifacts.SetPairSession(r.Address, "pair-stale", false)
	observer := &durableNameObserver{FakeThreadArtifactCollisionChecker: env.Artifacts, binding: PairSessionBinding{Name: r.SessionBinding.Name, Present: true}}
	env.Couch.Artifacts = observer
	observed, err := env.Couch.ObserveSlotSessions(context.Background(), *s.slot)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Absent || len(observer.named) != 1 {
		t.Fatalf("live bound terminal ignored: %+v, probes %v", observed, observer.named)
	}
}

type detachedNameRequestObserver struct {
	*FakeThreadArtifactCollisionChecker
	selected string
}

func (o *detachedNameRequestObserver) DetachedSessions(ctx context.Context, candidates []DetachedCandidate) ([]DetachedSessionObservation, error) {
	if len(candidates) == 1 {
		o.selected = candidates[0].SessionName
	}
	return nil, errors.New("stop after detached observation")
}
func TestResumePassesDurableNameToDetachedObserver(t *testing.T) {
	env := newTestEnv(t, "/repo")
	r := validThreadRecord(t)
	r.StartingPath = "/repo"
	r.WorkingPath = "/repo"
	r.Reservation = false
	r.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	r, err := env.Couch.Threads.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	r = bindAuthorityThread(t, env.Couch.Threads, r)
	observer := &detachedNameRequestObserver{FakeThreadArtifactCollisionChecker: env.Artifacts}
	env.Couch.Artifacts = observer
	_, _, _ = env.Couch.ResumeContext(context.Background(), r.Address)
	if observer.selected != r.SessionBinding.Name {
		t.Fatalf("requested detached name %q, want %q", observer.selected, r.SessionBinding.Name)
	}
}
func TestDetachUnknownDurableOwnerRefusesBeforeSignal(t *testing.T) {
	f := newDetachFixture(t)
	r, err := f.store.GetThread(f.address)
	if err != nil {
		t.Fatal(err)
	}
	r = bindAuthorityThread(t, f.store, r)
	observer := &durableNameObserver{FakeThreadArtifactCollisionChecker: f.artifact, binding: PairSessionBinding{Name: r.SessionBinding.Name}, err: errors.New("owner unknown")}
	f.couch.Artifacts = observer
	if _, err = f.couch.Detach(context.Background(), f.address); err == nil {
		t.Fatal("unknown owner allowed detach")
	}
	if len(f.proc.GroupSignals[f.identity.PID]) != 0 {
		t.Fatal("signalled before owner proof")
	}
}

func TestSlotAdmissionChecksRetainedCurrentWhenPendingIsAbsent(t *testing.T) {
	s := testLocalThreadStore(t)
	env := newTestEnv(t, s.slot.WorktreeRoot)
	env.Couch.Threads = s
	scope, _ := launcher.ResolveRepoScope(s.slot.WorktreeRoot)
	r := validThreadRecord(t)
	r.Address.RepoScope = scope.Key
	r.StartingPath = s.slot.WorktreeRoot
	r.WorkingPath = r.StartingPath
	r, err := s.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	r = bindAuthorityThread(t, s, r)
	r, err = s.updateExistingThread(r.Address, r.Revision, func(next *ThreadRecord) error {
		pending := *next.SessionBinding
		pending.M = 5
		pending.Name = "📁9-5"
		pending.StartNonce = "pending"
		next.Incarnations = []ThreadIncarnation{{State: IncarnationCreating, Start: &ThreadStartClaim{Nonce: "pending", OwnerPID: 42, OwnerIdentity: "dead-owner", SessionBinding: &pending}}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetSessionPresence(r.Address, SessionObservation{State: SessionAbsent})
	observer := &durableNameObserver{FakeThreadArtifactCollisionChecker: env.Artifacts, binding: PairSessionBinding{Name: r.SessionBinding.Name, Present: true}}
	env.Couch.Artifacts = observer
	observed, err := env.Couch.ObserveSlotSessions(context.Background(), *s.slot)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Absent {
		t.Fatalf("pending absence hid retained current terminal; queried %v", observer.named)
	}
}

func TestSlotAdmissionRetainsUnknownBindingFromDamagedRecord(t *testing.T) {
	s := testLocalThreadStore(t)
	env := newTestEnv(t, s.slot.WorktreeRoot)
	env.Couch.Threads = s
	scope, _ := launcher.ResolveRepoScope(s.slot.WorktreeRoot)
	r := validThreadRecord(t)
	r.Address.RepoScope = scope.Key
	r.StartingPath = s.slot.WorktreeRoot
	r.WorkingPath = r.StartingPath
	r, err := s.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	r = bindAuthorityThread(t, s, r)
	r.Revision = 0
	raw, _ := json.Marshal(r)
	if err = os.WriteFile(s.recordPath(r.Address), raw, 0600); err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetSessionPresence(r.Address, SessionObservation{State: SessionAbsent})
	observed, err := env.Couch.ObserveSlotSessions(context.Background(), *s.slot)
	if err == nil && observed.Absent {
		t.Fatal("damaged bound record silently proved terminal absent")
	}
}
