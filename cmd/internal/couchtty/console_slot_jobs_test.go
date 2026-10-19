package couchtty

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestConsoleHostedNavigationSurvivesSaturatedLifecycleLane(t *testing.T) {
	var switches atomic.Int32
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) {
		c.queueWorkers = 2
		fn := c.Ops()
		c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
			if call.Name == "switch" {
				switches.Add(1)
			}
			return fn(call)
		})
	})
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	_, err := f.con.operationQueue.Enqueue(operationRequest{key: "held", run: func() (any, error) { close(entered); <-release; return nil, nil }})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-testwait.After(t):
		t.Fatal("lifecycle worker did not enter")
	}
	for i := 0; i < 15; i++ {
		_, err = f.con.operationQueue.Enqueue(operationRequest{key: string(rune('a' + i)), run: func() (any, error) { <-release; return nil, nil }})
		if err != nil {
			t.Fatal(err)
		}
	}
	var attempt uint64
	err = f.con.runTerminalCommand(context.Background(), func() error {
		f.con.mu.Lock()
		address := f.con.panes["c1"].thread
		state := NewMenuState([]couchcore.ActionableThreadSummary{{Address: address, State: couchcore.ThreadLive}}, address)
		var effects []MenuEffect
		state, effects = dispatchThreadOperation(state, "switch", address)
		attempt = effects[0].Attempt
		f.con.menu, f.con.menuReady = state, true
		f.con.mu.Unlock()
		f.con.dispatchMenuEffects(effects)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	testwait.Until(t, "hosted navigation while all lifecycle admission is occupied", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		_, _, running := menuOperationForAttempt(f.con.menu, attempt)
		return !running && f.con.focus == FocusActor("c1")
	})
	if switches.Load() != 1 {
		t.Fatalf("hosted navigation never executed: calls=%d", switches.Load())
	}
	for _, job := range f.con.menuSnapshot().Operations {
		if job.Origin.Attempt == attempt && job.Phase != MenuOperationSucceeded {
			t.Fatalf("hosted switch refused: %+v", job)
		}
	}
}

func TestConsoleLateResumeNeverStealsNewerFocus(t *testing.T) {
	for _, waitHere := range []bool{false, true} {
		t.Run(map[bool]string{false: "navigate-away", true: "wait-on-row"}[waitHere], func(t *testing.T) {
			f := newFixture(t, 24, 80)
			address := menuAddress("late")
			start, _ := attachStartResult(t, "late-actor", address)
			t.Cleanup(func() { _ = start.Handle.(couchcore.TerminalHandle).Terminal().Close() })
			err := f.con.runTerminalCommand(context.Background(), func() error {
				f.con.mu.Lock()
				state := NewMenuState([]couchcore.ActionableThreadSummary{{Address: address, State: couchcore.ThreadParked}}, address)
				state, effects := dispatchThreadOperation(state, "resume", address)
				origin := menuOriginForAttempt(state, effects[0].Attempt)
				if !waitHere {
					state.LandingAttempt = 0
				}
				f.con.menu, f.con.menuReady = state, true
				f.con.mu.Unlock()
				completed := f.con.prepareOperationCompletion(operationCompletion{name: "resume", origin: origin, value: start})
				f.con.finishOperation(completed)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			f.con.mu.Lock()
			focus := f.con.focus
			_, adopted := f.con.panes[start.Handle.ID()]
			f.con.mu.Unlock()
			if !adopted {
				t.Fatal("late child must still be adopted")
			}
			want := FocusActor("c1")
			if waitHere {
				want = FocusActor(start.Handle.ID())
			}
			if focus != want {
				t.Fatalf("focus=%+v want%+v", focus, want)
			}
		})
	}
}

func TestSwitcherCopiesSelectedOperationFailure(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "clipboard")
	if err := os.WriteFile(filepath.Join(dir, "pbcopy"), []byte("#!/bin/sh\n/bin/cat > \"$TEST_NOTICE_CLIPBOARD\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("TEST_NOTICE_CLIPBOARD", output)
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(key, "")
	}
	full := "failure details 界 " + strings.Repeat("retained diagnostic ", 40) + "last byte"
	f := newFixtureBeforeRun(t, 24, 44, func(c *Console) {
		address := c.panes["c1"].thread
		state := NewMenuState([]couchcore.ActionableThreadSummary{{Address: address, State: couchcore.ThreadLive}}, address)
		state, effects := dispatchThreadOperation(state, "park", address)
		state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "park", Attempt: effects[0].Attempt, Address: address, Error: full})
		state.Notice = infoMenuNotice("unrelated subsequent event")
		c.menu, c.menuReady = state, true
		c.feed.Push(Notice{Body: "another unrelated notice"})
	})
	f.con.onMenuKey(PanelKey{Kind: KeyCopyNotice})
	got, err := os.ReadFile(output)
	if err != nil || string(got) != full {
		t.Fatalf("selected diagnostic copy = %q, %v", got, err)
	}
}

func TestConsoleSupersededSwitchReleasesCaptureWithoutAcknowledging(t *testing.T) {
	f := newFixture(t, 24, 80)
	address := menuAddress("paged")
	f.con.mu.Lock()
	f.con.attention.Mark(address, "needs attention")
	capture := f.con.attention.Capture(address)
	f.con.mu.Unlock()
	f.con.finishOperation(operationCompletion{origin: MenuOperationOrigin{Operation: "switch", Attempt: 123, Address: address, AttentionCapture: capture}})
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	if _, exists := f.con.attention.captures[capture]; exists {
		t.Fatal("superseded switch retained its attention capture")
	}
	if len(f.con.attention.Projection(address)) != 1 {
		t.Fatal("a switch that never landed acknowledged a notification")
	}
}

func TestSwitcherCopiesMissingTargetOperationNotice(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "clipboard")
	if err := os.WriteFile(filepath.Join(dir, "pbcopy"), []byte("#!/bin/sh\n/bin/cat > \"$TEST_NOTICE_CLIPBOARD\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("TEST_NOTICE_CLIPBOARD", output)
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(key, "")
	}
	full := "missing target: " + strings.Repeat("unclipped reason ", 40) + "final reason"
	f := newFixtureBeforeRun(t, 24, 44, func(c *Console) {
		c.menu.Notice = errorMenuNotice(full)
		c.feed.Push(Notice{Body: "unrelated message"})
	})
	f.con.onMenuKey(PanelKey{Kind: KeyCopyNotice})
	got, err := os.ReadFile(output)
	if err != nil || string(got) != full {
		t.Fatalf("missing-target diagnostic copy=%q,%v", got, err)
	}
}
