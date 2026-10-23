package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type testEnv struct {
	Couch     *Couch
	Runner    *FakeRunner
	Git       *FakeGit
	Proc      *FakeProcOps
	Artifacts *FakeThreadArtifactCollisionChecker
	Dir       string
	Now       time.Time
	// cwd is the thread at the test's working directory, for fixtures that
	// build a startup store around one (startup_proof_test.go).
	cwd ThreadAddress
}

// nameTree gives a worktree an operator label the way the data would actually
// arrive: persisted, then read back by Store.Load. Couch.SetName and
// Couch.SetDescription were the only writers of this table and had no
// production caller, so pair#170 M4 deleted them -- but ResolveRef still
// resolves by label, so the behaviour still needs pinning.
//
// Worth recording: the operator's live registry has "names": {}, and no
// production code path writes one. This naming layer predates the thread
// store, which carries its own Name/Description with a real setter
// (ApplyThreadMetadata). The table below is very likely dead too; that is a
// wider cut than M4's deletion sweep and is left to a follow-up.
func (e *testEnv) nameTree(t *testing.T, tree Worktree, name, description string) {
	t.Helper()
	store := NewStore(e.Dir)
	registry, names, err := store.Load()
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	if name != "" {
		names = names.SetName(tree, name)
	}
	if description != "" {
		names = names.SetDescription(tree, description)
	}
	if err := store.Save(registry, names); err != nil {
		t.Fatalf("save store: %v", err)
	}
	e.Couch.names = names
}

// treeRecords reads the persisted registry directly. Couch.Get was a one-line
// wrapper over it with no production caller (deleted in pair#170 M4), and
// going to the store is the more honest assertion anyway: it proves what
// SURVIVED the write, not what a live Couch happens to hold in memory.
func (e *testEnv) treeRecords(t *testing.T, tree Worktree) []ActorRecord {
	t.Helper()
	registry, _, err := NewStore(e.Dir).Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	return registry.Get(tree)
}

type realDescendantRunner struct {
	marker         string
	acknowledgeErr error
	pty            bool
	wait           time.Duration // how long Acknowledge may wait for the pidfile: the test's budget
}

func (r realDescendantRunner) Start(_ string, argv, env []string) (Handle, error) {
	return ExecRunner{}.Start(filepath.Dir(r.marker), argv, env)
}

func (r realDescendantRunner) StartBlocked(ctx context.Context, _ string, _ []string, _ []string, timeout time.Duration) (BlockedHandle, error) {
	script := `trap '' TERM; sleep 300 & child=$!; printf '%s' "$child" > "$PAIR_TEST_DESCENDANT_PID"; wait "$child"`
	argv := []string{"sh", "-c", script}
	env := []string{
		"PAIR_TEST_RUNNER_HELPER=1",
		"PAIR_TEST_DESCENDANT_PID=" + r.marker,
	}
	var h BlockedHandle
	var err error
	if r.pty {
		h, err = (&PtyRunner{LaunchHelper: os.Args[0]}).StartBlocked(ctx, filepath.Dir(r.marker), argv, env, timeout)
	} else {
		h, err = (ExecRunner{LaunchHelper: os.Args[0]}).StartBlocked(ctx, filepath.Dir(r.marker), argv, env, timeout)
	}
	if err != nil {
		return nil, err
	}
	return &realDescendantBlockedHandle{BlockedHandle: h, marker: r.marker, acknowledgeErr: r.acknowledgeErr, wait: r.wait}, nil
}

type realDescendantBlockedHandle struct {
	BlockedHandle
	marker         string
	acknowledgeErr error
	wait           time.Duration
}

func (h *realDescendantBlockedHandle) Acknowledge() error {
	if err := h.BlockedHandle.Acknowledge(); err != nil {
		return err
	}
	if _, err := waitForPIDFile(h.marker, h.wait); err != nil {
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
	t.Parallel()

	ns := testCouchNamespace(t)
	other := t.TempDir()
	_, err := New(
		ns,
		NewFakeRunner(), NewFakePathOps(nil), NewFakeGit(nil), NewFakeProcOps(),
		NewStore(other), FixedClock{}, NewFixedIDGen("id"), bytes.NewReader(make([]byte, 8)), NoThreadArtifactCollisions{},
	)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("New mismatched store err = %v", err)
	}
}

// newTestEnv wires the whole composition root against fakes, with a fixed
// clock and a scripted id generator so every assertion is deterministic.
func newTestEnv(t *testing.T, trees ...string) *testEnv {
	t.Helper()
	replies := map[GitCall]string{}
	for _, tr := range trees {
		replies[GitCall{Dir: tr, Args: "rev-parse --show-toplevel"}] = tr
		// The repository identity couch now derives locally, in the shape real
		// git answers it: relative in a main checkout.
		replies[GitCall{Dir: tr, Args: "rev-parse --git-common-dir"}] = ".git"
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
	// The default fixture launch completes the server/agent-ready handshake.
	// Failure/split fixtures replace this callback or set explicit observations;
	// a helper's process liveness never synthesizes this evidence.
	r.AfterAcknowledge = func(id string) error {
		child := r.Child(id)
		if !child.alive {
			return nil
		}
		address := ThreadAddress{RepoScope: environmentValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: ThreadTag(environmentValue(child.Env, "COUCH_THREAD_TAG"))}
		artifacts.SetSessionPresence(address, SessionObservation{State: SessionPresent})
		return nil
	}

	c, err := New(ns, r, NewFakePathOps(nil), g, proc, NewStore(dir), FixedClock{T: now}, NewFixedIDGen("ah8d", "b2c1"), newIncrementingEntropy(), artifacts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// No process is visible beyond what a fixture adds: teardown (pair#695)
	// reads the table, and a test that needs rows sets its own.
	c.ProcessTable = func() ([]procutil.Process, error) { return nil, nil }
	c.Identities = &legacyFixtureAllocator{c: c}
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
	e.Git.replies[GitCall{Dir: cwd, Args: "rev-parse --git-common-dir"}] = ".git"
}

func TestSpawnPreparedRefusesRepositoryPreferenceDriftBeforeEffects(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"new configuration", "changed configuration", "transient revision"} {
		t.Run(mode, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			profile := LaunchProfile{Agent: "claude", Argv: []string{"--model", "before"}}
			if mode != "new configuration" {
				seedSwitchRepoArguments(t, env.Couch, "/repo", profile)
			}
			args := StartArgs{Worktree: "/repo", Stack: "claude"}
			prepared, err := env.Couch.PrepareStart(context.Background(), args)
			if err != nil {
				t.Fatal(err)
			}
			seedSwitchRepoArguments(t, env.Couch, "/repo", LaunchProfile{Agent: "claude", Argv: []string{"--model", "after"}})
			if mode == "transient revision" {
				seedSwitchRepoArguments(t, env.Couch, "/repo", profile)
			}
			if _, _, err := env.Couch.SpawnPrepared(context.Background(), args, prepared.Resolution.Fingerprint); !errors.Is(err, ErrStartResolutionChanged) {
				t.Fatalf("repository drift accepted: %v", err)
			}
			assertPreparedStartHadNoEffects(t, env)
		})
	}
}

