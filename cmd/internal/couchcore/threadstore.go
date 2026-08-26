package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

var ErrThreadNotFound = errors.New("thread not found")

type ThreadExistsError struct{ Address ThreadAddress }

func (e *ThreadExistsError) Error() string {
	return fmt.Sprintf("thread already exists: %+v", e.Address)
}

type ThreadRevisionError struct {
	Address ThreadAddress
	Want    uint64
	Got     uint64
}

func (e *ThreadRevisionError) Error() string {
	return fmt.Sprintf("thread revision conflict for %+v: expected %d, found %d", e.Address, e.Want, e.Got)
}

type threadManifest struct {
	SchemaVersion int             `json:"schema_version"`
	Generation    uint64          `json:"generation"`
	Threads       []ThreadAddress `json:"threads"`
	LegacyCutover bool            `json:"legacy_cutover,omitempty"`
}

type ThreadStore struct {
	namespace CouchNamespace
	root      string
	hooks     threadStoreHooks
}

func NewThreadStore(namespace CouchNamespace) *ThreadStore {
	return &ThreadStore{namespace: namespace, root: filepath.Join(namespace.Dir(), "threadstore")}
}

type threadStoreHooks struct {
	AfterJournal func() error
	AfterTarget  func(int) error
}

func newThreadStoreWithHooks(namespace CouchNamespace, hooks threadStoreHooks) *ThreadStore {
	store := NewThreadStore(namespace)
	store.hooks = hooks
	return store
}

func (s *ThreadStore) manifestPath() string { return filepath.Join(s.root, "manifest.json") }

func (s *ThreadStore) recordPath(address ThreadAddress) string {
	return filepath.Join(s.root, "records", address.RepoScope, string(address.Tag)+".json")
}

func (s *ThreadStore) withLock(fn func() error) (err error) {
	if s == nil || s.namespace.Dir() == "" {
		return errors.New("thread store has no namespace")
	}
	lock, err := acquireThreadStoreLock(s.root)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	if err := s.recoverStoreJournalLocked(); err != nil {
		return err
	}
	return fn()
}

func (s *ThreadStore) RecoverStoreJournal() (err error) {
	if s == nil || s.namespace.Dir() == "" {
		return errors.New("thread store has no namespace")
	}
	lock, err := acquireThreadStoreLock(s.root)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	return s.recoverStoreJournalLocked()
}

func (s *ThreadStore) CreateThread(record ThreadRecord) (ThreadRecord, error) {
	record = cloneThreadRecord(record)
	if err := validateThreadAddress(record.Address); err != nil {
		return ThreadRecord{}, err
	}
	var created ThreadRecord
	err := s.withLock(func() error {
		manifest, manifestRaw, manifestExists, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		if _, exists, err := readOptionalFile(s.recordPath(record.Address)); err != nil {
			return err
		} else if exists || manifestContains(manifest, record.Address) {
			return &ThreadExistsError{Address: record.Address}
		}
		record.ClaimGeneration = manifest.Generation + 1
		if err := ValidateThreadRecord(record); err != nil {
			return err
		}
		recordRaw, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			return err
		}
		recordRaw = append(recordRaw, '\n')
		nextManifest := manifest
		nextManifest.SchemaVersion = 1
		nextManifest.Generation++
		nextManifest.Threads = append(nextManifest.Threads, record.Address)
		sortThreadAddresses(nextManifest.Threads)
		nextManifestRaw, err := json.MarshalIndent(nextManifest, "", "  ")
		if err != nil {
			return err
		}
		nextManifestRaw = append(nextManifestRaw, '\n')
		var expectedManifest *[]byte
		if manifestExists {
			copy := append([]byte{}, manifestRaw...)
			expectedManifest = &copy
		}
		afterRecord := append([]byte{}, recordRaw...)
		afterManifest := append([]byte{}, nextManifestRaw...)
		journal := storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{
			{Path: relativeStorePath(s.root, s.recordPath(record.Address)), After: &afterRecord},
			{Path: relativeStorePath(s.root, s.manifestPath()), Expected: expectedManifest, After: &afterManifest},
		}}
		if err := s.commitJournalLocked(journal); err != nil {
			return err
		}
		created = cloneThreadRecord(record)
		return nil
	})
	return created, err
}

func (s *ThreadStore) GetThread(address ThreadAddress) (ThreadRecord, error) {
	if err := validateThreadAddress(address); err != nil {
		return ThreadRecord{}, err
	}
	var result ThreadRecord
	err := s.withLock(func() error {
		record, err := s.readThreadLocked(address)
		if err != nil {
			return err
		}
		result = cloneThreadRecord(record)
		return nil
	})
	return result, err
}

