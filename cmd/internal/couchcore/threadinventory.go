package couchcore

import "sort"

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
	// Parked distinguishes the two ways a thread can have no incarnation: a
	// verified park tore its session down and its agent is gone, while a
	// detached thread's agent is still running behind a live zellij session.
	// Without it the diagnostic view reports both as "no agent running" and
	// contradicts the switcher, which offers the detached row for reattach.
	Parked bool `json:"parked,omitempty"`
}

func (s ThreadSummary) Label() string {
	if s.Name != "" {
		return s.Name
	}
	return string(s.Address.Tag)
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

func BuildThreadInventory(records []ThreadRecord) []ThreadSummary {
	rows := make([]ThreadSummary, 0, len(records))
	for _, record := range records {
		cloned := cloneThreadRecord(record)
		rows = append(rows, ThreadSummary{
			Address:          cloned.Address,
			StartingPath:     cloned.StartingPath,
			WorkingPath:      cloned.WorkingPath,
			Name:             cloned.Name,
			Description:      cloned.Description,
			PublishedSummary: cloned.PublishedSummary,
			Incarnations:     cloned.Incarnations,
			Parked:           cloned.VerifiedPark != nil,
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
	snapshot, err := c.Threads.Snapshot()
	if err != nil {
		return nil, err
	}
	return BuildThreadInventory(snapshot.Records), nil
}
