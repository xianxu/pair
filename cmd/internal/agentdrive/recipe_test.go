package agentdrive

import "testing"

func TestDriveRequestClosedIntent(t *testing.T) {
	good := Request{Model: &ModelRequest{Model: "gpt-6.1-sol", Effort: "medium"}}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	if good.Model.EffectiveScope() != ScopeSession {
		t.Fatal("scope must default to session")
	}
	cases := []Request{{}, {Keys: KeySequence{{Kind: KeyEnter}}, Model: good.Model}, {Model: &ModelRequest{}}, {Model: &ModelRequest{Model: "a", Scope: "both"}}, {Model: &ModelRequest{Model: "a\x1b"}}, {Model: &ModelRequest{Model: "a", Effort: "m\n"}}}
	for _, r := range cases {
		if r.Validate() == nil {
			t.Fatalf("invalid request admitted: %+v", r)
		}
	}
	a := good.IntentDigest()
	equivalent := Request{Model: &ModelRequest{Model: "gpt-6.1-sol", Effort: "medium", Scope: ScopeSession}}
	if a != equivalent.IntentDigest() {
		t.Fatal("omitted scope differs from default")
	}
	equivalent.Model.Scope = ScopeDefault
	if a == equivalent.IntentDigest() {
		t.Fatal("scope lost from intent")
	}
	equivalent.Model.Scope = ScopeSession
	equivalent.Model.Effort = "high"
	if a == equivalent.IntentDigest() {
		t.Fatal("effort lost from intent")
	}
	raw := Request{Keys: KeySequence{{Kind: KeyText, Text: "first"}}}
	changed := Request{Keys: KeySequence{{Kind: KeyText, Text: "second"}}}
	if raw.IntentDigest() == changed.IntentDigest() {
		t.Fatal("key text lost from intent")
	}
}

func TestDriveResultTerminalKnowledge(t *testing.T) {
	for _, outcome := range []Outcome{OutcomePending, OutcomeConfirmed, OutcomeRefused, OutcomeFailed, OutcomeUnconfirmed} {
		r := Result{Outcome: outcome}
		if r.Terminal() != (outcome != OutcomePending) {
			t.Fatal(outcome)
		}
	}
}
