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
	releaseOnce    sync.Once
	releaseErr     error
	dirty          *Endpoint
	selected       *Endpoint
	host           Geometry
	bottom         []Cell
	previous       Frame
	mouse          uv.Mouse
	effects        map[string]uint64
	origins        map[string]*Endpoint
	confirmedModes parentModes
	keyboardOwned  bool
	modesKnown     bool
}

func NewPresenter(w ttyio.Writer, policy ParentMousePolicy) *Presenter {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Presenter{writer: w, policy: policy, requests: make(chan presentRequest, MaxPendingEvents), wake: make(chan struct{}, 1), stop: make(chan context.Context, 1), done: make(chan struct{}), life: ctx, cancel: cancel, effects: make(map[string]uint64), origins: make(map[string]*Endpoint)}
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
			p.cancelDrag(ctx)
			cleanup := "\x18\x1b\\"
			if p.keyboardOwned {
				cleanup += "\x1b[<u"
			}
			p.releaseErr = p.write(ctx, []byte(cleanup+"\x1b[?9l\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[?1004l\x1b[?2004l\x1b[0m\x1b[r\x1b[?25h"), false)
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
	ctx, cancel := context.WithTimeout(ctx, WriteTimeout)
	defer cancel()
	if normal {
		stop := context.AfterFunc(p.life, cancel)
		defer stop()
	}
	accepted, err := writeComplete(ctx, p.writer, data)
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
	_ = p.cancelDrag(ctx)
	p.transition(ViewEvent{Kind: FailView})
	return err
}
func (p *Presenter) cancelDrag(ctx context.Context) error {
	v := p.View()
	if v.DragDestination == "" || p.selected == nil {
		return nil
	}
	if err := p.selected.Send(uv.MouseReleaseEvent(p.mouse)); err != nil {
		return err
	}
	return p.selected.Flush(ctx)
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
	data, err := Render(p.previous, f)
	if err == nil {
		delta := parentModeDelta(p.confirmedModes, desired, p.modesKnown)
		packet := append([]byte(delta), data...)
		err = p.write(ctx, packet, true)
		accepted := len(packet)
		var failure *WriteFailure
		if errors.As(err, &failure) {
			accepted = failure.Accepted
		}
		if push := strings.Index(delta, "\x1b[>3u"); push >= 0 && accepted >= push+len("\x1b[>3u") {
			p.keyboardOwned = true
		}
	}
	if err != nil {
		return p.fail(err)
	}
	p.confirmedModes = desired
	p.modesKnown = true
	p.previous = f.Clone()
	_, err = p.transition(ViewEvent{Kind: PresentView, EndpointID: f.EndpointID, Token: v.Token, Generation: f.Generation, GeometryEpoch: f.GeometryEpoch})
	return err
}
func (p *Presenter) paintEndpoint(ctx context.Context, e *Endpoint, selection bool) error {
	f, err := e.Snapshot(time.Now())
	if err != nil {
		return p.fail(err)
	}
	f, err = Compose(f, p.host, p.bottom)
	if err != nil {
		return p.fail(err)
	}
	return p.paint(ctx, f, selection)
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
		return p.paint(ctx, f, true)
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
func (p *Presenter) Input(ctx context.Context, event uv.Event) error {
	return p.call(ctx, func(ctx context.Context) error {
		v := p.View()
		if _, release := event.(uv.MouseReleaseEvent); release && v.SuppressDrag {
			p.transition(ViewEvent{Kind: ReleaseMouse})
			return nil
		}
		if v.State != Ready || v.Admitted == "" || p.selected == nil {
			return errors.New("terminal: no admitted endpoint")
		}
		if ev, ok := event.(uv.MouseEvent); ok {
			m := ev.Mouse()
			_, release := event.(uv.MouseReleaseEvent)
			if v.SuppressDrag {
				if release {
					p.transition(ViewEvent{Kind: ReleaseMouse})
				}
				return nil
			}
			outside := m.X < 0 || m.Y < 0 || m.X >= p.host.Cols || m.Y >= p.host.Rows-1
			if outside && v.DragDestination == "" {
				return nil
			}
			if outside {
				m.X = max(0, min(m.X, p.host.Cols-1))
				m.Y = max(0, min(m.Y, p.host.Rows-2))
				switch event.(type) {
				case uv.MouseReleaseEvent:
					event = uv.MouseReleaseEvent(m)
				case uv.MouseMotionEvent:
					event = uv.MouseMotionEvent(m)
				default:
					return nil
				}
			}
			if _, click := event.(uv.MouseClickEvent); click && p.selected.Modes().Tracking != 0 {
				if _, err := p.transition(ViewEvent{Kind: PressMouse}); err != nil {
					return err
				}
			}
			p.mouse = m
			if release {
				p.transition(ViewEvent{Kind: ReleaseMouse})
			}
		}
		return p.selected.Send(event)
	})
}
func (p *Presenter) Resize(ctx context.Context, host Geometry, apply func(Geometry) error) error {
	return p.call(ctx, func(ctx context.Context) error {
		if p.selected == nil {
			return errors.New("terminal: resize requires endpoint")
		}
		if err := host.Validate(); err != nil {
			return err
		}
		if host.Rows < 2 {
			return errors.New("terminal: no child rows")
		}
		if err := p.cancelDrag(ctx); err != nil {
			return p.fail(err)
		}
		if err := p.selected.Resize(Geometry{host.Cols, host.Rows - 1}, apply); err != nil {
			return err
		}
		p.host = host
		p.bottom = make([]Cell, host.Cols)
		return p.paintEndpoint(ctx, p.selected, true)
	})
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
		if err := p.paint(ctx, frame, false); err != nil {
			return err
		}
		p.bottom = owned
		return nil
	})
}
