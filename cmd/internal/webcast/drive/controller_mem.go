package drive

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// MaxName bounds a viewer's label, in runes.
const MaxName = 40

// MemController is the Controller (#467): seats in memory, the control
// state machine (recording.NextControl), ctl records into a sink at the
// clock's time, and grace timers. In a cast the sink is the recorder's
// queue, the stream's one writer; a change the sink refuses (the recorder
// hasn't opened: ErrNotLive) leaves control as it was. Seats die after
// grace without a stream, or at Close: nothing outlives the cast.
type MemController struct {
	mu      sync.Mutex
	sink    recording.Sink
	inj     Injector
	clock   func() int64
	grace   time.Duration
	next    int
	seats   map[string]*seat // by viewer id
	keys    map[string]string
	control recording.Control
	subs    map[chan recording.Control]struct{}
	closed  bool
}

type seat struct {
	Seat
	streams int
	timer   *time.Timer
}

// NewMemController: ctl records go to sink at clock(), the driver's input
// to inj; a seat with no stream leaves after grace.
func NewMemController(sink recording.Sink, inj Injector, clock func() int64, grace time.Duration) *MemController {
	return &MemController{sink: sink, inj: inj, clock: clock, grace: grace,
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
	s := &seat{Seat: Seat{Viewer: recording.Author{ID: id, Name: label(name, f.next)}, Key: hex.EncodeToString(key)}}
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
		if f.seats[s.Viewer.ID] != s || s.streams > 0 || f.closed {
			return
		}
		delete(f.seats, s.Viewer.ID)
		delete(f.keys, s.Key)
		if f.involvedLocked(s.Viewer.ID) {
			f.changeLocked(recording.CtlLeave, s.Viewer) // a sink error leaves control as it was; the seat is gone
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
	next, err := recording.NextControl(f.control, op, who)
	if err != nil {
		return err
	}
	r := &recording.Ctl{Op: op, Who: who, Control: next}
	r.SetTime(f.clock())
	if err := f.sink.Put(r); err != nil {
		return err
	}
	f.control = next
	for ch := range f.subs {
		offer(ch, next)
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

func (f *MemController) Request(key string) error { return f.viewerOp(key, recording.CtlRequest) }
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
	f.mu.Lock()
	s, err := f.bySeat(key)
	if err == nil && f.control.Driving() != s.Viewer.ID {
		err = ErrNotDriver
	}
	f.mu.Unlock()
	if err != nil {
		return err
	}
	return f.inj.Inject(ctx, s.Viewer.ID, b)
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
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	for _, s := range f.seats {
		if s.timer != nil {
			s.timer.Stop()
		}
	}
	f.seats, f.keys = map[string]*seat{}, map[string]string{}
	for ch := range f.subs {
		close(ch)
	}
	f.subs = nil
	return nil
}
