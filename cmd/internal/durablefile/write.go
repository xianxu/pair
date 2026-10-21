// Package durablefile publishes private files with file and directory durability.
package durablefile

import (
	"errors"
	"os"
	"path/filepath"
)

func WriteAtomic(path string, raw []byte, pattern string) error {
	return WriteAtomicMode(path, raw, pattern, 0600)
}

// WriteAtomicMode is WriteAtomic publishing with the given permissions, for a
// file someone else also reads (a shared document keeps its mode).
func WriteAtomicMode(path string, raw []byte, pattern string, perm os.FileMode) error {
	if e := EnsureDirectory(filepath.Dir(path)); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), pattern)
	if e != nil {
		return e
	}
	return publish(path, raw, f, perm)
}

// WriteAtomicStaged requires exclusive caller ownership of the stage pathname.
func WriteAtomicStaged(path string, raw []byte, stage string) error {
	if e := EnsureDirectory(filepath.Dir(path)); e != nil {
		return e
	}
	if info, e := os.Lstat(stage); e == nil {
		if !info.Mode().IsRegular() {
			return errors.New("unsafe durable publication stage")
		}
		if e = removeOwnedStage(stage, info); e != nil {
			return e
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	f, e := os.OpenFile(stage, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	return publish(path, raw, f, 0600)
}
func publish(path string, raw []byte, f *os.File, perm os.FileMode) error {
	defer f.Close()
	owned, e := f.Stat()
	if e != nil {
		return e
	}
	defer func() { _ = removeOwnedStage(f.Name(), owned) }()
	if e := f.Chmod(perm); e != nil {
		return e
	}
	if _, e := f.Write(raw); e != nil {
		return e
	}
	if e := f.Sync(); e != nil {
		return e
	}
	if e := f.Close(); e != nil {
		return e
	}
	if e := verifyOwnedStage(f.Name(), owned); e != nil {
		return e
	}
	if e := os.Rename(f.Name(), path); e != nil {
		return e
	}
	return SyncDirectory(filepath.Dir(path))
}

// A stage pathname grants cleanup/publication authority only while it names
// the file originally opened by the writer. Lstat refuses a substituted symlink.
func verifyOwnedStage(path string, owned os.FileInfo) error {
	current, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !current.Mode().IsRegular() || !os.SameFile(owned, current) {
		return errors.New("durable publication stage replaced")
	}
	return nil
}
func removeOwnedStage(path string, owned os.FileInfo) error {
	if err := verifyOwnedStage(path, owned); err != nil {
		return err
	}
	return os.Remove(path)
}

func SyncDirectory(path string) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}

// EnsureDirectory persists every newly created ancestor before publishing files.
func EnsureDirectory(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return errors.New("durable authority parent is not a directory")
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	parent := filepath.Dir(path)
	if parent == path {
		return err
	}
	if err = EnsureDirectory(parent); err != nil {
		return err
	}
	if err = os.Mkdir(path, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return SyncDirectory(parent)
}
