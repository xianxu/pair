package couchcore

import (
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

type ParkPhase string

const (
	ParkRequested          ParkPhase = "requested"
	ParkAwaitingCompletion ParkPhase = "awaiting_completion"
	ParkUnknown            ParkPhase = "unknown"
)

type ParkIdentity struct {
	Nonce           string        `json:"nonce"`
	Address         ThreadAddress `json:"address"`
	PID             int           `json:"pid"`
	ProcessIdentity string        `json:"process_identity"`
}

type ParkFailure struct {
	Code       pairlifecycle.FailureCode `json:"code"`
	Diagnostic string                    `json:"diagnostic"`
}

type ParkAttempt struct {
	Number   uint64       `json:"number"`
	Failure  *ParkFailure `json:"failure,omitempty"`
	TimedOut bool         `json:"timed_out,omitempty"`
	Closed   bool         `json:"closed,omitempty"`
}

type ParkTransaction struct {
	Identity          ParkIdentity  `json:"identity"`
	BaseRevision      uint64        `json:"base_revision"`
	RecordRevision    uint64        `json:"record_revision"`
	Phase             ParkPhase     `json:"phase"`
	Attempts          []ParkAttempt `json:"attempts"`
	Closed            bool          `json:"closed,omitempty"`
	Tombstoned        bool          `json:"tombstoned,omitempty"`
	SuccessfulAttempt uint64        `json:"successful_attempt,omitempty"`
}

type VerifiedPark struct {
	Identity ParkIdentity `json:"identity"`
	Attempt  uint64       `json:"attempt"`
	ParkedAt time.Time    `json:"parked_at"`
}

type ParkEventKind string

const (
	ParkBegin               ParkEventKind = "begin"
	ParkRequestCommitted    ParkEventKind = "request_committed"
	ParkFailureObserved     ParkEventKind = "failure_observed"
	ParkAttemptAppended     ParkEventKind = "attempt_appended"
	ParkCompletionSucceeded ParkEventKind = "completion_succeeded"
	ParkAbandoned           ParkEventKind = "abandoned"
)

type ParkEvent struct {
	Kind           ParkEventKind
	Identity       ParkIdentity
	BaseRevision   uint64
	RecordRevision uint64
	Attempt        uint64
	Failure        *ParkFailure
}

type ParkDecision struct {
	Finalize          bool
	HistoricalNoOp    bool
	SuccessfulAttempt uint64
}

// AdvanceParkTransaction is the pure authority for one park lifecycle. The
// caller supplies the revision written by its CAS; identity and base revision
// remain stable while RecordRevision advances with each durable mutation.
func AdvanceParkTransaction(current *ParkTransaction, event ParkEvent) (ParkTransaction, ParkDecision, error) {
	if current == nil {
		return beginParkTransaction(event)
	}
	next := cloneParkTransaction(*current)
	if next.Closed {
		switch event.Kind {
		case ParkCompletionSucceeded, ParkFailureObserved:
			return next, ParkDecision{HistoricalNoOp: true}, nil
		default:
			return ParkTransaction{}, ParkDecision{}, errors.New("park transaction is closed")
		}
	}
	if !zeroParkIdentity(event.Identity) && event.Identity != next.Identity {
		return ParkTransaction{}, ParkDecision{}, errors.New("park transaction identity cannot change")
	}
	if event.RecordRevision <= next.RecordRevision {
		return ParkTransaction{}, ParkDecision{}, fmt.Errorf("park record revision %d must advance past %d", event.RecordRevision, next.RecordRevision)
	}

	switch event.Kind {
	case ParkRequestCommitted:
		attempt, err := activeParkAttempt(&next, event.Attempt)
		if err != nil {
			return ParkTransaction{}, ParkDecision{}, err
		}
		if next.Phase != ParkRequested || attempt.Closed {
			return ParkTransaction{}, ParkDecision{}, errors.New("request commit requires an active requested attempt")
		}
		next.Phase = ParkAwaitingCompletion
	case ParkFailureObserved:
		attempt, err := parkAttempt(&next, event.Attempt)
		if err != nil {
			return ParkTransaction{}, ParkDecision{}, err
		}
		if attempt.Closed {
			return ParkTransaction{}, ParkDecision{}, fmt.Errorf("park attempt %d already has an immutable result", event.Attempt)
		}
		if event.Failure == nil || event.Failure.Diagnostic == "" || !validParkFailureCode(event.Failure.Code) {
			return ParkTransaction{}, ParkDecision{}, errors.New("park failure code and diagnostic are required")
		}
		historicalAttempt := next.Attempts[len(next.Attempts)-1].Number != event.Attempt
		failure := *event.Failure
		attempt.Failure = &failure
		switch failure.Code {
		case pairlifecycle.FailureRequestPublishFailed:
			if next.Phase != ParkRequested || attempt.Closed {
				return ParkTransaction{}, ParkDecision{}, errors.New("request publication failure requires a requested attempt")
			}
		case pairlifecycle.FailureTimeout:
			if next.Phase != ParkAwaitingCompletion || attempt.Closed {
				return ParkTransaction{}, ParkDecision{}, errors.New("timeout requires an awaiting attempt")
			}
			attempt.TimedOut = true
		case pairlifecycle.FailureCleanupFailed:
			attempt.Closed = true
			if !historicalAttempt {
				next.Phase = ParkUnknown
			}
		case pairlifecycle.FailureCompletionMissing,
			pairlifecycle.FailureRevisionConflict,
			pairlifecycle.FailureReplacementIncarnation:
			if !historicalAttempt {
				next.Phase = ParkUnknown
			}
		case pairlifecycle.FailureStaleCompletion:
			// Invalid evidence is retained without weakening the current phase.
		}
	case ParkAttemptAppended:
		if len(next.Attempts) == 0 {
			return ParkTransaction{}, ParkDecision{}, errors.New("a new park attempt requires an existing attempt")
		}
		previous := next.Attempts[len(next.Attempts)-1]
		if !previous.Closed && !previous.TimedOut && next.Phase != ParkUnknown {
			return ParkTransaction{}, ParkDecision{}, errors.New("a new park attempt requires a closed, timed-out, or unknown previous attempt")
		}
		next.Attempts = append(next.Attempts, ParkAttempt{Number: next.Attempts[len(next.Attempts)-1].Number + 1})
		next.Phase = ParkRequested
	case ParkCompletionSucceeded:
		attempt, err := parkAttempt(&next, event.Attempt)
		if err != nil {
			return ParkTransaction{}, ParkDecision{}, err
		}
		if next.Tombstoned {
			return next, ParkDecision{HistoricalNoOp: true}, nil
		}
		if attempt.Closed {
			return ParkTransaction{}, ParkDecision{}, fmt.Errorf("park attempt %d already has an immutable result", event.Attempt)
		}
		attempt.Closed = true
		next.Closed = true
		next.SuccessfulAttempt = event.Attempt
		next.RecordRevision = event.RecordRevision
		return next, ParkDecision{Finalize: true, SuccessfulAttempt: event.Attempt}, nil
	case ParkAbandoned:
		next.Closed = true
		next.Tombstoned = true
	default:
		return ParkTransaction{}, ParkDecision{}, fmt.Errorf("unknown park event %q", event.Kind)
	}
	next.RecordRevision = event.RecordRevision
	return next, ParkDecision{}, nil
}

func beginParkTransaction(event ParkEvent) (ParkTransaction, ParkDecision, error) {
	if event.Kind != ParkBegin {
		return ParkTransaction{}, ParkDecision{}, errors.New("park transaction must begin with begin event")
	}
	if err := validateParkIdentity(event.Identity); err != nil {
		return ParkTransaction{}, ParkDecision{}, err
	}
	if event.BaseRevision == 0 || event.RecordRevision <= event.BaseRevision {
		return ParkTransaction{}, ParkDecision{}, errors.New("park base revision must be positive and precede record revision")
	}
	return ParkTransaction{
		Identity: event.Identity, BaseRevision: event.BaseRevision, RecordRevision: event.RecordRevision,
		Phase: ParkRequested, Attempts: []ParkAttempt{{Number: 1}},
	}, ParkDecision{}, nil
}

func validateParkIdentity(identity ParkIdentity) error {
	if err := validateThreadAddress(identity.Address); err != nil {
		return err
	}
	request := pairlifecycle.QuitRequest{
		SchemaVersion: pairlifecycle.SchemaVersion,
		Identity: pairlifecycle.Identity{
			Nonce: identity.Nonce, RepoScope: identity.Address.RepoScope, Tag: string(identity.Address.Tag),
			PID: identity.PID, ProcessIdentity: identity.ProcessIdentity,
		},
		Attempt: 1, Session: "identity-validation", Mode: pairlifecycle.CleanupPreserveScrollback,
		CompletionKey: "identity-validation",
	}
	return pairlifecycle.ValidateQuitRequest(request)
}

func parkAttempt(transaction *ParkTransaction, number uint64) (*ParkAttempt, error) {
	if number == 0 {
		return nil, errors.New("park attempt must be positive")
	}
	for i := range transaction.Attempts {
		if transaction.Attempts[i].Number == number {
			return &transaction.Attempts[i], nil
		}
	}
	return nil, fmt.Errorf("park attempt %d does not exist", number)
}

func activeParkAttempt(transaction *ParkTransaction, number uint64) (*ParkAttempt, error) {
	attempt, err := parkAttempt(transaction, number)
	if err != nil {
		return nil, err
	}
	if len(transaction.Attempts) == 0 || transaction.Attempts[len(transaction.Attempts)-1].Number != number {
		return nil, fmt.Errorf("park attempt %d is historical", number)
	}
	return attempt, nil
}

func cloneParkTransaction(transaction ParkTransaction) ParkTransaction {
	copy := transaction
	copy.Attempts = append([]ParkAttempt(nil), transaction.Attempts...)
	for i := range copy.Attempts {
		if transaction.Attempts[i].Failure != nil {
			failure := *transaction.Attempts[i].Failure
			copy.Attempts[i].Failure = &failure
		}
	}
	return copy
}

func zeroParkIdentity(identity ParkIdentity) bool { return identity == (ParkIdentity{}) }

func validParkFailureCode(code pairlifecycle.FailureCode) bool {
	switch code {
	case pairlifecycle.FailureRequestPublishFailed, pairlifecycle.FailureCleanupFailed,
		pairlifecycle.FailureTimeout, pairlifecycle.FailureCompletionMissing,
		pairlifecycle.FailureStaleCompletion, pairlifecycle.FailureRevisionConflict,
		pairlifecycle.FailureReplacementIncarnation:
		return true
	default:
		return false
	}
}

// MonotonicLastActiveAt prevents a backward or equal wall clock from reducing
// durable recency.
func MonotonicLastActiveAt(previous, observed time.Time) time.Time {
	if observed.After(previous) {
		return observed
	}
	return previous
}
