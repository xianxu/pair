package couchtty

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// The reattach pass wired into the live console (pair#206 plan, Task 8).

// A BACKGROUND attach never takes focus -- the guard behind the plan review's
// Critical finding. c.active is empty whenever the last pane exited while the
// switcher was focused; a background completion arriving then must not hand
// the keyboard to a pane the operator never chose.
func TestABackgroundAttachNeverTakesFocus(t *testing.T) {
	f := newFixture(t, 24, 100)
	f.con.mu.Lock()
	f.con.active = ""
	f.con.focus = FocusPanel()
	f.con.mu.Unlock()

	child := ptychild.NewFakeChild(nil)
	address := menuAddress("couch-background")
	if err := f.con.installObservedThreadActor(context.Background(), "bg", "actor-bg", address,
		"/repo", "repo", child, couchcore.ProcessIdentity{PID: 7, Identity: "bg"}, true); err != nil {
		t.Fatal(err)
	}
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	if !f.con.focus.IsPanel() {
		t.Fatalf("focus = %+v after a background attach, want the switcher panel still focused", f.con.focus)
	}
	if _, installed := f.con.panes["bg"]; !installed {
		t.Fatal("the background pane was not installed")
	}
}

// And the foreground attach keeps today's behaviour: with nothing active, the
// first attach lands the operator on it.
func TestAForegroundAttachStillTakesFocusWhenNothingIsActive(t *testing.T) {
	f := newFixture(t, 24, 100)
	f.con.mu.Lock()
	f.con.active = ""
	f.con.focus = FocusPanel()
	f.con.mu.Unlock()

	child := ptychild.NewFakeChild(nil)
	if err := f.con.installObservedThreadActor(context.Background(), "fg", "actor-fg", menuAddress("couch-fg"),
		"/repo", "repo", child, couchcore.ProcessIdentity{PID: 8, Identity: "fg"}, false); err != nil {
		t.Fatal(err)
	}
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	if f.con.focus != FocusActor("fg") {
		t.Fatalf("focus = %+v, want the foreground pane", f.con.focus)
	}
}

// The pass, end to end through the console: the first inventory after arming
// dispatches warm-only background resumes, most recent first, one at a time.
// Each here fails, which marks its row and lets the pass continue.
func TestTheConsoleRunsThePassAsWarmOnlyBackgroundResumes(t *testing.T) {
	f := newFixture(t, 24, 100)
	f.con.mu.Lock()
	root := f.con.panes["c1"].thread
	f.con.mu.Unlock()

	var mu sync.Mutex
	var calls []map[string]string
	setTestOps(f.con, func(name string, args map[string]string) (any, error) {
		if name == "resume" {
			mu.Lock()
			calls = append(calls, args)
			mu.Unlock()
		}
		return nil, errors.New("spawn failed")
	})

	f.con.ArmReattachPass(root)
	now := time.Now()
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{
			{Address: root, State: couchcore.ThreadLive, LastActiveAt: now},
			{Address: menuAddress("couch-older"), State: couchcore.ThreadDetached, LastActiveAt: now.Add(-time.Hour)},
			{Address: menuAddress("couch-newer"), State: couchcore.ThreadDetached, LastActiveAt: now.Add(-time.Minute)},
		}, nil
	})

	waitUpTo(t, 2*time.Second, "two pass attempts", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(calls) >= 2
	})
	mu.Lock()
	got := append([]map[string]string(nil), calls...)
	mu.Unlock()
	if got[0]["tag"] != "couch-newer" || got[1]["tag"] != "couch-older" {
		t.Fatalf("attempt order = %s, %s; want the most recently active first", got[0]["tag"], got[1]["tag"])
	}
	for _, call := range got {
		if call["warm-only"] != "true" {
			t.Fatalf("a pass resume went out without warm-only: %v", call)
		}
	}
	waitUpTo(t, 2*time.Second, "both marked failed", func() bool {
		return len(f.con.menuSnapshot().Reattach.Failed) == 2
	})
}

// An unarmed console runs no pass and draws no placeholder -- every console
// that is not a bare startup keeps today's behaviour.
func TestAnUnarmedConsoleRunsNoPass(t *testing.T) {
	f := newFixture(t, 24, 100)
	var mu sync.Mutex
	resumes := 0
	setTestOps(f.con, func(name string, _ map[string]string) (any, error) {
		if name == "resume" {
			mu.Lock()
			resumes++
			mu.Unlock()
		}
		return nil, nil
	})
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{
			{Address: menuAddress("couch-detached"), State: couchcore.ThreadDetached, LastActiveAt: time.Now()},
		}, nil
	})
	waitUpTo(t, time.Second, "the inventory", func() bool { return f.con.menuSnapshot().InventoryReady })
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if resumes != 0 {
		t.Fatalf("an unarmed console dispatched %d resumes", resumes)
	}
	if placeholders := pendingPlaceholders(f.con.menuSnapshot().Reattach); len(placeholders) != 0 {
		t.Fatalf("an unarmed console has placeholders %+v", placeholders)
	}
}

