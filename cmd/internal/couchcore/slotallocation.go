package couchcore

import (
	"errors"
	"fmt"
)

// SlotAllocation is a proposed number; the guarded creation boundary must
// still reject a directory created after the inventory was observed. Exists
// says the number already has a checkout to reuse (:0 always does).
type SlotAllocation struct {
	Number int
	Exists bool
	// Notices name numbers skipped because they need attention that
	// reconcile cannot give (pair#387: never a repository-wide refusal).
	Notices []string
}

// SelectStartSlot picks where a new thread starts (#332): the lowest number,
// :0 included, that holds no thread, reusing its checkout when one exists; a
// new directory only when every number is taken. occupied names the numbers
// holding a thread (live, parked or otherwise). It consumes a complete
// repository inventory. A candidate known only from leftovers
// (ErrSlotNeedsReconcile) or not yet verified is an existing number to reuse,
// reconciled by the reuse route; any other failed candidate skips its own
// number with a notice, never the whole repository (pair#387).
func SelectStartSlot(candidates []SlotCandidate, occupied map[int]bool) (SlotAllocation, error) {
	used := make(map[int]bool, len(candidates))
	skipped := make(map[int]bool)
	var notices []string
	var repository SlotIdentity
	for _, candidate := range candidates {
		slot := candidate.Identity
		if candidate.Err != nil && !errors.Is(candidate.Err, ErrSlotNeedsReconcile) {
			// Only this number needs attention; the repository stays usable.
			skipped[slot.Number] = true
			notices = append(notices, fmt.Sprintf("slot %d skipped: %v", slot.Number, candidate.Err))
			continue
		}
		if err := slot.Validate(); err != nil {
			return SlotAllocation{}, err
		}
		if repository != (SlotIdentity{}) && (slot.RepoIdentity != repository.RepoIdentity || slot.PrimaryRoot != repository.PrimaryRoot || slot.Repo != repository.Repo) {
			return SlotAllocation{}, fmt.Errorf("slot inventory contains different repositories")
		}
		repository = slot
		if used[slot.Number] {
			return SlotAllocation{}, fmt.Errorf("duplicate slot number %d", slot.Number)
		}
		used[slot.Number] = true
	}
	// The first free number is at most len(occupied)+len(candidates)+1 away.
	// A reconcilable or unverified candidate is an existing number: the reuse
	// route reconciles it before anything starts there.
	for number := 0; ; number++ {
		if !occupied[number] && !skipped[number] {
			return SlotAllocation{Number: number, Exists: number == 0 || used[number], Notices: notices}, nil
		}
	}
}
