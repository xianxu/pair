package broadcast

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

var (
	// ErrIndicatorHidden ends a broadcast whose LIVE indicator stayed off the
	// operator's screen for longer than the grace period.
	ErrIndicatorHidden = errors.New("broadcast: LIVE indicator not visible on the operator's screen")
	ErrTooManyViewers  = errors.New("broadcast: too many viewers")
	ErrHubClosed       = errors.New("broadcast: ended")
)

const (
	DefaultGrace      = time.Second
	DefaultTick       = 100 * time.Millisecond
	DefaultMaxViewers = 16
	DefaultQueueDepth = 8
)

// HubOptions configures a Hub. Zero values take the defaults; After and Ticks
// are injected by tests to drive time by hand.
type HubOptions struct {
	ShowSwitcher bool
	Grace        time.Duration
	Tick         time.Duration
	MaxViewers   int
	QueueDepth   int
	// After returns a channel that fires once after d, and its stop function.
	After func(d time.Duration) (<-chan time.Time, func() bool)
	// Ticks, when set, replaces the hub's own resync ticker.
	Ticks <-chan time.Time
}

// watch is the indicator watch. Its transitions:
//
//	off    --Activate, indicator shown-->   shown
//	off    --Activate, indicator hidden-->  hidden (grace armed)
//	shown  --frame without indicator-->     hidden (grace armed)
//	hidden --frame with indicator-->        shown  (grace disarmed)
//	hidden --grace fires-->                 end(ErrIndicatorHidden)
//
// While off, frames are still gated by the indicator; only the timer waits.
type watch uint8

const (
	watchOff watch = iota
	watchShown
	watchHidden
)

type offered struct {
	frame terminal.Frame
	class terminal.FrameClass
}

// Hub turns tapped frames into the broadcast. One goroutine owns the stream,
// the subscribers and the grace timer, so a join and a frame are totally
// ordered. Offer never blocks the Presenter: the latest frame wins.
type Hub struct {
	opts HubOptions

	mu      sync.Mutex
	pending *offered
	wake    chan struct{}

	reqs      chan func()
	done      chan struct{}
	closeOnce sync.Once
	err       error        // loop-owned
	reason    atomic.Value // holds hubEnd; published before any end is observable

	// Owned by the loop goroutine.
	stream    Stream
	subs      map[*Subscription]struct{}
	resyncing int  // subscribers with resync set; the tick runs only while > 0
	shown     bool // the most recent offered frame showed the indicator
	watch     watch
	graceC    <-chan time.Time
	stopGrace func() bool
}

// Subscription is one viewer's queue. Messages closes when the broadcast ends
// or the subscription is closed.
type Subscription struct {
	h      *Hub
	c      chan Message
	resync bool // loop-owned: dropped diffs; next delivery must be a Join
}

func (s *Subscription) Messages() <-chan Message { return s.c }

// Close removes the subscription. Idempotent; safe after the hub ended.
func (s *Subscription) Close() {
	s.h.do(func() {
		if _, ok := s.h.subs[s]; ok {
			s.h.setResync(s, false)
			delete(s.h.subs, s)
			close(s.c)
		}
	})
}

func NewHub(opts HubOptions) *Hub {
	if opts.Grace <= 0 {
		opts.Grace = DefaultGrace
	}
	if opts.Tick <= 0 {
		opts.Tick = DefaultTick
	}
	if opts.MaxViewers <= 0 {
		opts.MaxViewers = DefaultMaxViewers
	}
	if opts.QueueDepth <= 0 {
		opts.QueueDepth = DefaultQueueDepth
	}
	if opts.After == nil {
		opts.After = func(d time.Duration) (<-chan time.Time, func() bool) {
			t := time.NewTimer(d)
			return t.C, t.Stop
		}
	}
	h := &Hub{opts: opts, wake: make(chan struct{}, 1), reqs: make(chan func()), done: make(chan struct{}), subs: make(map[*Subscription]struct{})}
	go h.run()
	return h
}

