package couchtty

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestContinuePassSeedsOnlyPositivelyListedOutcomes(t *testing.T) {
	inventory := []couchcore.ActionableThreadSummary{
		reattachRow("couch-root", couchcore.ThreadLive, 0),
		reattachRow("listed-live", couchcore.ThreadLive, 1),
		reattachRow("listed-detached", couchcore.ThreadDetached, 2),
		reattachRow("unknown", couchcore.ThreadUnusable, 3),
		reattachRow("busy", couchcore.ThreadBusy, 4),
		reattachRow("gone", couchcore.ThreadUnusable, 5),
		reattachRow("parked", couchcore.ThreadParked, 6),
	}
	inventory[3].Reason = couchcore.ReasonUnknown
	inventory[5].Reason = couchcore.ReasonSessionGone
	state := seedReattach(armedMenu("couch-root"), inventory)
	want := []couchcore.ThreadAddress{menuAddress("listed-live"), menuAddress("listed-detached")}
	if !reflect.DeepEqual(state.Reattach.Queue, want) {
		t.Fatalf("Continue queue=%v want%v", state.Reattach.Queue, want)
	}
}

func TestContinuePassFailuresKeepContinueAndOfferOneClickRetry(t *testing.T) {
	for _, test := range []struct {
		name       string
		busy       bool
		diagnostic couchcore.ResumeDiagnosticCode
		failure    string
	}{
		{name: "busy", busy: true, failure: "selected thread is busy"},
		{name: "not detached", diagnostic: couchcore.ResumeNotDetached},
		{name: "session gone", diagnostic: couchcore.ResumeSessionGone},
		{name: "helper failed", failure: "attachment failed after 2 attempts: temporary failure"},
	} {
		for _, rowState := range []couchcore.ActionableThreadState{couchcore.ThreadLive, couchcore.ThreadDetached} {
			t.Run(test.name+string(rowState), func(t *testing.T) {
				row := reattachRow("worker", rowState, 0)
				state := armReattach(NewMenuState([]couchcore.ActionableThreadSummary{row}, menuAddress("couch-root")), menuAddress("couch-root"))
				state = seedReattach(state, []couchcore.ActionableThreadSummary{row})
				state, effects := advanceReattach(state)
				if len(effects) != 1 {
					t.Fatalf("no initial attempt: %+v", effects)
				}
				event := passResult(effects[0].Attempt, "worker")
				event.Busy, event.Diagnostic, event.Error = test.busy, test.diagnostic, test.failure
				state = finishReattach(state, event)
				viewed, _ := menuThread(state, row.Address)
				view, owned := passViewOf(state.Reattach, row.Address)
				if !owned || view.State != PassFailed || view.Diagnostic == "" || couchcore.ContinuityOutcome(viewed) != couchcore.Continue || !menuRowSelectable(state, row.Address) {
					t.Fatalf("failed Continue disappeared or changed outcome: row%+v view%+v owned%v", viewed, view, owned)
				}
				lines, _ := renderRootMenuFrame(state, state.Frames[0], 180, 20, time.Now(), false)
				painted := strings.Join(lines, "\n")
				if !strings.Contains(painted, "Continue") || !strings.Contains(painted, view.Diagnostic) {
					t.Fatalf("failure outcome/reason hidden: %s", painted)
				}
				clicked, retry := ReduceMenu(state, MenuEvent{Kind: MenuEventMouseSwitch, Address: row.Address})
				if len(retry) != 1 || retry[0].Background || (retry[0].Operation != "resume" || retry[0].Args["warm-only"] != "true") || clicked.InFlight.Operation == "" {
					t.Fatalf("one click did not retry Continue: %+v", retry)
				}
				if len(state.Reattach.Queue) != 0 || len(state.Reattach.InFlight) != 0 {
					t.Fatal("failure added an automatic scheduler retry")
				}
			})
		}
	}
}

func TestContinuePassPendingRowsLeaveUnrelatedOperationsResponsive(t *testing.T) {
	inventory := []couchcore.ActionableThreadSummary{
		reattachRow("couch-root", couchcore.ThreadLive, 0),
		reattachRow("worker", couchcore.ThreadLive, 1),
		reattachRow("parked", couchcore.ThreadParked, 2),
	}
	state, effects := passMenu(t, inventory)
	if len(effects) != 1 || state.InFlight.Operation != "" {
		t.Fatalf("background Continue took operator slot: %+v", state.InFlight)
	}
	state.Frames[0].SelectedAddress = menuAddress("couch-root")
	next, manual := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(manual) != 1 || manual[0].Background || next.InFlight.Operation == "" {
		t.Fatal("pending worker blocked root interaction")
	}
	// A background completion leaves the manual operation and active address alone.
	event := passResult(effects[0].Attempt, "worker")
	event.Success = true
	completed, _ := ReduceMenu(next, event)
	if completed.InFlight != next.InFlight || completed.ActiveAddress != next.ActiveAddress {
		t.Fatal("background Continue stole focus or operator operation")
	}
}

func TestFailedContinueRetryUsesExactSlotAddressAndKeepsFailureWhenBlocked(t *testing.T) {
	row := reattachRow("worker", couchcore.ThreadLive, 0)
	row.Target.Kind = couchcore.ThreadTargetSlot
	row.Target.Slot.WorktreeRoot = "/slot"
	row.RowKey = couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetSlot, SlotPath: "/slot"}
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, menuAddress("couch-root"))
	state.Reattach.Failed = map[couchcore.ThreadAddress]string{row.Address: "helper failed"}
	blocked := state
	blocked.InFlight = MenuOperationOrigin{Operation: "park", Attempt: 99}
	unchanged, effects := dispatchMenuRow(blocked, "switch", row)
	if len(effects) != 0 || unchanged.Reattach.Failed[row.Address] != "helper failed" {
		t.Fatal("blocked retry cleared failure without dispatch")
	}
	next, effects := dispatchMenuRow(state, "switch", row)
	if len(effects) != 1 || effects[0].Operation != "resume" || effects[0].Args["warm-only"] != "true" || effects[0].Args["tag"] != string(row.Address.Tag) || effects[0].Args["repo-scope"] != row.Address.RepoScope || effects[0].Args["path"] != "" || next.InFlight.RowKey != row.RowKey {
		t.Fatalf("failed slot retry lost exact warm-only authority: %+v %+v", effects, next.InFlight)
	}
	if state.Reattach.Failed[row.Address] != "helper failed" {
		t.Fatal("retry mutated its previous state")
	}
	if next.Reattach.Failed[row.Address] != "" {
		t.Fatal("dispatched retry retained stale failure")
	}
	// An actual newer observation may change the outcome; the failed pass does
	// not invent positive listing over an authoritative unknown/gone state.
	changed := row
	changed.State = couchcore.ThreadUnusable
	changed.Reason = couchcore.ReasonSessionGone
	state.Inventory = []couchcore.ActionableThreadSummary{changed}
	lines, _ := renderRootMenuFrame(state, state.Frames[0], 180, 20, time.Now(), false)
	if painted := strings.Join(lines, "\n"); strings.Contains(painted, "Continue") || !strings.Contains(painted, "helper failed") {
		t.Fatalf("failure notice hid newer projected outcome: %s", painted)
	}
	if viewed := overlayPassRow(state.Reattach, changed); viewed.State != changed.State || viewed.Reason != changed.Reason {
		t.Fatal("failure hid newer inventory evidence")
	}
}
