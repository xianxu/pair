package couchmessage

import (
	"strings"
	"testing"
)

func TestSwitchArgumentsAndInterruptAreSwitchOnly(t *testing.T) {
	arguments, err := NewSwitchArguments(nil)
	if err != nil {
		t.Fatal(err)
	}
	// Retain valid existing shapes, including operations that return before
	// the bound-caller checks.
	for _, base := range []Request{
		{Op: "broadcast-status"},
		{Op: "reap-orphans", Confirmed: true},
		{Op: "actors", Scope: "s", Tag: "t", Session: "s", Nonce: "n"},
		{Op: "send", Scope: "s", Tag: "t", Session: "s", Nonce: "n", ID: "id", Target: "pair:2", Body: "hello"},
		{Op: "resume", Scope: "s", Tag: "t", Session: "s", Nonce: "n", ID: "id", Target: "pair:2"},
	} {
		if err := ValidateRequest(base); err != nil {
			t.Fatalf("invalid baseline %s: %v", base.Op, err)
		}
		withArguments, withInterrupt := base, base
		withArguments.AgentArgs = arguments
		withInterrupt.InterruptFirst = true
		for _, r := range []Request{withArguments, withInterrupt} {
			if err := ValidateRequest(r); err == nil {
				t.Fatalf("%s accepted switch-only fields: %#v", base.Op, r)
			}
		}
	}
}

func TestSwitchArgumentsValidateBothAgentsBeforeMutation(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, argv := range [][]string{nil, {"--model", "literal value"}, {"--", "resume", "$HOME"}} {
			r := leanSwitchRequest()
			r.Agent, r.InterruptFirst = agent, true
			var err error
			r.AgentArgs, err = NewSwitchArguments(argv)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateRequest(r); err != nil {
				t.Fatalf("%s rejected literal argv %q: %v", agent, argv, err)
			}
		}
		selector := []string{"--resume", "native-session"}
		if agent == "codex" {
			selector = []string{"resume", "native-session"}
		}
		r := leanSwitchRequest()
		r.Agent = agent
		r.AgentArgs, _ = NewSwitchArguments(selector)
		if err := ValidateRequest(r); err == nil {
			t.Fatalf("%s accepted native selector %q", agent, selector)
		}
	}
}

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
