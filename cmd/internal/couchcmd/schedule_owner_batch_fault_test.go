package couchcmd

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/durablefile"
)

func TestScheduleOwnerReceiptBatchRepairsTaskQuarantine(t *testing.T) {
	a, _, state, notices := scheduledDeliveryFixture(t)
	failing := true
	fault := errors.New("task directory sync unavailable")
	taskDir := filepath.Join(a.coordinator.runs.Namespace, "schedule", state.Key.HubID(), state.Key.Task)
	a.coordinator.runs.SyncDirectory = func(path string) error {
		if failing && (path == taskDir || strings.HasPrefix(path, taskDir+string(filepath.Separator))) {
			return fault
		}
		return durablefile.SyncDirectory(path)
	}
	owner := newScheduleOwner(a.coordinator)
	owner.configure(scheduleExecutionWorker{}, notices)
	owner.recovered = true
	if err := owner.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer owner.close()
	before, err := owner.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	state = before.Tasks[state.Key.ID()]
	clock := newSchedulerClockFake(a.coordinator.now())
	for i := 0; i < 2; i++ {
		if err := owner.tick(t.Context(), clock); err != nil {
			t.Fatal(err)
		}
		if err := owner.refresh(t.Context()); err != nil {
			t.Fatal(err)
		}
		l, err := owner.state.ViewDurable(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		got := l.Tasks[state.Key.ID()]
		if got.Allowed || !strings.Contains(got.Refusal, "history") || !reflect.DeepEqual(got.Active, state.Active) || !reflect.DeepEqual(got.Last, state.Last) {
			t.Fatalf("failed sync consumed authority: %+v", got)
		}
	}
	failing = false
	if err := owner.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := owner.tick(t.Context(), clock); err != nil {
		t.Fatal(err)
	}
	l, err := owner.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := l.Tasks[state.Key.ID()]
	if got.Active != nil || got.Last == nil || got.Last.Notice != "submitted" {
		t.Fatalf("repaired receipt stranded: %+v", got)
	}
	if notices.admissions != 0 {
		t.Fatal("repair resent notice")
	}
}

func TestScheduleOwnerReceiptBatchCanonicalWinsAfterFailedCommit(t *testing.T) {
	a, _, state, notices := scheduledDeliveryFixture(t)
	failing := true
	fault := errors.New("ledger unavailable")
	a.coordinator.store.BeforePersist = func() error {
		if failing {
			return fault
		}
		return nil
	}
	owner := newScheduleOwner(a.coordinator)
	owner.configure(scheduleExecutionWorker{}, notices)
	owner.recovered = true
	if err := owner.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer owner.close()
	before, err := owner.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	state = before.Tasks[state.Key.ID()]
	clock := newSchedulerClockFake(a.coordinator.now())
	for i := 0; i < 2; i++ {
		if err := owner.tick(t.Context(), clock); !errors.Is(err, fault) {
			t.Fatalf("ledger fault not surfaced: %v", err)
		}
		record, output, err := a.history().Read(t.Context(), scheduleRunKey(state))
		if err != nil || record.Notice != "submitted" || string(output) != "saved execution output" {
			t.Fatalf("terminal facts not durable before commit: %+v %q %v", record, output, err)
		}
		l, err := owner.state.ViewDurable(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got := l.Tasks[state.Key.ID()]; !reflect.DeepEqual(got.Active, state.Active) || !reflect.DeepEqual(got.Last, state.Last) {
			t.Fatal("failed ledger write acknowledged")
		}
	}
	failing = false
	// An unavailable receipt now suggests uncertainty; canonical submitted history
	// from the first attempt must win instead of requesting another finalization.
	owner.loop.port = lostScheduleReceipt{}
	if err := owner.tick(t.Context(), clock); err != nil {
		t.Fatal(err)
	}
	l, err := owner.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := l.Tasks[state.Key.ID()]
	if got.Active != nil || got.Last == nil || got.Last.Notice != "submitted" || !got.NextDue.Equal(state.NextDue) {
		t.Fatalf("canonical terminal facts lost: %+v", got)
	}
	if notices.admissions != 0 {
		t.Fatal("retry replayed notice")
	}
}

type lostScheduleReceipt struct{ offlineScheduleNotices }

func (lostScheduleReceipt) Observe(string) (string, bool, bool) { return "", false, false }
