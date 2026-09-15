package couchtty

import (
	"bytes"
	"syscall"
	"testing"
	"time"
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
			select {
			case <-f.done:
			case <-time.After(3 * time.Second):
				t.Fatal("Console did not complete teardown")
			}
			if h.flags() != 0 || h.RawDepth() != 0 || !h.Closed() {
				t.Fatal("completed teardown did not restore shell keyboard and host")
			}
			if !bytes.Equal(h.ctrlReturn(), []byte("\r")) {
				t.Fatal("main screen retained keyboard mode")
			}
		})
	}
}
