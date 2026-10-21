package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

// A renamed, readable bundle is not completion authority until the same native
// task-locked durability boundary succeeds. Every authority consumer sees the
// same damaged task alongside an independently dispatchable hub.
func TestScheduleDurableAdoptionConsumerInventory(t *testing.T) {
	for _, consumer := range []string{"startup", "completion", "catalogue manual", "reserved script", "reserved manual", "reserved notify", "passive", "manual replacement", "notice preparation", "notice finalization", "catalogue cancellation", "unknown", "quarantine release"} {
		t.Run(consumer, func(t *testing.T) {
			c, roots, now := scheduleServiceFixture(t)
			if consumer != "reserved notify" {
				if err := os.WriteFile(filepath.Join(roots[0], "scripts/scheduled/report/schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m","script_run":true}`), 0600); err != nil {
					t.Fatal(err)
				}
				if err := c.refresh(t.Context()); err != nil {
					t.Fatal(err)
				}
			}
			a := newAutomaticScheduler(c, nil, &resultScheduleNotices{})
			key := couchschedule.TaskKey{Hub: roots[0], Task: "report"}
			var before couchschedule.TaskState
			if err := c.store.Update(t.Context(), func(l *couchschedule.Ledger) error {
				s := l.Tasks[key.ID()]
				s.NextDue = now.Add(time.Hour)
				s.Active = &couchschedule.Attempt{ID: "renamed-result", Source: "scheduled", Revision: s.Revision, Stage: "running", DueAt: now.Add(-time.Minute), StartedAt: now.Add(-30 * time.Second), Deadline: now.Add(time.Minute)}
				s.Spec.ScriptRun = true
				switch consumer {
				case "reserved script":
					s.Active.Stage = "reserved"
				case "catalogue manual", "reserved manual":
					s.Active.Source = "manual"
					s.Active.Stage = "reserved"
				case "reserved notify":
					s.Active.Stage = "reserved"
					s.Spec.ScriptRun = false
				case "passive", "manual replacement":
					s.Last = &couchschedule.RunSummary{ID: s.Active.ID, Revision: s.Revision, Source: "scheduled", DueAt: s.Active.DueAt, StartedAt: s.Active.StartedAt, EndedAt: *now, Outcome: "skipped", Notice: "hub offline, skipped"}
					s.Active = nil
				case "notice preparation", "notice finalization", "catalogue cancellation":
					s.Active.Stage = "notifying"
					s.Active.NoticeMayHaveHappened = true
					s.Active.NoticeDeadline = now.Add(time.Minute)
					s.Last = &couchschedule.RunSummary{ID: s.Active.ID, Revision: s.Revision, Source: "scheduled", DueAt: s.Active.DueAt, StartedAt: s.Active.StartedAt, EndedAt: *now, Outcome: "wake", Notice: "pending"}
					if consumer == "notice finalization" {
						s.Active.Stage = "delivery"
						s.Active.ReceiptID = "saved-receipt"
						s.Active.NoticeBody = "frozen result"
						s.Active.NoticeBinding = []byte(`{"slot":"hub"}`)
						s.Active.NoticeMayHaveHappened = true
					}
				case "unknown":
					s.Active.Stage = "uncertain"
					s.Last = &couchschedule.RunSummary{ID: s.Active.ID, Revision: s.Revision, Source: "scheduled", DueAt: s.Active.DueAt, StartedAt: s.Active.StartedAt, EndedAt: *now, Outcome: "unknown", Notice: "uncertain"}
				case "quarantine release":
					s.Allowed = false
					s.Refusal = "task report history previous failure"
				}
				before = s
				l.Tasks[key.ID()] = s
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			reopened, err := c.store.View(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			before = reopened.Tasks[key.ID()]
			rec := operationalrun.Record{Version: 1, Key: operationalrun.RunKey{Hub: key.HubID(), Task: key.Task, ID: "renamed-result"}, DeclarationRevision: before.Revision, Source: "scheduled", ScheduledAt: now.Add(-time.Minute), StartedAt: now.Add(-30 * time.Second), EndedAt: *now, Outcome: "quiet", Notice: "none"}
			if before.Active != nil {
				rec.Source = before.Active.Source
			}
			if before.Last != nil {
				rec.Outcome = before.Last.Outcome
				rec.Notice = before.Last.Notice
			}
			if consumer == "reserved notify" {
				rec.Outcome = "completed"
				rec.Notice = "none"
			}
			taskDir := filepath.Join(c.runs.Namespace, "schedule", key.HubID(), key.Task)
			failing := true
			failure := errors.New("injected post-rename directory sync failure")
			c.runs.SyncDirectory = func(path string) error {
				if failing && (path == taskDir || strings.HasPrefix(path, taskDir+string(os.PathSeparator))) {
					return failure
				}
				return durablefile.SyncDirectory(path)
			}
			if _, _, err := c.runs.PublishOccurrence(t.Context(), rec.Key, rec, []byte("canonical retained output")); !errors.Is(err, failure) {
				t.Fatalf("publication must fail after rename: %v", err)
			}
			if _, out, err := c.runs.Read(t.Context(), rec.Key); err != nil || string(out) != "canonical retained output" {
				t.Fatalf("renamed bundle must remain readable: %q %v", out, err)
			}
			fake := couchschedule.NewFakeTaskRuntime()
			runner := &manualTaskRunner{namespace: c.runs.Namespace, history: c.runs, executor: fake, now: func() time.Time { return *now }, newID: func() (string, error) { return "replacement", nil }, withDeadline: func(ctx context.Context, _ time.Time) (context.Context, context.CancelFunc) {
				return context.WithCancel(ctx)
			}}
			if consumer == "catalogue cancellation" {
				if err := os.RemoveAll(filepath.Join(roots[0], "scripts/scheduled/report")); err != nil {
					t.Fatal(err)
				}
			}
			action := func() error {
				switch consumer {
				case "startup":
					return a.recover(t.Context())
				case "completion":
					return a.reconcileExecution(t.Context(), scheduleJobResult{job: scheduleJob{state: before}})
				case "catalogue manual", "catalogue cancellation", "quarantine release":
					return c.refresh(t.Context())
				case "reserved script", "reserved manual":
					_, err := runner.runReserved(t.Context(), before, nil)
					return err
				case "manual replacement":
					_, err := runner.run(t.Context(), roots[0], "report", nil)
					return err
				default:
					return a.step(t.Context())
				}
			}
			for retry := 0; retry < 2; retry++ {
				err := action()
				var local *taskHistoryError
				if err != nil && !errors.Is(err, failure) && !errors.As(err, &local) {
					t.Fatalf("task failure became global: %v", err)
				}
				ledger, err := c.store.View(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				got := ledger.Tasks[key.ID()]
				if got.Allowed || !strings.Contains(got.Refusal, "report") || !strings.Contains(got.Refusal, "history") {
					t.Fatalf("missing named task quarantine: %+v", got)
				}
				if !reflect.DeepEqual(got.Active, before.Active) || !reflect.DeepEqual(got.Last, before.Last) {
					t.Fatalf("readable non-durable result consumed authority/facts: before=%+v/%+v after=%+v/%+v", before.Active, before.Last, got.Active, got.Last)
				}
				if err := c.refresh(t.Context()); err != nil {
					t.Fatal(err)
				}
				ledger, err = c.store.View(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				if ledger.Tasks[key.ID()].Allowed {
					t.Fatal("catalogue released quarantine before durability repair")
				}
			}
			if len(fake.Runs()) != 0 {
				t.Fatal("failed durable adoption replayed executor")
			}
			if err := c.store.Update(t.Context(), func(l *couchschedule.Ledger) error {
				healthy := couchschedule.TaskKey{Hub: roots[1], Task: "report"}
				s := l.Tasks[healthy.ID()]
				s.NextDue = *now
				l.Tasks[healthy.ID()] = s
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if err := a.step(t.Context()); err != nil {
				t.Fatal(err)
			}
			healthy := false
			for len(a.jobs) > 0 {
				job := <-a.jobs
				if job.state.Key.Hub == roots[1] {
					healthy = true
				} else {
					t.Fatal("damaged task dispatched")
				}
			}
			if !healthy {
				t.Fatal("healthy second hub failed to dispatch")
			}
			failing = false
			// Repair the retained occurrence without issuing a new manual request.
			if consumer != "manual replacement" {
				if err := action(); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.refresh(t.Context()); err != nil {
				t.Fatal(err)
			}
			if err := a.step(t.Context()); err != nil {
				t.Fatal(err)
			}
			ledger, err := c.store.View(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			got := ledger.Tasks[key.ID()]
			canonical, output, err := c.runs.Read(t.Context(), rec.Key)
			if err != nil {
				t.Fatal(err)
			}
			wantSummary := runSummaryForRecord(canonical)
			if got.Last != nil && len(got.Last.Unknown) == 0 {
				got.Last.Unknown = nil
			}
			activeExpected := consumer == "unknown" || consumer == "notice preparation"
			if (!activeExpected && got.Active != nil) || got.Last == nil || !reflect.DeepEqual(wantSummary, *got.Last) {
				t.Fatalf("repair did not adopt canonical facts: state=%+v/%+v history=%+v", got.Active, got.Last, canonical)
			}
			if string(output) != "canonical retained output" {
				t.Fatalf("repair changed output: %q", output)
			}
			if len(fake.Runs()) != 0 {
				t.Fatal("repair replayed executor")
			}
		})
	}
}

// Embedding only these interfaces deliberately does not promise ReadDurable,
// even though the wrapped production store can implement that capability.
type inspectionOnlyRunPublisher struct {
	runPublisher
	runReader
}

func TestScheduleReservedReadOnlyPublisherHasNoDurabilityPromise(t *testing.T) {
	runner, fake, before, store := reservedExecutionFixture(t)
	native := operationalrun.Store{Namespace: runner.namespace}
	record := scheduledResultRecord(before, runner.now(), "quiet")
	if err := native.Publish(t.Context(), record.Key, record, []byte("readable output")); err != nil {
		t.Fatal(err)
	}
	runner.history = inspectionOnlyRunPublisher{runPublisher: native, runReader: native}
	if _, err := runner.runReserved(t.Context(), before, nil); err == nil || !strings.Contains(err.Error(), "cannot confirm durability") {
		t.Fatalf("read-only publisher falsely adopted completion: %v", err)
	}
	ledger, err := store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := ledger.Tasks[before.Key.ID()]
	if got.Active == nil || got.Active.ID != before.Active.ID || got.Allowed || !strings.Contains(got.Refusal, "history") || len(fake.Runs()) != 0 {
		t.Fatalf("read-only evidence changed authority or replayed: %+v", got)
	}
}
