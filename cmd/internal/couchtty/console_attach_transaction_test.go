package couchtty

import (
	"context"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// The presenter's existing resize callback gives this test a real FIFO barrier:
// an install may resize its child, but Register cannot finish until release.
func TestConsoleAttachPendingRegistrationReleasesMutexAndReservesIdentity(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	selected := ptychild.NewFakeChild(nil)
	if err := c.presenter.Select(context.Background(), selected.Endpoint(), terminal.Geometry{Cols: 80, Rows: 24}, nil); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	resizeDone := make(chan error, 1)
	go func() {
		resizeDone <- c.presenter.Resize(context.Background(), terminal.Geometry{Cols: 81, Rows: 24}, func(terminal.Geometry) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	child := ptychild.NewFakeChild(nil)
	wantSize := ptychild.Size{Rows: 23, Cols: 80}
	address := menuAddress("pending-registration")
	done := make(chan error, 1)
	go func() {
		done <- c.installObservedThreadActor(context.Background(), "pending", "pending-actor", address, "/repo", "repo", child, couchcore.ProcessIdentity{}, true)
	}()
	t.Cleanup(func() {
		close(release)
		<-resizeDone
		if err := <-done; err != nil {
			t.Errorf("registration did not commit after release: %v", err)
		}
		if child.Size() != wantSize {
			t.Errorf("installation committed stale geometry: got %+v, want %+v", child.Size(), wantSize)
		}
		c.Stop()
		c.workers.Wait()
		_ = c.release()
		_ = child.Close()
		_ = selected.Close()
	})
	testwait.Until(t, "installer child resize before pending registration", func() bool { return child.Size() == (ptychild.Size{Rows: 23, Cols: 80}) })
	if !c.mu.TryLock() {
		t.Fatal("pending presenter registration holds Console.mu and blocks unrelated input")
	}
	c.mu.Unlock()
	for _, tc := range []struct {
		name, handle string
		address      couchcore.ThreadAddress
		sameChild    bool
	}{
		{"same-handle", "pending", menuAddress("another-thread"), false},
		{"same-thread", "different-handle", address, false},
		{"same-child", "different-handle", menuAddress("another-thread"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other := child
			if !tc.sameChild {
				other = ptychild.NewFakeChild(nil)
				defer other.Close()
			}
			refused := make(chan error, 1)
			go func() {
				refused <- c.installObservedThreadActor(context.Background(), tc.handle, "another-actor", tc.address, "/repo", "repo", other, couchcore.ProcessIdentity{}, true)
			}()
			select {
			case err := <-refused:
				if err == nil {
					t.Fatal("duplicate pending installation was accepted")
				}
			case <-testwait.After(t):
				t.Fatal("duplicate waited behind presenter instead of refusing the reservation")
			}
			if other.Endpoint().InputEnded() {
				t.Fatal("refused installer closed the caller-owned child")
			}
		})
	}
	// An ordinary host resize may happen while Register is queued. The
	// installation must re-prepare to the latest size before exposing its pane.
	c.mu.Lock()
	c.size = ptychild.Size{Rows: 26, Cols: 91}
	c.mu.Unlock()
	wantSize = ptychild.Size{Rows: 25, Cols: 91}
}

func TestConsoleAttachStopDuringPendingRegistrationKeepsCallerOwnership(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	selected := ptychild.NewFakeChild(nil)
	if err := c.presenter.Select(context.Background(), selected.Endpoint(), terminal.Geometry{Cols: 80, Rows: 24}, nil); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	resizeDone := make(chan error, 1)
	go func() {
		resizeDone <- c.presenter.Resize(context.Background(), terminal.Geometry{Cols: 81, Rows: 24}, func(terminal.Geometry) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	child := ptychild.NewFakeChild(nil)
	done := make(chan error, 1)
	go func() {
		done <- c.installObservedThreadActor(context.Background(), "pending", "pending-actor", menuAddress("pending-registration"), "/repo", "repo", child, couchcore.ProcessIdentity{}, true)
	}()
	testwait.Until(t, "installer child resize before stop", func() bool { return child.Size() == (ptychild.Size{Rows: 23, Cols: 80}) })
	c.Stop()
	joined := make(chan struct{})
	go func() { c.workers.Wait(); close(joined) }()
	close(release)
	<-resizeDone
	var installErr error
	select {
	case installErr = <-done:
	case <-testwait.After(t):
		t.Fatal("stopped installation did not finish")
	}
	select {
	case <-joined:
	case <-testwait.After(t):
		t.Fatal("stopped installation did not release its lifetime worker")
	}
	t.Cleanup(func() { _ = c.release(); _ = child.Close(); _ = selected.Close() })
	if installErr == nil {
		t.Error("installation committed after Stop during registration")
	}
	if child.Endpoint().InputEnded() {
		t.Error("rejected installer closed caller-owned child")
	}
	c.mu.Lock()
	_, installed := c.panes["pending"]
	_, owned := c.ownedChildren[child]
	c.mu.Unlock()
	if installed || owned {
		t.Errorf("rejected installer retained child: pane=%t owned=%t", installed, owned)
	}
	if err := c.presenter.Retire(context.Background(), child.Endpoint()); err == nil {
		t.Error("rejected installation left its registration in presenter")
	}
}

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
	case <-testwait.After(t):
		t.Fatal("attach did not retire after concurrent Stop")
	}
	select {
	case <-f.done:
	case <-testwait.After(t):
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

func TestSwitchWithoutLocalPaneResumesExactThread(t *testing.T) {
	address := menuAddress("surviving-terminal")
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) {
		c.menu = NewMenuState([]couchcore.ActionableThreadSummary{{Address: address, State: couchcore.ThreadLive}}, address)
	})
	started, _ := attachStartResult(t, "surviving-client", address)
	calls := 0
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		calls++
		if call.Name != "resume" || call.Args["repo-scope"] != address.RepoScope || call.Args["tag"] != string(address.Tag) || call.Args["warm-only"] != "true" {
			t.Fatalf("unexpected fallback: %+v", call)
		}
		return started, nil
	})
	value, err := f.con.ExecuteConsoleOperation(couchcore.OperationCall{Name: "switch", Context: context.Background(), Args: map[string]string{"repo-scope": address.RepoScope, "tag": string(address.Tag)}})
	if err != nil || calls != 1 {
		t.Fatalf("Continue without pane: calls=%d err=%v", calls, err)
	}
	result, ok := value.(couchcore.StartResult)
	if !ok || result.Handle != started.Handle {
		t.Fatalf("missing attach completion: %#v", value)
	}
}
