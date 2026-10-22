package couchtty

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"github.com/xianxu/pair/cmd/internal/webcast"
)

const shiftCtrlAltB = "\x1b[98;8u"

// webcastRig is a console whose cast Chrome is a FakeBrowser. Launches are
// counted; gate, when set, holds each launch until closed.
type webcastRig struct {
	*consoleFixture
	tunnel *broadcast.FakeTunnel
	mu     sync.Mutex
	fakes  []*webcast.FakeBrowser
	gate   chan struct{}
	fail   error
	// wrap, when set, records the cast: Launch gets a recordings directory
	// and returns wrap's Browser around each fake (#470's web notes).
	wrap func(*webcast.FakeBrowser, *webcast.RecordOptions) webcast.Browser
	// screen also configures the screen broadcast, on the same tunnel.
	screen bool
}

func newWebcastRig(t *testing.T, tweak func(*webcastRig)) *webcastRig {
	t.Helper()
	r := &webcastRig{tunnel: &broadcast.FakeTunnel{}}
	if tweak != nil {
		tweak(r)
	}
	cfg := webcast.CastConfig{
		Host: broadcast.NewHost(broadcast.HostConfig{Tunnel: r.tunnel, ProbeTimeout: 2 * time.Second}),
		Ping: 50 * time.Millisecond,
		Launch: func(ctx context.Context, ro *webcast.RecordOptions) (webcast.Browser, error) {
			if r.gate != nil {
				select {
				case <-r.gate:
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			r.mu.Lock()
			defer r.mu.Unlock()
			if r.fail != nil {
				return nil, r.fail
			}
			f := webcast.NewFakeBrowser()
			r.fakes = append(r.fakes, f)
			if r.wrap != nil {
				return r.wrap(f, ro), nil
			}
			return f, nil
		},
	}
	if r.wrap != nil {
		cfg.RecordingsDir = t.TempDir()
	}
	r.consoleFixture = newFixtureBeforeRun(t, 24, 80, func(con *Console) {
		con.SetWebcast(cfg)
		if r.screen {
			con.SetBroadcast(broadcast.Config{Tunnel: r.tunnel, Ping: 50 * time.Millisecond})
		}
	})
	r.child.Feed([]byte("child says hi"))
	waitFor(t, "child output on screen", func() bool { return strings.Contains(r.screenText(), "child says hi") })
	return r
}

func (r *webcastRig) launches() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.fakes)
}

func (r *webcastRig) browser() *webcast.FakeBrowser {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.fakes[len(r.fakes)-1]
}

func (r *webcastRig) wphase() webcastPhase {
	r.con.mu.Lock()
	defer r.con.mu.Unlock()
	return r.con.wcast.phase
}

// drawnCell is the Chrome cell's label and 1-based column on the drawn row,
// or ("", 0). The cell is drawn before any notice, and notices can quote
// its labels, so the earliest label on the row is the cell.
func (r *webcastRig) drawnCell() (string, int) {
	row := []rune(r.statusRow())
	for i := range row {
		for _, l := range []string{WebcastIdleLabel, WebcastLiveLabel} {
			if strings.HasPrefix(string(row[i:]), l) {
				return l, i + 1
			}
		}
	}
	return "", 0
}

func (r *webcastRig) cell() int {
	_, x := r.drawnCell()
	return x
}

func (r *webcastRig) waitCell(t *testing.T, label string) {
	t.Helper()
	waitFor(t, "cell "+label, func() bool { l, _ := r.drawnCell(); return l == label })
}

func (r *webcastRig) open(t *testing.T) *webcast.FakeBrowser {
	t.Helper()
	_, _ = r.stdin.Write([]byte(shiftCtrlAltB))
	r.waitCell(t, webcastIdleLabelForTest)
	return r.browser()
}

func (r *webcastRig) clickCell(t *testing.T, button int) {
	t.Helper()
	x := r.cell()
	if x == 0 {
		t.Fatal("no Chrome cell to click")
	}
	_, _ = fmt.Fprintf(r.stdin, "\x1b[<%d;%d;%dM\x1b[<%d;%d;%dm", button, x+1, r.statusRowIndex(), button, x+1, r.statusRowIndex())
}

