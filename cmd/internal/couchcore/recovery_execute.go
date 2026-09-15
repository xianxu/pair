package couchcore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

const recoveryAttempts = 8
const recoveryObservationTimeout = 5 * time.Second

type contextPairSessionObserver interface {
	PairSessionContext(context.Context, ThreadAddress) (PairSessionBinding, error)
}

func (c *Couch) recoverySession(ctx context.Context, address ThreadAddress) (PairSessionBinding, error) {
	if observer, ok := c.Artifacts.(contextPairSessionObserver); ok {
		return observer.PairSessionContext(ctx, address)
	}
	if err := ctx.Err(); err != nil {
		return PairSessionBinding{}, err
	}
	observer, ok := c.Artifacts.(PairSessionIO)
	if !ok {
		return PairSessionBinding{}, errors.New("exact Pair session observer unavailable")
	}
	binding, err := observer.PairSession(address)
	if err == nil {
		err = ctx.Err()
	}
	return binding, err
}

// observeRecovery is read-only. Session presence and detached ownership are
// independent observations: an empty detached result never proves absence.
func (c *Couch) observeRecovery(ctx context.Context, record ThreadRecord) (RecoveryEvidence, error) {
	in := RecoveryEvidence{Thread: record, Helper: Dead, Presence: PresenceUnobserved, Checkpoint: record.Continuation != nil}
	ctx, cancel := context.WithTimeout(ctx, recoveryObservationTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return in, err
	}
	if c.Proc == nil {
		return in, errors.New("recovery process observer unavailable")
	}
	if len(record.Incarnations) == 1 {
		inc := record.Incarnations[0]
		in.Helper = observeExactProcess(c.Proc, ProcessIdentity{PID: inc.PID, Identity: inc.Identity})
	}
	// Open transactions and live/unknown owners need no external session probes.
	if record.Park != nil || len(record.Incarnations) > 1 || len(record.Incarnations) == 1 && (record.Incarnations[0].Start != nil || record.Incarnations[0].State != IncarnationLive || in.Helper != Dead) {
		return in, nil
	}
	return c.observeRecoverySession(ctx, record, in)
}

// observeRecoverySession supplies session evidence after the caller's own
// lifecycle admission. Ordinary recovery retains its stricter state guard.
func (c *Couch) observeRecoverySession(ctx context.Context, record ThreadRecord, in RecoveryEvidence) (RecoveryEvidence, error) {
	ctx, cancel := context.WithTimeout(ctx, recoveryObservationTimeout)
	defer cancel()
	binding, err := c.recoverySession(ctx, record.Address)
	if err != nil {
		return in, fmt.Errorf("recovery session state could not be checked; retry: %w", err)
	}
	in.Session = binding.Name
	in.Presence = PresenceAbsent
	if binding.Present {
		in.Presence = PresencePresent
		resolver, ok := c.Artifacts.(DetachedSessionResolver)
		if !ok {
			return in, errors.New("recovery detached session observer unavailable")
		}
		agent := ""
		if record.LatestLaunchProfile != nil {
			agent = record.LatestLaunchProfile.Agent
		}
		proof, err := resolver.DetachedSessions(ctx, []DetachedCandidate{{Address: record.Address, Agent: agent}})
		if err != nil {
			return in, fmt.Errorf("recovery detached state could not be checked; retry: %w", err)
		}
		in.Detached = detachedResumeProofMatches(record, proof) && proof[0].SessionName == binding.Name
	}
	if err := ctx.Err(); err != nil {
		return in, err
	}
	return in, nil
}

func (c *Couch) reconcileRecoveryHelper(ctx context.Context, address ThreadAddress) (ThreadRecord, RecoveryEvidence, error) {
	for attempt := 0; attempt < recoveryAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return ThreadRecord{}, RecoveryEvidence{}, err
		}
		record, err := c.Threads.GetThread(address)
		if err != nil {
			return record, RecoveryEvidence{}, err
		}
		evidence, err := c.observeRecovery(ctx, record)
		if err != nil {
			return record, evidence, err
		}
		decision := DecideRecovery(evidence)
		if !decision.Archive {
			return record, evidence, errors.New(decision.Diagnosis)
		}
		if len(record.Incarnations) == 0 {
			return record, evidence, nil
		}
		inc := record.Incarnations[0]
		if err := ctx.Err(); err != nil {
			return record, evidence, err
		}
		// The observer may have called an external session service. Recheck the
		// exact process identity immediately before committing its retirement.
		if observeExactProcess(c.Proc, ProcessIdentity{PID: inc.PID, Identity: inc.Identity}) != Dead {
			return record, evidence, errors.New("recovery helper death is no longer proved")
		}
		next, err := c.Threads.RetireIncarnation(address, record.Revision, ProcessIdentity{PID: inc.PID, Identity: inc.Identity}, record.LastActiveAt)
		var conflict *ThreadRevisionError
		if errors.As(err, &conflict) {
			continue
		}
		if err != nil {
			return record, evidence, err
		}
		previousSession, previousPresence := evidence.Session, evidence.Presence
		evidence, err = c.observeRecovery(ctx, next)
		if err == nil && (evidence.Session != previousSession || evidence.Presence != previousPresence) {
			err = errors.New("recovery session changed while retiring its dead helper; inspect and retry")
		}
		if err == nil && !DecideRecovery(evidence).Archive {
			err = errors.New(DecideRecovery(evidence).Diagnosis)
		}
		return next, evidence, err
	}
	return ThreadRecord{}, RecoveryEvidence{}, errors.New("thread changed during recovery; inspect and retry")
}

