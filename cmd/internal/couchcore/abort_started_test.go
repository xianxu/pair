package couchcore

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestAbortStartedQuiescesExactHandleAndReconcilesOwnership(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record, handle, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	cause := errors.New("attach failed")
	err = env.Couch.AbortStarted(StartResult{Record: record, Handle: handle}, cause)
	if !errors.Is(err, cause) {
		t.Fatalf("AbortStarted error = %v, want original cause", err)
	}
	if handle.Alive() {
		t.Fatal("aborted exact handle remained alive")
	}
	if got := env.Couch.reg.Records(); len(got) != 0 {
		t.Fatalf("aborted actor remained in registry: %+v", got)
	}
	if got := env.Artifacts.Quiesces(); !slices.Equal(got, []ThreadAddress{record.Thread}) {
		t.Fatalf("exact Pair quiescence = %+v, want %+v", got, record.Thread)
	}
	thread, getErr := env.Couch.Threads.GetThread(record.Thread)
	if getErr != nil || len(thread.Incarnations) != 1 || thread.Incarnations[0].State != IncarnationUnknown {
		t.Fatalf("reconciled durable thread = %+v, %v", thread, getErr)
	}
}

func TestAbortStartedRefusesMismatchedHandleIdentityWithoutEffects(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record, handle, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	forged := record
	forged.Identity = "different-process-start"
	err = env.Couch.AbortStarted(StartResult{Record: forged, Handle: handle}, errors.New("attach failed"))
	if err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("AbortStarted mismatch error = %v", err)
	}
	if !handle.Alive() || len(env.Artifacts.Quiesces()) != 0 || len(env.Couch.reg.Records()) != 1 {
		t.Fatalf("identity refusal changed ownership: child=%+v quiesces=%v registry=%+v",
			env.Runner.Child(handle.ID()), env.Artifacts.Quiesces(), env.Couch.reg.Records())
	}
	_ = handle.Signal(os.Kill)
}

func TestAbortStartedRefusesUnregisteredOrNilHandle(t *testing.T) {
	env := newTestEnv(t, "/repo")
	if err := env.Couch.AbortStarted(StartResult{}, errors.New("attach failed")); err == nil {
		t.Fatal("nil StartResult was accepted")
	}
	handle, err := env.Runner.Start("/repo", []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	record := ActorRecord{
		ID: "not-registered", Thread: ThreadAddress{RepoScope: "repo", Tag: "couch-missing"},
		Args: StartArgs{Worktree: "/repo"}, PID: handle.PID(), Identity: handle.Identity(),
	}
	if err := env.Couch.AbortStarted(StartResult{Record: record, Handle: handle}, errors.New("attach failed")); err == nil {
		t.Fatal("unregistered StartResult was accepted")
	}
	if !handle.Alive() {
		t.Fatal("unregistered handle was killed")
	}
	_ = handle.Signal(os.Kill)
}
