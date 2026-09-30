// Package durablefile publishes private files with file and directory durability.
package durablefile

import (
	"errors"
	"os"
	"path/filepath"
)

func WriteAtomic(path string, raw []byte, pattern string) error {
	if e := EnsureDirectory(filepath.Dir(path)); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), pattern)
	if e != nil {
		return e
	}
	return publish(path, raw, f)
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
		if e = os.Remove(stage); e != nil {
			return e
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	f, e := os.OpenFile(stage, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	return publish(path, raw, f)
}
func publish(path string, raw []byte, f *os.File) error {
	defer os.Remove(f.Name())
	defer f.Close()
	if e := f.Chmod(0600); e != nil {
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
	if e := os.Rename(f.Name(), path); e != nil {
		return e
	}
	return SyncDirectory(filepath.Dir(path))
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
