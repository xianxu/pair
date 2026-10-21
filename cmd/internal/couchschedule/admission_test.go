package couchschedule

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestAdmissionMachinePermitsAndTaskExclusion(t *testing.T) {
	root := t.TempDir()
	permits := t.TempDir()
	request := func(task string) ExecutionRequest {
		return ExecutionRequest{Namespace: t.TempDir(), HubRoot: root, Task: task, PermitsDir: permits}
	}
	first := request("one")
	second := request("two")
	third := request("three")
	a, err := acquireExecutionPermits(t.Context(), first)
	if err != nil {
		t.Fatal(err)
	}
	defer a()
	b, err := acquireExecutionPermits(t.Context(), second)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := acquireExecutionPermits(t.Context(), request("one")); err == nil {
		other()
		t.Fatal("same task entered another namespace")
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	type admitted struct {
		release func()
		err     error
	}
	done := make(chan admitted, 1)
	go func() { release, err := acquireExecutionPermits(ctx, third); done <- admitted{release, err} }()
	// The third task's own lease proves it reached admission. Neither of the
	// two machine permits can then be available while both peers hold theirs.
	deadline := time.Now().Add(time.Second)
	for {
		release, taken, err := executionLease(filepath.Join(permits, "tasks", (TaskKey{Hub: root, Task: "three"}).ID()+".lock"))
		if err != nil {
			t.Fatal(err)
		}
		if !taken {
			break
		}
		release()
		if time.Now().After(deadline) {
			t.Fatal("third caller never reached admission")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case got := <-done:
		if got.release != nil {
			got.release()
		}
		t.Fatal("third machine execution admitted")
	default:
	}
	b()
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatal(got.err)
		}
		got.release()
	case <-time.After(time.Second):
		t.Fatal("released machine permit did not admit third")
	}
}

func TestAdmissionReleaseCannotUnlockReusedDescriptor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lease")
	first, ok, err := executionLease(path)
	if err != nil || !ok {
		t.Fatal(err)
	}
	first()
	second, ok, err := executionLease(path)
	if err != nil || !ok {
		t.Fatal(err)
	}
	defer second()
	first()
	third, ok, err := executionLease(path)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		third()
		t.Fatal("repeated release unlocked a subsequent lease")
	}
}