func assertPreparedStartHadNoEffects(t *testing.T, env *testEnv) {
	t.Helper()
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("changed evidence forked: %q", env.Runner.Ops)
	}
	snapshot, err := env.Couch.Threads.Snapshot()
	if err != nil || len(snapshot.Records) != 0 {
		t.Fatalf("changed evidence allocated: %+v, %v", snapshot.Records, err)
	}
}

func TestSpawnStartsPairAndRecordsTheActor(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	startupBegan := time.Now()
	rec, h, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	startupReturned := time.Now()
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if rec.ID != "couch-ah8d" {
		t.Fatalf("id = %q", rec.ID)
	}
	// couch spawns pair, not claude: pair owns zellij, the layout, and the
	// agent's resume/session-id knowledge.
	if got := env.Runner.Ops[0]; got != "start /repo: pair --couch-session-v1 resume couch-0102030405060708 --layout3" {
		t.Fatalf("Ops[0] = %q", got)
	}
	child := env.Runner.Child(env.Runner.order[0])
	profileRaw, err := launcher.BuildCouchLaunchProfile(
		"couch-0102030405060708", "claude", []string{}, "root", "repo-default",
	)
	if err != nil {
		t.Fatal(err)
	}
	wantEnv := []string{
		"COUCH_TREE=/repo",
		"COUCH_STORE_DIR=" + env.Dir,
		"COUCH_THREAD_SCOPE=816fc349d3faebf8",
		"COUCH_THREAD_TAG=couch-0102030405060708",
		lifetimeLeaseEnvForTest(t, env.Dir, "/repo", ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}),
		"PAIR_RETENTION_BACKGROUND=",
		launcher.CouchLaunchProfileEnv + "=" + strings.TrimSpace(profileRaw),
		"PAIR_USE_REPO_DEFAULT=1",
	}
	wantEnv = append(wantEnv, managedIntentEnvForTest(t, child.Env),
		startupDeadlineEnvForTest(t, child.Env, startupBegan, startupReturned))
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
	t.Parallel()

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

func TestSpawnComposesProductionPairRegistrationBoundary(t *testing.T) {
	for _, freshSlot := range []bool{false, true} {
		name := "ordinary"
		if freshSlot {
			name = "fresh-slot"
		}
		t.Run(name, func(t *testing.T) { testComposedPairRegistrationBoundary(t, freshSlot) })
	}
}

func TestSpawnComposedRegistrationFailureCleansExactSession(t *testing.T) {
	for _, freshSlot := range []bool{false, true} {
		name := "ordinary"
		if freshSlot {
			name = "fresh-slot"
		}
		t.Run(name, func(t *testing.T) { testComposedPairRegistrationBoundaryFailure(t, freshSlot, true) })
	}
}

func TestSpawnComposedRegistrationIsolatesOuterGit(t *testing.T) {
	outer := t.TempDir()
	// Model a review checkout hosted in an outer Couch slot, including the
	// conventional path segment that must not choose the inner fake's workspace.
	checkout := filepath.Join(outer, "worktree", "pair-slot3", "pair")
	if err := os.MkdirAll(checkout, 0700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("git", "init", "--quiet", checkout)
	command.Env = testpair.Environment(t, outer)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("owned outer Git checkout: %v: %s", err, out)
	}
	nested := filepath.Join(checkout, "nested-tmp")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", nested)
	// Each subtest gets its own TempDir base beneath the owned outer checkout.
	// The real launch must keep the fixture's selected repository scope.
	for _, freshSlot := range []bool{false, true} {
		name := "ordinary"
		if freshSlot {
			name = "fresh-slot"
		}
		t.Run(name, func(t *testing.T) { testComposedPairRegistrationBoundary(t, freshSlot) })
	}
}

func testComposedPairRegistrationBoundary(t *testing.T, freshSlot bool) {
	t.Helper()
	testComposedPairRegistrationBoundaryFailure(t, freshSlot, false)
}

// Refusals happen only after Pair has completed its real owned launch. Every
// other probe, including the final absence proof, uses production owner reads.
type registrationCleanupRefusalProbe struct {
	SessionOwnerProber
	armed    bool
	refusals []error
	failure  error
}

func (p *registrationCleanupRefusalProbe) Probe(ctx context.Context, name, data, scope, tag string) (launcher.SessionOwnerObservation, error) {
	if p.armed && len(p.refusals) < 2 {
		p.refusals = append(p.refusals, p.failure)
		return launcher.SessionOwnerObservation{}, p.failure
	}
	return p.SessionOwnerProber.Probe(ctx, name, data, scope, tag)
}

