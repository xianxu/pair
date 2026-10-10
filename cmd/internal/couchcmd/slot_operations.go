package couchcmd

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// slotOperationRunner enqueues one slot operation on the console's queue
// (couchtty.Console.EnqueueRemoteOperation with Couch.PrepareSlotOperation).
// When it returns an error it never calls started or finished.
type slotOperationRunner func(key, op, target string, opts couchcore.LiveRestartOptions, started func(), finished func(any, error)) error

// maxSlotOperationReceipts bounds the receipts held at once (about 1 KiB each);
// the next admission is refused overloaded.
const maxSlotOperationReceipts = 64

// slotOperations is the socket side of resume/reboot (pair#367 M2): it admits
// a request from a verified caller, holds its receipt in memory, and answers
// operation-status. One mutex owns the receipt map; admit and status (socket
// handlers), started (the queue runner) and finished (the console loop) all
// take it, and none waits on the queue or the console while holding it.
type slotOperations struct {
	run   slotOperationRunner
	names func(context.Context) ([]couchcore.RepositoryName, error)
	now   func() time.Time

	mu       sync.Mutex
	receipts map[receiptKey]couchmessage.OperationReceipt
}

// receiptKey scopes a receipt to the slot that admitted it: another caller
// never reads it.
type receiptKey struct{ scope, tag, id string }

func newSlotOperations(run slotOperationRunner, names func(context.Context) ([]couchcore.RepositoryName, error), now func() time.Time) *slotOperations {
	return &slotOperations{run: run, names: names, now: now, receipts: map[receiptKey]couchmessage.OperationReceipt{}}
}

// handle answers one validated request from a verified, current caller.
func (s *slotOperations) handle(ctx context.Context, caller couchmessage.Binding, r couchmessage.Request) couchmessage.Response {
	key := receiptKey{caller.Scope, caller.Tag, r.ID}
	if r.Op == "operation-status" {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.sweepLocked()
		receipt := s.receipts[key]
		if receipt.Status == couchmessage.ReceiptNone {
			receipt = couchmessage.OperationReceipt{ID: r.ID, Status: couchmessage.ReceiptUnknown}
		}
		return couchmessage.Response{Code: "ok", Operation: &receipt}
	}
	if confirms, _ := couchcore.OperationConfirms(r.Op); confirms && !r.Confirmed {
		return couchmessage.Response{Code: "confirmation-required", Error: r.Op + " requires --confirm"}
	}
	queueKey, err := s.queueKey(ctx, r.Target)
	if err != nil {
		return couchmessage.Response{Code: couchcore.SlotOpUnknownSlot, Error: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()
	existing, held := s.receipts[key]
	receipt, err := couchmessage.ApplyReceiptEvent(existing, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptAdmit, ID: r.ID, Op: r.Op, Target: r.Target, At: s.now()})
	switch {
	case errors.Is(err, couchmessage.ErrReceiptIDConflict):
		return couchmessage.Response{Code: "id-conflict", Error: err.Error()}
	case err != nil:
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	case held:
		// A duplicate admission returns the receipt as it stands and never
		// enqueues a second job.
		return couchmessage.Response{Code: "accepted", Operation: &receipt}
	case len(s.receipts) >= maxSlotOperationReceipts:
		return couchmessage.Response{Code: "overloaded", Error: "too many slot operations held; read the report and retry later"}
	}
	s.receipts[key] = receipt
	// Enqueue never blocks, so holding the lock is safe; a job that starts at
	// once waits in started until this admission returns.
	opts := couchcore.LiveRestartOptions{SameBinary: r.SameBinary, ForceUnknown: r.ForceUnknown}
	err = s.run(queueKey, r.Op, r.Target, opts, func() { s.apply(key, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptStart, ID: r.ID}) },
		func(value any, err error) {
			s.apply(key, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptFinish, ID: r.ID, Outcome: slotOperationOutcome(value, err), At: s.now()})
		})
	if err != nil {
		// The single report of an enqueue refusal: drop the reservation.
		delete(s.receipts, key)
		code := "unavailable"
		switch {
		case errors.Is(err, couchtty.ErrRemotePending):
			code = "busy"
		case errors.Is(err, couchtty.ErrOperationQueueOverloaded):
			code = "overloaded"
		}
		return couchmessage.Response{Code: code, Error: err.Error()}
	}
	return couchmessage.Response{Code: "accepted", Operation: &receipt}
}

