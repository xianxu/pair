package couchcmd

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

func TestScheduleCanonicalRecoveryPreservesSummaryNamespace(t *testing.T) {
	for _, manual := range []bool{false, true} {
		name := "scheduled"
		if manual {
			name = "manual"
		}
		t.Run(name, func(t *testing.T) {
			var c *scheduleCoordinator
			var state couchschedule.TaskState
			var record operationalrun.Record
			if manual {
				var roots []string
				c, roots, _ = scheduleServiceFixture(t)
				state = serviceManualOwner(t, c, roots[0])
				exit := 0
				record = operationalrun.Record{Version: 1, Key: operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: state.Active.ID}, DeclarationRevision: state.Active.Revision, Source: "manual", StartedAt: state.Active.StartedAt, EndedAt: state.Active.StartedAt.Add(time.Second), Exit: &exit, Outcome: "quiet", Notice: "manual"}
			} else {
				a, _, s, _ := scheduledResultFixture(t)
				c, state = a.coordinator, s
				record = scheduledResultRecord(state, c.now().Add(time.Second), "quiet")
			}
			state.Last = &couchschedule.RunSummary{ID: state.Active.ID, Revision: state.Active.Revision, Source: state.Active.Source, DueAt: state.Active.DueAt, StartedAt: state.Active.StartedAt, EndedAt: record.EndedAt, Exit: record.Exit, Outcome: record.Outcome, Notice: record.Notice, Unknown: map[string]json.RawMessage{"future_summary_field": json.RawMessage(`{"nested":[42]}`)}}
			if err := c.store.Update(t.Context(), func(l *couchschedule.Ledger) error { l.Tasks[state.Key.ID()] = state; return nil }); err != nil {
				t.Fatal(err)
			}
			record.Unknown = map[string]json.RawMessage{"future_record_field": json.RawMessage(`{"value":99}`)}
			if err := (operationalrun.Store{Namespace: filepath.Dir(c.store.Dir)}).Publish(t.Context(), record.Key, record, nil); err != nil {
				t.Fatal(err)
			}
			if manual {
				if err := c.refresh(t.Context()); err != nil {
					t.Fatal(err)
				}
			} else if err := newAutomaticScheduler(c, nil, offlineScheduleNotices{}).recover(t.Context()); err != nil {
				t.Fatal(err)
			}
			l, err := c.store.View(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			got := l.Tasks[state.Key.ID()]
			if got.Active != nil || got.Last == nil || string(got.Last.Unknown["future_summary_field"]) != `{"nested":[42]}` {
				t.Fatalf("lost persisted summary metadata: %+v", got.Last)
			}
			if _, ok := got.Last.Unknown["future_record_field"]; ok {
				t.Fatal("record namespace leaked into summary")
			}
			if err := c.publish(t.Context()); err != nil {
				t.Fatal(err)
			}
			view := c.view()
			for _, task := range view.Tasks {
				if task.Hub == state.Key.Hub && task.Task == state.Key.Task {
					task.Last.Unknown["future_summary_field"][0] = 'x'
				}
			}
			for _, task := range c.view().Tasks {
				if task.Hub == state.Key.Hub && task.Task == state.Key.Task && string(task.Last.Unknown["future_summary_field"]) != `{"nested":[42]}` {
					t.Fatal("status view aliases published unknown bytes")
				}
			}
		})
	}
}
