package couchcore

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

func TestDispatchOperationRoutesOnlyThroughDeclaredExecutor(t *testing.T) {
	var direct, live []OperationCall
	executors := OperationExecutors{
		DirectStore: func(call OperationCall) (any, error) {
			direct = append(direct, call)
			return "stored", nil
		},
		LiveOwner: func(call OperationCall) (any, error) {
			live = append(live, call)
			return "owned", nil
		},
	}

	got, err := DispatchOperation(executors, OperationCall{Name: "list"})
	if err != nil || got != "stored" {
		t.Fatalf("list dispatch = %#v, %v", got, err)
	}
	got, err = DispatchOperation(executors, OperationCall{Name: "start", Args: map[string]string{"path": "/repo", "fingerprint": "grant"}, Implicit: true})
	if err != nil || got != "owned" {
		t.Fatalf("start dispatch = %#v, %v", got, err)
	}
	if len(direct) != 1 || direct[0].Operation.Name != "list" {
		t.Fatalf("direct calls = %+v", direct)
	}
	if len(live) != 1 || live[0].Operation.Name != "start" {
		t.Fatalf("live calls = %+v", live)
	}
}

func TestLifecycleOperationContextReachesParkLeaveAndResume(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	t.Run("park", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		record, _ := env.spawn(t, StartArgs{Cwd: "/repo"})
		installLifecycleForContextTest(t, env, record.Thread)
		_, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch)}, OperationCall{
			Name: "park", Context: canceled, Implicit: true,
			Args: map[string]string{"repo-scope": record.Thread.RepoScope, "tag": string(record.Thread.Tag)},
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("park context = %v, want canceled", err)
		}
	})
	for _, mode := range []string{"retry", "recover", "abandon"} {
		t.Run("park/"+mode, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			record, _ := env.spawn(t, StartArgs{Cwd: "/repo"})
			installLifecycleForContextTest(t, env, record.Thread)
			thread, err := env.Couch.Threads.GetThread(record.Thread)
			if err != nil {
				t.Fatal(err)
			}
			identity := ParkIdentity{Nonce: "park-context", Address: thread.Address, PID: thread.Incarnations[0].PID, ProcessIdentity: thread.Incarnations[0].Identity}
			if _, err := env.Couch.Threads.BeginPark(thread.Address, thread.Revision, identity); err != nil {
				t.Fatal(err)
			}
			_, err = DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch)}, OperationCall{
				Name: "park", Context: canceled, Implicit: true,
				Args: map[string]string{"repo-scope": record.Thread.RepoScope, "tag": string(record.Thread.Tag), "mode": mode},
			})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("park %s context = %v, want canceled", mode, err)
			}
		})
	}

	t.Run("leave", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		record, _ := env.spawn(t, StartArgs{Cwd: "/repo"})
		installLifecycleForContextTest(t, env, record.Thread)
		_, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch)}, OperationCall{
			Name: "leave", Context: canceled,
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("leave context = %v, want canceled", err)
		}
	})

	t.Run("resume", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		env.Couch.resumeRegistrationTimeout = time.Millisecond
		parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
		env.Artifacts.SetNativeBinding(parked.Address, "claude", "established", "native-root-1")
		_, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch)}, OperationCall{
			Name: "resume", Context: canceled, Implicit: true,
			Args: map[string]string{"repo-scope": parked.Address.RepoScope, "tag": string(parked.Address.Tag)},
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("resume context = %v, want canceled", err)
		}
	})
}

func installLifecycleForContextTest(t *testing.T, env *testEnv, address ThreadAddress) {
	t.Helper()
	model := pairlifecycletest.New(env.Now)
	model.SetSession("pair-context", true)
	env.Artifacts.SetPairSession(address, "pair-context", true)
	env.Couch.PairLifecycle = &PairLifecycleController{
		Threads: env.Couch.Threads, DataDir: env.Dir,
		Lifecycle: &fakeControllerLifecycle{model: model, store: env.Couch.Threads},
		Sessions:  env.Artifacts, Proc: env.Proc, Clock: FixedClock{T: env.Now},
		Nonce:             func() (string, error) { return "park-context", nil },
		CompletionTimeout: time.Millisecond, PollInterval: time.Millisecond,
	}
}

func TestDispatchOperationMissingOwnerRefusesWithoutFallback(t *testing.T) {
	directCalls := 0
	_, err := DispatchOperation(OperationExecutors{
		DirectStore: func(OperationCall) (any, error) {
			directCalls++
			return nil, nil
		},
	}, OperationCall{Name: "stop", Args: map[string]string{"ref": "thread"}})
	var routing *OwnerRoutingRequiredError
	if !errors.As(err, &routing) {
		t.Fatalf("error = %T %v, want OwnerRoutingRequiredError", err, err)
	}
	if directCalls != 0 {
		t.Fatalf("owner absence fell back to direct executor %d time(s)", directCalls)
	}
}

