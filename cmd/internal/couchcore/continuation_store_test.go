package couchcore

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"os"
	"reflect"
	"strings"
	"testing"
)

func testContinuationRequest(t *testing.T, record ThreadRecord) checkpoint.Request {
	t.Helper()
	agent := "codex"
	if record.LatestLaunchProfile != nil {
		agent = record.LatestLaunchProfile.Agent
	}
	cp, err := checkpoint.New("/tmp/continuation.md", "---\ntype: continuation\nagent: "+agent+"\n---\n## NEXT ACTION\ntest recovery\n")
	if err != nil {
		t.Fatal(err)
	}
	return checkpoint.Request{Version: 1, ID: checkpoint.RequestID(record.Address.RepoScope, string(record.Address.Tag), 2, cp.Digest), Checkpoint: cp, Source: checkpoint.Source{Agent: agent, Session: "pair-source", LaunchOrdinal: 2, Helper: checkpoint.Process{PID: 42, Identity: "source"}}, CreatedAt: record.CreatedAt, Phase: checkpoint.Pending}
}
func TestContinuationRecordRoundTripAndCAS(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := validThreadRecord(t)
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	request := testContinuationRequest(t, created)
	published, err := store.PublishContinuation(created.Address, created.Revision, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.PublishContinuation(created.Address, created.Revision, request); err == nil {
		t.Fatal("stale publication accepted")
	}
	reread, err := store.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reread.Continuation, published.Continuation) {
		t.Fatal("snapshot lost on reopen")
	}
	running, err := store.AdvanceContinuation(created.Address, reread.Revision, checkpoint.Event{Kind: checkpoint.Begin, RequestID: request.ID, Attempt: "attempt-1"})
	if err != nil {
		t.Fatal(err)
	}
	target := checkpoint.Process{PID: 43, Identity: "target"}
	// Probe deep-copy isolation independently of the lifecycle, whose exact
	// parked-to-registered transition is covered by the executor fixture.
	running.Continuation.Target = &checkpoint.Target{Process: target, ObservedAt: running.CreatedAt}
	cloned := cloneThreadRecord(running)
	cloned.Continuation.Target.Identity = "changed"
	persisted := toPersistedThreadRecord(running)
	persisted.Continuation.Target.Identity = "changed"
	if running.Continuation.Target.Identity != "target" {
		t.Fatal("clone or mirror aliased request")
	}
	bad := cloneThreadRecord(running)
	bad.Continuation.Checkpoint.Body = "tampered"
	if ValidateThreadRecord(bad) == nil {
		t.Fatal("invalid checkpoint accepted")
	}
}

func TestContinuationRequestScanIsolatesUnreadableSlot(t *testing.T) {
	store, _ := newTestThreadStore(t)
	first := validThreadRecord(t)
	first, err := store.CreateThread(first)
	if err != nil {
		t.Fatal(err)
	}
	second := validThreadRecord(t)
	second.Address.Tag = "couch-1111111111111111"
	second, err = store.CreateThread(second)
	if err != nil {
		t.Fatal(err)
	}
	request := testContinuationRequest(t, second)
	if _, err := store.PublishContinuation(second.Address, second.Revision, request); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.recordPath(first.Address), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	statuses, err := (&Couch{Threads: store}).ContinuationRequests(context.Background(), []ThreadAddress{first.Address, second.Address})
	if err == nil || len(statuses) != 1 || statuses[0].RequestID != request.ID {
		t.Fatalf("unreadable slot blocked other requests: %+v %v", statuses, err)
	}
}

