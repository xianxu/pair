package wrapcmd

import (
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/orientation"
)

func peerTestBinding() couchmessage.Binding { return couchmessage.Binding{Slot: "pair:1"} }

func TestWrapperSettledRule(t *testing.T) {
	// Exhaustive over the fact space: settled only when every fact is clear.
	for _, composer := range []PeerComposerState{PeerComposerUnknown, PeerComposerOccupied, PeerComposerEmpty} {
		for _, turn := range []bool{false, true} {
			for _, picker := range []bool{false, true} {
				for _, orienting := range []bool{false, true} {
					want := composer == PeerComposerEmpty && !turn && !picker && !orienting
					if got := wrapperSettled(composer, turn, picker, orienting); got != want {
						t.Errorf("composer=%v turn=%v picker=%v orienting=%v: got %v", composer, turn, picker, orienting, got)
					}
				}
			}
		}
	}
}

// fakeTimers is an injected clock: armed callbacks run only when the test fires
// them, so interleavings are exact rather than timing-dependent.
type fakeTimers struct {
	mu    sync.Mutex
	armed []*fakeTimer
}
type fakeTimer struct {
	f       func()
	stopped bool
}

func (t *fakeTimer) Stop() bool { was := !t.stopped; t.stopped = true; return was }
func (c *fakeTimers) after(d time.Duration, f func()) settleTimer {
	c.mu.Lock()
	defer c.mu.Unlock()
	if d != SettleInterval {
		panic("unexpected interval")
	}
	t := &fakeTimer{f: f}
	c.armed = append(c.armed, t)
	return t
}

// fireAll runs every armed timer, stopped ones included: a Stop that loses the
// race to the callback must still be harmless.
func (c *fakeTimers) fireAll() {
	c.mu.Lock()
	armed := c.armed
	c.armed = nil
	c.mu.Unlock()
	for _, t := range armed {
		t.f()
	}
}

func TestSettleTimerInterleavedActivity(t *testing.T) {
	clock := &fakeTimers{}
	sink := &recordingPeerSink{}
	idle := true
	d := newPeerDelivery(peerTestBinding(), time.Now)
	d.afterFunc = clock.after
	d.settleProbe = func() bool { return idle }
	d.session = sink
	d.mu.Lock()
	d.armSettleLocked()
	d.mu.Unlock()

	clock.fireAll() // quiet through the interval, probe idle: settles
	d.observeOutput([]byte("x"))
	d.outputForwarded() // production: handleChunk forwards, then reports it
	// The pre-activity timer (already armed for the old sequence) firing late
	// must not re-settle; only the timer armed by the activity may.
	d.settleFired(1) // the generation armed before the output: stale
	idle = false
	clock.fireAll() // busy at check time: stays unsettled
	d.admitInput([]byte("a"))
	d.inputForwarded(true, false)
	idle = true
	d.observeOutput([]byte("y")) // activity inside the interval re-arms
	clock.mu.Lock()
	early := len(clock.armed)
	clock.mu.Unlock()
	if early == 0 {
		t.Fatal("activity did not re-arm the settle check")
	}
	d.mu.Lock()
	if d.settled {
		t.Fatal("settled while output is unforwarded")
	}
	d.mu.Unlock()
	d.outputForwarded()
	clock.fireAll()

	sink.mu.Lock()
	got := append([]bool(nil), sink.settles...)
	sink.mu.Unlock()
	if want := []bool{true, false, true}; !reflect.DeepEqual(got, want) {
		t.Fatalf("settle transitions %v, want %v", got, want)
	}
}

func TestSettleNeverWithoutProbe(t *testing.T) {
	clock := &fakeTimers{}
	d := newPeerDelivery(peerTestBinding(), time.Now)
	d.afterFunc = clock.after
	d.mu.Lock()
	d.armSettleLocked()
	d.mu.Unlock()
	if len(clock.armed) != 0 {
		t.Fatal("armed a settle check with no probe")
	}
}

