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
	ResumeCreating           ResumeDiagnosticCode = "resume-creating"
	ResumeUnknown            ResumeDiagnosticCode = "resume-unknown"
	ResumeParking            ResumeDiagnosticCode = "resume-parking"
	ResumeTombstoned         ResumeDiagnosticCode = "resume-tombstoned"
	ResumeLegacyUnverified   ResumeDiagnosticCode = "resume-legacy-unverified"
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
)

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

func DecideResume(input ResumeEligibilityInput) (ResumeEligibility, error) {
	record := input.Thread
	if record.Park != nil {
		return ResumeEligibility{}, refuseResume(ResumeParking, "thread has an active park transaction")
	}
	// One occupancy rule, shared with archive. It used to be inlined here and
	// re-derived in the store with a narrower set, which is how archiving a
	// thread mid-start passed a guard that resume would have refused.
	if occupied, state := occupiedIncarnation(record); occupied {
		return ResumeEligibility{}, refuseResume(occupiedResumeCode(state), "thread still has an occupied incarnation")
	}
	if record.VerifiedPark == nil && !input.Detached {
		// The tombstone scan is reached ONLY when neither authority holds, and
		// that ordering is load-bearing. It refuses on ANY tombstoned entry in
		// the whole history, with no break, and AbandonPark appends tombstones
		// permanently -- so a thread once abandoned mid-park, later started
		// again and detached, would be permanently unreattachable if the
		// detached branch sat after it. The rule means "there is no valid park
		// to resume from"; a detached thread is not resuming from a park.
		for i := len(record.ParkHistory) - 1; i >= 0; i-- {
			if record.ParkHistory[i].Tombstoned {
				return ResumeEligibility{}, refuseResume(ResumeTombstoned, "latest park transaction was abandoned")
			}
		}
		return ResumeEligibility{}, refuseResume(ResumeLegacyUnverified, "thread has no verified park completion")
	}
	// The rules that do not depend on the thread being unoccupied. Shared with
	// relaunch, which asks them about a thread that is still LIVE.
	if err := CheckResumePreconditions(record, input.Binding, input.WorkingPathExists); err != nil {
		// The binding is the COLD path's proof only. A warm reattach consumes it
		// nowhere, so its refusal must not reach a detached thread -- which is
		// what CheckResumePreconditions cannot know and this caller does.
		if code := ResumeDiagnosticOf(err); !isBindingDiagnostic(code) || record.VerifiedPark != nil {
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
	if record.VerifiedPark != nil {
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
// workingPathExists is the one piece of evidence BOTH resume paths need, and it
// is cheap. Kept separate from the binding for that reason: a warm reattach must
// not pay for -- or be failed by -- a resolution it has no use for.
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
// PARKED?" -- and it cannot ask DecideResume, which refuses any occupied
// incarnation and so always refuses a relaunch target. Splitting the rules is
// what stops relaunch re-deriving them: two parallel derivations drift toward
// whichever cases each author thought about, which is how the archive guard came
// to admit `creating` while resume refused it (pair#181 M3).
//
// What stays with DecideResume is everything about THIS resume: the occupancy
// refusal, the choice between park and detached authority, and the tombstone
// scan. Those are not preconditions a park would satisfy.
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
		return refuseResume(code, bindingRefusalDiagnostic(code))
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

// occupiedResumeCode names WHICH occupancy refused, so the diagnostic stays as
// specific as it was when the three states were enumerated by hand.
func occupiedResumeCode(state IncarnationState) ResumeDiagnosticCode {
	switch state {
	case IncarnationLive:
		return ResumeLive
	case IncarnationCreating:
		return ResumeCreating
	}
	return ResumeUnknown
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
		return resolution, refuseResume(code, "native session binding is not one exact established root")
	}
	return resolution, nil
}

var _ NativeBindingResolver = SessionInventoryNativeBindingResolver{}

// Resume reoccupies one verified parked address using only its exact saved
// path, launch profile, and established native root binding.
func (c *Couch) Resume(address ThreadAddress) (ActorRecord, Handle, error) {
	return c.ResumeContext(context.Background(), address)
}

func (c *Couch) ResumeContext(ctx context.Context, address ThreadAddress) (ActorRecord, Handle, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ActorRecord{}, nil, err
	}
	if c == nil || c.Threads == nil {
		return ActorRecord{}, nil, errors.New("resume: Couch is unavailable")
	}
	thread, err := c.Threads.GetThread(address)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	agent := ""
	if thread.LatestLaunchProfile != nil {
		agent = thread.LatestLaunchProfile.Agent
	}
	bindings, ok := c.Artifacts.(NativeBindingResolver)
	if !ok {
		return ActorRecord{}, nil, errors.New("resume: native binding resolver is unavailable")
	}
	// Resolve the binding ONLY where it is the authority -- the cold path, which
	// is about to pass `--resume <native-id>`. A detached thread resumes warm off
	// its surviving session and needs no native id, so it asks for none: the
	// binding resolver is not consulted, cannot slow it down, and cannot fail it.
	// (ResolveEstablished returns an ERROR for a provisional binding, so asking
	// on the warm path refused the thread here, before DecideResume could decide
	// anything -- which is how a detached thread became unreachable.)
	pathExists := c.workingPathExists(thread)
	var binding NativeBindingResolution
	if thread.VerifiedPark != nil {
		resolved, err := c.resumeEvidence(ctx, thread)
		if err != nil {
			return ActorRecord{}, nil, err
		}
		binding = resolved
	}
	// A thread with no verified park may still be resumable: it may have been
	// DETACHED, in which case its zellij session is alive with no client and
	// that survival is the authority. Ask only when it could matter, so an
	// ordinary parked resume costs no extra observation.
	detached := false
	if thread.VerifiedPark == nil {
		if resolver, ok := c.Artifacts.(DetachedSessionResolver); ok {
			observed, observeErr := resolver.DetachedSessions(ctx, []DetachedCandidate{{
				Address: address, Agent: agent, NativeID: binding.NativeID,
			}})
			if observeErr != nil {
				return ActorRecord{}, nil, fmt.Errorf("observe detached session for %+v: %w", address, observeErr)
			}
			detached = len(observed) == 1 && observed[0].Address == address
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
	// The same single write the spawn path uses. Its precondition -- verified
	// park OR proved detachment -- was already checked by DecideResume above;
	// carrying both authorities forward here is what keeps M4 from silently
	// re-breaking detached reattachment, which M2 fixed and admission's second
	// verified-park gate used to enforce.
	repoIdentity, err := c.resolveRepoIdentity(ctx, thread.WorkingPath)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	thread, err = c.Threads.CommitStartClaim(address, thread.Revision, repoIdentity, startedAt, StartEvent{
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
		if err := c.confirmStillDetached(ctx, address, eligible.Profile.Agent); err != nil {
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
		ProfileRaw: profileRaw, Resume: true, Warm: detached,
	})
}

// confirmStillDetached re-proves a warm reattach's only precondition
// immediately before any child effect: the zellij session it means to attach to
// is still alive with no client.
//
// The cold path's equivalent is RequireNativeResumeBinding. Both exist for the
// same reason -- the world can change between projecting a row and launching --
// and each asks about the authority its own shape actually rests on.
func (c *Couch) confirmStillDetached(ctx context.Context, address ThreadAddress, agent string) error {
	resolver, ok := c.Artifacts.(DetachedSessionResolver)
	if !ok {
		return refuseResume(ResumeBindingUnbound, "detached sessions cannot be observed")
	}
	observed, err := resolver.DetachedSessions(ctx, []DetachedCandidate{{Address: address, Agent: agent}})
	if err != nil {
		return err
	}
	for _, observation := range observed {
		if observation.Address == address && observation.SessionName != "" {
			return nil
		}
	}
	return refuseResume(ResumeSessionGone, "the detached session is no longer running")
}