func TestDispatchOperationValidatesSchemaAndPreservesTypedPayload(t *testing.T) {
	payload := struct{ Token string }{Token: "terminal"}
	var received OperationCall
	executors := OperationExecutors{LiveOwner: func(call OperationCall) (any, error) {
		received = call
		return nil, nil
	}}
	call := OperationCall{
		Name:         "attach",
		Args:         map[string]string{"repo-scope": "scope", "tag": "couch-0102030405060708"},
		Implicit:     true,
		TypedPayload: payload,
	}
	if _, err := DispatchOperation(executors, call); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(received.TypedPayload, payload) {
		t.Fatalf("payload = %#v, want %#v", received.TypedPayload, payload)
	}

	for _, tc := range []OperationCall{
		{Name: "missing"},
		{Name: "show"},
		{Name: "show", Args: map[string]string{"ref": "thread"}, Implicit: true},
		{Name: "list", Args: map[string]string{"surprise": "x"}},
		{Name: "attach", Args: call.Args},
	} {
		if _, err := DispatchOperation(executors, tc); err == nil {
			t.Errorf("DispatchOperation(%+v) succeeded", tc)
		}
	}
}

func TestDispatchOperationRejectsEmptyValueRequiredArgumentBeforeExecutor(t *testing.T) {
	calls := 0
	_, err := DispatchOperation(OperationExecutors{
		LiveOwner: func(OperationCall) (any, error) {
			calls++
			return nil, nil
		},
	}, OperationCall{Name: "start", Args: map[string]string{"path": "/repo", "agent": ""}})
	if err == nil {
		t.Fatal("empty value-bearing agent argument accepted")
	}
	if calls != 0 {
		t.Fatalf("invalid operation reached executor %d time(s)", calls)
	}
}

// prepare-start resolves and starts nothing; start commits the resolution the
// preview accepted, identified by its fingerprint.
//
// This test used to assert that a REPLAY was refused with
// ErrStartGrantUnavailable -- the grant table's at-most-one-consumption. That
// table is gone (pair#170 M4), and its absence is deliberate rather than
// overlooked: a fingerprint says "this resolution still holds", not "this
// resolution has not been used", and couch-lite has one owner, so there is no
// second party to race for a claim. At-most-once is a property of the SUBMIT,
// and it lives where the submit is -- couchtty's armed start form, pinned by
// TestStartFormArmedSubmitDispatchesOnce.
//
// What the fingerprint must still do is refuse a resolution that drifted, and
// that is TestSpawnPreparedRefusesDriftByFingerprint.
func TestPreparedStartResolvesThenCommitsByFingerprint(t *testing.T) {
	env := newTestEnv(t, "/repo")
	executor := CouchLiveOwnerExecutor(env.Couch)
	preparedValue, err := DispatchOperation(OperationExecutors{LiveOwner: executor}, OperationCall{
		Name: "prepare-start", Args: map[string]string{"path": "/repo", "agent": "codex"},
	})
	prepared, ok := preparedValue.(PreparedStart)
	if err != nil || !ok || prepared.Resolution.Fingerprint == "" || len(env.Runner.Ops) != 0 {
		t.Fatalf("prepare operation = %#v, %v, runner=%q", preparedValue, err, env.Runner.Ops)
	}
	startArgs := prepared.Resolution.CommitArgs()
	startedValue, err := DispatchOperation(OperationExecutors{LiveOwner: executor}, OperationCall{
		Name: "start", Args: startArgs, Implicit: true,
	})
	if _, ok := startedValue.(StartResult); err != nil || !ok {
		t.Fatalf("start operation = %#v, %v", startedValue, err)
	}
	if _, err := DispatchOperation(OperationExecutors{LiveOwner: executor}, OperationCall{
		Name: "start", Args: map[string]string{
			"path": "/repo", "agent": "codex", "fingerprint": "stale-fingerprint",
		}, Implicit: true,
	}); !errors.Is(err, ErrStartResolutionChanged) {
		t.Fatalf("stale fingerprint err = %v, want ErrStartResolutionChanged", err)
	}
}

func TestResumeOperationResolutionBoundary(t *testing.T) {
	env := newTestEnv(t, "/repo")
	first := validThreadRecord(t)
	first.Name = "shared"
	first.WorkingPath = "/repo/shared-one"
	first.StartingPath = "/repo"
	created, err := env.Couch.Threads.CreateThread(first)
	if err != nil {
		t.Fatal(err)
	}
	second := cloneThreadRecord(first)
	second.Address.Tag = "couch-fedcba9876543210"
	second.WorkingPath = "/repo/shared-two"
	second.Revision = 1
	if _, err := env.Couch.Threads.CreateThread(second); err != nil {
		t.Fatal(err)
	}

	exact, err := resolveOperationThread(env.Couch, map[string]string{
		"repo-scope": created.Address.RepoScope, "tag": string(created.Address.Tag),
	})
	if err != nil || exact != created.Address {
		t.Fatalf("exact resolution = %+v, %v", exact, err)
	}
	if _, err := resolveOperationThread(env.Couch, map[string]string{
		"repo-scope": created.Address.RepoScope, "ref": "shared",
	}); err == nil {
		t.Fatal("ambiguous resume reference reached execution")
	}

	calls := 0
	_, err = DispatchOperation(OperationExecutors{LiveOwner: func(OperationCall) (any, error) {
		calls++
		return nil, nil
	}}, OperationCall{Name: "resume", Args: map[string]string{
		"repo-scope": created.Address.RepoScope, "tag": string(created.Address.Tag),
	}})
	if err == nil || calls != 0 {
		t.Fatalf("untrusted exact address = calls %d, err %v", calls, err)
	}
}
