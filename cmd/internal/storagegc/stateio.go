package storagegc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// atomicJSON publishes a bounded metadata record only after syncing its bytes,
// then syncs the rename. The caller holds coordination for the entire effect.
func (l *Locked) atomicJSON(path string, value any) error {
	if l.readOnly {
		return errors.New("read-only retention snapshot")
	}
	if !l.active {
		return errors.New("expired retention lock")
	}
	if err := l.CheckContext(); err != nil {
		return err
	}
	c := l.coordinator
	if c.BeforePersist != nil {
		if err := c.BeforePersist(); err != nil {
			return err
		}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) > 1<<20 {
		return errors.New("retention metadata limit reached")
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return errors.New("unsafe retention record type")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := l.CheckContext(); err != nil {
		return err
	}
	staging := c.PendingMetadataDir()
	if err := checkDirectory(staging, true); err != nil {
		return err
	}
	f, err := os.CreateTemp(staging, ".pending-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if c.AfterTempWrite != nil {
		if err := c.AfterTempWrite(f.Name()); err != nil {
			return err
		}
	}
	if err := l.CheckContext(); err != nil {
		return err
	}
	// Rename must stay atomic. Cross-filesystem publication fails; never copy.
	if err = os.Rename(f.Name(), path); err != nil {
		return err
	}
	return errors.Join(c.syncDirectory(filepath.Dir(path)), c.syncDirectory(staging))
}

// readStateJSON does not create directories or follow a final symlink. A
// nonblocking open permits rejecting substituted FIFOs without waiting.
func readStateJSON(path string, value any) error {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return errors.New("retention state is not a regular file")
	}
	raw, err := io.ReadAll(io.LimitReader(f, 1<<20+1))
	if err != nil {
		return err
	}
	if len(raw) > 1<<20 {
		return errors.New("retention metadata limit reached")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("trailing retention data")
	}
	return nil
}

// isPendingMetadata recognizes os.CreateTemp's private, unpublished names.
// Their contents never grant authority, even when a complete JSON write landed.
func isPendingMetadata(name string) bool {
	const prefix = ".pending-"
	suffix := strings.TrimPrefix(name, prefix)
	return suffix != name && suffix != "" && strings.Trim(suffix, "0123456789") == ""
}

// PendingMetadataDir isolates unpublished bytes from authoritative indexes.
func (c *Coordinator) PendingMetadataDir() string {
	return filepath.Join(c.Root, ".retention", "pending")
}

// RecoverPendingMetadata removes only exact private temporary regular files.
// The root lock proves no coordinated publisher can still own one. limit bounds
// all visited entries; callers account this work alongside collection work.
func (l *Locked) RecoverPendingMetadata(ctx context.Context, dir string, limit int) (int, error) {
	if !l.Writable(l.coordinator.Root) {
		return 0, errors.New("temporary metadata recovery requires writable root lock")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if limit <= 0 {
		return 0, nil
	}
	if err := l.validatePendingDirectory(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	f, err := os.Open(dir)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	names, err := f.Readdirnames(limit)
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}
	processed := 0
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		if err := l.CheckContext(); err != nil {
			return processed, err
		}
		processed++
		if !isPendingMetadata(name) {
			continue
		}
		if err := l.RemovePendingMetadata(ctx, dir, name); err != nil {
			return processed, err
		}
	}
	return processed, nil
}

func (l *Locked) validatePendingDirectory(dir string) error {
	relative, err := filepath.Rel(l.coordinator.Root, dir)
	if err != nil {
		return err
	}
	allowed := relative == "." || relative == ".retention" || relative == ".retention/owners" || relative == ".retention/transactions" || relative == ".retention/pending"
	parts := strings.Split(relative, string(filepath.Separator))
	if len(parts) == 2 && parts[0] == "repos" && parts[1] != "" && parts[1] != "." && parts[1] != ".." {
		allowed = true
	}
	if !allowed {
		return errors.New("not a metadata publication directory")
	}
	current := l.coordinator.Root
	for _, part := range parts {
		current = filepath.Join(current, part)
		if err := checkDirectory(current, false); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return err
			}
			return err
		}
	}
	return nil
}

// RemovePendingMetadata cleans an already visited legacy entry without another
// directory scan. The caller charges that entry to its discovery work budget.
func (l *Locked) RemovePendingMetadata(ctx context.Context, dir, name string) error {
	if !l.Writable(l.coordinator.Root) {
		return errors.New("temporary metadata recovery requires writable root lock")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := l.CheckContext(); err != nil {
		return err
	}
	if !isPendingMetadata(name) {
		return errors.New("not an unpublished metadata filename")
	}
	if err := l.validatePendingDirectory(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	path := filepath.Join(dir, name)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return l.coordinator.syncDirectory(dir)
}
