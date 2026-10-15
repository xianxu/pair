package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/osfs"
	"golang.org/x/sys/unix"
)

// StoreLayout adapts the shared transaction writer to single-current slot
// membership. Global stores retain their manifest; local stores derive it.
type StoreLayout struct{ Local bool }

func newSlotThreadStore(namespace CouchNamespace, slot SlotIdentity) *ThreadStore {
	return &ThreadStore{namespace: namespace, root: LayoutOf(slot).Store(), slot: &slot, layout: StoreLayout{Local: true}}
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

func (s *ThreadStore) localMembershipContext(ctx context.Context) (threadManifest, []byte, bool, error) {
	manifest := threadManifest{SchemaVersion: 1, Threads: []ThreadAddress{}}
	raw, err := s.readRetentionFileContext(ctx, filepath.Join(s.root, "thread.json"))
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
	if s.slot == nil {
		return nil
	}
	if _, belongs, err := RecordCheckoutMembership(record, s.slot.RepoIdentity, s.slot.WorktreeRoot); err != nil || !belongs {
		return fmt.Errorf("slot record identity does not match its host checkout: %v", err)
	}
	for _, path := range []string{record.StartingPath, record.WorkingPath} {
		if err := retainedPathWithinCheckout(s.slot.WorktreeRoot, path); err != nil {
			return fmt.Errorf("slot record path does not match its host checkout: %w", err)
		}
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
	if err := s.slot.validateLocation(); err != nil {
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
	return s.readOptionalPayloadContext(context.Background(), path)
}
func (s *ThreadStore) readOptionalPayloadContext(ctx context.Context, path string) ([]byte, bool, error) {
	raw, err := s.readPayloadContext(ctx, path)
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
	return s.readPayloadContext(context.Background(), path)
}
func (s *ThreadStore) readPayloadContext(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.layout.Local {
		return s.readLocalPayloadContext(ctx, path, s.payloadLimit(path))
	}
	if s.inspection != nil {
		return s.readInspectionPayloadContext(ctx, path)
	}
	if s.readOnly {
		return osfs.ReadRegularMetadata(ctx, path, localPayloadLimit)
	}
	return os.ReadFile(path)
}

// Open each component relative to the retained parent descriptor. Unlike a
// Lstat followed by ReadFile, replacement with a symlink cannot redirect reads.
func (s *ThreadStore) readLocalPayloadContext(ctx context.Context, path string, limit int64) ([]byte, error) {
	ctx, cleanup := s.metadataReadContext(ctx)
	defer cleanup()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
		if err := ctx.Err(); err != nil {
			return nil, err
		}
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
	return osfs.ReadOpenedRegularMetadata(ctx, file, limit)
}
