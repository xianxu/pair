package webcast

import (
	"context"
	"encoding/json"
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
	t.Helper()
	session := h.session(t, "")
	call := func(method string, params, result any) error { return h.c.call(session, method, params, result) }
	deadline := time.Now().Add(10 * time.Second)
	for {
		ready, err := clickLiveControl(call)
		if ready {
			if err != nil {
				t.Fatal(err)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("control not found: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// clickLiveControl attempts once. Ready distinguishes input errors (never
// retried) from a still-unavailable control, before any mouse input.
func clickLiveControl(call func(string, any, any) error) (ready bool, err error) {
	// The host contains fixed-position panels and no longer has the button's
	// geometry. CDP can inspect its closed shadow root without weakening it.
	var point [2]float64
	locate := func() error {
		var document struct {
			Root struct {
				NodeID int `json:"nodeId"`
			} `json:"root"`
		}
		if err := call("DOM.getDocument", map[string]any{"depth": 0}, &document); err != nil {
			return err
		}
		query := func(parent int, selector string) (int, error) {
			var result struct {
				NodeID int `json:"nodeId"`
			}
			err := call("DOM.querySelector", map[string]any{"nodeId": parent, "selector": selector}, &result)
			if err == nil && result.NodeID == 0 {
				err = errors.New("control node is not mounted")
			}
			return result.NodeID, err
		}
		host, err := query(document.Root.NodeID, "couchcast-control[data-couchcast-ui]")
		if err != nil {
			return err
		}
		var description struct {
			Node struct {
				ShadowRoots []struct {
					NodeID int `json:"nodeId"`
				} `json:"shadowRoots"`
			} `json:"node"`
		}
		if err := call("DOM.describeNode", map[string]any{"nodeId": host, "depth": 1, "pierce": true}, &description); err != nil {
			return err
		}
		if len(description.Node.ShadowRoots) != 1 {
			return errors.New("control shadow root is not mounted")
		}
		button, err := query(description.Node.ShadowRoots[0].NodeID, ".cast-button")
		if err != nil {
			return err
		}
		var object struct {
			Object struct {
				ID string `json:"objectId"`
			} `json:"object"`
		}
		if err := call("DOM.resolveNode", map[string]any{"nodeId": button}, &object); err != nil {
			return err
		}
		if object.Object.ID == "" {
			return errors.New("control object is unavailable")
		}
		var inspection struct {
			Result struct {
				Value struct {
					X, Y   float64
					Usable bool
				} `json:"value"`
			} `json:"result"`
			Exception json.RawMessage `json:"exceptionDetails"`
		}
		err = call("Runtime.callFunctionOn", map[string]any{
			"objectId": object.Object.ID, "returnByValue": true,
			"functionDeclaration": `function(){
 const r=this.getBoundingClientRect(),root=this.getRootNode(),x=r.x+r.width/2,y=r.y+r.height/2;
 let visible=this.isConnected&&!this.disabled;
 for(let n=this;n;n=n.parentElement||n.getRootNode().host){
  const s=getComputedStyle(n);
  if(s.display==='none'||s.visibility!=='visible'||Number(s.opacity)===0)visible=false;
 }
 return {x,y,usable:visible&&r.width>0&&r.height>0&&r.left>=0&&r.top>=0&&r.right<=innerWidth&&r.bottom<=innerHeight&&this.contains(root.elementFromPoint(x,y))&&document.elementFromPoint(x,y)===root.host};
}`}, &inspection)
		err = errors.Join(err, call("Runtime.releaseObject", map[string]any{"objectId": object.Object.ID}, nil))
		if err != nil {
			return err
		}
		if len(inspection.Exception) > 0 || !inspection.Result.Value.Usable {
			return errors.New("control is not visible and clickable")
		}
		point = [2]float64{inspection.Result.Value.X, inspection.Result.Value.Y}
		return nil
	}
	if err := locate(); err != nil {
		return false, err
	}
	for _, typ := range []string{"mousePressed", "mouseReleased"} {
		if err := call("Input.dispatchMouseEvent", map[string]any{
			"type": typ, "x": point[0], "y": point[1], "button": "left", "clickCount": 1,
		}, nil); err != nil {
			return true, err
		}
	}
	return true, nil
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
