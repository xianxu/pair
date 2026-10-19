package couchcore

import (
	"fmt"
	"time"
)

// OperationAttemptState retains ordering evidence only while attempts overlap.
// AdvanceOperationAttempt is its only mutation boundary.
type OperationAttemptState struct {
	active       map[uint64]struct{}
	successOrder uint64
	successAt    time.Time
}

// OperationAttemptEventKind names admission and observed terminal outcomes.
type OperationAttemptEventKind uint8

const (
	AttemptAdmitted OperationAttemptEventKind = iota + 1
	AttemptSucceeded
	AttemptFailed
)

// OperationAttemptEvent carries an admission order independent of clock time.
type OperationAttemptEvent struct {
	Kind  OperationAttemptEventKind
	Order uint64
	At    time.Time
}

// OperationAttemptDecision annotates a late failure without inventing success.
type OperationAttemptDecision struct {
	SupersededAt *time.Time
}

// AdvanceOperationAttempt orders terminal evidence by admission, never by its
// arrival time. Its input remains usable for independently ordered executions.
func AdvanceOperationAttempt(state OperationAttemptState, event OperationAttemptEvent) (OperationAttemptState, OperationAttemptDecision, error) {
	var decision OperationAttemptDecision
	if event.Order == 0 {
		return state, decision, fmt.Errorf("operation attempt has no admission order")
	}
	_, active := state.active[event.Order]
	switch event.Kind {
	case AttemptAdmitted:
		if active {
			return state, decision, fmt.Errorf("operation attempt already admitted")
		}
	case AttemptSucceeded, AttemptFailed:
		if !active || event.At.IsZero() {
			return state, decision, fmt.Errorf("operation attempt is not active or lacks completion time")
		}
	default:
		return state, decision, fmt.Errorf("unknown operation attempt event")
	}
	next := state
	next.active = make(map[uint64]struct{}, len(state.active)+1)
	for order := range state.active {
		next.active[order] = struct{}{}
	}
	if event.Kind == AttemptAdmitted {
		next.active[event.Order] = struct{}{}
		return next, decision, nil
	}
	delete(next.active, event.Order)
	if event.Kind == AttemptSucceeded && event.Order > state.successOrder {
		next.successOrder, next.successAt = event.Order, event.At
	}
	if event.Kind == AttemptFailed && state.successOrder > event.Order {
		at := state.successAt
		decision.SupersededAt = &at
	}
	return next, decision, nil
}