const webcastIdleLabelForTest = WebcastIdleLabel

func TestWebcastKeyOpensThenTogglesLive(t *testing.T) {
	r := newWebcastRig(t, nil)
	if r.cell() != 0 {
		t.Fatal("cell drawn before Chrome runs")
	}
	f := r.open(t)
	_, _ = r.stdin.Write([]byte(shiftCtrlAltB))
	r.waitCell(t, WebcastLiveLabel)
	if f.State().Phase != webcast.PhaseLive {
		t.Fatalf("browser %+v", f.State())
	}
	_, _ = r.stdin.Write([]byte(shiftCtrlAltB))
	r.waitCell(t, WebcastIdleLabel)
	// Ctrl+Alt+b is still the screen broadcast's, not this.
	_, _ = r.stdin.Write([]byte(ctrlAltB))
	waitFor(t, "screen broadcast notice", func() bool { return strings.Contains(r.notice(), "Broadcasting is not configured") })
	if f.State().Phase != webcast.PhaseIdle || r.launches() != 1 {
		t.Fatalf("Ctrl+Alt+b reached the cast: %+v launches %d", f.State(), r.launches())
	}
}

func TestWebcastClicks(t *testing.T) {
	r := newWebcastRig(t, nil)
	f := r.open(t)
	r.clickCell(t, 0)
	r.waitCell(t, WebcastLiveLabel)
	r.clickCell(t, 0)
	r.waitCell(t, WebcastIdleLabel)
	// The page's own control drives the same state.
	f.ClickPageControl()
	r.waitCell(t, WebcastLiveLabel)
	// A click elsewhere on the row does nothing to the cast.
	_, _ = fmt.Fprintf(r.stdin, "\x1b[<0;%d;24M\x1b[<0;%d;24m", 70, 70)
	time.Sleep(100 * time.Millisecond)
	if f.State().Phase != webcast.PhaseLive {
		t.Fatal("a click off the cell toggled the cast")
	}
}

func TestWebcastRightClickCopiesLink(t *testing.T) {
	r := newWebcastRig(t, func(r *webcastRig) { r.tunnel.OpenDelay = 300 * time.Millisecond })
	r.open(t)
	r.clickCell(t, 2)
	waitFor(t, "not ready notice", func() bool { return strings.Contains(r.notice(), "not ready") })
	var link string
	waitFor(t, "link", func() bool {
		r.con.mu.Lock()
		defer r.con.mu.Unlock()
		link = r.con.wcast.update.Link
		return link != ""
	})
	r.clickCell(t, 2)
	osc := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(link))
	waitFor(t, "OSC 52 with the link", func() bool { return strings.Contains(r.host.Written(), osc) })
	if strings.Contains(r.notice(), link) {
		t.Fatal("the notice shows the link")
	}
}

func TestWebcastCtrlClickEnds(t *testing.T) {
	for _, button := range []int{0 | 16, 2 | 16} {
		t.Run(fmt.Sprint(button), func(t *testing.T) {
			r := newWebcastRig(t, nil)
			f := r.open(t)
			r.clickCell(t, button)
			waitFor(t, "cell gone", func() bool { return r.cell() == 0 && r.wphase() == webcastOff })
			select {
			case <-f.Done():
			default:
				t.Fatal("Chrome not closed")
			}
			if f.Stats().Closes == 0 {
				t.Fatal("ended without Close")
			}
			if st := r.tunnel.Stats(); st.Opens != st.Closes {
				t.Fatalf("link left open: %+v", st)
			}
		})
	}
}

