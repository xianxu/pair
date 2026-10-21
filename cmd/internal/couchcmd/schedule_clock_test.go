package couchcmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type schedulerClockFake struct {
	mu         sync.Mutex
	now        time.Time
	timers     map[*schedulerTimerFake]time.Time
	registered chan *schedulerTimerFake
}
type schedulerTimerFake struct {
	clock   *schedulerClockFake
	at      time.Time
	channel chan time.Time
	stopped bool
}

func newSchedulerClockFake(now time.Time) *schedulerClockFake {
	return &schedulerClockFake{now: now, timers: map[*schedulerTimerFake]time.Time{}, registered: make(chan *schedulerTimerFake, 128)}
}
func (c *schedulerClockFake) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *schedulerClockFake) NewTimer(d time.Duration) scheduleTimer {
	c.mu.Lock()
	timer := &schedulerTimerFake{clock: c, at: c.now.Add(d), channel: make(chan time.Time, 1)}
	c.timers[timer] = timer.at
	c.mu.Unlock()
	c.registered <- timer
	return timer
}
func (t *schedulerTimerFake) C() <-chan time.Time { return t.channel }
func (t *schedulerTimerFake) Stop() {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	t.stopped = true
	delete(t.clock.timers, t)
}
func (c *schedulerClockFake) advance(t *testing.T, at time.Time) {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if at.Before(c.now) {
		t.Fatal("clock moved backwards")
	}
	c.now = at
	for timer, due := range c.timers {
		if !due.After(at) {
			timer.channel <- at
			delete(c.timers, timer)
		}
	}
}
func (c *schedulerClockFake) nextTimer(t *testing.T) *schedulerTimerFake {
	t.Helper()
	for {
		select {
		case timer := <-c.registered:
			c.mu.Lock()
			stopped := timer.stopped
			c.mu.Unlock()
			if !stopped {
				return timer
			}
		case <-testwait.After(t):
			t.Fatal("scheduler did not register its next timer")
			return nil
		}
	}
}

type schedulerNoticeFake struct {
	mu         sync.Mutex
	admissions []string
}

func (n *schedulerNoticeFake) Binding(string) (json.RawMessage, bool) { return nil, false }
func (n *schedulerNoticeFake) Admit(_ context.Context, _ string, _ json.RawMessage, id, _ string) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.admissions = append(n.admissions, id)
	return "submitted", nil
}
func (n *schedulerNoticeFake) Observe(string) (string, bool, bool) { return "", false, false }
func (n *schedulerNoticeFake) count() int                          { n.mu.Lock(); defer n.mu.Unlock(); return len(n.admissions) }
func startSchedulerClockRun(t *testing.T, a *automaticScheduler, clock scheduleClock) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); a.run(ctx, clock) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("scheduler workers did not join")
		}
	})
}
func clockLedger(t *testing.T, c *scheduleCoordinator) couchschedule.Ledger {
	t.Helper()
	l, err := c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return l
}
func clockRunRecord(t *testing.T, c *scheduleCoordinator, s couchschedule.TaskState) (operationalrun.Record, []byte) {
	t.Helper()
	if s.Last == nil {
		t.Fatal("missing result")
	}
	r, out, err := (operationalrun.Store{Namespace: filepath.Dir(c.store.Dir)}).Read(t.Context(), scheduleRunKey(s))
	if err != nil {
		t.Fatal(err)
	}
	return r, out
}

func TestScheduleClockRunWakesAtDueBeforeScanAndRetainsOfflineSkips(t *testing.T) {
	c, roots, initial := scheduleServiceFixture(t)
	clock := newSchedulerClockFake(initial.Add(50 * time.Second))
	notices := &schedulerNoticeFake{}
	startSchedulerClockRun(t, newAutomaticScheduler(c, nil, notices), clock)
	timer := clock.nextTimer(t)
	due := initial.Add(time.Minute)
	if !timer.at.Equal(due) {
		t.Fatalf("due wake = %s; want %s before 30s catalogue scan", timer.at, due)
	}
	clock.advance(t, due)
	clock.nextTimer(t) // registration follows the committed/published step
	for _, root := range roots {
		state := clockLedger(t, c).Tasks[(couchschedule.TaskKey{Hub: root, Task: "report"}).ID()]
		if state.Active != nil || state.Last == nil || state.Last.Outcome != "skipped" || state.Last.Notice != "hub offline, skipped" || !state.NextDue.After(due) {
			t.Fatalf("offline due not consumed: %+v", state)
		}
		record, _ := clockRunRecord(t, c, state)
		if record.Source != "scheduled" || !record.ScheduledAt.Equal(due) || record.Notice != "hub offline, skipped" {
			t.Fatal(record)
		}
	}
	if notices.count() != 0 {
		t.Fatal("offline due reached admission")
	}
}

