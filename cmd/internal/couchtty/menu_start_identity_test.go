package couchtty

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestMenuStartReservationSurvivesSlotMaterialization(t *testing.T) {
	for _, kind := range []couchcore.ThreadTargetKind{couchcore.ThreadTargetSlot, couchcore.ThreadTargetOrdinary} {
		t.Run(string(kind), func(t *testing.T) { testMenuStartMaterialization(t, kind) })
	}
}

func testMenuStartMaterialization(t *testing.T, kind couchcore.ThreadTargetKind) {
	t.Helper()
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Frames[0].PreviewResolution.Worktree = "/repo/new"
	state, effects := dispatchMenuOperation(state, MenuEffect{Operation: "start"}, couchcore.ThreadAddress{})
	if len(effects) != 1 {
		t.Fatal("start was not admitted")
	}
	_, original, _ := menuOperationForAttempt(state, effects[0].Attempt)
	row := menuThreads()[0]
	row.Address = menuAddress("created")
	row.RowKey = couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetSlot, SlotPath: "/repo/new"}
	if kind == couchcore.ThreadTargetOrdinary {
		scope, err := launcher.ResolveRepoScope("/repo/new")
		if err != nil {
			t.Fatal(err)
		}
		row.Address.RepoScope = scope.Key
		row.RowKey = couchcore.ThreadRowKey{Kind: kind, Address: row.Address}
	}
	state.Inventory = append(state.Inventory, row)
	state = pruneMenuOperations(state)

	t.Run("row progress", func(t *testing.T) {
		job, ok := menuOperationForRow(state, row)
		if !ok || job.Phase != MenuOperationRunning || job.Origin != original.Origin {
			t.Fatalf("materialized row lost its running start: job=%+v found=%v", job, ok)
		}
		if job, ok := menuOperationRows(state)[row.RowKey]; !ok || job.Origin != original.Origin {
			t.Fatalf("render projection lost the start: %+v found=%v", job, ok)
		}
	})
	t.Run("second mutation refused", func(t *testing.T) {
		next, second := dispatchThreadOperation(state, "detach", row.Address)
		if len(second) != 0 || !strings.Contains(next.Notice.Text, "busy: start") {
			t.Fatalf("materialization admitted another action: effects=%+v notice=%+v", second, next.Notice)
		}
	})
	t.Run("busy selection waits on original attempt", func(t *testing.T) {
		next, busy := selectBusyMenuRow(state, row)
		if !busy || next.LandingAttempt != original.Origin.Attempt {
			t.Fatalf("busy selection lost original attempt: busy=%v landing=%d", busy, next.LandingAttempt)
		}
	})
	t.Run("automatic address lookup", func(t *testing.T) {
		job, ok := menuRunningForAddress(state, row.Address)
		if !ok || !reflect.DeepEqual(job.Origin, original.Origin) {
			t.Fatalf("address lookup lost start owner: job=%+v found=%v", job, ok)
		}
	})
}
