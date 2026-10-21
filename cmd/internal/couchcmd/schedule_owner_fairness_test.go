package couchcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

func ownerFairnessFixture(t *testing.T, count int, script bool) (*scheduleOwner, []string, *time.Time) {
	t.Helper()
	c, roots, now := scheduleServiceFixture(t)
	for _, root := range roots {
		for i := 0; i < count/len(roots); i++ {
			task := "report"
			if i > 0 {
				task = fmt.Sprintf("task-%02d", i)
			}
			dir := filepath.Join(root, "scripts", "scheduled", task)
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			spec := fmt.Sprintf(`{"version":1,"cron":"* * * * *","timeout":"1m","script_run":%t}`, script)
			if err := os.WriteFile(filepath.Join(dir, "schedule"), []byte(spec), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	o := newScheduleOwner(c)
	o.configure(scheduleExecutionWorker{}, onlineScheduleNotices{})
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { o.close() })
	if err := o.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(time.Minute)
	return o, roots, now
}

func TestScheduleOwnerFairAdmissionAcrossHubs(t *testing.T) {
	for _, script := range []bool{false, true} {
		t.Run(fmt.Sprintf("script=%t", script), func(t *testing.T) {
			o, roots, now := ownerFairnessFixture(t, 8, script)
			// The first row of hub B is not due. It must not consume that hub's
			// position while eligible rows from hub A take both bounded workers.
			if err := o.state.Update(t.Context(), func(l *couchschedule.Ledger) error {
				key := (couchschedule.TaskKey{Hub: roots[1], Task: "report"}).ID()
				s := l.Tasks[key]
				s.NextDue = now.Add(time.Minute)
				l.Tasks[key] = s
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if err := o.tick(t.Context(), newSchedulerClockFake(*now)); err != nil {
				t.Fatal(err)
			}
			jobs := o.loop.notices
			if script {
				jobs = o.loop.execution
			}
			if len(jobs) != 2 {
				t.Fatalf("queued %d jobs, want two", len(jobs))
			}
			a, b := <-jobs, <-jobs
			defer a.cancel()
			defer b.cancel()
			if a.state.Key.Hub == b.state.Key.Hub {
				t.Fatalf("one hub occupied both worker slots: %s and %s", a.state.Key.ID(), b.state.Key.ID())
			}
		})
	}
}

type ownerCountedBindings struct {
	onlineScheduleNotices
	calls map[string]int
}

func (n *ownerCountedBindings) Binding(root string) (json.RawMessage, bool) {
	n.calls[root]++
	return n.onlineScheduleNotices.Binding(root)
}
func TestScheduleOwnerBindingLookupsBoundedPerHub(t *testing.T) {
	for _, count := range []int{8, 16} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			o, roots, now := ownerFairnessFixture(t, count, false)
			port := &ownerCountedBindings{calls: map[string]int{}}
			o.loop.port = port
			if err := o.tick(t.Context(), newSchedulerClockFake(*now)); err != nil {
				t.Fatal(err)
			}
			for _, root := range roots {
				// Each hub has one online observation and one admitted notice at either size.
				if n := port.calls[root]; n != 2 {
					t.Fatalf("%d tasks: hub %s had %d binding lookups, want two (online observation and admitted notice)", count, root, n)
				}
				t.Logf("tasks=%d hub=%s binding lookups=%d", count, filepath.Base(root), port.calls[root])
			}
			if len(o.loop.notices) != 2 {
				t.Fatalf("queued %d notices, want two", len(o.loop.notices))
			}
			for len(o.loop.notices) > 0 {
				job := <-o.loop.notices
				job.cancel()
			}
		})
	}
}
