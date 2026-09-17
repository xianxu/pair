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
	"github.com/xianxu/pair/cmd/internal/orientation"
)

var ErrSwitchResolutionChanged = errors.New("switch-agent: thread or path preferences changed; review the parameters again")

type SwitchAgentRequest struct {
	Address             ThreadAddress
	Agent               string
	Argv                []string
	AcceptedFingerprint string
}

type PreparedAgentSwitch struct {
	SourceRevision uint64        `json:"source_revision"`
	Address        ThreadAddress `json:"address"`
	SourceAgent    string        `json:"source_agent"`
	WorkingPath    string        `json:"working_path"`
	Profile        LaunchProfile `json:"profile"`
	Fingerprint    string        `json:"fingerprint"`
	record         ThreadRecord
	repoIdentity   string
}

type SwitchAgentOutcome string

const (
	SwitchRefused        SwitchAgentOutcome = "refused-before-park"
	SwitchParkIncomplete SwitchAgentOutcome = "park-incomplete"
	SwitchStartFailed    SwitchAgentOutcome = "start-failed"
	SwitchStarted        SwitchAgentOutcome = "started"
)

type SwitchAgentResult struct {
	Outcome     SwitchAgentOutcome
	Record      ActorRecord
	Handle      Handle
	Orientation *orientation.Request
	Warning     string
}

func (r SwitchAgentResult) Started() (StartResult, bool) {
	if r.Outcome != SwitchStarted {
		return StartResult{}, false
	}
	return StartResult{Record: r.Record, Handle: r.Handle}, true
}

// switchableWhenNothingRuns is the inactive-thread half of switch-agent's
// admission, for a record couch's own bookkeeping says nothing is running on.
//
// It used to be `record.VerifiedPark == nil` -> refuse, which read the park
// RECEIPT as proof of inactivity. #256 M2 widened who produces `parked`: a
// thread whose session is gone and whose ledger still resolves a conversation is
// parked with no receipt at all, and the switcher offers `switch-agent` on every
// parked row -- so the receipt check turned that offer into an action that
// always fails, which menu.go:1254 names as the way a switcher teaches an
// operator to distrust it.
//
// The fact the guard actually needs is that nothing is RUNNING, and since #256
// that is a fact about the session, not about bookkeeping. A switch launches a
// FRESH agent (launchTrackedThread with Fresh: true) and resumes no
// conversation, so a surviving session is the whole hazard: starting a second
// agent behind one is exactly the #272 shape from the other side.
//
// Strict at the action, per the optimistic-inventory rule: one exact observation
// for the single thread the operator chose. Unknown fails closed -- an
// unanswerable session is not an absent one, and the cost of being wrong here is
// two agents on one tree.
func (c *Couch) switchableWhenNothingRuns(ctx context.Context, record ThreadRecord) error {
	binding, err := c.recoverySession(ctx, record.Address)
	if errors.Is(err, ErrPairSessionBindingAbsent) {
		// No binding at all corroborates absence: nothing this record names can
		// be publishing one, because the occupied branch above already excluded
		// every incarnation that could.
		return nil
	}
	if err != nil {
		return fmt.Errorf("switch-agent: its session state could not be checked; inspect and retry: %w", err)
	}
	if binding.Present {
		return errors.New("switch-agent: its session is still up; park or detach the thread first")
	}
	return nil
}

