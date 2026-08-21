package couchcore

import (
	"fmt"
	"path/filepath"
)

// Worktree is a canonical absolute worktree-root path, in original case. The
// named type carries one invariant: this path has been canonicalised. A
// function taking a Worktree cannot be handed "../pair" by accident.
//
// Identity is the tree, not the issue and not a subdirectory: the collision
// hazards -- one index lock, one branch, one `git status` -- are properties of
// the tree. See workshop/projects/couch.md, scope event 2026-08-21.
type Worktree string

// Key is the registry lookup key: case-folded where the filesystem is.
func (w Worktree) Key() string { return FoldKey(string(w)) }

// Repo is the display name (basename), unfolded. Convention borrowed from
// launcher.repoDisplayName (scope.go:33, unexported).
func (w Worktree) Repo() string { return filepath.Base(string(w)) }

// Resolve canonicalises path, resolves it through the PathOps seam, asks git
// for its worktree root, and canonicalises that answer too.
//
// Both Physical calls are load-bearing and independently tested: the first so
// git runs in the resolved directory, the second because git can itself report
// a symlinked toplevel and two spellings would then become two trees.
func Resolve(path string, git GitRunner, p PathOps) (Worktree, error) {
	dir, err := p.Physical(NormalizePath(path))
	if err != nil {
		return "", fmt.Errorf("resolve worktree: %w", err)
	}
	top, err := git.Run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("resolve worktree for %s: %w", dir, err)
	}
	if top == "" {
		return "", fmt.Errorf("resolve worktree for %s: empty toplevel", dir)
	}
	root, err := p.Physical(NormalizePath(top))
	if err != nil {
		return "", fmt.Errorf("resolve worktree root: %w", err)
	}
	return Worktree(root), nil
}
