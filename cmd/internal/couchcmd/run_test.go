package couchcmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// testRT builds the domain over fakes. There is deliberately NO production
// branch: a test that can reach ExecGit resolves paths against whatever
// checkout it happens to run in, so it asserts on the developer's directory
// layout rather than on couch. One such test passed here and failed in a
// pristine worktree of the same commit.
type testRT struct {
	dir        string
	namespace  couchcore.CouchNamespace
	runner     *couchcore.FakeRunner
	proc       *couchcore.FakeProcOps
	git        *couchcore.FakeGit
	supervisor *fakeSupervisor
	policy     *couchcore.FakePolicyResolver
	artifacts  *couchcore.FakeThreadArtifactCollisionChecker
	env        map[string]string
	// ids is shared across invocations. Minting a fresh generator per
	// NewCouch restarts the sequence, so two starts both produce couch-ah8d
	// and no CLI test can hold two distinguishable actors.
	ids              couchcore.IDGen
	currentRepoScope string
	agentDefaults    map[string]launcher.AgentDefault
}

func (t testRT) Getenv(key string) string { return t.env[key] }
func (t testRT) StoreDir() string         { return t.dir }
func (t testRT) CurrentRepoScope() (string, error) {
	if t.currentRepoScope == "" {
		return "", fmt.Errorf("test runtime has no current repository scope")
	}
	return t.currentRepoScope, nil
}
func (t testRT) ResolveNamespace() (couchcore.CouchNamespace, error) {
	return t.namespace, nil
}
func (t testRT) AcquireSupervisor(couchcore.CouchNamespace) (io.Closer, error) {
	if t.supervisor.err != nil {
		return nil, t.supervisor.err
	}
	t.supervisor.acquired++
	return fakeSupervisorLease{state: t.supervisor}, nil
}

type fakeSupervisor struct {
	acquired int
	released int
	err      error
}

type fakeSupervisorLease struct{ state *fakeSupervisor }

func (l fakeSupervisorLease) Close() error {
	l.state.released++
	return nil
}

func (t testRT) NewCouch() (*couchcore.Couch, error) {
	return t.NewCouchWith(t.runner, t.namespace)
}

// NewCouchWith ignores the caller's runner and keeps the fake so typed
// orchestration tests drive the full dispatch without spawning a process.
func (t testRT) NewCouchWith(couchcore.Runner, couchcore.CouchNamespace) (*couchcore.Couch, error) {
	c, err := couchcore.New(
		t.namespace, t.runner, couchcore.NewFakePathOps(nil), t.git, t.proc,
		couchcore.NewStore(t.dir), couchcore.FixedClock{T: time.Unix(1, 0)}, t.ids, t.policy,
		rand.Reader, t.artifacts,
	)
	if err != nil {
		return nil, err
	}
	c.RootAgent = t.env["PAIR_AGENT"]
	c.RepoAgentDefault = func(repoRoot, agent string) (couchcore.LaunchProfile, bool, error) {
		value, ok := t.agentDefaults[repoRoot+"\x00"+agent]
		return couchcore.LaunchProfile{Agent: value.Agent, Argv: append([]string(nil), value.Args...)}, ok, nil
	}
	return c, nil
}

// newRT wires a Runtime whose git answers for the given trees.
func newRT(t *testing.T, trees ...string) testRT {
	t.Helper()
	runner := couchcore.NewFakeRunner()
	// Couch owns Handle.Wait for the child's lifetime -- right in
	// production, and a hang rather than a failure against a fake child that
	// never finishes.
	runner.AutoExit(0)
	replies := map[couchcore.GitCall]string{}
	for _, tr := range trees {
		replies[couchcore.GitCall{Dir: tr, Args: "rev-parse --show-toplevel"}] = tr
	}
	ns, err := couchcore.ResolveCouchNamespace(t.TempDir(), "/unused")
	if err != nil {
		t.Fatalf("ResolveCouchNamespace: %v", err)
	}
	var currentScope string
	if len(trees) > 0 {
		scope, err := launcher.ResolveRepoScope(trees[0])
		if err != nil {
			t.Fatal(err)
		}
		currentScope = scope.Key
	}
	artifacts := couchcore.NewFakeThreadArtifactCollisionChecker()
	artifacts.AutoEstablish(true)
	return testRT{
		dir:              ns.Dir(),
		namespace:        ns,
		runner:           runner,
		proc:             couchcore.NewFakeProcOps(),
		git:              couchcore.NewFakeGit(replies),
		supervisor:       &fakeSupervisor{},
		policy:           couchcore.NewFakePolicyResolver(),
		artifacts:        artifacts,
		env:              map[string]string{},
		ids:              couchcore.NewFixedIDGen("ah8d", "b2c1", "c3d2", "e4f5"),
		currentRepoScope: currentScope,
		agentDefaults:    map[string]launcher.AgentDefault{},
	}
}

