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
)

func liveOnly(t *testing.T) {
	t.Helper()
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 to run against real processes and real git")
	}
}

// waitFile polls for a readiness marker. ExecRunner.Start returns as soon as
// cmd.Start() succeeds, which is BEFORE the shell has reached its trap -- so
// signalling immediately is a genuine race, not a slow machine. A sleep would
// paper over it; a marker file does not.
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

	// The registry must treat them as two trees, which is what makes
	// worktree-parallel work at all.
	reg, err := NewRegistry().Register(ActorRecord{ID: "a", Args: StartArgs{Worktree: primaryRoot}})
	if err != nil {
		t.Fatalf("register primary: %v", err)
	}
	if _, err := reg.Register(ActorRecord{ID: "b", Args: StartArgs{Worktree: linkedRoot}}); err != nil {
		t.Fatalf("linked worktree refused against a real repo: %v", err)
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
