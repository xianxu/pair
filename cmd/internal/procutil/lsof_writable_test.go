package procutil

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func fakeWritableLsof(t *testing.T, stdout, script string) string {
	t.Helper()
	dir := t.TempDir()
	data := filepath.Join(dir, "output")
	args := filepath.Join(dir, "args")
	if err := os.WriteFile(data, []byte(stdout), 0600); err != nil {
		t.Fatal(err)
	}
	if script == "" {
		script = "exec /bin/cat \"$LSOF_OUTPUT\""
	}
	body := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$LSOF_ARGS\"\n" + script + "\n"
	if err := os.WriteFile(filepath.Join(dir, "lsof"), []byte(body), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("LSOF_OUTPUT", data)
	t.Setenv("LSOF_ARGS", args)
	return args
}

func TestLsofWritableNamesAccess(t *testing.T) {
	args := fakeWritableLsof(t, "p42\nfcwd\ntDIR\nn/work\nftxt\ntREG\nn/app\nf1\naw\ntREG\nn/work/write.jsonl\nf2\nau\ntREG\nn/work/update with spaces.jsonl\nf3\nar\ntREG\nn/work/read.jsonl\nf4\naw\ntFIFO\nn/pipe\n", "")
	got, err := LsofWritableNames(context.Background(), "42")
	want := []string{"/work/write.jsonl", "/work/update with spaces.jsonl"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("names=%q err=%v, want %q", got, err, want)
	}
	raw, err := os.ReadFile(args)
	if err != nil || string(raw) != "-nP\n-a\n-p\n42\n-Fpfatn\n" {
		t.Fatalf("unexpected command args: %q, %v", raw, err)
	}
}

func TestLsofWritableNamesDarwinUnnamedPolicyDescriptor(t *testing.T) {
	// Captured field shape from a Darwin Codex engine: NPOLICY has a name
	// field with no value. It conveys no regular-file authority.
	fakeWritableLsof(t, "p42\nfcwd\na \ntDIR\nn/work\nftxt\na \ntREG\nn/app\nf34\na \ntNPOLICY\nn\nf49\nau\ntREG\nn/work/rollout.jsonl\n", "")
	got, err := LsofWritableNames(context.Background(), "42")
	if err != nil || !reflect.DeepEqual(got, []string{"/work/rollout.jsonl"}) {
		t.Fatalf("names=%q err=%v", got, err)
	}
}

func TestLsofWritableNamesInvalidPID(t *testing.T) {
	args := fakeWritableLsof(t, "p42\n", "")
	for _, pid := range []string{"", "0", "-1", "42,43", "self", "2147483648"} {
		if got, err := LsofWritableNames(context.Background(), pid); err == nil || got != nil {
			t.Errorf("pid=%q names=%q err=%v", pid, got, err)
		}
	}
	if _, err := os.Stat(args); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid PID invoked lsof: %v", err)
	}
}

func TestLsofWritableNamesRejectsMalformed(t *testing.T) {
	for name, raw := range map[string]string{
		"empty": "", "wrong pid": "p43\n", "partial line": "p42\nf1\naw\ntREG\nn/path",
		"no process": "f1\naw\ntREG\nn/path\n", "missing access": "p42\nf1\ntREG\nn/path\n",
		"missing type": "p42\nf1\naw\nn/path\n", "missing name": "p42\nf1\naw\ntREG\n",
		"unknown field": "p42\nzbad\n", "bad access": "p42\nf1\nax\ntREG\nn/path\n",
		"partial after valid": "p42\nf1\naw\ntREG\nn/path\nf2\n",
		"relative path":       "p42\nf1\naw\ntREG\nnrelative\n",
		"duplicate field":     "p42\nf1\naw\nar\ntREG\nn/path\n",
		"empty descriptor":    "p42\nf\n",
		"empty regular name":  "p42\nf1\naw\ntREG\nn\n",
	} {
		t.Run(name, func(t *testing.T) {
			fakeWritableLsof(t, raw, "")
			if got, err := LsofWritableNames(context.Background(), "42"); err == nil || got != nil {
				t.Fatalf("malformed names=%q err=%v", got, err)
			}
		})
	}
}

func TestLsofWritableNamesEmptySuccessAndCommandFailure(t *testing.T) {
	t.Run("no writable files", func(t *testing.T) {
		fakeWritableLsof(t, "p42\nf1\nar\ntREG\nn/read\n", "")
		if got, err := LsofWritableNames(context.Background(), "42"); err != nil || len(got) != 0 {
			t.Fatalf("names=%q err=%v", got, err)
		}
	})
	t.Run("partial command failure", func(t *testing.T) {
		fakeWritableLsof(t, "p42\nf1\naw\ntREG\nn/write\n", "/bin/cat \"$LSOF_OUTPUT\"\nprintf 'permission denied' >&2\nexit 1")
		if got, err := LsofWritableNames(context.Background(), "42"); err == nil || got != nil || !strings.Contains(err.Error(), "permission denied") {
			t.Fatalf("names=%q err=%v", got, err)
		}
	})
}

