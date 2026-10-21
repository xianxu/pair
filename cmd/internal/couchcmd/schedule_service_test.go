package couchcmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/osfs"
)

func TestScheduleStatusHandlerReadsOnlyPublishedMemory(t *testing.T) {
	ctx := context.Background()
	s := &messageService{}
	if got := s.handle(ctx, couchmessage.Request{Op: "schedule-status"}); got.Code != "unsupported" {
		t.Fatalf("missing provider: %+v", got)
	}
	c := newScheduleCoordinator("/must-not-be-read", []string{"/ops"}, time.UTC, time.Now)
	s.schedule.Store(c)
	got := s.handle(ctx, couchmessage.Request{Op: "schedule-status"})
	if got.Code != "ok" || got.Schedule == nil || len(got.Schedule.Hubs) != 1 || got.Schedule.Hubs[0] != "/ops" {
		t.Fatalf("status refreshed IO instead of projecting: %+v", got)
	}
	if got := s.handle(ctx, couchmessage.Request{Op: "schedule-status", Body: "foreign"}); got.Code != "invalid-request" {
		t.Fatalf("foreign payload accepted: %+v", got)
	}
}

func scheduleServiceFixture(t *testing.T) (*scheduleCoordinator, []string, *time.Time) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	roots := []string{filepath.Join(base, "a"), filepath.Join(base, "b")}
	for _, root := range roots {
		dir := filepath.Join(root, "scripts/scheduled/report")
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
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	c := newScheduleCoordinator(filepath.Join(base, "namespace"), roots, time.UTC, func() time.Time { return now })
	if err := c.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	return c, roots, &now
}

