package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const startupHistoryEntryLimit = 65536

// startupHistoryLocked checks retained membership, not resumability. Its caller
// owns this store's lock; damaged record contents still establish history.
func (s *ThreadStore) startupHistoryLocked(ctx context.Context, scope string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if _, err := os.Lstat(s.journalPath()); err == nil {
		return false, errors.New("startup history has a pending store transaction")
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if s.layout.Local {
		info, err := os.Lstat(filepath.Join(s.root, "thread.json"))
		if err == nil {
			if !info.Mode().IsRegular() {
				return false, errors.New("unsafe startup current record")
			}
			return true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		// Recovery backups retain the exact outgoing conversation bytes even
		// when no readable current record or archive remains.
		present, err := startupHistoryDirectoryWithin(ctx, filepath.Join(s.root, "recovery"), "", startupHistoryEntryLimit, false)
		if err != nil || present {
			return present, err
		}
		scope = "" // A local store belongs to this exact checkout.
	} else {
		manifest, _, _, err := s.loadManifestLockedContext(ctx)
		if err != nil {
			return false, err
		}
		for _, address := range manifest.Threads {
			if scope == "" || address.RepoScope == scope {
				return true, nil
			}
		}
		present, err := startupHistoryTree(ctx, filepath.Join(s.root, "records"), scope)
		if err != nil || present {
			return present, err
		}
	}
	return startupHistoryTree(ctx, filepath.Join(s.root, "archive"), scope)
}

// startupHistoryTree reads names and file types only. Open directory descriptors
// and openat keep a swapped directory from redirecting inspection through links.
func startupHistoryTree(ctx context.Context, path, scope string) (bool, error) {
	return startupHistoryTreeWithin(ctx, path, scope, startupHistoryEntryLimit)
}

func startupHistoryTreeWithin(ctx context.Context, path, scope string, remaining int) (bool, error) {
	return startupHistoryDirectoryWithin(ctx, path, scope, remaining, true)
}

func startupHistoryDirectoryWithin(ctx context.Context, path, scope string, remaining int, scopes bool) (bool, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if errors.Is(err, unix.ENOENT) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	root := os.NewFile(uintptr(fd), path)
	defer root.Close()
	var inspect func(*os.File, bool) (bool, error)
	inspect = func(dir *os.File, scopes bool) (bool, error) {
		for {
			if err := ctx.Err(); err != nil {
				return false, err
			}
			entries, readErr := dir.ReadDir(64)
			for _, entry := range entries {
				remaining--
				if remaining < 0 {
					return false, errors.New("startup history exceeds entry budget")
				}
				var info unix.Stat_t
				if err := unix.Fstatat(int(dir.Fd()), entry.Name(), &info, unix.AT_SYMLINK_NOFOLLOW); err != nil {
					return false, err
				}
				if scopes {
					if info.Mode&unix.S_IFMT != unix.S_IFDIR {
						return false, fmt.Errorf("unsafe startup history scope %s", entry.Name())
					}
					if scope != "" && entry.Name() != scope {
						continue
					}
					childFD, err := unix.Openat(int(dir.Fd()), entry.Name(), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
					if err != nil {
						return false, err
					}
					child := os.NewFile(uintptr(childFD), entry.Name())
					present, inspectErr := inspect(child, false)
					closeErr := child.Close()
					if inspectErr != nil || closeErr != nil {
						return false, errors.Join(inspectErr, closeErr)
					}
					if present {
						return true, nil
					}
				} else {
					if info.Mode&unix.S_IFMT != unix.S_IFREG || filepath.Ext(entry.Name()) != ".json" {
						return false, fmt.Errorf("unsafe startup history entry %s", entry.Name())
					}
					return true, nil
				}
			}
			if errors.Is(readErr, io.EOF) {
				return false, nil
			}
			if readErr != nil {
				return false, readErr
			}
		}
	}
	return inspect(root, scopes)
}

// observeStartupHistory includes configured local stores even when this
// namespace has never enrolled their repositories. Observation never enrolls,
// reconciles, initializes storage or recovers journals.
func (c *Couch) observeStartupHistory(ctx context.Context, hubPaths []string) (bool, error) {
	if c == nil || c.Threads == nil {
		return false, errors.New("startup history store unavailable")
	}
	global := *c.Threads
	global.readOnly = true
	retained := false
	observe := func(store *ThreadStore) error {
		return store.withPreviewLockContext(ctx, func() error {
			present, err := store.startupHistoryLocked(ctx, "")
			retained = retained || present
			return err
		})
	}
	if err := observe(&global); err != nil {
		return false, err
	}
	seen := map[string]bool{}
	for _, path := range hubPaths {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		target, _, err := c.resolveSlotInput(ctx, path)
		if err != nil {
			return false, err
		}
		if target == nil {
			continue
		}
		local := newSlotThreadStore(c.Namespace, target.Slot)
		local.readOnly = true
		if seen[local.root] {
			continue
		}
		seen[local.root] = true
		if err := observe(local); err != nil {
			return false, err
		}
	}
	return retained, ctx.Err()
}
