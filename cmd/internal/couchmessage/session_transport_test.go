package couchmessage

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSessionFrameStrictness(t *testing.T) {
	b := registryBinding("pair:1", "a", 7)
	for name, raw := range map[string]string{
		"unknown field":      `{"Op":"activity","Observation":{"Sequence":1},"Extra":1}`,
		"hello no binding":   `{"Op":"hello"}`,
		"activity no obs":    `{"Op":"activity"}`,
		"unknown op":         `{"Op":"register"}`,
		"ack with a binding": fmt.Sprintf(`{"Op":"ack","Code":"ok","Binding":{"Slot":%q}}`, b.Slot),
	} {
		if _, err := DecodeSessionFrame([]byte(raw)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	raw, err := EncodeSessionFrame(SessionFrame{Op: FrameHello, Binding: &b})
	if err != nil {
		t.Fatal(err)
	}
	if f, err := DecodeSessionFrame(raw); err != nil || *f.Binding != b {
		t.Fatalf("round trip %+v %v", f, err)
	}
}

func TestReconnectBackoffDoublesToCap(t *testing.T) {
	b := DefaultReconnectBackoff
	want := []time.Duration{250 * time.Millisecond, 500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second, 5 * time.Second, 5 * time.Second}
	for i, w := range want {
		if got := b.Next(i); got != w {
			t.Fatalf("Next(%d) = %v, want %v", i, got, w)
		}
	}
	if got := b.Next(1000); got != b.Cap {
		t.Fatalf("Next(1000) = %v", got)
	}
}

// sessionLog is a SessionHandler recording events in arrival order.
type sessionLog struct {
	mu     sync.Mutex
	events []string
	opened map[SessionToken]Binding
	frames []SessionFrame
	builds []BuildIdentity
	refuse error
	notify chan struct{}
}

func newSessionLog() *sessionLog {
	return &sessionLog{opened: map[SessionToken]Binding{}, notify: make(chan struct{}, 1000)}
}
func (l *sessionLog) add(s string) {
	l.events = append(l.events, s)
	l.notify <- struct{}{}
}
func (l *sessionLog) handler() SessionHandler {
	return SessionHandler{
		Open: func(t SessionToken, h SessionHello) error {
			l.mu.Lock()
			defer l.mu.Unlock()
			if l.refuse != nil {
				return l.refuse
			}
			l.opened[t] = h.Binding
			if h.Build != nil {
				l.builds = append(l.builds, *h.Build)
			}
			l.add(fmt.Sprintf("open %d", t))
			return nil
		},
		Frame: func(t SessionToken, f SessionFrame) {
			l.mu.Lock()
			defer l.mu.Unlock()
			l.frames = append(l.frames, f)
			l.add(fmt.Sprintf("%s %d %d", f.Op, t, f.Observation.Sequence))
		},
		Closed: func(t SessionToken) {
			l.mu.Lock()
			defer l.mu.Unlock()
			l.add(fmt.Sprintf("closed %d", t))
		},
	}
}
func (l *sessionLog) waitFor(t *testing.T, want string) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		l.mu.Lock()
		for _, e := range l.events {
			if e == want {
				l.mu.Unlock()
				return
			}
		}
		events := append([]string(nil), l.events...)
		l.mu.Unlock()
		select {
		case <-l.notify:
		case <-deadline:
			t.Fatalf("never saw %q; events %v", want, events)
		}
	}
}
func (l *sessionLog) count(prefix string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, e := range l.events {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			n++
		}
	}
	return n
}

func sessionSocket(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "pair-session-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "registry.sock")
}

func selfBinding(slot string) Binding {
	b := registryBinding(slot, "a", os.Getpid())
	return b
}

func startSessionServer(t *testing.T, socket string, l *sessionLog) *SessionServer {
	t.Helper()
	s, err := StartSessionServer(context.Background(), socket, l.handler())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestPeerPIDIsTheDialingProcess(t *testing.T) {
	socket := sessionSocket(t)
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		c, err := net.Dial("unix", socket)
		if err == nil {
			time.Sleep(200 * time.Millisecond)
			c.Close()
		}
	}()
	conn, err := l.AcceptUnix()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if pid, err := PeerPID(conn); err != nil || pid != os.Getpid() {
		t.Fatalf("PeerPID = %d, %v; want %d", pid, err, os.Getpid())
	}
}