func TestStartComposesRootAgentAndMatchingRepoDefaultThroughSharedLauncherProfile(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.env["PAIR_AGENT"] = "codex"
	rt.agentDefaults["/repo\x00codex"] = launcher.AgentDefault{Agent: "codex", Args: []string{"--sandbox", "workspace-write"}}
	rt.boundedOne("/repo")

	if _, stderr, code := runLaunchRT(rt, "/repo", ""); code != 0 {
		t.Fatalf("start: code=%d stderr=%q", code, stderr)
	}
	child := rt.runner.Child("couch-fake-1")
	profileRaw := envValue(child.Env, launcher.CouchLaunchProfileEnv)
	if len(child.Argv) < 3 {
		t.Fatalf("child argv = %q", child.Argv)
	}
	parsed, err := launcher.ParseArgs([]string{"resume", child.Argv[2], "--layout2"})
	if err != nil {
		t.Fatal(err)
	}
	profileArgs, source, err := launcher.ApplyCouchLaunchProfile(parsed, profileRaw)
	if err != nil {
		t.Fatalf("profile env %q: %v", profileRaw, err)
	}
	if profileArgs.Agent != "codex" || !reflect.DeepEqual(profileArgs.AgentArgs, []string{"--sandbox", "workspace-write"}) || source != "repo-default" {
		t.Fatalf("resolved child profile = %+v source=%q", profileArgs, source)
	}
	if envValue(child.Env, "PAIR_USE_REPO_DEFAULT") != "1" {
		t.Fatalf("child env = %q; repo-default provenance marker missing", child.Env)
	}
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func seedThread(t *testing.T, rt testRT, path string) couchcore.ThreadRecord {
	t.Helper()
	scope, err := launcher.ResolveRepoScope(path)
	if err != nil {
		t.Fatal(err)
	}
	return seedThreadAtAddress(t, rt, scope.Key, "couch-0102030405060708", path)
}

func seedThreadAtAddress(t *testing.T, rt testRT, scope, tag, path string) couchcore.ThreadRecord {
	t.Helper()
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	record := couchcore.ThreadRecord{
		SchemaVersion: couchcore.ThreadSchemaVersion,
		Address: couchcore.ThreadAddress{
			RepoScope: scope,
			Tag:       couchcore.ThreadTag(tag),
		},
		StartingPath: path,
		WorkingPath:  path,
		CreatedAt:    time.Unix(1, 0).UTC(),
		Revision:     1,
	}
	created, err := c.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

func seedVerifiedPark(t *testing.T, rt testRT, path string) couchcore.ThreadRecord {
	t.Helper()
	rt.boundedOne(path)
	policy, err := rt.policy.ResolvePolicy(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := launcher.ResolveRepoScope(path)
	if err != nil {
		t.Fatal(err)
	}
	profile := couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	record := couchcore.ThreadRecord{
		SchemaVersion: couchcore.ThreadSchemaVersion,
		Address:       couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-0102030405060708"},
		StartingPath:  path, WorkingPath: path, CreatedAt: time.Unix(1, 0).UTC(), Revision: 1,
		Incarnations: []couchcore.ThreadIncarnation{{
			PID: 42, Identity: "pair-helper", State: couchcore.IncarnationLive,
			Policy: &policy, LaunchProfile: &profile,
		}},
		LatestLaunchProfile: &profile,
	}
	created, err := c.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	identity := couchcore.ParkIdentity{
		Nonce: "park-resume-cli", Address: created.Address, PID: 42, ProcessIdentity: "pair-helper",
	}
	begun, err := c.Threads.BeginPark(created.Address, created.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	parked, err := c.Threads.FinalizePark(created.Address, begun.Revision, identity, 1, time.Unix(2, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	return parked
}

func TestStartAcquiresAndReleasesSupervisorLease(t *testing.T) {
	rt := newRT(t, "/repo")
	if _, errw, code := runLaunchRT(rt, "/repo", ""); code != 0 {
		t.Fatalf("start: code=%d stderr=%q", code, errw)
	}
	if rt.supervisor.acquired != 1 || rt.supervisor.released != 1 {
		t.Fatalf("supervisor acquire/release = %d/%d, want 1/1", rt.supervisor.acquired, rt.supervisor.released)
	}
}

func TestResumeAcquiresAndReleasesSupervisorLease(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.supervisor.err = fmt.Errorf("resume reached singleton acquisition")
	_, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "resume", Args: map[string]string{"ref": "couch-0102030405060708"}})
	if code == 0 || !strings.Contains(errw, "resume reached singleton acquisition") {
		t.Fatalf("resume: code=%d stderr=%q", code, errw)
	}
}

func TestResumeRunsAsTheNewLiveOwner(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.boundedOne("/repo")
	parked := seedVerifiedPark(t, rt, "/repo")
	rt.artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	rt.runner.AfterAcknowledge = func(string) error {
		rt.artifacts.SetPairSession(parked.Address, "pair-"+string(parked.Address.Tag), true)
		return nil
	}

	out, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "resume", Args: map[string]string{"ref": string(parked.Address.Tag)}})
	if code != 0 {
		t.Fatalf("resume: code=%d stdout=%q stderr=%q", code, out, errw)
	}
	if rt.supervisor.acquired != 1 || rt.supervisor.released != 1 {
		t.Fatalf("supervisor acquire/release = %d/%d, want 1/1", rt.supervisor.acquired, rt.supervisor.released)
	}
	if len(rt.runner.Ops) == 0 || !strings.Contains(rt.runner.Ops[0], "pair resume "+string(parked.Address.Tag)+" --layout2") {
		t.Fatalf("resume child operations = %v", rt.runner.Ops)
	}
}

// seedDetachedThread leaves a record in the shape an alt+d detach produces:
// no incarnation, NO verified park, a saved launch profile, and a live zellij
// session with no client.
func seedDetachedThread(t *testing.T, rt testRT, path string) couchcore.ThreadRecord {
	t.Helper()
	rt.boundedOne(path)
	scope, err := launcher.ResolveRepoScope(path)
	if err != nil {
		t.Fatal(err)
	}
	profile := couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	created, err := c.Threads.CreateThread(couchcore.ThreadRecord{
		SchemaVersion: couchcore.ThreadSchemaVersion,
		Address:       couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-0102030405060708"},
		StartingPath:  path, WorkingPath: path, CreatedAt: time.Unix(1, 0).UTC(), Revision: 1,
		LatestLaunchProfile: &profile,
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := c.Threads.UpdateExistingThread(created.Address, created.Revision, func(next *couchcore.ThreadRecord) error {
		next.Reservation = false
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return updated
}

// The M3 acceptance case, across a RESTART: a couch that detached a thread and
// went away, then `couch` again in that tree. Driven through production
// interactive routing all the way to initial Console attach -- not below it,
// because reducer support is not user reachability.
func TestInteractiveLaunchReattachesUniqueDetachedRoot(t *testing.T) {
	rt := newRT(t, "/repo")
	detached := seedDetachedThread(t, rt, "/repo")
	rt.artifacts.SetNativeBinding(detached.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	// The surviving session IS the resume authority for a detached thread.
	rt.artifacts.SetDetachedSession(detached.Address, "pair-"+string(detached.Address.Tag))
	rt.runner = couchcore.NewFakeRunner()
	rt.runner.AfterAcknowledge = func(string) error {
		rt.artifacts.SetPairSession(detached.Address, "pair-"+string(detached.Address.Tag), true)
		return nil
	}
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	var attached couchcore.StartResult
	finish := func(console *couchtty.Console, c *couchcore.Couch, start couchcore.StartResult, _ io.Writer) int {
		wireResolver(console, c)
		if err := dispatchInitialAttach(console, start); err != nil {
			t.Fatalf("initial attach: %v", err)
		}
		attached = start
		return 0
	}
	op, _ := Resolve("start")
	var stdout, stderr bytes.Buffer
	code := runTypedOperationWithConsole(op, map[string]string{}, map[string]string{"path": "/repo"}, true, slave, slave, slave, &stdout, &stderr, rt, finish)
	if code != 0 {
		t.Fatalf("interactive launch: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	defer rt.runner.SetExited(attached.Handle.ID(), 0)

	if attached.Record.Thread != detached.Address {
		t.Fatalf("interactive root = %+v, want the detached thread %+v", attached.Record.Thread, detached.Address)
	}
	if len(rt.runner.Ops) == 0 || !strings.Contains(rt.runner.Ops[0], "pair resume "+string(detached.Address.Tag)+" --layout2") {
		t.Fatalf("child operations = %v, want the detached thread reattached", rt.runner.Ops)
	}
}

// Without the surviving session there is no resume authority, so startup must
// create a NEW thread rather than reattach one it cannot prove.
func TestInteractiveLaunchStartsNewWhenNoSessionSurvives(t *testing.T) {
	rt := newRT(t, "/repo")
	stale := seedDetachedThread(t, rt, "/repo")
	rt.artifacts.SetNativeBinding(stale.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	// Deliberately NO SetDetachedSession: the session did not survive.
	rt.runner = couchcore.NewFakeRunner()
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	var attached couchcore.StartResult
	finish := func(console *couchtty.Console, c *couchcore.Couch, start couchcore.StartResult, _ io.Writer) int {
		wireResolver(console, c)
		if err := dispatchInitialAttach(console, start); err != nil {
			t.Fatalf("initial attach: %v", err)
		}
		attached = start
		return 0
	}
	op, _ := Resolve("start")
	var stdout, stderr bytes.Buffer
	code := runTypedOperationWithConsole(op, map[string]string{}, map[string]string{"path": "/repo"}, true, slave, slave, slave, &stdout, &stderr, rt, finish)
	if code != 0 {
		t.Fatalf("interactive launch: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	defer rt.runner.SetExited(attached.Handle.ID(), 0)

	if attached.Record.Thread == stale.Address {
		t.Fatalf("startup reattached %+v with no surviving session", stale.Address)
	}
}

func TestInteractiveLaunchResumesUniqueParkedRoot(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.boundedOne("/repo")
	parked := seedVerifiedPark(t, rt, "/repo")
	rt.artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	rt.runner = couchcore.NewFakeRunner()
	rt.runner.AfterAcknowledge = func(string) error {
		rt.artifacts.SetPairSession(parked.Address, "pair-"+string(parked.Address.Tag), true)
		return nil
	}
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	var attached couchcore.StartResult
	finish := func(console *couchtty.Console, c *couchcore.Couch, start couchcore.StartResult, _ io.Writer) int {
		wireResolver(console, c)
		if err := dispatchInitialAttach(console, start); err != nil {
			t.Fatalf("initial attach: %v", err)
		}
		attached = start
		return 0
	}
	op, _ := Resolve("start")
	var stdout, stderr bytes.Buffer
	code := runTypedOperationWithConsole(op, map[string]string{}, map[string]string{"path": "/repo"}, true, slave, slave, slave, &stdout, &stderr, rt, finish)
	if code != 0 {
		t.Fatalf("interactive launch: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	defer rt.runner.SetExited(attached.Handle.ID(), 0)
	if attached.Record.Thread != parked.Address {
		t.Fatalf("interactive root = %+v, want %+v", attached.Record.Thread, parked.Address)
	}
	if len(rt.runner.Ops) == 0 || !strings.Contains(rt.runner.Ops[0], "pair resume "+string(parked.Address.Tag)+" --layout2") {
		t.Fatalf("interactive child operations = %v, want resumed parked root", rt.runner.Ops)
	}
	child := rt.runner.Child(attached.Handle.ID())
	if !strings.Contains(strings.Join(child.Env, "\n"), "native-root-1") {
		t.Fatalf("attached resume env = %v, want saved native root", child.Env)
	}
}

func TestDirectStoreOperationDoesNotAcquireSupervisorLease(t *testing.T) {
	rt := newRT(t)
	if _, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "list"}); code != 0 {
		t.Fatalf("list: code=%d stderr=%q", code, errw)
	}
	if rt.supervisor.acquired != 0 || rt.supervisor.released != 0 {
		t.Fatalf("direct-store list touched supervisor lease: %d/%d", rt.supervisor.acquired, rt.supervisor.released)
	}
}

func TestHeldSupervisorRefusesBeforeStartingActor(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.supervisor.err = fmt.Errorf("namespace is supervised by pid 42")
	_, errw, code := runLaunchRT(rt, "/repo", "")
	if code == 0 || !strings.Contains(errw, "pid 42") {
		t.Fatalf("start: code=%d stderr=%q", code, errw)
	}
	if len(rt.runner.Ops) != 0 {
		t.Fatalf("refused start ran child operations: %v", rt.runner.Ops)
	}
}

// markLive marks every registered actor's pid as running, which is what a real
// spawned child would be.
func (rt testRT) markLive(t *testing.T) {
	t.Helper()
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatalf("NewCouch: %v", err)
	}
	for _, r := range c.List() {
		rt.proc.Set(r.PID, r.Identity)
	}
}

func (rt testRT) boundedOne(path string) {
	rt.policy.SetDefault(couchcore.PolicyResult{
		PolicyVersion: 1,
		PolicyDigest:  strings.Repeat("a", 64),
		RepoIdentity:  "repo",
		AdmissionKey:  path,
		Capacity:      couchcore.PolicyCapacity{Kind: couchcore.CapacityBounded, Limit: 1},
		OnCapacity:    couchcore.CapacityReject,
	}, nil)
}

func runTypedRT(rt testRT, call couchcore.OperationCall) (string, string, int) {
	var out, errw bytes.Buffer
	op, ok := Resolve(call.Name)
	if !ok {
		fmt.Fprintf(&errw, "unknown typed operation %q\n", call.Name)
		return out.String(), errw.String(), 2
	}
	args := call.Args
	if args == nil {
		args = map[string]string{}
	}
	if op.Name == "publish-description" {
		args = cloneStringMap(args)
		args["repo-scope"] = rt.Getenv("COUCH_THREAD_SCOPE")
		args["tag"] = rt.Getenv("COUCH_THREAD_TAG")
	}
	code := runTypedOperation(op, args, nil, false, nil, nil, strings.NewReader(""), &out, &errw, rt)
	return out.String(), errw.String(), code
}

func runLaunchRT(rt testRT, path, agent string) (string, string, int) {
	var out, errw bytes.Buffer
	prepare := map[string]string{"path": path}
	if prepare["path"] == "" {
		prepare["path"] = "."
	}
	if agent != "" {
		prepare["agent"] = agent
	}
	op, _ := Resolve("start")
	code := runTypedOperation(op, map[string]string{}, prepare, false, nil, nil, strings.NewReader(""), &out, &errw, rt)
	return out.String(), errw.String(), code
}

func runPublicRT(rt testRT, args ...string) (string, string, int) {
	var out, errw bytes.Buffer
	code := RunWithRuntime(args, strings.NewReader(""), &out, &errw, rt)
	return out.String(), errw.String(), code
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func TestDispatchTableIsIdenticalToTheDeclaredOperationSet(t *testing.T) {
	var reachable []string
	for name := range Dispatch() {
		reachable = append(reachable, name)
	}
	sort.Strings(reachable)
	if declared := couchcore.OperationNames(); !reflect.DeepEqual(reachable, declared) {
		t.Fatalf("CLI reaches %v, declared %v", reachable, declared)
	}
}

func TestEveryOperationHasASummaryAndDescribedArgs(t *testing.T) {
	for _, op := range couchcore.Operations() {
		if op.Summary == "" {
			t.Errorf("%s: empty summary -- the advisor needs it to choose", op.Name)
		}
		for _, a := range op.Args {
			if a.Summary == "" {
				t.Errorf("%s: arg %q has no summary", op.Name, a.Name)
			}
		}
		if op.Effect == couchcore.EffectUnknown || op.Confirmation == couchcore.ConfirmUnknown || op.Result == couchcore.ResultUnknown {
			t.Errorf("%s: incomplete declaration", op.Name)
		}
	}
}

func TestOperationArityMatchesExpectation(t *testing.T) {
	// Declared in the test rather than read from the operation itself, so
	// this cannot degrade into asserting X == X.
	want := map[string]int{"prepare-start": 2, "start": 1, "list": 0, "show": 2, "stop": 1, "name": 3, "describe": 3, "publish-description": 3, "switch": 2, "attach": 2, "park": 4, "detach": 3, "resume": 3}
	for _, op := range couchcore.Operations() {
		if got := len(op.Args); got != want[op.Name] {
			t.Errorf("%s has %d args, want %d", op.Name, got, want[op.Name])
		}
	}
}

func TestPublishDescriptionUsesCompositeThreadEnvironment(t *testing.T) {
	rt := newRT(t)
	record := couchcore.ThreadRecord{
		SchemaVersion: couchcore.ThreadSchemaVersion,
		Address: couchcore.ThreadAddress{
			RepoScope: "816fc349d3faebf8",
			Tag:       "couch-0102030405060708",
		},
		StartingPath: "/repo/task",
		WorkingPath:  "/repo/task",
		CreatedAt:    time.Unix(1, 0).UTC(),
		Revision:     1,
	}
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	created, err := c.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	rt.env["COUCH_THREAD_SCOPE"] = created.Address.RepoScope
	rt.env["COUCH_THREAD_TAG"] = string(created.Address.Tag)

	if _, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "publish-description", Args: map[string]string{"description": "agent summary"}}); code != 0 {
		t.Fatalf("publish-description: code=%d stderr=%q", code, errw)
	}
	got, err := c.Threads.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if got.PublishedSummary != "agent summary" || got.Description != "" {
		t.Fatalf("published thread = %+v", got)
	}
	if _, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "publish-description", Args: map[string]string{"description": ""}}); code != 0 {
		t.Fatalf("clear publish-description: code=%d stderr=%q", code, errw)
	}
	got, err = c.Threads.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if got.PublishedSummary != "" {
		t.Fatalf("empty CLI summary did not clear field: %+v", got)
	}
}

func TestListOnEmptyThreadStore(t *testing.T) {
	out, errw, code := runTypedRT(newRT(t), couchcore.OperationCall{Name: "list"})
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errw)
	}
	if !strings.Contains(out, "no threads") {
		t.Fatalf("out = %q", out)
	}
}

func TestPublicListAndShowUseDiagnosticFlags(t *testing.T) {
	rt := newRT(t, "/repo")
	seedThread(t, rt, "/repo")
	var out, errw bytes.Buffer
	if code := RunWithRuntime([]string{"--list"}, strings.NewReader(""), &out, &errw, rt); code != 0 {
		t.Fatalf("--list: code=%d stderr=%q", code, errw.String())
	}
	if !strings.Contains(out.String(), "/repo") {
		t.Fatalf("--list output = %q", out.String())
	}
	out.Reset()
	errw.Reset()
	if code := RunWithRuntime([]string{"--show", "couch-0102030405060708"}, strings.NewReader(""), &out, &errw, rt); code != 0 {
		t.Fatalf("--show: code=%d stderr=%q", code, errw.String())
	}
	if !strings.Contains(out.String(), "address:") {
		t.Fatalf("--show output = %q", out.String())
	}
}

func TestPublicNonTerminalLaunchRefusesBeforeEffects(t *testing.T) {
	rt := newRT(t, "/repo")
	var out, errw bytes.Buffer
	code := RunWithRuntime([]string{"/repo"}, strings.NewReader(""), &out, &errw, rt)
	if code == 0 || !strings.Contains(errw.String(), "requires terminal") {
		t.Fatalf("launch: code=%d stderr=%q", code, errw.String())
	}
	if rt.supervisor.acquired != 0 || len(rt.runner.Ops) != 0 {
		t.Fatalf("non-terminal launch performed effects: supervisor=%d runner=%v", rt.supervisor.acquired, rt.runner.Ops)
	}
}

func TestPublicInternalPublishDescriptionIsTheOnlyHiddenOperation(t *testing.T) {
	rt := newRT(t)
	record := couchcore.ThreadRecord{
		SchemaVersion: couchcore.ThreadSchemaVersion,
		Address:       couchcore.ThreadAddress{RepoScope: "scope", Tag: "couch-0102030405060708"},
		StartingPath:  "/repo", WorkingPath: "/repo", CreatedAt: time.Unix(1, 0).UTC(), Revision: 1,
	}
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Threads.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	rt.env["COUCH_THREAD_SCOPE"] = "scope"
	rt.env["COUCH_THREAD_TAG"] = "couch-0102030405060708"
	var out, errw bytes.Buffer
	if code := RunWithRuntime([]string{"--internal", "publish-description", "working"}, strings.NewReader(""), &out, &errw, rt); code != 0 {
		t.Fatalf("internal publish: code=%d stderr=%q", code, errw.String())
	}
	got, err := c.Threads.GetThread(record.Address)
	if err != nil || got.PublishedSummary != "working" {
		t.Fatalf("published record = %+v, %v", got, err)
	}
	if code := RunWithRuntime([]string{"--internal", "list"}, strings.NewReader(""), &out, &errw, rt); code != 2 {
		t.Fatalf("internal list code=%d, want 2", code)
	}
}

func TestUnknownPublicOptionIsNonZero(t *testing.T) {
	out, errw, code := runPublicRT(newRT(t), "--frobnicate")
	if code == 0 {
		t.Fatal("unknown operation must be non-zero")
	}
	if !strings.Contains(errw, "unknown option") {
		t.Fatalf("stderr = %q", errw)
	}
	_ = out
}

func TestMissingRequiredArgumentIsRejectedBeforeAnyWork(t *testing.T) {
	_, errw, code := runPublicRT(newRT(t), "--show")
	if code == 0 {
		t.Fatal("a missing required argument must be non-zero")
	}
	if !strings.Contains(errw, "requires exactly one non-empty reference") {
		t.Fatalf("stderr = %q", errw)
	}
}

func TestPublicHelpListsOnlyPublicSurface(t *testing.T) {
	out, _, code := runPublicRT(newRT(t), "--help")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"couch [path]", "couch --list", "couch --show", "couch --help"} {
		if !strings.Contains(out, want) {
			t.Errorf("help omits %q", want)
		}
	}
	for _, hidden := range []string{"start", "park", "resume", "publish-description", "--internal"} {
		if strings.Contains(out, hidden) {
			t.Errorf("help exposes %q", hidden)
		}
	}
}

func TestReadmeDoesNotAdvertiseRemovedAdmissionFlags(t *testing.T) {
	raw, err := os.ReadFile("../../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	removed := "--same" + "-tree"
	if strings.Contains(string(raw), removed) {
		t.Fatalf("README still advertises removed flag %s", removed)
	}
}

func TestReadmeDoesNotAdvertiseOwnerRequiredStopAsExternalCommand(t *testing.T) {
	raw, err := os.ReadFile("../../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "couch stop <ref>") { // obsolete-argv-rejection
		t.Fatal("README advertises stop as a second-process command before #147 owner routing exists")
	}
}

func TestBindArgsAcceptsPrepareStartFlagsAndPositionals(t *testing.T) {
	var start couchcore.Operation
	for _, op := range couchcore.Operations() {
		if op.Name == "prepare-start" {
			start = op
		}
	}
	got, err := bindArgs(start, []string{"../pair", "--agent=claude"})
	if err != nil {
		t.Fatalf("bindArgs: %v", err)
	}
	if got["path"] != "../pair" || got["agent"] != "claude" {
		t.Fatalf("bound = %v", got)
	}
}

func TestBindArgsRejectsMissingOrEmptyValueBearingFlag(t *testing.T) {
	var start couchcore.Operation
	for _, op := range couchcore.Operations() {
		if op.Name == "prepare-start" {
			start = op
		}
	}
	for _, argv := range [][]string{{"--agent"}, {"--agent="}} {
		if _, err := bindArgs(start, argv); err == nil {
			t.Fatalf("bindArgs(%q) accepted agent flag without a value", argv)
		}
	}
}

func TestCLIRejectsMissingOrEmptyExplicitAgentBeforeSpawn(t *testing.T) {
	for _, argv := range [][]string{{"--agent"}, {"--agent="}} {
		t.Run(strings.Join(argv, "_"), func(t *testing.T) {
			rt := newRT(t, "/repo")
			rt.boundedOne("/repo")
			_, stderr, code := runPublicRT(rt, argv...)
			if code == 0 || !strings.Contains(stderr, "unknown option") {
				t.Fatalf("runPublicRT(%q): code=%d stderr=%q", argv, code, stderr)
			}
			if len(rt.runner.Ops) != 0 {
				t.Fatalf("runPublicRT(%q) reached runner operations %q", argv, rt.runner.Ops)
			}
		})
	}
}

func TestListShowsANamedTreeWithNoAgent(t *testing.T) {
	// The forgetting case: a tree with no running client has no actor, but it
	// is exactly the thread the operator loses track of. It must be a visible
	// row, not filtered out.
	//
	// This fixture carries no verified park, so it is the DETACHED-shaped case:
	// its agent may well still be running behind a live zellij session, and the
	// row must not claim otherwise. The parked wording is covered by
	// TestRenderThreadRowsDistinguishesParkedFromDetached.
	rt := newRT(t, "/repo")
	seedThread(t, rt, "/repo")
	if _, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "name", Args: map[string]string{"ref": "/repo", "name": "the pair tree"}}); code != 0 {
		t.Fatalf("name failed: %s", errw)
	}
	out, _, code := runTypedRT(rt, couchcore.OperationCall{Name: "list"})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "the pair tree") {
		t.Fatalf("out = %q; a named tree must appear even with no agent", out)
	}
	if !strings.Contains(out, "no client attached") {
		t.Fatalf("out = %q; the absence of a client must be stated", out)
	}
}

// A thread with no incarnation is not necessarily idle. Parked means the
// session was torn down and the agent is gone; detached means only the client
// left. Reporting both as "no agent running" contradicts the switcher, which
// offers the detached row for reattach.
func TestRenderThreadRowsDistinguishesParkedFromDetached(t *testing.T) {
	address := couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0001020304050607"}
	for _, test := range []struct {
		name   string
		parked bool
		want   string
		unwant string
	}{
		{name: "parked", parked: true, want: "no agent running", unwant: "may still be running"},
		{name: "detached", parked: false, want: "may still be running", unwant: "(parked"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			renderThreadRows(&buf, []couchcore.ThreadSummary{{
				Address: address, WorkingPath: "/repo", Name: "compiler", Parked: test.parked,
			}}, false)
			out := buf.String()
			if !strings.Contains(out, test.want) {
				t.Fatalf("out = %q, want it to mention %q", out, test.want)
			}
			if strings.Contains(out, test.unwant) {
				t.Fatalf("out = %q, must not mention %q", out, test.unwant)
			}
		})
	}
}

func TestCLIEmptyNameClearsHumanThreadName(t *testing.T) {
	rt := newRT(t, "/repo")
	created := seedThread(t, rt, "/repo")
	if _, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "name", Args: map[string]string{"ref": string(created.Address.Tag), "name": "compiler"}}); code != 0 {
		t.Fatalf("set name: code=%d stderr=%q", code, errw)
	}
	if _, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "name", Args: map[string]string{"ref": string(created.Address.Tag), "name": ""}}); code != 0 {
		t.Fatalf("clear name: code=%d stderr=%q", code, errw)
	}
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Threads.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "" {
		t.Fatalf("empty CLI name did not clear field: %+v", got)
	}
}

