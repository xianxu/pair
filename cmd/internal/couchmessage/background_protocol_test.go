package couchmessage

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func TestBackgroundFrameValidation(t *testing.T) {
	b := selfBinding("pair:1")
	yes := true
	bad := testBuild
	bad.SHA256 = "bad"
	for _, tc := range []struct {
		name string
		f    SessionFrame
		ok   bool
	}{
		{"v4", SessionFrame{Op: FrameHelloV4, Binding: &b, Build: &testBuild}, true},
		{"v4 missing build", SessionFrame{Op: FrameHelloV4, Binding: &b}, false},
		{"v4 bad build", SessionFrame{Op: FrameHelloV4, Binding: &b, Build: &bad}, false},
		{"activity", SessionFrame{Op: FrameActivity, Observation: &Observation{}, BackgroundWork: &yes}, true},
		{"submit", SessionFrame{Op: FrameSubmit, Observation: &Observation{}, BackgroundWork: &yes}, true},
		{"ack", SessionFrame{Op: FrameAck, Code: "ok", BackgroundWork: &yes}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.f.Validate(); (err == nil) != tc.ok {
				t.Fatalf("Validate()=%v want valid=%v", err, tc.ok)
			}
		})
	}
	for _, op := range []string{FrameHello, FrameHelloV2, FrameHelloV3, FrameHelloV4} {
		f := SessionFrame{Op: op, Binding: &b, BackgroundWork: &yes}
		if op != FrameHello {
			f.Build = &testBuild
		}
		if err := f.Validate(); err == nil {
			t.Errorf("%s allowed background evidence", op)
		}
	}
	for _, value := range []*bool{nil, boolPtr(false), boolPtr(true)} {
		f := SessionFrame{Op: FrameActivity, Observation: &Observation{}, BackgroundWork: value}
		raw, err := EncodeSessionFrame(f)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeSessionFrame(raw)
		if err != nil || !reflect.DeepEqual(got.BackgroundWork, value) {
			t.Fatalf("roundtrip %s: %+v %v", raw, got, err)
		}
	}
}

func boolPtr(v bool) *bool { return &v }

func backgroundFrames(log *sessionLog) []SessionFrame {
	log.mu.Lock()
	defer log.mu.Unlock()
	return append([]SessionFrame(nil), log.frames...)
}

func awaitBackgroundFrames(t *testing.T, log *sessionLog, n int) []SessionFrame {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		frames := backgroundFrames(log)
		if len(frames) >= n {
			return frames
		}
		select {
		case <-log.notify:
		case <-deadline:
			t.Fatalf("got %d frames, want %d", len(frames), n)
		}
	}
}

func TestBackgroundClientCopiesCoalescesAndClearsEvidence(t *testing.T) {
	socket := sessionSocket(t)
	log := newSessionLog()
	startSessionServer(t, socket, log)
	c := NewSessionClient()
	c.SetBuild(testBuild)
	evidence := true
	c.BackgroundWork(&evidence)
	evidence = false
	c.Settle(true)
	c.Picker(true)
	c.Update(Observation{Sequence: 1})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, socket, selfBinding("pair:1"), DefaultReconnectBackoff, 80*time.Millisecond)
	frames := awaitBackgroundFrames(t, log, 1)
	f := frames[0]
	if f.BackgroundWork == nil || !*f.BackgroundWork || f.Settled == nil || !*f.Settled || f.Picker == nil || !*f.Picker {
		t.Fatalf("missing independent evidence: %+v", f)
	}
	c.BackgroundWork(boolPtr(true))
	time.Sleep(120 * time.Millisecond)
	if got := len(backgroundFrames(log)); got != 1 {
		t.Fatalf("unchanged evidence sent %d frames", got)
	}
	c.BackgroundWork(boolPtr(false))
	frames = awaitBackgroundFrames(t, log, 2)
	if v := frames[1].BackgroundWork; v == nil || *v {
		t.Fatalf("false lost: %+v", frames[1])
	}
	// This burst lands inside the interval following that frame.
	c.BackgroundWork(boolPtr(true))
	c.BackgroundWork(boolPtr(false))
	c.BackgroundWork(boolPtr(true))
	c.BackgroundWork(nil)
	frames = awaitBackgroundFrames(t, log, 3)
	if frames[len(frames)-1].BackgroundWork != nil {
		t.Fatalf("nil did not clear evidence: %+v", frames)
	}
	c.BackgroundWork(nil)
	time.Sleep(120 * time.Millisecond)
	if got := len(backgroundFrames(log)); got != 3 {
		t.Fatalf("coalesced burst or unchanged nil sent %d frames", got)
	}
	c.BackgroundWork(boolPtr(false))
	frames = awaitBackgroundFrames(t, log, 4)
	if v := frames[len(frames)-1].BackgroundWork; v == nil || *v {
		t.Fatalf("false lost: %+v", frames)
	}
	c.Submit(Observation{Sequence: 2})
	frames = awaitBackgroundFrames(t, log, 5)
	if f := frames[len(frames)-1]; f.Op != FrameSubmit || f.BackgroundWork == nil || *f.BackgroundWork {
		t.Fatalf("submit evidence: %+v", f)
	}
}

