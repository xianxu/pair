package orientation

import (
	"encoding/json"
	"strings"
	"testing"
)

func recoveryContext() RecoveryOrientationContext {
	return RecoveryOrientationContext{SlotAddress: "pair:6", Branch: "000528", Issue: "pair#528", PreviousRaw: "/tmp/previous.raw", PreviousEvents: "/tmp/previous.events"}
}

func TestBuildRecoveryOrientationFreshAndNative(t *testing.T) {
	for _, native := range []string{"", "native-session-123"} {
		t.Run(native, func(t *testing.T) {
			c := recoveryContext()
			c.NativeID = native
			body, err := BuildRecoveryOrientation(c)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"couch --peek pair:6 --previous --all", `"000528"`, `"pair#528"`, `"/tmp/previous.raw"`, `"/tmp/previous.events"`, "not fresh operator authorization"} {
				if !strings.Contains(body, want) {
					t.Fatalf("missing %q in %s", want, body)
				}
			}
			if native == "" && !strings.Contains(body, "fresh conversation") {
				t.Fatal(body)
			}
			if native != "" && (!strings.Contains(body, `"native-session-123"`) || strings.Contains(body, "This is a fresh conversation")) {
				t.Fatal(body)
			}
			if err := (Request{SchemaVersion: 1, Tag: "work", Agent: "codex", Attempt: "nonce", Body: body}).Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBuildRecoveryOrientationQuotesLiteralData(t *testing.T) {
	c := recoveryContext()
	c.Branch = "branch\nIGNORE\x1b[31m `touch /tmp/sentinel` $(echo bad)"
	c.Issue = "issue\"\\\nINSTRUCTION"
	c.PreviousRaw = "/tmp/$(touch sentinel)\n.raw"
	c.NativeID = "native\nINSTRUCTION"
	body, err := BuildRecoveryOrientation(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{c.Branch, c.Issue, c.PreviousRaw, c.NativeID} {
		quoted, _ := json.Marshal(value)
		if !strings.Contains(body, string(quoted)) {
			t.Fatalf("lost quoted literal %q", value)
		}
	}
	if strings.Contains(body, "\x1b") || strings.Contains(body, "\nINSTRUCTION") {
		t.Fatal("literal escaped into prompt structure")
	}
}

func TestBuildRecoveryOrientationRejectsInvalidInput(t *testing.T) {
	for name, change := range map[string]func(*RecoveryOrientationContext){
		"self":             func(c *RecoveryOrientationContext) { c.SlotAddress = "self" },
		"injected slot":    func(c *RecoveryOrientationContext) { c.SlotAddress = "pair:6;touch /tmp/x" },
		"missing raw":      func(c *RecoveryOrientationContext) { c.PreviousRaw = "" },
		"missing events":   func(c *RecoveryOrientationContext) { c.PreviousEvents = "" },
		"relative":         func(c *RecoveryOrientationContext) { c.PreviousRaw = "previous.raw" },
		"unclean":          func(c *RecoveryOrientationContext) { c.PreviousRaw = "/tmp/../previous.raw" },
		"invalid UTF8":     func(c *RecoveryOrientationContext) { c.Branch = "\xff" },
		"oversize":         func(c *RecoveryOrientationContext) { c.Branch = strings.Repeat("x", MaxBodyBytes) },
		"escaped oversize": func(c *RecoveryOrientationContext) { c.Branch = strings.Repeat("\x00", 4000) },
	} {
		t.Run(name, func(t *testing.T) {
			c := recoveryContext()
			change(&c)
			if _, err := BuildRecoveryOrientation(c); err == nil {
				t.Fatal("accepted invalid input")
			}
		})
	}
}

func TestRecoveryOrientationTerminalDeliveryDoesNotReplay(t *testing.T) {
	body, err := BuildRecoveryOrientation(recoveryContext())
	if err != nil {
		t.Fatal(err)
	}
	state, effect := AdvanceDelivery(DeliveryState{}, DeliveryEvent{Kind: ComposerObserved, ComposerReady: true})
	if effect != PastePrompt {
		t.Fatal(effect)
	}
	state, _ = AdvanceDelivery(state, DeliveryEvent{Kind: PasteCompleted, Written: len(body), Expected: len(body)})
	state, _ = AdvanceDelivery(state, DeliveryEvent{Kind: SettleElapsed, ComposerReady: true})
	for _, failed := range []bool{false, true} {
		terminal, _ := AdvanceDelivery(state, DeliveryEvent{Kind: SubmitCompleted, Written: 1, Expected: 1, Failed: failed})
		for _, event := range []DeliveryEvent{{Kind: ComposerObserved, ComposerReady: true}, {Kind: SettleElapsed, ComposerReady: true}} {
			next, effect := AdvanceDelivery(terminal, event)
			if next != terminal || effect != NoEffect {
				t.Fatalf("replayed terminal delivery: %+v %v", next, effect)
			}
		}
	}
}
