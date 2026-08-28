package pairlog

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

// File is the durable temporary-file surface used by Store.
type File interface {
	io.Writer
	Chmod(os.FileMode) error
	Close() error
	Name() string
	Sync() error
}

// Runtime contains every filesystem effect in the session-log transaction.
type Runtime interface {
	MkdirAll(string, os.FileMode) error
	Lock(string) (io.Closer, error)
	ReadFile(string) ([]byte, error)
	CreateTemp(string, string) (File, error)
	Remove(string) error
	Rename(string, string) error
	SyncDirectory(string) error
}

// SessionLogStore serializes and atomically publishes Pair's existing markdown log.
// pair:155-concept integration new M2
type SessionLogStore struct{ Runtime Runtime }

// PersistSessionLog uses the production filesystem runtime.
func PersistSessionLog(path string, body []byte, now time.Time) error {
	return (SessionLogStore{Runtime: OSRuntime{}}).Persist(path, body, now)
}

func (s SessionLogStore) Persist(path string, body []byte, now time.Time) (err error) {
	if path == "" {
		return errors.New("session log path is empty")
	}
	if s.Runtime == nil {
		return errors.New("session log runtime is nil")
	}
	dir := filepath.Dir(path)
	if err := s.Runtime.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create session log directory: %w", err)
	}
	lock, err := s.Runtime.Lock(path + ".lock")
	if err != nil {
		return fmt.Errorf("lock session log: %w", err)
	}
	defer func() { err = errors.Join(err, lock.Close()) }()

	existing, err := s.Runtime.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		existing, err = nil, nil
	}
	if err != nil {
		return fmt.Errorf("read session log: %w", err)
	}
	entry := []byte("## " + now.Format("2006-01-02 15:04:05") + "\n\n")
	entry = append(entry, body...)
	entry = append(entry, []byte("\n\n---\n\n")...)
	contents := make([]byte, 0, len(existing)+len(entry))
	contents = append(contents, existing...)
	contents = append(contents, entry...)

	tmp, err := s.Runtime.CreateTemp(dir, ".pair-session-log-*")
	if err != nil {
		return fmt.Errorf("create session log replacement: %w", err)
	}
	tmpName := tmp.Name()
	defer s.Runtime.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod session log replacement: %w", err)
	}
	if err := writeFull(tmp, contents); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write session log replacement: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync session log replacement: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close session log replacement: %w", err)
	}
	if err := s.Runtime.Rename(tmpName, path); err != nil {
		return fmt.Errorf("publish session log replacement: %w", err)
	}
	if err := s.Runtime.SyncDirectory(dir); err != nil {
		return fmt.Errorf("sync session log directory: %w", err)
	}
	return nil
}

func writeFull(w io.Writer, content []byte) error {
	for len(content) > 0 {
		n, err := w.Write(content)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		content = content[n:]
	}
	return nil
}

// OSRuntime is the production durable filesystem implementation.
type OSRuntime struct{}

func (OSRuntime) MkdirAll(path string, mode os.FileMode) error { return os.MkdirAll(path, mode) }
func (OSRuntime) ReadFile(path string) ([]byte, error)         { return os.ReadFile(path) }
func (OSRuntime) CreateTemp(dir, pattern string) (File, error) { return os.CreateTemp(dir, pattern) }
func (OSRuntime) Remove(path string) error                     { return os.Remove(path) }
func (OSRuntime) Rename(oldPath, newPath string) error         { return os.Rename(oldPath, newPath) }

func (OSRuntime) Lock(path string) (io.Closer, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	unix.CloseOnExec(int(file.Fd()))
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return fileLock{file}, nil
}

func (OSRuntime) SyncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

type fileLock struct{ file *os.File }

func (l fileLock) Close() error {
	return errors.Join(unix.Flock(int(l.file.Fd()), unix.LOCK_UN), l.file.Close())
}
