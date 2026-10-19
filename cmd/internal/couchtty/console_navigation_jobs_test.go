package couchtty

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type navigationJobsFixture struct {
	f                                  *consoleFixture
	a, b, hub                          couchcore.ThreadAddress
	bChild, hubChild                   *ptychild.Child
	start                              couchcore.StartResult
	lifecycleEntered, lifecycleRelease chan struct{}
	adoptionEntered, adoptionRelease   chan struct{}
	releaseLifecycle, releaseAdoption  sync.Once
	lifecycleCalls, adoptionCalls      atomic.Int32
	attempt                            uint64
}

func newNavigationJobsFixture(t *testing.T) *navigationJobsFixture {
	t.Helper()
	r := &navigationJobsFixture{
		b: menuAddress("worker-b"), hub: menuAddress("hub"),
		bChild: ptychild.NewFakeChild(nil), hubChild: ptychild.NewFakeChild(nil),
		lifecycleEntered: make(chan struct{}), lifecycleRelease: make(chan struct{}),
		adoptionEntered: make(chan struct{}), adoptionRelease: make(chan struct{}),
	}
	r.f = newFixtureBeforeRun(t, 24, 100, func(c *Console) {
		r.a = c.panes["c1"].thread
		r.start, _ = attachStartResult(t, "replacement-a", r.a)
		t.Cleanup(func() { _ = r.start.Handle.(couchcore.TerminalHandle).Terminal().Close() })
		c.attachThreadActor("worker-b", "worker-b", r.b, "/worker-b", "worker-b", r.bChild)
		c.attachThreadActor("hub", "hub", r.hub, "/hub", "hub", r.hubChild)
		c.menu = NewMenuState([]couchcore.ActionableThreadSummary{
			{Address: r.a, Name: "worker-a", WorkingPath: "/worker-a", State: couchcore.ThreadLive},
			{Address: r.b, Name: "worker-b", WorkingPath: "/worker-b", State: couchcore.ThreadLive},
			{Address: r.hub, Name: "hub", WorkingPath: "/hub", State: couchcore.ThreadLive},
		}, r.a)
		c.menu.Hub = "/hub"
		c.menuReady = true
		c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
			switch call.Name {
			case "relaunch":
				if r.lifecycleCalls.Add(1) == 1 {
					close(r.lifecycleEntered)
				}
				<-r.lifecycleRelease
				r.f.child.Exit(0)
				return couchcore.RelaunchResult{Outcome: couchcore.Relaunched, Record: r.start.Record, Handle: r.start.Handle}, nil
			case "attach":
				if r.adoptionCalls.Add(1) == 1 {
					close(r.adoptionEntered)
				}
				<-r.adoptionRelease
				return c.ExecuteConsoleOperation(call)
			case "switch":
				return c.ExecuteConsoleOperation(call)
			default:
				return nil, fmt.Errorf("unexpected navigation fixture operation %s", call.Name)
			}
		})
	})
	t.Cleanup(func() {
		r.releaseLifecycle.Do(func() { close(r.lifecycleRelease) })
		r.releaseAdoption.Do(func() { close(r.adoptionRelease) })
	})
	return r
}

