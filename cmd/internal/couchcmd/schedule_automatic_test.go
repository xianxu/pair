package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type offlineScheduleNotices struct{}

func (offlineScheduleNotices) Binding(string) (json.RawMessage, bool) { return nil, false }
func (offlineScheduleNotices) Admit(context.Context, string, json.RawMessage, string, string) (string, error) {
	panic("offline admission")
}
func (offlineScheduleNotices) Observe(string) (string, bool, bool) { panic("offline observation") }

func TestScheduleAutomaticOfflineSkipPersistsHistory(t *testing.T) {
	c, roots, now := scheduleServiceFixture(t)
	a := newAutomaticScheduler(c, nil, offlineScheduleNotices{})
	*now = now.Add(time.Minute)
	if err := a.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, root := range roots {
		key := couchschedule.TaskKey{Hub: root, Task: "report"}
		ledger, err := c.store.View(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		state := ledger.Tasks[key.ID()]
		if state.Active != nil || state.Last == nil || state.Last.Notice != "hub offline, skipped" || !state.NextDue.After(*now) {
			t.Fatalf("offline due not consumed: %+v", state)
		}
		record, _, err := (operationalrun.Store{Namespace: filepath.Dir(c.store.Dir)}).Read(t.Context(), operationalrun.RunKey{Hub: key.HubID(), Task: key.Task, ID: state.Last.ID})
		if err != nil || record.Outcome != "skipped" || record.Source != "scheduled" {
			t.Fatalf("missing skip history: %+v %v", record, err)
		}
	}
	if len(a.jobs) != 0 {
		t.Fatal("offline task dispatched")
	}
}

type onlineScheduleNotices struct{ offlineScheduleNotices }

func (onlineScheduleNotices) Binding(string) (json.RawMessage, bool) {
	return json.RawMessage(`{"slot":"hub"}`), true
}

func TestScheduleCatalogCancellationFinalizesPendingNotice(t *testing.T) {
	c, roots, now := scheduleServiceFixture(t)
	a := newAutomaticScheduler(c, nil, onlineScheduleNotices{})
	*now = now.Add(time.Minute)
	if err := a.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	key := couchschedule.TaskKey{Hub: roots[0], Task: "report"}
	ledger, err := c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	id := ledger.Tasks[key.ID()].Active.ID
	if err := os.RemoveAll(filepath.Join(roots[0], "scripts/scheduled/report")); err != nil {
		t.Fatal(err)
	}
	if err := c.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	record, _, err := a.history().Read(t.Context(), operationalrun.RunKey{Hub: key.HubID(), Task: key.Task, ID: id})
	if err != nil || record.Notice != "uncertain" {
		t.Fatalf("cancelled delivery retains pending history: %+v %v", record, err)
	}
}

func TestScheduleAutomaticSharesWorkersBetweenHubs(t *testing.T) {
	c, roots, now := scheduleServiceFixture(t)
	for i := 0; i < 8; i++ {
		dir := filepath.Join(roots[0], "scripts/scheduled", fmt.Sprintf("a%d", i))
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m"}`), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main(){}\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	// A future task in the second hub must not consume that hub's turn.
	futureDir := filepath.Join(roots[1], "scripts/scheduled/a0")
	if err := os.MkdirAll(futureDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(futureDir, "schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(futureDir, "main.go"), []byte("package main\nfunc main(){}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := c.store.Update(t.Context(), func(l *couchschedule.Ledger) error {
		key := (couchschedule.TaskKey{Hub: roots[1], Task: "a0"}).ID()
		state := l.Tasks[key]
		state.NextDue = now.Add(2 * time.Minute)
		l.Tasks[key] = state
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	a := newAutomaticScheduler(c, nil, onlineScheduleNotices{})
	*now = now.Add(time.Minute)
	if err := a.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	first, second := <-a.jobs, <-a.jobs
	if first.state.Key.Hub == second.state.Key.Hub {
		t.Error("one hub occupied both worker slots while another hub was due")
	}
	ledger, err := c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	delivery := 0
	for _, state := range ledger.Tasks {
		if state.Active == nil {
			if state.Key.Hub == roots[1] && state.Key.Task == "a0" {
				continue
			}
			t.Fatalf("due notice lost authority: %+v", state)
		}
		if state.Active.Stage == "delivery" {
			delivery++
		}
	}
	if delivery != 2 {
		t.Fatalf("delivery authority without dispatched job: %d", delivery)
	}
}

func TestScheduleAutomaticRetriesUnavailableStoreAtScan(t *testing.T) {
	c, _, initial := scheduleServiceFixture(t)
	var fail atomic.Bool
	fail.Store(true)
	c.store.BeforePersist = func() error {
		if fail.Load() {
			return errors.New("temporary store failure")
		}
		return nil
	}
	clock := newSchedulerClockFake(*initial)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan struct{})
	a := newAutomaticScheduler(c, nil, offlineScheduleNotices{})
	go func() { defer close(done); a.run(ctx, clock) }()
	select {
	case <-done:
		t.Fatal("transient store refusal stopped scheduler permanently")
	case timer := <-clock.registered:
		if !timer.at.Equal(initial.Add(30 * time.Second)) {
			t.Fatalf("retry not scan bounded: %s", timer.at)
		}
	case <-testwait.After(t):
		t.Fatal("no retry timer")
	}
	fail.Store(false)
	clock.advance(t, initial.Add(30*time.Second))
	clock.nextTimer(t)
	if c.view().Error != "" {
		t.Fatalf("recovered store remains blocked: %s", c.view().Error)
	}
	cancel()
	select {
	case <-done:
	case <-testwait.After(t):
		t.Fatal("retry worker did not join")
	}
}

func TestScheduleAutomaticKnownNonAdmissionWaitsForClockRetry(t *testing.T) {
	c, _, now := scheduleServiceFixture(t)
	a := newAutomaticScheduler(c, nil, onlineScheduleNotices{})
	*now = now.Add(time.Minute)
	if err := a.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	first, second := <-a.jobs, <-a.jobs
	a.results <- scheduleJobResult{job: first, notice: "not-admitted"}
	a.results <- scheduleJobResult{job: second, notice: "not-admitted"}
	if err := a.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(a.jobs) != 0 {
		t.Fatal("known non-admission retried without advancing clock")
	}
	*now = now.Add(time.Second)
	if err := a.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(a.jobs) != 2 {
		t.Fatalf("retry never became eligible: %d jobs", len(a.jobs))
	}
	for range 2 {
		job := <-a.jobs
		if job.state.Active.ReceiptID != first.state.Active.ReceiptID && job.state.Active.ReceiptID != second.state.Active.ReceiptID {
			t.Fatal("retry allocated a new notice identity")
		}
	}
}

type countedScheduleBindings struct {
	onlineScheduleNotices
	calls int
}

func (n *countedScheduleBindings) Binding(root string) (json.RawMessage, bool) {
	n.calls++
	return n.onlineScheduleNotices.Binding(root)
}
func TestScheduleAutomaticResolvesEachHubOncePerDuePass(t *testing.T) {
	for _, count := range []int{8, 16} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			c, roots, now := scheduleServiceFixture(t)
			for i := 0; i < count; i++ {
				dir := filepath.Join(roots[i%len(roots)], "scripts/scheduled", fmt.Sprintf("t%d", i))
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m"}`), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main(){}\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.refresh(t.Context()); err != nil {
				t.Fatal(err)
			}
			notices := &countedScheduleBindings{}
			a := newAutomaticScheduler(c, nil, notices)
			*now = now.Add(time.Minute)
			if err := a.step(t.Context()); err != nil {
				t.Fatal(err)
			}
			if notices.calls != len(roots) {
				t.Fatalf("%d tasks resolved hub bindings %d times, want %d", count, notices.calls, len(roots))
			}
		})
	}
}

// A run bundle can survive a failed ledger commit. The next observation must
// reconcile that immutable occurrence, rather than constructing different facts.
func TestScheduleDueHistorySurvivesFailedOccurrenceCommit(t *testing.T) {
	for _, online := range []bool{false, true} {
		t.Run(fmt.Sprint(online), func(t *testing.T) {
			c, roots, now := scheduleServiceFixture(t)
			var port scheduleNoticePort = offlineScheduleNotices{}
			if online {
				port = onlineScheduleNotices{}
			}
			a := newAutomaticScheduler(c, nil, port)
			*now = now.Add(time.Minute)
			injected := errors.New("occurrence ledger commit failed")
			c.store.BeforePersist = func() error { return injected }
			if err := a.step(t.Context()); !errors.Is(err, injected) {
				t.Fatalf("missing occurrence fault: %v", err)
			}
			if len(a.jobs) != 0 {
				t.Fatal("effect dispatched before failed occurrence commit")
			}
			before, err := a.history().List(t.Context(), (couchschedule.TaskKey{Hub: roots[0], Task: "report"}).HubID(), "report", 10)
			if err != nil || len(before) != 1 {
				t.Fatalf("failure did not leave record: %v %v", before, err)
			}
			c.store.BeforePersist = nil
			*now = now.Add(time.Second)
			if err := a.step(t.Context()); err != nil {
				t.Fatalf("immutable due history prevents recovery: %v", err)
			}
			after, err := a.history().List(t.Context(), before[0].Key.Hub, "report", 10)
			if err != nil || len(after) != 1 || !after[0].StartedAt.Equal(before[0].StartedAt) || !after[0].EndedAt.Equal(before[0].EndedAt) {
				t.Fatalf("recovery rewrote run facts: before=%+v after=%+v err=%v", before, after, err)
			}
		})
	}
}
