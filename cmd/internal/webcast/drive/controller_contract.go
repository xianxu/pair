package drive

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// spyInjector records what reaches it; while hold is set, Inject waits for
// it to close.
type spyInjector struct {
	mu      sync.Mutex
	calls   []string // by
	hold    chan struct{}
	entered chan struct{}
}

func (s *spyInjector) Inject(ctx context.Context, by string, b Batch) error {
	s.mu.Lock()
	s.calls = append(s.calls, by)
	hold, entered := s.hold, s.entered
	s.mu.Unlock()
	if hold != nil {
		close(entered)
		<-hold
	}
	return nil
}

func (s *spyInjector) count() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.calls) }
func (s *spyInjector) last() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.calls) == 0 {
		return ""
	}
	return s.calls[len(s.calls)-1]
}

// RunControllerContract checks a Controller. newController builds one whose
// ctl records go to sink at clock(), whose driver's input goes to inj, and
// whose seats leave after grace without a stream. The suite owns sink (a
// live page already in it) and checks that what was put validates.
func RunControllerContract(t recording.TB, newController func(sink recording.Sink, inj Injector, clock func() int64, grace time.Duration) Controller) {
	t.Helper()
	const grace = 30 * time.Millisecond
	sink := &recording.MemSink{}
	for _, r := range DrivenPage(100) { // live, nobody driving yet
		sink.Put(r)
	}
	var now atomic.Int64
	now.Store(200)
	clock := func() int64 { return now.Add(1) }
	spy := &spyInjector{}
	c := newController(sink, spy, clock, grace)
	ctx := context.Background()
	states, _ := c.Subscribe()
	batch := Batch{Seg: 1, Acts: []Act{{A: ActText, V: "x"}}}
	driving := func() string { return c.Control().Driving() }
	ctls := func() int {
		n := 0
		for _, r := range sink.Records() {
			if _, ok := r.(*recording.Ctl); ok {
				n++
			}
		}
		return n
	}
	must := func(what string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	join := func(name string) (Seat, func()) {
		t.Helper()
		s, err := c.Join(name)
		must("join", err)
		detach, err := c.Attach(s.Viewer.ID)
		must("attach", err)
		return s, detach
	}

	ana, anaGone := join("Ana")
	bo, boGone := join("Bo")
	if ana.Viewer.ID == "" || ana.Viewer.ID == bo.Viewer.ID || len(ana.Key) < 32 || ana.Key == bo.Key || ana.Viewer.Name == "" {
		t.Fatalf("seats must have distinct ids and long distinct keys: %+v %+v", ana, bo)
	}
	if _, err := c.Attach("nobody"); !errors.Is(err, ErrNoSeat) {
		t.Errorf("attach an unknown viewer: %v", err)
	}
	if err := c.Request("not-a-key"); !errors.Is(err, ErrNoSeat) {
		t.Errorf("request with a bad key: %v", err)
	}
	if err := c.Grant("nobody"); !errors.Is(err, ErrNoSeat) {
		t.Errorf("grant an unknown viewer: %v", err)
	}
	if err := c.Input(ctx, ana.Key, batch); !errors.Is(err, ErrNotDriver) || spy.count() != 0 {
		t.Errorf("input before a grant: %v, %d injected", err, spy.count())
	}

	must("request", c.Request(ana.Key))
	if err := c.Request(ana.Key); !errors.Is(err, recording.ErrControl) {
		t.Errorf("request twice: %v", err)
	}
	must("grant", c.Grant(ana.Viewer.ID))
	must("input", c.Input(ctx, ana.Key, batch))
	if driving() != ana.Viewer.ID || spy.last() != ana.Viewer.ID {
		t.Errorf("after grant: driving %q, injected as %q", driving(), spy.last())
	}
	if err := c.Input(ctx, bo.Key, batch); !errors.Is(err, ErrNotDriver) {
		t.Errorf("input from a viewer who doesn't drive: %v", err)
	}

	// A grant to another viewer transfers control.
	must("request", c.Request(bo.Key))
	must("grant", c.Grant(bo.Viewer.ID))
	if err := c.Input(ctx, ana.Key, batch); driving() != bo.Viewer.ID || !errors.Is(err, ErrNotDriver) {
		t.Errorf("after the transfer: driving %q, the old driver's input %v", driving(), err)
	}
	must("release", c.Release(bo.Key))
	// Revoke with nobody driving does nothing, and says so by not failing.
	before := ctls()
	must("revoke with nobody driving", c.Revoke())
	if ctls() != before || driving() != "" {
		t.Errorf("revoke with nobody driving wrote a record or changed control")
	}
	must("request", c.Request(ana.Key))
	must("deny", c.Deny(ana.Viewer.ID))
	if len(c.Control().Pending) != 0 {
		t.Errorf("after deny: pending %v", c.Control().Pending)
	}

	// The operator takes control back even while the driver's batch is
	// being dispatched.
	must("request", c.Request(ana.Key))
	must("grant", c.Grant(ana.Viewer.ID))
	spy.mu.Lock()
	spy.hold, spy.entered = make(chan struct{}), make(chan struct{})
	hold, entered := spy.hold, spy.entered
	spy.mu.Unlock()
	go c.Input(ctx, ana.Key, batch)
	<-entered
	revoked := make(chan error, 1)
	go func() { revoked <- c.Revoke() }()
	select {
	case err := <-revoked:
		if err != nil || driving() != "" {
			t.Errorf("revoke mid-batch: %v, driving %q", err, driving())
		}
	case <-time.After(time.Second):
		t.Errorf("revoke waited on the driver's batch")
	}
	spy.mu.Lock()
	spy.hold = nil
	spy.mu.Unlock()
	close(hold)

	// A driver whose stream is gone for the grace period leaves, and
	// control comes back to the operator with a leave record.
	must("request", c.Request(ana.Key))
	must("grant", c.Grant(ana.Viewer.ID))
	anaGone()
	anaGone() // idempotent
	if !eventually(func() bool { return driving() == "" }) {
		t.Errorf("a gone driver still drives after the grace period")
	}
	if err := c.Request(ana.Key); !errors.Is(err, ErrNoSeat) {
		t.Errorf("a seat that left still answers: %v", err)
	}
	// A stream that comes back within the grace period keeps the seat.
	boGone()
	boAgain, err := c.Attach(bo.Viewer.ID)
	must("re-attach", err)
	must("request", c.Request(bo.Key))
	must("grant", c.Grant(bo.Viewer.ID))
	time.Sleep(4 * grace)
	if driving() != bo.Viewer.ID {
		t.Errorf("a viewer who reconnected within the grace period lost control")
	}
	// A seat that never attaches leaves too.
	cy, err := c.Join("Cy")
	must("join", err)
	if !eventually(func() bool { return errors.Is(c.Request(cy.Key), ErrNoSeat) }) {
		t.Errorf("a seat that never attached a stream didn't leave")
	}

	// Subscribers see the latest state.
	var latest recording.Control
	if !eventually(func() bool {
		for {
			select {
			case s := <-states:
				latest = s
				continue
			default:
			}
			return latest.Equal(c.Control())
		}
	}) {
		t.Errorf("subscriber's latest state is %q, control is %q", latest.Driving(), driving())
	}

	// What was put validates as a whole recording: every ctl states what
	// its op gives.
	files := map[string][]byte{}
	for _, r := range sink.Records() {
		line, err := recording.Encode(r)
		must("encode", err)
		f := recording.FileOf(r.Kind())
		files[f] = append(append(files[f], line...), '\n')
	}
	if fs, _ := recording.Validate(files); len(recording.Errors(fs)) > 0 {
		t.Errorf("the controller's records don't validate: %v", recording.Errors(fs))
	}

	boAgain()
	must("close", c.Close())
	must("close twice", c.Close())
	if _, open := <-drainControl(states); open {
		t.Errorf("a subscription stayed open after Close")
	}
	if _, err := c.Join("late"); !errors.Is(err, ErrClosed) {
		t.Errorf("join after close: %v", err)
	}
}

// eventually polls ok for up to a second.
func eventually(ok func() bool) bool {
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if ok() {
			return true
		}
	}
	return ok()
}

func drainControl(ch <-chan recording.Control) <-chan recording.Control {
	out := make(chan recording.Control)
	go func() {
		for range ch {
		}
		close(out)
	}()
	return out
}
