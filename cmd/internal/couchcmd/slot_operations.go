package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// slotOperationRunner enqueues one slot operation on the console's queue
// (couchtty.Console.EnqueueRemoteOperation with Couch.PrepareSlotOperation).
// When it returns an error it never calls started, progress or finished.
type slotOperationRunner func(key, op, target string, opts couchcore.LiveRestartOptions, started func(), progress couchcore.OperationProgress, finished func(any, error)) error

// maxSlotOperationReceipts bounds the receipts held at once (about 1 KiB each);
// the next admission is refused overloaded.
const maxSlotOperationReceipts = 64

// slotOperations is the socket side of resume/reboot (pair#367 M2): it admits
// a request from a verified caller, holds its receipt in memory, and answers
// operation-status. One mutex owns the receipt map; admit and status (socket
// handlers), started (the queue runner) and finished (the console loop) all
// take it, and none waits on the queue or the console while holding it.
type slotOperations struct {
	fleetRoot     string
	reference     func(context.Context, string) (string, error)
	recovery      func(string, string, couchmessage.Request, func(context.Context) error, func(), couchcore.OperationProgress, func(any, error)) error
	run           slotOperationRunner
	preparePark   func(context.Context, string, couchcore.RepositoryName, func(context.Context) error) (slotOperationRunner, error)
	authorizePark func(context.Context, couchmessage.Binding, couchcore.HubSlot) error
	guardedRun    func(string, string, string, couchcore.LiveRestartOptions, func(context.Context) error, func(), couchcore.OperationProgress, func(any, error)) error
	preview       func(context.Context, string) (couchcore.RecoveryPreview, error)
	candidate     func(string, string, string, func(), couchcore.OperationProgress, func(any, error), ...func(context.Context) error) error
	names         func(context.Context) ([]couchcore.RepositoryName, error)
	now           func() time.Time

	mu       sync.Mutex
	receipts map[receiptKey]couchmessage.OperationReceipt
}

// receiptKey scopes a receipt to the slot that admitted it: another caller
// never reads it.
type receiptKey struct{ scope, tag, id string }

// switchStatus is only called after proving the requesting live binding is the
// supervisor hub. Ordinary status remains scoped to the admitting slot.
func (s *slotOperations) switchStatus(id string, filters ...func(couchmessage.OperationReceipt) bool) (couchmessage.Response, bool) {
	s.mu.Lock()
	s.sweepLocked()
	var candidates []couchmessage.OperationReceipt
	for _, receipt := range s.receipts {
		if receipt.ID == id && receipt.Op == "switch-agent" {
			candidates = append(candidates, receipt)
		}
	}
	s.mu.Unlock()
	// Ownership resolution may read the catalog; never hold the receipt mutex
	// across that IO or a queued operation's completion would be blocked.
	for _, receipt := range candidates {
		allowed := true
		for _, filter := range filters {
			if !filter(receipt) {
				allowed = false
				break
			}
		}
		if allowed {
			return couchmessage.Response{Code: "ok", Operation: &receipt}, true
		}
	}
	return couchmessage.Response{}, false
}

func newSlotOperations(run slotOperationRunner, names func(context.Context) ([]couchcore.RepositoryName, error), now func() time.Time) *slotOperations {
	return &slotOperations{run: run, names: names, now: now, receipts: map[receiptKey]couchmessage.OperationReceipt{}}
}

