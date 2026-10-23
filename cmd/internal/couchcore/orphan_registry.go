package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
	"golang.org/x/sys/unix"
)

// leftRunningCap bounds the registry; the oldest entries leave first. An
// operator who never reaps keeps the newest evidence, never an unbounded file.
const leftRunningCap = 256

const leftRunningEncodedLimit = 1 << 20

const leftRunningSchemaVersion = 1

// LeftRunningRegistry is <namespace>/orphans.json: the processes teardown
// named in LeftRunning, kept so they stay attributable after their thread is
// gone (pair#695 M2). Teardown never kills them; `couch --reap-orphans
// --confirm` ends them by exact identity, and an entry whose identity is gone
// drops at the next sample.
type LeftRunningRegistry struct {
	path string
}

func NewLeftRunningRegistry(namespace CouchNamespace) *LeftRunningRegistry {
	return &LeftRunningRegistry{path: filepath.Join(namespace.Dir(), "orphans.json")}
}

func (r *LeftRunningRegistry) Path() string { return r.path }

func (r *LeftRunningRegistry) dir() string { return filepath.Dir(r.path) }

// leftRunningFile is the registry's envelope. Like every record born in this
// issue it is tolerant and preserving: members a newer build adds survive a
// rewrite by this one, at both levels (pair#695 §2a, BR-7).
type leftRunningFile struct {
	SchemaVersion int                        `json:"schema_version"`
	Orphans       []LeftRunningOrphan        `json:"orphans"`
	Unknown       map[string]json.RawMessage `json:"-"`
}

type leftRunningFileWire struct {
	SchemaVersion int                 `json:"schema_version"`
	Orphans       []LeftRunningOrphan `json:"orphans"`
}

type leftRunningOrphanWire struct {
	Origin    string             `json:"origin"`
	Thread    ThreadAddress      `json:"thread"`
	Process   leftRunningProcess `json:"process"`
	Command   string             `json:"command"`
	LeftAt    time.Time          `json:"left_at"`
	Operation string             `json:"operation,omitempty"`
}

// leftRunningProcess is an entry's process on disk: lowercase members, and
// tolerant and preserving like its parent.
type leftRunningProcess struct {
	PID      int                        `json:"pid"`
	Identity string                     `json:"identity"`
	Unknown  map[string]json.RawMessage `json:"-"`
}

type leftRunningProcessWire struct {
	PID      int    `json:"pid"`
	Identity string `json:"identity"`
}

var leftRunningProcessFields = strictjson.KnownFields(reflect.TypeOf(leftRunningProcessWire{}))

func (p *leftRunningProcess) UnmarshalJSON(raw []byte) error {
	var wire leftRunningProcessWire
	unknown, err := strictjson.DecodeKnown(raw, leftRunningProcessFields, &wire)
	if err != nil {
		return err
	}
	*p = leftRunningProcess{PID: wire.PID, Identity: wire.Identity}
	if len(unknown) > 0 {
		p.Unknown = unknown
	}
	return nil
}

func (p leftRunningProcess) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(leftRunningProcessWire{PID: p.PID, Identity: p.Identity})
	if err != nil {
		return nil, err
	}
	return strictjson.AppendMembers(raw, p.Unknown, leftRunningProcessFields)
}

var (
	leftRunningFileFields   = strictjson.KnownFields(reflect.TypeOf(leftRunningFileWire{}))
	leftRunningOrphanFields = strictjson.KnownFields(reflect.TypeOf(leftRunningOrphanWire{}))
)

func (f *leftRunningFile) UnmarshalJSON(raw []byte) error {
	var wire leftRunningFileWire
	unknown, err := strictjson.DecodeKnown(raw, leftRunningFileFields, &wire)
	if err != nil {
		return err
	}
	// A newer registry version is refused, never rewritten as this one: a
	// rewrite must not change a value it read (ARCH-EVOLVE).
	if wire.SchemaVersion != leftRunningSchemaVersion {
		return fmt.Errorf("unsupported orphan registry version %d; use a compatible Couch binary", wire.SchemaVersion)
	}
	*f = leftRunningFile{SchemaVersion: wire.SchemaVersion, Orphans: wire.Orphans}
	if len(unknown) > 0 {
		f.Unknown = unknown
	}
	return nil
}

func (f leftRunningFile) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(leftRunningFileWire{SchemaVersion: f.SchemaVersion, Orphans: f.Orphans})
	if err != nil {
		return nil, err
	}
	return strictjson.AppendMembers(raw, f.Unknown, leftRunningFileFields)
}

func (o *LeftRunningOrphan) UnmarshalJSON(raw []byte) error {
	var wire leftRunningOrphanWire
	unknown, err := strictjson.DecodeKnown(raw, leftRunningOrphanFields, &wire)
	if err != nil {
		return err
	}
	if wire.Process.PID <= 0 || wire.Process.Identity == "" {
		return fmt.Errorf("orphan registry entry has no exact process identity (pid %d)", wire.Process.PID)
	}
	*o = LeftRunningOrphan{Origin: wire.Origin, Thread: wire.Thread, Process: ProcessIdentity{PID: wire.Process.PID, Identity: wire.Process.Identity}, processUnknown: wire.Process.Unknown, Command: wire.Command, LeftAt: wire.LeftAt, Operation: wire.Operation}
	if len(unknown) > 0 {
		o.Unknown = unknown
	}
	return nil
}

