package terminal

import (
	"context"
	"errors"
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	vt "github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"strings"
	"testing"
	"time"
)

func presenterFixture(t *testing.T, policy ParentMousePolicy) (*Presenter, *ttyio.Fake, *Endpoint, *ttyio.Fake) {
	t.Helper()
	parent := ttyio.NewFake()
	p := NewPresenter(parent, policy)
	e, input := newEndpointTest(t, "selected")
	t.Cleanup(func() { p.Release(context.Background()) })
	return p, parent, e, input
}
func selectPresenter(t *testing.T, p *Presenter, e *Endpoint) {
	t.Helper()
	if err := p.Select(context.Background(), e, Geometry{8, 5}, make([]Cell, 8)); err != nil {
		t.Fatal(err)
	}
}
func TestPresenterUnwrittenReleaseDoesNotTouchParent(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, CouchAnyMotion)
	if err := p.Register(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if err := p.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if parent.Calls() != 0 {
		t.Fatalf("unacquired parent changed by disposal: %q", parent.Bytes())
	}
}
func TestPresenterAdmissionWaitsForCompletePaint(t *testing.T) {
	p, parent, e, input := presenterFixture(t, CouchAnyMotion)
	block := make(chan struct{})
	parent.Enqueue(ttyio.WriteStep{Block: block})
	selected := make(chan error, 1)
	go func() { selected <- p.Select(context.Background(), e, Geometry{8, 5}, make([]Cell, 8)) }()
	<-parent.Started()
	if p.View().Admitted != "" {
		t.Fatal("admitted before write completed")
	}
	sent := make(chan error, 1)
	go func() { sent <- p.Input(context.Background(), uv.KeyPressEvent{Code: 'x', Text: "x"}) }()
	select {
	case err := <-sent:
		t.Fatalf("input did not pause behind paint:%v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(block)
	if err := <-selected; err != nil {
		t.Fatal(err)
	}
	if err := <-sent; err != nil {
		t.Fatal(err)
	}
	if err := e.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if string(input.Bytes()) != "x" || p.View().Admitted != "selected" {
		t.Fatalf("admission %+v input%q", p.View(), input.Bytes())
	}
}
func TestPresenterPartialFailureClosesAdmission(t *testing.T) {
	p, parent, e, input := presenterFixture(t, ChildRequested)
	boom := errors.New("broken terminal")
	parent.Enqueue(ttyio.WriteStep{Limit: 5, Err: boom})
	if err := p.Select(context.Background(), e, Geometry{8, 5}, make([]Cell, 8)); !errors.Is(err, boom) {
		t.Fatalf("paint error%v", err)
	}
	if p.View().State != Failed || p.View().Admitted != "" {
		t.Fatalf("failedview %+v", p.View())
	}
	if err := p.Input(context.Background(), uv.KeyPressEvent{Code: 'x'}); err == nil {
		t.Fatal("input admitted after partial write")
	}
	if len(input.Bytes()) != 0 {
		t.Fatal("failedview wrote child")
	}
}
func TestPresenterReleaseCancelsBlockedWriteAndJoins(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, CouchAnyMotion)
	parent.Enqueue(ttyio.WriteStep{Block: make(chan struct{})})
	done := make(chan error, 1)
	go func() { done <- p.Select(context.Background(), e, Geometry{8, 5}, make([]Cell, 8)) }()
	<-parent.Started()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := p.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if p.View().State != Released {
		t.Fatalf("release did not finish %+v", p.View())
	}
	if err := <-done; err == nil {
		t.Fatal("canceled paint succeeded")
	}
	calls := parent.Calls()
	if err := p.Present(context.Background(), e); err == nil {
		t.Fatal("released presenter accepted refresh")
	}
	if parent.Calls() != calls {
		t.Fatal("write after release")
	}
}
func TestPresenterMousePolicyAndDragCancellation(t *testing.T) {
	for _, policy := range []ParentMousePolicy{CouchAnyMotion, ChildRequested} {
		t.Run(string(rune('0'+policy)), func(t *testing.T) {
			p, parent, a, aw := presenterFixture(t, policy)
			a.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
			selectPresenter(t, p, a)
			if policy == CouchAnyMotion && !strings.Contains(string(parent.Bytes()), "\x1b[?1003h") {
				t.Fatal("Couch parent lacks allmotion")
			}
			if err := p.Input(context.Background(), uv.MouseClickEvent{X: 2, Y: 1, Button: uv.MouseLeft}); err != nil {
				t.Fatal(err)
			}
			b, bw := newEndpointTest(t, "next")
			b.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
			selectPresenter(t, p, b)
			if got := string(aw.Bytes()); got != "\x1b[<0;3;2M\x1b[<0;3;2m" {
				t.Fatalf("old dragrelease=%q", got)
			}
			p.Input(context.Background(), uv.MouseMotionEvent{X: 3, Y: 2, Button: uv.MouseLeft})
			p.Input(context.Background(), uv.MouseReleaseEvent{X: 3, Y: 2, Button: uv.MouseLeft})
			b.Flush(context.Background())
			if len(bw.Bytes()) != 0 {
				t.Fatalf("remainder crossed selection:%q", bw.Bytes())
			}
			p.Input(context.Background(), uv.MouseClickEvent{X: 2, Y: 4, Button: uv.MouseLeft})
			b.Flush(context.Background())
			if len(bw.Bytes()) != 0 {
				t.Fatal("chrome click escaped to child")
			}
		})
	}
}
func TestPresenterEffectsOnceAndNeverReplayed(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, CouchAnyMotion)
	selectPresenter(t, p, e)
	hidden, _ := newEndpointTest(t, "hidden")
	if err := p.Register(context.Background(), hidden); err != nil {
		t.Fatal(err)
	}
	effects := []Effect{{Kind: BellEffect, EndpointID: "hidden", Sequence: 1}, {Kind: TitleEffect, EndpointID: "hidden", Sequence: 2, Text: "safe"}}
	policy := EffectPolicy{Bell: true, Title: true}
	if _, err := p.EmitEffects(context.Background(), effects, policy); err != nil {
		t.Fatal(err)
	}
	once := string(parent.Bytes())
	if _, err := p.EmitEffects(context.Background(), effects, policy); err != nil {
		t.Fatal(err)
	}
	if string(parent.Bytes()) != once {
		t.Fatal("effects replayed")
	}
	if strings.Count(once, "\a") != 1 || strings.Count(once, "\x1b]2;safe\x1b\\") != 1 {
		t.Fatal("typed effects missing")
	}
}

