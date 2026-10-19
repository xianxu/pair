package historyarchive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// SnapshotObservation is raw admission metadata; SQLite owns consistency.
// pair:m5-concept pure
type SnapshotObservation struct {
	DBSize, WALSize int64
	WALPresent      bool
}

func observeDatabase(path string) (SnapshotObservation, os.FileInfo, error) {
	db, err := os.Lstat(path)
	if err != nil {
		return SnapshotObservation{}, nil, err
	}
	if !db.Mode().IsRegular() {
		return SnapshotObservation{}, nil, errors.New("database is not a regular file")
	}
	o := SnapshotObservation{DBSize: db.Size()}
	for _, suffix := range []string{"-wal", "-shm"} {
		info, err := os.Lstat(path + suffix)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return SnapshotObservation{}, nil, err
		}
		if !info.Mode().IsRegular() {
			return SnapshotObservation{}, nil, fmt.Errorf("%s is not regular", suffix)
		}
		if suffix == "-wal" {
			o.WALPresent, o.WALSize = true, info.Size()
		}
	}
	return o, db, nil
}

// SQLiteSnapshotter captures an observed cold WAL file or pins a read-only
// source transaction for a size-scaled 5..30 second budget. Both paths validate
// privately before staging. SQLite may update SHM coordination on the active
// path, never source DB/WAL content.
// pair:m5-concept integration
type SQLiteSnapshotter struct {
	SQLite, Scratch string
	Runner          SQLiteRunner
}

// Snapshot never publishes an unconfirmed or unvalidated candidate.
func (s SQLiteSnapshotter) Snapshot(ctx context.Context, root, rel, stagingRoot string) (SnapshotResult, error) {
	if err := ctx.Err(); err != nil {
		return SnapshotResult{}, err
	}
	if !validPath(rel) {
		return SnapshotResult{}, errors.New("invalid database path")
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return snapshotFailure(err, "source root"), nil
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return SnapshotResult{}, err
	}
	if err := s.validateRoots(root, stagingRoot); err != nil {
		return snapshotFailure(err, "snapshot configuration"), nil
	}
	dir, err := openSafeDir(root, false)
	if err != nil {
		return snapshotFailure(err, "source root"), nil
	}
	defer dir.Close()
	parent, _, err := safeParent(dir, rel, false)
	if err != nil {
		return snapshotFailure(err, "source directory"), nil
	}
	defer parent.Close()
	parentBefore, err := parent.Stat()
	if err != nil {
		return SnapshotResult{}, err
	}
	src := filepath.Join(root, rel)
	before, dbBefore, err := observeDatabase(src)
	if err != nil {
		return snapshotFailure(err, rel), nil
	}
	// Keep the original inode alive until the identity fence completes. This
	// descriptor belongs to the Go collector, which owns no SQLite connections.
	// It cannot cancel the separate SQLite child's POSIX locks when closed.
	source, err := openRegularAt(dir, rel, unix.O_RDONLY)
	if err != nil {
		return snapshotFailure(err, rel), nil
	}
	defer source.Close()
	opened, err := source.Stat()
	if err != nil {
		return SnapshotResult{}, err
	}
	if !os.SameFile(dbBefore, opened) {
		return SnapshotResult{Status: SnapshotRejected, Detail: "source identity changed before backup"}, nil
	}
	if before.DBSize > MaxSnapshotRawBytes || before.WALSize > MaxSnapshotRawBytes-before.DBSize {
		return SnapshotResult{Status: SnapshotRejected, Detail: fmt.Sprintf("%s exceeds the %d-byte snapshot bound", rel, int64(MaxSnapshotRawBytes))}, nil
	}
	if err := os.MkdirAll(s.Scratch, 0700); err != nil {
		return SnapshotResult{}, err
	}
	scratch, err := filepath.EvalSymlinks(s.Scratch)
	if err != nil {
		return SnapshotResult{}, err
	}
	work, err := os.MkdirTemp(scratch, "sqlite-")
	if err != nil {
		return SnapshotResult{}, err
	}
	lease, err := snapshotLease(filepath.Join(work, "lease"), true)
	if err != nil {
		os.RemoveAll(work)
		return SnapshotResult{}, err
	}
	defer func() { os.RemoveAll(work); lease.Close() }()
	runner := s.Runner
	if runner == nil {
		runner = ExecSQLiteRunner{SQLite: s.SQLite}
	}
	snap := filepath.Join(work, "snapshot.sqlite")
	readSeconds := SnapshotReadSeconds(before.DBSize + before.WALSize)
	readCtx, cancelRead := context.WithTimeout(ctx, time.Duration(readSeconds)*time.Second)
	cold, coldErr := copyColdWAL(readCtx, coldSQLiteFile{File: source, parent: parent, path: src}, snap)
	cancelRead()
	if ctx.Err() != nil {
		return SnapshotResult{}, ctx.Err()
	}
	if coldErr != nil {
		result := snapshotFailure(coldErr, "")
		if errors.Is(coldErr, errColdCapture) {
			result.Status = SnapshotDeferred
		}
		return result, nil
	}
	var out SQLiteRunResult
	if !cold {
		out, err = runner.Run(ctx, SQLiteRequest{Purpose: SQLiteAcquire, Database: src, Work: work, Lease: lease, ReadSeconds: readSeconds})
		if ctx.Err() != nil {
			return SnapshotResult{}, ctx.Err()
		}
		if err != nil {
			return snapshotFailure(err, out.Stderr), nil
		}
	}
	_, dbAfter, err := observeDatabase(src)
	if err != nil {
		return snapshotFailure(err, rel), nil
	}
	parentAfter, err := os.Stat(filepath.Dir(src))
	if err != nil || !os.SameFile(dbBefore, dbAfter) || !os.SameFile(parentBefore, parentAfter) {
		return SnapshotResult{Status: SnapshotRejected, Detail: "source identity changed during backup"}, nil
	}
	info, err := os.Lstat(snap)
	if err != nil {
		return snapshotFailure(err, "backup output"), nil
	}
	if !info.Mode().IsRegular() || info.Size() > MaxSnapshotRawBytes {
		return SnapshotResult{Status: SnapshotRejected, Detail: "private snapshot exceeds bound or is not regular"}, nil
	}
	out, err = runner.Run(ctx, SQLiteRequest{Purpose: SQLiteValidate, Database: snap, Work: work, Lease: lease})
	if ctx.Err() != nil {
		return SnapshotResult{}, ctx.Err()
	}
	if err != nil {
		return snapshotFailure(err, out.Stderr), nil
	}
	if out.Overflow || strings.TrimSpace(out.Stdout) != "delete\nok" {
		return SnapshotResult{Status: SnapshotRejected, Detail: "private snapshot integrity check failed"}, nil
	}
	if err := os.Chmod(snap, 0o600); err != nil {
		return SnapshotResult{}, err
	}
	dest := filepath.Join(stagingRoot, rel)
	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return SnapshotResult{}, err
	}
	if same, err := sameContent(snap, dest); err == nil && same {
		return SnapshotResult{Status: SnapshotReady}, nil
	}
	if err := ctx.Err(); err != nil {
		return SnapshotResult{}, err
	}
	if err := os.Rename(snap, dest); err != nil {
		return SnapshotResult{}, err
	}
	return SnapshotResult{Status: SnapshotReady}, nil
}

