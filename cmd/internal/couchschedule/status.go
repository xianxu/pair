package couchschedule

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"
)

type TaskStatus struct {
	Hub      string      `json:"hub"`
	Task     string      `json:"task"`
	Revision string      `json:"revision"`
	Scope    string      `json:"scope"`
	Owner    string      `json:"owner,omitempty"`
	Cron     string      `json:"cron"`
	Location string      `json:"location"`
	State    string      `json:"state"`
	Reason   string      `json:"reason,omitempty"`
	NextDue  time.Time   `json:"next_due"`
	Last     *RunSummary `json:"last,omitempty"`
	Active   *Attempt    `json:"active,omitempty"`
}

type StatusSnapshot struct {
	Version int          `json:"version"`
	Error   string       `json:"error,omitempty"`
	Hubs    []string     `json:"hubs"`
	Tasks   []TaskStatus `json:"tasks"`
}

// BuildStatus projects only supplied values and detaches pointers before
// publication. Neither rendering nor validation refreshes the filesystem.
func BuildStatus(states []TaskState, location string) StatusSnapshot {
	snapshot := StatusSnapshot{Version: 1, Tasks: make([]TaskStatus, 0, len(states))}
	hubs := map[string]bool{}
	for _, s := range states {
		if !hubs[s.Key.Hub] {
			snapshot.Hubs = append(snapshot.Hubs, s.Key.Hub)
			hubs[s.Key.Hub] = true
		}
		row := TaskStatus{Hub: s.Key.Hub, Task: s.Key.Task, Revision: s.Revision, Scope: s.Spec.Scope, Owner: s.Spec.Owner, Cron: s.Spec.Cron, Location: location, State: "waiting", Reason: s.Refusal, NextDue: s.NextDue}
		if s.Last != nil {
			copy := *s.Last
			if copy.Exit != nil {
				code := *copy.Exit
				copy.Exit = &code
			}
			row.Last = &copy
		}
		if s.Active != nil {
			copy := *s.Active
			row.Active = &copy
			row.State = copy.Stage
		}
		if !s.Allowed {
			row.State = "refused"
		}
		snapshot.Tasks = append(snapshot.Tasks, row)
	}
	sort.Strings(snapshot.Hubs)
	sort.Slice(snapshot.Tasks, func(i, j int) bool {
		a, b := snapshot.Tasks[i], snapshot.Tasks[j]
		if a.Hub != b.Hub {
			return a.Hub < b.Hub
		}
		return a.Task < b.Task
	})
	return snapshot
}

func (s StatusSnapshot) Validate() error {
	if s.Version != 1 || s.Tasks == nil || len(s.Tasks) > 256 || len(s.Hubs) > 256 {
		return fmt.Errorf("invalid schedule status version or task bound")
	}
	seen := map[string]bool{}
	hubs := map[string]bool{}
	for _, hub := range s.Hubs {
		if !filepath.IsAbs(hub) || filepath.Clean(hub) != hub || hubs[hub] {
			return fmt.Errorf("invalid schedule hub catalog")
		}
		hubs[hub] = true
	}
	for _, t := range s.Tasks {
		key := TaskKey{Hub: t.Hub, Task: t.Task}.ID()
		if !hubs[t.Hub] || !filepath.IsAbs(t.Hub) || filepath.Clean(t.Hub) != t.Hub || !ValidTaskName(t.Task) || seen[key] {
			return fmt.Errorf("invalid schedule status task identity")
		}
		seen[key] = true
		switch t.State {
		case "waiting", "refused", "reserved", "queued", "building", "running", "notifying", "delivery", "stopping", "uncertain":
		default:
			return fmt.Errorf("invalid schedule status stage %q", t.State)
		}
	}
	return nil
}
