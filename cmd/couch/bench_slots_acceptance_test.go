package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testcouch"
)

// pair#847 native acceptance: after one Couch start with the bench hub as an
// extra hub, fixture:1 (the bench hub, whose .hub claims fixture:2) brings its
// worker up with an exact profile, verifies it from the receipt, restarts it
// with new argv, and retires it, with no restart and no console step. The main
// hub and the unclaimed production slot fixture:3 are refused, and the
// per-repository saved parameters never change.
func TestDisposableCouchBenchSlots(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_FLEET") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_FLEET=1 or make test-couch-fleet")
	}
	binaries := testcouch.Build(t)
	for _, agent := range []string{"claude", "codex"} {
		if !t.Run(agent, func(t *testing.T) {
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 3, ExplicitHub: true, Hubs: []int{1}, HubClaims: map[int][]string{1: {"fixture:2"}}, AgentKind: agent})
			setup, cancelSetup := context.WithTimeout(t.Context(), 240*time.Second)
			err := fleet.Prepare(setup)
			cancelSetup()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 300*time.Second)
			defer cancel()
			if err := fleet.Start(ctx); err != nil {
				t.Fatalf("fresh startup: %v\n%s", err, fleet.Screen())
			}
			if err := fleet.WaitAgent(ctx, 1); err != nil {
				t.Fatalf("bench hub start: %v\n%s", err, fleet.Screen())
			}
			if err := fleet.WaitActors(ctx, 2); err != nil {
				t.Fatalf("hub registration: %v\n%s", err, fleet.Screen())
			}
			preferences := benchPreferences(t, fleet)
			before := preferences()

			first := benchStart(t, ctx, fleet, agent, "bench-one")
			second := benchStart(t, ctx, fleet, agent, "bench-two")
			if first.Generation == "" || first.Generation == second.Generation {
				t.Fatalf("restart kept its generation: %q then %q", first.Generation, second.Generation)
			}
			ready, err := fleet.Agent(2)
			if err != nil {
				t.Fatal(err)
			}

			retire := benchCLI(t, ctx, fleet, 1, "--bench-retire", "fixture:2", "--confirm", "--json")
			if retire.Status != couchmessage.ReceiptSucceeded {
				t.Fatalf("bench-retire: %+v", retire)
			}
			if syscall.Kill(ready.PID, 0) == nil {
				t.Fatalf("retired agent %d still running", ready.PID)
			}
			if live := benchThreadsAt(t, fleet, fleet.Workspaces.Workers[2]); len(live) != 0 {
				t.Fatalf("threads left at the retired slot: %v", live)
			}
			if _, err := os.Stat(fleet.Workspaces.Workers[2]); err != nil {
				t.Fatalf("retire removed the worktree: %v", err)
			}

			// Production is out of reach: the main hub, and the unclaimed slot.
			for _, refused := range []struct {
				from int
				args []string
			}{
				{0, []string{"--bench-start", "fixture:2", "--agent", agent, "--agent-args", "", "--confirm", "--json"}},
				{1, []string{"--bench-start", "fixture:3", "--agent", agent, "--agent-args", "", "--confirm", "--json"}},
				{1, []string{"--bench-retire", "fixture:3", "--confirm", "--json"}},
			} {
				result, err := fleet.ActorCLI(ctx, refused.from, refused.args...)
				if err == nil && result.ExitCode == 0 {
					t.Fatalf("fixture:%d %v admitted: %+v", refused.from, refused.args, result)
				}
			}
			if ready, err := fleet.Agent(3); err == nil && syscall.Kill(ready.PID, 0) == nil {
				t.Fatalf("production slot fixture:3 started: %+v", ready)
			}
			if after := preferences(); !bytes.Equal(before, after) {
				t.Fatalf("saved parameters changed:\n%s\n%s", before, after)
			}

			cleanup, cancelCleanup := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelCleanup()
			if err := fleet.Close(cleanup); err != nil {
				t.Fatal(err)
			}
			if err := fleet.AssertCleaned(); err != nil {
				t.Fatal(err)
			}
		}) {
			break
		}
	}
}

// benchStart starts fixture:2 from the bench hub with --model <model> and
// checks the receipt's provenance against what the fixture actually ran.
func benchStart(t *testing.T, ctx context.Context, fleet *testcouch.Fleet, agent, model string) couchcore.LaunchProvenance {
	t.Helper()
	receipt := benchCLI(t, ctx, fleet, 1, "--bench-start", "fixture:2", "--agent", agent, "--agent-args", "--model "+model, "--confirm", "--json")
	p := receipt.Provenance
	if receipt.Status != couchmessage.ReceiptSucceeded || p == nil {
		t.Fatalf("bench-start %s: %+v", model, receipt)
	}
	if filepath.Clean(p.Root) != filepath.Clean(fleet.Workspaces.Workers[2]) || p.Agent != agent || !slices.Equal(p.Argv, []string{"--model", model}) || p.ArgvSource != "explicit" || p.Tag == "" {
		t.Fatalf("provenance %+v", p)
	}
	if err := waitFleet(ctx, func() (bool, error) {
		ready, err := fleet.Agent(2)
		if err != nil {
			return false, nil
		}
		return slices.Contains(ready.Argv, model) && syscall.Kill(ready.PID, 0) == nil, nil
	}); err != nil {
		t.Fatalf("fixture never ran --model %s: %v", model, err)
	}
	return *p
}

func benchCLI(t *testing.T, ctx context.Context, fleet *testcouch.Fleet, from int, args ...string) couchmessage.OperationReceipt {
	t.Helper()
	result, err := fleet.ActorCLI(ctx, from, args...)
	if err != nil {
		t.Fatalf("%v: %+v %v", args, result, err)
	}
	var receipt couchmessage.OperationReceipt
	if err := json.Unmarshal([]byte(strings.TrimSpace(result.Stdout)), &receipt); err != nil {
		t.Fatalf("%v: receipt %q: %v (stderr %q)", args, result.Stdout, err, result.Stderr)
	}
	return receipt
}

func benchStore(t *testing.T, fleet *testcouch.Fleet) couchcore.CouchNamespace {
	t.Helper()
	namespace, err := couchcore.ResolveCouchNamespace(fleet.Roots.Mutable["store"], fleet.Workspaces.Primary)
	if err != nil {
		t.Fatal(err)
	}
	return namespace
}

func benchPreferences(t *testing.T, fleet *testcouch.Fleet) func() []byte {
	path := filepath.Join(benchStore(t, fleet).Dir(), "threadstore", "repo-preferences.json")
	return func() []byte {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		return raw
	}
}

func benchThreadsAt(t *testing.T, fleet *testcouch.Fleet, path string) []string {
	t.Helper()
	snapshot, err := couchcore.NewThreadStore(benchStore(t, fleet)).Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, record := range snapshot.Records {
		if filepath.Clean(record.StartingPath) == filepath.Clean(path) {
			out = append(out, string(record.Address.Tag))
		}
	}
	return out
}
