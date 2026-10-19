package historyarchive

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/execalarm"
	"io"
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
	result, err := s.Snapshot(context.Background(), src, "state_5.sqlite", staging)
	if err != nil || result.Status != SnapshotReady {
		t.Fatalf("snapshot: result=%+v err=%v", result, err)
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

// fakeSQLiteRunner models a pinned source version and a writer that advances
// while that view remains stable; validation is a distinct later operation.
type fakeSQLiteRunner struct {
	generation, pinned, reads, validations int
	fail                                   error
	corrupt, replace                       bool
	cancel                                 context.CancelFunc
}

func (f *fakeSQLiteRunner) Run(ctx context.Context, req SQLiteRequest) (SQLiteRunResult, error) {
	if req.Lease == nil {
		return SQLiteRunResult{}, errors.New("no lifetime lease")
	}
	if req.Purpose == SQLiteAcquire {
		f.reads++
	} else {
		f.validations++
	}
	if f.fail != nil {
		return SQLiteRunResult{Stderr: "database is locked"}, f.fail
	}
	if req.Purpose == SQLiteAcquire {
		f.pinned = f.generation
		f.generation++
		if f.cancel != nil {
			f.cancel()
			return SQLiteRunResult{}, ctx.Err()
		}
		if f.replace {
			os.Remove(req.Database)
			os.WriteFile(req.Database, []byte("replacement"), 0600)
		}
		return SQLiteRunResult{}, os.WriteFile(filepath.Join(req.Work, "snapshot.sqlite"), []byte(fmt.Sprintf("version %d", f.pinned)), 0600)
	}
	if f.corrupt {
		return SQLiteRunResult{Stdout: "delete\ncorrupt\n"}, nil
	}
	return SQLiteRunResult{Stdout: "delete\nok\n"}, nil
}

func TestSQLiteSnapshotProgressDespiteChangingSource(t *testing.T) {
	src := t.TempDir()
	db := filepath.Join(src, "state_5.sqlite")
	os.WriteFile(db, []byte("source"), 0600)
	fake := &fakeSQLiteRunner{}
	s := SQLiteSnapshotter{Scratch: t.TempDir(), Runner: fake}
	for i := 0; i < 2; i++ {
		staging := t.TempDir()
		result, err := s.Snapshot(context.Background(), src, "state_5.sqlite", staging)
		if err != nil || result.Status != SnapshotReady {
			t.Fatalf("active source starved: %+v %v", result, err)
		}
		body, _ := os.ReadFile(filepath.Join(staging, "state_5.sqlite"))
		if string(body) != fmt.Sprintf("version %d", i) {
			t.Fatalf("wrong pinned version %q", body)
		}
	}
	if fake.reads != 2 || fake.validations != 2 || fake.generation != 2 {
		t.Fatalf("unbounded or unordered calls: %+v", fake)
	}
}

func TestSQLiteSnapshotRejectsPartialAndReplacedCandidates(t *testing.T) {
	for _, tc := range []struct {
		name string
		fake fakeSQLiteRunner
		want SnapshotStatus
	}{
		{"tool", fakeSQLiteRunner{fail: errors.New("exit 1")}, SnapshotRejected},
		{"deadline", fakeSQLiteRunner{fail: ErrSQLiteDeadline}, SnapshotDeferred},
		{"integrity", fakeSQLiteRunner{corrupt: true}, SnapshotRejected},
		{"replaced", fakeSQLiteRunner{replace: true}, SnapshotRejected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src, staging := t.TempDir(), t.TempDir()
			os.WriteFile(filepath.Join(src, "db.sqlite"), []byte("source"), 0600)
			dest := filepath.Join(staging, "db.sqlite")
			os.WriteFile(dest, []byte("old"), 0600)
			s := SQLiteSnapshotter{Scratch: t.TempDir(), Runner: &tc.fake}
			result, err := s.Snapshot(context.Background(), src, "db.sqlite", staging)
			if err != nil || result.Status != tc.want {
				t.Fatalf("%+v %v", result, err)
			}
			body, _ := os.ReadFile(dest)
			if string(body) != "old" {
				t.Fatalf("old staging replaced: %q", body)
			}
			if tc.fake.reads > 1 || tc.fake.validations > 1 {
				t.Fatal("retried")
			}
		})
	}
}

func TestSQLiteSnapshotCancellationNeverPublishes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "db.sqlite"), []byte("source"), 0600)
	s := SQLiteSnapshotter{Scratch: t.TempDir(), Runner: &fakeSQLiteRunner{cancel: cancel}}
	_, err := s.Snapshot(ctx, src, "db.sqlite", t.TempDir())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel swallowed: %v", err)
	}
	names, _ := os.ReadDir(s.Scratch)
	if len(names) != 0 {
		t.Fatal("private work leaked")
	}
}

func TestSQLiteSnapshotRefusesLinkedAndSpecialSourcePaths(t *testing.T) {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		t.Run(suffix, func(t *testing.T) {
			src := t.TempDir()
			db := filepath.Join(src, "db.sqlite")
			os.WriteFile(db, []byte("source"), 0600)
			if suffix == "" {
				os.Remove(db)
			}
			if err := os.Symlink(filepath.Join(t.TempDir(), "elsewhere"), db+suffix); err != nil {
				t.Fatal(err)
			}
			fake := &fakeSQLiteRunner{}
			result, err := (SQLiteSnapshotter{Scratch: t.TempDir(), Runner: fake}).Snapshot(context.Background(), src, "db.sqlite", t.TempDir())
			if err != nil || result.Status != SnapshotRejected || fake.reads != 0 {
				t.Fatalf("unsafe source admitted: %+v %v", result, err)
			}
		})
	}
}

