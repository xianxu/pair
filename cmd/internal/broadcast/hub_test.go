package broadcast

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

// manualClock drives the hub's grace timer and resync tick by hand.
type manualClock struct {
	grace   chan time.Time
	armed   int
	stopped int
	ticks   chan time.Time
}

func newManualClock() *manualClock {
	return &manualClock{grace: make(chan time.Time, 1), ticks: make(chan time.Time)}
}

func (c *manualClock) after(time.Duration) (<-chan time.Time, func() bool) {
	c.armed++
	ch := make(chan time.Time, 1)
	c.grace = ch
	return ch, func() bool { c.stopped++; return true }
}

// fireGrace fires the most recently armed grace timer, if it hasn't fired.
func (c *manualClock) fireGrace(h *Hub) {
	select {
	case c.grace <- time.Now():
	default:
	}
	h.sync()
}

// tick runs one resync tick on the hub's loop. The hub's own ticker channel
// (c.ticks) never fires in tests.
func (c *manualClock) tick(h *Hub) { h.do(h.onTick) }

func testHub(t *testing.T, opts HubOptions) (*Hub, *manualClock) {
	t.Helper()
	clock := newManualClock()
	opts.After = clock.after
	opts.Ticks = clock.ticks
	h := NewHub(opts)
	t.Cleanup(func() { h.Close(nil) })
	return h, clock
}

// viewer drains a subscription into an emulator screen.
type viewer struct {
	sub  *Subscription
	scr  screen
	msgs int
}

func subscribe(t *testing.T, h *Hub) *viewer {
	t.Helper()
	sub, err := h.Subscribe()
	if err != nil {
		t.Fatal(err)
	}
	return &viewer{sub: sub}
}

// drain applies every queued message; it reports whether the channel closed.
func (v *viewer) drain(t *testing.T) (closed bool) {
	t.Helper()
	for {
		select {
		case m, ok := <-v.sub.Messages():
			if !ok {
				return true
			}
			v.msgs++
			v.scr.apply(t, m)
		default:
			return false
		}
	}
}

func (v *viewer) next(t *testing.T) (Message, bool) {
	t.Helper()
	select {
	case m, ok := <-v.sub.Messages():
		return m, ok
	default:
		t.Fatal("no message queued")
		return Message{}, false
	}
}

func live(t *testing.T, body string) terminal.Frame {
	return textFrame(t, 40, 4, body, liveChrome("tabs"))
}

func hidden(t *testing.T, body string) terminal.Frame {
	return textFrame(t, 40, 4, body, "tabs")
}

func offer(h *Hub, f terminal.Frame, class terminal.FrameClass) {
	h.Offer(f, class)
	h.sync()
}

func TestHubLateJoinerGetsCurrentFrame(t *testing.T) {
	h, _ := testHub(t, HubOptions{})
	a := live(t, "alpha")
	offer(h, a, terminal.FramePublic)
	v := subscribe(t, h)
	m, _ := v.next(t)
	if !strings.Contains(string(m.Data), "\x1b[2J") {
		t.Fatal("late joiner's first message is not a full render")
	}
	v.scr.apply(t, m)
	b := live(t, "beta")
	offer(h, b, terminal.FramePublic)
	v.drain(t)
	if v.scr.text() != frameText(t, b) {
		t.Fatalf("viewer screen %q", v.scr.text())
	}
}

func TestHubWithholdsFramesWithoutIndicator(t *testing.T) {
	h, _ := testHub(t, HubOptions{})
	v := subscribe(t, h)
	offer(h, hidden(t, "WITHHELD"), terminal.FramePublic)
	v.drain(t)
	if v.msgs != 0 {
		t.Fatalf("withheld frame produced %d messages", v.msgs)
	}
	offer(h, live(t, "shown"), terminal.FramePublic)
	v.drain(t)
	if v.msgs != 1 || strings.Contains(v.scr.text(), "WITHHELD") {
		t.Fatalf("msgs=%d screen=%q", v.msgs, v.scr.text())
	}
}

