package couchtty

import (
	"bytes"
	"context"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"io"
	"sync/atomic"
	"testing"
)

func TestSecondTabRowClickUsesItsOwnChip(t *testing.T) {
	c, input, host, calls := newMouseFixture(t)
	host.SetSize(ptychild.Size{Rows: 24, Cols: 7})
	waitFor(t, "second clickable tab row", func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		for _, ch := range c.statusChips {
			if ch.Row == 1 && ch.Thread == menuAddress("two") {
				return c.statusRows == 2
			}
		}
		return false
	})
	c.mu.Lock()
	first := int(c.size.Rows) - c.statusRows + 1
	var col int
	for _, ch := range c.statusChips {
		if ch.Row == 1 && ch.Thread == menuAddress("two") {
			col = ch.Start + 1
		}
	}
	c.mu.Unlock()
	clickAt(t, input, col, first+1)
	select {
	case got := <-calls:
		if got.call.Args["tag"] != string(menuAddress("two").Tag) {
			t.Fatalf("wrong row target %+v", got.call)
		}
	case <-testwait.After(t):
		t.Fatal("second-row chip ignored")
	}
}

func TestStaleChromeRevisionDropsGeometryButKeepsKeys(t *testing.T) {
	c, _, _, calls := newMouseFixture(t)
	waitFor(t, "confirmed chrome", c.chromeConfirmed)
	err := c.runTerminalCommand(context.Background(), func() error {
		c.mu.Lock()
		ch := c.statusChips[1]
		row := int(c.size.Rows)
		revision := c.statusRevision
		c.statusRevision++
		c.mu.Unlock()
		defer func() { c.mu.Lock(); c.statusRevision = revision; c.mu.Unlock() }()
		var decoder terminal.Decoder
		events, err := decoder.Feed([]byte(fmt.Sprintf("\x1b[<0;%d;%dMx", ch.Start+1, row)))
		if err != nil {
			return err
		}
		for _, event := range events {
			c.routeInputEvent(event)
		}
		c.mu.Lock()
		pending := len(c.menu.Operations)
		c.mu.Unlock()
		if pending != 0 {
			return fmt.Errorf("stale geometry admitted %d operations", pending)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	child := c.activeChild()
	waitFor(t, "key after dropped click", func() bool {
		for _, b := range child.Writes() {
			if string(b) == "x" {
				return true
			}
		}
		return false
	})
	select {
	case got := <-calls:
		t.Fatalf("stale geometry dispatched %+v", got.call)
	default:
	}
}

// The parent can keep accepting a physical prefix while rejecting the complete
// publication. This is a transport outcome, not a presenter-state injection.
type uncertainChromeHost struct {
	*hostty.FakeHost
	blocked atomic.Bool
}

func (h *uncertainChromeHost) WriteContext(ctx context.Context, b []byte) (int, error) {
	if h.blocked.Load() {
		n, _ := h.FakeHost.WriteContext(ctx, b[:min(3, len(b))])
		return n, context.DeadlineExceeded
	}
	return h.FakeHost.WriteContext(ctx, b)
}

func TestConsolePendingChromeWaitsForAutomaticPaintBeforeClick(t *testing.T) {
	host := &uncertainChromeHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})}
	reader, input := io.Pipe()
	c := New(host, reader)
	c.SetTabStyle(TabStyleMultiline)
	first, second := ptychild.NewFakeChild(nil), ptychild.NewFakeChild(nil)
	t.Cleanup(func() { first.Close(); second.Close() })
	c.attachThreadActor("c1", "one", menuAddress("one"), "/w/one", "one", first)
	c.attachThreadActor("c2", "two", menuAddress("two"), "/w/two", "two", second)
	calls := make(chan couchcore.OperationCall, 8)
	c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) { calls <- call; return nil, nil })
	c.mu.Lock()
	c.active = "c1"
	c.focus = FocusActor("c1")
	c.menu = NewMenuState([]couchcore.ActionableThreadSummary{
		{Address: menuAddress("one"), Name: "one", WorkingPath: "/w/one", State: couchcore.ThreadLive},
		{Address: menuAddress("two"), Name: "two", WorkingPath: "/w/two", State: couchcore.ThreadLive},
	}, menuAddress("one"))
	c.menuReady = true
	c.mu.Unlock()
	done := make(chan struct{})
	go func() { c.Run(); close(done) }()
	t.Cleanup(func() {
		host.blocked.Store(false)
		c.Stop()
		input.Close()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("console teardown timed out")
		}
	})
	waitFor(t, "initial confirmed layout", c.chromeConfirmed)
	host.blocked.Store(true)
	host.SetSize(ptychild.Size{Rows: 24, Cols: 7})
	var chip ChipSpan
	var revision uint64
	waitFor(t, "admitted second-row extents", func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.statusRows != 2 {
			return false
		}
		for _, ch := range c.statusChips {
			if ch.Row == 1 && ch.Thread == menuAddress("two") {
				chip = ch
				revision = c.statusRevision
				return true
			}
		}
		return false
	})
	if c.chromeConfirmed() {
		t.Fatal("partial physical write confirmed pending extents")
	}
	clickAt(t, input, chip.Start+1, 24)
	if _, err := input.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "key after suppressed pending click", func() bool {
		for _, b := range first.Writes() {
			if string(b) == "x" {
				return true
			}
		}
		return false
	})
	select {
	case got := <-calls:
		t.Fatalf("pending geometry dispatched %+v", got)
	default:
	}
	host.blocked.Store(false)
	waitFor(t, "same pending revision automatically confirmed", func() bool {
		if !c.chromeConfirmed() {
			return false
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.statusRevision == revision
	})
	clickAt(t, input, chip.Start+1, 24)
	select {
	case got := <-calls:
		if got.Args["tag"] != string(menuAddress("two").Tag) {
			t.Fatalf("recovered click wrong target %+v", got)
		}
	case <-testwait.After(t):
		t.Fatal("automatically confirmed extents remained unusable")
	}
}