func TestBackgroundClientReconnectResendsLatest(t *testing.T) {
	socket := sessionSocket(t)
	first := newSessionLog()
	server := startSessionServer(t, socket, first)
	c := NewSessionClient()
	c.SetBuild(testBuild)
	c.Update(Observation{Sequence: 1})
	c.BackgroundWork(boolPtr(true))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, socket, selfBinding("pair:1"), ReconnectBackoff{Base: 10 * time.Millisecond, Cap: 10 * time.Millisecond, StableAfter: time.Second}, time.Millisecond)
	awaitBackgroundFrames(t, first, 1)
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	c.BackgroundWork(boolPtr(false))
	second := newSessionLog()
	startSessionServer(t, socket, second)
	frames := awaitBackgroundFrames(t, second, 1)
	if v := frames[0].BackgroundWork; v == nil || *v {
		t.Fatalf("reconnected evidence=%v", v)
	}
}

func TestBackgroundOldSessionsRejectedIndividually(t *testing.T) {
	for _, op := range []string{FrameHello, FrameHelloV2, FrameHelloV3} {
		t.Run(op, func(t *testing.T) {
			socket := sessionSocket(t)
			log := newSessionLog()
			startSessionServer(t, socket, log)
			dial := func(version string) net.Conn {
				t.Helper()
				conn, err := net.Dial("unix", socket)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { conn.Close() })
				b := selfBinding("pair:1")
				f := SessionFrame{Op: version, Binding: &b}
				if version != FrameHello {
					f.Build = &testBuild
				}
				if err := writeSessionFrame(conn, f); err != nil {
					t.Fatal(err)
				}
				if ack, err := readSessionFrame(conn); err != nil || ack.Code != "ok" {
					t.Fatalf("ack=%+v %v", ack, err)
				}
				return conn
			}
			old := dial(op)
			healthy := dial(FrameHelloV4)
			if err := writeSessionFrame(old, SessionFrame{Op: FrameActivity, Observation: &Observation{Sequence: 1}, BackgroundWork: boolPtr(true)}); err != nil {
				t.Fatal(err)
			}
			log.waitFor(t, "closed 1")
			if err := writeSessionFrame(healthy, SessionFrame{Op: FrameActivity, Observation: &Observation{Sequence: 2}, Settled: boolPtr(true), Picker: boolPtr(true), BackgroundWork: boolPtr(true)}); err != nil {
				t.Fatal(err)
			}
			log.waitFor(t, "activity 2 2")
			if frames := backgroundFrames(log); len(frames) != 1 {
				t.Fatalf("old evidence forwarded: %+v", frames)
			}
		})
	}
}

// v3Frame freezes the strict pre-background schema rather than using the new type.
type v3Frame struct {
	Op          string
	Binding     *Binding       `json:",omitempty"`
	Observation *Observation   `json:",omitempty"`
	Code        string         `json:",omitempty"`
	Error       string         `json:",omitempty"`
	Build       *BuildIdentity `json:",omitempty"`
	Settled     *bool          `json:",omitempty"`
	Picker      *bool          `json:",omitempty"`
}

func TestBackgroundNegotiationOlderSchemas(t *testing.T) {
	versions := []string{FrameHelloV4, FrameHelloV3, FrameHelloV2, FrameHello}
	for accepted := 1; accepted < len(versions); accepted++ {
		t.Run(versions[accepted], func(t *testing.T) {
			socket := sessionSocket(t)
			l, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
			if err != nil {
				t.Fatal(err)
			}
			defer l.Close()
			hellos := make(chan []string, 1)
			frame := make(chan v3Frame, 1)
			failures := make(chan error, 1)
			go func() {
				var seen []string
				for {
					conn, err := l.AcceptUnix()
					if err != nil {
						return
					}
					raw, err := transportReadFrameLimit(conn, MaxSessionFrameBytes)
					var hello v3Frame
					if err == nil {
						err = strictjson.Decode(raw, &hello)
					}
					if err != nil {
						failures <- err
						conn.Close()
						return
					}
					seen = append(seen, hello.Op)
					if hello.Op != versions[accepted] {
						conn.Close()
						continue
					}
					hellos <- seen
					ack, _ := json.Marshal(legacyFrame{Op: FrameAck, Code: "ok"})
					if err = transportWriteFrame(conn, ack); err != nil {
						failures <- err
						conn.Close()
						return
					}
					raw, err = transportReadFrameLimit(conn, MaxSessionFrameBytes)
					var f v3Frame
					if err == nil {
						switch accepted {
						case 1:
							err = strictjson.Decode(raw, &f)
						case 2:
							var old v2Frame
							err = strictjson.Decode(raw, &old)
							f = v3Frame{Op: old.Op, Settled: old.Settled}
						case 3:
							var old legacyFrame
							err = strictjson.Decode(raw, &old)
							f.Op = old.Op
						}
					}
					if err != nil {
						failures <- err
					} else {
						frame <- f
					}
					conn.Close()
					return
				}
			}()
			c := NewSessionClient()
			c.SetBuild(testBuild)
			c.Settle(true)
			c.Picker(true)
			c.BackgroundWork(boolPtr(true))
			c.Update(Observation{Sequence: 1})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- c.session(ctx, socket, selfBinding("pair:1"), time.Millisecond) }()
			select {
			case got := <-hellos:
				if !reflect.DeepEqual(got, append([]string{FrameHelloV5}, versions[:accepted+1]...)) {
					t.Fatalf("hellos=%v", got)
				}
			case err := <-failures:
				t.Fatal(err)
			case <-time.After(3 * time.Second):
				t.Fatal("no hello")
			}
			select {
			case got := <-frame:
				if got.Op != FrameActivity || (accepted <= 2 && (got.Settled == nil || !*got.Settled)) || (accepted == 1 && (got.Picker == nil || !*got.Picker)) {
					t.Fatalf("old fields lost: %+v", got)
				}
			case err := <-failures:
				t.Fatal(err)
			case <-time.After(3 * time.Second):
				t.Fatal("no old-schema frame")
			}
			cancel()
			<-done
		})
	}
}