func testComposedPairRegistrationBoundaryFailure(t *testing.T, freshSlot, failed bool) {
	t.Helper()
	testPackageDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	pairHome := filepath.Clean(filepath.Join(testPackageDir, "..", "..", ".."))
	if physical, err := filepath.EvalSymlinks(pairHome); err == nil {
		pairHome = physical
	}
	repo := t.TempDir()
	if physical, err := filepath.EvalSymlinks(repo); err == nil {
		repo = physical
	}
	var slot SlotIdentity
	if freshSlot {
		slot = conventionalSlot(filepath.Join(repo, "pair"), 1)
		if err := os.MkdirAll(slot.PrimaryRoot, 0700); err != nil {
			t.Fatal(err)
		}
		repo = slot.WorktreeRoot
		if err := os.MkdirAll(repo, 0700); err != nil {
			t.Fatal(err)
		}
	}
	home := t.TempDir()
	if physical, err := filepath.EvalSymlinks(home); err == nil {
		home = physical
	}
	pairData := launcher.ResolveDataDir(home, "")
	// Production Couch publishes its selected global root to every Pair child
	// as COUCH_PAIR_DATA_DIR (couchcmd OSRuntime.runtimeRunner), and the
	// launcher requires it to name an existing physical directory. Select this
	// fixture's root the same way so an inherited operator value cannot steer
	// LaunchNative away from the store the collision checker reads.
	if err := os.MkdirAll(pairData, 0o700); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	ready := filepath.Join(t.TempDir(), "zellij-ready")
	release := filepath.Join(t.TempDir(), "zellij-release")
	zellijLog := filepath.Join(t.TempDir(), "zellij.log")

	fixtureBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"zellij", "pair-wrap", "claude"} {
		if err := os.Symlink(fixtureBinary, filepath.Join(binDir, name)); err != nil {
			t.Fatal(err)
		}
	}
	//lint:allow tmp-literal — the terminal fixture appends contract_version_1 and the managed session name; its AF_UNIX pathname must remain short
	socketRoot, err := os.MkdirTemp("/tmp", "pair-registration-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(socketRoot) })
	// The disposable repository has no .git. Stop discovery at its parent
	// so a TMPDIR nested inside the reviewer checkout cannot supply that repo's
	// scope to the real LaunchNative path or workspace discovery.
	fixtureEnv := testpair.Environment(t, home,
		"GIT_CEILING_DIRECTORIES="+filepath.Dir(repo),
		"ZELLIJ_SOCKET_DIR="+socketRoot,
		"PAIR_TEST_COUCH_TERMINAL_FIXTURE=1",
		"PAIR_TEST_ZELLIJ_PANES="+filepath.Join(t.TempDir(), "panes.json"),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"COUCH_PAIR_DATA_DIR="+pairData,
		"PAIR_TEST_ZELLIJ_READY="+ready,
		"PAIR_TEST_ZELLIJ_RELEASE="+release,
		"PAIR_TEST_ZELLIJ_LOG="+zellijLog,
		"PAIR_TEST_COUCH_LAUNCH_NATIVE_SIDECAR=1",
	)
	testpair.ApplyEnvironment(t, home, fixtureEnv...)
	expectedProcessEnv := processEnvironmentForTest()

	ns := testCouchNamespace(t)
	runner := NewFakeRunner()
	proc := NewFakeProcOps()
	scope, err := launcher.ResolveRepoScope(repo)
	if err != nil {
		t.Fatal(err)
	}
	var address ThreadAddress
	checker := NewScopedThreadArtifactCollisionChecker(pairData)
	cleanupRefusal := errors.New("fixture cleanup observation refused")
	launchRefusal := errors.New("fixture acknowledgement refused after native launch")
	ownerProbe := &registrationCleanupRefusalProbe{SessionOwnerProber: launcher.SessionOwnerProbe{}, failure: cleanupRefusal}
	if failed {
		checker.OwnerProbe = ownerProbe
	}
	commonDir := ".git"
	if freshSlot {
		commonDir = slot.RepoIdentity
	}
	couch, err := New(
		ns, runner, NewFakePathOps(nil),
		NewFakeGit(map[GitCall]string{
			{Dir: repo, Args: "rev-parse --show-toplevel"}:  repo,
			{Dir: repo, Args: "rev-parse --git-common-dir"}: commonDir,
		}),
		proc, NewStore(ns.Dir()), FixedClock{T: time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)},
		NewFixedIDGen("composed"), newIncrementingEntropy(), checker,
	)
	if err != nil {
		t.Fatal(err)
	}

	// The scanner is host-wide by name. Derive a fixture-only allocated store
	// number from the unique directory before any terminal launch can occur.
	fixtureID, err := strconv.ParseUint(strings.TrimPrefix(filepath.Base(socketRoot), "pair-registration-"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	couch.Identities = &legacyFixtureAllocator{c: couch, storeID: (uint64(1) << 63) | fixtureID}
	if freshSlot {
		zero, primaryAddress, resting := 0, "pair:0", "main"
		primary := WorkspaceIdentity{SchemaVersion: 2, Repo: slot.Repo, RepoIdentity: slot.RepoIdentity,
			PrimaryRoot: slot.PrimaryRoot, FleetRoot: filepath.Dir(slot.PrimaryRoot),
			EnvironmentRoot: filepath.Dir(slot.PrimaryRoot), WorktreeRoot: slot.PrimaryRoot,
			Kind: "primary", Address: &primaryAddress, Slot: &zero, RestingBranch: &resting}
		// Resolve the selected fake workspace explicitly. A path nested beneath
		// an outer review slot must not fall back to parsing that outer slot.
		current := primary
		current.Kind, current.EnvironmentRoot, current.WorktreeRoot = "slot", slot.EnvironmentRoot, slot.WorktreeRoot
		number, slotAddress, slotResting, head := slot.Number, "pair:1", RestingBranch(slot.Number), strings.Repeat("a", 40)
		current.Slot, current.Address, current.RestingBranch, current.Head = &number, &slotAddress, &slotResting, &head
		current.EnvironmentHost = &EnvironmentHost{Repo: slot.Repo, Slot: slot.Number, RepoIdentity: slot.RepoIdentity, PrimaryRoot: slot.PrimaryRoot, WorktreeRoot: slot.WorktreeRoot}
		couch.Slots = &SlotCatalogFake{Repositories: map[string]SlotRepository{
			slot.PrimaryRoot: {Identity: primary, Slots: []SlotCandidate{{Identity: slot, Verified: true}}},
		}, Workspaces: map[string]WorkspaceIdentity{slot.WorktreeRoot: current}}
		local := newSlotThreadStore(ns, slot)
		couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
			return slotReadyResult(local), nil
		})
	}
	// Neither ordinary new-address launch may depend on fresh readiness.
	couch.FreshRegistration = nil

	type pairMutationReport struct {
		beforeNamespace map[string]string
		afterNamespace  map[string]string
		stderr          string
		code            int
		err             error
	}
	reserved := make(chan ThreadRecord, 1)
	pairReport := make(chan pairMutationReport, 1)
	waitCtx, cancelWait := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelWait()
	runner.BeforeAcknowledge = func(id string) error {
		child := runner.Child(id)
		address = ThreadAddress{RepoScope: environmentValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: ThreadTag(environmentValue(child.Env, "COUCH_THREAD_TAG"))}
		profile, _, err := launcher.ApplyCouchLaunchProfile(launcher.LaunchArgs{ForcedTag: string(address.Tag)}, environmentValue(child.Env, launcher.CouchLaunchProfileEnv))
		if err != nil {
			return err
		}
		if profile.FreshRequired || profile.ResumeRequired || profile.Prompt != "" || environmentValue(child.Env, "COUCH_THREAD_RESUME") != "" {
			return fmt.Errorf("new address received existing-conversation profile: %+v", profile)
		}
		thread, err := couch.Threads.GetThread(address)
		if err != nil {
			return err
		}
		markerRaw, err := os.ReadFile(launcher.NewScopedPaths(pairData, scope, string(address.Tag)).ThreadClaim())
		if err != nil || !strings.Contains(string(markerRaw), `"state":"reserved"`) {
			return fmt.Errorf("pre-ack marker = %q, %v", markerRaw, err)
		}
		reserved <- thread
		beforeNamespace, err := snapshotDirectoryForTest(ns.Dir())
		if err != nil {
			return err
		}
		restoreEnv := applyEnvironmentForTest(child.Env)
		defer restoreEnv()
		oldCwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := os.Chdir(child.Dir); err != nil {
			return err
		}
		defer func() { _ = os.Chdir(oldCwd) }()
		var stdout, stderr bytes.Buffer
		code, launchErr := launcher.LaunchNative(child.Argv[1:], pairHome, &stdout, &stderr)
		afterNamespace, snapshotErr := snapshotDirectoryForTest(ns.Dir())
		report := pairMutationReport{
			beforeNamespace: beforeNamespace, afterNamespace: afterNamespace,
			stderr: stderr.String(), code: code,
			err: errors.Join(launchErr, snapshotErr),
		}
		pairReport <- report
		if report.err != nil || code != 0 {
			return fmt.Errorf("LaunchNative = %d, %v; stderr=%s", code, report.err, report.stderr)
		}
		if failed {
			ownerProbe.armed = true
			return launchRefusal
		}
		return nil
	}

	type spawnResult struct {
		record ActorRecord
		handle Handle
		err    error
	}
	spawned := make(chan spawnResult, 1)
	spawnDone := make(chan struct{})
	go func() {
		defer close(spawnDone)
		if freshSlot {
			result, err := couch.StartFreshSlot(context.Background(), repo, "claude")
			spawned <- spawnResult{record: result.Record, handle: result.Handle, err: err}
		} else {
			record, handle, err := couch.Spawn(StartArgs{Cwd: repo})
			spawned <- spawnResult{record: record, handle: handle, err: err}
		}
	}()
	t.Cleanup(func() {
		// This cleanup owns the composed goroutine on every path, including a
		// Fatal before the normal release/receive below. Joining also guarantees
		// its process-wide environment restoration cannot overlap another test.
		if err := os.WriteFile(release, []byte("release"), 0o600); err != nil {
			t.Errorf("release composed Pair launch: %v", err)
		}
		select {
		case <-spawnDone:
		case <-testwait.After(t):
			t.Error("composed Couch spawn goroutine did not exit during cleanup")
		}
	})

	var preAck ThreadRecord
	select {
	case preAck = <-reserved:
	case <-spawnDone:
		result := <-spawned
		t.Fatalf("launch before acknowledgement: %v", result.err)
	case <-waitCtx.Done():
		t.Fatalf("wait for reserved pre-ack state: %v", waitCtx.Err())
	}
	if len(preAck.Incarnations) != 1 || preAck.Incarnations[0].State != IncarnationCreating || preAck.Incarnations[0].PID != 1000 || preAck.Incarnations[0].Identity != "fake-identity-couch-fake-1" {
		t.Fatalf("pre-ack durable helper = %+v", preAck.Incarnations)
	}
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		select {
		case <-spawnDone:
			result := <-spawned
			var report pairMutationReport
			select {
			case report = <-pairReport:
			default:
			}
			t.Fatalf("spawn exited before zellij readiness: err=%v handle=%v pair=%+v", result.err, result.handle, report)
		case <-waitCtx.Done():
			zellijCalls, _ := os.ReadFile(zellijLog)
			stack := make([]byte, 1<<20)
			stack = stack[:runtime.Stack(stack, true)]
			t.Fatalf("timed out waiting for %s: %v; zellij calls=%q\n%s", ready, waitCtx.Err(), zellijCalls, stack)
		case <-time.After(10 * time.Millisecond):
		}
	}
	markerPath := launcher.NewScopedPaths(pairData, scope, string(address.Tag)).ThreadClaim()
	markerRaw, err := os.ReadFile(markerPath)
	if err != nil || !strings.Contains(string(markerRaw), `"state":"established"`) {
		t.Fatalf("Pair handoff marker = %q, %v", markerRaw, err)
	}
	duringPair, err := couch.Threads.GetThread(address)
	if err != nil || duringPair.Incarnations[0].State != IncarnationCreating {
		t.Fatalf("Couch observed Pair before its own promotion = %+v, %v", duringPair, err)
	}
	child := runner.Child("couch-fake-1")
	wantArgv := []string{"pair", "--couch-session-v1", "resume", string(address.Tag), "--layout3"}
	if !slices.Equal(child.Argv, wantArgv) {
		t.Fatalf("argv = %q, want %q", child.Argv, wantArgv)
	}
	wantOpaque := map[string]string{
		"COUCH_TREE":         repo,
		"COUCH_STORE_DIR":    ns.Dir(),
		"COUCH_THREAD_SCOPE": scope.Key,
		"COUCH_THREAD_TAG":   string(address.Tag),
	}
	for key, want := range wantOpaque {
		if got := environmentValue(child.Env, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}

	// Readiness proves the fixture acquired admission's lock. Wait for that
	// lock to be released before ending its client; no policy file is involved.
	waitFixtureHandoffLease(t, waitCtx, filepath.Join(socketRoot, "contract_version_1", preAck.Incarnations[0].Start.SessionBinding.Name))

	var fixtureServer registrationFixtureServer
	serverRaw, err := os.ReadFile(ready + ".server")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(serverRaw, &fixtureServer); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(release, []byte("release"), 0o600); err != nil {
		t.Fatal(err)
	}
	var report pairMutationReport
	select {
	case report = <-pairReport:
	case <-waitCtx.Done():
		t.Fatalf("wait for Pair launch report: %v", waitCtx.Err())
	}
	if report.err != nil || report.code != 0 {
		t.Fatalf("LaunchNative report = %+v", report)
	}
	// LaunchNative joins its server; the server and wrapper each join their own
	// child before returning. Its exact server generation must now be gone.
	if procutil.Alive(strconv.Itoa(fixtureServer.PID)) && procutil.Identity(strconv.Itoa(fixtureServer.PID)) == fixtureServer.Identity {
		t.Fatalf("owned fixture server survived joined launch: %+v", fixtureServer)
	}
	if !reflect.DeepEqual(report.beforeNamespace, report.afterNamespace) {
		t.Fatalf("Pair mutated Couch namespace\nbefore: %#v\nafter:  %#v", report.beforeNamespace, report.afterNamespace)
	}
	var result spawnResult
	select {
	case result = <-spawned:
	case <-waitCtx.Done():
		t.Fatalf("wait for Couch spawn result: %v", waitCtx.Err())
	}
	if failed {
		if !errors.Is(result.err, launchRefusal) || !errors.Is(result.err, cleanupRefusal) {
			t.Fatalf("failed composed launch did not retain original and cleanup errors: %v", result.err)
		}
		if len(ownerProbe.refusals) != 2 || ownerProbe.refusals[0] != cleanupRefusal || ownerProbe.refusals[1] != cleanupRefusal {
			t.Fatalf("exact repeated cleanup refusal = %v", ownerProbe.refusals)
		}
		if result.handle == nil || result.handle.Alive() {
			t.Fatalf("failed helper not reaped: %v", result.handle)
		}
		retained, err := couch.Threads.GetThread(address)
		if err != nil || len(retained.Incarnations) != 1 || retained.Incarnations[0].State != IncarnationUnknown || retained.Incarnations[0].PID != preAck.Incarnations[0].PID || retained.Incarnations[0].Identity != preAck.Incarnations[0].Identity {
			t.Fatalf("failed spawn lost exact established ownership: %+v, %v", retained, err)
		}
		if _, err := os.Stat(ready + ".server"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("server state outlived reaped fixture: %v", err)
		}
		if afterEnv := processEnvironmentForTest(); !reflect.DeepEqual(expectedProcessEnv, afterEnv) {
			t.Fatalf("failed LaunchNative leaked environment")
		}
		return
	}
	if result.err != nil {
		t.Fatalf("Spawn: %v", result.err)
	}
	if afterEnv := processEnvironmentForTest(); !reflect.DeepEqual(expectedProcessEnv, afterEnv) {
		t.Fatalf("LaunchNative leaked process environment\nbefore: %#v\nafter:  %#v", expectedProcessEnv, afterEnv)
	}
	if result.record.Thread != address {
		t.Fatalf("generated address = %+v, want %+v", result.record.Thread, address)
	}
	after, err := couch.Threads.GetThread(address)
	if err != nil || after.Incarnations[0].State != IncarnationLive || after.Incarnations[0].PID != result.handle.PID() || after.Incarnations[0].Identity != result.handle.Identity() {
		t.Fatalf("post-registration thread = %+v, %v", after, err)
	}
	logRaw, err := os.ReadFile(zellijLog)
	if err != nil || !strings.Contains(string(logRaw), "--new-session-with-layout") {
		t.Fatalf("stateful zellij calls = %q, %v", logRaw, err)
	}
}

