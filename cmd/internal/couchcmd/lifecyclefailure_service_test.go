package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestLifecycleFailurePrivateServiceSurvivesReceiptLossAndUsesRecordedSuccess(t *testing.T) {
	r := newParkAdmissionRig(t, false)
	primary := r.before.StartingPath
	cause := "fixture workspace observation unavailable"
	r.catalog.mu.Lock()
	r.catalog.Errors = map[string]error{primary: errors.New(cause)}
	r.catalog.mu.Unlock()
	request := r.request("diagnostic-before-dispatch")
	accepted := r.call(t, request)
	if accepted.Code != "accepted" {
		t.Fatalf("private admission: %+v", accepted)
	}
	r.release()
	failed := r.terminal(t, request.ID)
	if failed.Status == couchmessage.ReceiptSucceeded || failed.FailureNonce == "" || !strings.Contains(failed.Detail, cause) {
		t.Fatalf("pre-dispatch error lost: %+v", failed)
	}
	reopened := couchcore.NewLaunchFailureStore(r.couch.Namespace)
	history, err := reopened.Read()
	if err != nil || len(history) != 1 || history[0].Nonce != failed.FailureNonce || history[0].ParkNonce != r.before.Park.Identity.Nonce {
		t.Fatalf("durable target/correlation: %+v %v", history, err)
	}
	r.catalog.mu.Lock()
	r.catalog.Errors = nil
	r.catalog.mu.Unlock()
	// Restart the actual private transport with a fresh receipt map. The store
	// remains in its portable fixture namespace; no operator socket is used.
	r.service.slotOps = consoleSlotOperations(r.console, r.couch, nil)
	r.service.restart(map[couchmessage.ThreadKey]couchmessage.PaneHandle{{Scope: r.caller.Scope, Tag: r.caller.Tag}: "fixture-hub"})
	r.service.wrapper(r.caller)
	r.service.waitConnected(r.caller, true)
	if status := r.call(t, statusRequest(r.caller, request.ID)); status.Operation == nil || status.Operation.Status != couchmessage.ReceiptUnknown {
		t.Fatalf("old transient receipt unexpectedly survived: %+v", status)
	}
	show := func() couchcore.ShowResult {
		value, err := couchcore.DispatchOperation(couchcore.OperationExecutors{DirectStore: couchcore.DirectStoreExecutor(r.couch)}, couchcore.OperationCall{Name: "show", Args: map[string]string{"repo-scope": r.before.Address.RepoScope, "ref": primary}, Implicit: true, Context: context.Background()})
		if err != nil {
			t.Fatal(err)
		}
		return value.(couchcore.ShowResult)
	}
	before := show()
	var rendered bytes.Buffer
	renderLaunchFailures(&rendered, before.LaunchFailures, before.LaunchFailureError)
	if len(before.LaunchFailures) != 1 || !strings.Contains(rendered.String(), failed.FailureNonce) || !strings.Contains(rendered.String(), cause) {
		t.Fatalf("show after private restart lost retained reason: %s", rendered.String())
	}
	successRequest := r.request("diagnostic-successor")
	if response := r.call(t, successRequest); response.Code != "accepted" {
		t.Fatalf("successor admission: %+v", response)
	}
	if success := r.terminal(t, successRequest.ID); success.Status != couchmessage.ReceiptSucceeded {
		t.Fatalf("successor: %+v", success)
	}
	// Recovery retirement has no success timestamp; it cannot clear failure text.
	if after := show(); len(after.LaunchFailures) != 1 {
		t.Fatal("retirement hid failure without a success event")
	}
	beforeBytes, err := os.ReadFile(reopened.Path())
	if err != nil {
		t.Fatal(err)
	}
	current, err := r.couch.Threads.GetThread(r.before.Address)
	if err != nil {
		t.Fatal(err)
	}
	profile := current.LatestLaunchProfile
	nonce := "fixture-committed-success"
	current, err = r.couch.Threads.CommitStartClaim(current.Address, current.Revision, primary+"/.git", history[0].FailedAt.Add(time.Second), couchcore.StartEvent{Kind: couchcore.StartClaimed, Shape: couchcore.StartFreshExisting, Nonce: nonce, Owner: couchcore.SupervisorOwner{PID: 100, Identity: "fixture-owner"}, Profile: profile})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []couchcore.StartEvent{{Kind: couchcore.StartHelperRecorded, Nonce: nonce, Helper: couchcore.ProcessIdentity{PID: 101, Identity: "fixture-helper"}}, {Kind: couchcore.StartRegistered, Nonce: nonce}} {
		current, err = r.couch.Threads.AdvanceStart(current.Address, current.Revision, event)
		if err != nil {
			t.Fatal(err)
		}
	}
	if after := show(); len(after.LaunchFailures) != 0 {
		t.Fatalf("recorded committed start did not hide failure: %+v", after.LaunchFailures)
	}
	afterBytes, err := os.ReadFile(reopened.Path())
	if err != nil || !bytes.Equal(beforeBytes, afterBytes) {
		t.Fatalf("success rewrote failure log: %v", err)
	}
	history, err = reopened.Read()
	if err != nil || len(history) != 1 || history[0].Nonce != failed.FailureNonce {
		t.Fatalf("historical failure lost: %+v %v", history, err)
	}
}