func (s *ThreadStore) UpdateExistingThread(address ThreadAddress, expectedRevision uint64, mutate func(*ThreadRecord) error) (ThreadRecord, error) {
	if err := validateThreadAddress(address); err != nil {
		return ThreadRecord{}, err
	}
	if mutate == nil {
		return ThreadRecord{}, errors.New("thread update has nil mutation")
	}
	var result ThreadRecord
	err := s.withLock(func() error {
		current, err := s.readThreadLocked(address)
		if err != nil {
			return err
		}
		if current.Revision != expectedRevision {
			return &ThreadRevisionError{Address: address, Want: expectedRevision, Got: current.Revision}
		}
		next := cloneThreadRecord(current)
		if err := mutate(&next); err != nil {
			return err
		}
		if next.Address != current.Address || next.SchemaVersion != current.SchemaVersion || next.StartingPath != current.StartingPath || next.CreatedAt != current.CreatedAt || next.ClaimGeneration != current.ClaimGeneration {
			return errors.New("thread update changed immutable identity or origin fields")
		}
		next.Revision++
		if err := ValidateThreadRecord(next); err != nil {
			return err
		}
		raw, err := json.MarshalIndent(next, "", "  ")
		if err != nil {
			return err
		}
		if err := writeAtomicBytes(s.recordPath(address), append(raw, '\n')); err != nil {
			return err
		}
		result = cloneThreadRecord(next)
		return nil
	})
	return result, err
}

func (s *ThreadStore) ManifestGeneration() (uint64, error) {
	var generation uint64
	err := s.withLock(func() error {
		manifest, _, _, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		generation = manifest.Generation
		return nil
	})
	return generation, err
}

// CutoverLegacyActors imports the old tree-keyed registry as one journaled
// membership mutation. Co-tenants remain conservative unknown incarnations of
// one legacy thread; identical display tags in different repo scopes remain
// separate composite records.
func (s *ThreadStore) CutoverLegacyActors(actors []ActorRecord) error {
	return s.withLock(func() error {
		manifest, manifestRaw, manifestExists, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		if manifest.LegacyCutover {
			return nil
		}
		grouped := map[ThreadAddress]ThreadRecord{}
		for _, actor := range actors {
			address, err := legacyThreadAddress(actor)
			if err != nil {
				return err
			}
			if actor.StartedAt.IsZero() {
				return fmt.Errorf("legacy actor %q has no start time", actor.ID)
			}
			record, ok := grouped[address]
			if !ok {
				path := string(actor.Args.Worktree)
				record = ThreadRecord{
					SchemaVersion:   ThreadSchemaVersion,
					Address:         address,
					StartingPath:    path,
					WorkingPath:     path,
					CreatedAt:       actor.StartedAt,
					Revision:        1,
					ClaimGeneration: manifest.Generation + 1,
				}
			}
			if actor.StartedAt.Before(record.CreatedAt) {
				record.CreatedAt = actor.StartedAt
			}
			record.Incarnations = append(record.Incarnations, ThreadIncarnation{
				LegacyActorID: actor.ID,
				PID:           actor.PID,
				Identity:      actor.Identity,
				State:         IncarnationUnknown,
				StartedAt:     actor.StartedAt,
			})
			grouped[address] = record
		}

		addresses := make([]ThreadAddress, 0, len(grouped))
		for address := range grouped {
			addresses = append(addresses, address)
		}
		sortThreadAddresses(addresses)
		entries := make([]storeJournalEntry, 0, len(addresses)+1)
		for _, address := range addresses {
			record := grouped[address]
			if err := ValidateThreadRecord(record); err != nil {
				return fmt.Errorf("legacy thread %+v: %w", address, err)
			}
			if _, exists, err := readOptionalFile(s.recordPath(address)); err != nil {
				return err
			} else if exists || manifestContains(manifest, address) {
				return fmt.Errorf("legacy cutover collides with existing thread %+v", address)
			}
			raw, err := json.MarshalIndent(record, "", "  ")
			if err != nil {
				return err
			}
			after := append(raw, '\n')
			entries = append(entries, storeJournalEntry{Path: relativeStorePath(s.root, s.recordPath(address)), After: &after})
		}
		nextManifest := manifest
		nextManifest.SchemaVersion = 1
		nextManifest.Generation++
		nextManifest.LegacyCutover = true
		nextManifest.Threads = append(nextManifest.Threads, addresses...)
		sortThreadAddresses(nextManifest.Threads)
		nextRaw, err := json.MarshalIndent(nextManifest, "", "  ")
		if err != nil {
			return err
		}
		afterManifest := append(nextRaw, '\n')
		var expectedManifest *[]byte
		if manifestExists {
			copy := append([]byte{}, manifestRaw...)
			expectedManifest = &copy
		}
		entries = append(entries, storeJournalEntry{Path: relativeStorePath(s.root, s.manifestPath()), Expected: expectedManifest, After: &afterManifest})
		return s.commitJournalLocked(storeJournal{SchemaVersion: 1, Entries: entries})
	})
}

