package couchscratch

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"strings"
	"syscall"
	"time"
)

// Result accounts for a sweep's work. Visited counts temp entry observations;
// Inspections includes every stat/fstat call, including directory identity and
// bookkeeping checks. BookkeepingReads counts owner declaration body reads.
// TempPayloadReads remains zero: cleanup never opens temporary payload bodies.
// Probe work is supplied independently through Options.Probe, not counted here.
type Result struct {
	Visited, Inspections, BookkeepingReads, TempPayloadReads int
	Removed, Retained                                        int
}

// observation records one complete eligible subtree before deletion begins.
type observation struct {
	name     string
	stat     unix.Stat_t
	children []*observation
}

// same compares identity and mutation-sensitive metadata, excluding access time.
func same(a, b unix.Stat_t) bool {
	return a.Dev == b.Dev && a.Ino == b.Ino && a.Mode == b.Mode && a.Size == b.Size && a.Mtim == b.Mtim && a.Ctim == b.Ctim
}

// eligibleStat rejects special nodes and entries younger than the strict age cutoff.
func eligibleStat(s unix.Stat_t, now time.Time) bool {
	var mode os.FileMode
	switch s.Mode & unix.S_IFMT {
	case unix.S_IFREG:
	case unix.S_IFDIR:
		mode = os.ModeDir
	default:
		return false
	}
	return ageEligible(mode, time.Unix(s.Mtim.Sec, s.Mtim.Nsec), now)
}

// observe snapshots an eligible tree using confined, nofollow metadata observations.
func observe(ctx context.Context, fd int, name string, dev uint64, now time.Time, result *Result) (*observation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	n := &observation{name: name}
	result.Visited++
	result.Inspections++
	if err := unix.Fstatat(fd, name, &n.stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return nil, err
	}
	if uint64(n.stat.Dev) != dev || !eligibleStat(n.stat, now) {
		return nil, fmt.Errorf("retained ineligible entry")
	}
	if n.stat.Mode&unix.S_IFMT != unix.S_IFDIR {
		return n, nil
	}
	child, err := openDir(fd, name)
	if err != nil {
		return nil, err
	}
	defer unix.Close(child)
	var opened unix.Stat_t
	result.Inspections++
	if err = unix.Fstat(child, &opened); err != nil || !same(opened, n.stat) {
		return nil, fmt.Errorf("directory changed")
	}
	names, err := dirNames(child)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		c, err := observe(ctx, child, name, dev, now, result)
		if err != nil {
			return nil, err
		}
		n.children = append(n.children, c)
	}
	return n, nil
}

// erase revalidates each nofollow directory entry immediately before acting. A
// concurrently added child prevents rmdir; replacement and mount changes abort.
func erase(ctx context.Context, fd int, n *observation, result *Result) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	result.Visited++
	if n.stat.Mode&unix.S_IFMT != unix.S_IFDIR {
		var st unix.Stat_t
		result.Inspections++
		if err := unix.Fstatat(fd, n.name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return err
		}
		if !same(st, n.stat) {
			return fmt.Errorf("entry changed")
		}
		return unix.Unlinkat(fd, n.name, 0)
	}
	child, err := openDir(fd, n.name)
	if err != nil {
		return err
	}
	defer unix.Close(child)
	var opened unix.Stat_t
	result.Inspections++
	if err = unix.Fstat(child, &opened); err != nil || !same(opened, n.stat) {
		return fmt.Errorf("directory changed")
	}

	for _, c := range n.children {
		if err = erase(ctx, child, c, result); err != nil {
			return err
		}
	}
	// Our own child removals alter directory times and size. Compare identity
	// again before rmdir so a renamed replacement directory is retained.
	var current unix.Stat_t
	result.Inspections++
	if err = unix.Fstatat(fd, n.name, &current, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	if current.Dev != opened.Dev || current.Ino != opened.Ino || current.Mode != opened.Mode {
		return fmt.Errorf("directory replaced")
	}
	return unix.Unlinkat(fd, n.name, unix.AT_REMOVEDIR)
}