func TestAwaitThreadRegistrationNeverPromotesNonEstablishedProductionEvidence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		marker string
		claim  bool
		dir    bool
	}{
		{name: "missing"},
		{name: "reserved", claim: true},
		{name: "malformed", marker: "not-json"},
		{name: "mismatched", marker: `{"schema":1,"scope":"fedcba9876543210","tag":"couch-0123456789abcdef","state":"established"}`},
		{name: "invalid state", marker: `{"schema":1,"scope":"0123456789abcdef","tag":"couch-0123456789abcdef","state":"unknown"}`},
		{name: "unreadable", dir: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, ns := newTestThreadStore(t)
			record := createHelperRecordedThreadForTest(t, store, ns)
			pairData := t.TempDir()
			checker := NewScopedThreadArtifactCollisionChecker(pairData)
			paths := launcher.NewScopedPaths(pairData, launcher.RepoScope{Key: record.Address.RepoScope}, string(record.Address.Tag))
			if tt.claim {
				claim, err := checker.Claim(record.Address)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = claim.Release() })
			} else if tt.marker != "" || tt.dir {
				if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
					t.Fatal(err)
				}
				if tt.dir {
					if err := os.Mkdir(paths.ThreadClaim(), 0o700); err != nil {
						t.Fatal(err)
					}
				} else if err := os.WriteFile(paths.ThreadClaim(), []byte(tt.marker), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			couch := &Couch{Threads: store, Artifacts: checker}
			ctx, cancelBound := context.WithTimeout(context.Background(), 2*time.Second)
			ctx, cancelNow := context.WithCancel(ctx)
			cancelNow()
			err := couch.awaitThreadRegistration(ctx, record.Address)
			cancelBound()
			if err == nil {
				t.Fatal("non-established evidence authorized registration")
			}
			after, getErr := store.GetThread(record.Address)
			if getErr != nil || after.Incarnations[0].State != IncarnationCreating || after.Revision != record.Revision {
				t.Fatalf("non-established evidence changed durable state = %+v, %v", after, getErr)
			}
		})
	}
}

