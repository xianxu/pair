package couchcore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

type testEnv struct {
	Couch     *Couch
	Runner    *FakeRunner
	Git       *FakeGit
	Proc      *FakeProcOps
	Artifacts *FakeThreadArtifactCollisionChecker
	Dir       string
	Now       time.Time
}

type realDescendantRunner struct {
	marker         string
	acknowledgeErr error
	pty            bool
}

func (r realDescendantRunner) Start(_ string, argv, env []string) (Handle, error) {
	return ExecRunner{}.Start(filepath.Dir(r.marker), argv, env)
}

func (r realDescendantRunner) StartBlocked(_ string, _ []string, _ []string, timeout time.Duration) (BlockedHandle, error) {
	script := `trap '' TERM; sleep 300 & child=$!; printf '%s' "$child" > "$PAIR_TEST_DESCENDANT_PID"; wait "$child"`
	argv := []string{"sh", "-c", script}
	env := []string{
		"PAIR_TEST_RUNNER_HELPER=1",
		"PAIR_TEST_DESCENDANT_PID=" + r.marker,
	}
	var h BlockedHandle
	var err error
	if r.pty {
		h, err = (&PtyRunner{LaunchHelper: os.Args[0]}).StartBlocked(filepath.Dir(r.marker), argv, env, timeout)
	} else {
		h, err = (ExecRunner{LaunchHelper: os.Args[0]}).StartBlocked(filepath.Dir(r.marker), argv, env, timeout)
	}
	if err != nil {
		return nil, err
	}
	return &realDescendantBlockedHandle{BlockedHandle: h, marker: r.marker, acknowledgeErr: r.acknowledgeErr}, nil
}

type realDescendantBlockedHandle struct {
	BlockedHandle
	marker         string
	acknowledgeErr error
}

func (h *realDescendantBlockedHandle) Acknowledge() error {
	if err := h.BlockedHandle.Acknowledge(); err != nil {
		return err
	}
	if _, err := waitForPIDFile(h.marker, 2*time.Second); err != nil {
		return err
	}
	return h.acknowledgeErr
}

func waitForPIDFile(path string, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(path)
		if err == nil {
			pid, parseErr := strconv.Atoi(strings.TrimSpace(string(raw)))
			if parseErr == nil && pid > 0 {
				return pid, nil
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	return 0, fmt.Errorf("timed out waiting for descendant pidfile %s", path)
}

func killAndVerifyProcess(pid int, timeout time.Duration) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	_ = proc.Kill()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return fmt.Errorf("process %d remained after kill", pid)
}

func TestStoreNamespaceMustMatchCouchNamespace(t *testing.T) {
	ns := testCouchNamespace(t)
	other := t.TempDir()
	_, err := New(
		ns,
		NewFakeRunner(), NewFakePathOps(nil), NewFakeGit(nil), NewFakeProcOps(),
		NewStore(other), FixedClock{}, NewFixedIDGen("id"), NewFakePolicyResolver(), bytes.NewReader(make([]byte, 8)), NoThreadArtifactCollisions{},
	)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("New mismatched store err = %v", err)
	}
}