// handle answers one validated request from a verified, current caller.
func (s *slotOperations) handle(ctx context.Context, caller couchmessage.Binding, r couchmessage.Request, guards ...func(context.Context) error) couchmessage.Response {
	if ctx.Err() != nil {
		return admissionRefusal("queue names", ctx.Err())
	}
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
	if err := slotOperationConfirmation(r); err != nil {
		return couchmessage.Response{Code: "confirmation-required", Error: err.Error()}
	}
	var parkRunner slotOperationRunner
	var queueKeys []string
	if r.Op == couchcore.OpRecoverPark {
		// Bind outside the receipt mutex. Existing IDs must retain their receipt
		// even after the original park or thread has disappeared.
		s.mu.Lock()
		s.sweepLocked()
		_, held := s.receipts[key]
		s.mu.Unlock()
		if !held {
			var err error
			var guard func(context.Context) error
			if len(guards) > 0 {
				guard = guards[0]
			}
			queueKey, repository, resolveErr := s.queueTarget(ctx, r.Target)
			if ctx.Err() != nil {
				return admissionRefusal("queue names", ctx.Err())
			}
			if resolveErr != nil {
				return couchmessage.Response{Code: couchcore.SlotOpUnknownSlot, Error: resolveErr.Error()}
			}
			queueKeys = []string{queueKey}
			ref, _, _ := couchcore.ParseWorkspaceReference(r.Target) // queueTarget validated it.
			if s.authorizePark != nil {
				if err := s.authorizePark(ctx, caller, couchcore.HubSlot{Repository: repository.Key, Number: ref.Number}); err != nil {
					return couchmessage.Response{Code: "refused", Error: err.Error()}
				}
			}
			if s.preparePark != nil {
				parkRunner, err = s.preparePark(ctx, r.Target, repository, guard)
				if err != nil {
					return couchmessage.Response{Code: "refused", Error: err.Error()}
				}
			}
		}
	}
	return s.admit(ctx, caller, r, func(key string, started func(), progress couchcore.OperationProgress, finished func(any, error)) error {
		if r.Op == couchcore.OpRecoverPark && s.preparePark != nil && parkRunner == nil {
			return errors.New("recovery receipt expired; inspect the current thread before recovering")
		}
		if parkRunner != nil {
			return parkRunner(key, r.Op, r.Target, couchcore.LiveRestartOptions{}, started, progress, finished)
		}
		if s.recovery != nil && (r.Op == "resume" || r.Op == "reboot" || r.Op == "recovery-confirmation") {
			var guard func(context.Context) error
			if len(guards) > 0 {
				guard = guards[0]
			}
			return s.recovery(key, recoveryActor(caller), r, guard, started, progress, finished)
		}
		if r.CandidateKey != "" {
			if s.candidate == nil {
				return errors.New("candidate resume unavailable")
			}
			return s.candidate(key, r.Target, r.CandidateKey, started, progress, finished, guards...)
		}
		if len(guards) > 0 {
			if s.guardedRun == nil {
				return errors.New("hub-aware operation queue unavailable")
			}
			return s.guardedRun(key, r.Op, r.Target, couchcore.LiveRestartOptions{SameBinary: r.SameBinary, ForceUnknown: r.ForceUnknown}, guards[0], started, progress, finished)
		}
		return s.run(key, r.Op, r.Target, couchcore.LiveRestartOptions{SameBinary: r.SameBinary, ForceUnknown: r.ForceUnknown}, started, progress, finished)
	}, queueKeys...)
}

// admit shares the receipt state machine and queue admission across slot operations.
func (s *slotOperations) admit(ctx context.Context, caller couchmessage.Binding, r couchmessage.Request,
	run func(string, func(), couchcore.OperationProgress, func(any, error)) error, queueKeys ...string) couchmessage.Response {
	key := receiptKey{caller.Scope, caller.Tag, r.ID}
	if err := slotOperationConfirmation(r); err != nil {
		return couchmessage.Response{Code: "confirmation-required", Error: err.Error()}
	}
	event := couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptAdmit, IntentDigest: operationIntentDigest(r), CandidateKey: r.CandidateKey, RecoveryConfirmation: r.RecoveryConfirmation, ID: r.ID, Op: r.Op, Target: r.Target, Agent: r.Agent, ToHub: r.ToHub, Persist: r.Persist, At: s.now()}
	s.mu.Lock()
	s.sweepLocked()
	existing, held := s.receipts[key]
	if held {
		receipt, err := couchmessage.ApplyReceiptEvent(existing, event)
		s.mu.Unlock()
		return receiptAdmissionResponse(receipt, err)
	}
	s.mu.Unlock()
	var queueKey string
	var err error
	if len(queueKeys) > 0 {
		queueKey = queueKeys[0]
	} else {
		queueKey, err = s.queueKey(ctx, r.Target)
	}
	if err != nil {
		if ctx.Err() != nil {
			return admissionRefusal("queue names", ctx.Err())
		}
		return couchmessage.Response{Code: couchcore.SlotOpUnknownSlot, Error: err.Error()}
	}
	if ctx.Err() != nil {
		return admissionRefusal("queue names", ctx.Err())
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()
	existing, held = s.receipts[key]
	receipt, err := couchmessage.ApplyReceiptEvent(existing, event)
	switch {
	case err != nil || held:
		return receiptAdmissionResponse(receipt, err)
	case len(s.receipts) >= maxSlotOperationReceipts:
		return couchmessage.Response{Code: "overloaded", Error: "too many slot operations held; read the report and retry later"}
	}
	if ctx.Err() != nil {
		return admissionRefusal("enqueue", ctx.Err())
	}
	s.receipts[key] = receipt
	// Enqueue never blocks, so holding the lock is safe; a job that starts at
	// once waits in started until this admission returns.
	err = run(queueKey, func() { s.apply(key, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptStart, ID: r.ID}) },
		func(stage string) {
			s.apply(key, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptProgress, ID: r.ID, Stage: stage, At: s.now()}, receipt.Admitted)
		},
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