func TestTwoRowAttentionBadgeActsOnlyOnFirstRow(t *testing.T) {
	c, input, host, _ := newMouseFixture(t)
	c.ObserveAttention(hubNotify(menuAddress("one"), "notice"))
	host.SetSize(ptychild.Size{Rows: 24, Cols: 11})
	waitFor(t, "two-row badge layout", func() bool {
		if !c.chromeConfirmed() {
			return false
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.statusRows == 2 && c.statusAttention.End > c.statusAttention.Start
	})
	c.mu.Lock()
	col := c.statusAttention.Start + 1
	c.mu.Unlock()
	clickAt(t, input, col, 24)
	if _, err := input.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	child := c.activeChild()
	waitFor(t, "key after blank second-row click", func() bool {
		for _, b := range child.Writes() {
			if string(b) == "x" {
				return true
			}
		}
		return false
	})
	c.attn.mu.Lock()
	open := c.attn.open
	c.attn.mu.Unlock()
	if open {
		t.Fatal("bottom row reused first-row badge span")
	}
	clickAt(t, input, col, 23)
	waitFor(t, "first-row badge opens attention", func() bool { c.attn.mu.Lock(); defer c.attn.mu.Unlock(); return c.attn.open })
	clickAt(t, input, col, 23)
	if _, err := input.Write([]byte("y")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "key after badge close", func() bool {
		for _, b := range child.Writes() {
			if string(b) == "y" {
				return true
			}
		}
		return false
	})
	c.attn.mu.Lock()
	open = c.attn.open
	c.attn.mu.Unlock()
	if open {
		t.Fatal("badge close click reopened attention")
	}
}

func TestZeroChromeTinyHostStillRoutesChildMouse(t *testing.T) {
	c, input, host, _ := newMouseFixture(t)
	child := c.activeChild()
	child.Feed([]byte("\x1b[?1003h\x1b[?1006h"))
	host.SetSize(ptychild.Size{Rows: 1, Cols: 80})
	waitFor(t, "zero-row chrome confirmed", func() bool {
		if !c.chromeConfirmed() {
			return false
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.statusRows == 0 && child.Size().Rows == 1
	})
	clickAt(t, input, 3, 1)
	if _, err := input.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "key after tiny-host mouse", func() bool {
		for _, b := range child.Writes() {
			if string(b) == "x" {
				return true
			}
		}
		return false
	})
	var all []byte
	for _, b := range child.Writes() {
		all = append(all, b...)
	}
	if !bytes.Contains(all, []byte("\x1b[<0;3;1M")) {
		t.Fatalf("child row swallowed as nonexistent chrome: %q", all)
	}
}