func (o LeftRunningOrphan) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(leftRunningOrphanWire{Origin: o.Origin, Thread: o.Thread, Process: leftRunningProcess{PID: o.Process.PID, Identity: o.Process.Identity, Unknown: o.processUnknown}, Command: o.Command, LeftAt: o.LeftAt, Operation: o.Operation})
	if err != nil {
		return nil, err
	}
	return strictjson.AppendMembers(raw, o.Unknown, leftRunningOrphanFields)
}

// Read returns the registered entries; a missing registry is empty.
func (r *LeftRunningRegistry) Read() ([]LeftRunningOrphan, error) {
	file, err := r.read()
	return file.Orphans, err
}

func (r *LeftRunningRegistry) read() (leftRunningFile, error) {
	if err := provisionSafePath(r.path); err != nil {
		return leftRunningFile{}, err
	}
	fd, err := unix.Open(r.path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if errors.Is(err, os.ErrNotExist) {
		return leftRunningFile{SchemaVersion: leftRunningSchemaVersion}, nil
	}
	if err != nil {
		return leftRunningFile{}, err
	}
	f := os.NewFile(uintptr(fd), r.path)
	defer f.Close()
	if info, err := f.Stat(); err != nil {
		return leftRunningFile{}, err
	} else if !info.Mode().IsRegular() {
		return leftRunningFile{}, errors.New("orphan registry is not a regular file")
	}
	raw, err := io.ReadAll(io.LimitReader(f, leftRunningEncodedLimit+1))
	if err != nil {
		return leftRunningFile{}, err
	}
	if len(raw) > leftRunningEncodedLimit {
		return leftRunningFile{}, errors.New("orphan registry exceeds its size limit")
	}
	var file leftRunningFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return leftRunningFile{}, fmt.Errorf("invalid orphan registry: %w", err)
	}
	return file, nil
}

// Add registers entries, one per exact process identity (a re-registered
// identity keeps its first entry), and evicts the oldest past the cap.
func (r *LeftRunningRegistry) Add(ctx context.Context, entries []LeftRunningOrphan) error {
	if len(entries) == 0 {
		return nil
	}
	return r.update(ctx, func(old []LeftRunningOrphan) ([]LeftRunningOrphan, bool) {
		return mergeLeftRunning(old, entries, leftRunningCap), true
	})
}

// Prune drops the entries whose exact identity is gone; gone reports that
// for one identity.
func (r *LeftRunningRegistry) Prune(ctx context.Context, gone func(ProcessIdentity) bool) error {
	// The sampler prunes every pass; a registry with nothing gone (most
	// often an empty or missing one) costs one read and no lock or write.
	entries, err := r.Read()
	if err != nil {
		return err
	}
	stale := false
	for _, entry := range entries {
		stale = stale || gone(entry.Process)
	}
	if !stale {
		return nil
	}
	return r.update(ctx, func(old []LeftRunningOrphan) ([]LeftRunningOrphan, bool) {
		kept := old[:0:0]
		for _, entry := range old {
			if !gone(entry.Process) {
				kept = append(kept, entry)
			}
		}
		return kept, len(kept) != len(old)
	})
}

// mergeLeftRunning is the registry's pure rule: dedupe by identity, keep the
// newest cap entries, oldest first.
func mergeLeftRunning(old, added []LeftRunningOrphan, limit int) []LeftRunningOrphan {
	seen := map[ProcessIdentity]bool{}
	out := make([]LeftRunningOrphan, 0, len(old)+len(added))
	for _, entry := range append(append([]LeftRunningOrphan(nil), old...), added...) {
		if seen[entry.Process] {
			continue
		}
		seen[entry.Process] = true
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].LeftAt.Before(out[j].LeftAt) })
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

func (r *LeftRunningRegistry) update(ctx context.Context, change func([]LeftRunningOrphan) ([]LeftRunningOrphan, bool)) error {
	dir := r.dir()
	if err := provisionMkdirAll(dir); err != nil {
		return err
	}
	lockpath := filepath.Join(dir, "orphans.lock")
	if err := provisionSafePath(lockpath); err != nil {
		return err
	}
	var lock *threadStoreLock
	for {
		var err error
		if lock, err = acquireLaunchFailureLock(lockpath); err == nil {
			break
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			return err
		}
		if err := waitForStoreLockContext(ctx); err != nil {
			return fmt.Errorf("orphan registry lock wait: %w", err)
		}
	}
	defer lock.Close()
	file, err := r.read()
	if err != nil {
		return err
	}
	next, changed := change(file.Orphans)
	if !changed {
		return nil
	}
	file.Orphans = next
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomicBytesWithPattern(r.path, append(raw, '\n'), ".orphans-*")
}
