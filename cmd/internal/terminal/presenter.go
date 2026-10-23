package terminal

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"strings"
	"sync"
	"time"
)

type ParentMousePolicy uint8

const (
	// AnyMotion always requests any-motion reports: the parent owns clickable
	// chrome (couch's status row) and forwards to the child only what the
	// child's own tracking mode asks for. Only safe when something below
	// implements selection -- couch's child is zellij.
	AnyMotion ParentMousePolicy = iota
	// ChildRequested mirrors the child's tracking, leaving the host its native
	// selection over a child that asked for none: pair term, whose host is
	// zellij and whose child is often a plain shell (#326).
	ChildRequested
)

type EffectPolicy struct{ Bell, Title, Clipboard, Notifications bool }
type presentRequest struct {
	ctx  context.Context
	run  func(context.Context) error
	done chan error
}

// Presenter owns every physical parent write. Its FIFO pauses input behind a
// selection's complete paint; endpoint output is only a coalesced dirty signal.
type Presenter struct {
	mu             sync.Mutex
	view           View
	closing        bool
	writer         ttyio.Writer
	policy         ParentMousePolicy
	requests       chan presentRequest
	wake           chan struct{}
	stop           chan context.Context
	done           chan struct{}
	life           context.Context
	cancel         context.CancelFunc
	failureOnce    sync.Once
	failed         chan struct{}
	failureErr     error
	releaseOnce    sync.Once
	releaseErr     error
	dirty          *Endpoint
	selected       *Endpoint
	host           Geometry
	bottom         []Cell
	previous       Frame
	cancelTarget   *Endpoint
	mouseEpoch     uint64
	mouse          uv.Mouse
	effects        map[string]uint64
	origins        map[string]*Endpoint
	confirmedModes parentModes
	keyboardOwned  bool // setup's push, on the screen the parent started on
	altOwned       bool
	// altKeyboardOwned: the alternate screen carries the push too. Kitty
	// keyboard flags are a stack per screen (#279); see writeFramePacket.
	altKeyboardOwned bool
	history          HistoryState
	tap              Tap
	overlay          Overlay
	operatorLayer    OperatorLayer
	panel            Frame      // the panel on screen, kept for Refresh; zero while an endpoint is selected
	class            FrameClass // of what is on screen: Select → public, Panel → its argument
	modesKnown       bool
	parentTouched    bool // actor-owned; even an interrupted first write requires release
	// stalled: a parent write ran out of its deadline, so the parent's screen
	// is unknown and nothing is painted until a probe gets through (#430).
	stalled bool
	stalls  uint64

	chromeRevision          uint64
	confirmedChromeRevision uint64
}

func NewPresenter(w ttyio.Writer, policy ParentMousePolicy) *Presenter {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Presenter{failed: make(chan struct{}), writer: w, policy: policy, requests: make(chan presentRequest, MaxPendingEvents), wake: make(chan struct{}, 1), stop: make(chan context.Context, 1), done: make(chan struct{}), life: ctx, cancel: cancel, effects: make(map[string]uint64), origins: make(map[string]*Endpoint)}
	go p.run()
	return p
}

// ChromeState is the admitted chrome layout and whether its current revision
// has completed a physical publication. A stalled parent revokes confirmation.
type ChromeState struct {
	Host      Geometry
	Rows      int
	Revision  uint64
	Confirmed bool
}

// ChromeState reads publication state on the presenter's FIFO.
func (p *Presenter) ChromeState(ctx context.Context) (ChromeState, error) {
	var state ChromeState
	err := p.call(ctx, func(context.Context) error {
		rows := p.panel.ChromeRows
		if p.selected != nil && p.host.Cols > 0 {
			rows = len(p.bottom) / p.host.Cols
		}
		state = ChromeState{Host: p.host, Rows: rows, Revision: p.chromeRevision, Confirmed: p.chromeConfirmed()}
		return nil
	})
	return state, err
}

func (p *Presenter) chromeConfirmed() bool {
	return p.chromeRevision > 0 && p.chromeRevision == p.confirmedChromeRevision && !p.stalled && p.previous.Geometry == p.host && p.View().State == Ready
}

func (p *Presenter) admitChrome() { p.chromeRevision++; p.confirmedChromeRevision = 0 }

func (p *Presenter) View() View { p.mu.Lock(); defer p.mu.Unlock(); return p.view }
func (p *Presenter) transition(e ViewEvent) (ViewEffects, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.transitionLocked(e)
}
func (p *Presenter) transitionLocked(e ViewEvent) (ViewEffects, error) {
	v, out, err := Transition(p.view, e)
	if err == nil {
		p.view = v
	}
	return out, err
}

