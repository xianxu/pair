package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testcouch"
)

// TestDisposableSlashKeys exercises the public CLI and native transport with
// deterministic agent personalities. It does not qualify actual model UIs.
func TestDisposableSlashKeys(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_SLASH_KEYS") != "1" {
		t.Skip("set PAIR_NATIVE_SLASH_KEYS=1 to exercise the disposable fleet")
	}
	agents := []string{"claude", "codex"}
	if agent := os.Getenv("PAIR_NATIVE_SLASH_AGENT"); agent != "" {
		if agent != "claude" && agent != "codex" {
			t.Fatalf("PAIR_NATIVE_SLASH_AGENT must be claude or codex, got %q", agent)
		}
		agents = []string{agent}
	}
	binaries := testcouch.Build(t)
	for _, agent := range agents {
		if !t.Run(agent, func(t *testing.T) {
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 1, AgentKind: agent})
			setup, cancelSetup := context.WithTimeout(t.Context(), 180*time.Second)
			err := fleet.Prepare(setup)
			cancelSetup()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
			defer cancel()
			if err := fleet.Start(ctx); err != nil {
				t.Fatal(err)
			}
			if err := fleet.StartWorker(ctx, 1); err != nil {
				t.Fatal(err)
			}
			actors, err := fleet.Actors(ctx)
			if err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for _, actor := range actors.Actors {
				seen[actor.Binding.Slot] = true
			}
			if len(actors.Actors) != 2 || !seen["fixture:0"] || !seen["fixture:1"] {
				t.Fatalf("wrong genuine steward/worker cohort: %+v", actors)
			}
			for n := 0; n <= 1; n++ {
				ready, err := fleet.Agent(n)
				if err != nil || ready.Agent != agent {
					t.Fatalf("actor %d personality: %+v %v", n, ready, err)
				}
			}

			afterCommand, capturedRefusal := false, false
			captureRefusal := func(resourceJSON string) {
				if capturedRefusal || ctx.Err() != nil {
					return
				}
				capturedRefusal = true
				// One pair of public observations, within the original runtime.
				// They are sequential evidence, not one atomic wrapper frame.
				if resourceJSON == "" {
					result, err := fleet.ActorCLI(ctx, 0, "--resources", "--json")
					resourceJSON = result.Stdout
					t.Logf("post-command resources exit=%d error=%v", result.ExitCode, err)
				}
				const retainedBytes = 8192
				t.Logf("post-command refusal resources at=%s bytes=%d truncated=%t json=%q", time.Now().Format(time.RFC3339Nano), len(resourceJSON), len(resourceJSON) > retainedBytes, resourceJSON[:min(len(resourceJSON), retainedBytes)])
				peek, err := fleet.ActorCLI(ctx, 0, "--peek", "fixture:1", "--lines", "40", "--json")
				t.Logf("post-command refusal live peek exit=%d error=%v bytes=%d truncated=%t json=%q", peek.ExitCode, err, len(peek.Stdout), peek.Truncated || len(peek.Stdout) > retainedBytes, peek.Stdout[:min(len(peek.Stdout), retainedBytes)])
			}

			waitSupervisorIdle := func(after time.Time) {
				t.Helper()
				// This proves only the supervisor's public settled evidence.
				// The single keys admission still checks wrapper drive readiness;
				// a refusal fails the test and is never retried.
				if err := waitFleet(ctx, func() (bool, error) {
					result, err := fleet.ActorCLI(ctx, 0, "--resources", "--json")
					if err != nil {
						return false, err
					}
					if result.ExitCode != 0 {
						return false, fmt.Errorf("public resources: %+v", result)
					}
					var resources couchcore.ResourceView
					if err := json.Unmarshal([]byte(result.Stdout), &resources); err != nil {
						return false, err
					}
					if !slashKeysSupervisorIdle(resources, after) {
						if afterCommand && resources.SampledAt.After(after) {
							captureRefusal(result.Stdout)
						}
						return false, nil
					}
					t.Logf("public idle sample=%s after=%s", resources.SampledAt.Format(time.RFC3339Nano), after.Format(time.RFC3339Nano))
					return true, nil
				}); err != nil {
					t.Fatalf("wait for supervisor idle evidence: %v", err)
				}
			}
			keys := func(steps ...string) *couchmessage.OperationReceipt {
				t.Helper()
				args := append([]string{"--keys", "fixture:1"}, steps...)
				result, err := fleet.ActorCLI(ctx, 0, append(args, "--json")...)
				if err != nil || result.ExitCode != 0 {
					if afterCommand {
						captureRefusal("")
					}
					t.Fatalf("public keys %q: %+v %v", steps, result, err)
				}
				var response couchmessage.Response
				if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
					t.Fatal(err)
				}
				r := response.Operation
				if r == nil || r.ID == "" || r.Op != "keys" || r.Target != "fixture:1" || r.Status != couchmessage.ReceiptSucceeded || r.Drive == nil || r.Drive.Outcome != agentdrive.OutcomeConfirmed {
					t.Fatalf("keys lacks succeeded/confirmed receipt: %s", result.Stdout)
				}
				return r
			}
			waitSupervisorIdle(time.Now())
			baseline, err := fleet.Accepted(1)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
			// Startup may submit the supervisor's restart notice. Freeze the
			// settled journal, then require exactly one appended command.
			assertAccepted := func() (bool, error) {
				accepted, err := fleet.Accepted(1)
				if errors.Is(err, os.ErrNotExist) {
					return false, nil
				}
				if err != nil {
					return false, err
				}
				if len(accepted) <= len(baseline) {
					return false, nil
				}
				if len(accepted) != len(baseline)+1 || accepted[len(baseline)].Body != "/model" || len(baseline) > 0 && !reflect.DeepEqual(accepted[:len(baseline)], baseline) {
					return false, fmt.Errorf("expected unchanged baseline plus exactly one /model submission: %+v", accepted)
				}
				return true, nil
			}
			receipt := keys("clear-line", "text:/model", "enter")
			afterCommand = true
			if err := waitFleet(ctx, assertAccepted); err != nil {
				t.Fatal(err)
			}
			t.Logf("simulated %s: receipt=%s succeeded/confirmed, accepted /model exactly once", agent, receipt.ID)

			// Ordinary Return keeps prose in the composer. Public live peek
			// witnesses the retained draft before explicit recovery clears it.
			prose := "slash-keys-ordinary-prose-" + agent
			waitSupervisorIdle(receipt.Finished)
			keys("text:"+prose, "enter")
			if err := waitFleet(ctx, func() (bool, error) {
				result, err := fleet.ActorCLI(ctx, 0, "--peek", "fixture:1", "--lines", "40", "--json")
				if err != nil {
					return false, err
				}
				if result.ExitCode != 0 {
					return false, fmt.Errorf("public peek: %+v", result)
				}
				var peek couchcore.PeekResult
				if err := json.Unmarshal([]byte(result.Stdout), &peek); err != nil {
					return false, err
				}
				return peek.Source == "live" && strings.Contains(strings.Join(peek.Lines, "\n"), prose), nil
			}); err != nil {
				t.Fatal(err)
			}
			if ok, err := assertAccepted(); err != nil || !ok {
				t.Fatalf("ordinary prose submitted unexpectedly: %v", err)
			}
			keys("clear-line")
			if ok, err := assertAccepted(); err != nil || !ok {
				t.Fatalf("recovery changed accepted inputs: %v", err)
			}

			cleanup, cancelCleanup := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancelCleanup()
			if err := fleet.Close(cleanup); err != nil {
				t.Fatal(err)
			}
			if err := fleet.AssertCleaned(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(fleet.Roots.Root); !os.IsNotExist(err) {
				t.Fatalf("owned root retained after successful cleanup: %v", err)
			}
		}) {
			break
		}
	}
}

// A later operation cannot inherit idle evidence sampled before its predecessor.
func slashKeysSupervisorIdle(resources couchcore.ResourceView, after time.Time) bool {
	if !resources.SampledAt.After(after) {
		return false
	}
	for _, slot := range resources.Slots {
		if slot.Slot == "fixture:1" {
			return slot.IdleKnown && slot.Idle
		}
	}
	return false
}

func TestSlashKeysSupervisorIdleRequiresFreshSample(t *testing.T) {
	finished := time.Unix(123, 0)
	for _, offset := range []time.Duration{-time.Second, 0, time.Nanosecond} {
		resources := couchcore.ResourceView{SampledAt: finished.Add(offset), Slots: []couchcore.ResourceSlotView{{Slot: "fixture:1", IdleKnown: true, Idle: true}}}
		if got := slashKeysSupervisorIdle(resources, finished); got != (offset > 0) {
			t.Fatalf("sample offset=%s admitted=%t", offset, got)
		}
	}
}
