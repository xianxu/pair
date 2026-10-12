package couchmessage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

// SessionHandler receives one wrapper connection's lifecycle. Open runs before
// the ack; a refusal closes the connection without a Closed call. Frame and
// Closed follow in the connection's own order.
type SessionHandler struct {
	Open   func(SessionToken, SessionHello) error
	Frame  func(SessionToken, SessionFrame)
	Closed func(SessionToken)
}

// SessionHello is what a wrapper said when it connected. Build is non-nil
// exactly for a hello-v2 session, the only kind that may report Settled.
type SessionHello struct {
	Binding Binding
	Build   *BuildIdentity
}

// SessionServer holds long-lived wrapper connections on the registry socket.
// There is no idle timeout: a unix connection does not drop silently, and the
// kernel closes it when the wrapper dies or execs.
type SessionServer struct {
	listener *net.UnixListener
	socket   string
	identity os.FileInfo
	ctx      context.Context
	cancel   context.CancelFunc
	handler  SessionHandler
	mu       sync.Mutex
	conns    map[*net.UnixConn]struct{}
	next     SessionToken
	wg       sync.WaitGroup
	done     chan struct{}
	closeErr error
}

func StartSessionServer(ctx context.Context, socket string, h SessionHandler) (*SessionServer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if h.Open == nil || h.Frame == nil || h.Closed == nil {
		return nil, errors.New("session handler is incomplete")
	}
	listener, info, err := transportListen(socket)
	if err != nil {
		return nil, err
	}
	serverCtx, cancel := context.WithCancel(ctx)
	s := &SessionServer{listener: listener, socket: socket, identity: info, ctx: serverCtx, cancel: cancel, handler: h, conns: map[*net.UnixConn]struct{}{}, done: make(chan struct{})}
	go s.run()
	return s, nil
}

func (s *SessionServer) run() {
	go func() {
		<-s.ctx.Done()
		s.listener.Close()
		s.mu.Lock()
		for conn := range s.conns {
			conn.Close()
		}
		s.mu.Unlock()
	}()
	for {
		conn, err := s.listener.AcceptUnix()
		if err != nil {
			break
		}
		s.mu.Lock()
		if s.ctx.Err() != nil {
			s.mu.Unlock()
			conn.Close()
			continue
		}
		full := len(s.conns) >= MaxActors
		s.conns[conn] = struct{}{}
		s.next++
		token := s.next
		s.wg.Add(1)
		s.mu.Unlock()
		go s.serve(conn, token, full)
	}
	s.cancel()
	s.wg.Wait()
	s.closeErr = transportRemoveOwned(s.socket, s.identity)
	close(s.done)
}

func (s *SessionServer) serve(conn *net.UnixConn, token SessionToken, full bool) {
	defer s.wg.Done()
	defer func() { conn.Close(); s.mu.Lock(); delete(s.conns, conn); s.mu.Unlock() }()
	if err := conn.SetDeadline(time.Now().Add(AdmissionTimeout)); err != nil {
		return
	}
	hello, err := readSessionFrame(conn)
	if err != nil || (hello.Op != FrameHello && hello.Op != FrameHelloV2 && hello.Op != FrameHelloV3) {
		return
	}
	v2, v3 := hello.Op != FrameHello, hello.Op == FrameHelloV3
	refuse := func(code string, err error) {
		_ = writeSessionFrame(conn, SessionFrame{Op: FrameAck, Code: code, Error: err.Error()})
	}
	if full {
		refuse("busy", ErrRegistryFull)
		return
	}
	// The binding is the wrapper's own claim; the kernel's record of who
	// connected must agree before anything else trusts it.
	if pid, err := PeerPID(conn); err != nil || pid != hello.Binding.PID {
		refuse("refused", errors.New("session peer is not the binding's process"))
		return
	}
	if err := s.handler.Open(token, SessionHello{Binding: *hello.Binding, Build: hello.Build}); err != nil {
		refuse("refused", err)
		return
	}
	defer s.handler.Closed(token)
	if err := writeSessionFrame(conn, SessionFrame{Op: FrameAck, Code: "ok"}); err != nil {
		return
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		return
	}
	for {
		f, err := readSessionFrame(conn)
		if err != nil || (f.Op != FrameActivity && f.Op != FrameSubmit) || (!v2 && f.Settled != nil) || (!v3 && f.Picker != nil) {
			return // EOF is the wrapper's departure; a malformed frame ends only this session
		}
		s.handler.Frame(token, f)
	}
}

