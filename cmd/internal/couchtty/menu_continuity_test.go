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

func TestFreshPrimaryRequestsSharedPreparationBeforeStopConfirmation(t *testing.T) {
	row := menuSlotRow(1, "uncertain")
	row.State, row.Reason = couchcore.ThreadUnusable, couchcore.ReasonBindingLost
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state.InventoryReady = true
	next, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyEnter}})
	if len(effects) != 1 || effects[0].Operation != "reboot" || next.CurrentFrame().Kind == MenuFrameConfirmation {
		t.Fatalf("fresh did not request shared preparation: frame=%+v effects=%+v", next.CurrentFrame(), effects)
	}
}

func TestFailedRecoveryPreviewFreshConfirmsOnlyExactStopOffer(t *testing.T) {
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
	if len(effects) != 1 || effects[0].Operation != "reboot" || state.CurrentFrame().Kind == MenuFrameConfirmation {
		t.Fatalf("fresh did not request preparation: frame=%+v effects=%+v", state.CurrentFrame(), effects)
	}
	offer := &couchcore.RecoveryConfirmationOffer{Token: "fresh-exact-stops", Stops: []couchcore.RecoveryStopProcess{{PID: 4321, Identity: "source-generation", Role: "terminal server"}}}
	state = reduceOperationResult(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "reboot", Attempt: effects[0].Attempt, Address: row.Address, RecoveryConfirmation: offer})
	frame := state.CurrentFrame()
	if frame.Kind != MenuFrameConfirmation || frame.RecoveryConfirmation == nil || frame.SelectedItem != "cancel" || frame.RecoveryConfirmation.Stops[0] != offer.Stops[0] {
		t.Fatalf("exact stop offer was lost: %+v", frame)
	}
	_, effects = reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 {
		t.Fatal("cancel dispatched effects")
	}
	state, _ = reduceConfirmationKey(state, PanelKey{Kind: KeyDown})
	_, effects = reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "recovery-confirmation" || effects[0].Args["token"] != offer.Token {
		t.Fatalf("accept lost exact capability: %+v", effects)
	}
}
