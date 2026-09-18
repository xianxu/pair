package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type ResumeDiagnosticCode string

const (
	ResumeLive               ResumeDiagnosticCode = "resume-live"
	ResumeUnknown            ResumeDiagnosticCode = "resume-unknown"
	ResumeParking            ResumeDiagnosticCode = "resume-parking"
	ResumeTombstoned         ResumeDiagnosticCode = "resume-tombstoned"
	ResumePathMissing        ResumeDiagnosticCode = "resume-path-missing"
	ResumeProfileMissing     ResumeDiagnosticCode = "resume-profile-missing"
	ResumeProfileInvalid     ResumeDiagnosticCode = "resume-profile-invalid"
	ResumeAgentUnsupported   ResumeDiagnosticCode = "resume-agent-unsupported"
	ResumeBindingProvisional ResumeDiagnosticCode = "resume-binding-provisional"
	ResumeBindingAmbiguous   ResumeDiagnosticCode = "resume-binding-ambiguous"
	ResumeBindingUnbound     ResumeDiagnosticCode = "resume-binding-unbound"
	ResumeBindingRootMissing ResumeDiagnosticCode = "resume-binding-root-missing"
	// ResumeSessionGone is the WARM path's staleness: the detached session a
	// reattach was projected against died before the launch.
	ResumeSessionGone ResumeDiagnosticCode = "resume-session-gone"
	// ResumeNotRunning is a relaunch target that is not running at all. It is
	// the OPPOSITE of ResumeLive, which the two used to share, so a parked row
	// was told "resume-live" -- a code naming the state it is not in.
	ResumeNotRunning ResumeDiagnosticCode = "resume-not-running"
	// ResumeNotDetached refuses a WARM-ONLY resume of a thread that is not warm
	// (pair#206): parked, or with no surviving session. It is raised before any
	// effect, so the caller -- the background reattach pass -- can skip the
	// thread without anything to undo.
	ResumeNotDetached ResumeDiagnosticCode = "resume-not-detached"
	// ResumeStarting is a start claim that is still someone's business: the
	// couch that made it is alive or unprovable, or the process it forked is.
	// It is distinct from ResumeLive, which names a thread with a running
	// agent -- here nothing may be running yet, and the refusal is about the
	// TRANSACTION, not the actor (#256 M2).
	ResumeStarting ResumeDiagnosticCode = "resume-starting"
)

// ResumeOptions narrows what a resume is allowed to do.
//
// The zero value is today's resume: warm if the thread is detached, cold if it
// is parked. WarmOnly restricts it to reattaching a thread whose agent is still
// running, and refuses everything else BEFORE any effect -- which is what makes
// it safe to run behind the operator's back.
type ResumeOptions struct {
	WarmOnly bool
}

type ResumeRefusal struct {
	Code       ResumeDiagnosticCode
	Diagnostic string
}

func (e *ResumeRefusal) Error() string {
	if e == nil {
		return "resume refused"
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Diagnostic)
}

func ResumeDiagnosticOf(err error) ResumeDiagnosticCode {
	var refusal *ResumeRefusal
	if errors.As(err, &refusal) {
		return refusal.Code
	}
	return ""
}

type NativeBindingResolution struct {
	Status   sessioninventory.BindingStatus
	NativeID string
}

type ResumeEligibilityInput struct {
	Thread            ThreadRecord
	WorkingPathExists bool
	Binding           NativeBindingResolution
	// Detached is proof that this thread's zellij session is alive with no
	// client attached. It is the resume authority for a thread that was
	// detached rather than parked -- a detached thread has no verified park to
	// point at, because nothing was torn down.
	Detached bool
}

type ResumeEligibility struct {
	Address           ThreadAddress
	WorkingPath       string
	Profile           LaunchProfile
	RequiredSessionID string
}

// coldResumeAuthorized reports whether there is a conversation to resume INTO.
//
// This is the cold path's whole authority, and it is a fact about the ledger.
// `record.VerifiedPark` used to stand in for it and cannot: the receipt carries
// a ParkIdentity and no native conversation id, so it attests that a park
// happened and never that anything survived it. Two threads read it wrong in
// opposite directions -- a parked thread whose conversation was gone was offered
// a resume that could not work, and a thread whose session died without a park
// was refused one that would have.
func coldResumeAuthorized(binding NativeBindingResolution) bool {
	return bindingResumeDiagnostic(binding) == ""
}

