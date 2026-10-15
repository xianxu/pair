package couchtty

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type consolePassTestKey struct{}

func TestConsoleReattachSharesOneLazyPassAcross21Slots(t *testing.T) {
	raiseBound(t, 3)
	var factories, calls atomic.Int32
	var passContext context.Context
	f := newFixtureBeforeRun(t, 24, 100, func(c *Console) {
		c.SetSessionOwnerPass(func(ctx context.Context) (context.Context, context.CancelFunc) {
			factories.Add(1)
			// A factory must be called outside the console state lock.
			c.mu.Lock()
			c.mu.Unlock()
			ctx, cancel := context.WithCancel(ctx)
			passContext = context.WithValue(ctx, consolePassTestKey{}, "shared")
			return passContext, cancel
		})
		root := c.panes["c1"].thread
		c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
			if call.Context.Value(consolePassTestKey{}) != "shared" {
				t.Error("background slot did not inherit shared pass")
			}
			if call.Context.Err() != nil {
				t.Error("one finished slot canceled a later sibling")
			}
			if call.Args["tag"] == string(root.Tag) {
				t.Error("background pass included root")
			}
			calls.Add(1)
			return nil, errors.New("fixture completion")
		})
		c.ArmReattachPass(root)
		c.ArmReattachPass(root)
		if calls.Load() != 0 {
			t.Fatal("arming performed worker IO before Run")
		}
		c.SetActionableProvider(detachedFleet(root, 21))
	})
	waitFor(t, "21 pass attempts complete and released", func() bool { return f.con.menuSnapshot().Reattach.Phase == ReattachDone && passContext.Err() != nil })
	if calls.Load() != 21 || factories.Load() != 1 {
		t.Fatalf("attempts=%d factories=%d, want 21/1", calls.Load(), factories.Load())
	}
	if passContext.Err() == nil {
		t.Fatal("completed reattach retained its pass")
	}
	if f.con.lifetime.Err() != nil {
		t.Fatal("pass completion canceled console lifetime")
	}
}

func awaitConsolePass[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-testwait.After(t):
		t.Fatal("console pass did not reach worker barrier")
		var zero T
		return zero
	}
}

func TestConsoleReattachReleasesEmptyInventoryPass(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(map[bool]string{false: "inventory producer", true: "semantic producer"}[direct], func(t *testing.T) {
			var passContext context.Context
			var canceled atomic.Int32
			f := newFixtureBeforeRun(t, 24, 100, func(c *Console) {
				c.SetSessionOwnerPass(func(ctx context.Context) (context.Context, context.CancelFunc) {
					ctx, cancel := context.WithCancel(ctx)
					passContext = ctx
					return ctx, func() { c.mu.Lock(); c.mu.Unlock(); canceled.Add(1); cancel() }
				})
				root := c.panes["c1"].thread
				c.ArmReattachPass(root)
				if !direct {
					c.SetActionableProvider(detachedFleet(root, 0))
				}
			})
			if direct {
				f.con.reduceMenu(MenuEvent{Kind: MenuEventInventory, Inventory: nil})
			}
			waitFor(t, "empty pass released", func() bool { return passContext.Err() != nil && canceled.Load() == 1 })
			f.con.mu.Lock()
			retained := f.con.reattachContext != nil || f.con.reattachCancel != nil
			f.con.mu.Unlock()
			if retained {
				t.Fatal("completed scope retained on console")
			}
			f.con.Stop()
			if canceled.Load() != 1 {
				t.Fatal("Stop canceled the completed pass twice")
			}
		})
	}
}

func TestConsoleReattachFinishedSlotCannotCancelSibling(t *testing.T) {
	raiseBound(t, 2)
	first, sibling := make(chan context.Context, 1), make(chan context.Context, 1)
	releaseFirst, releaseSibling := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	var passContext context.Context
	f := newFixtureBeforeRun(t, 24, 100, func(c *Console) {
		c.SetSessionOwnerPass(func(ctx context.Context) (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(ctx)
			passContext = ctx
			return ctx, cancel
		})
		c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
			var release <-chan struct{}
			if calls.Add(1) == 1 {
				first <- call.Context
				release = releaseFirst
			} else {
				sibling <- call.Context
				release = releaseSibling
			}
			select {
			case <-release:
				return nil, errors.New("fixture slot finished")
			case <-call.Context.Done():
				return nil, call.Context.Err()
			}
		})
		root := c.panes["c1"].thread
		c.ArmReattachPass(root)
		c.SetActionableProvider(detachedFleet(root, 2))
	})
	firstCtx, siblingCtx := awaitConsolePass(t, first), awaitConsolePass(t, sibling)
	close(releaseFirst)
	waitFor(t, "first operation canceled on completion", func() bool { return firstCtx.Err() != nil })
	if siblingCtx.Err() != nil || passContext.Err() != nil {
		t.Fatal("slot completion poisoned pass/sibling")
	}
	close(releaseSibling)
	waitFor(t, "sibling completed", func() bool { return f.con.menuSnapshot().Reattach.Phase == ReattachDone && passContext.Err() != nil })
}

