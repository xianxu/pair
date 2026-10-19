package couchtty

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestConsoleHostedNavigationLostProofTransfersSameRequestOrRefusesOverload(t *testing.T) {
	for _, full := range []bool{false, true} {
		t.Run(map[bool]string{false: "transfer-once", true: "lifecycle-full"}[full], func(t *testing.T) {
			lifecycleEntered, lifecycleRelease := make(chan struct{}), make(chan struct{})
			navigationEntered, navigationRelease := make(chan struct{}), make(chan struct{})
			var lifecycleOnce, navigationOnce sync.Once
			var navigationCalls, lifecycleCalls, resumeCalls, attachCalls, badLane atomic.Int32
			var start couchcore.StartResult
			var recordedOrigin MenuOperationOrigin
			var originMu sync.Mutex
			f := newFixtureBeforeRun(t, 24, 80, func(c *Console) {
				c.queueWorkers = 2 // One held lifecycle worker plus the existing navigation worker.
				address := c.panes["c1"].thread
				start, _ = attachStartResult(t, "transferred-actor", address)
				t.Cleanup(func() { _ = start.Handle.(couchcore.TerminalHandle).Terminal().Close() })
				setTestOps(c, func(name string, _ map[string]string) (any, error) {
					if name == "resume" {
						return start, nil
					}
					return nil, fmt.Errorf("unexpected owner operation %q", name)
				})
				dispatch := c.Ops()
				c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
					marker, _ := call.Context.Value(menuOperationContextKey{}).(menuOperationContext)
					switch call.Name {
					case "switch":
						if marker.hosted != "" {
							if navigationCalls.Add(1) != 1 {
								badLane.Add(1)
								return nil, fmt.Errorf("navigation retried same request")
							}
							originMu.Lock()
							recordedOrigin = marker.origin
							originMu.Unlock()
							close(navigationEntered)
							<-navigationRelease
						} else {
							lifecycleCalls.Add(1)
							originMu.Lock()
							same := recordedOrigin == marker.origin
							originMu.Unlock()
							if !same {
								badLane.Add(1)
							}
						}
					case "resume":
						resumeCalls.Add(1)
						if marker.hosted != "" {
							badLane.Add(1)
						}
					case "attach":
						attachCalls.Add(1)
						if marker.hosted != "" {
							badLane.Add(1)
						}
					}
					return dispatch(call)
				})
			})
			t.Cleanup(func() {
				lifecycleOnce.Do(func() { close(lifecycleRelease) })
				navigationOnce.Do(func() { close(navigationRelease) })
			})
			requireOperationQueueAdmission(t, f.con.operationQueue, operationRequest{key: "held-lifecycle", run: func() (any, error) { close(lifecycleEntered); <-lifecycleRelease; return nil, nil }})
			awaitOperationQueueSignal(t, lifecycleEntered)
			if full {
				for i := 0; i < 15; i++ {
					requireOperationQueueAdmission(t, f.con.operationQueue, operationRequest{key: fmt.Sprint("queued-lifecycle-", i), run: func() (any, error) { return nil, nil }})
				}
			}
			var attempt uint64
			err := f.con.runTerminalCommand(context.Background(), func() error {
				f.con.mu.Lock()
				address := f.con.panes["c1"].thread
				state := NewMenuState([]couchcore.ActionableThreadSummary{{Address: address, State: couchcore.ThreadLive}}, address)
				state, effects := dispatchThreadOperation(state, "switch", address)
				attempt = effects[0].Attempt
				f.con.menu, f.con.menuReady = state, true
				f.con.mu.Unlock()
				f.con.dispatchMenuEffects(effects)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			awaitOperationQueueSignal(t, navigationEntered)
			// Retain the hosted pane's output but revoke the exact input endpoint proof.
			// No child/process is killed and no native command is involved.
			f.child.Endpoint().EndInput()
			navigationOnce.Do(func() { close(navigationRelease) })
			key := fmt.Sprintf("menu\x00%d\x00switch", attempt)
			if !full {
				testwait.Until(t, "same request transferred behind held lifecycle worker", func() bool {
					q := f.con.operationQueue
					q.mu.Lock()
					defer q.mu.Unlock()
					return len(q.requests) == 1 && q.pending[key]
				})
				if lifecycleCalls.Load() != 0 || resumeCalls.Load() != 0 || attachCalls.Load() != 0 {
					t.Fatal("navigation worker executed lifecycle or adoption")
				}
				accepted, err := f.con.operationQueue.Enqueue(operationRequest{key: key, run: func() (any, error) { badLane.Add(1); return nil, nil }})
				if accepted || err != nil {
					t.Fatalf("transfer released exact pending owner: %v,%v", accepted, err)
				}
				lifecycleOnce.Do(func() { close(lifecycleRelease) })
			}
			testwait.Until(t, "final transferred switch result", func() bool {
				state := f.con.menuSnapshot()
				for _, job := range state.Operations {
					if job.Origin.Attempt == attempt {
						return job.Phase != MenuOperationRunning
					}
				}
				return false
			})
			state := f.con.menuSnapshot()
			var result MenuOperation
			for _, job := range state.Operations {
				if job.Origin.Attempt == attempt {
					result = job
				}
			}
			if full {
				if result.Phase != MenuOperationFailed || !strings.Contains(result.Detail, ErrOperationQueueOverloaded.Error()) || lifecycleCalls.Load() != 0 || resumeCalls.Load() != 0 || attachCalls.Load() != 0 {
					t.Fatalf("overload outcome=%+v life=%d resume=%d attach=%d", result, lifecycleCalls.Load(), resumeCalls.Load(), attachCalls.Load())
				}
			} else {
				if result.Phase != MenuOperationSucceeded || lifecycleCalls.Load() != 1 || resumeCalls.Load() != 1 || attachCalls.Load() != 1 {
					t.Fatalf("transfer outcome=%+v life=%d resume=%d attach=%d", result, lifecycleCalls.Load(), resumeCalls.Load(), attachCalls.Load())
				}
				f.con.mu.Lock()
				pane, adopted := f.con.panes[start.Handle.ID()]
				f.con.mu.Unlock()
				if !adopted || pane.child != start.Handle.(couchcore.TerminalHandle).Terminal() {
					t.Fatal("lifecycle lane did not adopt exact resumed terminal")
				}
			}
			if navigationCalls.Load() != 1 || badLane.Load() != 0 {
				t.Fatalf("navigation=%d wrong lane/origin=%d", navigationCalls.Load(), badLane.Load())
			}
			q := f.con.operationQueue
			q.mu.Lock()
			pending := q.pending[key]
			q.mu.Unlock()
			if pending {
				t.Fatal("final completion retained request key")
			}
		})
	}
}
