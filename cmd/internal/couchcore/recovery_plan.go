package couchcore

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

type RecoveryRetirement struct {
	Stops  []RecoveryStopProcess
	Digest string
	Check  func(context.Context) error
	Retire func(context.Context) error
}
type RecoveryRetirementPreparer func(context.Context, ThreadRecord) (RecoveryRetirement, error)

func (c *Couch) RecoverSelection(ctx context.Context, selection RecoverySelection) (RecoveryResumeResult, error) {
	return c.executePreparedRecovery(ctx, selection, "")
}

func (c *Couch) ConfirmRecovery(ctx context.Context, token string) (RecoveryResumeResult, error) {
	state, err := c.recoveryConfirmation(ctx, token)
	if err != nil {
		return RecoveryResumeResult{}, err
	}
	return c.executePreparedRecovery(ctx, state.Witness.Selection, token)
}

type RecoverySelectionKind = threadrecord.RecoveryMode

const (
	RecoveryProven   = threadrecord.RecoveryProven
	RecoveryInferred = threadrecord.RecoveryInferred
	RecoveryFresh    = threadrecord.RecoveryFresh
)

// RecoverySelection is an explicit request, never process-stop authority.
// Reference is the canonical repo:N used by the standalone hub confirmation.
type RecoverySelection struct {
	Target       ResumeTarget
	Kind         RecoverySelectionKind
	CandidateKey string
	Agent        string    // optional Fresh agent; argv comes from saved configuration unless Argv is set
	Argv         *[]string // explicit Fresh argv (pair#847 bench profile); requires Agent
	Reference    string
}

type RecoveryStopProcess struct {
	PID      int    `json:"pid"`
	Identity string `json:"identity"`
	Role     string `json:"role"`
}

type recoveryExecutionPhase uint8

const (
	recoveryPrepared recoveryExecutionPhase = iota
	recoveryPreserved
	recoveryRetired
	recoveryClaimed
	recoveryLaunched
)

// AdvanceRecoveryExecution owns the effect order; the coordinator cannot issue
// a retirement or launch without first accepting the preceding durable event.
func AdvanceRecoveryExecution(current, next recoveryExecutionPhase) (recoveryExecutionPhase, error) {
	if next != current+1 {
		return current, errors.New("out-of-order recovery effect")
	}
	return next, nil
}
