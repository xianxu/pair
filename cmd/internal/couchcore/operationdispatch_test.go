package couchcore

import (
	"errors"
	"reflect"
	"testing"
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
	got, err = DispatchOperation(executors, OperationCall{Name: "start", Args: map[string]string{"token": "grant"}, Implicit: true})
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

func TestPreparedStartOperationsIssueThenConsumeImplicitToken(t *testing.T) {
	env := newTestEnv(t, "/repo")
	executor := CouchLiveOwnerExecutor(env.Couch)
	preparedValue, err := DispatchOperation(OperationExecutors{LiveOwner: executor}, OperationCall{
		Name: "prepare-start", Args: map[string]string{"path": "/repo", "agent": "codex"},
	})
	prepared, ok := preparedValue.(PreparedStart)
	if err != nil || !ok || prepared.Token == "" || len(env.Runner.Ops) != 0 {
		t.Fatalf("prepare operation = %#v, %v, runner=%q", preparedValue, err, env.Runner.Ops)
	}
	startedValue, err := DispatchOperation(OperationExecutors{LiveOwner: executor}, OperationCall{
		Name: "start", Args: map[string]string{"token": string(prepared.Token)}, Implicit: true,
	})
	if _, ok := startedValue.(StartResult); err != nil || !ok {
		t.Fatalf("start operation = %#v, %v", startedValue, err)
	}
	if _, err := DispatchOperation(OperationExecutors{LiveOwner: executor}, OperationCall{
		Name: "start", Args: map[string]string{"token": string(prepared.Token)}, Implicit: true,
	}); !errors.Is(err, ErrStartGrantUnavailable) {
		t.Fatalf("token replay err = %v", err)
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
