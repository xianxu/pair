package couchcore

// Live conformance: does FakeRunner's MODEL match what real processes and real
// git actually do?
//
// ARCH-MOCK asks for a stateful fake behind the seam PLUS a live check that
// detects drift. The distinction that matters here is that conformance means
// comparing two implementations against one shared scenario -- not running each
// separately and asserting whatever each happens to produce. A check that drives
// the fake by hand to the value it then asserts tests nothing.
//
// Gated on PAIR_LIVE_COUCH=1 with t.Skip and deliberately NO build tag, matching
// harness_tty_live_test.go. A //go:build tag would stop this file compiling
// under `go test ./cmd/...`, so it would rot invisibly -- the exact failure the
// check exists to prevent.

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func liveOnly(t *testing.T) {
	t.Helper()
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 to run against real processes and real git")
	}
}

func waitFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("child never became ready (%s)", path)
}

func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// TestRunnerConformance_ExitCode: both implementations must report the same
// exit code and the same post-exit liveness.
func TestRunnerConformance_ExitCode(t *testing.T) {
	liveOnly(t)

	real, err := ExecRunner{}.Start(t.TempDir(), []string{"sh", "-c", "exit 3"}, nil)
	if err != nil {
		t.Fatalf("real Start: %v", err)
	}
	realCode, realAlive := real.Wait(), real.Alive()

	f := NewFakeRunner()
	fh, _ := f.Start(t.TempDir(), []string{"sh", "-c", "exit 3"}, nil)
	f.SetExited(fh.ID(), 3)
	fakeCode, fakeAlive := fh.Wait(), fh.Alive()

	if realCode != fakeCode {
		t.Errorf("exit code: real %d, fake %d", realCode, fakeCode)
	}
	if realAlive != fakeAlive {
		t.Errorf("alive after exit: real %v, fake %v", realAlive, fakeAlive)
	}
	if realCode != 3 {
		t.Errorf("real exit code = %d, want 3 -- the scenario itself is wrong", realCode)
	}
}

