package couchcore

import (
	"reflect"
	"testing"
)

func TestOperationDeclarationsAreClosureFreeCompleteAndOwned(t *testing.T) {
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
		if op.Execution == ExecuteUnknown || op.Execution != expected.execution || op.Effect != expected.effect || op.Confirmation != expected.confirmation || op.Result != expected.result {
			t.Errorf("%s declaration = execution %v effect %v confirmation %v result %v; want %+v",
				op.Name, op.Execution, op.Effect, op.Confirmation, op.Result, expected)
		}
		for _, arg := range op.Args {
			if arg.ValueRequired && !arg.FlagOnly {
				t.Errorf("%s argument %q requires a named value but is not flag-only", op.Name, arg.Name)
			}
		}
	}
	if len(want) > 0 {
		t.Fatalf("missing declarations: %v", want)
	}
}
