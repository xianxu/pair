package couchtty

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestConsoleFocusViewPersistsAcrossSwitchAndReopen(t *testing.T) {
	f := liveMenuFixture(t)
	address := f.con.menuSnapshot().ActiveAddress
	var description atomic.Value
	description.Store("working on focus view")
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{
			{Address: address, WorkingPath: "/root", State: couchcore.ThreadLive, PublishedSummary: description.Load().(string)},
			{Address: menuAddress("parked"), WorkingPath: "/parked", State: couchcore.ThreadParked, PublishedSummary: "not live"},
		}, nil
	})
	waitFor(t, "two inventory rows", func() bool { return len(f.con.menuSnapshot().Inventory) == 2 })
	send := func(text string) {
		t.Helper()
		if _, err := f.stdin.Write([]byte(text)); err != nil {
			t.Fatal(err)
		}
	}
	panel := func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.focus.IsPanel()
	}
	actorReady := func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		// Switching moves focus before its asynchronous completion clears the
		// in-flight operation. Reopening early can make a later Enter get
		// rejected as a concurrent operation instead of testing the view.
		return !f.con.focus.IsPanel() && f.con.menu.InFlight.Operation == ""
	}
	send("\x00")
	waitFor(t, "normal switcher", func() bool { return panel() && len(VisibleMenuThreads(f.con.menuSnapshot())) == 2 })
	send(" ")
	waitFor(t, "focus view", func() bool {
		state := f.con.menuSnapshot()
		return state.CurrentFrame().Filter == "" && len(VisibleMenuThreads(state)) == 1 && strings.Contains(f.screenText(), "◆ working on focus view")
	})
	description.Store("updated tag")
	f.con.requestMenuRefresh()
	waitFor(t, "published description refresh", func() bool { return strings.Contains(f.screenText(), "◆ updated tag") })
	description.Store("")
	f.con.requestMenuRefresh()
	waitFor(t, "untagged thread removed", func() bool {
		state := f.con.menuSnapshot()
		return len(VisibleMenuThreads(state)) == 0 && state.CurrentFrame().SelectedAddress == (couchcore.ThreadAddress{}) && strings.Contains(f.screenText(), "no tagged live threads")
	})
	description.Store("retagged")
	f.con.requestMenuRefresh()
	waitFor(t, "retagged thread restored", func() bool {
		return len(VisibleMenuThreads(f.con.menuSnapshot())) == 1 && strings.Contains(f.screenText(), "◆ retagged")
	})
	// The real input/operation path switches to the hosted actor, then Ctrl-Space
	// reopens the retained root frame and clears only its typeahead.
	send("root")
	waitFor(t, "focus typeahead", func() bool { return f.con.menuSnapshot().CurrentFrame().Filter == "root" })
	send("\r")
	waitFor(t, "actor switch completion", actorReady)
	send("\x00")
	waitFor(t, "reopened focus view", func() bool {
		state := f.con.menuSnapshot()
		return panel() && state.CurrentFrame().Filter == "" && len(VisibleMenuThreads(state)) == 1
	})
	if got := f.con.menuSnapshot().CurrentFrame().SelectedAddress; got != address {
		t.Fatalf("reopen selected %v, want active thread %v", got, address)
	}
	send(" ")
	waitFor(t, "normal view restored", func() bool { return len(VisibleMenuThreads(f.con.menuSnapshot())) == 2 })
	send("\r")
	waitFor(t, "actor switch completion again", actorReady)
	send("\x00")
	waitFor(t, "reopened normal view", func() bool {
		return panel() && len(VisibleMenuThreads(f.con.menuSnapshot())) == 2
	})
}
