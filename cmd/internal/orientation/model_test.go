package orientation

import (
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"strings"
	"testing"
)

func TestBuildPromptNamesOnlyTheOutgoingSessionAndWaits(t *testing.T) {
	body, err := BuildPrompt(OrientationContext{
		Tag: "work", WorkingPath: "/repo", SourceAgent: "codex", SourceSession: "source-2", TargetAgent: "claude",
		PairLog: "/data/log-work.md", ScrollbackRaw: "/data/parked-source.raw", ScrollbackEvents: "/data/parked-source.events.jsonl",
		NativeTranscripts: []string{"/native/source-2.jsonl"}, Renderer: "/pair/bin/pair",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"source-2", "/data/parked-source.raw", "/native/source-2.jsonl", `"/pair/bin/pair","scrollback","render","--plain"`, "fresh conversation", "wait for the operator", "historical context"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in %s", want, body)
		}
	}
	if strings.Contains(body, "write a continuation") {
		t.Fatal("source must not prepare a continuation")
	}
}

func TestBuildPromptMissingContextStillOrients(t *testing.T) {
	body, err := BuildPrompt(OrientationContext{Tag: "work", WorkingPath: "/repo", SourceAgent: "muse", TargetAgent: "agy"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "unavailable") || !strings.Contains(body, "do not invent") {
		t.Fatal(body)
	}
}

func TestRequestRejectsTerminalControlAndIdentityMismatch(t *testing.T) {
	r := Request{SchemaVersion: 1, Tag: "work", Agent: "codex", Attempt: "attempt-1", Body: "read\nand summarize"}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"\x1b[201~submit", "\x00", strings.Repeat("x", MaxBodyBytes+1)} {
		bad := r
		bad.Body = body
		if bad.Validate() == nil {
			t.Errorf("accepted invalid body %q", body[:min(len(body), 20)])
		}
	}
	if r.Matches("work", "codex", "other") {
		t.Fatal("accepted another attempt")
	}
}

func TestDeliveryDoesNotSubmitOperatorInputDuringSettle(t *testing.T) {
	state, effect := AdvanceDelivery(DeliveryState{}, DeliveryEvent{Kind: ComposerObserved, ComposerReady: true})
	if effect != PastePrompt {
		t.Fatal(state, effect)
	}
	state, effect = AdvanceDelivery(state, DeliveryEvent{Kind: PasteCompleted, Written: 20, Expected: 20})
	if effect != StartSettle {
		t.Fatal(state, effect)
	}
	state, effect = AdvanceDelivery(state, DeliveryEvent{Kind: OperatorInput})
	if state.Phase != DeliveryCancelled || !state.BodyMayBePresent() || effect != PublishStatus {
		t.Fatal(state, effect)
	}
	_, effect = AdvanceDelivery(state, DeliveryEvent{Kind: SettleElapsed, ComposerReady: true})
	if effect != NoEffect {
		t.Fatal("cancelled delivery submitted", effect)
	}
}

func TestDeliveryPartialWriteIsNeverRetried(t *testing.T) {
	state, _ := AdvanceDelivery(DeliveryState{}, DeliveryEvent{Kind: ComposerObserved, ComposerReady: true})
	state, effect := AdvanceDelivery(state, DeliveryEvent{Kind: PasteCompleted, Written: 2, Expected: 20, Failed: true})
	if state.Phase != DeliveryIndeterminate || effect != PublishStatus {
		t.Fatal(state, effect)
	}
	for kind := ComposerObserved; kind <= ChildExited; kind++ {
		next, effect := AdvanceDelivery(state, DeliveryEvent{Kind: kind, ComposerReady: true, Written: 20, Expected: 20})
		if next != state || effect != NoEffect {
			t.Fatal("terminal delivery restarted", next, effect)
		}
	}
}

func TestDeliverySubmitsOnlyAfterCompletePasteAndSettle(t *testing.T) {
	state, _ := AdvanceDelivery(DeliveryState{}, DeliveryEvent{Kind: ComposerObserved, ComposerReady: true})
	state, _ = AdvanceDelivery(state, DeliveryEvent{Kind: PasteCompleted, Written: 20, Expected: 20})
	state, effect := AdvanceDelivery(state, DeliveryEvent{Kind: SettleElapsed, ComposerReady: true})
	if effect != SubmitPrompt {
		t.Fatal(state, effect)
	}
	state, effect = AdvanceDelivery(state, DeliveryEvent{Kind: SubmitCompleted, Written: 1, Expected: 1})
	if state.Phase != DeliverySubmitted || effect != PublishStatus {
		t.Fatal(state, effect)
	}
	_, effect = AdvanceDelivery(state, DeliveryEvent{Kind: ComposerObserved, ComposerReady: true})
	if effect != NoEffect {
		t.Fatal("duplicate submit")
	}
}

func FuzzDeliveryTerminalStatesNeverEmitInput(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5})
	f.Fuzz(func(t *testing.T, events []byte) {
		state := DeliveryState{}
		submits := 0
		for _, value := range events {
			terminal := state.Terminal()
			next, effect := AdvanceDelivery(state, DeliveryEvent{Kind: DeliveryEventKind(value % 11), ComposerReady: true, Written: 1, Expected: 1})
			if terminal && (next != state || effect != NoEffect) {
				t.Fatal("terminal state changed")
			}
			if effect == SubmitPrompt {
				submits++
			}
			if submits > 1 {
				t.Fatal("duplicate automatic submission")
			}
			state = next
		}
	})
}

func TestRendererReceivesExplicitSourceOwner(t *testing.T) {
	owner, err := artifactpath.NewStorageOwner("/data", "scope", "tag")
	if err != nil {
		t.Fatal(err)
	}
	body, err := BuildPrompt(OrientationContext{Tag: "tag", WorkingPath: "/repo", TargetAgent: "codex", Renderer: "/bin/pair", ScrollbackRaw: "/data/repos/scope/capture.raw", Owner: &owner})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{`"--owner-dir","/data/repos/scope"`, `"--owner-scope","scope"`, `"--owner-tag","tag"`} {
		if !strings.Contains(body, value) {
			t.Fatalf("owner missing: %s", body)
		}
	}
}
