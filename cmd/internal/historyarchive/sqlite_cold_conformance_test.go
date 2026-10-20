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
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSQLiteClosedWALNativeConformance(t *testing.T) {
	tools := []string{"sqlite3"}
	if runtime.GOOS == "darwin" {
		tools = append(tools, "/usr/bin/sqlite3")
	}
	for _, tool := range tools {
		t.Run(strings.ReplaceAll(tool, "/", "_"), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			base, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			src, scratch, staging := filepath.Join(base, "source"), filepath.Join(base, "scratch"), filepath.Join(base, "staging")
			for _, dir := range []string{src, scratch, staging} {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			db := filepath.Join(src, "future.sqlite")
			fixture := `import sqlite3,sys
c=sqlite3.connect(sys.argv[1])
assert c.execute('PRAGMA journal_mode=WAL').fetchone()[0]=='wal'
c.executescript("CREATE TABLE future_unknown(payload BLOB, future_column TEXT); INSERT INTO future_unknown(rowid,payload,future_column) VALUES(7,X'0001FF','opaque'),(901,X'FE00','future'); PRAGMA user_version=12345;")
c.commit(); c.close()
`
			if out, err := exec.CommandContext(ctx, "python3", "-c", fixture, db).CombinedOutput(); err != nil {
				t.Fatalf("create closed WAL fixture: %s: %v", out, err)
			}
			before := cold701SourceImage(t, src)
			if len(before) != 1 {
				t.Fatalf("closed fixture unexpectedly has sidecars: %v", before)
			}
			version, err := exec.CommandContext(ctx, tool, "--version").CombinedOutput()
			if err != nil {
				t.Fatalf("SQLite version: %s: %v", version, err)
			}
			probeRoot, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			probeDB := filepath.Join(probeRoot, "probe.sqlite")
			if err := os.WriteFile(probeDB, before[filepath.Base(db)].data, 0600); err != nil {
				t.Fatal(err)
			}
			probe, probeErr := exec.CommandContext(ctx, tool, "-bail", "-batch", "-init", "/dev/null", "-nofollow", "-readonly", probeDB, "PRAGMA schema_version;").CombinedOutput()
			t.Logf("native SQLite %s: readonly closed-WAL probe error=%v output=%q", strings.TrimSpace(string(version)), probeErr, probe)
			if probeErr != nil && !strings.Contains(string(probe), "unable to open database file") {
				t.Fatalf("unexpected closed-WAL readonly refusal: %q %v", probe, probeErr)
			}
			cold701AssertSourceImage(t, src, before)
			result, err := (SQLiteSnapshotter{SQLite: tool, Scratch: scratch}).Snapshot(ctx, src, filepath.Base(db), staging)
			cold701AssertSourceImage(t, src, before)
			entries, readErr := os.ReadDir(scratch)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("snapshot scratch not cleaned: %v %v", entries, readErr)
			}
			if err != nil || result.Status != SnapshotReady {
				t.Fatalf("closed WAL snapshot must succeed: %+v err=%v", result, err)
			}
			outputDB := filepath.Join(staging, filepath.Base(db))
			query := "PRAGMA integrity_check; PRAGMA journal_mode; PRAGMA user_version; SELECT sql FROM sqlite_schema WHERE name='future_unknown'; SELECT rowid||':'||hex(payload)||':'||future_column FROM future_unknown ORDER BY rowid;"
			out, err := exec.CommandContext(ctx, tool, "-batch", "-bail", "-init", "/dev/null", "-readonly", outputDB, query).CombinedOutput()
			want := "ok\ndelete\n12345\nCREATE TABLE future_unknown(payload BLOB, future_column TEXT)\n7:0001FF:opaque\n901:FE00:future"
			if err != nil || strings.TrimSpace(string(out)) != want {
				t.Fatalf("opaque schema/rowids/BLOB changed: %q err=%v", out, err)
			}
			published, err := os.ReadDir(staging)
			if err != nil || len(published) != 1 || published[0].Name() != filepath.Base(db) {
				t.Fatalf("snapshot is not self-contained: %v %v", published, err)
			}
			cold701AssertSourceImage(t, src, before)
		})
	}
}

// The gate advances a real SQLite writer only after a nonempty read, so the
// capture observes an actual descriptor and actual filesystem mutation evidence.
type cold701WriterGate struct {
	coldSQLiteFile
	pass, target int
	fired        bool
	write        func() error
	writeErr     error
}

func (g *cold701WriterGate) Seek(off int64, whence int) (int64, error) {
	if off == 0 && whence == io.SeekStart {
		g.pass++
	}
	return g.coldSQLiteFile.Seek(off, whence)
}

func (g *cold701WriterGate) Read(p []byte) (int, error) {
	n, err := g.coldSQLiteFile.Read(p)
	if n > 0 && g.pass == g.target && !g.fired {
		g.fired = true
		g.writeErr = g.write()
		if g.writeErr != nil {
			return n, g.writeErr
		}
	}
	return n, err
}

