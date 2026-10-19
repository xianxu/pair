package couchcore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
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
	// ResumeNoSurvivor is a slot with no current record and no running agent
	// couch could prove is its own: there is no conversation to adopt.
	ResumeNoSurvivor ResumeDiagnosticCode = "resume-no-survivor"
	// ResumeSurvivorsAmbiguous is a slot with no current record and more than
	// one running agent that could be its conversation. Couch will not guess
	// between them; one must be stopped first.
	ResumeSurvivorsAmbiguous ResumeDiagnosticCode = "resume-survivors-ambiguous"
	// ResumeSurvivorUnproven is a slot with no current record whose agent
	// survives behind a detached session, where couch cannot prove WHICH agent
	// runs there: the detached proof echoes the agent it is asked about, so a
	// guessed agent proves nothing. The session is live, so the exit is to
	// attach to it or stop it -- reboot would refuse a live owner too.
	ResumeSurvivorUnproven ResumeDiagnosticCode = "resume-survivor-unproven"
	// ResumeOrphanedServer is a session whose exact zellij server is alive but
	// lost its socket (#399). Nothing can reach it, and its agent may still be
	// writing, so a resume would start a second agent on the conversation. The
	// way forward is reap, never reboot.
	ResumeOrphanedServer ResumeDiagnosticCode = "resume-orphaned-server"
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
	ObservationIncomplete bool
	FreshRequired         bool
	RequestedNativeID     string
	Status                sessioninventory.BindingStatus
	NativeID              string
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

// DecideResume separates existing-terminal attachment from native relaunch.
// Only the cold branch consumes a working checkout and native configuration.
func DecideResume(input ResumeEligibilityInput) (ResumeEligibility, error) {
	record := input.Thread
	if input.Detached {
		eligible := ResumeEligibility{Address: record.Address, WorkingPath: record.WorkingPath}
		// Optional display metadata only; no profile is invented or committed.
		if record.LatestLaunchProfile != nil {
			eligible.Profile = cloneLaunchProfile(*record.LatestLaunchProfile)
		}
		return eligible, nil
	}
	if !coldResumeAuthorized(input.Binding) {
		// A tombstone explains missing cold authority but cannot override an
		// established native binding or a surviving terminal.
		for i := len(record.ParkHistory) - 1; i >= 0; i-- {
			if record.ParkHistory[i].Tombstoned {
				return ResumeEligibility{}, refuseResume(ResumeTombstoned, "latest park transaction was abandoned")
			}
		}
		return ResumeEligibility{}, refuseResolvedBinding(input.Binding)
	}
	if err := CheckResumePreconditions(record, input.Binding, input.WorkingPathExists); err != nil {
		return ResumeEligibility{}, err
	}
	return ResumeEligibility{
		Address: record.Address, WorkingPath: record.WorkingPath,
		Profile: cloneLaunchProfile(*record.LatestLaunchProfile), RequiredSessionID: input.Binding.NativeID,
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
	bound, err := c.bindTranscript(ctx, thread)
	return bound.Resolution, err
}

// bindTranscript binds by the record's stored transcript (pair#695 P2): one
// exact ledger read and one stat, never a search once the file is known.
func (c *Couch) bindTranscript(ctx context.Context, thread ThreadRecord) (TranscriptBind, error) {
	if binder, ok := c.Artifacts.(TranscriptBinder); ok {
		return binder.BindTranscript(ctx, thread)
	}
	resolver, ok := c.Artifacts.(NativeBindingResolver)
	if !ok {
		return TranscriptBind{}, errors.New("transcript binder is unavailable")
	}
	return bindByResolver(ctx, resolver, thread)
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
		return refuseResolvedBinding(binding)
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
		if binding.NativeID != "" {
			return ""
		}
		// A chosen conversation id the agent never wrote: it never took a
		// turn, so there is no conversation to resume (pair#367 smoke test,
		// operator decision; #346 M2 had restarted it fresh under resume).
		// Reboot starts the fresh agent instead.
		if binding.FreshRequired && binding.RequestedNativeID != "" {
			return ResumeBindingUnbound
		}
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

func refuseResolvedBinding(binding NativeBindingResolution) error {
	if binding.ObservationIncomplete {
		return refuseResume(ResumeBindingProvisional, "native storage observation is unreadable or incomplete; retry when its storage can be listed")
	}
	return refuseBinding(bindingResumeDiagnostic(binding))
}

// refuseBinding builds the shared status-based binding refusal.
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

// NativeBindingResolver resolves a usable durable target. The historical method
// name also covers probation with a nonempty requested UUID.
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
	query, err := sessioninventory.QueryResumeTargetContext(ctx, r.Runtime, repoScope, tag, sessioninventory.Agent(agent))
	if err != nil {
		return NativeBindingResolution{}, err
	}
	resolution := NativeBindingResolution{Status: query.Status, NativeID: query.NativeID, FreshRequired: query.FreshRequired, RequestedNativeID: query.RequestedNativeID}
	if query.Status == sessioninventory.BindingProvisional && query.RequestedNativeID != "" && query.NativeID == "" && !query.FreshRequired {
		for _, diagnostic := range query.Diagnostics {
			if diagnostic.Code != sessioninventory.DiagnosticStorageAbsent {
				resolution.ObservationIncomplete = true
				break
			}
		}
	}
	if code := bindingResumeDiagnostic(resolution); code != "" {
		return resolution, refuseResolvedBinding(resolution)
	}
	return resolution, nil
}

var _ NativeBindingResolver = SessionInventoryNativeBindingResolver{}

// Resume reoccupies one resumable address using only its exact saved
// path, launch profile, and durable native resume target.
func (c *Couch) Resume(address ThreadAddress) (ActorRecord, Handle, error) {
	return c.ResumeContext(context.Background(), address)
}

func (c *Couch) ResumeContext(ctx context.Context, address ThreadAddress) (ActorRecord, Handle, error) {
	return c.ResumeContextWith(ctx, address, ResumeOptions{})
}

// confirmResumeReplacementAbsent protects the old successful profile from an
// interrupted replacement. Its target lives in Start until registration or
// unknown promotion; the old current binding cannot vouch for that target.
func (c *Couch) confirmResumeReplacementAbsent(ctx context.Context, thread ThreadRecord) error {
	agent := launchProfileAgent(thread)
	for _, incarnation := range thread.Incarnations {
		profile := incarnation.LaunchProfile
		binding := thread.SessionBinding
		if incarnation.Start != nil {
			profile = incarnation.Start.LaunchProfile
			binding = incarnation.Start.SessionBinding
		}
		if profile == nil || profile.Agent == agent {
			continue
		}
		if binding == nil {
			return refuseResume(ResumeUnknown, "target terminal identity is unavailable; inspect before resuming")
		}
		observed, err := c.observeRecoverySession(ctx, thread, RecoveryEvidence{})
		if err != nil || observed.Presence != PresenceAbsent || observeExactProcess(c.Proc, ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity}) != Dead {
			return refuseResume(ResumeUnknown, "target startup remains unresolved; inspect and recover before resuming")
		}
	}
	return nil
}

