package couchcore

import (
	"errors"
	"testing"
)

// The fake's quiesce must be observable by everything that reads session state,
// or an assertion that a session SURVIVED cannot fail (pair#230).
func TestFakeQuiesceMakesAThreadUnresumable(t *testing.T) {
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0102030405060708"}
	f := NewFakeThreadArtifactCollisionChecker()
	f.SetDetachedSession(address, "pair-repo-one")
	f.SetPairSession(address, "pair-repo-one", true)

	if err := f.Quiesce(address); err != nil {
		t.Fatal(err)
	}

	observed, err := f.DetachedSessions(t.Context(), []DetachedCandidate{{Address: address, Agent: "claude"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(observed) != 0 {
		t.Fatalf("DetachedSessions() = %+v after a quiesce, want none: the session was deleted", observed)
	}
	binding, err := f.PairSession(address)
	if err == nil && binding.Present {
		t.Fatalf("PairSession() = %+v after a quiesce, want absent", binding)
	}
}

// A refused quiesce did not take effect, so the session is still there. That is
// what the production retry loop is for.
func TestFakeQuiesceThatFailsLeavesTheSessionAlone(t *testing.T) {
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0102030405060708"}
	f := NewFakeThreadArtifactCollisionChecker()
	f.SetDetachedSession(address, "pair-repo-one")
	f.QuiesceHook = func(ThreadAddress) error { return errors.New("session server would not die") }

	if err := f.Quiesce(address); err == nil {
		t.Fatal("Quiesce() = nil, want the hook's refusal")
	}
	observed, err := f.DetachedSessions(t.Context(), []DetachedCandidate{{Address: address, Agent: "claude"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(observed) != 1 {
		t.Fatalf("DetachedSessions() = %+v after a REFUSED quiesce, want the session still observed", observed)
	}
}
