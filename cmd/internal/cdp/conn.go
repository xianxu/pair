package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var (
	// ErrBacklog fails a Conn whose consumer stopped reading Events: the
	// reader never blocks on the consumer (that would deadlock a consumer
	// making a synchronous Call from its event loop), so a stuck consumer
	// must fail loudly rather than grow memory without bound.
	ErrBacklog = fmt.Errorf("%w: event backlog exceeded; consumer is not reading events", ErrProtocol)
	ErrClosed  = errors.New("cdp: connection closed")
)

// maxQueuedEvents is a var so tests can trip the backlog cheaply.
var maxQueuedEvents = 4096

// Event is a CDP event; SessionID names the target session in flatten mode
// and is empty for browser-level events.
type Event struct {
	// Barrier is an internal fence, never a Chrome event. The consumer
	// closes it after applying all preceding events (including downstream
	// queues). A fence has no Method or Params.
	Barrier   chan struct{}
	SessionID string
	Method    string
	Params    json.RawMessage
	// Seq is the event's place among every message read (events and
	// replies), counted from 1. CallSeq returns a reply's: an event with a
	// smaller Seq was sent by Chrome before that reply.
	Seq uint64
}

// SyncEvents waits until the consumer has applied every event already read
// from Chrome. Call off the consumer's event loop. Unlike waiting for a CDP
// reply, this also waits for the asynchronous event pump and its consumer.
func (c *Conn) SyncEvents(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fence := make(chan struct{})
	c.mu.Lock()
	if c.err != nil {
		err := c.err
		c.mu.Unlock()
		return err
	}
	if len(c.queue) >= c.maxQueued {
		c.mu.Unlock()
		return ErrBacklog
	}
	c.queue = append(c.queue, Event{Barrier: fence})
	select {
	case c.signal <- struct{}{}:
	default:
	}
	c.mu.Unlock()
	select {
	case <-fence:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return c.Err()
	}
}

// Conn multiplexes Calls and Events over one pipe pair. One reader goroutine
// routes replies by id and queues events; one pump goroutine hands queued
// events to the consumer in arrival order.
type Conn struct {
	w          io.WriteCloser
	writeToken chan struct{}

	mu        sync.Mutex
	nextID    int64
	seq       uint64 // messages read so far
	pending   map[int64]chan Message
	queue     []Event
	maxQueued int
	err       error

	signal    chan struct{} // reader → pump: queue grew
	done      chan struct{} // closed on the first failure
	closed    chan struct{} // closed by Close; releases a pump blocked on a gone consumer
	closeOnce sync.Once
	closeErr  error
	readDone  chan struct{}
	pumpDone  chan struct{}
	events    chan Event
}

// New starts a Conn reading frames from r (Chrome's fd 4) and writing
// commands to w (Chrome's fd 3). Conn owns w: SetWriteDeadline must
// interrupt Write when available; otherwise Close must interrupt an active
// Write. Cancellation of a generic writer retires the connection. The reader
// goroutine exits only when r
// returns an error, which for a real pipe means Chrome closed its end.
func New(r io.Reader, w io.WriteCloser) *Conn {
	c := &Conn{
		w:          w,
		writeToken: make(chan struct{}, 1),
		pending:    map[int64]chan Message{},
		maxQueued:  maxQueuedEvents,
		signal:     make(chan struct{}, 1),
		done:       make(chan struct{}),
		closed:     make(chan struct{}),
		events:     make(chan Event),
		readDone:   make(chan struct{}), pumpDone: make(chan struct{}),
	}
	go c.read(r, newFrameReader(r, maxFrame))
	go c.pump()
	return c
}

// Call sends method (on sessionID when non-empty) and waits for its reply,
// decoding the result into result when both are non-nil. An error reply is
// returned as *Error.
func (c *Conn) Call(ctx context.Context, sessionID, method string, params, result any) error {
	_, err := c.CallSeq(ctx, sessionID, method, params, result)
	return err
}

