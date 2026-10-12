package webcast

import (
	"context"
	"fmt"
	"math"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// spyInjector notes every batch that reaches it, by viewer, and passes it
// to the injector under it (the fake's rules: driver, segment, Confine,
// stale target).
type spyInjector struct {
	mu   sync.Mutex
	got  []spyBatch
	next drive.Injector
}

type spyBatch struct {
	by string
	b  drive.Batch
}

func (s *spyInjector) Inject(ctx context.Context, by string, b drive.Batch) error {
	s.mu.Lock()
	s.got = append(s.got, spyBatch{by, b})
	s.mu.Unlock()
	return s.next.Inject(ctx, by, b)
}

// acts are every act that reached the injector from by, in order.
func (s *spyInjector) acts(by string) []drive.Act {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []drive.Act
	for _, g := range s.got {
		if g.by == by {
			out = append(out, g.b.Acts...)
		}
	}
	return out
}

// serialSink makes one writer of several: the controller's ctl records and
// the injector's caused records arrive from different goroutines, as they
// enter the recorder's one queue in a cast.
type serialSink struct {
	mu   sync.Mutex
	sink recording.Sink
}

func (s *serialSink) Put(r recording.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sink.Put(r)
}
func (s *serialSink) Close() error { return nil }

// Two viewers drive the blog cast (#467) in two headless Chromes: the real
// viewer page, served by Server with a MemController over the fake
// injector (the rules production runs, over the fixture's page), its ctl
// records and the injector's into the stream the viewers watch. Viewer A
// (mouse, desktop width) asks, is granted, clicks a link and types; B
// (touch at an iPad's portrait size) asks, the grant moves to B, and B's tap
// is a click; a revoke takes B's layer away; A asks again and gives control
// back with the button.
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run DriveLive -v
func TestDriveLiveTwoViewers(t *testing.T) {
	a, b := headlessPage(t), headlessPage(t)
	a.call(t, a.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1280, "height": 800, "deviceScaleFactor": 1, "mobile": false}, nil)
	b.call(t, b.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 768, "height": 1024, "deviceScaleFactor": 2, "mobile": true}, nil)
	b.call(t, b.session, "Emulation.setTouchEmulationEnabled", map[string]any{"enabled": true, "maxTouchPoints": 5}, nil)

	rec, err := recording.ReadDir(recording.Fixture("rec-blog"))
	if err != nil {
		t.Fatal(err)
	}
	store, rec := throughStore(t, rec)
	const at = 8284 // as TestLiveViewerPage: the scroll is still, layout known
	hub, mem := recording.NewHub(4096), &recording.MemSink{}
	stream := &serialSink{sink: recording.Tee(mem, hub)}
	var page []recording.Record
	for _, r := range rec.Records {
		if r.Time() <= at {
			page = append(page, r)
			if err := stream.Put(r); err != nil {
				t.Fatal(err)
			}
		}
	}
	inj, err := drive.NewFakeInjector(page, stream)
	if err != nil {
		t.Fatal(err)
	}
	spy := &spyInjector{next: inj}
	ctrl := drive.NewMemController(&serialSink{sink: recording.Tee(stream, inj)}, spy, func() int64 { return inj.Page().T + 1 }, time.Minute)
	t.Cleanup(func() { ctrl.Close() })
	viewer := NewServer("tok", NewFakeBrowser(), time.Second, hub, nil).WithDrive(ctrl, "drv")
	viewer.Assets().Bind(store)
	srv := httptest.NewServer(viewer)
	t.Cleanup(func() { srv.CloseClientConnections(); srv.Close() })
	t.Cleanup(func() { hub.Close() })

	button := `(() => { const b = document.querySelector('.drive-button'); return b ? b.textContent : ''; })()`
	layerOn := `(() => document.querySelector('.drive.on') ? 'on' : 'off')()`
	seatOf := func(p *chromePage) string {
		return p.evalString(t, `couchViewer.drive ? couchViewer.drive.seat.viewer : ''`)
	}
	for _, p := range []*chromePage{a, b} {
		p.call(t, p.session, "Page.navigate", map[string]any{"url": srv.URL + "/drv/"}, nil)
		waitEval(t, p, "the drive button", button, "Ask to drive")
		p.eval(t, `(async () => { await couchViewer.ready(); await new Promise((ok) => requestAnimationFrame(() => requestAnimationFrame(ok))); return ''; })()`)
	}
	ana, bo := seatOf(a), seatOf(b)
	if ana == "" || bo == "" || ana == bo {
		t.Fatalf("seats %q %q", ana, bo)
	}
	st, err := recording.Reduce(mem.Records(), math.MaxInt64)
	if err != nil {
		t.Fatal(err)
	}
	target := hoverTarget(t, st) // a link in view
	tb := st.Boxes[target]
	pending := func(id string) func() bool {
		return func() bool {
			for _, p := range ctrl.Control().Pending {
				if p.ID == id {
					return true
				}
			}
			return false
		}
	}

	// A asks; the operator grants.
	clickButton(t, a, `.drive-button`)
	eventually(t, "A's request", pending(ana))
	waitEval(t, a, "A waiting", button, "Cancel request")
	if err := ctrl.Grant(ana); err != nil {
		t.Fatal(err)
	}
	waitEval(t, a, "A driving", button, "Give back control")
	waitEval(t, a, "A's capture layer", layerOn, "on")
	waitEval(t, b, "B sees who drives", `document.querySelector('.drive-note').textContent`, fmt.Sprintf("%s is driving.", ctrl.Control().Driver.Name))

	// A clicks the link and types.
	c := clientOf(t, a, tb[0]+tb[2]/2, tb[1]+tb[3]/2)
	mouse(t, a, "mousePressed", c)
	mouse(t, a, "mouseReleased", c)
	a.call(t, a.session, "Input.insertText", map[string]any{"text": "invoice"}, nil)
	a.call(t, a.session, "Input.dispatchKeyEvent", map[string]any{"type": "rawKeyDown", "key": "Enter", "code": "Enter", "windowsVirtualKeyCode": 13}, nil)
	a.call(t, a.session, "Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": "Enter", "code": "Enter", "windowsVirtualKeyCode": 13}, nil)
	eventually(t, "A's acts reach the injector", func() bool { return len(spy.acts(ana)) >= 4 })
	acts := spy.acts(ana)
	click, text, enter := acts[0], acts[1], acts[2]
	if click.A != drive.ActClick || click.ID != target || math.Abs(click.X-(tb[0]+tb[2]/2)) > 2 || math.Abs(click.Y-(tb[1]+tb[3]/2)) > 2 {
		t.Errorf("A's click: %+v, want on %d at the box's middle", click, target)
	}
	if text.A != drive.ActText || text.V != "invoice" || enter.A != drive.ActKey || enter.Key != "Enter" || enter.Type != drive.KeyDown {
		t.Errorf("A's typing: %+v %+v", text, enter)
	}
	clicked := false
	for _, r := range mem.Records() {
		if c, ok := r.(*recording.Click); ok && c.By == ana && c.ID == target {
			clicked = true
		}
	}
	if !clicked {
		t.Error("A's click isn't in the stream, attributed: the injector's rules refused it")
	}
	if b.evalString(t, layerOn) != "off" {
		t.Error("B's layer takes input while A drives")
	}

	// B asks; the grant moves to B; B taps.
	clickButton(t, b, `.drive-button`)
	eventually(t, "B's request", pending(bo))
	if err := ctrl.Grant(bo); err != nil {
		t.Fatal(err)
	}
	waitEval(t, b, "B driving", layerOn, "on")
	waitEval(t, a, "A no longer driving", layerOn, "off")
	tc := clientOf(t, b, tb[0]+tb[2]/2, tb[1]+tb[3]/2)
	touch(t, b, "touchStart", tc)
	touch(t, b, "touchEnd", tc)
	eventually(t, "B's tap", func() bool { return len(spy.acts(bo)) > 0 })
	if tap := spy.acts(bo)[0]; tap.A != drive.ActClick || tap.ID != target {
		t.Errorf("B's tap: %+v", tap)
	}

	// The operator revokes: B's layer goes.
	if err := ctrl.Revoke(); err != nil {
		t.Fatal(err)
	}
	waitEval(t, b, "B revoked", layerOn, "off")
	waitEval(t, b, "B's button", button, "Ask to drive")

	// A asks again, is granted, and gives control back.
	clickButton(t, a, `.drive-button`)
	eventually(t, "A's second request", pending(ana))
	if err := ctrl.Grant(ana); err != nil {
		t.Fatal(err)
	}
	waitEval(t, a, "A driving again", button, "Give back control")
	clickButton(t, a, `.drive-button`)
	eventually(t, "A gave control back", func() bool { return ctrl.Control().Driver == nil })
	waitEval(t, a, "A's layer off", layerOn, "off")

	// The view link watches, points and notes: no seat, no drive button.
	a.call(t, a.session, "Page.navigate", map[string]any{"url": srv.URL + "/tok/"}, nil)
	a.eval(t, `(async () => { for (let i = 0; i < 100 && !window.couchViewer; i++) await new Promise((ok) => setTimeout(ok, 50)); await couchViewer.ready(); return ''; })()`)
	if got := a.evalString(t, `document.querySelector('.drive-button') ? 'button' : 'none'`); got != "none" {
		t.Error("the view link offers driving")
	}

	// The stream reduces: every ctl states what its op gives.
	recs := mem.Records()
	if _, err := recording.Reduce(recs, recs[len(recs)-1].Time()); err != nil {
		t.Fatalf("the driven stream doesn't reduce: %v", err)
	}
}
