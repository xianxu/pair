package wrapcmd

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// The idle floor's master-loop behaviour (#171). These rows enter through the
// production scheduling owner — p.masterPump() over an os.Pipe ptmx, the same
// seam TestMasterPumpForwardsPTYWhileLifecycleJournalIOIsBlocked uses — and
// observe the emit at the outer-TTY write seam. p.idleS is milliseconds, so
// nothing here waits on wall-clock.

type idleFloorHarness struct {
	proxy   *proxy
	writer  *os.File
	done    chan struct{}
	mu      sync.Mutex
	emitted []string
}

func newIdleFloorHarness(t *testing.T, agent, mode string, idle time.Duration) *idleFloorHarness {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	outer := filepath.Join(dir, "outer")
	if err := os.WriteFile(outer, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	sidecar := filepath.Join(dir, "outer-path")
	if err := os.WriteFile(sidecar, []byte(outer+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := &idleFloorHarness{writer: writer, done: make(chan struct{})}
	h.proxy = &proxy{
		ptmx: reader, agentBasename: agent, notifyModeActive: mode,
		stdoutPump: newStdoutPump(io.Discard), stdoutFlushEvery: 5 * time.Millisecond,
		captureWindow: defaultCaptureWindow, now: time.Now,
		lifecycleEvents: make(chan TurnObservation, 32),
		outerTTYFile:    sidecar,
		idleS:           idle,
		// Injected rather than debounced: this test's emit lands after a live
		// timer, so a wall-clock debounce could lapse on a loaded machine and
		// spawn a real slug run against the operator's machine (BR-11).
		spawnSlug: func() {},
	}
	h.proxy.writeTTY = func(_ int, data []byte) (int, error) {
		h.mu.Lock()
		h.emitted = append(h.emitted, string(data))
		h.mu.Unlock()
		return len(data), nil
	}
	go func() {
		h.proxy.masterPump()
		close(h.done)
	}()
	t.Cleanup(func() {
		_ = writer.Close()
		select {
		case <-h.done:
		case <-time.After(time.Second):
			t.Error("masterPump did not exit")
		}
		_ = reader.Close()
	})
	return h
}

func (h *idleFloorHarness) notifications() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.emitted...)
}

// settle waits out several idle intervals so a second (unwanted) alert would
// have had time to fire, then reports what was emitted.
func (h *idleFloorHarness) settle(d time.Duration) []string {
	time.Sleep(d)
	return h.notifications()
}

func TestIdleFloorArmsForEveryNotifyModeNotJustOneNothingSelects(t *testing.T) {
	// The bug this issue fixes: the timer was gated on a notify mode no agent
	// was ever assigned, so it never armed in any shipped configuration.
	for _, agent := range []struct{ name, mode string }{
		{"claude", "marker"},
		{"codex", notifyModeDefault},
	} {
		t.Run(agent.name+"/"+agent.mode, func(t *testing.T) {
			h := newIdleFloorHarness(t, agent.name, agent.mode, 40*time.Millisecond)
			h.proxy.publishLifecycleObservation(TurnObservation{Kind: ObservationUserSubmission})
			got := h.settle(400 * time.Millisecond)
			if len(got) != 1 {
				t.Fatalf("notifications = %d (%q), want exactly 1", len(got), got)
			}
			// Also confirms the honest message survives the OSC 777 envelope.
			if !strings.HasPrefix(got[0], "\x1b]777;notify;pair;no agent output for ") {
				t.Fatalf("notification = %q, want an OSC 777 reporting silence", got[0])
			}
		})
	}
}

func TestIdleFloorStaysSilentWhenTheIntervalIsZero(t *testing.T) {
	h := newIdleFloorHarness(t, "claude", "marker", 0)
	h.proxy.publishLifecycleObservation(TurnObservation{Kind: ObservationUserSubmission})
	if got := h.settle(200 * time.Millisecond); len(got) != 0 {
		t.Fatalf("PAIR_WRAP_IDLE_S=0 still notified: %q", got)
	}
}

func TestIdleFloorStaysSilentWhileTheAgentIsProducingOutput(t *testing.T) {
	// Output pushes the deadline out, so a working pane is never alerted on.
	h := newIdleFloorHarness(t, "claude", "marker", 60*time.Millisecond)
	h.proxy.publishLifecycleObservation(TurnObservation{Kind: ObservationUserSubmission})
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, err := h.writer.Write([]byte("working\r\n")); err != nil {
			t.Fatal(err)
		}
		time.Sleep(15 * time.Millisecond)
	}
	if got := h.notifications(); len(got) != 0 {
		t.Fatalf("floor alerted on a pane that was still producing output: %q", got)
	}
}