func DecideResume(input ResumeEligibilityInput) (ResumeEligibility, error) {
	record := input.Thread
	// #256: the park transaction and the incarnation are NOT read here.
	//
	// They used to be, as "one occupancy rule, shared with archive" -- and that
	// is exactly how this guard came to contradict the classifier. A thread whose
	// launcher died while its session survived classifies `detached`, is ranked
	// highest by SelectResumableRoot, is offered `resume` by the menu, and was
	// then refused here for an incarnation naming a process that dies with couch.
	// The operator saw a row that advertised recovery and could not deliver it,
	// which is the precise anti-pattern this issue exists to remove.
	//
	// Resume now rests on the same facts the classification does: a surviving
	// session (warm) or a ledger that resolves a conversation (cold). The
	// residual race -- reattaching while a park is genuinely mid-teardown -- is
	// seconds wide, needs a deliberate operator action on both sides, and is
	// dissolved by #275, which removes the durable transaction entirely.
	//
	// `hasOccupiedIncarnation` survives for relaunch and switch-agent, which ask
	// a DIFFERENT question: not "is this recoverable" but "is couch itself
	// already operating on this thread", where couch's own record is authority.
	//
	// #256 M2: COLD authority is the LEDGER, not `record.VerifiedPark`. The
	// classifier moved first -- a thread whose session is gone but whose ledger
	// still names a conversation reads `parked` -- and a guard that kept
	// demanding the receipt would refuse the very resume that row advertises.
	// That disagreement between the classification and the guard IS the
	// anti-pattern this issue exists to remove; it is the same one #272 hit from
	// the other side.
	if !input.Detached && !coldResumeAuthorized(input.Binding) {
		// There is nothing to resume INTO. The tombstone scan runs only here,
		// to say WHY in the most useful terms available: a park couch
		// deliberately abandoned is a better answer than "unbound".
		//
		// It is no longer a VETO, and that matters more after M2 than before.
		// It refuses on ANY tombstoned entry in the whole history with no
		// break, AbandonPark appends tombstones permanently, and archive now
		// abandons orphaned parks as a matter of course -- so a veto here would
		// make "couch crashed mid-park once" a permanent cold-resume ban. A
		// resolvable conversation beats a historical bookkeeping failure, which
		// is the same rule that freed the detached path (#272) applied to the
		// cold one.
		for i := len(record.ParkHistory) - 1; i >= 0; i-- {
			if record.ParkHistory[i].Tombstoned {
				return ResumeEligibility{}, refuseResume(ResumeTombstoned, "latest park transaction was abandoned")
			}
		}
		return ResumeEligibility{}, refuseBinding(bindingResumeDiagnostic(input.Binding))
	}
	// The rules that do not depend on the thread being unoccupied. Shared with
	// relaunch, which asks them about a thread that is still LIVE.
	if err := CheckResumePreconditions(record, input.Binding, input.WorkingPathExists); err != nil {
		// The binding is the COLD path's proof only. A warm reattach consumes it
		// nowhere, so its refusal must not reach a detached thread -- which is
		// what CheckResumePreconditions cannot know and this caller does.
		if code := ResumeDiagnosticOf(err); !isBindingDiagnostic(code) || !input.Detached {
			return ResumeEligibility{}, err
		}
	}
	profile := cloneLaunchProfile(*record.LatestLaunchProfile)
	// The native binding is the COLD resume's proof: a parked thread has no
	// agent, so Pair must create a session and relaunch it with
	// `--resume <native id>`, and an unresolved id means that relaunch cannot
	// work. A warm reattach relaunches nothing -- the agent is alive behind a
	// client-less zellij session and reattaching is `zellij attach` -- so the
	// id is proof for a step that does not happen. Demanding it there refused
	// threads that would have reattached fine (#179).
	//
	// The proof the warm path DOES require is the session itself, and
	// input.Detached is it: an unambiguous name binding to this exact address,
	// live, with zero clients.
	if !input.Detached {
		return ResumeEligibility{
			Address: record.Address, WorkingPath: record.WorkingPath,
			Profile: profile, RequiredSessionID: input.Binding.NativeID,
		}, nil
	}
	return ResumeEligibility{
		Address: record.Address, WorkingPath: record.WorkingPath, Profile: profile,
	}, nil
}