// RecoverThread never falls back to an unseeded launch. A path explicitly
// selects a legacy checkpoint; without one, a surviving agent takes precedence.
func (c *Couch) RecoverThread(ctx context.Context, address ThreadAddress, path string) (ContinuationResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ContinuationResult{}, err
	}
	if c == nil || c.Threads == nil || c.Proc == nil || c.Clock == nil {
		return ContinuationResult{}, errors.New("recovery services unavailable")
	}
	if err := validateThreadAddress(address); err != nil {
		return ContinuationResult{}, err
	}
	var selected *checkpoint.Checkpoint
	if path != "" {
		cp, err := checkpoint.ReadFile(path)
		if err != nil {
			return ContinuationResult{}, err
		}
		selected = &cp
	}
	record, err := c.Threads.GetThread(address)
	if err != nil {
		return ContinuationResult{}, err
	}
	if request := record.Continuation; request != nil && request.Phase != checkpoint.Complete {
		if selected != nil && selected.Digest != request.Checkpoint.Digest {
			return ContinuationResult{}, errors.New("another continuation is unresolved; recover its retained checkpoint or archive")
		}
		// Target/start recovery has stronger exact-attempt rules than stale-source
		// retirement. Let its existing reconciler own those interrupted boundaries.
		targetAttempt := request.Target != nil || request.SourcePark != "" || request.SourceAbsence != nil
		for _, inc := range record.Incarnations {
			targetAttempt = targetAttempt || inc.Start != nil && inc.Start.Nonce == request.Attempt
		}
		if targetAttempt {
			return c.RetryContinuation(ctx, address, request.ID)
		}
	}
	evidence, err := c.observeRecovery(ctx, record)
	if err != nil {
		return ContinuationResult{}, err
	}
	decision := DecideRecovery(evidence)
	if evidence.Presence == PresencePresent && decision.Recover {
		if selected != nil {
			return ContinuationResult{}, errors.New("the source session survives; Recover reattaches its running agent before any checkpoint replacement")
		}
		record, _, err = c.reconcileRecoveryHelper(ctx, address)
		if err != nil {
			return ContinuationResult{}, err
		}
		actor, handle, err := c.ResumeContextWith(ctx, address, ResumeOptions{WarmOnly: true})
		result := ContinuationResult{Record: actor, Handle: handle, SourceReattached: record.Continuation != nil}
		if status := continuationStatus(record); status != nil {
			result.Status = *status
		}
		return result, err
	}
	if !decision.FromCheckpoint {
		return ContinuationResult{}, errors.New(decision.Diagnosis)
	}
	if selected == nil && record.Continuation == nil {
		return ContinuationResult{}, errors.New("no retained checkpoint; choose Recover from checkpoint with its absolute path, or archive")
	}
	record, err = c.prepareAbsentContinuation(ctx, address, selected)
	if err != nil {
		return ContinuationResult{}, err
	}
	return c.RetryContinuation(ctx, address, record.Continuation.ID)
}

