package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testcouch"
)

// Production-case measurement for #863. Uses only the landed fleet owner and
// the same form gesture as an operator; no manufactured broker credentials.
func TestDisposableFreshHubCaller(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_FLEET") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_FLEET=1")
	}
	binaries := testcouch.Build(t)
	t.Run("claude", func(t *testing.T) {
		fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 2, ExplicitHub: true, Hubs: []int{1}, AgentKind: "claude", RuntimeBudget: 4 * time.Minute, RetainStartupServerLog: true})
		setup, stopSetup := context.WithTimeout(t.Context(), 180*time.Second)
		err := fleet.Prepare(setup)
		stopSetup()
		if err != nil {
			t.Fatal(err)
		}
		ctx, stop := context.WithTimeout(t.Context(), 210*time.Second)
		defer stop()
		startCtx, stopStart := context.WithCancel(ctx)
		started := make(chan error, 1)
		go func() { started <- fleet.Start(startCtx) }()
		defer func() { stopStart(); t.Logf("initial Start observer: %v", <-started) }()
		forceFleetWaitScreen(t, ctx, fleet, "Ctrl+Space: start")
		t.Logf("cold screen:\n%s", fleet.Screen())
		forceFleetPress(t, fleet, "\x00", 32, 5)
		forceFleetWaitScreen(t, ctx, fleet, "start thread")
		if err := fleet.Input([]byte(fleet.Workspaces.Primary)); err != nil {
			t.Fatal(err)
		}
		forceFleetPress(t, fleet, "\x1b[B", 57353, 1)
		if err := waitFleet(ctx, func() (bool, error) {
			s := fleet.Screen()
			return strings.Contains(s, "args  repo-default"), nil
		}); err != nil {
			t.Fatalf("start preview: %v\n%s", err, fleet.Screen())
		}
		t.Logf("start form:\n%s", fleet.Screen())
		forceFleetPress(t, fleet, "\r", 13, 1)
		if err := fleet.WaitAgent(ctx, 0); err != nil {
			t.Fatalf("agent readiness: %v\n%s", err, fleet.Screen())
		}
		ready, err := fleet.Agent(0)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("ready: %+v", ready)
		// Start's waiting observer has no native lifetime authority. Cancel/join it
		// before examining the caller, so no second query producer remains.
		stopStart()
		startErr := <-started
		started <- startErr
		t.Logf("joined startup observer: %v", startErr)
		for observation := 0; observation < 65; observation++ {
			result, err := fleet.ActorCLI(ctx, 0, "--actors", "--json")
			t.Logf("caller observation %d: result=%+v err=%v", observation, result, err)
			if err == nil && result.ExitCode == 0 && !result.CleanupUncertain {
				t.Log("fresh explicit hub is a live caller")
				return
			}
			if result.CleanupUncertain || result.ID == "" {
				t.Fatalf("caller observation uncertain: %+v %v", result, err)
			}
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		}
		t.Fatalf("ready fresh hub never became a live caller in 65 observations; screen:\n%s", fleet.Screen())
	})
}
