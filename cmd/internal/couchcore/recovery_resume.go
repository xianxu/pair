package couchcore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// RecoveryCapturePreserver copies a validated original recording through the
// existing capture retention owner. It must reject any changed input identity;
// it returns the archived identities, never a manufactured park receipt.
type RecoveryCapturePreserver func(context.Context, ThreadRecord, RecoveryObservation) (RecoveryObservation, error)

// RecoveryTerminalPreparation captures exact, positively attributed server
// generations. The returned action may retire only that unchanged set.
type RecoveryTerminalPreparation func(context.Context, ThreadRecord) (func(context.Context) error, error)

func selectRecoveryChoice(p RecoveryPreview, key string) (RecoveryChoice, error) {
	if p.Incomplete || p.Snapshot == nil {
		return RecoveryChoice{}, errors.New("recovery evidence incomplete; refresh resume candidates")
	}
	_, err := decodeRecoveryCandidateKey(key)
	if err != nil {
		return RecoveryChoice{}, err
	}
	for _, choice := range append(append([]RecoveryChoice(nil), p.Candidates...), func() []RecoveryChoice {
		if p.Fresh != nil {
			return []RecoveryChoice{*p.Fresh}
		}
		return nil
	}()...) {
		if choice.Key != key {
			continue
		}
		expected := RecoveryCandidateSelector{Version: 1, Address: p.Address, Agent: choice.Agent, NativeID: choice.NativeID, Digest: p.Snapshot.Digest}
		if _, err := ValidateRecoveryCandidateKey(key, expected); err != nil {
			return RecoveryChoice{}, err
		}
		return choice, nil
	}
	return RecoveryChoice{}, errors.New("recovery choice changed; refresh resume candidates")
}

