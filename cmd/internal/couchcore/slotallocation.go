package couchcore

import "fmt"

// SlotAllocation is a proposed number; the guarded creation boundary must
// still reject a directory created after the inventory was observed.
type SlotAllocation struct{ Number int }

// SelectNewSlot consumes a complete repository inventory. Unverified or failed
// candidates need attention before another slot can be allocated.
func SelectNewSlot(candidates []SlotCandidate) (SlotAllocation, error) {
	used := make(map[int]bool, len(candidates))
	var repository SlotIdentity
	for _, candidate := range candidates {
		if candidate.Err != nil {
			return SlotAllocation{}, fmt.Errorf("slot %d needs attention: %w", candidate.Identity.Number, candidate.Err)
		}
		if !candidate.Verified {
			return SlotAllocation{}, fmt.Errorf("slot %d is not verified", candidate.Identity.Number)
		}
		slot := candidate.Identity
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
	// The first missing positive integer is at most len(candidates)+1.
	for number := 1; ; number++ {
		if !used[number] {
			return SlotAllocation{Number: number}, nil
		}
	}
}
