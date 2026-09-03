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
	// ThreadDetached is a thread whose zellij session is still alive with no
	// client attached: the agent is running, only the view is gone. Reattaching
	// is a fresh `pair resume <tag>` onto the surviving session, which is why it
	// shares the resume path with ThreadParked rather than needing its own.
	ThreadDetached ActionableThreadState = "detached"
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

// DetachedSessionObservation is one thread whose zellij session is still alive
// with no client attached -- the state an `alt+d` detach leaves behind, and the
// one `pair resume` reattaches onto without recreating anything.
//
// Shaped like LiveTTYObservation and ParkedResumeObservation so the projector
// keeps one argument style: proof arrives as observations, never as persisted
// lifecycle state.
type DetachedSessionObservation struct {
	Address     ThreadAddress
	SessionName string
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

// Detached reports a thread whose agent is still running behind a client-less
// zellij session.
func (s ActionableThreadSummary) Detached() bool { return s.State == ThreadDetached }

// Resumable reports the states whose Enter reattaches rather than switches.
// Parked is cold (the session was torn down) and detached is warm (it survived),
// but both converge on one effect, so callers ask this rather than enumerating.
func (s ActionableThreadSummary) Resumable() bool {
	return s.State == ThreadParked || s.State == ThreadDetached
}

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
func ProjectActionableThreads(records []ThreadRecord, observations []LiveTTYObservation, resumable []ParkedResumeObservation, detached []DetachedSessionObservation) []ActionableThreadSummary {
	observed := make(map[ThreadAddress][]ProcessIdentity, len(observations))
	for _, observation := range observations {
		observed[observation.Address] = append(observed[observation.Address], observation.Process)
	}
	resumeProofs := make(map[ThreadAddress][]ParkedResumeObservation, len(resumable))
	for _, observation := range resumable {
		resumeProofs[observation.Address] = append(resumeProofs[observation.Address], observation)
	}
	detachedProofs := make(map[ThreadAddress][]DetachedSessionObservation, len(detached))
	for _, observation := range detached {
		detachedProofs[observation.Address] = append(detachedProofs[observation.Address], observation)
	}

	rows := make([]ActionableThreadSummary, 0, len(records))
	for _, record := range records {
		if ValidateThreadRecord(record) != nil {
			continue
		}
		state, ok := actionableThreadState(record, observed[record.Address], resumeProofs[record.Address], detachedProofs[record.Address])
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

func actionableThreadState(record ThreadRecord, observations []ProcessIdentity, resumable []ParkedResumeObservation, detached []DetachedSessionObservation) (ActionableThreadState, bool) {
	if record.Reservation || record.Park != nil {
		return "", false
	}
	if record.VerifiedPark != nil {
		if len(record.Incarnations) == 0 && len(observations) == 0 && parkedResumeProofMatches(record, resumable) {
			return ThreadParked, true
		}
		return "", false
	}
	// Detached is checked BEFORE the live rule and requires zero incarnations,
	// which is what keeps the projector fail-closed: a record still carrying an
	// incarnation -- the shape a crashed couch leaves -- falls through to the
	// live rule and is hidden there for want of a matching TTY observation. A
	// stale incarnation therefore can never masquerade as a clean detach.
	if len(record.Incarnations) == 0 {
		// The profile checks mirror parkedResumeProofMatches deliberately: this
		// function's contract is that it fails closed on its own, so it does not
		// rely on the caller having filtered candidates. Enter on a row whose
		// saved agent is unsupported cannot work, and a row that cannot work
		// must not be offered.
		if len(observations) == 0 && len(detached) == 1 &&
			detached[0].Address == record.Address && detached[0].SessionName != "" &&
			record.LatestLaunchProfile != nil &&
			launcher.IsSupportedAgent(record.LatestLaunchProfile.Agent) &&
			record.LatestLaunchProfile.Argv != nil {
			return ThreadDetached, true
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
	// detachedCandidates are the ONLY records that could be detached: no
	// incarnation, no verified park, and a saved profile to reattach with.
	// Passing candidates bounds WHETHER the zellij snapshot runs at all -- a
	// couch with nothing detachable pays nothing -- though not the snapshot's
	// own fan-out, which is per session on the host.
	var detachedCandidates []ThreadAddress
	resolver, _ := c.Artifacts.(NativeBindingResolver)
	for i := range snapshot.Records {
		record := snapshot.Records[i]
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if record.Reservation || record.Park != nil || len(record.Incarnations) != 0 || record.LatestLaunchProfile == nil {
			continue
		}
		agent := record.LatestLaunchProfile.Agent
		if !launcher.IsSupportedAgent(agent) || record.LatestLaunchProfile.Argv == nil || c.Path == nil {
			continue
		}
		// Physicalize for parked AND detached candidates alike. The startup
		// selector compares paths by exact string, so resolving one kind and
		// not the other would make an alias path match a parked row and miss
		// an otherwise identical detached one.
		physicalPath, pathErr := c.Path.Physical(record.WorkingPath)
		if pathErr != nil {
			continue
		}
		snapshot.Records[i].WorkingPath = physicalPath

		// The native-binding gate applies to BOTH resumable kinds, not just the
		// parked one. Startup has no fallback by design -- a Resume refusal
		// stops it rather than starting something new -- so a row the inventory
		// offers must be one resume can actually take. Gating parked rows and
		// not detached ones meant a thread whose agent session data was pruned,
		// rotated or raced was auto-selected and killed `couch` in that tree,
		// with detached being the NORMAL resting state since leave stopped
		// parking. Same rule actionableThreadState states for itself: a row
		// that cannot work must not be offered.
		if resolver == nil {
			continue
		}
		binding, resolveErr := resolver.ResolveEstablished(ctx, record.Address.RepoScope, string(record.Address.Tag), agent)
		if resolveErr != nil || bindingResumeDiagnostic(binding) != "" {
			continue
		}
		if record.VerifiedPark == nil {
			detachedCandidates = append(detachedCandidates, record.Address)
			continue
		}
		resumable = append(resumable, ParkedResumeObservation{Address: record.Address, Agent: agent, NativeID: binding.NativeID})
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var detached []DetachedSessionObservation
	if detachedResolver, ok := c.Artifacts.(DetachedSessionResolver); ok && len(detachedCandidates) > 0 {
		observed, detachErr := detachedResolver.DetachedSessions(ctx, detachedCandidates)
		if detachErr != nil {
			// One failed observation must not turn every OTHER row into an
			// authoritative absence, and it must not fail the whole refresh
			// either -- live and parked rows are proved by different evidence
			// and remain correct. Detached rows are simply not proved this
			// round, which is the fail-closed answer.
			detached = nil
		} else {
			detached = observed
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return ProjectActionableThreads(snapshot.Records, observations, resumable, detached), nil
}
