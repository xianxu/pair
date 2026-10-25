//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

// onStoreLockContended observes each contended poll of a foreground writer,
// so a test releases a held lock only once the writer is waiting. Tests only.
var onStoreLockContended func()

// acquireThreadStoreLockWithin takes store.lock for a foreground writer that
// already holds the retention coordinator (#827). Preview reads take
// store.lock outside the coordinator, so a writer that refused at the first
// EWOULDBLOCK failed whenever a sibling's routing read or a switcher refresh
// overlapped it -- the parks a full quit then left behind. It polls like
// retentionReadLock, within wait and while check passes, and past wait
// returns the typed ErrThreadStoreBusy -- or, for a wait the caller's deadline
// cut short, that deadline (pair#854).
func acquireThreadStoreLockWithin(root string, wait storeWait, check func() error) (*threadStoreLock, error) {
	deadline := time.Now().Add(wait.d)
	for {
		lock, err := acquireThreadStoreLockMode(root, true)
		if err == nil || !(errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN)) {
			return lock, err
		}
		if err := check(); err != nil {
			return nil, err
		}
		if !time.Now().Before(deadline) {
			if wait.deadline {
				return nil, fmt.Errorf("lock thread store: %w", context.DeadlineExceeded)
			}
			return nil, fmt.Errorf("lock thread store: %w", ErrThreadStoreBusy)
		}
		if onStoreLockContended != nil {
			onStoreLockContended()
		}
		time.Sleep(storeReadLockPoll)
	}
}

func acquireThreadStoreLockMode(root string, nonblocking bool) (*threadStoreLock, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create thread store root: %w", err)
	}
	path := filepath.Join(root, "store.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open thread store lock: %w", err)
	}
	unix.CloseOnExec(int(file.Fd()))
	mode := unix.LOCK_EX
	if nonblocking {
		mode |= unix.LOCK_NB
	}
	if err := unix.Flock(int(file.Fd()), mode); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock thread store: %w", err)
	}
	return &threadStoreLock{file: file}, nil
}

func unlockThreadStoreFile(file *os.File) error {
	if file == nil {
		return nil
	}
	return errors.Join(unix.Flock(int(file.Fd()), unix.LOCK_UN), file.Close())
}

// waitForStoreLockContext waits between nonblocking admission attempts.
// A cancellable wake avoids leaving a goroutine blocked in flock after return.
func waitForStoreLockContext(ctx context.Context) error {
	timer := time.NewTimer(10 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
