package couchcore

import (
	"context"
	"errors"
	"path/filepath"
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
	// ThreadArchived is a thread the operator retired. ClassifyThread never
	// returns it -- an archived record is out of the working set, so asking
	// whether its session is alive answers a question about a thread couch no
	// longer tracks. Only the archive listing sets it.
	ThreadArchived ActionableThreadState = "archived"
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
	return threadLabel(s.Name, s.WorkingPath, s.Address.Tag)
}

// threadLabel is what the operator reads instead of an opaque address.
//
// The tag is a 16-hex identity, unique and unmemorable; a switcher of them
// reads as noise. The working directory's last segment is what the operator
// actually calls the thread -- `brain`, `pair`, `arc-agi-3` -- and with one
// thread per path it identifies the row as well as the tag does. The tag stays
// the fallback for a record with no path, and `couch --show` still prints the
// full address, so nothing loses its exact identity.
func threadLabel(name, workingPath string, tag ThreadTag) string {
	if name != "" {
		return name
	}
	if base := filepath.Base(workingPath); base != "" && base != "." && base != string(filepath.Separator) {
		return base
	}
	return string(tag)
}

func (s ActionableThreadSummary) DisplaySummary() string {
	if s.PublishedSummary != "" {
		return s.PublishedSummary
	}
	return s.Description
}

// ThreadProjectionInput is everything a projection needs, as ONE value.
//
// The three parts used to travel separately, with the unreadable set as a
// trailing variadic -- so omitting it compiled cleanly and silently restored
// "some records get no row", which is the regression this issue exists to
// prevent. A consumer cannot opt out of part of a projection it is handed
// whole, and the next omission is a compile error rather than a hidden filter.
type ThreadProjectionInput struct {
	Records    []ThreadRecord
	Evidence   map[ThreadAddress]ThreadEvidence
	Unreadable []ThreadAddress
}

// FromSnapshot pairs a store snapshot with the evidence resolved about it, so
// the records and the addresses that could not become records stay together.
func FromSnapshot(snapshot ThreadSnapshot, evidence map[ThreadAddress]ThreadEvidence) ThreadProjectionInput {
	return ThreadProjectionInput{
		Records: snapshot.Records, Evidence: evidence, Unreadable: snapshot.Unreadable,
	}
}