// presenterRefusal preserves the first failure's complete error tree. Adding a
// synthetic leaf with errors.Join would misclassify cancellation-only shutdown.
func presenterRefusal(reason string, cause error) error {
	if cause != nil {
		return fmt.Errorf("terminal: presenter %s: %w", reason, cause)
	}
	return fmt.Errorf("terminal: presenter %s", reason)
}

func (p *Presenter) call(ctx context.Context, f func(context.Context) error) error {
	r := presentRequest{ctx: ctx, run: f, done: make(chan error, 1)}
	p.mu.Lock()
	if p.closing || p.view.State == Failed {
		err := presenterRefusal("unavailable", p.failureErr)
		p.mu.Unlock()
		return err
	}
	select {
	case p.requests <- r:
		p.mu.Unlock()
	default:
		p.mu.Unlock()
		return ErrBackpressure
	}
	select {
	case err := <-r.done:
		return err
	case <-p.done:
		return presenterRefusal("released", p.Failure())
	}
}
func (p *Presenter) run() {
	defer close(p.done)
	var timer *time.Timer
	var tick <-chan time.Time
	var scheduled time.Time
	var retry <-chan time.Time
	armRetry := func() {
		if p.stalled && retry == nil && p.View().State != Failed {
			retry = time.After(StallRetry)
		}
	}
	schedule := func(d time.Duration) {
		next := time.Now().Add(d)
		if tick != nil && !next.Before(scheduled) {
			return
		}
		scheduled = next
		if timer != nil {
			timer.Stop()
		}
		timer = time.NewTimer(d)
		tick = timer.C
	}
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case ctx := <-p.stop:
			// Release needs no caller deadline: the child's InputWriter bounds
			// drag-cancellation delivery and write bounds the reset, each by
			// WriteTimeout.
			cancelErr := p.cancelDrag(ctx)
			p.releaseErr = cancelErr
			if p.parentTouched {
				if err := p.write(ctx, append(p.releaseAlt(), parentReleaseControls(p.keyboardOwned)...), false); err != nil {
					p.releaseErr = errors.Join(cancelErr, fmt.Errorf("%w: %w", ErrModesNotRestored, err))
				}
			}
			p.transition(ViewEvent{Kind: ReleaseView})
			return
		case r := <-p.requests:
			if p.View().State == Failed {
				r.done <- presenterRefusal("failed", p.Failure())
				continue
			}
			if p.life.Err() != nil {
				r.done <- p.life.Err()
				continue
			}
			if err := r.ctx.Err(); err != nil {
				r.done <- err
				continue
			}
			ctx, cancel := context.WithCancel(r.ctx)
			stop := context.AfterFunc(p.life, cancel)
			err := r.run(ctx)
			stop()
			cancel()
			r.done <- err
			armRetry()
			if p.selected != nil && p.View().State == Ready {
				if deadline := p.selected.NextPublication(); !deadline.IsZero() {
					schedule(time.Until(deadline))
				}
			}
		case <-p.wake:
			schedule(FrameInterval)
		case <-retry:
			// Nothing may change on screen after a stall, so the repaint that
			// repairs it cannot wait for new output.
			retry = nil
			if p.stalled && p.View().State == Ready {
				_ = p.repaint(p.life)
			}
			armRetry()
		case <-tick:
			tick = nil
			p.mu.Lock()
			e := p.dirty
			p.dirty = nil
			p.mu.Unlock()
			if e == nil && p.selected != nil {
				deadline := p.selected.NextPublication()
				if !deadline.IsZero() && !time.Now().Before(deadline) {
					e = p.selected
				}
			}
			if e != nil && e == p.selected && p.View().State == Ready {
				_ = p.paintEndpoint(p.life, e, false)
			}
			armRetry()
			if p.selected != nil && p.View().State == Ready {
				if deadline := p.selected.NextPublication(); !deadline.IsZero() {
					schedule(time.Until(deadline))
				}
			}
		}
	}
}
func (p *Presenter) write(ctx context.Context, data []byte, normal bool) error {
	if len(data) > 0 {
		p.parentTouched = true
	}
	budget := time.Now().Add(WriteTimeout)
	deadline, bounded := ctx.Deadline()
	own := !bounded || deadline.After(budget) // which deadline binds this write
	ctx, cancel := context.WithDeadline(ctx, budget)
	defer cancel()
	if normal {
		stop := context.AfterFunc(p.life, cancel)
		defer stop()
	}
	accepted, err := writeComplete(ctx, p.writer, data)
	if err != nil {
		expired := own && errors.Is(err, context.DeadlineExceeded)
		return &WriteFailure{Op: "parent output", Accepted: accepted, Total: len(data), Err: err, Expired: expired}
	}
	return nil
}

