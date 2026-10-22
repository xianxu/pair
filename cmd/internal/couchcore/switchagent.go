package couchcore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

var ErrSwitchResolutionChanged = errors.New("switch-agent: thread or path preferences changed; review the parameters again")

// AgentSwitchGuard revalidates the coordinator's exact source binding and
// settled state at execution and immediately before the first mutation.
type AgentSwitchGuard interface {
	ConfirmAgentSwitch(context.Context, ThreadAddress, string, SwitchSourceEvidence) error
}

// AgentSwitchInterruptGuard performs the one cooperative input phase while
// SwitchAgent owns the lifecycle hold. ConfirmAgentSwitch still proves idle.
type AgentSwitchInterruptGuard interface {
	InterruptBeforeAgentSwitch(context.Context, ThreadAddress, string, SwitchSourceEvidence) error
}

type SwitchAgentRequest struct {
	Force               *AgentSwitchForceConsent `json:"-"`
	InterruptFirst      bool
	KnownConfig         bool
	Source              string
	Address             ThreadAddress
	Agent               string
	Argv                []string
	AcceptedFingerprint string
}

type PreparedAgentSwitch struct {
	Force          *AgentSwitchForceOffer `json:"-"`
	Configured     bool                   `json:"configured"`
	Conflict       bool                   `json:"conflict,omitempty"`
	Borrowed       *BorrowedArguments     `json:"borrowed,omitempty"`
	SourceRevision uint64                 `json:"source_revision"`
	Address        ThreadAddress          `json:"address"`
	SourceAgent    string                 `json:"source_agent"`
	WorkingPath    string                 `json:"working_path"`
	Profile        LaunchProfile          `json:"profile"`
	Fingerprint    string                 `json:"fingerprint"`
	record         ThreadRecord
	repoIdentity   string
	// state is the classification this preparation was ADMITTED on, carried to
	// the commit so both decisions rest on one observation. SwitchAgent used to
	// re-read `hasOccupiedIncarnation` to decide whether to park the source,
	// which is bookkeeping: a thread can be `parked` and still carry a start
	// claim its couch died mid-flight, so the preview admitted it and the commit
	// then tried to park a thread with no agent and failed `park-incomplete`
	// (#256 M2, round 3).
	state        ActionableThreadState
	sourceReason ThreadReason
}

// SwitchSourceEvidence carries only core's admitted classification. Its private
// fields prevent serialized operation arguments from inventing vacancy proof.
type SwitchSourceEvidence struct {
	state        ActionableThreadState
	reason       ThreadReason
	address      ThreadAddress
	revision     uint64
	workingPath  string
	repoIdentity string
}

// SourceEvidence exposes the source mode already admitted by this preparation.
func (p PreparedAgentSwitch) SourceEvidence() SwitchSourceEvidence {
	return SwitchSourceEvidence{state: p.state, reason: p.sourceReason, address: p.record.Address, revision: p.record.Revision, workingPath: p.record.WorkingPath, repoIdentity: p.repoIdentity}
}

// Vacant is true only for an admitted non-live classification, never disconnect.
func (e SwitchSourceEvidence) Vacant() bool {
	return e.state != ThreadLive && SwitchableState(e.state, e.reason)
}

// RepositoryIdentity names the canonical identity admitted by preparation.
func (e SwitchSourceEvidence) RepositoryIdentity() string { return e.repoIdentity }

// WorkingPath names the saved path whose current workspace must be verified.
func (e SwitchSourceEvidence) WorkingPath() string { return e.workingPath }

// MatchesRecord fences proof to the exact retained source generation and path.
func (e SwitchSourceEvidence) MatchesRecord(record ThreadRecord) bool {
	return e.address == record.Address && e.revision == record.Revision && e.workingPath == record.WorkingPath
}

type SwitchAgentOutcome string

const (
	SwitchRefused        SwitchAgentOutcome = "refused-before-park"
	SwitchParkIncomplete SwitchAgentOutcome = "park-incomplete"
	SwitchStartFailed    SwitchAgentOutcome = "start-failed"
	SwitchStarted        SwitchAgentOutcome = "started"
)

