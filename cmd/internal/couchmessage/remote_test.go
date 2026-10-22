package couchmessage

import "testing"

func TestRemoteControlHasNoResetOpcodeOrFields(t *testing.T) {
	for _, op := range []string{"remote-on", "remote-off", "remote-status"} {
		if err := ValidateRequest(Request{Op: op}); err != nil {
			t.Fatal(err)
		}
		for _, request := range []Request{{Op: op, Body: "reset"}, {Op: op, Confirmed: true}, {Op: op, Scope: "scope"}, {Op: op, Target: "hub"}, {Op: op, SameBinary: true}, {Op: op, ForceUnknown: true}} {
			if ValidateRequest(request) == nil {
				t.Fatal("remote extra fields accepted", request)
			}
		}
	}
	if ValidateRequest(Request{Op: "remote-reset"}) == nil {
		t.Fatal("remote reset opcode")
	}
}
