package couchcore

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestLaunchTimingObserverReportsSuccessfulWarmResume(t *testing.T) {
	env, address := warmDetachedThread(t)
	env.Runner.AfterAcknowledge = func(string) error {
		env.Artifacts.SetPairSession(address, "pair-"+string(address.Tag), true)
		return nil
	}
	var names []string
	ctx := context.WithValue(t.Context(), launchTimingObserverKey{}, launchTimingObserver(func(name string, elapsed time.Duration) {
		if elapsed < 0 {
			t.Errorf("negative elapsed for %s: %s", name, elapsed)
		}
		names = append(names, name)
	}))
	if _, _, err := env.Couch.ResumeContextWith(ctx, address, ResumeOptions{WarmOnly: true}); err != nil {
		t.Fatal(err)
	}
	want := []string{"claim", "checkout", "prepare", "spawn", "record+baseline", "ack", "registration", "promotion+registry"}
	if !slices.Equal(names, want) {
		t.Fatalf("phases=%v want %v", names, want)
	}
	// Supplying an observer affects only this operation context. A later
	// ordinary launch must not report into the previous measurement.
	ordinary, ordinaryAddress := warmDetachedThread(t)
	ordinary.Runner.AfterAcknowledge = func(string) error {
		ordinary.Artifacts.SetPairSession(ordinaryAddress, "pair-"+string(ordinaryAddress.Tag), true)
		return nil
	}
	if _, _, err := ordinary.Couch.ResumeContextWith(t.Context(), ordinaryAddress, ResumeOptions{WarmOnly: true}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(names, want) {
		t.Fatalf("observer escaped operation context: %v", names)
	}
}

func TestLaunchTimingObserverPreservesRegistrationFailureTimings(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.resumeRegistrationTimeout = 150 * time.Millisecond
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	var names []string
	ctx := context.WithValue(t.Context(), launchTimingObserverKey{}, launchTimingObserver(func(name string, _ time.Duration) { names = append(names, name) }))
	_, _, err := env.Couch.ResumeContextWith(ctx, parked.Address, ResumeOptions{})
	text := retainedLaunchError(t, env.Couch, err)
	if !strings.Contains(text, "[steps: claim ") || !strings.Contains(text, ", registration ") {
		t.Fatalf("missing retained failure timings: %s", text)
	}
	want := []string{"claim", "checkout", "prepare", "spawn", "record+baseline", "ack", "registration"}
	if !slices.Equal(names, want) {
		t.Fatalf("failure phases=%v want %v", names, want)
	}
}

func TestDetachTimingObserverReportsStagesAndIsScopedToContext(t *testing.T) {
	f := newDetachFixture(t)
	var names []string
	ctx := context.WithValue(t.Context(), launchTimingObserverKey{}, launchTimingObserver(func(name string, elapsed time.Duration) {
		if elapsed < 0 {
			t.Errorf("negative elapsed for %s: %s", name, elapsed)
		}
		names = append(names, name)
	}))
	if _, err := f.couch.Detach(ctx, f.address); err != nil {
		t.Fatal(err)
	}
	want := []string{"claim", "read+owner", "signal", "process-exit", "retirement"}
	if !slices.Equal(names, want) {
		t.Fatalf("detach stages=%v want %v", names, want)
	}
	ordinary := newDetachFixture(t)
	if _, err := ordinary.couch.Detach(t.Context(), ordinary.address); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(names, want) {
		t.Fatalf("detach observer escaped its context: %v", names)
	}
}