func TestSessionCoalescesActivityAndSendsSubmitAtOnce(t *testing.T) {
	socket := sessionSocket(t)
	log := newSessionLog()
	startSessionServer(t, socket, log)
	c := NewSessionClient()
	c.Update(Observation{Sequence: 1})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, socket, selfBinding("pair:1"), DefaultReconnectBackoff, 300*time.Millisecond)
	log.waitFor(t, "open 1")
	log.waitFor(t, "activity 1 1")
	// A burst of output inside one interval coalesces to one frame carrying
	// the latest observation.
	for i := 2; i <= 1000; i++ {
		c.Update(Observation{Sequence: uint64(i)})
	}
	log.waitFor(t, "activity 1 1000")
	if n := log.count("activity"); n > 3 {
		t.Fatalf("burst sent %d activity frames", n)
	}
	c.Submit(Observation{Sequence: 1000, Submission: 1})
	log.waitFor(t, "submit 1 1000")
	// Idle: nothing more.
	before := log.count("")
	time.Sleep(700 * time.Millisecond)
	if after := log.count(""); after != before {
		t.Fatalf("idle session sent %d frames", after-before)
	}
	cancel()
	log.waitFor(t, "closed 1")
}

func TestSessionRefusesForeignPeerAndCapacity(t *testing.T) {
	socket := sessionSocket(t)
	log := newSessionLog()
	startSessionServer(t, socket, log)
	foreign := registryBinding("pair:1", "a", os.Getpid()+1)
	c := NewSessionClient()
	if err := c.session(context.Background(), socket, foreign, time.Second); err == nil || log.count("open") != 0 {
		t.Fatalf("foreign-PID hello admitted: %v %v", err, log.events)
	}
	log.mu.Lock()
	log.refuse = errors.New("no")
	log.mu.Unlock()
	if err := c.session(context.Background(), socket, selfBinding("pair:1"), time.Second); err == nil {
		t.Fatal("handler refusal reached ack ok")
	}
	if log.count("closed") != 0 {
		t.Fatal("refused session reported Closed")
	}
}

func TestSessionMalformedFrameEndsOnlyThatSession(t *testing.T) {
	socket := sessionSocket(t)
	log := newSessionLog()
	startSessionServer(t, socket, log)
	good := NewSessionClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go good.Run(ctx, socket, selfBinding("pair:1"), DefaultReconnectBackoff, time.Millisecond)
	log.waitFor(t, "open 1")
	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	b := selfBinding("pair:2")
	if err := writeSessionFrame(conn, SessionFrame{Op: FrameHello, Binding: &b}); err != nil {
		t.Fatal(err)
	}
	if ack, err := readSessionFrame(conn); err != nil || ack.Code != "ok" {
		t.Fatalf("ack %+v %v", ack, err)
	}
	if err := transportWriteFrame(conn, []byte(`{"Op":"hello","Binding":null}`)); err != nil {
		t.Fatal(err)
	}
	log.waitFor(t, "closed 2")
	good.Update(Observation{Sequence: 9})
	log.waitFor(t, "activity 1 9")
}

func TestSessionClientReconnectsAfterBrokerRestart(t *testing.T) {
	socket := sessionSocket(t)
	first := newSessionLog()
	s := startSessionServer(t, socket, first)
	c := NewSessionClient()
	c.Update(Observation{Sequence: 5})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	backoff := ReconnectBackoff{Base: 20 * time.Millisecond, Cap: 100 * time.Millisecond, StableAfter: time.Minute}
	go c.Run(ctx, socket, selfBinding("pair:1"), backoff, time.Millisecond)
	first.waitFor(t, "activity 1 5")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	first.waitFor(t, "closed 1")
	second := newSessionLog()
	startSessionServer(t, socket, second)
	// The new broker learns the binding and the latest state without the
	// wrapper doing anything new.
	second.waitFor(t, "open 1")
	second.waitFor(t, "activity 1 5")
}

func TestSessionServerCapacity(t *testing.T) {
	socket := sessionSocket(t)
	log := newSessionLog()
	startSessionServer(t, socket, log)
	var conns []net.Conn
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()
	for i := 0; i <= MaxActors; i++ {
		conn, err := net.Dial("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, conn)
		b := selfBinding(fmt.Sprintf("pair:%d", i))
		if err := writeSessionFrame(conn, SessionFrame{Op: FrameHello, Binding: &b}); err != nil {
			t.Fatal(err)
		}
		ack, err := readSessionFrame(conn)
		if err != nil {
			t.Fatal(err)
		}
		if i < MaxActors && ack.Code != "ok" || i == MaxActors && ack.Code != "busy" {
			t.Fatalf("session %d ack %+v", i, ack)
		}
	}
}
