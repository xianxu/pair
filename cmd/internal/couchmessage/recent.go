package couchmessage

import (
	"context"
	"errors"
	"fmt"
)

// MaxRecentDeliveries bounds what a wrapper remembers of past deliveries:
// enough to answer a status query after a broker restart and to refuse a
// reused message ID, held in memory and gone with the wrapper (#365).
const MaxRecentDeliveries = 64

// RecentDeliveries is a bounded, insertion-ordered record of receipts by ID.
type RecentDeliveries struct {
	byID  map[string]Receipt
	order []string
}

func (r *RecentDeliveries) Add(receipt Receipt) {
	id := receipt.Message.ID
	if id == "" {
		return
	}
	if r.byID == nil {
		r.byID = map[string]Receipt{}
	}
	if _, ok := r.byID[id]; !ok {
		r.order = append(r.order, id)
	}
	r.byID[id] = receipt
	for len(r.order) > MaxRecentDeliveries {
		delete(r.byID, r.order[0])
		r.order = r.order[1:]
	}
}

func (r *RecentDeliveries) Get(id string) (Receipt, bool) {
	receipt, ok := r.byID[id]
	return receipt, ok
}

// ErrAlreadyCommitted: the recipient already holds a delivery with this ID.
var ErrAlreadyCommitted = errors.New("message ID already committed at recipient")

// AlreadyCommittedError carries the recipient's retained receipt, so the
// broker reports that outcome instead of pasting the message a second time.
type AlreadyCommittedError struct{ Receipt Receipt }

func (e *AlreadyCommittedError) Error() string {
	return fmt.Sprintf("%v: %s", ErrAlreadyCommitted, e.Receipt.Status)
}
func (e *AlreadyCommittedError) Is(target error) bool { return target == ErrAlreadyCommitted }

// ErrUnknownDelivery: the recipient retains no delivery with this ID.
var ErrUnknownDelivery = errors.New("unknown delivery")

// ReceiptHolder is implemented by endpoints that can report a retained
// receipt for an ID the broker no longer remembers (after a restart).
type ReceiptHolder interface {
	Retained(ctx context.Context, id string) (Receipt, error)
}