// #421 M1 review BR-2: a turn can open or close with no PTY bytes (watchdog,
// grace, transcript). Opening must unsettle at once; closing must re-arm.
func TestSettleFollowsSilentTurnTransitions(t *testing.T) {
	clock := &fakeTimers{}
	sink := &recordingPeerSink{}
	turn := false
	d := newPeerDelivery(peerTestBinding(), time.Now)
	d.afterFunc = clock.after
	d.settleProbe = func() bool { return !turn }
	d.session = sink
	d.mu.Lock()
	d.armSettleLocked()
	d.mu.Unlock()
	clock.fireAll() // idle: settled

	turn = true
	d.lifecycleTurnChanged(true) // opened silently
	d.mu.Lock()
	settled := d.settled
	d.mu.Unlock()
	if settled {
		t.Fatal("a silently opened turn left the wrapper settled")
	}
	clock.fireAll() // the re-armed check sees the open turn: stays unsettled

	turn = false
	d.lifecycleTurnChanged(false) // closed silently: re-armed
	clock.mu.Lock()
	armed := len(clock.armed)
	clock.mu.Unlock()
	if armed == 0 {
		t.Fatal("a silently closed turn did not re-arm the settle check")
	}
	clock.fireAll()

	sink.mu.Lock()
	got := append([]bool(nil), sink.settles...)
	sink.mu.Unlock()
	if want := []bool{true, false, true}; !reflect.DeepEqual(got, want) {
		t.Fatalf("settle transitions %v, want %v", got, want)
	}
}

// The rule behind both M1 review findings: a check in flight when ANY source
// moves must not publish. Here the turn opens silently while the probe runs.
func TestSettleInFlightCheckInvalidatedBySilentTurn(t *testing.T) {
	clock := &fakeTimers{}
	sink := &recordingPeerSink{}
	var d *peerDelivery
	d = newPeerDelivery(peerTestBinding(), time.Now)
	d.afterFunc = clock.after
	d.settleProbe = func() bool {
		d.lifecycleTurnChanged(true) // races in between probe and publish
		return true
	}
	d.session = sink
	d.mu.Lock()
	d.armSettleLocked()
	timers := clock.armed
	clock.armed = nil
	d.mu.Unlock()
	timers[0].f()
	d.mu.Lock()
	settled := d.settled
	d.mu.Unlock()
	if settled {
		t.Fatal("a check in flight across a silent turn open published Settled")
	}
	sink.mu.Lock()
	defer sink.mu.Unlock()
	for _, v := range sink.settles {
		if v {
			t.Fatalf("published Settled=true: %v", sink.settles)
		}
	}
}

// pair#427: a restart reports ready only once its session is briefed, so a
// pending orientation keeps the wrapper unsettled. Orientation can finalize
// with no output (its deadline), so finalizing must re-arm the check.
func TestSettleWaitsForOrientationAndFinalizeRearms(t *testing.T) {
	p, _ := orientationProxy(t)
	var rolling []byte
	p.handleChunk([]byte(strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1)+"\x1b[21;3H"), &rolling)
	if p.settledNow() {
		t.Fatal("settled while the orientation is pending")
	}
	clock := &fakeTimers{}
	d := newPeerDelivery(peerTestBinding(), time.Now)
	d.afterFunc = clock.after
	d.settleProbe = p.settledNow
	p.peer = d
	settle := time.NewTimer(time.Hour)
	defer settle.Stop()
	p.advanceOrientation(orientation.DeliveryEvent{Kind: orientation.ChildExited}, io.Discard, settle)
	clock.mu.Lock()
	armed := len(clock.armed)
	clock.mu.Unlock()
	if armed == 0 {
		t.Fatal("finalizing the orientation did not re-arm the settle check")
	}
	if !p.settledNow() {
		t.Fatal("still unsettled after the orientation finalized")
	}
}
