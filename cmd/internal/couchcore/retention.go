package couchcore

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"golang.org/x/sys/unix"
)

// ArchiveRetention binds the grace clock to the exact archived record bytes.
// ClockError means that archive must be retained, never aged from file mtime.
type ArchiveRetention struct {
	Address    ThreadAddress
	ArchivedAt time.Time
	RecordHash string
	ClockError string
}
type archiveGrace struct {
	Version    int           `json:"version"`
	Address    ThreadAddress `json:"address"`
	ArchivedAt time.Time     `json:"archived_at"`
	RecordHash string        `json:"record_hash"`
}

// StoreRetentionSnapshot uses manifest membership even when records cannot decode.
type StoreRetentionSnapshot struct {
	Visible  []ThreadAddress
	Archives []ArchiveRetention
}

func NewCoordinatedThreadStore(namespace CouchNamespace, c *storagegc.Coordinator) (*ThreadStore, error) {
	if c == nil {
		return nil, errors.New("coordinated store needs retention coordinator")
	}
	if err := c.RegisterStore(context.Background(), namespace.Dir()); err != nil {
		return nil, err
	}
	s := NewThreadStore(namespace)
	s.coordinator = c
	return s, nil
}

func (s *ThreadStore) archiveGracePath(address ThreadAddress) string {
	return filepath.Join(s.root, "archive-grace", address.RepoScope, string(address.Tag)+".json")
}
func (s *ThreadStore) archiveTime() time.Time {
	if s.coordinator != nil && s.coordinator.Now != nil {
		return s.coordinator.Now().UTC()
	}
	return time.Now().UTC()
}
func (s *ThreadStore) archiveGraceBytes(address ThreadAddress, raw []byte) ([]byte, error) {
	at := s.archiveTime()
	if at.IsZero() {
		return nil, errors.New("archive timestamp is missing")
	}
	return json.Marshal(archiveGrace{Version: 1, Address: address, ArchivedAt: at, RecordHash: fmt.Sprintf("%x", sha256.Sum256(raw))})
}

func (s *ThreadStore) readArchiveGraceLocked(address ThreadAddress) (ArchiveRetention, error) {
	result := ArchiveRetention{Address: address}
	raw, err := s.readRetentionFile(s.archivePath(address))
	if err != nil {
		return result, err
	}
	result.RecordHash = fmt.Sprintf("%x", sha256.Sum256(raw))
	clockRaw, err := s.readRetentionFile(s.archiveGracePath(address))
	if err != nil {
		return result, err
	}
	var clock archiveGrace
	if err := strictThreadStoreJSON(clockRaw, &clock); err != nil {
		return result, err
	}
	if clock.Version != 1 || clock.Address != address || clock.RecordHash != result.RecordHash || clock.ArchivedAt.IsZero() || clock.ArchivedAt.After(s.archiveTime()) {
		return result, errors.New("invalid archive grace or archive identity mismatch")
	}
	result.ArchivedAt = clock.ArchivedAt
	return result, nil
}

// readRetentionFile rejects symlinks and unexpected types in metadata paths.
func (s *ThreadStore) readRetentionFile(path string) ([]byte, error) {
	rel, err := filepath.Rel(s.namespace.Dir(), path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, errors.New("metadata outside store")
	}
	cursor := s.namespace.Dir()
	parts := strings.Split(rel, string(filepath.Separator))
	for i, part := range parts {
		cursor = filepath.Join(cursor, part)
		st, err := os.Lstat(cursor)
		if err != nil {
			return nil, err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("symlink in retention metadata")
		}
		if i < len(parts)-1 {
			if !st.IsDir() {
				return nil, errors.New("retention parent is not a directory")
			}
		} else if !st.Mode().IsRegular() || st.Size() > 4<<20 {
			return nil, errors.New("invalid retention metadata file")
		}
	}
	return os.ReadFile(path)
}

