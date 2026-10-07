package couchtty

import (
	"maps"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// Recover is offered exactly where the admission table offers any actor
// operation, and it leads the list: it is the default gesture (#399).
func TestRecoverIsOfferedExactlyWhereActorActionsOffersAnything(t *testing.T) {
	for _, shape := range everyMenuRowShape(t) {
		items := menuActionItems(shape.row)
		actor := couchcore.ActorActions(couchcore.ActorRowFactsOf(shape.row))
		live := shape.row.State == couchcore.ThreadLive
		want := !live && len(actor) > 0
		has := len(items) > 0 && items[0] == "recover"
		if has != want {
			t.Errorf("%s: items %v, actor actions %v", shape.name, items, actor)
		}
		if strings.Count(strings.Join(items, " "), "recover") > 1 || (!want && containsMenuItem(items, "recover")) {
			t.Errorf("%s: recover misplaced in %v", shape.name, items)
		}
	}
}

// pressRecover presses Tab, selects recover and presses Enter.
func pressRecover(t *testing.T, row couchcore.ActionableThreadSummary) (MenuState, []MenuEffect) {
	t.Helper()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state.InventoryReady = true
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "recover"
	return reduceKey(state, PanelKey{Kind: KeyEnter})
}

// Recover never asks (operator, 2026-10-07): Enter dispatches it at once,
// addressed by the row's ActorOperationArgs, with no frame and no preview --
// on a plain parked row and on an orphan whose steps reap.
func TestRecoverDispatchesWithoutAConfirmation(t *testing.T) {
	for name, row := range map[string]couchcore.ActionableThreadSummary{
		"parked": {Address: menuAddress("couch-primary"), WorkingPath: "/w/p", State: couchcore.ThreadParked},
		"orphan": {Address: menuAddress("couch-orphan"), WorkingPath: "/w/p",
			State: couchcore.ThreadUnusable, Reason: couchcore.ReasonOrphanedServer,
			Orphan: &launcher.SessionServerIdentity{PID: 812, Session: "📁1-37"}},
		"slot": menuSlotRow(1, "couch-slot"),
	} {
		next, effects := pressRecover(t, row)
		if len(effects) != 1 || effects[0].Operation != "recover" || effects[0].Preview != nil ||
			!maps.Equal(effects[0].Args, couchcore.ActorOperationArgs(row, "recover")) {
			t.Fatalf("%s: effects %+v, want recover %v", name, effects, couchcore.ActorOperationArgs(row, "recover"))
		}
		if next.CurrentFrame().Kind == MenuFrameConfirmation || next.InFlight.Operation != "recover" || !strings.Contains(next.Notice.Text, "recovering") {
			t.Fatalf("%s: frame %+v, in flight %+v, notice %q", name, next.CurrentFrame(), next.InFlight, next.Notice.Text)
		}
		if row.Target.Kind == couchcore.ThreadTargetSlot && next.InFlight.RowKey != menuRowKey(row) {
			t.Fatalf("%s: a slot's recover is not keyed by row: %+v", name, next.InFlight)
		}
	}
}

// A held row's recover refuses in the owner; the switcher shows the refusal as
// an error notice and nothing else happens.
func TestRecoverRefusalIsAnErrorNotice(t *testing.T) {
	row := couchcore.ActionableThreadSummary{Address: menuAddress("couch-primary"), WorkingPath: "/w/p", State: couchcore.ThreadParked}
	state, effects := pressRecover(t, row)
	refusal := &couchcore.RecoverRefusal{Code: couchcore.RecoverHeld, Detail: "conflict:claim-elsewhere: claimed in pair:3"}
	next, more := ReduceMenu(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "recover", Attempt: effects[0].Attempt,
		Address: row.Address, Error: refusal.Error()})
	if len(more) != 0 || next.InFlight.Operation != "" || next.Notice.Level != MenuNoticeError || !strings.Contains(next.Notice.Text, "claim-elsewhere") {
		t.Fatalf("effects %+v, in flight %+v, notice %+v", more, next.InFlight, next.Notice)
	}
}
