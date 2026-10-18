package couchcore

import (
	"context"
	"errors"
	"strings"
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

// Public warm resume still applies the fixture's registration budget to the
// listed-terminal attachment check. Spend it before that check is scheduled.
func TestRetentionResumeBudgetCanExpireBeforeAttachment(t *testing.T) {
	env, address := warmDetachedThread(t)
	armed := 0
	env.Couch.registrationDeadline = func(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
		if d != 200*time.Millisecond {
			t.Fatalf("inherited failure-fixture budget = %s", d)
		}
		armed++
		budget := newSpendableBudget()
		ctx, cancel := budget.bind(parent)
		budget.spend()
		return ctx, cancel
	}
	_, _, err := env.Couch.ResumeContextWith(retentionResumeContext(t), address, ResumeOptions{})
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "await Pair attachment") {
		t.Fatalf("resume = %v, want an attachment deadline failure", err)
	}
	if armed != 1 {
		t.Fatalf("attachment attempts = %d, want an unanswered observation to end admission", armed)
	}
}

// Cold/fresh resume still uses ownership registration. Its deadline can also
// expire before the first observer runs, with no completed poll to report.
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
	ctx, cancel := env.Couch.registrationDeadline(retentionResumeContext(t), env.Couch.resumeRegistrationTimeout)
	defer cancel()
	err := env.Couch.awaitResumeRegistration(ctx, address, nil)
	var wait *registrationWaitError
	if !errors.As(err, &wait) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("resume = %v, want a registration deadline failure", err)
	}
	if wait.phase != "ownership poll" || wait.polls != 0 || wait.last != "none" {
		t.Fatalf("registration failure = %+v, want no completed ownership observation", wait)
	}
}

func TestRetentionResumeAttachmentUsesParentBudget(t *testing.T) {
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
	var attachment context.Context
	env.Couch.registrationDeadline = func(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
		child, cancel := factory(ctx, d)
		attachment = child
		return child, cancel
	}
	observations := 0
	env.Artifacts.ListedSessionHook = func(ThreadAddress) error {
		if attachment != nil {
			observations++
			got, ok := attachment.Deadline()
			if !ok || !got.Equal(wantDeadline) {
				t.Errorf("attachment deadline = %s (present %t), want enclosing test deadline %s", got, ok, wantDeadline)
			}
		}
		return nil
	}
	_, handle, err := env.Couch.ResumeContextWith(parent, address, ResumeOptions{})
	if err != nil || handle == nil {
		t.Fatalf("resume = %v", err)
	}
	if observations == 0 {
		t.Fatal("attachment never reached the listed-terminal observer")
	}
}

func TestRetentionResumeAttachmentHonorsParentCancellation(t *testing.T) {
	env, address := retentionResumeThread(t)
	parent, cancel := context.WithCancel(retentionResumeContext(t))
	defer cancel()
	factory := env.Couch.registrationDeadline
	if factory == nil {
		factory = context.WithTimeout
	}
	var attachment context.Context
	env.Couch.registrationDeadline = func(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
		child, stop := factory(ctx, d)
		attachment = child
		return child, stop
	}
	observed := false
	env.Artifacts.ListedSessionHook = func(ThreadAddress) error {
		if attachment != nil && !observed {
			observed = true
			cancel() // cancel at attachment, never during fixture setup
			// Observe propagation before resume cleanup cancels the child itself.
			if !errors.Is(attachment.Err(), context.Canceled) {
				t.Error("attachment context did not inherit parent cancellation")
			}
			return context.Canceled
		}
		return nil
	}
	_, handle, err := env.Couch.ResumeContextWith(parent, address, ResumeOptions{})
	if !observed || !errors.Is(err, context.Canceled) || !errors.Is(attachment.Err(), context.Canceled) {
		t.Fatalf("attachment observed=%t, resume=%v, context=%v", observed, err, attachment)
	}
	if handle != nil && handle.Alive() {
		t.Fatal("cancelled resume left its helper running")
	}
}
