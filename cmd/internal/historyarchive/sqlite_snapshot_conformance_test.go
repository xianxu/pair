package historyarchive

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The owning SQLite process never performs raw descriptor inspection of its
// database. In-process raw closes would cancel SQLite's POSIX advisory locks.
const activeSQLiteWriter = `import sqlite3,sys,threading
c=sqlite3.connect(sys.argv[1]); c.execute('PRAGMA journal_mode=WAL'); c.execute('PRAGMA wal_autocheckpoint=0')
c.executescript('CREATE TABLE invariant(k INTEGER PRIMARY KEY,v INTEGER); INSERT INTO invariant VALUES(1,0),(2,0); CREATE TABLE future_unknown(payload BLOB, future_column TEXT); INSERT INTO future_unknown(rowid,payload,future_column) VALUES(7,zeroblob(32768),"opaque"),(901,zeroblob(32768),"future");')
c.executemany('INSERT INTO future_unknown(payload,future_column) VALUES(zeroblob(32768),?)', [('future',)]*256); c.commit()
stop=threading.Event(); state=[0]
def write():
 w=sqlite3.connect(sys.argv[1]); w.execute('PRAGMA wal_autocheckpoint=0')
 while not stop.is_set():
  n=state[0]+1; w.execute('BEGIN IMMEDIATE'); w.execute('UPDATE invariant SET v=CASE k WHEN 1 THEN ? ELSE ? END',(n,-n)); w.commit(); state[0]=n
 w.close()
thread=None
print('ready',flush=True)
for command in sys.stdin:
 if command.strip()=='start':
  thread=threading.Thread(target=write); thread.start(); print('writing',flush=True)
 elif command.strip()=='count': print(state[0],flush=True)
 elif command.strip()=='stop':
  stop.set()
  if thread: thread.join()
  print(state[0],flush=True); break
c.close()
`

func TestSQLiteSnapshotRealActiveWriterPreservesOpaqueSchemaAndRowids(t *testing.T) {
	src := t.TempDir()
	db := filepath.Join(src, "state_5.sqlite")
	writer := exec.Command("python3", "-u", "-c", activeSQLiteWriter, db)
	in, err := writer.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := writer.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	writer.Stderr = os.Stderr
	if err := writer.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { in.Close(); writer.Wait() }()
	reader := bufio.NewReader(out)
	read := func() string {
		t.Helper()
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(line)
	}
	if read() != "ready" {
		t.Fatal("writer not ready")
	}
	fmt.Fprintln(in, "start")
	if read() != "writing" {
		t.Fatal("writer not running")
	}
	fmt.Fprintln(in, "count")
	before, err := strconv.Atoi(read())
	if err != nil {
		t.Fatal(err)
	}
	staging := t.TempDir()
	result, err := (SQLiteSnapshotter{SQLite: "sqlite3", Scratch: t.TempDir()}).Snapshot(context.Background(), src, "state_5.sqlite", staging)
	fmt.Fprintln(in, "stop")
	after, parseErr := strconv.Atoi(read())
	if parseErr != nil || after <= before {
		t.Fatalf("writer did not progress: before=%d after=%d err=%v", before, after, parseErr)
	}
	if err != nil || result.Status != SnapshotReady {
		t.Fatalf("active snapshot %+v %v; commits %d", result, err, after-before)
	}
	output, err := exec.Command("sqlite3", filepath.Join(staging, "state_5.sqlite"), "PRAGMA integrity_check; SELECT sum(v) FROM invariant; SELECT rowid||':'||future_column||':'||length(payload) FROM future_unknown WHERE rowid IN (7,901) ORDER BY rowid;").CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "ok\n0\n7:opaque:32768\n901:future:32768" {
		t.Fatalf("opaque snapshot %q %v", output, err)
	}
	t.Logf("coherent opaque snapshot while writer committed %d transactions", after-before)
}

func TestSQLiteSnapshotQuiescentDatabaseAndWALBytesUnchanged(t *testing.T) {
	src := t.TempDir()
	db := walDatabase(t, src)
	type fact struct {
		info os.FileInfo
		hash [32]byte
	}
	before := map[string]fact{}
	for _, path := range []string{db, db + "-wal"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		before[path] = fact{info, sha256.Sum256(data)}
	}
	result, err := (SQLiteSnapshotter{Scratch: t.TempDir()}).Snapshot(context.Background(), src, filepath.Base(db), t.TempDir())
	if err != nil || result.Status != SnapshotReady {
		t.Fatalf("%+v %v", result, err)
	}
	for path, old := range before {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !os.SameFile(old.info, info) || old.hash != sha256.Sum256(data) || info.Mode() != old.info.Mode() {
			t.Fatalf("native DB/WAL changed: %s", path)
		}
	}
}

func TestSQLiteOwnerDeathKeepsScratchUntilKernelAlarm(t *testing.T) {
	src, scratch := t.TempDir(), t.TempDir()
	db := walDatabase(t, src)
	work := filepath.Join(scratch, "sqlite-orphan")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	lease, err := snapshotLease(filepath.Join(work, "lease"), true)
	if err != nil {
		t.Fatal(err)
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	owner := exec.Command(os.Args[0], "fixture-sqlite-owner", db)
	owner.Stdin = r
	owner.ExtraFiles = []*os.File{lease}
	out, err := owner.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	owner.Stderr = os.Stderr
	if err := owner.Start(); err != nil {
		t.Fatal(err)
	}
	r.Close()
	lease.Close()
	defer func() { owner.Process.Kill(); owner.Wait() }()
	if _, err := io.WriteString(w, "BEGIN;\nPRAGMA schema_version;\n.print pinned\n"); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(out)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(line) == "pinned" {
			break
		}
	}
	if err := owner.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := (SQLiteSnapshotter{Scratch: scratch}).Sweep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(work); err != nil {
		t.Fatalf("live orphan scratch removed: %v", err)
	}
	// Input remains open. Only the child's independent alarm ends its read pin.
	if rest, err := io.ReadAll(reader); err != nil || len(rest) != 0 {
		t.Fatalf("orphan output %q %v", rest, err)
	}
	if err := (SQLiteSnapshotter{Scratch: scratch}).Sweep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(work); !os.IsNotExist(err) {
		t.Fatalf("released orphan scratch retained: %v", err)
	}
}
