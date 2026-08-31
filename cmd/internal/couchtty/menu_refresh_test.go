package couchtty

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestAdvanceRefreshScheduleKeepsOneRunningAndOneDirtyFollowup(t *testing.T) {
	var state RefreshSchedule

	state, effects := AdvanceRefreshSchedule(state, RefreshScheduleEvent{Kind: RefreshRequested})
	if !reflect.DeepEqual(effects, []RefreshScheduleEffect{{Kind: RefreshStart, Generation: 1}}) || state.Running != 1 || state.Dirty {
		t.Fatalf("first request = state %+v effects %+v", state, effects)
	}

	state, effects = AdvanceRefreshSchedule(state, RefreshScheduleEvent{Kind: RefreshRequested})
	if len(effects) != 0 || state.Running != 1 || !state.Dirty {
		t.Fatalf("second request did not coalesce = state %+v effects %+v", state, effects)
	}
	state, effects = AdvanceRefreshSchedule(state, RefreshScheduleEvent{Kind: RefreshRequested})
	if len(effects) != 0 || state.Running != 1 || !state.Dirty {
		t.Fatalf("third request exceeded one dirty bit = state %+v effects %+v", state, effects)
	}

	before := state
	state, effects = AdvanceRefreshSchedule(state, RefreshScheduleEvent{Kind: RefreshFinished, Generation: 99})
	if len(effects) != 0 || !reflect.DeepEqual(state, before) {
		t.Fatalf("stale completion changed schedule = state %+v effects %+v", state, effects)
	}

	state, effects = AdvanceRefreshSchedule(state, RefreshScheduleEvent{Kind: RefreshFinished, Generation: 1})
	if !reflect.DeepEqual(effects, []RefreshScheduleEffect{{Kind: RefreshStart, Generation: 2}}) || state.Running != 2 || state.Dirty {
		t.Fatalf("matching completion did not admit one followup = state %+v effects %+v", state, effects)
	}
	state, effects = AdvanceRefreshSchedule(state, RefreshScheduleEvent{Kind: RefreshFinished, Generation: 1})
	if len(effects) != 0 || state.Running != 2 || state.Dirty {
		t.Fatalf("duplicate completion retired followup = state %+v effects %+v", state, effects)
	}
	state, effects = AdvanceRefreshSchedule(state, RefreshScheduleEvent{Kind: RefreshFinished, Generation: 2})
	if len(effects) != 0 || state.Running != 0 || state.Dirty {
		t.Fatalf("final completion did not idle schedule = state %+v effects %+v", state, effects)
	}
}

func TestAdvanceRefreshScheduleRefusesGenerationExhaustion(t *testing.T) {
	state := RefreshSchedule{Sequence: ^uint64(0)}
	got, effects := AdvanceRefreshSchedule(state, RefreshScheduleEvent{Kind: RefreshRequested})
	if len(effects) != 0 || !reflect.DeepEqual(got, state) {
		t.Fatalf("exhausted schedule authorized refresh = state %+v effects %+v", got, effects)
	}
}

func TestReduceMenuInventoryFailurePreservesLastGoodAndInitialUnavailable(t *testing.T) {
	active := menuAddress("couch-one")
	lastGood := menuThreads()
	state := NewMenuState(lastGood, active)
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventRefreshStarted})
	got, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Error: "corrupt manifest"})
	if len(effects) != 0 || !reflect.DeepEqual(got.Inventory, lastGood) || !got.InventoryReady || got.RefreshPending || got.Notice.Text != "thread inventory unavailable: corrupt manifest" {
		t.Fatalf("failed refresh replaced last-good state = state %+v effects %+v", got, effects)
	}

	initial := NewMenuState(nil, couchcore.ThreadAddress{})
	initial, _ = ReduceMenu(initial, MenuEvent{Kind: MenuEventRefreshStarted})
	if initial.InventoryReady || !initial.RefreshPending || initial.Notice.Text != "thread inventory unavailable" {
		t.Fatalf("initial refresh did not distinguish unavailable from empty = %+v", initial)
	}
	initial, _ = ReduceMenu(initial, MenuEvent{Kind: MenuEventInventory, Inventory: []couchcore.ActionableThreadSummary{}})
	if !initial.InventoryReady || initial.RefreshPending || len(initial.Inventory) != 0 {
		t.Fatalf("successful empty inventory remained unavailable = %+v", initial)
	}
}

func TestSuccessfulOperationsDeclareProjectionRefreshPolicy(t *testing.T) {
	for _, operation := range []string{"start", "park", "resume", "name", "describe", "switch", "leave"} {
		t.Run(operation, func(t *testing.T) {
			state := NewMenuState(menuThreads(), menuAddress("couch-one"))
			state.InFlight = MenuOperationOrigin{
				Operation: operation, Attempt: 1, Address: menuAddress("couch-one"),
				FrameInstance: state.Frames[0].Instance, FrameKind: MenuFrameRoot, Depth: 1,
			}
			got, _ := ReduceMenu(state, MenuEvent{
				Kind: MenuEventOperationResult, Operation: operation, Attempt: 1,
				Address: menuAddress("couch-one"), Success: true,
			})
			wantPending := operation == "start" || operation == "park" || operation == "resume" || operation == "name" || operation == "describe"
			if got.ProjectionPending != wantPending {
				t.Fatalf("ProjectionPending = %v, want %v", got.ProjectionPending, wantPending)
			}
		})
	}
}

func TestProjectionPendingSurvivesRefreshFailureAndClearsOnSuccess(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.ProjectionPending = true

	failed, _ := ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Error: "store unavailable"})
	if !failed.ProjectionPending || !strings.Contains(RenderMenu(failed, 80, 24, time.Time{}, false), "refresh pending") {
		t.Fatalf("failed refresh hid stale projection: %+v", failed)
	}

	fresh, _ := ReduceMenu(failed, MenuEvent{Kind: MenuEventInventory, Inventory: menuThreads()})
	if fresh.ProjectionPending {
		t.Fatalf("successful refresh retained pending projection: %+v", fresh)
	}
}
