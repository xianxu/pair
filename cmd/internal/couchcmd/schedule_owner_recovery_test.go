package couchcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

func ownerRecoveryFixture(t *testing.T, source, stage string) (*scheduleOwner, *couchschedule.FakeTaskRuntime, couchschedule.TaskState) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "hub")
	dir := filepath.Join(root, "scripts", "scheduled", "report")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for name, raw := range map[string]string{"schedule": `{"version":1,"cron":"* * * * *","timeout":"1m","script_run":true}`, "main.go": "package main\nfunc main(){}\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	c := newScheduleCoordinator(filepath.Join(base, "namespace"), []string{root}, time.UTC, func() time.Time { return now })
	fake := &couchschedule.FakeTaskRuntime{}
	o := reopenRecoveryOwner(t, c, fake)
	if err := o.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	key := couchschedule.TaskKey{Hub: root, Task: "report"}
	if source == "manual" {
		if _, err := o.submitManual(t.Context(), couchschedule.ManualIntent{ID: "restart-manual", Hub: root, Task: "report", Args: []string{"private-argv"}}); err != nil {
			t.Fatal(err)
		}
	} else {
		l, err := o.state.ViewDurable(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		now = l.Tasks[key.ID()].NextDue
		if _, err := o.apply(t.Context(), key, couchschedule.Event{Kind: couchschedule.ClockObserved, At: now, RunID: "restart-scheduled", HubOnline: true, Location: time.UTC}, nil); err != nil {
			t.Fatal(err)
		}
	}
	l, err := o.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	state := l.Tasks[key.ID()]
	if stage != "reserved" {
		declaration, err := manualDeclaration(t.Context(), root, "report")
		if err != nil {
			t.Fatal(err)
		}
		if err := o.authorize(t.Context(), state, couchschedule.OwnerIdentity{PID: 9191, Start: "departed-controller"}, declaration); err != nil {
			t.Fatal(err)
		}
		if stage == "stopping" {
			if _, err := o.apply(t.Context(), key, couchschedule.Event{Kind: couchschedule.CancelRequested, At: now, Attempt: state.Active.ID, Revision: state.Active.Revision}, nil); err != nil {
				t.Fatal(err)
			}
		}
	}
	l, err = o.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	state = l.Tasks[key.ID()]
	if state.Active == nil || state.Active.Stage != stage {
		t.Fatalf("fixture phase=%+v want %s", state.Active, stage)
	}
	return o, fake, state
}

func reopenRecoveryOwner(t *testing.T, c *scheduleCoordinator, fake *couchschedule.FakeTaskRuntime) *scheduleOwner {
	t.Helper()
	o := newScheduleOwner(c)
	o.configure(scheduleExecutionWorker{namespace: c.runs.Namespace, executor: fake, now: c.now}, &resultScheduleNotices{})
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { o.close() })
	return o
}

func assertRecoveryNoEffects(t *testing.T, o *scheduleOwner, fake *couchschedule.FakeTaskRuntime) {
	t.Helper()
	if len(o.loop.execution) != 0 || len(o.loop.notices) != 0 || len(o.loop.active) != 0 || len(fake.Runs()) != 0 {
		t.Fatalf("recovery replayed work: executions=%d notices=%d active=%v runs=%+v", len(o.loop.execution), len(o.loop.notices), o.loop.active, fake.Runs())
	}
}

func TestScheduleOwnerRestartUnknownNeverReplays(t *testing.T) {
	for _, source := range []string{"manual", "scheduled"} {
		for _, stage := range []string{"reserved", "running", "stopping"} {
			t.Run(source+"/"+stage, func(t *testing.T) {
				first, fake, state := ownerRecoveryFixture(t, source, stage)
				key := operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: state.Active.ID}
				if _, _, err := first.coordinator.runs.Read(t.Context(), key); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("fixture unexpectedly has native completion: %v", err)
				}
				if err := first.close(); err != nil {
					t.Fatal(err)
				}
				second := reopenRecoveryOwner(t, first.coordinator, fake)
				if len(second.intents) != 0 {
					t.Fatal("restart retained in-memory argv")
				}
				clock := newSchedulerClockFake(first.coordinator.now())
				for i := 0; i < 2; i++ {
					if err := second.refresh(t.Context()); err != nil {
						t.Fatal(err)
					}
					if err := second.tick(t.Context(), clock); err != nil {
						t.Fatal(err)
					}
					l, err := second.state.ViewDurable(t.Context())
					if err != nil {
						t.Fatal(err)
					}
					got := l.Tasks[state.Key.ID()]
					if got.Active == nil || got.Active.Stage != "uncertain" || got.Active.ID != state.Active.ID || got.Last == nil || got.Last.Outcome != "unknown" || got.Last.ID != state.Active.ID || !got.NextDue.Equal(state.NextDue) {
						t.Fatalf("restart lost uncertain authority: %+v", got)
					}
					if got.Active.ManualIntentDigest != state.Active.ManualIntentDigest || got.Last.ManualIntentDigest != state.Active.ManualIntentDigest {
						t.Fatal("restart changed intent identity")
					}
					assertRecoveryNoEffects(t, second, fake)
				}
				record, output, err := second.coordinator.runs.Read(t.Context(), key)
				if err != nil || record.Outcome != "unknown" || record.Source != source || len(output) != 0 {
					t.Fatalf("unknown diagnostic manufactured native facts: %+v %q %v", record, output, err)
				}
				rows, _, _, err := second.coordinator.runs.SweepPage(t.Context(), clock.Now().Add(40*24*time.Hour), operationalrun.Cursor{}, true, 100)
				if err != nil || len(rows) != 1 || rows[0].Eligible {
					t.Fatalf("unknown run not retention protected: %+v %v", rows, err)
				}
				if _, _, err := second.coordinator.runs.Read(t.Context(), key); err != nil {
					t.Fatalf("retention removed unknown evidence: %v", err)
				}
			})
		}
	}
}