// newIdleExpiryProxy builds a proxy with real (stopped) timers but no running
// master loop, so the expiry/boundary interleaving can be INJECTED rather than
// raced: queue the observation, then apply the expiry.
func newIdleExpiryProxy(t *testing.T) (*proxy, func() []string) {
	t.Helper()
	dir := t.TempDir()
	outer := filepath.Join(dir, "outer")
	if err := os.WriteFile(outer, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	sidecar := filepath.Join(dir, "outer-path")
	if err := os.WriteFile(sidecar, []byte(outer+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var emitted []string
	p := &proxy{
		agentBasename: "claude", notifyModeActive: "marker", now: time.Now,
		lifecycleEvents: make(chan TurnObservation, 32),
		outerTTYFile:    sidecar, idleS: time.Minute, spawnSlug: func() {},
	}
	p.writeTTY = func(_ int, data []byte) (int, error) {
		emitted = append(emitted, string(data))
		return len(data), nil
	}
	p.idleTimer = time.NewTimer(time.Hour)
	drainStop(p.idleTimer)
	p.lifecycleTimer = time.NewTimer(time.Hour)
	drainStop(p.lifecycleTimer)
	t.Cleanup(func() { p.idleTimer.Stop(); p.lifecycleTimer.Stop() })
	return p, func() []string { return emitted }
}

func TestIdleExpiryDoesNotAlertAgainstATurnOpenedByItsOwnDrain(t *testing.T) {
	// BR-2. The drain reduces a queued opener, which mints a new turn and
	// advances p.idleTimerToken in lockstep. Reading the token after the drain
	// matches that brand-new turn: it alerts microseconds after the turn opened
	// AND consumes its floor. The expired epoch must be snapshotted first.
	p, emitted := newIdleExpiryProxy(t)
	p.processLifecycleObservation(TurnObservation{Kind: ObservationUserSubmission})
	expired := p.idleTimerToken
	first := p.notificationLifecycle.Generation

	p.publishLifecycleObservation(TurnObservation{Kind: ObservationUserSubmission})
	p.applyIdleExpiry(expired)

	if got := emitted(); len(got) != 0 {
		t.Fatalf("alerted against a turn opened by its own drain: %q", got)
	}
	state := p.notificationLifecycle
	if state.Generation == first {
		t.Fatalf("drain did not open the queued turn: %+v", state)
	}
	if state.IdleNotified || state.IdleToken == 0 || p.idleTimerToken != state.IdleToken {
		t.Fatalf("the newly opened turn lost its floor: state %+v armed token %d", state, p.idleTimerToken)
	}
}

func TestIdleExpiryDrainAppliesAQueuedCompletionFirst(t *testing.T) {
	// BR-3. Pins the drain itself: with it removed, the idle alert wins and the
	// completion is never reduced, so this row fails.
	p, emitted := newIdleExpiryProxy(t)
	p.processLifecycleObservation(TurnObservation{Kind: ObservationUserSubmission})
	expired := p.idleTimerToken

	p.publishLifecycleObservation(TurnObservation{
		Kind: ObservationMarkerCompletion, Message: "agent finished working",
	})
	p.applyIdleExpiry(expired)

	got := emitted()
	if len(got) != 1 {
		t.Fatalf("notifications = %d (%q), want exactly 1", len(got), got)
	}
	if !strings.Contains(got[0], "finished working") {
		t.Fatalf("notification = %q, want the completion, not the idle alert", got[0])
	}
}

func TestIdleExpiryGivesTheLifecycleDeadlinePrecedenceWhenBothAreReady(t *testing.T) {
	// BR-6. Both deadlines are 60s and sit in sibling select cases, so Go picks
	// at random when they are co-ready; the 0.5s emit limiter would then drop
	// whichever lost. The informative message must win.
	p, emitted := newIdleExpiryProxy(t)
	p.processLifecycleObservation(TurnObservation{Kind: ObservationWorking})
	p.processLifecycleObservation(TurnObservation{Kind: ObservationStopped})
	expired := p.idleTimerToken
	// Make the grace deadline genuinely ready alongside the idle one.
	drainStop(p.lifecycleTimer)
	p.lifecycleTimer.Reset(0)
	time.Sleep(10 * time.Millisecond)

	p.applyIdleExpiry(expired)

	got := emitted()
	if len(got) != 1 {
		t.Fatalf("notifications = %d (%q), want exactly 1", len(got), got)
	}
	if !strings.Contains(got[0], "stopped working") {
		t.Fatalf("notification = %q, want the lifecycle message to win", got[0])
	}
}

func TestIdleAlertMessageReportsTheIntervalHonestlyAtAnyScale(t *testing.T) {
	// BR-10: "%.0fs" rendered every sub-second interval as "for 0s".
	for _, test := range []struct {
		interval time.Duration
		want     string
	}{
		{60 * time.Second, "no agent output for 60s"},
		{90 * time.Second, "no agent output for 90s"},
		{40 * time.Millisecond, "no agent output for 40ms"},
	} {
		if got := idleAlertMessage(test.interval); got != test.want {
			t.Errorf("idleAlertMessage(%v) = %q, want %q", test.interval, got, test.want)
		}
	}
}

// The knob itself, so "PAIR_WRAP_IDLE_S=0 disables the floor" is evidence
// rather than inference: envDuration parses "0" as a real zero rather than
// falling through to the default, which is what makes the opt-out reachable.
func TestIdleIntervalKnobParsesAnExplicitZeroAsDisabled(t *testing.T) {
	for _, test := range []struct {
		name, value string
		set         bool
		want        time.Duration
	}{
		{"unset falls back to the default", "", false, defaultIdleS},
		{"explicit zero disables", "0", true, 0},
		{"explicit interval wins", "5", true, 5 * time.Second},
		{"unparseable falls back to the default", "banana", true, defaultIdleS},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.set {
				t.Setenv("PAIR_WRAP_IDLE_S", test.value)
			} else {
				os.Unsetenv("PAIR_WRAP_IDLE_S")
			}
			if got := envDuration("PAIR_WRAP_IDLE_S", defaultIdleS); got != test.want {
				t.Fatalf("envDuration = %v, want %v", got, test.want)
			}
		})
	}
}

// ----- The three deliverables a mutation sweep found unpinned (BR-13) --------

func TestResolveNotifyConfigNeverZeroesTheIdleInterval(t *testing.T) {
	// The removed gate zeroed idleS for every mode except one nothing was ever
	// assigned, which is what made the floor dead code. Its absence is only
	// falsifiable through this seam: re-adding the gate reddens this row.
	for _, test := range []struct {
		agent, wantMode string
		wantMarkerRe    bool
	}{
		{"claude", "marker", true},
		{"codex", notifyModeDefault, false},
		{"agy", notifyModeDefault, false},
		{"some-unknown-agent", notifyModeDefault, false},
	} {
		t.Run(test.agent, func(t *testing.T) {
			got := resolveNotifyConfig(test.agent, defaultIdleS)
			if got.mode != test.wantMode {
				t.Errorf("mode = %q, want %q", got.mode, test.wantMode)
			}
			if got.idleS != defaultIdleS {
				t.Errorf("idleS = %v, want %v — the floor must not depend on notify mode",
					got.idleS, defaultIdleS)
			}
			if (got.endOfTurnRe != nil) != test.wantMarkerRe {
				t.Errorf("endOfTurnRe present = %v, want %v", got.endOfTurnRe != nil, test.wantMarkerRe)
			}
		})
	}
	if got := resolveNotifyConfig("claude", 0); got.idleS != 0 {
		t.Errorf("an explicitly disabled floor was resurrected: %v", got.idleS)
	}
}

func drainObservations(ch chan TurnObservation) []ObservationKind {
	var kinds []ObservationKind
	for {
		select {
		case observation := <-ch:
			kinds = append(kinds, observation.Kind)
		default:
			return kinds
		}
	}
}

func TestEmitPlainCRPublishesOnlyWhenTheReturnReachesTheAgent(t *testing.T) {
	// No test drove a plain Enter through emitPlainCR, so deleting the publish
	// left the package green while the floor lost its bare-CR opener entirely.
	unknownGate := harnessTTYProfile{
		keymap: sendKeymap{plainCR: []byte{'\n'}}, composerGate: composerGateUnknown,
	}
	activeComposer := harnessTTYProfile{
		keymap: sendKeymap{plainCR: []byte{'\n'}}, composerGate: composerGateLegacy,
	}
	for _, test := range []struct {
		name    string
		profile *harnessTTYProfile
		overlay bool
		want    []ObservationKind
	}{
		{"composer unknown reaches the agent", &unknownGate, false, []ObservationKind{ObservationBareReturn}},
		{"overlay confirm is pair-local", &unknownGate, true, nil},
		{"composer newline remap is not a submission", &activeComposer, false, nil},
		{"nil profile passes the CR straight through", nil, false, []ObservationKind{ObservationBareReturn}},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := &proxy{ttyProfile: test.profile, lifecycleEvents: make(chan TurnObservation, 8)}
			p.pickerActive.Store(test.overlay)
			p.emitPlainCR(nil)
			if got := drainObservations(p.lifecycleEvents); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("published %v, want %v", got, test.want)
			}
		})
	}
}

