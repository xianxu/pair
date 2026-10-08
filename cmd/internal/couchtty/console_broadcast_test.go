package couchtty

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/color"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	vt "github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

const ctrlAltB = "\x1b[98;7u"

func broadcastFixture(t *testing.T, ft *broadcast.FakeTunnel, mutate func(*broadcast.Config)) *consoleFixture {
	t.Helper()
	cfg := broadcast.Config{Tunnel: ft, Ping: 50 * time.Millisecond}
	if mutate != nil {
		mutate(&cfg)
	}
	f := newFixtureBeforeRun(t, 24, 80, func(con *Console) { con.SetBroadcast(cfg) })
	f.child.Feed([]byte("child says hi"))
	waitFor(t, "child output on screen", func() bool { return strings.Contains(f.screenText(), "child says hi") })
	return f
}

func (f *consoleFixture) phase() broadcastPhase {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.bcast.phase
}

func (f *consoleFixture) session() *broadcast.Session {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.bcast.session
}

func (f *consoleFixture) notice() string {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.feed.Row().Body
}

func (f *consoleFixture) lastRow() string {
	rows := strings.Split(f.screenText(), "\n")
	return rows[len(rows)-1]
}

func (f *consoleFixture) startLive(t *testing.T) *broadcast.Session {
	t.Helper()
	_, _ = f.stdin.Write([]byte(ctrlAltB))
	waitFor(t, "broadcast live", func() bool { return f.phase() == broadcastLive })
	waitFor(t, "LIVE cell drawn", func() bool { return strings.HasPrefix(f.lastRow(), broadcast.LiveLabel) })
	return f.session()
}

// click sends a press and release at 1-based column x on the status row.
func (f *consoleFixture) clickStatus(x int) {
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<0;%d;24M\x1b[<0;%d;24m", x, x)
}

// remoteViewer reads a broadcast's SSE stream into an emulator, as the
// browser viewer does.
type remoteViewer struct {
	t     *testing.T
	r     *bufio.Reader
	resp  *http.Response
	emu   *vt.Emulator
	ended string
	theme string
}

func openViewer(t *testing.T, link string) *remoteViewer {
	t.Helper()
	resp, err := http.Get(link + "events")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("viewer refused: %d", resp.StatusCode)
	}
	return &remoteViewer{t: t, r: bufio.NewReader(resp.Body), resp: resp}
}

// next applies one event; it returns false once the broadcast ended.
func (v *remoteViewer) next() bool {
	v.t.Helper()
	var name, data string
	for {
		line, err := v.r.ReadString('\n')
		if err != nil {
			v.t.Fatalf("viewer stream broke: %v", err)
		}
		line = strings.TrimSuffix(line, "\n")
		switch {
		case strings.HasPrefix(line, "event: "):
			name = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			data = strings.TrimPrefix(line, "data: ")
		case line == "" && name != "":
			switch name {
			case "theme":
				v.theme = data
			case "end":
				v.ended = data
				return false
			case "frame":
				var m struct {
					Cols, Rows int
					B          string
				}
				if err := json.Unmarshal([]byte(data), &m); err != nil {
					v.t.Fatal(err)
				}
				b, err := base64.StdEncoding.DecodeString(m.B)
				if err != nil {
					v.t.Fatal(err)
				}
				if v.emu == nil {
					v.emu = vt.NewEmulator(m.Cols, m.Rows)
				} else if v.emu.Width() != m.Cols || v.emu.Height() != m.Rows {
					v.emu.Resize(m.Cols, m.Rows)
				}
				_, _ = v.emu.Write(b)
				return true
			}
			name, data = "", ""
		}
	}
}

// until reads events until the screen satisfies cond; it fails on end.
func (v *remoteViewer) until(what string, cond func(string) bool) {
	v.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for v.emu == nil || !cond(v.emu.String()) {
		if time.Now().After(deadline) {
			v.t.Fatalf("viewer never showed %s", what)
		}
		if !v.next() {
			v.t.Fatalf("broadcast ended (%s) before the viewer showed %s", v.ended, what)
		}
	}
}

