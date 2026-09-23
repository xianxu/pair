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

func TestSelectNewSlotUsesLowestMissingNumber(t *testing.T) {
	for _, tt := range []struct {
		slots []SlotCandidate
		want  int
	}{
		{nil, 1}, {[]SlotCandidate{allocationCandidate(1)}, 2},
		{[]SlotCandidate{allocationCandidate(3), allocationCandidate(1)}, 2},
	} {
		got, err := SelectNewSlot(tt.slots)
		if err != nil || got.Number != tt.want {
			t.Fatalf("allocation %+v, %v; want %d", got, err, tt.want)
		}
	}
}

func TestSelectNewSlotRefusesUncertainInventory(t *testing.T) {
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
		if _, err := SelectNewSlot(candidates); err == nil {
			t.Fatalf("allocated around uncertain inventory: %+v", candidates)
		}
	}
}
