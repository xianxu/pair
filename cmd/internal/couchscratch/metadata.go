package couchscratch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// MetadataOptions supplies the startup-only cache route and native observation.
type MetadataOptions struct {
	CacheDir, Root string
	UseRAM         bool
	ServerPresence func(context.Context) (bool, error)
	Notify         func(string)
}

// MetadataDirectory matches Zellij's platform-specific cache directory.
func MetadataDirectory(home, cache, platform string) (string, error) {
	if !validMetadataPath(home) {
		return "", errors.New("invalid metadata home")
	}
	switch platform {
	case "darwin":
		return filepath.Join(home, "Library", "Caches", "org.Zellij-Contributors.Zellij", "contract_version_1"), nil
	case "linux":
		if cache == "" {
			cache = filepath.Join(home, ".cache")
		}
		if !validMetadataPath(cache) {
			return "", errors.New("invalid metadata cache")
		}
		return filepath.Join(cache, "zellij", "contract_version_1"), nil
	default:
		return "", fmt.Errorf("unsupported metadata platform %q", platform)
	}
}

// validMetadataPath rejects unusable roots before any path operations.
func validMetadataPath(path string) bool {
	return filepath.IsAbs(path) && !strings.ContainsRune(path, 0) && filepath.Clean(path) != "/"
}

// metadataOpenPath holds each directory while opening its child, refusing
// symlinks at every component. No route operation follows a pathname link.
func metadataOpenPath(path string, create bool) (int, error) {
	fd, err := openDir(unix.AT_FDCWD, "/")
	if err != nil {
		return -1, err
	}
	for _, name := range strings.Split(strings.TrimPrefix(filepath.Clean(path), "/"), "/") {
		next, e := openDir(fd, name)
		if errors.Is(e, unix.ENOENT) && create {
			e = unix.Mkdirat(fd, name, 0700)
			if e == nil || errors.Is(e, unix.EEXIST) {
				next, e = openDir(fd, name)
			}
		}
		unix.Close(fd)
		if e != nil {
			return -1, e
		}
		fd = next
	}
	return fd, nil
}

// metadataStat observes one entry without following its final symlink.
func metadataStat(fd int, name string) (*unix.Stat_t, error) {
	var st unix.Stat_t
	err := unix.Fstatat(fd, name, &st, unix.AT_SYMLINK_NOFOLLOW)
	if errors.Is(err, unix.ENOENT) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// metadataPrivateDirectory requires an owned directory without shared writers.
func metadataPrivateDirectory(st *unix.Stat_t) bool {
	return st != nil && st.Mode&unix.S_IFMT == unix.S_IFDIR && int(st.Uid) == os.Getuid() && st.Mode&0022 == 0
}

// ConfigureMetadata changes only the exact startup route after a complete
// native-server observation reports absence. The cache-directory lock controls
// Couch setup, not standalone native launches; this is quiescent maintenance.
func ConfigureMetadata(ctx context.Context, o MetadataOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validMetadataPath(o.CacheDir) || !validMetadataPath(o.Root) || o.ServerPresence == nil {
		return errors.New("invalid metadata configuration or missing native observation")
	}
	live, err := o.ServerPresence(ctx)
	if err != nil {
		return fmt.Errorf("metadata native observation: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if live {
		if o.Notify != nil {
			o.Notify("Zellij metadata stays on disk until the next reboot")
		}
		return nil
	}
	ram := -1
	if o.UseRAM {
		// Admission is only an observation: the mount may have disappeared.
		// Hold its existing directory before creating any cache directories.
		ram, err = metadataOpenPath(o.Root, false)
		if err != nil {
			return fmt.Errorf("metadata RAM root: %w", err)
		}
		defer unix.Close(ram)
	}
	cache, err := metadataOpenPath(o.CacheDir, true)
	if err != nil {
		return fmt.Errorf("metadata cache directory: %w", err)
	}
	defer unix.Close(cache)
	var parent unix.Stat_t
	if err := unix.Fstat(cache, &parent); err != nil {
		return err
	}
	if !metadataPrivateDirectory(&parent) {
		return errors.New("unsafe metadata cache directory")
	}
	if err := unix.Flock(cache, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fmt.Errorf("metadata setup lock: %w", err)
	}
	defer unix.Flock(cache, unix.LOCK_UN)
	route, err := metadataStat(cache, "session_info")
	if err != nil {
		return err
	}
	backup, err := metadataStat(cache, "session_info.disk")
	if err != nil {
		return err
	}
	if backup != nil && !metadataPrivateDirectory(backup) {
		return errors.New("unsafe metadata backup")
	}
	s := MetadataState{UseRAM: o.UseRAM, Backup: backup != nil}
	target := filepath.Join(o.Root, "zellij-session-info")
	if route != nil {
		s.Route = MetadataForeignRoute
		if metadataPrivateDirectory(route) {
			s.Route = MetadataDirectoryRoute
		}
		if route.Mode&unix.S_IFMT == unix.S_IFLNK && int(route.Uid) == os.Getuid() {
			buf := make([]byte, len(target)+1)
			n, e := unix.Readlinkat(cache, "session_info", buf)
			if e != nil {
				return e
			}
			if string(buf[:n]) == target {
				s.Route = MetadataOwnedLink
			}
		}
		if s.Route == MetadataDirectoryRoute && s.Backup && o.UseRAM {
			fd, e := openDir(cache, "session_info")
			if e != nil {
				return e
			}
			f := os.NewFile(uintptr(fd), "session_info")
			names, e := f.Readdirnames(1)
			f.Close()
			if e != nil && e != io.EOF {
				return e
			}
			s.Empty = len(names) == 0
		}
	}
	action := DecideMetadata(s)
	if action == MetadataRefuse {
		return errors.New("metadata route or retained backup conflicts with setup")
	}
	if o.UseRAM {
		dest, e := mkdirDir(ram, "zellij-session-info")
		if e != nil {
			return fmt.Errorf("metadata RAM destination: %w", e)
		}
		unix.Close(dest)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	switch action {
	case MetadataBackupAndLink:
		// The held directory lock serializes Couch's backup creation. Refuse
		// every preexisting backup, including an empty directory.
		if st, e := metadataStat(cache, "session_info.disk"); e != nil {
			return e
		} else if st != nil {
			return errors.New("metadata backup already exists")
		}
		if err := unix.Renameat(cache, "session_info", cache, "session_info.disk"); err != nil {
			return err
		}
		return unix.Symlinkat(target, cache, "session_info")
	case MetadataReplaceEmptyWithLink:
		// rmdir itself proves emptiness again; it cannot remove a filled tree.
		if err := unix.Unlinkat(cache, "session_info", unix.AT_REMOVEDIR); err != nil {
			return err
		}
		return unix.Symlinkat(target, cache, "session_info")
	case MetadataCreateLink:
		return unix.Symlinkat(target, cache, "session_info")
	case MetadataCreateDirectory:
		if s.Route == MetadataOwnedLink {
			if err := unix.Unlinkat(cache, "session_info", 0); err != nil {
				return err
			}
		}
		return unix.Mkdirat(cache, "session_info", 0700)
	default:
		return nil
	}
}
