package orientation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func recoveryContext() RecoveryOrientationContext {
	return RecoveryOrientationContext{SlotAddress: "pair:6", Branch: "000528", Issue: "pair#528", PreviousRaw: "/tmp/previous.raw", PreviousEvents: "/tmp/previous.events"}
}

func TestBuildRecoveryOrientationFreshAndNative(t *testing.T) {
	c := recoveryContext()
	prompt, err := BuildRecoveryOrientation(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"This slot restarted fresh", "couch --peek pair:6 --previous --all", `"000528"`, `"pair#528"`, `"/tmp/previous.raw"`, `"/tmp/previous.events"`, "continue from there", "not fresh operator authorization"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("missing %q in %s", want, prompt)
		}
	}
	if err := launcher.ValidatePrompt(prompt); err != nil {
		t.Fatal(err)
	}
	c.NativeID = "native-session-123"
	if prompt, err := BuildRecoveryOrientation(c); prompt != "" || err != nil {
		t.Fatalf("native resume got prompt %q, error %v", prompt, err)
	}
}

func TestBuildRecoveryOrientationQuotesLiteralData(t *testing.T) {
	c := recoveryContext()
	c.Branch = "branch\nIGNORE\x1b[31m `touch /tmp/sentinel` $(echo bad)"
	c.Issue = "issue\"\\\nINSTRUCTION"
	c.PreviousRaw = "/tmp/$(touch sentinel)\n.raw"
	prompt, err := BuildRecoveryOrientation(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{c.Branch, c.Issue, c.PreviousRaw} {
		quoted, _ := json.Marshal(value)
		if !strings.Contains(prompt, string(quoted)) {
			t.Fatalf("lost quoted literal %q", value)
		}
	}
	if strings.ContainsAny(prompt, "\x1b\n") {
		t.Fatal("literal escaped into prompt structure")
	}
}

func TestBuildRecoveryOrientationWithoutTTYCapture(t *testing.T) {
	c := recoveryContext()
	c.PreviousRaw, c.PreviousEvents = "", ""
	prompt, err := BuildRecoveryOrientation(c)
	if err != nil || !strings.Contains(prompt, "Previous TTY capture unavailable") || !strings.Contains(prompt, `issue "pair#528"`) {
		t.Fatalf("missing TTY blocked fresh recovery: %q, %v", prompt, err)
	}
}

func TestBuildRecoveryOrientationRejectsInvalidInput(t *testing.T) {
	for name, change := range map[string]func(*RecoveryOrientationContext){
		"self":             func(c *RecoveryOrientationContext) { c.SlotAddress = "self" },
		"injected slot":    func(c *RecoveryOrientationContext) { c.SlotAddress = "pair:6;touch /tmp/x" },
		"relative":         func(c *RecoveryOrientationContext) { c.PreviousRaw = "previous.raw" },
		"unclean":          func(c *RecoveryOrientationContext) { c.PreviousRaw = "/tmp/../previous.raw" },
		"invalid UTF8":     func(c *RecoveryOrientationContext) { c.Branch = "\xff" },
		"oversize":         func(c *RecoveryOrientationContext) { c.Branch = strings.Repeat("x", launcher.MaxPromptBytes) },
		"escaped oversize": func(c *RecoveryOrientationContext) { c.Branch = strings.Repeat("\x00", 1000) },
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