// RetentionSnapshot never recovers or initializes a store. A pending journal is
// blocking evidence for preview; mutating callers recover before trying again.
func (s *ThreadStore) RetentionSnapshot(held *storagegc.Locked) (snapshot StoreRetentionSnapshot, err error) {
	if s.coordinator == nil || !held.Holds(s.coordinator.Root) {
		return snapshot, errors.New("retention snapshot requires this root's live lock")
	}
	if err := held.CheckContext(); err != nil {
		return snapshot, err
	}
	// An unused registered namespace contains no working set and needs no writes.
	if _, err := os.Lstat(s.root); errors.Is(err, os.ErrNotExist) {
		return snapshot, nil
	} else if err != nil {
		return snapshot, err
	}
	lock, err := s.retentionReadLock()
	if err != nil {
		return snapshot, retentionLockError(err)
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	if err := held.CheckContext(); err != nil {
		return snapshot, err
	}
	if _, err := os.Lstat(s.journalPath()); err == nil {
		return snapshot, errors.New("thread store recovery pending")
	} else if !errors.Is(err, os.ErrNotExist) {
		return snapshot, err
	}
	if _, err := s.readRetentionFile(s.manifestPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return snapshot, err
	}
	manifest, _, _, err := s.loadManifestLocked()
	if err != nil {
		return snapshot, err
	}
	snapshot.Visible = append([]ThreadAddress(nil), manifest.Threads...)
	archiveRoot := filepath.Join(s.root, "archive")
	err = filepath.WalkDir(archiveRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := held.CheckContext(); err != nil {
			return err
		}
		if errors.Is(walkErr, os.ErrNotExist) && path == archiveRoot {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("symlink in archive inventory")
		}
		if entry.IsDir() {
			return nil
		}
		relative, _ := filepath.Rel(archiveRoot, path)
		parts := strings.Split(relative, string(filepath.Separator))
		if len(parts) != 2 || !strings.HasSuffix(parts[1], ".json") {
			return fmt.Errorf("unknown archive entry %s", relative)
		}
		address := ThreadAddress{RepoScope: parts[0], Tag: ThreadTag(strings.TrimSuffix(parts[1], ".json"))}
		if err := validateThreadAddress(address); err != nil {
			return err
		}
		evidence, err := s.readArchiveGraceLocked(address)
		if err != nil {
			evidence.ClockError = err.Error()
		}
		snapshot.Archives = append(snapshot.Archives, evidence)
		return nil
	})
	sort.Slice(snapshot.Archives, func(i, j int) bool {
		a, b := snapshot.Archives[i].Address, snapshot.Archives[j].Address
		if a.RepoScope != b.RepoScope {
			return a.RepoScope < b.RepoScope
		}
		return a.Tag < b.Tag
	})
	return snapshot, err
}

// RestoreThread restores exact archived bytes, including unreadable records.
// Manifest publication precedes removal of archive grace in the same journal.
func (s *ThreadStore) RestoreThread(address ThreadAddress) error {
	if err := validateThreadAddress(address); err != nil {
		return err
	}
	return s.withLock(func() error {
		manifest, manifestRaw, exists, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		if manifestContains(manifest, address) {
			return &ThreadExistsError{Address: address}
		}
		if _, present, err := readOptionalFile(s.recordPath(address)); err != nil {
			return err
		} else if present {
			return &ThreadExistsError{Address: address}
		}
		raw, err := s.readRetentionFile(s.archivePath(address))
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %+v", ErrThreadNotFound, address)
		}
		if err != nil {
			return err
		}
		grace, graceExists, err := readOptionalFile(s.archiveGracePath(address))
		if err != nil {
			return err
		}
		next := manifest
		next.Generation++
		next.Threads = append(next.Threads, address)
		sortThreadAddresses(next.Threads)
		nextRaw, err := json.MarshalIndent(next, "", "  ")
		if err != nil {
			return err
		}
		nextRaw = append(nextRaw, '\n')
		var expectedManifest, expectedGrace *[]byte
		if exists {
			expectedManifest = &manifestRaw
		}
		if graceExists {
			expectedGrace = &grace
		}
		return s.commitJournalLocked(storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{
			{Path: relativeStorePath(s.root, s.recordPath(address)), After: &raw},
			{Path: relativeStorePath(s.root, s.manifestPath()), Expected: expectedManifest, After: &nextRaw},
			{Path: relativeStorePath(s.root, s.archivePath(address)), Expected: &raw},
			{Path: relativeStorePath(s.root, s.archiveGracePath(address)), Expected: expectedGrace},
		}})
	})
}

func (s *ThreadStore) retentionReadLock() (*threadStoreLock, error) {
	st, err := os.Lstat(s.root)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("unsafe thread store root")
	}
	fd, err := unix.Open(filepath.Join(s.root, "store.lock"), unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "store.lock")
	st, err = file.Stat()
	if err != nil || !st.Mode().IsRegular() {
		file.Close()
		return nil, errors.New("unsafe thread store lock")
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		file.Close()
		return nil, err
	}
	return &threadStoreLock{file: file}, nil
}

