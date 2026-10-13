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