// prepareAbsentContinuation snapshots a legacy checkpoint and retires a proved
// dead source in one CAS. Source absence is not a verified park receipt.
func (c *Couch) prepareAbsentContinuation(ctx context.Context, address ThreadAddress, selected *checkpoint.Checkpoint) (ThreadRecord, error) {
	if c.ContinuationSource == nil {
		return ThreadRecord{}, errors.New("recovery source generation reader unavailable")
	}
	for attempt := 0; attempt < recoveryAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return ThreadRecord{}, err
		}
		record, err := c.Threads.GetThread(address)
		if err != nil {
			return record, err
		}
		evidence, err := c.observeRecovery(ctx, record)
		if err != nil {
			return record, err
		}
		decision := DecideRecovery(evidence)
		if evidence.Presence != PresenceAbsent || !decision.FromCheckpoint {
			return record, errors.New(decision.Diagnosis)
		}
		observeCtx, cancel := context.WithTimeout(ctx, recoveryObservationTimeout)
		source, err := c.ContinuationSource(observeCtx, address)
		if err == nil {
			err = observeCtx.Err()
		}
		cancel()
		if err != nil {
			return record, fmt.Errorf("recovery source generation could not be checked: %w", err)
		}
		if source.Session != evidence.Session {
			return record, errors.New("recovery source generation and exact session binding disagree")
		}
		if record.LatestLaunchProfile == nil || !launcher.IsSupportedAgent(source.Agent) || record.LatestLaunchProfile.Agent != source.Agent {
			return record, errors.New("recovery source and saved launch agent do not match")
		}
		old := record.Continuation
		var request checkpoint.Request
		if old != nil && old.Phase != checkpoint.Complete {
			request = old.Clone()
			if selected != nil && selected.Digest != request.Checkpoint.Digest {
				return record, errors.New("another continuation is unresolved; recover it or archive")
			}
			if err := AdmitRecoveryGeneration(request, source); err != nil {
				return record, err
			}
			if request.Target != nil {
				return record, errors.New("continuation target must be reconciled before source recovery")
			}
			if request.SourcePark != "" || request.SourceAbsence != nil {
				return record, nil
			}
			if observeExactProcess(c.Proc, ProcessIdentity{PID: request.Source.Helper.PID, Identity: request.Source.Helper.Identity}) != Dead {
				return record, errors.New("recorded continuation source helper is not proved dead")
			}
		} else {
			cp := selected
			if cp == nil && old != nil {
				cp = &old.Checkpoint
			}
			if cp == nil {
				return record, errors.New("no checkpoint selected")
			}
			if err := cp.Validate(); err != nil {
				return record, err
			}
			if cp.Agent() != source.Agent {
				return record, errors.New("checkpoint agent does not match exact source generation")
			}
			if old != nil && source.LaunchOrdinal <= old.Source.LaunchOrdinal {
				return record, errors.New("completed continuation source generation did not advance")
			}
			request = checkpoint.Request{Version: checkpoint.Version, ID: checkpoint.RequestID(address.RepoScope, string(address.Tag), source.LaunchOrdinal, cp.Digest), Checkpoint: *cp, Source: checkpoint.Source{Agent: source.Agent, Session: source.Session, LaunchOrdinal: source.LaunchOrdinal}, CreatedAt: c.Clock.Now(), Phase: checkpoint.Pending}
			if len(record.Incarnations) == 1 {
				inc := record.Incarnations[0]
				request.Source.Helper = checkpoint.Process{PID: inc.PID, Identity: inc.Identity}
			}
		}
		absence := checkpoint.SourceAbsence{Session: request.Source.Session, LaunchOrdinal: request.Source.LaunchOrdinal, ObservedAt: c.Clock.Now(), RecordRevision: record.Revision}
		if old != nil && old.Phase != checkpoint.Complete {
			request, err = checkpoint.Advance(request, checkpoint.Event{Kind: checkpoint.SourceAbsent, RequestID: request.ID, Attempt: request.Attempt, SourceAbsence: &absence})
			if err != nil {
				return record, err
			}
		} else {
			request.SourceAbsence = &absence
		}
		if err := ctx.Err(); err != nil {
			return record, err
		}
		if len(record.Incarnations) == 1 {
			inc := record.Incarnations[0]
			if observeExactProcess(c.Proc, ProcessIdentity{PID: inc.PID, Identity: inc.Identity}) != Dead {
				return record, errors.New("recovery source helper death changed")
			}
		}
		candidate := record
		candidate.Continuation = &request
		if err := c.verifyAbsentContinuation(ctx, candidate); err != nil {
			return record, err
		}
		next, err := c.Threads.UpdateExistingThread(address, record.Revision, func(next *ThreadRecord) error {
			copy := request.Clone()
			next.Continuation = &copy
			next.Incarnations = nil
			return nil
		})
		var conflict *ThreadRevisionError
		if errors.As(err, &conflict) {
			continue
		}
		return next, err
	}
	return ThreadRecord{}, errors.New("thread changed during recovery; inspect and retry")
}

func (c *Couch) verifyAbsentContinuation(ctx context.Context, record ThreadRecord) error {
	ctx, cancel := context.WithTimeout(ctx, recoveryObservationTimeout)
	defer cancel()
	binding, err := c.recoverySession(ctx, record.Address)
	if err != nil {
		return err
	}
	if binding.Present || binding.Name != record.Continuation.Source.Session {
		return errors.New("continuation source session is no longer exactly absent")
	}
	return c.verifyContinuationGeneration(ctx, record)
}

func (c *Couch) verifyContinuationGeneration(ctx context.Context, record ThreadRecord) error {
	ctx, cancel := context.WithTimeout(ctx, recoveryObservationTimeout)
	defer cancel()
	if c.ContinuationSource == nil {
		return errors.New("continuation generation observer unavailable")
	}
	source, err := c.ContinuationSource(ctx, record.Address)
	if err != nil {
		return err
	}
	if err := AdmitRecoveryGeneration(*record.Continuation, source); err != nil {
		return err
	}
	return ctx.Err()
}

func (c *Couch) continuationTargetGeneration(ctx context.Context, record ThreadRecord) (*checkpoint.TargetGeneration, error) {
	if c.ContinuationGeneration == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, recoveryObservationTimeout)
	defer cancel()
	request := record.Continuation
	generation, err := c.ContinuationGeneration(ctx, record.Address, request.Source.Agent, request.Attempt)
	if err != nil || generation == nil {
		return generation, err
	}
	if generation.Agent != request.Source.Agent || generation.Session != request.Source.Session || generation.Attempt != request.Attempt || generation.LaunchOrdinal <= request.Source.LaunchOrdinal {
		return nil, errors.New("continuation receipt generation does not match exact request target")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return generation, nil
}