func TestAwaitThreadRegistrationEstablishedEvidenceReachesRealPromotion(t *testing.T) {
	t.Parallel()

	store, ns := newTestThreadStore(t)
	record := createHelperRecordedThreadForTest(t, store, ns)
	pairData := t.TempDir()
	checker := NewScopedThreadArtifactCollisionChecker(pairData)
	claim, err := checker.Claim(record.Address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = claim.Release() })
	if err := launcher.EnsureThreadAddressForPair(pairData, launcher.RepoScope{Key: record.Address.RepoScope}, string(record.Address.Tag), true); err != nil {
		t.Fatal(err)
	}
	couch := &Couch{Threads: store, Artifacts: checker}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := couch.awaitThreadRegistration(ctx, record.Address); err != nil {
		t.Fatal(err)
	}
	promoted, err := store.AdvanceStart(record.Address, record.Revision, StartEvent{Kind: StartRegistered, Nonce: "start-0123456789abcdef"})
	if err != nil || promoted.Incarnations[0].State != IncarnationLive {
		t.Fatalf("promotion = %+v, %v", promoted, err)
	}
}

func createHelperRecordedThreadForTest(t *testing.T, store *ThreadStore, ns CouchNamespace) ThreadRecord {
	t.Helper()
	record := admittedStartRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := store.AdvanceStart(created.Address, created.Revision, StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef",
		Owner: SupervisorOwner{PID: 41, Identity: "owner-token"},
	})
	if err != nil {
		t.Fatal(err)
	}
	helper, err := store.AdvanceStart(claimed.Address, claimed.Revision, StartEvent{
		Kind: StartHelperRecorded, Nonce: "start-0123456789abcdef",
		Helper: ProcessIdentity{PID: 42, Identity: "helper-token"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return helper
}

func writeExecutableForTest(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
}

func waitForFileForTest(t *testing.T, ctx context.Context, path string) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s: %v", path, ctx.Err())
		case <-ticker.C:
		}
	}
}

func applyEnvironmentForTest(env []string) func() {
	prior := append([]string(nil), os.Environ()...)
	for _, item := range env {
		key, value, _ := strings.Cut(item, "=")
		_ = os.Setenv(key, value)
	}
	return func() {
		os.Clearenv()
		for _, item := range prior {
			key, value, _ := strings.Cut(item, "=")
			_ = os.Setenv(key, value)
		}
	}
}

func environmentValue(env []string, key string) string {
	for _, item := range env {
		name, value, _ := strings.Cut(item, "=")
		if name == key {
			return value
		}
	}
	return ""
}

func processEnvironmentForTest() map[string]string {
	result := make(map[string]string)
	for _, item := range os.Environ() {
		key, value, _ := strings.Cut(item, "=")
		result[key] = value
	}
	return result
}

func snapshotDirectoryForTest(root string) (map[string]string, error) {
	snapshot := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			snapshot[rel] = "dir:" + info.Mode().String()
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			snapshot[rel] = "symlink:" + target
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[rel] = "file:" + info.Mode().String() + ":" + string(raw)
		return nil
	})
	return snapshot, err
}

func TestSpawnPostAcknowledgementFailuresNeverLeaveWorkspaceWriter(t *testing.T) {
	t.Parallel()

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
						_, hookErr = env.Couch.Threads.updateExistingThread(got, current.Revision, func(next *ThreadRecord) error {
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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

	deadline := time.Now().Add(testwait.Budget(t))
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
	case <-testwait.After(t):
		t.Fatal("Spawn did not return after quiescence proof")
	}
}

type delayedQuiescenceHandle struct {
	mu        sync.Mutex
	done      chan struct{}
	waits     int
	kills     int
	alive     bool
	closeOnce sync.Once
}

func newDelayedQuiescenceHandle() *delayedQuiescenceHandle {
	return &delayedQuiescenceHandle{done: make(chan struct{}), alive: true}
}

func (h *delayedQuiescenceHandle) ID() string       { return "delayed" }
func (h *delayedQuiescenceHandle) PID() int         { return 4242 }
func (h *delayedQuiescenceHandle) Identity() string { return "start-token" }
func (h *delayedQuiescenceHandle) Alive() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.alive
}
func (h *delayedQuiescenceHandle) Signal(sig os.Signal) error {
	if sig != os.Kill {
		return nil
	}
	h.mu.Lock()
	h.kills++
	kills := h.kills
	h.mu.Unlock()
	if kills >= 3 {
		h.closeOnce.Do(func() {
			h.mu.Lock()
			h.alive = false
			h.mu.Unlock()
			close(h.done)
		})
	}
	return nil
}