// apply moves one held receipt; an event the table refuses (a late or
// repeated finish) leaves it unchanged.
func (s *slotOperations) apply(key receiptKey, e couchmessage.ReceiptEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	receipt, held := s.receipts[key]
	if !held {
		return
	}
	if next, err := couchmessage.ApplyReceiptEvent(receipt, e); err == nil {
		s.receipts[key] = next
	}
}

// sweepLocked removes terminal receipts past their retention.
func (s *slotOperations) sweepLocked() {
	now := s.now()
	for key, receipt := range s.receipts {
		if next, err := couchmessage.ApplyReceiptEvent(receipt, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptExpire, At: now}); err == nil && next.Status == couchmessage.ReceiptNone {
			delete(s.receipts, key)
		}
	}
}

// queueKey names the slot a request acts on by its resolved repository, so
// every spelling of one slot (name, alias, prefix) shares one pending key. A
// store read only: no git, no sdlc.
func (s *slotOperations) queueKey(ctx context.Context, target string) (string, error) {
	ref, _, err := couchcore.ParseWorkspaceReference(target)
	if err != nil {
		return "", err
	}
	names, err := s.names(ctx)
	if err != nil {
		return "", err
	}
	repo, _, err := couchcore.ResolveRepositoryName(ref.Repo, names)
	if err != nil {
		return "", err
	}
	return "remote\x00" + repo.Key + ":" + strconv.Itoa(ref.Number), nil
}

// slotOperationOutcome maps a finished job to its receipt outcome: a typed
// slot refusal is refused with its code, any other error failed with its
// resume diagnostic, and a success names the thread left running.
func slotOperationOutcome(value any, err error) couchmessage.ReceiptOutcome {
	var refusal *couchcore.SlotOperationError
	var recoverRefusal *couchcore.RecoverRefusal
	var reapRefusal *couchcore.ReapRefusal
	switch {
	case errors.As(err, &refusal):
		return couchmessage.ReceiptOutcome{Status: couchmessage.ReceiptRefused, Code: refusal.Code, Detail: refusal.Detail}
	case errors.As(err, &recoverRefusal):
		return couchmessage.ReceiptOutcome{Status: couchmessage.ReceiptRefused, Code: recoverRefusal.Code, Detail: recoverRefusal.Detail}
	case errors.As(err, &reapRefusal):
		return couchmessage.ReceiptOutcome{Status: couchmessage.ReceiptRefused, Code: "reap-refused", Detail: reapRefusal.Detail}
	case err != nil:
		out := couchmessage.ReceiptOutcome{Status: couchmessage.ReceiptFailed, Detail: err.Error(), Diagnostic: string(couchcore.ResumeDiagnosticOf(err))}
		var unconfirmed *couchcore.ReloadUnconfirmed
		if errors.As(err, &unconfirmed) {
			out.Code = "unconfirmed"
		}
		// A typed partial outcome (a relaunch that parked but did not
		// resume) and the admission note survive the failure (pair#421).
		if coded, ok := value.(interface{ ReceiptCode() string }); ok {
			out.Code = coded.ReceiptCode()
		}
		if warned, ok := value.(interface{ Warning() string }); ok {
			out.Warning = warned.Warning()
		}
		return out
	}
	out := couchmessage.ReceiptOutcome{Status: couchmessage.ReceiptSucceeded}
	if child, ok := value.(couchcore.StartedChild); ok {
		if started, _ := child.Started(); started.Record.Thread.Tag != "" {
			out.Tag = string(started.Record.Thread.Tag)
		}
	}
	if reboot, ok := value.(couchcore.RebootResult); ok {
		out.Archived = string(reboot.Archived.Tag)
	}
	if warned, ok := value.(interface{ Warning() string }); ok {
		out.Warning = warned.Warning()
	}
	return out
}

// notedResult carries an admission note (pair#421: freshness, an override
// used) to the receipt next to the operation's own result. It forwards
// Started and Warning, which slotOperationOutcome reads.
type notedResult struct {
	value any
	note  string
}

func (n notedResult) Started() (couchcore.StartResult, bool) {
	if child, ok := n.value.(couchcore.StartedChild); ok {
		return child.Started()
	}
	return couchcore.StartResult{}, false
}

func (n notedResult) ReceiptCode() string {
	if coded, ok := n.value.(interface{ ReceiptCode() string }); ok {
		return coded.ReceiptCode()
	}
	return ""
}

func (n notedResult) Warning() string {
	var parts []string
	if warned, ok := n.value.(interface{ Warning() string }); ok && warned.Warning() != "" {
		parts = append(parts, warned.Warning())
	}
	if n.note != "" {
		parts = append(parts, n.note)
	}
	return strings.Join(parts, "; ")
}
