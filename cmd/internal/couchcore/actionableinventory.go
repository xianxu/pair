package couchcore

import (
	"context"
	"sort"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
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

// ParkedResumeObservation is exact resume authority for one inactive thread.
// The projector accepts it only when it is the sole observation for the
// address and agrees with the thread's saved launch agent.
type ParkedResumeObservation struct {
	Address  ThreadAddress
	Agent    string
	NativeID string
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
func ProjectActionableThreads(records []ThreadRecord, observations []LiveTTYObservation, resumable []ParkedResumeObservation) []ActionableThreadSummary {
	observed := make(map[ThreadAddress][]ProcessIdentity, len(observations))
	for _, observation := range observations {
		observed[observation.Address] = append(observed[observation.Address], observation.Process)
	}
	resumeProofs := make(map[ThreadAddress][]ParkedResumeObservation, len(resumable))
	for _, observation := range resumable {
		resumeProofs[observation.Address] = append(resumeProofs[observation.Address], observation)
	}

	rows := make([]ActionableThreadSummary, 0, len(records))
	for _, record := range records {
		if ValidateThreadRecord(record) != nil {
			continue
		}
		state, ok := actionableThreadState(record, observed[record.Address], resumeProofs[record.Address])
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

func actionableThreadState(record ThreadRecord, observations []ProcessIdentity, resumable []ParkedResumeObservation) (ActionableThreadState, bool) {
	if record.Reservation || record.Park != nil {
		return "", false
	}
	if record.VerifiedPark != nil {
		if len(record.Incarnations) == 0 && len(observations) == 0 && parkedResumeProofMatches(record, resumable) {
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

func parkedResumeProofMatches(record ThreadRecord, observations []ParkedResumeObservation) bool {
	if record.LatestLaunchProfile == nil || !launcher.IsSupportedAgent(record.LatestLaunchProfile.Agent) || record.LatestLaunchProfile.Argv == nil || len(observations) != 1 {
		return false
	}
	observation := observations[0]
	return observation.Address == record.Address && observation.Agent == record.LatestLaunchProfile.Agent && observation.NativeID != ""
}

// ActionableThreadInventory takes one durable snapshot and delegates every
// lifecycle decision to ProjectActionableThreads.
func (c *Couch) ActionableThreadInventory(observations []LiveTTYObservation) ([]ActionableThreadSummary, error) {
	return c.ActionableThreadInventoryContext(context.Background(), observations)
}

func (c *Couch) ActionableThreadInventoryContext(ctx context.Context, observations []LiveTTYObservation) ([]ActionableThreadSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	snapshot, err := c.Threads.Snapshot()
	if err != nil {
		return nil, err
	}
	var resumable []ParkedResumeObservation
	resolver, _ := c.Artifacts.(NativeBindingResolver)
	if resolver != nil {
		for _, record := range snapshot.Records {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if record.VerifiedPark == nil || record.Park != nil || len(record.Incarnations) != 0 || record.LatestLaunchProfile == nil {
				continue
			}
			agent := record.LatestLaunchProfile.Agent
			if !launcher.IsSupportedAgent(agent) || record.LatestLaunchProfile.Argv == nil || c.Path == nil {
				continue
			}
			if _, pathErr := c.Path.Physical(record.WorkingPath); pathErr != nil {
				continue
			}
			binding, resolveErr := resolver.ResolveEstablished(ctx, record.Address.RepoScope, string(record.Address.Tag), agent)
			if resolveErr != nil || bindingResumeDiagnostic(binding) != "" {
				continue
			}
			resumable = append(resumable, ParkedResumeObservation{Address: record.Address, Agent: agent, NativeID: binding.NativeID})
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return ProjectActionableThreads(snapshot.Records, observations, resumable), nil
}
