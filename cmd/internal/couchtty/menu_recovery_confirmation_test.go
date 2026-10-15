package couchtty

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"strings"
	"testing"
)

func TestRecoveryStopConfirmationCancelAndAccept(t *testing.T) {
	row := menuSlotRow(1, "source")
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, effects := dispatchMenuOperation(state, MenuEffect{Operation: "resume", Args: couchcore.ActorOperationArgs(row, "resume")}, row.Address)
	offer := &couchcore.RecoveryConfirmationOffer{Token: "exact-token", Stops: []couchcore.RecoveryStopProcess{{PID: 123, Identity: "start-generation", Role: "helper"}}}
	state = reduceOperationResult(state, MenuEvent{Kind: MenuEventOperationResult, Operation: "resume", Attempt: effects[0].Attempt, Address: row.Address, RecoveryConfirmation: offer})
	frame := state.CurrentFrame()
	if frame.Kind != MenuFrameConfirmation || frame.RecoveryConfirmation == nil || frame.SelectedItem != "cancel" {
		t.Fatalf("missing confirmation %+v", frame)
	}
	label := strings.Join(confirmationMenuItems(state, frame), " ")
	if !strings.Contains(label, "123") || !strings.Contains(label, "start-generation") {
		t.Fatalf("stop identity missing: %s", label)
	}
	_, effects = reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 {
		t.Fatal("cancel effects")
	}
	state, _ = reduceConfirmationKey(state, PanelKey{Kind: KeyDown})
	_, effects = reduceConfirmationKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "recovery-confirmation" || effects[0].Args["token"] != "exact-token" {
		t.Fatalf("accept lost token: %+v", effects)
	}
}

func TestRecoveryConfirmationPreparedContextSurvivesQueue(t *testing.T) {
	r := newRemoteRig(t)
	type actorKey struct{}
	r.resume = func(call couchcore.OperationCall) (any, error) {
		if call.Context.Value(actorKey{}) != "authenticated-generation" {
			t.Fatal("queue replaced authenticated actor context")
		}
		return nil, nil
	}
	err := r.enqueue("recovery", "recovery-confirmation", func(ctx context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "recovery-confirmation", Args: map[string]string{"token": "opaque"}, Context: context.WithValue(ctx, actorKey{}, "authenticated-generation")}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	r.drain(t)
}
