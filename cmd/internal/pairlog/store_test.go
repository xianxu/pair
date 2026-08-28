package pairlog

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestPersistSessionLog(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "log-work.md")
	now := time.Date(2026, 8, 28, 12, 34, 56, 0, time.UTC)
	if err := PersistSessionLog(path, []byte("first\nline"), now); err != nil {
		t.Fatal(err)
	}
	if err := PersistSessionLog(path, []byte("second"), now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "## 2026-08-28 12:34:56\n<!-- pair-log-v1 bytes=10 -->\n\nfirst\nline\n\n---\n\n" +
		"## 2026-08-28 12:34:57\n<!-- pair-log-v1 bytes=6 -->\n\nsecond\n\n---\n\n"
	if string(raw) != want {
		t.Fatalf("log = %q, want %q", raw, want)
	}
	if info, err := os.Stat(path + ".lock"); err != nil || info.IsDir() {
		t.Fatalf("durable lock file missing: info=%v err=%v", info, err)
	}
}

func TestPersistSessionLogRoundTripsArbitraryMarkdown(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "log-work.md")
	body := []byte("before\n\n---\n\n## 2026-08-28 01:02:03\n\nafter")
	if err := PersistSessionLog(path, body, time.Date(2026, 8, 28, 12, 34, 56, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed := sessioninventory.ParsePairLog(raw, 0)
	if len(parsed.MalformedOffsets) != 0 || len(parsed.Facts) != 1 || parsed.Facts[0].Text != string(body) {
		t.Fatalf("parsed=%#v raw=%q", parsed, raw)
	}
}

func TestPersistSessionLogConcurrentProcessesLoseNoEntries(t *testing.T) {
	if os.Getenv("PAIRLOG_HELPER") != "" {
		path := os.Getenv("PAIRLOG_PATH")
		body := os.Getenv("PAIRLOG_BODY")
		if err := PersistSessionLog(path, []byte(body), time.Unix(0, 0).UTC()); err != nil {
			t.Fatal(err)
		}
		return
	}
	path := filepath.Join(t.TempDir(), "log.md")
	const writers = 12
	cmds := make([]*exec.Cmd, writers)
	outputs := make([]bytes.Buffer, writers)
	for i := range cmds {
		cmds[i] = exec.Command(os.Args[0], "-test.run=^TestPersistSessionLogConcurrentProcessesLoseNoEntries$")
		cmds[i].Env = append(os.Environ(), "PAIRLOG_HELPER=1", "PAIRLOG_PATH="+path, "PAIRLOG_BODY=entry-"+string(rune('a'+i)))
		cmds[i].Stdout = &outputs[i]
		cmds[i].Stderr = &outputs[i]
		if err := cmds[i].Start(); err != nil {
			t.Fatal(err)
		}
	}
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("helper: %v: %s", err, outputs[i].Bytes())
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < writers; i++ {
		body := "entry-" + string(rune('a'+i))
		if strings.Count(string(raw), "\n\n"+body+"\n\n---\n\n") != 1 {
			t.Errorf("%q count != 1 in %q", body, raw)
		}
	}
}

func TestPersistSessionLogFailuresPreserveExistingLog(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"read", "open", "write", "sync", "rename"} {
		failure := failure
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "log.md")
			if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
				t.Fatal(err)
			}
			store := SessionLogStore{Runtime: failingRuntime{Runtime: OSRuntime{}, fail: failure}}
			if err := store.Persist(path, []byte("new"), time.Time{}); err == nil {
				t.Fatal("Persist returned nil error")
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(raw) != "existing" {
				t.Fatalf("failed append changed log to %q", raw)
			}
		})
	}
}

type failingRuntime struct {
	Runtime
	fail string
}

func (f failingRuntime) CreateTemp(dir, pattern string) (File, error) {
	if f.fail == "open" {
		return nil, os.ErrPermission
	}
	file, err := f.Runtime.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return failingFile{File: file, fail: f.fail}, nil
}

func (f failingRuntime) ReadFile(path string) ([]byte, error) {
	if f.fail == "read" {
		return nil, os.ErrPermission
	}
	return f.Runtime.ReadFile(path)
}

func (f failingRuntime) Rename(oldPath, newPath string) error {
	if f.fail == "rename" {
		return os.ErrPermission
	}
	return f.Runtime.Rename(oldPath, newPath)
}

type failingFile struct {
	File
	fail string
}

func (f failingFile) Write(content []byte) (int, error) {
	if f.fail == "write" {
		return 0, os.ErrPermission
	}
	return f.File.Write(content)
}

func (f failingFile) Sync() error {
	if f.fail == "sync" {
		return os.ErrPermission
	}
	return f.File.Sync()
}
