package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// TestDisposableStartupArguments traverses the genuine hub CLI, switch receipt,
// launcher, and fixture readiness. Native processes belong to the shared fleet.
func TestDisposableStartupArguments(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_FLEET") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_FLEET=1 to run disposable native acceptance")
	}
	binaries := testcouch.Build(t)
	for _, source := range []string{"claude", "codex"} {
		if !t.Run(source, func(t *testing.T) {
			other := "codex"
			if source == other {
				other = "claude"
			}
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 1, ExplicitHub: true, AgentKind: source, RuntimeBudget: 4 * time.Minute})
			fleet.Environment = append(fleet.Environment, "COUCH_RESOURCE_CADENCE=1s")
			setup, cancelSetup := context.WithTimeout(t.Context(), 180*time.Second)
			err := fleet.Prepare(setup)
			cancelSetup()
			if err != nil {
				t.Fatal(err)
			}
			startupArgumentsBaseline(t, fleet)
			ctx, cancel := context.WithTimeout(t.Context(), 240*time.Second)
			defer cancel()
			if err := fleet.Start(ctx); err != nil {
				t.Fatal(err)
			}
			if err := fleet.StartWorker(ctx, 1); err != nil {
				t.Fatal(err)
			}
			previous, err := fleet.Agent(1)
			if err != nil || previous.Agent != source {
				t.Fatalf("initial source: %+v, %v", previous, err)
			}
			literal := `--model 'a b' '' '$HOME' '$(startup-arguments-not-a-command)'`
			argv := []string{"--model", "a b", "", "$HOME", "$(startup-arguments-not-a-command)"}
			previous = startupArgumentsSwitch(t, ctx, fleet, previous, true, other, argv, "--agent-args", literal)
			previous = startupArgumentsSwitch(t, ctx, fleet, previous, true, source, nil, "--agent-args", "")
			previous = startupArgumentsSwitch(t, ctx, fleet, previous, true, other, argv)
			// The first menu interaction starts from the attached child view.
			// Park through its public confirmation before switching the retained
			// source; an unavailable fixture is never itself vacancy authority.
			forceFleetSelectAction(t, ctx, fleet, "park")
			forceFleetWaitScreen(t, ctx, fleet, " › park")
			forceFleetWaitScreen(t, ctx, fleet, "cancel")
			if err := fleet.Input([]byte("park")); err != nil {
				t.Fatal(err)
			}
			if err := fleet.Press("\r", 13, 1); err != nil {
				t.Fatal(err)
			}
			if err := startupArgumentsWait(ctx, func() (bool, error) {
				_, err := fleet.Agent(1)
				return err != nil, nil
			}); err != nil {
				t.Fatalf("park source stayed live: %v\n%s", err, fleet.Screen())
			}
			// Process exit happens before ParkExpected finalizes its transaction
			// and the console releases operation ownership. Observe completion,
			// rather than treating disappearance as a completed park.
			forceFleetWaitScreen(t, ctx, fleet, "park complete")
			startupArgumentsSwitch(t, ctx, fleet, previous, false, source, nil)
			t.Log("parked-source switch reused the original agent's saved explicit-empty parameters")
			cleanup, cancelCleanup := context.WithTimeout(context.Background(), 15*time.Second)
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

func startupArgumentsSwitch(t *testing.T, ctx context.Context, fleet *testcouch.Fleet, previous agentfixture.Ready, sourceLive bool, agent string, want []string, extra ...string) agentfixture.Ready {
	t.Helper()
	if sourceLive {
		startupArgumentsWaitIdle(t, ctx, fleet, previous)
	}
	args := append([]string{"--switch-agent", "fixture:1", "--agent", agent, "--confirm", "--json"}, extra...)
	result, err := fleet.ActorCLI(ctx, 0, args...)
	var response couchmessage.Response
	if decodeErr := json.Unmarshal([]byte(result.Stdout), &response); decodeErr != nil || response.Operation == nil || response.Operation.ID == "" || result.CleanupUncertain || response.Code != "accepted" && response.Code != "ok" && response.Code != "uncertain" {
		t.Fatalf("switch admission %q: %+v, %v decode=%v", args, result, err, decodeErr)
	}
	id := response.Operation.ID
	if response.Code == "uncertain" {
		t.Logf("switch admission uncertain; inspecting exact receipt without resubmission: %+v", response.Operation)
	}
	// An uncertain admission is never resubmitted. Resolve only its exact
	// receipt through the public read-only path, within one bounded wait.
	receiptCtx, cancelReceipt := context.WithTimeout(ctx, 45*time.Second)
	defer cancelReceipt()
	if err := startupArgumentsWait(receiptCtx, func() (bool, error) {
		result, err := fleet.ActorCLI(receiptCtx, 0, "--operation-status", id, "--json")
		var status couchmessage.Response
		if decodeErr := json.Unmarshal([]byte(result.Stdout), &status); decodeErr != nil || result.CleanupUncertain || status.Code != "ok" {
			return false, fmt.Errorf("receipt %s: %+v, %v decode=%v", id, result, err, decodeErr)
		}
		if status.Operation == nil || status.Operation.ID != id {
			return false, fmt.Errorf("uncorrelated receipt %s: %s", id, result.Stdout)
		}
		switch status.Operation.Status {
		case couchmessage.ReceiptQueued, couchmessage.ReceiptRunning, couchmessage.ReceiptUnknown:
			return false, nil
		case couchmessage.ReceiptSucceeded:
			return true, nil
		default:
			return false, fmt.Errorf("switch receipt did not succeed: %+v", status.Operation)
		}
	}); err != nil {
		t.Fatal(err)
	}
	var ready agentfixture.Ready
	if err := startupArgumentsWait(ctx, func() (bool, error) {
		var err error
		ready, err = fleet.Agent(1)
		return err == nil && ready.Agent == agent && ready.PID != previous.PID && ready.Generation != previous.Generation, nil
	}); err != nil {
		t.Fatalf("replacement readiness: previous=%+v last=%+v: %v", previous, ready, err)
	}
	if ready.NativeID == "" || ready.NativeID == previous.NativeID {
		t.Fatalf("switch reused native conversation: previous=%+v current=%+v", previous, ready)
	}
	actual := append([]string{}, want...)
	if agent == "claude" {
		actual = append(actual, "--session-id", ready.NativeID)
	} else {
		actual = append([]string{"--disable", "shell_snapshot"}, actual...)
		actual = append(actual, "--no-alt-screen")
	}
	if !reflect.DeepEqual(ready.Argv, actual) {
		t.Fatalf("literal startup argv: got=%q want=%q", ready.Argv, actual)
	}
	t.Logf("native startup arguments: receipt=%s agent=%s pid=%d native=%s argv=%q", id, ready.Agent, ready.PID, ready.NativeID, ready.Argv)
	return ready
}

func startupArgumentsWait(ctx context.Context, check func() (bool, error)) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if done, err := check(); done || err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func startupArgumentsWaitIdle(t *testing.T, parent context.Context, fleet *testcouch.Fleet, expected agentfixture.Ready) {
	t.Helper()
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	started := time.Now()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	checkSource := func() {
		t.Helper()
		ready, err := fleet.Agent(1)
		if err != nil || ready.PID != expected.PID || ready.Generation != expected.Generation || ready.Agent != expected.Agent {
			t.Fatalf("source changed while awaiting idle: expected=%+v current=%+v err=%v", expected, ready, err)
		}
	}
	var last couchcore.ResourceView
	// Every iteration is read-only; the effectful switch is submitted once,
	// only after the wrapper's own qualified idle evidence reaches the broker.
	for probe := 0; probe < 30; probe++ {
		checkSource()
		actors, err := fleet.Actors(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var binding couchmessage.Binding
		matches := 0
		for _, actor := range actors.Actors {
			if actor.Binding.Slot == "fixture:1" {
				binding = actor.Binding
				matches++
			}
		}
		if matches > 1 {
			t.Fatalf("source actor ambiguous: %+v", actors.Actors)
		}
		// Native readiness precedes the replacement wrapper's registration.
		// Keep observing absence/old registration within this same bound;
		// only the expected actor may supply idle evidence for the next switch.
		if matches == 0 || binding.Agent != expected.Agent {
			select {
			case <-ctx.Done():
				t.Fatalf("source actor never registered: %v; actors=%+v", ctx.Err(), actors.Actors)
			case <-ticker.C:
			}
			continue
		}
		result, err := fleet.ActorCLI(ctx, 0, "--resources", "--json")
		if err != nil || result.ExitCode != 0 {
			t.Fatalf("idle resources: %+v, %v", result, err)
		}
		if err := json.Unmarshal([]byte(result.Stdout), &last); err != nil {
			t.Fatal(err)
		}
		checkSource()
		if startupArgumentsFreshIdle(last, binding, started) {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("source never became known idle: %v; sampled=%s slots=%+v", ctx.Err(), last.SampledAt, last.Slots)
		case <-ticker.C:
		}
	}
	t.Fatalf("source never became known idle after 30 public observations: sampled=%s slots=%+v", last.SampledAt, last.Slots)
}

func startupArgumentsFreshIdle(view couchcore.ResourceView, binding couchmessage.Binding, started time.Time) bool {
	if !view.SampledAt.After(started) || binding.Slot != "fixture:1" || binding.Scope == "" || binding.Tag == "" {
		return false
	}
	matched := 0
	idle := false
	for _, row := range view.Slots {
		if row.Slot != binding.Slot {
			continue
		}
		matched++
		idle = row.Thread.RepoScope == binding.Scope && string(row.Thread.Tag) == binding.Tag && row.IdleKnown && row.Idle
	}
	return matched == 1 && idle
}

func TestStartupArgumentsFreshIdle(t *testing.T) {
	started := time.Unix(100, 0)
	binding := couchmessage.Binding{Slot: "fixture:1", Scope: "scope", Tag: "retained"}
	row := couchcore.ResourceSlotView{Slot: binding.Slot, Thread: couchcore.ThreadAddress{RepoScope: binding.Scope, Tag: couchcore.ThreadTag(binding.Tag)}, IdleKnown: true, Idle: true}
	for _, tc := range []struct {
		name   string
		mutate func(*couchcore.ResourceView, *couchmessage.Binding)
		want   bool
	}{
		{name: "fresh exact idle", want: true},
		{name: "stale", mutate: func(v *couchcore.ResourceView, _ *couchmessage.Binding) { v.SampledAt = started.Add(-time.Second) }},
		{name: "equal timestamp", mutate: func(v *couchcore.ResourceView, _ *couchmessage.Binding) { v.SampledAt = started }},
		{name: "missing slot", mutate: func(v *couchcore.ResourceView, _ *couchmessage.Binding) { v.Slots = nil }},
		{name: "wrong slot", mutate: func(v *couchcore.ResourceView, _ *couchmessage.Binding) { v.Slots[0].Slot = "fixture:2" }},
		{name: "wrong scope", mutate: func(v *couchcore.ResourceView, _ *couchmessage.Binding) { v.Slots[0].Thread.RepoScope = "other" }},
		{name: "wrong tag", mutate: func(v *couchcore.ResourceView, _ *couchmessage.Binding) { v.Slots[0].Thread.Tag = "replaced" }},
		{name: "unknown idle", mutate: func(v *couchcore.ResourceView, _ *couchmessage.Binding) { v.Slots[0].IdleKnown = false }},
		{name: "busy", mutate: func(v *couchcore.ResourceView, _ *couchmessage.Binding) { v.Slots[0].Idle = false }},
		{name: "missing actor scope", mutate: func(_ *couchcore.ResourceView, b *couchmessage.Binding) { b.Scope = "" }},
		{name: "missing actor tag", mutate: func(_ *couchcore.ResourceView, b *couchmessage.Binding) { b.Tag = "" }},
		{name: "duplicate slot", mutate: func(v *couchcore.ResourceView, _ *couchmessage.Binding) { v.Slots = append(v.Slots, v.Slots[0]) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			view := couchcore.ResourceView{SampledAt: started.Add(time.Second), Slots: []couchcore.ResourceSlotView{row}}
			actor := binding
			if tc.mutate != nil {
				tc.mutate(&view, &actor)
			}
			if got := startupArgumentsFreshIdle(view, actor, started); got != tc.want {
				t.Fatalf("fresh idle=%v want=%v", got, tc.want)
			}
		})
	}
}

// startupArgumentsBaseline records only generated policy inputs in a disposable
// fixture; all subsequent ordinary switches still require a clean worktree.
func startupArgumentsBaseline(t *testing.T, fleet *testcouch.Fleet) {
	t.Helper()
	for _, args := range [][]string{
		{"add", "--", ".gitignore", ".gitattributes"},
		{"-c", "user.name=Couch Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "fixture generated setup baseline"},
		{"status", "--porcelain=v1", "--untracked-files=all"},
	} {
		command := exec.CommandContext(t.Context(), fleet.Binaries.Git, args...)
		command.Dir, command.Env = fleet.Workspaces.Workers[1], fleet.Environment
		output, err := command.CombinedOutput()
		if err != nil || (args[0] == "status" && len(output) != 0) {
			t.Fatalf("private setup baseline %q: %s %v", args, output, err)
		}
	}
}
