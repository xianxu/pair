package zellijquery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// fileLease owns a stable flock descriptor. Close never unlinks the inode:
// another process may already have that inode open while waiting for its turn.
type fileLease struct {
	file *os.File
	once sync.Once
	err  error
}

func (l *fileLease) Close() error {
	l.once.Do(func() { l.err = errors.Join(unix.Flock(int(l.file.Fd()), unix.LOCK_UN), l.file.Close()) })
	return l.err
}

// AcquireFileLease serializes cooperating Pair processes on one canonical
// terminal key. The caller owns reobservation after acquisition and the handoff
// boundary at which Close is safe. Process death releases the descriptor too.
func AcquireFileLease(ctx context.Context, path string) (io.Closer, error) {
	if ctx == nil {
		return nil, errors.New("terminal lease requires a context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("terminal lease path must be canonical and absolute")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("terminal lease directory: %w", err)
	}
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0600)
	if err != nil {
		return nil, fmt.Errorf("open terminal lease: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	owned := false
	defer func() {
		if !owned {
			_ = file.Close()
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("terminal lease must be a regular file")
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			// Never admit on an inode that was replaced while this caller waited.
			current, err := os.Lstat(path)
			if err != nil {
				return nil, err
			}
			if !current.Mode().IsRegular() || !os.SameFile(info, current) {
				return nil, errors.New("terminal lease inode changed during acquisition")
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			owned = true
			return &fileLease{file: file}, nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) && !errors.Is(err, unix.EINTR) {
			return nil, fmt.Errorf("acquire terminal lease: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
