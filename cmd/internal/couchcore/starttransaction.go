package couchcore

import (
	"errors"
	"fmt"
)

type ProcessIdentity struct {
	PID      int
	Identity string
}

// StartTransaction is the pure projection of the durable recovery facts. OS
// descriptors and in-memory handles deliberately remain in Runner.
type StartTransaction struct {
	Nonce  string
	Owner  ProcessIdentity
	Helper *ProcessIdentity
}

type StartEventKind string

const (
	StartClaimed          StartEventKind = "claimed"
	StartHelperRecorded   StartEventKind = "helper-recorded"
	StartRegistered       StartEventKind = "registered"
	StartRecoveredUnknown StartEventKind = "recovered-unknown"
)

type StartEvent struct {
	Kind    StartEventKind
	Nonce   string
	Owner   SupervisorOwner
	Helper  ProcessIdentity
	Profile *LaunchProfile
}

// AdvanceStartTransaction is the pure transition authority for one persisted
// creating incarnation. It never increments Revision; ThreadStore's CAS update
// does that after this transition validates the complete resulting record.
func AdvanceStartTransaction(record ThreadRecord, event StartEvent) (ThreadRecord, error) {
	next := cloneThreadRecord(record)
	switch event.Kind {
	case StartClaimed:
		if len(next.Incarnations) != 1 {
			return ThreadRecord{}, errors.New("start claim requires one creating incarnation")
		}
		incarnation := &next.Incarnations[0]
		if incarnation.State != IncarnationCreating || incarnation.Start != nil || incarnation.PID != 0 || incarnation.Identity != "" {
			return ThreadRecord{}, errors.New("start claim requires an untracked creating incarnation")
		}
		incarnation.Start = &ThreadStartClaim{
			Nonce:         event.Nonce,
			OwnerPID:      event.Owner.PID,
			OwnerIdentity: event.Owner.Identity,
		}
		if event.Profile != nil {
			profile := cloneLaunchProfile(*event.Profile)
			incarnation.Start.LaunchProfile = &profile
		}
	case StartHelperRecorded:
		incarnation, err := exactStartIncarnation(&next, event.Nonce)
		if err != nil {
			return ThreadRecord{}, err
		}
		if incarnation.PID != 0 || incarnation.Identity != "" {
			return ThreadRecord{}, errors.New("start helper is already recorded")
		}
		if event.Helper.PID <= 0 || event.Helper.Identity == "" {
			return ThreadRecord{}, errors.New("start helper pid and identity are required")
		}
		incarnation.PID = event.Helper.PID
		incarnation.Identity = event.Helper.Identity
	case StartRegistered, StartRecoveredUnknown:
		incarnation, err := exactStartIncarnation(&next, event.Nonce)
		if err != nil {
			return ThreadRecord{}, err
		}
		if incarnation.PID <= 0 || incarnation.Identity == "" {
			return ThreadRecord{}, errors.New("cannot register start before helper identity")
		}
		if event.Kind == StartRegistered {
			incarnation.State = IncarnationLive
		} else {
			incarnation.State = IncarnationUnknown
		}
		if incarnation.Start.LaunchProfile != nil {
			profile := cloneLaunchProfile(*incarnation.Start.LaunchProfile)
			incarnation.LaunchProfile = &profile
		}
		incarnation.Start = nil
	default:
		return ThreadRecord{}, fmt.Errorf("unknown start event %q", event.Kind)
	}
	if err := ValidateThreadRecord(next); err != nil {
		return ThreadRecord{}, err
	}
	return next, nil
}

func exactStartIncarnation(record *ThreadRecord, nonce string) (*ThreadIncarnation, error) {
	if nonce == "" {
		return nil, errors.New("start nonce is required")
	}
	var found *ThreadIncarnation
	for i := range record.Incarnations {
		incarnation := &record.Incarnations[i]
		if incarnation.Start == nil || incarnation.Start.Nonce != nonce {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("multiple incarnations carry start nonce %q", nonce)
		}
		found = incarnation
	}
	if found == nil {
		return nil, fmt.Errorf("start nonce %q not found", nonce)
	}
	if found.State != IncarnationCreating {
		return nil, fmt.Errorf("start nonce %q is not creating", nonce)
	}
	return found, nil
}

type RegistrationEvidence string

const (
	RegistrationUnknown     RegistrationEvidence = "unknown"
	RegistrationAbsent      RegistrationEvidence = "absent"
	RegistrationEstablished RegistrationEvidence = "established"
)

type StartObservation struct {
	Owner        Liveness
	Helper       Liveness
	Registration RegistrationEvidence
}

type StartReconcileAction string

const (
	StartKeepOccupied   StartReconcileAction = "keep-occupied"
	StartRollback       StartReconcileAction = "rollback"
	StartPromoteLive    StartReconcileAction = "promote-live"
	StartPromoteUnknown StartReconcileAction = "promote-unknown"
)

type StartReconcileDecision struct {
	Nonce  string
	Action StartReconcileAction
}

// ReconcileStart applies the occupied-or-proven-free rule to one interrupted
// transaction. Unknown evidence always keeps capacity occupied.
func ReconcileStart(record ThreadRecord, observation StartObservation) (StartReconcileDecision, error) {
	transaction, err := CurrentStartTransaction(record)
	if err != nil {
		return StartReconcileDecision{}, err
	}
	decision := StartReconcileDecision{Nonce: transaction.Nonce, Action: StartKeepOccupied}

	if transaction.Helper == nil {
		if observation.Registration == RegistrationAbsent && observation.Owner == Dead {
			decision.Action = StartRollback
		}
		return decision, nil
	}

	switch observation.Registration {
	case RegistrationUnknown:
		return decision, nil
	case RegistrationEstablished:
		if observation.Helper == Live {
			decision.Action = StartPromoteLive
		} else {
			decision.Action = StartPromoteUnknown
		}
		return decision, nil
	case RegistrationAbsent:
		if observation.Helper == Dead {
			decision.Action = StartRollback
		}
		return decision, nil
	default:
		return StartReconcileDecision{}, fmt.Errorf("invalid registration evidence %q", observation.Registration)
	}
}

func CurrentStartTransaction(record ThreadRecord) (StartTransaction, error) {
	var incarnation *ThreadIncarnation
	for i := range record.Incarnations {
		if record.Incarnations[i].Start == nil {
			continue
		}
		if incarnation != nil {
			return StartTransaction{}, errors.New("thread has multiple tracked starts")
		}
		incarnation = &record.Incarnations[i]
	}
	if incarnation == nil {
		return StartTransaction{}, errors.New("thread has no tracked start")
	}
	transaction := StartTransaction{
		Nonce: incarnation.Start.Nonce,
		Owner: ProcessIdentity{PID: incarnation.Start.OwnerPID, Identity: incarnation.Start.OwnerIdentity},
	}
	if incarnation.PID > 0 {
		helper := ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity}
		transaction.Helper = &helper
	}
	return transaction, nil
}
