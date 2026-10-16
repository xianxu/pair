package launcher

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	si "github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// The writer holds a committed WAL stable while the fixture copies DB + WAL.
// Closing it can checkpoint its own source; it cannot touch the separate copies.
func copyCodexWALFixture(t *testing.T, python, source, target, id, transcript, cwd, view string) {
	t.Helper()
	const script = `import sqlite3,sys
c=sqlite3.connect(sys.argv[1])
c.execute('PRAGMA journal_mode=WAL')
c.execute('PRAGMA wal_autocheckpoint=0')
table='backing' if sys.argv[5] else 'threads'
c.execute('CREATE TABLE '+table+'(id TEXT,rollout_path TEXT,updated_at_ms INTEGER,archived INTEGER,preview TEXT,cwd TEXT,source TEXT,model_provider TEXT)')
c.execute("INSERT INTO "+table+" VALUES(?,?,100000,0,'visible',?,'cli','openai')",sys.argv[2:5])
if sys.argv[5]:c.execute(sys.argv[5])
c.commit()
print('ready',flush=True)
sys.stdin.read(1)
c.close()
`
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	writer := exec.CommandContext(ctx, python, "-u", "-c", script, source, id, transcript, cwd, view)
	var stderr bytes.Buffer
	writer.Stderr = &stderr
	input, err := writer.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := writer.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		input.Close()
		if err := writer.Wait(); err != nil {
			t.Errorf("private WAL writer: %v %s", err, &stderr)
		}
	}()
	line, err := bufio.NewReader(output).ReadString('\n')
	if err != nil || line != "ready\n" {
		t.Fatalf("private WAL writer readiness: %q %v %s", line, err, &stderr)
	}
	for _, suffix := range []string{"", "-wal"} {
		raw, err := os.ReadFile(source + suffix)
		if err != nil || len(raw) == 0 {
			t.Fatalf("private WAL copy %q: bytes=%d %v", suffix, len(raw), err)
		}
		if err := os.WriteFile(target+suffix, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Lstat(target + "-shm"); !os.IsNotExist(err) {
		t.Fatalf("fixture unexpectedly has SHM: %v", err)
	}
}

type codexWALFileSignature struct {
	Size, Modified int64
	Mode           fs.FileMode
	Device, Inode  uint64
	Hash           [sha256.Size]byte
}

func codexWALTree(t *testing.T, root string) map[string]codexWALFileSignature {
	t.Helper()
	out := map[string]codexWALFileSignature{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("private fixture stat unavailable")
		}
		signature := codexWALFileSignature{Size: info.Size(), Modified: info.ModTime().UnixNano(), Mode: info.Mode(), Device: uint64(stat.Dev), Inode: stat.Ino}
		if info.Mode().IsRegular() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			signature.Hash = sha256.Sum256(raw)
		} else if !info.IsDir() {
			return fmt.Errorf("unexpected nonregular private fixture")
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[relative] = signature
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestCodexResumeOSWALObservationNeverMutatesNativeFiles(t *testing.T) {
	sqlite, err := exec.LookPath("sqlite3")
	if err != nil {
		t.Skip("sqlite3 unavailable")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 unavailable")
	}
	for _, mode := range []string{"tui", "exec"} {
		for _, outcome := range []string{"success", "query-error", "query-cancel", "hostile-init", "view-readfile", "view-writefile"} {
			t.Run(mode+"/"+outcome, func(t *testing.T) {
				root, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				native := filepath.Join(root, "native")
				home := filepath.Join(root, "home")
				codex := filepath.Join(native, "codex")
				state := filepath.Join(native, "state")
				cwd := filepath.Join(root, "work")
				scratch := filepath.Join(root, "scratch")
				bin := filepath.Join(root, "bin")
				source := filepath.Join(root, "writer")
				system := filepath.Join(root, "system")
				sessions := filepath.Join(codex, "sessions", "2026", "10", "15")
				for _, path := range []string{home, state, cwd, scratch, bin, source, system, sessions} {
					if err := os.MkdirAll(path, 0700); err != nil {
						t.Fatal(err)
					}
				}
				t.Setenv("HOME", home)
				t.Setenv("CODEX_HOME", codex)
				t.Setenv("CODEX_SQLITE_HOME", "")
				t.Setenv("CODEX_EXEC_SERVER_URL", "")
				t.Setenv("TMPDIR", scratch)
				if err := os.WriteFile(filepath.Join(codex, "config.toml"), []byte("forced_login_method='api'\nsqlite_home='../state'\n[features]\nworktrees=false\n"), 0600); err != nil {
					t.Fatal(err)
				}
				const id = "019d1111-1111-7111-8111-111111111111"
				transcript := filepath.Join(sessions, "rollout-wal-"+id+".jsonl")
				if err := os.WriteFile(transcript, []byte(nativeValidCodexSessionMeta(id, cwd)+nativeValidCodexTurnContext(cwd)), 0600); err != nil {
					t.Fatal(err)
				}
				db := filepath.Join(state, "state_5.sqlite")
				attackMarker := filepath.Join(root, "forbidden-marker")
				var view string
				quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
				if outcome == "hostile-init" {
					rc := fmt.Sprintf(".once %q\n.print hostile initialization ran\n.output stdout\n", attackMarker)
					if err := os.WriteFile(filepath.Join(home, ".sqliterc"), []byte(rc), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if outcome == "view-readfile" || outcome == "view-writefile" {
					expression := "CASE WHEN writefile(" + quote(attackMarker) + ",'unsafe')=6 THEN rollout_path ELSE '' END"
					if outcome == "view-readfile" {
						payload := filepath.Join(root, "private-view-path")
						if err := os.WriteFile(payload, []byte(transcript), 0600); err != nil {
							t.Fatal(err)
						}
						expression = "CAST(readfile(" + quote(payload) + ") AS TEXT)"
					}
					view = "CREATE VIEW threads AS SELECT id," + expression + " AS rollout_path,updated_at_ms,archived,preview,cwd,source,model_provider FROM backing"
				}
				copyCodexWALFixture(t, python, filepath.Join(source, "state_5.sqlite"), db, id, transcript, cwd, view)
				before := codexWALTree(t, native)
				record := filepath.Join(root, "query-path")
				// The ordinary process seam records the actual SQLite target. Error/cancel
				// occur after snapshot construction, so cleanup cannot pass by doing no work.
				wrapper := "#!" + python + "\n" + `import json,os,sys,time
from pathlib import Path
Path(os.environ['PAIR_WAL_QUERY_RECORD']+'.args').write_text(json.dumps(sys.argv[1:]))
record=Path(os.environ['PAIR_WAL_QUERY_RECORD'])
pending=record.with_suffix('.pending')
pending.write_text(sys.argv[-2])
os.replace(pending,record)
mode=os.environ['PAIR_WAL_QUERY_MODE']
if mode=='query-error':sys.exit(43)
if mode=='query-cancel':
 while True:time.sleep(.05)
os.execv(os.environ['PAIR_WAL_REAL_SQLITE'],[os.environ['PAIR_WAL_REAL_SQLITE'],*sys.argv[1:]])
`
				if err := os.WriteFile(filepath.Join(bin, "sqlite3"), []byte(wrapper), 0700); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
				t.Setenv("PAIR_WAL_QUERY_RECORD", record)
				t.Setenv("PAIR_WAL_QUERY_MODE", outcome)
				t.Setenv("PAIR_WAL_REAL_SQLITE", sqlite)
				reader := si.CodexConfigReader{SystemDir: system, ManagedPolicy: func() (si.CodexManagedPolicyStatus, error) { return si.CodexManagedPolicyAbsent, nil }}
				runtime := OSRuntime{DataDir: filepath.Join(root, "pair"), codexConfigReader: &reader}
				args := []string{"resume", "--last"}
				if mode == "exec" {
					args = append([]string{"exec"}, args...)
				}
				selector, err := ParseNativeResumeSelector("codex", args)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				type observation struct {
					result NativeResumeResolution
					err    error
				}
				done := make(chan observation, 1)
				go func() {
					result, err := runtime.ResolveNativeResume(ctx, NativeResumeRequest{Agent: "codex", CWD: cwd, Selector: selector})
					done <- observation{result, err}
				}()
				if outcome == "query-cancel" {
					ticker := time.NewTicker(10 * time.Millisecond)
					defer ticker.Stop()
					for {
						if _, err := os.Stat(record); err == nil {
							break
						}
						select {
						case <-ticker.C:
						case <-ctx.Done():
							t.Fatal("private SQLite command did not reach cancellation boundary")
						}
					}
					cancel()
				}
				var observed observation
				select {
				case observed = <-done:
				case <-ctx.Done():
					select {
					case observed = <-done:
					case <-time.After(3 * time.Second):
						t.Fatal("private query did not stop after cancellation")
					}
				}
				if outcome == "success" || outcome == "hostile-init" {
					if observed.err != nil || observed.result.Status != si.NativeSelectorResolved || observed.result.NativeID != id {
						t.Fatalf("WAL session not resolved: %+v %v", observed.result, observed.err)
					}
				} else if observed.result.Status == si.NativeSelectorResolved || observed.result.NativeID != "" {
					t.Fatalf("failed query gained authority: %+v %v", observed.result, observed.err)
				}
				queried, err := os.ReadFile(record)
				if err != nil {
					t.Fatalf("SQLite target was not observed: %v", err)
				}
				rawArgs, err := os.ReadFile(record + ".args")
				if err != nil {
					t.Fatal(err)
				}
				var queryArgs []string
				if err := json.Unmarshal(rawArgs, &queryArgs); err != nil || len(queryArgs) < 2 {
					t.Fatalf("private query argument observation unavailable: %v", err)
				}
				flags := queryArgs[:len(queryArgs)-2]
				init := slices.Index(flags, "-init")
				if !slices.Contains(flags, "-batch") || !slices.Contains(flags, "-safe") || !slices.Contains(flags, "-readonly") || init < 0 || init+1 >= len(flags) || flags[init+1] != os.DevNull {
					t.Error("SQLite command omitted hermetic batch/null-init/safe/readonly policy before database")
				}
				relative, err := filepath.Rel(scratch, string(queried))
				if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) || filepath.IsAbs(relative) {
					t.Errorf("SQLite query escaped private snapshot directory")
				}
				if _, err := os.Lstat(attackMarker); !os.IsNotExist(err) {
					t.Errorf("SQLite initialization or view wrote private forbidden marker: %v", err)
				}
				after := codexWALTree(t, native)
				if !reflect.DeepEqual(before, after) {
					newSHM := after[filepath.Join("state", "state_5.sqlite-shm")]
					t.Errorf("native SQLite observation changed files/bytes/inodes/size/mtime; resulting SHM bytes=%d", newSHM.Size)
				}
				residue, err := os.ReadDir(scratch)
				if err != nil || len(residue) != 0 {
					t.Errorf("private SQLite snapshot not cleaned: entries=%d err=%v", len(residue), err)
				}
			})
		}
	}
}
