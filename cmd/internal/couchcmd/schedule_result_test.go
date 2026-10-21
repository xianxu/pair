package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

type resultScheduleNotices struct{ admissions int }

func (*resultScheduleNotices) Binding(string) (json.RawMessage, bool) {
	return json.RawMessage(`{"slot":"hub"}`), true
}
func (n *resultScheduleNotices) Admit(context.Context, string, json.RawMessage, string, string) (string, error) {
	n.admissions++
	return "submitted", nil
}
func (*resultScheduleNotices) Observe(string) (string, bool, bool) { return "submitted", true, false }

// Use a historical fixed clock and actual ledger/history stores. The executor
// remains a stateful fake, so any attempt to replay execution is observable.
func scheduledResultFixture(t *testing.T) (*automaticScheduler, *couchschedule.FakeTaskRuntime, couchschedule.TaskState, *resultScheduleNotices) {
	t.Helper()
	runner, fake, state, store := reservedExecutionFixture(t)
	owner := couchschedule.OwnerIdentity{PID: 9191, Start: "saved-execution-owner"}
	state, _ = couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.OwnerRecorded, At: runner.now(), Attempt: state.Active.ID, Revision: state.Active.Revision, Owner: &owner})
	state, _ = couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.ReservationDurable, At: runner.now(), Attempt: state.Active.ID, Revision: state.Active.Revision})
	if state.Active == nil || state.Active.Stage != "running" {
		t.Fatalf("missing running authority: %+v", state)
	}
	if err := store.Update(t.Context(), func(l *couchschedule.Ledger) error { l.Tasks[state.Key.ID()] = state; return nil }); err != nil {
		t.Fatal(err)
	}
	c := newScheduleCoordinator(runner.namespace, []string{state.Key.Hub}, time.UTC, runner.now)
	notices := &resultScheduleNotices{}
	return newAutomaticScheduler(c, runner, notices), fake, state, notices
}
func scheduledResultRecord(s couchschedule.TaskState, at time.Time, outcome string) operationalrun.Record {
	exit, notice := 10, "pending"
	if outcome == "quiet" {
		exit, notice = 0, "none"
	}
	return operationalrun.Record{Version: 1, Key: operationalrun.RunKey{Hub: s.Key.HubID(), Task: s.Key.Task, ID: s.Active.ID}, DeclarationRevision: s.Active.Revision, Source: s.Active.Source, ScheduledAt: s.Active.DueAt, StartedAt: s.Active.StartedAt, EndedAt: at, Exit: &exit, Outcome: outcome, Notice: notice, Unknown: map[string]json.RawMessage{"build_key": json.RawMessage(`"saved-binary"`)}}
}
func saveScheduledResult(t *testing.T, a *automaticScheduler, r operationalrun.Record) {
	t.Helper()
	if err := a.history().Publish(t.Context(), r.Key, r, []byte("saved execution output")); err != nil {
		t.Fatal(err)
	}
}
func readScheduledResultState(t *testing.T, a *automaticScheduler, key couchschedule.TaskKey) couchschedule.TaskState {
	t.Helper()
	ledger, err := a.coordinator.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return ledger.Tasks[key.ID()]
}
func scheduledDeliveryFixture(t *testing.T) (*automaticScheduler, *couchschedule.FakeTaskRuntime, couchschedule.TaskState, *resultScheduleNotices) {
	t.Helper()
	a, fake, s, notices := scheduledResultFixture(t)
	record := scheduledResultRecord(s, a.coordinator.now(), "wake")
	saveScheduledResult(t, a, record)
	at := s.Active
	s, _ = couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.TaskFinished, At: a.coordinator.now(), Attempt: at.ID, Revision: at.Revision, Exit: 10, Outcome: "wake", HubOnline: true})
	s, _ = couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.NoticeFrozen, At: a.coordinator.now(), Attempt: at.ID, Revision: at.Revision, ReceiptID: "schedule-" + at.ID, NoticeBody: "saved task result", NoticeBinding: json.RawMessage(`{"slot":"hub"}`)})
	s, _ = couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.NoticeIntentDurable, At: a.coordinator.now(), Attempt: at.ID, Revision: at.Revision, HubOnline: true})
	if s.Active == nil || s.Active.Stage != "delivery" {
		t.Fatalf("missing delivery authority: %+v", s)
	}
	if err := a.coordinator.store.Update(t.Context(), func(l *couchschedule.Ledger) error { l.Tasks[s.Key.ID()] = s; return nil }); err != nil {
		t.Fatal(err)
	}
	return a, fake, s, notices
}

