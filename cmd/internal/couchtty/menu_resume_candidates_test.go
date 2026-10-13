package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"slices"
	"testing"
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
