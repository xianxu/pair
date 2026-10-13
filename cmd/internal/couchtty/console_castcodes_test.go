package couchtty

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/castlock"
)

func viewCode(t *testing.T, s *broadcast.Session) string {
	t.Helper()
	for _, c := range s.Codes() {
		if c.Link == broadcast.LinkView {
			return c.Code
		}
	}
	t.Fatal("no view code")
	return ""
}

// hoverStatus moves the mouse, no button held, over column x of the status
// row (1-based, as the terminal reports it).
func (f *consoleFixture) hoverStatus(x int) {
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<35;%d;24M", x)
}

// pair#495: hovering LIVE shows the broadcast's codes on the operator's
// screen, and a LIVE viewer -- who sees the tab bar -- never sees them.
func TestCastCodesShowOnlyOnTheOperatorsScreen(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	s := f.startLive(t)
	v := openViewer(t, s)
	code := viewCode(t, s)
	f.hoverStatus(2)
	waitFor(t, "the codes tooltip", func() bool { return strings.Contains(f.screenText(), "view "+code) })
	// A frame after the tooltip is up reaches the viewer, without it.
	f.child.Feed([]byte("AFTER-HOVER"))
	v.until("the frame after the hover", func(s string) bool { return strings.Contains(s, "AFTER-HOVER") })
	if strings.Contains(v.emu.String(), code) {
		t.Fatal("a LIVE viewer saw the view code")
	}
	f.hoverStatus(60)
	waitFor(t, "the tooltip gone", func() bool { return !strings.Contains(f.screenText(), "view "+code) })
}

// Alt+click on LIVE opens its Codes menu; choosing the view link rotates its
// code, disconnects its viewers, and the notice names no code.
func TestCodesMenuRotatesAndDisconnects(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	s := f.startLive(t)
	v := openViewer(t, s)
	old := viewCode(t, s)
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<8;2;24M\x1b[<8;2;24m") // Alt+click on LIVE
	waitFor(t, "the Codes menu", func() bool { return strings.Contains(f.screenText(), "view "+old+" · rotate") })
	if f.phase() != broadcastLive {
		t.Fatal("Alt+click stopped the broadcast")
	}
	f.con.codes.mu.Lock()
	box := f.con.codes.menuBox
	f.con.codes.mu.Unlock()
	var top int
	for _, it := range box.items {
		if it.ID == 1 {
			top = it.Top
		}
	}
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<0;%d;%dM\x1b[<0;%d;%dm", box.X+3, top+1, box.X+3, top+1)
	waitFor(t, "a new view code", func() bool { return viewCode(t, s) != old })
	waitFor(t, "the notice", func() bool { return strings.Contains(f.screenText(), "New LIVE view code") })
	if strings.Contains(f.lastRow(), viewCode(t, s)) {
		t.Fatal("the notice line (broadcast) shows the new code")
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, err := v.r.ReadString('\n'); err != nil {
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the viewer's stream outlived the rotation")
	}
}

// A guessing rotation is told without its code, coalesced per link, and its
// new code flashes in the operator-only layer; a late, older rotation never
// replaces a newer one.
func TestGuessRotationAlertAndFlash(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	f.startLive(t)
	attempt := f.screenAttempt()
	f.con.onCodeRotated(castScreen, attempt, broadcast.LinkView, castlock.Rotation{Code: "5193", Reason: castlock.ReasonGuessing, Gen: 2})
	waitFor(t, "the alert", func() bool { return strings.Contains(f.screenText(), "Too many wrong codes on the LIVE view link") })
	waitFor(t, "the flash", func() bool { return strings.Contains(f.screenText(), "LIVE view 5193") })
	if strings.Contains(f.lastRow(), "5193") {
		t.Fatal("the alert names the code")
	}
	f.con.onCodeRotated(castScreen, attempt, broadcast.LinkView, castlock.Rotation{Code: "1111", Reason: castlock.ReasonGuessing, Gen: 1})
	f.con.codes.mu.Lock()
	if fl := f.con.codes.flash; fl == nil || fl.code != "5193" {
		t.Fatalf("an older rotation replaced the newer: %+v", fl)
	}
	f.con.codes.mu.Unlock()
	f.con.onCodeRotated(castScreen, attempt, broadcast.LinkView, castlock.Rotation{Code: "2222", Reason: castlock.ReasonGuessing, Gen: 3})
	f.con.codes.mu.Lock()
	quiet := f.con.codes.quiet[linkKey(castScreen, attempt, broadcast.LinkView)]
	f.con.codes.mu.Unlock()
	if quiet != 1 {
		t.Fatalf("a second alert within the quiet window was not coalesced: %d", quiet)
	}
	waitFor(t, "the newest flash", func() bool { return strings.Contains(f.screenText(), "LIVE view 2222") })
	waitFor(t, "the flash over", func() bool { return !strings.Contains(f.screenText(), "LIVE view 2222") })
}

// The Chrome cell's tooltip shows the web cast's view code and, when it can be
// driven, its six-digit drive code; Alt+click opens its Codes menu without
// toggling the cast.
func TestWebCastCodesOnTheChromeCell(t *testing.T) {
	c := newDriveCast(t)
	codes := c.cast().Codes()
	if len(codes) != 2 || codes[1].Link != broadcast.LinkDrive || len(codes[1].Code) != 6 {
		t.Fatalf("codes %+v", codes)
	}
	x := c.cell()
	_, _ = fmt.Fprintf(c.stdin, "\x1b[<35;%d;24M", x+1)
	want := "view " + codes[0].Code + " · drive " + codes[1].Code
	waitFor(t, "the Chrome tooltip", func() bool { return strings.Contains(c.screenText(), want) })
	before := c.wphase()
	_, _ = fmt.Fprintf(c.stdin, "\x1b[<8;%d;24M\x1b[<8;%d;24m", x+1, x+1)
	waitFor(t, "the Chrome Codes menu", func() bool { return strings.Contains(c.screenText(), "Chrome codes") })
	if c.wphase() != before {
		t.Fatal("Alt+click toggled the cast")
	}
	if !strings.Contains(c.screenText(), "rotate all") {
		t.Fatal("no rotate-all item with two links")
	}
}

func (f *consoleFixture) screenAttempt() uint64 {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.bcast.attempt
}

// BR-16: a rotation reported late by a cast that has ended -- even with a high
// Gen -- is dropped, and never holds back the next cast's rotations.
func TestLateRotationFromAnEndedCastIsDropped(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	f.startLive(t)
	current := f.screenAttempt()
	f.con.onCodeRotated(castScreen, current-1, broadcast.LinkView, castlock.Rotation{Code: "9999", Reason: castlock.ReasonOperator, Gen: 50})
	f.con.codes.mu.Lock()
	stale := f.con.codes.flash
	f.con.codes.mu.Unlock()
	if stale != nil {
		t.Fatalf("an ended cast's rotation flashed: %+v", stale)
	}
	f.con.onCodeRotated(castScreen, current, broadcast.LinkView, castlock.Rotation{Code: "4242", Reason: castlock.ReasonOperator, Gen: 1})
	waitFor(t, "the current cast's flash", func() bool { return strings.Contains(f.screenText(), "LIVE view 4242") })
}
