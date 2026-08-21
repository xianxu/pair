// Package couchcore is couch's domain: the registry of agent actors keyed on a
// working tree, the actor loop, and the operation surface. All IO sits behind
// injected seams (Runner, PathOps, GitRunner, Store, Clock, IDGen) so the
// domain is unit-testable without processes, disk, wall-clock or randomness.
package couchcore

import (
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizePath canonicalises a path so that spellings of one location compare
// equal: absolute, cleaned, case preserved.
//
// Symlink resolution is deliberately absent. filepath.EvalSymlinks is
// lstat/readlink and returns a different answer depending on what exists on
// disk; putting it here would make every entity that takes a Worktree inherit
// a filesystem dependency. It lives on the PathOps seam instead.
//
// One documented impurity: filepath.Abs reads os.Getwd() for a relative input.
// That is accepted rather than hidden -- the alternative is threading cwd
// through every caller for a case that only arises at the CLI edge.
func NormalizePath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		// Abs already calls Clean on its result, so there is deliberately no
		// second Clean here. A deletion check proved an explicit one was dead
		// code on this path.
		return abs
	}
	// Only reachable when os.Getwd fails. Clean anyway so the contract
	// (canonical spelling) holds even on the degraded path.
	return filepath.Clean(p)
}

// FoldKey returns the registry lookup key for a path.
//
// On darwin the default filesystem is case-insensitive-preserving, so
// "/users/x" and "/Users/x" name one directory but differ as Go strings.
// Without folding the key, couch would accept both spellings as distinct trees
// and the one-agent-per-tree guard would fail open -- the exact hazard it
// exists to prevent. Only the key is folded; the stored path keeps its case so
// it still derives the same scope key pair does, and renders correctly.
func FoldKey(p string) string { return foldWith(p, caseInsensitiveFS()) }

// foldWith is the pure decision, split out so both directions are testable on
// any platform.
func foldWith(p string, insensitive bool) string {
	if insensitive {
		return strings.ToLower(p)
	}
	return p
}

// caseInsensitiveFS keys on GOOS, though case-sensitivity is really a volume
// property -- a case-sensitive APFS dev volume would be treated as insensitive
// and two genuinely distinct trees conflated. It fails closed (refusing a
// spawn rather than allowing a collision), which is the acceptable direction.
func caseInsensitiveFS() bool { return runtime.GOOS == "darwin" || runtime.GOOS == "windows" }
