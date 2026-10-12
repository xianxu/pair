package webcast

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

func startCast(t *testing.T, tunnel broadcast.Tunnel) (*Cast, *FakeBrowser) {
	t.Helper()
	f := NewFakeBrowser()
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: tunnel, ProbeTimeout: 2 * time.Second})
	c, err := StartCast(context.Background(), CastConfig{
		Launch: func(context.Context) (Browser, error) { return f, nil },
		Host:   host,
		Ping:   time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.End)
	return c, f
}

func waitUpdate(t *testing.T, c *Cast, what string, ok func(Update) bool) Update {
	t.Helper()
	updates, cancel := c.Subscribe()
	defer cancel()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case u, open := <-updates:
			if ok(u) {
				return u
			}
			if !open {
				t.Fatalf("%s: cast ended at %+v", what, u)
			}
		case <-deadline:
			t.Fatalf("%s: timed out at %+v", what, c.Current())
		}
	}
}

func TestCastBrowserFirstThenLink(t *testing.T) {
	tunnel := &broadcast.FakeTunnel{OpenDelay: 200 * time.Millisecond}
	c, _ := startCast(t, tunnel)
	if u := c.Current(); u.Link != "" || u.State.Phase != PhaseIdle {
		t.Fatalf("at start: %+v", u)
	}
	u := waitUpdate(t, c, "link", func(u Update) bool { return u.Link != "" })
	resp, err := http.Get(u.Link)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("viewer page: %v %v", resp, err)
	}
	resp.Body.Close()
}

func TestCastToggleAndViewerSees(t *testing.T) {
	c, f := startCast(t, &broadcast.FakeTunnel{})
	u := waitUpdate(t, c, "link", func(u Update) bool { return u.Link != "" })
	s := openSSE(t, u.Link+"events")
	s.nextState()
	f.Navigate("http://localhost:3000/", "Shop")
	if err := c.ToggleLive(); err != nil {
		t.Fatal(err)
	}
	waitUpdate(t, c, "live", func(u Update) bool { return u.State.Phase == PhaseLive })
	if got := s.nextState(); !got.Live || got.Title != "Shop" {
		t.Fatalf("viewer: %+v", got)
	}
	c.ToggleLive()
	waitUpdate(t, c, "paused", func(u Update) bool { return u.State.Phase == PhaseIdle })
}

func TestCastEndsWithTheBrowser(t *testing.T) {
	tunnel := &broadcast.FakeTunnel{}
	c, f := startCast(t, tunnel)
	u := waitUpdate(t, c, "link", func(u Update) bool { return u.Link != "" })
	s := openSSE(t, u.Link+"events")
	s.nextState()
	f.QuitByHand()
	if ev, _ := s.next(); ev != "end" {
		t.Fatalf("viewer got %s, want end", ev)
	}
	select {
	case <-c.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("cast not done")
	}
	if !errors.Is(c.Err(), ErrClosedByOperator) {
		t.Fatalf("err %v", c.Err())
	}
	if st := tunnel.Stats(); st.Opens != st.Closes {
		t.Fatalf("tunnel left open: %+v", st)
	}
}

func TestCastEnd(t *testing.T) {
	tunnel := &broadcast.FakeTunnel{}
	c, f := startCast(t, tunnel)
	waitUpdate(t, c, "link", func(u Update) bool { return u.Link != "" })
	c.End()
	if c.Err() != nil || f.Stats().Closes == 0 {
		t.Fatalf("err %v stats %+v", c.Err(), f.Stats())
	}
	if st := tunnel.Stats(); st.Opens != st.Closes {
		t.Fatalf("tunnel left open: %+v", st)
	}
}

// Ending while the link is still opening abandons it and closes whatever
// it opened.
func TestCastEndWhileLinkOpens(t *testing.T) {
	tunnel := &broadcast.FakeTunnel{OpenDelay: 300 * time.Millisecond, IgnoreCancel: true}
	c, _ := startCast(t, tunnel)
	c.End()
	if st := tunnel.Stats(); st.Opens != st.Closes {
		t.Fatalf("tunnel left open: %+v", st)
	}
}

type refusingTunnel struct{ broadcast.LocalOnly }

var errRefused = errors.New("no tunnel today")

func (refusingTunnel) Open(context.Context, net.Listener) (broadcast.Handle, error) {
	return nil, errRefused
}

func TestCastLinkFailureLeavesBrowserUp(t *testing.T) {
	c, f := startCast(t, refusingTunnel{})
	u := waitUpdate(t, c, "link error", func(u Update) bool { return u.LinkErr != nil })
	if !errors.Is(u.LinkErr, errRefused) || u.Link != "" {
		t.Fatalf("update %+v", u)
	}
	if err := c.ToggleLive(); err != nil || f.State().Phase != PhaseLive {
		t.Fatalf("browser not usable: %v %+v", err, f.State())
	}
}

func TestCastTunnelLostMidCast(t *testing.T) {
	tunnel := &broadcast.FakeTunnel{}
	c, f := startCast(t, tunnel)
	waitUpdate(t, c, "link", func(u Update) bool { return u.Link != "" })
	tunnel.Exit()
	u := waitUpdate(t, c, "link lost", func(u Update) bool { return u.LinkErr != nil })
	if u.Link != "" || !strings.Contains(u.LinkErr.Error(), "tunnel") {
		t.Fatalf("update %+v", u)
	}
	if f.Err() != nil {
		t.Fatal("browser ended with the tunnel")
	}
}
