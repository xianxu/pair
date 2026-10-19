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
	"syscall"
	"testing"
	"time"
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
	defer lease.Close()
	input, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer w.Close()
	// Own this pipe independently of Cmd.Wait: killing and reaping the owner
	// must leave the SQLite orphan's output readable until its kernel alarm.
	out, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	defer output.Close()
	owner := exec.Command(os.Args[0], "fixture-sqlite-owner", db)
	owner.Stdin = input
	owner.Stdout = output
	owner.Stderr = os.Stderr
	owner.ExtraFiles = []*os.File{lease}
	if err := owner.Start(); err != nil {
		t.Fatal(err)
	}
	input.Close()
	output.Close()
	lease.Close()
	defer func() { owner.Process.Kill(); owner.Wait() }()
	lines := make(chan string, 8)
	go func() {
		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	next := func() (string, bool) {
		t.Helper()
		select {
		case line, ok := <-lines:
			return line, ok
		case <-time.After(20 * time.Second):
			t.Fatal("SQLite fixture stopped reporting progress")
			return "", false
		}
	}
	pidLine, ok := next()
	if !ok || !strings.HasPrefix(pidLine, "sqlite-pid ") {
		t.Fatalf("missing owned SQLite PID: %q", pidLine)
	}
	pid, err := strconv.Atoi(strings.TrimPrefix(pidLine, "sqlite-pid "))
	if err != nil || pid <= 0 {
		t.Fatalf("invalid SQLite PID receipt: %q", pidLine)
	}
	child, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	childLive := true
	defer func() {
		if childLive {
			_ = child.Kill()
		}
		_ = child.Release()
	}()
	// SQLite's own shell command stops its known process before announcing the
	// pinned phase. Pending SIGALRM cannot release the lease while SIGSTOP holds,
	// however long the root test is descheduled during owner death and Sweep.
	script := fmt.Sprintf("BEGIN;\nPRAGMA schema_version;\n.shell kill -STOP %d; echo pinned\n", pid)
	if _, err := io.WriteString(w, script); err != nil {
		t.Fatal(err)
	}
	for {
		line, ok := next()
		if !ok {
			t.Fatal("SQLite exited before stopped pin receipt")
		}
		if strings.TrimSpace(line) == "pinned" {
			break
		}
	}
	if err := owner.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	err = owner.Wait()
	exit, ok := err.(*exec.ExitError)
	if !ok || !exit.Sys().(syscall.WaitStatus).Signaled() || exit.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
		t.Fatalf("owner not killed by test: %v", err)
	}
	if err := (SQLiteSnapshotter{Scratch: scratch}).Sweep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(work); err != nil {
		t.Fatalf("stopped orphan scratch removed: %v", err)
	}
	if err := child.Signal(syscall.SIGCONT); err != nil {
		t.Fatal(err)
	}
	// Input stays open. With its native read pin resumed, only the inherited
	// kernel alarm ends this SQLite process and closes the root-owned output.
	if line, ok := next(); ok {
		t.Fatalf("unexpected orphan output %q", line)
	}
	childLive = false
	if err := (SQLiteSnapshotter{Scratch: scratch}).Sweep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(work); !os.IsNotExist(err) {
		t.Fatalf("released orphan scratch retained: %v", err)
	}
}