func (p *Presenter) fail(err error) error {
	if p.life.Err() != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), WriteTimeout)
	defer cancel()
	cancelErr := p.cancelDrag(ctx)
	cause := errors.Join(err, cancelErr)
	p.failureOnce.Do(func() {
		p.mu.Lock()
		p.failureErr = cause
		p.transitionLocked(ViewEvent{Kind: FailView})
		p.mu.Unlock()
		close(p.failed)
	})
	return cause
}

// stall turns a parent write that ran out of its own budget into a stall rather
// than a failure (#430): a slow terminal is not a gone one, and ending the
// presentation would take every hosted session with it. The abandoned write
// left the parent mid-frame, so what it shows is forgotten: the next paint
// that gets through re-establishes modes and repaints in full. Any other
// write error (EIO, EPIPE: the terminal is gone), or a caller's own deadline,
// still fails.
func (p *Presenter) stall(err error) bool {
	var failure *WriteFailure
	if p.life.Err() != nil || !errors.As(err, &failure) || failure.Op != "parent output" || !failure.Expired {
		return false
	}
	if !p.stalled {
		p.stalls++
	}
	p.stalled = true
	p.confirmedChromeRevision = 0
	p.modesKnown = false
	p.previous = Frame{}
	p.history = HistoryState{AltScreen: p.altOwned}
	return true
}

// probe asks a stalled parent whether it accepts output again. Its bytes are
// harmless in any parser state: CAN/ST abort whatever control the abandoned
// write cut off, and synchronized output closes. A probe that times out
// leaves at most a prefix of that, which the next probe aborts in turn. Only
// a full paint that completes ends the stall, never the probe alone. Only the
// probe's own deadline means "still slow": any other error (EIO, EPIPE) means
// the terminal is gone, and fails.
func (p *Presenter) probe(ctx context.Context) (bool, error) {
	probeCtx, cancel := context.WithTimeout(ctx, StallProbeTimeout)
	defer cancel()
	err := p.write(probeCtx, []byte(stallProbe), true)
	var failure *WriteFailure
	if err == nil {
		return true, nil
	}
	if errors.As(err, &failure) && errors.Is(failure.Err, context.DeadlineExceeded) && ctx.Err() == nil {
		return false, nil
	}
	return false, p.fail(err)
}

// writeUnlessStalled writes a one-shot control (an effect, a clipboard copy).
// While the parent is stalled its bytes are dropped, not queued, and it is no
// error, so no caller ends couch over it; the caller's own bookkeeping (an
// effect's sequence, its attention mark) proceeds either way.
func (p *Presenter) writeUnlessStalled(ctx context.Context, data []byte) error {
	err := p.writeOneShot(ctx, data)
	if errors.Is(err, errParentStalled) {
		return nil
	}
	return err
}

var errParentStalled = errors.New("terminal: parent output stalled; control was not fully written")

// writeOneShot preserves stall recovery but reports a dropped or incomplete
// control to callers that acknowledge output, such as Notify. No control is
// retained for recovery; only the latest screen is repainted.
func (p *Presenter) writeOneShot(ctx context.Context, data []byte) error {
	if p.stalled {
		return errParentStalled
	}
	if err := p.write(ctx, data, true); err != nil {
		if p.stall(err) {
			return errors.Join(errParentStalled, err)
		}
		return p.fail(err)
	}
	return nil
}

// Stalls counts the stalls the parent has recovered from or is in.
func (p *Presenter) Stalls() uint64 {
	var n uint64
	_ = p.call(context.Background(), func(context.Context) error { n = p.stalls; return nil })
	return n
}

// cancelDrag revokes the gesture before enqueueing its one synthetic release.
// A canceled Flush leaves the existing delivery pending; retry only waits for
// that delivery and can never enqueue a second release.
func (p *Presenter) cancelDrag(ctx context.Context) error {
	v := p.View()
	if v.PendingMouseRelease == "" {
		if v.DragDestination == "" {
			return nil
		}
		if p.selected == nil || p.selected.id != v.DragDestination {
			return errors.New("terminal: cancellation destination unavailable")
		}
		if _, err := p.transition(ViewEvent{Kind: CancelMouse}); err != nil {
			return err
		}
		p.cancelTarget = p.selected
		accepted, err := p.cancelTarget.SendMouse(uv.MouseReleaseEvent(p.mouse), p.mouseEpoch)
		if err != nil || !accepted {
			p.settleCancellation()
			return err
		}
	}
	if p.cancelTarget == nil {
		return errors.New("terminal: pending cancellation lost endpoint")
	}
	if err := p.cancelTarget.Flush(ctx); err != nil {
		if errors.Is(err, ErrInputEnded) {
			p.settleCancellation()
			return nil
		}
		return err
	}
	p.settleCancellation()
	return nil
}
func (p *Presenter) settleCancellation() {
	if p.cancelTarget != nil {
		p.transition(ViewEvent{Kind: SettleMouseCancellation, EndpointID: p.cancelTarget.id})
		p.cancelTarget = nil
	}
}

