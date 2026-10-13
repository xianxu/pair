package webcast

import (
	"context"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

// The live checks drive a real Chrome: they are the conformance oracle for
// FakeBrowser and the only test of Chrome's process and protocol behaviour.
// Run them at each milestone close that touches webcast or cdp, and when
// Chrome's major version changes. Chrome runs headless so a test never takes
// focus from the operator's terminal; visibility under real windows is the
// smoke test's to confirm. They can't run inside the agent sandbox (Chrome
// exits at once):
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run Live -v
func liveChrome(t *testing.T) *Chrome {
	t.Helper()
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1 to drive a real Chrome")
	}
	bin, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := Launch(ctx, LaunchOptions{Binary: bin, ProfileDir: t.TempDir(), Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

type liveHarness struct{ c *Chrome }

func (h liveHarness) browser() Browser { return h.c }

// session is the attached session of target id (the active tab when "").
func (h liveHarness) session(t *testing.T, id string) string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var session string
		ran := make(chan struct{})
		h.c.do(func(s *loopState) {
			defer close(ran)
			if id == "" {
				tab, _ := s.tracker.Active()
				id = tab.ID
			}
			for sess, target := range s.sessions {
				if target == id {
					session = sess
				}
			}
		})
		<-ran
		if session != "" {
			return session
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("no session for target %q", id)
	return ""
}

func (h liveHarness) navigate(t *testing.T, title string) {
	if err := h.c.call(h.session(t, ""), "Page.navigate", map[string]any{"url": page(title)}, nil); err != nil {
		t.Fatal(err)
	}
}

func (h liveHarness) newTab(t *testing.T, title string) string {
	var r struct {
		TargetID string `json:"targetId"`
	}
	if err := h.c.call("", "Target.createTarget", map[string]any{"url": page(title)}, &r); err != nil {
		t.Fatal(err)
	}
	return r.TargetID
}

// activate stands in for the operator switching tabs. Headless Chrome has no
// window to switch in, and a test must never raise one, so the page is given
// focus by emulation and told it gained it; this checks the report → tracker
// path, and the smoke test checks real tab switches.
func (h liveHarness) activate(t *testing.T, id string) {
	session := h.session(t, id)
	if err := h.c.call(session, "Emulation.setFocusEmulationEnabled", map[string]any{"enabled": true}, nil); err != nil {
		t.Fatal(err)
	}
	if err := h.c.call(session, "Runtime.evaluate", map[string]any{"expression": "window.dispatchEvent(new Event('focus'))"}, nil); err != nil {
		t.Fatal(err)
	}
}

// clickControl clicks the in-page control with real mouse input at its
// position: its shadow root is closed, so nothing outside can reach it.
func (h liveHarness) clickControl(t *testing.T) {
	session := h.session(t, "")
	var r struct {
		Result struct {
			Value struct{ X, Y float64 } `json:"value"`
		} `json:"result"`
	}
	expr := `(() => { const r = document.querySelector('[data-couchcast-ui]').getBoundingClientRect();
		return { x: r.left + r.width / 2, y: r.top + r.height / 2 }; })()`
	deadline := time.Now().Add(10 * time.Second)
	for {
		err := h.c.call(session, "Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true}, &r)
		if err == nil && r.Result.Value.X > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("control not found: %v %+v", err, r)
		}
		time.Sleep(100 * time.Millisecond)
	}
	for _, typ := range []string{"mousePressed", "mouseReleased"} {
		if err := h.c.call(session, "Input.dispatchMouseEvent", map[string]any{
			"type": typ, "x": r.Result.Value.X, "y": r.Result.Value.Y, "button": "left", "clickCount": 1,
		}, nil); err != nil {
			t.Fatal(err)
		}
	}
}

// quitByHand closes Chrome without going through Close, as Cmd+Q would.
func (h liveHarness) quitByHand(t *testing.T) {
	if err := h.c.call("", "Browser.close", nil, nil); err != nil && h.c.Err() == nil {
		select {
		case <-h.c.Done():
		case <-time.After(time.Second):
			t.Fatal(err)
		}
	}
}

func TestLiveConformance(t *testing.T) {
	runConformance(t, liveHarness{liveChrome(t)})
}

func TestLiveClose(t *testing.T) {
	c := liveChrome(t)
	c.Close()
	if c.Err() != nil || c.State().Phase != PhaseEnded {
		t.Fatalf("err %v state %+v", c.Err(), c.State())
	}
	if !c.proc.(*osProcess).cmd.ProcessState.Exited() {
		t.Fatalf("process state %v", c.proc.(*osProcess).cmd.ProcessState)
	}
}

// TestLiveChromeExitsWhenPipeCloses is the property a crashed Couch relies
// on: when Couch's end of the pipe closes, Chrome exits by itself.
func TestLiveChromeExitsWhenPipeCloses(t *testing.T) {
	c := liveChrome(t)
	c.conn.Close()
	select {
	case <-c.proc.Exited():
	case <-time.After(closeWait - time.Second):
		t.Fatal("Chrome still running after its pipe closed")
	}
	if ws, ok := c.proc.(*osProcess).cmd.ProcessState.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		t.Fatalf("Chrome was killed (%v), not exited", ws.Signal())
	}
}

func TestLiveLastWindowClosed(t *testing.T) {
	c := liveChrome(t)

	tab := c.State().Tab.ID
	if err := c.call("", "Target.closeTarget", map[string]any{"targetId": tab}, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case <-c.Done():
	case <-time.After(noPagesGrace + closeWait + 5*time.Second):
		t.Fatal("Chrome with no pages did not end")
	}
	if !errors.Is(c.Err(), ErrClosedByOperator) {
		t.Fatalf("err %v", c.Err())
	}
}

func TestLiveProfileInUse(t *testing.T) {
	c := liveChrome(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	second, err := Launch(ctx, LaunchOptions{Binary: c.opts.Binary, ProfileDir: c.opts.ProfileDir, Headless: true})
	if err == nil {
		second.Close()
		t.Fatal("second Chrome on the same profile launched")
	}
	if !errors.Is(err, ErrProfileInUse) {
		t.Fatalf("err %v, want ErrProfileInUse", err)
	}
	if c.Err() != nil {
		t.Fatalf("first Chrome ended: %v", c.Err())
	}
}

// TestLiveCastEndToEnd: a real Chrome cast through the shared Host, read as
// a viewer reads it.
func TestLiveCastEndToEnd(t *testing.T) {
	c := liveChrome(t)
	h := liveHarness{c}
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: broadcast.LocalOnly{}, ProbeTimeout: 5 * time.Second})
	cast, err := StartCast(context.Background(), CastConfig{
		Launch: func(context.Context, *RecordOptions) (Browser, error) { return c, nil },
		Host:   host,
		Ping:   time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cast.End)
	u := waitUpdate(t, cast, "link", func(u Update) bool { return u.Link != "" })
	unlockCast(t, cast, u)
	v := openSSE(t, u.Link+"events")
	if got := v.nextState(); got != (wireState{}) {
		t.Fatalf("join: %+v", got)
	}
	h.navigate(t, "secret-before-live")
	waitState(t, c, "navigated", func(s State) bool { return s.Tab.Title == "secret-before-live" })
	if err := cast.ToggleLive(); err != nil {
		t.Fatal(err)
	}
	if got := v.nextState(); !got.Live || got.Title != "secret-before-live" {
		t.Fatalf("live: %+v", got)
	}
	// While the new page loads, Chrome's own placeholder (its URL) may show
	// as the title, as in Chrome's tab; the old page's title must never be
	// paired with the new URL, and the stream settles on the new title.
	h.navigate(t, "shop")
	for {
		got := v.nextState()
		if strings.Contains(got.URL, "shop") && got.Title == "secret-before-live" {
			t.Fatalf("old title with the new URL: %+v", got)
		}
		if got.Title == "shop" && strings.Contains(got.URL, "shop") {
			break
		}
	}
	h.clickControl(t)
	if got := v.nextState(); got != (wireState{}) {
		t.Fatalf("paused from the page: %+v", got)
	}
	cast.End()
	if ev, _ := v.next(); ev != "end" {
		t.Fatalf("viewer got %s, want end", ev)
	}
}
