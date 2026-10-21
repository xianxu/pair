package couchcmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

func TestScheduleExecutionSlotsDoNotConsumeNoticeAdmissionSlots(t *testing.T) {
	c, roots, now := scheduleServiceFixture(t)
	for _, root := range roots {
		if err := os.WriteFile(filepath.Join(root, "scripts/scheduled/report/schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1h","script_run":true}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	noticeDir := filepath.Join(roots[0], "scripts/scheduled/zz-notice")
	if err := os.MkdirAll(noticeDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(noticeDir, "schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(noticeDir, "main.go"), []byte("package main\nfunc main(){}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	a := newAutomaticScheduler(c, nil, onlineScheduleNotices{})
	*now = now.Add(time.Minute)
	if err := a.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	heldHubs := map[string]bool{}
	for range 2 {
		select {
		case job := <-a.jobs:
			if job.notice || job.state.Key.Task != "report" || !job.state.Spec.ScriptRun {
				t.Fatalf("expected fair script dispatch before notice: %+v", job)
			}
			heldHubs[job.state.Key.Hub] = true
			if !a.inFlight[job.state.Key.ID()] {
				t.Fatal("script not retained in flight")
			}
		default:
			t.Fatal("two due scripts were not dispatched")
		}
	}
	if len(heldHubs) != 2 {
		t.Fatal("one hub occupied both script starts")
	}
	// Taking the jobs simulates workers holding their execution leases. There is
	// deliberately no result: both execution slots remain occupied on the next pass.
	*now = now.Add(time.Second)
	if err := a.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	select {
	case job := <-a.jobs:
		if !job.notice || job.state.Key.Hub != roots[0] || job.state.Key.Task != "zz-notice" {
			t.Fatalf("expected independent notice admission, got %+v", job)
		}
		if job.state.Active == nil || job.state.Active.Stage != "delivery" {
			t.Fatal("notice dispatch missing durable delivery authority")
		}
	default:
		t.Fatal("two held script executions consumed notice admission slots")
	}
	select {
	case job := <-a.jobs:
		t.Fatalf("unexpected extra execution/admission: %+v", job)
	default:
	}
	ledger, err := c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range roots {
		state := ledger.Tasks[(couchschedule.TaskKey{Hub: root, Task: "report"}).ID()]
		if state.Active == nil || state.Active.Stage != "reserved" || !a.inFlight[state.Key.ID()] {
			t.Fatal("held script reservation changed or was redispatched")
		}
	}
}
