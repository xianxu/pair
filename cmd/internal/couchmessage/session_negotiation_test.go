package couchmessage

import (
	"context"
	"encoding/json"
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

// startLegacyBroker accepts connections the way a pre-#421 broker does and
// records each frame it accepted after the ack.
func startLegacyBroker(t *testing.T, socket string) (frames func() []string) {
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
				var hello legacyFrame
				if err != nil || strictjson.Decode(raw, &hello) != nil || hello.Op != FrameHello {
					return // the old broker: drop without an ack
				}
				ack, _ := json.Marshal(legacyFrame{Op: FrameAck, Code: "ok"})
				if transportWriteFrame(conn, ack) != nil {
					return
				}
				for {
					raw, err := transportReadFrameLimit(conn, MaxSessionFrameBytes)
					var f legacyFrame
					if err != nil || strictjson.Decode(raw, &f) != nil {
						return
					}
					mu.Lock()
					seen = append(seen, f.Op)
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
	} {
		if err := tc.f.Validate(); (err == nil) != tc.ok {
			t.Errorf("%+v: err=%v want ok=%v", tc.f, err, tc.ok)
		}
	}
}
