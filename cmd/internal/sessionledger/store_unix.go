//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package sessionledger

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

type OSRuntime struct{}

func (OSRuntime) MkdirAll(path string, mode os.FileMode) error { return os.MkdirAll(path, mode) }
func (OSRuntime) ReadFile(path string) ([]byte, error)         { return os.ReadFile(path) }
func (OSRuntime) OpenAppend(path string, mode os.FileMode) (AppendFile, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, mode)
}

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
	return osLock{file: file}, nil
}

func (OSRuntime) SyncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

type osLock struct{ file *os.File }

func (l osLock) Close() error {
	return errors.Join(unix.Flock(int(l.file.Fd()), unix.LOCK_UN), l.file.Close())
}
