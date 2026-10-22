package couchtty

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/castlock"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/webcast"
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

// Every guessing rotation is told without its code, and its
// new code flashes in the operator-only layer; a late, older rotation never
// replaces a newer one.
func TestGuessRotationAlertAndFlash(t *testing.T) {
	copied := noticeClipboard(t)
	const full = "Too many wrong codes on the LIVE view link: its code changed. Hover LIVE to see it."
	const shown = "Too many wrong codes on the LIVE view l…"
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	f.startLive(t)
	attempt := f.screenAttempt()
	f.con.onCodeRotated(castScreen, attempt, broadcast.LinkView, castlock.Rotation{Code: "5193", Reason: castlock.ReasonGuessing, Gen: 2})
	waitFor(t, "the bounded alert", func() bool { return strings.Contains(f.lastRow(), shown) })
	if got := f.notice(); got != full {
		t.Fatalf("rotation notice lost source text: %q", got)
	}
	if strings.Contains(f.lastRow(), full) {
		t.Fatal("chrome displayed the unbounded rotation notice")
	}
	f.con.copyLastNotice()
	if got := copied(); got != full {
		t.Fatalf("copied rotation notice %q; want %q", got, full)
	}
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
	f.con.setNotice("waiting for the second rotation")
	f.con.onCodeRotated(castScreen, attempt, broadcast.LinkView, castlock.Rotation{Code: "2222", Reason: castlock.ReasonGuessing, Gen: 3})
	waitFor(t, "the second immediate alert", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.feed.Row().Body == full
	})
	waitFor(t, "the newest flash", func() bool { return strings.Contains(f.screenText(), "LIVE view 2222") })
	waitFor(t, "the flash over", func() bool { return !strings.Contains(f.screenText(), "LIVE view 2222") })
}

func TestEveryGuessRotationNotifiesIncludingCap(t *testing.T) {
	for _, kind := range []castKind{castScreen, castWeb} {
		t.Run(kind.name(), func(t *testing.T) {
			var con *Console
			var attempt uint64
			links := []string{broadcast.LinkView, broadcast.LinkPointer}
			if kind == castScreen {
				f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
				f.startLive(t)
				con, attempt = f.con, f.screenAttempt()
			} else {
				r := newWebcastRig(t, nil)
				r.open(t)
				con = r.con
				con.mu.Lock()
				attempt = con.wcast.attempt
				con.mu.Unlock()
				links = append(links, broadcast.LinkDrive)
			}
			for _, link := range links {
				t.Run(link, func(t *testing.T) {
					for i, code := range []string{"519351", "222222", "838383"} {
						con.setNotice("waiting for the next rotation")
						capped := i == 2
						con.onCodeRotated(kind, attempt, link, castlock.Rotation{Code: code, Reason: castlock.ReasonGuessing, Gen: uint64(i + 2), Capped: capped})
						want := fmt.Sprintf("Too many wrong codes on the %s %s link: its code changed. Hover %s to see it.", kind.name(), link, kind.name())
						if capped {
							want = fmt.Sprintf("Too many wrong codes on the %s %s link: its code changed; the link is locked until you rotate its code.", kind.name(), link)
						}
						waitFor(t, fmt.Sprintf("rotation %d notice", i+1), func() bool {
							con.mu.Lock()
							defer con.mu.Unlock()
							return con.feed.Row().Body == want
						})
						con.mu.Lock()
						notice := con.feed.Row().Body
						con.mu.Unlock()
						for _, secret := range []string{"519351", "222222", "838383"} {
							if strings.Contains(notice, secret) {
								t.Fatalf("rotation notice exposed a code: %q", notice)
							}
						}
					}
				})
			}
		})
	}
}

func TestRapidGuessRotationsLeaveCapNotice(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	f.startLive(t)
	attempt := f.screenAttempt()
	for i := uint64(1); i <= 3; i++ {
		f.con.onCodeRotated(castScreen, attempt, broadcast.LinkView, castlock.Rotation{Code: "519351", Reason: castlock.ReasonGuessing, Gen: i + 1, Capped: i == 3})
	}
	f.con.mu.Lock()
	notice := f.con.feed.Row().Body
	f.con.mu.Unlock()
	want := "Too many wrong codes on the LIVE view link: its code changed; the link is locked until you rotate its code."
	if notice != want {
		t.Fatalf("rapid rotations left notice %q, want %q", notice, want)
	}
	var callbacks sync.WaitGroup
	start := make(chan struct{})
	for gen := uint64(5); gen <= 34; gen++ {
		callbacks.Go(func() {
			<-start
			f.con.onCodeRotated(castScreen, attempt, broadcast.LinkView, castlock.Rotation{Code: "838383", Reason: castlock.ReasonGuessing, Gen: gen, Capped: gen == 34})
		})
	}
	close(start)
	callbacks.Wait()
	f.con.mu.Lock()
	notice = f.con.feed.Row().Body
	f.con.mu.Unlock()
	if notice != want {
		t.Fatalf("concurrent callbacks overwrote the newest cap notice: %q", notice)
	}
}