func receiptAdmissionResponse(receipt couchmessage.OperationReceipt, err error) couchmessage.Response {
	if err != nil {
		code := "invalid-request"
		if errors.Is(err, couchmessage.ErrReceiptIDConflict) {
			code = "id-conflict"
		}
		return couchmessage.Response{Code: code, Error: err.Error()}
	}
	return couchmessage.Response{Code: "accepted", Operation: &receipt}
}

// apply moves one held receipt; an event the table refuses (a late or
// repeated finish) leaves it unchanged. Progress also binds the original
// admission, so a retired request ID cannot update a later incarnation.
func (s *slotOperations) apply(key receiptKey, e couchmessage.ReceiptEvent, admission ...time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	receipt, held := s.receipts[key]
	if !held || len(admission) > 0 && !receipt.Admitted.Equal(admission[0]) {
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
// local sibling check and store read only: no git, no sdlc.
func (s *slotOperations) queueKey(ctx context.Context, target string) (string, error) {
	key, _, err := s.queueTarget(ctx, target)
	return key, err
}

func (s *slotOperations) queueTarget(ctx context.Context, target string) (string, couchcore.RepositoryName, error) {
	ref, _, err := couchcore.ParseWorkspaceReference(target)
	if err != nil {
		return "", couchcore.RepositoryName{}, err
	}
	sibling, exists := couchcore.RepositorySiblingPrimary(ref.Repo, s.fleetRoot)
	repo := couchcore.RepositoryName{Key: sibling, Dir: ref.Repo}
	if !exists {
		names, namesErr := s.names(ctx)
		if namesErr != nil {
			return "", couchcore.RepositoryName{}, namesErr
		}
		repo, _, err = couchcore.ResolveRepositoryName(ref.Repo, names)
		if err != nil {
			return "", couchcore.RepositoryName{}, err
		}
	}
	return "remote\x00" + repo.Key + ":" + strconv.Itoa(ref.Number), repo, nil
}

func slotOperationConfirmation(r couchmessage.Request) error {
	if confirms, _ := couchcore.OperationConfirms(r.Op); confirms && !r.Confirmed {
		return errors.New(r.Op + " requires --confirm")
	}
	return nil
}

// slotOperationOutcome maps a finished job to its receipt outcome: a typed
// slot refusal is refused with its code, any other error failed with its
// resume diagnostic, and a success names the thread left running.
func slotOperationOutcome(value any, err error) couchmessage.ReceiptOutcome {
	var notes []string
	var nonce string
	for {
		switch wrapped := value.(type) {
		case lifecycleRecordedOutcome:
			nonce = wrapped.nonce
			notes = append(notes, wrapped.warning)
			value = wrapped.value
		case notedResult:
			notes = append(notes, wrapped.note)
			value = wrapped.value
		default:
			goto classify
		}
	}
classify:
	outcome := classifySlotOperationOutcome(value, err)
	outcome.FailureNonce = couchcore.FailureNonce(err)
	if outcome.FailureNonce == "" {
		outcome.FailureNonce = nonce
	}
	for _, note := range notes {
		if note != "" {
			if outcome.Warning != "" {
				outcome.Warning += "; "
			}
			outcome.Warning += note
		}
	}
	return outcome
}

// classifySlotOperationOutcome retains the typed operation result independently
// of diagnostic correlation, which applies uniformly after classification.
func classifySlotOperationOutcome(value any, err error) couchmessage.ReceiptOutcome {
	if result, ok := value.(agentdrive.Result); ok && err == nil {
		out := couchmessage.ReceiptOutcome{Drive: &result, Detail: result.Detail, Status: couchmessage.ReceiptFailed}
		switch result.Outcome {
		case agentdrive.OutcomeConfirmed:
			out.Status = couchmessage.ReceiptSucceeded
		case agentdrive.OutcomeRefused:
			out.Status = couchmessage.ReceiptRefused
		case agentdrive.OutcomeUnconfirmed:
			out.Code = "unconfirmed"
		}
		return out
	}
	var confirmation *couchcore.RecoveryConfirmationRequired
	if errors.As(err, &confirmation) {
		return couchmessage.ReceiptOutcome{Status: couchmessage.ReceiptRefused, Code: "confirmation-required", Detail: confirmation.Error(), Confirmation: &confirmation.Offer}
	}
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
		out := couchmessage.ReceiptOutcome{Status: couchmessage.ReceiptFailed, Detail: couchcore.OperatorError(err), Diagnostic: string(couchcore.ResumeDiagnosticOf(err))}
		var recoveryUnconfirmed *couchcore.RecoveryLaunchUnconfirmed
		var cleanupUnconfirmed *couchcore.SessionCleanupUnconfirmed
		var unconfirmed *couchcore.ReloadUnconfirmed
		var unready *couchcore.RestartUnready
		switch {
		case errors.As(err, &cleanupUnconfirmed):
			out.Code = "unconfirmed"
		case errors.As(err, &recoveryUnconfirmed):
			out.Code = "unconfirmed"
		case errors.As(err, &unconfirmed):
			out.Code = "unconfirmed"
		case errors.As(err, &unready):
			out.Code = "unready"
		}
		// A typed partial outcome (a relaunch that parked but did not
		// resume) and the admission note survive the failure (pair#421).
		// A completed relaunch has no partial code; keep the error's (unready).
		if coded, ok := value.(interface{ ReceiptCode() string }); ok && coded.ReceiptCode() != "" {
			out.Code = coded.ReceiptCode()
		}
		if warned, ok := value.(interface{ Warning() string }); ok {
			out.Warning = warned.Warning()
		}
		return out
	}
	out := couchmessage.ReceiptOutcome{Status: couchmessage.ReceiptSucceeded}
	if coded, ok := value.(interface{ ReceiptCode() string }); ok {
		out.Code = coded.ReceiptCode()
	}
	if child, ok := value.(couchcore.StartedChild); ok {
		if started, _ := child.Started(); started.Record.Thread.Tag != "" {
			out.Tag = string(started.Record.Thread.Tag)
		}
	}
	if tagged, ok := value.(interface{ ReceiptTag() string }); ok && out.Tag == "" {
		out.Tag = tagged.ReceiptTag()
	}
	if located, ok := value.(interface{ ReceiptPath() string }); ok {
		out.Path = located.ReceiptPath()
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

func (n notedResult) ReceiptTag() string {
	if tagged, ok := n.value.(interface{ ReceiptTag() string }); ok {
		return tagged.ReceiptTag()
	}
	return ""
}

func (n notedResult) ReceiptPath() string {
	if located, ok := n.value.(interface{ ReceiptPath() string }); ok {
		return located.ReceiptPath()
	}
	return ""
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

// recoveryActor uses only the binding authenticated by the live caller gate.
func recoveryActor(caller couchmessage.Binding) string {
	b, _ := json.Marshal(caller)
	return "hub:" + string(b)
}
