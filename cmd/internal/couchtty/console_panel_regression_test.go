package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// These regressions exercise Console lifecycle behavior that is not owned by
// the pure hierarchical-menu reducer. Menu navigation, filtering, forms, and
// operation restoration live in menu*_test.go and console_run_menu_test.go.

func TestNonRootAltXParksOnlySelectedActor(t *testing.T) {
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	rootAddress := menuAddress("root")
	otherAddress := menuAddress("other")
	con.attachThreadActor("root", "root", rootAddress, "/w/root", "root", ptychild.NewFakeChild(nil))
	con.attachThreadActor("other", "other", otherAddress, "/w/other", "other", ptychild.NewFakeChild(nil))
	con.mu.Lock()
	con.active = "other"
	con.focus = FocusActor("other")
	con.menu = NewMenuState([]couchcore.ActionableThreadSummary{
		{Address: rootAddress, Name: "root", State: couchcore.ThreadLive},
		{Address: otherAddress, Name: "other", State: couchcore.ThreadLive},
	}, otherAddress)
	con.menuReady = true
	con.mu.Unlock()

	con.onParkHotkey()
	state := con.menuSnapshot()
	if frame := state.CurrentFrame(); frame.Kind != MenuFrameConfirmation || frame.Action != "park" || frame.Thread != otherAddress {
		t.Fatalf("park confirmation = %+v", frame)
	}
}

func TestLastActorExitWhileMenuFocusedKeepsConsoleForResume(t *testing.T) {
	address := menuAddress("parked")
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	child := ptychild.NewFakeChild(nil)
	con.attachThreadActor("only", "actor", address, "/w/parked", "parked", child)
	con.mu.Lock()
	con.focus = FocusPanel()
	con.menu = NewMenuState([]couchcore.ActionableThreadSummary{{
		Address: address, WorkingPath: "/w/parked", Name: "parked", State: couchcore.ThreadLive,
	}}, address)
	con.menuReady = true
	con.mu.Unlock()

	if exitConsole := con.onExit(childExit{id: "only", code: 0}); exitConsole {
		t.Fatal("last actor exit closed the menu before its parked row could be refreshed")
	}
	con.mu.Lock()
	focus := con.focus
	con.mu.Unlock()
	if !focus.IsPanel() {
		t.Fatalf("last actor exit changed menu focus to %+v", focus)
	}
	select {
	case <-con.stop:
		t.Fatal("last actor exit stopped the focused menu")
	default:
	}
}

func TestExpectedParkExitDoesNotPublishActorExitNotice(t *testing.T) {
	for _, completionFirst := range []bool{false, true} {
		name := "exit-before-completion"
		if completionFirst {
			name = "completion-before-exit"
		}
		t.Run(name, func(t *testing.T) {
			address := menuAddress("parked")
			con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
			t.Cleanup(con.Stop)
			child := ptychild.NewFakeChild(nil)
			con.attachThreadActor("parked-handle", "parked-actor", address, "/w/parked", "parked", child)
			state := NewMenuState([]couchcore.ActionableThreadSummary{{Address: address, Name: "parked", State: couchcore.ThreadLive}}, address)
			state, effects := dispatchThreadOperation(state, "park", address)
			if len(effects) != 1 {
				t.Fatalf("park effects = %+v", effects)
			}
			con.mu.Lock()
			con.menu, con.menuReady, con.focus = state, true, FocusPanel()
			con.mu.Unlock()
			completion := operationCompletion{
				name: "park", origin: state.InFlight,
				value: couchcore.ParkResult{Thread: couchcore.ThreadRecord{Address: address}},
			}
			if completionFirst {
				con.finishOperation(completion)
			}
			con.onExit(childExit{id: "parked-handle", code: 0})
			if !completionFirst {
				con.finishOperation(completion)
			}
			if latest := con.feed.Latest(); strings.Contains(latest, "exited (0)") {
				t.Fatalf("expected park shutdown leaked actor exit notice: %q", latest)
			}
		})
	}
}

func TestActiveExitFallsBackToASurvivingActorForMenuActions(t *testing.T) {
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	root := ptychild.NewFakeChild(nil)
	other := ptychild.NewFakeChild(nil)
	con.attachThreadActor("root", "root-actor", menuAddress("root"), "/w/root", "root", root)
	con.attachThreadActor("other", "other-actor", menuAddress("other"), "/w/other", "other", other)
	con.mu.Lock()
	con.active = "other"
	con.focus = FocusPanel()
	con.mu.Unlock()

	if exitConsole := con.onExit(childExit{id: "other", code: 0}); exitConsole {
		t.Fatal("non-root exit closed a console whose root remained live")
	}
	con.mu.Lock()
	active := con.active
	con.mu.Unlock()
	if active != "root" {
		t.Fatalf("active target after non-root exit = %q, want root", active)
	}
}