// PrepareAgentSwitch reads the authoritative thread and shared path preference.
// Accepted argv is optional here so the form can first request a prefill.
func (c *Couch) PrepareAgentSwitch(ctx context.Context, address ThreadAddress, agent string, argv *[]string) (PreparedAgentSwitch, error) {
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
	if err := continuationGuard(record); err != nil {
		return PreparedAgentSwitch{}, err
	}
	if record.Park != nil {
		return PreparedAgentSwitch{}, errors.New("switch-agent: park is incomplete; use park retry/recover/abandon")
	}
	if hasOccupiedIncarnation(record) {
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
		for _, actor := range c.reg.Records() {
			if actor.Thread == address && actor.PID == incarnation.PID && actor.Identity == incarnation.Identity {
				owned = true
				break
			}
		}
		if !owned {
			return PreparedAgentSwitch{}, errors.New("switch-agent: source belongs to another owner; attach it to this Couch first")
		}
	} else if err := c.switchableWhenNothingRuns(ctx, record); err != nil {
		return PreparedAgentSwitch{}, err
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
	preference, found, err := c.Threads.GetPathLaunchPreference(repoIdentity, record.StartingPath)
	if err != nil {
		return PreparedAgentSwitch{}, err
	}
	input := LaunchProfileInputs{ExplicitAgent: agent, ExplicitArgv: argv}
	if found {
		input.Path = &preference
	}
	if c.RepoAgentDefault != nil {
		profile, exists, err := c.RepoAgentDefault(record.StartingPath, agent)
		if err != nil {
			return PreparedAgentSwitch{}, err
		}
		if exists {
			input.RepoDefault = &profile
		}
	}
	resolution, err := ResolveLaunchProfile(input)
	if err != nil {
		return PreparedAgentSwitch{}, err
	}
	if len(launcher.FormatLaunchParameters(resolution.Profile.Argv)) > 4096 {
		return PreparedAgentSwitch{}, errors.New("switch-agent: startup parameters exceed 4096 bytes")
	}
	// Prefill exposes remembered values for editing, including old resume
	// selectors. Only an explicit acceptance can authorize a fresh launch.
	if argv != nil {
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
		Default            *LaunchProfile
	}{record, resolution.Profile, preference.Revision, repoIdentity, input.RepoDefault}
	raw, err := json.Marshal(evidence)
	if err != nil {
		return PreparedAgentSwitch{}, err
	}
	digest := sha256.Sum256(raw)
	return PreparedAgentSwitch{
		Address: address, SourceAgent: source, WorkingPath: record.WorkingPath, SourceRevision: record.Revision,
		Profile: resolution.Profile, Fingerprint: hex.EncodeToString(digest[:]), record: record, repoIdentity: repoIdentity,
	}, nil
}

func (c *Couch) SwitchAgent(ctx context.Context, request SwitchAgentRequest) (SwitchAgentResult, error) {
	result := SwitchAgentResult{Outcome: SwitchRefused}
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.FreshRegistration == nil {
		return result, errors.New("switch-agent: fresh registration observer unavailable")
	}
	if request.AcceptedFingerprint == "" || request.Argv == nil {
		return result, errors.New("switch-agent: review and accept the startup parameters first")
	}
	prepared, err := c.PrepareAgentSwitch(ctx, request.Address, request.Agent, &request.Argv)
	if err != nil {
		return result, err
	}
	if prepared.Fingerprint != request.AcceptedFingerprint {
		return result, ErrSwitchResolutionChanged
	}
	owner, err := c.Proc.Current()
	if err != nil {
		return result, err
	}
	nonce, err := allocateStartNonce(c.Entropy)
	if err != nil {
		return result, err
	}
	profileRaw, err := launcher.BuildCouchFreshLaunchProfile(string(request.Address.Tag), request.Agent, request.Argv, string(AgentSourceExplicit), string(ArgvSourceExplicit))
	if err != nil {
		return result, err
	}
	contextData := orientation.OrientationContext{
		Tag: string(request.Address.Tag), WorkingPath: prepared.WorkingPath, SourceAgent: prepared.SourceAgent, TargetAgent: request.Agent,
	}
	if c.SwitchContext != nil {
		resolved, resolveErr := c.SwitchContext.Resolve(ctx, prepared.record)
		if resolveErr != nil {
			contextData.Unavailable = append(contextData.Unavailable, resolveErr.Error())
		} else {
			contextData = resolved
			contextData.TargetAgent = request.Agent
		}
	}
	body, err := orientation.BuildPrompt(contextData)
	if err != nil {
		return result, err
	}
	// Reserve room for the two bounded archive paths supplied by a completed
	// park; context size can never become a reason to abandon a parked source.
	if len(body) > orientation.MaxBodyBytes/2 {
		return result, errors.New("switch-agent: context references exceed the launch envelope budget")
	}
	thread := prepared.record
	if hasOccupiedIncarnation(thread) {
		parked, err := c.PairLifecycle.ParkExpected(ctx, thread.Address, thread.Revision)
		if err != nil {
			var revisionErr *ThreadRevisionError
			if errors.As(err, &revisionErr) && parked.Thread.Park == nil {
				return result, ErrSwitchResolutionChanged
			}
			result.Outcome = SwitchParkIncomplete
			return result, fmt.Errorf("switch-agent: park did not complete; use park retry/recover/abandon: %w", err)
		}
		thread = parked.Thread
		if parked.CleanupError != nil {
			result.Warning = parked.CleanupError.Error()
		}
	}
	if c.SwitchContext != nil {
		// Only update the preserved Pair paths after park. Native evidence was
		// selected while the outgoing agent still owned its ledger entry.
		if archive, ok := c.SwitchContext.(SwitchArchiveResolver); ok {
			if err := archive.ResolveArchive(thread, &contextData); err != nil {
				contextData.Unavailable = append(contextData.Unavailable, err.Error())
			}
		}
	}
	if finalBody, buildErr := orientation.BuildPrompt(contextData); buildErr == nil {
		body = finalBody
	} else {
		result.Warning = strings.TrimSpace(result.Warning + "\nSome source references were unavailable: " + buildErr.Error())
	}
	orient := orientation.Request{SchemaVersion: 1, Tag: string(request.Address.Tag), Agent: request.Agent, Attempt: nonce, Body: body}
	if err := ctx.Err(); err != nil {
		result.Outcome = SwitchStartFailed
		return result, err
	}
	startedAt := c.Clock.Now()
	thread, err = c.Threads.CommitStartClaim(thread.Address, thread.Revision, prepared.repoIdentity, startedAt, StartEvent{
		Kind: StartClaimed, Nonce: nonce, Owner: SupervisorOwner{PID: owner.PID, Identity: owner.Identity}, Profile: &prepared.Profile,
	})
	if err != nil {
		result.Outcome = SwitchStartFailed
		return result, fmt.Errorf("switch-agent: source parked but target claim failed; inspect the thread before retrying: %w", err)
	}
	record, handle, err := c.launchTrackedThread(trackedThreadLaunch{
		Context: ctx, Thread: thread, Nonce: nonce, Args: StartArgs{Worktree: Worktree(thread.StartingPath), Cwd: thread.WorkingPath, Stack: request.Agent, ExtraArgs: cloneArgv(request.Argv)},
		StartedAt: startedAt, ProfileRaw: profileRaw, Fresh: true, Orientation: &orient,
	})
	if err != nil {
		result.Outcome = SwitchStartFailed
		recovery := "inspect the occupied thread and recover it before trying again"
		if current, readErr := c.Threads.GetThread(thread.Address); readErr == nil && current.VerifiedPark != nil && !hasOccupiedIncarnation(current) {
			recovery = "the source is still parked; resume it or retry switch-agent"
		}
		return result, fmt.Errorf("switch-agent: target launch failed; %s: %w", recovery, err)
	}
	result.Outcome, result.Record, result.Handle, result.Orientation = SwitchStarted, record, handle, &orient
	return result, nil
}
