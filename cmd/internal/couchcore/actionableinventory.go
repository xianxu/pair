package couchcore

import (
	"context"
	"sort"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// ActionableThreadState is the complete state vocabulary of the ordinary
// switcher, and it is TOTAL: every record in the manifest gets one. Records
// whose lifecycle cannot be proved used to be absent, which meant the switcher
// showed four rows over a store of thirteen and could not say why (#181).
type ActionableThreadState string

const (
	ThreadLive   ActionableThreadState = "live"
	ThreadParked ActionableThreadState = "parked"
	// ThreadBusy is a park transaction in flight: not actionable, but not
	// broken either, and it resolves on its own.
	ThreadBusy ActionableThreadState = "busy"
	// ThreadUnusable is a real thread the operator cannot act on right now.
	// It always carries a ThreadReason.
	ThreadUnusable ActionableThreadState = "unusable"
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
	// Agent and NativeID carry the same resume proof ParkedResumeObservation
	// does, so the PURE projector enforces it rather than trusting the IO shell
	// to have filtered its candidates. An observation without them is not
	// evidence of a resumable thread, only of a running session.
	Agent    string
	NativeID string
}

// ProofStatus records whether the IO shell managed to ASK a question, as
// distinct from what the answer was.
//
// Without it a total classifier cannot tell "no session" from "we could not
// look", and turns every unresolved question into a positive claim -- one
// failed zellij query would assert session-gone on every detached row.
type ProofStatus uint8

const (
	// ProofUnresolved means the question was never asked, or asking failed.
	ProofUnresolved ProofStatus = iota
	// ProofResolved means the question was asked and answered, positively or not.
	ProofResolved
)

// ThreadEvidence is everything the IO shell resolved about one record, and
// whether it managed to resolve it.
//
// The shell gathers; it decides nothing. That split is the point: refusals used
// to be `continue` statements in an IO loop, so no test could see them and no
// row could report one (ARCH-PURE).
type ThreadEvidence struct {
	// Live is this console's proof that it hosts the recorded process.
	Live []ProcessIdentity
	// Parked and Detached are the two resume proofs, each with the status of
	// the question that produced it.
	Parked         []ParkedResumeObservation
	ParkedStatus   ProofStatus
	Detached       []DetachedSessionObservation
	DetachedStatus ProofStatus
	// PathError is a working path that could not be physicalized.
	PathError error
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
	// Reason is set exactly when State is ThreadUnusable, and says why.
	Reason       ThreadReason `json:"reason,omitempty"`
	LastActiveAt time.Time    `json:"last_active_at,omitempty"`
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
		// Bridge, replaced in the total form: these callers only ever pass
		// proofs they successfully resolved, so both questions are answered.
		state, _ := ClassifyThread(record, ThreadEvidence{
			Live:           observed[record.Address],
			Parked:         resumeProofs[record.Address],
			ParkedStatus:   ProofResolved,
			Detached:       detachedProofs[record.Address],
			DetachedStatus: ProofResolved,
		})
		if state != ThreadLive && state != ThreadParked && state != ThreadDetached {
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

// ClassifyThread is the single, TOTAL lifecycle rule: every record and its
// evidence produce a state, and an unusable one always says why.
//
// Branch order is load-bearing and follows the projector it replaced. The LIVE
// branch comes before the resume-shaped refusals because those refusals never
// touched a live row: the shell skipped records carrying an incarnation, so
// such a record was never physicalized and its profile was never read. Ordering
// path/profile/agent first would make a running agent whose directory moved
// classify unusable -- a behaviour change this does not make. Detached is
// checked only for records with zero incarnations, which is what keeps a stale
// incarnation from ever masquerading as a clean detach.
func ClassifyThread(record ThreadRecord, evidence ThreadEvidence) (ActionableThreadState, ThreadReason) {
	if ValidateThreadRecord(record) != nil {
		return ThreadUnusable, ReasonInvalid
	}
	if record.Reservation {
		return ThreadUnusable, ReasonNeverStarted
	}
	if record.Park != nil {
		return ThreadBusy, ""
	}
	if len(record.Incarnations) != 0 {
		if liveProofMatches(record, evidence.Live) {
			return ThreadLive, ""
		}
		return ThreadUnusable, ReasonStaleIncarnation
	}
	if len(evidence.Live) != 0 {
		return ThreadUnusable, ReasonUnrecordedChild
	}
	// Everything below is resume-shaped, and only here does the shell's
	// resolution of paths, profiles and proofs matter.
	if evidence.PathError != nil {
		return ThreadUnusable, ReasonPathMissing
	}
	if record.LatestLaunchProfile == nil {
		return ThreadUnusable, ReasonProfileMissing
	}
	if !launcher.IsSupportedAgent(record.LatestLaunchProfile.Agent) || record.LatestLaunchProfile.Argv == nil {
		return ThreadUnusable, ReasonAgentUnsupported
	}
	if record.VerifiedPark != nil {
		switch {
		case evidence.ParkedStatus == ProofUnresolved:
			return ThreadUnusable, ReasonUnknown
		case parkedResumeProofMatches(record, evidence.Parked):
			return ThreadParked, ""
		default:
			return ThreadUnusable, ReasonBindingLost
		}
	}
	switch {
	case evidence.DetachedStatus == ProofUnresolved:
		// Not "no session" -- "we could not ask".
		return ThreadUnusable, ReasonUnknown
	case detachedResumeProofMatches(record, evidence.Detached):
		return ThreadDetached, ""
	case len(evidence.Detached) != 0:
		// A live session whose binding is unusable: pair#168's shape.
		return ThreadUnusable, ReasonBindingLost
	default:
		return ThreadUnusable, ReasonSessionGone
	}
}

// liveProofMatches is the live rule, sitting beside the two proof matchers it
// is a sibling of: one recorded incarnation, one hosted process, exact
// identity match so a recycled PID cannot pass.
func liveProofMatches(record ThreadRecord, observations []ProcessIdentity) bool {
	if len(record.Incarnations) != 1 || len(observations) != 1 {
		return false
	}
	incarnation := record.Incarnations[0]
	if incarnation.State != IncarnationLive || incarnation.PID <= 0 || incarnation.Identity == "" {
		return false
	}
	return observations[0] == ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity}
}

// detachedResumeProofMatches is parkedResumeProofMatches' twin, and the
// symmetry is the point: both resumable kinds prove themselves the same way, in
// the same layer.
//
// The binding requirement used to live only in the IO shell, which made this
// function's own "fails closed on its own" claim false -- a caller that forgot
// to gate its candidates would have got rows resume cannot take. Startup has no
// fallback, so that is not a degraded row, it is `couch` refusing to start.
func detachedResumeProofMatches(record ThreadRecord, observations []DetachedSessionObservation) bool {
	if record.LatestLaunchProfile == nil || !launcher.IsSupportedAgent(record.LatestLaunchProfile.Agent) || record.LatestLaunchProfile.Argv == nil || len(observations) != 1 {
		return false
	}
	observation := observations[0]
	return observation.Address == record.Address && observation.SessionName != "" &&
		observation.Agent == record.LatestLaunchProfile.Agent && observation.NativeID != ""
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
	var detachedCandidates []DetachedCandidate
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
			detachedCandidates = append(detachedCandidates, DetachedCandidate{
				Address: record.Address, Agent: agent, NativeID: binding.NativeID,
			})
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
