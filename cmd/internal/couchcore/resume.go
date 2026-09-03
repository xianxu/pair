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
	// DetachedSession is the name of that surviving session, and it is the
	// whole reattach proof: the agent never died, so there is no native id to
	// verify, only a session to rejoin.
	DetachedSession string
}

type ResumeEligibility struct {
	Address           ThreadAddress
	WorkingPath       string
	Profile           LaunchProfile
	RequiredSessionID string
	// ReattachSession is set instead of RequiredSessionID when the thread was
	// detached. Exactly one of the two is ever non-empty.
	ReattachSession string
}

func DecideResume(input ResumeEligibilityInput) (ResumeEligibility, error) {
	record := input.Thread
	if record.Park != nil {
		return ResumeEligibility{}, refuseResume(ResumeParking, "thread has an active park transaction")
	}
	for _, state := range []struct {
		value IncarnationState
		code  ResumeDiagnosticCode
	}{
		{IncarnationLive, ResumeLive},
		{IncarnationCreating, ResumeCreating},
		{IncarnationUnknown, ResumeUnknown},
	} {
		for _, incarnation := range record.Incarnations {
			if incarnation.State == state.value {
				return ResumeEligibility{}, refuseResume(state.code, "thread still has an occupied incarnation")
			}
		}
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
	if record.WorkingPath == "" || !input.WorkingPathExists {
		return ResumeEligibility{}, refuseResume(ResumePathMissing, "saved working path is unavailable")
	}
	if record.LatestLaunchProfile == nil {
		return ResumeEligibility{}, refuseResume(ResumeProfileMissing, "thread has no successful saved launch profile")
	}
	if record.LatestLaunchProfile.Agent == "" || record.LatestLaunchProfile.Argv == nil {
		return ResumeEligibility{}, refuseResume(ResumeProfileInvalid, "saved launch profile is incomplete")
	}
	profile := cloneLaunchProfile(*record.LatestLaunchProfile)
	if !launcher.IsSupportedAgent(profile.Agent) {
		return ResumeEligibility{}, refuseResume(ResumeAgentUnsupported, "saved launch agent is unsupported")
	}
	if code := bindingResumeDiagnostic(input.Binding); code != "" {
		return ResumeEligibility{}, refuseResume(code, "native session binding is not one exact established root")
	}
	if input.Detached {
		if input.DetachedSession == "" {
			return ResumeEligibility{}, refuseResume(ResumeBindingUnbound, "detached thread has no session name to reattach to")
		}
		return ResumeEligibility{
			Address: record.Address, WorkingPath: record.WorkingPath,
			Profile: profile, ReattachSession: input.DetachedSession,
		}, nil
	}
	return ResumeEligibility{
		Address: record.Address, WorkingPath: record.WorkingPath,
		Profile: profile, RequiredSessionID: input.Binding.NativeID,
	}, nil
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
	pathExists := false
	if _, err := c.Path.Physical(thread.WorkingPath); err == nil {
		pathExists = true
	}
	bindings, ok := c.Artifacts.(NativeBindingResolver)
	if !ok {
		return ActorRecord{}, nil, errors.New("resume: native binding resolver is unavailable")
	}
	agent := ""
	if thread.LatestLaunchProfile != nil {
		agent = thread.LatestLaunchProfile.Agent
	}
	binding, err := bindings.ResolveEstablished(ctx, address.RepoScope, string(address.Tag), agent)
	if err != nil {
		return ActorRecord{}, nil, err
	}
	// A thread with no verified park may still be resumable: it may have been
	// DETACHED, in which case its zellij session is alive with no client and
	// that survival is the authority. Ask only when it could matter, so an
	// ordinary parked resume costs no extra observation.
	detached := false
	detachedSession := ""
	if thread.VerifiedPark == nil {
		if resolver, ok := c.Artifacts.(DetachedSessionResolver); ok {
			observed, observeErr := resolver.DetachedSessions(ctx, []DetachedCandidate{{
				Address: address, Agent: agent, NativeID: binding.NativeID,
			}})
			if observeErr != nil {
				return ActorRecord{}, nil, fmt.Errorf("observe detached session for %+v: %w", address, observeErr)
			}
			if len(observed) == 1 && observed[0].Address == address {
				detached, detachedSession = true, observed[0].SessionName
			}
		}
	}
	eligible, err := DecideResume(ResumeEligibilityInput{
		Thread: thread, WorkingPathExists: pathExists, Binding: binding,
		Detached: detached, DetachedSession: detachedSession,
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
	// effects. A native session replacement in this window is a refusal, never
	// permission to create a different session under the same Pair address.
	currentBinding, err := bindings.ResolveEstablished(ctx, address.RepoScope, string(address.Tag), eligible.Profile.Agent)
	if err != nil {
		return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
	}
	// Only a RECONSTRUCT needs the native binding to have held: it is about to
	// rebuild a conversation from that id, so a replacement in this window is a
	// refusal. A reattach rejoins a session that never died, so there is no id
	// to have changed -- and requiring one here is what made every detached
	// thread unreattachable. Its equivalent guard is the session-name equality
	// in createflow's resume-boundary check.
	var profileRaw string
	if eligible.ReattachSession != "" {
		profileRaw, err = launcher.BuildCouchReattachLaunchProfile(
			string(address.Tag), eligible.Profile.Agent, eligible.Profile.Argv, eligible.ReattachSession,
		)
	} else {
		if err := launcher.RequireNativeResumeBinding(eligible.RequiredSessionID, currentBinding.NativeID, currentBinding.Status); err != nil {
			return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
		}
		profileRaw, err = launcher.BuildCouchResumeLaunchProfile(
			string(address.Tag), eligible.Profile.Agent, eligible.Profile.Argv, eligible.RequiredSessionID,
		)
	}
	if err != nil {
		return ActorRecord{}, nil, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
	}
	args := StartArgs{
		Worktree: Worktree(thread.StartingPath), Cwd: eligible.WorkingPath,
		Stack: eligible.Profile.Agent, ExtraArgs: cloneArgv(eligible.Profile.Argv),
	}
	return c.launchTrackedThread(trackedThreadLaunch{
		Context: ctx,
		Thread:  thread, Nonce: nonce, Args: args, StartedAt: startedAt,
		ProfileRaw: profileRaw, Resume: true,
	})
}