func TestScheduleResultSavedExecutionAdoptedWithoutReplay(t *testing.T) {
	for _, outcome := range []string{"quiet", "wake"} {
		t.Run(outcome, func(t *testing.T) {
			a, fake, s, notices := scheduledResultFixture(t)
			record := scheduledResultRecord(s, a.coordinator.now(), outcome)
			saveScheduledResult(t, a, record)
			key := s.Key.ID()
			a.inFlight[key] = true
			a.results <- scheduleJobResult{job: scheduleJob{state: s}, err: errors.New("ledger completion write failed after saved result")}
			if err := a.step(t.Context()); err != nil {
				t.Fatal(err)
			}
			next := readScheduledResultState(t, a, s.Key)
			if next.Last == nil || next.Last.ID != s.Active.ID || next.Last.Outcome != outcome || !next.NextDue.Equal(s.NextDue) {
				t.Fatalf("saved execution not adopted: %+v", next)
			}
			if outcome == "quiet" && next.Active != nil {
				t.Fatalf("quiet completion retained authority: %+v", next)
			}
			if outcome == "wake" && (next.Active == nil || next.Active.Stage != "delivery" || next.Last.Notice != "pending") {
				t.Fatalf("saved wake did not reach notice admission: %+v", next)
			}
			if len(fake.Runs()) != 0 || notices.admissions != 0 {
				t.Fatal("adoption replayed an effect")
			}
			for len(a.jobs) > 0 {
				if job := <-a.jobs; !job.notice {
					t.Fatal("saved execution dispatched again")
				}
			}
			if len(a.pending) != 0 {
				t.Fatal("adopted worker result retained")
			}
		})
	}
}

func TestScheduleResultNoticeFinalizeSurvivesLedgerWriteFailure(t *testing.T) {
	a, fake, s, notices := scheduledDeliveryFixture(t)
	at := s.Active
	notice, err := notices.Admit(t.Context(), s.Key.Hub, at.NoticeBinding, at.ReceiptID, at.NoticeBody)
	if err != nil {
		t.Fatal(err)
	}
	job := scheduleJob{state: s, notice: true}
	key := s.Key.ID()
	a.inFlight[key] = true
	a.results <- scheduleJobResult{job: job, notice: notice}
	injected := errors.New("ledger commit refused after terminal history")
	a.coordinator.store.BeforePersist = func() error { return injected }
	if err = a.step(t.Context()); !errors.Is(err, injected) {
		t.Fatalf("missing ledger fault: %v", err)
	}
	record, output, err := a.history().Read(t.Context(), scheduleRunKey(s))
	if err != nil || record.Notice != "submitted" || string(output) != "saved execution output" {
		t.Fatalf("terminal history not saved first: %+v %q %v", record, output, err)
	}
	retained := readScheduledResultState(t, a, s.Key)
	if retained.Active == nil || retained.Active.Stage != "delivery" || retained.Last.Notice != "pending" {
		t.Fatalf("fault unexpectedly committed ledger: %+v", retained)
	}
	if _, ok := a.pending[key]; !ok || !a.inFlight[key] {
		t.Fatal("failed completion lost its retry ownership")
	}
	a.coordinator.store.BeforePersist = nil
	if err = a.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	next := readScheduledResultState(t, a, s.Key)
	record, output, err = a.history().Read(t.Context(), scheduleRunKey(s))
	if err != nil || next.Active != nil || next.Last == nil || next.Last.Notice != "submitted" || record.Notice != next.Last.Notice || record.Outcome != next.Last.Outcome || string(output) != "saved execution output" {
		t.Fatalf("history/ledger diverged after retry: %+v %+v %v", next, record, err)
	}
	if len(a.pending) != 0 || a.inFlight[key] || len(a.jobs) != 0 || notices.admissions != 1 || len(fake.Runs()) != 0 {
		t.Fatal("completion retry replayed or retained work")
	}
}

