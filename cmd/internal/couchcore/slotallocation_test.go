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

func TestSelectStartSlotRefusesUncertainInventory(t *testing.T) {
	partial := allocationCandidate(3)
	partial.Verified = false
	failed := allocationCandidate(3)
	failed.Err = errors.New("permission denied")
	invalid := allocationCandidate(0)
	foreign := allocationCandidate(2)
	foreign.Identity.RepoIdentity = "/other/pair/.git"
	for _, candidates := range [][]SlotCandidate{
		{allocationCandidate(1), partial}, {failed}, {invalid},
		{allocationCandidate(1), allocationCandidate(1)}, {allocationCandidate(1), foreign},
	} {
		if _, err := SelectStartSlot(candidates, occupiedSet(0)); err == nil {
			t.Fatalf("allocated around uncertain inventory: %+v", candidates)
		}
	}
}
