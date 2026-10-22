package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// The Claude screen is deliberately SYNTHETIC. Real quota-screen and operator
// force qualification remain the M3 live check, as recorded in pair#728's Log.
func TestDisposableCouchSYNTHETICLimitAndBusyForcePreserveDirtyFiles(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_FLEET") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_FLEET=1")
	}
	binaries := testcouch.Build(t)
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 1, ExplicitHub: true, AgentKind: agent, RuntimeBudget: 3 * time.Minute})
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
			defer cancel()
			if err := fleet.Prepare(ctx); err != nil {
				t.Fatal(err)
			}
			if err := fleet.Start(ctx); err != nil {
				t.Fatal(err)
			}
			if err := fleet.StartWorker(ctx, 1); err != nil {
				t.Fatal(err)
			}
			before, err := fleet.Agent(1)
			if err != nil {
				t.Fatal(err)
			}
			phase := agentfixture.Working
			needle := "Working"
			if agent == "claude" {
				phase = agentfixture.SyntheticClaudeLimit
				needle = "You've hit your limit"
			}
			if err := fleet.SetState(ctx, 1, phase); err != nil {
				t.Fatal(err)
			}
			probes := 0
			if err := waitFleet(ctx, func() (bool, error) {
				probes++
				if probes > 8 {
					return false, fmt.Errorf("source screen never observed after eight peeks")
				}
				result, err := fleet.ActorCLI(ctx, 0, "--peek", "fixture:1", "--json")
				if err != nil {
					return false, err
				}
				return result.ExitCode == 0 && strings.Contains(result.Stdout, needle), nil
			}); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(fleet.Workspaces.Workers[1], "unsaved-force-check.txt")
			original := []byte("operator edits stay on disk\x00\n")
			if err := os.WriteFile(path, original, 0600); err != nil {
				t.Fatal(err)
			}
			target := "claude"
			if agent == "claude" {
				target = "codex"
			}
			refused, err := fleet.ActorCLI(ctx, 0, "--switch-agent", "fixture:1", "--agent", target, "--agent-args", "", "--confirm", "--json")
			if err != nil || refused.ExitCode == 0 {
				t.Fatalf("ordinary dirty/busy switch admitted: %+v %v", refused, err)
			}
			afterRefusal, err := fleet.Agent(1)
			if err != nil || afterRefusal.PID != before.PID {
				t.Fatal("refusal replaced source", err)
			}
			forceFleetOpenForm(t, ctx, fleet, target)
			for range 3 {
				forceFleetPress(t, fleet, "\t", 9, 1)
			}
			forceFleetPress(t, fleet, "\r", 13, 1)
			forceFleetWaitScreen(t, ctx, fleet, "Force switch? The current turn's unsaved progress may be lost.")
			forceFleetWaitScreen(t, ctx, fleet, "uncommitted files stay as they are for the new agent")
			if ready, err := fleet.Agent(1); err != nil || ready.PID != before.PID {
				t.Fatal("force acted before confirmation", err)
			}
			forceFleetPress(t, fleet, "\x1b[D", 57350, 1)
			forceFleetPress(t, fleet, "\r", 13, 1)
			if err := waitFleet(ctx, func() (bool, error) {
				ready, err := fleet.Agent(1)
				return err == nil && ready.PID != before.PID && ready.Agent == target, nil
			}); err != nil {
				t.Fatalf("force replacement: %v\n%s", err, fleet.Screen())
			}
			actual, err := os.ReadFile(path)
			if err != nil || string(actual) != string(original) {
				t.Fatalf("force changed disk bytes: %q %v", actual, err)
			}
			t.Logf("agent=%s phase=%s target=%s explicit local confirmation preserved dirty file; synthetic=%v", agent, phase, target, agent == "claude")
		})
	}
}

func forceFleetPress(t *testing.T, f *testcouch.Fleet, legacy string, code, modifier int) {
	t.Helper()
	if err := f.Press(legacy, code, modifier); err != nil {
		t.Fatal(err)
	}
}
func forceFleetWaitScreen(t *testing.T, ctx context.Context, f *testcouch.Fleet, text string) {
	t.Helper()
	if err := waitFleet(ctx, func() (bool, error) { return strings.Contains(f.Screen(), text), nil }); err != nil {
		t.Fatalf("waiting for %q: %v\n%s", text, err, f.Screen())
	}
}
func forceFleetOpenForm(t *testing.T, ctx context.Context, f *testcouch.Fleet, target string) {
	t.Helper()
	forceFleetSelectAction(t, ctx, f, "switch coding")
	forceFleetWaitScreen(t, ctx, f, "switch coding agent")
	found := false
	for _, agent := range launcher.AgentInventory() {
		if agent == target {
			found = true
			break
		}
		forceFleetPress(t, f, "\x1b[B", 57353, 1)
	}
	if !found {
		t.Fatalf("unsupported target %s", target)
	}
	forceFleetPress(t, f, "\r", 13, 1)
	forceFleetWaitScreen(t, ctx, f, "parameters")
	if err := waitFleet(ctx, func() (bool, error) {
		screen := strings.ToLower(f.Screen())
		return strings.Contains(screen, "parameters") && !strings.Contains(screen, "resolving startup parameters"), nil
	}); err != nil {
		t.Fatalf("initial preview did not settle: %v\n%s", err, f.Screen())
	}
}

func forceFleetSelectAction(t *testing.T, ctx context.Context, f *testcouch.Fleet, action string) {
	t.Helper()
	forceFleetPress(t, f, "\x00", 32, 5)
	forceFleetWaitScreen(t, ctx, f, "Ctrl+Y copy last notice / selected error")
	if err := f.Input([]byte("fixture:1")); err != nil {
		t.Fatal(err)
	}
	forceFleetPress(t, f, "\t", 9, 1)
	if err := f.Input([]byte(action)); err != nil {
		t.Fatal(err)
	}
	forceFleetPress(t, f, "\r", 13, 1)
}
