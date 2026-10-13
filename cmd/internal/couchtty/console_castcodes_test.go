package couchtty

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/castlock"
	"github.com/xianxu/pair/cmd/internal/terminal"
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
	f.hoverStatus(f.liveSpan().Start + 1) // LIVE ⏸
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
	waitFor(t, "the Codes menu", func() bool { return strings.Contains(f.screenText(), "view "+old+" · rotate · copy-link") })
	if !strings.Contains(f.screenText(), "pointer (not made yet)") {
		t.Fatal("no pointer row before pointing made the link")
	}
	if f.phase() != broadcastLive {
		t.Fatal("Alt+click stopped the broadcast")
	}
	f.chooseCodes(t, codesAction{link: broadcast.LinkView})
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

// The cast row's rule on the web cast (pair#495 smoke): hovering Chrome ⏸
// shows all its codes and its link controls show none; Alt+click opens the
// Codes menu on the cell only; the menu's blue entries copy the links; and a
// right-click on the cell copies the view-only link, with or without Alt.
func TestWebCastCodesOnTheChromeCell(t *testing.T) {
	c := newDriveCast(t)
	codes := c.cast().Codes()
	if len(codes) != 2 || codes[1].Link != broadcast.LinkDrive || len(codes[1].Code) != 6 {
		t.Fatalf("codes %+v", codes)
	}
	c.con.mu.Lock()
	viewIcon, driveIcon := c.con.statusWebViewLink, c.con.statusWebDriveLink
	c.con.mu.Unlock()
	mouse := func(button, x int) {
		_, _ = fmt.Fprintf(c.stdin, "\x1b[<%d;%d;24M\x1b[<%d;%d;24m", button, x+1, button, x+1)
	}
	hover := func(x int) { _, _ = fmt.Fprintf(c.stdin, "\x1b[<35;%d;24M", x+1) }
	all := "view " + codes[0].Code + " · drive " + codes[1].Code
	hover(c.cell() - 1)
	waitFor(t, "all codes on Chrome ⏸", func() bool { return strings.Contains(c.screenText(), all) })
	for _, icon := range []ColumnSpan{viewIcon, driveIcon} {
		hover(icon.Start)
		waitFor(t, "no codes on a link control", func() bool {
			s := c.screenText()
			return !strings.Contains(s, codes[0].Code) && !strings.Contains(s, codes[1].Code)
		})
	}
	before := c.wphase()
	mouse(8, viewIcon.Start) // Alt+click on 👆: copies, no menu
	time.Sleep(100 * time.Millisecond)
	if strings.Contains(c.screenText(), "Chrome codes") {
		t.Fatal("Alt+click on a link control opened the menu")
	}
	mouse(8, c.cell()-1)
	waitFor(t, "the Chrome Codes menu", func() bool { return strings.Contains(c.screenText(), "Chrome codes") })
	if c.wphase() != before {
		t.Fatal("Alt+click toggled the cast")
	}
	for _, want := range []string{"view " + codes[0].Code + " · rotate · copy-link", "drive " + codes[1].Code + " · rotate · copy-link", "rotate all"} {
		if !strings.Contains(c.screenText(), want) {
			t.Fatalf("menu lacks %q", want)
		}
	}
	if !strings.Contains(c.host.Written(), "38;5;75") {
		t.Fatal("the copy entries are not drawn in the link blue")
	}
	c.chooseCodes(t, codesAction{copy: true, link: broadcast.LinkDrive})
	waitFor(t, "drive link copied from the menu", func() bool { return copies(c.consoleFixture, c.link) == 1 })
	mouse(2|8, c.cell()-1) // Alt+right-click: the view link now, never the drive link
	waitFor(t, "view link copied", func() bool { return copies(c.consoleFixture, c.viewLink) >= 1 })
	if copies(c.consoleFixture, c.link) != 1 {
		t.Fatal("Alt+right-click still copies the drive link")
	}
}

