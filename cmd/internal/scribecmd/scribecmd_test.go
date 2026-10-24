package scribecmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
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
// succeed exactly as in production. Returns the exit code, captured stdout and
// Run's own stderr. A locked-down environment (sandbox / container with no
// /dev/ptmx) can't allocate a pty; skip visibly there rather than hard-fail —
// the test runs on a normal dev machine, which is where `make test` is
// exercised.
func runWithPTY(t *testing.T, args []string) (int, string, string) {
	return runWithPTYReply(t, args, "")
}

type outputFunc func([]byte) (int, error)

func (f outputFunc) Write(p []byte) (int, error) { return f(p) }

// Run's stderr is returned, never discarded: every one of its fatal sites
// writes through fatalf, so a non-empty diagnostic means Run aborted and the
// int it returned is Run's own failure code, not the child's exit status.
func runWithPTYReply(t *testing.T, args []string, ready string) (int, string, string) {
	t.Helper()
	ptmx, tty, err := openPTY()
	if err != nil {
		if os.IsPermission(err) {
			t.Skipf("pty allocation not permitted in this environment: %v", err)
		}
		t.Fatalf("allocate fixture pty: %v", err)
	}
	defer ptmx.Close()
	defer tty.Close()
	var stdout, diag bytes.Buffer
	var replyErr error
	replied := false
	output := outputFunc(func(p []byte) (int, error) {
		n, err := stdout.Write(p)
		if ready != "" && !replied && strings.Contains(stdout.String(), ready) {
			replied = true
			_, replyErr = io.WriteString(ptmx, "captured\n")
		}
		return n, err
	})
	code := Run(args, tty, output, &diag)
	if replyErr != nil {
		t.Fatalf("acknowledge captured output: %v", replyErr)
	}
	if ready != "" && !replied {
		t.Fatal("child never reached output acknowledgement")
	}
	return code, stdout.String(), diag.String()
}

func TestRunPropagatesChildExitCode(t *testing.T) {
	log := filepath.Join(t.TempDir(), "log")
	for _, want := range []int{0, 5, 42} {
		code, _, diag := runWithPTY(t, []string{"-log", log, "--", "sh", "-c", "exit " + strconv.Itoa(want)})
		// Assert the abort first and separately. Reporting it as "child exit %d
		// propagated as %d" is the misdiagnosis that left #809 unclassifiable:
		// the child never started, so there was no status to propagate.
		if diag != "" {
			t.Fatalf("scribe aborted instead of running the child: %s", diag)
		}
		if code != want {
			t.Fatalf("child exit %d propagated as %d", want, code)
		}
	}
}

// nopCloser stands in for a slave this test never opened.
type nopCloser struct{}

func (nopCloser) Close() error { return nil }

// run maps every fatal site to exit code 1 plus a diagnostic naming it, so a
// caller can tell "scribe aborted" from "the child exited 1". #809 was the
// pty-start site surfacing as a bare 1 with nothing to distinguish it. The
// remaining sites are pinned in pty_darwin_test.go: MakeRaw by
// TestDarwinRunAbortsChildOnRawModeFailure, the non-fatal slave-release
// diagnostic by TestDarwinRunCleanupDiagnosticPreservesExit.
func TestRunFatalSitesExitOneWithDiagnostic(t *testing.T) {
	log := filepath.Join(t.TempDir(), "log")

	t.Run("log open", func(t *testing.T) {
		absent := filepath.Join(t.TempDir(), "absent", "log")
		var diag bytes.Buffer
		code := Run([]string{"-log", absent, "--", "/usr/bin/true"}, strings.NewReader(""), io.Discard, &diag)
		if code != 1 || !strings.Contains(diag.String(), "opening "+absent) {
			t.Fatalf("code=%d diag=%q", code, diag.String())
		}
	})

	t.Run("pty start", func(t *testing.T) {
		var diag bytes.Buffer
		code := run([]string{"-log", log, "--", "/usr/bin/true"}, strings.NewReader(""), io.Discard, &diag,
			func(*exec.Cmd) (*startedPTY, error) { return nil, errors.New("allocation refused") })
		if code != 1 || !strings.Contains(diag.String(), "pty.Start: allocation refused") {
			t.Fatalf("code=%d diag=%q", code, diag.String())
		}
	})

	t.Run("cmd.Wait", func(t *testing.T) {
		// A master already at EOF and a child that was never started: draining
		// completes, then cmd.Wait reports "not started", which is not an
		// *exec.ExitError, so it takes the fatal branch.
		master, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		var diag bytes.Buffer
		code := run([]string{"-log", log, "--", "/usr/bin/true"}, strings.NewReader(""), io.Discard, &diag,
			func(*exec.Cmd) (*startedPTY, error) {
				return &startedPTY{master: master, slave: nopCloser{}}, nil
			})
		if code != 1 || !strings.Contains(diag.String(), "cmd.Wait:") {
			t.Fatalf("code=%d diag=%q", code, diag.String())
		}
	})
}

// The pty output is teed to both stdout (never paused) and the -log file.
func TestRunTeesChildOutputToStdoutAndLog(t *testing.T) {
	log := filepath.Join(t.TempDir(), "log")
	// Opening /dev/tty establishes Darwin's exit-time drain reference. Bare
	// printf can close the last slave and discard output before Run can read it.
	// The acknowledgement crosses both real PTYs before the child emits its tail.
	code, stdout, _ := runWithPTYReply(t, []string{"-log", log, "--", "/bin/sh", "-c",
		`exec 3<>/dev/tty || exit 91
printf scribe-ready
IFS= read -r reply || exit 92
[ "$reply" = captured ] || exit 93
printf scribe-marker`}, "scribe-ready")
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
	if string(body) != stdout {
		t.Fatalf("tee differs: stdout=%q log=%q", stdout, body)
	}
	if !strings.Contains(string(body), "scribe-marker") {
		t.Fatalf("log missing child output:\n%q", string(body))
	}
}

// Keep enough native launches to reject the observed 2/100 failure rate rather
// than accepting a single lucky run. One scheduler exposes the launch gap.
func TestRunTeesChildOutputToStdoutAndLogStress(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	for i := 0; i < 1000; i++ {
		t.Run(strconv.Itoa(i), TestRunTeesChildOutputToStdoutAndLog)
	}
}