// ResumeContextWith is ResumeContext narrowed by opts.
func (c *Couch) ResumeContextWith(ctx context.Context, address ThreadAddress, opts ResumeOptions) (ActorRecord, Handle, error) {
	return c.resumeContextWith(ctx, address, opts, false)
}

// adoptWarm is reserved for explicit continuation recovery, whose caller has
// independently established authority to reconcile its lifecycle transaction.
func (c *Couch) resumeContextWith(ctx context.Context, address ThreadAddress, opts ResumeOptions, adoptWarm bool) (retRecord ActorRecord, retHandle Handle, retErr error) {
	begun := time.Now()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ActorRecord{}, nil, err
	}
	if c == nil || c.Threads == nil {
		return ActorRecord{}, nil, errors.New("resume: Couch is unavailable")
	}
	thread, err := c.Threads.PreviewThreadContext(ctx, address)
	if err != nil {
		return ActorRecord{}, nil, err
	}

	// Resolve the binding ONLY where it is the authority -- the cold path, which
	// is about to pass `--resume <native-id>`. A detached thread resumes warm off
	// its surviving session and needs no native id, so it asks for none: the
	// binding resolver is not consulted, cannot slow it down, and cannot fail it.
	pathExists := c.workingPathExists(thread)

	// An exact positive list row authorizes attachment with or without clients.
	// Only explicit continuation adoption also requires its detached/source proof.
	queryCtx, cancelAdmission := zellijreply.AdmissionContext(ctx)
	defer cancelAdmission()
	warmSession, detached, err := c.listedTerminal(queryCtx, thread)
	if err != nil {
		return ActorRecord{}, nil, err
	}

	if detached && adoptWarm {
		if err := c.confirmStillDetached(queryCtx, thread, warmSession); err != nil {
			return ActorRecord{}, nil, err
		}
	}
	if detached && !adoptWarm {
		return c.attachListedTerminalAdmitted(ctx, queryCtx, thread, warmSession, opts)
	}
	if opts.WarmOnly && !detached {
		return ActorRecord{}, nil, refuseResume(ResumeNotDetached, "thread has no positively listed terminal to attach to; an attach-only resume never starts an agent")
	}
	ctx, release, err := c.hold(ctx, address, "resume")
	if err != nil {
		return ActorRecord{}, nil, err
	}
	defer release()
	finishRetention, err := c.beginResumeRetention(ctx, address, !opts.WarmOnly)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	defer func() { retErr = errors.Join(retErr, finishRetention(retErr == nil)) }()
	current, err := c.Threads.GetThread(address)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	if current.Revision != thread.Revision {
		return ActorRecord{}, nil, &ThreadRevisionError{Address: address, Want: thread.Revision, Got: current.Revision}
	}
	if err := c.confirmResumeReplacementAbsent(queryCtx, thread); err != nil {
		return ActorRecord{}, nil, err
	}
	var binding NativeBindingResolution
	var transcript *threadrecord.TranscriptBinding
	if !detached {
		if err := continuationGuard(thread); err != nil {
			return ActorRecord{}, nil, err
		}
		bound, err := c.bindTranscript(queryCtx, thread)
		resolved := bound.Resolution
		transcript = bound.Store
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
		if err := c.validateContinuationWarm(queryCtx, thread); err != nil {
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
	if transcript != nil {
		transcript.BoundAt = startedAt
	}
	// The same single write the spawn path uses. Its precondition -- a resolvable
	// conversation OR proved detachment -- was already checked by DecideResume
	// above; carrying both authorities forward here is what keeps M4 from
	// silently re-breaking detached reattachment, which pair#181 M2 fixed and
	// admission's second gate used to enforce. (The cold authority was the park
	// RECEIPT until #256 M2 moved it to the ledger.)
	repoIdentity := ""
	if !detached {
		repoIdentity, err = c.resolveRepoIdentity(queryCtx, thread.WorkingPath)
		if err != nil {
			return ActorRecord{}, nil, err
		}
	} else {
		for _, incarnation := range thread.Incarnations {
			if incarnation.RepoIdentity != "" {
				repoIdentity = incarnation.RepoIdentity
				break
			}
		}
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
	// Session/native lookups above can outlive the initial absence observation.
	// Recheck before deleting a pending claim or retiring its promoted helper.
	if err := c.confirmResumeReplacementAbsent(queryCtx, thread); err != nil {
		return ActorRecord{}, nil, err
	}
	if err := queryCtx.Err(); err != nil {
		return ActorRecord{}, nil, err
	}
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
		Kind:       StartClaimed,
		Nonce:      nonce,
		Owner:      SupervisorOwner{PID: owner.PID, Identity: owner.Identity},
		Transcript: transcript,
		Profile: func() *LaunchProfile {
			if detached && thread.LatestLaunchProfile == nil {
				return nil
			}
			return &eligible.Profile
		}(),
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
	if err := c.prepareTrackedWorkspace(ctx, thread, nonce, detached); err != nil {
		return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
	}
	profileRaw := ""
	if detached {
		if err := c.confirmStillDetached(queryCtx, thread, warmSession); err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
		}
	} else {
		rebound, err := c.bindTranscript(queryCtx, thread)
		currentBinding := rebound.Resolution
		if err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
		}
		if err := launcher.RequireNativeResumeBinding(eligible.RequiredSessionID, currentBinding.NativeID, currentBinding.Status); err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
		}
		// The launcher takes no native admission for a Couch-owned resume
		// (pair#597); every Couch resume passes this registry check instead.
		if err := c.checkRecoveryConversationInUse(queryCtx, thread, RecoveryChoice{Agent: eligible.Profile.Agent, NativeID: eligible.RequiredSessionID}); err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
		}
		built, err := launcher.BuildCouchResumeLaunchProfile(string(address.Tag), eligible.Profile.Agent, eligible.Profile.Argv, eligible.RequiredSessionID)
		if err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
		}
		profileRaw = built
	}
	args := StartArgs{
		Worktree: Worktree(thread.StartingPath), Cwd: eligible.WorkingPath,
		Stack: eligible.Profile.Agent, ExtraArgs: cloneArgv(eligible.Profile.Argv),
	}
	if err := queryCtx.Err(); err != nil {
		return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
	}
	return c.launchTrackedThread(trackedThreadLaunch{
		Context: ctx,
		Thread:  thread, Nonce: nonce, Args: args, StartedAt: startedAt,
		ProfileRaw: profileRaw, Resume: true, Warm: detached, Background: opts.WarmOnly,
		Begun: begun,
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
	agent := ""
	if thread.LatestLaunchProfile != nil {
		agent = thread.LatestLaunchProfile.Agent
	}
	observed, err := resolver.DetachedSessions(ctx, []DetachedCandidate{{Address: thread.Address, Agent: agent, SessionName: recordSessionName(thread)}})
	if err != nil {
		return err
	}
	if detachedResumeProofMatches(thread, observed) && observed[0].SessionName == session {
		return nil
	}
	return refuseResume(ResumeSessionGone, "the same session can no longer be proved live, uniquely owned and detached")
}