// The screen broadcast's Codes menu copies its view-only link too.
func TestScreenCodesMenuCopiesTheViewLink(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	s := f.startLive(t)
	x := f.liveSpan().Start + 1
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<8;%d;24M\x1b[<8;%d;24m", x, x)
	waitFor(t, "the LIVE Codes menu", func() bool { return strings.Contains(f.screenText(), "· copy-link") })
	before := copies(f, s.Link())
	f.chooseCodes(t, codesAction{copy: true, link: broadcast.LinkView})
	waitFor(t, "view link copied", func() bool { return copies(f, s.Link()) == before+1 })
	if f.phase() != broadcastLive {
		t.Fatal("the menu stopped the broadcast")
	}
}

func (f *consoleFixture) liveSpan() ColumnSpan {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.statusControl
}

// chooseCodes clicks the open Codes menu's word that does want.
func (f *consoleFixture) chooseCodes(t *testing.T, want codesAction) {
	t.Helper()
	f.con.codes.mu.Lock()
	box := f.con.codes.menuBox
	f.con.codes.mu.Unlock()
	for _, r := range box.rows {
		for _, w := range r.words {
			if w.action == want {
				_, _ = fmt.Fprintf(f.stdin, "\x1b[<0;%d;%dM\x1b[<0;%d;%dm", w.x0+1, r.y+1, w.x0+1, r.y+1)
				return
			}
		}
	}
	t.Fatalf("no menu word %+v", want)
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

// The menu's words are its click targets, and the keyboard walks them:
// Up/Down the rows (skipping a row with no actions), Left/Right the words.
func TestCodesMenuLayoutAndSelection(t *testing.T) {
	rows := []codesMenuRow{
		{label: "view 3926", actions: []codesMenuWord{{text: "rotate", action: codesAction{link: "view"}}, {text: "copy-link", action: codesAction{copy: true, link: "view"}}}},
		{label: "pointer (not made yet)"},
		{label: "drive 903114", actions: []codesMenuWord{{text: "rotate", action: codesAction{link: "drive"}}, {text: "copy-link", action: codesAction{copy: true, link: "drive"}}}},
		{actions: []codesMenuWord{{text: "rotate all"}}},
	}
	m := layoutCodesMenu("Chrome codes", rows, 80, 24)
	if !m.Open() || m.X+m.W != 80 || m.Y+m.H != 23 {
		t.Fatalf("box %+v", m)
	}
	copyView := m.rows[0].words[1]
	if sel, ok := m.WordAt(copyView.x0, m.rows[0].y); !ok || sel != (codesMenuSel{0, 1}) {
		t.Fatalf("copy-link not under its own columns: %+v %v", sel, ok)
	}
	if _, ok := m.WordAt(m.X+2, m.rows[0].y); ok {
		t.Fatal("the label is a click target")
	}
	if _, ok := m.WordAt(m.X+2, m.rows[1].y); ok {
		t.Fatal("a row with no actions has a target")
	}
	m = m.Move(1, 0) // nothing selected: the first word
	if a, _ := m.Selected(); a != (codesAction{link: "view"}) {
		t.Fatalf("first selection %+v", a)
	}
	m = m.Move(0, 1)
	if a, _ := m.Selected(); a != (codesAction{copy: true, link: "view"}) {
		t.Fatalf("right %+v", a)
	}
	m = m.Move(1, 0) // skips the pointer row, keeps the column
	if a, _ := m.Selected(); a != (codesAction{copy: true, link: "drive"}) {
		t.Fatalf("down %+v", a)
	}
	m = m.Move(1, 0)
	if a, _ := m.Selected(); a != (codesAction{}) {
		t.Fatalf("rotate all %+v", a)
	}
	m = m.Move(1, 0).Move(0, 5)
	if a, _ := m.Selected(); a != (codesAction{}) {
		t.Fatalf("past the end moved: %+v", a)
	}
}

// From the keyboard: Down, Right and Enter on the view row copy its link;
// Escape closes the menu; other keys go on.
func TestCodesMenuKeyboard(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	s := f.startLive(t)
	x := f.liveSpan().Start + 1
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<8;%d;24M\x1b[<8;%d;24m", x, x)
	waitFor(t, "the menu", func() bool { return strings.Contains(f.screenText(), "LIVE codes") })
	before := copies(f, s.Link())
	_, _ = f.stdin.Write([]byte("\x1b[B"))
	_, _ = f.stdin.Write([]byte("\x1b[C"))
	_, _ = f.stdin.Write([]byte("\r"))
	waitFor(t, "view link copied by keyboard", func() bool { return copies(f, s.Link()) == before+1 })
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<8;%d;24M\x1b[<8;%d;24m", x, x)
	waitFor(t, "the menu again", func() bool { return strings.Contains(f.screenText(), "LIVE codes") })
	_, _ = f.stdin.Write([]byte("\x1b"))
	waitFor(t, "Escape closes it", func() bool { return !strings.Contains(f.screenText(), "LIVE codes") })
	if f.phase() != broadcastLive {
		t.Fatal("a menu key reached the broadcast toggle")
	}
}

// A wide character in a row (an emoji) takes its two cells: the menu still
// draws a valid frame, and the words after it keep their columns.
func TestCodesMenuDrawsWideCharacters(t *testing.T) {
	rows := []codesMenuRow{{label: "pointer 👆 8538", actions: []codesMenuWord{{text: "rotate", action: codesAction{link: "pointer"}}}}}
	m := layoutCodesMenu("LIVE codes", rows, 40, 10)
	f := terminal.Frame{Geometry: terminal.Geometry{Cols: 40, Rows: 10}, Cells: make([]terminal.Cell, 400)}
	for i := range f.Cells {
		f.Cells[i] = terminal.Cell{Content: " ", Width: 1}
	}
	out := m.Draw(f, codesMenuSel{row: -1})
	if err := out.Validate(); err != nil {
		t.Fatalf("invalid frame: %v", err)
	}
	w := m.rows[0].words[0]
	if got := out.Cells[m.rows[0].y*40+w.x0].Content; got != "r" {
		t.Fatalf("rotate drawn off its column: %q", got)
	}
}

// Running a word keeps the Codes popup open (Escape or × closes it): the
// pressed word flashes inverted for a moment, and a rotated code changes in
// place, its viewers disconnected.
func TestCodesMenuStaysOpenAndShowsThePress(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	s := f.startLive(t)
	x := f.liveSpan().Start + 1
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<8;%d;24M\x1b[<8;%d;24m", x, x)
	old := viewCode(t, s)
	waitFor(t, "the menu", func() bool { return strings.Contains(f.screenText(), "view "+old+" · rotate") })
	pressedBefore := strings.Count(f.host.Written(), "48;5;75")
	f.chooseCodes(t, codesAction{copy: true, link: broadcast.LinkView})
	waitFor(t, "the pressed word drawn inverted", func() bool { return strings.Count(f.host.Written(), "48;5;75") > pressedBefore })
	waitFor(t, "the link copied", func() bool { return copies(f, s.Link()) >= 1 })
	if !strings.Contains(f.screenText(), "LIVE codes") {
		t.Fatal("copy-link closed the popup")
	}
	f.con.codes.mu.Lock()
	until := f.con.codes.pressedUntil
	f.con.codes.mu.Unlock()
	if d := time.Until(until); d > codesPress {
		t.Fatalf("press lasts %v", d)
	}
	f.chooseCodes(t, codesAction{link: broadcast.LinkView})
	waitFor(t, "the new code in place", func() bool {
		code := viewCode(t, s)
		return code != old && strings.Contains(f.screenText(), "view "+code+" · rotate · copy-link")
	})
	if !strings.Contains(f.screenText(), "LIVE codes") {
		t.Fatal("rotate closed the popup")
	}
	f.con.codes.mu.Lock()
	box := f.con.codes.menuBox
	f.con.codes.mu.Unlock()
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<0;%d;%dM\x1b[<0;%d;%dm", box.X+box.W-1, box.Y+1, box.X+box.W-1, box.Y+1)
	waitFor(t, "× closes it", func() bool { return !strings.Contains(f.screenText(), "LIVE codes") })
}
