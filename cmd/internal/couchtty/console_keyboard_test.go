package couchtty

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func keyboardFixture(t *testing.T) (*consoleFixture, *keyboardHost, *ptychild.Child) {
	t.Helper()
	h := newKeyboardHost(24, 100)
	pr, pw := io.Pipe()
	c := New(h, pr)
	a, b := ptychild.NewFakeChild(nil), ptychild.NewFakeChild(nil)
	a.SetSink(func(batch ptychild.OutputBatch) { c.Deliver("c1", batch) })
	b.SetSink(func(batch ptychild.OutputBatch) { c.Deliver("c2", batch) })
	c.Attach("c1", "first", a)
	c.Attach("c2", "second", b)
	setTestOps(c, func(string, map[string]string) (any, error) { return nil, nil })
	f := &consoleFixture{host: h.FakeHost, child: a, con: c, stdin: pw, done: make(chan int, 1)}
	go func() { f.done <- c.Run() }()
	waitFor(t, "keyboard console startup", func() bool { return strings.Contains(h.Written(), hostty.EnableMouseClicks) })
	c.switchTo("c1", true, arrivalOrdinary)
	t.Cleanup(func() {
		c.Stop()
		_ = pw.Close()
		select {
		case <-f.done:
		case <-time.After(3 * time.Second):
			t.Error("keyboard console did not stop")
		}
	})
	return f, h, b
}

// Generate the operator's key from terminal state, not from the chord table.
func TestKeyboardPhysicalNotificationJump(t *testing.T) {
	for _, mode := range []string{"", "\x1b[=0u", "\x1b[>0u", "\x1b[<99u", "\x1bc", "\x1b[?1049h", "\x1b[=6u", "\x1b[?1049h\x1b[=0u\x1b[?1049l"} {
		t.Run(strings.ReplaceAll(mode, "\x1b", "ESC"), func(t *testing.T) {
			f, h, b := keyboardFixture(t)
			f.child.Feed([]byte(mode))
			f.con.mu.Lock()
			target := f.con.panes["c2"].thread
			f.con.attention.Mark(target, "ready")
			f.con.syncAttentionLocked()
			f.con.mu.Unlock()
			key := h.ctrlReturn()
			if _, err := f.stdin.Write(key); err != nil {
				t.Fatal(err)
			}
			waitFor(t, "jump or source input", func() bool { return activeOf(f) == "c2" || len(f.child.Writes()) > 0 })
			if got := activeOf(f); got != "c2" {
				t.Fatalf("physical Ctrl+Return encoded %q and reached source %q; active=%s", key, f.child.Writes(), got)
			}
			f.con.mu.Lock()
			pending := len(f.con.attention.Projection(target))
			f.con.mu.Unlock()
			if pending != 0 {
				t.Fatal("notification not acknowledged")
			}
			_, _ = f.stdin.Write([]byte("\r"))
			waitFor(t, "ordinary Return forwarded", func() bool { return len(b.Writes()) > 0 })
			if got := bytes.Join(b.Writes(), nil); !bytes.Equal(got, []byte("\r")) {
				t.Fatalf("ordinary Return = %q", got)
			}
		})
	}
}

func TestKeyboardReplayAndBackground(t *testing.T) {
	f, h, b := keyboardFixture(t)
	// Age the enable out of the actual Child ring, then retain a pop.
	b.Feed([]byte("\x1b[>1u" + strings.Repeat("x", ptychild.DefaultRingBytes+1) + "\x1b[<u"))
	if h.flags()&1 == 0 {
		t.Fatal("background output changed host flags")
	}
	for range 20 {
		f.con.switchTo("c2", true, arrivalOrdinary)
		if h.flags()&1 == 0 {
			t.Fatal("replay lost keyboard disambiguation")
		}
		f.con.takeOverScreen(nil, nil)
		if h.flags()&1 == 0 {
			t.Fatal("panel takeover lost keyboard disambiguation")
		}
		f.con.switchTo("c1", true, arrivalOrdinary)
	}
}

func TestKeyboardCompleteFramingAndCursorSave(t *testing.T) {
	// The mode control must never become part of a split or oversized string.
	for _, seq := range []string{"\x1b[=0u", "\x1b]title\x07", "\x1bPpayload\x1b\\", "\x1b_payload\x1b\\", "\x1b]" + strings.Repeat("x", 70*1024) + "\x07"} {
		cuts := []int{1, len(seq) - 1}
		if len(seq) < 40 {
			for n := 2; n < len(seq)-1; n++ {
				cuts = append(cuts, n)
			}
		}
		for _, cut := range cuts {
			h := newKeyboardHost(24, 80)
			c := New(h, strings.NewReader(""))
			c.writeChild([]byte(seq[:cut]))
			if got := h.Written(); got != seq[:cut] {
				t.Fatalf("injected inside control at %d: %q", cut, got)
			}
			c.writeChild([]byte(seq[cut:]))
			if h.flags()&1 == 0 {
				t.Fatalf("completion did not restore mode at cut %d", cut)
			}
		}
	}
	h := newKeyboardHost(24, 80)
	c := New(h, strings.NewReader(""))
	c.writeChild([]byte("\x1b7\x1b[=6u"))
	if got := h.flags(); got != 7 {
		t.Fatalf("cursor save or other flags blocked mode: %d", got)
	}
	depth := h.depth()
	for range 100 {
		c.writeChild([]byte("x"))
	}
	if h.depth() != depth {
		t.Fatal("Couch assertions grew keyboard stack")
	}
}

func TestKeyboardReleaseBothBuffersAndRejectLateOutput(t *testing.T) {
	h := newKeyboardHost(24, 80)
	c := New(h, strings.NewReader(""))
	c.writeChild([]byte("\x1b[=1u\x1b[?1049h\x1b[=1u"))
	c.release()
	if got := h.ctrlReturn(); !bytes.Equal(got, []byte("\r")) {
		t.Fatalf("restored main screen leaked key mode: %q", got)
	}
	before := h.Written()
	c.writeChild([]byte("late child"))
	c.takeOverScreen(nil, []byte("late takeover"))
	c.writeOwn("late paint")
	c.showMenu()
	c.release()
	if h.Written() != before {
		t.Fatal("output reached terminal after release")
	}
}

func TestKeyboardExplicitEventEncodings(t *testing.T) {
	for _, event := range []string{"1", "2", "3"} {
		key := []byte("\x1b[13;5:" + event + "u")
		for cut := 0; cut <= len(key); cut++ {
			var it Interceptor
			var forwarded []byte
			hits := 0
			for _, part := range [][]byte{key[:cut], key[cut:]} {
				before, hit, rest := it.FeedHit(part)
				forwarded = append(forwarded, before...)
				forwarded = append(forwarded, rest...)
				if hit == HitNewestPage {
					hits++
				} else if hit != HitNone {
					t.Fatalf("unexpected hit %v", hit)
				}
			}
			if event == "3" {
				if hits != 0 || !bytes.Equal(forwarded, key) {
					t.Fatalf("release consumed at split %d", cut)
				}
			} else if hits != 1 || len(forwarded) != 0 {
				t.Fatalf("event %s split %d: hits=%d forwarded=%q", event, cut, hits, forwarded)
			}
		}
	}
}
