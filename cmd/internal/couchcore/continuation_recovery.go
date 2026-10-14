package couchcore

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
)

func (c *Couch) validateContinuationWarm(ctx context.Context, record ThreadRecord) error {
	return withContinuationExits(record, c.checkContinuationWarm(ctx, record))
}

func (c *Couch) checkContinuationWarm(ctx context.Context, record ThreadRecord) error {
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
		return errors.New("warm session is neither the requested source nor its exact continuation target")
	}
	return nil
}

// ensureContinuationAttached never starts an agent. A dead helper is retired
// only after the existing detached-session observer proves unique ownership.
func (c *Couch) ensureContinuationAttached(ctx context.Context, record ThreadRecord, target bool) (ThreadRecord, ActorRecord, Handle, error) {
	// Explicit checkpoint execution already holds the thread lifecycle gate.
	// Reuse this Couch's borrowed client before considering a new attachment.
	var err error
	record, err = c.adoptOwnedAttachment(ctx, record)
	if err != nil {
		return record, ActorRecord{}, nil, err
	}
	if len(record.Incarnations) == 0 {
		for _, actor := range c.actorRegistry().Records() {
			if actor.Thread == record.Address && actor.Shape == StartWarmReattach && observeExactProcess(c.Proc, ProcessIdentity{PID: actor.PID, Identity: actor.Identity}) != Dead {
				return record, ActorRecord{}, nil, errors.New("continuation attachment has no local lifecycle owner")
			}
		}
	}
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
			if !registered || evidence.Session != continuationTargetSession(record) {
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
	actor, handle, err := c.resumeContextWith(ctx, record.Address, ResumeOptions{WarmOnly: true}, true)
	if err != nil {
		return record, actor, handle, err
	}
	updated, err := c.Threads.GetThread(record.Address)
	if err != nil {
		return record, ActorRecord{}, nil, c.AbortStarted(ctx, StartResult{Record: actor, Handle: handle}, err)
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
			generationErr = errors.Join(generationErr, c.AbortStarted(ctx, StartResult{Record: actor, Handle: handle}, generationErr))
		}
		return c.failContinuation(record, generationErr)
	}
	target := checkpoint.Process{PID: inc.PID, Identity: inc.Identity}
	updated, err := c.advanceContinuation(record.Address, checkpoint.Event{Kind: checkpoint.Registered, At: c.Clock.Now(), RequestID: record.Continuation.ID, Attempt: record.Continuation.Attempt, Target: &target, TargetGeneration: generation})
	if err != nil {
		if handle != nil {
			err = errors.Join(err, c.AbortStarted(ctx, StartResult{Record: actor, Handle: handle}, err))
		}
		return c.failContinuation(record, err)
	}
	record = updated
	return ContinuationResult{Status: *continuationStatus(record), Record: actor, Handle: handle}, nil
}