// The cast row's rule on the web cast (pair#495 smoke): hovering Chrome ⏸
// shows all its codes and its link controls show none; Alt+click opens the
// Codes menu on the cell only; the menu's blue entries copy the links; and a
// right-click on the cell copies the view-only link, with or without Alt. A
// driven cast is recorded, so it has all three links, each with a code
// (pair#506).
func TestWebCastCodesOnTheChromeCell(t *testing.T) {
	c := newDriveCast(t)
	codes := c.cast().Codes()
	if len(codes) != 3 || codes[0].Link != broadcast.LinkView || codes[1].Link != broadcast.LinkPointer || codes[2].Link != broadcast.LinkDrive || len(codes[2].Code) != 6 {
		t.Fatalf("codes %+v", codes)
	}
	c.con.mu.Lock()
	pointerIcon, driveIcon, pointerLink := c.con.statusWebPointerLink, c.con.statusWebDriveLink, c.con.wcast.update.PointerLink
	c.con.mu.Unlock()
	mouse := func(button, x int) {
		_, _ = fmt.Fprintf(c.stdin, "\x1b[<%d;%d;24M\x1b[<%d;%d;24m", button, x+1, button, x+1)
	}
	hover := func(x int) { _, _ = fmt.Fprintf(c.stdin, "\x1b[<35;%d;24M", x+1) }
	all := "view " + codes[0].Code + " · pointer " + codes[1].Code + " · drive " + codes[2].Code
	hover(c.cell() - 1)
	waitFor(t, "all codes on Chrome ⏸", func() bool { return strings.Contains(c.screenText(), all) })
	for _, icon := range []ColumnSpan{pointerIcon, driveIcon} {
		hover(icon.Start)
		waitFor(t, "no codes on a link control", func() bool {
			s := c.screenText()
			return !strings.Contains(s, codes[0].Code) && !strings.Contains(s, codes[1].Code) && !strings.Contains(s, codes[2].Code)
		})
	}
	before := c.wphase()
	mouse(8, pointerIcon.Start) // Alt+click on 👆: copies, no menu
	time.Sleep(100 * time.Millisecond)
	if strings.Contains(c.screenText(), "Chrome codes") {
		t.Fatal("Alt+click on a link control opened the menu")
	}
	mouse(8, c.cell()-1)
	waitFor(t, "the Chrome Codes menu", func() bool { return strings.Contains(c.screenText(), "Chrome codes") })
	if c.wphase() != before {
		t.Fatal("Alt+click toggled the cast")
	}
	for _, want := range []string{"view " + codes[0].Code + " · rotate · copy-link", "pointer " + codes[1].Code + " · rotate · copy-link", "drive " + codes[2].Code + " · rotate · copy-link", "rotate all", "Annotation · Off · Pen · Note"} {
		if !strings.Contains(c.screenText(), want) {
			t.Fatalf("menu lacks %q", want)
		}
	}
	if !strings.Contains(c.host.Written(), "38;5;75") {
		t.Fatal("the copy entries are not drawn in the link blue")
	}
	c.chooseCodes(t, codesAction{copy: true, link: broadcast.LinkDrive})
	waitFor(t, "drive link copied from the menu", func() bool { return copies(c.consoleFixture, c.link) == 1 })
	pointerCopies := copies(c.consoleFixture, pointerLink)
	c.chooseCodes(t, codesAction{copy: true, link: broadcast.LinkPointer})
	waitFor(t, "pointer link copied from the menu", func() bool { return copies(c.consoleFixture, pointerLink) == pointerCopies+1 })
	mouse(2|8, c.cell()-1) // Alt+right-click: the view-only link now, never the drive link
	waitFor(t, "view-only link copied", func() bool { return copies(c.consoleFixture, c.viewLink) >= 1 })
	// The click outside the (still open) popup closed it.
	waitFor(t, "the popup closed", func() bool { return !strings.Contains(c.screenText(), "Chrome codes") })
	if copies(c.consoleFixture, c.link) != 1 {
		t.Fatal("Alt+right-click still copies the drive link")
	}
	// rotate all, run from the menu: every link's code changes.
	mouse(8, c.cell()-1)
	waitFor(t, "the menu again", func() bool { return strings.Contains(c.screenText(), "Chrome codes") })
	c.chooseCodes(t, codesAction{})
	waitFor(t, "every code rotated", func() bool {
		now := c.cast().Codes()
		return len(now) == 3 && now[0].Code != codes[0].Code && now[1].Code != codes[1].Code && now[2].Code != codes[2].Code
	})
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
	if strings.Contains(f.screenText(), "Annotation") {
		t.Fatal("screen menu has Chrome annotation controls")
	}
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

// BR-23: a rotation reported after its cast ended -- no newer cast started,
// so the attempt still matches -- is dropped too: liveness, not the attempt
// alone, decides.
func TestRotationAfterTheCastEndedIsDropped(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	f.startLive(t)
	attempt := f.screenAttempt()
	f.clickStatus(f.liveSpan().Start + 1) // stop
	waitFor(t, "broadcast off", func() bool { return f.phase() == broadcastOff })
	waitFor(t, "session gone", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.bcast.session == nil
	})
	if f.screenAttempt() != attempt {
		t.Fatalf("the end bumped the attempt (%d → %d); this test needs the same one", attempt, f.screenAttempt())
	}
	f.con.onCodeRotated(castScreen, attempt, broadcast.LinkView, castlock.Rotation{Code: "9999", Reason: castlock.ReasonOperator, Gen: 9})
	f.con.codes.mu.Lock()
	fl := f.con.codes.flash
	f.con.codes.mu.Unlock()
	if fl != nil {
		t.Fatalf("an ended cast's rotation flashed: %+v", fl)
	}
}

