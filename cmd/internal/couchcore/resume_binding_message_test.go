package couchcore

import (
	"context"
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

// The message function must reach the path an OPERATOR travels. It shipped with
// one consumer while the real resolver and its stateful fake both hand-wrote the
// catch-all it replaced, so every actual refusal still read "native session
// binding is not one exact established root".
func TestTheResolverAndItsFakeRefuseWithTheActionableSentence(t *testing.T) {
	fake := NewFakeThreadArtifactCollisionChecker()
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	fake.SetNativeBinding(address, "claude", sessioninventory.BindingProvisional, "")

	_, err := fake.ResolveEstablished(context.Background(), address.RepoScope, string(address.Tag), "claude")
	if err == nil {
		t.Fatal("a provisional binding resolved without refusing")
	}
	if strings.Contains(err.Error(), "one exact established root") {
		t.Errorf("the fake still hand-writes the catch-all: %v", err)
	}
	if !strings.Contains(err.Error(), "has not completed a turn yet") {
		t.Errorf("refusal does not name the cause an operator can act on: %v", err)
	}
	if got := ResumeDiagnosticOf(err); got != ResumeBindingProvisional {
		t.Errorf("diagnostic = %q, want %q", got, ResumeBindingProvisional)
	}
}
