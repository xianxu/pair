package couchmessage

import "testing"

func TestRecoverParkProtocolUsesConfirmedSlotShape(t *testing.T) {
	base := Request{Op: "recover-park", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", ID: "recover-1", Target: "pair:2", Confirmed: true}
	if err := ValidateRequest(base); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Request){func(r *Request) { r.Target = "pair" }, func(r *Request) { r.Body = "quit" }, func(r *Request) { r.ForceUnknown = true }, func(r *Request) { r.Scope = "" }} {
		r := base
		mutate(&r)
		if ValidateRequest(r) == nil {
			t.Fatalf("accepted %+v", r)
		}
	}
}
