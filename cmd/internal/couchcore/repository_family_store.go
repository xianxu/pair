package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

func (s *ThreadStore) familyRootStore() *ThreadStore {
	if s == nil || !s.layout.Local {
		return s
	}
	root := NewThreadStore(s.namespace)
	root.coordinator = s.coordinator
	root.readOnly = s.readOnly
	return root
}

func (s *ThreadStore) PreviewRepositoryFamily(ctx context.Context, repository SlotRepository, requested RepositoryFamily) (RepositoryFamily, error) {
	return s.repositoryFamily(ctx, repository, requested, false)
}
func (s *ThreadStore) ReserveRepositoryFamily(ctx context.Context, repository SlotRepository, requested RepositoryFamily) (RepositoryFamily, error) {
	return s.repositoryFamily(ctx, repository, requested, true)
}
func (s *ThreadStore) repositoryFamily(ctx context.Context, repository SlotRepository, requested RepositoryFamily, reserve bool) (RepositoryFamily, error) {
	if s == nil || ctx == nil {
		return RepositoryFamily{}, errors.New("repository family admission requires a store and context")
	}
	if err := ctx.Err(); err != nil {
		return RepositoryFamily{}, err
	}
	s = s.familyRootStore()
	repository, requested, err := canonicalFamilyRequest(repository, requested)
	if err != nil {
		return RepositoryFamily{}, err
	}
	result, err := ResolveFamilyStart(nil, requested)
	if err != nil {
		return result, err
	}
	observedRoot := false
	run := func() error {
		observedRoot = true
		if err := ctx.Err(); err != nil {
			return err
		}
		manifest, raw, exists, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		var saved *RepositoryFamily
		for _, family := range manifest.RepositoryFamilies {
			if family.RepoIdentity == requested.RepoIdentity || family.PrimaryRoot == requested.PrimaryRoot {
				copy := family
				saved = &copy
				break
			}
		}
		if saved == nil {
			records, err := s.repositoryFamilyRecordsLocked(ctx, manifest, repository)
			if err != nil {
				return err
			}
			inferred, found, err := InferRepositoryFamily(repository, records)
			if err != nil {
				return err
			}
			if !found && slices.Contains(manifest.SlotRepositories, repository.Identity.PrimaryRoot) {
				inferred = RepositoryFamily{RepoIdentity: requested.RepoIdentity, PrimaryRoot: requested.PrimaryRoot, RelativeStart: "."}
				found = true
			}
			if found {
				saved = &inferred
			}
		}
		result, err = ResolveFamilyStart(saved, requested)
		if err != nil || !reserve {
			return err
		}
		for _, family := range manifest.RepositoryFamilies {
			if family == result {
				return nil
			}
		}
		next := manifest
		next.SchemaVersion = 2
		next.Generation++
		next.RepositoryFamilies = append(append([]RepositoryFamily(nil), manifest.RepositoryFamilies...), result)
		sort.Slice(next.RepositoryFamilies, func(i, j int) bool {
			return next.RepositoryFamilies[i].RepoIdentity < next.RepositoryFamilies[j].RepoIdentity
		})
		after, err := json.MarshalIndent(next, "", "  ")
		if err != nil {
			return err
		}
		after = append(after, '\n')
		var before *[]byte
		if exists {
			before = &raw
		}
		return s.commitJournalLockedChecked(storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{{Path: relativeStorePath(s.root, s.manifestPath()), Expected: before, After: &after}}}, ctx.Err)
	}
	if reserve {
		err = s.withLock(run)
	} else {
		err = s.withPreviewLock(run)
		// An absent global store has no lock or manifest to read, but retained
		// local checkouts can still establish the legacy family. This remains
		// a read-only preview; reservation repeats admission under the root lock.
		if err == nil && !observedRoot {
			err = run()
		}
	}
	return result, err
}