// workingPathExists is the shared answer to "does this thread's working path
// still resolve", and it carries the ONE nil-Path policy: no path operations
// means the path cannot be PROVED, which is a refusal rather than an
// assumption. Both resume paths and relaunch call it, which is what stops the
// divergence its first two copies already had -- ResumeContext started at false
// and called Physical unconditionally while Relaunch started at true and
// skipped the call when Path was nil, so a nil Path made relaunch PASS the path
// precondition and then panic one step later.
//
// It is kept separate from the native binding because a warm reattach must not
// pay for -- or be failed by -- a resolution it has no use for.
func (c *Couch) workingPathExists(thread ThreadRecord) bool {
	if c.Path == nil {
		return false
	}
	_, err := c.Path.Physical(thread.WorkingPath)
	return err == nil
}

// resumeEvidence resolves the native binding for a COLD resume: the one that
// will pass `--resume <native-id>` and therefore needs proof of which
// conversation it is resuming.
//
// It is not the warm path's evidence, and bundling the path check into it made
// it look like it was. A detached thread's authority is its SURVIVING SESSION,
// not a native id; asking here made a warm reattach do a ListFiles, an
// up-to-8MB read, proof validation and a possible catalog write only to discard
// the answer -- and, worse, fail outright when that resolver errored, which is a
// failure mode the warm path never had.
func (c *Couch) resumeEvidence(ctx context.Context, thread ThreadRecord) (NativeBindingResolution, error) {
	bindings, ok := c.Artifacts.(NativeBindingResolver)
	if !ok {
		return NativeBindingResolution{}, errors.New("native binding resolver is unavailable")
	}
	agent := ""
	if thread.LatestLaunchProfile != nil {
		agent = thread.LatestLaunchProfile.Agent
	}
	return bindings.ResolveEstablished(ctx, thread.Address.RepoScope, string(thread.Address.Tag), agent)
}

// CheckResumePreconditions is every resume rule that a park cannot change.
//
// It exists because relaunch has to ask "would this thread be resumable ONCE
// PARKED?" -- a question about the thread's durable shape, not about what is
// running on it right now. Splitting the rules is what stops relaunch
// re-deriving them: two parallel derivations drift toward whichever cases each
// author thought about, which is how the archive guard came to admit `creating`
// while resume refused it (pair#181 M3).
//
// What stays with DecideResume is everything about THIS resume: the choice
// between cold authority (a ledger that resolves a conversation) and warm
// (proved detachment), and the tombstone scan. Those are not preconditions a
// park would satisfy.
//
// It no longer says DecideResume "refuses any occupied incarnation": #256 M1
// deleted that refusal, because the incarnation names a launcher that dies with
// couch.
//
// The binding rule is included, and the caller decides whether it applies: it is
// the COLD path's proof, which a warm reattach consumes nowhere.
func CheckResumePreconditions(record ThreadRecord, binding NativeBindingResolution, workingPathExists bool) error {
	if record.WorkingPath == "" || !workingPathExists {
		return refuseResume(ResumePathMissing, "saved working path is unavailable")
	}
	if record.LatestLaunchProfile == nil {
		return refuseResume(ResumeProfileMissing, "thread has no successful saved launch profile")
	}
	if record.LatestLaunchProfile.Agent == "" || record.LatestLaunchProfile.Argv == nil {
		return refuseResume(ResumeProfileInvalid, "saved launch profile is incomplete")
	}
	if !launcher.IsSupportedAgent(record.LatestLaunchProfile.Agent) {
		return refuseResume(ResumeAgentUnsupported, "saved launch agent is unsupported")
	}
	if code := bindingResumeDiagnostic(binding); code != "" {
		return refuseBinding(code)
	}
	return nil
}

// isBindingDiagnostic reports the refusals that come from the native binding, so
// a caller can apply the rest and skip these.
func isBindingDiagnostic(code ResumeDiagnosticCode) bool {
	switch code {
	case ResumeBindingProvisional, ResumeBindingAmbiguous, ResumeBindingUnbound, ResumeBindingRootMissing:
		return true
	}
	return false
}