func TestHubStopsWhenIndicatorHiddenPastGrace(t *testing.T) {
	t.Run("hidden after live", func(t *testing.T) {
		h, clock := testHub(t, HubOptions{})
		h.Activate()
		v := subscribe(t, h)
		offer(h, live(t, "a"), terminal.FramePublic)
		offer(h, hidden(t, "b"), terminal.FramePublic)
		clock.fireGrace(h)
		assertEnded(t, h, v, ErrIndicatorHidden)
	})
	t.Run("never shown after activate", func(t *testing.T) {
		h, clock := testHub(t, HubOptions{})
		v := subscribe(t, h)
		h.Activate()
		clock.fireGrace(h)
		assertEnded(t, h, v, ErrIndicatorHidden)
	})
	t.Run("idle live screen at activate keeps running", func(t *testing.T) {
		// BR-1: the operator's screen already shows LIVE and then goes quiet;
		// no further frame arrives. Activate must not arm the grace timer.
		h, clock := testHub(t, HubOptions{})
		offer(h, live(t, "a"), terminal.FramePublic)
		h.Activate()
		clock.fireGrace(h)
		if clock.armed != 0 {
			t.Fatalf("grace armed %d times with the indicator showing", clock.armed)
		}
		select {
		case <-h.Done():
			t.Fatalf("idle live broadcast ended: %v", h.Err())
		default:
		}
	})
	t.Run("hidden at activate arms grace", func(t *testing.T) {
		h, clock := testHub(t, HubOptions{})
		v := subscribe(t, h)
		offer(h, live(t, "a"), terminal.FramePublic)
		offer(h, hidden(t, "b"), terminal.FramePublic)
		h.Activate()
		clock.fireGrace(h)
		assertEnded(t, h, v, ErrIndicatorHidden)
	})
	t.Run("no timer before activate", func(t *testing.T) {
		h, clock := testHub(t, HubOptions{})
		offer(h, hidden(t, "b"), terminal.FramePublic)
		if clock.armed != 0 {
			t.Fatalf("grace armed %d times before Activate", clock.armed)
		}
	})
}

func assertEnded(t *testing.T, h *Hub, v *viewer, want error) {
	t.Helper()
	select {
	case <-h.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("hub did not end")
	}
	if !errors.Is(h.Err(), want) {
		t.Fatalf("Err() = %v, want %v", h.Err(), want)
	}
	if !v.drain(t) {
		t.Fatal("subscriber channel not closed at end")
	}
	if _, err := h.Subscribe(); !errors.Is(err, ErrHubClosed) {
		t.Fatalf("subscribe after end: %v", err)
	}
}

func TestHubIndicatorReturnsBeforeGrace(t *testing.T) {
	h, clock := testHub(t, HubOptions{})
	h.Activate()
	offer(h, live(t, "a"), terminal.FramePublic)
	offer(h, hidden(t, "b"), terminal.FramePublic)
	offer(h, live(t, "c"), terminal.FramePublic)
	if clock.stopped == 0 {
		t.Fatal("grace timer not cancelled when the indicator returned")
	}
	select {
	case <-h.Done():
		t.Fatal("hub ended although the indicator returned")
	default:
	}
}

func TestHubPrivateFrame(t *testing.T) {
	for _, show := range []bool{false, true} {
		t.Run(fmt.Sprintf("showSwitcher=%v", show), func(t *testing.T) {
			h, _ := testHub(t, HubOptions{ShowSwitcher: show})
			v := subscribe(t, h)
			offer(h, live(t, "FLEET-SECRET"), terminal.FramePrivate)
			v.drain(t)
			leaked := strings.Contains(v.scr.text(), "FLEET-SECRET")
			if leaked != show {
				t.Fatalf("switcher visible=%v, want %v: %q", leaked, show, v.scr.text())
			}
			if !show && !strings.Contains(v.scr.text(), PlaceholderText) {
				t.Fatalf("placeholder missing: %q", v.scr.text())
			}
		})
	}
}

func TestHubSlowViewerResyncs(t *testing.T) {
	h, clock := testHub(t, HubOptions{QueueDepth: 3})
	fast, slow := subscribe(t, h), subscribe(t, h)
	var last terminal.Frame
	for i := range 10 {
		last = live(t, fmt.Sprintf("frame %d", i))
		offer(h, last, terminal.FramePublic)
		fast.drain(t)
	}
	if fast.msgs != 10 || fast.scr.text() != frameText(t, last) {
		t.Fatalf("fast viewer got %d messages, screen %q", fast.msgs, fast.scr.text())
	}
	// The slow viewer's queue holds a valid prefix; after it, the next message
	// must be a full render, with no diff it can't apply in between.
	for range 3 {
		m, _ := slow.next(t)
		slow.scr.apply(t, m)
	}
	clock.tick(h)
	m, _ := slow.next(t)
	if !strings.Contains(string(m.Data), "\x1b[2J") {
		t.Fatal("first message after overflow is not a full render")
	}
	slow.scr.apply(t, m)
	if slow.scr.text() != frameText(t, last) {
		t.Fatalf("slow viewer screen %q", slow.scr.text())
	}
}

