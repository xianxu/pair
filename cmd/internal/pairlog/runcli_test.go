package pairlog

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunCLIAppendsStdinToScopedLog(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "log.md")
	var stderr bytes.Buffer
	code := RunCLI(nil, strings.NewReader("authored"), func(key string) string {
		if key == "PAIR_LOG_PATH" {
			return path
		}
		return ""
	}, time.Date(2026, 8, 28, 1, 2, 3, 0, time.UTC), &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "\n\nauthored\n\n---\n\n") {
		t.Fatalf("log = %q", raw)
	}
}

func TestRunCLIRejectsMissingScopedLogAndArguments(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{nil, {"extra"}} {
		var stderr bytes.Buffer
		code := RunCLI(args, strings.NewReader("authored"), func(string) string { return "" }, time.Time{}, &stderr)
		if code != 2 || stderr.Len() == 0 {
			t.Errorf("args=%v code=%d stderr=%q", args, code, stderr.String())
		}
	}
}

func TestRunCLIReportsPersistenceFailure(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	code := runCLI(nil, strings.NewReader("authored"), func(string) string { return "/log.md" }, time.Time{}, &stderr,
		func(string, []byte, time.Time) error { return errors.New("boom") })
	if code != 1 || !strings.Contains(stderr.String(), "boom") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}