func TestSQLiteClosedWALRealWriterDuringCapture(t *testing.T) {
	for _, pass := range []int{1, 2} {
		for _, action := range []string{"commit", "rollback"} {
			t.Run(fmt.Sprintf("%s/pass%d", action, pass), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				src := filepath.Join(base, "source")
				if err := os.Mkdir(src, 0700); err != nil {
					t.Fatal(err)
				}
				db := filepath.Join(src, "future.sqlite")
				fixture := `import sqlite3,sys
c=sqlite3.connect(sys.argv[1]); c.execute('PRAGMA journal_mode=WAL')
c.executescript("CREATE TABLE future_unknown(payload BLOB, future_column TEXT); INSERT INTO future_unknown(rowid,payload,future_column) VALUES(7,X'0001FF','opaque'),(901,X'FE00','future');")
c.commit(); c.close()
`
				if out, err := exec.CommandContext(ctx, "python3", "-c", fixture, db).CombinedOutput(); err != nil {
					t.Fatalf("create: %s %v", out, err)
				}
				file, err := os.Open(db)
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				parent, err := os.Open(src)
				if err != nil {
					t.Fatal(err)
				}
				defer parent.Close()
				actual := coldSQLiteFile{File: file, parent: parent, path: db}
				before, err := actual.Observe()
				if err != nil || before.Sidecars != 0 {
					t.Fatalf("cold precondition: %+v %v", before, err)
				}
				beforeImage := cold701SourceImage(t, src)
				gate := &cold701WriterGate{coldSQLiteFile: actual, target: pass}
				gate.write = func() error {
					script := `import os,sqlite3,sys
p=sys.argv[1]; c=sqlite3.connect(p)
c.execute('BEGIN IMMEDIATE'); c.execute("UPDATE future_unknown SET future_column='writer' WHERE rowid=7")
assert os.path.exists(p+'-wal') and os.path.exists(p+'-shm'), 'writer never created sidecars'
getattr(c,sys.argv[2])(); c.close()
assert not any(os.path.exists(p+s) for s in ('-wal','-shm','-journal')), 'writer left sidecars'
print('sidecars-created-and-removed')
`
					out, err := exec.CommandContext(ctx, "python3", "-c", script, db, action).CombinedOutput()
					if err != nil || strings.TrimSpace(string(out)) != "sidecars-created-and-removed" {
						return fmt.Errorf("writer protocol: %q %w", out, err)
					}
					return nil
				}
				candidate := filepath.Join(base, "candidate.sqlite")
				handled, captureErr := copyColdWAL(ctx, gate, candidate)
				if !gate.fired || gate.writeErr != nil {
					t.Fatalf("writer gate not exercised: fired=%v err=%v", gate.fired, gate.writeErr)
				}
				if !handled || !errors.Is(captureErr, errColdCapture) {
					t.Fatalf("writer admitted: handled=%v err=%v", handled, captureErr)
				}
				if _, err := os.Lstat(candidate); !os.IsNotExist(err) {
					t.Fatalf("uncertain candidate retained: %v", err)
				}
				after, err := actual.Observe()
				if err != nil || after.Sidecars != 0 || before.Parent == after.Parent {
					t.Fatalf("transient sidecar evidence missing: before=%+v after=%+v err=%v", before, after, err)
				}
				if action == "rollback" {
					cold701AssertSourceImage(t, src, beforeImage)
					if before.DB != after.DB {
						t.Fatalf("rollback must isolate parent-only mutation: before=%+v after=%+v", before.DB, after.DB)
					}
				}
				verify := `import sqlite3,sys
c=sqlite3.connect(sys.argv[1]); assert c.execute("SELECT sql FROM sqlite_schema WHERE name='future_unknown'").fetchone()[0]=='CREATE TABLE future_unknown(payload BLOB, future_column TEXT)'
want='writer' if sys.argv[2]=='commit' else 'opaque'
assert c.execute('SELECT rowid,hex(payload),future_column FROM future_unknown ORDER BY rowid').fetchall()==[(7,'0001FF',want),(901,'FE00','future')]
c.close()
`
				if out, err := exec.CommandContext(ctx, "python3", "-c", verify, db, action).CombinedOutput(); err != nil {
					t.Fatalf("collector changed opaque source beyond writer transaction: %s %v", out, err)
				}
			})
		}
	}
}

type cold701ImageEntry struct {
	info os.FileInfo
	data []byte
}

func cold701SourceImage(t *testing.T, root string) map[string]cold701ImageEntry {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	image := make(map[string]cold701ImageEntry, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			t.Fatal(err)
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("source image contains nonregular file: %s", entry.Name())
		}
		data, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		image[entry.Name()] = cold701ImageEntry{info, data}
	}
	return image
}

func cold701AssertSourceImage(t *testing.T, root string, before map[string]cold701ImageEntry) {
	t.Helper()
	after := cold701SourceImage(t, root)
	if len(before) != len(after) {
		t.Fatalf("native file set changed: before=%v after=%v", before, after)
	}
	for name, old := range before {
		now, ok := after[name]
		if !ok || !os.SameFile(old.info, now.info) || old.info.Mode() != now.info.Mode() || !old.info.ModTime().Equal(now.info.ModTime()) || !bytes.Equal(old.data, now.data) {
			t.Fatalf("native file identity/mode/mtime/bytes changed: %s", name)
		}
	}
}
