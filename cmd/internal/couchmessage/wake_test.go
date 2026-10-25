package couchmessage

import "testing"

// couch --wake (pair#830) carries the caller's identity and a request ID,
// nothing else: the targets are the supervisor's to choose.
func TestWakeRequestTakesOnlyAnID(t *testing.T) {
	base := Request{Op: "wake", ID: "wake-1", Scope: "s", Tag: "t", Session: "p", Nonce: "n"}
	if err := ValidateRequest(base); err != nil {
		t.Fatalf("wake refused: %v", err)
	}
	for name, mutate := range map[string]func(*Request){
		"no ID":        func(r *Request) { r.ID = "" },
		"a target":     func(r *Request) { r.Target = "bench:0" },
		"a body":       func(r *Request) { r.Body = "x" },
		"confirmation": func(r *Request) { r.Confirmed = true },
		"no caller":    func(r *Request) { r.Nonce = "" },
	} {
		r := base
		mutate(&r)
		if ValidateRequest(r) == nil {
			t.Errorf("wake with %s admitted", name)
		}
	}
}
