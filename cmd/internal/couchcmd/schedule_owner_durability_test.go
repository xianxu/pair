package couchcmd

import (
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

// Exercise production owner consumers, not the retired test-only orchestration.
// The real publication has renamed its bundle before directory sync refuses:
// inspection can read output, but no transition may treat it as durable evidence.
func TestScheduleOwnerDurableAdoptionConsumerInventory(t *testing.T) {
	for _, consumer := range []string{"refresh", "startup", "completion", "reserved script authorization", "reserved manual authorization", "reserved notify", "due", "passive", "passive quarantine release", "manual replacement", "manual duplicate", "manual cancellation", "notice preparation", "notice finalization", "catalogue removal", "unknown", "quarantine release"} {
		t.Run(consumer, func(t *testing.T) {
			c, roots, now := scheduleServiceFixture(t)
			script := consumer != "reserved notify"
			if script {
				if err := os.WriteFile(filepath.Join(roots[0], "scripts/scheduled/report/schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m","script_run":true}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			o := newScheduleOwner(c)
			notices := &resultScheduleNotices{}
			fake := couchschedule.NewFakeTaskRuntime()
			o.configure(scheduleExecutionWorker{executor: fake}, notices)
			if err := o.open(t.Context()); err != nil {
				t.Fatal(err)
			}
			defer o.close()
			if err := o.refresh(t.Context()); err != nil {
				t.Fatal(err)
			}
			key := couchschedule.TaskKey{Hub: roots[0], Task: "report"}
			intent := couchschedule.ManualIntent{ID: "renamed-result", Hub: roots[0], Task: "report", Args: []string{"--sample"}}
			var before couchschedule.TaskState
			if err := o.state.Update(t.Context(), func(l *couchschedule.Ledger) error {
				s := l.Tasks[key.ID()]
				s.NextDue = now.Add(time.Hour)
				s.Active = &couchschedule.Attempt{ID: intent.ID, Source: "scheduled", Revision: s.Revision, Stage: "running", DueAt: now.Add(-time.Minute), StartedAt: now.Add(-30 * time.Second), Deadline: now.Add(time.Minute)}
				switch consumer {
				case "reserved script authorization", "reserved notify":
					s.Active.Stage = "reserved"
				case "reserved manual authorization", "manual duplicate", "manual cancellation":
					s.Active.Source = "manual"
					s.Active.Stage = "reserved"
					s.Active.ManualIntentDigest = intent.Digest()
				case "passive", "due", "passive quarantine release", "manual replacement":
					s.Last = &couchschedule.RunSummary{ID: s.Active.ID, Revision: s.Revision, Source: "scheduled", DueAt: s.Active.DueAt, StartedAt: s.Active.StartedAt, EndedAt: *now, Outcome: "skipped", Notice: "hub offline, skipped"}
					s.Active = nil
					if consumer == "due" {
						s.NextDue = *now
					}
					if consumer == "passive quarantine release" {
						s.Allowed = false
						s.Refusal = "task report history previous failure"
					}
				case "notice preparation", "notice finalization", "catalogue removal":
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
				l.Tasks[key.ID()] = s
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			readState := func() couchschedule.TaskState {
				t.Helper()
				l, err := c.store.View(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				return l.Tasks[key.ID()]
			}
			before = readState()
			rec := operationalrun.Record{Version: 1, Key: operationalrun.RunKey{Hub: key.HubID(), Task: key.Task, ID: intent.ID}, DeclarationRevision: before.Revision, Source: "scheduled", ScheduledAt: now.Add(-time.Minute), StartedAt: now.Add(-30 * time.Second), EndedAt: *now, Outcome: "quiet", Notice: "none"}
			if before.Active != nil {
				rec.Source = before.Active.Source
				rec.ManualIntentDigest = before.Active.ManualIntentDigest
			}
			if before.Last != nil {
				rec.Outcome = before.Last.Outcome
				rec.Notice = before.Last.Notice
				if rec.Notice == "pending" {
					rec.Notice = "submitted"
				}
			}
			if consumer == "reserved notify" {
				rec.Outcome = "completed"
			}
			taskDir := filepath.Join(c.runs.Namespace, "schedule", key.HubID(), key.Task)
			failing := true
			failure := errors.New("injected owner post-rename directory sync failure")
			c.runs.SyncDirectory = func(path string) error {
				if failing && (path == taskDir || strings.HasPrefix(path, taskDir+string(os.PathSeparator))) {
					return failure
				}
				return durablefile.SyncDirectory(path)
			}
			output := []byte("canonical retained output")
			if _, _, err := c.runs.PublishOccurrence(t.Context(), rec.Key, rec, output); !errors.Is(err, failure) {
				t.Fatalf("publication must fail after rename: %v", err)
			}
			canonical, readOutput, err := c.runs.Read(t.Context(), rec.Key)
			if err != nil || !reflect.DeepEqual(readOutput, output) {
				t.Fatalf("renamed bundle unreadable: %q %v", readOutput, err)
			}
			if consumer == "catalogue removal" {
				if err := os.Remove(filepath.Join(roots[0], "scripts/scheduled/report/schedule")); err != nil {
					t.Fatal(err)
				}
			}
			clock := newSchedulerClockFake(*now)
			action := func() error {
				switch consumer {
				case "refresh", "catalogue removal", "unknown", "quarantine release", "passive quarantine release":
					return o.refresh(t.Context())
				case "startup":
					return o.recoverBatch(t.Context(), clock)
				case "completion":
					return o.complete(t.Context(), before, scheduleExecutionFacts{result: couchschedule.ExecutionResult{Exit: 10, Outcome: "wake", Output: []byte("conflicting result must lose")}, ended: now.Add(time.Second)})
				case "reserved script authorization", "reserved manual authorization":
					d, err := manualDeclaration(t.Context(), roots[0], "report")
					if err != nil {
						return err
					}
					return o.authorize(t.Context(), before, couchschedule.OwnerIdentity{PID: 7474, Start: "controller"}, d)
				case "passive":
					_, err := o.apply(t.Context(), key, couchschedule.Event{Kind: couchschedule.ClockObserved, At: *now, RunID: "replacement", Location: c.location}, nil)
					return err
				case "manual replacement":
					m := intent
					m.ID = "replacement"
					_, err := o.submitManual(t.Context(), m)
					return err
				case "manual duplicate":
					_, err := o.submitManual(t.Context(), intent)
					return err
				case "manual cancellation":
					_, err := o.cancelManual(t.Context(), intent.Hub, intent.Task, intent.ID)
					return err
				default:
					o.recovered = true
					return o.tick(t.Context(), clock)
				}
			}
			for retry := 0; retry < 2; retry++ {
				err := action()
				if err != nil && !errors.Is(err, failure) && !ownerTaskLocal(err) && !strings.Contains(err.Error(), "authority changed") && !strings.Contains(err.Error(), "declaration changed") && !(consumer == "manual replacement" && strings.HasPrefix(err.Error(), "task refused: task report history ")) {
					t.Fatalf("unexpected consumer failure: %v", err)
				}
				got := readState()
				if !reflect.DeepEqual(got.Active, before.Active) || !reflect.DeepEqual(got.Last, before.Last) {
					t.Fatalf("non-durable evidence consumed facts: before=%+v/%+v after=%+v/%+v", before.Active, before.Last, got.Active, got.Last)
				}
				// A cancellation observing an already-readable terminal result is inspection,
				// not an authority consumer. Refresh still must not release that occurrence.
				if err := o.refresh(t.Context()); err != nil {
					t.Fatal(err)
				}
				got = readState()
				if !reflect.DeepEqual(got.Active, before.Active) || !reflect.DeepEqual(got.Last, before.Last) {
					t.Fatal("refresh consumed non-durable facts")
				}
				if got.Allowed || !strings.Contains(got.Refusal, "history") {
					t.Fatalf("quarantine released before durability repair: %+v", got)
				}
			}
			// A damaged task cannot monopolize the owner: another hub still dispatches.
			if err := o.state.Update(t.Context(), func(l *couchschedule.Ledger) error {
				k := couchschedule.TaskKey{Hub: roots[1], Task: "report"}
				s := l.Tasks[k.ID()]
				s.NextDue = *now
				l.Tasks[k.ID()] = s
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			o.recovered = true
			if err := o.tick(t.Context(), clock); err != nil {
				t.Fatal(err)
			}
			healthy := false
			for len(o.loop.notices) > 0 {
				job := <-o.loop.notices
				if job.state.Key.Hub != roots[1] {
					t.Fatal("damaged task dispatched a notice")
				}
				healthy = true
			}
			if !healthy {
				t.Fatal("healthy second hub failed to dispatch")
			}
			if len(o.loop.execution) != 0 || len(fake.Runs()) != 0 || notices.admissions != 0 {
				t.Fatal("durability failure replayed an external effect")
			}
			failing = false
			// Rediscovery/settlement, never a new manual submit, adopts retained facts.
			if err := o.refresh(t.Context()); err != nil {
				t.Fatal(err)
			}
			if err := o.close(); err != nil {
				t.Fatal(err)
			}
			if err := o.open(t.Context()); err != nil {
				t.Fatal(err)
			}
			got := readState()
			want := runSummaryForRecord(canonical)
			if got.Last != nil && len(got.Last.Unknown) == 0 {
				got.Last.Unknown = nil
			}
			if got.Last == nil || !reflect.DeepEqual(*got.Last, want) || (consumer != "unknown" && got.Active != nil) {
				t.Fatalf("reopened repair did not adopt canonical facts: active=%+v last=%+v want=%+v", got.Active, got.Last, want)
			}
			// Resume timer processing after repair too. A genuinely new due slot
			// may run, but the retained occurrence must never be dispatched again.
			if err := o.tick(t.Context(), clock); err != nil {
				t.Fatal(err)
			}
			for _, jobs := range []chan scheduleOwnerJob{o.loop.execution, o.loop.notices} {
				for len(jobs) > 0 {
					job := <-jobs
					if job.state.Key == key && job.state.Active.ID == intent.ID {
						t.Fatal("repaired occurrence replayed")
					}
				}
			}
			final, finalOutput, err := c.runs.Read(t.Context(), rec.Key)
			if err != nil || !reflect.DeepEqual(final, canonical) || !reflect.DeepEqual(finalOutput, output) {
				t.Fatalf("repair rewrote retained facts/output: %+v %q %v", final, finalOutput, err)
			}
			if len(o.loop.execution) != 0 || len(fake.Runs()) != 0 || notices.admissions != 0 {
				t.Fatal("repair replayed an external effect")
			}
		})
	}
}
