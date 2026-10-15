package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

func retentionResumeThread(t *testing.T) (*testEnv, ThreadAddress) {
	t.Helper()
	env, address := warmDetachedThread(t)
	// Retention semantics do not depend on startup latency. The caller's test
	// budget and cancellation still bound the wait; failure fixtures retain
	// warmDetachedThread's independent 200ms registration deadline.
	env.Couch.registrationDeadline = func(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
		return context.WithCancel(parent)
	}
	return env, address
}

func retentionResumeContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), testwait.Budget(t))
	t.Cleanup(cancel)
	return ctx
}

// Spending the registration budget before the first observer is scheduled
// reproduces the broad-run failure without a slow machine or a sleep.
func TestRetentionResumeBudgetCanExpireBeforeOwnership(t *testing.T) {
	env, address := warmDetachedThread(t)
	budget := newSpendableBudget()
	env.Couch.registrationDeadline = func(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
		if d != 200*time.Millisecond {
			t.Fatalf("inherited failure-fixture budget = %s", d)
		}
		ctx, cancel := budget.bind(parent)
		budget.spend()
		return ctx, cancel
	}
	_, _, err := env.Couch.ResumeContextWith(retentionResumeContext(t), address, ResumeOptions{})
	var wait *registrationWaitError
	if !errors.As(err, &wait) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("resume = %v, want a registration deadline failure", err)
	}
	if wait.phase != "ownership poll" || wait.polls != 0 || wait.last != "none" {
		t.Fatalf("registration failure = %+v, want no completed ownership observation", wait)
	}
}

func TestRetentionResumeRegistrationUsesParentBudget(t *testing.T) {
	env, address := retentionResumeThread(t)
	parent := retentionResumeContext(t)
	wantDeadline, ok := parent.Deadline()
	if !ok {
		t.Fatal("resume test has no enclosing budget")
	}
	factory := env.Couch.registrationDeadline
	if factory == nil {
		factory = context.WithTimeout
	}
	var registration context.Context
	env.Couch.registrationDeadline = func(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
		child, cancel := factory(ctx, d)
		registration = child
		return child, cancel
	}
	observations := 0
	env.Artifacts.BeforePairSession = func(ThreadAddress) error {
		if registration != nil {
			observations++
			got, ok := registration.Deadline()
			if !ok || !got.Equal(wantDeadline) {
				t.Errorf("registration deadline = %s (present %t), want enclosing test deadline %s", got, ok, wantDeadline)
			}
		}
		return nil
	}
	_, handle, err := env.Couch.ResumeContextWith(parent, address, ResumeOptions{})
	if err != nil || handle == nil {
		t.Fatalf("resume = %v", err)
	}
	if observations == 0 {
		t.Fatal("registration never reached the ownership observer")
	}
}

func TestRetentionResumeRegistrationHonorsParentCancellation(t *testing.T) {
	env, address := retentionResumeThread(t)
	parent, cancel := context.WithCancel(retentionResumeContext(t))
	defer cancel()
	factory := env.Couch.registrationDeadline
	if factory == nil {
		factory = context.WithTimeout
	}
	var registration context.Context
	env.Couch.registrationDeadline = func(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
		child, stop := factory(ctx, d)
		registration = child
		return child, stop
	}
	observed := false
	env.Artifacts.BeforePairSession = func(ThreadAddress) error {
		if registration != nil && !observed {
			observed = true
			cancel() // cancel at registration, never during fixture setup
			return context.Canceled
		}
		return nil
	}
	_, handle, err := env.Couch.ResumeContextWith(parent, address, ResumeOptions{})
	if !observed || !errors.Is(err, context.Canceled) || !errors.Is(registration.Err(), context.Canceled) {
		t.Fatalf("registration observed=%t, resume=%v, context=%v", observed, err, registration)
	}
	if handle != nil && handle.Alive() {
		t.Fatal("cancelled resume left its helper running")
	}
}