func (v *remoteViewer) waitEnd() string {
	v.t.Helper()
	for v.next() {
	}
	return v.ended
}

func TestBroadcastKeyStartsClickStops(t *testing.T) {
	ft := &broadcast.FakeTunnel{}
	f := broadcastFixture(t, ft, nil)
	liveWidth := len([]rune(broadcast.LiveLabel))
	for _, col := range []int{1, liveWidth / 2, liveWidth} {
		t.Run(fmt.Sprintf("click column %d", col), func(t *testing.T) {
			s := f.startLive(t)
			if !strings.Contains(f.host.Written(), "\x1b]52;c;"+base64.StdEncoding.EncodeToString([]byte(s.Link()))) {
				t.Fatal("the link was not copied to the clipboard")
			}
			if strings.Contains(f.screenText(), s.Link()) {
				t.Fatal("the link is drawn on screen, where it would be broadcast")
			}
			v := openViewer(t, s.Link())
			v.until("the composed screen", func(text string) bool {
				return strings.Contains(text, "child says hi") && strings.Contains(text, broadcast.LiveLabel)
			})
			f.clickStatus(col)
			if end := v.waitEnd(); !strings.Contains(end, "operator stopped") {
				t.Fatalf("end reason %q", end)
			}
			waitFor(t, "broadcast off", func() bool { return f.phase() == broadcastOff })
			waitFor(t, "LIVE cell gone", func() bool { return !strings.Contains(f.lastRow(), "LIVE") })
		})
	}
	if st := ft.Stats(); st.Opens != 3 || st.Closes != 3 {
		t.Fatalf("fake tunnel %+v", st)
	}
}

func TestBroadcastClickPastTheCellDoesNotStop(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	f.startLive(t)
	f.clickStatus(len([]rune(broadcast.LiveLabel)) + 1)
	time.Sleep(100 * time.Millisecond)
	if f.phase() != broadcastLive {
		t.Fatalf("a click past the red span changed the broadcast: phase %v", f.phase())
	}
}

func TestBroadcastKeyToggles(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	s := f.startLive(t)
	v := openViewer(t, s.Link())
	_, _ = f.stdin.Write([]byte(ctrlAltB))
	v.waitEnd()
	waitFor(t, "broadcast off", func() bool { return f.phase() == broadcastOff })
	for _, w := range f.child.Writes() {
		if strings.Contains(string(w), ctrlAltB) {
			t.Fatal("Ctrl+Alt+b reached the child")
		}
	}
}

func TestBroadcastSwitcherPrivate(t *testing.T) {
	for _, show := range []bool{false, true} {
		t.Run(fmt.Sprintf("showSwitcher=%v", show), func(t *testing.T) {
			f := broadcastFixture(t, &broadcast.FakeTunnel{}, func(c *broadcast.Config) { c.Hub.ShowSwitcher = show })
			s := f.startLive(t)
			v := openViewer(t, s.Link())
			v.until("the actor view", func(text string) bool { return strings.Contains(text, "child says hi") })
			_, _ = f.stdin.Write([]byte{0}) // Ctrl+Space opens the switcher
			waitFor(t, "switcher open", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); return f.con.focus.IsPanel() })
			if show {
				v.until("the switcher", func(text string) bool {
					return !strings.Contains(text, "child says hi") && !strings.Contains(text, broadcast.PlaceholderText)
				})
			} else {
				v.until("the placeholder", func(text string) bool { return strings.Contains(text, broadcast.PlaceholderText) })
			}
		})
	}
}

