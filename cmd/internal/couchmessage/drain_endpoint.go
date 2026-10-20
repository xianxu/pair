package couchmessage

import (
	"context"
	"errors"
)

// DrainObservation is wrapper-owned evidence, never inferred from timestamps.
// Turn changes at every lifecycle boundary; Input changes on operator input.
type DrainObservation struct {
	Instance                                                                     string
	Turn, Input                                                                  uint64
	Active, Idle, Picker, ComposerKnown, ComposerEmpty, AutomaticInput, Complete bool
}

func (o DrainObservation) Validate() error {
	if err := (DriveObservation{Instance: o.Instance}).Validate(); err != nil {
		return err
	}
	if o.ComposerEmpty && !o.ComposerKnown {
		return errors.New("empty composer without known composer")
	}
	if o.Idle && (!o.Complete || !o.ComposerEmpty || o.Active || o.Picker || o.AutomaticInput) {
		return errors.New("inconsistent drain idle evidence")
	}
	return nil
}

type DrainInterruptResult struct {
	ID, Instance, Outcome, Detail string
	Turn                          uint64
}
type DrainRevokeResult struct {
	ID, Instance, Outcome string
	Receipt               *Receipt              `json:",omitempty"`
	Interrupt             *DrainInterruptResult `json:",omitempty"`
}

// DrainEndpoint is an optional capability. Old wrappers must fail closed.
type DrainEndpoint interface {
	ObserveDrain(context.Context) (DrainObservation, error)
	InterruptDrain(context.Context, string, DrainObservation) (DrainInterruptResult, error)
	DrainInterruptStatus(context.Context, string) (DrainInterruptResult, error)
	RevokeDrain(context.Context, string) (DrainRevokeResult, error)
}

func (e RemoteEndpoint) ObserveDrain(ctx context.Context) (DrainObservation, error) {
	r, err := e.call(ctx, EndpointRequest{Op: "drain-observe"})
	if err != nil {
		return DrainObservation{}, err
	}
	if r.DrainObservation == nil {
		return DrainObservation{}, errors.New("wrapper drain capability unavailable; relaunch needed")
	}
	return *r.DrainObservation, r.DrainObservation.Validate()
}
func validDrainInterrupt(r *DrainInterruptResult, id string) (DrainInterruptResult, error) {
	if r == nil || r.ID != id {
		return DrainInterruptResult{}, errors.New("wrapper returned no matching drain interrupt result")
	}
	if err := (DriveObservation{Instance: r.Instance}).Validate(); err != nil {
		return DrainInterruptResult{}, err
	}
	switch r.Outcome {
	case "pending", "interrupted", "skipped", "refused", "unknown":
	default:
		return DrainInterruptResult{}, errors.New("invalid drain interrupt outcome")
	}
	if len(r.Detail) > MaxReceiptDetailBytes {
		return DrainInterruptResult{}, errors.New("drain interrupt detail exceeds bounds")
	}
	return *r, nil
}
func (e RemoteEndpoint) InterruptDrain(ctx context.Context, id string, o DrainObservation) (DrainInterruptResult, error) {
	r, err := e.call(ctx, EndpointRequest{Op: "drain-interrupt", ID: id, DrainObserved: &o})
	if err != nil {
		return DrainInterruptResult{}, err
	}
	return validDrainInterrupt(r.DrainInterrupt, id)
}
func (e RemoteEndpoint) DrainInterruptStatus(ctx context.Context, id string) (DrainInterruptResult, error) {
	r, err := e.call(ctx, EndpointRequest{Op: "drain-interrupt-status", ID: id})
	if err != nil {
		return DrainInterruptResult{}, err
	}
	return validDrainInterrupt(r.DrainInterrupt, id)
}
func (e RemoteEndpoint) RevokeDrain(ctx context.Context, id string) (DrainRevokeResult, error) {
	r, err := e.call(ctx, EndpointRequest{Op: "drain-revoke", ID: id})
	if err != nil {
		return DrainRevokeResult{}, err
	}
	if r.DrainRevoke == nil || r.DrainRevoke.ID != id {
		return DrainRevokeResult{}, errors.New("wrapper returned no matching drain revocation")
	}
	if err := (DriveObservation{Instance: r.DrainRevoke.Instance}).Validate(); err != nil {
		return DrainRevokeResult{}, err
	}
	if r.DrainRevoke.Interrupt != nil && r.DrainRevoke.Interrupt.Instance != r.DrainRevoke.Instance {
		return DrainRevokeResult{}, errors.New("drain revoke interrupt instance mismatch")
	}
	switch r.DrainRevoke.Outcome {
	case "not-committed", "unknown":
	case "already-committed":
		if r.DrainRevoke.Interrupt != nil {
			result, err := validDrainInterrupt(r.DrainRevoke.Interrupt, id)
			if err != nil || result.Outcome != "interrupted" {
				return DrainRevokeResult{}, errors.New("drain revocation omitted committed interrupt outcome")
			}
		} else if r.DrainRevoke.Receipt == nil || r.DrainRevoke.Receipt.Message.ID != id {
			return DrainRevokeResult{}, errors.New("drain revocation omitted committed receipt")
		}
	default:
		return DrainRevokeResult{}, errors.New("invalid drain revoke outcome")
	}
	return *r.DrainRevoke, nil
}

// ValidateDrainInstance keeps this optional system-only identity fence closed.
// Empty preserves the wire shape and semantics of ordinary messages.
func ValidateDrainInstance(m Message) error {
	if m.DrainInstance == "" {
		return nil
	}
	if m.Origin != OriginSystem {
		return errors.New("drain instance requires system origin")
	}
	return (DriveObservation{Instance: m.DrainInstance}).Validate()
}