// parentReleaseControls restores the post-presentation baseline even after an
// arbitrary accepted prefix. CAN/ST first abort incomplete CSI/OSC controls,
// then synchronized output closes: a frame write that failed after its bracket
// opened would otherwise hold the parent's display until its own timeout.
// Setup owns mouse, focus, paste and one keyboard-stack push per screen (the
// alternate screen's is released by releaseAlt, before it leaves). The DEC
// modes are terminal-wide, so setup writes them once; the kitty keyboard flags
// are the one setup state a terminal keeps per screen (#279). Render owns
// synchronized output, origin, margins, autowrap, SGR, hyperlinks and cursor
// style/visibility. Its pixels and cursor position remain; one-shot effects
// (including permitted title/clipboard changes) are not rolled back or replayed
// during cleanup.
func parentReleaseControls(keyboardOwned bool) []byte {
	controls := "\x18\x1b\\" + syncEnd
	if keyboardOwned {
		controls += keyboardPop
	}
	controls += "\x1b[?9l\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[?1004l\x1b[?2004l"
	controls += "\x1b[?6l\x1b[r\x1b[?7h\x1b[0m\x1b]8;;\x1b\\\x1b[0 q\x1b[?25h"
	return []byte(controls)
}

type parentModes struct{ tracking int }

func desiredParentModes(policy ParentMousePolicy, child Modes) parentModes {
	if policy == AnyMotion {
		return parentModes{1003}
	}
	return parentModes{child.Tracking}
}

// keyboardOwned skips the push when re-establishing modes after a stall: the
// stack already holds setup's entry, and release pops exactly one.
func parentModeDelta(before, after parentModes, known, keyboardOwned bool) string {
	var s string
	if !known {
		s = "\x1b[?9l\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[?1004h\x1b[?2004h"
		if !keyboardOwned {
			s += keyboardPush
		}
	} else if before.tracking == after.tracking {
		return ""
	} else if before.tracking != 0 {
		s = fmt.Sprintf("\x1b[?%dl\x1b[?1006l", before.tracking)
	}
	if after.tracking != 0 {
		s += fmt.Sprintf("\x1b[?%dh\x1b[?1006h", after.tracking)
	}
	return s
}
func (p *Presenter) paint(ctx context.Context, f Frame, selection bool) error {
	return p.paintPublication(ctx, f, nil, selection)
}
func (p *Presenter) paintPublication(ctx context.Context, f Frame, history *HistoryWindow, selection bool) error {
	if !selection {
		if err := p.reconcileGesture(ctx); err != nil {
			return p.fail(err)
		}
	}
	if err := f.Validate(); err != nil {
		return err
	}
	if p.overlay != nil {
		f = p.overlay(f, p.class)
		if err := f.Validate(); err != nil {
			return err
		}
	}
	// What a tap may see is fixed here; the operator-only layer draws on the
	// operator's copy alone (pair#495).
	tapped := f
	if p.operatorLayer != nil {
		f = p.operatorLayer(f, p.class)
		if err := f.Validate(); err != nil {
			return err
		}
	}
	v := p.View()
	kind := PublishFrame
	if selection {
		kind = SelectView
		v.Token++
	}
	if _, err := p.transition(ViewEvent{Kind: kind, EndpointID: f.EndpointID, Token: v.Token, Generation: f.Generation, GeometryEpoch: f.GeometryEpoch}); err != nil {
		return err
	}
	childModes := Modes{}
	if p.selected != nil {
		childModes = p.selected.Modes()
	}
	desired := desiredParentModes(p.policy, childModes)
	if p.stalled {
		if ok, err := p.probe(ctx); err != nil {
			return err
		} else if !ok {
			// The frame is admitted but not painted; the retry repaints it whole.
			_, err := p.transition(ViewEvent{Kind: PresentView, EndpointID: f.EndpointID, Token: v.Token, Generation: f.Generation, GeometryEpoch: f.GeometryEpoch})
			return err
		}
	}
	delta := parentModeDelta(p.confirmedModes, desired, p.modesKnown, p.keyboardOwned)
	err := p.write(ctx, []byte(delta), true)
	accepted := len(delta)
	var failure *WriteFailure
	if errors.As(err, &failure) {
		accepted = failure.Accepted
	}
	if push := strings.Index(delta, keyboardPush); push >= 0 && accepted >= push+len(keyboardPush) {
		p.keyboardOwned = true
	}
	var nextHistory HistoryState
	if err == nil {
		if history == nil {
			var data []byte
			data, err = Render(p.previous, f)
			if err == nil {
				err = p.write(ctx, data, true)
			}
		} else {
			var rendered HistoryRender
			rendered, err = RenderWithHistory(p.previous, f, *history, p.history)
			if err == nil {
				err = rendered.Emit(func(data []byte) error { return p.writeFramePacket(ctx, data) })
				nextHistory = rendered.NextState()
			}
		}
	}

	if err != nil && !p.stall(err) {
		return p.fail(err)
	}
	painted := err == nil
	if painted {
		p.stalled = false
		p.confirmedModes = desired
		p.modesKnown = true
		p.previous = f.Clone()
		p.confirmedChromeRevision = p.chromeRevision
		if history != nil {
			p.history = nextHistory
		}
	}
	_, err = p.transition(ViewEvent{Kind: PresentView, EndpointID: f.EndpointID, Token: v.Token, Generation: f.Generation, GeometryEpoch: f.GeometryEpoch})
	if err == nil && painted && p.tap != nil {
		p.tap(tapped.Clone(), p.class)
	}
	return err
}
func (p *Presenter) paintEndpoint(ctx context.Context, e *Endpoint, selection bool) error {
	pub, err := e.Publication(time.Now())
	if err != nil {
		return p.fail(err)
	}
	f, err := Compose(pub.Frame, p.host, p.bottom)
	if err != nil {
		return p.fail(err)
	}
	return p.paintPublication(ctx, f, &pub.History, selection)
}
func (p *Presenter) Select(ctx context.Context, e *Endpoint, host Geometry, bottom []Cell) error {
	if e == nil {
		return errors.New("terminal: missing endpoint")
	}
	if err := validateLayout(host, bottom); err != nil {
		return err
	}
	bottom = cloneCells(bottom)
	return p.call(ctx, func(ctx context.Context) error {
		f, err := e.Snapshot(time.Now())
		if err != nil {
			return err
		}
		if f, err = Compose(f, host, bottom); err != nil {
			return err
		}
		if err := p.register(e); err != nil {
			return err
		}
		if err = p.cancelDrag(ctx); err != nil {
			return p.fail(err)
		}
		p.selected = e
		p.class = FramePublic
		p.panel = Frame{}
		p.host = host
		p.bottom = bottom
		p.admitChrome()
		return p.paintEndpoint(ctx, e, true)
	})
}
func (p *Presenter) Present(ctx context.Context, e *Endpoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closing || p.view.State == Failed {
		return presenterRefusal("unavailable", p.failureErr)
	}
	if e == nil || e.id != p.view.Selected {
		return nil
	}
	p.dirty = e
	select {
	case p.wake <- struct{}{}:
	default:
	}
	return nil
}

