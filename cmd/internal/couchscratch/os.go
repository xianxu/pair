package couchscratch

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
)

// ReadProbe checks device identity and available space without creating paths.
func ReadProbe(root string) Probe {
	p := Probe{Root: root}
	var a, b unix.Stat_t
	if err := unix.Stat(root, &a); err != nil {
		p.Err = err
		return p
	}
	if err := unix.Stat(filepath.Dir(root), &b); err != nil {
		p.Err = err
		return p
	}
	p.Mounted = a.Dev != b.Dev
	if !p.Mounted {
		return p
	}
	var fs unix.Statfs_t
	if err := unix.Statfs(root, &fs); err != nil {
		p.Err = err
		return p
	}
	p.FreeBytes = uint64(fs.Bavail) * uint64(fs.Bsize)
	return p
}

// openDir opens a directory relative to a pinned descriptor without following a final symlink.
func openDir(parent int, name string) (int, error) {
	return unix.Openat(parent, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
}

// dirNames enumerates a pinned directory through an independent directory offset.
func dirNames(fd int) ([]string, error) {
	dup, err := unix.Openat(fd, ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(dup), "scratch")
	defer f.Close()
	return f.Readdirnames(-1)
}

// mkdirDir creates or opens a private, current-user directory without following symlinks.
func mkdirDir(fd int, name string) (int, error) {
	if err := unix.Mkdirat(fd, name, 0700); err != nil && err != unix.EEXIST {
		return -1, err
	}
	n, err := openDir(fd, name)
	if err == nil {
		var st unix.Stat_t
		err = unix.Fstat(n, &st)
		if err == nil && (st.Mode&0777 != 0700 || int(st.Uid) != os.Getuid()) {
			err = fmt.Errorf("unsafe scratch directory %s", name)
		}
		if err != nil {
			unix.Close(n)
			n = -1
		}
	}
	return n, err
}

// sharedTmpAllowed accepts an owner-controlled shared container without changing
// its existing read/execute permissions. Only its owner may write to it.
func sharedTmpAllowed(st unix.Stat_t) bool {
	return st.Mode&unix.S_IFMT == unix.S_IFDIR && int(st.Uid) == os.Getuid() && st.Mode&0700 == 0700 && st.Mode&0022 == 0 && st.Mode&07000 == 0
}

// openSharedTmp opens the shared tmp container, optionally creating it privately.
// Existing containers retain their inode and permissions; symlinks are refused.
func openSharedTmp(fd int, create bool, inspections *int) (int, error) {
	if create {
		if err := unix.Mkdirat(fd, "tmp", 0700); err != nil && err != unix.EEXIST {
			return -1, err
		}
	}
	n, err := openDir(fd, "tmp")
	if err != nil {
		return -1, err
	}
	var st unix.Stat_t
	if inspections != nil {
		*inspections++
	}
	err = unix.Fstat(n, &st)
	if err == nil && !sharedTmpAllowed(st) {
		err = fmt.Errorf("unsafe shared scratch tmp container")
	}
	if err != nil {
		unix.Close(n)
		return -1, err
	}
	return n, nil
}