func TestScheduleResultRecoveryAdoptsAlreadyFinalizedNotice(t *testing.T) {
	a, fake, s, notices := scheduledDeliveryFixture(t)
	if err := a.history().FinalizeNotice(t.Context(), scheduleRunKey(s), s.Active.Revision, "submitted"); err != nil {
		t.Fatal(err)
	}
	if err := a.recover(t.Context()); err != nil {
		t.Fatalf("terminal notice recovery conflicted with prior finalization: %v", err)
	}
	next := readScheduledResultState(t, a, s.Key)
	record, output, err := a.history().Read(t.Context(), scheduleRunKey(s))
	if err != nil || next.Active != nil || next.Last == nil || next.Last.Notice != "submitted" || next.Last.Outcome != "wake" || record.Notice != next.Last.Notice || string(output) != "saved execution output" {
		t.Fatalf("terminal evidence not adopted: %+v %+v %v", next, record, err)
	}
	if len(fake.Runs()) != 0 || notices.admissions != 0 || len(a.jobs) != 0 {
		t.Fatal("recovery replayed execution or notice")
	}
	// A second restart must observe the same terminal state.
	if err = a.recover(t.Context()); err != nil {
		t.Fatal(err)
	}
	again := readScheduledResultState(t, a, s.Key)
	if again.Active != nil || again.Last.Notice != "submitted" {
		t.Fatalf("second recovery changed evidence: %+v", again)
	}
}

func TestScheduleResultMismatchedCompletionRefusesAdoption(t *testing.T) {
	for _, field := range []string{"source"} {
		t.Run(field, func(t *testing.T) {
			a, fake, s, notices := scheduledResultFixture(t)
			record := scheduledResultRecord(s, a.coordinator.now(), "wake")
			if field == "source" {
				record.Source = "manual"
			} else {
				record.ScheduledAt = record.ScheduledAt.Add(time.Second)
			}
			saveScheduledResult(t, a, record)
			key := s.Key.ID()
			a.inFlight[key] = true
			a.results <- scheduleJobResult{job: scheduleJob{state: s}}
			if err := a.step(t.Context()); err == nil {
				t.Fatal("mismatched completion adopted")
			}
			next := readScheduledResultState(t, a, s.Key)
			if next.Active == nil || next.Active.Stage != "running" || next.Active.ID != s.Active.ID || next.Last != nil {
				t.Fatalf("mismatched evidence changed authority: %+v", next)
			}
			if _, ok := a.pending[key]; !ok || !a.inFlight[key] {
				t.Fatal("refused completion lost retry ownership")
			}
			if len(fake.Runs()) != 0 || notices.admissions != 0 || len(a.jobs) != 0 {
				t.Fatal("mismatched completion replayed effects")
			}
		})
	}
}

