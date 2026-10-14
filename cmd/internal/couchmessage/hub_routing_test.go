package couchmessage

import (
	"errors"
	"testing"
	"time"
)

func TestHubFamilyFilterPreservesGlobalIdentityAndExactRoutes(t *testing.T) {
	now := time.Unix(1000, 0)
	rows := []Candidate{shortCandidate("pair:1", "/w/pair/.git", "codex", now), shortCandidate("pair:2", "/w/pair/.git", "codex", now)}
	own := func(b Binding) bool { return b.Slot == "pair:2" }
	got, err := ResolveRecipient(Route{Target: "pair"}, rows, nil, now, own)
	if err != nil || got.Slot != "pair:2" {
		t.Fatalf("family %v %v", got, err)
	}
	got, err = ResolveRecipient(Route{Target: "pair:1"}, rows, nil, now, own)
	if err != nil || got.Slot != "pair:1" {
		t.Fatalf("exact %v %v", got, err)
	}
	rows[0].Binding.Repository = "/foreign/pair/.git"
	if _, err = ResolveRecipient(Route{Target: "pair"}, rows, nil, now, own); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("hidden identity ambiguity: %v", err)
	}
}
