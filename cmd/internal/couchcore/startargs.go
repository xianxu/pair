package couchcore

// StartArgs is the structured record of how to bring an actor up. It is
// persisted, so a revival reproduces the launch without the operator
// restating it.
//
// Spawn takes a peer repo, not an issue: what the agent works on is decided
// inside the session, and an issue crystallises mid-thread rather than being a
// precondition. Issue is optional metadata on the tree.
type StartArgs struct {
	Worktree  Worktree `json:"worktree"`
	Cwd       string   `json:"cwd,omitempty"`
	Stack     string   `json:"stack,omitempty"`
	Issue     string   `json:"issue,omitempty"`
	ExtraArgs []string `json:"extra_args,omitempty"`
	// SameTree is an inert legacy serialization field retained until M5 can
	// migrate old registry snapshots. New decisions must never read it.
	SameTree bool `json:"same_tree,omitempty"`
}

// WorkingDir is where the child actually starts: a subdirectory when the
// operator named one, the tree root otherwise.
func (a StartArgs) WorkingDir() string {
	if a.Cwd != "" {
		return a.Cwd
	}
	return string(a.Worktree)
}

func (a StartArgs) AgentStack() string {
	if a.Stack != "" {
		return a.Stack
	}
	return "claude"
}
