package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"
)

// deadlineOnlyContext has a near deadline whose Err stays nil and whose Done
// never closes: only a wait capped by storeLockBound can end at it. A call site
// that keeps the fixed wait polls it out and reports busy instead (pair#854).
type deadlineOnlyContext struct {
	context.Context
	deadline time.Time
}

func (c deadlineOnlyContext) Deadline() (time.Time, bool) { return c.deadline, true }
func (c deadlineOnlyContext) Done() <-chan struct{}       { return nil }
func (c deadlineOnlyContext) Err() error                  { return nil }

func deadlineOnly(budget time.Duration) context.Context {
	return deadlineOnlyContext{Context: context.Background(), deadline: time.Now().Add(budget)}
}

func assertDeadlineEnded(t *testing.T, err error, start time.Time) {
	t.Helper()
	if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrThreadStoreBusy) {
		t.Fatalf("want the caller's deadline, not busy: %v", err)
	}
	if waited := time.Since(start); waited >= storeReadLockWait {
		t.Fatalf("waited %s, the full fixed wait", waited)
	}
}

// Both park-recovery branches, coordinated as production builds them and not.
func TestUpdateLockWaitEndsAtTheCallerDeadline(t *testing.T) {
	for name, store := range map[string]*ThreadStore{"coordinated": coordinatedLockFixture(t), "plain": coordinatedLockFixture(t)} {
		t.Run(name, func(t *testing.T) {
			if name == "plain" {
				store.coordinator = nil
			}
			defer holdStoreLock(t, store)()
			start := time.Now()
			err := store.withUpdateLockContext(deadlineOnly(50*time.Millisecond), func() error { t.Error("ran without the lock"); return nil })
			assertDeadlineEnded(t, err, start)
		})
	}
}

func TestEmptyStartupLockWaitEndsAtACapOnlyDeadline(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	global := env.Couch.Threads
	defer holdStoreLock(t, global)()
	start := time.Now()
	publications := 0
	err := global.withEmptyStartupTarget(deadlineOnly(50*time.Millisecond), local, "empty-target", func() error { publications++; return nil })
	assertDeadlineEnded(t, err, start)
	if publications != 0 {
		t.Fatalf("published %d times without the lock", publications)
	}
}

// A capped wait can end the publication before ctx.Err is set; the refusal
// still names the contention that preceded it (#877).
func TestRepoPreferencePublicationJoinsBusyAtACapOnlyDeadline(t *testing.T) {
	store, global, before := contendedPreferenceFixture(t)
	defer holdStoreLock(t, global)()
	err := publishSecond(store, deadlineOnly(8*storeReadLockWait))
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, ErrThreadStoreBusy) {
		t.Fatalf("busy to a cap-only deadline: %v", err)
	}
	assertPreferencesUnchanged(t, global, before)
}
