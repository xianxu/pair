package couchtty

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func switcherRowClick(row couchcore.ActionableThreadSummary, reveal bool) MenuEvent {
	return MenuEvent{Kind: MenuEventMouseRowClick, RowKey: menuRowKey(row), Address: row.Address, RevealTarget: reveal}
}

func requireMouseSelection(t *testing.T, state MenuState, row couchcore.ActionableThreadSummary, effects []MenuEffect) {
	t.Helper()
	if len(effects) != 0 || len(state.Frames) != 1 || state.CurrentFrame().SelectedKey != menuRowKey(row) {
		t.Fatalf("click must select without effects: frame=%+v effects=%+v", state.CurrentFrame(), effects)
	}
}

func TestSwitcherRowClicksSelectThenActivateExactlyOnce(t *testing.T) {
	for _, count := range []int{8, 16} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			dispatched := 0
			for i := range count {
				rows := menuThreads()
				state := NewMenuState(rows, rows[0].Address)
				selectMenuRow(&state.Frames[0], rows[0])
				next, effects := ReduceMenu(state, switcherRowClick(rows[1], false))
				requireMouseSelection(t, next, rows[1], effects)
				if state.CurrentFrame().SelectedKey != menuRowKey(rows[0]) {
					t.Fatal("selection mutated the retained prior frame")
				}
				next, effects = ReduceMenu(next, switcherRowClick(rows[1], false))
				if len(effects) != 1 || effects[0].Operation != "resume" || effects[0].Args["tag"] != string(rows[1].Address.Tag) {
					t.Fatalf("pair %d selected-row activation: %+v", i, effects)
				}
				if origin := menuOriginForAttempt(next, effects[0].Attempt); !origin.Manual || origin.AttentionCapture != 0 {
					t.Fatalf("mouse activation acquired paging semantics: %+v", origin)
				}
				dispatched += len(effects)
			}
			if dispatched != count {
				t.Fatalf("%d click pairs dispatched %d operations", count, dispatched)
			}
		})
	}
}

func TestSwitcherClicksUseDistinctRecordlessSlotKeys(t *testing.T) {
	one, two := menuSlotRow(1, ""), menuSlotRow(2, "")
	state := NewMenuState([]couchcore.ActionableThreadSummary{one, two}, couchcore.ThreadAddress{})
	selectMenuRow(&state.Frames[0], one)
	state, effects := ReduceMenu(state, switcherRowClick(two, false))
	requireMouseSelection(t, state, two, effects)
	state, effects = ReduceMenu(state, switcherRowClick(one, false))
	requireMouseSelection(t, state, one, effects)
	_, effects = ReduceMenu(state, switcherRowClick(one, false))
	if len(effects) != 1 || effects[0].Operation != "resume" || effects[0].Args["path"] != one.WorkingPath {
		t.Fatalf("selected recordless slot: %+v", effects)
	}
}

func TestSwitcherClickSelectionSurvivesReorderAndSlotConversationReplacement(t *testing.T) {
	one, two := menuSlotRow(1, "old-one"), menuSlotRow(2, "old-two")
	state := NewMenuState([]couchcore.ActionableThreadSummary{one, two}, one.Address)
	selectMenuRow(&state.Frames[0], one)
	state, effects := ReduceMenu(state, switcherRowClick(two, false))
	requireMouseSelection(t, state, two, effects)
	replacement := menuSlotRow(2, "new-two")
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: []couchcore.ActionableThreadSummary{replacement, one}, Generation: 1})
	_, effects = ReduceMenu(state, switcherRowClick(two, false))
	if len(effects) != 1 || effects[0].Args["path"] != replacement.WorkingPath || effects[0].Args["tag"] == "old-two" {
		t.Fatalf("stable slot click did not resolve current inventory: %+v", effects)
	}

	// An ordinary row's former screen position cannot authorize its replacement.
	rows := menuThreads()
	state = NewMenuState(rows, rows[0].Address)
	state, _ = ReduceMenu(state, switcherRowClick(rows[1], false))
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: rows[:1], Generation: 2})
	_, effects = ReduceMenu(state, switcherRowClick(rows[1], false))
	if len(effects) != 0 {
		t.Fatalf("removed row's stale hit dispatched: %+v", effects)
	}
}

func TestSwitcherHiddenChipRevealsAndSelectsBeforeActivation(t *testing.T) {
	rows := []couchcore.ActionableThreadSummary{groupedRow("/workspace/pair", 1, "one"), groupedRow("/workspace/brain", 1, "two")}
	entry := PresentThreads(rows[:1], nil, "")[0]
	group := StatusGroupID{OwnerHub: entry.OwnerHub, GroupKey: entry.GroupKey}
	for _, scope := range []string{"filter", "group", "both"} {
		t.Run(scope, func(t *testing.T) {
			state := NewMenuState(rows, rows[0].Address)
			selectMenuRow(&state.Frames[0], rows[0])
			if scope != "group" {
				state.Frames[0].Filter = "pair:1"
			}
			if scope != "filter" {
				state.Frames[0].GroupScope = &group
			}
			before := state.CurrentFrame()
			ignored, effects := ReduceMenu(state, switcherRowClick(rows[1], false))
			if len(effects) != 0 || !reflect.DeepEqual(ignored.CurrentFrame(), before) {
				t.Fatalf("hidden root row escaped filter: frame=%+v effects=%+v", ignored.CurrentFrame(), effects)
			}
			state, effects = ReduceMenu(state, switcherRowClick(rows[1], true))
			requireMouseSelection(t, state, rows[1], effects)
			visible := false
			for _, row := range VisibleMenuThreads(state) {
				visible = visible || menuRowKey(row) == menuRowKey(rows[1])
			}
			if !visible {
				t.Fatal("revealed chip selection remains hidden")
			}
			_, effects = ReduceMenu(state, switcherRowClick(rows[1], true))
			if len(effects) != 1 || effects[0].Operation != "switch" || effects[0].Args["tag"] != "two" {
				t.Fatalf("revealed selected chip did not activate: %+v", effects)
			}
		})
	}
}