func TestBroadcastStopsWhenIndicatorCannotBeDrawn(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, func(c *broadcast.Config) { c.Hub.Grace = 100 * time.Millisecond })
	s := f.startLive(t)
	v := openViewer(t, s.Link())
	// Narrower than the label: the indicator is clipped on the operator's
	// screen, so nothing more may be streamed, and the broadcast must end.
	f.host.SetSize(ptychild.Size{Rows: 24, Cols: uint16(len([]rune(broadcast.LiveLabel)) - 1)})
	if end := v.waitEnd(); !strings.Contains(end, "LIVE indicator") {
		t.Fatalf("end reason %q", end)
	}
	waitFor(t, "broadcast off", func() bool { return f.phase() == broadcastOff })
	// At this width the row can't show it, so read the notice from the feed.
	waitFor(t, "notice explains", func() bool { return strings.Contains(f.notice(), "Broadcast ended") })
}

func TestBroadcastToggleWhileStarting(t *testing.T) {
	ft := &broadcast.FakeTunnel{OpenDelay: 300 * time.Millisecond, IgnoreCancel: true}
	f := broadcastFixture(t, ft, nil)
	_, _ = f.stdin.Write([]byte(ctrlAltB))
	waitFor(t, "starting", func() bool { return f.phase() == broadcastStarting })
	waitFor(t, "starting cell", func() bool { return strings.HasPrefix(f.lastRow(), broadcast.StartingLabel) })
	_, _ = f.stdin.Write([]byte(ctrlAltB))
	waitFor(t, "cancelled", func() bool { return f.phase() == broadcastOff })
	// The late tunnel finishes opening after the cancel; it is closed, not adopted.
	waitFor(t, "late tunnel closed", func() bool { st := ft.Stats(); return st.Opens == 1 && st.Closes == 1 })
	if f.session() != nil {
		t.Fatal("a cancelled start left a session")
	}
}

func TestBroadcastStopIsOffTheInputPath(t *testing.T) {
	release := make(chan struct{})
	ft := &broadcast.FakeTunnel{CloseBlock: release}
	f := broadcastFixture(t, ft, nil)
	defer close(release)
	s := f.startLive(t)
	v := openViewer(t, s.Link())
	f.clickStatus(1)
	v.waitEnd()
	waitFor(t, "cell gone while the tunnel is still closing", func() bool {
		return f.phase() == broadcastStopping && !strings.Contains(f.lastRow(), "LIVE")
	})
	_, _ = f.stdin.Write([]byte("typed during teardown"))
	waitFor(t, "input reaches the child", func() bool {
		return strings.Contains(string(bytes.Join(f.child.Writes(), nil)), "typed during teardown")
	})
	_, _ = f.stdin.Write([]byte(ctrlAltB))
	waitFor(t, "still-stopping notice", func() bool { return strings.Contains(f.notice(), "still stopping") })
	if f.phase() != broadcastStopping {
		t.Fatalf("toggle during teardown changed phase to %v", f.phase())
	}
}

func TestBroadcastStoppedOnShutdown(t *testing.T) {
	ft := &broadcast.FakeTunnel{}
	f := broadcastFixture(t, ft, nil)
	s := f.startLive(t)
	v := openViewer(t, s.Link())
	f.con.Stop()
	select {
	case <-f.done:
	case <-time.After(10 * time.Second):
		t.Fatal("console did not finish")
	}
	if end := v.waitEnd(); !strings.Contains(end, "operator stopped") {
		t.Fatalf("end reason %q", end)
	}
	if st := ft.Stats(); st.Closes != 1 {
		t.Fatalf("tunnel not closed at shutdown: %+v", st)
	}
}

func TestBroadcastNotConfigured(t *testing.T) {
	f := newFixture(t, 24, 80)
	_, _ = f.stdin.Write([]byte(ctrlAltB))
	waitFor(t, "notice", func() bool { return strings.Contains(f.notice(), "not configured") })
	if strings.Contains(f.lastRow(), "LIVE") || f.phase() != broadcastOff {
		t.Fatal("an unconfigured console drew a broadcast cell")
	}
}

// BR-11: exactly one side owns a finished start's session, however the
// adopt (loop) and abandon (start goroutine) calls race.
func TestStartClaimDecidesOnce(t *testing.T) {
	for range 2000 {
		var c startClaim
		var adopted, abandoned atomic.Bool
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); adopted.Store(c.adopt()) }()
		go func() { defer wg.Done(); abandoned.Store(c.abandon()) }()
		wg.Wait()
		if adopted.Load() == abandoned.Load() {
			t.Fatalf("adopted=%v abandoned=%v: want exactly one owner", adopted.Load(), abandoned.Load())
		}
	}
}

