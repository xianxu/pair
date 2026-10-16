package nativelock

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"golang.org/x/sys/unix"
)

// Handle holds a stable inode until Close. It is never inherited by native
// children. Close does not unlink the file or alter its diagnostic metadata.
// pair:m5-concept integration
type Handle struct {
	file *os.File
	once sync.Once
	err  error
}

// Acquire makes exactly one nonblocking exclusive flock attempt. The holder
// label is diagnostic only; acquisition and release are kernel decisions.
// pair:m5-concept integration
func Acquire(home string, key Key, holder string) (*Handle, error) {
	paths, err := artifactpath.ResolvePairCache(home)
	if err != nil {
		return nil, err
	}
	path, err := paths.NativeLock(key.agent, key.id)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create native lock directory: %w", err)
	}
	dirfd, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open native lock directory: %w", err)
	}
	defer unix.Close(dirfd)
	var dst unix.Stat_t
	if err := unix.Fstat(dirfd, &dst); err != nil {
		return nil, err
	}
	if dst.Uid != uint32(os.Geteuid()) || dst.Mode&0077 != 0 {
		return nil, fmt.Errorf("unsafe native lock directory owner or permissions")
	}
	fd, err := unix.Openat(dirfd, filepath.Base(path), unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0600)
	if err != nil {
		return nil, fmt.Errorf("open native lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	fail := func(err error) (*Handle, error) { file.Close(); return nil, err }
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return fail(err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != uint32(os.Geteuid()) || st.Mode&0777 != 0600 || st.Nlink != 1 {
		return fail(fmt.Errorf("unsafe native lock file: expected owner-only regular file"))
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			data, readErr := io.ReadAll(io.LimitReader(file, maxHolderBytes+1))
			label := "another Pair session"
			if readErr == nil {
				label = ParseHolder(data)
			}
			return fail(fmt.Errorf("conversation %s is in use by %s; Start fresh", key.id, label))
		}
		return fail(fmt.Errorf("acquire native lock: %w", err))
	}
	h := &Handle{file: file}
	if err := h.Relabel(holder); err != nil {
		return fail(err)
	}
	return h, nil
}

// Relabel rewrites the held lock's holder label. Only the owner writes it, and
// only after owning the lock; writing in place preserves the inode.
func (h *Handle) Relabel(holder string) error {
	if !validHolder(holder) {
		holder = "unlabelled"
	}
	// Truncate first: a contender reading in between sees an empty label, shown
	// as the generic holder, never a mix of the old and new labels.
	if err := h.file.Truncate(0); err != nil {
		return fmt.Errorf("truncate native lock holder: %w", err)
	}
	if _, err := h.file.WriteAt([]byte(holder), 0); err != nil {
		return fmt.Errorf("write native lock holder: %w", err)
	}
	return nil
}

func (h *Handle) Close() error {
	if h == nil {
		return nil
	}
	h.once.Do(func() { h.err = h.file.Close() })
	return h.err
}
