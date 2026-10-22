package drive

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// MaxName bounds a viewer's label, in runes.
const MaxName = 40

// releaseWait bounds letting go of what a driver held when control changes.
const releaseWait = 10 * time.Second

// MemController is the Controller (#467): seats in memory, the control
// state machine (recording.NextControl), ctl records into a sink at the
// clock's time, and grace timers. In a cast the sink is the recorder's
// queue, the stream's one writer; a change the sink refuses (the recorder
// hasn't opened: ErrNotLive) leaves control as it was. Seats die after
// grace without a stream, or at Close: nothing outlives the cast.
type MemController struct {
	mu        sync.Mutex
	sink      recording.Sink
	inj       Injector
	clock     func() int64
	grace     time.Duration
	next      int
	seats     map[string]*seat // by viewer id
	keys      map[string]string
	control   recording.Control
	subs      map[chan recording.Control]struct{}
	closed    bool
	authority capability.Snapshot
	capSubs   map[chan capability.Snapshot]struct{}
	reset     func(context.Context, uint64) error
	busy      chan struct{}
	busyErr   error
	cleanupBy string
	resetDone bool
	life      context.Context
	stop      context.CancelFunc
	work      sync.WaitGroup

	drawingEpoch   uint64        // browser-safe sketch turn; ordinary input is unchanged
	drawingWitness chan struct{} // nil while a handoff/reset outcome is unsettled
}

type seat struct {
	Seat
	streams     int
	timer       *time.Timer
	drawingLife chan struct{}
}

// NewMemController: ctl records go to sink at clock(), the driver's input
// to inj; a seat with no stream leaves after grace.
func NewMemController(sink recording.Sink, inj Injector, clock func() int64, grace time.Duration) *MemController {
	ctx, cancel := context.WithCancel(context.Background())
	return &MemController{drawingEpoch: 1, drawingWitness: make(chan struct{}), life: ctx, stop: cancel, capSubs: map[chan capability.Snapshot]struct{}{}, sink: sink, inj: inj, clock: clock, grace: grace,
		seats: map[string]*seat{}, keys: map[string]string{}, subs: map[chan recording.Control]struct{}{}}
}

func (f *MemController) Join(name string) (Seat, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return Seat{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return Seat{}, ErrClosed
	}
	f.next++
	id := fmt.Sprintf("v%d", f.next)
	s := &seat{Seat: Seat{Viewer: recording.Author{ID: id, Name: label(name, f.next)}, Key: hex.EncodeToString(key)}, drawingLife: make(chan struct{})}
	f.seats[id], f.keys[s.Key] = s, id
	f.idleLocked(s) // it must attach a stream within grace
	return s.Seat, nil
}

// label is a viewer's name as shown: trimmed plain text, or "Viewer n".
func label(name string, n int) string {
	// No control, format (bidi overrides, zero-width) or separator
	// characters: a label shows only what it holds, on one line.
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp) {
			return -1
		}
		return r
	}, strings.TrimSpace(name))
	if !utf8.ValidString(name) || name == "" {
		return fmt.Sprintf("Viewer %d", n)
	}
	if r := []rune(name); len(r) > MaxName {
		name = string(r[:MaxName])
	}
	return name
}

func (f *MemController) Attach(viewer string) (func(), error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.seats[viewer]
	if s == nil {
		return nil, ErrNoSeat
	}
	s.streams++
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.seats[viewer] != s {
				return
			}
			if s.streams--; s.streams == 0 {
				f.idleLocked(s)
			}
		})
	}, nil
}

// idleLocked starts s's grace period; when it ends with no stream, s leaves.
func (f *MemController) idleLocked(s *seat) {
	s.timer = time.AfterFunc(f.grace, func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.closed {
			return
		}
		f.work.Add(1)
		defer f.work.Done()
		// A leaving seat must not lose its cleanup because another change is
		// releasing. Wait off the lock, then recheck presence and authority.
		for f.busy != nil && !f.closed {
			done := f.busy
			f.mu.Unlock()
			select {
			case <-done:
			case <-f.life.Done():
			}
			f.mu.Lock()
		}
		if f.seats[s.Viewer.ID] != s || s.streams > 0 || f.closed {
			return
		}
		delete(f.seats, s.Viewer.ID)
		delete(f.keys, s.Key)
		close(s.drawingLife)
		if f.involvedLocked(s.Viewer.ID) && f.authority.Phase != capability.Disabling {
			if err := f.changeLocked(recording.CtlLeave, s.Viewer); err != nil {
				f.freezeLocked(err, s.Viewer.ID)
			}
		}

	})
}

func (f *MemController) involvedLocked(id string) bool {
	if f.control.Driving() == id {
		return true
	}
	for _, p := range f.control.Pending {
		if p.ID == id {
			return true
		}
	}
	return false
}