func TestPresenterRefreshCoalescesAndRecoversSynchronizationWithoutMoreOutput(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, ChildRequested)
	selectPresenter(t, p, e)
	baseline := parent.Calls()
	e.Feed([]byte("\x1b[?2026hheld"), time.Now())
	for i := 0; i < 1000; i++ {
		if err := p.Present(context.Background(), e); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(time.Second)
	for !strings.Contains(string(parent.Bytes()), "held") && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !strings.Contains(string(parent.Bytes()), "held") {
		t.Fatal("synchronized output never recovered without new output")
	}
	calls := parent.Calls()
	if calls-baseline > 3 {
		t.Fatalf("refresh failed to coalesce:%d writes", calls-baseline)
	}
	time.Sleep(40 * time.Millisecond)
	if parent.Calls() != calls {
		t.Fatal("idle repaint timer continued")
	}
}
func TestPresenterResizeAndChildMouseOff(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, ChildRequested)
	selectPresenter(t, p, e)
	if strings.Contains(string(parent.Bytes()), "\x1b[?1003h") || strings.Contains(string(parent.Bytes()), "\x1b[?1000h") {
		t.Fatal("mouse enabled for child that did not request it")
	}
	before := p.View()
	var got Geometry
	if err := p.Resize(context.Background(), Geometry{10, 6}, func(g Geometry) error { got = g; return nil }); err != nil {
		t.Fatal(err)
	}
	after := p.View()
	if got != (Geometry{10, 5}) || after.GeometryEpoch <= before.GeometryEpoch || after.Admitted != "selected" {
		t.Fatalf("resize got%+v view%+v", got, after)
	}
}
func TestPresenterBoundsAdmissionWhilePaintBlocked(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, ChildRequested)
	block := make(chan struct{})
	parent.Enqueue(ttyio.WriteStep{Block: block})
	selected := make(chan error, 1)
	go func() { selected <- p.Select(context.Background(), e, Geometry{8, 5}, make([]Cell, 8)) }()
	<-parent.Started()
	results := make(chan error, MaxPendingEvents)
	for i := 0; i < MaxPendingEvents; i++ {
		go func() { results <- p.Input(context.Background(), uv.KeyPressEvent{Code: 'x', Text: "x"}) }()
	}
	deadline := time.Now().Add(time.Second)
	for len(p.requests) != MaxPendingEvents && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(p.requests) != MaxPendingEvents {
		t.Fatal("queue did not fill")
	}
	if err := p.Input(context.Background(), uv.KeyPressEvent{Code: 'x'}); !errors.Is(err, ErrBackpressure) {
		t.Fatalf("overflow=%v", err)
	}
	close(block)
	if err := <-selected; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxPendingEvents; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
}
func TestPresenterRejectsControlInjectionAndStaleFrames(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, ChildRequested)
	selectPresenter(t, p, e)
	before := parent.Calls()
	if _, err := p.EmitEffects(context.Background(), []Effect{{Kind: TitleEffect, EndpointID: "x", Sequence: 1, Text: "unsafe\x1b[2J"}}, EffectPolicy{Title: true}); err == nil {
		t.Fatal("control injection accepted")
	}
	if parent.Calls() != before {
		t.Fatal("invalid effect reached writer")
	}
	frame, err := e.Snapshot(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	frame, err = Compose(frame, Geometry{8, 5}, make([]Cell, 8))
	if err != nil {
		t.Fatal(err)
	}
	frame.GeometryEpoch++
	if err := p.call(context.Background(), func(ctx context.Context) error { return p.paint(ctx, frame, false) }); err == nil {
		t.Fatal("unadmitted epoch accepted")
	}
	if parent.Calls() != before {
		t.Fatal("stale frame reached writer")
	}
}