func TestPassThroughChunkOpensATurnOnACarriageReturn(t *testing.T) {
	// BR-14: without a remap profile this is the ONLY turn-opening signal, so
	// PAIR_WRAP_REMAP_RETURN=0 and any agent outside harnessTTYProfiles would
	// otherwise have no floor at all.
	for _, test := range []struct {
		name string
		data string
		want []ObservationKind
	}{
		{"carriage return submits", "answer\r", []ObservationKind{ObservationBareReturn}},
		{"ordinary keystrokes do not", "answer", nil},
		{"newline alone does not", "answer\n", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := &proxy{lifecycleEvents: make(chan TurnObservation, 8)}
			out, leftover := p.passThroughChunk([]byte(test.data))
			if string(out) != test.data || leftover != nil {
				t.Fatalf("pass-through altered the stream: out=%q leftover=%q", out, leftover)
			}
			if got := drainObservations(p.lifecycleEvents); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("published %v, want %v", got, test.want)
			}
		})
	}
}

func TestSyncIdleTimerDoesNotRestartAnArmedDeadlineWithinTheSameEpoch(t *testing.T) {
	// BR-10's behavioural half. An unconditional resetIdleTimer would restart
	// the deadline on any reduced observation — including a journal record
	// carrying no pane output — silently extending a window whose message
	// claims byte-silence.
	//
	// Observed by receiving the pending expiry. Go 1.23 timer channels are
	// unbuffered, so len(C) is always 0 and Stop() still reports true for an
	// expiry nothing has received yet — neither can see this. A restart, by
	// contrast, drains that pending expiry and re-arms, so widening the
	// interval first makes the difference unambiguous: the original expiry is
	// still deliverable, a restarted one is an hour away.
	p, _ := newIdleExpiryProxy(t)
	p.idleS = time.Millisecond
	p.processLifecycleObservation(TurnObservation{Kind: ObservationUserSubmission})
	epoch := p.idleTimerToken
	if epoch == 0 {
		t.Fatal("submission did not arm the floor")
	}
	time.Sleep(50 * time.Millisecond) // the 1ms deadline has certainly expired

	p.idleS = time.Hour
	// A non-output observation that keeps the same turn — and so the same epoch.
	p.processLifecycleObservation(TurnObservation{Kind: ObservationWorking})
	if p.idleTimerToken != epoch {
		t.Fatalf("epoch changed: %d -> %d", epoch, p.idleTimerToken)
	}
	select {
	case <-p.idleTimer.C:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("a non-output observation restarted an already-expired deadline")
	}
}
