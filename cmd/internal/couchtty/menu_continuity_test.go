package couchtty

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestContinuityRowsAlwaysNameOnePrimaryAction(t *testing.T) {
	for _, tc := range []struct {
		state            couchcore.ActionableThreadState
		reason           couchcore.ThreadReason
		label, operation string
	}{
		{couchcore.ThreadLive, "", "Continue", "switch"},
		{couchcore.ThreadDetached, "", "Continue", "resume"},
		{couchcore.ThreadParked, "", "Resume from disk", "resume"},
		{couchcore.ThreadUnusable, couchcore.ReasonUnknown, "Relaunch", "relaunch"},
		{couchcore.ThreadUnusable, couchcore.ReasonNeedsRelaunch, "Relaunch", "relaunch"},
	} {
		row := menuSlotRow(1, "continuity")
		row.State, row.Reason = tc.state, tc.reason
		if got := threadStateText(row, time.Now()); !strings.Contains(got, tc.label) {
			t.Errorf("%s/%s status=%q want %q", tc.state, tc.reason, got, tc.label)
		}
		if got := enterOperationFor(row); got != tc.operation {
			t.Errorf("%s/%s primary=%q want %q", tc.state, tc.reason, got, tc.operation)
		}
	}
}

func TestFreshPrimaryRequiresExistingConfirmation(t *testing.T) {
	row := menuSlotRow(1, "uncertain")
	row.State, row.Reason = couchcore.ThreadUnusable, couchcore.ReasonBindingLost
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state.InventoryReady = true
	next, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyEnter}})
	if len(effects) != 0 || next.CurrentFrame().Kind != MenuFrameConfirmation || next.CurrentFrame().Action != "reboot" {
		t.Fatalf("fresh bypassed confirmation: frame=%+v effects=%+v", next.CurrentFrame(), effects)
	}
}

func TestFailedRecoveryPreviewStillOffersConfirmedFresh(t *testing.T) {
	row := menuSlotRow(1, "failed-preview")
	row.State, row.Reason = couchcore.ThreadUnusable, couchcore.ReasonBindingLost
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, _ = openRecoveryPreview(state, row)
	generation := state.CurrentFrame().Generation
	state, effects := reduceRecoveryPreview(state, MenuEvent{Generation: generation, Error: "scan deadline exceeded"})
	if len(effects) != 0 {
		t.Fatal("observation failure caused an effect")
	}
	choices := recoveryChoices(state.CurrentFrame().RecoveryPreview)
	if len(choices) != 1 || choices[0].NativeID != "" {
		t.Fatalf("fresh missing on failed scan: %+v", choices)
	}
	state, effects = reduceRecoveryKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || state.CurrentFrame().Kind != MenuFrameConfirmation || state.CurrentFrame().Action != "reboot" {
		t.Fatalf("fresh did not use confirmation: frame=%+v effects=%+v", state.CurrentFrame(), effects)
	}
}
