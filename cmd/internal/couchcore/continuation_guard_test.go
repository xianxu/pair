package couchcore

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"testing"
)

func TestContinuationBlocksCompetingTransitions(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	request := testContinuationRequest(t, source)
	source, err := env.Couch.Threads.PublishContinuation(source.Address, source.Revision, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.Relaunch(context.Background(), source.Address); err == nil {
		t.Fatal("relaunch bypassed continuation")
	}
	if _, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", nil); err == nil {
		t.Fatal("agent switch bypassed continuation")
	}
	empty, err := env.Couch.Threads.updateExistingThread(source.Address, source.Revision, func(next *ThreadRecord) error { next.Incarnations = nil; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if err := archivableRecord(empty); err != nil {
		t.Fatalf("explicit archive must retain an unoccupied pending continuation: %v", err)
	}
	profile := LaunchProfile{Agent: "codex", Argv: []string{}}
	event := StartEvent{Kind: StartClaimed, Nonce: "start-1111111111111111", Owner: SupervisorOwner{PID: 100, Identity: "owner"}, Profile: &profile}
	if _, err := env.Couch.Threads.CommitStartClaim(empty.Address, empty.Revision, "repo", env.Now, event); err == nil {
		t.Fatal("cold start bypassed continuation")
	}
	running, err := env.Couch.Threads.AdvanceContinuation(empty.Address, empty.Revision, checkpoint.Event{Kind: checkpoint.Begin, RequestID: request.ID, Attempt: event.Nonce})
	if err != nil {
		t.Fatal(err)
	}
	event.Shape = StartFreshExisting
	if _, err := env.Couch.Threads.CommitStartClaim(running.Address, running.Revision, "repo", env.Now, event); err != nil {
		t.Fatalf("authorized attempt refused: %v", err)
	}
}