func TestCouchRetainsInjectedPolicyResolver(t *testing.T) {
	ns := testCouchNamespace(t)
	resolver := NewFakePolicyResolver()
	c, err := New(
		ns,
		NewFakeRunner(), NewFakePathOps(nil), NewFakeGit(nil), NewFakeProcOps(),
		NewStore(ns.Dir()), FixedClock{}, NewFixedIDGen("id"), resolver, bytes.NewReader(make([]byte, 8)), NoThreadArtifactCollisions{},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.PolicyResolver != resolver {
		t.Fatal("Couch did not retain the injected policy resolver")
	}
}

// newTestEnv wires the whole composition root against fakes, with a fixed
// clock and a scripted id generator so every assertion is deterministic.
func newTestEnv(t *testing.T, trees ...string) *testEnv {
	t.Helper()
	replies := map[GitCall]string{}
	for _, tr := range trees {
		replies[GitCall{Dir: tr, Args: "rev-parse --show-toplevel"}] = tr
	}
	g := NewFakeGit(replies)
	r := NewFakeRunner()
	proc := NewFakeProcOps()
	dir := t.TempDir()
	ns, err := ResolveCouchNamespace(dir, "/unused")
	if err != nil {
		t.Fatalf("ResolveCouchNamespace: %v", err)
	}
	dir = ns.Dir()
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.AutoEstablish(true)
	c, err := New(ns, r, NewFakePathOps(nil), g, proc, NewStore(dir), FixedClock{T: now}, NewFixedIDGen("ah8d", "b2c1"), NewFakePolicyResolver(), newIncrementingEntropy(), artifacts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.postAckQuiesceTimeout = 5 * time.Millisecond
	return &testEnv{Couch: c, Runner: r, Git: g, Proc: proc, Artifacts: artifacts, Dir: dir, Now: now}
}

// spawn spawns and then marks the child live in FakeProcOps, which is what a
// real process would be. Tests that skip this are modelling a DEAD actor --
// which is a legitimate scenario, just not the default one.
func (e *testEnv) spawn(t *testing.T, args StartArgs) (ActorRecord, Handle) {
	t.Helper()
	rec, h, err := e.Couch.Spawn(args)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	e.Proc.Set(rec.PID, rec.Identity)
	return rec, h
}

func (e *testEnv) cannedTree(tree, cwd string) {
	e.Git.replies[GitCall{Dir: cwd, Args: "rev-parse --show-toplevel"}] = tree
}

func (e *testEnv) boundedOne(path string) {
	e.Couch.PolicyResolver.(*FakePolicyResolver).SetDefault(PolicyResult{
		PolicyVersion: 1,
		PolicyDigest:  strings.Repeat("a", 64),
		RepoIdentity:  "fake-repo",
		AdmissionKey:  path,
		Capacity:      PolicyCapacity{Kind: CapacityBounded, Limit: 1},
		OnCapacity:    CapacityReject,
	}, nil)
}

func TestSpawnStartsPairAndRecordsTheActor(t *testing.T) {
	env := newTestEnv(t, "/repo")
	rec, h, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if rec.ID != "couch-ah8d" {
		t.Fatalf("id = %q", rec.ID)
	}
	// couch spawns pair, not claude: pair owns zellij, the layout, and the
	// agent's resume/session-id knowledge.
	if got := env.Runner.Ops[0]; got != "start /repo: pair resume couch-0102030405060708 --layout2" {
		t.Fatalf("Ops[0] = %q", got)
	}
	child := env.Runner.Child(env.Runner.order[0])
	wantEnv := []string{
		"COUCH_TREE=/repo",
		"COUCH_STORE_DIR=" + env.Dir,
		"COUCH_THREAD_SCOPE=816fc349d3faebf8",
		"COUCH_THREAD_TAG=couch-0102030405060708",
		"PAIR_USE_REPO_DEFAULT=1",
	}
	if !slices.Equal(child.Env, wantEnv) {
		t.Fatalf("child env = %q, want %q", child.Env, wantEnv)
	}
	if !rec.StartedAt.Equal(env.Now) {
		t.Fatalf("StartedAt = %v, want the injected clock", rec.StartedAt)
	}
	if rec.Identity == "" || rec.PID == 0 {
		t.Fatalf("liveness fields not recorded: %+v", rec)
	}
	if rec.Thread != (ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}) {
		t.Fatalf("thread address = %+v", rec.Thread)
	}
	thread, err := env.Couch.Threads.GetThread(rec.Thread)
	if err != nil || len(thread.Incarnations) != 1 || thread.Incarnations[0].State != IncarnationLive {
		t.Fatalf("durable thread after spawn = %+v, %v", thread, err)
	}
	_ = h
}

func TestSpawnPersistsHelperIdentityBeforeAcknowledgingExec(t *testing.T) {
	env := newTestEnv(t, "/repo")
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	env.Runner.BeforeAcknowledge = func(id string) error {
		thread, err := env.Couch.Threads.GetThread(address)
		if err != nil {
			return err
		}
		if len(thread.Incarnations) != 1 {
			return fmt.Errorf("incarnations = %+v", thread.Incarnations)
		}
		incarnation := thread.Incarnations[0]
		if incarnation.State != IncarnationCreating || incarnation.Start == nil || incarnation.PID == 0 || incarnation.Identity == "" {
			return fmt.Errorf("helper was not durably recorded before ack: %+v", incarnation)
		}
		child := env.Runner.Child(id)
		if !child.Blocked || child.ExecCount != 0 {
			return fmt.Errorf("target ran before durable helper record: %+v", child)
		}
		return nil
	}
	record, handle, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if record.Thread != address || env.Runner.Child(handle.ID()).ExecCount != 1 {
		t.Fatalf("record/child = %+v / %+v", record, env.Runner.Child(handle.ID()))
	}
}

func TestSpawnPostAcknowledgementFailuresNeverLeaveWorkspaceWriter(t *testing.T) {
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	tests := []struct {
		name      string
		setup     func(*testing.T, *testEnv)
		wantState IncarnationState
		wantStart bool
		wantDesc  string
	}{
		{
			name: "registration evidence failure",
			setup: func(_ *testing.T, env *testEnv) {
				env.Artifacts.SetRegistration(address, RegistrationUnknown, errors.New("registration unreadable"))
			},
			wantState: IncarnationCreating,
			wantStart: true,
		},
		{
			name: "durable promotion conflict",
			setup: func(t *testing.T, env *testEnv) {
				var once sync.Once
				env.Artifacts.BeforeRegistration = func(got ThreadAddress) error {
					var hookErr error
					once.Do(func() {
						current, err := env.Couch.Threads.GetThread(got)
						if err != nil {
							hookErr = err
							return
						}
						_, hookErr = env.Couch.Threads.UpdateExistingThread(got, current.Revision, func(next *ThreadRecord) error {
							next.Description = "concurrent description"
							return nil
						})
					})
					return hookErr
				}
			},
			wantState: IncarnationUnknown,
			wantDesc:  "concurrent description",
		},
		{
			name: "legacy registry persistence failure",
			setup: func(t *testing.T, env *testEnv) {
				badStorePath := filepath.Join(t.TempDir(), "not-a-directory")
				if err := os.WriteFile(badStorePath, []byte("occupied"), 0o600); err != nil {
					t.Fatal(err)
				}
				env.Couch.Store = NewStore(badStorePath)
			},
			wantState: IncarnationUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			tt.setup(t, env)

			_, handle, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
			if err == nil || handle == nil {
				t.Fatalf("Spawn handle/error = %T, %v", handle, err)
			}
			child := env.Runner.Child(handle.ID())
			if handle.Alive() || child.ExecCount != 1 {
				t.Fatalf("post-ack child survived or never execed: %+v", child)
			}
			thread, getErr := env.Couch.Threads.GetThread(address)
			if getErr != nil || len(thread.Incarnations) != 1 {
				t.Fatalf("durable occupied thread = %+v, %v", thread, getErr)
			}
			incarnation := thread.Incarnations[0]
			if incarnation.State != tt.wantState || (incarnation.Start != nil) != tt.wantStart || thread.Description != tt.wantDesc {
				t.Fatalf("durable post-ack failure = %+v", thread)
			}
			if got := len(env.Couch.reg.Records()); got != 0 {
				t.Fatalf("failed handoff retained %d legacy registry actor(s)", got)
			}
			if got := env.Artifacts.Quiesces(); !slices.Equal(got, []ThreadAddress{address}) {
				t.Fatalf("whole-incarnation quiescence calls = %+v", got)
			}
		})
	}
}

