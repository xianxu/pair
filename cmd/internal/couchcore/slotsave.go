package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/storagegc"

	"golang.org/x/sys/unix"
)

// MaxSavedWork bounds a slot's saved-work entries; at the limit reconcile
// holds instead of evicting (ARCH-FUNERAL: collection is by age, pair#387 M3).
const MaxSavedWork = 16

// SavedWorkManifest records one checkout reconcile set aside (pair#387). It is
// written pending before the move and complete after, so a crash at any point
// leaves a record naming the checkout and how to restore it.
type SavedWorkManifest struct {
	SchemaVersion int       `json:"schema_version"`
	State         string    `json:"state"` // pending | complete
	SavedAt       time.Time `json:"saved_at"`
	Slot          string    `json:"slot"`
	Kind          string    `json:"kind"` // host | dep
	Path          string    `json:"path"`
	Branch        string    `json:"branch,omitempty"`
	Restore       string    `json:"restore"`
}

// The typed refusals of converge steps (ClassifyConvergeError maps each to
// the class the plan gives the same condition): weave holds its setup lock
// (retryable); an agent appeared, or saved work is full (holds).
var (
	errSetupRunning      = errors.New("weave setup is running in this slot environment")
	errAgentAppeared     = errors.New(StopReasonAgentLive + ": an agent appeared in the slot")
	errAgentUnobserved   = errors.New(StopReasonAgentUnknown + ": the slot's agent could not be observed")
	errSavedWorkFull     = errors.New(StopReasonSavedWorkFull)
	errCheckoutRecovered = errors.New("the checkout is no longer broken on positive evidence")
)

// setAsideHoldError is SetAsideHold's reason as the typed error the step
// returns, so ClassifyConvergeError lands on the class the plan gave it.
func setAsideHoldError(reason string) error {
	switch reason {
	case StopReasonAgentUnknown:
		return errAgentUnobserved
	case StopReasonSavedWorkFull:
		return errSavedWorkFull
	}
	return errAgentAppeared
}

// checkoutEvidence is the one positive-evidence reading of a checkout: git's
// own answer about the work tree (R2). "Not a git repository", or a work tree
// that is not the directory itself, is broken; any other git failure proves
// nothing.
func checkoutEvidence(ctx context.Context, io ProvisionIO, path string) (ObservedState, string, string) {
	out, err := io.Run(ctx, ProvisionCommand{Dir: path, Program: "git", Args: []string{"rev-parse", "--show-toplevel"}})
	top := strings.TrimSpace(string(out))
	switch {
	case notAGitRepository(err, top):
		return StateBroken, SubUnreadable, "git cannot read " + path
	case err != nil:
		return StateUnknown, "", err.Error()
	case filepath.Clean(top) != path:
		return StateBroken, SubUnreadable, path + " has no repository of its own"
	}
	return StatePresent, "", ""
}

// savedWorkEntries counts a slot's saved-work entries.
func savedWorkEntries(l SlotLayout) (int, error) {
	entries, err := os.ReadDir(l.SavedWork())
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	return len(entries), err
}

