package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestParkDefiniteUnpublishedFailureRollsBack(t *testing.T) {
	env, live := envWithLiveThread(t)
	env.Lifecycle.publishErr = errors.New("definite unpublished request failure")
	_, err := env.Couch.PairLifecycle.Park(context.Background(), live.Address)
	if err == nil {
		t.Fatal("lost publication error")
	}
	after, getErr := env.Couch.Threads.GetThread(live.Address)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if after.Park != nil {
		t.Fatalf("unpublished park still blocks the live child: %+v", after.Park)
	}
	if len(after.Incarnations) != 1 || after.Incarnations[0].PID != 42 || after.VerifiedPark != nil {
		t.Fatalf("rollback changed child or fabricated success: %+v", after)
	}
	if len(env.Artifacts.TriggeredQuits()) != 0 {
		t.Fatal("rollback sent a quit")
	}
}

type parkCancelClock struct {
	now    time.Time
	cancel func()
	calls  int
}

func (c *parkCancelClock) Now() time.Time {
	c.calls++
	if c.calls == 2 {
		c.cancel()
	}
	return c.now
}

func TestParkCanceledAfterBeginRollsBackBeforePublication(t *testing.T) {
	env, live := envWithLiveThread(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerDone := make(chan *parkFuture, 1)
	env.Couch.PairLifecycle.Clock = &parkCancelClock{now: env.Now, cancel: func() {
		w := env.Couch.PairLifecycle.worker
		w.mu.Lock()
		future := w.active[live.Address].future
		w.mu.Unlock()
		workerDone <- future
		cancel()
	}}
	_, err := env.Couch.PairLifecycle.Park(ctx, live.Address)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("lost caller cancellation: %v", err)
	}
	future := <-workerDone
	wait, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	_, _ = future.Await(wait)
	after, getErr := env.Couch.Threads.GetThread(live.Address)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if after.Park != nil || len(after.Incarnations) != 1 || len(env.Lifecycle.trace) != 0 {
		t.Fatalf("canceled unpublished begin not safely rolled back: %+v trace=%v", after, env.Lifecycle.trace)
	}
}

func TestParkBeginDeadlineRollsBackBeforePublication(t *testing.T) {
	env, live := envWithLiveThread(t)
	env.Couch.PairLifecycle.Clock = &sequenceClock{times: []time.Time{env.Now, env.Now.Add(time.Second)}}
	_, err := env.Couch.PairLifecycle.Park(context.Background(), live.Address)
	if err == nil {
		t.Fatal("deadline not reported")
	}
	after, _ := env.Couch.Threads.GetThread(live.Address)
	if after.Park != nil || len(after.Incarnations) != 1 || len(env.Lifecycle.trace) != 0 {
		t.Fatalf("deadline retained unpublished transaction: %+v", after)
	}
}