// changeLocked applies op, puts its ctl record, and tells subscribers.
func (f *MemController) changeLocked(op string, who recording.Author) error {
	if f.closed {
		return ErrClosed
	}
	if f.busy != nil || f.authority.Phase == capability.Disabling {
		return ErrDriveUnavailable
	}
	if (op == recording.CtlRequest || op == recording.CtlGrant) && f.authority.Phase != capability.On {
		return ErrDriveUnavailable
	}
	next, err := recording.NextControl(f.control, op, who)
	if err != nil {
		return err
	}
	r := &recording.Ctl{Op: op, Who: who, Control: next}
	old := f.control.Driving()
	holderChanged := next.Driving() != old || op == recording.CtlReset
	if holderChanged {
		f.beginDrawingTransitionLocked()
	}
	f.busy = make(chan struct{})
	f.busyErr = nil
	f.work.Add(1)
	ctx, cancel := context.WithTimeout(f.life, releaseWait)
	f.mu.Unlock()
	r.SetTime(f.clock())
	var putErr error
	if writer, ok := f.sink.(ControlWriter); ok {
		putErr = writer.WriteControl(ctx, r)
	} else {
		putErr = f.sink.Put(r)
	}
	var releaseErr error
	if putErr == nil && next.Driving() != old && old != "" {
		if rel, ok := f.inj.(Releaser); ok {
			releaseErr = rel.Release(ctx, old)
		}
	}
	cancel()
	f.mu.Lock()
	defer func() { close(f.busy); f.busy = nil; f.work.Done() }()
	if putErr != nil {
		// A definite refusal commits nothing. An unknown sink result can have
		// advanced the writer even though local control could not acknowledge it.
		if !errors.Is(putErr, ErrNotLive) && capability.FailureOutcome(putErr) != capability.Retryable {
			f.busyErr = putErr
			f.freezeLocked(putErr, old)
		}
		return putErr
	}
	// The ordered control record has committed, even when its ensuing release
	// failed. Publish that evidence, but never reopen admission on failure.
	if !f.closed {
		f.control = next
		for ch := range f.subs {
			offer(ch, next)
		}
	}
	if releaseErr != nil {
		f.busyErr = releaseErr
		f.freezeLocked(releaseErr, old)
		return releaseErr
	}
	if f.closed {
		return ErrClosed
	}
	if holderChanged {
		f.renewDrawingLocked()
	}

	return nil
}

// offer replaces whatever ch holds with c, never blocking.
func offer(ch chan recording.Control, c recording.Control) {
	select {
	case <-ch:
	default:
	}
	ch <- c
}

func (f *MemController) bySeat(key string) (*seat, error) {
	if s := f.seats[f.keys[key]]; s != nil && key != "" {
		return s, nil
	}
	return nil, ErrNoSeat
}

func (f *MemController) Request(key string) error {
	a, err := f.AdmitDrive()
	if err != nil {
		return err
	}
	return a.Request(key)
}
func (f *MemController) Release(key string) error { return f.viewerOp(key, recording.CtlRelease) }

func (f *MemController) viewerOp(key, op string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, err := f.bySeat(key)
	if err != nil {
		return err
	}
	return f.changeLocked(op, s.Viewer)
}

// Input checks the seat drives, then injects without holding the lock, so a
// Revoke never waits on a batch. A batch overtaken by a revoke is refused
// where it is dispatched: its records would name a viewer who no longer
// drives.
func (f *MemController) Input(ctx context.Context, key string, b Batch) error {
	a, err := f.AdmitDrive()
	if err != nil {
		return err
	}
	return a.Input(ctx, key, b)
}

func (f *MemController) Grant(viewer string) error { return f.operatorOp(viewer, recording.CtlGrant) }
func (f *MemController) Deny(viewer string) error  { return f.operatorOp(viewer, recording.CtlDeny) }

func (f *MemController) operatorOp(viewer, op string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.seats[viewer]
	if s == nil {
		return ErrNoSeat
	}
	return f.changeLocked(op, s.Viewer)
}

func (f *MemController) Revoke() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.control.Driver == nil || f.closed {
		return nil
	}
	return f.changeLocked(recording.CtlRevoke, *f.control.Driver)
}

func (f *MemController) Control() recording.Control {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.control
}

func (f *MemController) Subscribe() (<-chan recording.Control, func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan recording.Control, 1)
	ch <- f.control
	if f.closed {
		close(ch)
		return ch, func() {}
	}
	f.subs[ch] = struct{}{}
	return ch, func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		if _, ok := f.subs[ch]; ok {
			delete(f.subs, ch)
			close(ch)
		}
	}
}

func (f *MemController) Close() error {
	f.mu.Lock()
	if !f.closed {
		f.beginDrawingTransitionLocked()
		f.closed = true
		f.stop()
		f.authority.CapabilityState, _ = capability.NextCapability(f.authority.CapabilityState, capability.Event{Kind: capability.End})
		f.publishCapabilityLocked()
		for _, s := range f.seats {
			close(s.drawingLife)
			if s.timer != nil {
				s.timer.Stop()
			}
		}
		f.seats, f.keys = map[string]*seat{}, map[string]string{}
		for ch := range f.subs {
			close(ch)
		}
		f.subs = nil
		for ch := range f.capSubs {
			close(ch)
		}
		f.capSubs = nil
	}
	f.mu.Unlock()
	f.work.Wait()
	return nil
}
