package couchcmd

import (
	"context"
	"encoding/json"
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

// Diagnostic characterization for pair#797, not successful reboot acceptance.
// Preserve the actual judged native reply before the disposable store is removed.
func TestLiveRecoveryOwnerDiagnostic(t *testing.T) {
	if os.Getenv("PAIR_LIVE_RECOVERY_OWNER") != "1" {
		t.Skip("set PAIR_LIVE_RECOVERY_OWNER=1 for disposable ownership capture")
	}
	binaries := testcouch.Build(t)
	for _, kind := range []string{"claude", "codex"} {
		if !t.Run(kind, func(t *testing.T) {
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
			if !provenPreviewRefusal(out, callErr) || result.Status != couchmessage.ReceiptFailed || result.FailureNonce == "" || !strings.Contains(result.Detail, "unknown pane owner") {
				t.Fatalf("expected diagnostic refusal not reproduced: %+v %+v %v", out, result, callErr)
			}
			namespace, err := couchcore.ExistingCouchNamespace(fleet.Roots.Mutable["store"])
			if err != nil {
				t.Fatal(err)
			}
			history, err := couchcore.NewLaunchFailureStore(namespace).Read()
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, failure := range history {
				if failure.Nonce != result.FailureNonce {
					continue
				}
				found = true
				t.Logf("judged native query nonce=%s executable=%q argv=%q version=%q exit=%v stdout_bytes=%d stdout_truncated=%t stderr_bytes=%d stderr_truncated=%t", failure.Nonce, failure.ZellijExecutable, failure.ZellijArgs, failure.ZellijVersion, failure.ZellijExit, failure.ZellijStdoutBytes, failure.ZellijStdoutTruncated, failure.ZellijStderrBytes, failure.ZellijStderrTruncated)
				t.Logf("judged native stdout=%q", failure.ZellijStdout)
				t.Logf("judged native stderr=%q", failure.ZellijStderr)
				if len(failure.ZellijStdout) == 0 || failure.ZellijExit == nil || *failure.ZellijExit != 0 {
					t.Fatal("refusal did not preserve its successful native pane query")
				}
			}
			if !found {
				t.Fatal("correlated refusal evidence missing before cleanup")
			}
			cleanup, stopCleanup := context.WithTimeout(context.Background(), 15*time.Second)
			defer stopCleanup()
			if err = fleet.Close(cleanup); err != nil {
				t.Fatal(err)
			}
			if err = fleet.AssertCleaned(); err != nil {
				t.Fatal(err)
			}
		}) {
			break
		}
	}
}