func TestWebcastChromeClosedByHand(t *testing.T) {
	r := newWebcastRig(t, nil)
	f := r.open(t)
	f.QuitByHand()
	waitFor(t, "cell gone", func() bool { return r.cell() == 0 && r.wphase() == webcastOff })
	waitFor(t, "notice", func() bool { return strings.Contains(r.notice(), "Chrome closed") })
	// The key starts a fresh one.
	r.open(t)
	if r.launches() != 2 {
		t.Fatalf("launches %d", r.launches())
	}
}

func TestWebcastKeyWhileStartingLaunchesOnce(t *testing.T) {
	gate := make(chan struct{})
	r := newWebcastRig(t, func(r *webcastRig) { r.gate = gate })
	_, _ = r.stdin.Write([]byte(shiftCtrlAltB))
	waitFor(t, "starting", func() bool { return r.wphase() == webcastStarting })
	_, _ = r.stdin.Write([]byte(shiftCtrlAltB))
	waitFor(t, "notice", func() bool { return strings.Contains(r.notice(), "starting") })
	close(gate)
	r.waitCell(t, WebcastIdleLabel)
	if r.launches() != 1 {
		t.Fatalf("launches %d", r.launches())
	}
}

func TestWebcastLaunchFailure(t *testing.T) {
	r := newWebcastRig(t, func(r *webcastRig) { r.fail = errors.New("no Chrome here") })
	_, _ = r.stdin.Write([]byte(shiftCtrlAltB))
	waitFor(t, "notice", func() bool { return strings.Contains(r.notice(), "no Chrome here") })
	if r.wphase() != webcastOff || r.cell() != 0 {
		t.Fatal("failed start left state behind")
	}
}

func TestWebcastEndedOnShutdown(t *testing.T) {
	r := newWebcastRig(t, nil)
	f := r.open(t)
	r.con.Stop()
	select {
	case <-r.done:
	case <-testwait.After(t):
		t.Fatal("console did not finish")
	}
	select {
	case <-f.Done():
	default:
		t.Fatal("Chrome left running at shutdown")
	}
	if st := r.tunnel.Stats(); st.Opens != st.Closes {
		t.Fatalf("link left open: %+v", st)
	}
}

// A launch that finishes after Couch began shutting down is closed, never
// adopted.
func TestWebcastLateLaunchAfterShutdownIsClosed(t *testing.T) {
	gate := make(chan struct{})
	r := newWebcastRig(t, func(r *webcastRig) { r.gate = gate })
	_, _ = r.stdin.Write([]byte(shiftCtrlAltB))
	waitFor(t, "starting", func() bool { return r.wphase() == webcastStarting })
	r.con.Stop()
	close(gate)
	select {
	case <-r.done:
	case <-testwait.After(t):
		t.Fatal("console did not finish")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.fakes {
		select {
		case <-f.Done():
		default:
			t.Fatal("a late Chrome was left running")
		}
	}
}

func TestWebcastNotConfigured(t *testing.T) {
	f := newFixture(t, 24, 80)
	_, _ = f.stdin.Write([]byte(shiftCtrlAltB))
	waitFor(t, "notice", func() bool { return strings.Contains(f.notice(), "Web casting is not configured") })
	if strings.Contains(f.lastRow(), WebcastIdleLabel) {
		t.Fatal("cell drawn")
	}
}

// A replay share becoming ready is a notice (#477): the operator sees what
// the tunnel serves, whoever asked for it, and the notice never carries
// the link. With the tunnel off it says the link is this machine's only.
func TestReplayShareReadyIsANotice(t *testing.T) {
	for _, c := range []struct {
		tunnel broadcast.Tunnel
		want   string
	}{
		{&broadcast.FakeTunnel{}, "Replay of rec-1 shared through the broadcast tunnel until Couch exits (1 shared)."},
		{broadcast.LocalOnly{}, "Replay of rec-1 shared on this machine only (the tunnel is off) until Couch exits (1 shared)."},
	} {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "rec-1"), 0o755); err != nil {
			t.Fatal(err)
		}
		shares := webcast.NewReplayShares(broadcast.NewHost(broadcast.HostConfig{Tunnel: c.tunnel, ProbeTimeout: 2 * time.Second}), root)
		f := newFixtureBeforeRun(t, 24, 120, func(con *Console) { con.SetReplayShares(shares) })
		if got := f.con.ReplayShare("replay-share", "rec-1"); got.State != "opening" {
			t.Fatalf("share: %+v", got)
		}
		waitFor(t, "ready notice", func() bool { return strings.Contains(f.notice(), c.want) })
		st := f.con.ReplayShare("replay-status", "rec-1")
		if token := strings.TrimPrefix(st.Base[strings.LastIndex(st.Base, "/"):], "/"); token == "" || strings.Contains(f.notice(), token) {
			t.Fatalf("notice %q carries the link %q", f.notice(), st.Base)
		}
		if got := f.con.SharedReplays(); len(got) != 1 || got[0] != "rec-1" {
			t.Fatalf("shared %q", got)
		}
		shares.Close()
	}
}

