package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

func TestScheduleTaskHistoryIsolation(t *testing.T) {
	for _, consumer := range []string{"startup passive", "startup execution", "ongoing passive", "completion", "notice preparation", "catalogue cancellation", "unknown repair"} {
		t.Run(consumer, func(t *testing.T) {
			c, roots, now := scheduleServiceFixture(t)
			a := newAutomaticScheduler(c, nil, onlineScheduleNotices{})
			if consumer == "startup execution" || consumer == "completion" {
				if err := os.WriteFile(filepath.Join(roots[0], "scripts/scheduled/report/schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m","script_run":true}`), 0600); err != nil {
					t.Fatal(err)
				}
				if err := c.refresh(t.Context()); err != nil {
					t.Fatal(err)
				}
			}
			key := couchschedule.TaskKey{Hub: roots[0], Task: "report"}
			var retained couchschedule.TaskState
			if err := c.store.Update(t.Context(), func(l *couchschedule.Ledger) error {
				s := l.Tasks[key.ID()]
				if strings.Contains(consumer, "passive") {
					s.Last = &couchschedule.RunSummary{ID: "damaged", Revision: s.Revision, Source: "scheduled", StartedAt: *now, EndedAt: *now, DueAt: *now, Outcome: "skipped", Notice: "hub offline, skipped"}
				} else {
					s.Active = &couchschedule.Attempt{ID: "damaged", Revision: s.Revision, Source: "scheduled", Stage: "reserved", StartedAt: *now, DueAt: *now, Deadline: now.Add(time.Minute)}
					if consumer == "startup execution" || consumer == "completion" {
						s.Active.Stage = "running"
					}
					if consumer == "unknown repair" {
						s.Active.Stage = "uncertain"
						s.Last = &couchschedule.RunSummary{ID: s.Active.ID, Revision: s.Revision, Source: "scheduled", StartedAt: *now, DueAt: *now, EndedAt: *now, Outcome: "unknown", Notice: "uncertain"}
					}
					if consumer == "notice preparation" || consumer == "catalogue cancellation" {
						s.Active.Stage = "notifying"
						s.Active.NoticeMayHaveHappened = true
						s.Active.NoticeDeadline = now.Add(time.Minute)
						s.Last = &couchschedule.RunSummary{ID: "damaged", Revision: s.Revision, Source: "scheduled", StartedAt: *now, EndedAt: *now, DueAt: *now, Outcome: "completed", Notice: "pending"}
					}
				}
				retained = s
				l.Tasks[key.ID()] = s
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			// An existing bundle with a missing member is damage, never missing history.
			bundle := filepath.Join(filepath.Dir(c.store.Dir), "schedule", key.HubID(), key.Task, "runs", "damaged")
			if err := os.MkdirAll(bundle, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(bundle, "output"), []byte("retained"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(filepath.Dir(filepath.Dir(bundle)), ".runs.lock"), nil, 0600); err != nil {
				t.Fatal(err)
			}

			var err error
			switch consumer {
			case "startup passive", "startup execution":
				err = a.recover(t.Context())
			case "completion":
				a.pending[key.ID()] = scheduleJobResult{job: scheduleJob{state: retained}}
				a.inFlight[key.ID()] = true
				err = a.step(t.Context())
			case "unknown repair":
				err = ensureUncertainRuns(t.Context(), c.store, c.runs)
			case "catalogue cancellation":
				if e := os.RemoveAll(filepath.Join(roots[0], "scripts/scheduled/report")); e != nil {
					t.Fatal(e)
				}
				err = c.refresh(t.Context())
			default:
				err = a.step(t.Context())
			}
			if err != nil {
				t.Fatalf("task history blocked healthy hub: %v", err)
			}
			l, e := c.store.View(t.Context())
			if e != nil {
				t.Fatal(e)
			}
			got := l.Tasks[key.ID()]
			if got.Allowed || !strings.Contains(got.Refusal, "history") {
				t.Fatalf("missing visible history quarantine: %+v", got)
			}
			if retained.Active != nil && (got.Active == nil || got.Active.ID != retained.Active.ID || got.Active.Stage != retained.Active.Stage) {
				t.Fatalf("history damage changed retained authority: before=%+v after=%+v", retained.Active, got.Active)
			}
			if retained.Last != nil && (got.Last == nil || got.Last.ID != retained.Last.ID || got.Last.Outcome != retained.Last.Outcome) {
				t.Fatalf("history damage replaced facts: %+v", got)
			}
			if _, e := os.Stat(filepath.Join(bundle, "record.json")); !os.IsNotExist(e) {
				t.Fatalf("invented missing history member: %v", e)
			}
			// The other hub can still reserve and dispatch after the refusal.
			*now = now.Add(time.Minute)
			if e := a.step(t.Context()); e != nil {
				t.Fatalf("healthy subsequent dispatch blocked: %v", e)
			}
			healthy := false
			for len(a.jobs) > 0 {
				job := <-a.jobs
				if job.state.Key.Hub == roots[1] {
					healthy = true
				}
			}
			if !healthy {
				t.Fatal("healthy second hub did not dispatch")
			}
			if e := c.publish(t.Context()); e != nil {
				t.Fatal(e)
			}
			if c.view().Error != "" {
				t.Fatalf("task damage published global error: %s", c.view().Error)
			}
			if consumer == "startup execution" {
				repaired := operationalrun.Record{Version: 1, Key: operationalrun.RunKey{Hub: key.HubID(), Task: key.Task, ID: retained.Active.ID}, Source: "scheduled", DeclarationRevision: retained.Active.Revision, StartedAt: retained.Active.StartedAt, EndedAt: *now, ScheduledAt: retained.Active.DueAt, Outcome: "quiet", Notice: "none"}
				raw, e := json.Marshal(repaired)
				if e != nil {
					t.Fatal(e)
				}
				if e := os.WriteFile(filepath.Join(bundle, "record.json"), raw, 0600); e != nil {
					t.Fatal(e)
				}
				if e := c.refresh(t.Context()); e != nil {
					t.Fatal(e)
				}
				if e := a.step(t.Context()); e != nil {
					t.Fatal(e)
				}
				repairedLedger, e := c.store.View(t.Context())
				if e != nil {
					t.Fatal(e)
				}
				repairedState := repairedLedger.Tasks[key.ID()]
				if repairedState.Active != nil || repairedState.Last == nil || repairedState.Last.ID != repaired.Key.ID || repairedState.Last.Outcome != "quiet" {
					t.Fatalf("startup quarantine never recovered canonical completion: %+v", repairedState)
				}
			}

		})
	}
}

func TestScheduleManualHistoryFailureVisible(t *testing.T) {
	runner, fake, store, before := retainedPassiveFixture(t)
	dir := filepath.Join(runner.namespace, "schedule", before.Key.HubID())
	if err := os.WriteFile(dir, []byte("history unavailable"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.run(t.Context(), before.Key.Hub, before.Key.Task, nil); err == nil {
		t.Fatal("manual replacement ignored history failure")
	}
	l, err := store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := l.Tasks[before.Key.ID()]
	if got.Allowed || !strings.Contains(got.Refusal, "history") || got.Active != nil || got.Last.ID != before.Last.ID || len(fake.Runs()) != 0 {
		t.Fatalf("manual refusal lost quarantine/facts: %+v", got)
	}
}

func TestScheduleTaskHistoryGlobalFailuresPropagate(t *testing.T) {
	c, _, _ := scheduleServiceFixture(t)
	a := newAutomaticScheduler(c, nil, onlineScheduleNotices{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := a.recover(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was quarantined: %v", err)
	}
	failure := errors.New("shared ledger publication unavailable")
	c.store.BeforePersist = func() error { return failure }
	if err := a.recover(t.Context()); !errors.Is(err, failure) {
		t.Fatalf("ledger failure was quarantined: %v", err)
	}
}

func TestScheduleHistoryPendingEvidenceRetriedWithoutReplay(t *testing.T) {
	for _, admission := range []bool{false, true} {
		t.Run(fmt.Sprint("admission=", admission), func(t *testing.T) {
			c, roots, now := scheduleServiceFixture(t)
			a := newAutomaticScheduler(c, nil, onlineScheduleNotices{})
			key := couchschedule.TaskKey{Hub: roots[0], Task: "report"}
			var state couchschedule.TaskState
			if err := c.store.Update(t.Context(), func(l *couchschedule.Ledger) error {
				state = l.Tasks[key.ID()]
				state.Active = &couchschedule.Attempt{ID: "pending-result", Source: "scheduled", Revision: state.Revision, Stage: "reserved", StartedAt: *now, DueAt: *now, Deadline: now.Add(time.Minute)}
				if admission {
					state.Active.Stage = "delivery"
					state.Active.NoticeMayHaveHappened = true
					state.Active.NoticeDeadline = now.Add(time.Minute)
					state.Active.ReceiptID = "receipt"
					state.Active.NoticeBody = "frozen"
					state.Active.NoticeBinding = json.RawMessage(`{"slot":"hub"}`)
					state.Last = &couchschedule.RunSummary{ID: state.Active.ID, Revision: state.Revision, Source: "scheduled", StartedAt: *now, DueAt: *now, EndedAt: *now, Outcome: "completed", Notice: "pending"}
				}
				l.Tasks[key.ID()] = state
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			blocked := filepath.Join(filepath.Dir(c.store.Dir), "schedule", key.HubID())
			if err := os.WriteFile(blocked, []byte("unavailable"), 0600); err != nil {
				t.Fatal(err)
			}
			result := scheduleJobResult{job: scheduleJob{state: state, notice: admission}}
			if admission {
				result.notice = "submitted"
			}
			a.pending[key.ID()] = result
			a.inFlight[key.ID()] = true
			a.jobKinds[key.ID()] = admission
			if err := a.step(t.Context()); err != nil {
				t.Fatal(err)
			}
			if _, ok := a.pending[key.ID()]; !ok || !a.inFlight[key.ID()] {
				t.Fatal("history refusal lost pending completion evidence")
			}
			if err := c.refresh(t.Context()); err != nil {
				t.Fatal(err)
			}
			l, err := c.store.View(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if l.Tasks[key.ID()].Allowed {
				t.Fatal("catalogue erased history quarantine while damage persisted")
			}
			if err := os.Remove(blocked); err != nil {
				t.Fatal(err)
			}
			record := operationalrun.Record{Version: 1, Key: operationalrun.RunKey{Hub: key.HubID(), Task: key.Task, ID: state.Active.ID}, Source: "scheduled", DeclarationRevision: state.Revision, StartedAt: *now, EndedAt: *now, ScheduledAt: *now, Outcome: "quiet", Notice: "none"}
			if admission {
				record.Outcome = "completed"
				record.Notice = "pending"
			}
			if err := a.history().Publish(t.Context(), record.Key, record, nil); err != nil {
				t.Fatal(err)
			}
			// Before the existing retry deadline the result stays retained.
			if err := a.step(t.Context()); err != nil {
				t.Fatal(err)
			}
			if _, ok := a.pending[key.ID()]; !ok {
				t.Fatal("history retry ignored one-second bound")
			}
			*now = now.Add(time.Second)
			if err := a.step(t.Context()); err != nil {
				t.Fatal(err)
			}
			if _, ok := a.pending[key.ID()]; ok || a.inFlight[key.ID()] {
				t.Fatal("repaired history did not consume pending completion")
			}
			l, err = c.store.View(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			got := l.Tasks[key.ID()]
			if got.Active != nil || got.Last == nil || got.Last.ID != record.Key.ID {
				t.Fatalf("repair lost canonical terminal facts: %+v", got)
			}
			if admission && got.Last.Notice != "submitted" {
				t.Fatalf("lost pending terminal admission: %+v", got.Last)
			}
			if len(a.jobs) != 0 {
				t.Fatal("repair replayed execution or admission")
			}
		})
	}
}