// validateRoots runs before scratch cleanup, directory creation or SQLite.
// Existing ancestors are resolved even when output directories do not exist yet.
func (s SQLiteSnapshotter) validateRoots(root string, staging ...string) error {
	source, err := canonicalSnapshotPath(root)
	if err != nil {
		return err
	}
	outputs := append([]string{s.Scratch}, staging...)
	for _, output := range outputs {
		path, err := canonicalSnapshotPath(output)
		if err != nil {
			return fmt.Errorf("snapshot output %q: %w", output, err)
		}
		if overlaps(source, path) {
			return fmt.Errorf("source %q overlaps snapshot output %q; staging and scratch directories must be separate", source, path)
		}
	}
	return nil
}

func canonicalSnapshotPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty snapshot directory")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	// A dangling symlink is not a missing output directory: refuse it.
	if _, statErr := os.Lstat(abs); !errors.Is(statErr, os.ErrNotExist) {
		return "", err
	}
	parent, err := canonicalSnapshotPath(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(abs)), nil
}

func snapshotLease(path string, create bool) (*os.File, error) {
	flags := unix.O_RDWR | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
	if create {
		flags |= unix.O_CREAT | unix.O_EXCL
	}
	fd, err := unix.Open(path, flags, 0600)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	info, err := f.Stat()
	if err == nil && !info.Mode().IsRegular() {
		err = errors.New("scratch lease is not regular")
	}
	if err == nil {
		err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func copyRegularFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// sameContent compares two files in bounded memory.
func sameContent(a, b string) (bool, error) {
	x, err := os.Open(a)
	if err != nil {
		return false, err
	}
	defer x.Close()
	y, err := os.Open(b)
	if err != nil {
		return false, err
	}
	defer y.Close()
	xi, err := x.Stat()
	if err != nil {
		return false, err
	}
	yi, err := y.Stat()
	if err != nil || xi.Size() != yi.Size() {
		return false, err
	}
	bx, by := make([]byte, 1<<20), make([]byte, 1<<20)
	for {
		nx, ex := io.ReadFull(x, bx)
		ny, ey := io.ReadFull(y, by)
		if nx != ny || !bytes.Equal(bx[:nx], by[:ny]) {
			return false, nil
		}
		if ex == io.EOF || ex == io.ErrUnexpectedEOF {
			return ey == io.EOF || ey == io.ErrUnexpectedEOF, nil
		}
		if ex != nil {
			return false, ex
		}
		if ey != nil {
			return false, ey
		}
	}
}

// Sweep removes abandoned private work only after acquiring its lifetime lease.
func (s SQLiteSnapshotter) Sweep() error {
	entries, err := os.ReadDir(s.Scratch)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "sqlite-") {
			continue
		}
		work := filepath.Join(s.Scratch, entry.Name())
		lease, err := snapshotLease(filepath.Join(work, "lease"), false)
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			continue
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("scratch lease unavailable; work left intact at %s: %w", work, err)
		}
		removeErr := os.RemoveAll(work)
		if lease != nil {
			lease.Close()
		}
		if removeErr != nil {
			return removeErr
		}
	}
	return nil
}