// A click on ↗ (#501) asks the browser to come to the front; the Chrome
// cell itself still toggles live.
func TestWebcastFocusButton(t *testing.T) {
	r := newWebcastRig(t, nil)
	f := r.open(t)
	x := r.cell() + len([]rune(WebcastIdleLabel)) + 1 // 1-based column of ↗
	if got := []rune(r.lastRow())[x-1]; string(got) != WebcastFocusLabel {
		t.Fatalf("no button at %d: %q", x, r.lastRow())
	}
	_, _ = fmt.Fprintf(r.stdin, "\x1b[<0;%d;24M\x1b[<0;%d;24m", x, x)
	waitFor(t, "browser focused", func() bool { return f.Stats().Focuses == 1 })
	if f.State().Phase != webcast.PhaseIdle {
		t.Fatal("the button toggled the cast")
	}
}

// 👆 and 👽 in the Chrome group (#501) copy the pointer and the drive link
// on enable or right-click; right-click on
// the cell keeps the view-only link. A recorded cast, so it takes notes and
// has a pointer link.
func TestWebcastLinkButtons(t *testing.T) {
	r := newWebNotesCast(t, false)
	r.con.mu.Lock()
	cast := r.con.wcast.cast
	r.con.mu.Unlock()
	if err := cast.SetCapability(context.Background(), webcast.CapabilityAnnotation, false); err != nil {
		t.Fatal(err)
	}
	var u webcast.Update
	waitFor(t, "links", func() bool {
		r.con.mu.Lock()
		defer r.con.mu.Unlock()
		u = r.con.wcast.update
		return u.Link != "" && u.PointerLink != ""
	})
	click := func(span func(*Console) ColumnSpan, button int) {
		r.con.mu.Lock()
		x := span(r.con).Start + 1
		r.con.mu.Unlock()
		if x == 1 {
			t.Fatal("button not drawn")
		}
		_, _ = fmt.Fprintf(r.stdin, "\x1b[<%d;%d;24M\x1b[<%d;%d;24m", button, x, button, x)
	}
	osc := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(u.PointerLink))
	for i, button := range []int{0, 2} {
		click(func(c *Console) ColumnSpan { return c.statusWebPointerLink }, button)
		want := i + 1
		waitFor(t, "pointer link copied", func() bool { return strings.Count(r.host.Written(), osc) == want })
	}
	if strings.Contains(r.host.Written(), "\x1b]52;c;"+base64.StdEncoding.EncodeToString([]byte(u.Link))) {
		t.Fatal("👆 copied the view-only link")
	}
	click(func(c *Console) ColumnSpan { return c.statusWebDriveLink }, 2)
	if u.DriveLink != "" {
		osc = "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(u.DriveLink))
		waitFor(t, "drive link copied", func() bool { return strings.Contains(r.host.Written(), osc) })
	} else {
		// A fake can't be driven: the click says why instead.
		waitFor(t, "drive notice", func() bool { return strings.Contains(r.notice(), "can't be driven") })
	}
	if f := r.browser(); f.State().Phase != webcast.PhaseIdle {
		t.Fatal("a link button toggled the cast")
	}
}
