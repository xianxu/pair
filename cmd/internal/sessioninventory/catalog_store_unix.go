//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package sessioninventory

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

type CatalogOSRuntime struct{}

func (CatalogOSRuntime) MkdirAll(path string, mode os.FileMode) error { return os.MkdirAll(path, mode) }
func (CatalogOSRuntime) ReadFile(path string) ([]byte, error)         { return os.ReadFile(path) }
func (CatalogOSRuntime) CreateTemp(dir, pattern string) (CatalogFile, error) {
	return os.CreateTemp(dir, pattern)
}
func (CatalogOSRuntime) Remove(path string) error             { return os.Remove(path) }
func (CatalogOSRuntime) Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

func (CatalogOSRuntime) Lock(path string) (CatalogUnlocker, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	unix.CloseOnExec(int(file.Fd()))
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return catalogOSLock{file: file}, nil
}

func (CatalogOSRuntime) SyncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

type catalogOSLock struct{ file *os.File }

func (l catalogOSLock) Close() error {
	return errors.Join(unix.Flock(int(l.file.Fd()), unix.LOCK_UN), l.file.Close())
}
