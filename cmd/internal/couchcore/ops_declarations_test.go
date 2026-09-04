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
		presentation OperationPresentation
	}{
		"prepare-start":       {ExecuteLiveOwner, EffectAuthority, ConfirmNone, ResultStartResolution, PresentationTUI},
		"start":               {ExecuteLiveOwner, EffectProcess, ConfirmNone, ResultStart, PresentationTUI},
		"list":                {ExecuteDirectStore, EffectRead, ConfirmNone, ResultThreadInventory, PresentationList},
		"show":                {ExecuteDirectStore, EffectRead, ConfirmNone, ResultThreadInventory, PresentationShow},
		"stop":                {ExecuteLiveOwner, EffectProcess, ConfirmRequired, ResultStop, PresentationTUI},
		"name":                {ExecuteDirectStore, EffectMetadata, ConfirmNone, ResultThread, PresentationTUI},
		"describe":            {ExecuteDirectStore, EffectMetadata, ConfirmNone, ResultDescription, PresentationTUI},
		"publish-description": {ExecuteDirectStore, EffectMetadata, ConfirmNone, ResultThread, PresentationInternal},
		"switch":              {ExecuteLiveOwner, EffectConsole, ConfirmNone, ResultConsole, PresentationTUI},
		"attach":              {ExecuteLiveOwner, EffectConsole, ConfirmNone, ResultConsole, PresentationTUI},
		"park":                {ExecuteLiveOwner, EffectProcess, ConfirmRequired, ResultThread, PresentationTUI},
		"detach":              {ExecuteLiveOwner, EffectProcess, ConfirmNone, ResultThread, PresentationTUI},
		"leave":               {ExecuteLiveOwner, EffectProcess, ConfirmRequired, ResultConsole, PresentationTUI},
		"relaunch":            {ExecuteLiveOwner, EffectProcess, ConfirmRequired, ResultStart, PresentationTUI},
		"archive":             {ExecuteDirectStore, EffectMetadata, ConfirmRequired, ResultThread, PresentationTUI},
		"archived":            {ExecuteDirectStore, EffectRead, ConfirmNone, ResultThreadInventory, PresentationList},
		"resume":              {ExecuteLiveOwner, EffectProcess, ConfirmNone, ResultStart, PresentationTUI},
	}
	for _, op := range Operations() {
		expected, ok := want[op.Name]
		if !ok {
			t.Errorf("unexpected operation %q", op.Name)
			continue
		}
		delete(want, op.Name)
		if op.Execution == ExecuteUnknown || op.Presentation == PresentationUnknown || op.Execution != expected.execution || op.Effect != expected.effect || op.Confirmation != expected.confirmation || op.Result != expected.result || op.Presentation != expected.presentation {
			t.Errorf("%s declaration = execution %v effect %v confirmation %v result %v presentation %v; want %+v",
				op.Name, op.Execution, op.Effect, op.Confirmation, op.Result, op.Presentation, expected)
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

// This test used to assert that Couch exposes NO detach operation -- Alt+d was
// Pair-local. #170 reverses that: un-intercepted, Alt+d leaves Couch with a dead
// child and a stale live incarnation, which the fail-closed projector hides, so
// the operator's most common gesture would make the thread vanish. It is
// inverted rather than deleted, because the reversal is worth recording where
// the superseded claim lived.
func TestParkDetachLeaveAndResumeSurface(t *testing.T) {
	var park, resume, leave, detach *Operation
	for _, operation := range Operations() {
		op := operation
		switch op.Name {
		case "park":
			park = &op
		case "resume":
			resume = &op
		case "leave":
			leave = &op
		case "detach":
			detach = &op
		}
	}
	if park == nil || leave == nil || resume == nil || detach == nil {
		t.Fatalf("park/detach/leave/resume declarations = %+v, %+v, %+v, %+v", park, detach, leave, resume)
	}
	// Detach destroys nothing, so it must not demand a confirmation the way
	// park does -- that asymmetry is the whole reason both exist.
	if detach.Confirmation != ConfirmNone {
		t.Fatalf("detach confirmation = %v, want none", detach.Confirmation)
	}
	if park.Confirmation != ConfirmRequired {
		t.Fatalf("park confirmation = %v, want required", park.Confirmation)
	}
	wantParkArgs := []ArgSpec{
		{Name: "ref", Summary: "thread tag, path, or name", Required: false},
		{Name: "tag", Summary: "exact thread tag from trusted owner context", Implicit: true},
		{Name: "mode", Summary: "normal, retry, recover, or abandon (--mode=<mode>)", FlagOnly: true, ValueRequired: true},
		{Name: "repo-scope", Summary: "repository scope derived from caller context", Required: true, Implicit: true},
	}
	if !reflect.DeepEqual(park.Args, wantParkArgs) {
		t.Fatalf("park args = %+v", park.Args)
	}
}