type SwitchAgentResult struct {
	Outcome SwitchAgentOutcome
	Record  ActorRecord
	Handle  Handle
	Notice  string
}

func (r SwitchAgentResult) Started() (StartResult, bool) {
	if r.Outcome != SwitchStarted {
		return StartResult{}, false
	}
	return (StartResult{Record: r.Record, Handle: r.Handle}).Started()
}

func (r SwitchAgentResult) Warning() string {
	started, _ := r.Started()
	var parts []string
	if r.Notice != "" {
		parts = append(parts, r.Notice)
	}
	if notice := started.Warning(); notice != "" {
		parts = append(parts, notice)
	}
	return strings.Join(parts, "; ")
}

// classifyForAction answers "what is this thread, right now" through the SAME
// evidence pass and the SAME rule the switcher's rows come from.
//
// An action guard consumes the classification; it does not re-derive one. Two
// successive re-derivations of "nothing runs here" shipped in #256 M2 and both
// were wrong in a different direction, which is the whole argument: a second
// derivation drifts toward whichever cases its author thought about.
//
// Live proof comes from couch's OWN registry, which is what a console hosting a
// pty child records, so a thread couch is hosting reads `live` here exactly as
// it does in the switcher. The evidence pass is one round for the single thread
// the operator acted on -- the strict half of optimistic inventory.
//
// A registry record is a claim, not proof: nothing removes one when its child
// exits, so each is probed (pair#378). Live joins the hosted proof; Dead is
// residue and proves nothing; Unknown joins the evidence's Unproven side, so
// ClassifyThread's own precedence decides it -- live and busy outrank it, and
// otherwise it fails closed as `unusable/unknown`.
func (c *Couch) classifyForAction(ctx context.Context, address ThreadAddress) (ActionableThreadState, ThreadReason, error) {
	hosted := make([]LiveTTYObservation, 0, 4)
	var unproven []ProcessIdentity
	for _, actor := range c.actorRegistry().Records() {
		if actor.Thread != address {
			continue
		}
		process := ProcessIdentity{PID: actor.PID, Identity: actor.Identity}
		switch c.Liveness(actor) {
		case Live:
			hosted = append(hosted, LiveTTYObservation{Address: actor.Thread, Process: process})
		case Unknown:
			unproven = append(unproven, process)
		}
	}
	snapshot, evidence, err := c.gatherThreadEvidence(ctx, hosted, func(record ThreadRecord) bool {
		return record.Address == address
	})
	if err != nil {
		return "", "", err
	}
	for i := range snapshot.Records {
		if snapshot.Records[i].Address != address {
			continue
		}
		threadEvidence := evidence[address]
		threadEvidence.Unproven = append(threadEvidence.Unproven, unproven...)
		state, reason := ClassifyThread(snapshot.Records[i], threadEvidence)
		return state, reason, nil
	}
	return "", "", fmt.Errorf("%w: %+v", ErrThreadNotFound, address)
}

// PrepareAgentSwitch reads the authoritative thread and shared path preference.
// Accepted argv is optional here so the form can first request a prefill.
func (c *Couch) PrepareAgentSwitch(ctx context.Context, address ThreadAddress, agent string, argv *[]string) (PreparedAgentSwitch, error) {
	return c.prepareAgentSwitch(ctx, address, agent, argv, false)
}

// PrepareKnownAgentSwitch accepts only an existing successful path configuration
// or an explicit repository default, including configured empty arguments.
func (c *Couch) PrepareKnownAgentSwitch(ctx context.Context, address ThreadAddress, agent string) (PreparedAgentSwitch, error) {
	return c.prepareAgentSwitch(ctx, address, agent, nil, true)
}

