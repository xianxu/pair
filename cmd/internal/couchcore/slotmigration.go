package couchcore

import (
	"bytes"
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

const slotMigrationMaxBytes = 64 << 20

type slotMigrationPayload struct {
	source, target string
	raw            []byte
}
type slotMigrationBatch struct {
	local    *ThreadStore
	payloads []slotMigrationPayload
}

// EnrollSlotRepository consumes catalog proof obtained outside storage locks.
// All local payloads are published before the single root cutover. Until that
// global journal is committed they are staging only, and global remains owner.
// Its first entry publishes root enrollment; remaining entries retire source
// bytes. Recovery replays the same exact images before any subsequent writer.
func (s *ThreadStore) EnrollSlotRepository(ctx context.Context, repository SlotRepository) error {
	if ctx == nil {
		return errors.New("slot enrollment needs context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.layout.Local {
		return errors.New("slot enrollment requires global store")
	}
	if err := repository.Identity.validate(); err != nil {
		return err
	}
	if repository.Identity.Kind != "primary" {
		return errors.New("slot enrollment requires primary repository")
	}
	if len(repository.Slots) > MaxSlotCandidates {
		return errors.New("slot enrollment exceeds candidate limit")
	}
	slots := append([]SlotCandidate(nil), repository.Slots...)
	sort.Slice(slots, func(i, j int) bool { return slots[i].Identity.EnvironmentRoot < slots[j].Identity.EnvironmentRoot })
	byPath := make(map[string]*slotMigrationBatch)
	var batches []*slotMigrationBatch
	for _, candidate := range slots {
		id := candidate.Identity
		if err := id.Validate(); err != nil {
			return err
		}
		if id.PrimaryRoot != repository.Identity.PrimaryRoot || id.RepoIdentity != repository.Identity.RepoIdentity {
			return errors.New("slot enrollment mixes repositories")
		}
		if _, exists := byPath[id.WorktreeRoot]; exists {
			return errors.New("duplicate slot enrollment candidate")
		}
		batch := &slotMigrationBatch{local: newSlotThreadStore(s.namespace, id)}
		batch.local.hooks.AfterPublicationWrite = s.hooks.AfterPublicationWrite
		byPath[id.WorktreeRoot] = batch
		batches = append(batches, batch)
	}
	return s.withLock(func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		manifest, manifestRaw, exists, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		if slices.Contains(manifest.SlotRepositories, repository.Identity.PrimaryRoot) {
			return nil
		}
		observed, err := EnumerateSlotCandidates(repository.Identity.PrimaryRoot)
		if err != nil {
			return err
		}
		for _, candidate := range observed {
			if byPath[candidate.Identity.WorktreeRoot] == nil {
				return fmt.Errorf("slot appeared after discovery: %s", candidate.Identity.WorktreeRoot)
			}
		}
		total := 0
		read := func(path string) ([]byte, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			raw, err := s.readRetentionFile(path)
			if err != nil {
				return nil, err
			}
			total += len(raw)
			if total > slotMigrationMaxBytes {
				return nil, errors.New("slot migration exceeds 64 MiB metadata")
			}
			return raw, nil
		}
		removeAddresses := make(map[ThreadAddress]bool)
		currentPaths := make(map[string]bool)
		for _, address := range manifest.Threads {
			raw, err := read(s.recordPath(address))
			if err != nil {
				return fmt.Errorf("inspect migration source %+v: %w", address, err)
			}
			record, err := s.decodeThreadRaw(address, raw)
			if err != nil {
				return err
			}
			batch := byPath[record.StartingPath]
			if batch == nil {
				continue
			}
			if currentPaths[record.StartingPath] {
				return fmt.Errorf("multiple legacy current records for slot %s", record.StartingPath)
			}
			currentPaths[record.StartingPath] = true
			if err := batch.local.validateLocalOrigin(record); err != nil {
				return err
			}
			batch.payloads = append(batch.payloads, slotMigrationPayload{s.recordPath(address), batch.local.recordPath(address), raw})
			removeAddresses[address] = true
			// The checkpoint embedded in raw is authoritative; materialize only when
			// continuation is used. A stale derived global file can be removed safely.
			if derived, err := read(s.continuationPath(address)); err == nil {
				batch.payloads = append(batch.payloads, slotMigrationPayload{source: s.continuationPath(address), raw: derived})
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		archiveRoot := filepath.Join(s.root, "archive")
		err = filepath.WalkDir(archiveRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if errors.Is(walkErr, os.ErrNotExist) && path == archiveRoot {
				return nil
			}
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return errors.New("symlink in migration archives")
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(archiveRoot, path)
			if err != nil {
				return err
			}
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) != 2 || !strings.HasSuffix(parts[1], ".json") {
				return errors.New("unknown migration archive entry")
			}
			address := ThreadAddress{RepoScope: parts[0], Tag: ThreadTag(strings.TrimSuffix(parts[1], ".json"))}
			raw, err := read(path)
			if err != nil {
				return err
			}
			record, err := s.decodeThreadRaw(address, raw)
			if err != nil {
				return fmt.Errorf("unreadable migration archive %s: %w", path, err)
			}
			batch := byPath[record.StartingPath]
			if batch == nil {
				return nil
			}
			if err := batch.local.validateLocalOrigin(record); err != nil {
				return err
			}
			batch.payloads = append(batch.payloads, slotMigrationPayload{path, batch.local.archivePath(address), raw})
			gracePath := s.archiveGracePath(address)
			if grace, err := read(gracePath); err == nil {
				batch.payloads = append(batch.payloads, slotMigrationPayload{gracePath, batch.local.archiveGracePath(address), grace})
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
		for i, batch := range batches {
			id := batch.local.slot
			preferencePath := s.pathLaunchPreferencePath(id.RepoIdentity, id.WorktreeRoot)
			if raw, err := read(preferencePath); err == nil {
				var preference PathLaunchPreference
				if err := strictThreadStoreJSON(raw, &preference); err != nil {
					return err
				}
				if err := validatePathLaunchPreference(preference); err != nil {
					return err
				}
				if preference.RepoIdentity != id.RepoIdentity || preference.PhysicalPath != id.WorktreeRoot {
					return errors.New("migration preference identity mismatch")
				}
				batch.payloads = append(batch.payloads, slotMigrationPayload{preferencePath, batch.local.pathLaunchPreferencePath(id.RepoIdentity, id.WorktreeRoot), raw})
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if len(batch.payloads) > 0 && (!slots[i].Verified || slots[i].Err != nil) {
				return fmt.Errorf("slot %s requires verified Git identity before migration", id.WorktreeRoot)
			}
		}
		// Bound and validate the complete source set before writing any local bytes.
		for _, batch := range batches {
			if len(batch.payloads) == 0 {
				// With no legacy source there is nothing to migrate. Existing
				// local state is already authoritative, including damaged evidence
				// that explicit slot recovery may inspect later. Re-enrollment
				// after losing the root listing must leave these bytes untouched.
				if _, err := os.Lstat(batch.local.root); errors.Is(err, os.ErrNotExist) {
					continue
				} else if err != nil {
					return err
				}
				if err := batch.local.validateBackendPath(); err != nil {
					return err
				}
				continue
			}
			if err := batch.local.withStoreLock(func() error {
				allowed := map[string]bool{filepath.Join(batch.local.root, "store.lock"): true}
				for _, payload := range batch.payloads {
					if payload.target != "" {
						allowed[payload.target] = true
					}
				}
				// Matching retries may reuse exactly their staged files. Unmatched
				// local evidence cannot silently become authoritative at cutover.
				if err := filepath.WalkDir(batch.local.root, func(path string, entry fs.DirEntry, walkErr error) error {
					if walkErr != nil {
						return walkErr
					}
					if entry.Type()&os.ModeSymlink != 0 {
						return errors.New("symlink in staged slot metadata")
					}
					if !entry.IsDir() && !allowed[path] {
						return fmt.Errorf("unmatched local evidence requires recovery: %s", path)
					}
					return ctx.Err()
				}); err != nil {
					return err
				}
				var entries []storeJournalEntry
				for _, payload := range batch.payloads {
					if payload.target == "" {
						continue
					}
					old, err := batch.local.readRetentionFile(payload.target)
					if err == nil {
						if !bytes.Equal(old, payload.raw) {
							return fmt.Errorf("conflicting staged slot payload %s", payload.target)
						}
					} else if !errors.Is(err, os.ErrNotExist) {
						return err
					}
					after := append([]byte(nil), payload.raw...)
					var expected *[]byte
					if err == nil {
						expected = &old
					}
					entries = append(entries, storeJournalEntry{Path: relativeStorePath(batch.local.root, payload.target), Expected: expected, After: &after})
				}
				if len(entries) == 0 {
					return nil
				}
				return batch.local.commitJournalLockedChecked(storeJournal{SchemaVersion: 1, Entries: entries}, ctx.Err)
			}); err != nil {
				return err
			}
		}
		next := manifest
		next.SchemaVersion = 2
		next.Generation++
		next.SlotRepositories = append(append([]string(nil), manifest.SlotRepositories...), repository.Identity.PrimaryRoot)
		sort.Strings(next.SlotRepositories)
		next.Threads = make([]ThreadAddress, 0, len(manifest.Threads))
		for _, address := range manifest.Threads {
			if !removeAddresses[address] {
				next.Threads = append(next.Threads, address)
			}
		}
		after, err := json.MarshalIndent(next, "", "  ")
		if err != nil {
			return err
		}
		after = append(after, '\n')
		var expected *[]byte
		if exists {
			expected = &manifestRaw
		}
		entries := []storeJournalEntry{{Path: relativeStorePath(s.root, s.manifestPath()), Expected: expected, After: &after}}
		for _, batch := range batches {
			for _, payload := range batch.payloads {
				before := append([]byte(nil), payload.raw...)
				entries = append(entries, storeJournalEntry{Path: relativeStorePath(s.root, payload.source), Expected: &before})
			}
		}
		return s.commitJournalLockedChecked(storeJournal{SchemaVersion: 1, Entries: entries}, ctx.Err)
	})
}
