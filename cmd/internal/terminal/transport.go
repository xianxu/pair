package terminal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/xianxu/pair/cmd/internal/ttyio"
)

var ErrBackpressure = errors.New("terminal: input queue full")

type WriteFailure struct {
	Accepted, Total int
	Err             error
}

func (e *WriteFailure) Error() string {
	return fmt.Sprintf("terminal: input write accepted %d/%d bytes: %v", e.Accepted, e.Total, e.Err)
}
func (e *WriteFailure) Unwrap() error { return e.Err }

// InputWriter is the sole FIFO between one endpoint and its child. Admission is
// synchronous; Flush observes delivery. Bounds include the in-flight packet.
type InputWriter struct {
	mu                          sync.Mutex
	writer                      ttyio.Writer
	packets                     [][]byte
	bytes, maxPackets, maxBytes int
	failure                     error
	closed                      bool
	changed, notify             chan struct{}
	ctx                         context.Context
	cancel                      context.CancelFunc
	done                        chan struct{}
}

func NewInputWriter(writer ttyio.Writer, maxPackets, maxBytes int) *InputWriter {
	if maxPackets <= 0 {
		maxPackets = MaxInputPackets
	}
	if maxBytes <= 0 {
		maxBytes = MaxInputBytes
	}
	ctx, cancel := context.WithCancel(context.Background())
	w := &InputWriter{writer: writer, maxPackets: maxPackets, maxBytes: maxBytes, changed: make(chan struct{}), notify: make(chan struct{}, 1), ctx: ctx, cancel: cancel, done: make(chan struct{})}
	go w.run()
	return w
}
func (w *InputWriter) signalLocked() { close(w.changed); w.changed = make(chan struct{}) }
func (w *InputWriter) Enqueue(p []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.failure != nil {
		return w.failure
	}
	if w.closed {
		return os.ErrClosed
	}
	if len(p) == 0 {
		return nil
	}
	if len(w.packets) >= w.maxPackets || len(p) > w.maxBytes-w.bytes {
		return ErrBackpressure
	}
	w.packets = append(w.packets, append([]byte(nil), p...))
	w.bytes += len(p)
	w.signalLocked()
	select {
	case w.notify <- struct{}{}:
	default:
	}
	return nil
}
func (w *InputWriter) Failure() error { w.mu.Lock(); defer w.mu.Unlock(); return w.failure }
func (w *InputWriter) Pending() (int, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.packets), w.bytes
}
func (w *InputWriter) Flush(ctx context.Context) error {
	for {
		w.mu.Lock()
		err := w.failure
		if err == nil && w.closed {
			err = os.ErrClosed
		}
		empty := len(w.packets) == 0
		changed := w.changed
		w.mu.Unlock()
		if err != nil || empty {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}
func (w *InputWriter) run() {
	defer close(w.done)
	for {
		w.mu.Lock()
		var p []byte
		if len(w.packets) > 0 {
			p = w.packets[0]
		}
		closed := w.closed
		w.mu.Unlock()
		if closed {
			return
		}
		if p == nil {
			select {
			case <-w.ctx.Done():
				return
			case <-w.notify:
				continue
			}
		}
		ctx, cancel := context.WithTimeout(w.ctx, WriteTimeout)
		accepted, err := writeComplete(ctx, w.writer, p)
		cancel()
		w.mu.Lock()
		if err != nil {
			w.failure = &WriteFailure{accepted, len(p), err}
			w.packets = nil
			w.bytes = 0
			w.signalLocked()
			w.mu.Unlock()
			return
		}
		w.packets[0] = nil
		w.packets = w.packets[1:]
		w.bytes -= len(p)
		w.signalLocked()
		w.mu.Unlock()
	}
}
func writeComplete(ctx context.Context, writer ttyio.Writer, p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, err := writer.WriteContext(ctx, p)
		if n < 0 || n > len(p) {
			return total, errors.New("terminal: invalid write count")
		}
		total += n
		p = p[n:]
		if err != nil {
			return total, err
		}
		if n == 0 {
			return total, io.ErrNoProgress
		}
	}
	return total, nil
}
func (w *InputWriter) Close() {
	w.mu.Lock()
	if !w.closed {
		w.closed = true
		w.cancel()
		w.signalLocked()
	}
	w.mu.Unlock()
	<-w.done
	w.mu.Lock()
	w.packets = nil
	w.bytes = 0
	w.mu.Unlock()
}
