package cdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// fakeChrome is the scripted far end of the two pipes: it reads commands
// as Chrome reads fd 3 and writes frames as Chrome writes fd 4.
type fakeChrome struct {
	t    *testing.T
	cmds *frameReader
	out  *io.PipeWriter
}

func newPair(t *testing.T) (*Conn, *fakeChrome) {
	t.Helper()
	// Match Chrome's deadline-capable owned command transport. A Close-only
	// io.Pipe fake cannot distinguish peer consumption from Write returning:
	// cancellation in that window must conservatively retire that transport.
	cmdR, cmdW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outR, outW := io.Pipe()
	c := New(outR, cmdW)
	t.Cleanup(func() {
		c.Close()
		cmdR.Close()
		outW.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := c.Wait(ctx); err != nil {
			t.Errorf("CDP fixture join: %v", err)
		}
	})
	return c, &fakeChrome{t: t, cmds: newFrameReader(cmdR, MaxFrame), out: outW}
}

func (p *fakeChrome) recv() Message {
	p.t.Helper()
	frame, err := p.cmds.next()
	if err != nil {
		p.t.Errorf("peer recv: %v", err)
		return Message{}
	}
	var m Message
	if err := json.Unmarshal(frame, &m); err != nil {
		p.t.Errorf("peer decode %q: %v", frame, err)
	}
	return m
}

func (p *fakeChrome) send(raw string) {
	if _, err := p.out.Write(append([]byte(raw), 0)); err != nil {
		p.t.Errorf("peer send: %v", err)
	}
}

type callResult struct {
	v   map[string]int
	err error
}

func goCall(ctx context.Context, c *Conn, method string, params any) <-chan callResult {
	ch := make(chan callResult, 1)
	go func() {
		var r callResult
		r.err = c.Call(ctx, "", method, params, &r.v)
		ch <- r
	}()
	return ch
}

func await[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
		panic("unreachable")
	}
}

func TestCallRepliesRoutedByID(t *testing.T) {
	c, p := newPair(t)
	a := goCall(context.Background(), c, "A.a", map[string]string{"k": "v"})
	m1 := p.recv()
	b := goCall(context.Background(), c, "B.b", nil)
	m2 := p.recv()
	if string(m1.Params) != `{"k":"v"}` || m2.Params != nil {
		t.Fatalf("params = %s / %s", m1.Params, m2.Params)
	}
	p.send(fmt.Sprintf(`{"id":%d,"result":{"n":%d}}`, m2.ID, m2.ID))
	p.send(fmt.Sprintf(`{"id":%d,"result":{"n":%d}}`, m1.ID, m1.ID))
	for _, tc := range []struct {
		ch <-chan callResult
		id int64
	}{{a, m1.ID}, {b, m2.ID}} {
		r := await(t, tc.ch)
		if r.err != nil || int64(r.v["n"]) != tc.id {
			t.Fatalf("call for id %d got %v, %v", tc.id, r.v, r.err)
		}
	}
}

func TestCallSessionID(t *testing.T) {
	c, p := newPair(t)
	done := make(chan error, 1)
	go func() { done <- c.Call(context.Background(), "S1", "Page.enable", nil, nil) }()
	m := p.recv()
	if m.SessionID != "S1" || m.Method != "Page.enable" {
		t.Fatalf("command = %+v", m)
	}
	p.send(fmt.Sprintf(`{"id":%d,"result":{},"sessionId":"S1"}`, m.ID))
	if err := await(t, done); err != nil {
		t.Fatal(err)
	}
}

func TestCallErrorReply(t *testing.T) {
	c, p := newPair(t)
	r := goCall(context.Background(), c, "Target.attachToTarget", nil)
	m := p.recv()
	p.send(fmt.Sprintf(`{"id":%d,"error":{"code":-32000,"message":"No target with given id found","data":"x"}}`, m.ID))
	var ce *Error
	if err := await(t, r).err; !errors.As(err, &ce) || ce.Code != -32000 || ce.Message != "No target with given id found" || ce.Data != "x" {
		t.Fatalf("err = %#v", err)
	}
	// A non-string data field must not be treated as a malformed message.
	r = goCall(context.Background(), c, "X.y", nil)
	m = p.recv()
	p.send(fmt.Sprintf(`{"id":%d,"error":{"code":1,"message":"m","data":{"a":1}}}`, m.ID))
	if err := await(t, r).err; !errors.As(err, &ce) || ce.Data != `{"a":1}` {
		t.Fatalf("err = %#v", err)
	}
	if c.Err() != nil {
		t.Fatalf("conn failed: %v", c.Err())
	}
}

