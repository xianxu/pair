package couchcore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func recoverySelectionDigest(record ThreadRecord, selection RecoverySelection, terminal RecoveryRetirement) string {
	b, _ := json.Marshal(struct {
		Record    ThreadRecord
		Selection RecoverySelection
		Terminal  string
		Stops     []RecoveryStopProcess
	}{record, selection, terminal.Digest, terminal.Stops})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (c *Couch) recoveryObservation(ctx context.Context, record ThreadRecord, requests ...RecoveryObservationRequest) RecoveryObservation {
	observation := RecoveryObservation{Agent: RecordAgent(record), Incomplete: true, Reasons: []string{"Previous screen or native evidence unavailable; working context may be lost."}}
	request := RecoveryObservationRequest{ObservedAt: c.Clock.Now()}
	if len(requests) > 0 {
		request = requests[0]
	}
	if c.RecoveryEvidence != nil {
		for attempt := 0; attempt < 2; attempt++ {
			got, err := c.RecoveryEvidence(ctx, record, request)
			if err != nil {
				// A later failed acquisition cannot erase already observed facts.
				// They remain descriptive: the aggregate is incomplete below.
				if got.Screen.Availability == "" {
					got.Screen = observation.Screen
				}
				if got.Capture.Raw == "" {
					got.Capture = observation.Capture
				}
				if got.Source == nil {
					got.Source = observation.Source
				}
				if got.Generation == "" {
					got.Generation = observation.Generation
				}
				if got.Agent == "" {
					got.Agent = observation.Agent
				}
				if got.Branch == "" {
					got.Branch, got.Issue = observation.Branch, observation.Issue
				}
				if len(got.Ranked) == 0 {
					got.Ranked = observation.Ranked
				}
				got.Reasons = append(append([]string(nil), observation.Reasons...), got.Reasons...)
			}
			if got.BytesRead < request.BytesAlreadyRead {
				got.BytesRead = request.BytesAlreadyRead
			}
			observation = got
			if got.BytesRead > request.BytesAlreadyRead {
				request.BytesAlreadyRead = got.BytesRead
			}
			if request.Source == nil {
				request.Source = threadrecord.CloneNativeRecoverySource(got.Source)
			}
			if err == nil {
				if !got.Incomplete {
					break
				}
			} else {
				observation.Incomplete = true
				observation.Reasons = append(observation.Reasons, rowtext.Sanitize(err.Error()), "Previous screen or native evidence unavailable; working context may be lost.")
			}
			if ctx.Err() != nil {
				break
			}
		}
	}
	if observation.Agent == "" {
		observation.Agent = RecordAgent(record)
	}
	if observation.Branch == "" && c.Git != nil {
		observation.Branch, observation.Issue = (OSRecoveryEvidenceReader{Git: c.Git}).recoveryBranchIssue(ctx, record)
	}
	return observation
}

func (c *Couch) executePreparedRecovery(ctx context.Context, selection RecoverySelection, confirmation string) (result RecoveryResumeResult, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if selection.Kind != RecoveryProven && selection.Kind != RecoveryInferred && selection.Kind != RecoveryFresh {
		return result, errors.New("choose Resume from disk or Start fresh")
	}
	_, record, err := c.recoveryTarget(ctx, selection.Target)
	if err != nil {
		if selection.Kind == RecoveryFresh && selection.Target.Path != "" && errors.Is(err, ErrThreadNotFound) {
			return c.startFreshRecoveryWithoutRecord(ctx, selection)
		}
		return result, err
	}
	ctx, release, err := c.hold(ctx, record.Address, "confirmed-recovery")
	if err != nil {
		return result, err
	}
	defer release()
	current, err := c.Threads.PreviewThreadContext(ctx, record.Address)
	if err != nil {
		return result, err
	}
	record = current
	selection.Reference, err = c.recoverySlotAddress(ctx, record)
	if err != nil {
		return result, err
	}
	if selection.Kind == RecoveryProven {
		listed, listErr := c.observeRecoverySession(ctx, record, RecoveryEvidence{})
		if listErr == nil && listed.Presence == PresencePresent {
			actor, handle, err := c.ResumeContextWith(ctx, record.Address, ResumeOptions{WarmOnly: true})
			return RecoveryResumeResult{Mode: Continue, Start: StartResult{Record: actor, Handle: handle}, Notice: "Continue"}, err
		}
	}
	// Resolve an advertised key before mode-specific profile and lifecycle
	// guards: the candidate route can represent Fresh as well as native resume.
	request := RecoveryObservationRequest{ObservedAt: c.Clock.Now()}
	if selection.Kind == RecoveryInferred {
		selector, err := decodeRecoveryCandidateKey(selection.CandidateKey)
		if err != nil {
			return result, err
		}
		if selector.Address != record.Address {
			return result, errors.New("recovery selector belongs to another thread")
		}
		if selector.Source != nil {
			if err := threadrecord.ValidateNativeRecoverySourceAt(selector.Source, c.Clock.Now(), record.Revision); err != nil {
				return result, err
			}
			request.Source = selector.Source
		}
	}
	observe := func() RecoveryObservation {
		observationCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		return c.recoveryObservation(observationCtx, record, request)
	}
	keyedSelection := selection.Kind == RecoveryInferred
	var observed RecoveryObservation
	var selectedSnapshot *threadrecord.RecoveryContext
	var choice RecoveryChoice
	if keyedSelection {
		observed = observe()
		preview, err := DecideSlotRecovery(record, observed, c.Clock.Now())
		if err != nil {
			return result, err
		}
		choice, err = selectRecoveryChoice(preview, selection.CandidateKey)
		if err != nil {
			return result, err
		}
		if choice.NativeID == "" {
			selection.Kind = RecoveryFresh
		} else if preview.Snapshot != nil && preview.Snapshot.Source != nil {
			selectedSnapshot = threadrecord.CloneRecoveryContext(preview.Snapshot)
		}
	}
	profile, err := c.recoveryLaunchProfileForSelection(ctx, record, selection)
	if err != nil {
		return result, err
	}
	if selection.Kind != RecoveryFresh {
		if record.Park != nil {
			return result, refuseResume(ResumeParking, "a park transaction is unfinished; "+c.ParkRecoveryAdvice(ctx, record.Address))
		}
		if err := continuationGuard(record); err != nil {
			return result, err
		}
		if startClaimed(record) {
			if len(record.Incarnations) != 1 || record.Incarnations[0].Start == nil {
				return result, errors.New("multiple source claims require Start fresh preserving their records")
			}
			claim := record.Incarnations[0].Start
			owner := observeExactProcess(c.Proc, ProcessIdentity{PID: claim.OwnerPID, Identity: claim.OwnerIdentity})
			if !DecideStaleStart(record, ThreadEvidence{ObservedAt: c.Clock.Now(), StartOwner: owner}) {
				return result, errors.New("source is still starting; wait for its bounded launch or choose Start fresh")
			}
		}
	}
	if !keyedSelection {
		// A direct selection consumes no native candidates; the scan walks the
		// whole native store under the hub effect gate (pair#726).
		request.SkipCandidates = true
		observed = observe()
	}
	if choice.Agent == "" {
		choice.Agent = profile.Agent
	} else if choice.NativeID != "" && choice.Agent != profile.Agent {
		return result, errors.New("candidate no longer matches the saved agent")
	}
	if selection.Kind == RecoveryProven {
		binding, err := c.resumeEvidence(ctx, record)
		if err != nil {
			return result, err
		}
		if !coldResumeAuthorized(binding) {
			return result, refuseResolvedBinding(binding)
		}
		choice.NativeID = binding.NativeID
	}
	var retirement RecoveryRetirement
	var notes []string
	if c.RecoveryRetirement != nil {
		retirement, err = c.RecoveryRetirement(ctx, record)
		if err != nil {
			if selection.Kind != RecoveryFresh {
				return result, fmt.Errorf("source ownership unavailable; Start fresh leaves unproved remnants untouched: %w", err)
			}
			notes = append(notes, "unproved remnants left untouched: "+rowtext.Sanitize(err.Error()))
			retirement = RecoveryRetirement{}
		}
	} else if c.RecoveryTerminal != nil && c.recoverySourceInactive(record) == nil {
		retire, err := c.prepareRecoveryRetirement(ctx, record)
		if err != nil {
			return result, err
		}
		if retire != nil {
			retirement = RecoveryRetirement{Digest: "strict inactive source", Check: func(context.Context) error { return c.recoverySourceInactive(record) }, Retire: func(ctx context.Context) error { _, err := retire(ctx, record); return err }}
		}
	}
	if selection.Kind != RecoveryFresh {
		if err := c.checkRecoveryConversationInUse(ctx, record, choice); err != nil {
			return result, err
		}
		if len(retirement.Stops) == 0 {
			if err := c.recoverySourceInactive(record); err != nil {
				return result, err
			}
			if retirement.Retire == nil {
				if err := c.confirmRecoveryAbsent(ctx, record); err != nil {
					return result, err
				}
			}
		}
	}
	var parkHelper *ThreadIncarnation
	if len(retirement.Stops) > 0 {
		parkHelper = c.recoveryParkHelper(ctx, record)
		if parkHelper != nil {
			named := false
			for _, stop := range retirement.Stops {
				named = named || stop.PID == parkHelper.PID && stop.Identity == parkHelper.Identity
			}
			if !named {
				retirement.Stops = append(retirement.Stops, RecoveryStopProcess{PID: parkHelper.PID, Identity: parkHelper.Identity, Role: "client-helper"})
			}
		}
	}
	if selection.Kind == RecoveryFresh {
		var treeNotes []string
		if retirement, treeNotes, err = c.withSlotTree(ctx, record, retirement); err != nil {
			return result, err
		}
		notes = append(notes, treeNotes...)
	}
	witness := recoveryConfirmationWitness{Address: record.Address, Revision: record.Revision, Selection: selection, Digest: recoverySelectionDigest(record, selection, retirement), NativeID: choice.NativeID}
	if confirmation != "" {
		if err := c.consumeRecoveryConfirmation(ctx, confirmation, witness); err != nil {
			return result, err
		}
	} else if len(retirement.Stops) > 0 {
		offer, err := c.issueRecoveryConfirmation(ctx, witness, retirement.Stops)
		if err != nil {
			return result, err
		}
		return result, &RecoveryConfirmationRequired{Offer: offer}
	}
	if retirement.Check != nil {
		if err := retirement.Check(ctx); err != nil {
			return result, err
		}
	}
	repo, finish, err := c.prepareRecoveryLaunch(ctx, record, profile)
	if err != nil {
		return result, err
	}
	defer func() { retErr = errors.Join(retErr, finish(retErr == nil)) }()
	orient := &threadrecord.RecoveryOrientation{Branch: observed.Branch, Issue: observed.Issue}
	if (observed.Source == nil || selection.Kind == RecoveryFresh && observed.Screen.Availability == ScreenAvailable) && observed.Capture.RawIdentity.Inode != 0 && !observed.Capture.CapturedAt.IsZero() && !c.Clock.Now().Before(observed.Capture.CapturedAt) && c.Clock.Now().Before(observed.Capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)) {
		if c.RecoveryPreserve == nil {
			return result, errors.New("capture preservation unavailable; source was not stopped")
		}
		preserved, err := c.RecoveryPreserve(ctx, record, observed)
		if err != nil {
			return result, err
		}
		observed = preserved
		capture := preserved.Capture
		orient.Capture = &capture
	} else {
		orient.Unavailable = "Previous screen unavailable or expired; use branch and issue evidence. Working context may be lost."
	}
	intent := threadrecord.RecoveryIntent{Mode: selection.Kind, Address: toPersistedThreadAddress(record.Address), SourceRevision: record.Revision, CreatedAt: c.Clock.Now(), Agent: profile.Agent, Orientation: orient, SelectedKey: fmt.Sprintf("%s:%s:%d", selection.Kind, record.Address.Tag, record.Revision), NativeID: choice.NativeID}
	if selection.Kind == RecoveryInferred && selectedSnapshot != nil {
		intent = threadrecord.RecoveryIntent{Context: *selectedSnapshot, SelectedKey: choice.Key, NativeID: choice.NativeID, SourceRevision: selectedSnapshot.Source.Revision}
	} else if selection.Kind == RecoveryInferred {
		preview, err := DecideSlotRecovery(record, observed, c.Clock.Now())
		if err != nil {
			return result, err
		}
		var selected *RecoveryChoice
		for i := range preview.Candidates {
			if preview.Candidates[i].NativeID == choice.NativeID && preview.Candidates[i].Agent == choice.Agent {
				selected = &preview.Candidates[i]
				break
			}
		}
		if selected == nil || preview.Snapshot == nil {
			return result, errors.New("selected transcript disappeared during preservation")
		}
		choice = *selected
		intent = threadrecord.RecoveryIntent{Context: *preview.Snapshot, SelectedKey: choice.Key, NativeID: choice.NativeID, SourceRevision: record.Revision}
	}
	// The intent is compared with its persisted copy (start fence, workspace
	// recheck, fresh replacement), so it must equal that copy from the start:
	// the clock's monotonic reading survives only in memory (pair#605).
	if intent, err = persistedRecoveryIntent(intent); err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := c.verifyNativeRecoverySelection(ctx, record, intent); err != nil {
		return result, err
	}
	phase := recoveryPrepared
	record, err = c.Threads.PreserveRecoveryBeforeRetirement(record.Address, record.Revision, intent)
	if err != nil {
		return result, err
	}
	phase, err = AdvanceRecoveryExecution(phase, recoveryPreserved)
	if err != nil {
		return result, err
	}
	if retirement.Check != nil {
		if err := retirement.Check(ctx); err != nil {
			return result, err
		}
	}
	if retirement.Retire != nil {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if err := c.verifyNativeRecoverySelection(ctx, record, intent); err != nil {
			return result, err
		}
		record, err = c.parkPreparedRecovery(ctx, record, parkHelper)
		if err != nil {
			return result, err
		}
		if err := retirement.Retire(ctx); err != nil {
			return result, err
		}
	}
	phase, err = AdvanceRecoveryExecution(phase, recoveryRetired)
	if err != nil {
		return result, err
	}
	if selection.Kind == RecoveryFresh {
		return c.launchRecoveryFreshAddress(ctx, record, selection, profile, intent, phase, repo, notes)
	}
	if err := c.confirmRecoveryAbsent(ctx, record); err != nil {
		return result, fmt.Errorf("source cannot yet be proved stopped; Start fresh can leave it untouched: %w", err)
	}
	if err := c.verifyNativeRecoverySelection(ctx, record, intent); err != nil {
		return result, err
	}
	record, err = c.fenceRecoveryStart(ctx, record, intent)
	if err != nil {
		return result, err
	}
	if cleared, err := c.clearLifecycleDebris(record); err != nil {
		return result, err
	} else if cleared != nil {
		record = *cleared
	}
	if err := c.repairRetiredRecoveryWorkspace(ctx, record); err != nil {
		return result, err
	}
	return c.launchPreparedSelection(ctx, record, selection, profile, intent, choice, phase, repo, notes)
}

