package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type LaunchFailureStorage interface {
	Read() ([]LaunchFailure, error)
	// Save honors ctx while waiting for admission; admitted publication finishes
	// atomically even if ctx is subsequently canceled.
	Save(context.Context, LaunchFailure) error
	Path() string
}

type LaunchFailureStore struct {
	path        string
	waitForLock func(context.Context) error
}

func NewLaunchFailureStore(namespace CouchNamespace) *LaunchFailureStore {
	return &LaunchFailureStore{path: filepath.Join(namespace.Dir(), "launch-diagnostics", "failures.json")}
}
func (s *LaunchFailureStore) Path() string { return s.path }
func (s *LaunchFailureStore) Read() ([]LaunchFailure, error) {
	if err := provisionSafePath(s.path); err != nil {
		return nil, err
	}
	fd, err := unix.Open(s.path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), s.path)
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("launch diagnostics is not a regular file")
	}
	raw, err := io.ReadAll(io.LimitReader(f, launchFailureEncodedLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > launchFailureEncodedLimit {
		return nil, fmt.Errorf("launch diagnostics exceeds size limit")
	}
	history, err := decodeLifecycleFailureHistory(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid launch diagnostics: %w", err)
	}
	return history, nil
}
func (s *LaunchFailureStore) Save(ctx context.Context, f LaunchFailure) error {
	if _, err := normalizeLaunchFailure(f); err != nil {
		return err
	}
	return s.update(ctx, func(old LaunchFailureHistory) (LaunchFailureHistory, bool, error) {
		next, err := old.Upsert(f)
		return next, true, err
	})
}
func (s *LaunchFailureStore) update(ctx context.Context, change func(LaunchFailureHistory) (LaunchFailureHistory, bool, error)) error {
	dir := filepath.Dir(s.path)
	if err := provisionMkdirAll(dir); err != nil {
		return err
	}
	lockpath := filepath.Join(dir, "store.lock")
	if err := provisionSafePath(lockpath); err != nil {
		return err
	}
	if info, err := os.Lstat(lockpath); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("unsafe launch diagnostics lock")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	wait := s.waitForLock
	if wait == nil {
		wait = waitForStoreLockContext
	}
	var lock *threadStoreLock
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("launch diagnostics lock wait: %w", err)
		}
		var err error
		lock, err = acquireLaunchFailureLock(lockpath)
		if err == nil {
			break
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			return err
		}
		if err := wait(ctx); err != nil {
			return fmt.Errorf("launch diagnostics lock wait: %w", err)
		}
	}
	defer lock.Close()
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("launch diagnostics lock wait: %w", err)
	}
	old, err := s.Read()
	if err != nil {
		return err
	}
	next, changed, err := change(LaunchFailureHistory(old))
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	raw, err := encodeLaunchFailureHistory(next)
	if err != nil {
		return err
	}
	// All diagnostic writers hold this lock, so these private stages cannot
	// belong to an active publication. Reclaim interrupted writes before retry.
	stale, err := filepath.Glob(filepath.Join(dir, ".launch-failure-*"))
	if err != nil {
		return err
	}
	for _, path := range stale {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsafe launch diagnostics staging file")
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return writeAtomicBytesWithPattern(s.path, raw, ".launch-failure-*")
}

// The private lock uses the existing unlock primitive, but refuses symlinks and
// nonregular files before flock so a damaged diagnostics directory cannot hang
// launch cleanup while opening a FIFO or redirect the lock to another store.
func acquireLaunchFailureLock(path string) (*threadStoreLock, error) {
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return nil, fmt.Errorf("unsafe launch diagnostics lock")
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		file.Close()
		return nil, err
	}
	return &threadStoreLock{file: file}, nil
}
