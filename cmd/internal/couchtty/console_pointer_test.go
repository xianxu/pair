package couchtty

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

const markTint = ansi.IndexedColor(214)

func (f *consoleFixture) pointerPhase() pointerPhase {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.pointer
}

func (f *consoleFixture) pointerSpan() ColumnSpan {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.statusPointer
}

func (f *consoleFixture) clickPointer(t *testing.T, button int) {
	t.Helper()
	var span ColumnSpan
	waitFor(t, "pointer control drawn", func() bool { span = f.pointerSpan(); return span != (ColumnSpan{}) })
	x := span.Start + 1
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<%d;%d;24M\x1b[<%d;%d;24m", button, x, button, x)
}

// screenBg is the operator screen's background at a zero-based cell.
func (f *consoleFixture) screenBg(x, y int) any {
	f.screen.mu.Lock()
	defer f.screen.mu.Unlock()
	if c := f.screen.em.CellAt(x, y); c != nil {
		return c.Style.Bg
	}
	return nil
}

func copies(f *consoleFixture, link string) int {
	return strings.Count(f.host.Written(), "\x1b]52;c;"+base64.StdEncoding.EncodeToString([]byte(link)))
}

func postPoint(t *testing.T, link string, col, row int) int {
	t.Helper()
	body := fmt.Sprintf(`{"cols":80,"rows":24,"down":false,"points":[[%d,%d]]}`, col, row)
	resp, err := http.Post(link+"point", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// tapUntilMarked posts a tap until its mark is on the operator's screen. A
// point is dropped until the hub has seen a frame showing the active pointer
// marker, which can trail the operator's screen by a moment, so a single post
// right after the marker appears can be dropped; a helper would simply tap
// again.
func tapUntilMarked(t *testing.T, f *consoleFixture, link string, col, row int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if code := postPoint(t, link, col, row); code != http.StatusNoContent {
			t.Fatalf("POST %d", code)
		}
		for range 5 {
			if f.screenBg(col, row) == markTint {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		if time.Now().After(deadline) {
			t.Fatalf("no mark at %d,%d", col, row)
		}
	}
}

// pointerOnFixture is a live broadcast with pointing turned on by a click.
func pointerOnFixture(t *testing.T, mutate func(*broadcast.Config)) (*consoleFixture, *broadcast.Session, string) {
	t.Helper()
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, mutate)
	s := f.startLive(t)
	f.clickPointer(t, 0)
	waitFor(t, "pointing on", func() bool { return f.pointerPhase() == pointerOn })
	link := s.PointerLink()
	if link == "" || link == s.Link() {
		t.Fatalf("pointer link %q (view %q)", link, s.Link())
	}
	waitFor(t, "active 👆 drawn", func() bool { return f.screenBg(f.pointerSpan().Start, 23) == markTint })
	return f, s, link
}

func TestPointerClickMarksOperatorAndViewers(t *testing.T) {
	f, s, link := pointerOnFixture(t, nil)
	if copies(f, link) != 1 {
		t.Fatal("pointer link not copied")
	}
	if strings.Contains(f.screenText(), link) || strings.Contains(f.notice(), link) {
		t.Fatal("the pointer link is drawn on screen")
	}
	v := openViewer(t, s.Link())
	before := len(bytes.Join(f.child.Writes(), nil))
	tapUntilMarked(t, f, link, 5, 3)
	v.until("mark in the broadcast", func(string) bool {
		c := v.emu.CellAt(5, 3)
		return c != nil && c.Style.Bg == markTint
	})
	if after := len(bytes.Join(f.child.Writes(), nil)); after != before {
		t.Fatal("pointer input reached the child")
	}
}

func TestPointerToggleOffAndOnKeepsTheLink(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	tapUntilMarked(t, f, link, 5, 3)
	f.clickPointer(t, 0)
	waitFor(t, "pointing off", func() bool { return f.pointerPhase() == pointerOff })
	waitFor(t, "marks cleared", func() bool { return f.screenBg(5, 3) != markTint })
	waitFor(t, "off notice", func() bool { return strings.Contains(f.notice(), "Pointing off") })
	if code := postPoint(t, link, 6, 3); code != http.StatusForbidden {
		t.Fatalf("POST while off: %d", code)
	}
	f.clickPointer(t, 0)
	waitFor(t, "pointing on again", func() bool { return f.pointerPhase() == pointerOn })
	waitFor(t, "same link re-copied", func() bool { return copies(f, link) == 2 })
}

func TestPointerRightClickRecopies(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	f.clickPointer(t, 2)
	waitFor(t, "re-copied", func() bool { return copies(f, link) == 2 })
	if f.pointerPhase() != pointerOn {
		t.Fatal("right-click toggled pointing")
	}
}

func TestRemoteClickIsANoticeOnly(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	f.startLive(t)
	var span ColumnSpan
	waitFor(t, "👽 drawn", func() bool {
		f.con.mu.Lock()
		span = f.con.statusRemote
		f.con.mu.Unlock()
		return span != (ColumnSpan{})
	})
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<0;%d;24M\x1b[<0;%d;24m", span.Start+1, span.Start+1)
	waitFor(t, "notice", func() bool { return strings.Contains(f.notice(), "isn't available yet") })
	if f.pointerPhase() != pointerNone || f.phase() != broadcastLive {
		t.Fatal("👽 changed something")
	}
}

func TestPointerDroppedWhileSwitcherOpen(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	// First prove points land, so the drop below is the switcher's doing;
	// then let that mark fade.
	tapUntilMarked(t, f, link, 9, 4)
	waitFor(t, "probe mark faded", func() bool {
		f.con.pmarks.mu.Lock()
		defer f.con.pmarks.mu.Unlock()
		return !f.con.pmarks.marks.Live(time.Now())
	})
	_, _ = f.stdin.Write([]byte{0})
	waitFor(t, "switcher open", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); return f.con.focus.IsPanel() })
	postPoint(t, link, 5, 3)
	time.Sleep(100 * time.Millisecond)
	f.con.pmarks.mu.Lock()
	marks := f.con.pmarks.marks
	f.con.pmarks.mu.Unlock()
	if marks != nil && marks.Live(time.Now()) {
		t.Fatal("a point landed while the switcher was open")
	}
}

// A batch that passed the session's checks but reaches the Console after
// pointing turned off is dropped there (security review L1).
func TestPointerLateBatchAfterOffDropped(t *testing.T) {
	f, _, _ := pointerOnFixture(t, nil)
	f.clickPointer(t, 0)
	waitFor(t, "pointing off", func() bool { return f.pointerPhase() == pointerOff })
	_ = f.con.runTerminalCommand(f.con.lifetime, func() error {
		f.con.applyPoints(broadcast.PointBatch{Cols: 80, Rows: 24, Points: [][2]int{{5, 3}}})
		return nil
	})
	f.con.pmarks.mu.Lock()
	defer f.con.pmarks.mu.Unlock()
	if f.con.pmarks.marks != nil && f.con.pmarks.marks.Live(time.Now()) {
		t.Fatal("a late batch landed after pointing turned off")
	}
}

func TestPointerClippedTurnsOff(t *testing.T) {
	f, _, link := pointerOnFixture(t, func(c *broadcast.Config) { c.Hub.Grace = 100 * time.Millisecond })
	// Wide enough for LIVE, too narrow for 👆.
	f.host.SetSize(ptychild.Size{Rows: 24, Cols: uint16(len([]rune(broadcast.LiveLabel)) + 2)})
	waitFor(t, "pointing off by the watch", func() bool { return f.pointerPhase() == pointerOff })
	waitFor(t, "notice", func() bool { return strings.Contains(f.notice(), "wasn't visible") })
	if f.phase() != broadcastLive {
		t.Fatalf("the broadcast ended with the pointer (phase %v)", f.phase())
	}
	if code := postPoint(t, link, 1, 1); code != http.StatusForbidden {
		t.Fatalf("POST after the watch turned pointing off: %d", code)
	}
}

func TestPointerMarksFade(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	tapUntilMarked(t, f, link, 7, 2)
	deadline := time.Now().Add(broadcast.MarkLife + 3*time.Second)
	// Fading steps through dimmer tints; gone means the default background.
	for f.screenBg(7, 2) != nil {
		if time.Now().After(deadline) {
			t.Fatalf("mark still drawn after %v: %v", broadcast.MarkLife, f.screenBg(7, 2))
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestPointerEndsWithBroadcast(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	tapUntilMarked(t, f, link, 5, 3)
	f.clickStatus(1)
	waitFor(t, "broadcast off", func() bool { return f.phase() == broadcastOff })
	waitFor(t, "pointer forgotten", func() bool { return f.pointerPhase() == pointerNone })
	waitFor(t, "marks gone", func() bool { return f.screenBg(5, 3) != markTint })
}

// The lock discipline under load: the child paints continuously while
// batches arrive and pointing toggles. A lock held across the Presenter,
// session or loop would deadlock here; the deadline turns that into a failure.
func TestPointerStressNoDeadlock(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	stop := make(chan struct{})
	done := make(chan struct{}, 3)
	go func() {
		defer func() { done <- struct{}{} }()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			f.child.Feed([]byte(fmt.Sprintf("\r\nline %d", i)))
			time.Sleep(time.Millisecond)
		}
	}()
	go func() {
		defer func() { done <- struct{}{} }()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			postPoint(t, link, i%70, 1+i%20)
			time.Sleep(5 * time.Millisecond)
		}
	}()
	go func() {
		defer func() { done <- struct{}{} }()
		for range 6 {
			f.clickPointer(t, 0)
			time.Sleep(80 * time.Millisecond)
		}
	}()
	time.Sleep(time.Second)
	close(stop)
	deadline := time.After(10 * time.Second)
	for range 3 {
		select {
		case <-done:
		case <-deadline:
			t.Fatal("pointer stress deadlocked")
		}
	}
	// The console still answers.
	_, _ = f.stdin.Write([]byte("still alive"))
	waitFor(t, "input still reaches the child", func() bool {
		return strings.Contains(string(bytes.Join(f.child.Writes(), nil)), "still alive")
	})
}

// The helper can point at the tab bar (a thread chip), but not at the
// broadcast's own controls.
func TestPointerOnTabBar(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	tab := broadcast.StatusGuardCols + 4
	tapUntilMarked(t, f, link, tab, 23)
	postPoint(t, link, 2, 23)
	time.Sleep(100 * time.Millisecond)
	if f.screenBg(2, 23) == markTint {
		t.Fatal("a mark covered LIVE")
	}
}

// Right-click on LIVE ⏸ re-copies the view-only link and keeps broadcasting.
func TestLiveRightClickRecopiesViewLink(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	s := f.startLive(t)
	before := copies(f, s.Link())
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<2;2;24M\x1b[<2;2;24m")
	waitFor(t, "view link re-copied", func() bool { return copies(f, s.Link()) == before+1 })
	if f.phase() != broadcastLive {
		t.Fatal("right-click stopped the broadcast")
	}
	waitFor(t, "notice", func() bool { return strings.Contains(f.notice(), "link copied") })
}
