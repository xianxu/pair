package pairlog

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
)

func TestRunCLIAppendsStdinToScopedLog(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "log.md")
	var stderr bytes.Buffer
	code := RunCLI([]string{"--append-id", "attempt-a"}, strings.NewReader("authored"), func(key string) string {
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

func TestRunCommitCLIMakesPreparedTextCorrelationEligible(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "log.md")
	store := SessionLogStore{Runtime: OSRuntime{}}
	if err := store.PrepareWithID(path, []byte("authored"), time.Time{}, "attempt-a"); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := RunCommitCLI([]string{"--append-id", "attempt-a"}, func(string) string { return path }, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	raw, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(raw), "state=submitted") {
		t.Fatalf("raw=%q err=%v", raw, err)
	}
}

func TestRunCLIRejectsMissingScopedLogAndArguments(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{nil, {"extra"}, {"--append-id", "bad id"}} {
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
	code := runCLI([]string{"--append-id", "attempt-a"}, strings.NewReader("authored"), func(string) string { return "/log.md" }, time.Time{}, &stderr,
		func(string, []byte, time.Time, string) error { return errors.New("boom") })
	if code != 1 || !strings.Contains(stderr.String(), "boom") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestRunCLIConsumesPublicationOutcome(t *testing.T) {
	t.Parallel()
	for _, outcome := range []commitoutcome.Outcome{commitoutcome.Indeterminate, commitoutcome.Committed} {
		outcome := outcome
		t.Run(outcome.String(), func(t *testing.T) {
			var stderr bytes.Buffer
			gotID := ""
			code := runCLI([]string{"--append-id", "attempt-a"}, strings.NewReader("authored"), func(string) string { return "/log.md" }, time.Time{}, &stderr,
				func(_ string, _ []byte, _ time.Time, appendID string) error {
					gotID = appendID
					return commitoutcome.Wrap(outcome, errors.New("injected"))
				})
			wantCode := 1
			if outcome == commitoutcome.Committed {
				wantCode = 0
			}
			if code != wantCode || gotID != "attempt-a" || stderr.Len() == 0 {
				t.Fatalf("code=%d want=%d id=%q stderr=%q", code, wantCode, gotID, stderr.String())
			}
		})
	}
}