// after is the cleanup's clock for this child (#473): a wait gives up at once
// while the child lives, and once a KILL has ended it the wait lasts until its
// reap is delivered, however late the scheduler runs the waiter. The bound
// only turns a cleanup that never takes the reap into a failure, not a hang.
func (h *delayedQuiescenceHandle) after(time.Duration) <-chan time.Time {
	if h.Alive() {
		fired := make(chan time.Time, 1)
		fired <- time.Now()
		return fired
	}
	return time.After(10 * time.Second)
}

func (h *delayedQuiescenceHandle) Exited() <-chan struct{} { return h.done }

func (h *delayedQuiescenceHandle) Wait() int {
	h.mu.Lock()
	h.waits++
	h.mu.Unlock()
	<-h.done
	return -1
}

func TestPostAckHandleRetriesReuseOneWaiterUntilReap(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	env.Couch.postAckRetryDelay = time.Millisecond
	h := newDelayedQuiescenceHandle()
	env.Couch.after = h.after
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}

	record := validThreadRecord(t)
	record.Address = address
	record.StartingPath, record.WorkingPath = "/repo", "/repo"
	if _, err := env.Couch.Threads.CreateThread(record); err != nil {
		t.Fatal(err)
	}

	if err := env.Couch.quiescePostAckStart(address, h, StartSpawn); err == nil {
		t.Fatal("transient handle cleanup errors were not retained")
	}
	h.mu.Lock()
	waits, kills := h.waits, h.kills
	h.mu.Unlock()
	if waits != 1 || kills != 3 || h.Alive() {
		t.Fatalf("cleanup ownership = waits:%d kills:%d alive:%v", waits, kills, h.Alive())
	}
}

