package couchcore

import (
	"errors"
	"testing"
)

func rec(id string, w Worktree, same bool) ActorRecord {
	return ActorRecord{ID: ActorID(id), Args: StartArgs{Worktree: w, SameTree: same}}
}

func TestRegisterRefusesSecondActorAndNamesIncumbent(t *testing.T) {
	reg, err := NewRegistry().Register(rec("couch-a", "/repo", false))
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err = reg.Register(rec("couch-b", "/repo", false))
	var occ *TreeOccupiedError
	if !errors.As(err, &occ) {
		t.Fatalf("err = %v, want *TreeOccupiedError", err)
	}
	if len(occ.Incumbents) != 1 || occ.Incumbents[0].ID != "couch-a" {
		t.Fatalf("incumbents = %+v; the caller renders worktree-or-switch from these", occ.Incumbents)
	}
}

func TestRegisterFoldsCaseAccordingToPlatform(t *testing.T) {
	// The milestone's central invariant. Both directions are asserted rather
	// than one being skipped: on a case-insensitive volume the two spellings
	// are one tree; on a sensitive one they are genuinely two.
	reg, _ := NewRegistry().Register(rec("couch-a", "/Users/x/repo", false))
	_, err := reg.Register(rec("couch-b", "/users/x/repo", false))
	var occ *TreeOccupiedError
	if caseInsensitiveFS() {
		if !errors.As(err, &occ) {
			t.Fatalf("err = %v; differently-cased spellings name one tree here", err)
		}
		return
	}
	if err != nil {
		t.Fatalf("err = %v; these are distinct trees on a case-sensitive filesystem", err)
	}
}

func TestRegisterAllowsLinkedWorktreeOfSameRepo(t *testing.T) {
	reg, err := NewRegistry().Register(rec("couch-a", "/w/ariadne", false))
	if err != nil {
		t.Fatalf("primary: %v", err)
	}
	if _, err := reg.Register(rec("couch-b", "/w/worktree/ariadne/000031", false)); err != nil {
		t.Fatalf("linked worktree refused: %v", err)
	}
}

func TestSameTreeOverrideKeepsIncumbentEnumerable(t *testing.T) {
	reg, _ := NewRegistry().Register(rec("couch-a", "/repo", false))
	reg, err := reg.Register(rec("couch-b", "/repo", true))
	if err != nil {
		t.Fatalf("override refused: %v", err)
	}
	got := reg.Get("/repo")
	if len(got) != 2 {
		t.Fatalf("got %d actors; the incumbent must survive, not be orphaned", len(got))
	}
}

func TestRegisterDoesNotMutateReceiverOnFailure(t *testing.T) {
	// Registry wraps a map, which is a reference type: a "functional"
	// signature over a bare map is a lie that lets a failed Register mutate
	// the caller's state anyway.
	base, _ := NewRegistry().Register(rec("couch-a", "/repo", false))
	before := len(base.Get("/repo"))
	if _, err := base.Register(rec("couch-b", "/repo", false)); err == nil {
		t.Fatal("expected refusal")
	}
	if after := len(base.Get("/repo")); after != before {
		t.Fatalf("receiver mutated: %d -> %d", before, after)
	}
}

func TestRegisterSucceedsWithoutMutatingReceiver(t *testing.T) {
	base := NewRegistry()
	next, err := base.Register(rec("couch-a", "/repo", false))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if len(base.Get("/repo")) != 0 {
		t.Fatal("the original registry must be unchanged")
	}
	if len(next.Get("/repo")) != 1 {
		t.Fatal("the returned registry must hold the record")
	}
}

func TestUnregisterFreesTheTree(t *testing.T) {
	reg, _ := NewRegistry().Register(rec("couch-a", "/repo", false))
	reg = reg.Unregister("/repo")
	if _, err := reg.Register(rec("couch-b", "/repo", false)); err != nil {
		t.Fatalf("re-register after unregister: %v", err)
	}
}

func TestPolicyTableDefaultsAndLookup(t *testing.T) {
	empty := PolicyTable{}
	if got := empty.Mode("pair"); got != InPlaceSerial {
		t.Fatalf("default = %q, want the conservative in-place-serial", got)
	}
	pt := PolicyTable{"kbench": HeavyLocalState, "xianxu.dev": WorktreeParallel}
	if got := pt.Mode("kbench"); got != HeavyLocalState {
		t.Fatalf("kbench = %q", got)
	}
	if got := pt.Mode("xianxu.dev"); got != WorktreeParallel {
		t.Fatalf("xianxu.dev = %q", got)
	}
}

func TestTreeOccupiedErrorCarriesPolicyMode(t *testing.T) {
	// The refusal offer is policy-shaped: suggesting "make a worktree" is
	// wrong for a repo whose worktrees are expensive.
	reg, _ := NewRegistry().Register(rec("couch-a", "/w/kbench", false))
	_, err := reg.RegisterWithPolicy(rec("couch-b", "/w/kbench", false), PolicyTable{"kbench": HeavyLocalState})
	var occ *TreeOccupiedError
	if !errors.As(err, &occ) {
		t.Fatalf("err = %v", err)
	}
	if occ.Mode != HeavyLocalState {
		t.Fatalf("Mode = %q, want the policy for this repo", occ.Mode)
	}
}
