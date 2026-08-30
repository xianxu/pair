//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package pairlifecycle

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// OSRuntime is the production lifecycle file and advisory-lock runtime.
type OSRuntime struct{}

func (OSRuntime) MkdirAll(path string, mode os.FileMode) error { return os.MkdirAll(path, mode) }
func (OSRuntime) ReadFile(path string) ([]byte, error)         { return os.ReadFile(path) }
func (OSRuntime) CreateTemp(dir, pattern string) (StoreFile, error) {
	return os.CreateTemp(dir, pattern)
}
func (OSRuntime) Remove(path string) error             { return os.Remove(path) }
func (OSRuntime) Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

func (OSRuntime) Lock(path string) (Unlocker, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	unix.CloseOnExec(int(file.Fd()))
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return &osLock{file: file}, nil
}

func (OSRuntime) SyncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

type osLock struct{ file *os.File }

func (l *osLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	file := l.file
	l.file = nil
	return errors.Join(unix.Flock(int(file.Fd()), unix.LOCK_UN), file.Close())
}
