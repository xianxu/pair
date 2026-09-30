package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// slotRepositoryManifest retains locations and their independently verified
// common-directory authority in one snapshot. Older stores may lack authority.
func (s *ThreadStore) slotRepositoryManifest() (threadManifest, error) {
	var snapshot threadManifest
	err := s.withLock(func() error { manifest, _, _, err := s.loadManifestLocked(); snapshot = manifest; return err })
	return snapshot, err
}

func retainedSlotCommonGit(manifest threadManifest, root string) string {
	if identity := manifest.SlotRepositoryIdentities[root]; identity != "" {
		return identity
	}
	for _, family := range manifest.RepositoryFamilies {
		if family.PrimaryRoot == root {
			return family.RepoIdentity
		}
	}
	return ""
}

// discoveredBackendsFromManifest performs no locking or mutation. Retention uses
// this after reading enrollment while already holding its coordinator lock.
// Git/process proof belongs to the operation opening a slot; enumeration does
// not grant launch authority.
func (s *ThreadStore) discoveredBackendsFromManifest(manifest threadManifest) ([]*ThreadStore, error) {
	var stores []*ThreadStore
	for _, root := range manifest.SlotRepositories {
		candidates, err := EnumerateSlotCandidates(root)
		if err != nil {
			return nil, fmt.Errorf("enumerate enrolled repository %s: %w", root, err)
		}
		for _, candidate := range candidates {
			candidate.Identity.RepoIdentity = retainedSlotCommonGit(manifest, root)
			local := newSlotThreadStore(s.namespace, candidate.Identity)
			local.coordinator = s.coordinator
			local.readOnly = s.readOnly
			stores = append(stores, local)
		}
	}
	sort.Slice(stores, func(i, j int) bool { return stores[i].root < stores[j].root })
	return stores, nil
}

// slotForContainedPath recognizes a conventional checkout without requiring it
// to exist yet. Component boundaries keep sibling repository names distinct.
func slotForContainedPath(root, path, commonGit string) (SlotIdentity, bool) {
	worktrees := filepath.Join(filepath.Dir(root), "worktree")
	rel, err := filepath.Rel(worktrees, path)
	if err != nil || filepath.IsAbs(rel) {
		return SlotIdentity{}, false
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) < 2 || parts[1] != filepath.Base(root) {
		return SlotIdentity{}, false
	}
	n, ok := slotDirectoryNumber(root, parts[0])
	if !ok {
		return SlotIdentity{}, false
	}
	slot := conventionalSlot(root, n)
	slot.RepoIdentity = commonGit
	if _, err := RelativeFamilyPath(slot.WorktreeRoot, path); err != nil {
		return SlotIdentity{}, false
	}
	return slot, true
}

// retainedPhysicalPath resolves existing ancestors while retaining a missing
// suffix. Missing CWDs must stay visible, but an existing symlink cannot redirect
// a retained record or preference into another checkout's storage.
func retainedPhysicalPath(path string) (string, error) {
	missing := []string{}
	for {
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if _, linkErr := os.Lstat(path); linkErr == nil {
			return "", err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", err
		}
		missing = append(missing, filepath.Base(path))
		path = parent
	}
}

func retainedPathWithinCheckout(root, path string) error {
	if _, err := RelativeFamilyPath(root, path); err != nil {
		return err
	}
	physicalRoot, err := retainedPhysicalPath(root)
	if err != nil {
		return err
	}
	physicalPath, err := retainedPhysicalPath(path)
	if err != nil {
		return err
	}
	if _, err := RelativeFamilyPath(physicalRoot, physicalPath); err != nil {
		return fmt.Errorf("path escapes its host checkout: %s", path)
	}
	return nil
}

func (s *ThreadStore) storeForPath(physicalPath, scope, commonGit string) (*ThreadStore, error) {
	if s.layout.Local {
		if s.slot == nil {
			return nil, errors.New("local store has no slot identity")
		}
		if _, belongs, err := CheckoutMembership(s.slot.RepoIdentity, s.slot.WorktreeRoot, physicalPath, scope, commonGit); err != nil || !belongs {
			return nil, fmt.Errorf("path identity does not belong to local checkout: %s (%v)", physicalPath, err)
		}
		if err := retainedPathWithinCheckout(s.slot.WorktreeRoot, physicalPath); err != nil {
			return nil, err
		}
		return s, nil
	}
	manifest, err := s.slotRepositoryManifest()
	if err != nil {
		return nil, err
	}
	for _, root := range manifest.SlotRepositories {
		slot, ok := slotForContainedPath(root, physicalPath, retainedSlotCommonGit(manifest, root))
		if !ok {
			continue
		}
		if slot.RepoIdentity == "" && scope == "" {
			return nil, fmt.Errorf("slot repository identity is unavailable; open %s in Couch to verify its enrollment before reading path preferences", root)
		}
		if _, belongs, err := CheckoutMembership(slot.RepoIdentity, slot.WorktreeRoot, physicalPath, scope, commonGit); err != nil {
			return nil, err
		} else if !belongs {
			continue
		}
		// Exact checkout scope supplied by the verified operation can carry its
		// common directory through preview without changing retained metadata.
		if slot.RepoIdentity == "" && scope != "" {
			slot.RepoIdentity = commonGit
		}
		if err := retainedPathWithinCheckout(slot.WorktreeRoot, physicalPath); err != nil {
			return nil, err
		}
		local := newSlotThreadStore(s.namespace, slot)
		local.coordinator = s.coordinator
		local.readOnly = s.readOnly
		return local, nil
	}
	return s, nil
}