func TestPresenterPaintsDoNotResetMouseDuringDrag(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, CouchAnyMotion)
	e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
	selectPresenter(t, p, e)
	if err := p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft}); err != nil {
		t.Fatal(err)
	}
	before := len(parent.Bytes())
	for i := 0; i < 4; i++ {
		e.Feed([]byte("x"), time.Now())
		if err := p.call(context.Background(), func(ctx context.Context) error { return p.paintEndpoint(ctx, e, false) }); err != nil {
			t.Fatal(err)
		}
	}
	if wire := string(parent.Bytes()[before:]); strings.Contains(wire, "1003") || strings.Contains(wire, "1006") {
		t.Fatalf("mouse mode churn during drag:%q", wire)
	}
}
func TestPresenterReleaseInPanelClearsSuppression(t *testing.T) {
	p, _, e, _ := presenterFixture(t, CouchAnyMotion)
	e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
	selectPresenter(t, p, e)
	p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft})
	f, err := PanelFrame(Geometry{8, 5}, make([]Cell, 40), Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Panel(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if err := p.Input(context.Background(), uv.MouseReleaseEvent{X: 1, Y: 1, Button: uv.MouseLeft}); err != nil {
		t.Fatal(err)
	}
	selectPresenter(t, p, e)
	if p.View().Gesture == GestureParent {
		t.Fatal("physical release in panel left suppression stuck")
	}
}

func TestPresenterRetiresOriginsAndRefusesLateEffects(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, ChildRequested)
	selectPresenter(t, p, e)
	if err := p.Retire(context.Background(), e); err == nil {
		t.Fatal("retired selected endpoint")
	}
	hidden, _ := newEndpointTest(t, "hidden")
	for i := 0; i < MaxPendingEvents+1; i++ {
		if err := p.Register(context.Background(), hidden); err != nil {
			t.Fatal(err)
		}
		if err := p.Retire(context.Background(), hidden); err != nil {
			t.Fatal(err)
		}
	}
	before := parent.Calls()
	if _, err := p.EmitEffects(context.Background(), []Effect{{Kind: BellEffect, EndpointID: "hidden", Sequence: 1}}, EffectPolicy{Bell: true}); err == nil {
		t.Fatal("retired origin effect accepted")
	}
	if parent.Calls() != before {
		t.Fatal("late effect reached parent")
	}
}
func TestPresenterFailureCancelsDragToOldEndpoint(t *testing.T) {
	p, parent, e, input := presenterFixture(t, CouchAnyMotion)
	e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
	selectPresenter(t, p, e)
	if err := p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft}); err != nil {
		t.Fatal(err)
	}
	parent.Enqueue(ttyio.WriteStep{Limit: 2, Err: errors.New("parent failed")})
	e.Feed([]byte("x"), time.Now())
	if err := p.call(context.Background(), func(ctx context.Context) error { return p.paintEndpoint(ctx, e, false) }); err == nil {
		t.Fatal("write failure missing")
	}
	if got := string(input.Bytes()); got != "\x1b[<0;2;2M\x1b[<0;2;2m" {
		t.Fatalf("failed paint left drag captured:%q", got)
	}
}

