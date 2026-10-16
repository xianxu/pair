package couchcore

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// The fake's quiesce must be observable by everything that reads session state,
// or an assertion that a session SURVIVED cannot fail (pair#230).
func TestFakeQuiesceMakesAThreadUnresumable(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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

// The fake answers through production's rule, not around it. A session two
// threads claim proves nothing -- and a fake that skipped that refusal would
// let every fake-backed test pass on a proof production rejects (ARCH-MOCK).
func TestFakeDetachedSessionsAppliesTheDuplicateNameRule(t *testing.T) {
	t.Parallel()

	a := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0000000000000001"}
	b := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0000000000000002"}
	f := NewFakeThreadArtifactCollisionChecker()
	f.SetDetachedSession(a, "pair-shared")
	f.SetDetachedSession(b, "pair-shared")

	observed, err := f.DetachedSessions(t.Context(), []DetachedCandidate{{Address: a, Agent: "claude"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(observed) != 0 {
		t.Fatalf("observed %+v; two threads claim that session, so it proves nothing", observed)
	}

	// And the uncontested case still resolves.
	f.SetDetachedSession(b, "")
	observed, err = f.DetachedSessions(t.Context(), []DetachedCandidate{{Address: a, Agent: "claude"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(observed) != 1 || observed[0].Address != a {
		t.Fatalf("observed %+v, want the one uncontested observation", observed)
	}
}

// Index ownership survives host liveness and client attachment. A claimant
// outside the requested candidates must still contest the detached evidence.
func TestFakeDetachedSessionsCountsAllIndexClaims(t *testing.T) {
	for _, present := range []bool{true, false} {
		t.Run(fmt.Sprintf("claimant-present-%t", present), func(t *testing.T) {
			a := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0000000000000001"}
			b := ThreadAddress{RepoScope: a.RepoScope, Tag: "couch-0000000000000002"}
			f := NewFakeThreadArtifactCollisionChecker()
			f.SetDetachedSession(a, "pair-shared")
			f.SetPairSession(b, "pair-shared", present)
			observed, err := f.DetachedSessions(t.Context(), []DetachedCandidate{{Address: a, Agent: "claude"}})
			if err != nil {
				t.Fatal(err)
			}
			if len(observed) != 0 {
				t.Fatalf("observed %+v despite another index owner", observed)
			}
		})
	}
}

func TestFakeAttachSessionPreservesTheSurvivingServer(t *testing.T) {
	t.Parallel()
	a := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0000000000000001"}
	b := ThreadAddress{RepoScope: a.RepoScope, Tag: "couch-0000000000000002"}
	unique := ThreadAddress{RepoScope: a.RepoScope, Tag: "couch-0000000000000003"}
	f := NewFakeThreadArtifactCollisionChecker()
	f.SetDetachedSession(a, "pair-shared")
	f.SetDetachedSession(b, "pair-shared")
	f.SetDetachedSession(unique, "pair-unique")
	f.SetPaneSidecar(b, "claude")
	beforeBinding, _ := f.PairSession(b)
	beforePanes, _ := f.PaneSidecars(b)
	for range 2 {
		f.AttachSession(b)
	}
	binding, err := f.PairSession(b)
	if err != nil || binding != beforeBinding {
		t.Fatalf("attachment changed binding: %+v, %v; before %+v", binding, err, beforeBinding)
	}
	presence, err := f.SessionPresence(t.Context(), []ThreadAddress{b})
	if err != nil || presence[b].State != SessionPresent {
		t.Fatalf("attached server presence = %+v, %v", presence, err)
	}
	panes, err := f.PaneSidecars(b)
	if err != nil || !reflect.DeepEqual(panes, beforePanes) {
		t.Fatalf("attachment changed panes: %+v, %v; before %+v", panes, err, beforePanes)
	}
	// b stops being detached, still contests a, and leaves the unique session alone.
	for _, address := range []ThreadAddress{a, b, unique} {
		observed, err := f.DetachedSessions(t.Context(), []DetachedCandidate{{Address: address, Agent: "claude"}})
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		if address == unique {
			want = 1
		}
		if len(observed) != want {
			t.Fatalf("detached observation for %+v = %+v; want %d", address, observed, want)
		}
	}
}

func TestFakeAttachSessionDoesNotCreateAServer(t *testing.T) {
	t.Parallel()
	f := NewFakeThreadArtifactCollisionChecker()
	a := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0000000000000001"}
	f.AttachSession(a)
	binding, _ := f.PairSession(a)
	presence, err := f.SessionPresence(t.Context(), []ThreadAddress{a})
	panes, _ := f.PaneSidecars(a)
	if err != nil || binding.Name != "" || binding.Present || presence[a].State != SessionAbsent || len(panes) != 0 {
		t.Fatalf("attachment invented server evidence: binding=%+v presence=%+v panes=%+v err=%v", binding, presence, panes, err)
	}
}
