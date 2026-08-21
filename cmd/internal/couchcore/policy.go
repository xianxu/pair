package couchcore

// Mode is a repo's concurrency policy. It is a stable property of a repo, so
// it is recorded and read deterministically rather than inferred per spawn.
type Mode string

const (
	// InPlaceSerial: the checkout is the installation (pair, ariadne,
	// parley), so one agent at a time and worktrees are awkward.
	InPlaceSerial Mode = "in-place-serial"
	// WorktreeParallel: worktrees are cheap, so concurrency is free.
	WorktreeParallel Mode = "worktree-parallel"
	// HeavyLocalState: worktrees are expensive for reasons unrelated to
	// dogfooding -- large data, caches -- so suggesting one is bad advice.
	HeavyLocalState Mode = "heavy-local-state"
)

// PolicyTable is pure: repo display name -> Mode. Loading it from disk is the
// Store's job. A pure entity that reads a file is the defect this design has
// already corrected once.
type PolicyTable map[string]Mode

func (p PolicyTable) Mode(repo string) Mode {
	if m, ok := p[repo]; ok {
		return m
	}
	return InPlaceSerial
}