func bindingResumeDiagnostic(binding NativeBindingResolution) ResumeDiagnosticCode {
	switch binding.Status {
	case sessioninventory.BindingProvisional:
		return ResumeBindingProvisional
	case sessioninventory.BindingAmbiguous:
		return ResumeBindingAmbiguous
	case sessioninventory.BindingUnbound:
		return ResumeBindingUnbound
	case sessioninventory.BindingEstablished:
		if binding.NativeID == "" {
			return ResumeBindingRootMissing
		}
		return ""
	default:
		return ResumeBindingUnbound
	}
}

// refuseBinding is the ONLY way to build a binding refusal.
//
// bindingRefusalDiagnostic gave each status its own actionable sentence and then
// had exactly one consumer: the real resolver and its stateful fake both still
// passed the developer's catch-all by hand, so the path an OPERATOR actually
// travels never saw the improvement. A message function with one caller and two
// hand-written copies is not a fix, it is a fix that looks applied.
func refuseBinding(code ResumeDiagnosticCode) error {
	return refuseResume(code, bindingRefusalDiagnostic(code))
}

// bindingRefusalDiagnostic is the operator-facing half of
// bindingResumeDiagnostic, derived from the same code rather than written once
// for all four.
//
// One sentence covered every binding status, and it was a developer's sentence:
// "native session binding is not one exact established root" tells an operator
// neither what happened nor what to do. It also flattened the one status that is
// not a fault at all -- provisional is the ORDINARY state of a thread whose
// agent has not answered yet, and it is the refusal a relaunch is most likely to
// meet, because relaunching is something you do to a session you just started.
func bindingRefusalDiagnostic(code ResumeDiagnosticCode) string {
	switch code {
	case ResumeBindingProvisional:
		return "its agent has not completed a turn yet, so there is no proof of which conversation to resume -- give it one and retry"
	case ResumeBindingAmbiguous:
		return "more than one native session matches it, so resuming could resume the wrong conversation"
	case ResumeBindingUnbound:
		return "no native session is bound to it yet"
	case ResumeBindingRootMissing:
		return "its binding records no native session id"
	}
	return "native session binding is not one exact established root"
}

func refuseResume(code ResumeDiagnosticCode, diagnostic string) error {
	return &ResumeRefusal{Code: code, Diagnostic: diagnostic}
}

type NativeBindingResolver interface {
	ResolveEstablished(context.Context, string, string, string) (NativeBindingResolution, error)
}

type SessionInventoryNativeBindingResolver struct {
	Runtime sessioninventory.Runtime
}

func (r SessionInventoryNativeBindingResolver) ResolveEstablished(ctx context.Context, repoScope, tag, agent string) (NativeBindingResolution, error) {
	if r.Runtime == nil {
		return NativeBindingResolution{}, errors.New("native binding resolver has no runtime")
	}
	query, err := sessioninventory.QuerySessionContext(ctx, r.Runtime, repoScope, tag, sessioninventory.Agent(agent))
	if err != nil {
		return NativeBindingResolution{}, err
	}
	resolution := NativeBindingResolution{Status: query.Status}
	if query.Root != nil {
		resolution.NativeID = query.Root.NativeID
	}
	if code := bindingResumeDiagnostic(resolution); code != "" {
		return resolution, refuseBinding(code)
	}
	return resolution, nil
}

var _ NativeBindingResolver = SessionInventoryNativeBindingResolver{}

// Resume reoccupies one resumable address using only its exact saved
// path, launch profile, and established native root binding.
func (c *Couch) Resume(address ThreadAddress) (ActorRecord, Handle, error) {
	return c.ResumeContext(context.Background(), address)
}

func (c *Couch) ResumeContext(ctx context.Context, address ThreadAddress) (ActorRecord, Handle, error) {
	return c.ResumeContextWith(ctx, address, ResumeOptions{})
}

