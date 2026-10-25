package couchcore

import (
	"context"
	"fmt"
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
	}{{name: "default", succeed: false}, {name: "relaunch-wait", wait: RelaunchParkWait, succeed: true}} {
		t.Run(tc.name, func(t *testing.T) {
			store, _, thread := createControllerThread(t)
			now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
			lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(now)}
			artifacts := NewFakeThreadArtifactCollisionChecker()
			artifacts.SetPairSession(thread.Address, "pair-slow", true)
			proc := NewFakeProcOps()
			proc.Set(42, "pair-helper")
			// The fake completion is beyond the default budget but inside
			// the caller override. The wait observes that selected budget,
			// then either spends it or delivers completion deterministically.
			controller := PairLifecycleController{
				Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
				Clock: FixedClock{T: now}, Nonce: func() (string, error) { return "park-slow-quit", nil },
				// Match the ordinary controller budget configured by New. The
				// injected deadline below never starts a wall-clock timer.
				Proc: proc, CompletionTimeout: 15 * time.Second,
			}
			budget := newSpendableBudget()
			var selected time.Duration
			controller.completionDeadline = func(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
				selected = d
				return budget.bind(parent)
			}
			controller.Wait = func(ctx context.Context, _ time.Duration) error {
				if lifecycle.lastRequest.Attempt == 0 {
					t.Error("wait preceded quit request")
					return fmt.Errorf("wait preceded quit request")
				}
				if tc.wait == 0 {
					if selected != controller.CompletionTimeout {
						t.Errorf("default budget = %v", selected)
						return fmt.Errorf("wrong default budget: %v", selected)
					}
					budget.spend()
					return ctx.Err()
				}
				if selected != tc.wait || selected <= controller.CompletionTimeout {
					return fmt.Errorf("override budget = %v, default %v", selected, controller.CompletionTimeout)
				}
				completion := successCompletion(lifecycle.lastRequest, now)
				lifecycle.completion = &completion
				proc.Kill(42)
				return nil
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