// ResumeCandidate never guesses a choice. The two Couch observations bracket
// the durable start claim; Pair validates its persisted intent once more before
// composing native resume argv. Ordinary resume keeps its binding contract.
func (c *Couch) ResumeCandidate(ctx context.Context, t ResumeTarget, key string) (result RecoveryResumeResult, retErr error) {
	begun := time.Now()
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := decodeRecoveryCandidateKey(key); err != nil {
		return result, err
	}
	_, record, err := c.recoveryTarget(ctx, t)
	if err != nil {
		return result, err
	}
	ctx, release, err := c.hold(ctx, record.Address, "resume-candidate")
	if err != nil {
		return result, err
	}
	defer release()
	record, err = c.Threads.GetThread(record.Address)
	if err != nil {
		return result, err
	}
	if c.RecoveryEvidence == nil || c.RecoveryPreserve == nil {
		return result, errors.New("recovery evidence preservation unavailable")
	}
	if err := continuationGuard(record); err != nil {
		return result, err
	}
	if record.Park != nil {
		return result, errors.New("recovery has an unfinished park; recover the park first")
	}
	observed, err := c.RecoveryEvidence(ctx, record)
	if err != nil {
		return result, err
	}
	preview, err := DecideSlotRecovery(record, observed, c.Clock.Now())
	if err != nil {
		return result, err
	}
	choice, err := selectRecoveryChoice(preview, key)
	if err != nil {
		return result, err
	}
	if record.LatestLaunchProfile == nil || record.LatestLaunchProfile.Agent != choice.Agent || !c.workingPathExists(record) {
		return result, errors.New("recovery requires the saved agent configuration and working directory")
	}
	profile := cloneLaunchProfile(*record.LatestLaunchProfile)
	if err := launcher.ValidateFreshAgentArgs(profile.Agent, profile.Argv); err != nil {
		return result, err
	}
	if c.SwitchLaunchCheck != nil {
		if err := c.SwitchLaunchCheck(profile.Agent); err != nil {
			return result, err
		}
	}
	repoIdentity, err := c.resolveRepoIdentity(ctx, record.WorkingPath)
	if err != nil {
		return result, err
	}
	slotAddress, err := c.recoverySlotAddress(ctx, record)
	if err != nil {
		return result, err
	}
	owner, err := c.Proc.Current()
	if err != nil {
		return result, err
	}
	nonce, err := allocateStartNonce(c.Entropy)
	if err != nil {
		return result, err
	}
	retire, err := c.prepareRecoveryRetirement(ctx, record)
	if err != nil {
		return result, err
	}
	if err := c.checkRecoveryNativeOwner(ctx, record, choice, true); err != nil {
		return result, err
	}
	finishRetention, err := c.beginResumeRetention(ctx, record.Address, true)
	if err != nil {
		return result, err
	}
	defer func() { retErr = errors.Join(retErr, finishRetention(retErr == nil)) }()
	preserved, err := c.RecoveryPreserve(ctx, record, observed)
	if err != nil {
		return result, err
	}
	// Archival changes path/inode, not the explicitly chosen native conversation.
	// The preserver proves the copy and the new keys name the immutable archive.
	preview, err = DecideSlotRecovery(record, preserved, c.Clock.Now())
	if err != nil {
		return result, err
	}
	var selected *RecoveryChoice
	if choice.NativeID == "" {
		selected = preview.Fresh
	} else {
		for i := range preview.Candidates {
			candidate := &preview.Candidates[i]
			if candidate.Agent == choice.Agent && candidate.NativeID == choice.NativeID {
				selected = candidate
				break
			}
		}
	}
	if selected == nil {
		return result, errors.New("selected conversation disappeared during screen preservation")
	}
	intent := threadrecord.RecoveryIntent{Context: *preview.Snapshot, SelectedKey: selected.Key, NativeID: selected.NativeID}
	sourceRevision := record.Revision
	if retire != nil {
		// Persist the original screen and chosen target before quitting a live
		// source. StartClaim will own the same intent after park completes.
		retained := *threadrecord.CloneRecoveryContext(&intent.Context)
		retained.SelectedKey, retained.SelectedNativeID = intent.SelectedKey, intent.NativeID
		record, err = c.Threads.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error {
			next.RecoveryContext = threadrecord.CloneRecoveryContext(&retained)
			return nil
		})
		if err != nil {
			return result, err
		}
		record, err = retire(ctx, record)
		if err != nil {
			return result, err
		}
	}
	if err := c.confirmRecoveryAbsent(ctx, record); err != nil {
		return result, err
	}
	if cleared, err := c.clearLifecycleDebris(record); err != nil {
		return result, err
	} else if cleared != nil {
		record = *cleared
	}
	startedAt := c.Clock.Now()
	shape := StartColdResume
	if selected.NativeID == "" {
		shape = StartFreshExisting
	}
	record, err = c.Threads.CommitStartClaim(record.Address, record.Revision, repoIdentity, startedAt, StartEvent{Kind: StartClaimed, Shape: shape, Nonce: nonce, Owner: SupervisorOwner{PID: owner.PID, Identity: owner.Identity}, Profile: &profile, RecoveryIntent: &intent})
	if err != nil {
		return result, err
	}
	fail := func(err error) (RecoveryResumeResult, error) {
		return result, errors.Join(err, c.rollbackTrackedStart(record, nonce))
	}
	if err := c.prepareTrackedWorkspace(ctx, record, nonce, false); err != nil {
		return fail(err)
	}
	// Read through the claim's retained source, not a later current screen.
	reobserved, err := c.RecoveryEvidence(ctx, record)
	if err != nil {
		return fail(err)
	}
	sourceRecord := record
	sourceRecord.Revision = sourceRevision // the claim is our own checked transition
	finalPreview, err := DecideSlotRecovery(sourceRecord, reobserved, c.Clock.Now())
	if err != nil {
		return fail(err)
	}
	if _, err := selectRecoveryChoice(finalPreview, selected.Key); err != nil {
		return fail(err)
	}
	if err := c.confirmRecoveryAbsent(ctx, record); err != nil {
		return fail(err)
	}
	if err := c.checkRecoveryNativeOwner(ctx, record, *selected, false); err != nil {
		return fail(err)
	}
	backend, err := c.Threads.storeForAddress(record.Address)
	if err != nil {
		return fail(err)
	}
	authority := launcher.RecoveryAuthority{Version: 1, Address: intent.Context.Address, Attempt: nonce, RecordRevision: record.Revision, RecordPath: backend.recordPath(record.Address), Intent: intent}
	raw, err := launcher.BuildCouchRecoveryLaunchProfile(string(record.Address.Tag), profile.Agent, profile.Argv, authority)
	if err != nil {
		return fail(err)
	}
	body, err := orientation.BuildRecoveryOrientation(orientation.RecoveryOrientationContext{SlotAddress: slotAddress, Branch: intent.Context.Branch, Issue: intent.Context.Issue, PreviousRaw: intent.Context.Capture.Raw, PreviousEvents: intent.Context.Capture.Events, NativeID: selected.NativeID})
	if err != nil {
		return fail(err)
	}
	request := orientation.Request{SchemaVersion: 1, Tag: string(record.Address.Tag), Agent: profile.Agent, Attempt: nonce, Body: body}
	actor, handle, err := c.launchTrackedThread(trackedThreadLaunch{Context: ctx, Thread: record, Nonce: nonce, Args: StartArgs{Worktree: Worktree(record.StartingPath), Cwd: record.WorkingPath, Stack: profile.Agent, ExtraArgs: cloneArgv(profile.Argv)}, StartedAt: startedAt, ProfileRaw: raw, Fresh: selected.NativeID == "", Resume: selected.NativeID != "", Orientation: &request, Begun: begun})
	if err != nil {
		return result, fmt.Errorf("recovery launch failed; the selected evidence is retained for retry: %w", err)
	}
	result.Start = StartResult{Record: actor, Handle: handle}
	result.Notice = fmt.Sprintf("resumed %s: last active %s, %.0f%% tail match", selected.NativeID, selected.LastActivity.Local().Format("15:04 MST"), selected.TailMatch)
	if selected.NativeID == "" {
		result.Notice = "started fresh from the previous screen"
	}
	result.Orientation, err = c.ReadOrientationStatus(ctx, record.Address, profile.Agent, nonce)
	if err != nil {
		result.Notice += "; previous-screen orientation status unavailable: " + err.Error()
	} else if result.Orientation.Phase != orientation.DeliverySubmitted {
		phase := string(result.Orientation.Phase)
		if phase == "" {
			phase = "pending"
		}
		result.Notice += "; previous-screen orientation " + phase
		if result.Orientation.Reason != "" {
			result.Notice += ": " + result.Orientation.Reason
		}
	}
	return result, nil
}

