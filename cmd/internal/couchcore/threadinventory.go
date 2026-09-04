package couchcore

import (
	"context"
	"sort"
)

// ThreadSummary is the diagnostic CLI/advisor row. It preserves the composite
// address and every incarnation state; ordinary switchers use
// ActionableThreadSummary so undecodable lifecycle states remain invisible.
type ThreadSummary struct {
	Address          ThreadAddress       `json:"address"`
	StartingPath     string              `json:"starting_path"`
	WorkingPath      string              `json:"working_path"`
	Name             string              `json:"name,omitempty"`
	Description      string              `json:"description,omitempty"`
	PublishedSummary string              `json:"published_summary,omitempty"`
	Incarnations     []ThreadIncarnation `json:"incarnations,omitempty"`
	// State and Reason come from the SAME classifier the switcher uses. The
	// diagnostic view used to derive its own two-case answer from a `Parked`
	// bool, which is how one store could produce two different stories --
	// exactly what #181 exists to stop.
	State  ActionableThreadState `json:"state"`
	Reason ThreadReason          `json:"reason,omitempty"`
}

func (s ThreadSummary) Label() string {
	return threadLabel(s.Name, s.WorkingPath, s.Address.Tag)
}

func (s ThreadSummary) DisplaySummary() string {
	if s.PublishedSummary != "" {
		return s.PublishedSummary
	}
	return s.Description
}

func (s ThreadSummary) Live() bool {
	for _, incarnation := range s.Incarnations {
		if incarnation.State == IncarnationLive {
			return true
		}
	}
	return false
}

// BuildThreadInventory is the diagnostic projection: every record, with its
// lifecycle detail, classified by the one shared rule.
func BuildThreadInventory(records []ThreadRecord, evidence map[ThreadAddress]ThreadEvidence) []ThreadSummary {
	rows := make([]ThreadSummary, 0, len(records))
	for _, record := range records {
		cloned := cloneThreadRecord(record)
		state, reason := ClassifyThread(cloned, evidence[cloned.Address])
		rows = append(rows, ThreadSummary{
			Address:          cloned.Address,
			StartingPath:     cloned.StartingPath,
			WorkingPath:      cloned.WorkingPath,
			Name:             cloned.Name,
			Description:      cloned.Description,
			PublishedSummary: cloned.PublishedSummary,
			Incarnations:     cloned.Incarnations,
			State:            state,
			Reason:           reason,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Address.RepoScope != rows[j].Address.RepoScope {
			return rows[i].Address.RepoScope < rows[j].Address.RepoScope
		}
		return rows[i].Address.Tag < rows[j].Address.Tag
	})
	return rows
}

func (c *Couch) ThreadInventory() ([]ThreadSummary, error) {
	return c.ThreadInventoryContext(context.Background())
}

// ThreadInventoryContext resolves the same evidence the switcher does.
//
// A CLI has no console, so its live proof comes from the OS: the recorded
// process must still be alive AND carry the kernel start token recorded at
// launch. Without that a running thread would read as a stale incarnation here
// while the switcher called it live -- one store, two stories.
func (c *Couch) ThreadInventoryContext(ctx context.Context) ([]ThreadSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// No observations to pass: gatherThreadEvidence derives OS liveness for
	// every caller now, so the CLI and the console read the same proof and this
	// no longer needs its own snapshot to build one from.
	snapshot, evidence, err := c.gatherThreadEvidence(ctx, nil)
	if err != nil {
		return nil, err
	}
	return BuildThreadInventory(snapshot.Records, evidence), nil
}
