package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

func TestScheduleFactWorkerDoesNotWriteAuthorityOrHistory(t *testing.T) {
	runner, fake, reserved, store := reservedExecutionFixture(t)
	fake.Enqueue(couchschedule.ExecutionResult{Exit: 10, Outcome: "wake", Output: []byte("fact output")})
	before, err := os.ReadFile(filepath.Join(store.Dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	worker := scheduleExecutionWorker{namespace: runner.namespace, permitsDir: runner.permitsDir, executor: fake, now: runner.now, withDeadline: runner.withDeadline}
	ready := 0
	facts := worker.execute(t.Context(), reserved, []string{"script-run"}, func(_ context.Context, owner couchschedule.OwnerIdentity, d couchschedule.Declaration) error {
		ready++
		if owner.PID == 0 || d.Revision != reserved.Revision {
			t.Fatalf("invalid ready facts: %+v %+v", owner, d)
		}
		return nil
	})
	if facts.err != nil || facts.result.Exit != 10 || ready != 1 || string(facts.result.Output) != "fact output" {
		t.Fatalf("facts=%+v ready=%d", facts, ready)
	}
	after, err := os.ReadFile(filepath.Join(store.Dir, "state.json"))
	if err != nil || string(before) != string(after) {
		t.Fatalf("worker changed authority: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, reserved.Key.HubID(), reserved.Key.Task, "runs")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("worker published history: %v", err)
	}
	if facts.ended.IsZero() {
		t.Fatal("completion facts omit observation time")
	}
}

func TestScheduleFactWorkerRejectsChangedDeclarationBeforeRelease(t *testing.T) {
	runner, fake, reserved, _ := reservedExecutionFixture(t)
	if err := os.WriteFile(filepath.Join(reserved.Key.Hub, "scripts/scheduled/report/schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"2m","script_run":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	worker := scheduleExecutionWorker{namespace: runner.namespace, permitsDir: runner.permitsDir, executor: fake, now: runner.now, withDeadline: runner.withDeadline}
	called := false
	facts := worker.execute(t.Context(), reserved, nil, func(context.Context, couchschedule.OwnerIdentity, couchschedule.Declaration) error {
		called = true
		return nil
	})
	if facts.err == nil || called {
		t.Fatalf("changed declaration reached authorization: called=%v facts=%+v", called, facts)
	}
	for _, r := range fake.Runs() {
		if r.Authorized {
			t.Fatal("changed script authorized")
		}
	}
}

func TestScheduleFactWorkerRetainsReservationDeadline(t *testing.T) {
	runner, fake, reserved, _ := reservedExecutionFixture(t)
	var got time.Time
	worker := scheduleExecutionWorker{namespace: runner.namespace, permitsDir: runner.permitsDir, executor: fake, now: runner.now, withDeadline: func(ctx context.Context, d time.Time) (context.Context, context.CancelFunc) {
		got = d
		return context.WithCancel(ctx)
	}}
	facts := worker.execute(t.Context(), reserved, nil, func(context.Context, couchschedule.OwnerIdentity, couchschedule.Declaration) error {
		return errors.New("owner refused")
	})
	if facts.err == nil || !got.Equal(reserved.Active.Deadline) {
		t.Fatalf("deadline=%v facts=%+v", got, facts)
	}
	for _, r := range fake.Runs() {
		if r.Authorized {
			t.Fatal("refused owner released script")
		}
	}
}
