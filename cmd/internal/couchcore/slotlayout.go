package couchcore

import (
	"path/filepath"
	"strconv"
	"strings"
)

// SlotLayout is the one authority for where a :1+ slot's state lives
// (pair#387). Every slot-state path and name is spelled here and nowhere else;
// TestEverySlotStateSiteIsAResource enforces it, and each Location method is
// one resource of SlotResources().
//
// A slot N of the repository whose primary checkout is <fleet>/<repo> lives in
// the environment <fleet>/worktree/<repo>-slotN, which holds the host checkout
// <repo>, the dependency clones weave makes beside it, weave's setup lock and
// Couch's store .couch.
// pair:m5-concept pure
type SlotLayout struct {
	primary, common string
	n               int
}

// NewSlotLayout is the layout of slot n of the repository at primaryRoot whose
// Git common directory is common ("" when unknown: Intent is then "").
func NewSlotLayout(primaryRoot, common string, n int) SlotLayout {
	return SlotLayout{primary: primaryRoot, common: common, n: n}
}

// LayoutOf is the layout of an identified slot.
func LayoutOf(s SlotIdentity) SlotLayout {
	return NewSlotLayout(s.PrimaryRoot, s.RepoIdentity, s.Number)
}

func (l SlotLayout) repo() string { return filepath.Base(l.primary) }

// Env is the slot's environment directory.
func (l SlotLayout) Env() string {
	return filepath.Join(WorktreesRoot(filepath.Dir(l.primary)), EnvName(l.repo(), l.n))
}

// Host is the slot's own checkout of the repository.
func (l SlotLayout) Host() string { return filepath.Join(l.Env(), l.repo()) }

// Store is Couch's per-slot store.
func (l SlotLayout) Store() string { return filepath.Join(l.Env(), ".couch") }

// SetupLock is weave's environment setup lock (weave owns it; Couch only
// probes it).
func (l SlotLayout) SetupLock() string { return filepath.Join(l.Env(), ".weave-setup.lock") }

// Intent is the legacy creation intent (retired by pair#387's reconciler).
func (l SlotLayout) Intent() string {
	if l.common == "" {
		return ""
	}
	return filepath.Join(l.common, CouchWorkspacesDir, strconv.Itoa(l.n), "creation.json")
}

// Registrations is git's directory of worktree registrations in the common
// directory; a slot's registration is one entry in it ("" when the common
// directory is unknown).
func (l SlotLayout) Registrations() string {
	if l.common == "" {
		return ""
	}
	return filepath.Join(l.common, "worktrees")
}

// RestingBranch is the slot's resting branch.
func (l SlotLayout) RestingBranch() string { return RestingBranch(l.n) }

// RestingRef is RestingBranch's full ref name.
func (l SlotLayout) RestingRef() string { return "refs/heads/" + l.RestingBranch() }

// CouchWorkspacesDir is Couch's directory inside a repository's Git common
// directory (the host creation lease and legacy intents live there).
const CouchWorkspacesDir = "couch-workspaces"

// SetupMarkerPath is Couch's setup-success marker in a slot registration's
// administrative directory (git rev-parse --absolute-git-dir).
func SetupMarkerPath(admin string) string { return filepath.Join(admin, "couch-setup-success.json") }

// WorktreesRoot is the directory under a fleet root holding every slot
// environment.
func WorktreesRoot(fleet string) string { return filepath.Join(fleet, "worktree") }

// EnvName is slot n's environment directory name.
func EnvName(repo string, n int) string { return repo + "-slot" + strconv.Itoa(n) }

// RestingBranch is the branch a checkout rests on: main for :0, main-slotN for
// slot N. sdlc workspace asserts the same convention, which validate() checks
// through this helper.
func RestingBranch(n int) string {
	if n == 0 {
		return "main"
	}
	return "main-slot" + strconv.Itoa(n)
}

// canonicalSlotNumber accepts only a canonical positive decimal.
func canonicalSlotNumber(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 || strconv.Itoa(n) != s {
		return 0, false
	}
	return n, true
}

// ParseEnvName recognizes repo's slot environment directory name.
func ParseEnvName(repo, name string) (int, bool) {
	suffix, ok := strings.CutPrefix(name, repo+"-slot")
	if !ok {
		return 0, false
	}
	return canonicalSlotNumber(suffix)
}

// ParseRestingBranch recognizes a :1+ resting branch name.
func ParseRestingBranch(branch string) (int, bool) {
	suffix, ok := strings.CutPrefix(branch, "main-slot")
	if !ok {
		return 0, false
	}
	return canonicalSlotNumber(suffix)
}

// parseEnvDirName splits an environment directory name into its repository
// and number. The last "-slot" separates them, so a repository name may itself
// contain "-slot".
func parseEnvDirName(name string) (string, int, bool) {
	i := strings.LastIndex(name, "-slot")
	if i <= 0 {
		return "", 0, false
	}
	repo := name[:i]
	n, ok := canonicalSlotNumber(name[i+len("-slot"):])
	return repo, n, ok && workspaceRepoName(repo)
}

// ParseSlotPath recognizes an absolute path at or inside a slot environment
// (the environment, its host checkout, a dependency beside it, or anything
// below) and returns the slot's primary root and number. It is a location, not
// proof that the slot exists or belongs to Git. The outermost
// worktree/<repo>-slotN component wins.
func ParseSlotPath(path string) (string, int, bool) {
	if !workspaceAbsolute(path) {
		return "", 0, false
	}
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	for i := 1; i+1 < len(parts); i++ {
		if parts[i] != "worktree" {
			continue
		}
		repo, n, ok := parseEnvDirName(parts[i+1])
		if !ok {
			continue
		}
		fleet := string(filepath.Separator) + filepath.Join(parts[1:i]...)
		return filepath.Join(fleet, repo), n, true
	}
	return "", 0, false
}
