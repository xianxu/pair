package historyarchive

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// walDatabase leaves a WAL-mode database whose rows are still only in the
// -wal (no checkpoint, connection never closed), like a live Codex database.
func walDatabase(t *testing.T, dir string) string {
	t.Helper()
	db := filepath.Join(dir, "state_5.sqlite")
	script := "import sqlite3,os,sys\nc=sqlite3.connect(sys.argv[1])\nc.execute('PRAGMA journal_mode=WAL')\nc.execute('PRAGMA wal_autocheckpoint=0')\nc.execute('create table t(x)')\nc.executemany('insert into t values(?)',[(i,) for i in range(50)])\nc.commit()\nos._exit(0)\n"
	if out, err := exec.Command("python3", "-c", script, db).CombinedOutput(); err != nil {
		t.Fatalf("make wal db: %v %s", err, out)
	}
	if _, err := os.Stat(db + "-wal"); err != nil {
		t.Fatalf("fixture lacks a -wal: %v", err)
	}
	return db
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

func TestSQLiteSnapshotIsCoherentAndLeavesSourceUntouched(t *testing.T) {
	src := t.TempDir()
	walDatabase(t, src)
	before := dirNames(t, src)
	staging := filepath.Join(t.TempDir(), "snapshots")
	s := SQLiteSnapshotter{SQLite: "sqlite3", Scratch: t.TempDir()}
	ok, note, err := s.Snapshot(context.Background(), src, "state_5.sqlite", staging)
	if err != nil || !ok {
		t.Fatalf("snapshot: ok=%v note=%q err=%v", ok, note, err)
	}
	if after := dirNames(t, src); strings.Join(after, ",") != strings.Join(before, ",") {
		t.Fatalf("source directory changed: %v -> %v", before, after)
	}
	if names := dirNames(t, staging); strings.Join(names, ",") != "state_5.sqlite" {
		t.Fatalf("staging holds %v, want only the snapshot", names)
	}
	out, err := exec.Command("sqlite3", filepath.Join(staging, "state_5.sqlite"), "PRAGMA integrity_check; select count(*) from t;").Output()
	if err != nil || strings.TrimSpace(string(out)) != "ok\n50" {
		t.Fatalf("snapshot not coherent: %q %v", out, err)
	}
	if names := dirNames(t, staging); strings.Join(names, ",") != "state_5.sqlite" {
		t.Fatalf("opening the snapshot created %v; it must be self-contained", names)
	}
}

func TestSQLiteSnapshotCopiesWALFirstAndRefusesAChangedSource(t *testing.T) {
	src := t.TempDir()
	walDatabase(t, src)
	var order []string
	s := SQLiteSnapshotter{SQLite: "sqlite3", Scratch: t.TempDir(), copyFile: func(from, to string) error {
		order = append(order, filepath.Base(from))
		return copyRegularFile(from, to)
	}}
	if ok, _, err := s.Snapshot(context.Background(), src, "state_5.sqlite", filepath.Join(t.TempDir(), "s")); err != nil || !ok {
		t.Fatalf("snapshot: %v %v", ok, err)
	}
	if strings.Join(order, ",") != "state_5.sqlite-wal,state_5.sqlite" {
		t.Fatalf("copy order %v, want the -wal before the database", order)
	}
	// A checkpoint during the copy changes the source: skip and retry later.
	s.copyFile = func(from, to string) error {
		if err := copyRegularFile(from, to); err != nil {
			return err
		}
		later := time.Now().Add(time.Hour)
		return os.Chtimes(filepath.Join(src, "state_5.sqlite"), later, later)
	}
	staging := filepath.Join(t.TempDir(), "s")
	ok, note, err := s.Snapshot(context.Background(), src, "state_5.sqlite", staging)
	if err != nil || ok || !strings.Contains(note, "changed during copy") {
		t.Fatalf("changed source admitted: ok=%v note=%q err=%v", ok, note, err)
	}
	if _, err := os.Stat(filepath.Join(staging, "state_5.sqlite")); err == nil {
		t.Fatal("unstable copy was published")
	}
}
