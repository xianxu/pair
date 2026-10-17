package couchcore

import (
	"context"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

// A caller that asks for a longer completion wait (relaunch, pair#618) rides
// out a quit slower than the controller's default; the default still times out.
func TestParkCompletionWaitOverridesTheDefault(t *testing.T) {
	for _, tc := range []struct {
		name    string
		wait    time.Duration
		succeed bool
	}{{name: "default", succeed: false}, {name: "relaunch-wait", wait: 2 * time.Second, succeed: true}} {
		t.Run(tc.name, func(t *testing.T) {
			store, _, thread := createControllerThread(t)
			now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
			lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(now)}
			artifacts := NewFakeThreadArtifactCollisionChecker()
			artifacts.SetPairSession(thread.Address, "pair-slow", true)
			proc := NewFakeProcOps()
			proc.Set(42, "pair-helper")
			// The quit's clock starts at the first completion poll, not before
			// Park's setup, which can outlast it on a loaded machine.
			var firstPoll time.Time
			controller := PairLifecycleController{
				Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
				Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "park-slow-quit", nil },
				Proc: proc, CompletionTimeout: 100 * time.Millisecond, PollInterval: 10 * time.Millisecond,
				Wait: func(ctx context.Context, d time.Duration) error {
					if err := ctx.Err(); err != nil {
						return err // an expired deadline wins over a late-scheduled timer
					}
					if firstPoll.IsZero() {
						firstPoll = time.Now()
					}
					// The quit completes 300 ms into the wait: past the default, inside the override.
					if time.Since(firstPoll) > 300*time.Millisecond && lifecycle.completion == nil {
						completion := successCompletion(lifecycle.lastRequest, now)
						lifecycle.completion = &completion
						proc.Kill(42)
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(d):
						return nil
					}
				},
			}
			ctx := context.Background()
			if tc.wait > 0 {
				ctx = WithParkCompletionWait(ctx, tc.wait)
			}
			_, err := controller.Park(ctx, thread.Address)
			if (err == nil) != tc.succeed {
				t.Fatalf("park err = %v, want success %v", err, tc.succeed)
			}
		})
	}
}
