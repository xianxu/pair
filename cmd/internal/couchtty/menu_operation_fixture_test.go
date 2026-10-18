package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"reflect"
	"testing"
)

// operationForTest projects the latest running operation for older single-job
// fixtures. Production callers must resolve their exact target or attempt.
func (state MenuState) operationForTest() MenuOperationOrigin {
	var latest MenuOperationOrigin
	found := false
	for _, job := range state.Operations {
		if job.Phase == MenuOperationRunning && (!found || job.Origin.Attempt > latest.Attempt) {
			latest = job.Origin
			found = true
		}
	}
	return latest
}

// setTestMenuOperation replaces only the old single-job fixture state; every
// other menu field and the supplied origin, including attempt zero, is retained.
func setTestMenuOperation(state *MenuState, origin MenuOperationOrigin) {
	if origin == (MenuOperationOrigin{}) {
		state.Operations = nil
		return
	}
	target := menuOperationTarget{RowKey: origin.RowKey}
	if origin.Operation == "leave" {
		target = menuOperationTarget{Couch: true}
	} else if target.RowKey == (couchcore.ThreadRowKey{}) {
		target.RowKey = couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetOrdinary, Address: origin.Address}
	}
	state.Operations = map[menuOperationTarget]MenuOperation{target: {Origin: origin, Phase: MenuOperationRunning}}
}

func TestLegacyMenuOperationFixtureKeepsExactOriginAndUnrelatedState(t *testing.T) {
	for _, origin := range []MenuOperationOrigin{
		{Operation: "park", Address: menuAddress("ordinary")},
		{Operation: "recover", Attempt: 7, RowKey: couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetSlot, SlotPath: "/repo"}},
		{Operation: "leave", Attempt: 9},
	} {
		state := MenuState{OperationSequence: 17, Notice: MenuNotice{Text: "kept"}}
		original := state
		setTestMenuOperation(&state, origin)
		if len(state.Operations) != 1 || state.operationForTest() != origin {
			t.Fatalf("fixture lost exact origin %+v: %+v", origin, state.Operations)
		}
		for key := range state.Operations {
			if origin.Operation == "leave" && !key.Couch {
				t.Fatal("leave did not own Couch")
			}
			if origin.RowKey != (couchcore.ThreadRowKey{}) && origin.Operation != "leave" && key.RowKey != origin.RowKey {
				t.Fatal("fixture lost exact row")
			}
			if origin.RowKey == (couchcore.ThreadRowKey{}) && origin.Operation != "leave" && key.RowKey != (couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetOrdinary, Address: origin.Address}) {
				t.Fatal("fixture lost ordinary target")
			}
		}
		setTestMenuOperation(&state, MenuOperationOrigin{})
		if !reflect.DeepEqual(state, original) {
			t.Fatalf("fixture changed unrelated menu state: %+v", state)
		}
	}
}

func TestLegacyMenuOperationFixtureProjectsLatestRunningOnly(t *testing.T) {
	state := MenuState{Operations: map[menuOperationTarget]MenuOperation{}}
	for _, entry := range []struct {
		tag     string
		attempt uint64
		phase   MenuOperationPhase
	}{
		{"zero", 0, MenuOperationRunning}, {"latest", 8, MenuOperationRunning}, {"older", 4, MenuOperationRunning}, {"finished", 10, MenuOperationSucceeded},
	} {
		key := menuOperationTarget{RowKey: couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetOrdinary, Address: menuAddress(entry.tag)}}
		state.Operations[key] = MenuOperation{Origin: MenuOperationOrigin{Operation: entry.tag, Attempt: entry.attempt}, Phase: entry.phase}
	}
	if got := state.operationForTest(); got.Operation != "latest" || got.Attempt != 8 {
		t.Fatalf("latest running=%+v", got)
	}
	for key, job := range state.Operations {
		job.Phase = MenuOperationSucceeded
		state.Operations[key] = job
	}
	if got := state.operationForTest(); got != (MenuOperationOrigin{}) {
		t.Fatalf("finished operation projected as running: %+v", got)
	}
}
