package wrapcmd

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/creack/pty"
)

// Argument errors are reported before any PTY/terminal op, so they need no tty.
func TestRunArgErrors(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{"no command", nil, "usage: pair-wrap"},
		{"unknown flag", []string{"-bogus"}, "unknown flag"},
		{"scrollback flag but no command", []string{"--scrollback-log", "/tmp/x"}, "usage: pair-wrap"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(c.args, strings.NewReader(""), &stdout, &stderr)
			if code != 1 {
				t.Fatalf("code = %d, want 1", code)
			}
			if !strings.Contains(stderr.String(), c.wantMsg) {
				t.Fatalf("stderr = %q, want substring %q", stderr.String(), c.wantMsg)
			}
			// The pair-wrap: prefix is applied by Run's error path.
			if !strings.HasPrefix(stderr.String(), "pair-wrap: ") {
				t.Fatalf("stderr = %q, want pair-wrap: prefix", stderr.String())
			}
		})
	}
}

// A wrapped child's exit code must propagate through Run — the pre-extraction
// binary did this via os.Exit(exitErr.ExitCode()); Run now returns it, so both
// the shim and the `pair wrap` route agree.
func TestRunPropagatesChildExitCode(t *testing.T) {
	// Isolate from any leaked live-session env (PAIR_TAG/PAIR_DATA_DIR) so the
	// proxy's optional pidfile/scrollback side effects stay disabled.
	t.Setenv("PAIR_TAG", "")
	t.Setenv("PAIR_DATA_DIR", "")
	for _, want := range []int{0, 7, 42} {
		ptmx, tty, err := pty.Open()
		if err != nil {
			if os.IsPermission(err) {
				t.Skipf("pty allocation not permitted in this environment: %v", err)
			}
			t.Fatalf("pty.Open: %v", err)
		}
		var stdout bytes.Buffer
		code := Run([]string{"sh", "-c", "exit " + strconv.Itoa(want)}, tty, &stdout, io.Discard)
		ptmx.Close()
		tty.Close()
		if code != want {
			t.Fatalf("child exit %d propagated as %d", want, code)
		}
	}
}
