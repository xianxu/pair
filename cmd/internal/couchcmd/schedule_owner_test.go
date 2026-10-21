package couchcmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

func TestScheduleOwnerCanonicalTerminalPrecedesCatalogueEdit(t *testing.T) {
	for _, edit := range []string{"removed", "revision", "owner-policy"} {
		t.Run(edit, func(t *testing.T) {
			a, fake, state, notices := scheduledDeliveryFixture(t)
			key := scheduleRunKey(state)
			if err := a.history().FinalizeNotice(t.Context(), key, state.Active.Revision, "submitted"); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(state.Key.Hub, "scripts/scheduled/report/schedule")
			switch edit {
			case "removed":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "revision":
				if err := os.WriteFile(path, []byte(`{"version":1,"cron":"*/2 * * * *","timeout":"1m","script_run":true}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "owner-policy":
				if err := os.WriteFile(path, []byte(`{"version":1,"cron":"* * * * *","timeout":"1m","script_run":true,"scope":"team","owner":"absent-owner"}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			fail := true
			a.coordinator.store.BeforePersist = func() error {
				if fail {
					return errors.New("authority unavailable")
				}
				return nil
			}
			owner := newScheduleOwner(a.coordinator)
			if err := owner.open(t.Context()); err != nil {
				t.Fatal(err)
			}
			defer owner.close()
			if err := owner.refresh(t.Context()); err == nil {
				t.Fatal("failed authority commit acknowledged")
			}
			ledger, err := a.coordinator.store.View(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if got := ledger.Tasks[state.Key.ID()]; got.Active == nil || got.Last.Notice != "pending" {
				t.Fatalf("failed commit released prior authority: %+v", got)
			}
			fail = false
			if err := owner.refresh(t.Context()); err != nil {
				t.Fatal(err)
			}
			ledger, err = a.coordinator.store.View(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			got := ledger.Tasks[state.Key.ID()]
			if got.Active != nil || got.Last == nil || got.Last.Notice != "submitted" {
				t.Fatalf("terminal history stranded by edit: %+v", got)
			}
			record, output, err := a.history().Read(t.Context(), key)
			if err != nil || record.Notice != "submitted" || string(output) != "saved execution output" {
				t.Fatalf("canonical facts changed: %+v %q %v", record, output, err)
			}
			if len(fake.Runs()) != 0 || notices.admissions != 0 {
				t.Fatal("settlement replayed an external effect")
			}
		})
	}
}

func TestScheduleOwnerCatalogueEditBusyWhileExecuting(t *testing.T) {
	a, _, state, _ := scheduledResultFixture(t)
	if err := os.Remove(filepath.Join(state.Key.Hub, "scripts/scheduled/report/schedule")); err != nil {
		t.Fatal(err)
	}
	owner := newScheduleOwner(a.coordinator)
	if err := owner.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer owner.close()
	if err := owner.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	ledger, err := a.coordinator.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := ledger.Tasks[state.Key.ID()]
	if got.Active == nil || got.Active.Stage != "running" || got.Revision != state.Revision || !got.Allowed {
		t.Fatalf("edit cancelled retained execution: %+v", got)
	}
	if owner.busy[state.Key.ID()] == "" {
		t.Fatal("busy catalogue edit is invisible")
	}
	_ = couchschedule.SettlementStable
}

func TestScheduleOwnerCompletionPublishesCanonicalFactsBeforeAuthority(t *testing.T) {
	a, _, state, _ := scheduledResultFixture(t)
	fail := false
	a.coordinator.store.BeforePersist = func() error {
		if fail {
			return errors.New("ledger commit failed")
		}
		return nil
	}
	owner := newScheduleOwner(a.coordinator)
	if err := owner.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer owner.close()
	facts := &scheduleExecutionFacts{result: couchschedule.ExecutionResult{Exit: 10, Outcome: "wake", Output: []byte("new result")}, ended: a.coordinator.now().Add(time.Second)}
	event := couchschedule.Event{Kind: couchschedule.TaskFinished, At: facts.ended, Attempt: state.Active.ID, Revision: state.Active.Revision, Exit: 10, Outcome: "wake", HubOnline: true}
	fail = true
	if _, err := owner.apply(t.Context(), state.Key, event, facts); err == nil {
		t.Fatal("failed authority commit acknowledged")
	}
	record, output, err := a.history().Read(t.Context(), scheduledResultRecord(state, facts.ended, "wake").Key)
	if err != nil || record.Notice != "pending" || string(output) != "new result" {
		t.Fatalf("completion facts not saved: %+v %q %v", record, output, err)
	}
	fail = false
	if _, err := owner.apply(t.Context(), state.Key, event, facts); err != nil {
		t.Fatal(err)
	}
	l, err := a.coordinator.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := l.Tasks[state.Key.ID()]
	if got.Last == nil || got.Last.Outcome != "wake" || got.Active == nil || got.Active.Stage != "notifying" {
		t.Fatalf("completion not canonical: %+v", got)
	}
}
