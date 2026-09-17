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
	ThreadLive ActionableThreadState = "live"
	// ThreadParked is a thread with no agent running that couch can start one
	// for, on the exact conversation it left off in.
	//
	// The authority is the LEDGER, not `record.VerifiedPark` (#256 M2): a
	// receipt names a ParkIdentity and no conversation, so it attests that a
	// park happened and never that anything survived it. Two records therefore
	// produce this state -- one parked deliberately, one whose session simply
	// died while its ledger kept the conversation -- and every consumer must
	// accept BOTH. That enumeration is the rule
	// `TestEveryParkedProducerIsAcceptedByEveryActionTheMenuOffers` pins, after
	// a widened producer set reached one reader that had not been swept.
	ThreadParked ActionableThreadState = "parked"
	// ThreadBusy is a START couch has claimed and not yet finished -- see
	// startClaimed. It is NOT a park in flight: #256 removed `record.Park` from
	// the classification path entirely, because a park whose owner died read
	// "parking in progress" forever with no timeout and no owner check (#271).
	//
	// Its predecessor's doc said "it resolves on its own", which was the bug
	// stated as a comment. A start claim does resolve -- reconcileInterruptedStarts
	// runs at couch.New (couch.go) -- but a row that reads busy must still have
	// an escape, because ReconcileStart deliberately keeps a claim occupied when
	// its evidence is Unknown.
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
	// Agent correlates the requested launch profile. Native conversation
	// evidence is absent: reattachment consumes only the surviving session.
	Agent string
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
	// Session is this thread's zellij session: present, absent, or unresolved.
	// Unlike Parked/Detached it needs no separate status field -- its zero value
	// IS unresolved, so a record no gather branch reached cannot claim absence.
	//
	// Gathered for EVERY record, including ones carrying an incarnation or an
	// open park. That is the difference #256 turns on: the resume-shaped gate
	// below meant a record with an incarnation was never asked about its
	// session, so the evidence that its agent survived was never collected.
	Session SessionObservation
	// Parked is COLD-resume proof: the native conversation this thread would
	// resume into, read from its ledger. Warm reattachment needs no proof here
	// -- the session's own presence is the proof, and the ACTION path
	// re-observes attach state before committing (resume.go), which is where a
	// `list-clients` is worth its ~250 ms.
	//
	// Since #256 M2 it is gathered for EVERY record whose session is not
	// present, not only ones carrying a VerifiedPark receipt, and it is the
	// AUTHORITY for `parked` rather than corroboration of the receipt. The
	// receipt names a ParkIdentity and no conversation, so it could never have
	// answered this question; gating the read on it meant a thread whose session
	// died without a park was never asked, and read `session-gone`.
	//
	// ParkedStatus is the third value: unresolved means the ledger was not read
	// or could not be, never that there is nothing to resume.
	Parked       []ParkedResumeObservation
	ParkedStatus ProofStatus
	// StartOwner is the liveness of the couch process that CLAIMED this
	// thread's start -- not the helper it was starting. The two are different
	// processes, and a claim outlives its claimant.
	//
	// Its zero value is Unknown (procops.go:27), so a record no gather branch
	// reached cannot have its start declared driverless. That is the same
	// fail-closed-by-construction shape as SessionUnresolved, and it is what
	// makes the multiple-tracked-starts case safe: CurrentStartTransaction
	// refuses to resolve one, so nothing is written here at all.
	StartOwner Liveness
	// PathError is a working path that could not be physicalized.
	PathError error
}

