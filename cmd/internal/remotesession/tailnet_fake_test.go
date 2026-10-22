package remotesession

import (
	"context"
	"errors"
	"io"
	"net/netip"
	"sync"
)

var errFakeTailnetUnavailable = errors.New("fake tailnet unavailable")

// fakeTailnetNode is a stateful IO model, not security or native API evidence.
// It never starts networking, owns keys, uses a host listener, or spawns children.
type fakeTailnetNode struct {
	mu                         sync.Mutex
	started, closing, closed   bool
	status                     tailnetNodeStatus
	peers                      map[netip.AddrPort]tailnetObservation
	lookupErr                  error
	lookups                    int
	watch                      *fakeTailnetWatch
	owners, watches            int
	startGate, closeGate       chan struct{}
	startEntered, closeEntered chan struct{}
	startErr, closeErr         error
	closeDone                  chan struct{}
	startDone, startCancel     chan struct{}
	beforeStatusPublish        func(tailnetNodeStatus)
}
type fakeTailnetCounts struct{ owners, watches, lookups int }

func newFakeTailnetNode() *fakeTailnetNode {
	return &fakeTailnetNode{peers: make(map[netip.AddrPort]tailnetObservation), startEntered: make(chan struct{}), closeEntered: make(chan struct{}), closeDone: make(chan struct{}), startDone: make(chan struct{}), startCancel: make(chan struct{})}
}

var _ tailnetNode = (*fakeTailnetNode)(nil)

func (f *fakeTailnetNode) Start(ctx context.Context) error {
	f.mu.Lock()
	if f.started || f.closing || f.closed {
		f.mu.Unlock()
		return errFakeTailnetUnavailable
	}
	if err := ctx.Err(); err != nil {
		f.mu.Unlock()
		return err
	}
	f.started = true
	f.owners = 1
	gate, err := f.startGate, f.startErr
	close(f.startEntered)
	f.mu.Unlock()
	defer close(f.startDone)
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return ctx.Err()
		case <-f.startCancel:
			return errFakeTailnetUnavailable
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closing || f.closed {
		return errFakeTailnetUnavailable
	}
	if err != nil {
		return err
	}
	return ctx.Err()
}
func (f *fakeTailnetNode) Status(ctx context.Context) (tailnetNodeStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return tailnetNodeStatus{}, err
	}
	if f.closed {
		return tailnetNodeStatus{phase: tailnetNodeClosed}, nil
	}
	if !f.started || f.closing {
		return tailnetNodeStatus{}, errFakeTailnetUnavailable
	}
	return f.status, nil
}
func (f *fakeTailnetNode) WhoIs(ctx context.Context, addr netip.AddrPort) (tailnetObservation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return tailnetObservation{}, err
	}
	if !f.started || f.closing || f.closed || f.status.phase != tailnetNodeRunning {
		return tailnetObservation{}, errFakeTailnetUnavailable
	}
	f.lookups++
	if f.lookupErr != nil {
		return tailnetObservation{}, f.lookupErr
	}
	o, ok := f.peers[addr]
	if !ok {
		return tailnetObservation{}, errFakeTailnetUnavailable
	}
	return o, nil
}
func (f *fakeTailnetNode) Watch(ctx context.Context) (tailnetWatch, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !f.started || f.closing || f.closed || f.watch != nil {
		return nil, errFakeTailnetUnavailable
	}
	w := &fakeTailnetWatch{owner: f, ctx: ctx, done: make(chan struct{}), events: make(chan tailnetNodeStatus, 16), readEntered: make(chan struct{}, 1)}
	w.join = sync.NewCond(&w.mu)
	w.events <- f.status
	f.watch = w
	f.watches = 1
	return w, nil
}
func (f *fakeTailnetNode) Close() error {
	f.mu.Lock()
	if f.closing {
		done := f.closeDone
		f.mu.Unlock()
		<-done
		f.mu.Lock()
		err := f.closeErr
		f.mu.Unlock()
		return err
	}
	if f.closed {
		f.mu.Unlock()
		return nil
	}
	f.closing = true
	started := f.started
	close(f.startCancel)
	gate := f.closeGate
	w := f.watch
	close(f.closeEntered)
	f.mu.Unlock()
	if started {
		<-f.startDone
	}
	if w != nil {
		_ = w.Close()
	}
	if gate != nil {
		<-gate
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closeErr == nil {
		f.closed = true
		f.owners = 0
		f.status = tailnetNodeStatus{phase: tailnetNodeClosed}
	}
	close(f.closeDone)
	return f.closeErr
}
func (f *fakeTailnetNode) counts() fakeTailnetCounts {
	f.mu.Lock()
	defer f.mu.Unlock()
	return fakeTailnetCounts{f.owners, f.watches, f.lookups}
}
func (f *fakeTailnetNode) setPeer(addr netip.AddrPort, o tailnetObservation) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.peers[addr] = o
}
func (f *fakeTailnetNode) setStatus(s tailnetNodeStatus) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.started || f.closing || f.closed {
		return
	}
	f.status = s
	w := f.watch
	// State and ordered watch publication share one critical section. The
	// owner->watch lock order is safe: watch.Close releases watch.mu before
	// taking owner.mu, including after joining any active Next calls.
	if f.beforeStatusPublish != nil {
		f.beforeStatusPublish(s)
	}
	if w != nil {
		w.push(s)
	}
}
func (f *fakeTailnetNode) loseWatch(err error) {
	f.mu.Lock()
	w := f.watch
	f.mu.Unlock()
	if w != nil {
		w.fail(err)
	}
}

type fakeTailnetWatch struct {
	mu          sync.Mutex
	join        *sync.Cond
	owner       *fakeTailnetNode
	ctx         context.Context
	done        chan struct{}
	events      chan tailnetNodeStatus
	readEntered chan struct{}
	stopped     bool
	err         error
	readers     int
}

func (w *fakeTailnetWatch) Next() (tailnetNodeStatus, error) {
	w.mu.Lock()
	if w.stopped {
		err := w.err
		w.mu.Unlock()
		return tailnetNodeStatus{}, err
	}
	w.readers++
	w.mu.Unlock()
	defer func() { w.mu.Lock(); w.readers--; w.join.Broadcast(); w.mu.Unlock() }()
	var s tailnetNodeStatus
	select {
	case s = <-w.events:
	default:
		select {
		case w.readEntered <- struct{}{}:
		default:
		}
		select {
		case s = <-w.events:
		case <-w.done:
		case <-w.ctx.Done():
			return tailnetNodeStatus{}, w.ctx.Err()
		}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return tailnetNodeStatus{}, w.err
	}
	if err := w.ctx.Err(); err != nil {
		return tailnetNodeStatus{}, err
	}
	return s, nil
}
func (w *fakeTailnetWatch) fail(err error) {
	if err == nil {
		err = errFakeTailnetUnavailable
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.stopped {
		w.stopped = true
		w.err = err
		close(w.done)
	}
}
func (w *fakeTailnetWatch) push(s tailnetNodeStatus) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return
	}
	select {
	case w.events <- s:
	default:
		w.stopped = true
		w.err = errors.New("fake tailnet watch overflow: health unknown")
		close(w.done)
	}
}
func (w *fakeTailnetWatch) Close() error {
	w.mu.Lock()
	if !w.stopped {
		w.stopped = true
		w.err = io.EOF
		close(w.done)
	}
	for w.readers != 0 {
		w.join.Wait()
	}
	w.mu.Unlock()
	w.owner.mu.Lock()
	if w.owner.watch == w {
		w.owner.watch = nil
		w.owner.watches = 0
	}
	w.owner.mu.Unlock()
	return nil
}
