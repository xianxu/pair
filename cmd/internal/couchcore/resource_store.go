package couchcore

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/durablefile"
	"golang.org/x/sys/unix"
)

const ResourceCheckpointLimit = 2 << 20
const ResourceHistoryLimit = 4 << 20

// ResourceSnapshotStore has one writer: the namespace supervisor's sampler. Separate
// snapshots let alert admission checkpoint before dispatch without rewriting
// history. Validators are mandatory and must check schema and semantic bounds;
// bytes never become ownership authority simply because JSON decodes.
type ResourceSnapshotStore struct {
	Dir                string
	ValidateCheckpoint func([]byte) error
	ValidateHistory    func([]byte) error
}

// ResourceSnapshotCorruptError means bytes from a safely opened regular file
// cannot be used. They may be replaced with fresh observations. Filesystem and
// unsafe-path failures deliberately do not carry this recovery permission.
type ResourceSnapshotCorruptError struct {
	Name string
	Err  error
}

func (e *ResourceSnapshotCorruptError) Error() string {
	return fmt.Sprintf("corrupt resource snapshot %s: %v", e.Name, e.Err)
}
func (e *ResourceSnapshotCorruptError) Unwrap() error { return e.Err }

// ResourceSnapshotLoadError preserves independent artifact failures. Load still
// returns the validated bytes of either healthy artifact.
type ResourceSnapshotLoadError struct{ Checkpoint, History error }

func (e *ResourceSnapshotLoadError) Error() string {
	return errors.Join(e.Checkpoint, e.History).Error()
}
func (e *ResourceSnapshotLoadError) Unwrap() []error {
	var out []error
	if e.Checkpoint != nil {
		out = append(out, e.Checkpoint)
	}
	if e.History != nil {
		out = append(out, e.History)
	}
	return out
}
func (s ResourceSnapshotStore) Load() (checkpoint, history []byte, err error) {
	checkpoint, cpErr := s.LoadCheckpoint()
	history, hErr := s.LoadHistory()
	if cpErr != nil || hErr != nil {
		err = &ResourceSnapshotLoadError{Checkpoint: cpErr, History: hErr}
	}
	return checkpoint, history, err
}
func (s ResourceSnapshotStore) LoadCheckpoint() ([]byte, error) {
	return s.read("checkpoint.json", ResourceCheckpointLimit, s.ValidateCheckpoint)
}
func (s ResourceSnapshotStore) LoadHistory() ([]byte, error) {
	return s.read("history.json", ResourceHistoryLimit, s.ValidateHistory)
}

func (s ResourceSnapshotStore) read(name string, limit int, validate func([]byte) error) ([]byte, error) {
	if validate == nil {
		return nil, errors.New("resource snapshot validator missing")
	}
	path := filepath.Join(s.Dir, name)
	if err := provisionSafePath(path); err != nil {
		return nil, err
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("unsafe resource snapshot %s", name)
	}
	if info.Size() > int64(limit) {
		return nil, &ResourceSnapshotCorruptError{Name: name, Err: errors.New("byte limit exceeded")}
	}
	raw, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > limit {
		return nil, &ResourceSnapshotCorruptError{Name: name, Err: errors.New("byte limit exceeded")}
	}
	if err := validate(raw); err != nil {
		return nil, &ResourceSnapshotCorruptError{Name: name, Err: err}
	}
	return raw, nil
}

func (s ResourceSnapshotStore) SaveCheckpoint(raw []byte) error {
	return s.save("checkpoint.json", ".checkpoint-stage", raw, ResourceCheckpointLimit, s.ValidateCheckpoint)
}
func (s ResourceSnapshotStore) SaveHistory(raw []byte) error {
	return s.save("history.json", ".history-stage", raw, ResourceHistoryLimit, s.ValidateHistory)
}
func (s ResourceSnapshotStore) save(name, stage string, raw []byte, limit int, validate func([]byte) error) error {
	if validate == nil {
		return errors.New("resource snapshot validator missing")
	}
	if len(raw) > limit {
		return fmt.Errorf("resource snapshot %s exceeds limit", name)
	}
	if err := validate(raw); err != nil {
		return err
	}
	if err := provisionMkdirAll(s.Dir); err != nil {
		return err
	}
	path := filepath.Join(s.Dir, name)
	if err := provisionSafePath(path); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return errors.New("unsafe resource snapshot")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// A fixed exclusive stage bounds crash residue; only this supervisor owns it.
	return durablefile.WriteAtomicStaged(path, raw, filepath.Join(s.Dir, stage))
}
