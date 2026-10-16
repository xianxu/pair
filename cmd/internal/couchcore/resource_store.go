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

const ResourceHistoryLimit = 4 << 20

// ResourceHistoryStore publishes only advisory aggregate history. One snapshot
// and one fixed exclusive stage bound its disk footprint to 8 MiB.
type ResourceHistoryStore struct{ Dir string }

func (s ResourceHistoryStore) Load() ([]byte, error) {
	path := filepath.Join(s.Dir, "history.json")
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
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("unsafe resource history")
	}
	if info.Size() > ResourceHistoryLimit {
		return nil, errors.New("resource history exceeds byte cap")
	}
	raw, err := io.ReadAll(io.LimitReader(file, ResourceHistoryLimit+1))
	if err != nil {
		return nil, err
	}
	if err := ValidateResourceHistory(raw); err != nil {
		return nil, fmt.Errorf("resource history: %w", err)
	}
	return raw, nil
}
func (s ResourceHistoryStore) Save(raw []byte) error {
	if err := ValidateResourceHistory(raw); err != nil {
		return err
	}
	if err := provisionMkdirAll(s.Dir); err != nil {
		return err
	}
	path := filepath.Join(s.Dir, "history.json")
	if err := provisionSafePath(path); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return errors.New("unsafe resource history")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return durablefile.WriteAtomicStaged(path, raw, filepath.Join(s.Dir, ".history-stage"))
}
