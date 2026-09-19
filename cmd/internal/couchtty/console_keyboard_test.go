package couchtty

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func keyboardFixture(t *testing.T) (*consoleFixture, *keyboardHost, *ptychild.Child) {
	t.Helper()
	h := newKeyboardHost(24, 100)
	pr, pw := io.Pipe()
	c := New(h, pr)
	a, b := ptychild.NewFakeChild(nil), ptychild.NewFakeChild(nil)
	a.SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return c.Deliver(ctx, "c1", batch) })
	b.SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return c.Deliver(ctx, "c2", batch) })
	c.Attach("c1", "first", a)
	c.Attach("c2", "second", b)
	setTestOps(c, func(string, map[string]string) (any, error) { return nil, nil })
	f := &consoleFixture{host: h.FakeHost, child: a, con: c, stdin: pw, done: make(chan int, 1)}
	go func() { f.done <- c.Run(); close(f.done) }()
	waitFor(t, "keyboard console startup", func() bool { return strings.Contains(h.Written(), "\x1b[?1003h") })
	c.switchTo("c1", true, arrivalOrdinary)
	t.Cleanup(func() {
		defer a.Close()
		defer b.Close()
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
			// The key comes from the screen the child's output left the parent
			// on, and that output is painted asynchronously. Encoding it at once
			// read the main screen, so the ?1049h case passed while the
			// alternate screen had no keyboard push at all (#279).
			awaitScreen(t, h, strings.HasSuffix(mode, "\x1b[?1049h"))
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

// The operator's report behind #279: after a thread had been shown, Alt+d in
// the switcher typed a d into the filter. A zellij actor puts the parent on its
// alternate screen and the switcher paints over it there. Alt+d has no legacy
// chord, so it reaches Couch only if that screen carries the keyboard push too.
func TestKeyboardPhysicalAltDLeavesFromTheSwitcher(t *testing.T) {
	type step struct {
		output    string
		alternate bool
	}
	for _, tc := range []struct {
		name  string
		steps []step
	}{
		{"main", nil},
		{"alternate", []step{{"\x1b[?1049hALT", true}}},
		{"alternate-and-back", []step{{"\x1b[?1049hALT", true}, {"\x1b[?1049lMAIN", false}}},
		{"back-again", []step{{"\x1b[?1049hALT", true}, {"\x1b[?1049lMAIN", false}, {"\x1b[?1049hALT", true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, h, _ := keyboardFixture(t)
			dispatched := make(chan map[string]string, 1)
			setTestOps(f.con, func(name string, args map[string]string) (any, error) {
				if name == "leave" {
					dispatched <- args
				}
				return nil, nil
			})
			for _, s := range tc.steps {
				f.child.Feed([]byte(s.output))
				awaitScreen(t, h, s.alternate)
			}
			_, _ = f.stdin.Write(h.ctrlSpace())
			waitFor(t, "the switcher", func() bool {
				f.con.mu.Lock()
				defer f.con.mu.Unlock()
				return f.con.focus.IsPanel()
			})
			key := h.altD()
			_, _ = f.stdin.Write(key)
			select {
			case args := <-dispatched:
				if args["mode"] != string(couchcore.LeaveDetach) {
					t.Fatalf("leave args = %+v, want mode=detach", args)
				}
			case <-time.After(time.Second):
				t.Fatalf("physical Alt+d encoded %q (flags %d) and dispatched nothing", key, h.flags())
			}
		})
	}
}

// awaitScreen waits for the parent to reach the screen a child's output asks
// for. The key an operator presses depends on that screen's keyboard stack.
func awaitScreen(t *testing.T, h *keyboardHost, alternate bool) {
	t.Helper()
	waitFor(t, fmt.Sprintf("parent alternate screen = %v", alternate), func() bool { return h.alternate() == alternate })
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
		f.con.showMenu()
		if h.flags()&1 == 0 {
			t.Fatal("panel takeover lost keyboard disambiguation")
		}
		f.con.switchTo("c1", true, arrivalOrdinary)
	}
}

func TestKeyboardChildControlsNeverChangeParentProtocol(t *testing.T) {
	for _, seq := range []string{"\x1b[=0u", "\x1b]title\x07", "\x1bPpayload\x1b\\", "\x1b_payload\x1b\\", "\x1b7\x1b[=6u", "\x1b]" + strings.Repeat("x", 70*1024) + "\x07"} {
		t.Run(seq[:min(8, len(seq))], func(t *testing.T) {
			f, h, _ := keyboardFixture(t)
			depth := h.depth()
			for _, part := range [][]byte{[]byte(seq[:1]), []byte(seq[1:])} {
				f.child.Feed(part)
				if err := f.con.presenter.Flush(context.Background()); err != nil {
					t.Fatal(err)
				}
				if h.flags() != 3 || h.depth() != depth {
					t.Fatalf("child altered parent flags=%d depth=%d", h.flags(), h.depth())
				}
			}
		})
	}
}

func TestKeyboardReleaseRejectsLatePresentation(t *testing.T) {
	f, h, _ := keyboardFixture(t)
	f.con.Stop()
	select {
	case <-f.done:
	case <-time.After(3 * time.Second):
		t.Fatal("Console did not complete teardown")
	}
	if got := h.ctrlReturn(); !bytes.Equal(got, []byte("\r")) {
		t.Fatalf("shell inherited keyboard state: %q", got)
	}
	before := h.Written()
	if err := f.con.presenter.Present(context.Background(), f.child.Endpoint()); err == nil {
		t.Fatal("released presenter accepted output")
	}
	f.con.showMenu()
	f.con.release()
	if h.Written() != before {
		t.Fatal("late output reached terminal")
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