// Catalog identity is the external Git proof; canonicalize readable physical
// aliases without requiring future or retained missing checkout paths to exist.
func canonicalFamilyPath(path string) (string, error) {
	if !workspaceAbsolute(path) {
		return "", fmt.Errorf("invalid family path %q", path)
	}
	physical, err := filepath.EvalSymlinks(path)
	if errors.Is(err, os.ErrNotExist) {
		return path, nil
	}
	return physical, err
}
func canonicalFamilyRequest(repository SlotRepository, requested RepositoryFamily) (SlotRepository, RepositoryFamily, error) {
	if err := repository.Identity.validate(); err != nil {
		return repository, requested, err
	}
	var err error
	identity, primary := repository.Identity.RepoIdentity, repository.Identity.PrimaryRoot
	if requested.RepoIdentity == "" {
		requested.RepoIdentity = identity
	}
	if requested.PrimaryRoot == "" {
		requested.PrimaryRoot = primary
	}
	repository.Identity.RepoIdentity, err = canonicalFamilyPath(identity)
	if err != nil {
		return repository, requested, err
	}
	repository.Identity.PrimaryRoot, err = canonicalFamilyPath(primary)
	if err != nil {
		return repository, requested, err
	}
	requested.RepoIdentity, err = canonicalFamilyPath(requested.RepoIdentity)
	if err != nil {
		return repository, requested, err
	}
	requested.PrimaryRoot, err = canonicalFamilyPath(requested.PrimaryRoot)
	if err != nil {
		return repository, requested, err
	}
	if requested.RepoIdentity != repository.Identity.RepoIdentity || requested.PrimaryRoot != repository.Identity.PrimaryRoot {
		return repository, requested, errors.New("requested family disagrees with observed repository identity")
	}
	repository.Slots = append([]SlotCandidate(nil), repository.Slots...)
	for i := range repository.Slots {
		slot := &repository.Slots[i].Identity
		if slot.RepoIdentity != identity || slot.PrimaryRoot != primary {
			return repository, requested, errors.New("family catalog mixes repository identities")
		}
		slot.WorktreeRoot, err = canonicalFamilyPath(slot.WorktreeRoot)
		if err != nil {
			return repository, requested, err
		}
		slot.RepoIdentity, slot.PrimaryRoot = repository.Identity.RepoIdentity, repository.Identity.PrimaryRoot
	}
	return repository, requested, nil
}

func (s *ThreadStore) repositoryFamilyRecordsLocked(ctx context.Context, manifest threadManifest, repository SlotRepository) ([]ThreadRecord, error) {
	var records []ThreadRecord
	total, count := 0, 0
	read := func(store *ThreadStore, address ThreadAddress, path string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		count++
		if count > 4096 {
			return errors.New("family inference exceeds 4096 retained records; resolve legacy metadata before admission")
		}
		raw, err := store.readRetentionFile(path)
		if err != nil {
			return fmt.Errorf("read retained family evidence %s: %w", path, err)
		}
		total += len(raw)
		if total > slotMigrationMaxBytes {
			return errors.New("family inference exceeds 64 MiB of retained metadata")
		}
		record, err := store.decodeThreadRaw(address, raw)
		if err != nil {
			return nil
		}
		record.StartingPath, err = canonicalFamilyPath(record.StartingPath)
		if err != nil {
			return err
		}
		records = append(records, record)
		return nil
	}
	archives := func(store *ThreadStore) error {
		root := filepath.Join(store.root, "archive")
		return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if errors.Is(walkErr, os.ErrNotExist) && path == root {
				return nil
			}
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return errors.New("symlink in retained family archive")
			}
			if entry.IsDir() {
				return nil
			}
			relative, _ := filepath.Rel(root, path)
			parts := strings.Split(relative, string(filepath.Separator))
			if len(parts) != 2 || !strings.HasSuffix(parts[1], ".json") {
				return fmt.Errorf("unknown family archive entry %s", path)
			}
			address := ThreadAddress{RepoScope: parts[0], Tag: ThreadTag(strings.TrimSuffix(parts[1], ".json"))}
			if err := validateThreadAddress(address); err != nil {
				return err
			}
			return read(store, address, path)
		})
	}
	for _, address := range manifest.Threads {
		if err := read(s, address, s.recordPath(address)); err != nil {
			return nil, err
		}
	}
	if err := archives(s); err != nil {
		return nil, err
	}
	for _, candidate := range repository.Slots {
		local := newSlotThreadStore(s.namespace, candidate.Identity)
		if _, err := os.Lstat(local.root); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}
		if err := local.withPreviewLock(func() error {
			membership, _, _, err := local.loadManifestLocked()
			if err != nil {
				return err
			}
			for _, address := range membership.Threads {
				if err := read(local, address, local.recordPath(address)); err != nil {
					return err
				}
			}
			return archives(local)
		}); err != nil {
			return nil, err
		}
	}
	return records, nil
}
