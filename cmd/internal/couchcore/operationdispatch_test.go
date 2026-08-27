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
	got, err = DispatchOperation(executors, OperationCall{Name: "start", Args: map[string]string{"path": "/repo"}})
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
