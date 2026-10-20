package couchcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestLifecycleFailureNonceSurvivesEveryReceiptClassification(t *testing.T) {
	tests := []struct {
		name   string
		cause  error
		status couchmessage.ReceiptStatus
		code   string
	}{
		{"slot-refused", &couchcore.SlotOperationError{Code: "not-offered", Detail: "operation unavailable"}, couchmessage.ReceiptRefused, "not-offered"},
		{"recover-refused", &couchcore.RecoverRefusal{Code: "park-active", Detail: "park still finishing"}, couchmessage.ReceiptRefused, "park-active"},
		{"reap-refused", &couchcore.ReapRefusal{Detail: "live holder remains"}, couchmessage.ReceiptRefused, "reap-refused"},
		{"confirmation-required", &couchcore.RecoveryConfirmationRequired{}, couchmessage.ReceiptRefused, "confirmation-required"},
		{"generic-failed", errors.New("terminal execution failed"), couchmessage.ReceiptFailed, ""},
		{"recovery-unconfirmed", &couchcore.RecoveryLaunchUnconfirmed{Nonce: "launch-nonce", Cause: errors.New("registration unavailable")}, couchmessage.ReceiptFailed, "unconfirmed"},
		{"reload-unconfirmed", &couchcore.ReloadUnconfirmed{Detail: "reload acknowledgement unavailable"}, couchmessage.ReceiptFailed, "unconfirmed"},
		{"unready", &couchcore.RestartUnready{Detail: "readiness deadline elapsed"}, couchmessage.ReceiptFailed, "unready"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := couchcore.NewFakeLaunchFailureStore()
			n := 0
			recorder := couchcore.NewLifecycleFailureRecorder(store, func() time.Time { return time.Unix(100, 0) }, func() (string, error) { n++; return fmt.Sprintf("attempt-%d", n), nil })
			attempt, err := recorder.Begin("resume", couchcore.LifecycleFailureTarget{StartingPath: "/physical/slot", Slot: "pair:5"})
			if err != nil {
				t.Fatal(err)
			}
			value, finished := attempt.Finish(nil, tt.cause)
			if !errors.Is(finished, tt.cause) {
				t.Fatalf("original error lost: %v", finished)
			}
			outcome := slotOperationOutcome(value, finished)
			if outcome.Status != tt.status || outcome.Code != tt.code {
				t.Fatalf("classification changed: %+v", outcome)
			}
			receipt, err := couchmessage.ApplyReceiptEvent(couchmessage.OperationReceipt{}, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptAdmit, ID: "receipt", Op: "resume", Target: "pair:5"})
			if err != nil {
				t.Fatal(err)
			}
			receipt, err = couchmessage.ApplyReceiptEvent(receipt, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptFinish, ID: "receipt", Outcome: outcome})
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(receipt)
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &wire); err != nil {
				t.Fatal(err)
			}
			var nonce string
			_ = json.Unmarshal(wire["FailureNonce"], &nonce)
			if nonce != attempt.Nonce() {
				t.Fatalf("receipt nonce lost: got %q want %q; %s", nonce, attempt.Nonce(), encoded)
			}
			history, err := store.Read()
			if err != nil || len(history) != 1 || history[0].Nonce != nonce {
				t.Fatalf("receipt/store mismatch: %+v %v", history, err)
			}
			expected := map[string]string{"slot-refused": "refused", "recover-refused": "refused", "reap-refused": "refused", "confirmation-required": "refused", "generic-failed": "failed", "recovery-unconfirmed": "unproven", "reload-unconfirmed": "unproven", "unready": "unproven"}
			if history[0].Outcome != expected[tt.name] || history[0].Phase != "" {
				t.Fatalf("durable diagnostic misclassified the outcome: %+v", history[0])
			}
		})
	}
}

func TestLifecycleFailureReceiptMetadataPreservesOriginalTypedResult(t *testing.T) {
	value := notedResult{value: lifecycleRecordedOutcome{value: couchcore.RebootResult{Archived: couchcore.ThreadAddress{Tag: "old-thread"}}, nonce: "failure-exact", warning: "diagnostic warning"}, note: "admission note"}
	outcome := slotOperationOutcome(value, nil)
	if outcome.Status != couchmessage.ReceiptSucceeded || outcome.Archived != "old-thread" || outcome.FailureNonce != "failure-exact" || !strings.Contains(outcome.Warning, "diagnostic warning") || !strings.Contains(outcome.Warning, "admission note") {
		t.Fatalf("diagnostic envelope changed the lifecycle result: %+v", outcome)
	}
}

func TestSessionCleanupUnconfirmedReceiptRetainsDiagnosticIdentity(t *testing.T) {
	store := couchcore.NewFakeLaunchFailureStore()
	recorder := couchcore.NewLifecycleFailureRecorder(store, func() time.Time { return time.Unix(100, 0) }, func() (string, error) { return "cleanup-attempt", nil })
	attempt, err := recorder.Begin("resume", couchcore.LifecycleFailureTarget{StartingPath: "/physical/slot", Slot: "pair:5"})
	if err != nil {
		t.Fatal(err)
	}
	value, finished := attempt.Finish(nil, &couchcore.SessionCleanupUnconfirmed{Session: "📁9-4", Cause: errors.New("observation refused")})
	out := slotOperationOutcome(value, finished)
	if out.Status != couchmessage.ReceiptFailed || out.Code != "unconfirmed" || out.FailureNonce != attempt.Nonce() || !strings.Contains(out.Detail, "ownership retained") {
		t.Fatalf("cleanup uncertainty lost at receipt: %+v", out)
	}
	history, err := store.Read()
	if err != nil || len(history) != 1 || history[0].Phase != "launch" || history[0].Outcome != "unproven" {
		t.Fatalf("cleanup history rejected: %+v %v", history, err)
	}
}