// A placeholder is drawn greyed and is UNCLICKABLE by construction: it records
// no chip span, and the attached chips keep exactly the columns they had.
func TestPlaceholdersAreUnclickableAndLeaveAttachedChipsInPlace(t *testing.T) {
	attached := StatusActor{Label: "brain", Thread: menuAddress("couch-attached"), Active: true}
	without := RenderStatusRow(100, StatusModel{Actors: []StatusActor{attached}})
	with := RenderStatusRow(100, StatusModel{Spinner: 1, Actors: []StatusActor{
		attached,
		{Label: "worker", Thread: menuAddress("couch-loading"), Placeholder: true, Loading: true},
		{Label: "notes", Thread: menuAddress("couch-queued"), Placeholder: true},
	}})

	if len(with.Chips) != 1 || with.Chips[0] != without.Chips[0] {
		t.Fatalf("chips with placeholders = %+v, without = %+v; placeholders must add no span and move none", with.Chips, without.Chips)
	}
	for column := without.Chips[0].End; column < 100; column++ {
		if thread, ok := with.ColumnToActor(column); ok {
			t.Fatalf("column %d resolves to %v; nothing past the attached chip is clickable", column, thread)
		}
	}
	if !strings.Contains(with.Body, placeholderSGR) {
		t.Fatalf("body %q draws no placeholder greyed", with.Body)
	}
	if !strings.Contains(with.Body, spinnerGlyph(1)) {
		t.Fatalf("body %q shows no spinner on the loading placeholder", with.Body)
	}
}

// consoleThread reads one of the fixture's attached threads.
func consoleThread(f *consoleFixture, handle string) couchcore.ThreadAddress {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.panes[handle].thread
}

func detachedPair(root couchcore.ThreadAddress) func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
	now := time.Now()
	return func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{
			{Address: root, State: couchcore.ThreadLive, LastActiveAt: now},
			{Address: menuAddress("couch-first"), State: couchcore.ThreadDetached, LastActiveAt: now.Add(-time.Minute)},
			{Address: menuAddress("couch-second"), State: couchcore.ThreadDetached, LastActiveAt: now.Add(-time.Hour)},
		}, nil
	}
}

// Quitting mid-attempt cancels the attempt in flight and starts no other: the
// queue is dropped with the console (pair#206 cell 13). The dispatcher is set
// directly, not through setTestOps, because setTestOps does not forward the
// call's Context -- and cancellation is the whole point here.
func TestStopCancelsAnInFlightPassAttemptAndRunsNoMore(t *testing.T) {
	f := newFixture(t, 24, 100)
	root := consoleThread(f, "c1")
	started := make(chan struct{}, 4)
	cancelled := make(chan struct{}, 4)
	var resumes int32
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		if call.Name != "resume" {
			return nil, nil
		}
		atomic.AddInt32(&resumes, 1)
		started <- struct{}{}
		<-call.Context.Done()
		cancelled <- struct{}{}
		return nil, call.Context.Err()
	})
	f.con.ArmReattachPass(root)
	f.con.SetActionableProvider(detachedPair(root))

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("the first pass attempt never started")
	}
	f.con.Stop()
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not cancel the attempt in flight")
	}
	select {
	case <-f.done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after Stop")
	}
	if n := atomic.LoadInt32(&resumes); n != 1 {
		t.Fatalf("resumes = %d after Stop, want exactly the one that was in flight", n)
	}
}