func TestBackgroundNegotiationRefusalAndMalformedAckDoNotFallback(t *testing.T) {
	for name, reply := range map[string][]byte{"refused": []byte(`{"Op":"ack","Code":"refused","Error":"no"}`), "malformed": []byte(`{"Op":"ack","Code":"ok","Unknown":true}`)} {
		t.Run(name, func(t *testing.T) {
			socket := sessionSocket(t)
			l, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
			if err != nil {
				t.Fatal(err)
			}
			defer l.Close()
			first := make(chan string, 1)
			go func() {
				conn, err := l.AcceptUnix()
				if err != nil {
					return
				}
				defer conn.Close()
				f, err := readSessionFrame(conn)
				if err != nil {
					return
				}
				first <- f.Op
				_ = transportWriteFrame(conn, reply)
			}()
			c := NewSessionClient()
			c.SetBuild(testBuild)
			if err := c.session(context.Background(), socket, selfBinding("pair:1"), time.Millisecond); err == nil || errors.Is(err, errNoAck) {
				t.Fatalf("refusal=%v", err)
			}
			if got := <-first; got != FrameHelloV5 {
				t.Fatalf("first hello=%s", got)
			}
			_ = l.SetDeadline(time.Now().Add(30 * time.Millisecond))
			if conn, err := l.AcceptUnix(); err == nil {
				conn.Close()
				t.Fatal("fallback after explicit ack")
			}
		})
	}
}

func TestBackgroundNegotiationTimeoutDoesNotFallback(t *testing.T) {
	socket := sessionSocket(t)
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	first := make(chan string, 1)
	release := make(chan struct{})
	defer close(release)
	go func() {
		conn, err := l.AcceptUnix()
		if err != nil {
			return
		}
		defer conn.Close()
		f, err := readSessionFrame(conn)
		if err != nil {
			return
		}
		first <- f.Op
		<-release
	}()
	c := NewSessionClient()
	c.SetBuild(testBuild)
	err = c.session(context.Background(), socket, selfBinding("pair:1"), time.Millisecond)
	var timeout net.Error
	if !errors.As(err, &timeout) || !timeout.Timeout() || errors.Is(err, errNoAck) {
		t.Fatalf("timeout=%v", err)
	}
	if got := <-first; got != FrameHelloV5 {
		t.Fatalf("first hello=%s", got)
	}
	_ = l.SetDeadline(time.Now().Add(30 * time.Millisecond))
	if conn, err := l.AcceptUnix(); err == nil {
		conn.Close()
		t.Fatal("fallback after timeout")
	}
}

func TestBackgroundEvidenceWithoutObservationWaits(t *testing.T) {
	socket := sessionSocket(t)
	log := newSessionLog()
	startSessionServer(t, socket, log)
	c := NewSessionClient()
	c.SetBuild(testBuild)
	c.BackgroundWork(boolPtr(true))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, socket, selfBinding("pair:1"), DefaultReconnectBackoff, time.Millisecond)
	select {
	case <-c.connected:
	case <-time.After(3 * time.Second):
		t.Fatal("no connection")
	}
	time.Sleep(30 * time.Millisecond)
	if n := len(backgroundFrames(log)); n != 0 {
		t.Fatalf("evidence invented observation: %d frames", n)
	}
	c.Update(Observation{Sequence: 9})
	f := awaitBackgroundFrames(t, log, 1)[0]
	if f.BackgroundWork == nil || !*f.BackgroundWork || f.Observation.Sequence != 9 {
		t.Fatalf("evidence lost: %+v", f)
	}
}
