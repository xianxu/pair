package couchmessage

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// The four old/new pairings of #421's hello-v2 negotiation. "Old" peers are
// reproduced byte-for-byte from the pre-#421 rules: frames decode strictly into
// the old field set, an unknown op ends the connection without an ack, and an
// old wrapper treats any byte after the ack as loss.

var testBuild = BuildIdentity{SHA256: strings.Repeat("ab", 32), Revision: "abc123"}

// legacyFrame is the pre-#421 SessionFrame field set.
type legacyFrame struct {
	Op          string
	Binding     *Binding     `json:",omitempty"`
	Observation *Observation `json:",omitempty"`
	Code        string       `json:",omitempty"`
	Error       string       `json:",omitempty"`
}

// v2Frame is the #421 SessionFrame field set, before Picker (pair#465).
type v2Frame struct {
	Op          string
	Binding     *Binding       `json:",omitempty"`
	Observation *Observation   `json:",omitempty"`
	Code        string         `json:",omitempty"`
	Error       string         `json:",omitempty"`
	Build       *BuildIdentity `json:",omitempty"`
	Settled     *bool          `json:",omitempty"`
}

// startLegacyBroker accepts connections the way a pre-#421 broker does and
// records each frame it accepted after the ack.
func startLegacyBroker(t *testing.T, socket string) (frames func() []string) {
	return startOldBroker(t, socket, false)
}

