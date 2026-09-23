package couchcore

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type storeJournal struct {
	SchemaVersion int                 `json:"schema_version"`
	Nonce         string              `json:"nonce,omitempty"`
	Entries       []storeJournalEntry `json:"entries"`
}

type storeJournalEntry struct {
	Path     string  `json:"path"`
	Expected *[]byte `json:"expected"`
	After    *[]byte `json:"after"`
}

func (s *ThreadStore) journalPath() string { return filepath.Join(s.root, "journal.json") }

func assignStoreJournalNonce(journal storeJournal) (storeJournal, error) {
	if journal.Nonce != "" {
		return journal, nil
	}
	raw, err := json.Marshal(journal.Entries)
	if err != nil {
		return storeJournal{}, err
	}
	digest := sha256.Sum256(raw)
	journal.Nonce = fmt.Sprintf("txn-%x", digest[:16])
	return journal, nil
}

func (s *ThreadStore) recoverStoreJournalLockedChecked(check func() error) error {
	if err := checkStoreContext(check); err != nil {
		return err
	}
	if err := s.clearStorePublicationLocked(check); err != nil {
		return err
	}
	raw, err := s.readPayload(s.journalPath())
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
		if err := s.applyJournalEntryChecked(entry, check); err != nil {
			return err
		}
	}
	if err := checkStoreContext(check); err != nil {
		return err
	}
	if err := os.Remove(s.journalPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear thread store journal: %w", err)
	}
	return syncDirectory(s.root)
}

func (s *ThreadStore) applyJournalEntryChecked(entry storeJournalEntry, check func() error) error {
	if err := checkStoreContext(check); err != nil {
		return err
	}
	if entry.Path == "" || filepath.IsAbs(entry.Path) || filepath.Clean(entry.Path) != entry.Path || entry.Path == ".." || len(entry.Path) >= 3 && entry.Path[:3] == "../" {
		return fmt.Errorf("unsafe thread store journal path %q", entry.Path)
	}
	target := filepath.Join(s.root, entry.Path)
	current, exists, err := s.readOptionalPayload(target)
	if err != nil {
		return err
	}
	if imageMatches(current, exists, entry.After) {
		return nil
	}
	if !imageMatches(current, exists, entry.Expected) {
		return fmt.Errorf("thread store journal target %q is neither expected-before nor exact after-image", entry.Path)
	}
	if err := checkStoreContext(check); err != nil {
		return err
	}
	if entry.After == nil {
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return syncDirectory(filepath.Dir(target))
	}
	return s.writeStoreAtomicLockedChecked(target, *entry.After, check)
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

// writeAtomicBytes also serves unlocked continuation materialization. Its old
// generic staging files are outside coordinated store cleanup authority.
func writeAtomicBytes(path string, raw []byte) error {
	return writeAtomicBytesWithPattern(path, raw, ".thread-store-*")
}

func writeAtomicBytesWithPattern(path string, raw []byte, pattern string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), pattern)
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

// Exactly one reserved staging pathname belongs to store-lock publications.
// Generic .thread-store-* files may have live unlocked writers and are not ours
// to sweep. Interrupted legacy generic staging remains outside GC authority.
func (s *ThreadStore) publicationPath() string {
	return filepath.Join(s.root, ".thread-store-publication")
}

func (s *ThreadStore) clearStorePublicationLocked(check func() error) error {
	if err := checkStoreContext(check); err != nil {
		return err
	}
	path := s.publicationPath()
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("unsafe thread store publication stage")
	}
	if err := checkStoreContext(check); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncDirectory(s.root)
}

// writeStoreAtomicLocked requires the existing exclusive store lock. A crash
// before journal publication leaves only this reclaimable stage; a crash during
// a target publication leaves the durable journal as replay authority.
func (s *ThreadStore) writeStoreAtomicLocked(path string, raw []byte) error {
	return s.writeStoreAtomicLockedChecked(path, raw, nil)
}

func (s *ThreadStore) writeStoreAtomicLockedChecked(path string, raw []byte, check func() error) (err error) {
	if s.readOnly {
		return errors.New("cannot write through a preview store")
	}
	if err := checkStoreContext(check); err != nil {
		return err
	}
	relative, e := filepath.Rel(s.root, path)
	if e != nil || relative == "." || relative == ".." || filepath.IsAbs(relative) || len(relative) > 3 && relative[:3] == "../" || path == s.publicationPath() {
		return errors.New("publication target is outside store payloads")
	}
	if s.layout.Local {
		if int64(len(raw)) > s.payloadLimit(path) {
			return errors.New("local metadata exceeds size limit")
		}
		if _, _, err := s.readOptionalPayload(path); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	stage := s.publicationPath()
	if err := checkStoreContext(check); err != nil {
		return err
	}
	file, err := os.OpenFile(stage, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	defer func() {
		if e := os.Remove(stage); e == nil {
			err = errors.Join(err, syncDirectory(s.root))
		} else if !errors.Is(e, os.ErrNotExist) {
			err = errors.Join(err, e)
		}
	}()
	if err := checkStoreContext(check); err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		return err
	}
	if err := checkStoreContext(check); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if s.hooks.AfterPublicationWrite != nil {
		if err := s.hooks.AfterPublicationWrite(path); err != nil {
			return err
		}
	}
	if err := checkStoreContext(check); err != nil {
		return err
	}
	if err := os.Rename(stage, path); err != nil {
		return err
	}
	targetDir := filepath.Dir(path)
	if targetDir == s.root {
		return syncDirectory(s.root)
	}
	return errors.Join(syncDirectory(targetDir), syncDirectory(s.root))
}

func checkStoreContext(check func() error) error {
	if check != nil {
		return check()
	}
	return nil
}