func TestConsoleStopReleasesPassAndCancelsWorkers(t *testing.T) {
	raiseBound(t, 3)
	started := make(chan context.Context, 3)
	var passContext context.Context
	var canceled atomic.Int32
	f := newFixtureBeforeRun(t, 24, 100, func(c *Console) {
		c.SetSessionOwnerPass(func(ctx context.Context) (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(ctx)
			passContext = ctx
			return ctx, func() { c.mu.Lock(); c.mu.Unlock(); canceled.Add(1); cancel() }
		})
		c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
			started <- call.Context
			<-call.Context.Done()
			return nil, call.Context.Err()
		})
		root := c.panes["c1"].thread
		c.ArmReattachPass(root)
		c.SetActionableProvider(detachedFleet(root, 21))
	})
	contexts := []context.Context{awaitConsolePass(t, started), awaitConsolePass(t, started), awaitConsolePass(t, started)}
	f.con.Stop()
	for _, ctx := range contexts {
		if ctx.Err() == nil {
			t.Fatal("Stop retained worker context")
		}
	}
	if passContext.Err() == nil || canceled.Load() != 1 {
		t.Fatalf("Stop retained pass: err=%v cancels=%d", passContext.Err(), canceled.Load())
	}
	f.con.mu.Lock()
	retained := f.con.reattachContext != nil || f.con.reattachCancel != nil
	f.con.mu.Unlock()
	if retained {
		t.Fatal("Stop retained pass fields")
	}
}

func TestConsoleOperatorOperationDoesNotInheritReattachPass(t *testing.T) {
	observed := make(chan context.Context, 1)
	f := newFixtureBeforeRun(t, 24, 100, func(c *Console) {
		c.SetSessionOwnerPass(func(ctx context.Context) (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(ctx)
			return context.WithValue(ctx, consolePassTestKey{}, "shared"), cancel
		})
		c.ArmReattachPass(c.panes["c1"].thread)
		c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
			observed <- call.Context
			return nil, errors.New("fixture operator complete")
		})
	})
	f.con.mu.Lock()
	root := f.con.panes["c1"].thread
	state, effects := dispatchMenuOperation(f.con.menu, MenuEffect{Operation: "name"}, root)
	f.con.menu = state
	f.con.mu.Unlock()
	if len(effects) != 1 {
		t.Fatal("operator operation was not admitted")
	}
	f.con.dispatchMenuEffects(effects)
	ctx := awaitConsolePass(t, observed)
	if ctx.Value(consolePassTestKey{}) != nil || f.con.lifetime.Value(consolePassTestKey{}) != nil {
		t.Fatal("operator/root lifetime inherited background pass")
	}
}

func TestConsoleCanceledPassRefusesQueuedWorkerIO(t *testing.T) {
	var calls atomic.Int32
	f := newFixtureBeforeRun(t, 24, 100, func(c *Console) {
		var cancelPass context.CancelFunc
		c.SetSessionOwnerPass(func(ctx context.Context) (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(ctx)
			cancelPass = cancel
			return ctx, cancel
		})
		c.SetOperationDispatcher(func(couchcore.OperationCall) (any, error) {
			calls.Add(1)
			return nil, errors.New("unexpected canceled worker")
		})
		root := c.panes["c1"].thread
		c.ArmReattachPass(root)
		c.runBackgroundOperation(MenuEffect{Operation: "resume", Attempt: 1, Background: true, Args: map[string]string{"repo-scope": root.RepoScope, "tag": "couch-canceled"}})
		cancelPass()
	})
	waitFor(t, "queued attempt drained", func() bool {
		f.con.operationQueue.mu.Lock()
		defer f.con.operationQueue.mu.Unlock()
		return len(f.con.operationQueue.pending) == 0
	})
	if calls.Load() != 0 {
		t.Fatalf("canceled queued pass invoked worker IO %d times", calls.Load())
	}
}
