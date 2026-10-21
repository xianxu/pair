package couchcmd

import (
	"context"
	"path/filepath"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type scheduleCoordinator struct {
	roots         []string
	store         couchschedule.StateStore
	runs          operationalrun.Store
	catalog       couchschedule.Catalog
	location      *time.Location
	now           func() time.Time
	snapshot      atomic.Pointer[couchschedule.StatusSnapshot]
	stopOwner     func(couchschedule.Attempt) error
	ownerIdentity func(int) string
}

func newScheduleCoordinator(namespace string, roots []string, location *time.Location, now func() time.Time) *scheduleCoordinator {
	c := &scheduleCoordinator{roots: append([]string(nil), roots...), runs: operationalrun.Store{Namespace: namespace}, store: couchschedule.StateStore{Dir: filepath.Join(namespace, "schedule")}, location: location, now: now}
	sort.Strings(c.roots)
	snapshot := couchschedule.BuildStatus(nil, location.String())
	snapshot.Hubs = append([]string(nil), c.roots...)
	snapshot.Error = "waiting for scheduler catalogue"
	c.snapshot.Store(&snapshot)
	c.stopOwner = func(a couchschedule.Attempt) error {
		return couchschedule.StopOwner(couchschedule.OwnerIdentity{PID: a.ControllerPID, Start: a.ControllerStart})
	}
	c.ownerIdentity = func(pid int) string {
		if pid <= 0 {
			return ""
		}
		return procutil.StrictIdentity(strconv.Itoa(pid))
	}
	return c
}

func (c *scheduleCoordinator) view() couchschedule.StatusSnapshot {
	old := c.snapshot.Load()
	if old == nil {
		return couchschedule.StatusSnapshot{Version: 1, Tasks: []couchschedule.TaskStatus{}, Hubs: append([]string(nil), c.roots...), Error: "scheduler unavailable"}
	}
	copy := *old
	copy.Hubs = append([]string(nil), old.Hubs...)
	copy.Tasks = append([]couchschedule.TaskStatus{}, old.Tasks...)
	for i := range copy.Tasks {
		if copy.Tasks[i].Active != nil {
			active := *copy.Tasks[i].Active
			active.Unknown = strictjson.CloneRaw(active.Unknown)
			active.NoticeBinding = append(active.NoticeBinding[:0:0], active.NoticeBinding...)
			copy.Tasks[i].Active = &active
		}
		if copy.Tasks[i].Last != nil {
			last := *copy.Tasks[i].Last
			last.Unknown = strictjson.CloneRaw(last.Unknown)
			if last.Exit != nil {
				code := *last.Exit
				last.Exit = &code
			}
			copy.Tasks[i].Last = &last
		}
	}
	return copy
}

func (c *scheduleCoordinator) blocked(err error) error {
	snapshot := c.view()
	snapshot.Error = err.Error()
	if len(snapshot.Error) > 4096 {
		snapshot.Error = snapshot.Error[:4096]
	}
	c.snapshot.Store(&snapshot)
	return err
}

func (c *scheduleCoordinator) publish(ctx context.Context) error {
	ledger, err := c.store.View(ctx)
	if err != nil {
		return c.blocked(err)
	}
	states := make([]couchschedule.TaskState, 0, len(ledger.Tasks))
	configured := map[string]bool{}
	for _, root := range c.roots {
		configured[root] = true
	}
	for _, state := range ledger.Tasks {
		if configured[state.Key.Hub] {
			states = append(states, state)
		}
	}
	snapshot := couchschedule.BuildStatus(states, c.location.String())
	snapshot.Hubs = append([]string(nil), c.roots...)
	c.snapshot.Store(&snapshot)
	return nil
}
