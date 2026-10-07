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

// openRecoverFrame presses Tab, selects recover and presses Enter: the frame
// waits on a prepare-recover addressed by the row's ActorOperationArgs.
func openRecoverFrame(t *testing.T, row couchcore.ActionableThreadSummary) (MenuState, uint64) {
	t.Helper()
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state.InventoryReady = true
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state.Frames[len(state.Frames)-1].SelectedItem = "recover"
	state, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	frame := state.CurrentFrame()
	if len(effects) != 1 || effects[0].Preview == nil || effects[0].Operation != "" || frame.Kind != MenuFrameConfirmation || frame.Action != "recover" || frame.PreviewPending == 0 {
		t.Fatalf("recover did not ask for its preview: frame %+v, effects %+v", frame, effects)
	}
	if !maps.Equal(effects[0].Preview.RecoverArgs, couchcore.ActorOperationArgs(row, "recover")) {
		t.Fatalf("preview args %v", effects[0].Preview.RecoverArgs)
	}
	// Enter while the report is still being read does nothing.
	waiting := state
	waiting.Frames = append([]MenuFrame(nil), state.Frames...)
	waiting.Frames[len(waiting.Frames)-1].SelectedItem = "recover"
	if _, effects := reduceKey(waiting, PanelKey{Kind: KeyEnter}); len(effects) != 0 {
		t.Fatalf("Enter dispatched before the preview: %+v", effects)
	}
	return state, frame.PreviewPending
}

func TestRecoverWithoutConfirmationDispatchesAtOnce(t *testing.T) {
	row := couchcore.ActionableThreadSummary{Address: menuAddress("couch-primary"), WorkingPath: "/w/p", State: couchcore.ThreadParked}
	state, generation := openRecoverFrame(t, row)
	next, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: generation,
		RecoverPreview: &couchcore.RecoverPreview{Steps: []string{"resume"}, Text: "resume p"}})
	want := couchcore.ActorOperationArgs(row, "recover")
	want["steps"] = "resume"
	if len(effects) != 1 || effects[0].Operation != "recover" || !maps.Equal(effects[0].Args, want) {
		t.Fatalf("effects %+v, want recover %v", effects, want)
	}
	if next.CurrentFrame().Kind == MenuFrameConfirmation || next.InFlight.Operation != "recover" || !strings.Contains(next.Notice.Text, "recovering") {
		t.Fatalf("frame %+v, in flight %+v, notice %q", next.CurrentFrame(), next.InFlight, next.Notice.Text)
	}
}

func TestRecoverConfirmsADestructivePlanNamingIt(t *testing.T) {
	row := couchcore.ActionableThreadSummary{Address: menuAddress("couch-orphan"), WorkingPath: "/w/p",
		State: couchcore.ThreadUnusable, Reason: couchcore.ReasonOrphanedServer,
		Orphan: &launcher.SessionServerIdentity{PID: 812, Session: "📁1-37"}}
	state, generation := openRecoverFrame(t, row)
	preview := &couchcore.RecoverPreview{Steps: []string{"reap", "resume"}, Confirm: true, Text: "reap orphaned server PID 812 and everything under it, then resume p"}
	state, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: generation, RecoverPreview: preview})
	if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameConfirmation {
		t.Fatalf("a destructive plan dispatched unconfirmed: %+v", effects)
	}
	items := confirmationMenuItems(state, state.CurrentFrame())
	if len(items) != 2 || items[1] != "recover "+row.Label()+" — "+preview.Text {
		t.Fatalf("items %q", items)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	_, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
	want := couchcore.ActorOperationArgs(row, "recover")
	want["steps"], want["confirmed"] = "reap,resume", "true"
	if len(effects) != 1 || effects[0].Operation != "recover" || !maps.Equal(effects[0].Args, want) {
		t.Fatalf("effects %+v, want recover %v", effects, want)
	}
}

func TestRecoverHoldIsAnErrorNotice(t *testing.T) {
	row := couchcore.ActionableThreadSummary{Address: menuAddress("couch-primary"), WorkingPath: "/w/p", State: couchcore.ThreadParked}
	for name, event := range map[string]MenuEvent{
		"hold":  {RecoverPreview: &couchcore.RecoverPreview{Hold: "conflict:claim-elsewhere: claimed in pair:3"}},
		"error": {Error: "sdlc timed out"},
	} {
		state, generation := openRecoverFrame(t, row)
		event.Kind, event.Generation = MenuEventPreviewResult, generation
		next, effects := ReduceMenu(state, event)
		if len(effects) != 0 || next.CurrentFrame().Kind == MenuFrameConfirmation || next.Notice.Level != MenuNoticeError {
			t.Fatalf("%s: effects %+v, frame %+v, notice %+v", name, effects, next.CurrentFrame(), next.Notice)
		}
		if name == "hold" && !strings.Contains(next.Notice.Text, "claim-elsewhere") {
			t.Fatalf("hold notice %q", next.Notice.Text)
		}
	}
}