// Panel presents a compositor-owned surface. class tells a tap whether the
// surface may leave the operator's screen.
func (p *Presenter) Panel(ctx context.Context, f Frame, class FrameClass) error {
	if err := f.Validate(); err != nil {
		return err
	}
	f = f.Clone()
	if f.EndpointID != "" {
		return errors.New("terminal: panel has endpoint")
	}
	return p.call(ctx, func(ctx context.Context) error {
		if err := f.Validate(); err != nil {
			return err
		}
		if err := p.cancelDrag(ctx); err != nil {
			return p.fail(err)
		}
		p.selected = nil
		p.class = class
		p.panel = f
		p.host = f.Geometry
		p.bottom = nil
		p.admitChrome()
		return p.paint(ctx, f, true)
	})
}

// reconcileGesture observes changes even when a child toggled tracking off and
// back on in one output batch. That new protocol generation cannot inherit a
// press admitted under the previous one.
func (p *Presenter) reconcileGesture(ctx context.Context) error {
	v := p.View()
	if v.PendingMouseRelease != "" {
		return p.cancelDrag(ctx)
	}
	if v.Gesture != GestureChild || p.selected == nil || p.selected.Modes().MouseEpoch == p.mouseEpoch {
		return nil
	}
	return p.cancelDrag(ctx)
}
func (p *Presenter) Input(ctx context.Context, event uv.Event) error {
	return p.call(ctx, func(ctx context.Context) error {
		if err := p.reconcileGesture(ctx); err != nil {
			return p.fail(err)
		}
		if mouse, ok := event.(uv.MouseEvent); ok {
			return p.mouseInput(event, mouse.Mouse())
		}
		v := p.View()
		if v.State != Ready || v.Admitted == "" || p.selected == nil {
			return noDestination("no admitted endpoint", v)
		}
		return p.selected.Send(event)
	})
}
func (p *Presenter) mouseInput(event uv.Event, m uv.Mouse) error {
	v := p.View()
	if v.State != Ready {
		return noDestination("no presented mouse destination", v)
	}
	_, release := event.(uv.MouseReleaseEvent)
	if v.Gesture == GestureParent {
		if release {
			_, _ = p.transition(ViewEvent{Kind: ReleaseMouse, Button: int(m.Button)})
		}
		return nil
	}
	inside := p.selected != nil && v.Admitted != "" && m.X >= 0 && m.Y >= 0 && m.X < p.host.Cols && m.Y < p.childRows()
	modes := Modes{}
	if p.selected != nil {
		modes = p.selected.Modes()
	}
	if v.Gesture == GestureNone {
		switch event.(type) {
		case uv.MouseClickEvent:
			if m.Button == uv.MouseNone {
				return nil
			}
			kind := ParentPressMouse
			if inside && p.chromeConfirmed() && modes.Tracking != 0 {
				kind = PressMouse
			}
			if _, err := p.transition(ViewEvent{Kind: kind, Button: int(m.Button)}); err != nil {
				return err
			}
			if kind == ParentPressMouse {
				return nil
			}
			p.mouseEpoch = modes.MouseEpoch
		case uv.MouseMotionEvent:
			if m.Button != uv.MouseNone {
				_, err := p.transition(ViewEvent{Kind: ParentPressMouse, Button: int(m.Button)})
				return err
			}
			if !inside || modes.Tracking != 1003 {
				return nil
			}
		case uv.MouseReleaseEvent:
			return nil
		case uv.MouseWheelEvent:
			if !inside || !p.chromeConfirmed() {
				return nil
			}
		default:
			return nil
		}
	} else {
		switch event.(type) {
		case uv.MouseClickEvent:
			return nil // a second button cannot steal capture
		case uv.MouseMotionEvent:
			if int(m.Button) != v.Button {
				return nil
			}
		case uv.MouseReleaseEvent:
			if m.Button != uv.MouseNone && int(m.Button) != v.Button {
				return nil
			}
			m.Button = uv.MouseButton(v.Button)
		case uv.MouseWheelEvent:
			if !inside || !p.chromeConfirmed() {
				return nil
			}
		default:
			return nil
		}
		if !inside {
			m.X = max(0, min(m.X, p.host.Cols-1))
			m.Y = max(0, min(m.Y, p.childRows()-1))
		}
		switch event.(type) {
		case uv.MouseMotionEvent:
			event = uv.MouseMotionEvent(m)
		case uv.MouseReleaseEvent:
			event = uv.MouseReleaseEvent(m)
		}
	}
	epoch := modes.MouseEpoch
	if p.View().Gesture == GestureChild {
		epoch = p.mouseEpoch
	}
	accepted, err := p.selected.SendMouse(event.(uv.MouseEvent), epoch)
	if err != nil {
		return err
	}
	if !accepted {
		if p.View().Gesture == GestureChild {
			p.transition(ViewEvent{Kind: CancelMouse})
			p.transition(ViewEvent{Kind: SettleMouseCancellation, EndpointID: p.selected.id})
		}
		if release {
			p.transition(ViewEvent{Kind: ReleaseMouse, Button: int(m.Button)})
		}
		return nil
	}
	if _, wheel := event.(uv.MouseWheelEvent); !wheel {
		p.mouse = m
	}
	if release {
		_, err := p.transition(ViewEvent{Kind: ReleaseMouse, Button: int(m.Button)})
		return err
	}
	return nil
}
func (p *Presenter) Resize(ctx context.Context, host Geometry, apply func(Geometry) error) error {
	if err := host.Validate(); err != nil {
		return err
	}
	return p.call(ctx, func(ctx context.Context) error {
		var bottom []Cell
		if len(p.bottom) > 0 {
			reserved := len(p.bottom) / p.host.Cols
			if host.Rows <= reserved {
				return errors.New("terminal: no child rows")
			}
			bottom = make([]Cell, host.Cols*reserved)
		}
		return p.resizeLayout(ctx, host, bottom, apply)
	})
}

