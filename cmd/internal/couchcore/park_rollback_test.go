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
	for _, tc := range []struct {
		name     string
		duration time.Duration
		overrun  bool
	}{
		{"2s commit proceeds", 2 * time.Second, false},
		{"3s commit proceeds", 3 * time.Second, false},
		{"just under threshold proceeds", parkCommitDeadline - time.Nanosecond, false},
		{"threshold rolls back", parkCommitDeadline, true},
		{"over threshold rolls back", parkCommitDeadline + 5*time.Second, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, live := envWithLiveThread(t)
			env.Couch.PairLifecycle.Clock = &sequenceClock{times: []time.Time{env.Now, env.Now.Add(tc.duration)}}
			result, err := env.Couch.PairLifecycle.Park(context.Background(), live.Address)
			var overrun *ParkCommitOverrunError
			if !tc.overrun {
				if errors.As(err, &overrun) {
					t.Fatalf("%s commit refused: %v", tc.duration, err)
				}
				if result.RequestedCommitDuration != tc.duration || len(env.Lifecycle.trace) == 0 {
					t.Fatalf("commit did not proceed to publication: duration=%s trace=%v err=%v", result.RequestedCommitDuration, env.Lifecycle.trace, err)
				}
				return
			}
			if !errors.As(err, &overrun) || overrun.Observed != tc.duration || overrun.Threshold != parkCommitDeadline {
				t.Fatalf("overrun lacks typed duration evidence: %#v %v", overrun, err)
			}
			after, _ := env.Couch.Threads.GetThread(live.Address)
			if after.Park != nil || len(after.Incarnations) != 1 || len(env.Lifecycle.trace) != 0 {
				t.Fatalf("deadline retained unpublished transaction: %+v", after)
			}
		})
	}
}

// A rollback that cannot be written leaves the transaction open: the overrun
// keeps its evidence, and the retained Park routes the next drive through the
// existing transaction rather than a fresh nonce.
func TestParkBeginDeadlineFailedRollbackRetainsTransaction(t *testing.T) {
	env, live := envWithLiveThread(t)
	env.Couch.PairLifecycle.Clock = &sequenceClock{times: []time.Time{env.Now, env.Now.Add(parkCommitDeadline)}}
	stop := errors.New("rollback write refused")
	env.Couch.Threads.hooks.BeforeUpdate = func(old, next ThreadRecord) error {
		if old.Park != nil && next.Park == nil {
			return stop
		}
		return nil
	}
	_, err := env.Couch.PairLifecycle.Park(context.Background(), live.Address)
	var overrun *ParkCommitOverrunError
	if !errors.As(err, &overrun) || !errors.Is(err, stop) {
		t.Fatalf("failed rollback lost evidence: %v", err)
	}
	after, _ := env.Couch.Threads.GetThread(live.Address)
	if after.Park == nil || len(env.Lifecycle.trace) != 0 {
		t.Fatalf("failed rollback dropped the uncertain transaction or published: %+v trace=%v", after.Park, env.Lifecycle.trace)
	}
}

// #827 raised the commit threshold; relaunch's diagnostic read must not ride along.
func TestRelaunchObservationTimeoutIndependentOfCommitThreshold(t *testing.T) {
	if relaunchObservationTimeout != time.Second || parkCommitDeadline == relaunchObservationTimeout {
		t.Fatalf("relaunch observation %s coupled to commit threshold %s", relaunchObservationTimeout, parkCommitDeadline)
	}
}
