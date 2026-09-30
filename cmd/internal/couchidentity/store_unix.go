//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package couchidentity

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"time"
)

type allocationLock struct{ file *os.File }

func (l *allocationLock) Close() error {
	return errors.Join(unix.Flock(int(l.file.Fd()), unix.LOCK_UN), l.file.Close())
}
func openRegular(path string, flags int) (*os.File, error) {
	fd, e := unix.Open(path, flags|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0600)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), path)
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() {
		f.Close()
		if e != nil {
			return nil, e
		}
		return nil, errors.New("identity authority must be a regular file")
	}
	return f, nil
}
func acquireLock(ctx context.Context, path string) (*allocationLock, error) {
	f, e := openRegular(path, unix.O_CREAT|unix.O_RDWR)
	if e != nil {
		return nil, e
	}
	for {
		if e = ctx.Err(); e != nil {
			f.Close()
			return nil, e
		}
		e = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if e == nil {
			return &allocationLock{f}, nil
		}
		if !errors.Is(e, unix.EWOULDBLOCK) && !errors.Is(e, unix.EAGAIN) {
			f.Close()
			return nil, e
		}
		select {
		case <-ctx.Done():
			f.Close()
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}
