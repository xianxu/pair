package couchmessage

import (
	"context"
	"errors"
	"time"
)

// BatchDeliveryEndpoint prepares without input, then freezes the receiver's
// current FIFO queue exactly once when its input writer reports readiness.
// The freeze result contains the original head followed by immutable members.
type BatchDeliveryEndpoint interface {
	DeliverBatch(context.Context, Message, func() ([]Message, error), func(Receipt)) (Receipt, error)
}

var _ BatchDeliveryEndpoint = RemoteEndpoint{}

// endpointRefusal distinguishes an explicit wrapper refusal from a transport
// failure with the same text. Only the former can prove preparation never ran.
type endpointRefusal struct{ message string }

func (e *endpointRefusal) Error() string { return e.message }

func (e RemoteEndpoint) DeliverBatch(ctx context.Context, head Message, freeze func() ([]Message, error), progress func(Receipt)) (Receipt, error) {
	if freeze == nil {
		return Receipt{}, errors.New("batch delivery requires a freeze callback")
	}
	wire := head
	wire.AcceptedAt = time.Time{}
	prepareCtx, cancel := context.WithDeadline(ctx, head.Deadline)
	response, err := e.call(prepareCtx, EndpointRequest{Op: "prepare-batch", ID: head.ID, Message: &wire})
	cancel()
	if err != nil {
		var refusal *endpointRefusal
		if errors.As(err, &refusal) && refusal.message == `unknown endpoint operation "prepare-batch"` {
			return e.Deliver(ctx, head, progress)
		}
		return Receipt{}, err
	}
	timer := time.NewTimer(50 * time.Millisecond)
	defer timer.Stop()
	var last Receipt
	for {
		if response.Receipt != nil {
			receipt := *response.Receipt
			if !matchingReceipt(receipt, head) {
				return Receipt{}, errors.New("endpoint returned a mismatched receipt")
			}
			if receipt.Status.Terminal() {
				return receipt, nil
			}
			if receipt.Status != Queued && receipt.Status != Held {
				return Receipt{}, errors.New("endpoint returned an invalid batch preparation status")
			}
			if progress != nil && (receipt.Status != last.Status || receipt.Detail != last.Detail) {
				progress(receipt)
			}
			last = receipt
		}
		if response.BatchReady {
			break
		}
		select {
		case <-ctx.Done():
			return Receipt{}, ctx.Err()
		case <-timer.C:
		}
		response, err = e.call(ctx, EndpointRequest{Op: "batch-status", ID: head.ID})
		if err != nil {
			return Receipt{}, err
		}
		if response.Receipt == nil {
			return Receipt{}, errors.New("endpoint returned no batch preparation status")
		}
		timer.Reset(peerHeldPoll)
	}
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	members, err := freeze()
	if err != nil {
		return Receipt{}, err
	}
	if err := ValidateBatch(members, head.To, head.ID); err != nil {
		return Receipt{}, err
	}
	if !matchingReceipt(Receipt{Message: members[0]}, head) {
		return Receipt{}, errors.New("batch freeze changed the prepared head")
	}
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	// Admission timestamps remain broker evidence. Strip every member on a copy,
	// preserving the callback's original immutable identities and deadlines.
	members = append([]Message(nil), members...)
	for i := range members {
		members[i].AcceptedAt = time.Time{}
	}
	response, err = e.call(ctx, EndpointRequest{Op: "commit-batch", ID: head.ID, Messages: members})
	if err != nil {
		return Receipt{}, err
	}
	return e.waitForDelivery(ctx, head, response, progress)
}
