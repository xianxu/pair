package couchtty

import (
	"bytes"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/hostty"
)

func FuzzKeyboardDisambiguated(f *testing.F) {
	f.Add([]byte("\x1b[=0u"), true)
	f.Add([]byte("\x1b]unterminated"), false)
	f.Fuzz(func(t *testing.T, p []byte, complete bool) {
		before := bytes.Clone(p)
		got := keyboardDisambiguated(p, complete)
		want := bytes.Clone(p)
		if complete {
			want = append(want, hostty.EnableKeyboardDisambiguation...)
		}
		if !bytes.Equal(got, want) || !bytes.Equal(p, before) {
			t.Fatal("payload, boundary or source-buffer contract violated")
		}
	})
}

// Hold a real Host write between the scanner decision and actual emission.
// The barrier and TryLock prove ownership while the write is in flight; the
// following takeover/release then exercises the resulting observable wire order.
type blockedKeyboardHost struct {
	*keyboardHost
	entered chan struct{}
	proceed chan struct{}
	once    sync.Once
}

func (h *blockedKeyboardHost) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("BLOCK")) {
		h.once.Do(func() { close(h.entered); <-h.proceed })
	}
	return h.keyboardHost.Write(p)
}

func TestKeyboardOutputTransactions(t *testing.T) {
	for _, release := range []bool{false, true} {
		t.Run(map[bool]string{false: "takeover", true: "release"}[release], func(t *testing.T) {
			h := &blockedKeyboardHost{keyboardHost: newKeyboardHost(24, 80), entered: make(chan struct{}), proceed: make(chan struct{})}
			c := New(h, strings.NewReader(""))
			first := make(chan struct{})
			go func() { c.writeChild([]byte("BLOCK\x1b[=0u")); close(first) }()
			<-h.entered
			held := !c.terminalMu.TryLock()
			if !held {
				c.terminalMu.Unlock()
			}
			next := make(chan struct{})
			go func() {
				if release {
					c.release()
				} else {
					c.takeOverScreen(nil, []byte("NEXT\x1b[=0u"))
				}
				close(next)
			}()
			close(h.proceed)
			for _, done := range []chan struct{}{first, next} {
				select {
				case <-done:
				case <-time.After(3 * time.Second):
					t.Fatal("terminal output deadlocked")
				}
			}
			if !held {
				t.Fatal("scanner/write transaction was not held across Host.Write")
			}
			if release {
				if h.flags() != 0 {
					t.Fatal("release did not win final wire state")
				}
				before := h.Written()
				c.writeHostControl("late")
				c.takeOverScreen(nil, nil)
				if before != h.Written() {
					t.Fatal("late output followed cleanup")
				}
			} else {
				wire := h.Written()
				if strings.Index(wire, "BLOCK") > strings.Index(wire, "NEXT") || h.flags()&1 == 0 {
					t.Fatal("takeover wire order or keyboard state lost")
				}
			}
		})
	}
}

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
