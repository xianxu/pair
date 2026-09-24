package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// discoveredBackends rebuilds locations from enrolled repository roots, never
// from cached conversation addresses. Git/process proof belongs to the operation
// opening a slot; enumerating metadata here cannot grant launch authority.
func (s *ThreadStore) discoveredBackends() ([]*ThreadStore, error) {
	if s.layout.Local {
		return nil, nil
	}
	roots, err := s.slotRepositoryRoots()
	if err != nil {
		return nil, err
	}
	return s.discoveredBackendsFromRoots(roots)
}

func (s *ThreadStore) slotRepositoryRoots() ([]string, error) {
	var roots []string
	if err := s.withLock(func() error {
		manifest, _, _, err := s.loadManifestLocked()
		roots = append(roots, manifest.SlotRepositories...)
		return err
	}); err != nil {
		return nil, err
	}
	return roots, nil
}

// discoveredBackendsFromRoots performs no locking or mutation. Retention uses
// this after reading enrollment while already holding its coordinator lock.
func (s *ThreadStore) discoveredBackendsFromRoots(roots []string) ([]*ThreadStore, error) {
	var stores []*ThreadStore
	for _, root := range roots {
		candidates, err := EnumerateSlotCandidates(root)
		if err != nil {
			return nil, fmt.Errorf("enumerate enrolled repository %s: %w", root, err)
		}
		for _, candidate := range candidates {
			local := newSlotThreadStore(s.namespace, candidate.Identity)
			local.coordinator = s.coordinator
			local.readOnly = s.readOnly
			stores = append(stores, local)
		}
	}
	sort.Slice(stores, func(i, j int) bool { return stores[i].root < stores[j].root })
	return stores, nil
}

func (s *ThreadStore) storeForPath(physicalPath string) (*ThreadStore, error) {
	if s.layout.Local {
		return s, nil
	}
	roots, err := s.slotRepositoryRoots()
	if err != nil {
		return nil, err
	}
	for _, root := range roots {
		env := filepath.Dir(physicalPath)
		n, numbered := slotDirectoryNumber(root, filepath.Base(env))
		if numbered && filepath.Dir(env) == filepath.Join(filepath.Dir(root), "worktree") && filepath.Base(physicalPath) == filepath.Base(root) {
			local := newSlotThreadStore(s.namespace, conventionalSlot(root, n))
			local.coordinator = s.coordinator
			local.readOnly = s.readOnly
			return local, nil
		}
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
	roots, err := s.slotRepositoryRoots()
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 {
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
	if legacyErr == nil && !pathInSlotRepositories(legacy.StartingPath, roots) {
		return s, nil
	}
	stores, err := s.discoveredBackendsFromRoots(roots)
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
	if pathInSlotRepositories(legacy.StartingPath, roots) {
		return nil, fmt.Errorf("slot %s has no local current conversation %+v", legacy.StartingPath, address)
	}
	return s, nil
}

func pathInSlotRepositories(path string, roots []string) bool {
	for _, root := range roots {
		env := filepath.Dir(path)
		_, numbered := slotDirectoryNumber(root, filepath.Base(env))
		if numbered && filepath.Dir(env) == filepath.Join(filepath.Dir(root), "worktree") && filepath.Base(path) == filepath.Base(root) {
			return true
		}
	}
	return false
}

// repoLaunchDefault reads the primary repository's defaults for a numbered
// workspace. fallback preserves the caller's ordinary-path behavior and the
// known primary root during first creation, before the slot is enrolled.
func (c *Couch) repoLaunchDefault(path, fallback, agent string) (LaunchProfile, bool, error) {
	view := *c.Threads
	view.readOnly = true // Default lookup must not turn a start preview into a write.
	backend, err := view.storeForPath(path)
	if err != nil {
		return LaunchProfile{}, false, err
	}
	if backend.slot != nil {
		fallback = backend.slot.PrimaryRoot
	}
	return c.RepoAgentDefault(fallback, agent)
}