func TestShowResolvesANameToItsTreePath(t *testing.T) {
	rt := newRT(t, "/repo")
	created := seedThread(t, rt, "/repo")
	if _, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "name", Args: map[string]string{"ref": "/repo", "name": "pairtree"}}); code != 0 {
		t.Fatalf("name failed: %s", errw)
	}
	out, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "show", Args: map[string]string{"ref": "pairtree"}})
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errw)
	}
	if !strings.Contains(out, "/repo") {
		t.Fatalf("out = %q; show must print the tree path", out)
	}
	if !strings.Contains(out, string(created.Address.Tag)) {
		t.Fatalf("out = %q; show must retain the immutable thread tag", out)
	}
}

func TestCLICompositeReferencesDeriveCurrentRepositoryScope(t *testing.T) {
	rt := newRT(t, "/repo")
	localScope, err := launcher.ResolveRepoScope("/repo")
	if err != nil {
		t.Fatal(err)
	}
	otherScope, err := launcher.ResolveRepoScope("/other")
	if err != nil {
		t.Fatal(err)
	}
	const repeatedTag = "couch-0102030405060708"
	local := seedThreadAtAddress(t, rt, localScope.Key, repeatedTag, "/repo")
	other := seedThreadAtAddress(t, rt, otherScope.Key, repeatedTag, "/other")

	if _, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "name", Args: map[string]string{"ref": repeatedTag, "name": "local thread"}}); code != 0 {
		t.Fatalf("name: code=%d stderr=%q", code, errw)
	}
	if _, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "describe", Args: map[string]string{"ref": repeatedTag, "description": "local description"}}); code != 0 {
		t.Fatalf("describe: code=%d stderr=%q", code, errw)
	}
	out, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "show", Args: map[string]string{"ref": repeatedTag}})
	if code != 0 || !strings.Contains(out, "/repo") || strings.Contains(out, "/other") {
		t.Fatalf("show: code=%d out=%q stderr=%q", code, out, errw)
	}

	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	gotLocal, err := c.Threads.GetThread(local.Address)
	if err != nil {
		t.Fatal(err)
	}
	gotOther, err := c.Threads.GetThread(other.Address)
	if err != nil {
		t.Fatal(err)
	}
	if gotLocal.Name != "local thread" || gotLocal.Description != "local description" {
		t.Fatalf("local metadata = name %q description %q", gotLocal.Name, gotLocal.Description)
	}
	if gotOther.Name != "" || gotOther.Description != "" {
		t.Fatalf("other repository thread was mutated: %+v", gotOther)
	}
}

