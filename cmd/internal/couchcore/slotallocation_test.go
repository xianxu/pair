package couchcore

import (
	"errors"
	"fmt"
	"testing"
)

func allocationCandidate(n int) SlotCandidate {
	slot := targetTestSlot(n)
	slot.EnvironmentRoot = fmt.Sprintf("/src/worktree/pair-slot%d", n)
	slot.WorktreeRoot = slot.EnvironmentRoot + "/pair"
	return SlotCandidate{Identity: slot, Verified: true}
}

func occupiedSet(numbers ...int) map[int]bool {
	set := map[int]bool{}
	for _, n := range numbers {
		set[n] = true
	}
	return set
}

// #332: one rule for every number, :0 included -- the lowest number with no
// thread wins, reusing its directory when one exists; only when every number
// is taken does a new directory get allocated.
func TestSelectStartSlotFillsLowestFreeNumber(t *testing.T) {
	for _, tt := range []struct {
		name     string
		slots    []SlotCandidate
		occupied map[int]bool
		want     int
		exists   bool
	}{
		{"free primary, no slots", nil, occupiedSet(), 0, true},
		{"free primary beside live slots", []SlotCandidate{allocationCandidate(1)}, occupiedSet(1), 0, true},
		{"busy primary, no slots", nil, occupiedSet(0), 1, false},
		{"busy primary, one busy slot", []SlotCandidate{allocationCandidate(1)}, occupiedSet(0, 1), 2, false},
		{"missing directory hole", []SlotCandidate{allocationCandidate(3), allocationCandidate(1)}, occupiedSet(0, 1, 3), 2, false},
		{"threadless directory hole", []SlotCandidate{allocationCandidate(1), allocationCandidate(2)}, occupiedSet(0, 2), 1, true},
		{"lowest hole wins", []SlotCandidate{allocationCandidate(1), allocationCandidate(2)}, occupiedSet(1), 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SelectStartSlot(tt.slots, tt.occupied)
			if err != nil || got.Number != tt.want || got.Exists != tt.exists {
				t.Fatalf("allocation %+v, %v; want %d exists=%v", got, err, tt.want, tt.exists)
			}
		})
	}
}

// TestSelectStartSlotRefusesInconsistentInventory: an inventory that
// contradicts itself (an invalid number, a duplicate, another repository's
// slot) is still refused whole; it is not one slot's problem.
func TestSelectStartSlotRefusesInconsistentInventory(t *testing.T) {
	invalid := allocationCandidate(0)
	foreign := allocationCandidate(2)
	foreign.Identity.RepoIdentity = "/other/pair/.git"
	for _, candidates := range [][]SlotCandidate{
		{invalid}, {allocationCandidate(1), allocationCandidate(1)}, {allocationCandidate(1), foreign},
	} {
		if _, err := SelectStartSlot(candidates, occupiedSet(0)); err == nil {
			t.Fatalf("allocated around an inconsistent inventory: %+v", candidates)
		}
	}
}

// TestSelectStartSlotReconcilable (pair#387): a slot known only from leftovers
// or not yet verified is an existing number to reuse (reconciled on the reuse
// route), and any other failed slot skips only its own number. No candidate
// error refuses the whole repository.
func TestSelectStartSlotReconcilable(t *testing.T) {
	leftover := allocationCandidate(1)
	leftover.Verified = false
	leftover.Err = fmt.Errorf("%w: only its resting branch remains", ErrSlotNeedsReconcile)
	got, err := SelectStartSlot([]SlotCandidate{leftover}, occupiedSet(0))
	if err != nil || got.Number != 1 || !got.Exists || len(got.Notices) != 0 {
		t.Fatalf("leftover slot: %+v, %v; want reuse of 1", got, err)
	}
	unverified := allocationCandidate(1)
	unverified.Verified = false
	if got, err := SelectStartSlot([]SlotCandidate{unverified}, occupiedSet(0)); err != nil || got.Number != 1 || !got.Exists {
		t.Fatalf("unverified slot: %+v, %v; want reuse of 1", got, err)
	}
	broken := allocationCandidate(2)
	broken.Err = errors.New("permission denied")
	got, err = SelectStartSlot([]SlotCandidate{allocationCandidate(1), broken}, occupiedSet(0, 1))
	if err != nil || got.Number != 3 || got.Exists || len(got.Notices) != 1 {
		t.Fatalf("broken slot 2: %+v, %v; want a new slot 3 and a notice naming 2", got, err)
	}
	// The domain of candidate readings (verified, unverified, needs-reconcile,
	// any other error): none refuses the repository.
	readings := []func(SlotCandidate) SlotCandidate{
		func(c SlotCandidate) SlotCandidate { return c },
		func(c SlotCandidate) SlotCandidate { c.Verified = false; return c },
		func(c SlotCandidate) SlotCandidate { c.Verified, c.Err = false, ErrSlotNeedsReconcile; return c },
		func(c SlotCandidate) SlotCandidate { c.Verified, c.Err = false, errors.New("anything else"); return c },
	}
	for i, a := range readings {
		for j, b := range readings {
			if _, err := SelectStartSlot([]SlotCandidate{a(allocationCandidate(1)), b(allocationCandidate(2))}, occupiedSet(0)); err != nil {
				t.Errorf("readings %d,%d refused the repository: %v", i, j, err)
			}
		}
	}
}
