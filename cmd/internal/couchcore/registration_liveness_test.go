package couchcore

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// pair#489: the registration wait is a hang ceiling, not a latency budget. At
// load ~200 a healthy start took ~20 s (a no-op make build 9.6 s, launch to
// pane 9-11 s) and the old fixed 15 s failed five relaunches. The ceiling has to
// clear that with room to spare; a dead launch no longer needs it, because its
// process group exiting ends the wait (the next two tests).
func TestRegistrationCeilingClearsAStartSlowedByLoad(t *testing.T) {
	if pairRegistrationTimeout < time.Minute {
		t.Fatalf("pairRegistrationTimeout = %s; a start measured at ~20 s under load needs a hang ceiling, not a latency budget", pairRegistrationTimeout)
	}
}

// A start that is alive and slow is waited for and adopted: Pair's pane is born
// only after an injected delay, many wait polls in, and the resume succeeds.
func TestColdResumeAdoptsALateStart(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	address := parked.Address
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	proposed, released := "", false
	env.Runner.AfterAcknowledge = func(id string) error {
		proposed = continuationChildSession(t, env.Runner, id)
		released = true
		return nil
	}
	var mu sync.Mutex
	var since time.Time
	env.Artifacts.PaneSidecarsHook = func(ThreadAddress) error {
		mu.Lock()
		defer mu.Unlock()
		if !released {
			return nil
		}
		if since.IsZero() {
			since = time.Now()
		}
		// The injected slow start: the pane appears only after several
		// helper-liveness checks have run, so the wait must have kept going.
		if time.Since(since) > 5*helperExitPoll {
			env.Artifacts.SetPaneSidecar(address, "claude")
			env.Artifacts.SetPairSession(address, proposed, true)
		}
		return nil
	}

	if _, _, err := env.Couch.Resume(address); err != nil {
		t.Fatalf("a slow but live start failed: %v", err)
	}
}

// A launch whose Pair exits before registering fails at once, says so, and
// leaves the thread parked and resumable: the cleanup is the ordinary post-ack
// one, so nothing is left for the operator to puzzle out.
func TestColdResumeFailsAtOnceWhenPairExitsBeforeRegistering(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	// Far beyond the test's patience: only the exit can end this wait.
	env.Couch.resumeRegistrationTimeout = time.Hour
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	address := parked.Address
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Runner.SetExited(id, 1) // Pair died during startup
		return nil
	}

	_, _, err := env.Couch.Resume(address)
	text := errorText(err)
	if !strings.Contains(text, "Pair exited before registering") {
		t.Fatalf("error %q does not name the exit", text)
	}
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(text, "(waited ") {
		t.Fatalf("error %q reads as a timeout; the exit ended the wait", text)
	}
	after, getErr := env.Couch.Threads.GetThread(address)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if hasOccupiedIncarnation(after) || after.VerifiedPark == nil {
		t.Fatalf("thread not left parked and resumable: occupied=%v park=%v", hasOccupiedIncarnation(after), after.VerifiedPark)
	}
}
