package couchcore

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"time"
)

func (c *Couch) validateContinuationWarm(ctx context.Context, record ThreadRecord) error {
	r := record.Continuation
	if r == nil || r.Phase == checkpoint.Complete {
		return nil
	}
	if c.FreshRegistration != nil && r.Attempt != "" {
		registered, err := c.FreshRegistration(ctx, record.Address, r.Source.Agent, r.Attempt)
		if err != nil {
			return err
		}
		if registered {
			return nil
		}
	}
	if c.ContinuationSource == nil {
		return errors.New("continuation source proof unavailable")
	}
	current, err := c.ContinuationSource(ctx, record.Address)
	if err != nil {
		return err
	}
	if !sameContinuationSource(current, ContinuationSource{Agent: r.Source.Agent, Session: r.Source.Session, LaunchOrdinal: r.Source.LaunchOrdinal}) {
		return withContinuationExits(record, errors.New("warm session is neither the requested source nor its exact continuation target"))
	}
	return nil
}

// ensureContinuationAttached never starts an agent. A dead helper is retired
// only after the existing detached-session observer proves unique ownership.
func (c *Couch) ensureContinuationAttached(ctx context.Context, record ThreadRecord, target bool) (ThreadRecord, ActorRecord, Handle, error) {
	if len(record.Incarnations) == 1 {
		inc := record.Incarnations[0]
		live := observeExactProcess(c.Proc, ProcessIdentity{PID: inc.PID, Identity: inc.Identity})
		if live == Live {
			if !c.ownsContinuationHelper(record.Address, inc) {
				return record, ActorRecord{}, nil, errors.New("continuation helper belongs to another live owner")
			}
			return record, ActorRecord{}, nil, nil
		}
		if live != Dead {
			return record, ActorRecord{}, nil, errors.New("continuation helper ownership is unresolved")
		}
		if inc.Start != nil {
			if !target || inc.Start.Nonce != record.Continuation.Attempt {
				return record, ActorRecord{}, nil, errors.New("continuation helper start is unresolved")
			}
			// The caller proved the exact fresh ready receipt. Promote through
			// the existing transaction before retiring its dead helper.
			advanced, err := c.Threads.AdvanceStart(record.Address, record.Revision, StartEvent{Kind: StartRegistered, Nonce: inc.Start.Nonce})
			if err != nil {
				return record, ActorRecord{}, nil, err
			}
			record = advanced
			inc = record.Incarnations[0]
		}
		settledUnknown := target && inc.State == IncarnationUnknown && inc.Start == nil
		var evidence RecoveryEvidence
		var err error
		if settledUnknown {
			evidence, err = c.observeRecoverySession(ctx, record, RecoveryEvidence{Thread: record, Helper: Dead, Checkpoint: true})
		} else {
			evidence, err = c.observeRecovery(ctx, record)
		}
		if err != nil {
			return record, ActorRecord{}, nil, err
		}
		if evidence.Presence != PresencePresent || !evidence.Detached {
			return record, ActorRecord{}, nil, errors.New("continuation session has no unique detached ownership proof")
		}
		if !target && evidence.Session != record.Continuation.Source.Session {
			return record, ActorRecord{}, nil, errors.New("detached continuation source session changed")
		}
		if settledUnknown {
			r := record.Continuation
			if c.FreshRegistration == nil {
				return record, ActorRecord{}, nil, errors.New("continuation target observer unavailable")
			}
			registered, err := c.FreshRegistration(ctx, record.Address, r.Source.Agent, r.Attempt)
			if err != nil {
				return record, ActorRecord{}, nil, err
			}
			if !registered || evidence.Session != r.Source.Session {
				return record, ActorRecord{}, nil, errors.New("exact continuation target receipt is no longer proved")
			}
			if err := ctx.Err(); err != nil {
				return record, ActorRecord{}, nil, err
			}
			identity := ProcessIdentity{PID: inc.PID, Identity: inc.Identity}
			if observeExactProcess(c.Proc, identity) != Dead {
				return record, ActorRecord{}, nil, errors.New("continuation helper death is no longer proved")
			}
			updated, err := c.Threads.ReconcileRegisteredTarget(record.Address, record.Revision, RegisteredTargetProof{RequestID: r.ID, Agent: r.Source.Agent, Session: evidence.Session, Attempt: r.Attempt, Helper: identity})
			if err != nil {
				return record, ActorRecord{}, nil, err
			}
			record = updated
		} else {
			updated, _, err := c.reconcileRecoveryHelper(ctx, record.Address)
			if err != nil {
				return record, ActorRecord{}, nil, err
			}
			record = updated
		}
	}
	if len(record.Incarnations) != 0 {
		return record, ActorRecord{}, nil, errors.New("continuation has ambiguous incarnations")
	}
	if target && record.VerifiedPark != nil {
		evidence, err := c.observeRecovery(ctx, record)
		if err != nil {
			return record, ActorRecord{}, nil, err
		}
		if evidence.Presence != PresencePresent || !evidence.Detached {
			return record, ActorRecord{}, nil, errors.New("continuation target is not uniquely detached")
		}
		updated, err := c.Threads.ClearVerifiedPark(record.Address, record.Revision)
		if err != nil {
			return record, ActorRecord{}, nil, err
		}
		record = updated
	}
	actor, handle, err := c.ResumeContextWith(ctx, record.Address, ResumeOptions{WarmOnly: true})
	if err != nil {
		return record, actor, handle, err
	}
	updated, err := c.Threads.GetThread(record.Address)
	if err != nil {
		return record, ActorRecord{}, nil, c.AbortStarted(StartResult{Record: actor, Handle: handle}, err)
	}
	return updated, actor, handle, nil
}
func (c *Couch) observeContinuationTarget(ctx context.Context, record ThreadRecord, registered bool) (ContinuationResult, error) {
	if !registered {
		return c.failContinuation(record, errors.New("continuation target registration is unavailable; inspect it before retrying"))
	}
	next, actor, handle, err := c.ensureContinuationAttached(ctx, record, true)
	if err != nil {
		return c.failContinuation(record, err)
	}
	record = next
	if len(record.Incarnations) != 1 {
		return c.failContinuation(record, errors.New("registered continuation has no unique target"))
	}
	inc := record.Incarnations[0]
	generation, generationErr := c.continuationTargetGeneration(ctx, record)
	if generationErr != nil {
		if handle != nil {
			generationErr = errors.Join(generationErr, c.AbortStarted(StartResult{Record: actor, Handle: handle}, generationErr))
		}
		return c.failContinuation(record, generationErr)
	}
	target := checkpoint.Process{PID: inc.PID, Identity: inc.Identity}
	updated, err := c.advanceContinuation(record.Address, checkpoint.Event{Kind: checkpoint.Registered, At: c.Clock.Now(), RequestID: record.Continuation.ID, Attempt: record.Continuation.Attempt, Target: &target, TargetGeneration: generation})
	if err != nil {
		if handle != nil {
			err = errors.Join(err, c.AbortStarted(StartResult{Record: actor, Handle: handle}, err))
		}
		return c.failContinuation(record, err)
	}
	record = updated
	if handle != nil {
		return ContinuationResult{Status: *continuationStatus(record), Record: actor, Handle: handle}, nil
	}
	status, err := c.ReconcileContinuation(ctx, record.Address, record.Continuation.ID, record.Continuation.Attempt)
	return ContinuationResult{Status: status, Record: actor, Handle: handle}, err
}
func (c *Couch) ReconcileContinuation(ctx context.Context, address ThreadAddress, id, attempt string) (ContinuationStatus, error) {
	record, err := c.requestRecord(address, id)
	if err != nil {
		return ContinuationStatus{}, err
	}
	r := record.Continuation
	if attempt != "" && r.Attempt != attempt {
		return *continuationStatus(record), nil
	}
	if r.Phase != checkpoint.Running {
		return *continuationStatus(record), nil
	}
	state, err := c.ReadOrientationStatus(ctx, address, r.Source.Agent, r.Attempt)
	if err != nil {
		result, failErr := c.failContinuation(record, err)
		return result.Status, failErr
	}
	if !state.Terminal() {
		if r.Target != nil && !c.Clock.Now().Before(r.Target.ObservedAt.Add(30*time.Second)) {
			result, err := c.failContinuation(record, errors.New("continuation submission was not confirmed within 30s; inspect the existing target before retrying"))
			return result.Status, err
		}
		return *continuationStatus(record), nil
	}
	if state.Phase != orientation.DeliverySubmitted {
		reason := fmt.Sprintf("continuation delivery %s: %s; inspect the existing target before retrying", state.Phase, state.Reason)
		if state.BodyMayBePresent() {
			reason += "; text may already be present"
		}
		result, err := c.failContinuation(record, errors.New(reason))
		return result.Status, err
	}
	if r.Target == nil {
		if len(record.Incarnations) != 1 {
			return *continuationStatus(record), errors.New("submitted continuation has no unique target")
		}
		inc := record.Incarnations[0]
		target := checkpoint.Process{PID: inc.PID, Identity: inc.Identity}
		record, err = c.advanceContinuation(address, checkpoint.Event{Kind: checkpoint.Registered, At: c.Clock.Now(), RequestID: r.ID, Attempt: r.Attempt, Target: &target})
		if err != nil {
			return ContinuationStatus{}, err
		}
	}
	next, err := c.advanceContinuation(address, checkpoint.Event{Kind: checkpoint.Submitted, RequestID: r.ID, Attempt: r.Attempt})
	if err != nil {
		return *continuationStatus(record), err
	}
	return *continuationStatus(next), nil
}
func (c *Couch) RetryContinuation(ctx context.Context, address ThreadAddress, id string) (ContinuationResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ContinuationResult{}, err
	}
	record, err := c.requestRecord(address, id)
	if err != nil {
		return ContinuationResult{}, err
	}
	r := record.Continuation
	if r.Phase != checkpoint.Failed {
		return c.Continue(ctx, address, r.ID)
	}
	if c.FreshRegistration == nil {
		return ContinuationResult{}, errors.New("continuation target observer unavailable")
	}
	registered, err := c.FreshRegistration(ctx, address, r.Source.Agent, r.Attempt)
	if err != nil {
		return ContinuationResult{Status: *continuationStatus(record)}, err
	}
	if registered {
		record, err = c.advanceContinuation(address, checkpoint.Event{Kind: checkpoint.RetryObserve, RequestID: r.ID, Attempt: r.Attempt})
		if err != nil {
			return ContinuationResult{}, err
		}
		return c.observeContinuationTarget(ctx, record, true)
	}
	if record.Park != nil {
		recovered, err := c.PairLifecycle.Recover(ctx, address)
		if err != nil {
			return ContinuationResult{Status: *continuationStatus(record)}, err
		}
		record = recovered.Thread
	}
	// Existing source is allowed; a target or matching start claim must first be
	// conclusively absent. Unknown process/session evidence never permits spawn.
	generation, generationErr := c.continuationTargetGeneration(ctx, record)
	if generationErr != nil {
		return ContinuationResult{Status: *continuationStatus(record)}, generationErr
	}
	targetAttempt := r.Target != nil || continuationSourceParked(record) || r.SourceAbsence != nil || generation != nil
	for _, inc := range record.Incarnations {
		if inc.Start != nil && inc.Start.Nonce == r.Attempt {
			targetAttempt = true
		}
	}
	if targetAttempt {
		if len(record.Incarnations) > 1 {
			return ContinuationResult{Status: *continuationStatus(record)}, errors.New("continuation has multiple recorded helpers; resolve ownership before retry")
		}
		admission := record
		request := record.Continuation.Clone()
		if generation != nil {
			request.Target = &checkpoint.Target{Generation: generation}
		}
		admission.Continuation = &request
		if err := c.verifyContinuationGeneration(ctx, admission); err != nil {
			return ContinuationResult{Status: *continuationStatus(record)}, err
		}
		presence, err := c.observeSessionPresenceContext(ctx, address)
		if err != nil || presence != PresenceAbsent {
			if err == nil {
				err = errors.New("previous continuation target may still exist; inspect or copy the checkpoint")
			}
			return ContinuationResult{Status: *continuationStatus(record)}, err
		}
		for _, inc := range record.Incarnations {
			if inc.Start != nil && inc.PID == 0 && observeExactProcess(c.Proc, ProcessIdentity{PID: inc.Start.OwnerPID, Identity: inc.Start.OwnerIdentity}) != Dead {
				return ContinuationResult{Status: *continuationStatus(record)}, errors.New("previous continuation owner may still record its blocked helper; retry after ownership is resolved")
			}
			if observeExactProcess(c.Proc, ProcessIdentity{PID: inc.PID, Identity: inc.Identity}) != Dead {
				return ContinuationResult{Status: *continuationStatus(record)}, errors.New("previous continuation helper has not been proved dead")
			}
		}
		if len(record.Incarnations) > 0 {
			// Existing cleanup owns rollback/promotion and refuses unreadable evidence.
			if record.Incarnations[0].Start != nil {
				if err := c.rollbackTrackedStart(record, r.Attempt); err != nil {
					return ContinuationResult{}, err
				}
			} else {
				record, err = c.Threads.RetireProvedDeadIncarnations(address, record.Revision)
				if err != nil {
					return ContinuationResult{}, err
				}
			}
			record, err = c.requestRecord(address, r.ID)
			if err != nil {
				return ContinuationResult{}, err
			}
		}
	}
	attempt, err := allocateStartNonce(c.Entropy)
	if err != nil {
		return ContinuationResult{}, err
	}
	record, err = c.Threads.AdvanceContinuation(address, record.Revision, checkpoint.Event{Kind: checkpoint.RetryAbsent, RequestID: r.ID, Attempt: attempt, TargetGeneration: generation})
	if err != nil {
		return ContinuationResult{}, err
	}
	return c.executeContinuation(ctx, record)
}