func TestEventsInOrder(t *testing.T) {
	c, p := newPair(t)
	go func() {
		for i := range 5 {
			p.send(fmt.Sprintf(`{"method":"E.e","params":{"i":%d},"sessionId":"S%d"}`, i, i))
		}
	}()
	for i := range 5 {
		ev := await(t, c.Events())
		if ev.Method != "E.e" || ev.SessionID != fmt.Sprintf("S%d", i) || string(ev.Params) != fmt.Sprintf(`{"i":%d}`, i) {
			t.Fatalf("event %d = %+v", i, ev)
		}
	}
}

func TestPeerCloseFailsPendingAndDrainsEvents(t *testing.T) {
	c, p := newPair(t)
	r := goCall(context.Background(), c, "A.a", nil)
	p.recv()
	p.send(`{"method":"E.last"}`)
	p.out.Close()
	if err := await(t, r).err; !errors.Is(err, io.EOF) {
		t.Fatalf("pending call err = %v, want io.EOF", err)
	}
	await(t, c.Done())
	if !errors.Is(c.Err(), io.EOF) {
		t.Fatalf("Err() = %v", c.Err())
	}
	var got []string
	for ev := range c.Events() {
		got = append(got, ev.Method)
	}
	if len(got) != 1 || got[0] != "E.last" {
		t.Fatalf("drained events = %q", got)
	}
	if err := c.Call(context.Background(), "", "B.b", nil, nil); !errors.Is(err, io.EOF) {
		t.Fatalf("call after done = %v", err)
	}
}

func TestBacklogFailsConn(t *testing.T) {
	old := maxQueuedEvents
	maxQueuedEvents = 3
	t.Cleanup(func() { maxQueuedEvents = old })
	c, p := newPair(t)
	r := goCall(context.Background(), c, "A.a", nil)
	p.recv()
	for i := range 5 {
		p.send(fmt.Sprintf(`{"method":"E.e","params":{"i":%d}}`, i))
	}
	if err := await(t, r).err; !errors.Is(err, ErrBacklog) {
		t.Fatalf("pending call err = %v", err)
	}
	if !errors.Is(c.Err(), ErrBacklog) {
		t.Fatalf("Err() = %v", c.Err())
	}
	// The reader keeps draining, so the peer is never blocked writing.
	p.send(`{"method":"E.after"}`)
}

func TestCancelledCallIgnoresLateReply(t *testing.T) {
	c, p := newPair(t)
	ctx, cancel := context.WithCancel(context.Background())
	r := goCall(ctx, c, "Slow.call", nil)
	m1 := p.recv()
	cancel()
	if err := await(t, r).err; !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	p.send(fmt.Sprintf(`{"id":%d,"result":{"n":1}}`, m1.ID))
	r = goCall(context.Background(), c, "Next.call", nil)
	m2 := p.recv()
	p.send(fmt.Sprintf(`{"id":%d,"result":{"n":2}}`, m2.ID))
	if got := await(t, r); got.err != nil || got.v["n"] != 2 {
		t.Fatalf("second call = %v, %v", got.v, got.err)
	}
}

func TestCloseIdempotentAndCallFailsFast(t *testing.T) {
	c, _ := newPair(t)
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	await(t, c.Done())
	if err := c.Call(context.Background(), "", "A.a", nil, nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("call after close = %v", err)
	}
	if _, ok := <-c.Events(); ok {
		t.Fatal("events channel open after Close")
	}
}

func TestMalformedMessageFailsConn(t *testing.T) {
	c, p := newPair(t)
	p.send(`{not json`)
	await(t, c.Done())
	if c.Err() == nil || errors.Is(c.Err(), io.EOF) {
		t.Fatalf("Err() = %v", c.Err())
	}
}

