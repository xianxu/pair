package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// confirmEditedRestartVacancy refreshes the source after this operation's own
// park/rollback. It consumes the same classifier and service guards as ordinary
// switching, with the exact current revision returned by that owner.
func (c *Couch) confirmEditedRestartVacancy(ctx context.Context, prepared PreparedAgentSwitch, current ThreadRecord, forced bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	latest, err := c.Threads.PreviewThreadContext(ctx, current.Address)
	if err != nil {
		return err
	}
	if latest.Revision != current.Revision || latest.WorkingPath != prepared.record.WorkingPath || latest.StartingPath != prepared.record.StartingPath || hasOccupiedIncarnation(latest) || latest.Park != nil {
		return ErrSwitchResolutionChanged
	}
	repo, err := c.resolveRepoIdentity(ctx, latest.StartingPath)
	if err != nil {
		return err
	}
	if repo != prepared.repoIdentity || !c.workingPathExists(latest) {
		return ErrSwitchResolutionChanged
	}
	state, reason, err := c.classifyForAction(ctx, latest.Address)
	if err != nil {
		return err
	}
	if state == ThreadLive || !SwitchableState(state, reason) {
		return errors.New("switch-agent: edited restart source is not proven vacant")
	}
	evidence := SwitchSourceEvidence{state: state, reason: reason, address: latest.Address, revision: latest.Revision, workingPath: latest.WorkingPath, repoIdentity: repo}
	if c.SwitchGuard != nil {
		if forced {
			guard, ok := c.SwitchGuard.(AgentSwitchForceGuard)
			if !ok {
				return errors.New("switch-agent: local force guard unavailable")
			}
			err = guard.ConfirmForcedAgentSwitch(ctx, latest.Address, "", evidence)
		} else {
			_, err = c.SwitchGuard.ConfirmAgentSwitch(ctx, latest.Address, "", evidence)
		}
		if err != nil {
			return err
		}
	}
	after, err := c.Threads.PreviewThreadContext(ctx, current.Address)
	if err != nil {
		return err
	}
	if after.Revision != current.Revision {
		return ErrSwitchResolutionChanged
	}
	return ctx.Err()
}

func editedRestartEvent(intent EditedRestartIntent) ResumeAttemptEvidence {
	return ResumeAttemptEvidence{OperationID: intent.OperationID, Address: intent.Address, SourceRevision: intent.SourceRevision, SourceFingerprint: intent.SourceFingerprint}
}