func TestScheduleClockRunExecutesOfflineScriptAndDropsWakeNotice(t *testing.T) {
	c, roots, initial := scheduleServiceFixture(t)
	for _, root := range roots {
		if err := os.WriteFile(filepath.Join(root, "scripts/scheduled/report/schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m","script_run":true}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	clock := newSchedulerClockFake(initial.Add(50 * time.Second))
	fake := couchschedule.NewFakeTaskRuntime()
	fake.Hold()
	for range roots {
		fake.Enqueue(couchschedule.ExecutionResult{Exit: 10, Outcome: "wake", Output: []byte("retained wake output")})
	}
	runner := &manualTaskRunner{namespace: filepath.Dir(c.store.Dir), executor: fake, now: clock.Now, withDeadline: func(ctx context.Context, _ time.Time) (context.Context, context.CancelFunc) {
		return context.WithCancel(ctx)
	}}
	notices := &schedulerNoticeFake{}
	startSchedulerClockRun(t, newAutomaticScheduler(c, runner, notices), clock)
	timer := clock.nextTimer(t)
	clock.advance(t, timer.at)
	started := map[string]bool{}
	for range roots {
		select {
		case r := <-fake.Started():
			if len(r.Args) != 1 || r.Args[0] != "script-run" {
				t.Fatalf("automatic executor argv = %q; want exact script-run tail", r.Args)
			}
			started[r.HubRoot] = true
		case <-testwait.After(t):
			t.Fatal("offline direct task did not start")
		}
	}
	if len(started) != len(roots) {
		t.Fatal("not both hubs executed", started)
	}
	fake.Release()
	// Every worker completion wakes the run loop and registers a new timer.
	for {
		clock.nextTimer(t)
		ledger := clockLedger(t, c)
		complete := true
		for _, root := range roots {
			s := ledger.Tasks[(couchschedule.TaskKey{Hub: root, Task: "report"}).ID()]
			if s.Active != nil || s.Last == nil || s.Last.Notice != "dropped: hub offline" {
				complete = false
			}
		}
		if complete {
			break
		}
	}
	for _, root := range roots {
		s := clockLedger(t, c).Tasks[(couchschedule.TaskKey{Hub: root, Task: "report"}).ID()]
		record, out := clockRunRecord(t, c, s)
		if record.Outcome != "wake" || record.Notice != "dropped: hub offline" || string(out) != "retained wake output" {
			t.Fatal(record, string(out))
		}
	}
	if notices.count() != 0 || len(fake.Runs()) != 2 {
		t.Fatal("offline notice sent or script replayed", notices.count(), fake.Runs())
	}
}

func TestScheduleClockRunRestartRetainsUnknownWithoutReplay(t *testing.T) {
	c, roots, initial := scheduleServiceFixture(t)
	due := initial.Add(time.Minute)
	key := couchschedule.TaskKey{Hub: roots[0], Task: "report"}
	if err := c.store.Update(t.Context(), func(l *couchschedule.Ledger) error {
		s := l.Tasks[key.ID()]
		s, _ = couchschedule.Advance(s, couchschedule.Event{Kind: couchschedule.ClockObserved, At: due, RunID: "before-restart", HubOnline: true, Location: time.UTC})
		l.Tasks[key.ID()] = s
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	fake := couchschedule.NewFakeTaskRuntime()
	clock := newSchedulerClockFake(due.Add(5 * time.Second))
	notices := &schedulerNoticeFake{}
	runner := &manualTaskRunner{namespace: filepath.Dir(c.store.Dir), executor: fake, now: clock.Now}
	startSchedulerClockRun(t, newAutomaticScheduler(c, runner, notices), clock)
	timer := clock.nextTimer(t)
	state := clockLedger(t, c).Tasks[key.ID()]
	if state.Active == nil || state.Active.ID != "before-restart" || state.Active.Stage != "uncertain" {
		t.Fatalf("lost retained attempt: %+v", state)
	}
	record, _ := clockRunRecord(t, c, state)
	if record.Source != "scheduled" || record.Outcome != "unknown" || record.Key.ID != "before-restart" || !record.ScheduledAt.Equal(due) {
		t.Fatalf("missing matching retained unknown history: %+v", record)
	}
	clock.advance(t, timer.at)
	clock.nextTimer(t)
	if len(fake.Runs()) != 0 || notices.count() != 0 {
		t.Fatal("restarted uncertain attempt replayed")
	}
	state = clockLedger(t, c).Tasks[key.ID()]
	if state.Active == nil || state.Active.ID != "before-restart" || state.Active.Stage != "uncertain" {
		t.Fatalf("scan forgot uncertain owner: %+v", state)
	}
}

func TestScheduleClockRunRealBrokerFinalizesFrozenNoticeWithoutReplay(t *testing.T) {
	c, roots, initial := scheduleServiceFixture(t)
	c.roots = append([]string(nil), roots[:1]...)
	root := roots[0]
	r := newServiceRig(t)
	r.world.hub = root
	r.restart(nil)
	hub := r.world.add(1)
	r.world.set(hub, func(slot *worldSlot) { slot.root = root; slot.workspace.WorktreeRoot = root })
	endpoint := newControlledEndpoint()
	r.world.endpoints[hub] = endpoint
	r.attach(hub, "scheduled-hub")
	r.wrapper(hub)
	r.waitConnected(hub, true)
	clock := newSchedulerClockFake(initial.Add(50 * time.Second))
	startSchedulerClockRun(t, newAutomaticScheduler(c, nil, scheduleNotice{service: r.s}), clock)
	due := initial.Add(time.Minute)
	timer := clock.nextTimer(t)
	if !timer.at.Equal(due) {
		t.Fatal("wrong due timer", timer.at)
	}
	clock.advance(t, due)
	var message couchmessage.Message
	select {
	case message = <-endpoint.started:
	case <-testwait.After(t):
		t.Fatal("due notice did not reach real broker endpoint")
	}
	key := couchschedule.TaskKey{Hub: root, Task: "report"}
	state := clockLedger(t, c).Tasks[key.ID()]
	if state.Active == nil || state.Active.Stage != "delivery" || state.Last == nil || state.Last.Notice != "pending" {
		t.Fatalf("missing durable delivery intent: %+v", state)
	}
	var frozen couchmessage.Binding
	if err := json.Unmarshal(state.Active.NoticeBinding, &frozen); err != nil {
		t.Fatal(err)
	}
	if frozen != hub || message.From != hub || message.To != hub || message.Origin != couchmessage.OriginSystem || state.Active.ReceiptID != message.ID || state.Active.NoticeBody != message.Body {
		t.Fatalf("not exact frozen authenticated admission: %+v %+v", state.Active, message)
	}
	if !strings.Contains(message.Body, due.UTC().Format(time.RFC3339)) {
		t.Fatal("frozen body omitted actual due instant", message.Body)
	}
	id := state.Last.ID
	record, _ := clockRunRecord(t, c, state)
	if record.Notice != "pending" || record.Outcome != "completed" {
		t.Fatalf("execution history not pending before endpoint completion: %+v", record)
	}
	// A clock poll while the endpoint is held must preserve pending history.
	clock.nextTimer(t)
	clock.advance(t, due.Add(time.Second))
	clock.nextTimer(t)
	state = clockLedger(t, c).Tasks[key.ID()]
	if state.Active == nil || state.Last.Notice != "pending" {
		t.Fatal("held delivery finalized", state)
	}
	endpoint.finish <- couchmessage.Receipt{Status: couchmessage.Submitted}
	testwait.Until(t, "canonical real broker submission", func() bool {
		receipt, known, active := r.s.broker.ReceiptSnapshot(message.ID)
		return known && !active && receipt.Status == couchmessage.Submitted
	})
	clock.advance(t, due.Add(2*time.Second))
	testwait.Until(t, "submitted delivery committed to ledger", func() bool {
		ledger, err := c.store.View(t.Context())
		if err != nil {
			return false
		}
		s := ledger.Tasks[key.ID()]
		return s.Active == nil && s.Last != nil && s.Last.ID == id && s.Last.Notice == "submitted"
	})
	clock.nextTimer(t)
	state = clockLedger(t, c).Tasks[key.ID()]
	record, _ = clockRunRecord(t, c, state)
	if record.Notice != "submitted" || !record.ScheduledAt.Equal(due) {
		t.Fatal("terminal delivery not retained", record)
	}
	// Cross the 30-second catalogue refresh, staying before the next cron slot.
	clock.advance(t, initial.Add(80*time.Second))
	clock.nextTimer(t)
	state = clockLedger(t, c).Tasks[key.ID()]
	if state.Active != nil || state.Last.ID != id || state.Last.Notice != "submitted" || endpoint.deliveries() != 1 {
		t.Fatalf("refresh replayed finalized occurrence: %+v deliveries=%d", state, endpoint.deliveries())
	}
	receipt, known, active := r.s.broker.ReceiptSnapshot(message.ID)
	if !known || active || receipt.Status != couchmessage.Submitted || receipt.Message != message {
		t.Fatal("lost canonical receipt", receipt, known, active)
	}
}