func (s *ThreadStore) storeForAddress(address ThreadAddress) (*ThreadStore, error) {
	if err := validateThreadAddress(address); err != nil {
		return nil, err
	}
	if s.layout.Local {
		return s, nil
	}
	manifest, err := s.slotRepositoryManifest()
	if err != nil {
		return nil, err
	}
	if len(manifest.SlotRepositories) == 0 {
		return s, nil
	}
	// A valid ordinary record proves its own global storage origin, independent
	// of damage in some other slot's local metadata.
	var legacy ThreadRecord
	legacyErr := s.withLock(func() error {
		raw, err := os.ReadFile(s.recordPath(address))
		if errors.Is(err, os.ErrNotExist) {
			raw, err = s.readRetentionFile(s.archivePath(address))
		}
		if err != nil {
			return err
		}
		legacy, err = s.decodeThreadRaw(address, raw)
		return err
	})
	if legacyErr == nil && !recordInSlotRepositories(legacy, manifest) {
		return s, nil
	}
	stores, err := s.discoveredBackendsFromManifest(manifest)
	if err != nil {
		return nil, err
	}
	var found *ThreadStore
	var damaged error
	var explicitCollision bool
	for _, local := range stores {
		// Missing .couch is a visible recovery row, not permission to create state
		// merely because somebody looked up an unrelated tag.
		if _, err := os.Lstat(local.root); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}
		matched := false
		err := local.withLock(func() error {
			manifest, _, _, err := local.loadManifestLocked()
			if err != nil {
				// A readable address in a damaged envelope is collision evidence,
				// never permission to load or mutate that conversation.
				if raw, readErr := local.readRetentionFile(local.recordPath(address)); readErr == nil {
					var envelope struct {
						Address ThreadAddress `json:"address"`
					}
					if json.Unmarshal(raw, &envelope) == nil && envelope.Address == address {
						explicitCollision = true
					}
				}
				return err
			}
			matched = manifestContains(manifest, address)
			if !matched {
				_, err = local.readRetentionFile(local.archivePath(address))
				if err == nil {
					matched = true
				} else if !errors.Is(err, os.ErrNotExist) {
					return err
				}
			}
			return nil
		})
		if err != nil {
			damaged = errors.Join(damaged, fmt.Errorf("slot %s metadata requires recovery: %w", local.slot.WorktreeRoot, err))
			continue
		}
		if matched {
			if found != nil {
				return nil, fmt.Errorf("conversation %+v appears in multiple slots", address)
			}
			found = local
		}
	}
	if explicitCollision {
		return nil, damaged
	}
	if found != nil {
		return found, nil
	}
	if damaged != nil {
		return nil, damaged
	}
	// Any left-over global bytes for an enrolled slot are stale evidence, never
	// a fallback when the local current record was lost.
	if errors.Is(legacyErr, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: %+v", ErrThreadNotFound, address)
	}
	if legacyErr != nil {
		return nil, legacyErr
	}
	if recordInSlotRepositories(legacy, manifest) {
		return nil, fmt.Errorf("slot %s has no local current conversation %+v", legacy.StartingPath, address)
	}
	return s, nil
}

func recordInSlotRepositories(record ThreadRecord, manifest threadManifest) bool {
	for _, root := range manifest.SlotRepositories {
		slot, ok := slotForContainedPath(root, record.StartingPath, retainedSlotCommonGit(manifest, root))
		if !ok {
			continue
		}
		if _, belongs, _ := RecordCheckoutMembership(record, slot.RepoIdentity, slot.WorktreeRoot); belongs {
			return true
		}
	}
	return false
}

// repoLaunchDefault reads the primary repository's defaults for a numbered
// workspace. fallback preserves the caller's ordinary-path behavior and the
// known primary root during first creation, before the slot is enrolled.
func (c *Couch) repoLaunchDefault(path, fallback, agent, commonGit, scope string) (LaunchProfile, bool, error) {
	view := *c.Threads
	view.readOnly = true // Default lookup must not turn a start preview into a write.
	backend, err := view.storeForPath(path, scope, commonGit)
	if err != nil {
		return LaunchProfile{}, false, err
	}
	if backend.slot != nil {
		fallback = backend.slot.PrimaryRoot
	}
	return c.RepoAgentDefault(fallback, agent)
}
