package couchcmd

import (
	"bytes"
	"context"
	"crypto/rand"
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

// NewCouchWith IGNORES the caller's runner and keeps the fake.
//
// That is the point: production picks a PtyRunner for `start`, and a CLI test
// must still drive the whole dispatch against fakes. What the test asserts is
// which BRANCH was taken (console vs --no-console), which is observable in the
// rendered output, not which concrete runner object was constructed.
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
	// couch start blocks on Handle.Wait for the child's lifetime -- right in
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

	if _, stderr, code := runRT(rt, "start", "/repo", "--no-console"); code != 0 {
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
	if _, errw, code := runRT(rt, "start", "/repo"); code != 0 {
		t.Fatalf("start: code=%d stderr=%q", code, errw)
	}
	if rt.supervisor.acquired != 1 || rt.supervisor.released != 1 {
		t.Fatalf("supervisor acquire/release = %d/%d, want 1/1", rt.supervisor.acquired, rt.supervisor.released)
	}
}

func TestResumeAcquiresAndReleasesSupervisorLease(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.supervisor.err = fmt.Errorf("resume reached singleton acquisition")
	_, errw, code := runRT(rt, "resume", "couch-0102030405060708")
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

	out, errw, code := runRT(rt, "resume", string(parked.Address.Tag))
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

func TestDirectStoreOperationDoesNotAcquireSupervisorLease(t *testing.T) {
	rt := newRT(t)
	if _, errw, code := runRT(rt, "list"); code != 0 {
		t.Fatalf("list: code=%d stderr=%q", code, errw)
	}
	if rt.supervisor.acquired != 0 || rt.supervisor.released != 0 {
		t.Fatalf("direct-store list touched supervisor lease: %d/%d", rt.supervisor.acquired, rt.supervisor.released)
	}
}

func TestHeldSupervisorRefusesBeforeStartingActor(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.supervisor.err = fmt.Errorf("namespace is supervised by pid 42")
	_, errw, code := runRT(rt, "start", "/repo")
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

func runRT(rt testRT, args ...string) (string, string, int) {
	var out, errw bytes.Buffer
	code := RunWithRuntime(args, strings.NewReader(""), &out, &errw, rt)
	return out.String(), errw.String(), code
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
	want := map[string]int{"prepare-start": 2, "start": 4, "list": 0, "show": 2, "stop": 1, "name": 3, "describe": 3, "publish-description": 3, "switch": 2, "attach": 2, "park": 4, "resume": 3}
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

	if _, errw, code := runRT(rt, "publish-description", "agent summary"); code != 0 {
		t.Fatalf("publish-description: code=%d stderr=%q", code, errw)
	}
	got, err := c.Threads.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if got.PublishedSummary != "agent summary" || got.Description != "" {
		t.Fatalf("published thread = %+v", got)
	}
	if _, errw, code := runRT(rt, "publish-description", ""); code != 0 {
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
	out, errw, code := runRT(newRT(t), "list")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errw)
	}
	if !strings.Contains(out, "no threads") {
		t.Fatalf("out = %q", out)
	}
}

func TestUnknownOperationIsNonZeroAndListsWhatExists(t *testing.T) {
	out, errw, code := runRT(newRT(t), "frobnicate")
	if code == 0 {
		t.Fatal("unknown operation must be non-zero")
	}
	if !strings.Contains(errw, "unknown operation") || !strings.Contains(errw, "start") {
		t.Fatalf("stderr = %q; the error should name what does exist", errw)
	}
	_ = out
}

func TestMissingRequiredArgumentIsRejectedBeforeAnyWork(t *testing.T) {
	_, errw, code := runRT(newRT(t), "show")
	if code == 0 {
		t.Fatal("a missing required argument must be non-zero")
	}
	if !strings.Contains(errw, "missing required argument") {
		t.Fatalf("stderr = %q", errw)
	}
}

func TestHelpListsEveryDeclaredOperation(t *testing.T) {
	out, _, code := runRT(newRT(t), "--help")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, name := range couchcore.OperationNames() {
		if !strings.Contains(out, name) {
			t.Errorf("help omits %q", name)
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
	if strings.Contains(string(raw), "couch stop <ref>") {
		t.Fatal("README advertises stop as a second-process command before #147 owner routing exists")
	}
}

func TestBindArgsAcceptsFlagsAndPositionals(t *testing.T) {
	var start couchcore.Operation
	for _, op := range couchcore.Operations() {
		if op.Name == "start" {
			start = op
		}
	}
	got, err := bindArgs(start, []string{"../pair", "--no-console"})
	if err != nil {
		t.Fatalf("bindArgs: %v", err)
	}
	if got["path"] != "../pair" || got["no-console"] != "true" {
		t.Fatalf("bound = %v", got)
	}
}

func TestBindArgsRejectsMissingOrEmptyValueBearingFlag(t *testing.T) {
	var start couchcore.Operation
	for _, op := range couchcore.Operations() {
		if op.Name == "start" {
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
	for _, argv := range [][]string{{"start", "/repo", "--agent", "--no-console"}, {"start", "/repo", "--agent=", "--no-console"}} {
		t.Run(strings.Join(argv, "_"), func(t *testing.T) {
			rt := newRT(t, "/repo")
			rt.boundedOne("/repo")
			_, stderr, code := runRT(rt, argv...)
			if code == 0 || !strings.Contains(stderr, "--agent requires a non-empty value") {
				t.Fatalf("runRT(%q): code=%d stderr=%q", argv, code, stderr)
			}
			if len(rt.runner.Ops) != 0 {
				t.Fatalf("runRT(%q) reached runner operations %q", argv, rt.runner.Ops)
			}
		})
	}
}

func TestListShowsANamedTreeWithNoAgent(t *testing.T) {
	// The forgetting case: a tree that was named and then parked has no actor,
	// but it is exactly the thread the operator loses track of. It must be a
	// visible row, not filtered out.
	rt := newRT(t, "/repo")
	seedThread(t, rt, "/repo")
	if _, errw, code := runRT(rt, "name", "/repo", "the pair tree"); code != 0 {
		t.Fatalf("name failed: %s", errw)
	}
	out, _, code := runRT(rt, "list")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "the pair tree") {
		t.Fatalf("out = %q; a named tree must appear even with no agent", out)
	}
	if !strings.Contains(out, "(no agent running)") {
		t.Fatalf("out = %q; the absence of an agent must be stated", out)
	}
}

func TestCLIEmptyNameClearsHumanThreadName(t *testing.T) {
	rt := newRT(t, "/repo")
	created := seedThread(t, rt, "/repo")
	if _, errw, code := runRT(rt, "name", string(created.Address.Tag), "compiler"); code != 0 {
		t.Fatalf("set name: code=%d stderr=%q", code, errw)
	}
	if _, errw, code := runRT(rt, "name", string(created.Address.Tag), ""); code != 0 {
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
	if _, errw, code := runRT(rt, "name", "/repo", "pairtree"); code != 0 {
		t.Fatalf("name failed: %s", errw)
	}
	out, errw, code := runRT(rt, "show", "pairtree")
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

	if _, errw, code := runRT(rt, "name", repeatedTag, "local thread"); code != 0 {
		t.Fatalf("name: code=%d stderr=%q", code, errw)
	}
	if _, errw, code := runRT(rt, "describe", repeatedTag, "local description"); code != 0 {
		t.Fatalf("describe: code=%d stderr=%q", code, errw)
	}
	out, errw, code := runRT(rt, "show", repeatedTag)
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
	_, _, _ = runRT(rt, "name", "/repo", "plain")
	out, _, _ := runRT(rt, "list")
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
// and the suite stayed green. This drives the CLI itself: every declared name
// must resolve, and a corpus of plausible undeclared names must be rejected.
// It is not a proof for arbitrary strings -- that guarantee comes from
// RunWithRuntime having a single table-only Resolve and no switch -- but it
// does catch the attack that got through.
func TestCLIAcceptsExactlyTheDeclaredOperations(t *testing.T) {
	declared := map[string]bool{}
	for _, name := range couchcore.OperationNames() {
		declared[name] = true
		if _, ok := Resolve(name); !ok {
			t.Errorf("declared operation %q does not resolve in the CLI", name)
		}
	}
	for _, name := range []string{"nuke", "kill", "restart", "attach", "switch", "ls", "run", "exec"} {
		if declared[name] {
			continue
		}
		if _, ok := Resolve(name); ok {
			t.Errorf("CLI resolves %q, which is not a declared operation", name)
		}
		if _, errw, code := runRT(newRT(t), name); code == 0 {
			t.Errorf("CLI accepted undeclared operation %q (stderr %q)", name, errw)
		}
	}
}

func TestStartRendersTheRefusalWithThePolicyShapedOffer(t *testing.T) {
	// Done-when 2's rendering had no reachable test before the Runtime seam.
	rt := newRT(t, "/repo")
	rt.boundedOne("/repo")
	if out, errw, code := runRT(rt, "start", "/repo"); code != 0 {
		t.Fatalf("first start: code=%d out=%q err=%q", code, out, errw)
	}
	// Mark the child live so the guard has something real to refuse for.
	rt.markLive(t)
	_, errw, code := runRT(rt, "start", "/repo")
	if code == 0 {
		t.Fatal("a second start on an occupied tree must fail")
	}
	for _, want := range []string{"at capacity 1", `admission key "/repo"`, "couch list"} {
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
	if strings.Contains(got, "couch start ") {
		t.Fatalf("provision refusal invented a runnable path: %q", got)
	}
}

func TestExternalStopRefusesUntilOwnerRoutingExists(t *testing.T) {
	rt := newRT(t, "/repo")
	if _, errw, code := runRT(rt, "start", "/repo"); code != 0 {
		t.Fatalf("start: %s", errw)
	}
	rt.markLive(t)
	_, errw, code := runRT(rt, "stop", "/repo")
	if code == 0 || !strings.Contains(errw, "routing requires #147") {
		t.Fatalf("stop: code=%d err=%q", code, errw)
	}
}

func TestRemovedAdmissionBypassCannotBindInAnyForm(t *testing.T) {
	rt := newRT(t, "/repo")
	_, errw, code := runRT(rt, "start", "/repo", "true")
	if code == 0 {
		t.Fatal("a positional word was accepted as an admission bypass")
	}
	if !strings.Contains(errw, "unexpected argument") {
		t.Fatalf("stderr = %q", errw)
	}
	if _, errw, code := runRT(rt, "start", "/repo", "--same-tree"); code == 0 || !strings.Contains(errw, "unknown flag") {
		t.Fatalf("removed bypass was accepted: code=%d stderr=%q", code, errw)
	}
}

func TestOptionalPositionalArgsStillBind(t *testing.T) {
	// The rule is "guard bypasses are flag-only", NOT "optional args never
	// bind" -- the broader version broke `couch describe <ref> <text>`.
	rt := newRT(t, "/repo")
	seedThread(t, rt, "/repo")
	if _, errw, code := runRT(rt, "name", "/repo", "thing"); code != 0 {
		t.Fatalf("name: %s", errw)
	}
	if _, errw, code := runRT(rt, "describe", "thing", "what it is doing"); code != 0 {
		t.Fatalf("describe with a positional description: %s", errw)
	}
	out, _, _ := runRT(rt, "describe", "thing")
	if !strings.Contains(out, "what it is doing") {
		t.Fatalf("out = %q", out)
	}
}

// The escape hatch announces itself. A silent degradation is how a fallback
// becomes the default nobody noticed (Decision 2).
func TestStartWithNoConsoleAnnouncesTheFallback(t *testing.T) {
	out, errw, code := runRT(newRT(t, "/repo"), "start", "/repo", "--no-console")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errw)
	}
	if !strings.Contains(out, "no console") {
		t.Fatalf("the fallback did not announce itself: %q", out)
	}
	if !strings.Contains(out, "started ") {
		t.Fatalf("the no-console path did not report the actor: %q", out)
	}
}

// A guard bypass must never bind positionally: a stray word must not be able to
// turn off the console. It must remain explicitly named.
func TestNoConsoleNeverBindsPositionally(t *testing.T) {
	_, errw, code := runRT(newRT(t, "/repo"), "start", "/repo", "no-console")
	if code == 0 {
		t.Fatalf("a positional `no-console` was accepted; it must not bind (stderr %q)", errw)
	}
}

// `couch start` with no terminal must fall back, loudly, to the stdio path.
//
// The first cut spawned the child, sized it to a ZERO-ROW pty, then exited 1
// with nothing printed -- so a scripted or piped invocation left a registered
// actor the operator could neither see nor use (M2 BR-23). runRT drives exactly
// this shape: its stdout is a buffer, not a tty.
func TestStartWithoutATerminalFallsBackLoudly(t *testing.T) {
	out, errw, code := runRT(newRT(t, "/repo"), "start", "/repo")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errw)
	}
	if !strings.Contains(out, "no console") {
		t.Fatalf("the fallback did not announce itself: %q", out)
	}
	if !strings.Contains(out, "started ") {
		t.Fatalf("no actor was reported: %q", out)
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
		args        map[string]string
		hasTerminal bool
		want        bool
	}{
		{"start on a terminal", "start", nil, true, true},
		{"start with --no-console", "start", map[string]string{"no-console": "true"}, true, false},
		{"start with no terminal", "start", nil, false, false},
		{"resume on a terminal", "resume", nil, true, true},
		{"resume with no terminal", "resume", nil, false, false},
		{"a read-only operation", "list", nil, true, false},
		{"stop never takes the terminal", "stop", nil, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := WantsConsole(c.op, c.args, c.hasTerminal); got != c.want {
				t.Fatalf("WantsConsole(%q, %v, %v) = %v, want %v", c.op, c.args, c.hasTerminal, got, c.want)
			}
		})
	}
}

// The plumbing half, still unconditional: with no terminal there must be no
// console and the stdio runner.
func TestConsoleRunnerDeclinesWithoutATerminal(t *testing.T) {
	console, runner := consoleRunner("start", map[string]string{}, strings.NewReader(""), &bytes.Buffer{})
	if console != nil {
		t.Fatal("a console was built with no terminal")
	}
	if _, ok := runner.(couchcore.ExecRunner); !ok {
		t.Fatalf("runner = %T, want couchcore.ExecRunner", runner)
	}
}

// `start` with no path defaults to "." -- which is what makes `cd brain && couch
// start` the way home is chosen (Decision 1).
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
	out, errw, code := runRT(rt, "start")
	if code != 0 {
		t.Fatalf("`couch start` with no path: exit %d, stderr %q", code, errw)
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
	console, runner := consoleRunnerFor("start", map[string]string{}, strings.NewReader(""), true, nil, nil)
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

	console, runner := consoleRunner("start", map[string]string{}, slave, slave)
	if console == nil {
		t.Fatal("production consoleRunner declined a real pty")
	}
	if _, ok := runner.(*couchcore.PtyRunner); !ok {
		t.Fatalf("runner = %T, want *couchcore.PtyRunner", runner)
	}
}

func TestConsoleRunnerDeclinesWithoutATerminalWiring(t *testing.T) {
	console, runner := consoleRunnerFor("start", map[string]string{}, strings.NewReader(""), false, nil, nil)
	if console != nil {
		t.Fatal("a console was built with no terminal")
	}
	if _, ok := runner.(couchcore.ExecRunner); !ok {
		t.Fatalf("runner = %T, want couchcore.ExecRunner", runner)
	}
}

// The panel's resolver must be couch's own rule, not left nil.
//
// Decision 12's wiring check: an injection seam nothing passes is a seam that
// does nothing, and the panel would silently degrade to "show everything" with
// typeahead inert. Asserting the FUNCTION IDENTITY is the only way to catch
// that, since a nil resolver still renders a panel.
func TestConsoleGetsCouchsOwnResolver(t *testing.T) {
	rt := newRT(t, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatalf("NewCouch: %v", err)
	}
	console, _ := consoleRunnerFor("start", map[string]string{}, strings.NewReader(""), true, nil, nil)
	if console == nil {
		t.Fatal("no console to wire")
	}
	if console.Resolver() != nil {
		t.Fatal("a resolver was set before the run path; this test would prove nothing")
	}
	if console.Summaries() != nil {
		t.Fatal("a summary provider was set before the run path; this test would prove nothing")
	}

	// Drive the REAL path. The child has already exited, so Run returns at once
	// instead of blocking -- which is what AutoExit models.
	rec, h, err := c.Spawn(couchcore.StartArgs{Cwd: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runConsole(console, c, couchcore.StartResult{Record: rec, Handle: h}, &bytes.Buffer{})

	if console.Resolver() == nil {
		t.Fatal("the run path left the panel's resolver nil — typeahead would be inert")
	}
	if console.Summaries() == nil {
		t.Fatal("the run path left the panel's summary provider nil — parked trees would disappear")
	}
	if got, err := console.Resolver()("anything"); err != nil || len(got) != 0 {
		t.Fatalf("resolver returned %v, %v for an empty registry", got, err)
	}
}

func TestWireResolverInjectsActionableInventoryWithObservations(t *testing.T) {
	rt := newRT(t, "/repo")
	parked := seedVerifiedPark(t, rt, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	console, _ := consoleRunnerFor("start", map[string]string{}, strings.NewReader(""), true, nil, nil)
	wireResolver(console, c)
	provider := console.ActionableProvider()
	if provider == nil {
		t.Fatal("production wiring left actionable provider nil")
	}
	rows, err := provider(context.Background(), nil)
	if err != nil || len(rows) != 1 || rows[0].Address != parked.Address || rows[0].State != couchcore.ThreadParked {
		t.Fatalf("actionable rows = %+v, %v", rows, err)
	}
}

func TestConsoleWiringPropagatesAuthoritativeThreadStoreFailures(t *testing.T) {
	rt := newRT(t, "/repo")
	seedThread(t, rt, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	console, _ := consoleRunnerFor("start", map[string]string{}, strings.NewReader(""), true, nil, nil)
	wireResolver(console, c)
	if err := os.WriteFile(filepath.Join(rt.dir, "threadstore", "manifest.json"), []byte("{corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := console.Summaries()(); err == nil {
		t.Fatal("production summary callback swallowed corrupt ThreadStore")
	}
	if _, err := console.Resolver()("repo"); err == nil {
		t.Fatal("production reference callback swallowed corrupt ThreadStore")
	}
}

func TestConsoleWiringReturnsEveryAmbiguousHumanMatch(t *testing.T) {
	rt := newRT(t, "/repo")
	localScope, err := launcher.ResolveRepoScope("/repo")
	if err != nil {
		t.Fatal(err)
	}
	otherScope, err := launcher.ResolveRepoScope("/other")
	if err != nil {
		t.Fatal(err)
	}
	first := seedThreadAtAddress(t, rt, localScope.Key, "couch-0102030405060708", "/repo")
	second := seedThreadAtAddress(t, rt, otherScope.Key, "couch-1112131415161718", "/other")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	name := "compiler"
	for _, address := range []couchcore.ThreadAddress{first.Address, second.Address} {
		if _, err := c.ApplyThreadMetadata(address, couchcore.ThreadMetadataPatch{Name: &name}); err != nil {
			t.Fatal(err)
		}
	}
	console, _ := consoleRunnerFor("start", map[string]string{}, strings.NewReader(""), true, nil, nil)
	wireResolver(console, c)
	matches, err := console.Resolver()(name)
	if err != nil || len(matches) != 2 {
		t.Fatalf("ambiguous typeahead matches = %+v, %v", matches, err)
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
	console, _ := consoleRunnerFor("start", map[string]string{}, strings.NewReader(""), true, nil, nil)
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
	console, _ := consoleRunnerFor("start", map[string]string{}, strings.NewReader(""), true, nil, nil)
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
	console, _ := consoleRunnerFor("start", map[string]string{}, strings.NewReader(""), true, nil, nil)
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
	console, _ := consoleRunnerFor("start", map[string]string{}, strings.NewReader(""), true, nil, nil)
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
	if _, errw, code := runRT(rt, "start", "/repo"); code != 0 {
		t.Fatalf("first start failed: %d %q", code, errw)
	}
	rt.markLive(t) // the guard needs a live incumbent to refuse for
	_, errw, code := runRT(rt, "start", "/repo")
	if code == 0 {
		t.Fatal("a second start on an occupied tree was allowed")
	}

	if strings.Contains(errw, "switch to it") {
		t.Errorf("the refusal still offers an action couch cannot perform: %q", errw)
	}
	// Every `couch <verb>` it suggests must be a declared operation.
	declared := map[string]bool{}
	for _, n := range couchcore.OperationNames() {
		declared[n] = true
	}
	// Only the SUGGESTION lines (`  -> couch <verb> ...`) are commands; the
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
		if !declared[fields[0]] {
			t.Errorf("the refusal suggests `couch %s`, which is not a declared operation", fields[0])
		}
	}
	if found == 0 {
		t.Errorf("the refusal names no runnable command at all: %q", errw)
	}
}
