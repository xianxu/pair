package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// readOptionalPayload retains ordinary-store compatibility while local metadata
// uses the same physical-path and size checks as discovery and retention.
func (s *ThreadStore) readOptionalPayload(path string) ([]byte, bool, error) {
	if !s.layout.Local {
		return readOptionalFile(path)
	}
	raw, err := s.readPayload(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return raw, err == nil, err
}

// Local journals may carry both before and after images for the 64 MiB
// migration budget, encoded as base64. Keep envelope headroom and enforce the
// same serialized bound on publication and recovery.
const localJournalLimit int64 = 256 << 20
const localPayloadLimit int64 = 4 << 20

func (s *ThreadStore) payloadLimit(path string) int64 {
	if path == s.journalPath() {
		return localJournalLimit
	}
	return localPayloadLimit
}

func (s *ThreadStore) readPayload(path string) ([]byte, error) {
	if !s.layout.Local {
		return os.ReadFile(path)
	}
	return s.readLocalPayload(path, s.payloadLimit(path))
}

// Open each component relative to the retained parent descriptor. Unlike a
// Lstat followed by ReadFile, replacement with a symlink cannot redirect reads.
func (s *ThreadStore) readLocalPayload(path string, limit int64) ([]byte, error) {
	if s.slot == nil {
		return nil, errors.New("local store has no slot identity")
	}
	storeRel, err := filepath.Rel(s.root, path)
	if err != nil || storeRel == "." || storeRel == ".." || filepath.IsAbs(storeRel) || strings.HasPrefix(storeRel, ".."+string(filepath.Separator)) {
		return nil, errors.New("metadata outside store")
	}
	rel, err := filepath.Rel(s.slot.EnvironmentRoot, path)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, errors.New("metadata outside store")
	}
	fd, err := unix.Open(s.slot.EnvironmentRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer func() { unix.Close(fd) }()
	parts := strings.Split(rel, string(filepath.Separator))
	for _, part := range parts[:len(parts)-1] {
		next, err := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			return nil, fmt.Errorf("unsafe metadata parent %s: %w", path, err)
		}
		unix.Close(fd)
		fd = next
	}
	fileFD, err := unix.Openat(fd, parts[len(parts)-1], unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open metadata %s: %w", path, err)
	}
	file := os.NewFile(uintptr(fileFD), path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("invalid local metadata file type or size")
	}
	raw, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, errors.New("local metadata exceeds size limit")
	}
	return raw, nil
}
