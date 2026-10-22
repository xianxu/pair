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

// RefusalClass selects recovery independently of diagnostic wording.
type RefusalClass string

const RefusalRelaunch RefusalClass = "relaunch"

// RefusalInfo is the unauthenticated reason/action projection, never a receipt.
type RefusalInfo struct {
	Reason string       `json:"reason"`
	Class  RefusalClass `json:"class,omitempty"`
}

// RelaunchRequiredError marks ownership evidence that a resend cannot repair.
// The cause keeps its original diagnostic and error identity.
type RelaunchRequiredError struct{ Cause error }

func (e *RelaunchRequiredError) Error() string { return e.Cause.Error() }
func (e *RelaunchRequiredError) Unwrap() error { return e.Cause }

func refusalInfo(err error) RefusalInfo {
	info := RefusalInfo{Reason: refusalReason(err)}
	var relaunch *RelaunchRequiredError
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) && errors.As(err, &relaunch) {
		info.Class = RefusalRelaunch
	}
	return info
}

type refusal struct {
	class   RefusalClass
	receipt Receipt
	target  string // the route's target in canonical slot spelling, as --list labels slots
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
// The target is canonicalized against the connected slots and enrolled
// families, as admission resolves it, so an alias-spelled send is listed
// under the slot's own name; one that doesn't resolve keeps its spelling.
func (b *Broker) refuseLocked(from Binding, id string, route Route, body string, err error, families map[string]string, p messageProvenance) error {
	info := refusalInfo(err)
	reason := info.Reason
	now := b.now()
	var candidates []Candidate
	for binding := range b.actors {
		candidates = append(candidates, Candidate{Binding: binding})
	}
	target := route.Target
	if canonical, ok, cerr := canonicalTarget(route.Target, candidates, families); cerr == nil && ok {
		target = canonical
	}
	b.refusals[id] = refusal{class: info.Class, receipt: Receipt{Message: Message{Origin: p.origin, RemoteSource: p.source, RemoteEdited: p.edited, DocumentID: p.documentID, ID: id, From: from, To: Binding{Slot: target}, Body: body}, Status: Refused, Detail: reason, RetainUntil: now.Add(RefusalRetention)}, target: target, at: now}
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

// LatestRefusals is each requested slot's newest retained refusal reason that
// no later admission to that slot has superseded, read from memory (for
// `couch --list`). It answers an unauthenticated request, so it carries only
// the canonical target, reason and recovery class: never the sender's
// binding, which is a caller credential, nor the body.
func (b *Broker) LatestRefusals() map[string]RefusalInfo {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sweep()
	latest := map[string]refusal{}
	for _, r := range b.refusals {
		if cur, ok := latest[r.target]; !ok || r.at.After(cur.at) {
			latest[r.target] = r
		}
	}
	out := map[string]RefusalInfo{}
	for slot, r := range latest {
		out[slot] = RefusalInfo{Reason: r.receipt.Detail, Class: r.class}
		for _, rec := range b.receipts {
			if rec.Message.To.Slot == slot && !rec.Message.AcceptedAt.Before(r.at) {
				delete(out, slot)
				break
			}
		}
	}
	return out
}
