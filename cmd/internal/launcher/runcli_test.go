package launcher

import (
	"bytes"
	"errors"
	"testing"
)

// The shell-only test/debug seams (bin/pair-shell short-circuits them before any
// zellij/fzf) must each make LaunchNative decline to the shell — otherwise a bare
// `pair`/`continue` under them enters the native path and blocks on fzf's / the
// create prompt's /dev/tty (the M5a hang class; PAIR_DEBUG_ARGS is used by the
// pair-continue probe tests). Guard until the shell retires at M5c.
func TestLaunchNativeDeclinesShellOnlySeams(t *testing.T) {
	for _, seam := range []string{"PAIR_TEST_CALL", "PAIR_DEBUG_ARGS", "PAIR_DEBUG_HISTORY"} {
		t.Run(seam, func(t *testing.T) {
			t.Setenv(seam, "1")
			var stdout, stderr bytes.Buffer
			code, err := LaunchNative(nil, "/pair", &stdout, &stderr)
			if !errors.Is(err, ErrFallbackToShell) {
				t.Fatalf("%s must decline to the shell; got code=%d err=%v", seam, code, err)
			}
		})
	}
}
