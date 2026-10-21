package couchschedule

import (
	"context"
	"testing"
)

func TestFakeTaskRuntimeRecordsHeldAuthorizedCancellation(t *testing.T) {
	f := NewFakeTaskRuntime()
	f.Hold()
	f.Enqueue(ExecutionResult{Exit: 10, Outcome: "wake", Output: []byte("notice")})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan ExecutionResult, 1)
	r := ExecutionRequest{HubRoot: "/hub", Task: "task", RunID: "run1"}
	go func() { result, _ := f.Execute(ctx, r, func(OwnerIdentity) error { return nil }); done <- result }()
	<-f.Started()
	if rows := f.Runs(); len(rows) != 1 || !rows[0].Authorized || rows[0].Finished {
		t.Fatalf("held state %#v", rows)
	}
	cancel()
	if got := <-done; got.Outcome != "cancelled" {
		t.Fatalf("cancelled fake %#v", got)
	}
	if rows := f.Runs(); !rows[0].Finished || !rows[0].Cancelled {
		t.Fatalf("finished state %#v", rows)
	}
}
