package couchcmd

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"strings"
	"testing"
	"time"
)

func TestScheduleOwnerLoopIndependentBoundedDispatch(t *testing.T) {
	o := &scheduleOwner{operations: map[string]*scheduleOwnerOperation{}}
	o.configure(scheduleExecutionWorker{}, nil)
	state := func(task string) couchschedule.TaskState {
		return couchschedule.TaskState{Key: couchschedule.TaskKey{Hub: "/hub", Task: task}, Active: &couchschedule.Attempt{ID: task, Stage: "reserved"}}
	}
	for _, name := range []string{"a", "b"} {
		if !o.launchExecution(t.Context(), state(name), nil) {
			t.Fatal("execution capacity missing")
		}
	}
	if o.launchExecution(t.Context(), state("c"), nil) {
		t.Fatal("exceeded two executions")
	}
	if o.launchExecution(t.Context(), state("a"), nil) {
		t.Fatal("duplicate execution")
	}
	for _, name := range []string{"n", "m"} {
		if !o.launchNotice(t.Context(), nativeScheduleClock{}, state(name)) {
			t.Fatal("execution exhausted notice capacity")
		}
	}
	if o.launchNotice(t.Context(), nativeScheduleClock{}, state("p")) {
		t.Fatal("exceeded two notices")
	}
}
func TestScheduleOwnerLoopUnavailableAndCancelledCommands(t *testing.T) {
	o := &scheduleOwner{}
	if _, err := o.Submit(t.Context(), couchschedule.ManualIntent{}); err == nil {
		t.Fatal("unconfigured transport accepted")
	}
	o.configure(scheduleExecutionWorker{}, nil)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := o.Observe(ctx, "/hub", "task", "run", 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled command: %v", err)
	}
	if len(o.loop.commands) != 0 {
		t.Fatal("cancelled caller queued work")
	}
}
func TestScheduleOwnerLoopQueuedGateCancellationWaitsForResolution(t *testing.T) {
	o := &scheduleOwner{}
	o.configure(scheduleExecutionWorker{}, nil)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = o.loop.gate(ctx, couchschedule.TaskState{}, func() (string, error) { t.Error("stale send executed"); return "submitted", nil })
	}()
	command := <-o.loop.commands
	cancel()
	select {
	case <-done:
		t.Fatal("gate released before queued closure resolved")
	default:
	}
	o.handleCommand(command, nativeScheduleClock{})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("resolved gate callback did not return")
	}
}

func TestScheduleOwnerLoopFullCommandHandoffReportsBusy(t *testing.T) {
	o := &scheduleOwner{}
	o.configure(scheduleExecutionWorker{}, nil)
	for i := 0; i < cap(o.loop.commands); i++ {
		o.loop.commands <- scheduleOwnerCommand{}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if _, err := o.Observe(ctx, "/hub", "task", "run", 0); err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("full handoff must refuse promptly as busy: %v", err)
	}
}
func TestScheduleOwnerLoopGateSettlesTerminalHistoryBeforeSend(t *testing.T) {
	a, _, state, _ := scheduledDeliveryFixture(t)
	if err := a.history().FinalizeNotice(t.Context(), scheduleRunKey(state), state.Active.Revision, "submitted"); err != nil {
		t.Fatal(err)
	}
	o := newScheduleOwner(a.coordinator)
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer o.close()
	o.configure(scheduleExecutionWorker{}, nil)
	sent := false
	cmd := scheduleOwnerCommand{kind: ownerGateReady, ctx: t.Context(), state: state, reply: make(chan scheduleOwnerReply, 1), send: func() (string, error) { sent = true; return "submitted", nil }}
	o.handleCommand(cmd, newSchedulerClockFake(a.coordinator.now()))
	reply := <-cmd.reply
	if sent {
		t.Fatal("sent from stale ledger after terminal history committed")
	}
	if reply.err != nil {
		t.Fatal(reply.err)
	}
	ledger, err := o.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if ledger.Tasks[state.Key.ID()].Active != nil {
		t.Fatal("did not adopt canonical terminal history")
	}
}

