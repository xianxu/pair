package couchcore

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestContinuationPublishExecuteAndReceipt(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	proof := ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 2}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) { return proof, nil }
	path := filepath.Join(t.TempDir(), "handoff.md")
	body := "---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nVerify the durable token.\n"
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	status, err := env.Couch.RequestContinuation(context.Background(), source.Address, proof, path)
	if err != nil {
		t.Fatal(err)
	}
	again, err := env.Couch.RequestContinuation(context.Background(), source.Address, proof, path)
	if err != nil || again.RequestID != status.RequestID {
		t.Fatalf("duplicate = %+v %v", again, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	registered := false
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return registered, nil }
	env.Runner.AfterAcknowledge = func(string) error {
		registered = true
		env.Artifacts.SetPairSession(source.Address, "pair-exact", true)
		return nil
	}
	result, err := env.Couch.Continue(context.Background(), source.Address, status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	started, ok := result.Started()
	if !ok || started.Record.Thread != source.Address {
		t.Fatalf("start = %+v", result)
	}
	if result.Status.Phase != checkpoint.Running || result.Orientation == nil {
		t.Fatalf("premature completion %+v", result)
	}
	materialized, err := os.ReadFile(env.Couch.continuationPath(source.Address))
	if err != nil || string(materialized) != body {
		t.Fatalf("snapshot %q %v", materialized, err)
	}
	env.Proc.Set(started.Record.PID, started.Record.Identity)
	env.Couch.OrientationStatus = func(context.Context, ThreadAddress, string, string) (orientation.DeliveryState, error) {
		return orientation.DeliveryState{Phase: orientation.DeliverySubmitted}, nil
	}
	completed, err := env.Couch.ReconcileContinuation(context.Background(), source.Address, status.RequestID, result.Status.Attempt)
	if err != nil || completed.Phase != checkpoint.Complete {
		t.Fatalf("receipt = %+v %v", completed, err)
	}
}

func TestContinuationRejectsChangedSourceBeforePark(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return false, nil }
	request := testContinuationRequest(t, source)
	request.Source.Agent = "claude"
	request.Source.Session = "pair-exact"
	request.Source.Helper = checkpoint.Process{PID: 42, Identity: "pair-helper"}
	if _, err := env.Couch.Threads.PublishContinuation(source.Address, source.Revision, request); err != nil {
		t.Fatal(err)
	}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) {
		return ContinuationSource{Agent: "codex", Session: "pair-exact", LaunchOrdinal: 3}, nil
	}
	if _, err := env.Couch.Continue(context.Background(), source.Address, request.ID); err == nil {
		t.Fatal("changed generation accepted")
	}
	current, _ := env.Couch.Threads.GetThread(source.Address)
	if current.Continuation.Phase != checkpoint.Failed || current.Park != nil {
		t.Fatalf("wrong failure state %+v", current)
	}
}

func TestContinuationRejectsChangedCheckpointDigestBeforePublication(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	proof := ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 2, ExpectedDigest: "outdated-digest"}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) {
		return ContinuationSource{Agent: proof.Agent, Session: proof.Session, LaunchOrdinal: proof.LaunchOrdinal}, nil
	}
	path := filepath.Join(t.TempDir(), "checkpoint.md")
	if err := os.WriteFile(path, []byte("---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nModified handoff.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.RequestContinuation(context.Background(), source.Address, proof, path); err == nil {
		t.Fatal("changed snapshot accepted")
	}
	record, _ := env.Couch.Threads.GetThread(source.Address)
	if record.Continuation != nil || record.Park != nil {
		t.Fatal("digest failure mutated the source")
	}
}

func TestContinuationConcurrentPublicationDeduplicates(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	proof := ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 2}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) { return proof, nil }
	path := filepath.Join(t.TempDir(), "checkpoint.md")
	if err := os.WriteFile(path, []byte("---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nOne accepted request.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	results := make(chan ContinuationStatus, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			s, err := env.Couch.RequestContinuation(context.Background(), source.Address, proof, path)
			results <- s
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	id := ""
	for s := range results {
		if id != "" && id != s.RequestID {
			t.Fatal("publication created multiple requests")
		}
		id = s.RequestID
	}
	record, err := env.Couch.Threads.GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if record.Continuation.ID != id || record.Park != nil || record.Continuation.Phase != checkpoint.Pending {
		t.Fatalf("concurrent publication %+v", record)
	}
}
