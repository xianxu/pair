package couchtty

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// raiseBound runs one console test above the package's pinned bound of 1
// (pair#205 M2 review BR-9): the console reads LifecycleParallelism at
// construction (queue workers) and at arm time (the pass's Limit).
func raiseBound(t *testing.T, n int) {
	t.Helper()
	previous := couchcore.LifecycleParallelism
	couchcore.LifecycleParallelism = n
	t.Cleanup(func() { couchcore.LifecycleParallelism = previous })
}

func detachedFleet(root couchcore.ThreadAddress, n int) func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
	now := time.Now()
	return func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		rows := []couchcore.ActionableThreadSummary{{Address: root, State: couchcore.ThreadLive, LastActiveAt: now}}
		for i := 0; i < n; i++ {
			rows = append(rows, couchcore.ActionableThreadSummary{
				Address: menuAddress(fmt.Sprintf("couch-fleet-%d", i)), State: couchcore.ThreadDetached,
				LastActiveAt: now.Add(-time.Duration(i+1) * time.Minute),
			})
		}
		return rows, nil
	}
}

// Quit mid-pass above bound 1, end to end through the console's worker pool:
// the pass runs Limit attempts at once, Stop cancels every one of them, and no
// further attempt starts -- the parallel form of pair#206 cell 13.
func TestStopMidPassCancelsEveryParallelAttemptAndStartsNoMore(t *testing.T) {
	raiseBound(t, 3)
	f := newFixture(t, 24, 100)
	root := consoleThread(f, "c1")
	started := make(chan struct{}, 8)
	cancelled := make(chan struct{}, 8)
	var resumes int32
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		if call.Name != "resume" {
			return nil, nil
		}
		atomic.AddInt32(&resumes, 1)
		started <- struct{}{}
		<-call.Context.Done()
		cancelled <- struct{}{}
		return nil, call.Context.Err()
	})
	f.con.ArmReattachPass(root)
	f.con.SetActionableProvider(detachedFleet(root, 5))

	for i := 0; i < 3; i++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatalf("only %d of 3 parallel pass attempts started", i)
		}
	}
	f.con.Stop()
	for i := 0; i < 3; i++ {
		select {
		case <-cancelled:
		case <-time.After(5 * time.Second):
			t.Fatalf("Stop cancelled %d of the 3 attempts in flight", i)
		}
	}
	select {
	case <-f.done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after Stop")
	}
	if n := atomic.LoadInt32(&resumes); n != 3 {
		t.Fatalf("resumes = %d after Stop, want exactly the 3 in flight", n)
	}
}