func TestCurrentRepoScopeUsesGitRootFromSubdirectory(t *testing.T) {
	git := couchcore.NewFakeGit(map[couchcore.GitCall]string{
		{Dir: "/repo/subdir", Args: "rev-parse --show-toplevel"}: "/repo",
	})
	got, err := resolveCurrentRepoScope("/repo/subdir", git, couchcore.NewFakePathOps(nil))
	if err != nil {
		t.Fatal(err)
	}
	want, err := launcher.ResolveRepoScope("/repo")
	if err != nil {
		t.Fatal(err)
	}
	if got != want.Key {
		t.Fatalf("scope = %q, want Git-root scope %q", got, want.Key)
	}
}

func TestRenderedOutputHasNoANSIWhenNotATerminal(t *testing.T) {
	// A bytes.Buffer is not a terminal, so dimming must be suppressed --
	// otherwise piped or captured output carries escape codes.
	rt := newRT(t, "/repo")
	seedThread(t, rt, "/repo")
	_, _, _ = runTypedRT(rt, couchcore.OperationCall{Name: "name", Args: map[string]string{"ref": "/repo", "name": "plain"}})
	out, _, _ := runTypedRT(rt, couchcore.OperationCall{Name: "list"})
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("ANSI leaked into non-terminal output: %q", out)
	}
}