func (r *navigationJobsFixture) begin(t *testing.T) {
	t.Helper()
	if err := r.f.con.runTerminalCommand(context.Background(), func() error {
		c := r.f.con
		c.mu.Lock()
		state, effects := dispatchThreadOperation(c.menu, "relaunch", r.a)
		if len(effects) != 1 {
			c.mu.Unlock()
			return fmt.Errorf("relaunch effects = %+v", effects)
		}
		r.attempt = effects[0].Attempt
		c.menu, c.focus = state, FocusPanel()
		c.mu.Unlock()
		c.showMenu()
		c.dispatchMenuEffects(effects)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	r.await(t, "relaunch entry", r.lifecycleEntered)
}

func (r *navigationJobsFixture) await(t *testing.T, what string, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-testwait.After(t):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// Selection is serialized on the real Run loop and executes its normal Enter
// reducer/dispatch path. The fixture only supplies the selected row.
func (r *navigationJobsFixture) choose(t *testing.T, address couchcore.ThreadAddress) {
	t.Helper()
	if err := r.f.con.runTerminalCommand(context.Background(), func() error {
		c := r.f.con
		c.mu.Lock()
		row, ok := menuThread(c.menu, address)
		if !ok {
			c.mu.Unlock()
			return fmt.Errorf("selected row %v vanished", address)
		}
		c.menu.Frames = c.menu.Frames[:1]
		selectMenuRow(&c.menu.Frames[0], row)
		c.focus = FocusPanel()
		c.mu.Unlock()
		c.onMenuKey(PanelKey{Kind: KeyEnter})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func (r *navigationJobsFixture) input(t *testing.T, handle string, child *ptychild.Child, text string) {
	t.Helper()
	testwait.Until(t, "hosted selection "+handle, func() bool {
		r.f.con.mu.Lock()
		defer r.f.con.mu.Unlock()
		return r.f.con.focus == FocusActor(handle)
	})
	before := len(child.Writes())
	if _, err := r.f.stdin.Write([]byte(text)); err != nil {
		t.Fatal(err)
	}
	testwait.Until(t, "input delivery to "+handle, func() bool {
		writes := child.Writes()
		return len(writes) > before && string(writes[len(writes)-1]) == text
	})
}

func (r *navigationJobsFixture) completed(t *testing.T) {
	t.Helper()
	testwait.Until(t, "relaunch final reduction", func() bool {
		for _, job := range r.f.con.menuSnapshot().Operations {
			if job.Origin.Attempt == r.attempt {
				return job.Phase == MenuOperationSucceeded
			}
		}
		return false
	})
	// Result reduction precedes the final landing within finishOperation. This
	// command acknowledges the entire completion before focus is inspected.
	if err := r.f.con.runTerminalCommand(context.Background(), func() error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestConsoleNavigationAndInputProceedDuringRelaunchAndAdoption(t *testing.T) {
	r := newNavigationJobsFixture(t)
	r.begin(t)
	r.choose(t, r.b)
	r.input(t, "worker-b", r.bChild, "b")
	r.choose(t, r.hub)
	r.input(t, "hub", r.hubChild, "h")
	r.releaseLifecycle.Do(func() { close(r.lifecycleRelease) })
	r.await(t, "adoption entry", r.adoptionEntered)
	r.choose(t, r.b)
	r.input(t, "worker-b", r.bChild, "B")
	r.choose(t, r.hub)
	r.input(t, "hub", r.hubChild, "H")
	r.releaseAdoption.Do(func() { close(r.adoptionRelease) })
	r.completed(t)
	r.f.con.mu.Lock()
	focus := r.f.con.focus
	_, adopted := r.f.con.panes[r.start.Handle.ID()]
	r.f.con.mu.Unlock()
	if !adopted {
		t.Fatal("late replacement was discarded instead of adopted")
	}
	if focus != FocusActor("hub") {
		t.Fatalf("late completion stole focus: %v", focus)
	}
}

func TestConsoleBusyRowSelectionLandsAfterHeldAdoption(t *testing.T) {
	r := newNavigationJobsFixture(t)
	r.begin(t)
	r.choose(t, r.b)
	r.input(t, "worker-b", r.bChild, "b")
	r.choose(t, r.a)
	if state := r.f.con.menuSnapshot(); state.LandingAttempt != r.attempt {
		t.Fatalf("busy-row intent = %d, want %d", state.LandingAttempt, r.attempt)
	}
	r.releaseLifecycle.Do(func() { close(r.lifecycleRelease) })
	r.await(t, "adoption entry", r.adoptionEntered)
	r.choose(t, r.a)
	r.releaseAdoption.Do(func() { close(r.adoptionRelease) })
	r.completed(t)
	r.input(t, r.start.Handle.ID(), r.start.Handle.(couchcore.TerminalHandle).Terminal(), "a")
	if got := r.lifecycleCalls.Load(); got != 1 {
		t.Fatalf("busy selection launched %d lifecycle effects", got)
	}
	if got := r.adoptionCalls.Load(); got != 1 {
		t.Fatalf("busy selection adopted %d times", got)
	}
}
