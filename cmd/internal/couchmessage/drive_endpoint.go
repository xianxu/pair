package couchmessage

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"strings"
	"time"
	"unicode"
)

const DriveBudget = 30 * time.Second

type DriveObservation struct {
	Instance string
	// Sequence fences operator input for an Interruptible proof and all
	// activity for other proofs. Native working redraws do not revoke interrupt.
	Sequence uint64
	Idle     bool
	// Recoverable positively identifies a known composer without an active turn,
	// picker or buffered input; only recovery-only keys may use it.
	Recoverable bool
	// Dismissable positively identifies a panel whose footer declares Esc as
	// its dismissal; only an exact [esc] may use it (pair#691).
	Dismissable bool
	// Interruptible positively identifies a working turn the wrapper may stop;
	// only an interrupt request may use it (pair#717).
	Interruptible bool
}

func (o DriveObservation) Admission() agentdrive.Admission {
	return agentdrive.Admission{Idle: o.Idle, Recoverable: o.Recoverable, Dismissable: o.Dismissable, Interruptible: o.Interruptible}
}

func (o DriveObservation) Validate() error {
	if o.Instance == "" || len(o.Instance) > 128 || strings.IndexFunc(o.Instance, unicode.IsControl) >= 0 {
		return errors.New("invalid wrapper drive instance")
	}
	return nil
}

// Instance and Sequence bind admission to a fresh wrapper observation. The
// absolute deadline belongs to the drive, never the short endpoint RPC.
type DriveStart struct {
	Request  agentdrive.Request
	Deadline time.Time
	Instance string
	Sequence uint64
}

func (s DriveStart) Validate(now time.Time) error {
	if err := s.Request.Validate(); err != nil {
		return err
	}
	if err := (DriveObservation{Instance: s.Instance}).Validate(); err != nil {
		return err
	}
	if s.Deadline.IsZero() || !s.Deadline.After(now) || s.Deadline.After(now.Add(DriveBudget)) {
		return errors.New("drive deadline must be within the next 30 seconds")
	}
	return nil
}
func (e RemoteEndpoint) ObserveDrive(ctx context.Context) (DriveObservation, error) {
	r, err := e.call(ctx, EndpointRequest{Op: "drive-observe"})
	if err != nil {
		return DriveObservation{}, err
	}
	if r.DriveObservation == nil {
		return DriveObservation{}, errors.New("wrapper returned no drive observation")
	}
	if err := r.DriveObservation.Validate(); err != nil {
		return DriveObservation{}, err
	}
	return *r.DriveObservation, nil
}
func validateDriveResult(r *agentdrive.Result, id string) (agentdrive.Result, error) {
	if r == nil {
		return agentdrive.Result{}, errors.New("wrapper returned no drive result")
	}
	if r.ID != id {
		return agentdrive.Result{}, errors.New("wrapper returned result for another drive")
	}
	switch r.Outcome {
	case agentdrive.OutcomePending, agentdrive.OutcomeConfirmed, agentdrive.OutcomeRefused, agentdrive.OutcomeFailed, agentdrive.OutcomeUnconfirmed:
	default:
		return agentdrive.Result{}, errors.New("wrapper returned unknown drive outcome")
	}
	if len(r.Detail) > MaxReceiptDetailBytes || r.Step < 0 || r.Step > agentdrive.MaxNavigationSteps+agentdrive.MaxKeySteps+8 || len(r.Tail) > MaxTailLines {
		return agentdrive.Result{}, errors.New("wrapper drive result exceeds bounds")
	}
	size := 0
	for _, line := range r.Tail {
		size += len(line) + 1
	}
	if size > agentdrive.MaxResultTailBytes {
		return agentdrive.Result{}, fmt.Errorf("wrapper drive tail exceeds %d bytes", agentdrive.MaxResultTailBytes)
	}
	return *r, nil
}
func (e RemoteEndpoint) StartDrive(ctx context.Context, id string, start DriveStart) (agentdrive.Result, error) {
	r, err := e.call(ctx, EndpointRequest{Op: "drive-start", ID: id, Drive: &start})
	if err != nil {
		return agentdrive.Result{}, err
	}
	return validateDriveResult(r.DriveResult, id)
}
func (e RemoteEndpoint) DriveStatus(ctx context.Context, id string) (agentdrive.Result, error) {
	r, err := e.call(ctx, EndpointRequest{Op: "drive-status", ID: id})
	if err != nil {
		return agentdrive.Result{}, err
	}
	return validateDriveResult(r.DriveResult, id)
}
func (e RemoteEndpoint) CancelDrive(ctx context.Context, id string) (agentdrive.Result, error) {
	r, err := e.call(ctx, EndpointRequest{Op: "drive-cancel", ID: id})
	if err != nil {
		return agentdrive.Result{}, err
	}
	return validateDriveResult(r.DriveResult, id)
}