func TestPresenterRequestsKeyEventsAndEndpointFiltersReleases(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			p, parent, e, input := presenterFixture(t, ChildRequested)
			if enabled {
				e.Feed([]byte("\x1b[>3u"), time.Now())
			}
			selectPresenter(t, p, e)
			if !strings.Contains(string(parent.Bytes()), "\x1b[>3u") {
				t.Fatal("parent did not request keyboard event types")
			}
			if err := p.Input(context.Background(), uv.KeyReleaseEvent{Code: uv.KeyEnter, Mod: uv.ModCtrl}); err != nil {
				t.Fatal(err)
			}
			if err := e.Flush(context.Background()); err != nil {
				t.Fatal(err)
			}
			want := ""
			if enabled {
				want = "\x1b[13;5:3u"
			}
			if got := string(input.Bytes()); got != want {
				t.Fatalf("release got%q want%q", got, want)
			}
		})
	}
}

func TestPresenterReleaseRestoresOnlyOwnedKeyboardPush(t *testing.T) {
	setup := parentModeDelta(parentModes{}, parentModes{}, false)
	pushEnd := strings.Index(setup, "\x1b[>3u") + len("\x1b[>3u")
	for _, tc := range []struct {
		name       string
		limit      int
		selectView bool
	}{{"unacquired", 0, false}, {"zero", 0, true}, {"before-push", pushEnd - 1, true}, {"after-push", pushEnd + 2, true}} {
		t.Run(tc.name, func(t *testing.T) {
			p, parent, e, _ := presenterFixture(t, ChildRequested)
			if tc.selectView {
				parent.Enqueue(ttyio.WriteStep{Limit: tc.limit, ZeroProgress: tc.limit == 0, Err: errors.New("interrupted")})
				if err := p.Select(context.Background(), e, Geometry{8, 5}, make([]Cell, 8)); err == nil {
					t.Fatal("expected interruption")
				}
			}
			if err := p.Release(context.Background()); err != nil {
				t.Fatal(err)
			}
			host := vt.NewEmulator(8, 5)
			defer host.Close()
			host.WriteString("\x1b[>7u")
			host.Write(parent.Bytes())
			if got := host.KeyboardFlags(); got != 7 {
				t.Fatalf("ambient flags=%d afterwire%q", got, parent.Bytes())
			}
		})
	}
}

func TestPresenterReleaseCancelsEffectAndJoinsResult(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, ChildRequested)
	if err := p.Register(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	parent.Enqueue(ttyio.WriteStep{Block: make(chan struct{})})
	done := make(chan error, 1)
	go func() {
		_, err := p.EmitEffects(context.Background(), []Effect{{Kind: BellEffect, EndpointID: "selected", Sequence: 1}}, EffectPolicy{Bell: true})
		done <- err
	}()
	<-parent.Started()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := p.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err == nil {
		t.Fatal("canceled effect succeeded")
	}
	calls := parent.Calls()
	time.Sleep(time.Millisecond)
	if parent.Calls() != calls || p.View().State != Released {
		t.Fatal("effect outlived release")
	}
}
func TestPresenterFailedResizePreservesPresentedGeometry(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, ChildRequested)
	selectPresenter(t, p, e)
	before := p.View()
	calls := parent.Calls()
	boom := errors.New("ioctl failed")
	if err := p.Resize(context.Background(), Geometry{12, 7}, func(Geometry) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("resize=%v", err)
	}
	if p.View() != before || parent.Calls() != calls {
		t.Fatal("failed PTY resize changed physical admission")
	}
	f, err := e.Snapshot(time.Now())
	if err != nil || f.Geometry != (Geometry{8, 4}) {
		t.Fatalf("endpoint geometry changed:%+v %v", f.Geometry, err)
	}
}

