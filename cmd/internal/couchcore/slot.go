package couchcore

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// SlotIdentity names a durable numbered environment. Identity alone is not
// filesystem or Git proof; consumers must validate current observations.
type SlotIdentity struct {
	Repo, RepoIdentity, PrimaryRoot, EnvironmentRoot, WorktreeRoot string
	Number                                                         int
}

func (s SlotIdentity) Validate() error {
	if !workspaceRepoName(s.Repo) || s.Number <= 0 || filepath.Base(s.PrimaryRoot) != s.Repo {
		return fmt.Errorf("invalid slot name or number")
	}
	for _, p := range []string{s.RepoIdentity, s.PrimaryRoot, s.EnvironmentRoot, s.WorktreeRoot} {
		if !workspaceAbsolute(p) {
			return fmt.Errorf("invalid slot path %q", p)
		}
	}
	expected := filepath.Join(filepath.Dir(s.PrimaryRoot), "worktree", s.Repo+"-slot"+strconv.Itoa(s.Number))
	if s.EnvironmentRoot != expected || s.WorktreeRoot != filepath.Join(expected, s.Repo) {
		return fmt.Errorf("inconsistent slot paths")
	}
	return nil
}

func SlotIdentityFromWorkspace(id WorkspaceIdentity) (SlotIdentity, error) {
	if err := id.validate(); err != nil {
		return SlotIdentity{}, err
	}
	if id.Kind != "slot" {
		return SlotIdentity{}, fmt.Errorf("workspace is not a numbered slot")
	}
	s := SlotIdentity{Repo: id.Repo, RepoIdentity: id.RepoIdentity, PrimaryRoot: id.PrimaryRoot, EnvironmentRoot: id.EnvironmentRoot, WorktreeRoot: id.WorktreeRoot, Number: *id.Slot}
	return s, s.Validate()
}
