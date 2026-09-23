package couchcore

import (
	"context"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func slotContinuationFixture(t *testing.T, phase checkpoint.Phase, target bool) (*testEnv, *ThreadStore, ThreadRecord) {
	t.Helper()
	env, local := slotRecoveryOperationFixture(t)
	record := validThreadRecord(t)
	scope, err := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	if err != nil {
		t.Fatal(err)
	}
	record.Address.RepoScope = scope.Key
	record.StartingPath, record.WorkingPath = local.slot.WorktreeRoot, local.slot.WorktreeRoot
	record.Reservation = false
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	record, err = local.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	request := testContinuationRequest(t, record)
	if target {
		request.SourceAbsence = &checkpoint.SourceAbsence{Session: request.Source.Session, LaunchOrdinal: request.Source.LaunchOrdinal, ObservedAt: env.Now, RecordRevision: record.Revision}
	}
	record, err = local.PublishContinuation(record.Address, record.Revision, request)
	if err != nil {
		t.Fatal(err)
	}
	if phase != checkpoint.Pending {
		record, err = local.AdvanceContinuation(record.Address, record.Revision, checkpoint.Event{Kind: checkpoint.Begin, RequestID: request.ID, Attempt: "continuation-attempt"})
		if err != nil {
			t.Fatal(err)
		}
	}
	if target {
		process := checkpoint.Process{PID: 43, Identity: "target"}
		record, err = local.AdvanceContinuation(record.Address, record.Revision, checkpoint.Event{Kind: checkpoint.Registered, RequestID: request.ID, Attempt: "continuation-attempt", Target: &process, At: env.Now})
		if err != nil {
			t.Fatal(err)
		}
	}
	if phase == checkpoint.Failed {
		record, err = local.AdvanceContinuation(record.Address, record.Revision, checkpoint.Event{Kind: checkpoint.Fail, RequestID: request.ID, Attempt: "continuation-attempt", Failure: "interrupted"})
		if err != nil {
			t.Fatal(err)
		}
	}
	return env, local, record
}

func dispatchSlotContinuation(env *testEnv, op, path string) (any, error) {
	return DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch), DirectStore: DirectStoreExecutor(env.Couch)}, OperationCall{Name: op, Args: map[string]string{"path": path, "agent": "claude"}, Implicit: true, Context: context.Background()})
}

func TestSlotFreshContinuationProtectsLiveAndUnknownOwners(t *testing.T) {
	for _, target := range []bool{false, true} {
		for _, unknown := range []bool{false, true} {
			name := "source"
			if target {
				name = "target"
			}
			if unknown {
				name += "-unknown"
			}
			t.Run(name, func(t *testing.T) {
				env, local, record := slotContinuationFixture(t, checkpoint.Running, target)
				pid, identity := 42, "source"
				if target {
					pid, identity = 43, "target"
				}
				if unknown {
					env.Proc.SetUnknown(pid)
				} else {
					env.Proc.Set(pid, identity)
				}
				_, err := dispatchSlotContinuation(env, "fresh-slot", record.StartingPath)
				if err == nil {
					t.Fatal("fresh replaced a continuation whose owner was not proved stopped")
				}
				if !strings.Contains(err.Error(), "owner") {
					t.Fatalf("refused for unrelated reason: %v", err)
				}
				current, readErr := local.GetThread(record.Address)
				if readErr != nil || current.Revision != record.Revision || len(env.Runner.Ops) != 0 {
					t.Fatalf("refusal changed state: %+v %v %v", current, readErr, env.Runner.Ops)
				}
			})
		}
	}
}

func TestSlotFreshRetainsStoppedContinuationWithoutArchiveGesture(t *testing.T) {
	for _, phase := range checkpoint.AllPhases() {
		if phase != checkpoint.Pending && phase != checkpoint.Failed {
			continue
		}
		t.Run(string(phase), func(t *testing.T) {
			env, local, record := slotContinuationFixture(t, phase, false)
			result, err := dispatchSlotContinuation(env, "fresh-slot", record.StartingPath)
			if err != nil {
				t.Fatal(err)
			}
			start := result.(StartResult)
			if start.Handle == nil || start.Record.Thread == record.Address {
				t.Fatalf("not fresh: %+v", start)
			}
			retained, err := local.ArchivedThreads()
			if err != nil || len(retained) != 1 || retained[0].Continuation == nil || retained[0].Continuation.Phase != phase {
				t.Fatalf("continuation evidence lost: %+v %v", retained, err)
			}
		})
	}
}

func TestSlotOpenColdUsesContinuationGuard(t *testing.T) {
	env, local, record := slotContinuationFixture(t, checkpoint.Failed, false)
	_, err := dispatchSlotContinuation(env, "open-slot", record.StartingPath)
	if err == nil || !strings.Contains(err.Error(), "continuation "+record.Continuation.ID+" is failed;") {
		t.Fatalf("cold slot open bypassed continuation guard: %v", err)
	}
	current, readErr := local.GetThread(record.Address)
	if readErr != nil || current.Revision != record.Revision || len(env.Runner.Ops) != 0 {
		t.Fatalf("open changed state: %+v %v %v", current, readErr, env.Runner.Ops)
	}
}