func TestSpawnAcknowledgementFailureCancelsHelperBeforeRollback(t *testing.T) {
	env := newTestEnv(t, "/repo")
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	env.Artifacts.AutoEstablish(false)
	env.Runner.BeforeAcknowledge = func(string) error { return errors.New("ack transport failed") }
	_, handle, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err == nil || handle == nil {
		t.Fatalf("Spawn = %T, %v", handle, err)
	}
	if handle.Alive() || env.Runner.Child(handle.ID()).ExecCount != 0 {
		t.Fatalf("failed ack helper = %+v", env.Runner.Child(handle.ID()))
	}
	if _, err := env.Couch.Threads.GetThread(address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("cancelled exact start remained: %v", err)
	}
}

func TestSpawnPossiblyDeliveredAcknowledgementQuiescesBeforeRollback(t *testing.T) {
	env := newTestEnv(t, "/repo")
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	env.Artifacts.AutoEstablish(false)
	env.Runner.AfterAcknowledge = func(string) error { return errors.New("close after delivered acknowledgement") }
	_, handle, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err == nil || handle == nil {
		t.Fatalf("Spawn = %T, %v", handle, err)
	}
	child := env.Runner.Child(handle.ID())
	if child.ExecCount != 1 || handle.Alive() {
		t.Fatalf("possibly delivered acknowledgement left writer: %+v", child)
	}
	if _, err := env.Couch.Threads.GetThread(address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("dead unregistered ambiguous start remained: %v", err)
	}
}

func TestSpawnRetainsOwnershipAndRetriesUntilQuiescenceIsProven(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.postAckRetryDelay = time.Millisecond
	env.Artifacts.SetRegistration(ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}, RegistrationUnknown, errors.New("registration unreadable"))

	var mu sync.Mutex
	failing := true
	attempts := 0
	env.Artifacts.QuiesceHook = func(ThreadAddress) error {
		mu.Lock()
		defer mu.Unlock()
		attempts++
		if failing {
			return errors.New("zellij absence unobservable")
		}
		return nil
	}
	done := make(chan error, 1)
	go func() {
		_, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
		done <- err
	}()

	deadline := time.Now().Add(time.Second)
	for {
		mu.Lock()
		gotAttempts := attempts
		mu.Unlock()
		if gotAttempts >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("cleanup owner did not retry")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case err := <-done:
		t.Fatalf("Spawn returned before quiescence proof: %v", err)
	default:
	}

	mu.Lock()
	failing = false
	mu.Unlock()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "zellij absence unobservable") {
			t.Fatalf("Spawn retry result = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Spawn did not return after quiescence proof")
	}
}

func TestSpawnPostAckFailuresQuiesceRealPersistentDescendant(t *testing.T) {
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	for _, runnerMode := range []struct {
		name string
		pty  bool
	}{{name: "stdio"}, {name: "pty", pty: true}} {
		t.Run(runnerMode.name, func(t *testing.T) {
			for _, mode := range []string{"acknowledgement ambiguous", "registration", "promotion", "registry save"} {
				t.Run(mode, func(t *testing.T) {
					env := newTestEnv(t, "/repo")
					env.Couch.postAckQuiesceTimeout = 250 * time.Millisecond
					marker := filepath.Join(t.TempDir(), "descendant.pid")
					runner := realDescendantRunner{marker: marker, pty: runnerMode.pty}
					if mode == "acknowledgement ambiguous" {
						runner.acknowledgeErr = errors.New("close after acknowledgement delivery")
						env.Artifacts.AutoEstablish(false)
					}
					env.Couch.Runner = runner
					if mode == "registration" {
						env.Artifacts.SetRegistration(address, RegistrationUnknown, errors.New("registration unreadable"))
					}
					if mode == "promotion" {
						var once sync.Once
						env.Artifacts.BeforeRegistration = func(got ThreadAddress) error {
							var hookErr error
							once.Do(func() {
								current, err := env.Couch.Threads.GetThread(got)
								if err != nil {
									hookErr = err
									return
								}
								_, hookErr = env.Couch.Threads.UpdateExistingThread(got, current.Revision, func(next *ThreadRecord) error {
									next.Description = "concurrent description"
									return nil
								})
							})
							return hookErr
						}
					}
					if mode == "registry save" {
						badStorePath := filepath.Join(t.TempDir(), "not-a-directory")
						if err := os.WriteFile(badStorePath, []byte("occupied"), 0o600); err != nil {
							t.Fatal(err)
						}
						env.Couch.Store = NewStore(badStorePath)
					}

					var descendantPID int
					t.Cleanup(func() {
						if descendantPID == 0 {
							descendantPID, _ = waitForPIDFile(marker, 50*time.Millisecond)
						}
						if descendantPID != 0 {
							_ = killAndVerifyProcess(descendantPID, 100*time.Millisecond)
						}
					})

					_, handle, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
					if err == nil || handle == nil {
						t.Fatalf("Spawn = %T, %v", handle, err)
					}
					descendantPID, err = waitForPIDFile(marker, time.Second)
					if err != nil {
						t.Fatal(err)
					}
					if handle.Alive() || descendantPID == 0 {
						t.Fatalf("quiescence did not cover client and descendant: handle_alive=%v descendant=%d", handle.Alive(), descendantPID)
					}
					if err := syscall.Kill(descendantPID, 0); !errors.Is(err, syscall.ESRCH) {
						t.Fatalf("persistent descendant %d survived: %v", descendantPID, err)
					}
				})
			}
		})
	}
}