// ActionableThreadSummary contains only fields the ordinary switcher needs.
// It deliberately excludes diagnostic lifecycle state.
type ActionableThreadSummary struct {
	Recovery         *RecoveryDecision     `json:"recovery,omitempty"`
	Continuation     *ContinuationStatus   `json:"continuation,omitempty"`
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
	// Layout is the thread's witnessed pair layout, already normalized: the
	// projection runs NormalizeLayout, so a record predating #198 reads as
	// Layout2 here and an unreadable one as LayoutUnknown. Consumers compare
	// it directly and never re-parse.
	Layout Layout `json:"layout,omitempty"`
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
// prevent. The struct does not make omission impossible -- a literal can still
// leave `Unreadable` unset -- but it makes the omission NAMED and visible at
// every construction site, where a trailing variadic was invisible by
// construction. `FromSnapshot` is the form that cannot forget.
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
			// Normalized here too, not just on the readable branch: the field
			// documents itself as always normalized, and a struct with a
			// documented invariant must satisfy it at EVERY construction site
			// or the invariant is only a comment. The record could not be read,
			// so its layout is unknown by definition -- which is exactly what
			// NormalizeLayout says about an unreadable value.
			Layout: NormalizeLayout("unreadable"),
		})
	}
	for _, record := range records {
		state, reason := ClassifyThread(record, evidence[record.Address])
		rows = append(rows, ActionableThreadSummary{
			Address:          record.Address,
			Continuation:     continuationStatus(record),
			Recovery:         ProjectRecoveryChoices(record, evidence[record.Address], state, reason),
			StartingPath:     record.StartingPath,
			WorkingPath:      record.WorkingPath,
			Name:             record.Name,
			Description:      record.Description,
			PublishedSummary: record.PublishedSummary,
			State:            state,
			Reason:           reason,
			LastActiveAt:     record.LastActiveAt,
			// THE normalization point for the layout witness: the raw persisted
			// value is untrusted, and "" (a pre-#198 record) must read as
			// Layout2 here regardless of the default for new Couch processes.
			Layout: NormalizeLayout(string(record.Layout)),
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
// Branch order is load-bearing, and the rule it encodes is a DELETION: the
// classifier reads the WORLD, not couch's bookkeeping about the world.
//
// `record.Incarnations` liveness fields and `record.Park` appear nowhere below.
// They named the launcher process, which is couch's own child and dies with it,
// so a proof keyed to them reported every crash as a lost thread (#272) and an
// unfinished park as a thread that would never be usable again (#271). The
// measurement behind that: the zellij server is PPID 1 AT BIRTH, so a couch
// death kills only the launcher -- which means a clean `alt+d` detach and a
// couch crash leave IDENTICAL external state. The only thing that used to tell
// them apart was whether couch survived long enough to write a record, and that
// difference decided `detached` (recoverable) versus `stale` (debris).
//
// The one thing still read from an incarnation is a ThreadStartClaim, and that
// is not a liveness claim: it is couch's durable record of its OWN in-flight
// operation, with recovery semantics of its own (see startClaimed).
func ClassifyThread(record ThreadRecord, evidence ThreadEvidence) (ActionableThreadState, ThreadReason) {
	if ValidateThreadRecord(record) != nil {
		return ThreadUnusable, ReasonInvalid
	}
	if record.Reservation {
		return ThreadUnusable, ReasonNeverStarted
	}
	// A start couch claimed, has not finished, and is still driving. This must
	// precede every session verdict: between claiming a start and the launcher
	// acquiring a pid there is no session yet, and without this branch a thread
	// starting NORMALLY would classify `session-gone` -- an archive-eligible
	// reason.
	if startInFlight(record, evidence) {
		return ThreadBusy, ""
	}
	// Couch hosts this thread's process right now. The union behind
	// evidence.Live is POSITIVE-ONLY: its absence proves nothing and falls
	// through to the session, which is the whole of #272's fix. It stays a
	// union (console pty children plus OS-vouched recorded processes) because
	// the CLI passes no observations of its own and would otherwise read every
	// running thread as detached -- #181's "one store, two stories".
	if len(evidence.Live) != 0 {
		return ThreadLive, ""
	}
	// Resume authority gates BOTH the warm and the cold path, and must be
	// checked before either. Reattaching still goes through DecideResume, which
	// needs the path and the profile -- so a row shown `detached` without them
	// would be offered a resume that always fails, the precise anti-pattern this
	// issue exists to remove. These are checked AFTER the live branch, because a
	// thread couch is hosting is not made unusable by a directory that moved.
	if evidence.PathError != nil {
		return ThreadUnusable, ReasonPathMissing
	}
	if record.LatestLaunchProfile == nil {
		return ThreadUnusable, ReasonProfileMissing
	}
	if !launcher.IsSupportedAgent(record.LatestLaunchProfile.Agent) || record.LatestLaunchProfile.Argv == nil {
		return ThreadUnusable, ReasonAgentUnsupported
	}
	if evidence.Session.State == SessionPresent {
		// The session outlived whatever hosted it. Reattaching is a fresh
		// `pair resume` onto the surviving session; no cold-resume proof is
		// needed, because nothing is being relaunched. Warm beats cold, so this
		// stays above the park branch.
		return ThreadDetached, ""
	}
	// COLD-RESUME authority is the LEDGER, and it is DURABLE: it does not depend
	// on the session existing, because a parked thread's session was torn down
	// on purpose. So it is consulted before the unresolved-session refusal.
	//
	// Ordering the refusal first would mean one failed `list-sessions` demoted
	// EVERY resumable row in the store to `unusable/unknown`, despite their
	// resume authority being intact. Fail-closed is right where the unknown
	// answer could flip an actionable verdict into a destructive one; here the
	// only two verdicts a session answer can produce for a record with a
	// resolvable conversation are `detached` and `parked`, both actionable, so
	// refusing both is strictly worse than taking the durable one.
	//
	// `record.VerifiedPark` is NOT the authority and no longer gates this. It is
	// a receipt: it carries a ParkIdentity and no native conversation id, so it
	// can attest that a park happened and never that there is something to
	// resume into. Letting it decide meant a thread whose session died without a
	// park was never asked about its ledger at all, and read `session-gone` --
	// archive-eligible -- with a live conversation still recorded. The receipt
	// survives below only to say WHICH kind of loss this is.
	// An unanswerable session question stops a COLD verdict here -- unless couch
	// itself tore the session down, which is the one thing `record.VerifiedPark`
	// does attest.
	//
	// The asymmetry is the point. For a deliberately parked thread the session
	// answer is uninformative (it was quiesced on purpose), so refusing every
	// parked row because one `list-sessions` failed is strictly worse than
	// taking the durable authority. For every other thread the session may be
	// ALIVE, and `parked` invites a cold resume, which relaunches an agent --
	// so a second agent would join a conversation that already has one. The
	// action path re-observes before committing, but a row must not advertise
	// what couch has no evidence for.
	if evidence.Session.State == SessionUnresolved && record.VerifiedPark == nil {
		return ThreadUnusable, ReasonUnknown
	}
	if evidence.ParkedStatus == ProofUnresolved {
		return ThreadUnusable, ReasonUnknown
	}
	if parkedResumeProofMatches(record, evidence.Parked) {
		return ThreadParked, ""
	}
	if record.VerifiedPark != nil {
		// The receipt as a diagnostic, not an authority: couch parked this
		// thread deliberately and the conversation it preserved can no longer
		// be resolved. That is a different story from a thread whose session
		// simply ended, and the operator reads the difference.
		return ThreadUnusable, ReasonBindingLost
	}
	if evidence.Session.State == SessionUnresolved {
		// Not "no session" -- "we could not ask". The distinction is
		// load-bearing: session-gone is archive-eligible.
		return ThreadUnusable, ReasonUnknown
	}
	return ThreadUnusable, ReasonSessionGone
}

// startClaimed reports a start couch has claimed and not yet completed.
//
// It reads `Incarnation.Start`, NOT the pid, identity or lifecycle state. That
// distinction is the whole point: a ThreadStartClaim is couch's record of an
// operation it is performing, carrying the supervisor identity that initiated
// it, and recoverable on its own terms. The pid/identity/state fields describe
// an EXTERNAL process whose lifetime is shorter than the thread's, and nothing
// in classification may read those again.
func startClaimed(record ThreadRecord) bool {
	for _, incarnation := range record.Incarnations {
		if incarnation.Start != nil {
			return true
		}
	}
	return false
}

// startInFlight is the whole of that "on its own terms" clause, made to happen.
//
// A start is in flight while the couch that claimed it is alive, or while its
// liveness cannot be established. A claim whose owner is provably Dead has no
// driver: nothing will advance it, nothing will roll it back until the next
// couch starts, and reporting it as `starting...` tells the operator to wait for
// an event that is never coming -- while withholding archive and resume, because
// `busy` offers neither. That is #271's wedge in its last hiding place, one
// level below where the classifier's other deletions cut.
//
// Unknown keeps the row busy, which is the same direction ReconcileStart already
// fails ("Unknown evidence always keeps capacity occupied"). Releasing on
// ignorance would offer to archive a thread that is starting normally.
//
// Releasing does not decide anything on its own: the row falls through to the
// world, and reads live, detached or session-gone exactly as any other record
// with the same evidence would. The claim itself is untouched -- clearing it is
// a write, and writes belong to reconcileInterruptedStarts and to the archive
// path, each of which re-probes before acting.
func startInFlight(record ThreadRecord, evidence ThreadEvidence) bool {
	return startClaimed(record) && evidence.StartOwner != Dead
}

// detachedResumeProofMatches is the warm-session contract shared by execution
// and the final recheck. The INVENTORY no longer calls it -- since #256 the
// refresh reads session presence instead, and never counts clients. ProjectDetachedSessions proves live,
// client-free, unique ownership; this matcher correlates that proof with the
// thread. Occupancy belongs to the caller's lifecycle stage, since the final
// recheck runs after the attempt has claimed a creating incarnation.
func detachedResumeProofMatches(record ThreadRecord, observations []DetachedSessionObservation) bool {
	if record.LatestLaunchProfile == nil || !launcher.IsSupportedAgent(record.LatestLaunchProfile.Agent) || record.LatestLaunchProfile.Argv == nil || len(observations) != 1 {
		return false
	}
	observation := observations[0]
	return observation.Address == record.Address && observation.SessionName != "" &&
		observation.Agent == record.LatestLaunchProfile.Agent
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
	snapshot, evidence, err := c.gatherThreadEvidence(ctx, observations, nil)
	if err != nil {
		return nil, err
	}
	return ProjectActionableThreads(FromSnapshot(snapshot, evidence)), nil
}

// gatherThreadEvidence resolves what is knowable about every record and decides
// nothing. Both inventories consume it, so the switcher and the diagnostic view
// cannot derive different states from the same store (ARCH-DRY).
//
// ask narrows the COLD-resume resolution, not the record set: a candidate it
// rejects keeps ProofUnresolved, so a resume-shaped row there classifies
// `unknown` and appears as a row nobody can act on rather than vanishing. It
// does NOT narrow session presence, which is one host-wide call for every
// record (#256). nil asks about every candidate, which is what the switcher's
// refresh wants; startup passes a predicate because its readers filter before
// they read (pair#206 M1). The one home for which readers those are is
// startup.go -- see its comment there rather than restating the list here.
//
// It returns the snapshot too, because physicalizing a working path mutates the
// record the caller projects.
func (c *Couch) gatherThreadEvidence(ctx context.Context, observations []LiveTTYObservation, ask func(ThreadRecord) bool) (ThreadSnapshot, map[ThreadAddress]ThreadEvidence, error) {
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
	//
	// Since #256 it is POSITIVE-ONLY: its absence proves nothing and falls
	// through to the session. Pinned by TestAbsentLiveEvidenceProvesNothing.
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

	// Session presence for EVERY record, not only the resume-shaped ones. One
	// host-wide `list-sessions`, no `list-clients` -- see SessionPresence for
	// why that is both affordable and sufficient here.
	//
	// A resolver that is absent or fails leaves every observation at its zero
	// value, which is unresolved. That is the honest degraded answer: couch
	// could not look, so no thread is told its session is gone.
	//
	// Asked BEFORE the per-record loop, because the loop's ledger read is
	// gated on the answer: a thread whose session is up needs no cold-resume
	// proof, and reading a ledger per record is the one cost this pass has to
	// keep bounded (#256 M2).
	presence := map[ThreadAddress]SessionObservation{}
	if presenceResolver, ok := c.Artifacts.(SessionPresenceResolver); ok && len(snapshot.Records) > 0 {
		addresses := make([]ThreadAddress, 0, len(snapshot.Records))
		for i := range snapshot.Records {
			addresses = append(addresses, snapshot.Records[i].Address)
		}
		if observed, presenceErr := presenceResolver.SessionPresence(ctx, addresses); presenceErr == nil {
			presence = observed
		}
	}

	evidence := make(map[ThreadAddress]ThreadEvidence, len(snapshot.Records))
	var resumable []ParkedResumeObservation
	resolver, _ := c.Artifacts.(NativeBindingResolver)
	for i := range snapshot.Records {
		record := snapshot.Records[i]
		if err := ctx.Err(); err != nil {
			return ThreadSnapshot{}, nil, err
		}
		item := ThreadEvidence{Live: observed[record.Address], Session: presence[record.Address]}
		// Who is driving this thread's start, if anything is. Gathered BEFORE
		// the resume-shaped gate below, because a record mid-start has no
		// LatestLaunchProfile yet -- it is written when the start registers --
		// so gating this on resume shape would leave every driverless claim
		// unprobed and every such row wedged at `starting...`.
		//
		// An unresolvable transaction leaves StartOwner at Unknown, which is
		// the fail-closed answer. See startInFlight.
		if c.Proc != nil {
			if transaction, transactionErr := CurrentStartTransaction(record); transactionErr == nil {
				item.StartOwner = observeExactProcess(c.Proc, transaction.Owner)
			}
		}
		// Physicalization and binding resolution are RESUME-SHAPED work, and
		// since #256 that is decided by RESUME AUTHORITY rather than by the
		// bookkeeping. A record carrying an incarnation DOES reach both now --
		// it has to, because its session may have outlived its launcher and a
		// reattach needs the path. The call-count guard still bounds the work.
		// Resume-shaped is now about RESUME AUTHORITY, not about the
		// bookkeeping. It used to exclude any record carrying an incarnation or
		// a park, which is precisely why a #272 record was never physicalized
		// and never had its conversation resolved: the evidence that would have
		// recovered it was gated behind the state that had gone stale.
		resumeShaped := !record.Reservation && record.LatestLaunchProfile != nil &&
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
		if item.PathError != nil {
			evidence[record.Address] = item
			continue
		}
		// Applied AFTER physicalization, because the predicate compares working
		// paths and an alias would otherwise miss its own thread -- and BEFORE
		// ResolveEstablished, which reads this thread's ledger and is half of
		// what startup was paying for.
		if ask != nil && !ask(snapshot.Records[i]) {
			evidence[record.Address] = item
			continue
		}
		// COLD-RESUME proof is asked of the LEDGER, for every record whose
		// session is not up -- not only the ones carrying a park receipt.
		//
		// That gate used to be `record.VerifiedPark != nil`, which made a
		// receipt the authority over recoverability. It is not one: a
		// VerifiedPark carries a ParkIdentity and no native conversation id, so
		// it can say a park HAPPENED and never that there is something to
		// resume into. The conversation id lives in `ledger-<tag>.jsonl`, and
		// ResolveEstablished is the only thing that reads it. A thread whose
		// session died without a park therefore had a resumable conversation
		// nobody asked about, and read `session-gone` -- archive-eligible.
		//
		// A warm thread still needs no cold proof: its session's own presence
		// is all reattachment consumes, and skipping it is what keeps this read
		// off the common path.
		if item.Session.State != SessionPresent && len(item.Live) == 0 && resolver != nil {
			agent := record.LatestLaunchProfile.Agent
			binding, resolveErr := resolver.ResolveEstablished(ctx, record.Address.RepoScope, string(record.Address.Tag), agent)
			// The parked question is answered either way: a refusal is a
			// resolved "no binding", not an unresolved question.
			item.ParkedStatus = ProofResolved
			if resolveErr == nil && bindingResumeDiagnostic(binding) == "" {
				resumable = append(resumable, ParkedResumeObservation{
					Address: record.Address, Agent: agent, NativeID: binding.NativeID,
				})
			}
		}
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
