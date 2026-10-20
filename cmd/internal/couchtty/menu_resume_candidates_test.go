package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
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
	for _, route := range []string{"candidates-item", "enter", "relaunch-item"} {
		t.Run(route, func(t *testing.T) {
			row := menuSlotRow(1, "couch-unsafe")
			row.State, row.Reason = couchcore.ThreadUnusable, couchcore.ReasonNeedsRelaunch
			state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
			state.InventoryReady = true
			var effects []MenuEffect
			switch route {
			case "enter":
				state, effects = reduceRootKey(state, PanelKey{Kind: KeyEnter})
			case "relaunch-item":
				state, _ = reduceRootKey(state, PanelKey{Kind: KeyTab})
				state.Frames[len(state.Frames)-1].SelectedItem = "relaunch"
				state, effects = reduceActionKey(state, PanelKey{Kind: KeyEnter})
			case "candidates-item":
				state, _ = reduceRootKey(state, PanelKey{Kind: KeyTab})
				state.Frames[len(state.Frames)-1].SelectedItem = "resume-candidates"
				state, effects = reduceActionKey(state, PanelKey{Kind: KeyEnter})

			}
			frame := state.CurrentFrame()
			if frame.Kind != MenuFrameRecovery || len(effects) != 1 || effects[0].Preview == nil || effects[0].Operation != "" || state.operationForTest() != (MenuOperationOrigin{}) {
				t.Fatalf("primary bypassed read-only preview: frame=%+v effects=%+v inflight=%+v", frame, effects, state.operationForTest())
			}
			if target := effects[0].Preview.RecoveryTarget; target == nil || target.Path != row.Target.Slot.WorktreeRoot || target.Address != (couchcore.ThreadAddress{}) {
				t.Fatalf("lost slot target: %+v", target)
			}
			preview := &couchcore.RecoveryPreview{Address: row.Address, Candidates: []couchcore.RecoveryChoice{{Key: "selected-key", Label: "resume (best candidate)"}}}
			state, effects = ReduceMenu(state, MenuEvent{Kind: MenuEventPreviewResult, Generation: frame.Generation, RecoveryPreview: preview})
			if len(effects) != 0 || state.operationForTest().Operation != "" {
				t.Fatal("preview completion launched automatically")
			}
			state, effects = reduceRecoveryKey(state, PanelKey{Kind: KeyEnter})
			if len(effects) != 1 || effects[0].Operation != "resume" || effects[0].Args["candidate"] != "selected-key" || effects[0].Args["path"] != row.Target.Slot.WorktreeRoot || state.operationForTest().RowKey != menuRowKey(row) {
				t.Fatalf("explicit choice lost queue/target: %+v %+v", state.operationForTest(), effects)
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

func TestRecoveryMenuNativeOnlyUnconfirmedEvidence(t *testing.T) {
	activity := time.Date(2026, 10, 19, 10, 12, 0, 0, time.UTC)
	for _, key := range []string{"v2.exact", ""} {
		t.Run(key, func(t *testing.T) {
			frame := MenuFrame{RecoveryPreview: &couchcore.RecoveryPreview{Candidates: []couchcore.RecoveryChoice{{SourceKind: sessioninventory.RecoverySourceNative, Key: key, Label: "Resume native transcript", Agent: "codex", NativeID: "native-id", CWD: "/physical/repo\x1b]0;hostile\x07", LastActivity: activity}}, Reasons: []string{"Screen unavailable: capture missing"}, Fresh: &couchcore.RecoveryChoice{Label: "Start fresh"}}}
			rendered := strings.Join(renderRecoveryMenu(frame, 240, 30), "\n")
			for _, want := range []string{"unconfirmed", "physical cwd /physical/repo", "native activity " + activity.Local().Format("2006-01-02 15:04 MST"), "native-id", "Screen unavailable: capture missing"} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("missing %q: %s", want, rendered)
				}
			}
			for _, unwanted := range []string{"best ·", "tail ", "matched ", "0%", "\x1b", "\x07"} {
				if strings.Contains(rendered, unwanted) {
					t.Fatalf("misleading/unsafe %q: %s", unwanted, rendered)
				}
			}
		})
	}
}

func TestRecoveryMenuRetainsPartialScreenActivity(t *testing.T) {
	activity := time.Date(2026, 10, 19, 11, 12, 0, 0, time.UTC)
	frame := MenuFrame{RecoveryPreview: &couchcore.RecoveryPreview{Screen: couchcore.ScreenObservation{Availability: couchcore.ScreenUnavailable, Reason: "Capture missing", LastActivity: activity, ActivitySource: "capture-file activity\x1b]0;unsafe\x07"}, Reasons: []string{"Capture missing"}}}
	text := strings.Join(renderRecoveryMenu(frame, 240, 30), "\n")
	if !strings.Contains(text, "capture-file activity") || !strings.Contains(text, activity.Local().Format("2006-01-02 15:04 MST")) || strings.Count(text, "Capture missing") != 1 || strings.ContainsAny(text, "\x1b\x07") {
		t.Fatalf("partial screen fact lost or unsafe: %s", text)
	}
}