// Sweep removes stale ordinary top-level entries inside declared slot roots.
// It never removes the roots or reads scratch file contents. Unknown ownership,
// special nodes, incomplete observations and changed trees are retained.
func (m *Manager) Sweep(ctx context.Context) (result Result, sweepErr error) {
	m.mu.Lock()
	defer func() {
		m.mu.Unlock()
		if result.Removed == 0 && (sweepErr == nil || errors.Is(sweepErr, context.Canceled)) {
			return
		}
		message := fmt.Sprintf("scratch sweep: removed %d entries; retained %d", result.Removed, result.Retained)
		if sweepErr != nil {
			message += "; error: " + sweepErr.Error()
		}
		message = strings.ReplaceAll(strings.ReplaceAll(message, "\n", " "), "\r", " ")
		if len(message) > 512 {
			message = message[:509] + "..."
		}
		m.note(message)
	}()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	p := m.options.Probe()
	if !p.Mounted || p.Err != nil {
		return result, nil
	}
	fd, err := openDir(unix.AT_FDCWD, p.Root)
	if err != nil {
		return result, err
	}
	defer unix.Close(fd)
	var rootStat unix.Stat_t
	result.Inspections++
	if err = unix.Fstat(fd, &rootStat); err != nil {
		return result, err
	}
	book, err := openDir(fd, "couch-scratch")
	if err == unix.ENOENT {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	defer unix.Close(book)
	owners, err := openDir(book, "owners")
	if err != nil {
		return result, err
	}
	defer unix.Close(owners)
	names, err := dirNames(owners)
	if err != nil {
		return result, err
	}
	tmp, err := openSharedTmp(fd, false, &result.Inspections)
	if err != nil {
		return result, err
	}
	defer unix.Close(tmp)
	now := m.options.Now()
	var failures []error
	for _, name := range names {
		if err = ctx.Err(); err != nil {
			return result, err
		}
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		owner, err := readOwnerAt(owners, name, &result)
		if err != nil || name != slotName(owner)+".json" {
			continue
		}
		slot, err := openDir(tmp, slotName(owner))
		if err != nil {
			continue
		}
		var slotStat unix.Stat_t
		result.Inspections++
		err = unix.Fstat(slot, &slotStat)
		if err != nil || slotStat.Dev != rootStat.Dev {
			unix.Close(slot)
			continue
		}
		entries, err := dirNames(slot)
		if err != nil {
			failures = append(failures, err)
			unix.Close(slot)
			continue
		}
		for _, entry := range entries {
			n, e := observe(ctx, slot, entry, uint64(rootStat.Dev), now, &result)
			if e != nil {
				result.Retained++
				continue
			}
			if m.beforeRemove != nil {
				m.beforeRemove()
			}
			if e = erase(ctx, slot, n, &result); e != nil {
				result.Retained++
				if ctx.Err() != nil {
					unix.Close(slot)
					return result, ctx.Err()
				}
				continue
			}
			result.Removed++
		}
		unix.Close(slot)
	}
	return result, errors.Join(failures...)
}

// readOwnerAt reads only a bounded regular bookkeeping declaration, never a payload.
func readOwnerAt(fd int, name string, result *Result) (Slot, error) {
	f, err := unix.Openat(fd, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return Slot{}, err
	}
	file := os.NewFile(uintptr(f), name)
	defer file.Close()
	result.Inspections++
	st, err := file.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return Slot{}, fmt.Errorf("invalid owner")
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok || int(sys.Uid) != os.Getuid() {
		return Slot{}, fmt.Errorf("invalid owner uid")
	}
	result.BookkeepingReads++
	data, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil {
		return Slot{}, err
	}
	if len(data) > 1<<20 {
		return Slot{}, fmt.Errorf("owner declaration too large")
	}
	return decodeOwner(data)
}