func (c *Couch) recoverySlotAddress(ctx context.Context, record ThreadRecord) (string, error) {
	backend, err := c.Threads.storeForAddress(record.Address)
	if err != nil {
		return "", err
	}
	if backend.slot != nil {
		return fmt.Sprintf("%s:%d", backend.slot.Repo, backend.slot.Number), nil
	}
	root, err := c.Git.RunContext(ctx, record.StartingPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(root) || !workspaceRepoName(filepath.Base(root)) {
		return "", errors.New("recovery cannot identify the primary workspace")
	}
	return filepath.Base(root) + ":0", nil
}

func (c *Couch) prepareRecoveryRetirement(ctx context.Context, record ThreadRecord) (func(context.Context, ThreadRecord) (ThreadRecord, error), error) {
	var live *ThreadIncarnation
	for i := range record.Incarnations {
		inc := &record.Incarnations[i]
		state := observeExactProcess(c.Proc, ProcessIdentity{PID: inc.PID, Identity: inc.Identity})
		if state == Unknown {
			return nil, errors.New("recovery source ownership is unknown; inspect its process before relaunch")
		}
		if state == Live {
			if live != nil || inc.Start != nil || inc.State != IncarnationLive {
				return nil, errors.New("recovery source is busy with another launch")
			}
			live = inc
		}
	}
	if live != nil {
		owned := false
		for _, actor := range c.actorRegistry().Records() {
			if actor.Thread == record.Address && actor.PID == live.PID && actor.Identity == live.Identity {
				owned = true
				break
			}
		}
		if !owned || c.LiveRestart == nil {
			return nil, errors.New("recovery source is not a verified hosted actor; attach it before choosing another conversation")
		}
		if err := c.LiveRestart.ConfirmNotBusy(ctx, record.Address, false); err != nil {
			return nil, err
		}
		if c.PairLifecycle == nil {
			return nil, errors.New("recovery park controller unavailable")
		}
		pid, identity := live.PID, live.Identity
		return func(ctx context.Context, current ThreadRecord) (ThreadRecord, error) {
			if observeExactProcess(c.Proc, ProcessIdentity{PID: pid, Identity: identity}) != Live {
				return current, errors.New("recovery source changed before park; refresh candidates")
			}
			if err := c.LiveRestart.ConfirmNotBusy(ctx, current.Address, false); err != nil {
				return current, err
			}
			parked, err := c.PairLifecycle.ParkExpected(ctx, current.Address, current.Revision)
			if err != nil {
				return current, fmt.Errorf("recovery park did not complete; recover the park before resuming: %w", err)
			}
			return parked.Thread, parked.CleanupError
		}, nil
	}
	if err := c.confirmRecoveryAbsent(ctx, record); err == nil {
		return nil, nil
	} else if c.RecoveryTerminal == nil {
		return nil, err
	}
	retire, err := c.RecoveryTerminal(ctx, record)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, current ThreadRecord) (ThreadRecord, error) {
		if err := retire(ctx); err != nil {
			return current, err
		}
		return current, nil
	}, nil
}

func (c *Couch) confirmRecoveryAbsent(ctx context.Context, record ThreadRecord) error {
	for _, inc := range record.Incarnations {
		if inc.Start != nil && inc.PID == 0 && inc.Identity == "" {
			continue
		}
		if observeExactProcess(c.Proc, ProcessIdentity{PID: inc.PID, Identity: inc.Identity}) != Dead {
			return errors.New("recovery source still runs or its ownership is uncertain; settle and park it before choosing a conversation")
		}
	}
	observation, err := c.observeRecoverySession(ctx, record, RecoveryEvidence{})
	if err != nil {
		return err
	}
	if observation.Presence != PresenceAbsent {
		return errors.New("recovery terminal still exists; retire its verified owner before relaunch")
	}
	return nil
}
