package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const MaxSlotCandidates = 128

type SlotCatalog interface {
	Discover(context.Context, string) (SlotRepository, error)
}
type SlotRepository struct {
	Identity WorkspaceIdentity
	Slots    []SlotCandidate
}
type SlotCandidate struct {
	Identity SlotIdentity
	Verified bool
	Err      error
}
type OSSlotCatalog struct{ IO ProvisionIO }

func NewOSSlotCatalog(commandIO ProvisionIO) *OSSlotCatalog { return &OSSlotCatalog{IO: commandIO} }

func conventionalSlot(primary string, n int) SlotIdentity {
	repo := filepath.Base(primary)
	env := filepath.Join(filepath.Dir(primary), "worktree", repo+"-slot"+strconv.Itoa(n))
	return SlotIdentity{Repo: repo, RepoIdentity: filepath.Join(primary, ".git"), PrimaryRoot: primary, EnvironmentRoot: env, WorktreeRoot: filepath.Join(env, repo), Number: n}
}
func slotDirectoryNumber(primary, name string) (int, bool) {
	prefix := filepath.Base(primary) + "-slot"
	if !strings.HasPrefix(name, prefix) {
		return 0, false
	}
	suffix := strings.TrimPrefix(name, prefix)
	n, err := strconv.Atoi(suffix)
	return n, err == nil && n > 0 && strconv.Itoa(n) == suffix
}

// checkSlotDirectory permits absence, but never a symlink or non-directory.
func checkSlotDirectory(path string) error {
	if err := provisionSafePath(path); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("foreign slot path %s", path)
	}
	return nil
}
func inspectSlot(s SlotIdentity) (SlotCandidate, error) {
	c := SlotCandidate{Identity: s}
	for _, p := range []string{s.EnvironmentRoot, s.WorktreeRoot, filepath.Join(s.EnvironmentRoot, ".couch")} {
		if err := checkSlotDirectory(p); err != nil {
			return c, err
		}
	}
	if _, err := os.Stat(s.WorktreeRoot); err != nil {
		c.Err = fmt.Errorf("incomplete slot host %s: %w", s.WorktreeRoot, err)
	}
	return c, nil
}

// EnumerateSlotCandidates performs filesystem-only inventory. Returned identities
// are conventional locations, never Git verification. Missing Couch metadata is
// deliberately included; this function creates no directories or files.
func EnumerateSlotCandidates(primaryRoot string) ([]SlotCandidate, error) {
	if !workspaceAbsolute(primaryRoot) || !workspaceRepoName(filepath.Base(primaryRoot)) {
		return nil, fmt.Errorf("invalid primary root")
	}
	if err := checkSlotDirectory(primaryRoot); err != nil {
		return nil, err
	}
	if _, err := os.Stat(primaryRoot); err != nil {
		return nil, err
	}
	root := filepath.Join(filepath.Dir(primaryRoot), "worktree")
	if err := checkSlotDirectory(root); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var slots []SlotCandidate
	for _, entry := range entries {
		n, ok := slotDirectoryNumber(primaryRoot, entry.Name())
		if !ok {
			continue
		}
		if len(slots) >= MaxSlotCandidates {
			return nil, fmt.Errorf("slot candidate limit %d exceeded", MaxSlotCandidates)
		}
		c, err := inspectSlot(conventionalSlot(primaryRoot, n))
		if err != nil {
			return nil, err
		}
		slots = append(slots, c)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].Identity.Number < slots[j].Identity.Number })
	return slots, nil
}

// Discover probes SDLC and Git outside store locks. Incomplete and unverifiable
// hosts remain rows with errors; suspicious physical paths fail the inventory.
func (c *OSSlotCatalog) Discover(ctx context.Context, path string) (SlotRepository, error) {
	if ctx == nil || c == nil || c.IO == nil {
		return SlotRepository{}, fmt.Errorf("slot discovery unavailable")
	}
	if err := ctx.Err(); err != nil {
		return SlotRepository{}, err
	}
	if err := provisionSafePath(path); err != nil {
		return SlotRepository{}, err
	}
	p := NewWorkspaceProvisioner(c.IO)
	primary, err := p.identity(ctx, path)
	if err != nil {
		return SlotRepository{}, err
	}
	if primary.Kind != "primary" || primary.PrimaryRoot != path {
		return SlotRepository{}, fmt.Errorf("slot discovery requires primary checkout")
	}
	if err := provisionSafePath(primary.RepoIdentity); err != nil {
		return SlotRepository{}, err
	}
	slots, err := EnumerateSlotCandidates(primary.PrimaryRoot)
	if err != nil {
		return SlotRepository{}, err
	}
	raw, err := p.git(ctx, path, nil, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return SlotRepository{}, err
	}
	seen := make(map[int]bool)
	for _, s := range slots {
		seen[s.Identity.Number] = true
	}
	for _, field := range strings.Split(raw, "\x00") {
		if !strings.HasPrefix(field, "worktree ") {
			continue
		}
		host := strings.TrimPrefix(field, "worktree ")
		env := filepath.Dir(host)
		if filepath.Dir(env) != filepath.Join(primary.FleetRoot, "worktree") || filepath.Base(host) != primary.Repo {
			continue
		}
		n, ok := slotDirectoryNumber(path, filepath.Base(env))
		if !ok || seen[n] {
			continue
		}
		if len(slots) >= MaxSlotCandidates {
			return SlotRepository{}, fmt.Errorf("slot candidate limit %d exceeded", MaxSlotCandidates)
		}
		row, err := inspectSlot(conventionalSlot(path, n))
		if err != nil {
			return SlotRepository{}, err
		}
		slots = append(slots, row)
		seen[n] = true
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].Identity.Number < slots[j].Identity.Number })
	for i := range slots {
		if err := ctx.Err(); err != nil {
			return SlotRepository{}, err
		}
		row := &slots[i]
		row.Identity.RepoIdentity = primary.RepoIdentity
		if row.Err != nil {
			continue
		}
		before, err := os.Stat(row.Identity.WorktreeRoot)
		if err != nil {
			row.Err = err
			continue
		}
		host, err := p.verifyHost(ctx, primary, row.Identity.Number, nil)
		if err != nil {
			row.Err = err
			continue
		}
		id, err := SlotIdentityFromWorkspace(host.identity)
		if err != nil {
			row.Err = err
			continue
		}
		// A physical path replaced during subprocess discovery never becomes authority.
		checked, err := inspectSlot(id)
		if err != nil {
			return SlotRepository{}, err
		}
		if checked.Err != nil {
			row.Err = checked.Err
			continue
		}
		after, err := os.Stat(id.WorktreeRoot)
		if err != nil {
			row.Err = err
			continue
		}
		if !os.SameFile(before, after) {
			return SlotRepository{}, fmt.Errorf("slot host replaced during discovery: %s", id.WorktreeRoot)
		}
		row.Identity = id
		row.Verified = true
	}
	if err := ctx.Err(); err != nil {
		return SlotRepository{}, err
	}
	return SlotRepository{Identity: primary, Slots: slots}, nil
}
