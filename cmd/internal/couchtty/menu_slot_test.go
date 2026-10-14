package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"slices"
	"testing"
	"time"
)

func menuSlotRow(n int, tag couchcore.ThreadTag) couchcore.ActionableThreadSummary {
	slot := couchcore.SlotIdentity{Repo: "pair", RepoIdentity: "/src/pair/.git", PrimaryRoot: "/src/pair", EnvironmentRoot: "/src/worktree/pair-slot1", WorktreeRoot: "/src/worktree/pair-slot1/pair", Number: 1}
	if n == 2 {
		slot.Number = 2
		slot.EnvironmentRoot = "/src/worktree/pair-slot2"
		slot.WorktreeRoot = slot.EnvironmentRoot + "/pair"
	}
	target := couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: slot}
	key, _ := target.RowKey()
	address := couchcore.ThreadAddress{}
	if tag != "" {
		address = couchcore.ThreadAddress{RepoScope: "scope", Tag: tag}
	}
	return couchcore.ActionableThreadSummary{Target: target, RowKey: key, Address: address, WorkingPath: slot.WorktreeRoot, State: couchcore.ThreadUnusable, Reason: couchcore.ReasonUnreadable}
}
func TestMenuSelectsAddresslessSlotsIndependently(t *testing.T) {
	first, second := menuSlotRow(1, ""), menuSlotRow(2, "")
	state := NewMenuState([]couchcore.ActionableThreadSummary{first, second}, couchcore.ThreadAddress{})
	state, _ = reduceRootKey(state, PanelKey{Kind: KeyDown})
	row, ok := selectedMenuThread(state)
	if !ok || row.RowKey != second.RowKey {
		t.Fatalf("selection %+v", row)
	}
	state, effects := reduceRootKey(state, PanelKey{Kind: KeyEnter})
	state, effects = confirmSlotFresh(t, state, effects)
	if len(effects) != 1 || effects[0].Operation != "reboot" || effects[0].Args["path"] != second.WorkingPath {
		t.Fatalf("effects %+v", effects)
	}
}
func TestMenuSlotSelectionSurvivesConversationReplacement(t *testing.T) {
	state := NewMenuState([]couchcore.ActionableThreadSummary{menuSlotRow(1, "old"), menuSlotRow(2, "")}, couchcore.ThreadAddress{})
	state, _ = reduceRootKey(state, PanelKey{Kind: KeyDown})
	updated := []couchcore.ActionableThreadSummary{menuSlotRow(1, "old"), menuSlotRow(2, "new")}
	state, previous := replaceMenuInventory(state, updated)
	state = reconcileMenuFrames(state, previous)
	row, ok := selectedMenuThread(state)
	if !ok || row.Address.Tag != "new" {
		t.Fatalf("replacement selection %+v", row)
	}
}
func TestMenuSlotRebootConfirmsAndSendsThePath(t *testing.T) {
	row := menuSlotRow(1, "")
	items := menuActionItems(row)
	if !slices.Contains(items, "reboot") {
		t.Fatalf("actions %v", items)
	}
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, couchcore.ThreadAddress{})
	state, _ = reduceRootKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "reboot"
	state, effects := reduceActionKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameConfirmation {
		t.Fatalf("reboot bypassed confirmation: %+v", effects)
	}
	state.Frames[len(state.Frames)-1].SelectedItem = "reboot"
	_, effects = reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "reboot" || effects[0].Args["path"] != row.WorkingPath {
		t.Fatalf("reboot effects %+v", effects)
	}
}

func TestMenuSlotMouseAndFilterPreserveDistinctRows(t *testing.T) {
	first, second := menuSlotRow(1, ""), menuSlotRow(2, "")
	state := NewMenuState([]couchcore.ActionableThreadSummary{first, second}, couchcore.ThreadAddress{})
	state.Frames[0].Filter = ":2"
	rows := VisibleMenuThreads(state)
	if len(rows) != 1 || rows[0].RowKey != second.RowKey {
		t.Fatalf("filter: %+v", rows)
	}
	state.Frames[0].Filter = ""
	state, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventMouseSwitch, RowKey: second.RowKey})
	state, effects = confirmSlotFresh(t, state, effects)
	if len(effects) != 1 || effects[0].Args["path"] != second.WorkingPath {
		t.Fatalf("click effects: %+v", effects)
	}
}

func TestMenuSlotRenderedExtentsAddressBothMissingRecords(t *testing.T) {
	state := NewMenuState([]couchcore.ActionableThreadSummary{menuSlotRow(1, ""), menuSlotRow(2, "")}, couchcore.ThreadAddress{})
	_, extents := renderRootMenuFrame(state, state.CurrentFrame(), 100, 20, time.Time{}, false)
	if len(extents) != 2 || extents[0].RowKey == extents[1].RowKey {
		t.Fatalf("extents %+v", extents)
	}
	key, _, ok := (RenderedMenu{Extents: extents}).PointToRow(extents[1].Start, 0)
	if !ok || key != menuSlotRow(2, "").RowKey {
		t.Fatalf("hit %+v %v", key, ok)
	}
}
func TestMenuSlotOperationCompletionAcceptsNewConversation(t *testing.T) {
	row := menuSlotRow(1, "")
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, couchcore.ThreadAddress{})
	state, effects := reduceRootKey(state, PanelKey{Kind: KeyEnter})
	state, effects = confirmSlotFresh(t, state, effects)
	next := reduceOperationResult(state, MenuEvent{Operation: "reboot", Attempt: effects[0].Attempt, Success: true, Address: couchcore.ThreadAddress{RepoScope: "scope", Tag: "created"}})
	if next.InFlight.Operation != "" || !next.ProjectionPending {
		t.Fatalf("completion not consumed: %+v", next.InFlight)
	}
}

func confirmSlotFresh(t *testing.T, state MenuState, effects []MenuEffect) (MenuState, []MenuEffect) {
	t.Helper()
	if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameConfirmation || state.CurrentFrame().Action != "reboot" {
		t.Fatalf("fresh bypassed confirmation: %+v %+v", state.CurrentFrame(), effects)
	}
	state.Frames[len(state.Frames)-1].SelectedItem = "reboot"
	next, effects := reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "reboot" {
		t.Fatalf("fresh confirmation effects: %+v", effects)
	}
	return next, effects
}