// annotationBrowser keeps the selected mode as the real isolated-world rail
// does. Refusals leave that mode alone; Browser's Focus remains observable.
type annotationBrowser struct {
	webcast.Browser
	mu      sync.Mutex
	mode    string
	refusal error
}

func (b *annotationBrowser) SetAnnotationMode(mode string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.refusal != nil {
		return b.refusal
	}
	b.mode = mode
	return nil
}

func (f *consoleFixture) chooseCodesText(t *testing.T, text string) {
	t.Helper()
	f.con.codes.mu.Lock()
	box := f.con.codes.menuBox
	f.con.codes.mu.Unlock()
	for _, r := range box.rows {
		for _, w := range r.words {
			if w.text == text {
				_, _ = fmt.Fprintf(f.stdin, "\x1b[<0;%d;%dM\x1b[<0;%d;%dm", w.x0+1, r.y+1, w.x0+1, r.y+1)
				return
			}
		}
	}
	t.Fatalf("no menu word %q", text)
}

func TestChromeCodesMenuSelectsAnnotationWithoutFocus(t *testing.T) {
	b := &annotationBrowser{mode: "watch"}
	r := newWebcastRig(t, func(r *webcastRig) {
		r.wrap = func(f *webcast.FakeBrowser, _ *webcast.RecordOptions) webcast.Browser {
			b.Browser = f
			return b
		}
	})
	f := r.open(t)
	before := f.Stats()
	r.con.mu.Lock()
	cast := r.con.wcast.cast
	r.con.mu.Unlock()
	codesBefore := fmt.Sprint(cast.Codes())
	r.clickCell(t, 8)
	waitFor(t, "Chrome menu", func() bool { return strings.Contains(r.screenText(), "Chrome codes") })
	for _, tc := range []struct{ label, mode string }{{"Pen", "point"}, {"Note", "note"}, {"Off", "watch"}} {
		r.chooseCodesText(t, tc.label)
		waitFor(t, tc.label+" selected", func() bool {
			b.mu.Lock()
			defer b.mu.Unlock()
			return b.mode == tc.mode
		})
	}
	b.mu.Lock()
	b.refusal = errors.New("annotation is unavailable while paused or a viewer is driving")
	b.mu.Unlock()
	r.chooseCodesText(t, "Note")
	waitFor(t, "annotation refusal notice", func() bool {
		r.con.mu.Lock()
		defer r.con.mu.Unlock()
		return strings.Contains(r.con.feed.Row().Body, "annotation is unavailable while paused or a viewer is driving")
	})
	b.mu.Lock()
	mode := b.mode
	b.mu.Unlock()
	if mode != "watch" {
		t.Fatalf("refused mode changed to %q", mode)
	}
	if got := f.Stats(); got != before {
		t.Fatalf("annotation changed browser controls: before %+v, after %+v", before, got)
	}
	if got := fmt.Sprint(cast.Codes()); got != codesBefore {
		t.Fatalf("annotation rotated codes: before %s, after %s", codesBefore, got)
	}
}

func TestChromeCodesMenuReportsAnnotationUnavailable(t *testing.T) {
	r := newWebcastRig(t, nil) // no optional annotation capability
	r.open(t)
	r.clickCell(t, 8)
	waitFor(t, "Chrome menu", func() bool { return strings.Contains(r.screenText(), "Chrome codes") })
	r.chooseCodesText(t, "Pen")
	waitFor(t, "annotation unavailable notice", func() bool {
		r.con.mu.Lock()
		defer r.con.mu.Unlock()
		return strings.Contains(r.con.feed.Row().Body, webcast.ErrNotRecording.Error())
	})
}
