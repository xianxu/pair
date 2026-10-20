package couchmessage

import (
	"context"
	"errors"
)

// DrainObservation negotiates generation authority separately from screen facts.
// Epoch zero and empty Generation mean this incarnation has accepted no drain.
type DrainObservation struct {
	Protocol                                                                     int
	Namespace, Generation                                                        string
	Epoch                                                                        uint64
	Instance                                                                     string
	Turn, Input                                                                  uint64
	Active, Idle, Picker, ComposerKnown, ComposerEmpty, AutomaticInput, Complete bool
}

func (o DrainObservation) Validate() error {
	if o.Protocol != 2 {
		return errors.New("wrapper drain protocol2 unavailable; relaunch needed")
	}
	if _, err := NewDrainGeneration(o.Namespace, o.Instance); err != nil {
		return err
	}
	if o.Epoch == 0 {
		if o.Generation != "" {
			return errors.New("drain observation generation without epoch")
		}
	} else if !(DrainFence{Namespace: o.Namespace, Generation: o.Generation, Epoch: o.Epoch, Instance: o.Instance, Kind: DrainNoticeControl}).valid() {
		return errors.New("invalid observed drain generation")
	}
	if o.ComposerEmpty && !o.ComposerKnown {
		return errors.New("empty composer without known composer")
	}
	if o.Idle && (!o.Complete || !o.ComposerEmpty || o.Active || o.Picker || o.AutomaticInput) {
		return errors.New("inconsistent drain idle evidence")
	}
	return nil
}
func (f DrainFence) valid() bool { return f.Validate() == nil }

type DrainInterruptResult struct {
	ID              string
	Fence           DrainFence
	Outcome, Detail string
	Turn            uint64
}
type DrainRevokeResult struct {
	ID        string
	Fence     DrainFence
	Outcome   string
	Receipt   *Receipt              `json:",omitempty"`
	Interrupt *DrainInterruptResult `json:",omitempty"`
}

type DrainEndpoint interface {
	ObserveDrain(context.Context) (DrainObservation, error)
	InterruptDrain(context.Context, string, DrainFence, DrainObservation) (DrainInterruptResult, error)
	DrainInterruptStatus(context.Context, string, DrainFence) (DrainInterruptResult, error)
	RevokeDrain(context.Context, string, DrainFence) (DrainRevokeResult, error)
}

// DrainReservationEndpoint prevents an unfenced old release from mutating a
// reused control ID. Ordinary peer reservation and cleanup keep their old API.
type DrainReservationEndpoint interface {
	ReserveDrain(context.Context, string, uint64, DrainFence) error
	ReleaseDrain(context.Context, string, DrainFence) error
}

