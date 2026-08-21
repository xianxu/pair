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
	// SameTree is the loud escape hatch past the one-agent-per-tree guard.
	// It is recorded here and nowhere else -- a duplicate flag on ActorRecord
	// would leave a test unable to say which one Register read.
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
