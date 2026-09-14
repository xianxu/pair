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