func TestNewReconcilesDeadUnregisteredHelperByExactNonce(t *testing.T) {
	ns := testCouchNamespace(t)
	store := NewThreadStore(ns)
	record := admittedStartRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record, _ = AdvanceStartTransaction(record, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner-token"},
	})
	record, _ = AdvanceStartTransaction(record, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: 42, Identity: "helper-token"},
	})
	if _, err := store.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetRegistration(record.Address, RegistrationAbsent, nil)
	_, err := New(ns, NewFakeRunner(), NewFakePathOps(nil), NewFakeGit(nil), NewFakeProcOps(), NewStore(ns.Dir()), FixedClock{}, NewFixedIDGen("id"), NewFakePolicyResolver(), newIncrementingEntropy(), artifacts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := store.GetThread(record.Address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("dead unregistered start remained: %v", err)
	}
}

func TestNewPromotesEstablishedSurvivingHelper(t *testing.T) {
	ns := testCouchNamespace(t)
	store := NewThreadStore(ns)
	record := admittedStartRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record, _ = AdvanceStartTransaction(record, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner-token"},
	})
	record, _ = AdvanceStartTransaction(record, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: 42, Identity: "helper-token"},
	})
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	proc := NewFakeProcOps()
	proc.Set(42, "helper-token")
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetRegistration(record.Address, RegistrationEstablished, nil)
	_, err = New(ns, NewFakeRunner(), NewFakePathOps(nil), NewFakeGit(nil), proc, NewStore(ns.Dir()), FixedClock{}, NewFixedIDGen("id"), NewFakePolicyResolver(), newIncrementingEntropy(), artifacts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got, err := store.GetThread(record.Address)
	if err != nil || got.Revision != created.Revision+1 || got.Incarnations[0].State != IncarnationLive || got.Incarnations[0].Start != nil {
		t.Fatalf("promoted thread = %+v, %v", got, err)
	}
}