func (c *Couch) prepareAgentSwitch(ctx context.Context, address ThreadAddress, agent string, argv *[]string, known bool) (PreparedAgentSwitch, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return PreparedAgentSwitch{}, err
	}
	if c == nil || c.Threads == nil || c.Runner == nil {
		return PreparedAgentSwitch{}, errors.New("switch-agent: launch services unavailable")
	}
	if err := validateThreadAddress(address); err != nil {
		return PreparedAgentSwitch{}, err
	}
	if !launcher.IsSupportedAgent(agent) {
		return PreparedAgentSwitch{}, fmt.Errorf("switch-agent: unsupported agent %q", agent)
	}
	record, err := c.Threads.GetThread(address)
	if err != nil {
		return PreparedAgentSwitch{}, err
	}
	persisted := record
	record, _, err = c.ownedAttachmentFacts(record)
	if err != nil {
		return PreparedAgentSwitch{}, err
	}
	if record.LatestLaunchProfile != nil && record.LatestLaunchProfile.Agent == agent {
		return PreparedAgentSwitch{}, errors.New("switch-agent: select a different agent")
	}

	if err := continuationGuard(record); err != nil {
		return PreparedAgentSwitch{}, c.withParkRecoveryAdvice(ctx, record, err)
	}
	if record.Park != nil {
		return PreparedAgentSwitch{}, c.withParkRecoveryAdvice(ctx, record, errors.New("switch-agent: park is incomplete"))
	}
	// The admission rule, consumed rather than re-derived: exactly the states
	// the switcher offers switch-agent on, decided by the same classification
	// that produced the offer (#256 M2, BR-33).
	state, reason, err := c.classifyForAction(ctx, address)
	if err != nil {
		return PreparedAgentSwitch{}, fmt.Errorf("switch-agent: its state could not be classified; inspect and retry: %w", err)
	}
	if !SwitchableState(state, reason) {
		detail := string(state)
		if reason != "" {
			detail += "/" + string(reason)
		}
		return PreparedAgentSwitch{}, fmt.Errorf(
			"switch-agent: thread is %s; only a live thread this couch hosts or one with nothing running can switch agents", detail)
	}
	if state == ThreadLive {
		if !hasOccupiedIncarnation(record) {
			// Couch hosts it, but the record names no incarnation to park --
			// and SwitchAgent parks the source only when it does. Admitting this
			// starts a second agent on the same tree.
			return PreparedAgentSwitch{}, errors.New(
				"switch-agent: couch hosts this thread but its record names no incarnation to park; detach or park it first")
		}
		if c.PairLifecycle == nil {
			return PreparedAgentSwitch{}, errors.New("switch-agent: Pair lifecycle controller unavailable")
		}
		incarnation, err := soleParkableIncarnation(record)
		if err != nil {
			return PreparedAgentSwitch{}, fmt.Errorf("switch-agent: occupied thread: %w", err)
		}
		if incarnation.State != IncarnationLive || observeExactProcess(c.Proc, ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity}) != Live {
			return PreparedAgentSwitch{}, errors.New("switch-agent: source is not a verified live actor; attach or recover it first")
		}

		owned := false
		for _, actor := range c.actorRegistry().Records() {
			if actor.Thread == address && actor.PID == incarnation.PID && actor.Identity == incarnation.Identity {
				owned = true
				break
			}
		}
		if !owned {
			return PreparedAgentSwitch{}, errors.New("switch-agent: source belongs to another owner; attach it to this Couch first")
		}
	}
	if !c.workingPathExists(record) {
		return PreparedAgentSwitch{}, refuseResume(ResumePathMissing, "switch-agent: saved working path is unavailable")
	}
	if c.SwitchLaunchCheck != nil {
		if err := c.SwitchLaunchCheck(agent); err != nil {
			return PreparedAgentSwitch{}, err
		}
	}
	repoIdentity, err := c.resolveRepoIdentity(ctx, record.StartingPath)
	if err != nil {
		return PreparedAgentSwitch{}, err
	}
	doc, _, err := c.previewRepositoryPreferences(ctx)
	if err != nil && argv == nil {
		return PreparedAgentSwitch{}, err
	}
	if err != nil {
		doc = RepoPreferences{}
	}
	var saved *RepoLaunchPreference
	if value, found := doc.Repositories[repoIdentity]; found {
		saved = &value
	}
	arguments, err := ResolveRepoArguments(saved, agent, argv)
	if err != nil {
		return PreparedAgentSwitch{}, err
	}
	own, err := ResolveRepoArguments(saved, agent, nil)
	if err != nil {
		return PreparedAgentSwitch{}, err
	}
	var borrowed *BorrowedArguments
	if known && !own.Found {
		if own.Conflict {
			return PreparedAgentSwitch{}, fmt.Errorf("switch-agent: repository arguments for %s conflict; explicitly accept startup parameters in the switch form", agent)
		}
		return PreparedAgentSwitch{}, fmt.Errorf("switch-agent: no saved configuration for %s in this repository; accept the switch form's marked offer or pass --agent-args explicitly", agent)
	}
	if argv == nil && !own.Found && !own.Conflict && !known {
		if offer, found, borrowErr := ChooseBorrowedArguments(doc.Repositories, repoIdentity, agent); borrowErr != nil {
			return PreparedAgentSwitch{}, borrowErr
		} else if found {
			borrowed = &offer
			arguments.Argv = cloneArgv(offer.Argv)
		}
	}
	resolution := LaunchProfileResolution{Profile: LaunchProfile{Agent: agent, Argv: cloneArgv(arguments.Argv)}}
	if len(launcher.FormatLaunchParameters(resolution.Profile.Argv)) > 4096 {
		return PreparedAgentSwitch{}, errors.New("switch-agent: startup parameters exceed 4096 bytes")
	}
	// Prefill exposes remembered values for editing, including old resume
	// selectors. Execution and known-configuration admission require fresh args.
	if argv != nil || known {
		if err := launcher.ValidateFreshAgentArgs(agent, resolution.Profile.Argv); err != nil {
			return PreparedAgentSwitch{}, err
		}
	}
	source := ""
	if record.LatestLaunchProfile != nil {
		source = record.LatestLaunchProfile.Agent
	}
	evidence := struct {
		Record             ThreadRecord
		Profile            LaunchProfile
		PreferenceRevision uint64
		RepoIdentity       string
		Borrowed           *BorrowedArguments
		Conflict           bool
		SourceState        ActionableThreadState
		SourceReason       ThreadReason
	}{record, resolution.Profile, own.Revision, repoIdentity, borrowed, own.Conflict, state, reason}
	raw, err := json.Marshal(evidence)
	if err != nil {
		return PreparedAgentSwitch{}, err
	}
	digest := sha256.Sum256(raw)
	return PreparedAgentSwitch{
		Address: address, SourceAgent: source, WorkingPath: record.WorkingPath, SourceRevision: record.Revision,
		Profile: resolution.Profile, Fingerprint: hex.EncodeToString(digest[:]), record: persisted, repoIdentity: repoIdentity,
		Configured: own.Found, Conflict: own.Conflict, Borrowed: borrowed,
		state: state, sourceReason: reason,
	}, nil
}

