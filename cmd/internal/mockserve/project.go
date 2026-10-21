package mockserve

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"golang.org/x/sys/unix"
)

// ErrPathChanged means an observed entry was replaced before its handle was admitted.
var ErrPathChanged = errors.New("project path changed during open")

const MaxEntries = 1024
const MaxBytes int64 = 32 << 20
const MaxHTMLBytes int64 = 1 << 20
const selectionLimit int64 = 256
const selectionLockName = ".mock-current.lock"
const selectionStageName = ".mock-current.tmp"

// Version is a validated mock directory name, ordered by number then variant.
type Version struct{ name, number, variant string }

func ParseVersion(name string) (Version, error) {
	if !strings.HasPrefix(name, "mock-") || len(name) > 128 {
		return Version{}, fmt.Errorf("invalid mock version %q", name)
	}
	digits := strings.TrimPrefix(name, "mock-")
	variant := ""
	if len(digits) > 0 && digits[len(digits)-1] >= 'a' && digits[len(digits)-1] <= 'z' {
		variant = digits[len(digits)-1:]
		digits = digits[:len(digits)-1]
	}
	if len(digits) == 0 || digits[0] == '0' {
		return Version{}, fmt.Errorf("invalid mock version %q", name)
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			return Version{}, fmt.Errorf("invalid mock version %q", name)
		}
	}
	return Version{name: name, number: digits, variant: variant}, nil
}
func (v Version) less(w Version) bool {
	if len(v.number) != len(w.number) {
		return len(v.number) < len(w.number)
	}
	if v.number != w.number {
		return v.number < w.number
	}
	return v.variant < w.variant
}
func SelectVersion(names []string, explicit string) (string, error) {
	if explicit != "" {
		if _, err := ParseVersion(explicit); err != nil {
			return "", err
		}
		for _, name := range names {
			if name == explicit {
				return name, nil
			}
		}
		return "", fmt.Errorf("mock version %q does not exist", explicit)
	}
	var best Version
	for _, name := range names {
		v, e := ParseVersion(name)
		if e == nil && (best.name == "" || best.less(v)) {
			best = v
		}
	}
	if best.name == "" {
		return "", fmt.Errorf("no mock version directories")
	}
	return best.name, nil
}

// Project owns the directory handle used by every project read and write.
type Project struct {
	root *os.Root
	// scanWork observes actual entry visits and read bytes at the IO boundary.
	scanWork func(int, int64)
	// beforeOpen is a deterministic boundary for concurrent-replacement tests.
	beforeOpen func(string)
	// beforeSelectionCommit observes the completed stage while its writer holds the lock.
	beforeSelectionCommit func()
	// selectionContended observes an OS-confirmed lock conflict.
	selectionContended func()
}

func OpenProject(dir string) (*Project, error) {
	r, e := os.OpenRoot(dir)
	if e != nil {
		return nil, e
	}
	return &Project{root: r}, nil
}
func (p *Project) Close() error { return p.root.Close() }

// openDirectory pins every observed directory identity before descending. A
// project-root Root alone would allow an ancestor link back to private files.
func (p *Project) openDirectory(name string) (*os.Root, error) {
	if !fs.ValidPath(name) {
		return nil, fmt.Errorf("invalid project path %q", name)
	}
	dir, err := p.root.OpenRoot(".")
	if err != nil {
		return nil, err
	}
	if name == "." {
		return dir, nil
	}
	relative := ""
	for _, part := range strings.Split(name, "/") {
		if relative != "" {
			relative += "/"
		}
		relative += part
		next, err := p.openChildDirectory(dir, part, relative)
		dir.Close()
		if err != nil {
			return nil, err
		}
		dir = next
	}
	return dir, nil
}
func (p *Project) openChildDirectory(parent *os.Root, name, relative string) (*os.Root, error) {
	observed, err := parent.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !observed.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", relative)
	}
	if p.beforeOpen != nil {
		p.beforeOpen(relative)
	}
	child, err := parent.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	actual, err := child.Stat(".")
	if err != nil || !os.SameFile(observed, actual) {
		child.Close()
		return nil, fmt.Errorf("%w: directory %s", ErrPathChanged, relative)
	}
	return child, nil
}
func (p *Project) openRegularAt(parent *os.Root, name, relative string) (*os.File, error) {
	observed, err := parent.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !observed.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", relative)
	}
	if p.beforeOpen != nil {
		p.beforeOpen(relative)
	}
	f, err := parent.Open(name)
	if err != nil {
		return nil, err
	}
	actual, err := f.Stat()
	if err != nil || !actual.Mode().IsRegular() || !os.SameFile(observed, actual) {
		f.Close()
		return nil, fmt.Errorf("%w: file %s", ErrPathChanged, relative)
	}
	return f, nil
}
func (p *Project) readBounded(name string, limit int64) ([]byte, error) {
	if !fs.ValidPath(name) || name == "." {
		return nil, fmt.Errorf("invalid project file %q", name)
	}
	dir, err := p.openDirectory(path.Dir(name))
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	f, err := p.openRegularAt(dir, path.Base(name), name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if st.Size() > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", name, limit)
	}
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", name, limit)
	}
	return b, nil
}
func (p *Project) existsScreen(version, screenPath string) bool {
	if _, e := ParseVersion(version); e != nil {
		return false
	}
	if screenPath == "" || path.Clean(screenPath) != screenPath || strings.HasPrefix(screenPath, "/") || screenPath == ".." || strings.HasPrefix(screenPath, "../") || !strings.HasSuffix(screenPath, ".html") {
		return false
	}
	_, e := p.readBounded(version+"/"+screenPath, MaxHTMLBytes)
	return e == nil
}