// ProjectActionableThreads emits ONE ROW PER RECORD. It is total, and every
// row that is not actionable says why.
//
// It used to fail closed by omission: a record whose lifecycle could not be
// proved simply produced nothing, so the operator's switcher showed four rows
// over a store of thirteen and had no way to report the other nine. Failing
// closed is still the rule -- an unproved row is not actionable and startup
// will not select it -- but it is now expressed as a state rather than as
// absence (#181).
func ProjectActionableThreads(input ThreadProjectionInput) []ActionableThreadSummary {
	records, evidence := input.Records, input.Evidence
	rows := make([]ActionableThreadSummary, 0, len(records))
	// A record couch could not read is still a thread the operator has. It gets
	// a row carrying the only thing that could be read -- its address -- so it
	// can be seen rather than silently removing itself from the inventory.
	for _, address := range input.Unreadable {
		rows = append(rows, ActionableThreadSummary{
			Address: address, State: ThreadUnusable, Reason: ReasonUnreadable,
		})
	}
	for _, record := range records {
		state, reason := ClassifyThread(record, evidence[record.Address])
		rows = append(rows, ActionableThreadSummary{
			Address:          record.Address,
			StartingPath:     record.StartingPath,
			WorkingPath:      record.WorkingPath,
			Name:             record.Name,
			Description:      record.Description,
			PublishedSummary: record.PublishedSummary,
			State:            state,
			Reason:           reason,
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
		// A start in flight is not a crash. Between `start` and Pair
		// registration an incarnation is `creating` and nothing hosts it yet --
		// calling that "stale, couch exited unexpectedly" describes the normal
		// path of every thread ever started. What separates it from a start
		// that DIED is process evidence, not the recorded state: a creating
		// incarnation whose process is gone is stale in exactly the way the
		// word means.
		if startInFlight(record, evidence.Live) {
			return ThreadBusy, ""
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

// startInFlight reports a thread mid-start: an incarnation that does not yet
// claim to be live, whose process the caller can still see.
func startInFlight(record ThreadRecord, observations []ProcessIdentity) bool {
	for _, incarnation := range record.Incarnations {
		if incarnation.State == IncarnationLive {
			continue
		}
		for _, observation := range observations {
			if observation == (ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity}) {
				return true
			}
		}
	}
	return false
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
	snapshot, evidence, err := c.gatherThreadEvidence(ctx, observations)
	if err != nil {
		return nil, err
	}
	return ProjectActionableThreads(FromSnapshot(snapshot, evidence)), nil
}

// gatherThreadEvidence resolves what is knowable about every record and decides
// nothing. Both inventories consume it, so the switcher and the diagnostic view
// cannot derive different states from the same store (ARCH-DRY).
//
// It returns the snapshot too, because physicalizing a working path mutates the
// record the caller projects.
func (c *Couch) gatherThreadEvidence(ctx context.Context, observations []LiveTTYObservation) (ThreadSnapshot, map[ThreadAddress]ThreadEvidence, error) {
	if err := ctx.Err(); err != nil {
		return ThreadSnapshot{}, nil, err
	}
	snapshot, err := c.Threads.Snapshot()
	if err != nil {
		return ThreadSnapshot{}, nil, err
	}
	// Live proof is the UNION of what the caller can see: a console's own pty
	// children, plus any recorded process the OS still vouches for. Keeping
	// them separate is what let one store tell two stories -- the switcher
	// calling a thread stale because it does not host it, while `couch --list`
	// called the same thread live from the same records.
	observed := make(map[ThreadAddress][]ProcessIdentity, len(observations))
	seen := make(map[ThreadAddress]map[ProcessIdentity]bool, len(observations))
	add := func(address ThreadAddress, process ProcessIdentity) {
		if seen[address] == nil {
			seen[address] = map[ProcessIdentity]bool{}
		}
		if seen[address][process] {
			// Deduplicated on purpose: the live rule demands exactly one
			// observation, so counting the same process through both proofs
			// would classify a hosted thread as stale.
			return
		}
		seen[address][process] = true
		observed[address] = append(observed[address], process)
	}
	for _, observation := range observations {
		add(observation.Address, observation.Process)
	}
	for _, observation := range c.ObserveRecordedProcesses(snapshot.Records) {
		add(observation.Address, observation.Process)
	}

	evidence := make(map[ThreadAddress]ThreadEvidence, len(snapshot.Records))
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
			return ThreadSnapshot{}, nil, err
		}
		item := ThreadEvidence{Live: observed[record.Address]}
		// Physicalization and binding resolution are RESUME-SHAPED work. A
		// record carrying an incarnation never reached either before, and must
		// not start to: a running agent whose directory moved is still running.
		// This is one contract, and the call-count guard is what binds it.
		resumeShaped := !record.Reservation && record.Park == nil && len(record.Incarnations) == 0 &&
			len(item.Live) == 0 && record.LatestLaunchProfile != nil &&
			launcher.IsSupportedAgent(record.LatestLaunchProfile.Agent) &&
			record.LatestLaunchProfile.Argv != nil
		if !resumeShaped {
			evidence[record.Address] = item
			continue
		}
		switch {
		case c.Path == nil:
			item.PathError = errors.New("path operations are unavailable")
		default:
			// Physicalize for parked AND detached candidates alike. The startup
			// selector compares paths by exact string, so resolving one kind
			// and not the other would make an alias path match a parked row and
			// miss an otherwise identical detached one.
			physicalPath, pathErr := c.Path.Physical(record.WorkingPath)
			if pathErr != nil {
				item.PathError = pathErr
			} else {
				snapshot.Records[i].WorkingPath = physicalPath
			}
		}
		if item.PathError != nil || resolver == nil {
			evidence[record.Address] = item
			continue
		}
		agent := record.LatestLaunchProfile.Agent
		binding, resolveErr := resolver.ResolveEstablished(ctx, record.Address.RepoScope, string(record.Address.Tag), agent)
		if record.VerifiedPark != nil {
			// The parked question is answered either way: a refusal is a
			// resolved "no binding", not an unresolved question.
			item.ParkedStatus = ProofResolved
			if resolveErr == nil && bindingResumeDiagnostic(binding) == "" {
				resumable = append(resumable, ParkedResumeObservation{
					Address: record.Address, Agent: agent, NativeID: binding.NativeID,
				})
			}
			evidence[record.Address] = item
			continue
		}
		// A detach candidate regardless of binding health: whether the SESSION
		// is alive is a different question from whether the agent's transcript
		// id resolved, and conflating them is what hid a live detached thread.
		//
		// The id itself still travels only when it RESOLVED CLEANLY. That keeps
		// the proof exactly as strict as it was -- an ambiguous binding carries
		// a non-empty NativeID, so passing it through unchecked would make such
		// a row actionable and offer it to startup selection, which is not a
		// change this milestone makes. What the candidate buys here is the
		// answer to the session question, which turns an invisible row into a
		// visible `binding-lost` one instead of an asserted `session-gone`.
		nativeID := ""
		if resolveErr == nil && bindingResumeDiagnostic(binding) == "" {
			nativeID = binding.NativeID
		}
		detachedCandidates = append(detachedCandidates, DetachedCandidate{
			Address: record.Address, Agent: agent, NativeID: nativeID,
		})
		evidence[record.Address] = item
	}
	if err := ctx.Err(); err != nil {
		return ThreadSnapshot{}, nil, err
	}

	for _, observation := range resumable {
		item := evidence[observation.Address]
		item.Parked = append(item.Parked, observation)
		evidence[observation.Address] = item
	}

	// A couch with no session resolver cannot ask the question at all, so its
	// candidates stay unresolved rather than being told their sessions are
	// gone. Production always has one; this is the degraded path, and the
	// honest answer there is "unknown".
	if detachedResolver, ok := c.Artifacts.(DetachedSessionResolver); ok && len(detachedCandidates) > 0 {
		observed, detachErr := detachedResolver.DetachedSessions(ctx, detachedCandidates)
		if detachErr == nil {
			// Only now is the detached question answered. On failure every
			// candidate keeps ProofUnresolved, which classifies `unknown`
			// rather than asserting a session is gone -- the difference
			// between a row that waits and a row retirement acts on.
			for _, candidate := range detachedCandidates {
				item := evidence[candidate.Address]
				item.DetachedStatus = ProofResolved
				evidence[candidate.Address] = item
			}
			for _, observation := range observed {
				item := evidence[observation.Address]
				item.Detached = append(item.Detached, observation)
				evidence[observation.Address] = item
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return ThreadSnapshot{}, nil, err
	}
	return snapshot, evidence, nil
}

// ObserveRecordedProcesses derives live proof from the OS rather than from a
// console's own children.
//
// The switcher's live proof is "I am hosting this pty", which only the couch
// holding the terminal can supply. A CLI has no console, so without this it
// would classify every running thread as a stale incarnation and disagree with
// the switcher about the same store -- the exact split (#181) exists to close.
// The defence against a recycled PID is the same either way: the kernel start
// token must match the one recorded at launch.
func (c *Couch) ObserveRecordedProcesses(records []ThreadRecord) []LiveTTYObservation {
	if c == nil || c.Proc == nil {
		return nil
	}
	var observations []LiveTTYObservation
	for _, record := range records {
		for _, incarnation := range record.Incarnations {
			// Every recorded incarnation, not only the ones claiming to be
			// live: a `creating` incarnation is what a start in flight looks
			// like, and whether its process still exists is the difference
			// between "starting" and "the start died".
			if incarnation.PID <= 0 || incarnation.Identity == "" {
				continue
			}
			if c.Proc.Exists(incarnation.PID) != Live {
				continue
			}
			identity, err := c.Proc.Identity(incarnation.PID)
			if err != nil || identity != incarnation.Identity {
				continue
			}
			observations = append(observations, LiveTTYObservation{
				Address: record.Address,
				Process: ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity},
			})
		}
	}
	return observations
}

// LabelRow is one row's identity for display: what it would like to be called,
// and the address that makes it unique.
type LabelRow struct {
	Address ThreadAddress
	Label   string
}

// LabelsFor builds the disambiguated label map from anything that can report an
// address and a label, so the two renderers share the adapter instead of each
// writing the same loop over a different summary type (ARCH-DRY).
func LabelsFor[T any](rows []T, address func(T) ThreadAddress, label func(T) string) map[ThreadAddress]string {
	out := make([]LabelRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, LabelRow{Address: address(row), Label: label(row)})
	}
	return DisambiguateLabels(out)
}

// DisambiguateLabels gives every row a label the operator can act on.
//
// A thread's label is its directory's last segment, which is readable and, with
// one thread per path, unique. A store that predates that rule is not: the
// operator's holds six rows for `brain`, and six rows all reading `brain` is
// worse than six opaque tags -- readable and useless. Colliding rows keep the
// name and gain the short tail of their tag, so the common case stays clean and
// the exceptional one stays actionable.
func DisambiguateLabels(rows []LabelRow) map[ThreadAddress]string {
	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.Label]++
	}
	out := make(map[ThreadAddress]string, len(rows))
	for _, row := range rows {
		if counts[row.Label] < 2 {
			out[row.Address] = row.Label
			continue
		}
		tag := string(row.Address.Tag)
		if len(tag) > 8 {
			tag = tag[len(tag)-8:]
		}
		out[row.Address] = row.Label + "·" + tag
	}
	return out
}
