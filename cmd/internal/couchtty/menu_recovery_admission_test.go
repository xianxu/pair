package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"strings"
	"testing"
)

func TestRecoveryPreviewDoesNotOfferUnexecutableFreshOrResume(t *testing.T) {
	row := menuSlotRow(1, "held-source")
	row.State, row.Reason = couchcore.ThreadUnusable, couchcore.ReasonUnknown
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, _ = openRecoveryPreview(state, row)
	preview := &couchcore.RecoveryPreview{Candidates: []couchcore.RecoveryChoice{{NativeID: "held-native", Label: "resume held", TailMatch: 92}}, Fresh: &couchcore.RecoveryChoice{Label: "Start fresh"}, Reasons: []string{"source owner is still running"}}
	state, _ = reduceRecoveryPreview(state, MenuEvent{Generation: state.CurrentFrame().Generation, RecoveryPreview: preview})
	if got := recoveryChoices(state.CurrentFrame().RecoveryPreview); len(got) != 0 {
		t.Fatalf("unexecutable actions offered: %+v", got)
	}
	shown := strings.Join(renderRecoveryMenu(state.CurrentFrame(), 160, 30), "\n")
	if !strings.Contains(shown, "held-native") || !strings.Contains(shown, "92%") {
		t.Fatalf("lost read-only evidence: %s", shown)
	}
	_, effects := reduceRecoveryKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 {
		t.Fatalf("evidence dispatched effects: %+v", effects)
	}
}

func TestStartingRecoveryDetailsNeverDispatchLifecycle(t *testing.T) {
	row := menuSlotRow(1, "starting")
	row.State = couchcore.ThreadBusy
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, _ = openRecoveryPreview(state, row)
	preview := &couchcore.RecoveryPreview{Candidates: []couchcore.RecoveryChoice{{Key: "candidate", NativeID: "native", Label: "resume"}}, Fresh: &couchcore.RecoveryChoice{Key: "fresh", Label: "Start fresh"}}
	state, _ = reduceRecoveryPreview(state, MenuEvent{Generation: state.CurrentFrame().Generation, RecoveryPreview: preview})
	if got := recoveryChoices(state.CurrentFrame().RecoveryPreview); len(got) != 0 {
		t.Fatalf("inflight start offered lifecycle: %+v", got)
	}
}
