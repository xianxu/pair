package couchcore

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
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
	var readyAddress ThreadAddress
	var readyAgent, readyNonce string
	env.Runner.AfterAcknowledge = func(id string) error {
		child := env.Runner.Child(id)
		var profile launcher.TrustedLaunchProfile
		if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
			return err
		}
		address := ThreadAddress{RepoScope: environmentValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: ThreadTag(profile.Tag)}
		current, err := local.GetThread(address)
		if err != nil {
			return err
		}
		if !profile.FreshRequired || profile.ResumeRequired || profile.RequiredSessionID != "" || profile.FreshNonce == "" || profile.Prompt == "" || address == record.Address || current.Incarnations[0].Start == nil || current.Incarnations[0].Start.Nonce != profile.FreshNonce {
			return fmt.Errorf("Fresh profile does not match the new claim: %+v", profile)
		}
		readyAddress, readyAgent, readyNonce = address, profile.Agent, profile.FreshNonce
		env.Artifacts.SetPairSession(address, continuationChildSession(t, env.Runner, id), true)
		env.Artifacts.SetSessionPresence(address, SessionObservation{State: SessionPresent})
		return nil
	}
	env.Couch.FreshRegistration = func(_ context.Context, address ThreadAddress, agent, nonce string) (bool, error) {
		return nonce != "" && nonce == readyNonce && address == readyAddress && agent == readyAgent, nil
	}
	return env, local, record
}

func dispatchSlotContinuation(env *testEnv, op, path string) (any, error) {
	return DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch), DirectStore: DirectStoreExecutor(env.Couch)}, OperationCall{Name: op, Args: map[string]string{"path": path, "agent": "claude"}, Implicit: true, Context: context.Background()})
}

// Explicit Fresh preserves the retained continuation while starting a new tag.
// A process without exact retirement authority remains untouched.
func TestSlotRebootContinuationLeavesUnattributableOwnersUntouched(t *testing.T) {
	t.Parallel()

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
				// Reboot retires through archive's admission, which reads the
				// record's exact session binding (absent session, known name).
				env.Artifacts.SetPairSession(record.Address, "pair-slot-fixture", false)
				pid, identity := 42, "source"
				if target {
					pid, identity = 43, "target"
				}
				if unknown {
					env.Proc.SetUnknown(pid)
				} else {
					env.Proc.Set(pid, identity)
				}
				result, err := dispatchSlotContinuation(env, "reboot", record.StartingPath)
				if err != nil {
					t.Fatal(err)
				}
				started, ok := result.(StartedChild).Started()
				if !ok || started.Handle == nil || started.Record.Thread == record.Address {
					t.Fatalf("not a fresh tag: %+v", result)
				}
				retained, readErr := local.ArchivedThreads()
				if readErr != nil || len(retained) != 1 || !reflect.DeepEqual(retained[0].Continuation, record.Continuation) {
					t.Fatalf("continuation evidence changed: %+v %v", retained, readErr)
				}
				gotIdentity, identityErr := env.Proc.Identity(pid)
				if (!unknown && (identityErr != nil || gotIdentity != identity)) || (unknown && identityErr == nil) || len(env.Proc.Signals[pid]) != 0 || len(env.Proc.GroupSignals[pid]) != 0 || len(env.Artifacts.Quiesces()) != 0 {
					t.Fatalf("unattributable owner changed: identity=%q err=%v signals=%v groups=%v quiesces=%v", gotIdentity, identityErr, env.Proc.Signals, env.Proc.GroupSignals, env.Artifacts.Quiesces())
				}
			})
		}
	}
}

func TestSlotRebootRetainsStoppedContinuationInTheArchive(t *testing.T) {
	t.Parallel()

	// A pending request is the only stopped one a record holds: a request
	// that didn't finish is gone, leaving its line (#522).
	for _, phase := range checkpoint.AllPhases() {
		if phase != checkpoint.Pending {
			continue
		}
		t.Run(string(phase), func(t *testing.T) {
			env, local, record := slotContinuationFixture(t, phase, false)
			env.Artifacts.SetPairSession(record.Address, "pair-slot-fixture", false)
			result, err := dispatchSlotContinuation(env, "reboot", record.StartingPath)
			if err != nil {
				t.Fatal(err)
			}
			start, started := result.(StartedChild).Started()
			if !started || start.Handle == nil || start.Record.Thread == record.Address {
				t.Fatalf("not fresh: %+v", start)
			}
			retained, err := local.ArchivedThreads()
			if err != nil || len(retained) != 1 || !reflect.DeepEqual(retained[0].Continuation, record.Continuation) {
				t.Fatalf("continuation evidence lost: %+v %v", retained, err)
			}
		})
	}
}

func TestSlotOpenColdUsesContinuationGuard(t *testing.T) {
	t.Parallel()

	env, local, record := slotContinuationFixture(t, checkpoint.Pending, false)
	// OpenSlot directly: resume routes a retained request to RecoverThread or
	// Continue before it reaches OpenSlot, so the guard is OpenSlot's own defence.
	_, err := env.Couch.OpenSlot(context.Background(), record.StartingPath, "claude")
	if err == nil || !strings.Contains(err.Error(), "continuation "+record.Continuation.ID+" is pending;") {
		t.Fatalf("cold slot open bypassed continuation guard: %v", err)
	}
	current, readErr := local.GetThread(record.Address)
	if readErr != nil || current.Revision != record.Revision || len(env.Runner.Ops) != 0 {
		t.Fatalf("open changed state: %+v %v %v", current, readErr, env.Runner.Ops)
	}
}