// BR-12: Couch's exit is bounded even when the tunnel never finishes
// closing.
func TestBroadcastShutdownBoundedByStuckTunnel(t *testing.T) {
	old := broadcastShutdownWait
	broadcastShutdownWait = 300 * time.Millisecond
	t.Cleanup(func() { broadcastShutdownWait = old })
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	f := broadcastFixture(t, &broadcast.FakeTunnel{CloseBlock: release}, nil)
	f.startLive(t)
	start := time.Now()
	f.con.Stop()
	select {
	case <-f.done:
	case <-time.After(5 * time.Second):
		t.Fatal("Couch's exit waited on a tunnel that never closes")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("exit took %v", d)
	}
}

// With no broadcast running, the row's first columns are not a broadcast
// target: clicking them changes nothing about broadcasting.
func TestBroadcastOffHasNoClickTarget(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	for col := 1; col <= len([]rune(broadcast.LiveLabel)); col++ {
		f.clickStatus(col)
	}
	time.Sleep(100 * time.Millisecond)
	if f.phase() != broadcastOff || f.session() != nil {
		t.Fatalf("a click on an off row started a broadcast: phase %v", f.phase())
	}
	f.con.mu.Lock()
	control := f.con.statusControl
	f.con.mu.Unlock()
	if control != (ColumnSpan{}) {
		t.Fatalf("off row has a control span %+v", control)
	}
}

// The operator's palette, as the terminal reported it, reaches a viewer
// before its first frame (#395 M4).
func TestBroadcastSendsOperatorTheme(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	for _, ev := range []uv.Event{
		uv.ForegroundColorEvent{Color: color.RGBA{0xee, 0xee, 0xee, 0xff}},
		uv.BackgroundColorEvent{Color: color.RGBA{0x10, 0x10, 0x10, 0xff}},
		uv.UnknownOscEvent("\x1b]4;1;rgb:ffff/5555/5555\x1b\\"),
	} {
		f.con.routeInputEvent(terminal.InputEvent{Event: ev, Reply: true})
	}
	s := f.startLive(t)
	v := openViewer(t, s.Link())
	v.until("a frame", func(string) bool { return true })
	var theme broadcast.Theme
	if err := json.Unmarshal([]byte(v.theme), &theme); err != nil {
		t.Fatalf("theme event %q: %v", v.theme, err)
	}
	if theme.Foreground != "#eeeeee" || theme.Background != "#101010" || theme.ANSI[1] != "#ff5555" || theme.ANSI[0] != "" {
		t.Fatalf("theme %+v", theme)
	}
}

// The ordering behind BR-11, driven deterministically: the start goroutine
// abandoned (and so is stopping) the session before the loop ran its
// handover. The loop must not adopt it.
func TestBroadcastLateHandoverAfterAbandonIsNotAdopted(t *testing.T) {
	ft := &broadcast.FakeTunnel{}
	f := broadcastFixture(t, ft, nil)
	s, err := broadcast.Start(context.Background(), broadcast.Config{Tunnel: ft})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.Stop(nil); <-s.Done() }()
	claim := &startClaim{}
	if !claim.abandon() {
		t.Fatal("fresh claim refused abandon")
	}
	err = f.con.runTerminalCommand(context.Background(), func() error {
		f.con.mu.Lock()
		f.con.bcast = broadcastState{phase: broadcastStarting, attempt: 41, cancel: func() {}}
		f.con.mu.Unlock()
		f.con.broadcastStarted(41, s, nil, claim)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.phase() != broadcastOff || f.session() != nil {
		t.Fatalf("adopted an abandoned session: phase %v", f.phase())
	}
	if strings.Contains(f.lastRow(), "LIVE") {
		t.Fatal("LIVE drawn for an abandoned session")
	}
}
