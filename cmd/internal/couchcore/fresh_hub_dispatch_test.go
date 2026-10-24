package couchcore

import (
	"context"
	"testing"
)

func TestFreshHubDispatchStartsNeverUsedManagedSlot(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	result, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch)}, OperationCall{Name: "start-hub", Implicit: true, Args: map[string]string{"path": local.slot.WorktreeRoot, "action": string(StartFresh)}, Context: context.Background()})
	if err != nil {
		t.Fatal(err)
	}
	child, ok := result.(StartedChild)
	if !ok {
		t.Fatalf("missing child result: %T", result)
	}
	start, ok := child.Started()
	if !ok || start.Handle == nil {
		t.Fatalf("fresh managed hub not started: %+v", result)
	}
	if start.Record.Args.WorkingDir() != local.slot.WorktreeRoot {
		t.Fatalf("wrong hub: %+v", start.Record)
	}
}