// ReadStoreRetention adapts a registered namespace for preview without
// registering it again or creating any files in the namespace.
func ReadStoreRetention(namespace CouchNamespace, coordinator *storagegc.Coordinator, held *storagegc.Locked) (StoreRetentionSnapshot, error) {
	if namespace.Dir() == "" || coordinator == nil {
		return StoreRetentionSnapshot{}, errors.New("retention reader needs namespace and coordinator")
	}
	store := NewThreadStore(namespace)
	store.coordinator = coordinator
	return store.RetentionSnapshot(held)
}

func (c *Couch) beginResumeRetention(ctx context.Context, address ThreadAddress, meaningful bool) (func(bool) error, error) {
	coordinator := c.Threads.coordinator
	if coordinator == nil {
		return func(bool) error { return nil }, nil
	}
	owner, err := artifactpath.NewStorageOwner(coordinator.Root, address.RepoScope, string(address.Tag))
	if err != nil {
		return nil, err
	}
	process, err := storagegc.CurrentProcessIdentity(os.Getpid())
	if err != nil {
		return nil, err
	}
	registration, err := coordinator.RegisterProcess(ctx, owner, process, "couch-resume")
	if err != nil {
		return nil, err
	}
	return func(success bool) (err error) {
		finalContext := context.Background()
		if success && meaningful {
			id, e := coordinator.BeginUse(finalContext, owner, process, "explicit-resume")
			if e == nil {
				e = coordinator.CompleteUse(finalContext, owner, id)
			}
			err = errors.Join(err, e)
		}
		return errors.Join(err, coordinator.ReleaseProcess(finalContext, owner, registration))
	}, nil
}

// OnboardArchiveGrace grants legacy archives a full grace interval starting at
// apply. Only selected addresses are considered; nil selects nothing. Existing
// grace (including malformed evidence) is never repaired or renewed here.
func (s *ThreadStore) OnboardArchiveGrace(ctx context.Context, held *storagegc.Locked, addresses []ThreadAddress) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.withRetentionWrite(held, func() error {
		manifest, _, _, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		for _, address := range addresses {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := validateThreadAddress(address); err != nil {
				return err
			}
			if manifestContains(manifest, address) {
				continue
			}
			raw, err := s.readRetentionFile(s.archivePath(address))
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			// Lstat distinguishes an absent clock from present but unreadable, invalid,
			// or symlinked evidence; only absence authorizes onboarding.
			if _, err := os.Lstat(s.archiveGracePath(address)); err == nil {
				continue
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			// Validate parent components before publishing through the store journal.
			if _, err := s.readRetentionFile(s.archiveGracePath(address)); !errors.Is(err, os.ErrNotExist) {
				if err != nil {
					return err
				}
				continue
			}
			grace, err := s.archiveGraceBytes(address, raw)
			if err != nil {
				return err
			}
			// The unchanged archive entry is an identity guard on every journal replay,
			// including a crash after the grace sidecar was already installed.
			if err := s.commitJournalLockedChecked(storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{
				{Path: relativeStorePath(s.root, s.archivePath(address)), Expected: &raw, After: &raw},
				{Path: relativeStorePath(s.root, s.archiveGracePath(address)), After: &grace},
			}}, held.CheckContext); err != nil {
				return err
			}
		}
		return nil
	})
}

func OnboardStoreArchiveGrace(ctx context.Context, namespace CouchNamespace, c *storagegc.Coordinator, held *storagegc.Locked, addresses []ThreadAddress) error {
	if namespace.Dir() == "" || c == nil {
		return errors.New("archive onboarding needs namespace and coordinator")
	}
	store := NewThreadStore(namespace)
	store.coordinator = c
	return store.OnboardArchiveGrace(ctx, held, addresses)
}

// RecoverStoreRetention is the apply-only pre-inventory journal recovery seam.
// Preview uses ReadStoreRetention and never enters this writable path.
func RecoverStoreRetention(ctx context.Context, namespace CouchNamespace, c *storagegc.Coordinator, held *storagegc.Locked) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if namespace.Dir() == "" || c == nil {
		return errors.New("archive recovery needs namespace and coordinator")
	}
	store := NewThreadStore(namespace)
	store.coordinator = c
	return store.withRetentionWrite(held, ctx.Err)
}

// Busy nested stores yield through the same maintenance scheduling contract as
// a busy Pair root; callers must not turn contention into permanent failure.
func retentionLockError(err error) error {
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return fmt.Errorf("Couch store busy: %w", storagegc.ErrCoordinatorBusy)
	}
	return err
}
