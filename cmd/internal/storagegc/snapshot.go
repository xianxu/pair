package storagegc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

// WithReadLock never initializes storage. Before the first managed mutation
// there is no coordination inode; that snapshot can only report untracked
// evidence. It must never be used as authorization for collection.
func (c *Coordinator) WithReadLock(ctx context.Context, fn func(*Locked) error) (err error) {
	if err = ctx.Err(); err != nil {
		return err
	}
	dir := filepath.Join(c.Root, ".retention")
	err = checkDirectory(dir, false)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	fd := -1
	if err == nil {
		fd, err = unix.Open(filepath.Join(dir, "coordinator.lock"), unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil {
			defer unix.Close(fd)
			var st unix.Stat_t
			if err = unix.Fstat(fd, &st); err != nil {
				return err
			}
			if st.Mode&unix.S_IFMT != unix.S_IFREG {
				return errors.New("snapshot lock is not regular")
			}
			for {
				err = unix.Flock(fd, unix.LOCK_SH|unix.LOCK_NB)
				if err == nil {
					break
				}
				if !errors.Is(err, unix.EWOULDBLOCK) {
					return err
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(5 * time.Millisecond):
				}
			}
			defer func() { err = errors.Join(err, unix.Flock(fd, unix.LOCK_UN)) }()
		}
	}
	held := &Locked{coordinator: c, active: true, readOnly: true}
	defer func() { held.active = false }()
	return fn(held)
}

// Writable rejects preview tokens at cross-store mutation boundaries.
func (l *Locked) Writable(root string) bool { return l.Holds(root) && !l.readOnly }