func (c *Couch) executeEditedRestart(ctx context.Context, prepared PreparedAgentSwitch, thread ThreadRecord, accepted SwitchAgentRequest, owner SupervisorOwner, nonce string, result SwitchAgentResult) (SwitchAgentResult, error) {
	result.Outcome = SwitchStartFailed
	intent := cloneEditedRestartIntent(accepted.Restart)
	if intent == nil {
		return result, errors.New("switch-agent: accepted native restart intent unavailable")
	}
	intent.OperationID = nonce
	if err := c.confirmEditedRestartVacancy(ctx, prepared, thread, accepted.Force != nil); err != nil {
		return result, err
	}
	native, err := c.resolveEditedRestartEvidence(ctx, thread, &intent.ExpectedNativeID)
	if err != nil {
		return result, err
	}
	event := editedRestartEvent(*intent)
	event.Kind, event.NativeKind, event.NativeID = EditedRestartAdmission, native.Kind, native.NativeID
	event.Admission, event.Occupancy, event.RecordRevision, event.NextNonce = EditedRestartSourceAdmitted, EditedRestartVacant, thread.Revision, nonce
	state, launch, err := AdvanceEditedRestart(*intent, EditedRestartState{}, event)
	if err != nil {
		return result, err
	}
	for launch != nil {
		startedAt := c.Clock.Now()
		var transcript *threadrecord.TranscriptBinding
		if launch.Shape == StartColdResume {
			transcript = threadrecord.CloneTranscriptBinding(native.Transcript)
			if transcript != nil {
				transcript.BoundAt = startedAt
			}
		}
		claimed, err := c.Threads.CommitStartClaim(thread.Address, launch.ExpectedRevision, prepared.repoIdentity, startedAt, StartEvent{Kind: StartClaimed, Shape: launch.Shape, Nonce: launch.Nonce, Owner: owner, Profile: &launch.Profile, Transcript: transcript, RememberConfiguration: true})
		if err != nil {
			return result, err
		}
		claimEvent := editedRestartEvent(*intent)
		claimEvent.Kind, claimEvent.AttemptNonce, claimEvent.ClaimRevision, claimEvent.RecordRevision = EditedRestartClaimRecorded, launch.Nonce, claimed.Revision, claimed.Revision
		state, _, err = AdvanceEditedRestart(*intent, state, claimEvent)
		if err != nil {
			return result, errors.Join(err, c.rollbackTrackedStart(claimed, launch.Nonce))
		}
		receipt := &startRollbackReceipt{}
		var record ActorRecord
		var handle Handle
		launchErr := c.prepareTrackedWorkspace(ctx, claimed, launch.Nonce, false)
		profileRaw := ""
		if launchErr == nil {
			native, launchErr = c.resolveEditedRestartEvidence(ctx, claimed, &intent.ExpectedNativeID)
			if launchErr == nil {
				if launch.Shape == StartColdResume {
					if native.Kind != EditedRestartNativeFound {
						launchErr = refuseResume(ResumeTranscriptMissing, "accepted native conversation disappeared before edited resume")
					} else if launchErr = c.checkRecoveryConversationInUse(ctx, claimed, RecoveryChoice{Agent: launch.Profile.Agent, NativeID: launch.RequiredNativeID}); launchErr == nil {
						profileRaw, launchErr = launcher.BuildCouchResumeLaunchProfile(string(thread.Address.Tag), launch.Profile.Agent, launch.Profile.Argv, launch.RequiredNativeID)
					}
				} else if native.Kind != EditedRestartNativeMissing {
					launchErr = ErrSwitchResolutionChanged
				} else {
					profileRaw, launchErr = launcher.BuildCouchFreshLaunchProfile(string(thread.Address.Tag), launch.Profile.Agent, launch.Profile.Argv, string(AgentSourceExplicit), string(ArgvSourceExplicit))
				}
			}
		}
		if launchErr != nil {
			launchErr = errors.Join(launchErr, c.rollbackTrackedStartObserved(claimed, launch.Nonce, receipt))
			// An authorization or binding refusal observed before launching is
			// terminal. A later absence cannot erase a replacement or IO error.
			// Only positive absence at this boundary can request clean fallback.
			if launch.Shape != StartColdResume || native.Kind != EditedRestartNativeMissing || ResumeDiagnosticOf(launchErr) != ResumeTranscriptMissing {
				return result, launchErr
			}
		} else {
			record, handle, launchErr = c.launchTrackedThread(trackedThreadLaunch{Context: ctx, Thread: claimed, Nonce: launch.Nonce, Args: StartArgs{Worktree: Worktree(thread.StartingPath), Cwd: thread.WorkingPath, Stack: launch.Profile.Agent, ExtraArgs: cloneArgv(launch.Profile.Argv)}, StartedAt: startedAt, ProfileRaw: profileRaw, Resume: launch.Shape == StartColdResume, Fresh: launch.Shape == StartFreshExisting, Rollback: receipt})
		}
		if launchErr == nil {
			result.Outcome, result.Record, result.Handle = SwitchStarted, record, handle
			result.Notice = strings.Join(nonemptyRestartNotices(result.Notice, launch.Notice), "; ")
			return result, nil
		}
		if launch.Shape != StartColdResume || ctx.Err() != nil {
			return result, launchErr
		}
		if receipt.Nonce != launch.Nonce || receipt.Record.Revision == 0 {
			return result, launchErr
		}
		if handle != nil {
			select {
			case <-handle.Exited():
				_ = handle.Wait()
			default:
				return result, launchErr
			}
			if handle.Alive() {
				return result, launchErr
			}
		}
		thread = receipt.Record
		native, err = c.resolveEditedRestartEvidence(ctx, thread, &intent.ExpectedNativeID)
		if err != nil || native.Kind != EditedRestartNativeMissing {
			return result, errors.Join(launchErr, err)
		}
		if err := c.confirmEditedRestartVacancy(ctx, prepared, thread, accepted.Force != nil); err != nil {
			return result, errors.Join(launchErr, err)
		}
		nextNonce, err := allocateStartNonce(c.Entropy)
		if err != nil {
			return result, errors.Join(launchErr, err)
		}
		failed := editedRestartEvent(*intent)
		failed.Kind, failed.NativeKind, failed.NativeID = EditedRestartAttemptFailed, native.Kind, native.NativeID
		failed.Admission, failed.Occupancy, failed.Cleanup = EditedRestartSourceAdmitted, EditedRestartVacant, EditedRestartCleanupConfirmed
		failed.AttemptNonce, failed.ClaimRevision, failed.RecordRevision, failed.CleanupRevision, failed.NextNonce = launch.Nonce, state.ClaimRevision, thread.Revision, receipt.Record.Revision, nextNonce
		state, launch, err = AdvanceEditedRestart(*intent, state, failed)
		if err != nil {
			return result, errors.Join(launchErr, err)
		}
		if launch == nil {
			return result, fmt.Errorf("switch-agent: edited resume cleanup remains unconfirmed: %w", launchErr)
		}
	}
	return result, errors.New("switch-agent: edited restart produced no launch")
}

func nonemptyRestartNotices(values ...string) []string {
	var out []string
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
