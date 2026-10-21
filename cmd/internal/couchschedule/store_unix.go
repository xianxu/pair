//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package couchschedule

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/osfs"
	"golang.org/x/sys/unix"
)

// openScheduleDir walks from the filesystem root without following any link.
// Callers retain the returned descriptor through all child IO/publication.
func openScheduleDir(ctx context.Context, path string, create bool) (*os.File, error) {
	if ctx == nil {
		return nil, errors.New("schedule IO requires context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("schedule path must be clean and absolute")
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	current := os.NewFile(uintptr(fd), "/")
	for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if part == "" {
			continue
		}
		if err := ctx.Err(); err != nil {
			current.Close()
			return nil, err
		}
		next, err := unix.Openat(int(current.Fd()), part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if errors.Is(err, unix.ENOENT) && create {
			if err = unix.Mkdirat(int(current.Fd()), part, 0700); err != nil && !errors.Is(err, unix.EEXIST) {
				current.Close()
				return nil, fmt.Errorf("schedule mkdir %s: %w", part, err)
			}
			if err = current.Sync(); err != nil {
				current.Close()
				return nil, fmt.Errorf("schedule parent sync %s: %w", part, err)
			}
			next, err = unix.Openat(int(current.Fd()), part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		}
		if err != nil {
			current.Close()
			return nil, fmt.Errorf("schedule directory %s: %w", part, err)
		}
		name := filepath.Join(current.Name(), part)
		current.Close()
		current = os.NewFile(uintptr(next), name)
	}
	return current, nil
}

func openScheduleChild(dir *os.File, name string, flags int) (*os.File, error) {
	fd, err := unix.Openat(int(dir.Fd()), name, flags|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0600)
	if err != nil {
		return nil, fmt.Errorf("schedule open %s: %w", name, err)
	}
	return os.NewFile(uintptr(fd), filepath.Join(dir.Name(), name)), nil
}

func readScheduleFile(ctx context.Context, dir *os.File, name string, max int64) ([]byte, error) {
	f, err := openScheduleChild(dir, name, unix.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return osfs.ReadOpenedRegularMetadata(ctx, f, max)
}

func lockSchedule(ctx context.Context, dir *os.File) (func(), error) {
	// Establish a single permanent inode explicitly: exclusive creation has
	// one winner and other callers open that existing inode without recreating it.
	f, err := openScheduleChild(dir, ".state.lock", unix.O_CREAT|unix.O_EXCL|unix.O_RDWR)
	if errors.Is(err, unix.EEXIST) {
		f, err = openScheduleChild(dir, ".state.lock", unix.O_RDWR)
	}
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() {
		f.Close()
		return nil, errors.New("schedule lock must be regular")
	}
	for {
		if err = ctx.Err(); err != nil {
			f.Close()
			return nil, err
		}
		err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return func() { unix.Flock(int(f.Fd()), unix.LOCK_UN); f.Close() }, nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			f.Close()
			return nil, err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			f.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// publishSchedule uses durablefile's write/sync/rename/directory-sync order.
// Its pathname API cannot retain the nofollow ancestry fence during rename;
// this focused publisher keeps every operation relative to the admitted fd.
func publishSchedule(ctx context.Context, dir *os.File, raw []byte, beforeDirSync func() error) error {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	name := ".state-" + hex.EncodeToString(random[:]) + ".tmp"
	f, err := openScheduleChild(dir, name, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY)
	if err != nil {
		return err
	}
	defer f.Close()
	defer unix.Unlinkat(int(dir.Fd()), name, 0)
	if _, err = f.Write(raw); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = unix.Renameat(int(dir.Fd()), name, int(dir.Fd()), "state.json"); err != nil {
		return fmt.Errorf("schedule rename %s: %w", name, err)
	}
	if beforeDirSync != nil {
		if err = beforeDirSync(); err != nil {
			return err
		}
	}
	return dir.Sync()
}
