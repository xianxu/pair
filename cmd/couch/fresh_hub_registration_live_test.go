package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testcouch"
)

// Both configured hubs must actually register, independently of native fixture
// readiness. The shared owner supplies all processes and exact cleanup.
func TestDisposableFreshHubCaller(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_FLEET") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_FLEET=1")
	}
	binaries := testcouch.Build(t)
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 2, ExplicitHub: true, Hubs: []int{1}, AgentKind: agent, RuntimeBudget: 4 * time.Minute, RetainStartupServerLog: true})
			ctx, stop := context.WithTimeout(t.Context(), 210*time.Second)
			defer stop()
			if err := fleet.Prepare(ctx); err != nil {
				t.Fatal(err)
			}
			if err := fleet.Start(ctx); err != nil {
				t.Fatalf("fresh primary startup: %v; screen:\n%s", err, fleet.Screen())
			}
			if err := fleet.WaitAgent(ctx, 1); err != nil {
				t.Fatalf("extra hub readiness: %v; screen:\n%s", err, fleet.Screen())
			}
			for _, n := range []int{0, 1} {
				ready, err := fleet.Agent(n)
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("hub%d native ready: %+v", n, ready)
				for observation := 0; ; observation++ {
					result, err := fleet.ActorCLI(ctx, n, "--actors", "--json")
					t.Logf("hub%d broker observation%d: %+v err=%v", n, observation, result, err)
					if err == nil && result.ExitCode == 0 && !result.CleanupUncertain {
						break
					}
					if result.CleanupUncertain || result.ID == "" {
						t.Fatalf("uncertain caller: %+v %v", result, err)
					}
					select {
					case <-time.After(2 * time.Second):
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
				}
				status, err := fleet.ActorCLI(ctx, n, "--automation", "status")
				if err != nil || status.ExitCode != 0 || !strings.Contains(status.Stdout, "hub automation off (startup: fresh)") {
					t.Fatalf("hub%d initial automation: %+v %v", n, status, err)
				}
			}
			for _, n := range []int{0, 1} {
				accepted, err := fleet.Accepted(n)
				if err != nil || len(accepted) != 0 {
					t.Fatalf("hub%d received automatic input: %+v err=%v", n, accepted, err)
				}
			}
			if screen := fleet.Screen(); strings.Contains(screen, "hub failed") || strings.Contains(screen, "identity mismatch") {
				t.Fatalf("startup failure notice:\n%s", screen)
			}
			t.Log("both fresh configured hubs are live callers with automation off")
		})
	}
}
