package couchmessage

import (
	"strings"
	"testing"
)

func leanSwitchRequest() Request {
	return Request{Op: "switch-agent", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", ID: "id", Target: "pair:2", Agent: "codex", Confirmed: true}
}
func TestSwitchWireShapes(t *testing.T) {
	if err := ValidateRequest(leanSwitchRequest()); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"self", "family", "body", "agent", "confirmation", "override"} {
		t.Run(name, func(t *testing.T) {
			r := leanSwitchRequest()
			switch name {
			case "self":
				r.Target = "self"
			case "family":
				r.Target = "pair"
			case "body":
				r.Body = "brief"
			case "agent":
				r.Agent = "bogus"
			case "confirmation":
				r.Confirmed = false
			case "override":
				r.ForceUnknown = true
			}
			if ValidateRequest(r) == nil {
				t.Fatalf("accepted %#v", r)
			}
		})
	}
}
func TestSwitchReceiptIdentityIncludesAgent(t *testing.T) {
	r, err := ApplyReceiptEvent(OperationReceipt{}, ReceiptEvent{Kind: ReceiptAdmit, ID: "one", Op: "switch-agent", Target: "pair:2", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyReceiptEvent(r, ReceiptEvent{Kind: ReceiptAdmit, ID: "one", Op: "switch-agent", Target: "pair:2", Agent: "codex"}); err == nil {
		t.Fatal("changed target agent accepted")
	}
	if _, err = ApplyReceiptEvent(r, ReceiptEvent{Kind: ReceiptAdmit, ID: "one", Op: "switch-agent", Target: "pair:2", Agent: "claude"}); err != nil {
		t.Fatal(err)
	}
}
func FuzzSwitchClosedShape(f *testing.F) {
	f.Add("self", "body")
	f.Fuzz(func(t *testing.T, target, body string) {
		r := leanSwitchRequest()
		r.Target = target
		r.Body = body
		if ValidateRequest(r) == nil && (body != "" || !strings.Contains(target, ":")) {
			t.Fatalf("accepted %#v", r)
		}
	})
}
