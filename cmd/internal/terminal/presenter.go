package terminal

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"strings"
	"sync"
	"time"
)

type ParentMousePolicy uint8

const (
	CouchAnyMotion ParentMousePolicy = iota
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
	keyboardOwned  bool
	altOwned       bool
	history        HistoryState
	modesKnown     bool
	parentTouched  bool // actor-owned; even an interrupted first write requires release
}

func NewPresenter(w ttyio.Writer, policy ParentMousePolicy) *Presenter {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Presenter{failed: make(chan struct{}), writer: w, policy: policy, requests: make(chan presentRequest, MaxPendingEvents), wake: make(chan struct{}, 1), stop: make(chan context.Context, 1), done: make(chan struct{}), life: ctx, cancel: cancel, effects: make(map[string]uint64), origins: make(map[string]*Endpoint)}
	go p.run()
	return p
}
func (p *Presenter) View() View { p.mu.Lock(); defer p.mu.Unlock(); return p.view }
func (p *Presenter) transition(e ViewEvent) (ViewEffects, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, out, err := Transition(p.view, e)
	if err == nil {
		p.view = v
	}
	return out, err
}
func (p *Presenter) call(ctx context.Context, f func(context.Context) error) error {
	r := presentRequest{ctx: ctx, run: f, done: make(chan error, 1)}
	p.mu.Lock()
	if p.closing || p.view.State == Failed {
		p.mu.Unlock()
		return errors.New("terminal: presenter unavailable")
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
		return errors.New("terminal: presenter released")
	}
}
func (p *Presenter) run() {
	defer close(p.done)
	var timer *time.Timer
	var tick <-chan time.Time
	var scheduled time.Time
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
			cancelErr := p.cancelDrag(ctx)
			p.releaseErr = cancelErr
			if p.parentTouched {
				p.releaseErr = errors.Join(cancelErr, p.write(ctx, append(p.releaseAlt(), parentReleaseControls(p.keyboardOwned)...), false))
			}
			p.transition(ViewEvent{Kind: ReleaseView})
			return
		case r := <-p.requests:
			if p.View().State == Failed {
				r.done <- errors.New("terminal: presenter failed")
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
			if p.selected != nil && p.View().State == Ready {
				if deadline := p.selected.NextPublication(); !deadline.IsZero() {
					schedule(time.Until(deadline))
				}
			}
		case <-p.wake:
			schedule(FrameInterval)
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
	ctx, cancel := context.WithTimeout(ctx, WriteTimeout)
	defer cancel()
	if normal {
		stop := context.AfterFunc(p.life, cancel)
		defer stop()
	}
	accepted, err := writeComplete(ctx, p.writer, data)
	if accepted == len(data) {
		if string(data) == "\x1b[?1049h" {
			p.altOwned = true
		}
		if string(data) == "\x1b[?1049l" {
			p.altOwned = false
		}
	}
	if err != nil {
		return &WriteFailure{Accepted: accepted, Total: len(data), Err: err}
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
	p.transition(ViewEvent{Kind: FailView})
	p.failureOnce.Do(func() { p.mu.Lock(); p.failureErr = errors.Join(err, cancelErr); p.mu.Unlock(); close(p.failed) })
	return errors.Join(err, cancelErr)
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
// arbitrary accepted prefix. CAN/ST first abort incomplete CSI/OSC controls.
// Setup owns mouse, focus, paste and one keyboard-stack push. Render owns
// origin, margins, autowrap, SGR, hyperlinks and cursor style/visibility. Its
// pixels and cursor position remain; one-shot effects (including permitted
// title/clipboard changes) are not rolled back or replayed during cleanup.
func parentReleaseControls(keyboardOwned bool) []byte {
	controls := "\x18\x1b\\"
	if keyboardOwned {
		controls += "\x1b[<u"
	}
	controls += "\x1b[?9l\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[?1004l\x1b[?2004l"
	controls += "\x1b[?6l\x1b[r\x1b[?7h\x1b[0m\x1b]8;;\x1b\\\x1b[0 q\x1b[?25h"
	return []byte(controls)
}

type parentModes struct{ tracking int }

func desiredParentModes(policy ParentMousePolicy, child Modes) parentModes {
	if policy == CouchAnyMotion {
		return parentModes{1003}
	}
	return parentModes{child.Tracking}
}
func parentModeDelta(before, after parentModes, known bool) string {
	var s string
	if !known {
		s = "\x1b[?9l\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[?1004h\x1b[?2004h\x1b[>3u"
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
	delta := parentModeDelta(p.confirmedModes, desired, p.modesKnown)
	err := p.write(ctx, []byte(delta), true)
	accepted := len(delta)
	var failure *WriteFailure
	if errors.As(err, &failure) {
		accepted = failure.Accepted
	}
	if push := strings.Index(delta, "\x1b[>3u"); push >= 0 && accepted >= push+len("\x1b[>3u") {
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
				err = rendered.Emit(func(data []byte) error { return p.write(ctx, data, true) })
				nextHistory = rendered.NextState()
			}
		}
	}

	if err != nil {
		return p.fail(err)
	}
	p.confirmedModes = desired
	p.modesKnown = true
	p.previous = f.Clone()
	if history != nil {
		p.history = nextHistory
	}
	_, err = p.transition(ViewEvent{Kind: PresentView, EndpointID: f.EndpointID, Token: v.Token, Generation: f.Generation, GeometryEpoch: f.GeometryEpoch})
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
	bottom = append([]Cell(nil), bottom...)
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
		p.host = host
		p.bottom = bottom
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
		return errors.New("terminal: presenter unavailable")
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
func (p *Presenter) Panel(ctx context.Context, f Frame) error {
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
		p.host = f.Geometry
		p.bottom = nil
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
			return errors.New("terminal: no admitted endpoint")
		}
		return p.selected.Send(event)
	})
}
func (p *Presenter) mouseInput(event uv.Event, m uv.Mouse) error {
	v := p.View()
	if v.State != Ready {
		return errors.New("terminal: no presented mouse destination")
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
			if inside && modes.Tracking != 0 {
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
			if !inside {
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
			if !inside {
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
	return p.call(ctx, func(ctx context.Context) error {
		var bottom []Cell
		if len(p.bottom) > 0 {
			bottom = make([]Cell, host.Cols)
		}
		return p.resizeLayout(ctx, host, bottom, apply)
	})
}

// ResizeLayout changes geometry and chrome reservation as one admitted layout.
func (p *Presenter) ResizeLayout(ctx context.Context, host Geometry, bottom []Cell, apply func(Geometry) error) error {
	owned := make([]Cell, len(bottom))
	for i, c := range bottom {
		owned[i] = cloneCell(c)
	}
	return p.call(ctx, func(ctx context.Context) error { return p.resizeLayout(ctx, host, owned, apply) })
}
func (p *Presenter) resizeLayout(ctx context.Context, host Geometry, bottom []Cell, apply func(Geometry) error) error {
	if p.selected == nil {
		return errors.New("terminal: resize requires endpoint")
	}
	if err := host.Validate(); err != nil {
		return err
	}
	reserved := 0
	if len(bottom) > 0 {
		reserved = 1
		if len(bottom) != host.Cols {
			return errors.New("terminal: chrome width mismatch")
		}
	}
	if host.Rows <= reserved {
		return errors.New("terminal: no child rows")
	}
	// Validate chrome before mutating child or physical geometry.
	chromeFrame := Frame{Geometry: Geometry{Cols: host.Cols, Rows: 1}, Cells: bottom}
	if reserved > 0 {
		if err := chromeFrame.Validate(); err != nil {
			return err
		}
	}
	if err := p.cancelDrag(ctx); err != nil {
		return p.fail(err)
	}
	if err := p.selected.Resize(Geometry{host.Cols, host.Rows - reserved}, apply); err != nil {
		return err
	}
	p.host = host
	p.bottom = bottom
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
			case DirectoryEffect:
			default:
				return errors.New("terminal: unknown effect")
			}
			if err := p.write(ctx, []byte(s), true); err != nil {
				return p.fail(err)
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

// Register reserves an active origin for hidden effects. IDs must identify a
// unique endpoint lifetime; consumers must never reuse an ID after Retire.
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

// UpdateChrome publishes a new reserved row without changing selection or drag
// ownership. Validation precedes retained-state mutation.
func (p *Presenter) UpdateChrome(ctx context.Context, cells []Cell) error {
	owned := make([]Cell, len(cells))
	for i, c := range cells {
		owned[i] = cloneCell(c)
	}
	return p.call(ctx, func(ctx context.Context) error {
		if p.selected == nil {
			return errors.New("terminal: chrome requires selected endpoint")
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
		return p.host.Rows - 1
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
		if err := p.write(ctx, []byte("\x1b]52;c;"+encoded+"\x1b\\"), true); err != nil {
			return p.fail(err)
		}
		return nil
	})
}

func (p *Presenter) releaseAlt() []byte {
	if p.altOwned {
		return []byte("\x18\x1b\\\x1b[?1049l")
	}
	return nil
}