func TestScheduleOwnerLoopCompletionFactRetainedWithoutExecutionCapacity(t *testing.T) {
	a, _, state, _ := scheduledResultFixture(t)
	fail := false
	a.coordinator.store.BeforePersist = func() error {
		if fail {
			return errors.New("authority fault")
		}
		return nil
	}
	o := newScheduleOwner(a.coordinator)
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer o.close()
	o.configure(scheduleExecutionWorker{}, nil)
	key := state.Key.ID()
	o.loop.active[key] = false
	facts := scheduleExecutionFacts{result: couchschedule.ExecutionResult{Exit: 0, Outcome: "quiet", Output: []byte("immutable result")}, ended: a.coordinator.now()}
	o.collect(scheduleOwnerResult{job: scheduleOwnerJob{state: state}, facts: facts})
	clock := newSchedulerClockFake(a.coordinator.now())
	fail = true
	if err := o.retry(t.Context(), clock); err == nil {
		t.Fatal("shared authority failure did not block owner admission")
	}
	if err := o.retry(t.Context(), clock); err == nil {
		t.Fatal("shared failure forgotten during retry delay")
	}
	if len(o.loop.pending) != 1 || len(o.loop.active) != 0 {
		t.Fatalf("lost result or wasted worker: pending=%d active=%d", len(o.loop.pending), len(o.loop.active))
	}
	if o.launchExecution(t.Context(), state, nil) {
		t.Fatal("replayed unsettled occurrence")
	}
	other := state
	other.Key.Task = "healthy"
	other.Active = &couchschedule.Attempt{ID: "healthy", Stage: "reserved"}
	if !o.launchExecution(t.Context(), other, nil) {
		t.Fatal("quarantined completion blocked healthy worker")
	}
	fail = false
	clock.advance(t, clock.Now().Add(time.Second))
	o.retry(t.Context(), clock)
	if len(o.loop.pending) != 0 {
		t.Fatal("repair failed to settle retained completion")
	}
}
func TestScheduleOwnerLoopNoticeDeadlineIncludesQueuedGate(t *testing.T) {
	a, _, state, port := scheduledDeliveryFixture(t)
	o := newScheduleOwner(a.coordinator)
	o.configure(scheduleExecutionWorker{}, port)
	clock := newSchedulerClockFake(a.coordinator.now())
	type result struct {
		notice string
		err    error
	}
	done := make(chan result, 1)
	go func() { notice, err := o.loop.admit(t.Context(), clock, state); done <- result{notice, err} }()
	cmd := <-o.loop.commands
	clock.advance(t, state.Active.NoticeDeadline)
	select {
	case <-done:
		t.Fatal("deadline released gate while owner closure queued")
	default:
	}
	o.handleCommand(cmd, clock)
	select {
	case got := <-done:
		if got.err != nil || got.notice != "expired" {
			t.Fatalf("deadline result: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("deadline did not finish")
	}
	if port.admissions != 0 {
		t.Fatal("queued expired notice entered broker")
	}
}

func TestScheduleOwnerLoopLongScriptLeavesOwnerResponsiveAndJoinsWorker(t *testing.T) {
	runner, fake, reserved, store := reservedExecutionFixture(t)
	if err := store.Update(t.Context(), func(l *couchschedule.Ledger) error {
		s := l.Tasks[reserved.Key.ID()]
		s.Active = nil
		s.Last = nil
		l.Tasks[reserved.Key.ID()] = s
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	c := newScheduleCoordinator(runner.namespace, []string{reserved.Key.Hub}, time.UTC, runner.now)
	o := newScheduleOwner(c)
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer o.close()
	o.configure(scheduleExecutionWorker{namespace: runner.namespace, permitsDir: runner.permitsDir, executor: runner.executor, now: runner.now, withDeadline: runner.withDeadline}, nil)
	m := couchschedule.ManualIntent{ID: "manual-worker", Hub: reserved.Key.Hub, Task: reserved.Key.Task, Args: []string{"--help"}}
	if _, err := o.submitManual(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	ledger, err := o.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	state := ledger.Tasks[reserved.Key.ID()]
	fake.Hold()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	if !o.launchExecution(ctx, state, m.Args) {
		t.Fatal("manual work not admitted")
	}
	workerDone := make(chan struct{})
	go func() { defer close(workerDone); o.work(ctx, o.loop.execution, nativeScheduleClock{}) }()
	select {
	case cmd := <-o.loop.commands:
		o.handleCommand(cmd, newSchedulerClockFake(runner.now()))
	case <-time.After(time.Second):
		t.Fatal("worker did not request owner authorization")
	}
	select {
	case <-fake.Started():
	case <-time.After(time.Second):
		t.Fatal("authorized worker did not start")
	}
	observed := make(chan error, 1)
	go func() {
		status, err := o.Observe(ctx, m.Hub, m.Task, m.ID, 0)
		if err == nil && status.Stage != "running" {
			err = errors.New("long script not reported running")
		}
		observed <- err
	}()
	select {
	case cmd := <-o.loop.commands:
		o.handleCommand(cmd, newSchedulerClockFake(runner.now()))
	case <-time.After(time.Second):
		t.Fatal("long execution blocked owner command")
	}
	if err := <-observed; err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case result := <-o.loop.results:
		if result.facts.err == nil && result.facts.result.Outcome != "cancelled" {
			t.Fatal("worker cancellation not observed")
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled worker did not return facts")
	}
	select {
	case <-workerDone:
	case <-time.After(time.Second):
		t.Fatal("worker did not join")
	}
	runs := fake.Runs()
	if len(runs) != 1 || !runs[0].Finished || !runs[0].Cancelled {
		t.Fatalf("shutdown before worker cleanup: %+v", runs)
	}
}

func TestScheduleOwnerLoopKnownNonAdmissionBackoff(t *testing.T) {
	a, _, state, port := scheduledDeliveryFixture(t)
	o := newScheduleOwner(a.coordinator)
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer o.close()
	o.configure(scheduleExecutionWorker{}, port)
	clock := newSchedulerClockFake(a.coordinator.now())
	o.collect(scheduleOwnerResult{job: scheduleOwnerJob{state: state, notice: true}, notice: "not-admitted"})
	if err := o.retry(t.Context(), clock); err != nil {
		t.Fatal(err)
	}
	if o.launchNotice(t.Context(), clock, state) {
		t.Fatal("known refusal immediately hammered broker without backoff")
	}
	clock.advance(t, clock.Now().Add(time.Second))
	if !o.launchNotice(t.Context(), clock, state) {
		t.Fatal("known non-admission did not become eligible after bounded retry")
	}
}