func TestSwitcherVisibleChipPreservesFilterAndGroup(t *testing.T) {
	rows := []couchcore.ActionableThreadSummary{groupedRow("/workspace/pair", 1, "one"), groupedRow("/workspace/pair", 2, "two")}
	entry := PresentThreads(rows[:1], nil, "")[0]
	group := StatusGroupID{OwnerHub: entry.OwnerHub, GroupKey: entry.GroupKey}
	state := NewMenuState(rows, rows[0].Address)
	selectMenuRow(&state.Frames[0], rows[0])
	state.Frames[0].Filter = "pair"
	state.Frames[0].GroupScope = &group
	state, effects := ReduceMenu(state, switcherRowClick(rows[1], true))
	requireMouseSelection(t, state, rows[1], effects)
	if state.Frames[0].Filter != "pair" || state.Frames[0].GroupScope == nil || *state.Frames[0].GroupScope != group {
		t.Fatalf("visible chip unnecessarily cleared context: %+v", state.Frames[0])
	}
}

func TestSwitcherNestedChipReturnsToSelectionBeforeActivation(t *testing.T) {
	rows := menuThreads()
	state := NewMenuState(rows, rows[0].Address)
	selectMenuRow(&state.Frames[0], rows[0])
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	if state.CurrentFrame().Kind != MenuFrameActions {
		t.Fatal("fixture did not open actions")
	}
	state, effects := ReduceMenu(state, switcherRowClick(rows[0], true))
	requireMouseSelection(t, state, rows[0], effects)
	_, effects = ReduceMenu(state, switcherRowClick(rows[0], true))
	if len(effects) != 1 || effects[0].Operation != "switch" {
		t.Fatalf("selected root chip did not activate: %+v", effects)
	}
}

func TestSwitcherSelectedClickPreservesExistingOperationVetoes(t *testing.T) {
	t.Run("pending reattach", func(t *testing.T) {
		state, _ := passMenu(t, passInventory())
		row, ok := menuThread(state, menuAddress("couch-a"))
		if !ok {
			t.Fatal("missing pending fixture")
		}
		before := state.CurrentFrame()
		for range 2 {
			var effects []MenuEffect
			state, effects = ReduceMenu(state, switcherRowClick(row, false))
			if len(effects) != 0 || !reflect.DeepEqual(state.CurrentFrame(), before) {
				t.Fatalf("pending row accepted selection or operation: frame=%+v effects=%+v", state.CurrentFrame(), effects)
			}
		}
	})
	t.Run("same target operation", func(t *testing.T) {
		rows := menuThreads()
		state := NewMenuState(rows, rows[0].Address)
		selectMenuRow(&state.Frames[0], rows[0])
		origin := MenuOperationOrigin{Operation: "park", Attempt: 7, Address: rows[0].Address}
		setTestMenuOperation(&state, origin)
		state, effects := ReduceMenu(state, switcherRowClick(rows[0], false))
		if len(effects) != 0 || state.operationForTest() != origin {
			t.Fatalf("click replaced or relabelled in-flight work: origin=%+v effects=%+v", state.operationForTest(), effects)
		}
	})
	for _, rowState := range []couchcore.ActionableThreadState{couchcore.ThreadBusy, couchcore.ThreadUnusable} {
		t.Run(string(rowState), func(t *testing.T) {
			row := menuThreads()[0]
			row.State, row.Reason = rowState, couchcore.ReasonUnknown
			state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
			selectMenuRow(&state.Frames[0], row)
			wantState, wantEffects := ReduceMenu(state, MenuEvent{Kind: MenuEventMouseSwitch, RowKey: menuRowKey(row), Address: row.Address})
			gotState, gotEffects := ReduceMenu(state, switcherRowClick(row, false))
			if !reflect.DeepEqual(gotEffects, wantEffects) || !reflect.DeepEqual(gotState, wantState) {
				t.Fatalf("selected click bypassed existing authority: got frame=%+v effects=%+v; want frame=%+v effects=%+v", gotState.CurrentFrame(), gotEffects, wantState.CurrentFrame(), wantEffects)
			}
		})
	}
}

func TestSwitcherSelectionClearsPriorNavigationFeedback(t *testing.T) {
	rows := menuThreads()
	state := NewMenuState(rows, rows[0].Address)
	state.LandingAttempt = 42
	state.CopyFeedback = CopyFeedback{Kind: CopyFeedbackSuccess, Text: "copied"}
	state.Notice = infoMenuNotice("old notice")
	next, effects := ReduceMenu(state, switcherRowClick(rows[1], false))
	requireMouseSelection(t, next, rows[1], effects)
	if next.LandingAttempt != 0 || next.CopyFeedback.Kind != CopyFeedbackNone || next.Notice.Text != "" {
		t.Fatalf("selection retained old navigation feedback: %+v", next)
	}
}
