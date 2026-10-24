package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/pathscope"
	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// TestDisposableEditedRestart exercises real Couch/Pair/Zellij processes with
// fixture native agents. Supported real-agent flags remain the operator smoke.
func TestDisposableEditedRestart(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_FLEET") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_FLEET=1 to run disposable native acceptance")
	}
	binaries, baselineSource := startupArgumentsBinaries(t)
	for _, agent := range []string{"claude", "codex"} {
		if !t.Run(agent, func(t *testing.T) {
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 1, ExplicitHub: true, AgentKind: agent, RuntimeBudget: 4 * time.Minute})
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
			start := fleet.Start
			if baselineSource != "" {
				start = fleet.StartCold
			}
			if err := start(ctx); err != nil {
				t.Fatal(err)
			}
			if baselineSource != "" {
				startupArgumentsStartHub(t, ctx, fleet)
			}
			if err := fleet.StartWorker(ctx, 1); err != nil {
				t.Fatal(err)
			}
			previous, err := fleet.Agent(1)
			if err != nil || previous.Agent != agent || previous.NativeID == "" {
				t.Fatalf("initial fixture: %+v %v", previous, err)
			}
			// A Claude fixture creates its file before it has a native conversation.
			// Real public delivery establishes the native transcript and ledger proof.
			startupRestartEstablishConversation(t, ctx, fleet, previous)
			startupArgumentsWaitIdle(t, ctx, fleet, previous)
			edited := []string{"--model", "edited model"}
			resumed, resumeReceipt := startupRestartRequest(t, ctx, fleet, previous, "--model 'edited model'")
			if resumed.NativeID != previous.NativeID || resumed.Transcript != previous.Transcript || resumeReceipt.Warning != "" {
				t.Fatalf("resume changed conversation or warned: before=%+v after=%+v receipt=%+v", previous, resumed, resumeReceipt)
			}
			startupRestartAssertArgv(t, resumed, edited, true)
			startupArgumentsWaitIdle(t, ctx, fleet, resumed)
			startupRestartPark(t, ctx, fleet)
			backup, digest := startupRestartHideTranscript(t, fleet, resumed)
			fresh, freshReceipt := startupRestartRequest(t, ctx, fleet, resumed, "")
			if fresh.NativeID == "" || fresh.NativeID == resumed.NativeID || fresh.Transcript == resumed.Transcript {
				t.Fatalf("positive absence reused previous conversation: before=%+v after=%+v", resumed, fresh)
			}
			if freshReceipt.Warning != couchcore.EditedRestartFallbackNotice {
				t.Fatalf("public receipt lost exact fallback notice: %+v", freshReceipt)
			}
			startupRestartAssertArgv(t, fresh, []string{}, false)
			retained, err := os.ReadFile(backup)
			if err != nil || sha256.Sum256(retained) != digest {
				t.Fatalf("prior transcript bytes changed: %s %v", backup, err)
			}
			t.Logf("native edited restart: agent=%s resumed_receipt=%s fallback_receipt=%s old_native=%s fresh_native=%s prior_transcript_sha256=%x warning=%q", agent, resumeReceipt.ID, freshReceipt.ID, resumed.NativeID, fresh.NativeID, digest, freshReceipt.Warning)
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

func startupRestartEstablishConversation(t *testing.T, ctx context.Context, fleet *testcouch.Fleet, expected agentfixture.Ready) {
	t.Helper()
	body := "establish edited-restart fixture conversation " + fleet.RunID
	// Send's public CLI prints its receipt ID; JSON is supported by the
	// read-only message-status route, not by --send-to.
	result, err := fleet.ActorCLI(ctx, 0, "--send-to", "fixture:1", "--message", body)
	fields := strings.Fields(result.Stdout)
	if err != nil || result.ExitCode != 0 || result.CleanupUncertain || len(fields) == 0 {
		t.Fatalf("establish native conversation: %+v %v", result, err)
	}
	id := fields[0]
	delivery, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if err := startupArgumentsWait(delivery, func() (bool, error) {
		result, err := fleet.ActorCLI(delivery, 0, "--message-status", id, "--json")
		var status couchmessage.Response
		if decodeErr := json.Unmarshal([]byte(result.Stdout), &status); decodeErr != nil || result.CleanupUncertain || status.Receipt == nil || status.Receipt.Message.ID != id {
			return false, fmt.Errorf("message receipt %s: %+v %v decode=%v", id, result, err, decodeErr)
		}
		if status.Receipt.Status == couchmessage.Submitted {
			return true, nil
		}
		if status.Receipt.Status.Terminal() {
			return false, fmt.Errorf("fixture transcript seed not submitted: %+v", status.Receipt)
		}
		return false, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := startupArgumentsWait(delivery, func() (bool, error) {
		accepted, err := fleet.Accepted(1)
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		for _, input := range accepted {
			if input.PID == expected.PID && input.NativeID == expected.NativeID && strings.Contains(input.Body, body) {
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// Each effectful request is submitted exactly once. Unknown admission/status is
// resolved only by its original receipt; no timeout authorizes resubmission.
func startupRestartRequest(t *testing.T, ctx context.Context, fleet *testcouch.Fleet, previous agentfixture.Ready, parameters string) (agentfixture.Ready, couchmessage.OperationReceipt) {
	t.Helper()
	result, err := fleet.ActorCLI(ctx, 0, "--switch-agent", "fixture:1", "--agent", previous.Agent, "--agent-args", parameters, "--confirm", "--json")
	var response couchmessage.Response
	if decodeErr := json.Unmarshal([]byte(result.Stdout), &response); decodeErr != nil || result.CleanupUncertain || response.Operation == nil || response.Operation.ID == "" || response.Code != "accepted" && response.Code != "ok" && response.Code != "uncertain" {
		t.Fatalf("edited restart admission: %+v %v decode=%v", result, err, decodeErr)
	}
	id := response.Operation.ID
	receiptCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	var receipt couchmessage.OperationReceipt
	if err := startupArgumentsWait(receiptCtx, func() (bool, error) {
		result, err := fleet.ActorCLI(receiptCtx, 0, "--operation-status", id, "--json")
		var status couchmessage.Response
		if decodeErr := json.Unmarshal([]byte(result.Stdout), &status); decodeErr != nil || result.CleanupUncertain || status.Code != "ok" || status.Operation == nil || status.Operation.ID != id {
			return false, fmt.Errorf("restart receipt %s: %+v %v decode=%v", id, result, err, decodeErr)
		}
		receipt = *status.Operation
		switch receipt.Status {
		case couchmessage.ReceiptQueued, couchmessage.ReceiptRunning, couchmessage.ReceiptUnknown:
			return false, nil
		case couchmessage.ReceiptSucceeded:
			return true, nil
		default:
			return false, fmt.Errorf("edited restart refused/failed: %+v", receipt)
		}
	}); err != nil {
		t.Fatal(err)
	}
	var ready agentfixture.Ready
	if err := startupArgumentsWait(ctx, func() (bool, error) {
		var err error
		ready, err = fleet.Agent(1)
		return err == nil && ready.Agent == previous.Agent && ready.PID != previous.PID && ready.Generation != previous.Generation, nil
	}); err != nil {
		t.Fatalf("edited replacement readiness: previous=%+v last=%+v: %v", previous, ready, err)
	}
	return ready, receipt
}

func startupRestartAssertArgv(t *testing.T, ready agentfixture.Ready, accepted []string, resume bool) {
	t.Helper()
	actual := append([]string{}, accepted...)
	if ready.Agent == "claude" {
		if resume {
			actual = append(actual, "--resume", ready.NativeID)
		} else {
			actual = append(actual, "--session-id", ready.NativeID)
		}
	} else {
		if resume {
			actual = append([]string{"resume", ready.NativeID}, actual...)
		}
		actual = append([]string{"--disable", "shell_snapshot"}, actual...)
		actual = append(actual, "--no-alt-screen")
	}
	if !reflect.DeepEqual(ready.Argv, actual) {
		t.Fatalf("edited native argv: got=%q want=%q", ready.Argv, actual)
	}
}

func startupRestartPark(t *testing.T, ctx context.Context, fleet *testcouch.Fleet) {
	t.Helper()
	forceFleetSelectAction(t, ctx, fleet, "park")
	forceFleetWaitScreen(t, ctx, fleet, " › park")
	forceFleetWaitScreen(t, ctx, fleet, "cancel")
	if err := fleet.Input([]byte("park")); err != nil {
		t.Fatal(err)
	}
	forceFleetPress(t, fleet, "\r", 13, 1)
	if err := startupArgumentsWait(ctx, func() (bool, error) { _, err := fleet.Agent(1); return err != nil, nil }); err != nil {
		t.Fatalf("parked fixture stayed live: %v\n%s", err, fleet.Screen())
	}
	// Agent exit precedes the park transaction's commit and operation release.
	forceFleetWaitScreen(t, ctx, fleet, "park complete")
}

func startupRestartHideTranscript(t *testing.T, fleet *testcouch.Fleet, previous agentfixture.Ready) (string, [sha256.Size]byte) {
	t.Helper()
	if previous.RunID != fleet.RunID || previous.NativeID == "" || previous.Transcript == "" {
		t.Fatal("transcript is not identified by this owned fixture")
	}
	root, err := filepath.EvalSymlinks(fleet.Roots.Root)
	if err != nil {
		t.Fatal(err)
	}
	path, err := filepath.EvalSymlinks(previous.Transcript)
	if err != nil {
		t.Fatal(err)
	}
	if !pathscope.Within(root, path) || !strings.HasSuffix(filepath.Base(path), previous.NativeID+".jsonl") {
		t.Fatalf("transcript is not the exact private native artifact: %s", path)
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > 64*1024 {
		t.Fatalf("private transcript is not a bounded established file: %s %v", path, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	backup := path + ".parked-missing-" + fleet.RunID
	if _, err := os.Lstat(backup); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("transcript backup already exists or cannot be checked: %s %v", backup, err)
	}
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("prior transcript absence not proved: %s %v", path, err)
	}
	return backup, sha256.Sum256(raw)
}
