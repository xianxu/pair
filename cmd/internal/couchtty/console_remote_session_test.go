package couchtty

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/remotesession"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

type fixtureRemoteSession struct {
	mu              sync.Mutex
	status          remotesession.RemoteStatus
	link            string
	on, off, joined int
	changed         chan struct{}
}

func (s *fixtureRemoteSession) On() error { s.mu.Lock(); defer s.mu.Unlock(); s.on++; return nil }
func (s *fixtureRemoteSession) Off()      { s.mu.Lock(); defer s.mu.Unlock(); s.off++ }
func (s *fixtureRemoteSession) Join(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.joined++
	return nil
}
func (s *fixtureRemoteSession) Snapshot() remotesession.RemoteStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}
func (s *fixtureRemoteSession) ConsoleLink() string      { s.mu.Lock(); defer s.mu.Unlock(); return s.link }
func (s *fixtureRemoteSession) Changes() <-chan struct{} { return s.changed }

func TestRemoteSessionResetRequiresDecodedHostKeyPress(t *testing.T) {
	var reset atomic.Int32
	s := &fixtureRemoteSession{status: remotesession.RemoteStatus{Status: "disabled", CanReset: true}}
	f := newFixtureBeforeRun(t, 24, 100, func(c *Console) {
		c.remoteSession = s
		c.remoteConsoleReset = func() error { reset.Add(1); return nil }
		c.refreshRemoteSession()
	})
	key := uv.Key{Code: 'r', Mod: uv.ModCtrl | uv.ModAlt}
	for _, event := range []terminal.InputEvent{
		{Event: uv.PasteEvent{Content: "\x1b[114;7u"}, Raw: []byte("\x1b[114;7u")},
		{Event: uv.KeyReleaseEvent(key), Raw: []byte("\x1b[114;7:3u")},
		{Event: uv.KeyPressEvent(key), Reply: true},
		{Event: uv.KeyPressEvent{Code: 'r', Mod: uv.ModCtrl | uv.ModAlt, IsRepeat: true}},
	} {
		f.con.routeInputEvent(event)
	}
	if reset.Load() != 0 {
		t.Fatalf("non-host-press reset %d times", reset.Load())
	}
	// Actual host bytes pass through the decoder and the same host-only branch.
	_, _ = f.stdin.Write([]byte("\x1b[114;7u"))
	waitFor(t, "decoded host reset", func() bool { return reset.Load() == 1 })
	s.mu.Lock()
	s.status.CanReset = false
	s.mu.Unlock()
	f.con.refreshRemoteSession()
	f.con.routeInputEvent(terminal.InputEvent{Event: uv.KeyPressEvent(key)})
	if reset.Load() != 1 {
		t.Fatal("reset ran before teardown confirmed")
	}
}

func TestRemoteSessionClickTogglesAndShutdownJoins(t *testing.T) {
	s := &fixtureRemoteSession{status: remotesession.RemoteStatus{Status: "off"}}
	t.Run("owned console", func(t *testing.T) {
		f := newFixtureBeforeRun(t, 24, 100, func(c *Console) { c.remoteSession = s; c.refreshRemoteSession() })
		f.con.mu.Lock()
		x := f.con.statusRemoteSession.Start + 1
		f.con.mu.Unlock()
		_, _ = fmt.Fprintf(f.stdin, "\x1b[<0;%d;24M\x1b[<0;%d;24m", x, x)
		waitFor(t, "remote On from click", func() bool { s.mu.Lock(); defer s.mu.Unlock(); return s.on == 1 })
		s.mu.Lock()
		s.status.Status = "on"
		s.mu.Unlock()
		f.con.refreshRemoteSession()
		_, _ = fmt.Fprintf(f.stdin, "\x1b[<0;%d;24M\x1b[<0;%d;24m", x, x)
		waitFor(t, "remote Off from click", func() bool { s.mu.Lock(); defer s.mu.Unlock(); return s.off == 1 })
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.joined != 1 || s.off != 2 {
		t.Fatalf("shutdown did not off and join: off=%d joined=%d", s.off, s.joined)
	}
}

func TestRemoteSessionLinkIsOperatorOnly(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	s := &fixtureRemoteSession{status: remotesession.RemoteStatus{Status: "on"}, link: "https://remote.example.com/remote/THIS-CAPABILITY-IS-SECRET/"}
	f.con.mu.Lock()
	f.con.remoteSession = s
	f.con.mu.Unlock()
	f.con.refreshRemoteSession()
	v := openViewer(t, f.startLive(t))
	waitFor(t, "operator-only remote link", func() bool { return strings.Contains(f.screenText(), "THIS-CAPABILITY-IS-SECRET") })
	f.child.Feed([]byte("AFTER-REMOTE-LINK"))
	v.until("frame after remote link", func(s string) bool { return strings.Contains(s, "AFTER-REMOTE-LINK") })
	if strings.Contains(v.emu.String(), "THIS-CAPABILITY-IS-SECRET") {
		t.Fatal("remote capability leaked to broadcast")
	}
	if strings.Contains(f.lastRow(), "THIS-CAPABILITY-IS-SECRET") {
		t.Fatal("remote capability leaked to chrome")
	}
}

func TestRemoteSessionControlIsDistinctAndClips(t *testing.T) {
	for width := 1; width < 100; width++ {
		r := RenderStatusRow(width, StatusModel{Broadcast: BroadcastLive, RemoteStatus: "off", Webcast: WebcastLive})
		if r.RemoteSession.End > width {
			t.Fatalf("width %d remote span %+v", width, r.RemoteSession)
		}
		if r.RemoteSession.End > r.RemoteSession.Start && r.WebDriveLink.End > r.WebDriveLink.Start && r.RemoteSession.End > r.WebDriveLink.Start {
			t.Fatal("remote and Chrome drive spans overlap")
		}
	}
	r := RenderStatusRow(100, StatusModel{RemoteStatus: "off"})
	if !strings.Contains(r.Body, "👽 remote") || r.RemoteSession == (ColumnSpan{}) {
		t.Fatal("missing remote toggle")
	}
}
