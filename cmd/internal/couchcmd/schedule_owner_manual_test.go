package couchcmd

import (
	"bytes"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"reflect"
	"testing"
)

func ownerManualFixture(t *testing.T) (*scheduleOwner, couchschedule.ManualIntent) {
	t.Helper()
	a, _, state, _ := scheduledResultFixture(t)
	if err := a.coordinator.store.Update(t.Context(), func(l *couchschedule.Ledger) error {
		s := l.Tasks[state.Key.ID()]
		s.Active = nil
		s.Last = nil
		l.Tasks[state.Key.ID()] = s
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	o := newScheduleOwner(a.coordinator)
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { o.close() })
	return o, couchschedule.ManualIntent{ID: "manual-1", Hub: state.Key.Hub, Task: state.Key.Task, Args: []string{"--sample"}}
}
func TestScheduleOwnerManualReservationDuplicateAndConflict(t *testing.T) {
	o, m := ownerManualFixture(t)
	status, err := o.submitManual(t.Context(), m)
	if err != nil || status.ID != m.ID || status.Stage != "reserved" {
		t.Fatalf("status=%+v error=%v", status, err)
	}
	ledger, err := o.coordinator.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	s := ledger.Tasks[(couchschedule.TaskKey{Hub: m.Hub, Task: m.Task}).ID()]
	if s.Active == nil || s.Active.ManualIntentDigest != m.Digest() {
		t.Fatalf("undurable reservation %+v", s)
	}
	if _, err := o.submitManual(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	changed := m
	changed.Args = []string{"different"}
	if _, err := o.submitManual(t.Context(), changed); err == nil {
		t.Fatal("accepted conflicting duplicate")
	}
	if _, err := o.observeManual(t.Context(), m.Hub, m.Task, m.ID, 0); err != nil {
		t.Fatal(err)
	}
}
func TestScheduleOwnerManualFailedCommitDoesNotBindDispatch(t *testing.T) {
	o, m := ownerManualFixture(t)
	if err := o.close(); err != nil {
		t.Fatal(err)
	}
	o.coordinator.store.BeforePersist = func() error { return errors.New("commit failed") }
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := o.submitManual(t.Context(), m); err == nil {
		t.Fatal("failed commit acknowledged")
	}
	if len(o.intents) != 0 {
		t.Fatal("failed reservation bound executable args")
	}
}
func TestScheduleOwnerManualCancellationIsExactAndIdempotent(t *testing.T) {
	o, m := ownerManualFixture(t)
	if _, err := o.submitManual(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	if _, err := o.cancelManual(t.Context(), m.Hub, m.Task, "other"); err == nil {
		t.Fatal("foreign ID accepted")
	}
	if _, err := o.cancelManual(t.Context(), m.Hub, m.Task, m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := o.cancelManual(t.Context(), m.Hub, m.Task, m.ID); err != nil {
		t.Fatal(err)
	}
}
func TestScheduleOwnerManualNativeRetainedDuplicateAndReadOnlyPages(t *testing.T) {
	o, m := ownerManualFixture(t)
	key := couchschedule.TaskKey{Hub: m.Hub, Task: m.Task}
	exit := 10
	record := operationalrun.Record{Version: 1, Key: operationalrun.RunKey{Hub: key.HubID(), Task: m.Task, ID: m.ID}, DeclarationRevision: "old-revision", Source: "manual", StartedAt: o.coordinator.now(), EndedAt: o.coordinator.now(), Exit: &exit, Outcome: "wake", Notice: "manual", ManualIntentDigest: m.Digest()}
	output := bytes.Repeat([]byte("x"), 40000)
	if err := o.coordinator.runs.Publish(t.Context(), record.Key, record, output); err != nil {
		t.Fatal(err)
	}
	before, err := o.coordinator.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, offset := range []int{0, couchschedule.MaxManualOutputPage, 40000} {
		status, err := o.observeManual(t.Context(), m.Hub, m.Task, m.ID, offset)
		if err != nil || status.Stage != "done" || status.Total != len(output) || status.Offset != offset || len(status.Output) > couchschedule.MaxManualOutputPage {
			t.Fatalf("page=%+v error=%v", status, err)
		}
	}
	if status, err := o.submitManual(t.Context(), m); err != nil || status.ID != m.ID || status.Stage != "done" {
		t.Fatalf("duplicate=%+v %v", status, err)
	}
	after, err := o.coordinator.store.View(t.Context())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("inspection/oldduplicate changed authority: %v", err)
	}
	if len(o.intents) != 0 {
		t.Fatal("terminal duplicate enqueued execution")
	}
	legacy := m
	legacy.ID = "legacy"
	record.Key.ID = legacy.ID
	record.ManualIntentDigest = ""
	if err := o.coordinator.runs.Publish(t.Context(), record.Key, record, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := o.submitManual(t.Context(), legacy); err == nil {
		t.Fatal("legacy unknown intent adopted arbitrary args")
	}
}
