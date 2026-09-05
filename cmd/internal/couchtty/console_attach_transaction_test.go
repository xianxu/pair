package couchtty

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func attachStartResult(t *testing.T, id couchcore.ActorID, address couchcore.ThreadAddress) (couchcore.StartResult, *couchcore.FakeRunner) {
	t.Helper()
	runner := couchcore.NewFakeRunner()
	handle, err := runner.Start("/repo", []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return couchcore.StartResult{Record: couchcore.ActorRecord{
		ID: id, Thread: address, Args: couchcore.StartArgs{Worktree: "/repo"},
		PID: handle.PID(), Identity: handle.Identity(),
	}, Handle: handle}, runner
}

func attachCall(ctx context.Context, start couchcore.StartResult) couchcore.OperationCall {
	return couchcore.OperationCall{
		Name: "attach", Context: ctx, Implicit: true, TypedPayload: start,
		Args: map[string]string{"repo-scope": start.Record.Thread.RepoScope, "tag": string(start.Record.Thread.Tag)},
	}
}

func TestConsoleAttachRollbackRejectsExitedChildWithoutChangingRouting(t *testing.T) {
	f := newFixture(t, 24, 80)
	address := couchcore.ThreadAddress{RepoScope: "repo", Tag: "couch-dead"}
	start, runner := attachStartResult(t, "dead-actor", address)
	runner.SetExited(start.Handle.ID(), 7)
	f.con.mu.Lock()
	wantOrder := append([]string(nil), f.con.order...)
	wantActive := f.con.active
	f.con.mu.Unlock()

	if _, err := f.con.ExecuteConsoleOperation(attachCall(context.Background(), start)); err == nil {
		t.Fatal("attach accepted an already-exited child")
	}
	f.con.mu.Lock()
	gotOrder, gotActive := append([]string(nil), f.con.order...), f.con.active
	_, installed := f.con.panes[start.Handle.ID()]
	f.con.mu.Unlock()
	if installed || !reflect.DeepEqual(gotOrder, wantOrder) || gotActive != wantActive {
		t.Fatalf("failed attach changed routing: installed=%t order=%v active=%q", installed, gotOrder, gotActive)
	}
}

func TestConsoleStopDuringAttachRefusesPartialPaneAndKeepsTerminalRestored(t *testing.T) {
	f := newFixture(t, 24, 80)
	address := couchcore.ThreadAddress{RepoScope: "repo", Tag: "couch-stopped"}
	start, _ := attachStartResult(t, "stopped-actor", address)
	entered := make(chan struct{})
	attachDone := make(chan error, 1)
	// Hold the commit mutex so Stop occurs after the attach call begins but
	// before it can install routing state or reserve its watcher.
	f.con.mu.Lock()
	go func() {
		close(entered)
		_, err := f.con.ExecuteConsoleOperation(attachCall(context.Background(), start))
		attachDone <- err
	}()
	<-entered
	f.con.Stop()
	f.con.mu.Unlock()
	select {
	case err := <-attachDone:
		if err == nil {
			t.Fatal("stopped Console accepted a partial pane")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("attach did not retire after concurrent Stop")
	}
	select {
	case <-f.done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Console.Run did not finish Stop before attach")
	}
	assertConsoleRestored(t, f)
	f.con.mu.Lock()
	_, installed := f.con.panes[start.Handle.ID()]
	f.con.mu.Unlock()
	if installed {
		t.Fatal("stopped attach left a pane installed")
	}
}
