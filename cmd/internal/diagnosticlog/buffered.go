package diagnosticlog

import (
	"errors"
	"io"
	"sync"
	"time"
)

const BatchFlushInterval = 5 * time.Second

// BufferedStats accounts logical optional-diagnostic bytes, including rejected
// records and bytes abandoned on a failed orderly drain. LastError records the
// latest sink outcome; a later successful write clears an earlier error.
type BufferedStats struct {
	AcceptedBytes, WrittenBytes, DroppedBytes int64
	PendingBytes, Records                     int
	LastError                                 error
}
type flushRequest struct {
	target int64
	result chan error
}

// BufferedWriter owns one worker and timer. Write only copies/admit records;
// all sink IO is worker-owned. Healthy dirty data flushes within five seconds
// subject to scheduling/IO latency; a process crash can lose all pending bytes
// (at most 256KiB). This is not a power-loss durability guarantee.
type BufferedWriter struct {
	mu            sync.Mutex
	state         BatchState
	sink          io.WriteCloser
	ticks         <-chan time.Time
	stop          func()
	wake          chan struct{}
	flush         chan flushRequest
	closing, done chan struct{}
	closeErr      error
}

func OpenBuffered(path string, options Options) (*BufferedWriter, error) {
	sink, err := Open(path, options)
	if err != nil {
		return nil, err
	}
	if options.FlushTicks != nil {
		return newBufferedWriter(sink, options.FlushTicks, nil), nil
	}
	ticker := time.NewTicker(BatchFlushInterval)
	return newBufferedWriter(sink, ticker.C, ticker.Stop), nil
}
func newBufferedWriter(sink io.WriteCloser, ticks <-chan time.Time, stop func()) *BufferedWriter {
	w := &BufferedWriter{sink: sink, ticks: ticks, stop: stop, wake: make(chan struct{}, 1), flush: make(chan flushRequest), closing: make(chan struct{}), done: make(chan struct{})}
	go w.run()
	return w
}
func (w *BufferedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	var effect BatchEffect
	w.state, effect = BatchTransition(w.state, BatchEvent{Kind: BatchEnqueue, Data: p})
	wake := effect.Err == nil && w.state.PendingBytes >= BatchFlushBytes && !w.state.isBlocked()
	w.mu.Unlock()
	if wake {
		select {
		case w.wake <- struct{}{}:
		default:
		}
	}
	if effect.Err != nil {
		return 0, effect.Err
	}
	return len(p), nil
}
func (w *BufferedWriter) Stats() BufferedStats {
	w.mu.Lock()
	defer w.mu.Unlock()
	s := w.state
	return BufferedStats{s.AcceptedBytes, s.WrittenBytes, s.DroppedBytes, s.PendingBytes, s.Records, s.LastError}
}

// Flush waits for bytes accepted before the barrier. It permits one retry of
// a previous failed batch, then stops on any new error; it never spins on errors.
func (w *BufferedWriter) Flush() error {
	w.mu.Lock()
	target := w.state.AcceptedBytes
	closing := w.state.isClosing()
	w.mu.Unlock()
	if closing {
		return io.ErrClosedPipe
	}
	req := flushRequest{target: target, result: make(chan error, 1)}
	select {
	case w.flush <- req:
		return <-req.result
	case <-w.done:
		return io.ErrClosedPipe
	}
}
func (w *BufferedWriter) Close() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	if !w.state.isClosing() {
		w.state, _ = BatchTransition(w.state, BatchEvent{Kind: BatchClosing})
		close(w.closing)
	}
	w.mu.Unlock()
	<-w.done
	return w.closeErr
}
func (w *BufferedWriter) target() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state.AcceptedBytes
}
func (w *BufferedWriter) drain(target int64, retry bool) error {
	for {
		w.mu.Lock()
		if w.state.WrittenBytes >= target {
			w.mu.Unlock()
			return nil
		}
		var effect BatchEffect
		w.state, effect = BatchTransition(w.state, BatchEvent{Kind: BatchBegin, Retry: retry})
		w.mu.Unlock()
		retry = false
		if len(effect.Data) == 0 {
			return nil
		}
		n, err := w.sink.Write(effect.Data)
		w.mu.Lock()
		w.state, effect = BatchTransition(w.state, BatchEvent{Kind: BatchResult, N: n, Err: err})
		w.mu.Unlock()
		if effect.Err != nil {
			return effect.Err
		}
	}
}
func (w *BufferedWriter) run() {
	defer close(w.done)
	defer func() {
		if w.stop != nil {
			w.stop()
		}
	}()
	for {
		select {
		case <-w.closing:
			err := w.drain(w.target(), true)
			err = errors.Join(err, w.sink.Close())
			w.mu.Lock()
			w.state, _ = BatchTransition(w.state, BatchEvent{Kind: BatchClosed, Err: err})
			w.closeErr = err
			w.mu.Unlock()
			return
		case request := <-w.flush:
			request.result <- w.drain(request.target, true)
		case _, ok := <-w.ticks:
			if !ok {
				w.ticks = nil
				continue
			}
			_ = w.drain(w.target(), true)
		case <-w.wake:
			_ = w.drain(w.target(), false)
		}
	}
}

var _ io.WriteCloser = (*BufferedWriter)(nil)
