package historyarchive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"golang.org/x/sys/unix"
)

// LabelIndexStore holds the index mutation lock and its owned staging directory
// for an entire synchronization. Readers may open the last complete snapshot.
// pair:m5-concept integration
type LabelIndexStore struct {
	StageDir  *os.File
	StagePath string
	ctx       context.Context
	dir       *os.File
	lock      *os.File
	stage     *OwnedStaging
	base      string
	fault     func(string) error
	outcome   commitoutcome.Outcome
}

func OpenLabelIndexStore(ctx context.Context, catalogPath string) (s *LabelIndexStore, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(catalogPath) || filepath.Base(catalogPath) == "." {
		return nil, errors.New("invalid catalog path")
	}
	dir, err := openSafeDir(filepath.Dir(catalogPath), true)
	if err != nil {
		return nil, err
	}
	s = &LabelIndexStore{ctx: ctx, dir: dir, base: filepath.Base(catalogPath)}
	defer func(owned *LabelIndexStore) {
		if err != nil {
			err = errors.Join(err, owned.Close())
		}
	}(s)
	s.lock, err = openRegularAt(dir, s.base+".index.lock", unix.O_RDWR|unix.O_CREAT)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(s.lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return nil, fmt.Errorf("label index busy; retry: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	name := s.base + ".index-staging"
	s.stage, err = OpenOwnedStaging(dir, name, "history-label-index", MaxIndexRecords+16)
	if err != nil {
		return nil, err
	}
	s.StageDir = s.stage.Dir
	s.StagePath = filepath.Join(filepath.Dir(catalogPath), name)
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return s, nil
}

// ReadLabelIndex reads exactly one existing snapshot without creating directories,
// locks or staging. Both parent traversal and the snapshot refuse symlinks.
func ReadLabelIndex(path string) (LabelIndex, error) {
	if !filepath.IsAbs(path) || filepath.Base(path) == "." {
		return LabelIndex{}, errors.New("invalid label index path")
	}
	dir, err := openSafeDir(filepath.Dir(path), false)
	if err != nil {
		return LabelIndex{}, err
	}
	defer dir.Close()
	return readLabelIndexAt(dir, filepath.Base(path))
}

func readLabelIndexAt(dir *os.File, name string) (LabelIndex, error) {
	f, err := openRegularAt(dir, name, unix.O_RDONLY)
	if err != nil {
		return LabelIndex{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return LabelIndex{}, err
	}
	if info.Size() > MaxIndexBytes {
		return LabelIndex{}, errIndexBound
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxIndexBytes+1))
	if err != nil {
		return LabelIndex{}, err
	}
	return DecodeLabelIndex(raw)
}

func (s *LabelIndexStore) Read() (LabelIndex, error) {
	if s.dir == nil {
		return LabelIndex{}, errors.New("label index store closed")
	}
	if err := s.ctx.Err(); err != nil {
		return LabelIndex{}, err
	}
	i, err := readLabelIndexAt(s.dir, s.base+".index.json")
	if errors.Is(err, os.ErrNotExist) {
		return LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, nil
	}
	if err == nil {
		err = s.ctx.Err()
	}
	return i, err
}

func (s *LabelIndexStore) inject(phase string) error {
	if s.fault != nil {
		return s.fault(phase)
	}
	return nil
}

// Commit publishes one complete validated snapshot. A failure before rename
// preserves the old snapshot; a failure afterward reports uncertain durability.
func (s *LabelIndexStore) Commit(i LabelIndex) (err error) {
	if s.dir == nil {
		return errors.New("label index store closed")
	}
	if err = s.ctx.Err(); err != nil {
		return err
	}
	if err = ValidateLabelIndex(i); err != nil {
		return err
	}
	raw, err := json.Marshal(i)
	if err != nil {
		return err
	}
	if len(raw) > MaxIndexBytes {
		return errIndexBound
	}
	// Re-read the whole current snapshot even on retries after uncertain publication.
	// Replacing it again with the intended complete bytes reconfirms durability.
	if _, err = s.Read(); err != nil {
		return err
	}
	f, err := openRegularAt(s.StageDir, "output", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL)
	if err != nil {
		return err
	}
	defer func() {
		cleanupErr := unix.Unlinkat(int(s.StageDir.Fd()), "output", 0)
		if errors.Is(cleanupErr, unix.ENOENT) {
			cleanupErr = nil
		}
		if cleanupErr != nil {
			err = commitoutcome.Join(commitoutcome.Of(err), err, cleanupErr)
		}
	}()
	writeErr := s.inject("write")
	if writeErr == nil {
		var n int
		n, writeErr = f.Write(raw)
		if writeErr == nil && n != len(raw) {
			writeErr = io.ErrShortWrite
		}
	}
	if writeErr == nil {
		writeErr = s.ctx.Err()
	}
	if writeErr == nil {
		writeErr = s.inject("sync")
	}
	if writeErr == nil {
		writeErr = f.Sync()
	}
	if err = errors.Join(writeErr, f.Close()); err != nil {
		return err
	}
	if err = s.ctx.Err(); err != nil {
		return err
	}
	if err = s.inject("rename"); err != nil {
		return err
	}
	if err = unix.Renameat(int(s.StageDir.Fd()), "output", int(s.dir.Fd()), s.base+".index.json"); err != nil {
		return err
	}
	s.outcome = commitoutcome.Indeterminate
	if err = s.inject("dirsync"); err == nil {
		err = s.dir.Sync()
	}
	if err != nil {
		return commitoutcome.Wrap(commitoutcome.Indeterminate, err)
	}
	s.outcome = commitoutcome.Committed
	return nil
}

func (s *LabelIndexStore) Close() error {
	if s == nil {
		return nil
	}
	var err error
	if s.stage != nil {
		err = s.stage.Close()
		s.stage = nil
		s.StageDir = nil
	}
	if s.lock != nil {
		err = errors.Join(err, unix.Flock(int(s.lock.Fd()), unix.LOCK_UN), s.lock.Close())
		s.lock = nil
	}
	if s.dir != nil {
		err = errors.Join(err, s.dir.Close())
		s.dir = nil
	}
	if err != nil && s.outcome != commitoutcome.NotAuthoritative {
		return commitoutcome.Wrap(s.outcome, err)
	}
	return err
}
