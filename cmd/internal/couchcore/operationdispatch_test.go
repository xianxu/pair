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
		{Name: "list", Args: map[string]string{"surprise": "x"}},
		{Name: "attach", Args: call.Args},
	} {
		if _, err := DispatchOperation(executors, tc); err == nil {
			t.Errorf("DispatchOperation(%+v) succeeded", tc)
		}
	}
}

func TestOperationDeclarationsAreClosureFreeAndComplete(t *testing.T) {
	if _, found := reflect.TypeOf(Operation{}).FieldByName("Invoke"); found {
		t.Fatal("Operation still embeds an execution closure")
	}
	want := map[string]struct {
		execution    OperationExecution
		effect       OperationEffect
		confirmation OperationConfirmation
		result       OperationResult
	}{
		"start":               {ExecuteLiveOwner, EffectProcess, ConfirmNone, ResultStart},
		"list":                {ExecuteDirectStore, EffectRead, ConfirmNone, ResultThreadInventory},
		"show":                {ExecuteDirectStore, EffectRead, ConfirmNone, ResultThreadInventory},
		"stop":                {ExecuteLiveOwner, EffectProcess, ConfirmRequired, ResultStop},
		"name":                {ExecuteDirectStore, EffectMetadata, ConfirmNone, ResultThread},
		"describe":            {ExecuteDirectStore, EffectMetadata, ConfirmNone, ResultDescription},
		"publish-description": {ExecuteDirectStore, EffectMetadata, ConfirmNone, ResultThread},
		"switch":              {ExecuteLiveOwner, EffectConsole, ConfirmNone, ResultConsole},
		"attach":              {ExecuteLiveOwner, EffectConsole, ConfirmNone, ResultConsole},
	}
	for _, op := range Operations() {
		expected, ok := want[op.Name]
		if !ok {
			t.Errorf("unexpected operation %q", op.Name)
			continue
		}
		delete(want, op.Name)
		if op.Execution != expected.execution || op.Effect != expected.effect || op.Confirmation != expected.confirmation || op.Result != expected.result {
			t.Errorf("%s declaration = execution %v effect %v confirmation %v result %v; want %+v",
				op.Name, op.Execution, op.Effect, op.Confirmation, op.Result, expected)
		}
	}
	if len(want) > 0 {
		t.Fatalf("missing declarations: %v", want)
	}
}