// setAside moves a checkout broken on positive evidence, whole, into the
// slot's saved work: the only way reconcile removes a checkout, and never a
// deletion. Every check that can refuse runs before anything is written, so a
// refusal leaves no residue (no entry counts toward the cap): refuse at the
// limit, hold weave's setup lock, re-check the agent and
// the evidence; then write the pending manifest, rename, mark it complete.
// Every refusal is a typed error ClassifyConvergeError maps to the class the
// plan would give it.
func (cv *slotConverger) setAside(ctx context.Context, s PlannedStep) error {
	l := cv.layout
	path := filepath.Clean(s.Path)
	if filepath.Dir(path) != l.Env() || provisionSafePath(path) != nil {
		return fmt.Errorf("refusing to set aside %s: not a checkout directly in %s", path, l.Env())
	}
	if info, err := os.Lstat(path); err != nil {
		return err
	} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("refusing to set aside %s: not a directory", path)
	}
	n, err := savedWorkEntries(l)
	if err != nil {
		return err
	}
	if reason, held := SetAsideHold(AgentNone, n >= MaxSavedWork); held {
		return fmt.Errorf("%w: %d entries (limit %d); restore or remove old entries first", setAsideHoldError(reason), n, MaxSavedWork)
	}
	unlock, err := holdSetupLock(l.SetupLock())
	if err != nil {
		return err
	}
	defer unlock()
	if cv.agentNow != nil {
		if reason, held := SetAsideHold(cv.agentNow(ctx), false); held {
			return fmt.Errorf("%w; nothing was moved", setAsideHoldError(reason))
		}
	}
	if state, _, _ := checkoutEvidence(ctx, cv.p.IO, path); state != StateBroken {
		return fmt.Errorf("%w: %s; nothing was moved", errCheckoutRecovered, path)
	}
	now := time.Now().UTC()
	entry := l.SavedWorkEntry(path, now)
	if err := provisionMkdirAll(entry); err != nil {
		return err
	}
	kind := "dep"
	if s.Resource == ResourceHost {
		kind = "host"
	}
	manifest := SavedWorkManifest{SchemaVersion: 1, State: "pending", SavedAt: now, Slot: WorkspaceReference{Repo: l.repo(), Number: l.n}.String(),
		Kind: kind, Path: path, Branch: s.Branch, Restore: SavedWorkRestoreCommand(entry, path)}
	manifestPath := filepath.Join(entry, "manifest.json")
	if err := cv.p.Store.Write(manifestPath, manifest); err != nil {
		return err
	}
	rename := cv.rename
	if rename == nil {
		rename = os.Rename
	}
	if err := rename(path, filepath.Join(entry, "tree")); err != nil {
		return err
	}
	// The move is the effect: report it now, before the bookkeeping that
	// follows can fail (its pending manifest already names the restore).
	if cv.setAsideDone != nil {
		cv.setAsideDone(entry)
	}
	manifest.State = "complete"
	return cv.p.Store.Write(manifestPath, manifest)
}

// SavedWorkRestoreCommand is the POSIX shell command that puts a set-aside
// checkout back from the state reconcile leaves: whatever now stands at the
// path (the recreated checkout) moves into the entry as "recreated", then the
// saved tree moves back to the path.
func SavedWorkRestoreCommand(entry, path string) string {
	recreated, tree := filepath.Join(entry, "recreated"), filepath.Join(entry, "tree")
	return fmt.Sprintf("{ [ ! -e %s ] || mv %s %s; } && mv %s %s",
		ShellQuote(path), ShellQuote(path), ShellQuote(recreated), ShellQuote(tree), ShellQuote(path))
}

// SavedWorkManifests reads the manifests of the given saved-work entries
// (an unreadable one is skipped: the entry is still on disk under its name).
func SavedWorkManifests(entries []string) []SavedWorkManifest {
	var out []SavedWorkManifest
	for _, entry := range entries {
		var m SavedWorkManifest
		if exists, err := (ProvisionStore{}).Read(filepath.Join(entry, "manifest.json"), &m); err == nil && exists {
			out = append(out, m)
		}
	}
	return out
}

// collectSavedWork removes saved-work entries older than the storage retention
// period (by the manifest's saved_at, else the entry directory's age). The
// reconciler that writes saved work collects it at the start of every run on
// the slot (ARCH-FUNERAL); at most MaxSavedWork entries wait for that.
func collectSavedWork(l SlotLayout, now time.Time) ([]string, error) {
	entries, err := os.ReadDir(l.SavedWork())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var removed []string
	for _, e := range entries {
		entry := l.SavedWorkEntryNamed(e.Name())
		info, err := os.Lstat(entry)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue // not an entry this code wrote; never followed or removed
		}
		saved := info.ModTime()
		var m SavedWorkManifest
		// A saved_at in the future (a hand-edited or skewed manifest) is not
		// believed: it would keep the entry forever.
		if exists, err := (ProvisionStore{}).Read(filepath.Join(entry, "manifest.json"), &m); err == nil && exists && !m.SavedAt.IsZero() && !m.SavedAt.After(now) {
			saved = m.SavedAt
		}
		if now.Sub(saved) < storagegc.RetentionPeriod {
			continue
		}
		if err := os.RemoveAll(entry); err != nil {
			return removed, err
		}
		removed = append(removed, entry)
	}
	return removed, nil
}

// holdSetupLock takes weave's setup lock if its file exists (an absent file
// means no setup has ever run, so none can be running); held by weave is
// errSetupRunning.
func holdSetupLock(path string) (func(), error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if errors.Is(err, unix.ENOENT) {
		return func() {}, nil
	}
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		unix.Close(fd)
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, errSetupRunning
		}
		return nil, err
	}
	return func() { unix.Close(fd) }, nil
}
