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

func (s ResourceSnapshotStore) Load() (checkpoint, history []byte, err error) {
	checkpoint, err = s.read("checkpoint.json", ResourceCheckpointLimit, s.ValidateCheckpoint)
	if err != nil {
		return nil, nil, err
	}
	history, err = s.read("history.json", ResourceHistoryLimit, s.ValidateHistory)
	if err != nil {
		return nil, nil, err
	}
	return
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
	if !info.Mode().IsRegular() || info.Size() > int64(limit) {
		return nil, fmt.Errorf("unsafe or oversized resource snapshot %s", name)
	}
	raw, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > limit {
		return nil, fmt.Errorf("resource snapshot %s exceeds limit", name)
	}
	if err := validate(raw); err != nil {
		return nil, fmt.Errorf("resource snapshot %s: %w", name, err)
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
