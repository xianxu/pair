package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestThreadStoreLockSerializesIndependentHandles(t *testing.T) {
	t.Parallel()

	_, ns := newTestThreadStore(t)
	root := NewThreadStore(ns).root
	first, err := acquireThreadStoreLockMode(root, false)
	if err != nil {
		t.Fatalf("acquire first: %v", err)
	}
	acquired := make(chan *threadStoreLock, 1)
	errs := make(chan error, 1)
	go func() {
		second, err := acquireThreadStoreLockMode(root, false)
		if err != nil {
			errs <- err
			return
		}
		acquired <- second
	}()
	select {
	case second := <-acquired:
		_ = second.Close()
		t.Fatal("second independent handle entered the critical section")
	case err := <-errs:
		t.Fatalf("second lock errored instead of waiting: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if err := first.Close(); err != nil {
		t.Fatalf("release first: %v", err)
	}
	select {
	case second := <-acquired:
		defer second.Close()
	case err := <-errs:
		t.Fatalf("second lock after release: %v", err)
	case <-testwait.After(t):
		t.Fatal("second lock did not acquire after release")
	}
}

// pair#854: a foreground store.lock wait never outlives the caller's deadline,
// and reports that deadline even when the context check cannot see the expiry
// (a deschedule between the check and the wait's own deadline test).
func TestStoreLockWaitEndsAtTheCallerDeadline(t *testing.T) {
	t.Parallel()
	_, ns := newTestThreadStore(t)
	s := NewThreadStore(ns)
	held, err := acquireThreadStoreLockMode(s.root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	blind := func() error { return nil }
	ran := func() error { t.Error("ran without the lock"); return nil }

	// The fixed wait alone ignores the deadline: busy, after the full wait.
	if err := s.withStoreLockChecked(ran, blind, 200*time.Millisecond); !errors.Is(err, ErrThreadStoreBusy) {
		t.Fatalf("fixed wait = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	err = s.withStoreLockBounded(ran, blind, storeLockBound(ctx, time.Second))
	if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrThreadStoreBusy) {
		t.Fatalf("bounded wait = %v", err)
	}
	if elapsed := time.Since(start); elapsed >= time.Second {
		t.Fatalf("bounded wait ran the full second: %s", elapsed)
	}
	// Spent budget: nothing to wait, and still the deadline, not busy.
	<-ctx.Done()
	if err := s.withStoreLockBounded(ran, blind, storeLockBound(ctx, time.Second)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("spent budget = %v", err)
	}
}

// pair#854: with more budget than the wait, contention outlasting the wait is
// still busy.
func TestStoreLockWaitWithinBudgetStaysBusy(t *testing.T) {
	t.Parallel()
	_, ns := newTestThreadStore(t)
	s := NewThreadStore(ns)
	held, err := acquireThreadStoreLockMode(s.root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	wait := storeLockBound(ctx, 100*time.Millisecond)
	if wait.deadline || wait.d != 100*time.Millisecond {
		t.Fatalf("bound = %+v", wait)
	}
	err = s.withStoreLockBounded(func() error { return nil }, ctx.Err, wait)
	if !errors.Is(err, ErrThreadStoreBusy) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("within budget = %v", err)
	}
}