func TestRenderThreadsIsNameFirstAndKeepsSamePathThreadsDistinct(t *testing.T) {
	rows := []couchcore.ThreadSummary{
		{Address: couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000001"}, WorkingPath: "/repo", Name: "compiler", PublishedSummary: "agent work"},
		{Address: couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000002"}, WorkingPath: "/repo"},
	}
	var out bytes.Buffer
	renderThreads(&out, rows)
	text := out.String()
	if !strings.Contains(text, "compiler") || strings.Contains(strings.Split(text, "\n")[0], "couch-0000000000000001") {
		t.Fatalf("named row leads with opaque id: %q", text)
	}
	if !strings.Contains(text, "couch-0000000000000002") || strings.Count(text, "/repo") != 2 {
		t.Fatalf("same-path thread rows collapsed: %q", text)
	}
}

// TestCLIAcceptsExactlyTheDeclaredOperations replaces an audit that compared
// two views of one source and therefore could not fail.
//
// A reviewer added an undeclared `couch nuke` branch ahead of the table lookup
// The in-process registry is closed independently from the public argv parser.
func TestTypedRegistryResolvesExactlyDeclaredOperations(t *testing.T) {
	declared := map[string]bool{}
	for _, name := range couchcore.OperationNames() {
		declared[name] = true
		if _, ok := Resolve(name); !ok {
			t.Errorf("declared operation %q does not resolve", name)
		}
	}
	for _, name := range []string{"nuke", "kill", "restart", "attach", "switch", "ls", "run", "exec"} {
		if declared[name] {
			continue
		}
		if _, ok := Resolve(name); ok {
			t.Errorf("typed registry resolves undeclared operation %q", name)
		}
	}
}

func TestStartRendersTheRefusalWithThePolicyShapedOffer(t *testing.T) {
	// Done-when 2's rendering had no reachable test before the Runtime seam.
	rt := newRT(t, "/repo")
	rt.boundedOne("/repo")
	if out, errw, code := runLaunchRT(rt, "/repo", ""); code != 0 {
		t.Fatalf("first start: code=%d out=%q err=%q", code, out, errw)
	}
	// Mark the child live so the guard has something real to refuse for.
	rt.markLive(t)
	_, errw, code := runLaunchRT(rt, "/repo", "")
	if code == 0 {
		t.Fatal("a second start on an occupied tree must fail")
	}
	for _, want := range []string{"at capacity 1", `admission key "/repo"`, "couch --list"} {
		if !strings.Contains(errw, want) {
			t.Errorf("refusal missing %q; got %q", want, errw)
		}
	}
}

func TestProvisionWorktreeRefusalNames153WithoutInventingAPath(t *testing.T) {
	var out bytes.Buffer
	renderError(&out, &couchcore.CapacityExceededError{
		RepoIdentity: "web", AdmissionKey: "/repo", Limit: 1,
		Action: couchcore.CapacityProvisionWorktree,
	})
	got := out.String()
	if !strings.Contains(got, "pair#153") || !strings.Contains(got, "no path was created") {
		t.Fatalf("provision refusal = %q", got)
	}
	if strings.Contains(got, "couch start ") { // obsolete-argv-rejection
		t.Fatalf("provision refusal invented a runnable path: %q", got)
	}
}

func TestExternalStopRefusesUntilOwnerRoutingExists(t *testing.T) {
	rt := newRT(t, "/repo")
	if _, errw, code := runLaunchRT(rt, "/repo", ""); code != 0 {
		t.Fatalf("start: %s", errw)
	}
	rt.markLive(t)
	_, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "stop", Args: map[string]string{"ref": "/repo"}})
	if code == 0 || !strings.Contains(errw, "routing requires #147") {
		t.Fatalf("stop: code=%d err=%q", code, errw)
	}
}