// ResumeContextWith is ResumeContext narrowed by opts.
func (c *Couch) ResumeContextWith(ctx context.Context, address ThreadAddress, opts ResumeOptions) (retRecord ActorRecord, retHandle Handle, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ActorRecord{}, nil, err
	}
	if c == nil || c.Threads == nil {
		return ActorRecord{}, nil, errors.New("resume: Couch is unavailable")
	}
	finishRetention, err := c.beginResumeRetention(ctx, address, !opts.WarmOnly)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	defer func() { retErr = errors.Join(retErr, finishRetention(retErr == nil)) }()
	thread, err := c.Threads.GetThread(address)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	agent := ""
	if thread.LatestLaunchProfile != nil {
		agent = thread.LatestLaunchProfile.Agent
	}
	// Resolve the binding ONLY where it is the authority -- the cold path, which
	// is about to pass `--resume <native-id>`. A detached thread resumes warm off
	// its surviving session and needs no native id, so it asks for none: the
	// binding resolver is not consulted, cannot slow it down, and cannot fail it.
	// (ResolveEstablished returns an ERROR for a provisional binding, so asking
	// on the warm path refused the thread here, before DecideResume could decide
	// anything -- which is how a detached thread became unreachable.)
	pathExists := c.workingPathExists(thread)

	// WARM first, for every thread, then COLD only if warm did not answer.
	//
	// Both halves used to be gated on `thread.VerifiedPark`, which is the same
	// receipt-as-authority defect #256 M2 removed from the classifier -- and
	// keeping it here would have left the guard refusing the resume the
	// classification advertises, one level below where that was fixed. A
	// receipt says a park HAPPENED; it does not say a session is gone, and it
	// carries no conversation id.
	//
	// So: ask the session, because a surviving one is the cheapest and safest
	// authority and the classifier already prefers it; then ask the ledger,
	// because a cold resume is about to pass `--resume <native-id>` and that id
	// is the only thing that makes the relaunch land in the right conversation.
	// The warm path asks for no id -- ResolveEstablished ERRORS on a
	// provisional binding, so asking there refused threads before DecideResume
	// could decide anything, which is how a detached thread became unreachable.
	//
	// This is the strict-action half of optimistic inventory: one `list-clients`
	// for the single thread the operator pressed Enter on.
	detached := false
	warmSession := ""
	if resolver, ok := c.Artifacts.(DetachedSessionResolver); ok {
		observed, observeErr := resolver.DetachedSessions(ctx, []DetachedCandidate{{
			Address: address, Agent: agent,
		}})
		if observeErr != nil {
			return ActorRecord{}, nil, fmt.Errorf("observe detached session for %+v: %w", address, observeErr)
		}
		detached = detachedResumeProofMatches(thread, observed)
		if detached {
			warmSession = observed[0].SessionName
		}
	}
	if opts.WarmOnly && !detached {
		return ActorRecord{}, nil, refuseResume(ResumeNotDetached,
			"thread has no detached session to reattach to; a warm-only resume never starts an agent")
	}
	var binding NativeBindingResolution
	var bindings NativeBindingResolver
	if !detached {
		if err := continuationGuard(thread); err != nil {
			return ActorRecord{}, nil, err
		}
		var ok bool
		bindings, ok = c.Artifacts.(NativeBindingResolver)
		if !ok {
			return ActorRecord{}, nil, errors.New("resume: native binding resolver is unavailable")
		}
		resolved, err := c.resumeEvidence(ctx, thread)
		// A binding DIAGNOSTIC is evidence, not a verdict, and it travels with
		// the resolution it describes (every resolver returns both). Bailing on
		// it here made DecideResume's cold-refusal branch unreachable from the
		// only production caller -- so the tombstone answer the plan and the
		// atlas both promise ("abandoned" beats "unbound") never reached the
		// operator, and nothing could observe the gap.
		//
		// Guidance belongs at the CONSUMER (#256 M1, round 3): the resolver
		// reports what it found, DecideResume decides what it means. Anything
		// else -- an unreadable ledger, a missing resolver -- is a real failure
		// and still stops here.
		if err != nil && !isBindingDiagnostic(ResumeDiagnosticOf(err)) {
			return ActorRecord{}, nil, err
		}
		binding = resolved
	}
	if thread.Continuation != nil && detached {
		if err := c.validateContinuationWarm(ctx, thread); err != nil {
			return ActorRecord{}, nil, err
		}
	}
	eligible, err := DecideResume(ResumeEligibilityInput{
		Thread: thread, WorkingPathExists: pathExists, Binding: binding, Detached: detached,
	})
	if err != nil {
		return ActorRecord{}, nil, err
	}
	owner, err := c.Proc.Current()
	if err != nil {
		return ActorRecord{}, nil, fmt.Errorf("identify couch supervisor: %w", err)
	}
	nonce, err := allocateStartNonce(c.Entropy)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	startedAt := c.Clock.Now()
	// The same single write the spawn path uses. Its precondition -- a resolvable
	// conversation OR proved detachment -- was already checked by DecideResume
	// above; carrying both authorities forward here is what keeps M4 from
	// silently re-breaking detached reattachment, which pair#181 M2 fixed and
	// admission's second gate used to enforce. (The cold authority was the park
	// RECEIPT until #256 M2 moved it to the ledger.)
	repoIdentity, err := c.resolveRepoIdentity(ctx, thread.WorkingPath)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	// RE-ADOPTION (#272, #256). A thread whose launcher died while its session
	// survived still carries that launcher's bookkeeping, and the store's guards
	// refuse a record that does -- correctly, since one incarnation at a time
	// and no open park are structural invariants, not lifecycle opinions.
	//
	// So the stale claims are cleared HERE, by the caller that has both the
	// evidence and the authority. See clearLifecycleDebris for the
	// rules that govern it; do not restate them here, because a claim restated
	// away from its test is how two of them came to be wrong.
	if retired, retireErr := c.clearLifecycleDebris(thread); retireErr != nil {
		return ActorRecord{}, nil, retireErr
	} else if retired != nil {
		thread = *retired
	}
	thread, err = c.Threads.CommitStartClaim(address, thread.Revision, repoIdentity, startedAt, StartEvent{
		Shape: func() StartShape {
			if detached {
				return StartWarmReattach
			}
			return StartColdResume
		}(),
		Kind:    StartClaimed,
		Nonce:   nonce,
		Owner:   SupervisorOwner{PID: owner.PID, Identity: owner.Identity},
		Profile: &eligible.Profile,
	})
	if err != nil {
		return ActorRecord{}, nil, err
	}

	// Recheck after the durable address claim and immediately before any child
	// effects. Each shape rechecks its OWN authority: a cold resume rechecks
	// the native binding, because a session replacement in this window must be
	// a refusal rather than permission to create a different session under the
	// same Pair address. A warm reattach rechecks that its session is still
	// there, which is the equivalent staleness -- and if it died in the window,
	// there is nothing to attach to.
	profileRaw := ""
	if detached {
		if err := c.confirmStillDetached(ctx, thread, warmSession); err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
		}
	} else {
		currentBinding, err := bindings.ResolveEstablished(ctx, address.RepoScope, string(address.Tag), eligible.Profile.Agent)
		if err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
		}
		if err := launcher.RequireNativeResumeBinding(eligible.RequiredSessionID, currentBinding.NativeID, currentBinding.Status); err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
		}
		built, err := launcher.BuildCouchResumeLaunchProfile(
			string(address.Tag), eligible.Profile.Agent, eligible.Profile.Argv, eligible.RequiredSessionID,
		)
		if err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
		}
		profileRaw = built
	}
	args := StartArgs{
		Worktree: Worktree(thread.StartingPath), Cwd: eligible.WorkingPath,
		Stack: eligible.Profile.Agent, ExtraArgs: cloneArgv(eligible.Profile.Argv),
	}
	return c.launchTrackedThread(trackedThreadLaunch{
		Context: ctx,
		Thread:  thread, Nonce: nonce, Args: args, StartedAt: startedAt,
		ProfileRaw: profileRaw, Resume: true, Warm: detached, Background: opts.WarmOnly,
	})
}

// confirmStillDetached re-proves a warm reattach's only precondition
// immediately before any child effect: the zellij session it means to attach to
// is still alive with no client.
//
// The cold path's equivalent is RequireNativeResumeBinding. Both exist for the
// same reason -- the world can change between projecting a row and launching --
// and each asks about the authority its own shape actually rests on.
func (c *Couch) confirmStillDetached(ctx context.Context, thread ThreadRecord, session string) error {
	resolver, ok := c.Artifacts.(DetachedSessionResolver)
	if !ok {
		return refuseResume(ResumeUnknown, "detached sessions cannot be observed")
	}
	observed, err := resolver.DetachedSessions(ctx, []DetachedCandidate{{Address: thread.Address, Agent: thread.LatestLaunchProfile.Agent}})
	if err != nil {
		return err
	}
	if detachedResumeProofMatches(thread, observed) && observed[0].SessionName == session {
		return nil
	}
	return refuseResume(ResumeSessionGone, "the same session can no longer be proved live, uniquely owned and detached")
}
