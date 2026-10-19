package couchscratch

import (
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// DefaultRoot is the optional mounted scratch volume.
const DefaultRoot = "/Volumes/TestScratch"

// Floor reserves two GiB before adopting scratch for a slot.
const Floor uint64 = 2 << 30

// Slot names a canonical repository and Git common-directory identity.
type Slot struct {
	Repo, Identity string
	Number         int
}

// Probe describes one observation of the scratch volume.
type Probe struct {
	Root      string
	Mounted   bool
	FreeBytes uint64
	Err       error
}

// ChooseScratch reports whether a probe permits preparing scratch.
func ChooseScratch(p Probe) bool {
	return p.Err == nil && p.Mounted && p.FreeBytes >= Floor && filepath.IsAbs(p.Root)
}

// slotName derives the stable directory basename for a canonical slot.
func slotName(s Slot) string { return couchcore.EnvName(s.Repo, s.Number) }