func TestScheduleOwnerRecoveryPreservesAdditiveNamespaces(t *testing.T) {
	first, fake, state := ownerRecoveryFixture(t, "scheduled", "running")
	ledgerFields := map[string]json.RawMessage{"future_ledger": json.RawMessage(`{"v":[1,"ledger"]}`)}
	taskFields := map[string]json.RawMessage{"future_task": json.RawMessage(`{"v":[2,"task"]}`)}
	activeFields := map[string]json.RawMessage{"future_active": json.RawMessage(`{"v":[3,"active"]}`)}
	lastFields := map[string]json.RawMessage{"future_last": json.RawMessage(`{"v":[4,"summary"]}`)}
	if err := first.state.Update(t.Context(), func(l *couchschedule.Ledger) error {
		l.Unknown = ledgerFields
		s := l.Tasks[state.Key.ID()]
		s.Unknown = taskFields
		s.Active.Unknown = activeFields
		l.Tasks[state.Key.ID()] = s
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	facts := scheduleExecutionFacts{ended: first.coordinator.now().Add(time.Second), result: couchschedule.ExecutionResult{Exit: 10, Outcome: "wake", Output: []byte{'n', 'a', 't', 'i', 'v', 'e', 0, 0xff}, BuildKey: "canonical-build"}}
	if err := first.complete(t.Context(), state, facts); err != nil {
		t.Fatal(err)
	}
	if err := first.state.Update(t.Context(), func(l *couchschedule.Ledger) error {
		s := l.Tasks[state.Key.ID()]
		s.Last.Unknown = lastFields
		l.Tasks[state.Key.ID()] = s
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Emulate an additive writer from a newer schema. The record/key namespaces
	// are distinct from the ledger summary namespace, even for the same run.
	runKey := operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: state.Active.ID}
	recordPath := filepath.Join(first.coordinator.runs.Namespace, "schedule", runKey.Hub, runKey.Task, "runs", runKey.ID, "record.json")
	raw, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	recordField := json.RawMessage(`{"v":[5,"record"]}`)
	keyField := json.RawMessage(`{"v":[6,"key"]}`)
	object["future_record"] = recordField
	var keyObject map[string]json.RawMessage
	if err := json.Unmarshal(object["key"], &keyObject); err != nil {
		t.Fatal(err)
	}
	keyObject["future_key"] = keyField
	object["key"], err = json.Marshal(keyObject)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(recordPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	l, err := first.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	current := l.Tasks[state.Key.ID()]
	if !reflect.DeepEqual(current.Active.Unknown, activeFields) {
		t.Fatal("completion lost additive active fields")
	}
	for _, event := range []couchschedule.Event{
		{Kind: couchschedule.NoticeFrozen, ReceiptID: "recovery-receipt", NoticeBody: "canonical body", NoticeBinding: json.RawMessage(`{"slot":"hub"}`)},
		{Kind: couchschedule.NoticeIntentDurable, HubOnline: true},
	} {
		event.At = first.coordinator.now()
		event.Attempt = state.Active.ID
		event.Revision = state.Active.Revision
		if _, err := first.apply(t.Context(), state.Key, event, nil); err != nil {
			t.Fatal(err)
		}
	}
	fault := errors.New("authority failed after terminal history")
	if err := first.close(); err != nil {
		t.Fatal(err)
	}
	c := first.coordinator
	c.store.BeforePersist = func() error { return fault }
	first = reopenRecoveryOwner(t, c, fake)
	if _, err := first.apply(t.Context(), state.Key, couchschedule.Event{Kind: couchschedule.NoticeObserved, At: first.coordinator.now(), Attempt: state.Active.ID, Revision: state.Active.Revision, Notice: "submitted", HubOnline: true}, nil); !errors.Is(err, fault) {
		t.Fatalf("missing authority failure: %v", err)
	}
	// History finalizes ahead of authority. Its output and additive record/key
	// bytes must survive both finalization and reopening the owner.
	terminalRaw, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	record, saved, err := first.coordinator.runs.Read(t.Context(), runKey)
	if err != nil || record.Notice != "submitted" || !bytes.Equal(saved, facts.result.Output) {
		t.Fatalf("canonical terminal facts: %+v %q %v", record, saved, err)
	}
	l, err = first.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	current = l.Tasks[state.Key.ID()]
	if current.Active == nil || current.Last.Notice != "pending" || !reflect.DeepEqual(current.Active.Unknown, activeFields) || !reflect.DeepEqual(current.Last.Unknown, lastFields) {
		t.Fatalf("failed commit consumed additive authority: %+v", current)
	}
	if err := first.close(); err != nil {
		t.Fatal(err)
	}
	first.coordinator.store.BeforePersist = nil
	second := reopenRecoveryOwner(t, first.coordinator, fake)
	for i := 0; i < 2; i++ {
		if err := second.refresh(t.Context()); err != nil {
			t.Fatal(err)
		}
		if err := second.tick(t.Context(), newSchedulerClockFake(second.coordinator.now())); err != nil {
			t.Fatal(err)
		}
	}
	l, err = second.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	current = l.Tasks[state.Key.ID()]
	if !reflect.DeepEqual(l.Unknown, ledgerFields) || !reflect.DeepEqual(current.Unknown, taskFields) || current.Active != nil || current.Last == nil || current.Last.Notice != "submitted" || !reflect.DeepEqual(current.Last.Unknown, lastFields) || !current.NextDue.Equal(state.NextDue) {
		t.Fatalf("recovery lost additive ledger facts: ledger=%v state=%+v", l.Unknown, current)
	}
	record, saved, err = second.coordinator.runs.Read(t.Context(), runKey)
	if err != nil || !bytes.Equal(saved, facts.result.Output) || !reflect.DeepEqual(record.Unknown["future_record"], recordField) || !reflect.DeepEqual(record.Key.Unknown["future_key"], keyField) {
		t.Fatalf("recovery lost additive record namespaces: %+v output=%q error=%v", record, saved, err)
	}
	afterRaw, err := os.ReadFile(recordPath)
	if err != nil || !bytes.Equal(terminalRaw, afterRaw) {
		t.Fatalf("owner rewrote immutable terminal record: %v", err)
	}
	assertRecoveryNoEffects(t, second, fake)
	t.Logf("terminal output %d bytes retained across failed authority and reopened owner", len(saved))
}

func TestScheduleOwnerUnboundManualReservationNeverDispatches(t *testing.T) {
	for _, binding := range []string{"missing", "mismatched"} {
		t.Run(binding, func(t *testing.T) {
			first, fake, state := ownerRecoveryFixture(t, "manual", "reserved")
			if err := first.close(); err != nil {
				t.Fatal(err)
			}
			owner := reopenRecoveryOwner(t, first.coordinator, fake)
			// Exercise the ongoing tick check independently of the conservative
			// startup sweep: a reservation can also outlive an uncertain submit.
			owner.recovered = true
			if binding == "mismatched" {
				owner.intents[state.Key.ID()] = couchschedule.ManualIntent{ID: state.Active.ID, Hub: state.Key.Hub, Task: state.Key.Task, Args: []string{"different-argv"}}
				if owner.intents[state.Key.ID()].Digest() == state.Active.ManualIntentDigest {
					t.Fatal("fixture did not change intent digest")
				}
			}
			for i := 0; i < 2; i++ {
				if err := owner.tick(t.Context(), newSchedulerClockFake(owner.coordinator.now())); err != nil {
					t.Fatal(err)
				}
				assertRecoveryNoEffects(t, owner, fake)
			}
			ledger, err := owner.state.ViewDurable(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			got := ledger.Tasks[state.Key.ID()]
			if got.Active == nil || got.Active.Stage != "uncertain" || got.Active.ID != state.Active.ID || got.Active.ManualIntentDigest != state.Active.ManualIntentDigest || got.Last == nil || got.Last.Outcome != "unknown" {
				t.Fatalf("unbound manual reservation lost uncertainty: %+v", got)
			}
		})
	}
}
