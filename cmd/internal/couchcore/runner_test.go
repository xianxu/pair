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
