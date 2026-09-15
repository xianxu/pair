package couchtty

import (
	"bytes"
	"syscall"
	"testing"
)

func TestKeyboardShutdownPaths(t *testing.T) {
	for _, path := range []string{"stop", "eof", "signal"} {
		t.Run(path, func(t *testing.T) {
			f, h, _ := keyboardFixture(t)
			f.child.Feed([]byte("\x1b[?1049h\x1b[=7u"))
			switch path {
			case "stop":
				f.con.Stop()
			case "eof":
				_ = f.stdin.Close()
				// EOF ends the input pump, not Console ownership.
				f.con.Stop()
			case "signal":
				h.Terminate(syscall.SIGTERM)
			}
			waitFor(t, "shell keyboard restored", func() bool { return h.flags() == 0 && h.RawDepth() == 0 })
			if !bytes.Equal(h.ctrlReturn(), []byte("\r")) {
				t.Fatal("main screen retained keyboard mode")
			}
		})
	}
}
