package couchcore

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFakeRunnerSignalDoesNotKill(t *testing.T) {
	// Real processes catch, ignore or delay SIGINT, and pair's restart/quit
	// loop depends on that. A fake that dies on any signal models a falsehood
	// that no live check would contradict.
	r := NewFakeRunner()
	h, err := r.Start("/repo", []string{"pair", "--layout2"}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !h.Alive() {
		t.Fatal("child must be alive after Start")
	}
	if err := h.Signal(os.Interrupt); err != nil {
		t.Fatalf("Signal: %v", err)
	}
	if !h.Alive() {
		t.Fatal("child must survive a signal it was not told to die on")
	}
}

func TestFakeRunnerKillAlwaysEndsExactChild(t *testing.T) {
	r := NewFakeRunner()
	h, err := r.Start("/repo", []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Signal(os.Kill); err != nil {
		t.Fatal(err)
	}
	if h.Alive() {
		t.Fatalf("killed child remained live: %+v", r.Child(h.ID()))
	}
	if h.Wait() != -1 || !r.Terminal(h.ID()).Done() {
		t.Fatalf("killed child did not complete as killed: %+v", r.Child(h.ID()))
	}
}

func TestFakeRunnerSetExitedEndsChildAndUnblocksWait(t *testing.T) {
	r := NewFakeRunner()
	h, _ := r.Start("/repo", []string{"pair"}, nil)

	got := make(chan int, 1)
	go func() { got <- h.Wait() }()

	select {
	case code := <-got:
		t.Fatalf("Wait returned %d before the child exited", code)
	case <-time.After(50 * time.Millisecond):
	}

	r.SetExited(h.ID(), 3)
	select {
	case code := <-got:
		if code != 3 {
			t.Fatalf("Wait = %d, want 3", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SetExited did not unblock Wait")
	}
	if h.Alive() {
		t.Fatal("child must be dead after SetExited")
	}
}

func TestFakeRunnerHandleWritesBackToRunnerOps(t *testing.T) {
	// The handle must record into the Runner's log, not its own: a contract
	// where each child keeps a private signal log makes this ordering
	// assertion impossible.
	r := NewFakeRunner()
	h, _ := r.Start("/repo", []string{"pair", "--layout2"}, nil)
	_ = h.Signal(os.Interrupt)
	want := []string{"start /repo: pair --layout2", "signal couch-fake-1: interrupt"}
	if !reflect.DeepEqual(r.Ops, want) {
		t.Fatalf("Ops = %v, want %v", r.Ops, want)
	}
}

func TestFakeRunnerRecordsArgvDirAndEnv(t *testing.T) {
	r := NewFakeRunner()
	h, _ := r.Start("/repo/sub", []string{"pair"}, []string{"K=V"})
	c := r.Child(h.ID())
	if c.Dir != "/repo/sub" || len(c.Env) != 1 || c.Env[0] != "K=V" {
		t.Fatalf("child = %+v", c)
	}
}

func TestExecRunnerPropagatesExitCode(t *testing.T) {
	h, err := ExecRunner{}.Start(t.TempDir(), []string{"sh", "-c", "exit 3"}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if code := h.Wait(); code != 3 {
		t.Fatalf("Wait = %d, want 3", code)
	}
	if h.PID() == 0 {
		t.Fatal("PID must be recorded")
	}
}

func TestExecRunnerChildEnvironmentOverridesInheritedValue(t *testing.T) {
	t.Setenv("PAIR_USE_REPO_DEFAULT", "1")
	output := filepath.Join(t.TempDir(), "raw-child-env")
	capture, err := os.Create(output)
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = capture
	h, startErr := ExecRunner{}.Start(t.TempDir(), []string{"env"}, []string{"PAIR_USE_REPO_DEFAULT="})
	os.Stdout = oldStdout
	if startErr != nil {
		_ = capture.Close()
		t.Fatal(startErr)
	}
	if code := h.Wait(); code != 0 {
		t.Fatalf("Wait = %d", code)
	}
	if err := capture.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var entries []string
	for _, item := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if strings.HasPrefix(item, "PAIR_USE_REPO_DEFAULT=") {
			entries = append(entries, item)
		}
	}
	if !reflect.DeepEqual(entries, []string{"PAIR_USE_REPO_DEFAULT="}) {
		t.Fatalf("raw child policy environment = %q, want one authoritative empty entry", entries)
	}
}

func TestMergeChildEnvironmentMakesSuppliedKeysUniqueAndAuthoritative(t *testing.T) {
	got := mergeChildEnvironment(
		[]string{"KEEP=parent", "PAIR_USE_REPO_DEFAULT=1", "DUP=parent"},
		[]string{"PAIR_USE_REPO_DEFAULT=", "DUP=first", "DUP=last"},
	)
	want := []string{"KEEP=parent", "PAIR_USE_REPO_DEFAULT=", "DUP=last"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeChildEnvironment = %q, want %q", got, want)
	}
}

func TestExecRunnerBuildsCommandWithUniqueAuthoritativeChildEnvironment(t *testing.T) {
	t.Setenv("PAIR_USE_REPO_DEFAULT", "1")
	cmd, err := buildExecCommand(t.TempDir(), []string{"env"}, []string{"PAIR_USE_REPO_DEFAULT="}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var entries []string
	for _, item := range cmd.Env {
		if strings.HasPrefix(item, "PAIR_USE_REPO_DEFAULT=") {
			entries = append(entries, item)
		}
	}
	if !reflect.DeepEqual(entries, []string{"PAIR_USE_REPO_DEFAULT="}) {
		t.Fatalf("raw exec.Cmd policy environment = %q, want one authoritative empty entry", entries)
	}
}

func TestExecRunnerBlockedStartRunsTargetOnlyAfterAcknowledgement(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "target-ran")
	r := ExecRunner{LaunchHelper: os.Args[0]}
	h, err := r.StartBlocked(t.TempDir(), []string{"sh", "-c", "printf exec > \"$PAIR_TEST_TARGET_MARKER\""}, []string{
		"PAIR_TEST_RUNNER_HELPER=1",
		"PAIR_TEST_TARGET_MARKER=" + marker,
	}, 2*time.Second)
	if err != nil {
		t.Fatalf("StartBlocked: %v", err)
	}
	time.Sleep(75 * time.Millisecond)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("target ran before acknowledgement: %v", err)
	}
	if err := h.Acknowledge(); err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	if code := h.Wait(); code != 0 {
		t.Fatalf("Wait = %d", code)
	}
	if raw, err := os.ReadFile(marker); err != nil || string(raw) != "exec" {
		t.Fatalf("target marker = %q, %v", raw, err)
	}
}

func TestExecRunnerBlockedStartCancelNeverRunsTarget(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "target-ran")
	r := ExecRunner{LaunchHelper: os.Args[0]}
	h, err := r.StartBlocked(t.TempDir(), []string{"sh", "-c", "printf exec > \"$PAIR_TEST_TARGET_MARKER\""}, []string{
		"PAIR_TEST_RUNNER_HELPER=1",
		"PAIR_TEST_TARGET_MARKER=" + marker,
	}, 2*time.Second)
	if err != nil {
		t.Fatalf("StartBlocked: %v", err)
	}
	if err := h.Cancel(); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if code := h.Wait(); code == 0 {
		t.Fatal("cancelled helper exited successfully")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("cancelled helper ran target: %v", err)
	}
}

// One child, ONE notion of exited. The fake used to end its handle while its
// terminal double kept running, which hung a console test rather than failing
// it -- the same divergence class as BR-18, at a different seam.
func TestFakeRunnerExitEndsTheHandleAndTheTerminalTogether(t *testing.T) {
	f := NewFakeRunner()
	h, err := f.Start("/repo", []string{"pair"}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	child := h.(TerminalHandle).Terminal()

	if child.Done() {
		t.Fatal("the terminal double reports exited before the child did")
	}
	f.SetExited(h.ID(), 5)

	if h.Alive() {
		t.Fatal("handle still alive after SetExited")
	}
	if !child.Done() {
		t.Fatal("the terminal double is still running after the handle exited")
	}
	if got := child.Wait(); got != 5 {
		t.Fatalf("terminal Wait() = %d, want the handle's code 5", got)
	}
}

// AutoExit models "the child ran and exited"; both halves must reflect that or
// a CLI test that relies on it hangs.
func TestFakeRunnerAutoExitEndsTheTerminalToo(t *testing.T) {
	f := NewFakeRunner()
	f.AutoExit(0)
	h, err := f.Start("/repo", []string{"pair"}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !h.(TerminalHandle).Terminal().Done() {
		t.Fatal("AutoExit left the terminal double running")
	}
}

func TestFakeRunnerBlockedStartDoesNotExecUntilAcknowledged(t *testing.T) {
	f := NewFakeRunner()
	h, err := f.StartBlocked("/repo", []string{"pair", "resume", "tag"}, []string{"K=V"}, time.Second)
	if err != nil {
		t.Fatalf("StartBlocked: %v", err)
	}
	before := f.Child(h.ID())
	if !before.Blocked || before.ExecCount != 0 {
		t.Fatalf("before ack = %+v", before)
	}
	if err := h.Acknowledge(); err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	after := f.Child(h.ID())
	if after.Blocked || after.ExecCount != 1 {
		t.Fatalf("after ack = %+v", after)
	}
	if err := h.Acknowledge(); err == nil || f.Child(h.ID()).ExecCount != 1 {
		t.Fatalf("second acknowledgement = %v, child=%+v", err, f.Child(h.ID()))
	}
}

func TestFakeRunnerBlockedStartCancelNeverExecs(t *testing.T) {
	f := NewFakeRunner()
	h, err := f.StartBlocked("/repo", []string{"pair"}, nil, time.Second)
	if err != nil {
		t.Fatalf("StartBlocked: %v", err)
	}
	if err := h.Cancel(); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	child := f.Child(h.ID())
	if child.ExecCount != 0 || child.alive {
		t.Fatalf("cancelled child = %+v", child)
	}
}