func TestHubOfferNeverBlocks(t *testing.T) {
	h, _ := testHub(t, HubOptions{})
	release := make(chan struct{})
	stalled := make(chan struct{})
	go h.do(func() { close(stalled); <-release })
	<-stalled
	f := live(t, "x")
	start := time.Now()
	for range 10000 {
		h.Offer(f, terminal.FramePublic)
	}
	close(release)
	if d := time.Since(start); d > time.Second {
		t.Fatalf("10k offers against a stalled hub took %v", d)
	}
}

func TestHubMaxViewers(t *testing.T) {
	h, _ := testHub(t, HubOptions{MaxViewers: 2})
	subscribe(t, h)
	v := subscribe(t, h)
	if _, err := h.Subscribe(); !errors.Is(err, ErrTooManyViewers) {
		t.Fatalf("third subscribe: %v", err)
	}
	v.sub.Close()
	subscribe(t, h)
}

func TestHubCloseIdempotent(t *testing.T) {
	h, _ := testHub(t, HubOptions{})
	v := subscribe(t, h)
	reason := errors.New("operator stopped")
	h.Close(reason)
	h.Close(errors.New("second"))
	assertEnded(t, h, v, reason)
	h.Offer(live(t, "late"), terminal.FramePublic)
}

// TestHubRandomInterleavings checks the hub's invariants against seeded
// random interleavings of offers, subscriptions, slow drains, ticks, grace
// expiry and close:
//  1. every viewer screen equals some frame the hub accepted, and after
//     quiescence plus a tick it equals the current one;
//  2. nothing derived from a frame without the indicator reaches a viewer;
//  3. no private body reaches a viewer unless ShowSwitcher;
//  4. after the end, every queue is closed.
func TestHubRandomInterleavings(t *testing.T) {
	for seed := int64(1); seed <= 500; seed++ {
		if !t.Run(fmt.Sprint(seed), func(t *testing.T) { hubInterleaving(t, seed) }) {
			t.Fatalf("seed %d failed", seed)
		}
	}
}

func hubInterleaving(t *testing.T, seed int64) {
	r := rand.New(rand.NewSource(seed))
	show := r.Intn(2) == 0
	// The pointer watch (#412) runs alongside, on its own clock, against a
	// model: OnPointerHidden fires exactly when armed with the marker hidden,
	// disarms, and never ends the hub.
	pclock := newManualClock()
	var hiddenCalls atomic.Int32
	h, clock := testHub(t, HubOptions{ShowSwitcher: show, QueueDepth: 1 + r.Intn(4),
		PointerAfter: pclock.after, OnPointerHidden: func() { hiddenCalls.Add(1) }})
	armed, pshown := false, false
	accepted := map[string]bool{"": true}
	current := ""
	var viewers []*viewer
	ended := false
	// check inspects screens without draining: slow viewers must stay slow,
	// or no queue ever overflows and resync goes untested.
	check := func(step string) {
		t.Helper()
		for i, v := range viewers {
			text := v.scr.text()
			if !accepted[text] {
				t.Fatalf("%s: viewer %d shows a frame the hub never accepted: %q", step, i, text)
			}
			if strings.Contains(text, "WITHHELD") {
				t.Fatalf("%s: viewer %d got a frame without the indicator", step, i)
			}
			if !show && strings.Contains(text, "FLEET-SECRET") {
				t.Fatalf("%s: viewer %d got a private body", step, i)
			}
		}
	}
	// Activate at a random point (or never): the watch must judge the
	// indicator by the frames already offered, not assume it is missing.
	activateAt := r.Intn(70)
	shown := false // the indicator on the most recent offered frame
	for step := range 60 {
		if step == activateAt {
			h.Activate()
			if shown {
				clock.fireGrace(h)
				select {
				case <-h.Done():
					t.Fatalf("activate with LIVE showing ended the hub: %v", h.Err())
				default:
				}
			}
		}
		if ended {
			break
		}
		switch op := r.Intn(12); {
		case op == 10:
			if armed {
				h.DisarmPointer()
			} else {
				h.ArmPointer()
			}
			armed = !armed
		case op == 11:
			before := hiddenCalls.Load()
			pclock.fireGrace(h)
			want := armed && !pshown
			if got := hiddenCalls.Load() - before; got != map[bool]int32{true: 1, false: 0}[want] {
				t.Fatalf("step %d: pointer fire called back %d times (armed=%v shown=%v)", step, got, armed, pshown)
			}
			if want {
				armed = false
			}
			select {
			case <-h.Done():
				if !ended {
					t.Fatalf("step %d: the pointer watch ended the hub", step)
				}
			default:
			}
		case op < 5:
			// A burst, so queues of depth 1-4 overflow.
			for n := range 1 + r.Intn(6) {
				class := terminal.FramePublic
				body := fmt.Sprintf("frame %d.%d", step, n)
				if r.Intn(4) == 0 {
					class, body = terminal.FramePrivate, body+" FLEET-SECRET"
				}
				if r.Intn(5) == 0 {
					offer(h, hidden(t, body+" WITHHELD"), class)
					shown, pshown = false, false
					continue
				}
				f := live(t, body)
				if r.Intn(2) == 0 {
					f = livePointer(t, body)
				}
				offer(h, f, class)
				shown, pshown = true, PointerShown(f)
				vf, err := ViewerFrame(f, class, show)
				if err != nil {
					t.Fatal(err)
				}
				current = frameText(t, vf)
				accepted[current] = true
			}
		case op < 6 && len(viewers) < 5:
			viewers = append(viewers, subscribe(t, h))
		case op < 8 && len(viewers) > 0:
			viewers[r.Intn(len(viewers))].drain(t)
		case op < 9:
			clock.tick(h)
		default:
			if r.Intn(4) == 0 {
				clock.fireGrace(h)
				select {
				case <-h.Done():
					ended = true
					if shown || step < activateAt {
						t.Fatalf("step %d: grace ended the hub (shown=%v, activated=%v)", step, shown, step >= activateAt)
					}
				default:
				}
			}
		}
		check(fmt.Sprintf("step %d", step))
	}
	if !ended {
		// Quiescence: drain everyone, tick, drain again. Everyone is current.
		for _, v := range viewers {
			v.drain(t)
		}
		clock.tick(h)
		for i, v := range viewers {
			v.drain(t)
			if current != "" && v.scr.text() != current {
				t.Fatalf("viewer %d not current after quiescence: %q, want %q", i, v.scr.text(), current)
			}
		}
		h.Close(errors.New("done"))
		<-h.Done()
	}
	for i, v := range viewers {
		if !v.drain(t) {
			t.Fatalf("viewer %d queue not closed after end", i)
		}
	}
}

