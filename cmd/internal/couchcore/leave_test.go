package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// leaveFleet is n live, detachable threads under one Couch, in snapshot order.
func leaveFleet(t *testing.T, n int) (*Couch, *FakeProcOps, []ThreadRecord) {
	t.Helper()
	store, ns := newTestThreadStore(t)
	artifacts := NewFakeThreadArtifactCollisionChecker()
	proc := NewFakeProcOps()
	threads := make([]ThreadRecord, 0, n)
	for i := 0; i < n; i++ {
		record := validThreadRecord(t)
		record.Address.Tag = ThreadTag(fmt.Sprintf("couch-%016x", 0xa000+i))
		record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
		record.Reservation = false
		profile := LaunchProfile{Agent: "codex"}
		pid := 100 + i
		record.Incarnations = []ThreadIncarnation{{PID: pid, Identity: fmt.Sprintf("pair-%d", i), State: IncarnationLive, LaunchProfile: &profile}}
		record.LatestLaunchProfile = &profile
		created, err := store.CreateThread(record)
		if err != nil {
			t.Fatal(err)
		}
		artifacts.SetPairSession(created.Address, fmt.Sprintf("pair-session-%d", i), true)
		proc.Set(pid, fmt.Sprintf("pair-%d", i))
		proc.DiesOn[pid] = syscall.SIGTERM
		threads = append(threads, created)
	}
	c := &Couch{Threads: store, Proc: proc, Artifacts: artifacts, Clock: FixedClock{T: time.Unix(100, 0).UTC()}, sleep: func(time.Duration) {}}
	return c, proc, threads
}

func withParallelism(t *testing.T, n int) {
	t.Helper()
	previous := LifecycleParallelism
	LifecycleParallelism = n
	t.Cleanup(func() { LifecycleParallelism = previous })
}

// Leave drives threads concurrently, never more than the bound at once
// (pair#205 M2).
func TestLeaveDetachesThreadsConcurrentlyUpToTheBound(t *testing.T) {
	withParallelism(t, 2)
	c, proc, threads := leaveFleet(t, 5)
	var mu sync.Mutex
	inFlight, peak := 0, 0
	reachedBound := make(chan struct{})
	var once sync.Once
	release := make(chan struct{})
	proc.OnSignal = func(int, os.Signal) {
		mu.Lock()
		inFlight++
		if inFlight > peak {
			peak = inFlight
		}
		if inFlight == 2 {
			once.Do(func() { close(reachedBound) })
		}
		mu.Unlock()
		<-release
		mu.Lock()
		inFlight--
		mu.Unlock()
	}
	done := make(chan error, 1)
	var result LeaveResult
	go func() {
		var err error
		result, err = c.Leave(context.Background(), LeaveDetach)
		done <- err
	}()
	select {
	case <-reachedBound:
	case <-time.After(5 * time.Second):
		t.Fatal("Leave never had two detaches in flight at once")
	}
	time.Sleep(30 * time.Millisecond) // a third would arrive now if the bound leaked
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if peak != 2 {
		t.Fatalf("peak concurrent detaches = %d, want the bound 2", peak)
	}
	if len(result.Detached) != len(threads) {
		t.Fatalf("detached %d of %d threads: %+v", len(result.Detached), len(threads), result)
	}
}

// One thread's failure neither stops its siblings nor hides itself (D6).
func TestLeaveFinishesSiblingsWhenOneFails(t *testing.T) {
	withParallelism(t, 3)
	c, proc, threads := leaveFleet(t, 4)
	stubborn := threads[1]
	delete(proc.DiesOn, stubborn.Incarnations[0].PID) // ignores SIGTERM: its detach fails
	result, err := c.Leave(context.Background(), LeaveDetach)
	if err == nil || !strings.Contains(err.Error(), string(stubborn.Address.Tag)) {
		t.Fatalf("Leave err = %v, want it to name %s", err, stubborn.Address.Tag)
	}
	if len(result.Detached) != 3 {
		t.Fatalf("detached = %+v, want the three healthy siblings", result.Detached)
	}
	for _, address := range result.Detached {
		if address == stubborn.Address {
			t.Fatal("the failed thread was reported detached")
		}
	}
}

// The operator's report keeps snapshot order whatever order the work ends in.
func TestLeaveResultIsInSnapshotOrder(t *testing.T) {
	withParallelism(t, 4)
	c, proc, threads := leaveFleet(t, 4)
	// The first thread's detach finishes last.
	firstPID := threads[0].Incarnations[0].PID
	others := make(chan struct{}, 3)
	proc.OnSignal = func(pid int, _ os.Signal) {
		if pid == firstPID {
			for i := 0; i < 3; i++ {
				select {
				case <-others:
				case <-time.After(2 * time.Second):
				}
			}
			return
		}
		others <- struct{}{}
	}
	result, err := c.Leave(context.Background(), LeaveDetach)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := c.Threads.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	var want []ThreadAddress
	for _, record := range snapshot.Records {
		want = append(want, record.Address)
	}
	if fmt.Sprint(result.Detached) != fmt.Sprint(want) {
		t.Fatalf("detached order = %v, want snapshot order %v", result.Detached, want)
	}
}

// Counted invariant (pair#205; #204's suite is not built yet): Leave of N
// threads sends exactly one SIGTERM per thread, however many run at once --
// fan-out must not repeat or skip work.
func TestLeaveSignalsEachThreadExactlyOnce(t *testing.T) {
	withParallelism(t, 3)
	c, proc, threads := leaveFleet(t, 7)
	if _, err := c.Leave(context.Background(), LeaveDetach); err != nil {
		t.Fatal(err)
	}
	for _, thread := range threads {
		pid := thread.Incarnations[0].PID
		if got := len(proc.Signals[pid]); got != 1 {
			t.Errorf("%s got %d signals, want exactly 1", thread.Address.Tag, got)
		}
	}
}

// Cancelling Leave partway through its fan-out stops STARTING threads: only the
// ones already started were ever signalled, and the cancellation is reported
// (D6, M2 review). The started ones see the cancellation through their ctx.
func TestLeaveCancelledMidFanOutStartsNoFurtherThread(t *testing.T) {
	withParallelism(t, 2)
	c, proc, threads := leaveFleet(t, 5)
	ctx, cancel := context.WithCancel(context.Background())
	inFlight := make(chan struct{}, 5)
	release := make(chan struct{})
	proc.OnSignal = func(int, os.Signal) {
		inFlight <- struct{}{}
		<-release
	}
	done := make(chan error, 1)
	var result LeaveResult
	go func() {
		var err error
		result, err = c.Leave(ctx, LeaveDetach)
		done <- err
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-inFlight:
		case <-time.After(5 * time.Second):
			t.Fatalf("only %d of 2 detaches started", i)
		}
	}
	cancel()
	close(release)
	err := <-done
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Leave = %v, want the cancellation", err)
	}
	if len(result.Detached) > 2 {
		t.Fatalf("detached = %v, want at most the 2 already started", result.Detached)
	}
	signalled := 0
	for _, thread := range threads {
		signalled += len(proc.Signals[thread.Incarnations[0].PID])
	}
	if signalled != 2 {
		t.Fatalf("%d threads signalled after cancellation, want only the 2 started", signalled)
	}
}
