package historyarchive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

var errColdCapture = errors.New("closed WAL capture uncertain; leave for the next scheduled pass")

// coldSQLiteInput is one held source, not a pathname that can be reopened.
type coldSQLiteInput interface {
	io.ReadSeeker
	io.ReaderAt
	Observe() (coldSourceStamp, error)
}

type coldSQLiteFile struct {
	*os.File
	parent *os.File
	path   string
}

func (f coldSQLiteFile) Observe() (coldSourceStamp, error) {
	return observeColdSource(f.File, f.parent, f.path)
}

// observeColdSource checks held identities against no-follow current names.
// Directory mutation is evidence even if a sidecar was created then removed.
func observeColdSource(source, parent *os.File, path string) (coldSourceStamp, error) {
	var db, dir, namedDB, namedDir unix.Stat_t
	if err := unix.Fstat(int(source.Fd()), &db); err != nil {
		return coldSourceStamp{}, err
	}
	if err := unix.Fstat(int(parent.Fd()), &dir); err != nil {
		return coldSourceStamp{}, err
	}
	if err := unix.Fstatat(unix.AT_FDCWD, path, &namedDB, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return coldSourceStamp{}, err
	}
	if err := unix.Fstatat(unix.AT_FDCWD, filepath.Dir(path), &namedDir, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return coldSourceStamp{}, err
	}
	if db.Mode&unix.S_IFMT != unix.S_IFREG || dir.Mode&unix.S_IFMT != unix.S_IFDIR || coldStat(db) != coldStat(namedDB) || coldStat(dir) != coldStat(namedDir) {
		return coldSourceStamp{}, errors.New("database or parent identity changed")
	}
	stamp := coldSourceStamp{DB: coldStat(db), Parent: coldStat(dir)}
	for i, suffix := range []string{"-wal", "-shm", "-journal"} {
		var sidecar unix.Stat_t
		err := unix.Fstatat(int(parent.Fd()), filepath.Base(path)+suffix, &sidecar, unix.AT_SYMLINK_NOFOLLOW)
		if errors.Is(err, unix.ENOENT) {
			continue
		}
		if err != nil {
			return coldSourceStamp{}, err
		}
		if sidecar.Mode&unix.S_IFMT != unix.S_IFREG {
			return coldSourceStamp{}, fmt.Errorf("%s is not regular", suffix)
		}
		stamp.Sidecars |= 1 << i
	}
	return stamp, nil
}

func coldStat(s unix.Stat_t) coldFileStamp {
	return coldFileStamp{Device: uint64(s.Dev), Inode: uint64(s.Ino), Size: s.Size, Mode: uint32(s.Mode), MtimeSec: s.Mtim.Sec, MtimeNsec: s.Mtim.Nsec, CtimeSec: s.Ctim.Sec, CtimeNsec: s.Ctim.Nsec}
}

// copyColdWAL returns false only when ordinary SQLite must acquire the source.
// A selected cold capture never retries or falls through after uncertainty.
func copyColdWAL(ctx context.Context, source coldSQLiteInput, output string) (handled bool, err error) {
	uncertain := func(cause error) (bool, error) {
		return true, fmt.Errorf("%w: %w", errColdCapture, cause)
	}
	if err := ctx.Err(); err != nil {
		return uncertain(err)
	}
	before, err := source.Observe()
	if err != nil {
		return uncertain(err)
	}
	if before.Sidecars != 0 {
		return false, nil
	}
	var header [100]byte
	n, err := source.ReadAt(header[:], 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return uncertain(err)
	}
	if !coldWALHeader(header[:n]) {
		return false, nil
	}
	if before.DB.Size < int64(len(header)) || before.DB.Size > MaxSnapshotRawBytes {
		return uncertain(errors.New("closed database exceeds snapshot bounds"))
	}
	out, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return uncertain(err)
	}
	keep := false
	defer func() {
		_ = out.Close()
		if !keep {
			_ = os.Remove(output)
		}
	}()
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return uncertain(err)
	}
	length, digest, err := fileDigest(ctx, source, out, before.DB.Size)
	if err != nil {
		return uncertain(err)
	}
	if err := out.Close(); err != nil {
		return uncertain(err)
	}
	middle, err := source.Observe()
	if err != nil {
		return uncertain(err)
	}
	if length != before.DB.Size || !sameColdSource(before, middle) {
		return uncertain(errors.New("database or parent changed during the first read"))
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return uncertain(err)
	}
	verifiedLength, verifiedDigest, err := fileDigest(ctx, source, nil, before.DB.Size)
	if err != nil {
		return uncertain(err)
	}
	after, err := source.Observe()
	if err != nil {
		return uncertain(err)
	}
	if length != verifiedLength || digest != verifiedDigest || !sameColdSource(before, after) {
		return uncertain(errors.New("database or parent changed during verification"))
	}
	if err := ctx.Err(); err != nil {
		return uncertain(err)
	}
	keep = true
	return true, nil
}
