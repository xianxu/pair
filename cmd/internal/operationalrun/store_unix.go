package operationalrun

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

var ErrBusy = errors.New("operational task busy")

func safeDir(path string, create bool) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return errors.New("operational namespace must be absolute")
	}
	if path == string(filepath.Separator) {
		return nil
	}
	if e := safeDir(filepath.Dir(path), create); e != nil {
		return e
	}
	st, e := os.Lstat(path)
	if errors.Is(e, os.ErrNotExist) && create {
		if e = os.Mkdir(path, 0700); e != nil && !errors.Is(e, os.ErrExist) {
			return e
		}
		st, e = os.Lstat(path)
	}
	if e != nil {
		return e
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("unsafe operational directory")
	}
	return nil
}
func readFile(path string, limit int) ([]byte, error) {
	if e := safeDir(filepath.Dir(path), false); e != nil {
		return nil, e
	}
	fd, e := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() || st.Size() > int64(limit) {
		return nil, errors.New("unsafe or oversized operational file")
	}
	raw, e := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if len(raw) > limit {
		return nil, errors.New("operational file exceeds byte cap")
	}
	return raw, e
}
func taskLock(ctx context.Context, path string, create bool) (func(), error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if e := safeDir(path, create); e != nil {
		return nil, e
	}
	flags := unix.O_RDWR | unix.O_NOFOLLOW | unix.O_CLOEXEC | unix.O_NONBLOCK
	if create {
		flags |= unix.O_CREAT
	}
	fd, e := unix.Open(filepath.Join(path, ".runs.lock"), flags, 0600)
	if e != nil {
		return nil, e
	}
	close := func() { unix.Close(fd) }
	var st unix.Stat_t
	if e = unix.Fstat(fd, &st); e != nil || st.Mode&unix.S_IFMT != unix.S_IFREG {
		close()
		return nil, errors.New("unsafe operational lock")
	}
	if e = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); e != nil {
		close()
		if errors.Is(e, unix.EWOULDBLOCK) || errors.Is(e, unix.EAGAIN) {
			return nil, ErrBusy
		}
		return nil, e
	}
	if e = ctx.Err(); e != nil {
		close()
		return nil, e
	}
	return close, nil
}
