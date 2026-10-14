package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestRecoveryPreviewRequiresExplicitChoiceAndRejectsStaleResult(t *testing.T) {
	address := menuAddress("thread")
	row := couchcore.ActionableThreadSummary{Address: address, State: couchcore.ThreadUnusable, Reason: couchcore.ReasonUnknown}
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, couchcore.ThreadAddress{})
	state, effects := openRecoveryPreview(state, row)
	if len(effects) != 1 || effects[0].Preview == nil || effects[0].Operation != "" {
		t.Fatalf("preview caused mutation: %+v", effects)
	}
	generation := state.CurrentFrame().Generation
	preview := &couchcore.RecoveryPreview{Address: address, Candidates: []couchcore.RecoveryChoice{{Key: "opaque-original", Label: "candidate"}}, Fresh: &couchcore.RecoveryChoice{Key: "opaque-fresh", Label: "start fresh from previous screen"}}
	stale, _ := reduceRecoveryPreview(state, MenuEvent{Generation: generation + 1, RecoveryPreview: preview})
	if stale.CurrentFrame().RecoveryPreview != nil {
		t.Fatal("stale preview accepted")
	}
	state, effects = reduceRecoveryPreview(state, MenuEvent{Generation: generation, RecoveryPreview: preview})
	if len(effects) != 0 {
		t.Fatal("preview automatically resumed")
	}
	state, _ = reduceRecoveryKey(state, PanelKey{Kind: KeyDown})
	_, effects = reduceRecoveryKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "resume" || effects[0].Args["candidate"] != "opaque-fresh" {
		t.Fatalf("selection lost exact fresh key: %+v", effects)
	}
}

func TestRecoveryAlternativesRemainExplicitlyAvailableAfterResume(t *testing.T) {
	for _, state := range []couchcore.ActionableThreadState{couchcore.ThreadLive, couchcore.ThreadUnusable, couchcore.ThreadParked} {
		row := couchcore.ActionableThreadSummary{Address: menuAddress("thread"), State: state}
		if !slices.Contains(menuActionsFor(NewMenuState(nil, couchcore.ThreadAddress{}), row), "resume-candidates") {
			t.Fatalf("missing alternatives in state %v", state)
		}
	}
}

func TestRecoveryMenuLegacyFreshShowsUnknownActivityWithoutNativeScore(t *testing.T) {
	frame := MenuFrame{RecoveryPreview: &couchcore.RecoveryPreview{Fresh: &couchcore.RecoveryChoice{Key: "fresh", Label: "start fresh from previous screen", Agent: "codex"}}}
	rendered := strings.Join(renderRecoveryMenu(frame, 120, 30), "\n")
	if !strings.Contains(rendered, "unknown") || strings.Contains(rendered, "0001") || strings.Contains(rendered, "0%") || strings.Contains(rendered, "matched") {
		t.Fatalf("invented fresh evidence: %s", rendered)
	}
}

func TestRecoveryMenuActivityNamesLocalTimezone(t *testing.T) {
	activity := time.Date(2026, 10, 13, 14, 54, 0, 0, time.UTC)
	frame := MenuFrame{RecoveryPreview: &couchcore.RecoveryPreview{Candidates: []couchcore.RecoveryChoice{{Key: "native", NativeID: "id", LastActivity: activity, TailMatch: 75, Matched: 3, Total: 4}}}}
	rendered := strings.Join(renderRecoveryMenu(frame, 160, 30), "\n")
	if !strings.Contains(rendered, activity.Local().Format("2006-01-02 15:04 MST")) || !strings.Contains(rendered, "75%") {
		t.Fatalf("activity lost zone or score: %s", rendered)
	}
}

func TestRecoveryAlternativesOpenReadOnlyPreview(t *testing.T) {
	for _, route := range []string{"candidates-item", "restart-hotkey"} {
		t.Run(route, func(t *testing.T) {
			row := menuSlotRow(1, "couch-unsafe")
			row.State, row.Reason = couchcore.ThreadUnusable, couchcore.ReasonNeedsRelaunch
			state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
			state.InventoryReady = true
			var effects []MenuEffect
			switch route {
			case "enter":
				state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyEnter}})
			case "mouse":
				state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventMouseSwitch, Address: row.Address, RowKey: menuRowKey(row)})
			case "candidates-item":
				state, _ = reduceRootKey(state, PanelKey{Kind: KeyTab})
				state.Frames[len(state.Frames)-1].SelectedItem = "resume-candidates"
				state, effects = reduceActionKey(state, PanelKey{Kind: KeyEnter})
			case "restart-hotkey":
				state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventParkHotkey, Operation: "relaunch", Address: row.Address})
			}
			frame := state.CurrentFrame()
			if frame.Kind != MenuFrameRecovery || len(effects) != 1 || effects[0].Preview == nil || effects[0].Operation != "" || state.InFlight != (MenuOperationOrigin{}) {
				t.Fatalf("primary bypassed read-only preview: frame=%+v effects=%+v inflight=%+v", frame, effects, state.InFlight)
			}
			if target := effects[0].Preview.RecoveryTarget; target == nil || target.Path != row.Target.Slot.WorktreeRoot || target.Address != (couchcore.ThreadAddress{}) {
				t.Fatalf("lost slot target: %+v", target)
			}
			preview := &couchcore.RecoveryPreview{Address: row.Address, Candidates: []couchcore.RecoveryChoice{{Key: "selected-key", Label: "resume (best candidate)"}}}
			state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: frame.Generation, RecoveryPreview: preview})
			if len(effects) != 0 || state.InFlight.Operation != "" {
				t.Fatal("preview completion launched automatically")
			}
			state, effects = reduceRecoveryKey(state, PanelKey{Kind: KeyEnter})
			if len(effects) != 1 || effects[0].Operation != "resume" || effects[0].Args["candidate"] != "selected-key" || effects[0].Args["path"] != row.Target.Slot.WorktreeRoot || state.InFlight.RowKey != menuRowKey(row) {
				t.Fatalf("explicit choice lost queue/target: %+v %+v", state.InFlight, effects)
			}
		})
	}
}

func TestHealthyRelaunchKeepsExistingConfirmation(t *testing.T) {
	row := menuSlotRow(1, "couch-healthy")
	row.State = couchcore.ThreadLive
	for _, hotkey := range []bool{false, true} {
		state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
		var effects []MenuEffect
		if hotkey {
			state, effects = reduceParkHotkey(state, MenuEvent{Operation: "relaunch", Address: row.Address})
		} else {
			state, _ = reduceRootKey(state, PanelKey{Kind: KeyTab})
			state.Frames[len(state.Frames)-1].SelectedItem = "relaunch"
			state, effects = reduceActionKey(state, PanelKey{Kind: KeyEnter})
		}
		if state.CurrentFrame().Kind != MenuFrameConfirmation || state.CurrentFrame().Action != "relaunch" || len(effects) != 0 {
			t.Fatalf("healthy relaunch changed: %+v %+v", state.CurrentFrame(), effects)
		}
	}
}