func TestScheduleResultSavedPendingCancelledBeforeRecovery(t *testing.T) {
	for _, change := range []string{"stopping", "revision"} {
		t.Run(change, func(t *testing.T) {
			a, fake, running, notices := scheduledResultFixture(t)
			record := scheduledResultRecord(running, a.coordinator.now(), "wake")
			saveScheduledResult(t, a, record)
			if err := a.coordinator.store.Update(t.Context(), func(l *couchschedule.Ledger) error {
				state := l.Tasks[running.Key.ID()]
				event := couchschedule.Event{Kind: couchschedule.CancelRequested, At: a.coordinator.now(), Attempt: state.Active.ID, Revision: state.Active.Revision}
				if change == "revision" {
					event.Kind = couchschedule.CatalogChanged
					event.Location = time.UTC
					event.Declaration = &couchschedule.Declaration{Key: state.Key, Revision: state.Revision + "-changed", Spec: state.Spec, Allowed: true}
				}
				state, _ = couchschedule.Advance(state, event)
				if state.Active == nil || state.Active.Stage != "stopping" {
					return errors.New("cancellation did not retain stopping authority")
				}
				l.Tasks[state.Key.ID()] = state
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			a.inFlight[running.Key.ID()] = true
			a.results <- scheduleJobResult{job: scheduleJob{state: running}, err: errors.New("prior ledger completion failed")}
			if err := a.step(t.Context()); err != nil {
				t.Fatal(err)
			}
			next := readScheduledResultState(t, a, running.Key)
			got, output, err := a.history().Read(t.Context(), record.Key)
			if err != nil || next.Active != nil || next.Last == nil || next.Last.Notice != "dropped: task cancelled" || got.Notice != next.Last.Notice || got.Outcome != "wake" || string(output) != "saved execution output" {
				t.Fatalf("cancelled saved result regained notice authority: %+v %+v %v", next, got, err)
			}
			if len(a.jobs) != 0 || len(a.pending) != 0 || len(fake.Runs()) != 0 || notices.admissions != 0 {
				t.Fatal("cancelled recovery replayed or retained effects")
			}
		})
	}
}

func TestScheduleResultCanonicalTimesAdoptedWithoutReplay(t *testing.T) {
	a, fake, s, notices := scheduledResultFixture(t)
	record := scheduledResultRecord(s, a.coordinator.now().Add(3*time.Second), "quiet")
	record.ScheduledAt = record.ScheduledAt.Add(time.Second)
	record.StartedAt = record.StartedAt.Add(2 * time.Second)
	saveScheduledResult(t, a, record)
	a.inFlight[s.Key.ID()] = true
	a.results <- scheduleJobResult{job: scheduleJob{state: s}}
	if err := a.step(t.Context()); err != nil {
		t.Fatal("canonical occurrence rejected", err)
	}
	next := readScheduledResultState(t, a, s.Key)
	if next.Active != nil || next.Last == nil || !next.Last.DueAt.Equal(record.ScheduledAt) || !next.Last.StartedAt.Equal(record.StartedAt) || !next.Last.EndedAt.Equal(record.EndedAt) {
		t.Fatalf("canonical facts not adopted: %+v", next)
	}
	if len(fake.Runs()) != 0 || notices.admissions != 0 || len(a.jobs) != 0 {
		t.Fatal("adoption replayed effects")
	}
}

func TestScheduleUnknownPublicationRetryPreservesDurableFacts(t *testing.T) {
	a, _, s, _ := scheduledResultFixture(t)
	blocked := filepath.Join(a.coordinator.store.Dir, s.Key.HubID())
	commits := 0
	a.coordinator.store.BeforePersist = func() error {
		commits++
		if commits == 1 {
			return os.WriteFile(blocked, []byte("history temporarily unavailable"), 0600)
		}
		return nil
	}
	result := scheduleJobResult{job: scheduleJob{state: s}}
	if err := a.reconcileExecution(t.Context(), result); err != nil {
		t.Fatal("task history outage became a global failure", err)
	}
	retained := readScheduledResultState(t, a, s.Key)
	if retained.Allowed || retained.Refusal == "" || retained.Active == nil || retained.Active.Stage != "uncertain" || retained.Last == nil || retained.Last.Outcome != "unknown" {
		t.Fatalf("classification not durable before history repair: %+v", retained)
	}
	a.coordinator.store.BeforePersist = nil
	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	retry := a.coordinator.now().Add(time.Second)
	a.coordinator.now = func() time.Time { return retry }
	if err := a.reconcileExecution(t.Context(), result); err != nil {
		t.Fatal(err)
	}
	next := readScheduledResultState(t, a, s.Key)
	record, _, err := a.history().Read(t.Context(), scheduleRunKey(next))
	if err != nil || !next.Last.EndedAt.Equal(retained.Last.EndedAt) || !record.EndedAt.Equal(retained.Last.EndedAt) || record.DeclarationRevision != retained.Last.Revision {
		t.Fatalf("retry replaced sole durable unknown facts: before=%+v after=%+v record=%+v err=%v", retained.Last, next.Last, record, err)
	}
}