// startOldBroker is a pre-#421 broker, or with v2 a #421 one: it accepts
// hello-v2 and decodes frames strictly with v2Frame's fields. Each accepted
// frame is recorded as its op, plus " settled" when it carried Settled.
func startOldBroker(t *testing.T, socket string, v2 bool) (frames func() []string) {
	t.Helper()
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	var mu sync.Mutex
	var seen []string
	go func() {
		for {
			conn, err := l.AcceptUnix()
			if err != nil {
				return
			}
			go func(conn *net.UnixConn) {
				defer conn.Close()
				raw, err := transportReadFrameLimit(conn, MaxSessionFrameBytes)
				var hello v2Frame
				if err != nil || strictjson.Decode(raw, &hello) != nil || !(hello.Op == FrameHello && hello.Build == nil || v2 && hello.Op == FrameHelloV2) {
					return // the old broker: drop without an ack
				}
				ack, _ := json.Marshal(legacyFrame{Op: FrameAck, Code: "ok"})
				if transportWriteFrame(conn, ack) != nil {
					return
				}
				for {
					raw, err := transportReadFrameLimit(conn, MaxSessionFrameBytes)
					var f v2Frame
					if err != nil || strictjson.Decode(raw, &f) != nil || (!v2 && f.Settled != nil) {
						return
					}
					op := f.Op
					if f.Settled != nil {
						op += " settled"
					}
					mu.Lock()
					seen = append(seen, op)
					mu.Unlock()
				}
			}(conn)
		}
	}()
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

func TestNegotiationNewWrapperNewBroker(t *testing.T) {
	socket := sessionSocket(t)
	log := newSessionLog()
	startSessionServer(t, socket, log)
	c := NewSessionClient()
	c.SetBuild(testBuild)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, socket, selfBinding("pair:1"), DefaultReconnectBackoff, time.Millisecond)
	log.waitFor(t, "open 1")
	c.Update(Observation{Sequence: 3})
	c.Settle(true)
	deadline := time.Now().Add(5 * time.Second)
	for {
		log.mu.Lock()
		var settled *bool
		if n := len(log.frames); n > 0 {
			settled = log.frames[n-1].Settled
		}
		builds := append([]BuildIdentity(nil), log.builds...)
		log.mu.Unlock()
		if len(builds) != 1 || builds[0] != testBuild {
			t.Fatalf("broker did not receive the build: %v", builds)
		}
		if settled != nil && *settled {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("settled never reached the broker")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestNegotiationNewWrapperOldBrokerFallsBack(t *testing.T) {
	socket := sessionSocket(t)
	frames := startLegacyBroker(t, socket)
	c := NewSessionClient()
	c.SetBuild(testBuild)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, socket, selfBinding("pair:1"), DefaultReconnectBackoff, time.Millisecond)
	select {
	case <-c.connected:
	case <-time.After(5 * time.Second):
		t.Fatal("never fell back to plain hello")
	}
	c.Settle(true)
	c.Update(Observation{Sequence: 4})
	deadline := time.Now().Add(5 * time.Second)
	for len(frames()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("the old broker never accepted an activity frame")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Strict old decoding accepted every frame, so none carried Settled.
	for _, op := range frames() {
		if op != FrameActivity && op != FrameSubmit {
			t.Fatalf("unexpected frame %q", op)
		}
	}
}

func TestNegotiationOldWrapperNewBroker(t *testing.T) {
	socket := sessionSocket(t)
	log := newSessionLog()
	startSessionServer(t, socket, log)
	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	b := selfBinding("pair:1")
	if err := writeSessionFrame(conn, SessionFrame{Op: FrameHello, Binding: &b}); err != nil {
		t.Fatal(err)
	}
	if ack, err := readSessionFrame(conn); err != nil || ack.Code != "ok" || ack.Build != nil {
		t.Fatalf("ack %+v %v", ack, err)
	}
	log.mu.Lock()
	if len(log.builds) != 0 {
		t.Fatal("a plain hello produced a build")
	}
	log.mu.Unlock()
	if err := writeSessionFrame(conn, SessionFrame{Op: FrameActivity, Observation: &Observation{Sequence: 5}}); err != nil {
		t.Fatal(err)
	}
	log.waitFor(t, "activity 1 5")
	// A legacy session that claims Settled is malformed: only that session ends.
	yes := true
	if err := writeSessionFrame(conn, SessionFrame{Op: FrameActivity, Observation: &Observation{Sequence: 6}, Settled: &yes}); err != nil {
		t.Fatal(err)
	}
	log.waitFor(t, "closed 1")
}

// pair#465: a new wrapper and a #421 broker (Couch not yet restarted) settle
// on hello-v2: Settled still flows, and Picker never reaches a broker whose
// strict decoding would end the session over it.
func TestNegotiationNewWrapperV2BrokerKeepsSettled(t *testing.T) {
	socket := sessionSocket(t)
	frames := startOldBroker(t, socket, true)
	c := NewSessionClient()
	c.SetBuild(testBuild)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, socket, selfBinding("pair:1"), DefaultReconnectBackoff, time.Millisecond)
	select {
	case <-c.connected:
	case <-time.After(5 * time.Second):
		t.Fatal("never fell back to hello-v2")
	}
	c.Update(Observation{Sequence: 4})
	c.Picker(true)
	c.Settle(true)
	deadline := time.Now().Add(5 * time.Second)
	for {
		got := frames()
		if n := len(got); n > 0 && got[n-1] == FrameActivity+" settled" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Settled never reached the hello-v2 broker: %v", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case <-c.connected:
		t.Fatal("the session was lost and redialed: a frame broke the v2 broker's decoding")
	default:
	}
}

func TestNegotiationPickerReachesNewBroker(t *testing.T) {
	socket := sessionSocket(t)
	log := newSessionLog()
	startSessionServer(t, socket, log)
	c := NewSessionClient()
	c.SetBuild(testBuild)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx, socket, selfBinding("pair:1"), DefaultReconnectBackoff, time.Millisecond)
	log.waitFor(t, "open 1")
	c.Update(Observation{Sequence: 3})
	c.Picker(true)
	deadline := time.Now().Add(5 * time.Second)
	for {
		log.mu.Lock()
		var picker *bool
		if n := len(log.frames); n > 0 {
			picker = log.frames[n-1].Picker
		}
		log.mu.Unlock()
		if picker != nil && *picker {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("Picker never reached the broker")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A hello-v2 session that claims Picker is malformed: only that session ends.
func TestHelloV2SessionClaimingPickerEnds(t *testing.T) {
	socket := sessionSocket(t)
	log := newSessionLog()
	startSessionServer(t, socket, log)
	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	b := selfBinding("pair:1")
	if err := writeSessionFrame(conn, SessionFrame{Op: FrameHelloV2, Binding: &b, Build: &testBuild}); err != nil {
		t.Fatal(err)
	}
	if ack, err := readSessionFrame(conn); err != nil || ack.Code != "ok" {
		t.Fatalf("ack %+v %v", ack, err)
	}
	yes := true
	if err := writeSessionFrame(conn, SessionFrame{Op: FrameActivity, Observation: &Observation{Sequence: 5}, Settled: &yes}); err != nil {
		t.Fatal(err)
	}
	log.waitFor(t, "activity 1 5")
	if err := writeSessionFrame(conn, SessionFrame{Op: FrameActivity, Observation: &Observation{Sequence: 6}, Picker: &yes}); err != nil {
		t.Fatal(err)
	}
	log.waitFor(t, "closed 1")
}

func TestHelloV2FrameValidation(t *testing.T) {
	b := selfBinding("pair:1")
	bad := testBuild
	bad.SHA256 = "XYZ"
	yes := true
	for _, tc := range []struct {
		f  SessionFrame
		ok bool
	}{
		{SessionFrame{Op: FrameHelloV2, Binding: &b, Build: &testBuild}, true},
		{SessionFrame{Op: FrameHelloV2, Binding: &b}, false},
		{SessionFrame{Op: FrameHelloV2, Binding: &b, Build: &bad}, false},
		{SessionFrame{Op: FrameHello, Binding: &b, Build: &testBuild}, false},
		{SessionFrame{Op: FrameHello, Binding: &b, Settled: &yes}, false},
		{SessionFrame{Op: FrameAck, Code: "ok", Settled: &yes}, false},
		{SessionFrame{Op: FrameActivity, Observation: &Observation{}, Settled: &yes}, true},
		{SessionFrame{Op: FrameActivity, Observation: &Observation{}, Build: &testBuild}, false},
		{SessionFrame{Op: FrameHelloV3, Binding: &b, Build: &testBuild}, true},
		{SessionFrame{Op: FrameHelloV3, Binding: &b}, false},
		{SessionFrame{Op: FrameHelloV3, Binding: &b, Build: &testBuild, Picker: &yes}, false},
		{SessionFrame{Op: FrameAck, Code: "ok", Picker: &yes}, false},
		{SessionFrame{Op: FrameActivity, Observation: &Observation{}, Picker: &yes}, true},
	} {
		if err := tc.f.Validate(); (err == nil) != tc.ok {
			t.Errorf("%+v: err=%v want ok=%v", tc.f, err, tc.ok)
		}
	}
}

// A slow new broker that does not ack in time is not an old broker: the client
// must not downgrade to plain hello (#421 M1 review).
func TestNegotiationTimeoutDoesNotFallBack(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	go func() { _, _ = transportReadFrameLimit(server, MaxSessionFrameBytes) }() // read hello-v2, never ack
	err := sessionHandshake(client, selfBinding("pair:1"), &testBuild, FrameHelloV2, nil)
	if err == nil || errors.Is(err, errNoAck) {
		t.Fatalf("timeout handled as an old broker: %v", err)
	}
}

// A broker that drops the hello and is gone before the fallback redial: the
// session ends with the dial error, and never closes a connection it does not
// hold (pair#465 close review: the redial reassigned the deferred conn to nil).
func TestNegotiationFallbackRedialFailureReturnsError(t *testing.T) {
	socket := sessionSocket(t)
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		conn, err := l.AcceptUnix()
		if err != nil {
			return
		}
		_, _ = transportReadFrameLimit(conn, MaxSessionFrameBytes)
		l.Close() // removes the socket: the redial fails
		conn.Close()
	}()
	c := NewSessionClient()
	c.SetBuild(testBuild)
	err = c.session(context.Background(), socket, selfBinding("pair:1"), time.Millisecond)
	if err == nil || errors.Is(err, errNoAck) {
		t.Fatalf("session = %v, want the redial's dial error", err)
	}
}

// Frozen v4 field set: a modern client must omit the cue from both its
// downgraded hello and every subsequent observation.
type workerCueLegacyFrame struct {
	Op             string
	Binding        *Binding       `json:",omitempty"`
	Observation    *Observation   `json:",omitempty"`
	Code           string         `json:",omitempty"`
	Error          string         `json:",omitempty"`
	Build          *BuildIdentity `json:",omitempty"`
	Settled        *bool          `json:",omitempty"`
	Picker         *bool          `json:",omitempty"`
	BackgroundWork *bool          `json:",omitempty"`
}

func TestWorkerCueNegotiatesDownWithoutLeakingFields(t *testing.T) {
	for _, accepted := range []string{FrameHello, FrameHelloV2, FrameHelloV3, FrameHelloV4} {
		t.Run(accepted, func(t *testing.T) {
			socket := sessionSocket(t)
			listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			result := make(chan error, 1)
			go func() {
				for {
					conn, err := listener.AcceptUnix()
					if err != nil {
						return
					}
					raw, err := transportReadFrameLimit(conn, MaxSessionFrameBytes)
					var hello workerCueLegacyFrame
					if err != nil || strictjson.Decode(raw, &hello) != nil || hello.Op != accepted {
						conn.Close()
						continue
					}
					ack, _ := json.Marshal(workerCueLegacyFrame{Op: FrameAck, Code: "ok"})
					if err := transportWriteFrame(conn, ack); err != nil {
						conn.Close()
						result <- err
						return
					}
					raw, err = transportReadFrameLimit(conn, MaxSessionFrameBytes)
					var activity workerCueLegacyFrame
					if err == nil {
						err = strictjson.Decode(raw, &activity)
					}
					if err == nil && (activity.Op != FrameActivity || activity.Observation == nil || activity.Observation.Sequence != 9) {
						err = errors.New("ordinary messaging lost on downgrade")
					}
					conn.Close()
					result <- err
					return
				}
			}()
			c := NewSessionClient()
			c.SetBuild(testBuild)
			c.WorkerCue(&WorkerCueReport{LaunchOrdinal: 7, State: WorkerCueNeeded})
			c.Update(Observation{Sequence: 9})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go c.Run(ctx, socket, selfBinding("pair:1"), DefaultReconnectBackoff, time.Millisecond)
			select {
			case err := <-result:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("no compatible downgraded observation")
			}
		})
	}
}

func TestWorkerCueValidationKeepsOldHelloStrict(t *testing.T) {
	b := selfBinding("pair:1")
	cue := &WorkerCueReport{LaunchOrdinal: 7, State: WorkerCueNeeded}
	for _, op := range []string{FrameHello, FrameHelloV2, FrameHelloV3, FrameHelloV4, FrameAck} {
		f := SessionFrame{Op: op, Binding: &b, Build: &testBuild, WorkerCue: cue, Code: "ok"}
		if _, err := EncodeSessionFrame(f); err == nil {
			t.Fatalf("%s accepted v5 evidence", op)
		}
	}
	for _, state := range []WorkerCueState{"", "unexpected"} {
		bad := *cue
		bad.State = state
		if _, err := EncodeSessionFrame(SessionFrame{Op: FrameHelloV5, Binding: &b, Build: &testBuild, WorkerCue: &bad}); err == nil {
			t.Fatal("unknown cue state accepted")
		}
	}
}
