package storagegc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

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
	f, err := os.CreateTemp(filepath.Dir(path), ".pending-")
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
	if err = os.Rename(f.Name(), path); err != nil {
		return err
	}
	return c.syncDirectory(filepath.Dir(path))
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
