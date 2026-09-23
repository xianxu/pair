package couchcore

import "fmt"

type WorkspaceCandidateState string

const (
	WorkspaceAbsent   WorkspaceCandidateState = "absent"
	WorkspaceReady    WorkspaceCandidateState = "ready"
	WorkspaceOccupied WorkspaceCandidateState = "occupied"
	WorkspacePartial  WorkspaceCandidateState = "partial"
	WorkspaceUnknown  WorkspaceCandidateState = "unknown"
)

type WorkspaceCandidate struct {
	Slot  int
	State WorkspaceCandidateState
}

func SelectWorkspaceNumber(candidates []WorkspaceCandidate) (int, error) {
	seen := make(map[int]WorkspaceCandidateState, len(candidates))
	ready := 0
	for _, c := range candidates {
		if c.Slot <= 0 {
			return 0, fmt.Errorf("invalid workspace number %d", c.Slot)
		}
		if _, ok := seen[c.Slot]; ok {
			return 0, fmt.Errorf("duplicate workspace number %d", c.Slot)
		}
		seen[c.Slot] = c.State
		switch c.State {
		case WorkspaceReady:
			if ready == 0 || c.Slot < ready {
				ready = c.Slot
			}
		case WorkspaceAbsent, WorkspaceOccupied:
		case WorkspacePartial, WorkspaceUnknown:
			return 0, fmt.Errorf("workspace %d requires inspection before number selection", c.Slot)
		default:
			return 0, fmt.Errorf("workspace %d has ambiguous state %q", c.Slot, c.State)
		}
	}
	if ready > 0 {
		return ready, nil
	}
	// At most len(seen) positive numbers can be occupied; a gap must follow.
	for n := 1; ; n++ {
		state, ok := seen[n]
		if !ok || state == WorkspaceAbsent {
			return n, nil
		}
	}
}
