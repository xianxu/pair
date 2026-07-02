// Package osfs is the shared string-based filesystem seam that pair's per-command
// OSRuntimes embed, so the trivial fs primitives aren't re-implemented per port
// (#93 M3, folding in the M2 review's forward note). Each command's Runtime
// interface still declares only the subset it uses; embedding FS just supplies
// the boilerplate. sessionwatch is deliberately NOT a consumer — its ReadFile is
// []byte/error-based, a genuine divergence kept separate.
package osfs

import (
	"os"
	"path/filepath"
	"time"
)

// FS provides the common filesystem operations as methods (embed it in an
// OSRuntime). All are string-based; read/stat failures return the zero value +
// ok=false rather than an error where the callers treat "absent" as a normal case.
type FS struct{}

func (FS) ReadFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	return string(b), err
}

func (FS) WriteFile(path, data string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(data), 0644)
}

// WriteAtomic writes via a sibling temp file + rename so a concurrent reader
// never sees a torn write (the shell's `> .tmp && mv -f`).
func (FS) WriteAtomic(path, data string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.WriteString(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

func (FS) Remove(path string) { _ = os.Remove(path) }

func (FS) FileSize(path string) (int64, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, false
	}
	return info.Size(), true
}

func (FS) ModTime(path string) (time.Time, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}

func (FS) Touch(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	return f.Close()
}

func (FS) Executable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0111 != 0
}
