package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"strings"
	"testing"
	"time"
)

func TestGroupMenuShowsScopeAndClearAction(t *testing.T) {
	rows := []couchcore.ActionableThreadSummary{presentationOrdinary("/src/repo", "one")}
	state := NewMenuState(rows, rows[0].Address)
	entry := PresentThreads(rows, nil, "")[0]
	scope := StatusGroupID{OwnerHub: entry.OwnerHub, GroupKey: entry.GroupKey}
	state.Frames[0].GroupScope = &scope
	lines, _ := renderRootMenuFrame(state, state.Frames[0], 80, 24, time.Time{}, false)
	if !strings.Contains(lines[0], entry.Label) || !strings.Contains(lines[0], "Esc: all") {
		t.Fatalf("scope has no visible recovery: %q", lines[0])
	}
	state.Inventory = nil
	lines, _ = renderRootMenuFrame(state, state.Frames[0], 80, 24, time.Time{}, false)
	if !strings.Contains(lines[0], "unavailable") || !strings.Contains(lines[0], "Esc: all") {
		t.Fatalf("empty scope has no recovery: %q", lines[0])
	}
}

func TestGroupMenuExactOwnerAndGroup(t *testing.T) {
	rows := []couchcore.ActionableThreadSummary{presentationSlot("/src/repo", 1), presentationSlot("/src/repo", 2), presentationSlot("/other/repo", 1)}
	ownership, err := couchcore.NewHubOwnership([]couchcore.HubDefinition{
		{Root: "/src/hub", Workspace: "/src", Slot: couchcore.HubSlot{Repository: "/src/repo", Number: 0}},
		{Root: "/src/other-hub", Workspace: "/src", Slot: couchcore.HubSlot{Repository: "/src/repo", Number: 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	state := NewMenuState(rows, couchcore.ThreadAddress{})
	state.HubOwnership = ownership
	entries := PresentThreads(rows, nil, "", ownership)
	var group StatusGroupID
	for _, p := range entries {
		if p.Row.RowKey == rows[0].RowKey {
			group = StatusGroupID{OwnerHub: p.OwnerHub, GroupKey: p.GroupKey}
		}
	}
	state.Frames[0].GroupScope = &group
	reconcileRootSelection(&state, couchcore.ThreadAddress{})
	got := VisibleMenuThreads(state)
	if len(got) != 1 || got[0].RowKey != rows[0].RowKey {
		t.Fatalf("scope leaked same-name/owner rows: %+v", got)
	}
	if state.Frames[0].SelectedKey != rows[0].RowKey {
		t.Fatal("selection escaped scope")
	}
	state.Frames[0].Filter = "no-match"
	if got := VisibleMenuThreads(state); len(got) != 0 {
		t.Fatalf("filter escaped scope: %+v", got)
	}
	state, effects := reduceKey(state, PanelKey{Kind: KeyEscape})
	if len(effects) != 0 || state.Frames[0].GroupScope == nil || state.Frames[0].Filter != "" {
		t.Fatal("first escape must clear text only")
	}
	state, effects = reduceKey(state, PanelKey{Kind: KeyEscape})
	if len(effects) != 0 || state.Frames[0].GroupScope != nil || len(VisibleMenuThreads(state)) != 3 {
		t.Fatal("second escape must clear scope only")
	}
}

func TestGroupMenuRefreshAndNestedReturnPreserveScope(t *testing.T) {
	rows := []couchcore.ActionableThreadSummary{presentationOrdinary("/src/repo", "one"), presentationOrdinary("/other/repo", "two")}
	rows[0].State = couchcore.ThreadLive
	rows[1].State = couchcore.ThreadLive
	state := NewMenuState(rows, rows[1].Address)
	entry := PresentThreads(rows[:1], nil, "")[0]
	group := StatusGroupID{OwnerHub: entry.OwnerHub, GroupKey: entry.GroupKey}
	state.Frames[0].GroupScope = &group
	reconcileRootSelection(&state, rows[1].Address)
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	if len(state.Frames) < 2 {
		t.Fatal("expected nested actions")
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
	if state.Frames[0].GroupScope == nil || len(VisibleMenuThreads(state)) != 1 {
		t.Fatal("nested return lost scope")
	}
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: rows[1:]})
	if state.Frames[0].GroupScope == nil || len(VisibleMenuThreads(state)) != 0 || state.Frames[0].SelectedAddress != (couchcore.ThreadAddress{}) {
		t.Fatalf("removed group broadened scope: %+v", state)
	}
}

func TestGroupMenuOpenAndOrdinaryReopen(t *testing.T) {
	rows := []couchcore.ActionableThreadSummary{presentationOrdinary("/src/repo", "one"), presentationOrdinary("/other/repo", "two")}
	c := New(newVTHost(24, 80), strings.NewReader(""))
	c.menu = NewMenuState(rows, rows[1].Address)
	c.menuReady = true
	c.active = "unchanged-child"
	c.focus = FocusActor(c.active)
	entry := PresentThreads(rows[:1], nil, "")[0]
	group := StatusGroupID{OwnerHub: entry.OwnerHub, GroupKey: entry.GroupKey}
	c.openGroupSwitcher(group)
	if !c.focus.IsPanel() || c.active != "unchanged-child" || c.menu.ActiveAddress != rows[1].Address {
		t.Fatal("group opener switched actor")
	}
	if c.menu.Frames[0].GroupScope == nil || len(VisibleMenuThreads(c.menu)) != 1 || c.menu.Frames[0].SelectedAddress != rows[0].Address {
		t.Fatal("opener failed to select inside scope")
	}
	c.menu.Frames[0].Filter = "draft"
	c.openGroupSwitcher(StatusGroupID{})
	if c.menu.Frames[0].GroupScope != nil || c.menu.Frames[0].Filter != "" || len(VisibleMenuThreads(c.menu)) != 2 {
		t.Fatal("global count didn't reset scope/filter")
	}
	c.openGroupSwitcher(group)
	c.focus = FocusActor(c.active)
	c.onHotkey()
	if c.menu.Frames[0].GroupScope != nil || len(VisibleMenuThreads(c.menu)) != 2 {
		t.Fatal("ordinary opener retained scope")
	}
}

func TestGroupMenuOwnershipChangeReconcilesSelection(t *testing.T) {
	row := presentationSlot("/src/repo", 2)
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, couchcore.ThreadAddress{})
	entry := PresentThreads([]couchcore.ActionableThreadSummary{row}, nil, "")[0]
	scope := StatusGroupID{OwnerHub: entry.OwnerHub, GroupKey: entry.GroupKey}
	state.Frames[0].GroupScope = &scope
	ownership, err := couchcore.NewHubOwnership([]couchcore.HubDefinition{{Root: "/src/hub", Workspace: "/src", Slot: couchcore.HubSlot{Repository: "/src/repo", Number: 0}}, {Root: "/src/other-hub", Workspace: "/src", Slot: couchcore.HubSlot{Repository: "/src/repo", Number: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventHubOwnership, HubOwnership: ownership})
	if len(VisibleMenuThreads(state)) != 0 || state.Frames[0].SelectedKey != (couchcore.ThreadRowKey{}) {
		t.Fatal("ownership refresh left selection outside scope")
	}
}
