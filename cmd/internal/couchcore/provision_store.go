package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/strictjson"
	"golang.org/x/sys/unix"
)

const provisionRecordLimit = 16 << 10

// CreationIntent retains only evidence needed to finish interrupted Git creation.
// Setup progress is observed from Git and the success marker, never journaled.
type CreationIntent struct {
	SchemaVersion int    `json:"schema_version"`
	Primary       string `json:"primary"`
	Common        string `json:"common"`
	Host          string `json:"host"`
	Slot          int    `json:"slot"`
	Remote        string `json:"remote"`
	BaselineSHA   string `json:"baseline_sha"`
	Token         string `json:"token"`
	DirDevice     uint64 `json:"dir_device"`
	DirInode      uint64 `json:"dir_inode"`
}
type SetupSuccess struct {
	SchemaVersion int    `json:"schema_version"`
	Host          string `json:"host"`
	Common        string `json:"common"`
	Admin         string `json:"admin"`
	Slot          int    `json:"slot"`
	BaselineSHA   string `json:"baseline_sha"`
}

// ProvisionStorage allows stateful failure injection across atomic publications.
// Its mutations are called only while the repository creation lease is held.
type ProvisionStorage interface {
	Read(string, any) (bool, error)
	Write(string, any) error
	Remove(string) error
}
type ProvisionStore struct{}

func (ProvisionStore) Read(path string, dst any) (bool, error) {
	if err := provisionSafePath(path); err != nil {
		return false, err
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("provision record is not a regular file: %s", path)
	}
	raw, err := io.ReadAll(io.LimitReader(f, provisionRecordLimit+1))
	if err != nil {
		return false, err
	}
	if len(raw) > provisionRecordLimit {
		return false, fmt.Errorf("provision record exceeds %d bytes", provisionRecordLimit)
	}
	if err := strictjson.Decode(raw, dst); err != nil {
		return false, fmt.Errorf("invalid provision record %s: %w", path, err)
	}
	return true, nil
}
func (ProvisionStore) Write(path string, value any) error {
	if err := provisionSafePath(path); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("unsafe provision record %s", path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(raw) > provisionRecordLimit {
		return fmt.Errorf("provision record too large")
	}
	if err := provisionMkdirAll(filepath.Dir(path)); err != nil {
		return err
	}
	// Only this writer uses the prefix, and every writer/cleanup holds the same
	// creation lease, including setup publication after Weave returns.
	stale, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".couch-provision-*"))
	if err != nil {
		return err
	}
	for _, tmp := range stale {
		info, err := os.Lstat(tmp)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsafe provision temporary %s", tmp)
		}
		if err := os.Remove(tmp); err != nil {
			return err
		}
	}
	return writeAtomicBytesWithPattern(path, raw, ".couch-provision-*")
}
func (ProvisionStore) Remove(path string) error {
	if err := provisionSafePath(path); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsafe provision record %s", path)
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

// Missing suffixes are allowed; every existing component must be a physical path.
func provisionSafePath(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("noncanonical provision path %q", path)
	}
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in provision path %s", current)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if current == filepath.Dir(current) {
			return nil
		}
	}
}
func provisionMkdirAll(path string) error {
	if err := provisionSafePath(path); err != nil {
		return err
	}
	return os.MkdirAll(path, 0700)
}
func provisionDirIdentity(path string) (uint64, uint64, error) {
	if err := provisionSafePath(path); err != nil {
		return 0, 0, err
	}
	var st unix.Stat_t
	if err := unix.Lstat(path, &st); err != nil {
		return 0, 0, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR {
		return 0, 0, fmt.Errorf("not a provision directory: %s", path)
	}
	return uint64(st.Dev), uint64(st.Ino), nil
}