// Native authority always names the original selected snapshot. Record revisions
// can advance while preserving and retiring it; the source revision cannot.
func (c *Couch) verifyNativeRecoverySelection(ctx context.Context, record ThreadRecord, intent threadrecord.RecoveryIntent) error {
	if intent.Kind() != threadrecord.RecoveryInferred || intent.Context.Source == nil {
		return nil
	}
	if err := threadrecord.ValidateRecoveryIntent(intent, toPersistedThreadAddress(record.Address)); err != nil {
		return err
	}
	if intent.SourceRevision > record.Revision {
		return errors.New("native recovery source revision changed")
	}
	admit := func() error {
		return threadrecord.ValidateNativeRecoverySourceAt(intent.Context.Source, c.Clock.Now(), intent.SourceRevision)
	}
	if err := admit(); err != nil {
		return err
	}
	if c.RecoveryVerifySelection == nil {
		return errors.New("native recovery selection verification unavailable; source was not admitted")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.RecoveryVerifySelection(ctx, record, intent.Context, intent.SelectedKey, intent.NativeID); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return admit()
}

func (c *Couch) launchPreparedSelection(ctx context.Context, record ThreadRecord, selection RecoverySelection, profile LaunchProfile, intent threadrecord.RecoveryIntent, choice RecoveryChoice, phase recoveryExecutionPhase, preparedRepo string, notes []string) (RecoveryResumeResult, error) {
	var result RecoveryResumeResult
	if _, err := AdvanceRecoveryExecution(phase, recoveryClaimed); err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	repo, err := c.resolveRepoIdentity(ctx, record.WorkingPath)
	if err != nil {
		return result, err
	}
	if repo != preparedRepo {
		return result, errors.New("working repository changed during recovery; refresh the choice")
	}
	owner, err := c.Proc.Current()
	if err != nil {
		return result, err
	}
	nonce, err := allocateStartNonce(c.Entropy)
	if err != nil {
		return result, err
	}
	shape := StartColdResume
	if selection.Kind == RecoveryFresh {
		shape = StartFreshExisting
	}
	if err := c.verifyNativeRecoverySelection(ctx, record, intent); err != nil {
		return result, err
	}
	record, err = c.Threads.CommitStartClaim(record.Address, record.Revision, repo, c.Clock.Now(), StartEvent{Kind: StartClaimed, Shape: shape, Nonce: nonce, Owner: SupervisorOwner{PID: owner.PID, Identity: owner.Identity}, Profile: &profile, RecoveryIntent: &intent})
	if err != nil {
		return result, err
	}
	phase, err = AdvanceRecoveryExecution(phase, recoveryClaimed)
	if err != nil {
		return result, err
	}
	return c.launchRecoveryClaim(ctx, record, nonce, selection, profile, intent, choice, phase, notes, c.Workspaces)
}

func (c *Couch) launchRecoveryClaim(ctx context.Context, record ThreadRecord, nonce string, selection RecoverySelection, profile LaunchProfile, intent threadrecord.RecoveryIntent, choice RecoveryChoice, phase recoveryExecutionPhase, notes []string, readiness WorkspaceReadiness) (result RecoveryResumeResult, retErr error) {
	if _, err := AdvanceRecoveryExecution(phase, recoveryLaunched); err != nil {
		return result, err
	}
	fail := func(err error) (RecoveryResumeResult, error) {
		return result, errors.Join(err, c.rollbackTrackedStart(record, nonce))
	}
	if err := c.prepareTrackedWorkspaceWith(ctx, record, nonce, false, readiness); err != nil {
		return fail(err)
	}
	if selection.Kind == RecoveryInferred {
		observed, err := c.RecoveryEvidence(ctx, record, RecoveryObservationRequest{ObservedAt: c.Clock.Now(), Source: intent.Context.Source})
		if err != nil {
			return fail(err)
		}
		source := record
		source.Revision = intent.SourceRevision
		preview, err := DecideSlotRecovery(source, observed, c.Clock.Now())
		if err != nil {
			return fail(err)
		}
		if _, err := selectRecoveryChoice(preview, intent.SelectedKey); err != nil {
			return fail(err)
		}
	} else if selection.Kind == RecoveryProven {
		binding, err := c.resumeEvidence(ctx, record)
		if err != nil {
			return fail(err)
		}
		if err := launcher.RequireNativeResumeBinding(intent.NativeID, binding.NativeID, binding.Status); err != nil {
			return fail(err)
		}
	}
	if selection.Kind != RecoveryFresh {
		if err := c.checkRecoveryConversationInUse(ctx, record, choice); err != nil {
			return fail(err)
		}
	}
	backend, err := c.Threads.storeForAddress(record.Address)
	if err != nil {
		return fail(err)
	}
	authority := launcher.RecoveryAuthority{Version: 1, Address: toPersistedThreadAddress(record.Address), Attempt: nonce, RecordRevision: record.Revision, RecordPath: backend.recordPath(record.Address), Intent: intent}
	raw, err := launcher.BuildCouchRecoveryLaunchProfile(string(record.Address.Tag), profile.Agent, profile.Argv, authority)
	if err != nil {
		return fail(err)
	}
	orient := intent.Orientation
	if orient == nil {
		orient = &threadrecord.RecoveryOrientation{Branch: intent.Context.Branch, Issue: intent.Context.Issue}
		if intent.Context.Source == nil {
			orient.Capture = &intent.Context.Capture
		} else {
			orient.Unavailable = "Previous screen unavailable; native candidate selected explicitly."
		}
	}
	context := orientation.RecoveryOrientationContext{SlotAddress: selection.Reference, Branch: orient.Branch, Issue: orient.Issue, NativeID: intent.NativeID}
	if orient.Capture != nil {
		context.PreviousRaw, context.PreviousEvents = orient.Capture.Raw, orient.Capture.Events
	}
	prompt, err := orientation.BuildRecoveryOrientation(context)
	if err != nil {
		return fail(err)
	}
	actor, handle, err := c.launchTrackedThread(trackedThreadLaunch{Context: ctx, Thread: record, Nonce: nonce, Args: StartArgs{Worktree: Worktree(record.StartingPath), Cwd: record.WorkingPath, Stack: profile.Agent, ExtraArgs: cloneArgv(profile.Argv)}, StartedAt: c.Clock.Now(), ProfileRaw: raw, Fresh: selection.Kind == RecoveryFresh, Resume: selection.Kind != RecoveryFresh, Prompt: prompt, Recovery: true})
	if err != nil {
		return result, err
	}
	result.Start = StartResult{Record: actor, Handle: handle}
	result.Mode = ResumeDisk
	result.Notice = fmt.Sprintf("resumed %s", intent.NativeID)
	if selection.Kind == RecoveryInferred {
		if intent.Context.Source != nil {
			result.Notice = fmt.Sprintf("resumed %s: unconfirmed native candidate, native last activity %s", intent.NativeID, choice.LastActivity.Local().Format("15:04 MST"))
		} else {
			result.Notice = fmt.Sprintf("resumed %s: last active %s, %.0f%% tail match", intent.NativeID, choice.LastActivity.Local().Format("15:04 MST"), choice.TailMatch)
		}
	}
	if selection.Kind == RecoveryFresh {
		result.Mode = Fresh
		result.Notice = "started fresh"
		if orient.Unavailable != "" {
			notes = append(notes, orient.Unavailable)
		}
	}
	if len(notes) > 0 {
		result.Notice += "; " + strings.Join(notes, "; ")
	}
	return result, nil
}

func (c *Couch) launchRecoveryFreshAddress(ctx context.Context, old ThreadRecord, selection RecoverySelection, profile LaunchProfile, intent threadrecord.RecoveryIntent, phase recoveryExecutionPhase, preparedRepo string, notes []string) (RecoveryResumeResult, error) {
	if _, err := AdvanceRecoveryExecution(phase, recoveryClaimed); err != nil {
		return RecoveryResumeResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return RecoveryResumeResult{}, err
	}
	backend, err := c.Threads.storeForAddress(old.Address)
	if err != nil {
		return RecoveryResumeResult{}, err
	}
	// Why the source may go (pair#695). The confirmed offer already stopped
	// what the operator agreed to; nothing more is stopped here. A source
	// proven down leaves by proof, with the park or claim its dead owners
	// left folded; otherwise Fresh starts beside the remnants, as chosen.
	var authority RetireAuthority
	proof, err := c.proveDownAndFold(ctx, old, "recover-fresh")
	if err == nil {
		authority = proof
		if proof.Revision() != old.Revision {
			if old, err = c.Threads.GetThread(old.Address); err != nil {
				return RecoveryResumeResult{}, err
			}
		}
	} else if authority, err = newFreshLeavesRemnants(old, err); err != nil {
		return RecoveryResumeResult{}, err
	}
	sourceAbsent := c.confirmRecoveryAbsent(ctx, old) == nil
	if sourceAbsent {
		if err := c.repairRetiredRecoveryWorkspace(ctx, old); err != nil {
			return RecoveryResumeResult{}, err
		}
	}
	var observed slotCurrentObservation
	if backend.layout.Local {
		observed, err = backend.observeSlotCurrentContext(ctx)
		if err != nil {
			return RecoveryResumeResult{}, err
		}
		if observed.Record == nil || observed.Record.Revision != old.Revision {
			return RecoveryResumeResult{}, errors.New("slot changed during fresh recovery")
		}
	}
	repo, err := c.resolveRepoIdentity(ctx, old.WorkingPath)
	if err != nil {
		return RecoveryResumeResult{}, err
	}
	if repo != preparedRepo {
		return RecoveryResumeResult{}, errors.New("working repository changed during recovery; refresh the choice")
	}
	var nextIntent threadrecord.RecoveryIntent
	prefix := filepath.Base(old.StartingPath)
	if backend.slot != nil {
		prefix = filepath.Base(backend.slot.PrimaryRoot)
	}
	next, nonce, err := c.claimFreshRecord(ctx, freshClaimInput{ScopeKey: old.Address.RepoScope, Cwd: old.WorkingPath, RepoIdentity: repo, TagPrefix: prefix, Store: backend, Profile: LaunchProfileResolution{Profile: profile, AgentSource: AgentSourcePath, ArgvSource: ArgvSourcePath}, Commit: func(next ThreadRecord) error {
		// The intent moves to the new address record, so its source revision
		// is that record's (pair#600); the old record's revision is larger and
		// the launcher refuses a source newer than the record it validates.
		nextIntent = *threadrecord.CloneRecoveryIntent(&intent)
		nextIntent.Address, nextIntent.SourceRevision = toPersistedThreadAddress(next.Address), next.Revision
		next.RecoveryIntent = threadrecord.CloneRecoveryIntent(&nextIntent)
		next.Incarnations[0].Start.RecoveryIntent = threadrecord.CloneRecoveryIntent(&nextIntent)
		if backend.layout.Local {
			return backend.replaceSlotCurrent(observed, next, authority)
		}
		return backend.ReplaceThreadExpected(old.Address, old.Revision, next, authority)
	}})
	if err != nil {
		return RecoveryResumeResult{}, err
	}
	next, err = backend.GetThread(next.Address)
	if err != nil {
		return RecoveryResumeResult{}, err
	}
	readiness := c.Workspaces
	if !sourceAbsent || c.confirmRecoveryAbsent(ctx, old) != nil {
		notes = append(notes, "previous unproved owners were left untouched; using a new terminal identity; workspace setup was not refreshed")
		if backend.layout.Local {
			if backend.slot == nil {
				return RecoveryResumeResult{}, errors.New("local recovery workspace has no slot identity")
			}
			readiness = recoveryExistingWorkspace{c: c, slot: *backend.slot}
		}
	}
	phase, err = AdvanceRecoveryExecution(phase, recoveryClaimed)
	if err != nil {
		return RecoveryResumeResult{}, err
	}
	return c.launchRecoveryClaim(ctx, next, nonce, selection, profile, nextIntent, RecoveryChoice{Agent: profile.Agent}, phase, notes, readiness)
}

// repairRetiredRecoveryWorkspace reuses normal slot convergence only after the
// explicit recovery has proved its previous owners absent. It never repairs a
// workspace underneath foreign survivors; healthy hosts retain one readiness pass.
func (c *Couch) repairRetiredRecoveryWorkspace(ctx context.Context, record ThreadRecord) error {
	if c.Slots == nil {
		return nil
	}
	backend, err := c.Threads.storeForAddressContext(ctx, record.Address)
	if err != nil {
		return err
	}
	if backend.slot == nil {
		return nil
	}
	repository, err := c.Slots.Discover(ctx, backend.slot.PrimaryRoot)
	if err != nil {
		return err
	}
	for _, candidate := range repository.Slots {
		if candidate.Identity.WorktreeRoot == backend.slot.WorktreeRoot && (!candidate.Verified || candidate.Err != nil) {
			if err := c.confirmRecoveryAbsent(ctx, record); err != nil {
				return err
			}
			_, _, err := c.selectedSlot(ctx, backend.slot.WorktreeRoot)
			return err
		}
	}
	return nil
}

// persistedRecoveryIntent is the intent as the store will decode it: times lose
// their monotonic reading and keep only what their encoding carries.
func persistedRecoveryIntent(intent threadrecord.RecoveryIntent) (threadrecord.RecoveryIntent, error) {
	raw, err := json.Marshal(intent)
	if err != nil {
		return threadrecord.RecoveryIntent{}, err
	}
	var persisted threadrecord.RecoveryIntent
	return persisted, json.Unmarshal(raw, &persisted)
}
