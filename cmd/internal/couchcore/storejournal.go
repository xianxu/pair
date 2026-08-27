package couchcore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type storeJournal struct {
	SchemaVersion int                 `json:"schema_version"`
	Entries       []storeJournalEntry `json:"entries"`
}

type storeJournalEntry struct {
	Path     string  `json:"path"`
	Expected *[]byte `json:"expected"`
	After    *[]byte `json:"after"`
}

func (s *ThreadStore) journalPath() string { return filepath.Join(s.root, "journal.json") }

func (s *ThreadStore) recoverStoreJournalLocked() error {
	raw, err := os.ReadFile(s.journalPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read thread store journal: %w", err)
	}
	var journal storeJournal
	if err := strictThreadStoreJSON(raw, &journal); err != nil {
		return fmt.Errorf("decode thread store journal: %w", err)
	}
	if journal.SchemaVersion != 1 || len(journal.Entries) == 0 {
		return fmt.Errorf("invalid thread store journal")
	}
	for _, entry := range journal.Entries {
		if err := s.applyJournalEntry(entry); err != nil {
			return err
		}
	}
	if err := os.Remove(s.journalPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear thread store journal: %w", err)
	}
	return syncDirectory(s.root)
}

func (s *ThreadStore) applyJournalEntry(entry storeJournalEntry) error {
	if entry.Path == "" || filepath.IsAbs(entry.Path) || filepath.Clean(entry.Path) != entry.Path || entry.Path == ".." || len(entry.Path) >= 3 && entry.Path[:3] == "../" {
		return fmt.Errorf("unsafe thread store journal path %q", entry.Path)
	}
	target := filepath.Join(s.root, entry.Path)
	current, exists, err := readOptionalFile(target)
	if err != nil {
		return err
	}
	if imageMatches(current, exists, entry.After) {
		return nil
	}
	if !imageMatches(current, exists, entry.Expected) {
		return fmt.Errorf("thread store journal target %q is neither expected-before nor exact after-image", entry.Path)
	}
	if entry.After == nil {
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return syncDirectory(filepath.Dir(target))
	}
	return writeAtomicBytes(target, *entry.After)
}

func imageMatches(current []byte, exists bool, image *[]byte) bool {
	if image == nil {
		return !exists
	}
	return exists && bytes.Equal(current, *image)
}

func readOptionalFile(path string) ([]byte, bool, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read %q: %w", path, err)
	}
	return raw, true, nil
}

func writeAtomicBytes(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".thread-store-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func strictThreadStoreJSON(raw []byte, target any) error {
	return strictjson.Decode(raw, target)
}