func (c *Couch) SwitchAgent(ctx context.Context, request SwitchAgentRequest) (_ SwitchAgentResult, returnedErr error) {
	result := SwitchAgentResult{Outcome: SwitchRefused}
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.FreshRegistration == nil {
		return result, errors.New("switch-agent: fresh registration observer unavailable")
	}
	if request.AcceptedFingerprint == "" || (!request.KnownConfig && request.Argv == nil) {
		return result, errors.New("switch-agent: review and accept the startup parameters first")
	}
	if request.KnownConfig && request.Argv != nil {
		return result, errors.New("switch-agent: saved configuration cannot include explicit arguments")
	}
	prepare := func() (PreparedAgentSwitch, error) {
		if request.KnownConfig {
			return c.PrepareKnownAgentSwitch(ctx, request.Address, request.Agent)
		}
		return c.PrepareAgentSwitch(ctx, request.Address, request.Agent, &request.Argv)
	}
	if request.InterruptFirst && c.SwitchGuard == nil {
		return result, errors.New("switch-agent: cooperative interrupt guard unavailable")
	}
	if request.Force == nil && !request.InterruptFirst && (c.SwitchGuard != nil || request.Source != "") {
		if c.SwitchGuard == nil {
			return result, errors.New("switch-agent: source guard unavailable")
		}
		source, err := prepare()
		if err != nil {
			return result, err
		}
		if source.Fingerprint != request.AcceptedFingerprint {
			return result, ErrSwitchResolutionChanged
		}
		if err := c.SwitchGuard.ConfirmAgentSwitch(ctx, request.Address, request.Source, source.SourceEvidence()); err != nil {
			return result, err
		}
	}
	ctx, release, err := c.hold(ctx, request.Address, "switch-agent")
	if err != nil {
		return result, err
	}
	defer release()
	prepared, err := prepare()
	if err != nil {
		return result, err
	}
	if prepared.Fingerprint != request.AcceptedFingerprint {
		return result, ErrSwitchResolutionChanged
	}
	if request.Force != nil {
		if request.InterruptFirst {
			return result, errors.New("switch-agent: force cannot include cooperative interrupt")
		}
		source, err := c.consumeAgentSwitchForce(request.Force, prepared)
		if err != nil {
			return result, err
		}
		request.Source = source
	}
	if request.InterruptFirst && !prepared.SourceEvidence().Vacant() {
		cooperative, ok := c.SwitchGuard.(AgentSwitchInterruptGuard)
		if !ok {
			return result, errors.New("switch-agent: cooperative interrupt unavailable")
		}
		if err := cooperative.InterruptBeforeAgentSwitch(ctx, request.Address, request.Source, prepared.SourceEvidence()); err != nil {
			return result, err
		}
		// Interruption is not permanent admission: reclassify the exact accepted
		// generation before the final idle/Git guard and normal park transaction.
		prepared, err = prepare()
		if err != nil {
			return result, err
		}
		if prepared.Fingerprint != request.AcceptedFingerprint {
			return result, ErrSwitchResolutionChanged
		}
	}
	owner, err := c.Proc.Current()
	if err != nil {
		return result, err
	}
	nonce, err := allocateStartNonce(c.Entropy)
	if err != nil {
		return result, err
	}
	// Standalone switches own one terminal attempt, so nested launch capture
	// remains evidence-only until the final source-restoration result is known.
	var attempt *OperationAttempt
	if OperationAttemptFromContext(ctx) == nil {
		attempt, err = c.BeginOperationAttempt("switch-agent", LifecycleFailureTarget{Address: prepared.record.Address, StartingPath: prepared.record.StartingPath})
		if err != nil {
			return result, err
		}
		attempt.nonce = nonce
		ctx = WithOperationAttempt(ctx, attempt)
	}
	defer func() {
		if returnedErr != nil {
			returnedErr = c.finishSwitchFailure(trackedThreadLaunch{Context: ctx, Thread: prepared.record, Nonce: nonce, Args: StartArgs{Stack: request.Agent}, Fresh: true}, returnedErr)
		}
		if attempt != nil {
			_, returnedErr = attempt.Finish(nil, returnedErr)
		}
	}()
	profileRaw, err := launcher.BuildCouchFreshLaunchProfile(string(request.Address.Tag), request.Agent, prepared.Profile.Argv, string(AgentSourceExplicit), string(ArgvSourceExplicit))
	if err != nil {
		return result, err
	}
	if request.Force != nil {
		guard, ok := c.SwitchGuard.(AgentSwitchForceGuard)
		if !ok {
			return result, errors.New("switch-agent: local force guard unavailable")
		}
		if err := guard.ConfirmForcedAgentSwitch(ctx, request.Address, request.Source, prepared.SourceEvidence()); err != nil {
			return result, err
		}
	} else if c.SwitchGuard != nil || request.Source != "" {
		if err := c.SwitchGuard.ConfirmAgentSwitch(ctx, request.Address, request.Source, prepared.SourceEvidence()); err != nil {
			return result, err
		}
	}
	thread := prepared.record
	// Park the source only when an agent is actually RUNNING, which is what the
	// classification says and what `hasOccupiedIncarnation` only approximates.
	// PrepareAgentSwitch has already refused a live row with nothing to park, so
	// this branch always has an incarnation to act on.
	if prepared.state == ThreadLive {
		thread, err = c.adoptOwnedAttachment(ctx, thread)
		if err != nil {
			return result, err
		}
		parked, err := c.PairLifecycle.ParkExpected(ctx, thread.Address, thread.Revision)
		if err != nil {
			var revisionErr *ThreadRevisionError
			if errors.As(err, &revisionErr) && parked.Thread.Park == nil {
				return result, ErrSwitchResolutionChanged
			}
			result.Outcome = SwitchParkIncomplete
			return result, fmt.Errorf("switch-agent: park did not complete; %s: %w", c.ParkRecoveryAdvice(ctx, thread.Address), err)
		}
		thread = parked.Thread
		if parked.CleanupError != nil {
			result.Notice = parked.CleanupError.Error()
		}
	}
	// Clear whatever the last couch left behind, on the same terms resume and
	// archive do (#256 M2). A thread can be `parked` and still carry a start
	// claim its couch died mid-flight -- the store's one-incarnation-at-a-time
	// invariant would then refuse the target claim below, uncoded, after the
	// source had already been parked. Screened before any write, and refusing
	// with a code if the debris cannot be proved orphaned.
	if cleared, clearErr := c.clearLifecycleDebris(thread); clearErr != nil {
		if errors.Is(clearErr, ErrThreadRolledBack) {
			result.Outcome = SwitchStartFailed
			return result, fmt.Errorf("switch-agent: the source carried nothing but an unfinished start and was rolled back; start a new thread instead")
		}
		result.Outcome = SwitchStartFailed
		return result, fmt.Errorf("switch-agent: %w", clearErr)
	} else if cleared != nil {
		thread = *cleared
	}
	// A retry from the same verified park retains its exact tty capture;
	// a separately parked replacement must supersede the older source evidence.
	if thread.VerifiedPark != nil && (thread.PreviousSession == nil || thread.PreviousSession.Source != toPersistedParkIdentity(thread.VerifiedPark.Identity)) {
		if previous := previousSessionEvidence(prepared.record, thread); previous != nil {
			thread, err = c.Threads.RecordPreviousSession(thread.Address, thread.Revision, *previous)
			if err != nil {
				result.Outcome = SwitchStartFailed
				return result, fmt.Errorf("switch-agent: preserve previous session: %w", err)
			}
		}
	}
	if err := ctx.Err(); err != nil {
		result.Outcome = SwitchStartFailed
		return result, err
	}
	startedAt := c.Clock.Now()
	thread, err = c.Threads.CommitStartClaim(thread.Address, thread.Revision, prepared.repoIdentity, startedAt, StartEvent{
		RememberConfiguration: true,
		Kind:                  StartClaimed, Nonce: nonce, Owner: SupervisorOwner{PID: owner.PID, Identity: owner.Identity}, Profile: &prepared.Profile,
	})
	if err != nil {
		result.Outcome = SwitchStartFailed
		return result, fmt.Errorf("switch-agent: source parked but target claim failed; inspect the thread before retrying: %w", err)
	}
	if err := c.prepareTrackedWorkspace(ctx, thread, nonce, false); err != nil {
		result.Outcome = SwitchStartFailed
		return result, errors.Join(err, c.rollbackTrackedStart(thread, nonce))
	}
	record, handle, err := c.launchTrackedThread(trackedThreadLaunch{
		Context: ctx, Thread: thread, Nonce: nonce, Args: StartArgs{Worktree: Worktree(thread.StartingPath), Cwd: thread.WorkingPath, Stack: request.Agent, ExtraArgs: cloneArgv(prepared.Profile.Argv)},
		StartedAt: startedAt, ProfileRaw: profileRaw, Fresh: true,
	})
	if err != nil {
		result.Outcome = SwitchStartFailed
		recovery := "inspect the occupied thread and recover it before trying again"
		if current, readErr := c.Threads.GetThread(thread.Address); readErr == nil && current.VerifiedPark != nil && !hasOccupiedIncarnation(current) {
			recovery = "the source is still parked; resume it or retry switch-agent"
		}
		return result, fmt.Errorf("switch-agent: target launch failed; %s: %w", recovery, err)
	}
	result.Outcome, result.Record, result.Handle = SwitchStarted, record, handle
	return result, nil
}
