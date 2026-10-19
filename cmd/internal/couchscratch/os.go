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
