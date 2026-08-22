package couchcmd

import (
	"bytes"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// testRT builds the domain over fakes. There is deliberately NO production
// branch: a test that can reach ExecGit resolves paths against whatever
// checkout it happens to run in, so it asserts on the developer's directory
// layout rather than on couch. One such test passed here and failed in a
// pristine worktree of the same commit.
type testRT struct {
	dir    string
	runner *couchcore.FakeRunner
	proc   *couchcore.FakeProcOps
	git    *couchcore.FakeGit
}

func (t testRT) Getenv(string) string { return "" }
func (t testRT) StoreDir() string     { return t.dir }

func (t testRT) NewCouch() (*couchcore.Couch, error) {
	return couchcore.New(
		t.runner, couchcore.NewFakePathOps(nil), t.git, t.proc,
		couchcore.NewStore(t.dir), couchcore.FixedClock{}, couchcore.NewFixedIDGen("ah8d", "b2c1"),
	)
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
	return testRT{
		dir:    t.TempDir(),
		runner: runner,
		proc:   couchcore.NewFakeProcOps(),
		git:    couchcore.NewFakeGit(replies),
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
		if op.Invoke == nil {
			t.Errorf("%s: declared but not invocable", op.Name)
		}
	}
}

func TestOperationArityMatchesExpectation(t *testing.T) {
	// Declared in the test rather than read from the operation itself, so
	// this cannot degrade into asserting X == X.
	want := map[string]int{"start": 2, "list": 0, "show": 1, "stop": 1, "name": 2, "describe": 2, "publish-description": 2}
	for _, op := range couchcore.Operations() {
		if got := len(op.Args); got != want[op.Name] {
			t.Errorf("%s has %d args, want %d", op.Name, got, want[op.Name])
		}
	}
}

func TestListOnEmptyRegistry(t *testing.T) {
	out, errw, code := runRT(newRT(t), "list")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errw)
	}
	if !strings.Contains(out, "no trees") {
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

func TestBindArgsAcceptsFlagsAndPositionals(t *testing.T) {
	var start couchcore.Operation
	for _, op := range couchcore.Operations() {
		if op.Name == "start" {
			start = op
		}
	}
	got, err := bindArgs(start, []string{"../pair", "--same-tree"})
	if err != nil {
		t.Fatalf("bindArgs: %v", err)
	}
	if got["path"] != "../pair" || got["same-tree"] != "true" {
		t.Fatalf("bound = %v", got)
	}
}

func TestListShowsANamedTreeWithNoAgent(t *testing.T) {
	// The forgetting case: a tree that was named and then parked has no actor,
	// but it is exactly the thread the operator loses track of. It must be a
	// visible row, not filtered out.
	rt := newRT(t, "/repo")
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

func TestShowResolvesANameToItsTreePath(t *testing.T) {
	rt := newRT(t, "/repo")
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
}

func TestRenderedOutputHasNoANSIWhenNotATerminal(t *testing.T) {
	// A bytes.Buffer is not a terminal, so dimming must be suppressed --
	// otherwise piped or captured output carries escape codes.
	rt := newRT(t, "/repo")
	_, _, _ = runRT(rt, "name", "/repo", "plain")
	out, _, _ := runRT(rt, "list")
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("ANSI leaked into non-terminal output: %q", out)
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
	if out, errw, code := runRT(rt, "start", "/repo"); code != 0 {
		t.Fatalf("first start: code=%d out=%q err=%q", code, out, errw)
	}
	// Mark the child live so the guard has something real to refuse for.
	rt.markLive(t)
	_, errw, code := runRT(rt, "start", "/repo")
	if code == 0 {
		t.Fatal("a second start on an occupied tree must fail")
	}
	for _, want := range []string{"already has an agent", "share a branch and index", "--same-tree"} {
		if !strings.Contains(errw, want) {
			t.Errorf("refusal missing %q; got %q", want, errw)
		}
	}
}

func TestStopReportsWhetherItActuallySignalled(t *testing.T) {
	rt := newRT(t, "/repo")
	if _, errw, code := runRT(rt, "start", "/repo"); code != 0 {
		t.Fatalf("start: %s", errw)
	}
	rt.markLive(t)
	out, errw, code := runRT(rt, "stop", "/repo")
	if code != 0 {
		t.Fatalf("stop: code=%d err=%q", code, errw)
	}
	if !strings.Contains(out, "signalled") {
		t.Fatalf("out = %q; stop must say it signalled a live child", out)
	}
}