func TestSpawnStartsInASubdirectoryButRegistersTheTree(t *testing.T) {
	// The kbench/competition/arc-agi-3 case.
	env := newTestEnv(t)
	env.cannedTree("/w/kbench", "/w/kbench/competition/arc-agi-3")
	rec, _, err := env.Couch.Spawn(StartArgs{Cwd: "/w/kbench/competition/arc-agi-3"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if rec.Args.Worktree != "/w/kbench" {
		t.Fatalf("registered under %q, want the tree root", rec.Args.Worktree)
	}
	if got := env.Runner.Child(env.Runner.order[0]).Dir; got != "/w/kbench/competition/arc-agi-3" {
		t.Fatalf("child started in %q, want the requested subdirectory", got)
	}
}

func TestRefusedSpawnStartsNoProcess(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.boundedOne("/repo")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	before := len(env.Runner.Ops)
	_, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	var occ *CapacityExceededError
	if !errors.As(err, &occ) {
		t.Fatalf("err = %v, want *CapacityExceededError", err)
	}
	if len(env.Runner.Ops) != before {
		t.Fatal("a refused spawn must not fork a child")
	}
}

func TestSnapshotIsOnDiskWhileTheChildIsStillAlive(t *testing.T) {
	// `couch start` blocks for the child's lifetime, so if Save happened after
	// Wait a second shell running `couch list` would see nothing for the whole
	// session -- which is most of the time.
	env := newTestEnv(t, "/repo")
	_, h := env.spawn(t, StartArgs{Worktree: "/repo"})
	if !h.Alive() {
		t.Fatal("child should still be running")
	}
	raw, err := os.ReadFile(filepath.Join(env.Dir, "registry.json"))
	if err != nil {
		t.Fatalf("snapshot not written before Wait: %v", err)
	}
	var snap struct {
		Actors []ActorRecord `json:"actors"`
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(snap.Actors) != 1 {
		t.Fatalf("snapshot has %d actors", len(snap.Actors))
	}
}

func TestSpawnFailureLeavesTheTreeFree(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Runner.FailNextStart(errors.New("boom"))
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err == nil {
		t.Fatal("expected a start failure")
	}
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err != nil {
		t.Fatalf("tree still held after a failed spawn: %v", err)
	}
}

func TestSpawnCapacityRefusalDoesNotForkAndRollsBackOpaqueReservation(t *testing.T) {
	env := newTestEnv(t, "/repo")
	bounded := PolicyResult{
		PolicyVersion: 1, PolicyDigest: strings.Repeat("a", 64),
		RepoIdentity: "repo", AdmissionKey: "/repo",
		Capacity: PolicyCapacity{Kind: CapacityBounded, Limit: 1}, OnCapacity: CapacityReject,
	}
	env.Couch.PolicyResolver.(*FakePolicyResolver).SetDefault(bounded, nil)
	first, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	before := len(env.Runner.Ops)
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err == nil {
		t.Fatal("second bounded spawn was admitted")
	}
	if len(env.Runner.Ops) != before {
		t.Fatal("capacity refusal forked a child")
	}
	snapshot, err := env.Couch.Threads.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Records) != 1 || snapshot.Records[0].Address != first.Thread {
		t.Fatalf("refused reservation leaked: %+v", snapshot.Records)
	}
	if got := env.Artifacts.Releases(); len(got) != 1 {
		t.Fatalf("capacity refusal released claims = %+v", got)
	}
}

func TestSpawnPolicyInstabilityDoesNotForkAndRollsBackOpaqueReservation(t *testing.T) {
	env := newTestEnv(t, "/repo")
	scope, err := launcher.ResolveRepoScope("/repo")
	if err != nil {
		t.Fatal(err)
	}
	incumbent, err := env.Couch.Threads.AllocateThreadTag(scope.Key, "/repo", env.Now, bytes.NewReader(bytes.Repeat([]byte{9}, 8)), NoThreadArtifactCollisions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.Threads.UpdateExistingThread(incumbent.Address, incumbent.Revision, func(next *ThreadRecord) error {
		next.Reservation = false
		next.Incarnations = []ThreadIncarnation{{State: IncarnationCreating, StartedAt: env.Now}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	epochA := PolicyResult{
		PolicyVersion: 1, PolicyDigest: strings.Repeat("a", 64), RepoIdentity: "repo", AdmissionKey: "/repo",
		Capacity: PolicyCapacity{Kind: CapacityBounded, Limit: 2}, OnCapacity: CapacityReject,
	}
	epochB := epochA
	epochB.PolicyDigest = strings.Repeat("b", 64)
	resolver := env.Couch.PolicyResolver.(*FakePolicyResolver)
	for range 3 {
		resolver.Queue("/repo", epochA, nil)
		resolver.Queue("/repo", epochB, nil)
	}
	before := len(env.Runner.Ops)
	_, _, err = env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	var unstable *PolicyUnstableError
	if !errors.As(err, &unstable) {
		t.Fatalf("err = %T %v, want *PolicyUnstableError", err, err)
	}
	if len(env.Runner.Ops) != before {
		t.Fatal("unstable policy forked a child")
	}
	snapshot, err := env.Couch.Threads.Snapshot()
	if err != nil || len(snapshot.Records) != 1 || snapshot.Records[0].Address != incumbent.Address {
		t.Fatalf("unstable reservation leaked: %+v, %v", snapshot.Records, err)
	}
	if got := env.Artifacts.Releases(); len(got) != 1 {
		t.Fatalf("unstable policy released claims = %+v", got)
	}
}

func TestIsLiveRejectsARecycledPID(t *testing.T) {
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	if !env.Couch.IsLive(rec) {
		t.Fatal("a running actor must read as live")
	}
	// Same PID, different process: the kernel start token differs.
	env.Proc.Set(rec.PID, "some-other-process")
	if env.Couch.IsLive(rec) {
		t.Fatal("a recycled PID must not read as the original actor")
	}
	env.Proc.Kill(rec.PID)
	if env.Couch.IsLive(rec) {
		t.Fatal("a dead PID must not read as live")
	}
}

func TestResolveRefFindsActorsByOperatorName(t *testing.T) {
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	if err := env.Couch.SetName("/repo", "refactor thing"); err != nil {
		t.Fatalf("SetName: %v", err)
	}
	got, _, err := env.Couch.ResolveRef("refactor")
	if err != nil {
		t.Fatalf("ResolveRef: %v", err)
	}
	if len(got) != 1 || got[0].ID != rec.ID {
		t.Fatalf("ResolveRef = %+v", got)
	}
}

// The panel renders Worktree.Repo() when a tree has no explicit name. A label
// that is visible but cannot be typed back into the shared resolver makes
// typeahead lie: it shows "pair" and returns no match for "pair".
func TestLookupTreesMatchesTheDisplayedRepoFallback(t *testing.T) {
	env := newTestEnv(t, "/w/pair")
	env.spawn(t, StartArgs{Worktree: "/w/pair"})

	got := env.Couch.LookupTrees("pair")
	if len(got) != 1 || got[0] != "/w/pair" {
		t.Fatalf("LookupTrees(pair) = %v, want [/w/pair]", got)
	}
}

func TestNameAndDescriptionChangeMidSession(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	_ = env.Couch.SetName("/repo", "first")
	_ = env.Couch.SetName("/repo", "second")
	if got, _, err := env.Couch.ResolveRef("second"); err != nil || len(got) != 1 {
		t.Fatalf("rename did not take effect: %v %v", got, err)
	}
	if got, _, err := env.Couch.ResolveRef("first"); err == nil && len(got) > 0 {
		t.Fatal("the old name must stop resolving")
	}
	_ = env.Couch.SetDescription("/repo", "reworking the composer gate")
	if got, _, err := env.Couch.ResolveRef("composer"); err != nil || len(got) != 1 {
		t.Fatalf("description did not take effect: %v %v", got, err)
	}
}

func TestNameSurvivesActorReplacement(t *testing.T) {
	// A real lifecycle: spawn, name, the child exits, the actor is forgotten,
	// a new one is spawned. The name must still resolve, because it hangs off
	// the tree rather than the incarnation.
	env := newTestEnv(t, "/repo")
	first, h := env.spawn(t, StartArgs{Worktree: "/repo"})
	_ = env.Couch.SetName("/repo", "long lived")

	env.Runner.SetExited(h.ID(), 0)
	env.Proc.Kill(first.PID)
	if err := env.Couch.Forget("/repo", first.ID); err != nil {
		t.Fatalf("Forget: %v", err)
	}

	second, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	if second.ID == first.ID {
		t.Fatal("the revival must be a new incarnation")
	}
	got, _, err := env.Couch.ResolveRef("long lived")
	if err != nil || len(got) != 1 || got[0].ID != second.ID {
		t.Fatalf("name lost across revival: %+v %v", got, err)
	}
}

// --- close-review regressions: each of these fails against the shipped code ---

func TestDeadPairClientDoesNotFreeWholeIncarnationCapacity(t *testing.T) {
	// A Pair client can die while its detached zellij session and workspace-
	// writing panes survive. Client death is not whole-incarnation quiescence.
	env := newTestEnv(t, "/repo")
	env.boundedOne("/repo")
	first, h := env.spawn(t, StartArgs{Worktree: "/repo"})

	env.Runner.SetExited(h.ID(), 0)
	env.Proc.Kill(first.PID) // the process is gone; the record is not

	_, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	var full *CapacityExceededError
	if !errors.As(err, &full) {
		t.Fatalf("dead client err = %T %v, want occupied capacity", err, err)
	}
	thread, err := env.Couch.Threads.GetThread(first.Thread)
	if err != nil || len(thread.Incarnations) != 1 {
		t.Fatalf("whole incarnation was freed with its client: %+v, %v", thread, err)
	}
}

func TestLiveActorStillBlocksItsTree(t *testing.T) {
	// The complement of BR-1: pruning must not weaken the guard.
	env := newTestEnv(t, "/repo")
	env.boundedOne("/repo")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err == nil {
		t.Fatal("a live actor must still refuse its tree")
	}
}

func TestStopSignalsTheChildBeforeForgettingIt(t *testing.T) {
	// BR-2. Forgetting first frees the tree while the agent keeps running, so
	// the next start is allowed and two agents share one index lock.
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})

	signalled, err := env.Couch.Stop(rec)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !signalled {
		t.Fatal("Stop must signal a live child, not merely forget it")
	}
	got := env.Proc.Signals[rec.PID]
	if len(got) != 1 || got[0] != TermSignal {
		t.Fatalf("signals = %v, want one SIGTERM", got)
	}
	if len(env.Couch.Get("/repo")) != 0 {
		t.Fatal("the record should be gone after Stop")
	}
}

func TestStopOnADeadActorForgetsWithoutSignalling(t *testing.T) {
	env := newTestEnv(t, "/repo")
	rec, h := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.Runner.SetExited(h.ID(), 0)
	env.Proc.Kill(rec.PID)

	signalled, err := env.Couch.Stop(rec)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if signalled {
		t.Fatal("a dead actor must not be reported as signalled -- that implies a running agent was terminated")
	}
}

func TestShowFilterRestrictsRatherThanAdds(t *testing.T) {
	// BR-3. Summarize took a filter and then folded in every registry record,
	// so `show <ref>` printed exactly what `list` printed. The old test passed
	// only because its fixture had a single tree.
	env := newTestEnv(t, "/repo", "/other")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	env.spawn(t, StartArgs{Worktree: "/other"})

	got := env.Couch.Summarize([]Worktree{"/repo"})
	if len(got) != 1 {
		var trees []Worktree
		for _, s := range got {
			trees = append(trees, s.Tree)
		}
		t.Fatalf("Summarize([/repo]) returned %v; a filter must restrict, not add", trees)
	}
	if got[0].Tree != "/repo" {
		t.Fatalf("returned %q", got[0].Tree)
	}
	if len(env.Couch.Summarize(nil)) != 2 {
		t.Fatal("an empty filter must still list everything")
	}
}

func TestReplayPreservesSameTreeExactly(t *testing.T) {
	// BR-4. Load used to set SameTree=true on every record to dodge its own
	// re-register refusal, and the next Save persisted the lie -- after which
	// no reader could tell which actors really used the escape hatch.
	dir := t.TempDir()
	s := NewStore(dir)
	reg := NewRegistry().Insert(ActorRecord{ID: "plain", Args: StartArgs{Worktree: "/repo"}})
	reg = reg.Insert(ActorRecord{ID: "hatch", Args: StartArgs{Worktree: "/repo", SameTree: true}})
	if err := s.Save(reg, NewNamingTable()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, names, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := s.Save(loaded, names); err != nil { // round two is where the lie used to stick
		t.Fatalf("re-Save: %v", err)
	}
	again, _, _ := s.Load()

	byID := map[ActorID]bool{}
	for _, r := range again.Records() {
		byID[r.ID] = r.Args.SameTree
	}
	if byID["plain"] {
		t.Error("SameTree fabricated on a record that never used the escape hatch")
	}
	if !byID["hatch"] {
		t.Error("SameTree lost on a record that did use it")
	}
}

func TestUnreadableRegistryErrorsRatherThanReadingAsFirstRun(t *testing.T) {
	// BR-5. Load discarded every ReadFile error, so an unreadable snapshot
	// looked like a fresh install and the next Save destroyed it.
	dir := t.TempDir()
	s := NewStore(dir)
	if err := s.Save(NewRegistry().Insert(ActorRecord{ID: "a", Args: StartArgs{Worktree: "/repo"}}), NewNamingTable()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := os.Chmod(filepath.Join(dir, "registry.json"), 0o000); err != nil {
		t.Skipf("cannot chmod in this environment: %v", err)
	}
	defer func() { _ = os.Chmod(filepath.Join(dir, "registry.json"), 0o644) }()

	if _, _, err := s.Load(); err == nil {
		t.Fatal("an unreadable registry must error, not read as an empty one")
	}
}

func TestAliveIsFalseForAnExitedChildWithoutCallingWait(t *testing.T) {
	// BR-8. This pins the reaper in the DEFAULT suite. procutil.Alive is
	// `kill -0`, which succeeds for a zombie, so the pre-fix implementation
	// reported an exited-but-unreaped child as running.
	h, err := ExecRunner{}.Start(t.TempDir(), []string{"sh", "-c", "exit 0"}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !h.Alive() {
			return // reaped without anyone calling Wait
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("Alive() stayed true for an exited child -- a zombie is being reported as running")
}

func TestSpawnTellsTheChildWhichTreeItIs(t *testing.T) {
	// BR-9. Without COUCH_TREE and COUCH_STORE_DIR the agent cannot publish a
	// description, and Describe's cache has nothing to cache from.
	env := newTestEnv(t, "/repo")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	got := env.Runner.Child(env.Runner.order[0]).Env
	var tree, store bool
	for _, kv := range got {
		if kv == "COUCH_TREE=/repo" {
			tree = true
		}
		if len(kv) > len("COUCH_STORE_DIR=") && kv[:len("COUCH_STORE_DIR=")] == "COUCH_STORE_DIR=" {
			store = true
		}
	}
	if !tree || !store {
		t.Fatalf("child env = %v; needs COUCH_TREE and COUCH_STORE_DIR", got)
	}
}

func TestDescribePrefersTheAgentsPublishedLineOverTheOperators(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	if err := env.Couch.SetDescription("/repo", "what the operator typed"); err != nil {
		t.Fatalf("SetDescription: %v", err)
	}
	if err := env.Couch.PublishDescription("/repo", "what the agent is doing"); err != nil {
		t.Fatalf("PublishDescription: %v", err)
	}
	if got := env.Couch.Describe("/repo"); got != "what the agent is doing" {
		t.Fatalf("Describe = %q; the agent's own line must win", got)
	}
}

func TestPruneKeepsRecordsWhoseLivenessIsUnknown(t *testing.T) {
	// The smoke-test bug: a probe that could not answer read as "dead", the
	// record was pruned, and a second agent was let onto a tree that already
	// had a running one. Unknown must fail CLOSED.
	env := newTestEnv(t, "/repo")
	env.boundedOne("/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.Proc.SetUnknown(rec.PID)

	if got := env.Couch.Liveness(rec); got != Unknown {
		t.Fatalf("Liveness = %v, want Unknown", got)
	}
	if err := env.Couch.PruneDead(); err != nil {
		t.Fatalf("PruneDead: %v", err)
	}
	if len(env.Couch.Get("/repo")) != 1 {
		t.Fatal("an unknown-liveness record was pruned; the guard now protects nothing")
	}
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err == nil {
		t.Fatal("a second agent was admitted while the incumbent's state was unknown")
	}
}

func TestUnreadableIdentityIsUnknownNotDead(t *testing.T) {
	// A process that exists but whose token cannot be read is not evidence of
	// anything, so it must not be treated as gone.
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.Proc.IdentityErr[rec.PID] = true

	if got := env.Couch.Liveness(rec); got != Unknown {
		t.Fatalf("Liveness = %v, want Unknown", got)
	}
}

func TestStopSignalsEvenWhenLivenessIsUnknown(t *testing.T) {
	// Refusing to signal because we could not confirm liveness would free the
	// tree while leaving a running agent behind -- the hazard Stop closes.
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.Proc.SetUnknown(rec.PID)

	signalled, err := env.Couch.Stop(rec)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !signalled {
		t.Fatal("Stop must attempt a signal when liveness is unknown")
	}
}

func TestAgentPublishedDescriptionResolvesNotJustDisplays(t *testing.T) {
	// BR-23. Display derived from the agent's published line while resolution
	// still only searched the operator's -- half of Done-when 3.
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	if err := env.Couch.PublishDescription("/repo", "reworking the composer gate"); err != nil {
		t.Fatalf("PublishDescription: %v", err)
	}
	got, _, err := env.Couch.ResolveRef("composer")
	if err != nil {
		t.Fatalf("ResolveRef: %v -- the agent's own line must resolve, not only render", err)
	}
	if len(got) != 1 || got[0].ID != rec.ID {
		t.Fatalf("ResolveRef = %+v", got)
	}
}

func TestCoTenantsAreAddressableByActorID(t *testing.T) {
	// BR-24. --same-tree co-tenants share a path and a label, so without an
	// ActorID branch the escape hatch creates a state couch cannot exit.
	env := newTestEnv(t, "/repo")
	first, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	second, _ := env.spawn(t, StartArgs{Worktree: "/repo", SameTree: true})
	if first.ID == second.ID {
		t.Fatal("expected two distinct actors")
	}

	if got, _, err := env.Couch.ResolveRef("/repo"); err != nil || len(got) != 2 {
		t.Fatalf("path ref resolved to %+v (%v), want both co-tenants", got, err)
	}
	got, _, err := env.Couch.ResolveRef(string(second.ID))
	if err != nil {
		t.Fatalf("ResolveRef by id: %v", err)
	}
	if len(got) != 1 || got[0].ID != second.ID {
		t.Fatalf("ResolveRef(%q) = %+v, want exactly that actor", second.ID, got)
	}
	if _, err := env.Couch.Stop(got[0]); err != nil {
		t.Fatalf("Stop by id: %v", err)
	}
	if len(env.Couch.Get("/repo")) != 1 {
		t.Fatal("stopping one co-tenant must leave the other")
	}
}

func TestUnknownRefSaysMissingNotAmbiguous(t *testing.T) {
	env := newTestEnv(t, "/repo")
	_, _, err := env.Couch.ResolveRef("nothing-like-this")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "no actor or tree matches") {
		t.Fatalf("err = %v; absence must not read as ambiguity", err)
	}
}

func TestPersistedCwdIsCanonicalNotAsTyped(t *testing.T) {
	// BR-32. StartArgs is persisted so a revival can reproduce the launch;
	// recording the operator's relative path makes that record meaningless
	// from any other directory.
	env := newTestEnv(t)
	env.cannedTree("/w/kbench", "/w/kbench/competition/arc-agi-3")

	// A path as an operator would plausibly type it: uncanonical, with dot
	// segments and a trailing slash. Passing an already-canonical path here
	// would make this test unable to fail -- which it was, until a deletion
	// check said so.
	typed := "/w/kbench/competition/other/../arc-agi-3/"
	rec, _ := env.spawn(t, StartArgs{Cwd: typed})

	if rec.Args.Cwd == typed {
		t.Fatalf("persisted cwd is the as-typed path %q; replay needs the canonical one", typed)
	}
	if want := NormalizePath(typed); rec.Args.Cwd != want {
		t.Fatalf("persisted cwd = %q, want %q", rec.Args.Cwd, want)
	}

	// And it survives a round trip, which is the point of persisting it.
	reg, _, err := env.Couch.Store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := reg.Get("/w/kbench")
	if len(got) != 1 || got[0].Args.Cwd != rec.Args.Cwd {
		t.Fatalf("round-tripped cwd = %+v", got)
	}
}

// Spawn resumes a tag rather than creating an unnamed session, so a console
// restart lands back on the SAME zellij session instead of pair's fzf picker
// (Decision 11). DecideLaunch with no tag and a detached session present
// returns ActionPick, which inside couch's own pty is an interactive prompt the
// operator never asked for.
func TestSpawnResumesAnOpaqueThreadTag(t *testing.T) {
	env := newTestEnv(t, "/repo")
	if _, _, err := env.Couch.Spawn(StartArgs{Cwd: "/repo"}); err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	got := env.Runner.Ops[0]
	if !strings.Contains(got, "pair resume ") {
		t.Fatalf("argv = %q, want `pair resume <tag>`", got)
	}
	// Layout pinned to layout2 (operator decision 2026-08-22): couch owns
	// terminal switching, so layout3's third pane is the layer couch replaces.
	// This is accepted BECAUSE ParseArgs strips layout flags before the
	// positional guard -- only a stray positional errors.
	if !strings.Contains(got, "--layout2") {
		t.Fatalf("argv does not pin layout2: %q", got)
	}
}

// The tag comes from the WORKTREE ROOT, not the cwd. A spawn inside
// kbench/competition/arc-agi-3/ must resume kbench's tag, because kbench is the
// tree couch keyed the actor on.
func TestSpawnUsesRepoScopeButKeepsRequestedSubdirectoryAsWorkingPath(t *testing.T) {
	env := newTestEnv(t)
	env.Git.replies[GitCall{Dir: "/repo/sub/dir", Args: "rev-parse --show-toplevel"}] = "/repo"

	if _, _, err := env.Couch.Spawn(StartArgs{Cwd: "/repo/sub/dir"}); err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	got := env.Runner.Ops[0]
	if !strings.Contains(got, "pair resume couch-0102030405060708") {
		t.Fatalf("argv = %q, want opaque thread tag", got)
	}
	snapshot, err := env.Couch.Threads.Snapshot()
	if err != nil || len(snapshot.Records) != 1 || snapshot.Records[0].WorkingPath != "/repo/sub/dir" {
		t.Fatalf("thread working path = %+v, %v", snapshot.Records, err)
	}
}

func newIncrementingEntropy() *incrementingEntropy { return &incrementingEntropy{next: 1} }

type incrementingEntropy struct{ next byte }

func (r *incrementingEntropy) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.next
		r.next++
	}
	return len(p), nil
}

// An empty path must be refused, not resolved to wherever the process happens
// to be. `filepath.Abs("")` returns the cwd, so without this the CLI's explicit
// "." default was dead weight -- deletable with every test still green, which is
// two mechanisms for one result and therefore neither pinned.
func TestSpawnRefusesAnEmptyPath(t *testing.T) {
	env := newTestEnv(t)
	if _, _, err := env.Couch.Spawn(StartArgs{}); err == nil {
		t.Fatal("Spawn with no path returned nil error")
	}
}