func TestSQLiteSnapshotUsesLiteralPathsAndPrivatePermissions(t *testing.T) {
	base := filepath.Join(t.TempDir(), "spaces ' quote \" and\nnewline")
	os.MkdirAll(base, 0700)
	walDatabase(t, base)
	staging := filepath.Join(base, "staging")
	result, err := (SQLiteSnapshotter{Scratch: filepath.Join(base, "scratch")}).Snapshot(context.Background(), base, "state_5.sqlite", staging)
	if err != nil || result.Status != SnapshotReady {
		t.Fatalf("literal path backup: %+v %v", result, err)
	}
	info, err := os.Stat(filepath.Join(staging, "state_5.sqlite"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("private output mode: %v %v", info, err)
	}
}

func TestSQLiteSnapshotSkipsOverLimitDatabaseWithoutCopying(t *testing.T) {
	src := t.TempDir()
	f, err := os.Create(filepath.Join(src, "big.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxSnapshotRawBytes + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	fake := &fakeSQLiteRunner{}
	result, err := (SQLiteSnapshotter{Scratch: t.TempDir(), Runner: fake}).Snapshot(context.Background(), src, "big.sqlite", t.TempDir())
	if err != nil || result.Status != SnapshotRejected || fake.reads != 0 {
		t.Fatalf("over limit admitted: %+v %v", result, err)
	}
}

func TestSQLiteSweepPreservesHeldLease(t *testing.T) {
	scratch := t.TempDir()
	work := filepath.Join(scratch, "sqlite-held")
	os.Mkdir(work, 0700)
	lease, err := snapshotLease(filepath.Join(work, "lease"), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := (SQLiteSnapshotter{Scratch: scratch}).Sweep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(work); err != nil {
		t.Fatalf("held work removed: %v", err)
	}
	lease.Close()
	if err := (SQLiteSnapshotter{Scratch: scratch}).Sweep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(work); !os.IsNotExist(err) {
		t.Fatalf("abandoned work retained: %v", err)
	}
}

// A killed pass's private copies are swept; nothing else in scratch is.
func TestSQLiteSnapshotterSweepRemovesLeftoverWorkDirs(t *testing.T) {
	scratch := t.TempDir()
	os.MkdirAll(filepath.Join(scratch, "sqlite-123", "x"), 0o700)
	os.WriteFile(filepath.Join(scratch, "keep.txt"), []byte("k"), 0o600)
	if err := (SQLiteSnapshotter{Scratch: scratch}).Sweep(); err != nil {
		t.Fatal(err)
	}
	if names := dirNames(t, scratch); strings.Join(names, ",") != "keep.txt" {
		t.Fatalf("scratch after sweep: %v", names)
	}
	if err := (SQLiteSnapshotter{Scratch: filepath.Join(scratch, "absent")}).Sweep(); err != nil {
		t.Fatalf("absent scratch: %v", err)
	}
}

func TestSameContentStreams(t *testing.T) {
	dir := t.TempDir()
	a, b, c := filepath.Join(dir, "a"), filepath.Join(dir, "b"), filepath.Join(dir, "c")
	body := strings.Repeat("x", 3<<20+7)
	os.WriteFile(a, []byte(body), 0o600)
	os.WriteFile(b, []byte(body), 0o600)
	os.WriteFile(c, []byte(body[:len(body)-1]+"y"), 0o600)
	if same, err := sameContent(a, b); err != nil || !same {
		t.Fatalf("equal files: %v %v", same, err)
	}
	if same, err := sameContent(a, c); err != nil || same {
		t.Fatalf("different last byte: %v %v", same, err)
	}
}

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "-bail" && os.Getenv("PAIR688_FAKE_SQLITE") != "" {
		input, _ := io.ReadAll(os.Stdin)
		if !strings.Contains(string(input), "BEGIN;\nPRAGMA schema_version;") {
			os.Exit(3)
		}
		if os.Getenv("PAIR688_FAKE_SQLITE") == "blocked" {
			if err := os.WriteFile(os.Getenv("PAIR688_SQLITE_READY"), []byte("ready"), 0600); err != nil {
				os.Exit(4)
			}
			for {
				time.Sleep(time.Hour)
			}
		}
		if os.Getenv("PAIR688_FAKE_SQLITE") == "exit" {
			fmt.Fprintln(os.Stderr, "database is locked")
			os.Exit(1)
		}
		fmt.Fprint(os.Stdout, strings.Repeat("x", 128<<10))
		fmt.Fprint(os.Stderr, strings.Repeat("e", 32<<10))
		os.Exit(0)
	}
	if len(os.Args) == 3 && os.Args[1] == "fixture-sqlite-owner" {
		child := exec.Command(os.Args[0], "internal", "exec-alarm", "5", "--", "sqlite3", "-readonly", "-batch", "-bail", "-init", "/dev/null", os.Args[2])
		child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
		child.ExtraFiles = []*os.File{os.NewFile(3, "lease")}
		if err := child.Run(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	if len(os.Args) > 2 && os.Args[1] == "internal" && os.Args[2] == "exec-alarm" {
		if err := execalarm.Run(os.Args[3:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}
