//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package couchcore

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

var ErrHostCreationBusy = errors.New("workspace host creation is busy")

type HostCreationLease struct{ file *os.File }

func (l *HostCreationLease) File() *os.File { return l.file }

// Close releases only our descriptor. Inherited descriptors keep exclusion until
// their producers exit; an explicit LOCK_UN would release their lock too.
func (l *HostCreationLease) Close() error { return l.file.Close() }

func AcquireHostCreationLease(commonDir string) (*HostCreationLease, error) {
	root, err := unix.Open(commonDir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open Git common directory: %w", err)
	}
	defer unix.Close(root)
	if err = unix.Mkdirat(root, "couch-workspaces", 0700); err != nil && !errors.Is(err, unix.EEXIST) {
		return nil, err
	}
	dir, err := unix.Openat(root, "couch-workspaces", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
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
	if err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return nil, ErrHostCreationBusy
		}
		return nil, err
	}
	return &HostCreationLease{file: file}, nil
}