func (e RemoteEndpoint) ObserveDrain(ctx context.Context) (DrainObservation, error) {
	r, err := e.call(ctx, EndpointRequest{Op: "drain-observe"})
	if err != nil {
		return DrainObservation{}, err
	}
	if r.DrainObservation == nil {
		return DrainObservation{}, errors.New("wrapper drain capability unavailable; relaunch needed")
	}
	o := *r.DrainObservation
	if err := o.Validate(); err != nil {
		return DrainObservation{}, err
	}
	namespace, err := DrainNamespaceKey(e.Namespace)
	if err != nil || o.Namespace != namespace {
		return DrainObservation{}, errors.New("wrapper drain observation namespace mismatch")
	}
	return o, nil
}
func (e RemoteEndpoint) validateDrainFence(f DrainFence) error {
	if err := f.Validate(); err != nil {
		return err
	}
	namespace, err := DrainNamespaceKey(e.Namespace)
	if err != nil || f.Namespace != namespace {
		return errors.New("drain fence does not name endpoint namespace")
	}
	return nil
}
func validDrainInterrupt(r *DrainInterruptResult, id string, f DrainFence) (DrainInterruptResult, error) {
	if r == nil || r.ID != id || r.Fence != f || f.Kind != DrainInterruptControl || f.Validate() != nil {
		return DrainInterruptResult{}, errors.New("wrapper returned no matching drain interrupt fence")
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
func (e RemoteEndpoint) InterruptDrain(ctx context.Context, id string, f DrainFence, o DrainObservation) (DrainInterruptResult, error) {
	if err := e.validateDrainFence(f); err != nil {
		return DrainInterruptResult{}, err
	}
	r, err := e.call(ctx, EndpointRequest{Op: "drain-interrupt", ID: id, DrainFence: &f, DrainObserved: &o})
	if err != nil {
		return DrainInterruptResult{}, err
	}
	return validDrainInterrupt(r.DrainInterrupt, id, f)
}
func (e RemoteEndpoint) DrainInterruptStatus(ctx context.Context, id string, f DrainFence) (DrainInterruptResult, error) {
	if err := e.validateDrainFence(f); err != nil {
		return DrainInterruptResult{}, err
	}
	r, err := e.call(ctx, EndpointRequest{Op: "drain-interrupt-status", ID: id, DrainFence: &f})
	if err != nil {
		return DrainInterruptResult{}, err
	}
	return validDrainInterrupt(r.DrainInterrupt, id, f)
}
func (e RemoteEndpoint) RevokeDrain(ctx context.Context, id string, f DrainFence) (DrainRevokeResult, error) {
	if err := e.validateDrainFence(f); err != nil {
		return DrainRevokeResult{}, err
	}
	r, err := e.call(ctx, EndpointRequest{Op: "drain-revoke", ID: id, DrainFence: &f})
	if err != nil {
		return DrainRevokeResult{}, err
	}
	result := r.DrainRevoke
	if result == nil || result.ID != id || result.Fence != f {
		return DrainRevokeResult{}, errors.New("wrapper returned no matching drain revocation fence")
	}
	if result.Receipt != nil && (f.Kind != DrainNoticeControl || result.Receipt.Message.ID != id || result.Receipt.Message.DrainFence != f) {
		return DrainRevokeResult{}, errors.New("drain revoke receipt fence mismatch")
	}
	if result.Interrupt != nil {
		if _, err := validDrainInterrupt(result.Interrupt, id, f); err != nil {
			return DrainRevokeResult{}, err
		}
	}
	switch result.Outcome {
	case "not-committed", "unknown":
	case "already-committed":
		if f.Kind == DrainInterruptControl {
			if result.Interrupt == nil || result.Interrupt.Outcome != "interrupted" {
				return DrainRevokeResult{}, errors.New("drain revocation omitted committed interrupt outcome")
			}
		} else if result.Receipt == nil {
			return DrainRevokeResult{}, errors.New("drain revocation omitted committed receipt")
		}
	default:
		return DrainRevokeResult{}, errors.New("invalid drain revoke outcome")
	}
	return *result, nil
}
func (e RemoteEndpoint) ReserveDrain(ctx context.Context, id string, sequence uint64, f DrainFence) error {
	if err := e.validateDrainFence(f); err != nil {
		return err
	}
	_, err := e.call(ctx, EndpointRequest{Op: "drain-reserve", ID: id, Sequence: sequence, DrainFence: &f})
	return err
}
func (e RemoteEndpoint) ReleaseDrain(ctx context.Context, id string, f DrainFence) error {
	if err := e.validateDrainFence(f); err != nil {
		return err
	}
	_, err := e.call(ctx, EndpointRequest{Op: "drain-release", ID: id, DrainFence: &f})
	return err
}

// ValidateDrainInstance keeps the optional system-only drain capability closed.
// The name remains compatible with existing envelope admission call sites.
func ValidateDrainInstance(m Message) error {
	if m.DrainFence.IsZero() {
		return nil
	}
	if m.Origin != OriginSystem || m.DrainFence.Kind != DrainNoticeControl {
		return errors.New("drain notice fence requires system origin and notice kind")
	}
	return m.DrainFence.Validate()
}
