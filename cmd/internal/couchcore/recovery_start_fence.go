package couchcore

import (
	"context"
	"errors"
	"reflect"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// fenceRecoveryStart revokes only the exact expired attempt after explicit
// recovery preserved its selection and retired its terminal. Age never grants
// permission to signal the Couch supervisor that owns the old claim.
func (c *Couch) fenceRecoveryStart(ctx context.Context, record ThreadRecord, intent threadrecord.RecoveryIntent) (ThreadRecord, error) {
	if !startClaimed(record) {
		return record, nil
	}
	if ctx == nil {
		return ThreadRecord{}, errors.New("recovery start fence requires context")
	}
	if err := ctx.Err(); err != nil {
		return ThreadRecord{}, err
	}
	if c == nil || c.Threads == nil || c.Proc == nil || c.Clock == nil {
		return ThreadRecord{}, errors.New("recovery start fence observers unavailable")
	}
	if len(record.Incarnations) != 1 || record.Incarnations[0].Start == nil {
		return ThreadRecord{}, errors.New("recovery start fence requires one exact start")
	}
	if record.RecoveryIntent == nil || !reflect.DeepEqual(record.RecoveryIntent, &intent) {
		return ThreadRecord{}, errors.New("recovery start fence requires the exact retained selection")
	}
	if err := threadrecord.ValidateRecoveryIntent(intent, toPersistedThreadAddress(record.Address)); err != nil {
		return ThreadRecord{}, err
	}
	incarnation := record.Incarnations[0]
	owner := observeExactProcess(c.Proc, ProcessIdentity{PID: incarnation.Start.OwnerPID, Identity: incarnation.Start.OwnerIdentity})
	binding, err := observeRecordSession(withoutSessionOwnerPass(ctx), c.Artifacts, record)
	if err != nil {
		return ThreadRecord{}, err
	}
	if binding.Present {
		return ThreadRecord{}, errors.New("recovery terminal is still live; start cannot be fenced")
	}
	if !DecideStaleStart(record, ThreadEvidence{ObservedAt: c.Clock.Now(), StartOwner: owner, Session: SessionObservation{State: SessionAbsent}}) {
		return ThreadRecord{}, errors.New("recovery source is still starting; its claim has not expired")
	}
	if incarnation.PID != 0 || incarnation.Identity != "" {
		if incarnation.PID <= 0 || incarnation.Identity == "" || observeExactProcess(c.Proc, ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity}) != Dead {
			return ThreadRecord{}, errors.New("recovery start helper is live or could not be proved dead")
		}
	}
	if err := ctx.Err(); err != nil {
		return ThreadRecord{}, err
	}
	return c.Threads.AdvanceStart(record.Address, record.Revision, StartEvent{Kind: StartRecoveryFenced, Nonce: incarnation.Start.Nonce, RecoveryIntent: &intent})
}
