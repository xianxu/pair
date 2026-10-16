package couchcore

import (
	"bytes"
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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
		if i%10 == 0 {
			request := testContinuationRequest(t, created)
			if _, err := store.PublishContinuation(created.Address, created.Revision, request); err != nil {
				t.Fatal(err)
			}
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
// runningContinuation publishes a request on record and begins it: an
// in-flight compaction, the one state that still constrains a thread.
func runningContinuation(t *testing.T, store *ThreadStore, record ThreadRecord) (ThreadRecord, checkpoint.Request) {
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
	return record, *record.Continuation
}

// legacyFailedContinuation stores what a Couch before #522 left on a thread
// whose compaction failed: a retained request in phase failed. It writes the
// record directly, as no event reaches that phase any more.
func legacyFailedContinuation(t *testing.T, store *ThreadStore, record ThreadRecord) (ThreadRecord, checkpoint.Request) {
	t.Helper()
	running, request := runningContinuation(t, store, record)
	failed := request.Clone()
	failed.Phase, failed.Failure = checkpoint.Failed, "operator input interrupted automatic orientation"
	if _, err := store.updateExistingThread(running.Address, running.Revision, func(next *ThreadRecord) error {
		next.Continuation = &failed
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	read, err := store.GetThread(running.Address)
	if err != nil {
		t.Fatal(err)
	}
	return read, failed
}

// A request that failed before #522 blocked its thread until retry or
// dismissal. It now reads as the one non-blocking line, and the next write
// persists that.
func TestALegacyFailedRequestReadsAsItsLine(t *testing.T) {
	t.Parallel()

	store, _ := newTestThreadStore(t)
	created, err := store.CreateThread(validThreadRecord(t))
	if err != nil {
		t.Fatal(err)
	}
	read, request := legacyFailedContinuation(t, store, created)
	m := read.ContinuationMissed
	if read.Continuation != nil || m == nil || m.RequestID != request.ID || m.Reason != request.Failure || m.Path != request.Checkpoint.SourcePath || m.TargetMayExist {
		t.Fatalf("legacy failed request read as %+v / %+v", read.Continuation, m)
	}
	if err := continuationGuard(read); err != nil {
		t.Fatalf("the line blocks: %v", err)
	}
	written, err := store.updateExistingThread(read.Address, read.Revision, func(next *ThreadRecord) error { next.Name = "renamed"; return nil })
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(store.recordPath(written.Address))
	if err != nil || bytes.Contains(raw, []byte(`"phase":"failed"`)) || !bytes.Contains(raw, []byte(`"continuation_missed"`)) {
		t.Fatalf("the next write did not persist the line: %s %v", raw, err)
	}
}

// MissContinuation ends only the exact request, clearing it and writing the
// line in one revision; a refusal writes nothing. Publishing the next request
// clears the line (#522).
func TestMissContinuationClearsTheRequestAndLeavesOneLine(t *testing.T) {
	t.Parallel()

	store, _ := newTestThreadStore(t)
	created, err := store.CreateThread(validThreadRecord(t))
	if err != nil {
		t.Fatal(err)
	}
	running, request := runningContinuation(t, store, created)
	miss := func(r ThreadRecord) checkpoint.Miss {
		return checkpoint.MissOf(*r.Continuation, "launch failed", running.CreatedAt, false)
	}
	if _, err := store.MissContinuation(running.Address, running.Revision, strings.Repeat("0", 64), miss); err == nil {
		t.Fatal("an obsolete request was missed")
	}
	if after, err := store.GetThread(running.Address); err != nil || after.Revision != running.Revision {
		t.Fatalf("a refusal wrote: %v", err)
	}
	missed, err := store.MissContinuation(running.Address, running.Revision, request.ID, miss)
	if err != nil || missed.Continuation != nil || missed.ContinuationMissed == nil || missed.ContinuationMissed.RequestID != request.ID || missed.Revision != running.Revision+1 {
		t.Fatalf("miss = %+v / %+v, %v", missed.Continuation, missed.ContinuationMissed, err)
	}
	next := testContinuationRequest(t, missed)
	next.Source.LaunchOrdinal = 3
	next.ID = checkpoint.RequestID(missed.Address.RepoScope, string(missed.Address.Tag), 3, next.Checkpoint.Digest)
	published, err := store.PublishContinuation(missed.Address, missed.Revision, next)
	if err != nil || published.ContinuationMissed != nil || published.Continuation == nil {
		t.Fatalf("publishing after a miss = %+v, %v", published.ContinuationMissed, err)
	}
}

// The watcher's scan (#507) reuses a thread's status while its record file is
// the one it read, and reads again once a write replaces it.
func TestContinuationRequestsReuseAnUnchangedRecord(t *testing.T) {
	t.Parallel()

	store, _ := newTestThreadStore(t)
	record, err := store.CreateThread(validThreadRecord(t))
	if err != nil {
		t.Fatal(err)
	}
	c := &Couch{Threads: store}
	scan := func() ([]ContinuationStatus, error) {
		return c.ContinuationRequests(context.Background(), []ThreadAddress{record.Address})
	}
	if statuses, err := scan(); err != nil || len(statuses) != 0 {
		t.Fatalf("before a request: %+v %v", statuses, err)
	}
	request := testContinuationRequest(t, record)
	if _, err := store.PublishContinuation(record.Address, record.Revision, request); err != nil {
		t.Fatal(err)
	}
	if statuses, err := scan(); err != nil || len(statuses) != 1 || statuses[0].RequestID != request.ID {
		t.Fatalf("a published request was not read: %+v %v", statuses, err)
	}
	// Overwrite in place, keeping the file, its size and its mtime: the scan
	// must not read it (it would fail to decode).
	path := store.recordPath(record.Address)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(bytes.Repeat([]byte("x"), int(info.Size()))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if statuses, err := scan(); err != nil || len(statuses) != 1 {
		t.Fatalf("an unchanged record was read again: %+v %v", statuses, err)
	}
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := scan(); err == nil {
		t.Fatal("a changed record was not read again")
	}
}
