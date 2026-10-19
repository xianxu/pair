package couchtty

import (
	"context"
	"encoding/json"
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
	if state.CurrentFrame().Kind != MenuFrameRoot {
		t.Fatal("late confirmation offer stole current frame")
	}
	state, effects = reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 {
		t.Fatal("selecting confirmation row executed effects")
	}
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

func TestRecoveryProvenMenuDispatchesPlainResumeWithoutSavedScreen(t *testing.T) {
	row := menuSlotRow(1, "source")
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	state, _ = openRecoveryPreview(state, row)
	var preview couchcore.RecoveryPreview
	if err := json.Unmarshal([]byte(`{"proven":{"label":"Resume from disk (confirmed)","agent":"claude","native_id":"proven-native"},"incomplete":true,"fresh":{"label":"Start fresh"}}`), &preview); err != nil {
		t.Fatal(err)
	}
	state, _ = reduceRecoveryPreview(state, MenuEvent{Generation: state.CurrentFrame().Generation, RecoveryPreview: &preview})
	choices := recoveryChoices(state.CurrentFrame().RecoveryPreview)
	if len(choices) != 2 || choices[0].NativeID != "proven-native" {
		t.Fatalf("proven choice missing: %+v", choices)
	}
	_, effects := reduceRecoveryKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "resume" || effects[0].Args["candidate"] != "" || effects[0].Args["path"] != row.Target.Slot.WorktreeRoot {
		t.Fatalf("proven selection: %+v", effects)
	}
}

func TestRecoveryProvenRetainsFourExplicitAlternatives(t *testing.T) {
	preview := &couchcore.RecoveryPreview{Proven: &couchcore.RecoveryChoice{Label: "Resume from disk (confirmed)", NativeID: "proven"}, Candidates: []couchcore.RecoveryChoice{{Key: "one", Label: "inferred one"}, {Key: "two"}, {Key: "three"}, {Key: "four"}, {Key: "five"}, {Key: "six"}}, Fresh: &couchcore.RecoveryChoice{Label: "Start fresh"}}
	choices := recoveryChoices(preview)
	if len(choices) != 6 || choices[0].NativeID != "proven" || choices[1].Key != "one" || choices[4].Key != "four" || choices[5].Label != "Start fresh" {
		t.Fatalf("choice ordering/cap changed: %+v", choices)
	}
	shown := strings.Join(renderRecoveryMenu(MenuFrame{RecoveryPreview: preview}, 160, 60), "\n")
	if strings.Contains(shown, "best · Resume from disk (confirmed)") || !strings.Contains(shown, "best · inferred one") {
		t.Fatalf("best label shifted onto proven choice: %s", shown)
	}
}
