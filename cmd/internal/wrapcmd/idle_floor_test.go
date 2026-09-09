package wrapcmd

import (
	"io"
	"os"
	"path/filepath"
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
		// Suppress the fire-and-forget slug subprocess emitOuter would spawn.
		lastSlug: time.Now(),
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

func TestIdleFloorExpiryYieldsToABoundaryQueuedBeforeIt(t *testing.T) {
	// The adversarial case for the master-loop select: a completion is
	// published but not yet reduced when the idle deadline expires. Arrival
	// order is injected — the completion is enqueued and the expiry is then
	// allowed to fire — rather than sampled from a real race. The expiry
	// branch drains lifecycleEvents first, so the completion wins and the
	// floor must not alert against a turn that has just been reported.
	h := newIdleFloorHarness(t, "claude", "marker", 60*time.Millisecond)
	h.proxy.publishLifecycleObservation(TurnObservation{Kind: ObservationUserSubmission})
	time.Sleep(20 * time.Millisecond)
	h.proxy.publishLifecycleObservation(TurnObservation{
		Kind: ObservationMarkerCompletion, Message: "agent finished working",
	})
	got := h.settle(400 * time.Millisecond)
	if len(got) != 1 {
		t.Fatalf("notifications = %d (%q), want exactly 1", len(got), got)
	}
	if !strings.Contains(got[0], "finished working") {
		t.Fatalf("notification = %q, want the completion, not the idle alert", got[0])
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
