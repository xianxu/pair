package couchcore

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestDetachedSessionsWithoutPolicyOrReadiness(t *testing.T) {
	data := t.TempDir()
	const name = "📁repo-a"
	indexSession(t, data, addressA, name)
	checker, _ := sandboxedChecker(t, data, map[string]string{name: "detached"})
	checker.OwnerProbe = nil
	got, err := checker.DetachedSessions(t.Context(), []DetachedCandidate{{Address: addressA, Agent: "claude", SessionName: name}})
	if err != nil || len(got) != 1 {
		t.Fatalf("listed owned session must continue without metadata: %v %v", got, err)
	}
}

type unavailableServerSnapshot struct{}

func (unavailableServerSnapshot) ServerStates(context.Context) (map[string]launcher.ServerState, error) {
	return nil, errors.New("process table unavailable")
}
func TestListedSessionSurvivesMissingProcessEvidence(t *testing.T) {
	data := t.TempDir()
	indexSession(t, data, addressA, "📁listed")
	indexSession(t, data, addressB, "📁missing")
	checker, _ := sandboxedChecker(t, data, map[string]string{"📁listed": "detached"})
	checker.Servers = unavailableServerSnapshot{}
	got, err := checker.SessionPresence(t.Context(), []ThreadAddress{addressA, addressB})
	if err != nil {
		t.Fatal(err)
	}
	if got[addressA].State != SessionPresent || got[addressB].State != SessionUnresolved {
		t.Fatalf("presence=%v", got)
	}
}
