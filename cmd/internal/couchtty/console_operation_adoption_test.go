package couchtty

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// Synthetic completion tests explicitly execute both real stages. Production
// only prepares on lifecycle workers and reduces the final completion on Run.
func (c *Console) finishPreparedOperationForTest(completed operationCompletion) bool {
	return c.finishOperation(c.prepareOperationCompletion(completed))
}

func TestConsoleRunAcceptsInputWhileAdoptionHeldAndRemoteWaits(t *testing.T) {
	start, _ := attachStartResult(t, "adopting", menuAddress("adopting"))
	t.Cleanup(func() { _ = start.Handle.(couchcore.TerminalHandle).Terminal().Close() })
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) {
		c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
			if call.Name == "attach" {
				close(entered)
				<-release
				return c.ExecuteConsoleOperation(call)
			}
			return start, nil
		})
	})
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	finished := make(chan error, 1)
	if err := f.con.EnqueueRemoteOperation("held-adoption", "resume", func(ctx context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "resume", Context: ctx, Args: map[string]string{"repo-scope": start.Record.Thread.RepoScope, "tag": string(start.Record.Thread.Tag)}}, nil
	}, func() {}, func(_ any, err error) { finished <- err }); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-testwait.After(t):
		t.Fatal("adoption was not entered")
	}
	select {
	case err := <-finished:
		t.Fatalf("remote callback ran before adoption: %v", err)
	default:
	}
	written := len(f.child.Writes())
	if _, err := f.stdin.Write([]byte("z")); err != nil {
		t.Fatal(err)
	}
	testwait.Until(t, "ordinary input while a different slot adopts", func() bool { return len(f.child.Writes()) > written })
	once.Do(func() { close(release) })
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("adoption failed: %v", err)
		}
	case <-testwait.After(t):
		t.Fatal("remote callback did not follow adoption")
	}
}

func TestConsoleOperationCompletionSettlesExactlyOnce(t *testing.T) {
	f := newFixture(t, 24, 80)
	var finished atomic.Int32
	completed := operationRequest{origin: MenuOperationOrigin{Operation: "park"}, remote: &remoteOperation{
		finished: func(any, error) { finished.Add(1) },
	}}.complete(nil, nil)
	completed = f.con.prepareOperationCompletion(completed)
	f.con.finishOperation(completed)
	f.con.finishOperation(completed)
	f.con.abandonOperationCompletion(completed)
	if got := finished.Load(); got != 1 {
		t.Fatalf("completion callback count = %d, want 1", got)
	}
}

func TestConsoleStopSettlesAdoptingAndQueuedRemoteCalls(t *testing.T) {
	start, _ := attachStartResult(t, "stopping", menuAddress("stopping"))
	t.Cleanup(func() { _ = start.Handle.(couchcore.TerminalHandle).Terminal().Close() })
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) {
		c.queueWorkers = 1
		c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
			if call.Name == "attach" {
				close(entered)
				<-release
				return c.ExecuteConsoleOperation(call)
			}
			return start, nil
		})
	})
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	finished := make(chan error, 2)
	prepare := func(ctx context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "resume", Context: ctx}, nil
	}
	for _, key := range []string{"adopting", "queued"} {
		if err := f.con.EnqueueRemoteOperation(key, "resume", prepare, func() {}, func(_ any, err error) { finished <- err }); err != nil {
			t.Fatal(err)
		}
		if key == "adopting" {
			select {
			case <-entered:
			case <-testwait.After(t):
				t.Fatal("adoption was not entered")
			}
		}
	}
	f.con.Stop()
	once.Do(func() { close(release) })
	for i := 0; i < 2; i++ {
		select {
		case err := <-finished:
			if err == nil {
				t.Error("stopped remote call reported success")
			}
		case <-testwait.After(t):
			t.Fatal("stopped accepted remote caller was abandoned without completion")
		}
	}
	select {
	case <-f.done:
	case <-testwait.After(t):
		t.Fatal("Console did not join stopped adoption")
	}
	if start.Handle.(couchcore.TerminalHandle).Terminal().Endpoint().InputEnded() {
		t.Fatal("failed adoption closed caller-owned child instead of leaving typed rollback ownership")
	}
}
