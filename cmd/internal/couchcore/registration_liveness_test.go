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
// pane 9-11 s) and the old fixed 15 s failed five relaunches. A start that is
// alive and slow is waited for and adopted: Pair's pane is born only many wait
// polls in, under a deadline the launch armed at least a minute out. A dead
// launch no longer needs that deadline; its exit ends the wait (next test).
func TestColdResumeAdoptsALateStart(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	address := parked.Address
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	var armed time.Duration
	env.Couch.registrationDeadline = func(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
		armed = d
		return context.WithTimeout(parent, d)
	}
	proposed, released := "", false
	env.Runner.AfterAcknowledge = func(id string) error {
		proposed = continuationChildSession(t, env.Runner, id)
		released = true
		return nil
	}
	var mu sync.Mutex
	polls := 0
	env.Artifacts.PaneSidecarsHook = func(ThreadAddress) error {
		mu.Lock()
		defer mu.Unlock()
		if !released {
			return nil
		}
		// The injected slow start: the pane appears only on the 50th look.
		if polls++; polls == 50 {
			env.Artifacts.SetPaneSidecar(address, "claude")
			env.Artifacts.SetPairSession(address, proposed, true)
		}
		return nil
	}

	if _, _, err := env.Couch.Resume(address); err != nil {
		t.Fatalf("a slow but live start failed: %v", err)
	}
	if polls < 50 {
		t.Fatalf("registered after %d looks; the late start was not waited for", polls)
	}
	if armed < time.Minute {
		t.Fatalf("registration deadline = %s; a start measured at ~20 s under load needs a hang ceiling", armed)
	}
}

// A launch whose Pair exits before registering fails at once, says so, and
// leaves the thread parked and resumable: the cleanup is the ordinary post-ack
// one, so nothing is left for the operator to puzzle out. Only the direct child
// dies: a Couch-launched Pair's title poller stays in its process group for up
// to 30 s, and must not hold the wait open (close review BR-1).
func TestColdResumeFailsAtOnceWhenPairExitsBeforeRegistering(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	// Far beyond the test's patience: only the exit can end this wait.
	env.Couch.resumeRegistrationTimeout = time.Hour
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	address := parked.Address
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Runner.SetChildExited(id) // Pair died; its sidecar's group lives on
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
