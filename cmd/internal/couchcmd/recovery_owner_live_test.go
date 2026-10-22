package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testcouch"
)

// Live reboot acceptance for pair#797, including delivery to the new worker.
// Failed confirmations retain their judged native reply before store cleanup.
func TestLiveRecoveryOwnerReboot(t *testing.T) {
	if os.Getenv("PAIR_LIVE_RECOVERY_OWNER") != "1" {
		t.Skip("set PAIR_LIVE_RECOVERY_OWNER=1 for disposable native reboot acceptance")
	}
	binaries := testcouch.Build(t)
	for _, kind := range []string{"claude", "codex"} {
		t.Run(kind, func(t *testing.T) {
			fleet := testcouch.New(t, binaries, testcouch.Options{ExplicitHub: true, Workers: 1, AgentKind: kind, RuntimeBudget: 120 * time.Second})
			fleet.Environment = append(fleet.Environment, "COUCH_RESOURCE_CADENCE=1s")
			setup, stopSetup := context.WithTimeout(t.Context(), 180*time.Second)
			defer stopSetup()
			if err := fleet.Prepare(setup); err != nil {
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
			old, err := fleet.Agent(1)
			if err != nil {
				t.Fatal(err)
			}
			cohort, err := fleet.RetainDescendants(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(cohort.Processes) != 6 {
				t.Fatalf("expected exact six retained descendants, got %+v", cohort.Processes)
			}
			if err = cohort.ReleaseParent(ctx); err != nil {
				t.Fatal(err)
			}
			if err = awaitProvenNative(ctx, func() (bool, error) {
				rows, _, e := (procutil.NativeResourceReader{}).Table(ctx)
				if e != nil {
					return false, e
				}
				seen := 0
				for _, p := range cohort.Processes {
					for _, row := range rows {
						if row.PID == p.PID && row.Identity == p.Generation && row.PPID == 1 {
							seen++
						}
					}
				}
				return seen == len(cohort.Processes), nil
			}); err != nil {
				t.Fatal(err)
			}
			out, callErr := fleet.ActorCLI(ctx, 0, "--reboot", "fixture:1", "--confirm", "--json")
			var offered couchmessage.OperationReceipt
			if err = json.Unmarshal([]byte(out.Stdout), &offered); err != nil {
				t.Fatalf("offer: %+v %v: %v", out, callErr, err)
			}
			if !provenPreviewRefusal(out, callErr) || offered.Code != "confirmation-required" || offered.Confirmation == nil || offered.Confirmation.Token == "" {
				t.Fatalf("exact confirmation offer unavailable: %+v %+v %v", out, offered, callErr)
			}
			t.Logf("native offer personality=%s old_agent=%d/%s stops=%+v", kind, old.PID, old.Generation, offered.Confirmation.Stops)
			out, callErr = fleet.ActorCLI(ctx, 0, "--recovery-confirmation", offered.Confirmation.Token, "--json")
			var result couchmessage.OperationReceipt
			if err = json.Unmarshal([]byte(out.Stdout), &result); err != nil {
				t.Fatalf("confirmation: %+v %v: %v", out, callErr, err)
			}
			t.Logf("native confirmation personality=%s result=%s err=%v old_agent_identity_now=%q", kind, out.Stdout, callErr, procutil.Identity(strconv.Itoa(old.PID)))
			if callErr != nil || out.ExitCode != 0 || result.Status != couchmessage.ReceiptSucceeded {
				namespace, evidenceErr := couchcore.ExistingCouchNamespace(fleet.Roots.Mutable["store"])
				if evidenceErr != nil {
					t.Fatalf("confirmation failed: %+v %+v %v; evidence namespace: %v", out, result, callErr, evidenceErr)
				}
				history, evidenceErr := couchcore.NewLaunchFailureStore(namespace).Read()
				if evidenceErr != nil {
					t.Fatalf("confirmation failed: %+v %+v %v; evidence read: %v", out, result, callErr, evidenceErr)
				}
				found := false
				for _, failure := range history {
					if result.FailureNonce == "" || failure.Nonce != result.FailureNonce {
						continue
					}
					found = true
					exit := "unobserved"
					if failure.ZellijExit != nil {
						exit = strconv.Itoa(*failure.ZellijExit)
					}
					t.Logf("judged native query nonce=%s executable=%q argv=%q version=%q exit=%s stdout_bytes=%d stdout_truncated=%t stderr_bytes=%d stderr_truncated=%t", failure.Nonce, failure.ZellijExecutable, failure.ZellijArgs, failure.ZellijVersion, exit, failure.ZellijStdoutBytes, failure.ZellijStdoutTruncated, failure.ZellijStderrBytes, failure.ZellijStderrTruncated)
					t.Logf("judged native stdout=%q", failure.ZellijStdout)
					t.Logf("judged native stderr=%q", failure.ZellijStderr)
				}
				t.Fatalf("confirmation did not succeed: %+v %+v %v; correlated_evidence=%t", out, result, callErr, found)
			}
			if result.Tag == "" {
				t.Fatalf("successful reboot omitted new binding tag: %+v", result)
			}
			if err = awaitProvenNative(ctx, func() (bool, error) {
				current, e := fleet.Agent(1)
				return e == nil && current.Generation != old.Generation && procutil.Identity(strconv.Itoa(old.PID)) != old.Generation, nil
			}); err != nil {
				t.Fatalf("new agent generation or old generation exit: %v", err)
			}
			current, err := fleet.Agent(1)
			if err != nil {
				t.Fatal(err)
			}
			if err = fleet.WaitActors(ctx, 2); err != nil {
				t.Fatalf("new actor registration: %v", err)
			}
			if err = awaitProvenNative(ctx, func() (bool, error) {
				actors, e := fleet.Actors(ctx)
				if e != nil {
					return false, e
				}
				for _, actor := range actors.Actors {
					if actor.Binding.Slot == "fixture:1" && actor.Binding.Tag == result.Tag {
						return true, nil
					}
				}
				return false, nil
			}); err != nil {
				t.Fatalf("new exact binding registration: %v", err)
			}
			body := "recovery-owner-current-worker-" + kind
			sent, err := fleet.ActorCLI(ctx, 0, "--send-to", "fixture:1", "--message", body)
			if err != nil || sent.ExitCode != 0 {
				t.Fatalf("post-reboot delivery: %+v %v", sent, err)
			}
			if err = awaitProvenNative(ctx, func() (bool, error) {
				accepted, e := fleet.Accepted(1)
				if errors.Is(e, os.ErrNotExist) {
					return false, nil
				}
				for _, input := range accepted {
					if input.PID == current.PID && strings.Contains(input.Body, body) {
						return true, nil
					}
				}
				return false, e
			}); err != nil {
				t.Fatalf("new worker did not accept message body: %v", err)
			}
			t.Logf("native reboot accepted personality=%s old_agent=%d/%s current_agent=%d/%s message=%q result=%s", kind, old.PID, old.Generation, current.PID, current.Generation, body, out.Stdout)
			cleanup, stopCleanup := context.WithTimeout(context.Background(), 15*time.Second)
			defer stopCleanup()
			if err = fleet.Close(cleanup); err != nil {
				t.Fatal(err)
			}
			if err = fleet.AssertCleaned(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