// The helper half of cleanup is the SAME work whether or not the start owns its
// session: the helper belongs to this start either way. Only the session half
// is ownership-gated (pair#230), so a non-owning cleanup must still drive the
// child all the way to reaped -- and must ask for no quiesce at all.
func TestPostAckCleanupEndsItsHelperWithoutOwningTheSession(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	env.Couch.postAckRetryDelay = time.Millisecond
	h := newDelayedQuiescenceHandle()
	env.Couch.after = h.after
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	env.Artifacts.SetDetachedSession(address, "pair-couch-0102030405060708")

	if err := env.Couch.quiescePostAckStart(address, h, StartWarmReattach); err == nil {
		t.Fatal("transient handle cleanup errors were not retained")
	}
	h.mu.Lock()
	waits, kills := h.waits, h.kills
	h.mu.Unlock()
	if waits != 1 || kills != 3 || h.Alive() {
		t.Fatalf("cleanup ownership = waits:%d kills:%d alive:%v, want the same helper work as an owning cleanup", waits, kills, h.Alive())
	}
	if quiesced := env.Artifacts.Quiesces(); len(quiesced) != 0 {
		t.Fatalf("a non-owning cleanup quiesced %v", quiesced)
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
					marker := filepath.Join(t.TempDir(), "descendant.pid")
					runner := realDescendantRunner{marker: marker, pty: runnerMode.pty, wait: testwait.Budget(t)}
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
								_, hookErr = env.Couch.Threads.updateExistingThread(got, current.Revision, func(next *ThreadRecord) error {
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
					descendantPID, err = waitForPIDFile(marker, testwait.Budget(t))
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

func TestExplicitReconcileDeletesDeadUnregisteredHelperByExactNonce(t *testing.T) {
	t.Parallel()

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
	c, err := New(ns, NewFakeRunner(), NewFakePathOps(nil), NewFakeGit(nil), NewFakeProcOps(), NewStore(ns.Dir()), FixedClock{}, NewFixedIDGen("id"), newIncrementingEntropy(), artifacts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.reconcileInterruptedStarts(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetThread(record.Address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("dead unregistered start remained: %v", err)
	}
}

func TestExplicitReconcilePromotesEstablishedSurvivingHelper(t *testing.T) {
	t.Parallel()

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
	c, err := New(ns, NewFakeRunner(), NewFakePathOps(nil), NewFakeGit(nil), proc, NewStore(ns.Dir()), FixedClock{}, NewFixedIDGen("id"), newIncrementingEntropy(), artifacts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.reconcileInterruptedStarts(); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetThread(record.Address)
	if err != nil || got.Revision != created.Revision+1 || got.Incarnations[0].State != IncarnationLive || got.Incarnations[0].Start != nil {
		t.Fatalf("promoted thread = %+v, %v", got, err)
	}
}

func TestSpawnStartsInASubdirectoryButRegistersTheTree(t *testing.T) {
	t.Parallel()

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

func TestSnapshotIsOnDiskWhileTheChildIsStillAlive(t *testing.T) {
	t.Parallel()

	// Couch owns the child for its lifetime, so if Save happened after Wait a
	// second shell running `couch --list` would see nothing for the whole
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
	t.Parallel()

	env := newTestEnv(t, "/repo")
	env.Runner.FailNextStart(errors.New("boom"))
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err == nil {
		t.Fatal("expected a start failure")
	}
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"}); err != nil {
		t.Fatalf("tree still held after a failed spawn: %v", err)
	}
}

func TestSpawnFailureDoesNotCommitLaunchPreferences(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t, "/repo")
	env.Couch.RootAgent = "codex"
	seedSwitchRepoArguments(t, env.Couch, "/repo", LaunchProfile{Agent: "codex", Argv: []string{"--search"}})
	before, err := env.Couch.Threads.previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	env.Runner.FailNextStart(errors.New("boom"))
	if _, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo", ExtraArgs: []string{"--model", "unadopted"}}); err == nil {
		t.Fatal("expected start failure")
	}
	after, err := env.Couch.Threads.previewRepoPreferences()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("failed start changed repository settings: %+v %v", after, err)
	}
}

func childEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

// Asserted through Liveness rather than the deleted IsLive wrapper, which
// folded Unknown into false -- so this now pins the exact state, not just
// "not live" (pair#170 M4).
func TestLivenessRejectsARecycledPID(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	if got := env.Couch.Liveness(rec); got != Live {
		t.Fatalf("a running actor reads as %v, want Live", got)
	}
	// Same PID, different process: the kernel start token differs.
	env.Proc.Set(rec.PID, "some-other-process")
	if got := env.Couch.Liveness(rec); got != Dead {
		t.Fatalf("a recycled PID reads as %v, want Dead", got)
	}
	env.Proc.Kill(rec.PID)
	if got := env.Couch.Liveness(rec); got != Dead {
		t.Fatalf("a dead PID reads as %v, want Dead", got)
	}
}

func TestResolveRefFindsActorsByOperatorName(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.nameTree(t, "/repo", "refactor thing", "")
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
	t.Parallel()

	env := newTestEnv(t, "/w/pair")
	env.spawn(t, StartArgs{Worktree: "/w/pair"})

	got := env.Couch.LookupTrees("pair")
	if len(got) != 1 || got[0] != "/w/pair" {
		t.Fatalf("LookupTrees(pair) = %v, want [/w/pair]", got)
	}
}

func TestNameAndDescriptionChangeMidSession(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	env.nameTree(t, "/repo", "first", "")
	env.nameTree(t, "/repo", "second", "")
	if got, _, err := env.Couch.ResolveRef("second"); err != nil || len(got) != 1 {
		t.Fatalf("rename did not take effect: %v %v", got, err)
	}
	if got, _, err := env.Couch.ResolveRef("first"); err == nil && len(got) > 0 {
		t.Fatal("the old name must stop resolving")
	}
	env.nameTree(t, "/repo", "", "reworking the composer gate")
	if got, _, err := env.Couch.ResolveRef("composer"); err != nil || len(got) != 1 {
		t.Fatalf("description did not take effect: %v %v", got, err)
	}
}

func TestNameSurvivesActorReplacement(t *testing.T) {
	t.Parallel()

	// A real lifecycle: spawn, name, the child exits, the actor is forgotten,
	// a new one is spawned. The name must still resolve, because it hangs off
	// the tree rather than the incarnation.
	env := newTestEnv(t, "/repo")
	first, h := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.nameTree(t, "/repo", "long lived", "")

	env.Runner.SetExited(h.ID(), 0)
	env.Proc.Kill(first.PID)
	env.Artifacts.SetSessionPresence(first.Thread, SessionObservation{State: SessionAbsent})
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

// This test asserted the OPPOSITE until pair#170 M4, and the reversal is the
// deletion sweep's most visible behavioural consequence, so it is inverted
// rather than removed.
//
// A live actor used to refuse its own tree: admission enforced fleet capacity,
// and one thread per path was the common bound. couch-lite has no capacity to
// enforce -- the path is a start attribute, not the thread's identity, and
// running several threads in one directory is a thing the operator does on
// purpose (brain-style repositories were the motivating case). What still keeps
// two agents off one workspace is Pair's own address claim, not couch's.
// #181 REVERSES this again, and the operator made the call: several threads at
// one PATH without separate worktrees is confusing, and the accumulation it
// produced -- six threads in one repo -- was the concrete cost. couch keeps one
// thread per path until per-repo policy is modelled.
//
// pair#363 widens it again, to one PRIMARY thread per repository: a start in a
// subdirectory of a repository whose :0 exists returns to it or refuses
// (TestSpawnInSubdirectoryOfALivePrimaryRefuses). Slots are how a repository
// gets a second agent. Co-tenants from a store that predates the rule still
// resolve, which is what TestCoTenantsAreAddressableByActorID exercises.
func TestASecondThreadAtOnePathIsRefused(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	first, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	_, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err == nil {
		t.Fatal("a second thread started at a path that already had one")
	}
	// The refusal has to be navigable: which thread holds the path, and what to
	// do about it. A bare "refused" leaves the operator with no next step.
	// Every named next step must be one the operator can actually take from
	// where this fires. The first version named `couch <path>` (the command
	// that just refused) and `couch --show` as a way to retire (it is
	// read-only) -- advice that fails at the moment someone is already stuck.
	//
	// The held thread is LIVE, and a live row offers no reboot (pair#363), so
	// "Tab → reboot" would name an action the row does not offer; it says to
	// restart inside Pair instead.
	for _, want := range []string{string(first.Thread.Tag), "ctrl-space", "Alt+Shift+N", "couch --show"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal %q does not mention %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "reboot") {
		t.Fatalf("refusal %q names reboot, which a live row does not offer", err)
	}
}

// pair#363: one primary per repository. The start form in a subdirectory of
// a repository whose :0 is live refuses through SpawnPrepared, names the
// thread it found, and starts nothing.
func TestSpawnInSubdirectoryOfALivePrimaryRefuses(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	env.cannedTree("/repo", "/repo/sub")
	first, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	before, err := env.Couch.Threads.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	opsBefore := len(env.Runner.Ops)

	args := StartArgs{Worktree: "/repo", Cwd: "/repo/sub"}
	prepared, err := env.Couch.PrepareStart(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = env.Couch.SpawnPrepared(context.Background(), args, prepared.Resolution.Fingerprint)
	if err == nil {
		t.Fatal("a second primary started in a subdirectory of a live one")
	}
	for _, want := range []string{"one primary slot per repository", "repo", string(first.Thread.Tag), "ctrl-space"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal %q does not mention %q", err, want)
		}
	}
	after, err := env.Couch.Threads.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Records) != len(before.Records) || len(env.Runner.Ops) != opsBefore {
		t.Fatalf("the refusal had effects: records %d -> %d, ops %q", len(before.Records), len(after.Records), env.Runner.Ops[opsBefore:])
	}
}

// The refusal's "start fresh" line names only what the held thread's row
// offers (lessons: refusal-names-unoffered-action): reboot where reboot is
// permitted, which is exactly what the switcher offers it on; a live :0 gets
// no reboot (pair#363 Spec), so it is told to restart inside Pair instead.
// Derived over every state the occupancy guard can hold.
func TestPrimaryOccupiedFreshStepNamesOnlyWhatTheRowOffers(t *testing.T) {
	t.Parallel()

	for _, state := range AllThreadStates() {
		for _, reason := range append(AllThreadReasons(), "") {
			if (state == ThreadUnusable) != (reason != "") {
				continue
			}
			row := ActionableThreadSummary{Address: ThreadAddress{RepoScope: "scope", Tag: "couch-0000000000000001"}, WorkingPath: "/repo", State: state, Reason: reason}
			held, ok := ScopeHoldsUsableThread([]ActionableThreadSummary{row}, "scope")
			if !ok {
				continue
			}
			step := primaryFreshStep(held)
			if names := strings.Contains(step, "reboot"); names != RebootableState(state, reason) {
				t.Errorf("%s/%s: fresh step %q names reboot = %v, reboot permitted = %v", state, reason, step, names, RebootableState(state, reason))
			}
			if step == "" {
				t.Errorf("%s/%s: no way to start fresh", state, reason)
			}
		}
	}
}

// pair#363 resolved ambiguity 1, operator-confirmed: couch starts only inside
// a Git repository. A directory git does not recognise refuses at the first
// seam that asks (Resolve's `rev-parse --show-toplevel`), before any record or
// child exists. So no non-Git row can exist, and the action table has no
// non-Git kind; supporting such starts would be its own issue.
func TestStartInANonGitDirectoryRefuses(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t) // no canned trees: git answers nothing for /plain
	args := StartArgs{Cwd: "/plain"}
	for name, start := range map[string]func() error{
		"start form": func() error { _, err := env.Couch.PrepareStart(context.Background(), args); return err },
		"explicit startup": func() error {
			args.Action = StartCreate
			_, err := env.Couch.StartInteractive(context.Background(), args)
			return err
		},
	} {
		err := start()
		if err == nil || !strings.Contains(err.Error(), "resolve worktree") {
			t.Fatalf("%s in a non-Git directory = %v, want Resolve's refusal", name, err)
		}
		snapshot, snapErr := env.Couch.Threads.Snapshot()
		if snapErr != nil || len(snapshot.Records) != 0 || len(env.Runner.Ops) != 0 {
			t.Fatalf("%s refusal had effects: records %+v, ops %q, %v", name, snapshot.Records, env.Runner.Ops, snapErr)
		}
	}
	if !slices.Contains(env.Git.Ops, "/plain: rev-parse --show-toplevel") {
		t.Fatalf("git calls = %q, want the refusal to come from asking git for the toplevel", env.Git.Ops)
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
	if len(env.treeRecords(t, "/repo")) != 0 {
		t.Fatal("the record should be gone after Stop")
	}
}

func TestStopOnADeadActorForgetsWithoutSignalling(t *testing.T) {
	t.Parallel()

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

func TestReplayPreservesSameTreeExactly(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	// BR-8. This pins the reaper in the DEFAULT suite. procutil.Alive is
	// `kill -0`, which succeeds for a zombie, so the pre-fix implementation
	// reported an exited-but-unreaped child as running.
	h, err := ExecRunner{}.Start(t.TempDir(), []string{"sh", "-c", "exit 0"}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	deadline := time.Now().Add(testwait.Budget(t))
	for time.Now().Before(deadline) {
		if !h.Alive() {
			return // reaped without anyone calling Wait
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("Alive() stayed true for an exited child -- a zombie is being reported as running")
}

func TestSpawnTellsTheChildWhichTreeItIs(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	env := newTestEnv(t, "/repo")
	env.spawn(t, StartArgs{Worktree: "/repo"})
	env.nameTree(t, "/repo", "", "what the operator typed")
	if err := env.Couch.PublishDescription("/repo", "what the agent is doing"); err != nil {
		t.Fatalf("PublishDescription: %v", err)
	}
	if got := env.Couch.Describe("/repo"); got != "what the agent is doing" {
		t.Fatalf("Describe = %q; the agent's own line must win", got)
	}
}

func TestPruneKeepsRecordsWhoseLivenessIsUnknown(t *testing.T) {
	t.Parallel()

	// The smoke-test bug: a probe that could not answer read as "dead", the
	// record was pruned, and a second agent was let onto a tree that already
	// had a running one. Unknown must fail CLOSED.
	env := newTestEnv(t, "/repo")
	rec, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	env.Proc.SetUnknown(rec.PID)

	if got := env.Couch.Liveness(rec); got != Unknown {
		t.Fatalf("Liveness = %v, want Unknown", got)
	}
	pruned, removed := env.Couch.withoutDead(env.Couch.reg)
	if removed != 0 || len(pruned.Records()) != 1 {
		t.Fatal("an unknown-liveness record was pruned; the guard now protects nothing")
	}
	// The second half of this test used to assert that ADMISSION refused a
	// second agent at the same path while the incumbent was unknown. Capacity
	// went with admission (pair#170 M4): couch-lite admits multiple threads per
	// path deliberately. What survives -- and is what the test is named for --
	// is that pruning does not DELETE a record whose liveness is unknown, which
	// is the fail-closed half and is asserted above.
}

func TestUnreadableIdentityIsUnknownNotDead(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	// BR-24. Co-tenants share a tree and a label, so without an ActorID branch
	// the escape hatch creates a state couch cannot exit.
	//
	// They differ by PATH. Since pair#363 a repository has one primary, so
	// couch no longer creates the second one; a store that predates the rule
	// still holds such pairs, and the state this test protects is unchanged --
	// both resolve from the tree ref, and only an ActorID separates them. The
	// second is created past the occupancy guard (no rows), as the older couch
	// that wrote such a store did.
	env := newTestEnv(t, "/repo")
	env.cannedTree("/repo", "/repo/sub")
	first, _ := env.spawn(t, StartArgs{Worktree: "/repo"})
	resolution, err := env.Couch.resolveStartResolution(context.Background(), StartArgs{Worktree: "/repo", Cwd: "/repo/sub"})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := env.Couch.spawnResolved(context.Background(), resolution, nil)
	if err != nil {
		t.Fatalf("seed the legacy co-tenant: %v", err)
	}
	env.Proc.Set(second.PID, second.Identity)
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
	if len(env.treeRecords(t, "/repo")) != 1 {
		t.Fatal("stopping one co-tenant must leave the other")
	}
}

func TestUnknownRefSaysMissingNotAmbiguous(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	env := newTestEnv(t, "/repo")
	if _, _, err := env.Couch.Spawn(StartArgs{Cwd: "/repo"}); err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	got := env.Runner.Ops[0]
	if !strings.Contains(got, "pair --couch-session-v1 resume ") {
		t.Fatalf("argv = %q, want `pair resume <tag>`", got)
	}
	// A cold start sends couch's OWN layout, and this env is a default couch,
	// so that is --layout3 after #242. What has not changed is why a
	// layout flag is accepted here at all -- ParseArgs strips layout flags
	// before the positional guard, so only a stray positional errors.
	if !strings.Contains(got, "--layout3") {
		t.Fatalf("a default couch did not send its layout: %q", got)
	}
}

// The tag comes from the WORKTREE ROOT, not the cwd. A spawn inside
// kbench/competition/arc-agi-3/ must resume kbench's tag, because kbench is the
// tree couch keyed the actor on.
func TestSpawnUsesRepoScopeButKeepsRequestedSubdirectoryAsWorkingPath(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	env.Git.replies[GitCall{Dir: "/repo/sub/dir", Args: "rev-parse --show-toplevel"}] = "/repo"
	env.Git.replies[GitCall{Dir: "/repo/sub/dir", Args: "rev-parse --git-common-dir"}] = ".git"

	if _, _, err := env.Couch.Spawn(StartArgs{Cwd: "/repo/sub/dir"}); err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	got := env.Runner.Ops[0]
	if !strings.Contains(got, "pair --couch-session-v1 resume couch-0102030405060708") {
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
	t.Parallel()

	env := newTestEnv(t)
	if _, _, err := env.Couch.Spawn(StartArgs{}); err == nil {
		t.Fatal("Spawn with no path returned nil error")
	}
}

// A fresh hub receives a new name and leaves its orphaned predecessor alone.
func TestStartupBesideAnOrphanedPrimaryUsesANewName(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	first, h := env.spawn(t, StartArgs{Worktree: "/repo"})
	// The launcher dies with couch; the session's server survives, orphaned.
	env.Runner.SetExited(h.ID(), 0)
	env.Proc.Kill(first.PID)
	if err := env.Couch.Forget("/repo", first.ID); err != nil {
		t.Fatalf("Forget: %v", err)
	}
	server := launcher.SessionServerIdentity{PID: 9090, Identity: "t", Session: "📁repo-1"}
	env.Artifacts.SetSessionPresence(first.Thread, SessionObservation{State: SessionOrphaned, Orphan: &server})
	opsBefore := len(env.Runner.Ops)
	retained, err := env.Couch.Threads.GetThread(first.Thread)
	if err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(retained)
	if err != nil {
		t.Fatal(err)
	}

	result, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Handle == nil || result.Record.Thread == first.Thread {
		t.Fatal("hub did not start under a fresh name beside remnant")
	}
	if len(env.Runner.Ops) != opsBefore+2 {
		t.Fatalf("startup had unexpected effects: %q", env.Runner.Ops[opsBefore:])
	}
	retained, err = env.Couch.Threads.GetThread(first.Thread)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(retained)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("fresh hub changed orphan record")
	}
	if len(env.Proc.Signals[server.PID]) != 0 {
		t.Fatal("fresh hub signalled orphan owner")
	}

}