func TestExecRunnerInheritsExactCouchNamespace(t *testing.T) {
	liveOnly(t)
	base := t.TempDir()
	realStore := filepath.Join(base, "real-store")
	if err := os.Mkdir(realStore, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(base, "store-alias")
	if err := os.Symlink(realStore, alias); err != nil {
		t.Fatal(err)
	}
	physicalStore, err := filepath.EvalSymlinks(realStore)
	if err != nil {
		t.Fatal(err)
	}
	namespace, err := ResolveCouchNamespace(alias, "/unused")
	if err != nil {
		t.Fatal(err)
	}
	capture := filepath.Join(base, "captured")
	handle, err := (ExecRunner{}).Start(base,
		[]string{"sh", "-c", `printf '%s' "$COUCH_STORE_DIR" > "$COUCH_CAPTURE"`},
		[]string{"COUCH_STORE_DIR=" + namespace.Dir(), "COUCH_CAPTURE=" + capture})
	if err != nil {
		t.Fatal(err)
	}
	if code := handle.Wait(); code != 0 {
		t.Fatalf("namespace probe exited %d", code)
	}
	raw, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(raw); got != namespace.Dir() || got != physicalStore {
		t.Fatalf("inherited namespace = %q, canonical = %q, physical = %q", got, namespace.Dir(), physicalStore)
	}
}

// TestRunnerConformance_SignalIgnored is the important one: it validates the
// fake's most opinionated modelling choice, that a signal alone does not kill.
// A real child that traps INT must stay alive, and so must the fake's default.
func TestRunnerConformance_SignalIgnored(t *testing.T) {
	liveOnly(t)

	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	real, err := ExecRunner{}.Start(dir,
		[]string{"sh", "-c", "trap '' INT; touch " + ready + "; sleep 5"}, nil)
	if err != nil {
		t.Fatalf("real Start: %v", err)
	}
	waitFile(t, ready)
	if err := real.Signal(os.Interrupt); err != nil {
		t.Fatalf("real Signal: %v", err)
	}
	time.Sleep(100 * time.Millisecond) // give a wrong implementation time to die
	realAlive := real.Alive()

	f := NewFakeRunner()
	fh, _ := f.Start(dir, []string{"sh", "-c", "trap '' INT; sleep 5"}, nil)
	_ = fh.Signal(os.Interrupt)
	fakeAlive := fh.Alive()

	if realAlive != fakeAlive {
		t.Errorf("alive after ignored SIGINT: real %v, fake %v", realAlive, fakeAlive)
	}
	if !realAlive {
		t.Error("a child trapping INT died -- the fake's model would be wrong, not the test")
	}
	_ = real.Signal(os.Kill)
	_ = real.Wait()
}

// TestRunnerConformance_SignalFatal is the complementary disposition: a child
// with the default handler DOES die. The fake must be scripted to model it,
// and scripting it is the point -- the fake cannot know a child's disposition,
// so it has to be told, and this check proves the told version matches reality.
func TestRunnerConformance_SignalFatal(t *testing.T) {
	liveOnly(t)

	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	real, err := ExecRunner{}.Start(dir,
		[]string{"sh", "-c", "touch " + ready + "; exec sleep 5"}, nil)
	if err != nil {
		t.Fatalf("real Start: %v", err)
	}
	waitFile(t, ready)
	if err := real.Signal(os.Interrupt); err != nil {
		t.Fatalf("real Signal: %v", err)
	}
	waitUntil(t, "real child to die on SIGINT", func() bool { return !real.Alive() })
	_ = real.Wait()

	f := NewFakeRunner()
	fh, _ := f.Start(dir, []string{"sh", "-c", "sleep 5"}, nil)
	f.SetDiesOn(fh.ID(), os.Interrupt, 130)
	_ = fh.Signal(os.Interrupt)

	if real.Alive() != fh.Alive() {
		t.Errorf("alive after fatal SIGINT: real %v, fake %v", real.Alive(), fh.Alive())
	}
}

// TestGitConformance_LinkedWorktree exercises the case the whole identity model
// rests on: a primary checkout and a linked worktree of the SAME repo must
// resolve to distinct Worktrees, and a subdirectory of either must resolve to
// its own root.
func TestGitConformance_LinkedWorktree(t *testing.T) {
	liveOnly(t)

	base := t.TempDir()
	primary := filepath.Join(base, "repo")
	linked := filepath.Join(base, "wt")

	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}
	if err := os.MkdirAll(filepath.Join(primary, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	run(base, "init", "-q", "-b", "main", primary)
	run(primary, "config", "user.email", "t@example.com")
	run(primary, "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(primary, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(primary, "add", ".")
	run(primary, "commit", "-qm", "init")
	run(primary, "worktree", "add", "-q", "-b", "side", linked)

	git, pathOps := ExecGit{}, OSPathOps{}

	primaryRoot, err := Resolve(filepath.Join(primary, "sub"), git, pathOps)
	if err != nil {
		t.Fatalf("resolve primary subdirectory: %v", err)
	}
	linkedRoot, err := Resolve(linked, git, pathOps)
	if err != nil {
		t.Fatalf("resolve linked worktree: %v", err)
	}

	wantPrimary, _ := pathOps.Physical(NormalizePath(primary))
	if string(primaryRoot) != wantPrimary {
		t.Errorf("primary subdirectory resolved to %q, want %q", primaryRoot, wantPrimary)
	}
	if primaryRoot == linkedRoot {
		t.Fatalf("primary and linked worktree collapsed to one identity (%q) -- "+
			"both could then never host agents concurrently", primaryRoot)
	}

	// The transitional display cache must preserve them as two trees. Runtime
	// admission itself comes only from normalized provider keys.
	reg := NewRegistry().Insert(ActorRecord{ID: "a", Args: StartArgs{Worktree: primaryRoot}})
	reg = reg.Insert(ActorRecord{ID: "b", Args: StartArgs{Worktree: linkedRoot}})
	if len(reg.Get(primaryRoot)) != 1 || len(reg.Get(linkedRoot)) != 1 {
		t.Fatalf("linked worktree cache collapsed: %+v", reg.Records())
	}

	// And FakeGit, canned from the real answers, must agree.
	fake := NewFakeGit(map[GitCall]string{
		{Dir: wantPrimary, Args: "rev-parse --show-toplevel"}: wantPrimary,
	})
	fakeRoot, err := Resolve(wantPrimary, fake, pathOps)
	if err != nil {
		t.Fatalf("fake resolve: %v", err)
	}
	if fakeRoot != primaryRoot {
		t.Errorf("fake resolved %q, real resolved %q", fakeRoot, primaryRoot)
	}
}

// Terminal conformance: does FakeRunner's pty double behave like a real pty?
//
// The comparison is over CONTRACT PREDICATES that neither side is told -- does
// a write succeed while running and fail after exit, does a resize, does Done
// flip, does Wait report the scripted code. The first draft of this compared
// snapshot CONTENT, which the fake side had to be hand-fed with Emit; a check
// that drives the fake to the value it then asserts tests nothing, and this repo
// has a lesson saying so. Content is the wrong axis anyway: a fake has no shell,
// so making it produce shell output proves only that Emit works.
//
// The property that content WOULD have covered -- "the child actually observed
// the resize", the drift a silently-accepting fake would hide -- is pinned on
// the real side where it is meaningful, by
// ptychild.TestChildResizeIsObservedByTheChild.
func TestTerminalConformance_LifecyclePredicates(t *testing.T) {
	liveOnly(t)

	type predicates struct {
		writeWhileRunningOK  bool
		resizeWhileRunningOK bool
		doneBeforeExit       bool
		doneAfterExit        bool
		waitCode             int
		writeAfterExitErrors bool
	}

	observe := func(child *ptychild.Child, end func()) predicates {
		var p predicates
		_, err := child.Write([]byte("ping\n"))
		p.writeWhileRunningOK = err == nil
		p.resizeWhileRunningOK = child.Resize(ptychild.Size{Rows: 40, Cols: 100}) == nil
		p.doneBeforeExit = child.Done()

		end()
		p.waitCode = child.Wait()
		p.doneAfterExit = child.Done()
		_, err = child.Write([]byte("after"))
		p.writeAfterExitErrors = err != nil
		return p
	}

	r := &PtyRunner{Size: func() ptychild.Size { return ptychild.Size{Rows: 24, Cols: 80} }}
	rh, err := r.Start(t.TempDir(), []string{"sh", "-c", "read line; exit 3"}, nil)
	if err != nil {
		t.Fatalf("real Start: %v", err)
	}
	realChild := rh.(TerminalHandle).Terminal()
	realPreds := observe(realChild, func() {
		// The child exits on its own once it has read the line written above.
	})

	f := NewFakeRunner()
	fh, err := f.Start(t.TempDir(), []string{"sh"}, nil)
	if err != nil {
		t.Fatalf("fake Start: %v", err)
	}
	fakeChild := fh.(TerminalHandle).Terminal()
	fakePreds := observe(fakeChild, func() { fakeChild.Exit(3) })

	if realPreds != fakePreds {
		t.Fatalf("terminal conformance drift:\n  real = %+v\n  fake = %+v", realPreds, fakePreds)
	}
	// A scenario where nothing was running and nothing exited would compare two
	// sets of zeroes and pass.
	if !realPreds.writeWhileRunningOK || realPreds.doneBeforeExit || !realPreds.doneAfterExit {
		t.Fatalf("the shared scenario never exercised a running-then-exited child: %+v", realPreds)
	}
}
