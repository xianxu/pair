package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"strings"
	"testing"
)

func TestRecoveryPreviewRetainsFreshWithIncompleteOwnerEvidence(t *testing.T) {
	row := menuSlotRow(1, "held-source")
	row.State, row.Reason = couchcore.ThreadUnusable, couchcore.ReasonUnknown
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, _ = openRecoveryPreview(state, row)
	preview := &couchcore.RecoveryPreview{Candidates: []couchcore.RecoveryChoice{{NativeID: "held-native", Label: "resume held", TailMatch: 92}}, Fresh: &couchcore.RecoveryChoice{Label: "Start fresh"}, Reasons: []string{"source owner is still running"}}
	state, _ = reduceRecoveryPreview(state, MenuEvent{Generation: state.CurrentFrame().Generation, RecoveryPreview: preview})
	if got := recoveryChoices(state.CurrentFrame().RecoveryPreview); len(got) != 1 {
		t.Fatalf("fresh choice missing: %+v", got)
	}
	shown := strings.Join(renderRecoveryMenu(state.CurrentFrame(), 160, 30), "\n")
	if !strings.Contains(shown, "held-native") || !strings.Contains(shown, "92%") {
		t.Fatalf("lost read-only evidence: %s", shown)
	}
	_, effects := reduceRecoveryKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "reboot" {
		t.Fatalf("fresh did not dispatch shared recovery: %+v", effects)
	}
}

func TestStartingRecoveryLeavesAdmissionToSharedCore(t *testing.T) {
	row := menuSlotRow(1, "starting")
	row.State = couchcore.ThreadBusy
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, _ = openRecoveryPreview(state, row)
	preview := &couchcore.RecoveryPreview{Candidates: []couchcore.RecoveryChoice{{Key: "candidate", NativeID: "native", Label: "resume"}}, Fresh: &couchcore.RecoveryChoice{Key: "fresh", Label: "Start fresh"}}
	state, _ = reduceRecoveryPreview(state, MenuEvent{Generation: state.CurrentFrame().Generation, RecoveryPreview: preview})
	if got := recoveryChoices(state.CurrentFrame().RecoveryPreview); len(got) != 2 {
		t.Fatalf("explicit recovery choices missing: %+v", got)
	}
}
