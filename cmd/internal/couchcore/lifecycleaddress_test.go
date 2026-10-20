package couchcore

import (
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

	// A hand-built value is checked too.
	broken := []LifecycleAddress{
		{},
		{Root: "relative"},
		{Slot: slot, Root: "/w/elsewhere"},
	}
	for i, a := range broken {
		if err := a.Validate(); err == nil {
			t.Errorf("broken address %d validated: %+v", i, a)
		}
	}
}

func TestLifecycleAddressKeyDistinctPerSlot(t *testing.T) {
	one, err := NewSlotAddress(SlotIdentity{}, "/w/pair")
	if err != nil {
		t.Fatal(err)
	}
	two, err := NewSlotAddress(testSlotIdentity(t, "/w/pair", 2), testSlotIdentity(t, "/w/pair", 2).WorktreeRoot)
	if err != nil {
		t.Fatal(err)
	}
	if one.Key() == two.Key() || !strings.HasPrefix(one.Key(), "slot:") {
		t.Fatalf("keys %q and %q", one.Key(), two.Key())
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
	// Routing #683's key through the address narrowed its input from any
	// absolute path to a clean one. A non-clean path has no slot key and the
	// failure matches by thread address instead; starting paths are canonical.
	for _, path := range []string{"/w/pair/", "/w/../w/pair", "relative"} {
		if key := LifecycleFailureSlotKey(LifecycleFailureTarget{StartingPath: path}); key != "" {
			t.Errorf("failure slot key for %q = %q, want none", path, key)
		}
	}
	thread := LifecycleFailureTarget{StartingPath: "/w/pair/", Address: ThreadAddress{RepoScope: "scope", Tag: "tag"}}
	if key := lifecycleFailureTargetKey(thread); key != "thread:scope/tag" {
		t.Fatalf("non-clean starting path key = %q, want the thread key", key)
	}
}