func TestTypedMetadataOperationsPreserveOptionalDescription(t *testing.T) {
	rt := newRT(t, "/repo")
	seedThread(t, rt, "/repo")
	if _, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "name", Args: map[string]string{"ref": "/repo", "name": "thing"}}); code != 0 {
		t.Fatalf("name: %s", errw)
	}
	if _, errw, code := runTypedRT(rt, couchcore.OperationCall{Name: "describe", Args: map[string]string{"ref": "thing", "description": "what it is doing"}}); code != 0 {
		t.Fatalf("describe: %s", errw)
	}
	out, _, _ := runTypedRT(rt, couchcore.OperationCall{Name: "describe", Args: map[string]string{"ref": "thing"}})
	if !strings.Contains(out, "what it is doing") {
		t.Fatalf("out = %q", out)
	}
}

// The milestone's central wiring, pinned WITHOUT a terminal.
//
// At M2's boundary, disabling the console left the whole suite green (BR-24),
// and the first attempt to fix that used a real pty -- which skips in the
// sandbox this issue documents as its own environment, so the mutation stayed
// green anyway. That is the third time a gated-only pin has been written on this
// issue. The decision is pure now, so it is pinned unconditionally.
func TestWantsConsole(t *testing.T) {
	cases := []struct {
		name        string
		op          string
		hasTerminal bool
		want        bool
	}{
		{"start on a terminal", "start", true, true},
		{"start with no terminal", "start", false, false},
		{"resume on a terminal", "resume", true, true},
		{"resume with no terminal", "resume", false, false},
		{"a read-only operation", "list", true, false},
		{"stop never takes the terminal", "stop", true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := WantsConsole(c.op, c.hasTerminal); got != c.want {
				t.Fatalf("WantsConsole(%q, %v) = %v, want %v", c.op, c.hasTerminal, got, c.want)
			}
		})
	}
}