func TestPresenterQueuedEffectsDoNotWriteAfterPaintFailure(t *testing.T) {
	p, parent, e, _ := presenterFixture(t, ChildRequested)
	if err := p.Register(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	block := make(chan struct{})
	parent.Enqueue(ttyio.WriteStep{Block: block, Limit: 2, Err: errors.New("broken")})
	painted := make(chan error, 1)
	go func() { painted <- p.Select(context.Background(), e, Geometry{8, 5}, make([]Cell, 8)) }()
	<-parent.Started()
	effected := make(chan error, 1)
	go func() {
		_, err := p.EmitEffects(context.Background(), []Effect{{Kind: BellEffect, EndpointID: "selected", Sequence: 1}}, EffectPolicy{Bell: true})
		effected <- err
	}()
	deadline := time.Now().Add(time.Second)
	for len(p.requests) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(block)
	if err := <-painted; err == nil {
		t.Fatal("paint succeeded")
	}
	if err := <-effected; err == nil {
		t.Fatal("queued effect survived failed presenter")
	}
	if parent.Calls() != 1 {
		t.Fatal("parent output continued after failure")
	}
}

func TestPresenterChromeUpdatePreservesDragAndOwnsCells(t *testing.T) {
	p, _, e, input := presenterFixture(t, CouchAnyMotion)
	e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
	selectPresenter(t, p, e)
	if err := p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft}); err != nil {
		t.Fatal(err)
	}
	before := p.View()
	cells, err := StyledRows("status", 8, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.UpdateChrome(context.Background(), cells); err != nil {
		t.Fatal(err)
	}
	cells[0].Content = "X"
	if p.bottom[0].Content != "s" {
		t.Fatal("chrome aliases caller cells")
	}
	after := p.View()
	if after.DragDestination != before.DragDestination || after.Token != before.Token || after.Admitted != before.Admitted {
		t.Fatalf("chrome changed ownership:%+v -> %+v", before, after)
	}
	if err := e.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := string(input.Bytes()); got != "\x1b[<0;2;2M" {
		t.Fatalf("chrome canceled drag:%q", got)
	}
	if err := p.UpdateChrome(context.Background(), cells[:1]); err == nil {
		t.Fatal("invalid chrome width accepted")
	}
	if p.bottom[0].Content != "s" || p.View() != after {
		t.Fatal("invalid chrome mutated view")
	}
}

