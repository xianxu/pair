package launcher

import (
	"bytes"
	"errors"
	"testing"
)

// PAIR_TEST_CALL is a shell-only helper dispatcher (bin/pair-shell short-circuits
// it before any zellij/fzf). LaunchNative must decline to the shell when it is
// set — otherwise a bare `pair` under it would enter the M5a native pick and
// block on fzf's /dev/tty (regression the pair-continue / cmux-ownership shell
// contract tests caught). Guard until the shell retires at M5c.
func TestLaunchNativeDeclinesShellTestCall(t *testing.T) {
	t.Setenv("PAIR_TEST_CALL", "park_scrollback")
	var stdout, stderr bytes.Buffer
	code, err := LaunchNative(nil, "/pair", &stdout, &stderr)
	if !errors.Is(err, ErrFallbackToShell) {
		t.Fatalf("PAIR_TEST_CALL must decline to the shell; got code=%d err=%v", code, err)
	}
}
