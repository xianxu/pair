package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

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
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 1, ExplicitHub: true, AgentKind: source})
			setup, cancelSetup := context.WithTimeout(t.Context(), 180*time.Second)
			err := fleet.Prepare(setup)
			cancelSetup()
			if err != nil {
				t.Fatal(err)
			}
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
			previous = startupArgumentsSwitch(t, ctx, fleet, previous, other, argv, "--agent-args", literal)
			previous = startupArgumentsSwitch(t, ctx, fleet, previous, source, nil, "--agent-args", "")
			previous = startupArgumentsSwitch(t, ctx, fleet, previous, other, argv)
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
			startupArgumentsSwitch(t, ctx, fleet, previous, source, nil)
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

func startupArgumentsSwitch(t *testing.T, ctx context.Context, fleet *testcouch.Fleet, previous agentfixture.Ready, agent string, want []string, extra ...string) agentfixture.Ready {
	t.Helper()
	args := append([]string{"--switch-agent", "fixture:1", "--agent", agent, "--confirm", "--json"}, extra...)
	result, err := fleet.ActorCLI(ctx, 0, args...)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("switch admission %q: %+v, %v", args, result, err)
	}
	var response couchmessage.Response
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil || response.Operation == nil || response.Operation.ID == "" {
		t.Fatalf("missing switch receipt: %+v, %v", result, err)
	}
	id := response.Operation.ID
	if err := startupArgumentsWait(ctx, func() (bool, error) {
		result, err := fleet.ActorCLI(ctx, 0, "--operation-status", id, "--json")
		if err != nil || result.ExitCode != 0 {
			return false, fmt.Errorf("receipt %s: %+v, %v", id, result, err)
		}
		var status couchmessage.Response
		if err := json.Unmarshal([]byte(result.Stdout), &status); err != nil {
			return false, err
		}
		if status.Operation == nil || status.Operation.ID != id {
			return false, fmt.Errorf("uncorrelated receipt %s: %s", id, result.Stdout)
		}
		switch status.Operation.Status {
		case couchmessage.ReceiptQueued, couchmessage.ReceiptRunning:
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