func TestPresenterParentAndOrphanGesturesNeverReachChild(t *testing.T) {
	for _, origin := range []string{"chrome", "panel", "orphan-motion", "orphan-release"} {
		t.Run(origin, func(t *testing.T) {
			p, _, e, input := presenterFixture(t, CouchAnyMotion)
			e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
			selectPresenter(t, p, e)
			switch origin {
			case "chrome":
				p.Input(context.Background(), uv.MouseClickEvent{X: 2, Y: 4, Button: uv.MouseLeft})
			case "panel":
				f, err := PanelFrame(Geometry{8, 5}, make([]Cell, 40), Cursor{})
				if err != nil {
					t.Fatal(err)
				}
				if err := p.Panel(context.Background(), f); err != nil {
					t.Fatal(err)
				}
				p.Input(context.Background(), uv.MouseClickEvent{X: 2, Y: 2, Button: uv.MouseLeft})
				selectPresenter(t, p, e)
			case "orphan-motion":
			case "orphan-release":
				p.Input(context.Background(), uv.MouseReleaseEvent{X: 2, Y: 2, Button: uv.MouseLeft})
				e.Flush(context.Background())
				if len(input.Bytes()) != 0 {
					t.Fatalf("orphan release=%q", input.Bytes())
				}
				return
			}
			p.Input(context.Background(), uv.MouseMotionEvent{X: 2, Y: 2, Button: uv.MouseLeft})
			p.Input(context.Background(), uv.MouseReleaseEvent{X: 2, Y: 2, Button: uv.MouseLeft})
			e.Flush(context.Background())
			if len(input.Bytes()) != 0 {
				t.Fatalf("unowned gesture leaked:%q", input.Bytes())
			}
			// A complete subsequent gesture remains usable.
			p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft})
			p.Input(context.Background(), uv.MouseReleaseEvent{X: 1, Y: 1, Button: uv.MouseLeft})
			e.Flush(context.Background())
			if got := string(input.Bytes()); got != "\x1b[<0;2;2M\x1b[<0;2;2m" {
				t.Fatalf("new gesture=%q", got)
			}
		})
	}
}
func TestPresenterMouseModeChangeEndsCapturedGesture(t *testing.T) {
	p, _, e, input := presenterFixture(t, CouchAnyMotion)
	e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
	selectPresenter(t, p, e)
	p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft})
	e.Flush(context.Background())
	e.Feed([]byte("\x1b[?1002l"), time.Now())
	p.Present(context.Background(), e)
	p.Flush(context.Background())
	e.Feed([]byte("\x1b[?1002h"), time.Now())
	p.Input(context.Background(), uv.MouseMotionEvent{X: 2, Y: 2, Button: uv.MouseLeft})
	p.Input(context.Background(), uv.MouseReleaseEvent{X: 2, Y: 2, Button: uv.MouseLeft})
	e.Flush(context.Background())
	if got := string(input.Bytes()); got != "\x1b[<0;2;2M" {
		t.Fatalf("gesture resumed after tracking change:%q", got)
	}
}

func TestPresenterModeEpochAndButtonOwnership(t *testing.T) {
	for _, change := range []string{"\x1b[?1002l\x1b[?1002h", "\x1b[?1006l\x1b[?1006h", "\x1bc\x1b[?1002h\x1b[?1006h"} {
		t.Run(fmt.Sprintf("%x", change), func(t *testing.T) {
			p, _, e, input := presenterFixture(t, CouchAnyMotion)
			e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
			selectPresenter(t, p, e)
			p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft})
			e.Feed([]byte(change), time.Now())
			p.Input(context.Background(), uv.MouseMotionEvent{X: 2, Y: 2, Button: uv.MouseLeft})
			if p.View().Gesture != GestureParent {
				t.Fatal("new protocol generation inherited child gesture")
			}
			p.Input(context.Background(), uv.MouseReleaseEvent{X: 2, Y: 2, Button: uv.MouseLeft})
			e.Flush(context.Background())
			if got := string(input.Bytes()); got != "\x1b[<0;2;2M" {
				t.Fatalf("old gesture crossed mode epoch:%q", got)
			}
		})
	}
	p, _, e, input := presenterFixture(t, CouchAnyMotion)
	e.Feed([]byte("\x1b[?1003h\x1b[?1006h"), time.Now())
	selectPresenter(t, p, e)
	p.Input(context.Background(), uv.MouseMotionEvent{X: 1, Y: 1, Button: uv.MouseNone})
	e.Flush(context.Background())
	if got := string(input.Bytes()); got != "\x1b[<35;2;2M" {
		t.Fatalf("no-button hover lost:%q", got)
	}
	p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft})
	p.Input(context.Background(), uv.MouseReleaseEvent{X: 1, Y: 1, Button: uv.MouseRight})
	if p.View().Gesture != GestureChild {
		t.Fatal("unrelated release ended owned gesture")
	}
	p.Input(context.Background(), uv.MouseReleaseEvent{X: 1, Y: 1, Button: uv.MouseLeft})
	e.Flush(context.Background())
	if got := string(input.Bytes()); got != "\x1b[<35;2;2M\x1b[<0;2;2M\x1b[<0;2;2m" {
		t.Fatalf("button ownership:%q", got)
	}
}