// An operator switch dispatched while a pass attempt runs waits behind THAT
// attempt only: the pass holds while the operator's operation is in flight
// (cell 10), so the next attempt cannot slip in ahead of the switch. The bound
// the plan promises is "at most one attempt", and this is its observation.
func TestAnOperatorSwitchWaitsBehindAtMostTheRunningAttempt(t *testing.T) {
	f := newFixture(t, 24, 100)
	root := consoleThread(f, "c1")
	second := ptychild.NewFakeChild(nil)
	second.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
	f.con.Attach("c2", "worker", second)
	live := consoleThread(f, "c2")

	order := make(chan string, 8)
	release := make(chan struct{})
	var once sync.Once
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		switch call.Name {
		case "resume":
			order <- "resume:" + call.Args["tag"]
			once.Do(func() { <-release }) // hold only the first attempt
			return nil, errors.New("spawn failed")
		case "switch":
			order <- "switch"
			return f.con.ExecuteConsoleOperation(call)
		}
		return nil, nil
	})
	now := time.Now()
	f.con.ArmReattachPass(root)
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{
			{Address: root, State: couchcore.ThreadLive, LastActiveAt: now},
			{Address: live, State: couchcore.ThreadLive, LastActiveAt: now},
			{Address: menuAddress("couch-first"), State: couchcore.ThreadDetached, LastActiveAt: now.Add(-time.Minute)},
			{Address: menuAddress("couch-second"), State: couchcore.ThreadDetached, LastActiveAt: now.Add(-time.Hour)},
		}, nil
	})

	if got := <-order; got != "resume:couch-first" {
		t.Fatalf("first dispatch = %q, want the first pass attempt", got)
	}
	// The operator switches to the live thread while that attempt is held.
	f.con.mu.Lock()
	f.con.menu.Frames[0].SelectedAddress = live
	f.con.mu.Unlock()
	f.con.reduceMenu(MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyEnter}})
	close(release)

	var rest []string
	for len(rest) < 2 {
		select {
		case got := <-order:
			rest = append(rest, got)
		case <-time.After(2 * time.Second):
			t.Fatalf("after the held attempt, got %v; want the switch then the next attempt", rest)
		}
	}
	if rest[0] != "switch" || rest[1] != "resume:couch-second" {
		t.Fatalf("order after the held attempt = %v, want [switch resume:couch-second]", rest)
	}
}

// The status row's spinner ticks only while a thread is loading. It is a new
// periodic repaint, so it must stop the moment the pass has nothing in flight.
func TestTheStatusTickRunsOnlyWhileAThreadIsLoading(t *testing.T) {
	f := newFixture(t, 24, 100)
	root := consoleThread(f, "c1")
	release := make(chan struct{})
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		if call.Name == "resume" {
			<-release
			return nil, errors.New("spawn failed")
		}
		return nil, nil
	})
	f.con.ArmReattachPass(root)
	now := time.Now()
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{
			{Address: root, State: couchcore.ThreadLive, LastActiveAt: now},
			{Address: menuAddress("couch-only"), State: couchcore.ThreadDetached, LastActiveAt: now},
		}, nil
	})
	spinner := func() uint8 {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.statusSpinner
	}
	waitUpTo(t, 2*time.Second, "a thread loading", func() bool {
		return f.con.menuSnapshot().Reattach.Loading != (couchcore.ThreadAddress{})
	})
	before := spinner()
	waitUpTo(t, 2*time.Second, "the spinner to advance while loading", func() bool { return spinner() != before })

	close(release)
	waitUpTo(t, 2*time.Second, "the pass to finish", func() bool {
		return f.con.menuSnapshot().Reattach.Loading == (couchcore.ThreadAddress{})
	})
	settled := spinner()
	time.Sleep(4 * statusSpinnerInterval)
	if now := spinner(); now != settled {
		t.Fatalf("spinner moved %d -> %d with nothing loading; the tick must stop", settled, now)
	}
}

// statusModelOf reads the status model the way paintNow builds it.
func statusModelOf(f *consoleFixture) StatusModel {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.statusModelLocked()
}

// The status model is where a placeholder appears or silently does not, so it
// is pinned HERE rather than only through RenderStatusRow with a hand-built
// model: paintNow must add one placeholder per pending thread, after the
// attached chips, the loading one flagged, labelled as its chip will be.
func TestTheStatusModelCarriesAPlaceholderPerPendingThread(t *testing.T) {
	f := newFixture(t, 24, 100)
	root := consoleThread(f, "c1")
	release := make(chan struct{})
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		if call.Name == "resume" {
			<-release
			return nil, errors.New("spawn failed")
		}
		return nil, nil
	})
	t.Cleanup(func() { close(release) })
	f.con.ArmReattachPass(root)
	now := time.Now()
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{
			{Address: root, State: couchcore.ThreadLive, LastActiveAt: now},
			{Address: menuAddress("couch-first"), StartingPath: "/work/first", State: couchcore.ThreadDetached, LastActiveAt: now.Add(-time.Minute)},
			{Address: menuAddress("couch-second"), StartingPath: "/work/second", State: couchcore.ThreadDetached, LastActiveAt: now.Add(-time.Hour)},
		}, nil
	})
	waitUpTo(t, 2*time.Second, "the first thread loading", func() bool {
		return f.con.menuSnapshot().Reattach.Loading == menuAddress("couch-first")
	})

	model := statusModelOf(f)
	var placeholders []StatusActor
	for i, actor := range model.Actors {
		if actor.Placeholder {
			placeholders = append(placeholders, actor)
			continue
		}
		if len(placeholders) > 0 {
			t.Fatalf("attached chip %q at %d follows a placeholder; attached chips come first", actor.Label, i)
		}
	}
	if len(placeholders) != 2 {
		t.Fatalf("placeholders = %+v, want one per pending thread", placeholders)
	}
	first, second := placeholders[0], placeholders[1]
	if first.Thread != menuAddress("couch-first") || !first.Loading || second.Thread != menuAddress("couch-second") || second.Loading {
		t.Fatalf("placeholders = %+v, want couch-first loading then couch-second queued", placeholders)
	}
	// Labelled as the chip will be -- the starting path's repository -- so the
	// label does not change when the thread arrives.
	if first.Label != couchcore.Worktree("/work/first").Repo() || second.Label != couchcore.Worktree("/work/second").Repo() {
		t.Fatalf("labels %q, %q; want each thread's repository", first.Label, second.Label)
	}
}

