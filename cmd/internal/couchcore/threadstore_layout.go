package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// StoreLayout adapts the shared transaction writer to single-current slot
// membership. Global stores retain their manifest; local stores derive it.
type StoreLayout struct{ Local bool }

func newSlotThreadStore(namespace CouchNamespace, slot SlotIdentity) *ThreadStore {
	return &ThreadStore{namespace: namespace, root: filepath.Join(slot.EnvironmentRoot, ".couch"), slot: &slot, layout: StoreLayout{Local: true}}
}

// journalEntries removes only the synthetic membership update. Every lifecycle
// transaction still uses the same journal for its record/preferences/history.
func (l StoreLayout) journalEntries(entries []storeJournalEntry) []storeJournalEntry {
	if !l.Local {
		return entries
	}
	result := make([]storeJournalEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Path != "manifest.json" {
			result = append(result, entry)
		}
	}
	return result
}

func (s *ThreadStore) localMembership() (threadManifest, []byte, bool, error) {
	manifest := threadManifest{SchemaVersion: 1, Threads: []ThreadAddress{}}
	raw, err := s.readRetentionFile(filepath.Join(s.root, "thread.json"))
	if errors.Is(err, os.ErrNotExist) {
		return manifest, nil, false, nil
	}
	if err != nil {
		return manifest, nil, false, err
	}
	var envelope struct {
		Address ThreadAddress `json:"address"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return manifest, nil, true, fmt.Errorf("slot current record: %w", err)
	}
	record, err := s.decodeThreadRaw(envelope.Address, raw)
	if err != nil {
		return manifest, nil, true, err
	}
	manifest.Threads = []ThreadAddress{record.Address}
	manifest.Generation = record.Revision
	return manifest, nil, true, nil
}

func (s *ThreadStore) validateLocalOrigin(record ThreadRecord) error {
	if s.slot != nil && (record.StartingPath != s.slot.WorktreeRoot || record.WorkingPath != s.slot.WorktreeRoot) {
		return errors.New("slot record path does not match its host checkout")
	}
	return nil
}

func (s *ThreadStore) validateBackendPath() error {
	if !s.layout.Local {
		return nil
	}
	if s.slot == nil {
		return errors.New("local store has no slot identity")
	}
	if err := s.slot.Validate(); err != nil {
		return err
	}
	for _, path := range []string{s.slot.EnvironmentRoot, s.slot.WorktreeRoot, s.root} {
		physical, err := filepath.EvalSymlinks(path)
		if errors.Is(err, os.ErrNotExist) && path == s.root {
			continue
		}
		if err != nil {
			return err
		}
		if physical != path {
			return fmt.Errorf("slot storage physical path changed: %s", path)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("slot storage is not a directory: %s", path)
		}
	}
	return nil
}

func (s *ThreadStore) localBackendMissing() (bool, error) {
	if s == nil {
		return false, nil
	}
	if !s.layout.Local {
		return false, nil
	}
	if err := s.validateBackendPath(); err != nil {
		return false, err
	}
	_, err := os.Lstat(s.root)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	return false, err
}