// Close is idempotent: it closes every session (each reports Closed) and
// removes only this listener's socket inode.
func (s *SessionServer) Close() error {
	s.cancel()
	<-s.done
	return s.closeErr
}

func readSessionFrame(conn net.Conn) (SessionFrame, error) {
	raw, err := transportReadFrameLimit(conn, MaxSessionFrameBytes)
	if err != nil {
		return SessionFrame{}, err
	}
	return DecodeSessionFrame(raw)
}

func writeSessionFrame(conn net.Conn, f SessionFrame) error {
	raw, err := EncodeSessionFrame(f)
	if err != nil {
		return err
	}
	return transportWriteFrame(conn, raw)
}

// SessionClient is a wrapper's side of its session. Update and Submit only
// record state and wake the sender, so callers on hot paths do no IO.
type SessionClient struct {
	mu        sync.Mutex
	latest    Observation
	have      bool
	dirty     bool // activity not yet sent on the current connection
	submit    bool // a submission not yet sent
	build     *BuildIdentity
	settled   bool
	picker    bool
	wake      chan struct{}
	connected chan struct{} // test hook: signalled after each ack
}

func NewSessionClient() *SessionClient {
	return &SessionClient{wake: make(chan struct{}, 1), connected: make(chan struct{}, 1)}
}