// ResizeLayout changes geometry and chrome reservation as one admitted layout.
func (p *Presenter) ResizeLayout(ctx context.Context, host Geometry, bottom []Cell, apply func(Geometry) error) error {
	if err := validateLayout(host, bottom); err != nil {
		return err
	}
	owned := cloneCells(bottom)
	return p.call(ctx, func(ctx context.Context) error { return p.resizeLayout(ctx, host, owned, apply) })
}
func validateLayout(host Geometry, bottom []Cell) error {
	_, err := ChildGeometry(host, bottom)
	return err
}

func (p *Presenter) resizeLayout(ctx context.Context, host Geometry, bottom []Cell, apply func(Geometry) error) error {
	if p.selected == nil {
		return noDestination("resize requires endpoint", p.View())
	}
	if err := validateLayout(host, bottom); err != nil {
		return err
	}
	reserved := len(bottom) / host.Cols
	if err := p.cancelDrag(ctx); err != nil {
		return p.fail(err)
	}
	if err := p.selected.Resize(Geometry{host.Cols, host.Rows - reserved}, apply); err != nil {
		return err
	}
	p.host = host
	p.bottom = bottom
	p.admitChrome()
	return p.paintEndpoint(ctx, p.selected, true)
}
func (p *Presenter) EmitEffects(ctx context.Context, effects []Effect, policy EffectPolicy) ([]Effect, error) {
	if len(effects) > MaxPendingEvents {
		return nil, ErrBackpressure
	}
	for _, e := range effects {
		if len(e.Data) > MaxStringBytes || len(e.Text) > MaxStringBytes || len(e.Selection) > MaxStringBytes {
			return nil, ErrBackpressure
		}
	}
	effects = append([]Effect(nil), effects...)
	for i := range effects {
		effects[i].Data = append([]byte(nil), effects[i].Data...)
	}
	var delivered []Effect
	err := p.call(ctx, func(ctx context.Context) error {
		for _, e := range effects {
			if e.EndpointID == "" || e.Sequence == 0 || !plainText(e.Text) || !plainText(e.Selection) || len(e.Text) > MaxStringBytes || len(e.Data) > MaxStringBytes {
				return errors.New("terminal: invalid effect")
			}
			if p.origins[e.EndpointID] == nil {
				return errors.New("terminal: effect origin is not registered")
			}
			if e.Sequence <= p.effects[e.EndpointID] {
				continue
			}
			if len(p.effects) >= MaxPendingEvents && p.effects[e.EndpointID] == 0 {
				return ErrBackpressure
			}
			var s string
			switch e.Kind {
			case BellEffect:
				if policy.Bell {
					s = "\a"
				}
			case TitleEffect:
				if policy.Title {
					s = "\x1b]2;" + e.Text + "\x1b\\"
				}
			case ClipboardEffect:
				if policy.Clipboard {
					if strings.ContainsAny(e.Selection, ";?") {
						return errors.New("terminal: invalid clipboard selection")
					}
					s = "\x1b]52;" + e.Selection + ";" + base64.StdEncoding.EncodeToString(e.Data) + "\x1b\\"
				}
			case NotificationEffect:
				if policy.Notifications {
					s = "\x1b]777;notify;" + strings.ReplaceAll(e.Selection, ";", ",") + ";" + e.Text + "\x1b\\"
				}
			case DirectoryEffect, SuppressedNotificationEffect:
			default:
				return errors.New("terminal: unknown effect")
			}
			if s != "" {
				if err := p.writeUnlessStalled(ctx, []byte(s)); err != nil {
					return err
				}
			}
			p.effects[e.EndpointID] = e.Sequence
			delivered = append(delivered, e)
		}
		return nil
	})
	return delivered, err
}
func (p *Presenter) Release(ctx context.Context) error {
	p.releaseOnce.Do(func() { p.mu.Lock(); p.closing = true; p.mu.Unlock(); p.stop <- ctx; p.cancel() })
	<-p.done
	return p.releaseErr
}

