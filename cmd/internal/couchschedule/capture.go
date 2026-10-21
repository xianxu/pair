package couchschedule

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// Two permanent leases bound abandoned source/binary captures without growing
// the lock inventory. The Go process inherits the held open-file description.
const (
	captureSlots          = 2
	maxCaptureTreeEntries = 22000
	maxCaptureDepth       = 128
)

var captureName = regexp.MustCompile(`^capture-[0-9]+-[0-9]+$`)

type captureContextKey struct{}
type buildScratch struct {
	root, slot, path string
	lease            *artifactLease
	identity         cacheFileIdentity
	// A raw CLOEXEC duplicate intentionally survives os.File finalization if
	// a standalone caller discards an uncertain cleanup error. Join closes it.
	retainedFD int
	closeOnce  sync.Once
	closeMu    sync.Mutex
	closeErr   error
}

func captureExtraFiles(ctx context.Context) []*os.File {
	if lease, ok := ctx.Value(captureContextKey{}).(*artifactLease); ok {
		return []*os.File{lease.file}
	}
	return nil
}

func acquireBuildScratch(ctx context.Context, cache string) (*buildScratch, error) {
	if cache == "" {
		return nil, errors.New("build cache directory required")
	}
	cache, err := filepath.Abs(cache)
	if err != nil {
		return nil, err
	}
	root := cache + ".captures"
	dir, err := openScheduleDir(ctx, root, true)
	if err != nil {
		return nil, err
	}
	if err = dir.Close(); err != nil {
		return nil, err
	}

	for {
		unlock, err := buildLock(ctx, filepath.Join(root, "allocation.lock"))
		if err != nil {
			return nil, err
		}
		var chosen *buildScratch
		var unsafe error
		for i := 0; i < captureSlots; i++ {
			slot := filepath.Join(root, fmt.Sprintf("slot-%d", i))
			lease, e := cacheFileLease(slot+".lock", unix.LOCK_EX)
			if errors.Is(e, errCacheLive) {
				continue
			}
			if e != nil {
				unsafe = e
				continue
			}
			if e = os.Mkdir(slot, 0700); e != nil && !os.IsExist(e) {
				lease.Close()
				unsafe = e
				continue
			}
			if e = buildSafePath(root, slot); e == nil {
				e = cleanCaptureSlot(ctx, slot)
			}
			if e != nil {
				lease.Close()
				unsafe = e
				continue
			}
			if chosen == nil {
				raw, e := unix.FcntlInt(lease.file.Fd(), unix.F_DUPFD_CLOEXEC, 0)
				if e != nil {
					lease.Close()
					unsafe = e
					continue
				}
				chosen = &buildScratch{root: root, slot: slot, lease: lease, retainedFD: raw}
			} else {
				lease.Close()
			}
		}
		if chosen != nil {
			chosen.path, err = os.MkdirTemp(chosen.slot, fmt.Sprintf("capture-%d-", time.Now().UnixNano()))
			if err == nil {
				var st os.FileInfo
				st, err = os.Lstat(chosen.path)
				if err == nil {
					chosen.identity, err = cacheFileID(st)
				}
			}
			unlock()
			if err != nil {
				chosen.lease.Close()
				unix.Close(chosen.retainedFD)
				return nil, err
			}
			return chosen, nil
		}
		unlock()
		if unsafe != nil {
			return nil, unsafe
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}
func (s *buildScratch) close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.closeContext(ctx)
}
func (s *buildScratch) closeContext(ctx context.Context) error {
	// A caller waiting for another Join must retain its own deadline too.
	for !s.closeMu.TryLock() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
	defer s.closeMu.Unlock()
	s.closeOnce.Do(func() { s.closeErr = s.cleanup(ctx) })
	return s.closeErr
}
func (s *buildScratch) cleanup(ctx context.Context) error {
	defer s.lease.Close()
	defer unix.Close(s.retainedFD)
	unlock, err := buildLock(ctx, filepath.Join(s.root, "allocation.lock"))
	if err != nil {
		return err
	}
	defer unlock()
	if err = buildSafePath(s.root, s.slot); err != nil {
		return err
	}
	st, err := os.Lstat(s.path)
	if err != nil {
		return err
	}
	id, err := cacheFileID(st)
	if err != nil || !s.identity.matches(id) {
		return errors.New("build capture replaced")
	}
	return cleanCaptureSlot(ctx, s.slot)
}

// Snapshot the entire bounded tree before any deletion. Descriptor-relative
// traversal refuses symlinks, special files and replaced names at every level.
type captureTree struct {
	id       cacheFileIdentity
	children map[string]*captureTree
}
type captureBounds struct {
	entries int
}

func snapshotCapture(ctx context.Context, f *os.File, depth int, budget *captureBounds) (*captureTree, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	id, err := cacheFileID(st)
	if err != nil {
		return nil, err
	}
	budget.entries++
	if st.Mode().IsRegular() {
		if stat, ok := st.Sys().(*syscall.Stat_t); !ok || stat.Nlink != 1 {
			return nil, errors.New("hardlinked build capture member")
		}
	}
	if budget.entries > maxCaptureTreeEntries || depth > maxCaptureDepth {
		return nil, errors.New("build capture exceeds recovery bounds")
	}
	n := &captureTree{id: id}
	if st.Mode().IsRegular() {
		return n, nil
	}
	if !st.IsDir() {
		return nil, errors.New("unsafe build capture member")
	}
	names, err := f.Readdirnames(maxCaptureTreeEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	n.children = map[string]*captureTree{}
	for _, name := range names {
		if depth == 0 && name != "view" && name != "capture.mod" && name != "capture.sum" && name != "task" {
			return nil, errors.New("unknown build capture member")
		}
		if depth == 0 && name == "view" {
			var st unix.Stat_t
			if err = unix.Fstatat(int(f.Fd()), name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
				return nil, err
			}
			if st.Mode&unix.S_IFMT != unix.S_IFDIR {
				return nil, errors.New("unsafe build capture view")
			}
		}
		if depth == 0 && name != "view" { // Metadata and binary are always regular files.
			var st unix.Stat_t
			if err = unix.Fstatat(int(f.Fd()), name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
				return nil, err
			}
			if st.Mode&unix.S_IFMT != unix.S_IFREG {
				return nil, errors.New("unsafe build capture payload")
			}
		}
		fd, e := unix.Openat(int(f.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
		if e != nil {
			return nil, e
		}
		child := os.NewFile(uintptr(fd), name)
		n.children[name], e = snapshotCapture(ctx, child, depth+1, budget)
		child.Close()
		if e != nil {
			return nil, e
		}
	}
	return n, nil
}
func unlinkCapture(ctx context.Context, parent *os.File, name string, want *captureTree, guards ...func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, guard := range guards {
		if err := guard(); err != nil {
			return err
		}
	}
	fd, err := unix.Openat(int(parent.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), name)
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	id, err := cacheFileID(st)
	if err != nil || !want.id.matches(id) {
		return errors.New("build capture member replaced")
	}
	if want.children != nil {
		names, e := f.Readdirnames(maxCaptureTreeEntries + 1)
		if e != nil && !errors.Is(e, io.EOF) {
			return e
		}
		if len(names) != len(want.children) {
			return errors.New("build capture members changed")
		}
		for _, name := range names {
			if want.children[name] == nil {
				return errors.New("build capture members changed")
			}
		}
		for _, name := range names {
			if err = unlinkCapture(ctx, f, name, want.children[name], guards...); err != nil {
				return err
			}
		}
	}
	// Re-open the directory entry after traversal, fencing replacements.
	check, e := unix.Openat(int(parent.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if e != nil {
		return e
	}
	cf := os.NewFile(uintptr(check), name)
	cs, e := cf.Stat()
	cf.Close()
	if e != nil {
		return e
	}
	ci, e := cacheFileID(cs)
	if e != nil || !want.id.matches(ci) {
		return errors.New("build capture member replaced")
	}
	for _, guard := range guards {
		if err := guard(); err != nil {
			return err
		}
	}
	flags := 0
	if want.children != nil {
		flags = unix.AT_REMOVEDIR
	}
	return unix.Unlinkat(int(parent.Fd()), name, flags)
}
func cleanCaptureSlot(ctx context.Context, slot string) error {
	fd, err := unix.Open(slot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), slot)
	defer f.Close()
	slotStat, err := f.Stat()
	if err != nil {
		return err
	}
	slotID, err := cacheFileID(slotStat)
	if err != nil {
		return err
	}
	names, err := f.Readdirnames(3)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(names) > 1 {
		return errors.New("unknown build capture slot occupants")
	}
	for _, name := range names {
		if err = ctx.Err(); err != nil {
			return err
		}
		if !captureName.MatchString(name) {
			return errors.New("unknown build capture generation")
		}
		child, e := unix.Openat(fd, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if e != nil {
			return e
		}
		cf := os.NewFile(uintptr(child), name)
		tree, e := snapshotCapture(ctx, cf, 0, &captureBounds{})
		cf.Close()
		if e != nil {
			return e
		}
		guard := func() error {
			if e := verifyCacheDirName(slot, slotID); e != nil {
				return e
			}
			return verifyCacheDirName(filepath.Join(slot, name), tree.id)
		}
		if e = unlinkCapture(ctx, f, name, tree, guard); e != nil {
			return e
		}
	}
	return nil
}

// Materialization stays within the recovery entry/depth bound, reserving
// entries for Go-owned transient go.mod replacements inside the source view.
func validateCaptureInputs(inputs []buildInput, mapped map[string]string) error {
	names := map[string]bool{"view": true, "capture.mod": true, "capture.sum": true, "task": true}
	for _, in := range inputs {
		path := filepath.Join("view", mapped[in.Module], filepath.FromSlash(in.Path))
		if len(strings.Split(path, string(filepath.Separator))) > maxCaptureDepth {
			return errors.New("build capture depth exceeds bound")
		}
		for path != "." {
			names[path] = true
			if len(names) > maxCaptureTreeEntries-32 {
				return errors.New("build capture entries exceed bound")
			}
			path = filepath.Dir(path)
		}
	}
	return nil
}