// CallSeq is Call, also returning the reply's place in the read order (see
// Event.Seq); 0 when no reply came.
func (c *Conn) CallSeq(ctx context.Context, sessionID, method string, params, result any) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m := Message{Method: method, SessionID: sessionID}
	if params != nil {
		raw, err := json.Marshal(params)
		if err != nil {
			return 0, fmt.Errorf("cdp: %s: encode params: %w", method, err)
		}
		m.Params = raw
	}
	ch := make(chan Message, 1)
	c.mu.Lock()
	if c.err != nil {
		err := c.err
		c.mu.Unlock()
		return 0, fmt.Errorf("cdp: %s: %w", method, err)
	}
	c.nextID++
	m.ID = c.nextID
	c.pending[m.ID] = ch
	c.mu.Unlock()

	frame, err := encode(m)
	if err != nil {
		c.forget(m.ID)
		return 0, fmt.Errorf("cdp: %s: encode: %w", method, err)
	}
	if err = c.writeFrame(ctx, frame); err != nil {
		c.forget(m.ID)
		return 0, fmt.Errorf("cdp: %s: %w", method, err)
	}

	select {
	case reply := <-ch:
		return reply.Seq, decodeReply(method, reply, result)
	case <-ctx.Done():
		c.forget(m.ID)
		return 0, ctx.Err()
	case <-c.done:
		// A reply routed just before the failure still wins.
		select {
		case reply := <-ch:
			return reply.Seq, decodeReply(method, reply, result)
		default:
		}
		c.forget(m.ID)
		return 0, fmt.Errorf("cdp: %s: %w", method, c.Err())
	}
}

type writeDeadlineSetter interface{ SetWriteDeadline(time.Time) error }

// writeFrame owns both frame serialization and cancellation until the write's
// callback is joined and its deadline reset. No next writer can inherit an old
// callback or interrupt. A partial frame always retires the protocol stream.
func (c *Conn) writeFrame(ctx context.Context, frame []byte) error {
	select {
	case c.writeToken <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return c.Err()
	}
	defer func() { <-c.writeToken }()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.Err(); err != nil {
		return err
	}
	deadlineWriter, hasDeadline := c.w.(writeDeadlineSetter)
	if hasDeadline {
		deadline, _ := ctx.Deadline()
		if err := deadlineWriter.SetWriteDeadline(deadline); err != nil {
			return c.retireWrite(err)
		}
	}
	callbackDone := make(chan struct{})
	var interruptErr error
	stop := context.AfterFunc(ctx, func() {
		defer close(callbackDone)
		if hasDeadline {
			interruptErr = deadlineWriter.SetWriteDeadline(time.Now())
			if interruptErr == nil {
				return
			}
		}
		// Generic owned transports promise Close interrupts Write. Closing is also
		// the safe fallback when a declared deadline setter fails during cancel.
		_ = c.retireWrite(errors.Join(ctx.Err(), interruptErr))
	})
	var n int
	var err error
	if err = ctx.Err(); err == nil {
		n, err = c.w.Write(frame)
	}
	if !stop() {
		<-callbackDone
	}
	if hasDeadline {
		if resetErr := deadlineWriter.SetWriteDeadline(time.Time{}); resetErr != nil {
			return c.retireWrite(errors.Join(err, resetErr))
		}
	}
	if interruptErr != nil {
		return c.retireWrite(errors.Join(err, interruptErr))
	}
	if n == len(frame) && err == nil {
		// Reply cancellation does not corrupt a fully emitted frame.
		return c.Err()
	}
	contextErr := ctx.Err()
	if contextErr == nil {
		if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
			contextErr = context.DeadlineExceeded
		}
	}
	if n == 0 && hasDeadline && contextErr != nil && c.Err() == nil && (errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, contextErr)) {
		// No frame byte was emitted; after reset this owned pipe remains usable.
		return contextErr
	}
	if err == nil {
		err = io.ErrShortWrite
	}
	return c.retireWrite(errors.Join(err, contextErr))
}
func (c *Conn) retireWrite(err error) error {
	c.fail(fmt.Errorf("cdp: write: %w", err))
	closeErr := c.Close()
	return errors.Join(c.Err(), closeErr)
}