// resumeSucceeds makes a pass resume of `tag` SUCCEED with a real attachable
// child, and every other operation behave as the console would. The console
// tests above all fail their resumes, which never reaches the success path --
// and the success path is where focus could be stolen.
func resumeSucceeds(t *testing.T, f *consoleFixture, tag string) couchcore.StartResult {
	t.Helper()
	started, _ := attachStartResult(t, "actor-pass", menuAddress(tag))
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		switch call.Name {
		case "resume":
			if call.Args["tag"] == tag {
				return started, nil
			}
			return nil, errors.New("spawn failed")
		case "attach", "switch":
			return f.con.ExecuteConsoleOperation(call)
		}
		return nil, nil
	})
	return started
}

// A background SUCCESS never steals focus: the operator is in their thread, the
// pass reattaches another behind them, and the keyboard stays where it was.
func TestABackgroundSuccessNeverStealsFocus(t *testing.T) {
	f := newFixture(t, 24, 100)
	root := consoleThread(f, "c1")
	started := resumeSucceeds(t, f, "couch-first")
	f.con.ArmReattachPass(root)
	f.con.SetActionableProvider(detachedPair(root))

	waitUpTo(t, 2*time.Second, "the pass to attach couch-first", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		_, attached := f.con.panes[started.Handle.ID()]
		return attached
	})
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	if f.con.focus != FocusActor("c1") {
		t.Fatalf("focus = %+v after a background attach, want the operator's own thread c1", f.con.focus)
	}
}

// And with NOTHING active -- the last pane exited while the switcher was
// focused -- a background success still leaves the switcher focused. This is
// the end-to-end form of the installer guard: it only holds if finishOperation
// carries `background` across the declared attach operation to the installer.
func TestABackgroundSuccessWithNothingActiveLeavesTheSwitcherFocused(t *testing.T) {
	f := newFixture(t, 24, 100)
	root := consoleThread(f, "c1")
	started := resumeSucceeds(t, f, "couch-first")
	f.con.mu.Lock()
	f.con.active = ""
	f.con.focus = FocusPanel()
	f.con.mu.Unlock()
	f.con.ArmReattachPass(root)
	f.con.SetActionableProvider(detachedPair(root))

	waitUpTo(t, 2*time.Second, "the pass to attach couch-first", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		_, attached := f.con.panes[started.Handle.ID()]
		return attached
	})
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	if !f.con.focus.IsPanel() {
		t.Fatalf("focus = %+v, want the switcher still focused: the operator never chose this pane", f.con.focus)
	}
}

// The frame the pass leaves behind is painted (found in the operator's smoke
// test, 2026-09-12). While a thread loads, the status tick is the only thing
// that repaints the status row. So when the last attempt landed and the tick
// stopped, the row kept showing that thread's spinning placeholder until
// something else repainted it, such as opening the switcher.
//
// statusChips is what the last paint made clickable, so the attached thread's
// chip appears there only once a frame is painted after its attach.
func TestTheFrameThePassLeavesBehindIsPainted(t *testing.T) {
	f := newFixture(t, 24, 100)
	root := consoleThread(f, "c1")
	started := resumeSucceeds(t, f, "couch-first")
	f.con.ArmReattachPass(root)
	now := time.Now()
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{
			{Address: root, State: couchcore.ThreadLive, LastActiveAt: now},
			{Address: menuAddress("couch-first"), State: couchcore.ThreadDetached, LastActiveAt: now.Add(-time.Minute)},
		}, nil
	})
	waitUpTo(t, 2*time.Second, "the pass to attach couch-first and finish", func() bool {
		f.con.mu.Lock()
		_, attached := f.con.panes[started.Handle.ID()]
		f.con.mu.Unlock()
		return attached && f.con.menuSnapshot().Reattach.Phase == ReattachDone
	})
	waitUpTo(t, time.Second, "a status row painted with couch-first's chip", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		for _, chip := range f.con.statusChips {
			if chip.Thread == menuAddress("couch-first") {
				return true
			}
		}
		return false
	})
}