func TestPresenterFailedResizeDoesNotReviveCanceledGesture(t *testing.T) {
	p, _, e, input := presenterFixture(t, CouchAnyMotion)
	e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
	selectPresenter(t, p, e)
	p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft})
	e.Flush(context.Background())
	before := p.View()
	boom := errors.New("resize failed")
	for i := 0; i < 2; i++ {
		if err := p.Resize(context.Background(), Geometry{9, 6}, func(Geometry) error { return boom }); !errors.Is(err, boom) {
			t.Fatalf("resize:%v", err)
		}
	}
	after := p.View()
	if after.GeometryEpoch != before.GeometryEpoch || after.Token != before.Token {
		t.Fatal("failed resize changed geometry")
	}
	p.Input(context.Background(), uv.MouseMotionEvent{X: 2, Y: 2, Button: uv.MouseLeft})
	p.Input(context.Background(), uv.MouseReleaseEvent{X: 2, Y: 2, Button: uv.MouseLeft})
	e.Flush(context.Background())
	if got := string(input.Bytes()); got != "\x1b[<0;2;2M\x1b[<0;2;2m" {
		t.Fatalf("canceled gesture revived:%q", got)
	}
	p.Input(context.Background(), uv.MouseClickEvent{X: 3, Y: 1, Button: uv.MouseLeft})
	p.Input(context.Background(), uv.MouseReleaseEvent{X: 3, Y: 1, Button: uv.MouseLeft})
	e.Flush(context.Background())
	if got := string(input.Bytes()); got != "\x1b[<0;2;2M\x1b[<0;2;2m\x1b[<0;4;2M\x1b[<0;4;2m" {
		t.Fatalf("fresh gesture:%q", got)
	}
}
func TestPresenterInterruptedCancellationDoesNotEnqueueAgain(t *testing.T) {
	p, _, e, input := presenterFixture(t, CouchAnyMotion)
	e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
	selectPresenter(t, p, e)
	p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft})
	e.Flush(context.Background())
	block := make(chan struct{})
	input.Enqueue(ttyio.WriteStep{Block: block})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := p.call(context.Background(), func(context.Context) error { return p.cancelDrag(ctx) }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel delivery:%v", err)
	}
	close(block)
	if err := p.call(context.Background(), func(ctx context.Context) error { return p.cancelDrag(ctx) }); err != nil {
		t.Fatal(err)
	}
	e.Flush(context.Background())
	if got := string(input.Bytes()); got != "\x1b[<0;2;2M\x1b[<0;2;2m" {
		t.Fatalf("retry duplicated release:%q", got)
	}
}

func TestPresenterCancellationCallersResumeOnePendingDelivery(t *testing.T) {
	for _, caller := range []string{"release", "failure", "selection", "panel", "reconciliation", "resize"} {
		t.Run(caller, func(t *testing.T) {
			p, parent, e, input := presenterFixture(t, CouchAnyMotion)
			e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
			selectPresenter(t, p, e)
			p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft})
			e.Flush(context.Background())
			block := make(chan struct{})
			input.Enqueue(ttyio.WriteStep{Block: block})
			canceled, cancel := context.WithCancel(context.Background())
			cancel()
			if err := p.call(context.Background(), func(context.Context) error { return p.cancelDrag(canceled) }); !errors.Is(err, context.Canceled) {
				t.Fatalf("interrupt:%v", err)
			}
			close(block)
			// Each production cancellation caller must resume the pending acknowledgment,
			// not enqueue another release or restore the old gesture on a later failure.
			switch caller {
			case "release":
				if err := p.Release(context.Background()); err != nil {
					t.Fatal(err)
				}
				p.Release(context.Background())
			case "failure":
				parent.Enqueue(ttyio.WriteStep{Limit: 1, Err: errors.New("parent broke")})
				if _, err := p.EmitEffects(context.Background(), []Effect{{Kind: BellEffect, EndpointID: "selected", Sequence: 1}}, EffectPolicy{Bell: true}); err == nil {
					t.Fatal("missing parent failure")
				}
			case "selection":
				b, _ := newEndpointTest(t, "b")
				selectPresenter(t, p, b)
				selectPresenter(t, p, b)
			case "panel":
				f, err := PanelFrame(Geometry{8, 5}, make([]Cell, 40), Cursor{})
				if err != nil {
					t.Fatal(err)
				}
				if err := p.Panel(context.Background(), f); err != nil {
					t.Fatal(err)
				}
				p.Panel(context.Background(), f)
			case "reconciliation":
				if err := p.Input(context.Background(), uv.MouseMotionEvent{X: 2, Y: 2, Button: uv.MouseLeft}); err != nil {
					t.Fatal(err)
				}
			case "resize":
				for i := 0; i < 2; i++ {
					if err := p.Resize(context.Background(), Geometry{9, 6}, func(Geometry) error { return errors.New("resize failed") }); err == nil {
						t.Fatal("missing resize failure")
					}
				}
			}
			if pending := p.View().PendingMouseRelease; pending != "" {
				t.Fatalf("caller left acknowledged cancellation pending:%s", pending)
			}
			p.Input(context.Background(), uv.MouseMotionEvent{X: 2, Y: 2, Button: uv.MouseLeft})
			p.Input(context.Background(), uv.MouseReleaseEvent{X: 2, Y: 2, Button: uv.MouseLeft})
			e.Flush(context.Background())
			if got := string(input.Bytes()); got != "\x1b[<0;2;2M\x1b[<0;2;2m" {
				t.Fatalf("caller duplicated/revived cancellation:%q", got)
			}
		})
	}
}