// The real ticker runs only while a viewer awaits resync; this drives it
// through that gate rather than calling onTick by hand.
func TestHubRealTickerResyncsQuietScreen(t *testing.T) {
	h := NewHub(HubOptions{QueueDepth: 1, Tick: 5 * time.Millisecond})
	t.Cleanup(func() { h.Close(nil) })
	slow := subscribe(t, h)
	var last terminal.Frame
	for i := range 5 {
		last = live(t, fmt.Sprintf("quiet %d", i))
		offer(h, last, terminal.FramePublic)
	}
	// The screen is now quiet. Reading the stale prefix frees room; the tick
	// alone must bring the viewer current.
	deadline := time.After(5 * time.Second)
	for slow.scr.text() != frameText(t, last) {
		select {
		case m := <-slow.sub.Messages():
			slow.scr.apply(t, m)
		case <-deadline:
			t.Fatalf("viewer never caught up: %q", slow.scr.text())
		}
	}
	var pending int
	h.do(func() { pending = h.resyncing })
	if pending != 0 {
		t.Fatalf("resyncing = %d after catch-up", pending)
	}
}

// BR-5: a consumer that sees its queue close must read the real reason at
// that moment, not nil (which the server reported as an operator stop).
// The end state is published before the end is observable.
func TestHubReasonVisibleWhenQueueCloses(t *testing.T) {
	reason := errors.New("tunnel died")
	for i := range 2000 {
		h := NewHub(HubOptions{Ticks: make(chan time.Time)})
		sub, err := h.Subscribe()
		if err != nil {
			t.Fatal(err)
		}
		go h.Close(reason)
		for range sub.Messages() {
		}
		if got := h.Err(); !errors.Is(got, reason) {
			t.Fatalf("iteration %d: Err() = %v when the queue closed", i, got)
		}
	}
}

// livePointer is a frame whose status row shows LIVE and the active pointer
// marker, as Couch draws it while pointing is on.
func livePointer(t *testing.T, body string) terminal.Frame {
	return textFrame(t, 40, 4, body, LiveSGR+LiveLabel+"\x1b[0m "+PointerSGR+PointerLabel+"\x1b[0m tabs")
}

