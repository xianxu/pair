package couchtty

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// menuRowShape is one row the projection can produce, described by the facts
// the Spec table is written over, so the expectation can be stated without
// calling into production.
type menuRowShape struct {
	name    string
	slot    bool
	state   couchcore.ActionableThreadState
	reason  couchcore.ThreadReason
	phase   checkpoint.Phase // "" = no retained request
	recover bool             // unusable :0 with Recovery.Recover
	row     couchcore.ActionableThreadSummary
}

// everyMenuRowShape is the DERIVED row domain every per-row sweep iterates:
// kinds {:0, :1} x AllThreadStates() (minus archived) x AllThreadReasons() ∪ ""
// (only the combinations the projection produces: a reason exactly when
// unusable) x AllPhases() ∪ none, plus Recovery.Recover on unusable :0 rows. A
// new state, reason or phase lands here without anyone listing it.
func everyMenuRowShape(t *testing.T) []menuRowShape {
	t.Helper()
	scope, err := launcher.ResolveRepoScope("/w/xianxu.dev")
	if err != nil {
		t.Fatal(err)
	}
	phases := []checkpoint.Phase{""}
	phases = append(phases, checkpoint.AllPhases()...)
	var shapes []menuRowShape
	for _, slot := range []bool{false, true} {
		for _, state := range couchcore.AllThreadStates() {
			if state == couchcore.ThreadArchived {
				continue
			}
			for _, reason := range append(couchcore.AllThreadReasons(), "") {
				if (state == couchcore.ThreadUnusable) != (reason != "") {
					continue
				}
				for _, phase := range phases {
					recovers := []bool{false}
					if !slot && state == couchcore.ThreadUnusable {
						recovers = append(recovers, true)
					}
					for _, recover := range recovers {
						var row couchcore.ActionableThreadSummary
						if slot {
							row = menuSlotRow(1, "couch-slot")
						} else {
							row = couchcore.ActionableThreadSummary{
								Address:      couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-primary"},
								StartingPath: "/w/xianxu.dev", WorkingPath: "/w/xianxu.dev",
							}
						}
						row.State, row.Reason = state, reason
						if phase != "" {
							row.Continuation = &couchcore.ContinuationStatus{RequestID: "request", Phase: phase}
						}
						if recover {
							row.Recovery = &couchcore.RecoveryDecision{Recover: true, Archive: true}
						}
						kind := ":0"
						if slot {
							kind = ":1"
						}
						shapes = append(shapes, menuRowShape{
							name: kind + "/" + string(state) + "/" + string(reason) + "/" + string(phase) + map[bool]string{true: "/recover"}[recover],
							slot: slot, state: state, reason: reason, phase: phase, recover: recover, row: row,
						})
					}
				}
			}
		}
	}
	return shapes
}

// expectedRowActions is the Spec table, written out as its own statement.
func expectedRowActions(s menuRowShape) []string {
	unfinished := s.phase != "" && s.phase != checkpoint.Complete
	switch s.state {
	case couchcore.ThreadBusy:
		return nil
	case couchcore.ThreadLive:
		switch {
		case unfinished && s.phase == checkpoint.Running:
			return []string{"retry-continuation"}
		case unfinished && s.phase == checkpoint.Pending:
			return nil
		case unfinished && s.phase == checkpoint.Failed:
			// relaunch and switch-agent are what a failed request refuses.
			if s.slot {
				return []string{"detach", "retry-continuation", "dismiss-continuation", "park"}
			}
			return []string{"detach", "retry-continuation", "dismiss-continuation", "park", "alias", "add-slot"}
		}
		if s.slot {
			return []string{"detach", "relaunch", "park", "switch-agent"}
		}
		return []string{"detach", "relaunch", "park", "switch-agent", "alias", "add-slot"}
	case couchcore.ThreadParked, couchcore.ThreadDetached:
		return []string{"resume", "reboot"}
	case couchcore.ThreadUnusable:
		switch {
		case s.reason == couchcore.ReasonUnknown:
			return nil
		case s.reason == couchcore.ReasonPathMissing && s.slot:
			return nil
		case s.reason == couchcore.ReasonPathMissing:
			return []string{"reboot"}
		case s.slot || s.recover || unfinished:
			return []string{"resume", "reboot"}
		}
		return []string{"reboot"}
	}
	return nil
}

func TestRowActionTableMatchesTheSpec(t *testing.T) {
	shapes := everyMenuRowShape(t)
	if len(shapes) < 50 {
		t.Fatalf("derived domain has only %d shapes", len(shapes))
	}
	for _, s := range shapes {
		got, want := menuActionItems(s.row), expectedRowActions(s)
		if !slices.Equal(got, want) {
			t.Errorf("%s: offers %v, Spec says %v", s.name, got, want)
		}
	}
}

// A busy row's status and its Enter notice come from the table's busy phase,
// one per-row authority, and both say what is actually happening.
func TestBusyRowSaysStartingElsewhere(t *testing.T) {
	busy := couchcore.ActionableThreadSummary{
		Address: menuAddress("couch-one"), WorkingPath: "/repo/one", State: couchcore.ThreadBusy,
	}
	if got := rootStateText(busy, time.Now()); got != "starting elsewhere" {
		t.Fatalf("busy status = %q, want %q", got, "starting elsewhere")
	}
	state := NewMenuState([]couchcore.ActionableThreadSummary{busy}, couchcore.ThreadAddress{})
	state.InventoryReady = true
	got, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 || !strings.Contains(got.Notice.Text, "starting elsewhere") {
		t.Fatalf("Enter on busy: effects %v, notice %q", effects, got.Notice.Text)
	}
}