func (p *Presenter) register(e *Endpoint) error {
	if e == nil {
		return errors.New("terminal: missing origin")
	}
	if old := p.origins[e.id]; old != nil {
		if old != e {
			return errors.New("terminal: reused origin identity")
		}
		return nil
	}
	if len(p.origins) >= MaxPendingEvents {
		return ErrBackpressure
	}
	p.origins[e.id] = e
	return nil
}

// Register reserves an active origin for hidden effects. IDs must identify a
// unique endpoint lifetime; consumers must never reuse an ID after Retire.
func (p *Presenter) Register(ctx context.Context, e *Endpoint) error {
	return p.call(ctx, func(context.Context) error { return p.register(e) })
}

// Retire follows deselection and rejects late effects without keeping tombstones.
func (p *Presenter) Retire(ctx context.Context, e *Endpoint) error {
	return p.call(ctx, func(context.Context) error {
		if e == nil || p.origins[e.id] != e {
			return errors.New("terminal: unknown origin")
		}
		if p.selected == e {
			return errors.New("terminal: deselect before retiring origin")
		}
		delete(p.origins, e.id)
		delete(p.effects, e.id)
		return nil
	})
}

// Flush completes an already requested refresh. It does not end a child's
// synchronized-output hold early; that publication keeps its own deadline.
func (p *Presenter) Flush(ctx context.Context) error {
	return p.call(ctx, func(ctx context.Context) error {
		p.mu.Lock()
		e := p.dirty
		p.dirty = nil
		p.mu.Unlock()
		if e != nil && e == p.selected {
			return p.paintEndpoint(ctx, e, false)
		}
		return nil
	})
}