func legacyThreadAddress(actor ActorRecord) (ThreadAddress, error) {
	if actor.Args.Worktree == "" {
		return ThreadAddress{}, errors.New("legacy actor has no worktree")
	}
	scope, err := launcher.ResolveRepoScope(string(actor.Args.Worktree))
	if err != nil {
		return ThreadAddress{}, fmt.Errorf("resolve legacy repo scope: %w", err)
	}
	return ThreadAddress{RepoScope: scope.Key, Tag: ThreadTag(launcher.DefaultTag(string(actor.Args.Worktree)))}, nil
}

func sortThreadAddresses(addresses []ThreadAddress) {
	sort.Slice(addresses, func(i, j int) bool {
		if addresses[i].RepoScope != addresses[j].RepoScope {
			return addresses[i].RepoScope < addresses[j].RepoScope
		}
		return addresses[i].Tag < addresses[j].Tag
	})
}

func (s *ThreadStore) readThreadLocked(address ThreadAddress) (ThreadRecord, error) {
	raw, err := os.ReadFile(s.recordPath(address))
	if errors.Is(err, os.ErrNotExist) {
		return ThreadRecord{}, fmt.Errorf("%w: %+v", ErrThreadNotFound, address)
	}
	if err != nil {
		return ThreadRecord{}, err
	}
	var record ThreadRecord
	if err := strictThreadStoreJSON(raw, &record); err != nil {
		return ThreadRecord{}, err
	}
	if err := ValidateThreadRecord(record); err != nil {
		return ThreadRecord{}, err
	}
	if record.ClaimGeneration == 0 {
		return ThreadRecord{}, errors.New("stored thread has zero claim generation")
	}
	if record.Address != address {
		return ThreadRecord{}, errors.New("thread record path/address mismatch")
	}
	return record, nil
}

func (s *ThreadStore) loadManifestLocked() (threadManifest, []byte, bool, error) {
	raw, exists, err := readOptionalFile(s.manifestPath())
	if err != nil {
		return threadManifest{}, nil, false, err
	}
	if !exists {
		return threadManifest{SchemaVersion: 1, Threads: []ThreadAddress{}}, nil, false, nil
	}
	var manifest threadManifest
	if err := strictThreadStoreJSON(raw, &manifest); err != nil {
		return threadManifest{}, nil, true, err
	}
	if manifest.SchemaVersion != 1 || manifest.Threads == nil {
		return threadManifest{}, nil, true, errors.New("invalid thread store manifest")
	}
	seen := map[ThreadAddress]bool{}
	for _, address := range manifest.Threads {
		if err := validateThreadAddress(address); err != nil {
			return threadManifest{}, nil, true, fmt.Errorf("invalid manifest address: %w", err)
		}
		if seen[address] {
			return threadManifest{}, nil, true, fmt.Errorf("duplicate manifest address %+v", address)
		}
		seen[address] = true
	}
	return manifest, raw, true, nil
}

func manifestContains(manifest threadManifest, address ThreadAddress) bool {
	for _, candidate := range manifest.Threads {
		if candidate == address {
			return true
		}
	}
	return false
}

func relativeStorePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		panic(err)
	}
	return rel
}

func (s *ThreadStore) commitJournalLocked(journal storeJournal) error {
	raw, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomicBytes(s.journalPath(), append(raw, '\n')); err != nil {
		return err
	}
	if s.hooks.AfterJournal != nil {
		if err := s.hooks.AfterJournal(); err != nil {
			return err
		}
	}
	for i, entry := range journal.Entries {
		if err := s.applyJournalEntry(entry); err != nil {
			return err
		}
		if s.hooks.AfterTarget != nil {
			if err := s.hooks.AfterTarget(i); err != nil {
				return err
			}
		}
	}
	if err := os.Remove(s.journalPath()); err != nil {
		return err
	}
	return syncDirectory(s.root)
}