func decodeReply(method string, reply Message, result any) error {
	if reply.Error != nil {
		return reply.Error
	}
	if result != nil && len(reply.Result) > 0 {
		if err := json.Unmarshal(reply.Result, result); err != nil {
			return fmt.Errorf("cdp: %s: decode result: %w", method, err)
		}
	}
	return nil
}

func (c *Conn) forget(id int64) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

// Events delivers events in arrival order. The channel closes once the Conn
// is done and every queued event has been delivered, so ranging over it
// terminates; after Close, still-queued events are dropped instead. A
// consumer that stops reading must Close, or the pump stays blocked on it.
func (c *Conn) Events() <-chan Event { return c.events }

// Done is closed when the Conn fails or is closed.
func (c *Conn) Done() <-chan struct{} { return c.done }

// Err is nil while the Conn runs, then the first failure. Failures from what
// the peer sent (oversized, malformed, a backlog) are ErrProtocol; the pipe
// ending is a read error wrapping io.EOF (or the pipe's own close error), and
// our Close is ErrClosed.
func (c *Conn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Close closes the command pipe and ends the Conn; it is idempotent. Chrome
// exits when its command pipe closes, which in turn ends the reader.
func (c *Conn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
		c.fail(ErrClosed)
		c.closeErr = c.w.Close()
	})
	return c.closeErr
}

func (c *Conn) fail(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return
	}
	c.err = err
	close(c.done)
}

// read routes frames until the pipe ends. After a failure that leaves the
// pipe open (oversized or malformed) it keeps reading and discarding bytes,
// so the peer never blocks writing and sees its command pipe close instead.
func (c *Conn) read(r io.Reader, fr *frameReader) {
	defer close(c.readDone)
	for {
		frame, err := fr.next()
		if err != nil {
			if errors.Is(err, ErrFrameTooLarge) {
				c.fail(err)
				io.Copy(io.Discard, r)
			} else {
				c.fail(fmt.Errorf("cdp: read: %w", err))
			}
			return
		}
		var m Message
		if err := json.Unmarshal(frame, &m); err != nil {
			c.fail(fmt.Errorf("%w: malformed message: %v", ErrProtocol, err))
			io.Copy(io.Discard, r)
			return
		}
		c.dispatch(m)
	}
}

// dispatch routes one message. Once the Conn is done the reader keeps
// draining, so Chrome never blocks writing, but delivers nothing.
func (c *Conn) dispatch(m Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return
	}
	c.seq++
	m.Seq = c.seq
	if m.ID != 0 {
		// Unknown ids are replies to cancelled Calls.
		if ch, ok := c.pending[m.ID]; ok {
			delete(c.pending, m.ID)
			ch <- m
		}
		return
	}
	if m.Method == "" {
		return
	}
	if len(c.queue) >= c.maxQueued {
		c.err = ErrBacklog
		close(c.done)
		return
	}
	c.queue = append(c.queue, Event{SessionID: m.SessionID, Method: m.Method, Params: m.Params, Seq: m.Seq})
	select {
	case c.signal <- struct{}{}:
	default:
	}
}

func (c *Conn) pump() {
	defer close(c.pumpDone)
	defer close(c.events)
	for {
		c.mu.Lock()
		if len(c.queue) == 0 {
			finished := c.err != nil
			c.mu.Unlock()
			if finished {
				return
			}
			select {
			case <-c.signal:
			case <-c.done:
			}
			continue
		}
		ev := c.queue[0]
		c.queue[0] = Event{}
		c.queue = c.queue[1:]
		c.mu.Unlock()
		select {
		case c.events <- ev:
		case <-c.closed:
			return
		}
	}
}

// Wait joins both protocol goroutines. Close ends command handling but cannot
// interrupt an arbitrary supplied Reader: its owner must close that input (or
// arrange peer exit) before joining. Context bounds uncertain peer cleanup.
func (c *Conn) Wait(ctx context.Context) error {
	for _, done := range []<-chan struct{}{c.readDone, c.pumpDone} {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
