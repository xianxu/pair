package historyarchive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SnapshotObservation is a database file and its write-ahead log as seen at
// one moment: sizes and mtimes, and whether the log exists.
// pair:m5-concept pure
type SnapshotObservation struct {
	DBSize, WALSize   int64
	DBMtime, WALMtime time.Time
	WALPresent        bool
}

// SnapshotStable reports a source that did not change between two
// observations, so files copied between them belong to one state. A checkpoint
// mid-copy moves the database's size or mtime and is caught here.
// pair:m5-concept pure
func SnapshotStable(before, after SnapshotObservation) bool {
	return before.DBSize == after.DBSize && before.DBMtime.Equal(after.DBMtime) &&
		before.WALPresent == after.WALPresent && before.WALSize == after.WALSize && before.WALMtime.Equal(after.WALMtime)
}

func observeDatabase(path string) (SnapshotObservation, error) {
	db, err := os.Lstat(path)
	if err != nil {
		return SnapshotObservation{}, err
	}
	if !db.Mode().IsRegular() {
		return SnapshotObservation{}, errors.New("database is not a regular file")
	}
	o := SnapshotObservation{DBSize: db.Size(), DBMtime: db.ModTime()}
	if wal, err := os.Lstat(path + "-wal"); err == nil && wal.Mode().IsRegular() {
		o.WALPresent, o.WALSize, o.WALMtime = true, wal.Size(), wal.ModTime()
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return SnapshotObservation{}, err
	}
	return o, nil
}

// SQLiteSnapshotter makes a coherent private copy of a live SQLite database
// without ever opening it: it copies the -wal and then the database into a
// private directory, refuses the copy if the source changed meanwhile, runs
// SQLite's backup on the private copy (folding the WAL in) and an integrity
// check, and publishes the result under the staging root (pair#592). The live
// database's directory is only read.
type SQLiteSnapshotter struct {
	SQLite   string // sqlite3 binary
	Scratch  string // private working directory, outside every source
	copyFile func(from, to string) error
}

// Snapshot writes stagingRoot/rel. ok=false with a note means skipped this
// pass (source busy or not coherent); err means the pass itself failed.
func (s SQLiteSnapshotter) Snapshot(ctx context.Context, root, rel, stagingRoot string) (ok bool, note string, err error) {
	if !validPath(rel) {
		return false, "", errors.New("invalid database path")
	}
	src := filepath.Join(root, rel)
	copyFile := s.copyFile
	if copyFile == nil {
		copyFile = copyRegularFile
	}
	before, err := observeDatabase(src)
	if err != nil {
		return false, fmt.Sprintf("%s unreadable: %v", rel, err), nil
	}
	// Over the snapshot bound the copy could not be archived anyway; do not
	// copy gigabytes every pass only to skip them.
	if before.DBSize+before.WALSize > MaxSnapshotRawBytes {
		return false, fmt.Sprintf("%s exceeds the %d-byte snapshot bound; skipped", rel, int64(MaxSnapshotRawBytes)), nil
	}
	if err := os.MkdirAll(s.Scratch, 0o700); err != nil {
		return false, "", err
	}
	work, err := os.MkdirTemp(s.Scratch, "sqlite-")
	if err != nil {
		return false, "", err
	}
	defer os.RemoveAll(work)
	private := filepath.Join(work, filepath.Base(rel))
	// The log first: a checkpoint between the two copies then shows up as a
	// changed database, never as a database newer than its log.
	if before.WALPresent {
		if err := copyFile(src+"-wal", private+"-wal"); err != nil {
			return false, fmt.Sprintf("%s: copy log: %v", rel, err), nil
		}
	}
	if err := copyFile(src, private); err != nil {
		return false, fmt.Sprintf("%s: copy database: %v", rel, err), nil
	}
	after, err := observeDatabase(src)
	if err != nil || !SnapshotStable(before, after) {
		return false, fmt.Sprintf("%s changed during copy; retry next pass", rel), nil
	}
	snap := filepath.Join(work, "snapshot.sqlite")
	if out, err := s.run(ctx, private, ".backup main '"+snap+"'"); err != nil {
		return false, fmt.Sprintf("%s: backup failed: %v %s", rel, err, strings.TrimSpace(out)), nil
	}
	// A rollback-journal snapshot is one self-contained file: opening it later
	// creates no -wal or -shm beside it.
	if out, err := s.run(ctx, snap, "PRAGMA journal_mode=DELETE; PRAGMA integrity_check;"); err != nil || strings.TrimSpace(out) != "delete\nok" {
		return false, fmt.Sprintf("%s: integrity check failed; retry next pass", rel), nil
	}
	dest := filepath.Join(stagingRoot, rel)
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return false, "", err
	}
	if same, _ := sameContent(snap, dest); same {
		return true, "", nil
	}
	if err := os.Rename(snap, dest); err != nil {
		return false, "", err
	}
	return true, "", nil
}

func (s SQLiteSnapshotter) run(ctx context.Context, db, command string) (string, error) {
	bin := s.SQLite
	if bin == "" {
		bin = "sqlite3"
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-bail", "-batch", db, command)
	cmd.WaitDelay = time.Second
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.String(), err
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

// Sweep removes private working copies a killed pass left behind. The
// continuous pass calls it under its lock, so no snapshot is in progress.
func (s SQLiteSnapshotter) Sweep() error {
	entries, err := os.ReadDir(s.Scratch)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "sqlite-") {
			if err := os.RemoveAll(filepath.Join(s.Scratch, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
