package couchcore

import (
	"os"
	"reflect"
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