// A consumer that Calls from its own event loop must not deadlock while the
// peer floods events ahead of the reply.
func TestCallFromEventLoopWithQueuedEvents(t *testing.T) {
	c, p := newPair(t)
	peerDone := make(chan struct{})
	go func() {
		defer close(peerDone)
		p.send(`{"method":"E.first"}`)
		m := p.recv()
		for i := range 10 {
			p.send(fmt.Sprintf(`{"method":"E.more","params":{"i":%d}}`, i))
		}
		p.send(fmt.Sprintf(`{"id":%d,"result":{"n":7}}`, m.ID))
	}()
	if ev := await(t, c.Events()); ev.Method != "E.first" {
		t.Fatalf("first event = %+v", ev)
	}
	// Synchronous, as an event loop would do it; nobody reads Events meanwhile.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var res map[string]int
	if err := c.Call(ctx, "", "A.a", nil, &res); err != nil || res["n"] != 7 {
		t.Fatalf("call = %v, %v", res, err)
	}
	for i := range 10 {
		if ev := await(t, c.Events()); string(ev.Params) != fmt.Sprintf(`{"i":%d}`, i) {
			t.Fatalf("event %d = %+v", i, ev)
		}
	}
	await(t, peerDone)
}

// After a failure that leaves the pipe open, the reader keeps draining: the
// peer's writes never block, so it goes on to see its command pipe close.
func TestReaderDrainsAfterProtocolFailure(t *testing.T) {
	for name, poison := range map[string]string{
		"malformed": `{not json`,
		"oversized": `{"method":"` + strings.Repeat("x", 64) + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			defer func(m int) { maxFrame = m }(maxFrame)
			maxFrame = 32
			c, p := newPair(t)
			p.send(poison)
			await(t, c.Done())
			if !errors.Is(c.Err(), ErrProtocol) || (name == "oversized") != errors.Is(c.Err(), ErrFrameTooLarge) {
				t.Fatalf("Err() = %v, want ErrProtocol", c.Err())
			}
			wrote := make(chan struct{})
			go func() {
				for i := 0; i < 1000; i++ {
					p.out.Write([]byte(`{"method":"X.y","params":{}}` + "\x00"))
				}
				close(wrote)
			}()
			select {
			case <-wrote:
			case <-time.After(5 * time.Second):
				t.Fatal("peer blocked writing after the failure")
			}
		})
	}
}

// The transport closing is never ErrProtocol; the failures from what the
// peer sent always are.
func TestErrClasses(t *testing.T) {
	c, p := newPair(t)
	p.out.Close()
	await(t, c.Done())
	if errors.Is(c.Err(), ErrProtocol) || !errors.Is(c.Err(), io.EOF) {
		t.Fatalf("peer close: %v", c.Err())
	}
	for _, err := range []error{ErrBacklog, ErrFrameTooLarge} {
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("%v is not ErrProtocol", err)
		}
	}
	c2, _ := newPair(t)
	c2.Close()
	if errors.Is(c2.Err(), ErrProtocol) || !errors.Is(c2.Err(), ErrClosed) {
		t.Fatalf("Close: %v", c2.Err())
	}
}

// Seq orders events and replies as Chrome sent them: an event sent before a
// reply has a smaller Seq than CallSeq returns, whatever order the consumer
// reads them in (#458: a snapshot's reply marks which queued DOM events it
// already includes).
func TestSeqOrdersEventsAndReplies(t *testing.T) {
	c, p := newPair(t)
	type res struct {
		seq uint64
		err error
	}
	done := make(chan res, 1)
	go func() {
		seq, err := c.CallSeq(context.Background(), "S", "DOM.getDocument", nil, nil)
		done <- res{seq, err}
	}()
	m := p.recv()
	p.send(`{"method":"DOM.childNodeInserted","params":{},"sessionId":"S"}`)
	p.send(fmt.Sprintf(`{"id":%d,"result":{},"sessionId":"S"}`, m.ID))
	p.send(`{"method":"DOM.attributeModified","params":{},"sessionId":"S"}`)
	r := <-done
	if r.err != nil {
		t.Fatal(r.err)
	}
	before, after := await(t, c.Events()), await(t, c.Events())
	if !(before.Seq < r.seq && r.seq < after.Seq) || before.Seq == 0 {
		t.Fatalf("seqs: event %d, reply %d, event %d", before.Seq, r.seq, after.Seq)
	}
}