// Offer hands the hub a frame the operator's screen just showed. It never
// blocks; an unprocessed earlier offer is replaced, which is safe because the
// stream diffs against what it last sent, not against the previous offer.
func (h *Hub) Offer(f terminal.Frame, class terminal.FrameClass) {
	h.mu.Lock()
	h.pending = &offered{f, class}
	h.mu.Unlock()
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

// Activate starts the indicator watch: from now on the LIVE indicator must be
// on the operator's screen, or return to it within Grace. Called once the tap
// is installed, not while the tunnel is still opening. Idempotent.
func (h *Hub) Activate() {
	h.do(func() {
		if h.watch != watchOff {
			return
		}
		if h.shown {
			h.watch = watchShown
		} else {
			h.watch = watchHidden
			h.armGrace()
		}
	})
}

// Subscribe adds a viewer. Its first message is the current frame in full.
func (h *Hub) Subscribe() (*Subscription, error) {
	var sub *Subscription
	err := ErrHubClosed
	h.do(func() {
		if len(h.subs) >= h.opts.MaxViewers {
			err = ErrTooManyViewers
			return
		}
		sub = &Subscription{h: h, c: make(chan Message, h.opts.QueueDepth)}
		if m, ok, jerr := h.stream.Join(); jerr != nil {
			h.end(jerr)
			return
		} else if ok {
			sub.c <- m
		}
		h.subs[sub] = struct{}{}
		err = nil
	})
	return sub, err
}

// Close ends the broadcast with reason (nil: ErrHubClosed). Idempotent.
func (h *Hub) Close(reason error) {
	if reason == nil {
		reason = ErrHubClosed
	}
	h.do(func() { h.end(reason) })
}

func (h *Hub) Done() <-chan struct{} { return h.done }

// endReason boxes the error so atomic.Value always stores one concrete type.
type hubEnd struct{ err error }

// Err is why the hub ended; nil while it runs. The reason is published before
// any subscriber queue closes, so a consumer reacting to its closed queue
// always reads the real reason, even before Done closes.
func (h *Hub) Err() error {
	if r, ok := h.reason.Load().(hubEnd); ok {
		return r.err
	}
	return nil
}

// do runs f on the loop, after any pending offer, so callers observe their own
// Offer-then-Subscribe order. It returns without running f once the hub ended.
func (h *Hub) do(f func()) {
	ran := make(chan struct{})
	select {
	case h.reqs <- func() { f(); close(ran) }:
		select {
		case <-ran:
		case <-h.done:
		}
	case <-h.done:
	}
}

// sync waits until the loop has processed everything offered so far.
func (h *Hub) sync() { h.do(func() {}) }

func (h *Hub) run() {
	ticks := h.opts.Ticks
	if ticks == nil {
		t := time.NewTicker(h.opts.Tick)
		defer t.Stop()
		ticks = t.C
	}
	for {
		var tickC <-chan time.Time
		if h.resyncing > 0 {
			tickC = ticks
		}
		select {
		case <-h.wake:
			h.takePending()
		case f := <-h.reqs:
			h.takePending()
			f()
		case <-h.graceC:
			h.graceC = nil
			if h.watch == watchHidden {
				h.end(ErrIndicatorHidden)
			}
		case <-tickC:
			h.onTick()
		}
		if h.err != nil {
			close(h.done)
			return
		}
	}
}

// onTick retries pending resyncs, so a viewer that dropped diffs catches up
// even when the operator's screen has gone quiet.
func (h *Hub) onTick() { h.deliver(Message{}, false) }

func (h *Hub) takePending() {
	h.mu.Lock()
	o := h.pending
	h.pending = nil
	h.mu.Unlock()
	if o != nil && h.err == nil {
		h.accept(*o)
	}
}

func (h *Hub) accept(o offered) {
	h.shown = IndicatorShown(o.frame)
	switch {
	case !h.shown && h.watch == watchShown:
		h.watch = watchHidden
		h.armGrace()
	case h.shown && h.watch == watchHidden:
		h.watch = watchShown
		h.disarmGrace()
	}
	if !h.shown {
		// Withheld: viewers never get a frame the operator didn't see marked.
		return
	}
	vf, err := ViewerFrame(o.frame, o.class, h.opts.ShowSwitcher)
	if err != nil {
		h.end(err)
		return
	}
	m, ok, err := h.stream.Next(vf)
	if err != nil {
		h.end(err)
		return
	}
	h.deliver(m, ok)
}

// deliver sends the diff m (when ok) to up-to-date viewers, and a full render
// to any viewer that dropped diffs, once its queue has room. A viewer never
// gets a diff against a frame it didn't receive.
func (h *Hub) deliver(m Message, ok bool) {
	var join *Message
	for sub := range h.subs {
		if sub.resync {
			if join == nil {
				j, jok, err := h.stream.Join()
				if err != nil {
					h.end(err)
					return
				}
				if !jok {
					h.setResync(sub, false)
					continue
				}
				join = &j
			}
			select {
			case sub.c <- *join:
				h.setResync(sub, false)
			default:
			}
			continue
		}
		if !ok {
			continue
		}
		select {
		case sub.c <- m:
		default:
			h.setResync(sub, true)
		}
	}
}

func (h *Hub) setResync(sub *Subscription, on bool) {
	if sub.resync == on {
		return
	}
	sub.resync = on
	if on {
		h.resyncing++
	} else {
		h.resyncing--
	}
}

func (h *Hub) armGrace() {
	h.graceC, h.stopGrace = h.opts.After(h.opts.Grace)
}

func (h *Hub) disarmGrace() {
	if h.stopGrace != nil {
		h.stopGrace()
	}
	h.graceC, h.stopGrace = nil, nil
}

func (h *Hub) end(reason error) {
	if h.err != nil {
		return
	}
	h.err = reason
	h.reason.Store(hubEnd{reason})
	h.disarmGrace()
	for sub := range h.subs {
		close(sub.c)
	}
	h.subs = nil
}
