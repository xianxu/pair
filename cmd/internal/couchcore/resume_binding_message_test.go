package couchcore

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// Every binding status refused with one developer's sentence -- "native session
// binding is not one exact established root" -- which an operator cannot act on.
// The commonest of them is not a fault at all: provisional means the agent has
// not answered yet, and relaunch is something you do to a session you just
// started, so that is the refusal it meets most.
func TestEachBindingRefusalSaysSomethingDifferentAndUsable(t *testing.T) {
	seen := map[string]sessioninventory.BindingStatus{}
	for _, status := range []sessioninventory.BindingStatus{
		sessioninventory.BindingProvisional,
		sessioninventory.BindingAmbiguous,
		sessioninventory.BindingUnbound,
	} {
		code := bindingResumeDiagnostic(NativeBindingResolution{Status: status})
		if code == "" {
			t.Fatalf("status %v produced no refusal code", status)
		}
		message := bindingRefusalDiagnostic(code)
		if strings.Contains(message, "one exact established root") {
			t.Errorf("status %v still uses the catch-all wording: %q", status, message)
		}
		if previous, duplicate := seen[message]; duplicate {
			t.Errorf("status %v repeats the message used by %v: %q", status, previous, message)
		}
		seen[message] = status
	}
	// The one an operator hits after starting a session must say what to do.
	provisional := bindingRefusalDiagnostic(ResumeBindingProvisional)
	if !strings.Contains(provisional, "retry") {
		t.Errorf("provisional refusal does not tell the operator what to do: %q", provisional)
	}
}
