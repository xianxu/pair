package couchcore

import "fmt"

// SlotAllocation is a proposed number; the guarded creation boundary must
// still reject a directory created after the inventory was observed. Exists
// says the number already has a checkout to reuse (:0 always does).
type SlotAllocation struct {
	Number int
	Exists bool
}

// SelectStartSlot picks where a new thread starts (#332): the lowest number,
// :0 included, that holds no thread, reusing its checkout when one exists; a
// new directory only when every number is taken. occupied names the numbers
// holding a thread (live, parked or otherwise). It consumes a complete
// repository inventory: unverified or failed candidates need attention first.
func SelectStartSlot(candidates []SlotCandidate, occupied map[int]bool) (SlotAllocation, error) {
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
	// The first free number is at most len(occupied)+len(candidates)+1 away.
	for number := 0; ; number++ {
		if !occupied[number] {
			return SlotAllocation{Number: number, Exists: number == 0 || used[number]}, nil
		}
	}
}
