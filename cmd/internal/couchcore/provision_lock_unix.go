//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package couchcore

import (
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

var ErrHostCreationBusy = errors.New("workspace host creation is busy")

// Active or unknown ownership retains a short refusal. A released owner can
// leave a fork copy behind until exec; creationLeaseRetry bounds that backoff.
const hostCreationSettle = 100 * time.Millisecond

type HostCreationLease struct{ file *os.File }

func (l *HostCreationLease) File() *os.File { return l.file }

// Close releases only our descriptor. Inherited descriptors keep exclusion until
// their producers exit; an explicit LOCK_UN would release their lock too.
func (l *HostCreationLease) Close() error {
	_, err := l.file.WriteAt([]byte{'R'}, 0)
	return errors.Join(err, l.file.Close())
}

func AcquireHostCreationLease(commonDir string) (*HostCreationLease, error) {
	root, err := unix.Open(commonDir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open Git common directory: %w", err)
	}
	defer unix.Close(root)
	if err = unix.Mkdirat(root, CouchWorkspacesDir, 0700); err != nil && !errors.Is(err, unix.EEXIST) {
		return nil, err
	}
	dir, err := unix.Openat(root, CouchWorkspacesDir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open workspace metadata directory: %w", err)
	}
	defer unix.Close(dir)
	fd, err := unix.Openat(dir, "creation.lock", unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0600)
	if err != nil {
		return nil, fmt.Errorf("open creation lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), "creation.lock")
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		return nil, errors.New("creation lock must be a regular file")
	}
	started := time.Now()
	for attempt := 0; ; attempt++ {
		err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
		busy := errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN)
		if busy {
			var hint [1]byte
			n, readErr := file.ReadAt(hint[:], 0)
			released := readErr == nil && n == 1 && hint[0] == 'R'
			if delay := creationLeaseRetry(time.Since(started), attempt, released); delay > 0 {
				time.Sleep(delay)
				continue
			}
		}
		if err != nil {
			file.Close()
			if busy {
				return nil, ErrHostCreationBusy
			}
			return nil, err
		}
		break
	}
	// Advisory only: a stale R can add bounded waiting, but only flock admits.
	if _, err = file.WriteAt([]byte{'H'}, 0); err != nil {
		return nil, errors.Join(fmt.Errorf("mark creation lease held: %w", err), file.Close())
	}
	return &HostCreationLease{file: file}, nil
}