// selectionLock admits one persistent regular-file inode before and after taking
// its advisory lock. Writers never unlink this inode: a process exit releases the
// OS lock without creating an independent lock domain for the next writer.
func (p *Project) selectionLock() (*os.File, error) {
	for {
		observed, err := p.root.Lstat(selectionLockName)
		if errors.Is(err, os.ErrNotExist) {
			f, createErr := p.root.OpenFile(selectionLockName, os.O_CREATE|os.O_EXCL|os.O_RDWR|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0600)
			if errors.Is(createErr, os.ErrExist) {
				continue
			}
			if createErr != nil {
				return nil, createErr
			}
			f.Close()
			continue
		}
		if err != nil {
			return nil, err
		}
		if !observed.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular file", selectionLockName)
		}
		if p.beforeOpen != nil {
			p.beforeOpen(selectionLockName)
		}
		f, err := p.root.OpenFile(selectionLockName, os.O_RDWR|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0600)
		if err != nil {
			return nil, err
		}
		actual, err := f.Stat()
		if err != nil || !actual.Mode().IsRegular() || !os.SameFile(observed, actual) {
			f.Close()
			return nil, fmt.Errorf("%w: selection lock", ErrPathChanged)
		}
		err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			if p.selectionContended != nil {
				p.selectionContended()
			}
			for {
				err = unix.Flock(int(f.Fd()), unix.LOCK_EX)
				if !errors.Is(err, unix.EINTR) {
					break
				}
			}
		}
		if err != nil {
			f.Close()
			return nil, err
		}
		current, err := p.root.Lstat(selectionLockName)
		if err != nil || !current.Mode().IsRegular() || !os.SameFile(actual, current) {
			f.Close()
			return nil, fmt.Errorf("%w: selection lock after acquisition", ErrPathChanged)
		}
		return f, nil
	}
}

func (p *Project) Use(version string) error {
	if _, err := ParseVersion(version); err != nil {
		return err
	}
	lock, err := p.selectionLock()
	if err != nil {
		return err
	}
	defer lock.Close()
	if _, err = p.readBounded(version+"/index.html", MaxHTMLBytes); err != nil {
		return err
	}
	// Only a writer holding the persistent lock can reclaim an interrupted stage.
	old, err := p.root.Lstat(selectionStageName)
	if err == nil {
		if !old.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", selectionStageName)
		}
		if err = p.root.Remove(selectionStageName); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := p.root.OpenFile(selectionStageName, os.O_CREATE|os.O_EXCL|os.O_WRONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0600)
	if err != nil {
		return err
	}
	defer p.root.Remove(selectionStageName)
	actual, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	if !actual.Mode().IsRegular() {
		f.Close()
		return fmt.Errorf("selection stage is not a regular file")
	}
	_, err = io.WriteString(f, version+"\n")
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if p.beforeSelectionCommit != nil {
		p.beforeSelectionCommit()
	}
	current, err := p.root.Lstat(selectionStageName)
	if err != nil || !current.Mode().IsRegular() || !os.SameFile(actual, current) {
		return fmt.Errorf("%w: selection stage", ErrPathChanged)
	}
	return p.root.Rename(selectionStageName, ".mock-current")
}