func TestLsofWritableNamesCancellation(t *testing.T) {
	fakeWritableLsof(t, "", "exec /bin/sleep 30")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if got, err := LsofWritableNames(ctx, "42"); !errors.Is(err, context.DeadlineExceeded) || got != nil {
		t.Fatalf("names=%q err=%v", got, err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("lsof exceeded caller budget")
	}
}

func TestLsofWritableNamesMaximumDeadline(t *testing.T) {
	fakeWritableLsof(t, "", "exec /bin/sleep 30")
	start := time.Now()
	if got, err := LsofWritableNames(context.Background(), "42"); !errors.Is(err, context.DeadlineExceeded) || got != nil {
		t.Fatalf("names=%q err=%v", got, err)
	}
	if elapsed := time.Since(start); elapsed > 2500*time.Millisecond {
		t.Fatalf("lsof exceeded maximum budget: %v", elapsed)
	}
}

func TestLsofWritableNamesPreCanceled(t *testing.T) {
	args := fakeWritableLsof(t, "p42\n", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := LsofWritableNames(ctx, "42"); !errors.Is(err, context.Canceled) || got != nil {
		t.Fatalf("names=%q err=%v", got, err)
	}
	if _, err := os.Stat(args); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pre-canceled read invoked lsof: %v", err)
	}
}

func TestLsofWritableNamesWarningIsUncertain(t *testing.T) {
	fakeWritableLsof(t, "p42\nf1\naw\ntREG\nn/write\n", "/bin/cat \"$LSOF_OUTPUT\"\nprintf 'cannot stat filesystem' >&2")
	if got, err := LsofWritableNames(context.Background(), "42"); err == nil || got != nil {
		t.Fatalf("partial inspection names=%q err=%v", got, err)
	}
}

func TestLsofWritableNamesOutputLimit(t *testing.T) {
	for _, stream := range []string{"stdout", "stderr"} {
		t.Run(stream, func(t *testing.T) {
			script := "exec /usr/bin/head -c 2097153 /dev/zero"
			if stream == "stderr" {
				script += " >&2"
			}
			fakeWritableLsof(t, "", script)
			if got, err := LsofWritableNames(context.Background(), "42"); err == nil || got != nil || !strings.Contains(err.Error(), "limit") {
				t.Fatalf("names=%q err=%v", got, err)
			}
		})
	}
}

func FuzzParseLsofWritableNames(f *testing.F) {
	for _, raw := range []string{
		"p42\nf1\naw\ntREG\nn/write\n",
		"p42\nf2\nau\ntREG\nn/update with spaces\n",
		"p42\nf1\nar\ntREG\nn/read\n",
		"p42\nfcwd\ntDIR\nn/work\nftxt\ntREG\nn/app\n",
		"p42\nf34\na \ntNPOLICY\nn\nf49\nau\ntREG\nn/write\n",
		"p43\nf1\naw\ntREG\nn/foreign\n", "", "p42\nf1\n",
	} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		names, err := parseLsofWritableNames([]byte(raw), "42")
		if err != nil {
			if names != nil {
				t.Fatal("malformed output retained partial authority")
			}
			return
		}
		for _, name := range names {
			if !filepath.IsAbs(name) {
				t.Fatalf("non-path retained: %q", name)
			}
		}
		if !strings.HasPrefix(raw, "p42\n") {
			t.Fatalf("successful read lacks requested process: %q", raw)
		}
		if got, err := parseLsofWritableNames([]byte(strings.TrimSuffix(raw, "\n")), "42"); err == nil || got != nil {
			t.Fatal("truncated output became success")
		}
		if got, err := parseLsofWritableNames([]byte(raw), "43"); err == nil || got != nil {
			t.Fatal("foreign process became success")
		}
		readOnly := strings.NewReplacer("\naw\n", "\nar\n", "\nau\n", "\nar\n").Replace(raw)
		if got, err := parseLsofWritableNames([]byte(readOnly), "42"); err != nil || len(got) != 0 {
			t.Fatalf("read-only authority retained: %q, %v", got, err)
		}
		nonRegular := strings.ReplaceAll(raw, "\ntREG\n", "\ntDIR\n")
		if got, err := parseLsofWritableNames([]byte(nonRegular), "42"); err != nil || len(got) != 0 {
			t.Fatalf("non-regular authority retained: %q, %v", got, err)
		}
	})
}
