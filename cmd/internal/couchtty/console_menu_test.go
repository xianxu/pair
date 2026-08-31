package couchtty

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

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
	legacyRead := make(chan string, 2)
	f.con.SetSummaries(func() ([]couchcore.ThreadSummary, error) {
		legacyRead <- "summaries"
		return nil, nil
	})
	f.con.SetResolver(func(string) ([]couchcore.ThreadAddress, error) {
		legacyRead <- "resolver"
		return nil, nil
	})
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
	waitUpTo(t, 250*time.Millisecond, "panel repaint while inventory is blocked", func() bool {
		return strings.Contains(f.host.Written(), "couch — actors")
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
	select {
	case call := <-legacyRead:
		t.Fatalf("menu keystroke performed synchronous legacy %s I/O", call)
	default:
	}

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