type pointerHub struct {
	h       *Hub
	clock   *manualClock
	pclock  *manualClock
	mu      sync.Mutex
	hiddens int
}

func (p *pointerHub) hidden() int { p.mu.Lock(); defer p.mu.Unlock(); return p.hiddens }

func newPointerHub(t *testing.T) *pointerHub {
	t.Helper()
	p := &pointerHub{clock: newManualClock(), pclock: newManualClock()}
	p.h = NewHub(HubOptions{
		After: p.clock.after, PointerAfter: p.pclock.after, Ticks: p.clock.ticks,
		OnPointerHidden: func() { p.mu.Lock(); p.hiddens++; p.mu.Unlock() },
	})
	t.Cleanup(func() { p.h.Close(nil) })
	return p
}

func TestPointerShown(t *testing.T) {
	if !PointerShown(livePointer(t, "x")) {
		t.Fatal("active pointer marker not recognised")
	}
	for name, f := range map[string]terminal.Frame{
		"live only":        live(t, "x"),
		"dim marker":       textFrame(t, 40, 4, "x", LiveSGR+LiveLabel+"\x1b[0m "+PointerLabel+" tabs"),
		"marker misplaced": textFrame(t, 40, 4, "x", LiveSGR+LiveLabel+"\x1b[0m  "+PointerSGR+PointerLabel+"\x1b[0m"),
		"clipped":          textFrame(t, 8, 4, "x", LiveSGR+LiveLabel+"\x1b[0m "+PointerSGR+PointerLabel+"\x1b[0m"),
	} {
		if PointerShown(f) {
			t.Errorf("%s: PointerShown true", name)
		}
	}
}

func TestHubPointerWatch(t *testing.T) {
	t.Run("hidden past grace turns pointing off, hub lives", func(t *testing.T) {
		p := newPointerHub(t)
		offer(p.h, livePointer(t, "a"), terminal.FramePublic)
		p.h.ArmPointer()
		offer(p.h, live(t, "b"), terminal.FramePublic) // marker gone, LIVE still shown
		p.pclock.fireGrace(p.h)
		if p.hidden() != 1 {
			t.Fatalf("OnPointerHidden called %d times", p.hidden())
		}
		select {
		case <-p.h.Done():
			t.Fatal("the pointer watch ended the hub")
		default:
		}
		// Disarmed after firing: a later grace fire does nothing.
		p.pclock.fireGrace(p.h)
		if p.hidden() != 1 {
			t.Fatal("fired twice")
		}
	})
	t.Run("armed while hidden", func(t *testing.T) {
		p := newPointerHub(t)
		offer(p.h, live(t, "a"), terminal.FramePublic)
		p.h.ArmPointer()
		p.pclock.fireGrace(p.h)
		if p.hidden() != 1 {
			t.Fatal("marker never shown after arming, yet pointing stayed on")
		}
	})
	t.Run("marker returns before grace", func(t *testing.T) {
		p := newPointerHub(t)
		offer(p.h, livePointer(t, "a"), terminal.FramePublic)
		p.h.ArmPointer()
		offer(p.h, live(t, "b"), terminal.FramePublic)
		offer(p.h, livePointer(t, "c"), terminal.FramePublic)
		p.pclock.fireGrace(p.h)
		if p.hidden() != 0 {
			t.Fatal("fired although the marker came back")
		}
	})
	t.Run("never fires unarmed or after disarm", func(t *testing.T) {
		p := newPointerHub(t)
		offer(p.h, live(t, "a"), terminal.FramePublic)
		p.pclock.fireGrace(p.h)
		p.h.ArmPointer()
		p.h.DisarmPointer()
		p.pclock.fireGrace(p.h)
		if p.hidden() != 0 {
			t.Fatalf("fired %d times unarmed", p.hidden())
		}
	})
}

func TestHubCurrent(t *testing.T) {
	p := newPointerHub(t)
	if p.h.Current().OK {
		t.Fatal("Current before any frame")
	}
	offer(p.h, live(t, "a"), terminal.FramePrivate)
	c := p.h.Current()
	if !c.OK || c.Geometry != (terminal.Geometry{Cols: 40, Rows: 4}) || c.Class != terminal.FramePrivate || c.PointerShown {
		t.Fatalf("Current %+v", c)
	}
	offer(p.h, livePointer(t, "b"), terminal.FramePublic)
	if c := p.h.Current(); !c.PointerShown || c.Class != terminal.FramePublic {
		t.Fatalf("Current %+v", c)
	}
}
