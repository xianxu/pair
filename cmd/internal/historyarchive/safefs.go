package historyarchive

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
)

func safeRelative(p string) error {
	if !validPath(p) {
		return errors.New("unsafe relative archive path")
	}
	return nil
}
func childDir(parent *os.File, name string, create bool) (*os.File, error) {
	if create {
		if e := unix.Mkdirat(int(parent.Fd()), name, 0700); e != nil && !errors.Is(e, unix.EEXIST) {
			return nil, e
		}
	}
	fd, e := unix.Openat(int(parent.Fd()), name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if e != nil {
		return nil, e
	}
	return os.NewFile(uintptr(fd), name), nil
}

// openSafeDir rejects symlinks in every component, including the configured root.
func openSafeDir(p string, create bool) (*os.File, error) {
	absolute, e := filepath.Abs(p)
	if e != nil {
		return nil, e
	}
	dir, e := os.Open("/")
	if e != nil {
		return nil, e
	}
	for _, part := range strings.Split(strings.TrimPrefix(absolute, "/"), "/") {
		if part == "" {
			continue
		}
		next, e := childDir(dir, part, create)
		if e == nil && create {
			e = dir.Sync()
		}
		dir.Close()
		if e != nil {
			if next != nil {
				next.Close()
			}
			return nil, e
		}
		dir = next
	}
	return dir, nil
}

// safeParent returns an owned directory descriptor and a validated final name.
func safeParent(root *os.File, rel string, create bool) (*os.File, string, error) {
	if e := safeRelative(rel); e != nil {
		return nil, "", e
	}
	dir, e := childDir(root, ".", false)
	if e != nil {
		return nil, "", e
	}
	parts := strings.Split(rel, "/")
	for _, part := range parts[:len(parts)-1] {
		next, e := childDir(dir, part, create)
		if e == nil && create {
			e = dir.Sync()
		}
		dir.Close()
		if e != nil {
			if next != nil {
				next.Close()
			}
			return nil, "", e
		}
		dir = next
	}
	return dir, parts[len(parts)-1], nil
}
func openRegularAt(root *os.File, rel string, flags int) (*os.File, error) {
	dir, name, e := safeParent(root, rel, false)
	if e != nil {
		return nil, e
	}
	defer dir.Close()
	fd, e := unix.Openat(int(dir.Fd()), name, flags|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0600)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), rel)
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() {
		f.Close()
		if e == nil {
			e = errors.New("archive path is not a regular file")
		}
		return nil, e
	}
	return f, nil
}
