package scribecmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/creack/pty"
)

// Usage errors are reported before any PTY/terminal op, so they need no tty.
func TestRunUsageErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"no -log, no cmd", nil},
		{"-log but no cmd", []string{"-log", "/tmp/x"}},
		{"cmd but no -log", []string{"--", "echo", "hi"}},
		{"unknown flag", []string{"-nope"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(c.args, strings.NewReader(""), &stdout, &stderr)
			if code != 2 {
				t.Fatalf("code = %d, want 2 (usage error)", code)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if stderr.Len() == 0 {
				t.Fatalf("stderr empty, want a usage/flag message")
			}
		})
	}
}

// runWithPTY drives Run with a real pty slave as stdin so raw-mode + winsize
// succeed exactly as in production. Returns the exit code and captured stdout.
// A locked-down environment (sandbox / container with no /dev/ptmx) can't
// allocate a pty; skip visibly there rather than hard-fail — the test runs on a
// normal dev machine, which is where `make test` is exercised.
func runWithPTY(t *testing.T, args []string) (int, string) {
	t.Helper()
	ptmx, tty, err := pty.Open()
	if err != nil {
		if os.IsPermission(err) {
			t.Skipf("pty allocation not permitted in this environment: %v", err)
		}
		t.Fatalf("pty.Open: %v", err)
	}
	defer ptmx.Close()
	defer tty.Close()
	var stdout bytes.Buffer
	code := Run(args, tty, &stdout, io.Discard)
	return code, stdout.String()
}

func TestRunPropagatesChildExitCode(t *testing.T) {
	log := filepath.Join(t.TempDir(), "log")
	for _, want := range []int{0, 5, 42} {
		code, _ := runWithPTY(t, []string{"-log", log, "--", "sh", "-c", "exit " + strconv.Itoa(want)})
		if code != want {
			t.Fatalf("child exit %d propagated as %d", want, code)
		}
	}
}

// The pty output is teed to both stdout (never paused) and the -log file.
func TestRunTeesChildOutputToStdoutAndLog(t *testing.T) {
	log := filepath.Join(t.TempDir(), "log")
	code, stdout := runWithPTY(t, []string{"-log", log, "--", "printf", "scribe-marker"})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "scribe-marker") {
		t.Fatalf("stdout missing child output:\n%q", stdout)
	}
	body, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(body), "scribe-marker") {
		t.Fatalf("log missing child output:\n%q", string(body))
	}
}