// The plumbing half, still unconditional: with no terminal there must be no
// console and the stdio runner.
func TestConsoleRunnerDeclinesWithoutATerminal(t *testing.T) {
	console, runner := consoleRunner("start", strings.NewReader(""), &bytes.Buffer{})
	if console != nil {
		t.Fatal("a console was built with no terminal")
	}
	if _, ok := runner.(couchcore.ExecRunner); !ok {
		t.Fatalf("runner = %T, want couchcore.ExecRunner", runner)
	}
}

// Internal launch with no path defaults to ".", matching bare Couch launch.
//
// It asserts the SPAWN, not the ArgSpec: the first version checked that `path`
// was not Required, which stayed green with the default deleted (BR-24).
func TestStartDefaultsItsPathToCwd(t *testing.T) {
	// "." resolves to an absolute path before git sees it, so the fake is
	// seeded with the process's own cwd rather than a hardcoded one -- this
	// asserts couch's behaviour, not the developer's directory layout.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	rt := newRT(t, wd)
	out, errw, code := runLaunchRT(rt, "", "")
	if code != 0 {
		t.Fatalf("launch with no path: exit %d, stderr %q", code, errw)
	}
	if !strings.Contains(out, "started ") {
		t.Fatalf("no actor was started: %q", out)
	}
	if len(rt.runner.Ops) == 0 || !strings.HasPrefix(rt.runner.Ops[0], "start "+wd+":") {
		t.Fatalf("runner ops = %v, want a spawn in the cwd %q", rt.runner.Ops, wd)
	}
}

// The wiring itself, pinned without a pty: with a terminal, `start` must get a
// console AND a PtyRunner. Forcing consoleRunner to decline left the whole suite
// green twice over (BR-24).
func TestConsoleRunnerWiresThePtyRunnerWhenATerminalExists(t *testing.T) {
	console, runner := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil)
	if console == nil {
		t.Fatal("no console was built for `start` with a terminal")
	}
	if _, ok := runner.(*couchcore.PtyRunner); !ok {
		t.Fatalf("runner = %T, want *couchcore.PtyRunner — children would get no pty", runner)
	}
}

// Pin the production entry link, including its real terminal detection. A
// consoleRunnerFor-only test stays green if consoleRunner is replaced with the
// fallback outright (BR-24).
func TestConsoleRunnerDetectsARealPTY(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatalf("open pty: %v", err)
	}
	defer master.Close()
	defer slave.Close()

	console, runner := consoleRunner("start", slave, slave)
	if console == nil {
		t.Fatal("production consoleRunner declined a real pty")
	}
	if _, ok := runner.(*couchcore.PtyRunner); !ok {
		t.Fatalf("runner = %T, want *couchcore.PtyRunner", runner)
	}
}

func TestConsoleRunnerDeclinesWithoutATerminalWiring(t *testing.T) {
	console, runner := consoleRunnerFor("start", strings.NewReader(""), false, nil, nil)
	if console != nil {
		t.Fatal("a console was built with no terminal")
	}
	if _, ok := runner.(couchcore.ExecRunner); !ok {
		t.Fatalf("runner = %T, want couchcore.ExecRunner", runner)
	}
}

// The hierarchical switcher's actionable provider must be wired on the real
// run path. Typeahead itself is deliberately pure and in-memory.
func TestConsoleGetsCouchsActionableProvider(t *testing.T) {
	rt := newRT(t, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatalf("NewCouch: %v", err)
	}
	console, _ := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil)
	if console == nil {
		t.Fatal("no console to wire")
	}
	if console.ActionableProvider() != nil {
		t.Fatal("an actionable provider was set before the run path; this test would prove nothing")
	}

	// Drive the REAL path. The child has already exited, so Run returns at once
	// instead of blocking -- which is what AutoExit models.
	rec, h, err := c.Spawn(couchcore.StartArgs{Cwd: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runConsole(console, c, couchcore.StartResult{Record: rec, Handle: h}, &bytes.Buffer{})

	provider := console.ActionableProvider()
	if provider == nil {
		t.Fatal("the run path left the actionable provider nil")
	}
	if got, err := provider(context.Background(), nil); err != nil || len(got) != 0 {
		t.Fatalf("provider returned %v, %v for an empty registry", got, err)
	}
}

func TestWireResolverOmitsUnboundParkButRetainsDiagnosticInventory(t *testing.T) {
	rt := newRT(t, "/repo")
	parked := seedVerifiedPark(t, rt, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	console, _ := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil)
	wireResolver(console, c)
	provider := console.ActionableProvider()
	if provider == nil {
		t.Fatal("production wiring left actionable provider nil")
	}
	rows, err := provider(context.Background(), nil)
	if err != nil || len(rows) != 0 {
		t.Fatalf("unbound actionable rows = %+v, %v", rows, err)
	}
	diagnostic, err := c.ThreadInventory()
	if err != nil || len(diagnostic) != 1 || diagnostic[0].Address != parked.Address {
		t.Fatalf("diagnostic inventory = %+v, %v; parked record must remain visible to list/show", diagnostic, err)
	}
}

type blockingActionableArtifacts struct {
	*couchcore.FakeThreadArtifactCollisionChecker
	entered chan struct{}
	release chan struct{}
}

func (a *blockingActionableArtifacts) ResolveEstablished(ctx context.Context, _, _, _ string) (couchcore.NativeBindingResolution, error) {
	close(a.entered)
	select {
	case <-ctx.Done():
		return couchcore.NativeBindingResolution{}, ctx.Err()
	case <-a.release:
		return couchcore.NativeBindingResolution{}, errors.New("released without cancellation")
	}
}

func TestWireResolverPropagatesContextIntoActionableInventory(t *testing.T) {
	rt := newRT(t, "/repo")
	seedVerifiedPark(t, rt, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	artifacts := &blockingActionableArtifacts{
		FakeThreadArtifactCollisionChecker: couchcore.NewFakeThreadArtifactCollisionChecker(),
		entered:                            make(chan struct{}),
		release:                            make(chan struct{}),
	}
	c.Artifacts = artifacts
	console, _ := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil)
	wireResolver(console, c)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := console.ActionableProvider()(ctx, nil)
		done <- err
	}()
	<-artifacts.entered
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("provider error = %v, want context canceled", err)
		}
	case <-time.After(250 * time.Millisecond):
		close(artifacts.release)
		<-done
		t.Fatal("provider did not propagate cancellation into binding resolution")
	}
}

