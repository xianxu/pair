package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// errSetupRunning: weave holds its setup lock.
var errSetupRunning = errors.New("weave setup is running in this slot environment")

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
// deletion. Order (plan Task 3.1): register the store for collection, refuse
// at the limit, write the pending manifest, hold weave's setup lock, re-check
// the agent and the evidence, rename, mark complete.
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
	if cv.registerStore != nil {
		if err := cv.registerStore(ctx, l.Store()); err != nil {
			return fmt.Errorf("register the slot store for collection: %w", err)
		}
	}
	if n, err := savedWorkEntries(l); err != nil {
		return err
	} else if n >= MaxSavedWork {
		return fmt.Errorf("%s: %d saved-work entries (limit %d); restore or remove old entries first", StopReasonSavedWorkFull, n, MaxSavedWork)
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
		Kind: kind, Path: path, Branch: s.Branch, Restore: fmt.Sprintf("mv %q %q", filepath.Join(entry, "tree"), path)}
	manifestPath := filepath.Join(entry, "manifest.json")
	if err := cv.p.Store.Write(manifestPath, manifest); err != nil {
		return err
	}
	unlock, err := holdSetupLock(l.SetupLock())
	if err != nil {
		return err
	}
	defer unlock()
	if cv.agentNow != nil {
		if running, known := AgentRunning(cv.agentNow(ctx)); running || !known {
			return fmt.Errorf("%s: an agent appeared in the slot; nothing was moved", StopReasonAgentLive)
		}
	}
	if state, _, _ := checkoutEvidence(ctx, cv.p.IO, path); state != StateBroken {
		return fmt.Errorf("%s is no longer broken on positive evidence; nothing was moved", path)
	}
	rename := cv.rename
	if rename == nil {
		rename = os.Rename
	}
	if err := rename(path, filepath.Join(entry, "tree")); err != nil {
		return err
	}
	manifest.State = "complete"
	return cv.p.Store.Write(manifestPath, manifest)
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
