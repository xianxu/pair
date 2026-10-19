package couchcore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testSlotIdentity(t *testing.T, primary string, n int) SlotIdentity {
	t.Helper()
	s := conventionalSlot(primary, n)
	s.RepoIdentity = filepath.Join(primary, ".git")
	if err := s.Validate(); err != nil {
		t.Fatalf("fixture slot: %v", err)
	}
	return s
}

func testTeamCheckout(t *testing.T, teams ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range teams {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, ".hub"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestLifecycleAddressValidate(t *testing.T) {
	slot := testSlotIdentity(t, "/w/pair", 2)
	numbered, err := NewSlotAddress(slot, slot.WorktreeRoot)
	if err != nil {
		t.Fatalf("numbered slot: %v", err)
	}
	if err := numbered.Validate(); err != nil {
		t.Fatalf("numbered slot invalid: %v", err)
	}
	primary, err := NewSlotAddress(SlotIdentity{}, "/w/pair")
	if err != nil {
		t.Fatalf(":0 primary by root: %v", err)
	}
	if err := primary.Validate(); err != nil {
		t.Fatalf(":0 primary invalid: %v", err)
	}
	if _, err := NewSlotAddress(slot, "/w/elsewhere"); err == nil {
		t.Fatal("a numbered slot accepted a root that is not its own checkout")
	}
	for _, root := range []string{"", "relative/path", "/w/../w/pair", "/w/pair/"} {
		if _, err := NewSlotAddress(SlotIdentity{}, root); err == nil {
			t.Errorf("slot root %q accepted", root)
		}
	}

	checkout := testTeamCheckout(t, "teams/devinfra")
	team, err := NewTeamAddress(checkout, "teams/devinfra")
	if err != nil {
		t.Fatalf("team: %v", err)
	}
	if err := team.Validate(); err != nil {
		t.Fatalf("team invalid: %v", err)
	}
	if team.Team.Name != "devinfra" || team.Team.Scope == "" {
		t.Fatalf("team identity not resolved: %+v", team.Team)
	}
	for _, dir := range []string{"", ".", "../teams/devinfra", "/abs/teams/devinfra", "teams/devinfra/", "teams/missing"} {
		if _, err := NewTeamAddress(checkout, dir); err == nil {
			t.Errorf("team dir %q accepted", dir)
		}
	}
	if _, err := NewTeamAddress("relative", "teams/devinfra"); err == nil {
		t.Error("relative checkout root accepted")
	}

	// Exactly one arm, matching the kind; a hand-built value is checked too.
	broken := []LifecycleAddress{
		{},
		{Kind: AddressSlot},
		{Kind: AddressTeam},
		{Kind: AddressSlot, Root: "/w/pair", Team: team.Team},
		{Kind: AddressTeam, Team: team.Team, Root: "/w/pair"},
		{Kind: AddressTeam, Team: TeamIdentity{CheckoutRoot: checkout, TeamDir: "teams/devinfra", Name: "other", Scope: team.Team.Scope}},
		{Kind: AddressTeam, Team: TeamIdentity{CheckoutRoot: checkout, TeamDir: "teams/devinfra", Name: "devinfra"}},
		{Kind: "hub", Root: "/w/pair"},
	}
	for i, a := range broken {
		if err := a.Validate(); err == nil {
			t.Errorf("broken address %d validated: %+v", i, a)
		}
	}
}

func TestLifecycleAddressKeyDistinctPerTeam(t *testing.T) {
	checkout := testTeamCheckout(t, "teams/devinfra", "teams/ariadne")
	a, err := NewTeamAddress(checkout, "teams/devinfra")
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewTeamAddress(checkout, "teams/ariadne")
	if err != nil {
		t.Fatal(err)
	}
	if a.Key() == b.Key() {
		t.Fatalf("two teams in one checkout share a key: %s", a.Key())
	}
	if !strings.HasPrefix(a.Key(), "team:") {
		t.Fatalf("team key %q lacks its kind prefix", a.Key())
	}
}

func TestLifecycleAddressKeyDistinctPerCheckout(t *testing.T) {
	one := testTeamCheckout(t, "teams/devinfra")
	two := testTeamCheckout(t, "teams/devinfra")
	a, err := NewTeamAddress(one, "teams/devinfra")
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewTeamAddress(two, "teams/devinfra")
	if err != nil {
		t.Fatal(err)
	}
	if a.Key() == b.Key() {
		t.Fatalf("the same team folder in two checkouts shares a key: %s", a.Key())
	}
	slot, err := NewSlotAddress(SlotIdentity{}, one)
	if err != nil {
		t.Fatal(err)
	}
	if slot.Key() == a.Key() {
		t.Fatal("a slot and a team in one checkout share a key")
	}
}

// #683's failure log keys slot entries by the physical starting path; the
// lifecycle address's slot arm must produce the same key so failure records
// need no migration when callers move to LifecycleAddress.
func TestLifecycleAddressSlotKeyMatchesFailureSlotKey(t *testing.T) {
	slot := testSlotIdentity(t, "/w/pair", 3)
	numbered, err := NewSlotAddress(slot, slot.WorktreeRoot)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := numbered.Key(), LifecycleFailureSlotKey(LifecycleFailureTarget{StartingPath: slot.WorktreeRoot}); got != want {
		t.Fatalf("numbered slot key %q, failure slot key %q", got, want)
	}
	primary, err := NewSlotAddress(SlotIdentity{}, "/w/pair")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := primary.Key(), LifecycleFailureSlotKey(LifecycleFailureTarget{StartingPath: "/w/pair"}); got != want {
		t.Fatalf(":0 key %q, failure slot key %q", got, want)
	}
}
