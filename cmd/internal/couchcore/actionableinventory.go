package couchcore

import (
	"sort"
	"time"
)

// ActionableThreadState is the complete state vocabulary of the ordinary
// switcher. Records whose lifecycle cannot be proved are intentionally absent.
type ActionableThreadState string

const (
	ThreadLive   ActionableThreadState = "live"
	ThreadParked ActionableThreadState = "parked"
)

// LiveTTYObservation is the owner's current proof that a terminal process is
// still the exact process recorded by the durable thread incarnation.
type LiveTTYObservation struct {
	Address ThreadAddress
	Process ProcessIdentity
}

// ActionableThreadSummary contains only fields the ordinary switcher needs.
// It deliberately excludes diagnostic lifecycle state.
type ActionableThreadSummary struct {
	Address          ThreadAddress         `json:"address"`
	StartingPath     string                `json:"starting_path"`
	WorkingPath      string                `json:"working_path"`
	Name             string                `json:"name,omitempty"`
	Description      string                `json:"description,omitempty"`
	PublishedSummary string                `json:"published_summary,omitempty"`
	State            ActionableThreadState `json:"state"`
	LastActiveAt     time.Time             `json:"last_active_at,omitempty"`
}

func (s ActionableThreadSummary) Live() bool { return s.State == ThreadLive }

func (s ActionableThreadSummary) Label() string {
	if s.Name != "" {
		return s.Name
	}
	return string(s.Address.Tag)
}

func (s ActionableThreadSummary) DisplaySummary() string {
	if s.PublishedSummary != "" {
		return s.PublishedSummary
	}
	return s.Description
}

// ProjectActionableThreads fails closed: a row is emitted only when one
// durable lifecycle state has one unambiguous proof.
func ProjectActionableThreads(records []ThreadRecord, observations []LiveTTYObservation) []ActionableThreadSummary {
	observed := make(map[ThreadAddress][]ProcessIdentity, len(observations))
	for _, observation := range observations {
		observed[observation.Address] = append(observed[observation.Address], observation.Process)
	}

	rows := make([]ActionableThreadSummary, 0, len(records))
	for _, record := range records {
		state, ok := actionableThreadState(record, observed[record.Address])
		if !ok {
			continue
		}
		rows = append(rows, ActionableThreadSummary{
			Address:          record.Address,
			StartingPath:     record.StartingPath,
			WorkingPath:      record.WorkingPath,
			Name:             record.Name,
			Description:      record.Description,
			PublishedSummary: record.PublishedSummary,
			State:            state,
			LastActiveAt:     record.LastActiveAt,
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

func actionableThreadState(record ThreadRecord, observations []ProcessIdentity) (ActionableThreadState, bool) {
	if record.Reservation || record.Park != nil {
		return "", false
	}
	if record.VerifiedPark != nil {
		if len(record.Incarnations) == 0 && len(observations) == 0 {
			return ThreadParked, true
		}
		return "", false
	}
	if len(record.Incarnations) != 1 || len(observations) != 1 {
		return "", false
	}
	incarnation := record.Incarnations[0]
	if incarnation.State != IncarnationLive || incarnation.PID <= 0 || incarnation.Identity == "" {
		return "", false
	}
	if observations[0] != (ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity}) {
		return "", false
	}
	return ThreadLive, true
}

// ActionableThreadInventory takes one durable snapshot and delegates every
// lifecycle decision to ProjectActionableThreads.
func (c *Couch) ActionableThreadInventory(observations []LiveTTYObservation) ([]ActionableThreadSummary, error) {
	snapshot, err := c.Threads.Snapshot()
	if err != nil {
		return nil, err
	}
	return ProjectActionableThreads(snapshot.Records, observations), nil
}
