package couchtty

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestConsoleStopCancelsOperationAndJoinsWorker(t *testing.T) {
	for _, operation := range []string{"park", "name"} {
		t.Run(operation, func(t *testing.T) {
			f := newFixture(t, 24, 80)
			started := make(chan struct{})
			finished := make(chan error, 1)
			releaseMissingContext := make(chan struct{})
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(releaseMissingContext) }) }
			defer release()
			calls := 0
			f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
				calls++
				close(started)
				if call.Context == nil {
					<-releaseMissingContext
					finished <- errors.New("operation context was nil")
					return nil, errors.New("operation context was nil")
				}
				<-call.Context.Done()
				finished <- call.Context.Err()
				return nil, call.Context.Err()
			})
			origin := MenuOperationOrigin{
				Operation: operation, Attempt: 1, Address: menuAddress("couch-cancel"),
				FrameInstance: 1, FrameKind: MenuFrameRoot, Depth: 1,
			}
			f.con.mu.Lock()
			f.con.menu = NewMenuState(menuThreads(), menuAddress("couch-one"))
			f.con.menu.InFlight = origin
			f.con.menuReady = true
			f.con.mu.Unlock()
			f.con.runMenuOperation(MenuEffect{
				Operation: operation, Attempt: 1,
				Args: map[string]string{"repo-scope": "repo", "tag": "couch-cancel"},
			})
			select {
			case <-started:
			case <-time.After(250 * time.Millisecond):
				t.Fatal("accepted operation did not start")
			}

			f.con.Stop()
			select {
			case err := <-finished:
				if !errors.Is(err, context.Canceled) {
					release()
					t.Fatalf("blocked dispatcher finished with %v, want context cancellation", err)
				}
			case <-time.After(250 * time.Millisecond):
				release()
				t.Fatal("Console.Stop did not cancel blocked dispatcher")
			}
			select {
			case <-f.done:
			case <-time.After(250 * time.Millisecond):
				release()
				t.Fatal("Console.Run did not join blocked operation worker")
			}
			if calls != 1 {
				t.Fatalf("dispatcher calls = %d, want one", calls)
			}
		})
	}
}

func TestConsoleSnapshotsExactPaneObservations(t *testing.T) {
	f := newFixture(t, 24, 80)
	address := couchcore.ThreadAddress{RepoScope: "repo", Tag: "couch-exact"}
	process := couchcore.ProcessIdentity{PID: 42, Identity: "pid-start:exact"}
	child := ptychild.NewFakeChild(nil)
	child.SetSink(func(chunk []byte) { f.con.Deliver("exact", chunk) })
	f.con.attachObservedThreadActor("exact", "actor-exact", address, "/repo", "exact", child, process)

	got := f.con.snapshotMenuObservations()
	want := []couchcore.LiveTTYObservation{{Address: address, Process: process}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pane observations = %+v, want %+v", got, want)
	}
}

func TestMenuOpenDoesNotWaitForActionableProvider(t *testing.T) {
	f := newFixture(t, 24, 80)
	started := make(chan struct{}, 1)
	canceled := make(chan struct{}, 2)
	f.con.SetActionableProvider(func(ctx context.Context, observations []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-ctx.Done()
		canceled <- struct{}{}
		return nil, ctx.Err()
	})
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	f.host.Reset()

	_, _ = f.stdin.Write([]byte("\x00"))
	waitUpTo(t, 250*time.Millisecond, "menu repaint while inventory is blocked", func() bool {
		return strings.Contains(f.host.Written(), "threads")
	})
	select {
	case <-started:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("menu open did not request actionable inventory")
	}

	_, _ = f.stdin.Write([]byte("x"))
	waitUpTo(t, 250*time.Millisecond, "filter repaint while inventory is blocked", func() bool {
		return strings.Contains(f.host.Written(), "filter: x")
	})
	f.con.Stop()
	_ = f.stdin.Close()
	select {
	case <-canceled:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Console.Stop did not cancel blocked actionable inventory")
	}
	select {
	case <-f.done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Console.Run did not join blocked actionable inventory")
	}
}

func TestConsoleActionableObservationResultEntersMenuReducer(t *testing.T) {
	f := newFixture(t, 24, 80)
	address := couchcore.ThreadAddress{RepoScope: "repo", Tag: "couch-result"}
	want := []couchcore.ActionableThreadSummary{{
		Address: address, WorkingPath: "/repo", Name: "result", State: couchcore.ThreadParked,
	}}
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return want, nil
	})

	waitUpTo(t, 250*time.Millisecond, "actionable result", func() bool {
		state := f.con.menuSnapshot()
		return state.InventoryReady && !state.RefreshPending && reflect.DeepEqual(state.Inventory, want)
	})
}
