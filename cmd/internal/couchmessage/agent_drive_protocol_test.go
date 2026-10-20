package couchmessage

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

func TestDriveProtocolClosedShape(t *testing.T) {
	r := Request{Op: "keys", Scope: "s", Tag: "t", Session: "s", Nonce: "n", ID: "drive-1", Target: "pair:2", Drive: &agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEnter}}}}
	if err := ValidateRequest(r); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Request){func(r *Request) { r.Body = "hidden" }, func(r *Request) { r.ToHub = "pair:1" }, func(r *Request) { r.Lines = 1 }, func(r *Request) { r.Confirmed = true }, func(r *Request) { r.Target = "pair" }, func(r *Request) { r.Op = "operation-status"; r.Target = "" }} {
		bad := r
		change(&bad)
		if err := ValidateRequest(bad); err == nil {
			t.Fatalf("accepted %#v", bad)
		}
	}
}

func TestInterruptProtocolNeedsItsOwnIntentAndConfirmation(t *testing.T) {
	r := Request{Op: "interrupt", Scope: "s", Tag: "t", Session: "s", Nonce: "n", ID: "drive-1", Target: "pair:2", Confirmed: true, Drive: &agentdrive.Request{Interrupt: true}}
	if err := ValidateRequest(r); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*Request){
		"no confirmation": func(r *Request) { r.Confirmed = false },
		"keys intent": func(r *Request) {
			r.Drive = &agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEsc}}}
		},
		"interrupt through keys":      func(r *Request) { r.Op = "keys"; r.Confirmed = false },
		"interrupt through set-model": func(r *Request) { r.Op = "set-model" },
	} {
		bad := r
		change(&bad)
		if err := ValidateRequest(bad); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
