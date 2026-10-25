package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// A real preview takes this same kernel lock outside the retention coordinator.
// The observer releases it only after a foreground acquisition has contended.
func TestEmptyStartupWaitsForPreviewAtBothLockSites(t *testing.T) {
	for _, kind := range []string{"primary-global", "slot-global", "slot-local"} {
		for _, n := range []int{2, 4} {
			t.Run(kind+string(rune('0'+n)), func(t *testing.T) {
				env, local := slotRecoveryOperationFixture(t)
				global := env.Couch.Threads
				target, held := global, global
				if kind != "primary-global" {
					target = local
				}
				if kind == "slot-local" {
					held = local
				}
				if err := held.withStoreLock(func() error { return nil }); err != nil {
					t.Fatal(err)
				}
				publications, contentions := 0, 0
				for range n {
					unlock := holdStoreLock(t, held)
					var once sync.Once
					release := func() { once.Do(unlock) }
					t.Cleanup(release)
					observeStoreLockContention(t, func() { contentions++; release() })
					err := global.withEmptyStartupTarget(t.Context(), target, "empty-target", func() error { publications++; return nil })
					release()
					if err != nil {
						t.Fatalf("foreground empty reservation refused preview contention: %v", err)
					}
				}
				if publications != n || contentions != n {
					t.Fatalf("effects grew or bypassed admission: publications=%d contentions=%d want%d", publications, contentions, n)
				}
			})
		}
	}
}

func TestEmptyStartupLockCancellationNeverPublishes(t *testing.T) {
	for _, localHold := range []bool{false, true} {
		t.Run(map[bool]string{false: "global", true: "local"}[localHold], func(t *testing.T) {
			env, local := slotRecoveryOperationFixture(t)
			global, held := env.Couch.Threads, env.Couch.Threads
			if localHold {
				held = local
			}
			if err := held.withStoreLock(func() error { return nil }); err != nil {
				t.Fatal(err)
			}
			release := holdStoreLock(t, held)
			defer release()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			observeStoreLockContention(t, cancel)
			publications := 0
			err := global.withEmptyStartupTarget(ctx, local, "empty-target", func() error { publications++; return nil })
			if !errors.Is(err, context.Canceled) || publications != 0 {
				t.Fatalf("cancelled reservation: publications=%d err=%v", publications, err)
			}
		})
	}
}

func TestEmptyStartupLockExhaustionIsTypedAndNeverPublishes(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	global := env.Couch.Threads
	release := holdStoreLock(t, global)
	defer release()
	saved := storeReadLockWait
	storeReadLockWait = time.Nanosecond
	defer func() { storeReadLockWait = saved }()
	publications := 0
	err := global.withEmptyStartupTarget(t.Context(), local, "empty-target", func() error { publications++; return nil })
	if !errors.Is(err, ErrThreadStoreBusy) || publications != 0 {
		t.Fatalf("exhausted reservation: publications=%d err=%v", publications, err)
	}
}

func TestEmptyStartupRechecksRetainedEvidenceAfterWaiting(t *testing.T) {
	for _, family := range []string{"journal", "current", "archive"} {
		for _, localHold := range []bool{false, true} {
			t.Run(family+map[bool]string{false: "-global", true: "-local"}[localHold], func(t *testing.T) {
				env, local := slotRecoveryOperationFixture(t)
				global, held := env.Couch.Threads, env.Couch.Threads
				if localHold {
					held = local
				}
				if err := held.withStoreLock(func() error { return nil }); err != nil {
					t.Fatal(err)
				}
				unlock := holdStoreLock(t, held)
				var once sync.Once
				release := func() { once.Do(unlock) }
				defer release()
				path := held.journalPath()
				address := ThreadAddress{RepoScope: "empty-target", Tag: "retained"}
				if family == "current" {
					path = held.recordPath(address)
				}
				if family == "archive" {
					path = held.archivePath(address)
				}
				const original = "retained evidence must never be changed by startup"
				observed := false
				observeStoreLockContention(t, func() {
					observed = true
					if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(original), 0600); err != nil {
						t.Fatal(err)
					}
					release()
				})
				publications := 0
				err := global.withEmptyStartupTarget(t.Context(), local, "empty-target", func() error { publications++; return nil })
				if !observed || err == nil || publications != 0 {
					t.Fatalf("evidence appeared while waiting: publications=%d err=%v", publications, err)
				}
				got, readErr := os.ReadFile(path)
				if readErr != nil || string(got) != original {
					t.Fatalf("retained evidence was touched: %q %v", got, readErr)
				}
			})
		}
	}
}
