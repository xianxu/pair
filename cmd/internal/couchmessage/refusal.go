package couchmessage

import (
	"context"
	"errors"
	"time"
)

// A refusal records that a send was never admitted, and why (pair#618). Before
// it, an admission that failed (most often its ownership checks outrunning the
// admission budget under load) left no trace: the sender saw "uncertain" and
// the status query could only say no recipient retains the message. Admission
// is the only way into a recipient's inbox, so a send without a receipt was not
// delivered; the refusal says so with the reason.
//
// Refusals live beside the receipts with their own bounds: shorter retention,
// a small cap that evicts the oldest, never counted against MaxReceipts, and a
// resend with the same ID is a fresh attempt.
const (
	RefusalRetention = 10 * time.Minute
	MaxRefusals      = 64
)

type refusal struct {
	receipt Receipt
	route   Route
	at      time.Time
}

// RefusedError is a send that was not admitted. It is definite: the message
// reached no inbox.
type RefusedError struct{ Reason string }

func (e *RefusedError) Error() string { return "not admitted: " + e.Reason }

// refusalReason names an admission failure for the sender.
func refusalReason(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "admission timed out before the recipient accepted it (its checks took longer than " + AdmissionTimeout.String() + ")"
	case errors.Is(err, context.Canceled):
		return "admission was cancelled before the recipient accepted it"
	}
	return err.Error()
}

// refuseLocked records why id was not admitted and returns the sender's error.
// Errors that are already definite keep their own code; an uncertain or
// interrupted admission becomes a definite refusal, because no receipt exists.
func (b *Broker) refuseLocked(from Binding, id string, route Route, body string, err error) error {
	reason := refusalReason(err)
	now := b.now()
	to := Binding{Slot: route.Target}
	b.refusals[id] = refusal{receipt: Receipt{Message: Message{ID: id, From: from, To: to, Body: body}, Status: Refused, Detail: reason, RetainUntil: now.Add(RefusalRetention)}, route: route, at: now}
	for len(b.refusals) > MaxRefusals {
		var oldest string
		for k, r := range b.refusals {
			if oldest == "" || r.at.Before(b.refusals[oldest].at) {
				oldest = k
			}
		}
		delete(b.refusals, oldest)
	}
	if errors.Is(err, ErrUncertain) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &RefusedError{Reason: reason}
	}
	return err
}

func (b *Broker) sweepRefusalsLocked(now time.Time) {
	for id, r := range b.refusals {
		if !now.Before(r.receipt.RetainUntil) {
			delete(b.refusals, id)
		}
	}
}

// LatestRefusals is each requested slot's newest retained refusal that no
// later admission to that slot has superseded, read from memory (for
// `couch --list`). Keys are the route targets the senders named.
func (b *Broker) LatestRefusals() map[string]Receipt {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sweep()
	latest := map[string]refusal{}
	for _, r := range b.refusals {
		slot := r.route.Target
		if cur, ok := latest[slot]; !ok || r.at.After(cur.at) {
			latest[slot] = r
		}
	}
	out := map[string]Receipt{}
	for slot, r := range latest {
		out[slot] = r.receipt
		for _, rec := range b.receipts {
			if rec.Message.To.Slot == slot && !rec.Message.AcceptedAt.Before(r.at) {
				delete(out, slot)
				break
			}
		}
	}
	return out
}