func TestConsoleWiringPropagatesAuthoritativeThreadStoreFailures(t *testing.T) {
	rt := newRT(t, "/repo")
	seedThread(t, rt, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	console, _ := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil)
	wireResolver(console, c)
	if err := os.WriteFile(filepath.Join(rt.dir, "threadstore", "manifest.json"), []byte("{corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := console.ActionableProvider()(context.Background(), nil); err == nil {
		t.Fatal("production actionable provider swallowed corrupt ThreadStore")
	}
}

// The panel's action dispatcher must be wired on the run path, not left nil --
// the first cut declared four panel actions with nothing behind them, so the
// operator could not start a second child at all.
func TestConsoleGetsAnActionDispatcher(t *testing.T) {
	rt := newRT(t, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatalf("NewCouch: %v", err)
	}
	console, _ := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil)
	if console == nil {
		t.Fatal("no console to wire")
	}
	if console.Ops() != nil {
		t.Fatal("a dispatcher was set before the run path; this test would prove nothing")
	}

	rec, h, err := c.Spawn(couchcore.StartArgs{Cwd: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runConsole(console, c, couchcore.StartResult{Record: rec, Handle: h}, &bytes.Buffer{})

	ops := console.Ops()
	if ops == nil {
		t.Fatal("the run path left the panel's dispatcher nil — its actions would refuse")
	}
	// It must reach couch's own table: an unknown name is refused rather than
	// silently succeeding.
	if _, err := ops(couchcore.OperationCall{Name: "no-such-operation"}); err == nil {
		t.Fatal("the dispatcher accepted an operation couch does not declare")
	}
	// And a real one is accepted.
	if _, err := ops(couchcore.OperationCall{Name: "list"}); err != nil {
		t.Fatalf("list through the panel dispatcher: %v", err)
	}
}

func TestInitialConsoleAttachDispatchesDeclaredOperation(t *testing.T) {
	console, _ := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil)
	if console == nil {
		t.Fatal("no console")
	}
	wantAddress := couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"}
	start := couchcore.StartResult{Record: couchcore.ActorRecord{Thread: wantAddress}}
	var got couchcore.OperationCall
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		got = call
		return wantAddress, nil
	})

	if err := dispatchInitialAttach(console, start); err != nil {
		t.Fatal(err)
	}
	if got.Name != "attach" || !got.Implicit || !reflect.DeepEqual(got.TypedPayload, start) ||
		got.Args["repo-scope"] != wantAddress.RepoScope || got.Args["tag"] != string(wantAddress.Tag) {
		t.Fatalf("initial attach call = %+v", got)
	}
}

func TestWireAttachAbortCleansStartedActorAfterConsoleRefusal(t *testing.T) {
	rt := newRT(t, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	record, handle, err := c.Spawn(couchcore.StartArgs{Cwd: "/repo"})
	if err != nil {
		t.Fatal(err)
	}
	console, _ := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil)
	console.Stop()
	wireResolver(console, c)

	_, err = console.Ops()(couchcore.OperationCall{
		Name: "attach", Implicit: true, TypedPayload: couchcore.StartResult{Record: record, Handle: handle},
		Args: map[string]string{"repo-scope": record.Thread.RepoScope, "tag": string(record.Thread.Tag)},
	})
	if err == nil {
		t.Fatal("stopped Console attach unexpectedly succeeded")
	}
	if handle.Alive() {
		t.Fatal("failed wired attach left started handle alive")
	}
	if got := c.List(); len(got) != 0 {
		t.Fatalf("failed wired attach left actor registered: %+v", got)
	}
}

func TestConsoleExitForgetsThroughCouchRegistry(t *testing.T) {
	rt := newRT(t, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatalf("NewCouch: %v", err)
	}
	console, _ := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil)
	rec, h, err := c.Spawn(couchcore.StartArgs{Cwd: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if len(c.List()) != 1 {
		t.Fatal("test setup has no registered actor")
	}

	runConsole(console, c, couchcore.StartResult{Record: rec, Handle: h}, &bytes.Buffer{})

	if got := c.List(); len(got) != 0 {
		t.Fatalf("registry after terminal child exit = %+v, want empty", got)
	}
}

// A refusal is a next-action spec: every remedy it names must be a command the
// operator can run.
func TestCapacityRefusalNamesOnlyRunnableCommands(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.boundedOne("/repo")
	if _, errw, code := runLaunchRT(rt, "/repo", ""); code != 0 {
		t.Fatalf("first start failed: %d %q", code, errw)
	}
	rt.markLive(t) // the guard needs a live incumbent to refuse for
	_, errw, code := runLaunchRT(rt, "/repo", "")
	if code == 0 {
		t.Fatal("a second start on an occupied tree was allowed")
	}

	if strings.Contains(errw, "switch to it") {
		t.Errorf("the refusal still offers an action couch cannot perform: %q", errw)
	}
	// Every suggested Couch argv must be accepted by the public parser.
	// Only the SUGGESTION lines (`  -> couch <args> ...`) are commands; the
	// rest is prose and may legitimately mention couch.
	found := 0
	for _, line := range strings.Split(errw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "-> couch ") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "-> couch "))
		if len(fields) == 0 {
			continue
		}
		found++
		if _, err := ParseCLI(fields[:1], couchcore.Operations()); err != nil {
			t.Errorf("the refusal suggests unrunnable `couch %s`: %v", fields[0], err)
		}
	}
	if found == 0 {
		t.Errorf("the refusal names no runnable command at all: %q", errw)
	}
}