func serviceManualOwner(t *testing.T, c *scheduleCoordinator, root string) couchschedule.TaskState {
	t.Helper()
	key := couchschedule.TaskKey{Hub: root, Task: "report"}
	var state couchschedule.TaskState
	if err := c.store.Update(t.Context(), func(l *couchschedule.Ledger) error {
		state = l.Tasks[key.ID()]
		state.Active = &couchschedule.Attempt{ID: "manual-live", Revision: state.Revision, Source: "manual", Stage: "running", StartedAt: c.now(), Deadline: c.now().Add(time.Minute), ControllerPID: 12345, ControllerStart: "exact-owner"}
		l.Tasks[key.ID()] = state
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestScheduleRemovedStateAgesFromFirstRemovalAndProtectsOwners(t *testing.T) {
	c, roots, now := scheduleServiceFixture(t)
	key := couchschedule.TaskKey{Hub: roots[0], Task: "report"}
	if err := os.RemoveAll(filepath.Join(roots[0], "scripts/scheduled/report")); err != nil {
		t.Fatal(err)
	}
	if err := c.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	removed := *now
	*now = now.Add(29 * 24 * time.Hour)
	if err := c.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	l, err := c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !l.Tasks[key.ID()].RemovedAt.Equal(removed) {
		t.Fatalf("scan extended removal retention: %+v", l.Tasks[key.ID()])
	}
	// An uncertain owner outlives ordinary removed-state retention.
	protected := l.Tasks[key.ID()]
	protected.Key.Task = "uncertain"
	protected.Active = &couchschedule.Attempt{ID: "old-owner", Revision: protected.Revision, Source: "scheduled", Stage: "uncertain", StartedAt: removed, Deadline: removed.Add(time.Minute)}
	protected.Last = &couchschedule.RunSummary{ID: protected.Active.ID, Source: protected.Active.Source, DueAt: protected.Active.DueAt, StartedAt: protected.Active.StartedAt, Outcome: "unknown"}
	if err := c.store.Update(t.Context(), func(l *couchschedule.Ledger) error { l.Tasks[protected.Key.ID()] = protected; return nil }); err != nil {
		t.Fatal(err)
	}
	*now = removed.Add(30 * 24 * time.Hour)
	if err := c.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	l, err = c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := l.Tasks[key.ID()]; exists {
		t.Fatal("removed state survived retention deadline")
	}
	if l.Tasks[protected.Key.ID()].Active == nil {
		t.Fatal("pruned uncertain execution authority")
	}
}

func TestScheduleSnapshotExcludesRetiredHubAuthority(t *testing.T) {
	c, roots, _ := scheduleServiceFixture(t)
	old := couchschedule.TaskKey{Hub: "/old-unconfigured-hub", Task: "report"}
	if err := c.store.Update(t.Context(), func(l *couchschedule.Ledger) error {
		copy := l.Tasks[(couchschedule.TaskKey{Hub: roots[0], Task: "report"}).ID()]
		copy.Key = old
		l.Tasks[old.ID()] = copy
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := c.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	snapshot := c.view()
	if len(snapshot.Hubs) != len(roots) {
		t.Fatalf("retired hub admitted: %+v", snapshot.Hubs)
	}
	for _, hub := range snapshot.Hubs {
		if hub == old.Hub {
			t.Fatal("retired hub selectable")
		}
	}
	for _, row := range snapshot.Tasks {
		if row.Hub == old.Hub {
			t.Fatal("retired task visible")
		}
	}
}

func TestScheduleLiveManualOwnerSurvivesRestartAndUnknownOwnerNeverReplays(t *testing.T) {
	for _, identity := range []string{"exact-owner", "", "other-incarnation"} {
		t.Run("identity="+identity, func(t *testing.T) {
			c, roots, _ := scheduleServiceFixture(t)
			state := serviceManualOwner(t, c, roots[0])
			c = newScheduleCoordinator(filepath.Dir(c.store.Dir), roots, time.UTC, c.now)
			stops, probes := 0, 0
			c.stopOwner = func(couchschedule.Attempt) error { stops++; return nil }
			c.ownerIdentity = func(pid int) string {
				probes++
				if pid != 12345 {
					t.Fatalf("wrong owner probe %d", pid)
				}
				return identity
			}
			if err := c.refresh(t.Context()); err != nil {
				t.Fatal(err)
			}
			l, err := c.store.View(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			got := l.Tasks[state.Key.ID()]
			if got.Active == nil || got.Active.ID != state.Active.ID || stops != 0 || probes != 1 {
				t.Fatalf("lost/restarted owner: %+v stops=%d probes=%d", got, stops, probes)
			}
			if identity == "exact-owner" {
				if got.Active.Stage != "running" || got.Last != nil {
					t.Fatalf("live manual owner cancelled: %+v", got)
				}
			} else {
				if got.Active.Stage != "uncertain" || got.Last == nil || got.Last.Outcome != "unknown" {
					t.Fatalf("unknown owner lost uncertainty: %+v", got)
				}
				record, _, err := (operationalrun.Store{Namespace: filepath.Dir(c.store.Dir)}).Read(t.Context(), operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: state.Active.ID})
				if err != nil || record.Source != "manual" || record.Outcome != "unknown" {
					t.Fatalf("unknown manual run missing history: %+v %v", record, err)
				}

			}
		})
	}
}

func TestScheduleCompletedManualRecordWinsBeforeNativeProbe(t *testing.T) {
	c, roots, _ := scheduleServiceFixture(t)
	state := serviceManualOwner(t, c, roots[0])
	exit := 10
	record := operationalrun.Record{Version: 1, Key: operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: state.Active.ID}, DeclarationRevision: state.Active.Revision, Source: "manual", StartedAt: state.Active.StartedAt, EndedAt: state.Active.StartedAt.Add(time.Second), Exit: &exit, Outcome: "wake", Notice: "manual"}
	if err := (operationalrun.Store{Namespace: filepath.Dir(c.store.Dir)}).Publish(t.Context(), record.Key, record, []byte("saved")); err != nil {
		t.Fatal(err)
	}
	c.ownerIdentity = func(int) string { t.Fatal("known completed run must precede native probe"); return "" }
	if err := c.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	l, err := c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := l.Tasks[state.Key.ID()]
	if got.Active != nil || got.Last == nil || got.Last.Outcome != "wake" {
		t.Fatalf("known result lost: %+v", got)
	}
}

func TestScheduleDamagedManualHistoryRefusesOnlyItsTask(t *testing.T) {
	for _, damage := range []string{"corrupt", "revision", "source", "started"} {
		t.Run(damage, func(t *testing.T) {
			c, roots, _ := scheduleServiceFixture(t)
			state := serviceManualOwner(t, c, roots[0])
			key := operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: state.Active.ID}
			exit := 0
			record := operationalrun.Record{Version: 1, Key: key, DeclarationRevision: state.Active.Revision, Source: "manual", StartedAt: state.Active.StartedAt, EndedAt: state.Active.StartedAt.Add(time.Second), Exit: &exit, Outcome: "quiet", Notice: "manual"}
			namespace := filepath.Dir(c.store.Dir)
			if err := (operationalrun.Store{Namespace: namespace}).Publish(t.Context(), key, record, nil); err != nil {
				t.Fatal(err)
			}
			switch damage {
			case "revision":
				record.DeclarationRevision = "foreign"
			case "source":
				record.Source = "scheduled"
			case "started":
				record.StartedAt = record.StartedAt.Add(-time.Hour)
			}
			raw, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if damage == "corrupt" {
				raw = []byte("broken")
			}
			if err := os.WriteFile(filepath.Join(namespace, "schedule", key.Hub, key.Task, "runs", key.ID, "record.json"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			c.ownerIdentity = func(int) string { return "exact-owner" }
			c.stopOwner = func(couchschedule.Attempt) error { t.Fatal("damaged history must not kill manual owner"); return nil }
			if err := c.refresh(t.Context()); err != nil {
				t.Fatalf("one damaged bundle blocked all hubs: %v", err)
			}
			snapshot := c.view()
			if snapshot.Error != "" {
				t.Fatalf("global error: %s", snapshot.Error)
			}
			seenBad, seenGood := false, false
			for _, row := range snapshot.Tasks {
				if row.Hub == roots[0] {
					if damage == "started" {
						// Same-ID history is canonical; timestamps are not identity guards.
						seenBad = row.State == "waiting" && row.Active == nil && row.Last != nil && row.Last.StartedAt.Equal(record.StartedAt)
					} else {
						seenBad = row.State == "refused" && row.Active != nil && row.Active.Stage == state.Active.Stage && strings.Contains(row.Reason, "history manual recovery")
					}
				} else if row.Hub == roots[1] {
					seenGood = row.State == "waiting"
				}
			}
			if !seenBad || !seenGood {
				t.Fatalf("task-local refusal missing: %+v", snapshot)
			}
		})
	}
}

func TestScheduleWorkerRefreshesOnTicksAndJoinsOnClosedStream(t *testing.T) {
	c, _, _ := scheduleServiceFixture(t)
	reads := make(chan struct{}, 4)
	c.catalog.ReadFile = func(ctx context.Context, f *os.File, max int64) ([]byte, error) {
		raw, err := osfs.ReadOpenedRegularMetadata(ctx, f, max)
		reads <- struct{}{}
		return raw, err
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	ticks := make(chan time.Time)
	done := make(chan struct{})
	go func() { c.run(ctx, ticks); close(done) }()
	for i := 0; i < 2; i++ {
		select {
		case <-reads:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	select {
	case ticks <- c.now():
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	for i := 0; i < 2; i++ {
		select {
		case <-reads:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	close(ticks)
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("worker did not join")
	}
}

func TestScheduleRefreshPublishesManualCompletionsAcrossTwoHubs(t *testing.T) {
	ctx := context.Background()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	roots := []string{filepath.Join(base, "ops"), filepath.Join(base, "echo")}
	for _, root := range roots {
		task := filepath.Join(root, "scripts", "scheduled", "report")
		if err := os.MkdirAll(task, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(task, "schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m"}`), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(task, "main.go"), []byte("package main\nfunc main(){}\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	c := newScheduleCoordinator(filepath.Join(base, "couch"), roots, time.UTC, func() time.Time { return now })
	if err := c.refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if snapshot := c.view(); snapshot.Error != "" || len(snapshot.Tasks) != 2 {
		t.Fatalf("refresh: %+v", snapshot)
	}
	key := couchschedule.TaskKey{Hub: roots[0], Task: "report"}
	if err := c.store.Update(ctx, func(ledger *couchschedule.Ledger) error {
		state := ledger.Tasks[key.ID()]
		state.Last = &couchschedule.RunSummary{ID: "manual", Outcome: "quiet", Source: "manual"}
		ledger.Tasks[key.ID()] = state
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := c.refresh(ctx); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range c.view().Tasks {
		if row.Hub == roots[0] {
			found = row.Last != nil && row.Last.ID == "manual" && row.NextDue.Equal(now.Add(time.Minute))
		}
	}
	if !found {
		t.Fatalf("manual completion lost: %+v", c.view())
	}
	if err := os.WriteFile(filepath.Join(base, "couch", "schedule", "state.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if c.refresh(ctx) == nil || c.view().Error == "" {
		t.Fatal("corrupt authority must become visible and block refresh")
	}
}

func TestScheduleViewFrozenBindingBufferDetached(t *testing.T) {
	c := newScheduleCoordinator(t.TempDir(), []string{"/hub"}, time.UTC, time.Now)
	binding := []byte(`{"nonce":"frozen"}`)
	snapshot := couchschedule.StatusSnapshot{Version: 1, Hubs: []string{"/hub"}, Tasks: []couchschedule.TaskStatus{{Hub: "/hub", Task: "task", Active: &couchschedule.Attempt{NoticeBinding: binding}}}}
	c.snapshot.Store(&snapshot)
	first := c.view()
	first.Tasks[0].Active.NoticeBinding[0] = '['
	if string(snapshot.Tasks[0].Active.NoticeBinding) != `{"nonce":"frozen"}` {
		t.Fatal("view binding buffer aliases cached snapshot")
	}
	second := c.view()
	if string(second.Tasks[0].Active.NoticeBinding) != `{"nonce":"frozen"}` {
		t.Fatal("view mutation changed later caller snapshot")
	}
	snapshot.Tasks[0].Active.NoticeBinding[0] = '['
	if string(second.Tasks[0].Active.NoticeBinding) != `{"nonce":"frozen"}` {
		t.Fatal("cached snapshot mutation changed detached view")
	}
}
