package couchcore

import (
	"fmt"
	"path/filepath"
)

// PathOps is the seam over symlink resolution -- lstat/readlink, so it cannot
// live in the pure core.
//
// Physical returns an error rather than falling back to its input. The house
// precedent, reviewcmd.Runtime.PhysicalDir (reviewcmd/runtime.go:45-57),
// returns "" on failure for the same reason: the caller must be able to detect
// a path that does not resolve. Silently accepting an unresolvable path would
// let it become its own actor identity.
type PathOps interface {
	Physical(path string) (string, error)
}

type OSPathOps struct{}

var _ PathOps = OSPathOps{}

func (OSPathOps) Physical(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", path, err)
	}
	return resolved, nil
}

// FakePathOps maps link -> real; unmapped paths pass through unchanged, and
// paths marked with Fail return an error.
type FakePathOps struct {
	links map[string]string
	fails map[string]bool
}

var _ PathOps = (*FakePathOps)(nil)

func NewFakePathOps(links map[string]string) *FakePathOps {
	if links == nil {
		links = map[string]string{}
	}
	return &FakePathOps{links: links, fails: map[string]bool{}}
}

func (f *FakePathOps) Fail(path string) { f.fails[path] = true }

func (f *FakePathOps) Physical(path string) (string, error) {
	if f.fails[path] {
		return "", fmt.Errorf("resolve %s: no such file or directory", path)
	}
	if real, ok := f.links[path]; ok {
		return real, nil
	}
	return path, nil
}