func TestContinuationRequestsBoundedOwnedSet(t *testing.T) {
	store, _ := newTestThreadStore(t)
	var addresses []ThreadAddress
	wanted := map[ThreadAddress]string{}
	for i := 0; i < 100; i++ {
		record := validThreadRecord(t)
		record.Address.Tag = ThreadTag(fmt.Sprintf("couch-%016x", i+1))
		created, err := store.CreateThread(record)
		if err != nil {
			t.Fatal(err)
		}
		request := testContinuationRequest(t, created)
		if _, err := store.PublishContinuation(created.Address, created.Revision, request); err != nil {
			t.Fatal(err)
		}
		if i%10 == 0 {
			addresses = append(addresses, created.Address, created.Address)
			wanted[created.Address] = request.ID
		}
		// An unrequested corrupt row makes a repository-wide scan observable.
		if i == 99 {
			if err := os.WriteFile(store.recordPath(created.Address), []byte("broken"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	statuses, err := (&Couch{Threads: store}).ContinuationRequests(context.Background(), addresses)
	if err != nil {
		t.Fatalf("read outside requested set: %v", err)
	}
	if len(statuses) != len(wanted) {
		t.Fatalf("statuses = %d, want %d unique owned addresses", len(statuses), len(wanted))
	}
	for _, status := range statuses {
		id, ok := wanted[status.Address]
		if !ok || id != status.RequestID || status.Phase != checkpoint.Pending {
			t.Fatalf("unexpected status %+v", status)
		}
		delete(wanted, status.Address)
	}
	encoded, err := json.Marshal(statuses)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"test recovery", "NEXT ACTION", "checkpoint", "digest", "source_path", "body"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("metadata projection leaked %q: %s", forbidden, encoded)
		}
	}
	if len(encoded) > 4096 {
		t.Fatalf("ten projected requests unexpectedly large: %d bytes", len(encoded))
	}
}

// failedContinuation publishes a request and drives it to Failed through the
// production transitions, returning the record and the request.
func failedContinuation(t *testing.T, store *ThreadStore, record ThreadRecord) (ThreadRecord, checkpoint.Request) {
	t.Helper()
	request := testContinuationRequest(t, record)
	record, err := store.PublishContinuation(record.Address, record.Revision, request)
	if err != nil {
		t.Fatal(err)
	}
	record, err = store.AdvanceContinuation(record.Address, record.Revision, checkpoint.Event{Kind: checkpoint.Begin, RequestID: request.ID, Attempt: "start-0102030405060708"})
	if err != nil {
		t.Fatal(err)
	}
	record, err = store.AdvanceContinuation(record.Address, record.Revision, checkpoint.Event{Kind: checkpoint.Fail, RequestID: request.ID, Attempt: "start-0102030405060708", Failure: "operator input interrupted automatic orientation"})
	if err != nil {
		t.Fatal(err)
	}
	return record, request
}

// Dismissal retires ONLY the exact failed request, and every refusal writes
// nothing -- asserted by the unchanged revision, not a bare error (#280).
func TestDismissFailedContinuationRetiresOnlyTheExactFailedRequest(t *testing.T) {
	store, _ := newTestThreadStore(t)
	created, err := store.CreateThread(validThreadRecord(t))
	if err != nil {
		t.Fatal(err)
	}
	refuses := func(t *testing.T, r ThreadRecord, id, want string) {
		t.Helper()
		_, err := store.DismissFailedContinuation(r.Address, r.Revision, id)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("dismiss(%q) = %v, want refusal %q", id, err, want)
		}
		after, err := store.GetThread(r.Address)
		if err != nil || after.Revision != r.Revision {
			t.Fatalf("refusal wrote: revision %d -> %d (%v)", r.Revision, after.Revision, err)
		}
	}

	refuses(t, created, "", "no continuation request")
	request := testContinuationRequest(t, created)
	pending, err := store.PublishContinuation(created.Address, created.Revision, request)
	if err != nil {
		t.Fatal(err)
	}
	refuses(t, pending, request.ID, "is pending; only a failed continuation can be dismissed")
	running, err := store.AdvanceContinuation(pending.Address, pending.Revision, checkpoint.Event{Kind: checkpoint.Begin, RequestID: request.ID, Attempt: "start-0102030405060708"})
	if err != nil {
		t.Fatal(err)
	}
	refuses(t, running, request.ID, "is running; only a failed continuation can be dismissed")
	failed, err := store.AdvanceContinuation(running.Address, running.Revision, checkpoint.Event{Kind: checkpoint.Fail, RequestID: request.ID, Attempt: "start-0102030405060708", Failure: "interrupted"})
	if err != nil {
		t.Fatal(err)
	}
	refuses(t, failed, strings.Repeat("0", 64), "obsolete continuation request")

	dismissed, err := store.DismissFailedContinuation(failed.Address, failed.Revision, request.ID)
	if err != nil || dismissed.Continuation != nil || dismissed.Revision <= failed.Revision {
		t.Fatalf("dismiss = %+v, %v; want the request deleted in a new revision", dismissed.Continuation, err)
	}
	refuses(t, dismissed, request.ID, "no continuation request")
}

// Retry and dismissal race through the revision CAS: whichever commits first
// wins, and the other refuses on what it finds -- both orders, driven.
func TestRetryAndDismissalSerializeInEitherOrder(t *testing.T) {
	for _, dismissFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "retry-first", true: "dismiss-first"}[dismissFirst], func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			created, err := store.CreateThread(validThreadRecord(t))
			if err != nil {
				t.Fatal(err)
			}
			failed, request := failedContinuation(t, store, created)
			retry := checkpoint.Event{Kind: checkpoint.RetryAbsent, RequestID: request.ID, Attempt: "start-1111111111111111"}
			if dismissFirst {
				gone, err := store.DismissFailedContinuation(failed.Address, failed.Revision, request.ID)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := store.AdvanceContinuation(gone.Address, gone.Revision, retry); err == nil || !strings.Contains(err.Error(), "no continuation request") {
					t.Fatalf("retry after dismissal = %v", err)
				}
				return
			}
			retried, err := store.AdvanceContinuation(failed.Address, failed.Revision, retry)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.DismissFailedContinuation(retried.Address, retried.Revision, request.ID); err == nil || !strings.Contains(err.Error(), "is running") {
				t.Fatalf("dismissal after retry = %v", err)
			}
		})
	}
}
