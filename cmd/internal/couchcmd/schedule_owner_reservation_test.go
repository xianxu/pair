package couchcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

type ownerReservationFixture struct {
	owner  *scheduleOwner
	fake   *couchschedule.FakeTaskRuntime
	state  couchschedule.TaskState
	now    time.Time
	before func() error
}

func newOwnerReservationFixture(t *testing.T, kind string) *ownerReservationFixture {
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
	raw := fmt.Sprintf(`{"version":1,"cron":"0 * * * *","timeout":"10s","script_run":%t}`, kind != "notify")
	for name, body := range map[string]string{"schedule": raw, "main.go": "package main\nfunc main(){}\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	f := &ownerReservationFixture{now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC), fake: &couchschedule.FakeTaskRuntime{}}
	c := newScheduleCoordinator(filepath.Join(base, "namespace"), []string{root}, time.UTC, func() time.Time { return f.now })
	c.store.BeforePersist = func() error {
		if f.before != nil {
			return f.before()
		}
		return nil
	}
	o := newScheduleOwner(c)
	f.owner = o
	o.configure(scheduleExecutionWorker{namespace: c.runs.Namespace, executor: f.fake, now: c.now}, &resultScheduleNotices{})
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { o.close() })
	if err := o.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	key := couchschedule.TaskKey{Hub: root, Task: "report"}
	if kind == "manual" {
		if _, err := o.submitManual(t.Context(), couchschedule.ManualIntent{ID: "known-undispatched-manual", Hub: root, Task: "report", Args: []string{"bound-argv"}}); err != nil {
			t.Fatal(err)
		}
	} else {
		l, err := o.state.ViewDurable(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		f.now = l.Tasks[key.ID()].NextDue
		if _, err := o.apply(t.Context(), key, couchschedule.Event{Kind: couchschedule.ClockObserved, At: f.now, RunID: "known-undispatched-scheduled", HubOnline: true, Location: time.UTC}, nil); err != nil {
			t.Fatal(err)
		}
	}
	l, err := o.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	f.state = l.Tasks[key.ID()]
	if f.state.Active == nil || f.state.Active.Stage != "reserved" || f.state.Active.ControllerPID != 0 {
		t.Fatalf("fixture did not retain undispatched reservation: %+v", f.state)
	}
	// These are live owner turns, not startup recovery of ambiguous old work.
	o.recovered = true
	// Occupy both independently bounded pools. The target reservation itself has
	// never entered either pool, so expiry cannot depend on a worker returning.
	o.loop.active = map[string]bool{"occupied-script-1": false, "occupied-script-2": false, "occupied-notice-1": true, "occupied-notice-2": true}
	return f
}
func (f *ownerReservationFixture) edit(t *testing.T, edit string) {
	t.Helper()
	path := filepath.Join(f.state.Key.Hub, "scripts", "scheduled", "report", "schedule")
	switch edit {
	case "revision":
		raw := fmt.Sprintf(`{"version":1,"cron":"*/2 * * * *","timeout":"10s","script_run":%t}`, f.state.Spec.ScriptRun)
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	case "removal":
		if err := os.RemoveAll(filepath.Dir(path)); err != nil {
			t.Fatal(err)
		}
	case "owner-policy":
		raw := fmt.Sprintf(`{"version":1,"cron":"0 * * * *","timeout":"10s","script_run":%t,"scope":"team","owner":"absent-owner"}`, f.state.Spec.ScriptRun)
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.owner.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if edit != "none" && f.owner.busy[f.state.Key.ID()] == "" {
		t.Fatalf("%s did not hold catalogue edit busy", edit)
	}
}
func (f *ownerReservationFixture) assertNoEffects(t *testing.T) {
	t.Helper()
	if len(f.owner.loop.execution) != 0 || len(f.owner.loop.notices) != 0 || len(f.owner.loop.active) != 4 || len(f.fake.Runs()) != 0 {
		t.Fatalf("reserved work escaped capacity: executions=%d notices=%d active=%v runs=%+v", len(f.owner.loop.execution), len(f.owner.loop.notices), f.owner.loop.active, f.fake.Runs())
	}
}
func (f *ownerReservationFixture) view(t *testing.T) couchschedule.TaskState {
	t.Helper()
	l, err := f.owner.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return l.Tasks[f.state.Key.ID()]
}
func (f *ownerReservationFixture) cancel(t *testing.T) {
	t.Helper()
	if f.state.Active.Source == "manual" {
		if _, err := f.owner.cancelManual(t.Context(), f.state.Key.Hub, f.state.Key.Task, f.state.Active.ID); err != nil {
			t.Fatal(err)
		}
	} else {
		if _, err := f.owner.apply(t.Context(), f.state.Key, couchschedule.Event{Kind: couchschedule.CancelRequested, At: f.now, Attempt: f.state.Active.ID, Revision: f.state.Active.Revision}, nil); err != nil {
			t.Fatal(err)
		}
	}
}
func (f *ownerReservationFixture) assertSettled(t *testing.T, outcome string) {
	t.Helper()
	got := f.view(t)
	if got.Active != nil {
		t.Fatalf("known-undispatched reservation stranded after deadline/cancel: %+v", got)
	}
	if outcome != "" {
		if got.Last == nil || got.Last.ID != f.state.Active.ID || got.Last.Outcome != outcome || got.Last.Revision != f.state.Active.Revision {
			t.Fatalf("lost terminal reservation facts: %+v", got)
		}
		record, output, err := f.owner.coordinator.runs.ReadDurable(t.Context(), operationalrun.RunKey{Hub: f.state.Key.HubID(), Task: f.state.Key.Task, ID: f.state.Active.ID})
		if err != nil || record.Outcome != outcome || record.DeclarationRevision != f.state.Active.Revision || len(output) != 0 {
			t.Fatalf("reservation terminal history: %+v %q %v", record, output, err)
		}
	}
	f.assertNoEffects(t)
}

func TestScheduleOwnerReservationExpiryAndCancellationMatrix(t *testing.T) {
	for _, kind := range []string{"manual", "script", "notify"} {
		for _, edit := range []string{"none", "revision", "removal", "owner-policy"} {
			for _, sequence := range []string{"expiry", "cancel-before-expiry", "expiry-then-cancel", "cancel-at-expiry"} {
				t.Run(kind+"/"+edit+"/"+sequence, func(t *testing.T) {
					f := newOwnerReservationFixture(t, kind)
					f.edit(t, edit)
					outcome := "timeout"
					if sequence == "cancel-before-expiry" || sequence == "cancel-at-expiry" {
						f.now = f.state.Active.Deadline
						if sequence == "cancel-before-expiry" {
							f.now = f.now.Add(-time.Nanosecond)
						}
						f.cancel(t)
						outcome = ""
						if kind == "manual" {
							outcome = "cancelled"
						}
					}
					f.now = f.state.Active.Deadline
					if err := f.owner.tick(t.Context(), newSchedulerClockFake(f.now)); err != nil {
						t.Fatal(err)
					}
					if sequence == "expiry-then-cancel" {
						f.cancel(t)
					}
					f.assertSettled(t, outcome)
					// Settlement releases the busy catalogue edit. Rediscovery applies current
					// policy/revision before another tick can admit an old-revision occurrence.
					for i := 0; i < 2; i++ {
						if err := f.owner.refresh(t.Context()); err != nil {
							t.Fatal(err)
						}
						if err := f.owner.tick(t.Context(), newSchedulerClockFake(f.now)); err != nil {
							t.Fatal(err)
						}
						f.assertSettled(t, outcome)
					}
					got := f.view(t)
					switch edit {
					case "revision":
						if got.Revision == f.state.Revision {
							t.Fatal("settlement left revision edit busy")
						}
					case "removal":
						if got.RemovedAt.IsZero() || got.Allowed {
							t.Fatalf("removal did not apply: %+v", got)
						}
					case "owner-policy":
						if got.Allowed {
							t.Fatal("refused team-owner policy did not apply")
						}
					}
				})
			}
		}
	}
}

func TestScheduleOwnerExpiredReservationSettlesDespiteQuarantine(t *testing.T) {
	for _, kind := range []string{"manual", "script", "notify"} {
		t.Run(kind, func(t *testing.T) {
			f := newOwnerReservationFixture(t, kind)
			if err := f.owner.state.Update(t.Context(), func(l *couchschedule.Ledger) error {
				s := l.Tasks[f.state.Key.ID()]
				s.Allowed = false
				s.Refusal = "task report history prior diagnostic unavailable"
				l.Tasks[f.state.Key.ID()] = s
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			f.now = f.state.Active.Deadline
			if err := f.owner.tick(t.Context(), newSchedulerClockFake(f.now)); err != nil {
				t.Fatal(err)
			}
			f.assertSettled(t, "timeout")
			if err := f.owner.refresh(t.Context()); err != nil {
				t.Fatal(err)
			}
			if !f.view(t).Allowed {
				t.Fatal("durable terminal history did not release diagnostic quarantine")
			}
		})
	}
}

func TestScheduleOwnerExpiredReservationHistoryBeforeAuthorityRetry(t *testing.T) {
	for _, kind := range []string{"manual", "script", "notify"} {
		t.Run(kind, func(t *testing.T) {
			f := newOwnerReservationFixture(t, kind)
			f.edit(t, "revision")
			fault := errors.New("reservation terminal authority unavailable")
			f.now = f.state.Active.Deadline
			key := operationalrun.RunKey{Hub: f.state.Key.HubID(), Task: f.state.Key.Task, ID: f.state.Active.ID}
			guardCalls := 0
			f.before = func() error {
				guardCalls++
				record, output, err := f.owner.coordinator.runs.ReadDurable(t.Context(), key)
				if err != nil || record.Outcome != "timeout" || len(output) != 0 {
					t.Errorf("ledger ack attempted before timeout history: %+v %q %v", record, output, err)
				}
				return fault
			}
			var canonical []byte
			for i := 0; i < 3; i++ {
				if err := f.owner.tick(t.Context(), newSchedulerClockFake(f.now)); !errors.Is(err, fault) {
					t.Fatalf("expiry failed to surface terminal authority error: %v", err)
				}
				got := f.view(t)
				if !reflect.DeepEqual(got.Active, f.state.Active) || got.Last != nil {
					t.Fatalf("failed commit consumed reservation: %+v", got)
				}
				path := filepath.Join(f.owner.coordinator.runs.Namespace, "schedule", key.Hub, key.Task, "runs", key.ID, "record.json")
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if i == 0 {
					canonical = raw
				} else if !bytes.Equal(canonical, raw) {
					t.Fatal("failed-ledger retry changed canonical timeout record")
				}
				f.assertNoEffects(t)
			}
			if guardCalls < 3 {
				t.Fatal("repeated expiry did not retry authority")
			}
			f.before = nil
			if err := f.owner.tick(t.Context(), newSchedulerClockFake(f.now)); err != nil {
				t.Fatal(err)
			}
			f.assertSettled(t, "timeout")
			if err := f.owner.refresh(t.Context()); err != nil {
				t.Fatal(err)
			}
			if f.view(t).Revision == f.state.Revision {
				t.Fatal("recovered terminal reservation still blocks edit")
			}
		})
	}
}

func TestScheduleOwnerExpiredReservationCanonicalCompletionWins(t *testing.T) {
	for _, kind := range []string{"manual", "script", "notify"} {
		for _, cancel := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/cancel=%t", kind, cancel), func(t *testing.T) {
				f := newOwnerReservationFixture(t, kind)
				f.edit(t, "revision")
				key := operationalrun.RunKey{Hub: f.state.Key.HubID(), Task: f.state.Key.Task, ID: f.state.Active.ID}
				code := 0
				record := operationalrun.Record{Version: 1, Key: key, DeclarationRevision: f.state.Active.Revision, Source: f.state.Active.Source, ScheduledAt: f.state.Active.DueAt, StartedAt: f.state.Active.StartedAt, EndedAt: f.now.Add(time.Second), Exit: &code, Outcome: "quiet", Notice: "none", ManualIntentDigest: f.state.Active.ManualIntentDigest, Unknown: map[string]json.RawMessage{"future_canonical": json.RawMessage(`{"complete":true}`)}}
				if kind == "manual" {
					record.Notice = "manual"
				}
				output := []byte("already durable native output")
				if kind == "notify" {
					record.Outcome, record.Notice, record.Exit = "completed", "submitted", nil
					output = nil
				}
				if err := f.owner.coordinator.runs.Publish(t.Context(), key, record, output); err != nil {
					t.Fatal(err)
				}
				record, _, err := f.owner.coordinator.runs.ReadDurable(t.Context(), key)
				if err != nil {
					t.Fatal(err)
				}
				f.now = f.state.Active.Deadline
				if cancel {
					f.cancel(t)
				}
				if err := f.owner.tick(t.Context(), newSchedulerClockFake(f.now)); err != nil {
					t.Fatal(err)
				}
				got := f.view(t)
				if got.Active != nil || got.Last == nil || got.Last.Outcome != record.Outcome || got.Last.ID != key.ID {
					t.Fatalf("expiry/cancel displaced completed canonical facts: %+v", got)
				}
				saved, bytesOut, err := f.owner.coordinator.runs.ReadDurable(t.Context(), key)
				if err != nil || !reflect.DeepEqual(record, saved) || !bytes.Equal(output, bytesOut) {
					t.Fatalf("canonical completion changed: %+v %q %v", saved, bytesOut, err)
				}
				f.assertNoEffects(t)
			})
		}
	}
}

func TestScheduleOwnerExpiredUnboundManualReservationStaysUnknown(t *testing.T) {
	for _, edit := range []string{"none", "revision", "removal", "owner-policy"} {
		for _, binding := range []string{"missing", "mismatched"} {
			t.Run(edit+"/"+binding, func(t *testing.T) {
				f := newOwnerReservationFixture(t, "manual")
				f.edit(t, edit)
				delete(f.owner.intents, f.state.Key.ID())
				if binding == "mismatched" {
					f.owner.intents[f.state.Key.ID()] = couchschedule.ManualIntent{ID: f.state.Active.ID, Hub: f.state.Key.Hub, Task: f.state.Key.Task, Args: []string{"different-argv"}}
				}
				f.now = f.state.Active.Deadline
				for i := 0; i < 2; i++ {
					if err := f.owner.tick(t.Context(), newSchedulerClockFake(f.now)); err != nil {
						t.Fatal(err)
					}
					got := f.view(t)
					if got.Active == nil || got.Active.Stage != "uncertain" || got.Last == nil || got.Last.Outcome != "unknown" || got.Last.ID != f.state.Active.ID || got.Last.ManualIntentDigest != f.state.Active.ManualIntentDigest {
						t.Fatalf("unknown submit classified as known-unstarted timeout: %+v", got)
					}
					f.assertNoEffects(t)
				}
				record, output, err := f.owner.coordinator.runs.ReadDurable(t.Context(), operationalrun.RunKey{Hub: f.state.Key.HubID(), Task: f.state.Key.Task, ID: f.state.Active.ID})
				if err != nil || record.Outcome != "unknown" || len(output) != 0 {
					t.Fatalf("missing unknown diagnostic: %+v %q %v", record, output, err)
				}
			})
		}
	}
}

func TestScheduleOwnerExpiredReservationStillOwnedByWorkerOrFacts(t *testing.T) {
	for _, custody := range []string{"admitted-worker", "retained-facts"} {
		t.Run(custody, func(t *testing.T) {
			f := newOwnerReservationFixture(t, "manual")
			// Free precisely one execution slot and admit the target through the real
			// bounded handoff. The other script and both notice workers stay occupied.
			delete(f.owner.loop.active, "occupied-script-1")
			if !f.owner.launchExecution(t.Context(), f.state, []string{"bound-argv"}) {
				t.Fatal("target did not acquire the available execution slot")
			}
			if len(f.owner.loop.active) != 4 || len(f.owner.loop.execution) != 1 {
				t.Fatal("fixture exceeded or lost bounded worker capacity")
			}
			job := <-f.owner.loop.execution
			defer job.cancel()
			if custody == "retained-facts" {
				f.owner.collect(scheduleOwnerResult{job: job, facts: scheduleExecutionFacts{ended: f.state.Active.Deadline, result: couchschedule.ExecutionResult{Outcome: "uncertain", Exit: -1, Output: []byte("retained worker facts")}}})
			}
			f.edit(t, "revision")
			f.now = f.state.Active.Deadline
			for i := 0; i < 2; i++ {
				if err := f.owner.tick(t.Context(), newSchedulerClockFake(f.now)); err != nil {
					t.Fatal(err)
				}
				got := f.view(t)
				if !reflect.DeepEqual(got.Active, f.state.Active) || got.Last != nil {
					t.Fatalf("expiry stole %s authority: %+v", custody, got)
				}
				if len(f.owner.loop.execution) != 0 || len(f.owner.loop.notices) != 0 || len(f.fake.Runs()) != 0 {
					t.Fatal("ownership check admitted another effect")
				}
				if custody == "admitted-worker" {
					if notice, ok := f.owner.loop.active[f.state.Key.ID()]; !ok || notice || len(f.owner.loop.active) != 4 {
						t.Fatal("expiry released admitted execution ownership")
					}
				} else {
					pending, ok := f.owner.loop.pending[f.state.Key.ID()]
					if !ok || !bytes.Equal(pending.facts.result.Output, []byte("retained worker facts")) || pending.job.state.Active.ID != f.state.Active.ID || len(f.owner.loop.active) != 3 {
						t.Fatal("expiry discarded retained completion facts")
					}
				}
			}
			key := operationalrun.RunKey{Hub: f.state.Key.HubID(), Task: f.state.Key.Task, ID: f.state.Active.ID}
			if _, _, err := f.owner.coordinator.runs.Read(t.Context(), key); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("known-unstarted timeout manufactured under %s custody: %v", custody, err)
			}
		})
	}
}