// UpdateChrome publishes a rectangular reserved surface without changing selection or drag
// ownership. Validation precedes retained-state mutation.
func (p *Presenter) UpdateChrome(ctx context.Context, cells []Cell) error {
	owned := cloneCells(cells)
	return p.call(ctx, func(ctx context.Context) error {
		if p.selected == nil {
			return noDestination("chrome requires selected endpoint", p.View())
		}
		child, err := p.selected.Snapshot(time.Now())
		if err != nil {
			return p.fail(err)
		}
		frame, err := Compose(child, p.host, owned)
		if err != nil {
			return err
		}
		_ = frame
		previousBottom := p.bottom
		p.bottom = owned
		p.admitChrome()
		if err := p.paintEndpoint(ctx, p.selected, false); err != nil {
			p.bottom = previousBottom
			return err
		}
		p.bottom = owned
		return nil
	})
}

// Failed closes when an asynchronous or synchronous presentation fails.
func (p *Presenter) Failed() <-chan struct{} { return p.failed }
func (p *Presenter) Failure() error          { p.mu.Lock(); defer p.mu.Unlock(); return p.failureErr }
func (p *Presenter) childRows() int {
	if len(p.bottom) > 0 {
		return p.host.Rows - len(p.bottom)/p.host.Cols
	}
	return p.host.Rows
}

// Copy emits an application-owned clipboard action through the same parent writer.
func (p *Presenter) Copy(ctx context.Context, data []byte) error {
	if len(data) > MaxStringBytes {
		return errors.New("terminal: clipboard payload exceeds limit")
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return p.call(ctx, func(ctx context.Context) error {
		return p.writeUnlessStalled(ctx, []byte("\x1b]52;c;"+encoded+"\x1b\\"))
	})
}

func (p *Presenter) releaseAlt() []byte {
	if !p.altOwned {
		return nil
	}
	controls := "\x18\x1b\\"
	if p.altKeyboardOwned {
		controls += keyboardPop
	}
	return []byte(controls + altLeave)
}

const (
	stallProbe = "\x18\x1b\\" + syncEnd
	altEnter   = "\x1b[?1049h"
	altLeave   = "\x1b[?1049l"
	// keyboardPush is the presenter's keyboard-stack entry: disambiguate (1)
	// and report event types (2). keyboardPop removes exactly that entry.
	keyboardPush = "\x1b[>3u"
	keyboardPop  = "\x1b[<u"
)

// writeFramePacket writes one packet of a frame, and is the one place the
// parent's screen is accounted for. A screen switch is always a packet of its
// own (HistoryRender.Emit), so a completed switch is recorded even if a later
// chunk of the frame fails, and release leaves only a screen actually entered.
//
// The kitty keyboard flags are a stack PER SCREEN, so setup's push is absent
// on the other screen. There the parent sends legacy keys, and a chord that
// exists only enhanced -- Alt+d, Ctrl+Return -- is dead (#279). So the push
// follows the parent onto the alternate screen and is popped before leaving it:
// each stack is left as it was found.
func (p *Presenter) writeFramePacket(ctx context.Context, data []byte) error {
	switch string(data) {
	case altEnter:
		entered, err := p.writeWhole(ctx, altEnter)
		if entered {
			p.altOwned = true
		}
		if err != nil {
			return err
		}
		p.altKeyboardOwned, err = p.writeWhole(ctx, keyboardPush)
		return err
	case altLeave:
		if p.altKeyboardOwned {
			popped, err := p.writeWhole(ctx, keyboardPop)
			p.altKeyboardOwned = !popped
			if err != nil {
				return err
			}
		}
		left, err := p.writeWhole(ctx, altLeave)
		if left {
			p.altOwned = false
		}
		return err
	}
	return p.write(ctx, data, true)
}

// writeWhole writes one control and reports whether all of it reached the
// parent -- which a write that then failed can still have done. Ownership is
// recorded from that, never from the error alone.
func (p *Presenter) writeWhole(ctx context.Context, control string) (bool, error) {
	err := p.write(ctx, []byte(control), true)
	var failure *WriteFailure
	return !errors.As(err, &failure) || failure.Accepted == len(control), err
}

// Notify emits an application-owned desktop notification through the sole host
// writer. It reports dropped or incomplete output as an error, including a
// recoverable parent stall. Like Copy, it is never replayed on repaint.
func (p *Presenter) Notify(ctx context.Context, title, body string) error {
	encoded, err := notifyosc.EncodeDesktop(title, body)
	if err != nil {
		return err
	}
	return p.call(ctx, func(ctx context.Context) error {
		return p.writeOneShot(ctx, encoded)
	})
}