func TestPresenterFailedChildCancellationStillReleasesParent(t *testing.T) {
	p, parent, e, input := presenterFixture(t, CouchAnyMotion)
	e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
	selectPresenter(t, p, e)
	p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft})
	e.Flush(context.Background())
	boom := errors.New("child input broke")
	input.Enqueue(ttyio.WriteStep{Limit: 2, Err: boom})
	resized := false
	if err := p.Resize(context.Background(), Geometry{9, 6}, func(Geometry) error { resized = true; return nil }); !errors.Is(err, boom) {
		t.Fatalf("resize cancellation:%v", err)
	}
	if resized || p.View().Gesture != GestureParent || p.View().State != Failed {
		t.Fatalf("failed cancellation resumed operation:%+v", p.View())
	}
	childCalls := input.Calls()
	parentBefore := len(parent.Bytes())
	if err := p.Release(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("release hid child cancellation failure:%v", err)
	}
	if p.View().State != Released || input.Calls() != childCalls {
		t.Fatal("release retried failed child write")
	}
	if !strings.Contains(string(parent.Bytes()[parentBefore:]), "\x18\x1b\\") {
		t.Fatal("child failure skipped parent cleanup")
	}
	parentCalls := parent.Calls()
	p.Release(context.Background())
	if parent.Calls() != parentCalls || input.Calls() != childCalls {
		t.Fatal("release retry repeated effects")
	}
}

func TestPresenterCanceledSelectionJoinsExistingReleaseWithoutRetry(t *testing.T) {
	p, _, e, input := presenterFixture(t, CouchAnyMotion)
	e.Feed([]byte("\x1b[?1002h\x1b[?1006h"), time.Now())
	selectPresenter(t, p, e)
	p.Input(context.Background(), uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft})
	e.Flush(context.Background())
	<-input.Started()
	b, _ := newEndpointTest(t, "b")
	block := make(chan struct{})
	input.Enqueue(ttyio.WriteStep{Block: block})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Select(ctx, b, Geometry{8, 5}, make([]Cell, 8)) }()
	<-input.Started()
	cancel()
	// Delivery may finish after its first waiter cancels. The failure path must
	// join that same queued packet instead of emitting another release.
	close(block)
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("selection cancellation:%v", err)
	}
	if p.View().Gesture != GestureParent || p.View().PendingMouseRelease != "" {
		t.Fatalf("cancellation state:%+v", p.View())
	}
	p.Release(context.Background())
	if got := string(input.Bytes()); got != "\x1b[<0;2;2M\x1b[<0;2;2m" {
		t.Fatalf("selection retry duplicated release:%q", got)
	}
}