func (c *SessionClient) signal() {
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

// SetBuild opts the client into hello-v2 (#421). Call before Run.
func (c *SessionClient) SetBuild(b BuildIdentity) {
	c.mu.Lock()
	c.build = &b
	c.mu.Unlock()
}

// Settle records whether the wrapper is settled; it rides the next activity
// frame, and only on a hello-v2 session.
func (c *SessionClient) Settle(settled bool) {
	c.mu.Lock()
	if c.settled != settled {
		c.settled, c.dirty = settled, c.have || c.dirty
	}
	c.mu.Unlock()
	c.signal()
}

// Picker records whether an input-holding picker is up (pair#465); it rides
// the next activity frame, and only on a hello-v3 session.
func (c *SessionClient) Picker(open bool) {
	c.mu.Lock()
	if c.picker != open {
		c.picker, c.dirty = open, c.have || c.dirty
	}
	c.mu.Unlock()
	c.signal()
}

// Update records the wrapper's current observation; the sender coalesces.
func (c *SessionClient) Update(o Observation) {
	c.mu.Lock()
	c.latest, c.have, c.dirty = o, true, true
	c.mu.Unlock()
	c.signal()
}

// Submit records a genuine operator submission; it is sent without coalescing.
func (c *SessionClient) Submit(o Observation) {
	c.mu.Lock()
	c.latest, c.have, c.submit = o, true, true
	c.mu.Unlock()
	c.signal()
}

// Run holds a session for the wrapper's lifetime, redialing with backoff when
// the broker is absent or restarts. Activity frames are at most one per
// minInterval; idle wrappers send nothing.
func (c *SessionClient) Run(ctx context.Context, socket string, b Binding, backoff ReconnectBackoff, minInterval time.Duration) {
	attempt := 0
	for ctx.Err() == nil {
		started := time.Now()
		_ = c.session(ctx, socket, b, minInterval)
		if ctx.Err() != nil {
			return
		}
		if time.Since(started) >= backoff.StableAfter {
			attempt = 0
		}
		timer := time.NewTimer(backoff.Next(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		attempt++
	}
}

var errNoAck = errors.New("broker closed before acknowledging hello")

// sessionHandshake sends the hello op (build rides hello-v2 and hello-v3)
// and reads the ack.
func sessionHandshake(conn net.Conn, b Binding, build *BuildIdentity, op string) error {
	if err := conn.SetDeadline(time.Now().Add(AdmissionTimeout)); err != nil {
		return err
	}
	hello := SessionFrame{Op: op, Binding: &b}
	if op != FrameHello {
		hello.Build = build
	}
	if err := writeSessionFrame(conn, hello); err != nil {
		return err
	}
	ack, err := readSessionFrame(conn)
	if err != nil {
		// Only a close is an old broker's answer to a newer hello; a timeout is a
		// slow broker, and downgrading it would cost the session Settled.
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, syscall.ECONNRESET) {
			return fmt.Errorf("%w: %v", errNoAck, err)
		}
		return err
	}
	if ack.Op != FrameAck || ack.Code != "ok" {
		return errors.New("session refused: " + ack.Error)
	}
	return nil
}

func (c *SessionClient) session(ctx context.Context, socket string, b Binding, minInterval time.Duration) error {
	c.mu.Lock()
	build := c.build
	c.mu.Unlock()
	// Newest hello first. An old broker drops a hello it doesn't know without
	// an ack: redial with the next older one, which reports less (hello-v2
	// never sends Picker, hello never sends Settled). Each attempt owns its
	// own connection; only the one that took a hello outlives the loop.
	hellos := []string{FrameHello}
	if build != nil {
		hellos = []string{FrameHelloV3, FrameHelloV2, FrameHello}
	}
	var conn net.Conn
	var op string
	for i := 0; ; i++ {
		op = hellos[i]
		dialCtx, cancel := context.WithTimeout(ctx, AdmissionTimeout)
		attempt, err := (&net.Dialer{}).DialContext(dialCtx, "unix", socket)
		cancel()
		if err != nil {
			return err
		}
		err = sessionHandshake(attempt, b, build, op)
		if err == nil {
			conn = attempt
			break
		}
		attempt.Close()
		if i == len(hellos)-1 || !errors.Is(err, errNoAck) {
			return err
		}
	}
	defer conn.Close()
	v2, v3 := op != FrameHello, op == FrameHelloV3
	if err := conn.SetDeadline(time.Time{}); err != nil {
		return err
	}
	select {
	case c.connected <- struct{}{}:
	default:
	}
	// A new broker knows nothing: resend the latest state once.
	c.mu.Lock()
	c.dirty = c.dirty || c.have
	c.mu.Unlock()
	lost := make(chan struct{})
	go func() {
		defer close(lost)
		var one [1]byte
		_, _ = conn.Read(one[:]) // the broker sends nothing after ack; any return is loss
	}()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	var last time.Time
	var throttle <-chan time.Time
	c.signal()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-lost:
			return errors.New("session lost")
		case <-c.wake:
		case <-throttle:
			throttle = nil
		}
		c.mu.Lock()
		f := SessionFrame{}
		obs := c.latest
		var settled, picker *bool
		if v2 {
			v := c.settled
			settled = &v
		}
		if v3 {
			v := c.picker
			picker = &v
		}
		switch {
		case c.submit:
			f = SessionFrame{Op: FrameSubmit, Observation: &obs, Settled: settled, Picker: picker}
			c.submit, c.dirty = false, false
		case c.dirty && throttle == nil:
			if wait := minInterval - time.Since(last); !last.IsZero() && wait > 0 {
				throttle = time.After(wait)
			} else {
				f = SessionFrame{Op: FrameActivity, Observation: &obs, Settled: settled, Picker: picker}
				c.dirty = false
			}
		}
		c.mu.Unlock()
		if f.Op == "" {
			continue
		}
		if err := conn.SetWriteDeadline(time.Now().Add(TransportTimeout)); err != nil {
			return err
		}
		if err := writeSessionFrame(conn, f); err != nil {
			c.mu.Lock()
			c.dirty = true // resend on the next connection
			if f.Op == FrameSubmit {
				c.submit = true
			}
			c.mu.Unlock()
			return err
		}
		last = time.Now()
	}
}
