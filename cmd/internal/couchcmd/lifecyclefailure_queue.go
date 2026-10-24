package couchcmd

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

type lifecycleRemoteQueue struct {
	couch    *couchcore.Couch
	rows     func() []couchcore.ActionableThreadSummary
	enqueue  func(string, string, func(context.Context) (couchcore.OperationCall, error), func(), func(any, error)) error
	notify   func(couchcore.ThreadAddress, string)
	lifetime context.Context
}

func newLifecycleRemoteQueue(console *couchtty.Console, c *couchcore.Couch) *lifecycleRemoteQueue {
	return &lifecycleRemoteQueue{couch: c, lifetime: console.OperationLifetime(), rows: func() []couchcore.ActionableThreadSummary {
		snapshot := console.ResourceInventory()
		if !snapshot.InventoryKnown {
			return nil
		}
		return snapshot.Rows
	}, enqueue: console.EnqueueRemoteOperation, notify: func(thread couchcore.ThreadAddress, text string) {
		console.Warn(couchtty.Notice{Kind: "status", Thread: thread, Body: text})
	}}
}

// Enqueue gives preparation, execution, adoption and readiness one terminal
// owner. The underlying console queue retains its ordinary scheduling contract.
func (q *lifecycleRemoteQueue) Enqueue(key, op string, prepare func(context.Context) (couchcore.OperationCall, error), started func(), finished func(any, error), awaits ...*func(context.Context) error) error {
	return q.EnqueueForReference(key, op, strings.TrimPrefix(key, "remote\x00"), prepare, started, finished, awaits...)
}

// EnqueueForReference retains the dispatched reference even without a cache hit.
// Diagnostic evidence never changes operation admission or target resolution.
func (q *lifecycleRemoteQueue) EnqueueForReference(key, op, reference string, prepare func(context.Context) (couchcore.OperationCall, error), started func(), finished func(any, error), awaits ...*func(context.Context) error) error {
	target, _ := lifecycleFailureQueueTarget(q.rows(), key)
	target.Target = reference
	attempt, err := q.couch.OperationAttemptForCall(couchcore.OperationCall{Name: op, FailureTarget: &target})
	if err != nil {
		return err
	}
	terminal := func(value any, err error) {
		if attempt != nil {
			value, err = attempt.FinishContext(q.ownerContext(), value, err)
			warning := attempt.DiagnosticWarning()
			if warning != "" {
				if q.notify != nil {
					q.notify(target.Address, warning)
				}
				value = lifecycleRecordedOutcome{value: value, nonce: attempt.RecordedFailureNonce(), warning: warning}
			}
		}
		finished(value, err)
	}
	completion := terminal
	if len(awaits) != 0 && awaits[0] != nil {
		completion = awaitReadiness(q.ownerContext(), awaits[0], terminal)
	}
	err = q.enqueue(key, op, func(ctx context.Context) (couchcore.OperationCall, error) {
		if attempt != nil {
			ctx = couchcore.WithOperationAttempt(ctx, attempt)
		}
		call, err := prepare(ctx)
		if attempt != nil {
			attempt.ObserveCall(call)
		}
		if err == nil && attempt != nil {
			if call.Context == nil {
				call.Context = ctx
			} else {
				call.Context = couchcore.WithOperationAttempt(call.Context, attempt)
			}
		}
		return call, err
	}, started, completion)
	return err
}

// Queue is the ordinary console callback shape, with no readiness extension.
func (q *lifecycleRemoteQueue) Queue(key, op string, prepare func(context.Context) (couchcore.OperationCall, error), started func(), finished func(any, error)) error {
	return q.Enqueue(key, op, prepare, started, finished)
}

// lifecycleRecordedOutcome is receipt-only metadata added after adoption and
// readiness; the classifier unwraps it before inspecting the original result.
type lifecycleRecordedOutcome struct {
	value          any
	nonce, warning string
}

// lifecycleFailureQueueTarget consumes the canonical queue identity and cached
// rows. It never resolves a mutable alias, Git checkout or replacement at finish.
func lifecycleFailureQueueTarget(rows []couchcore.ActionableThreadSummary, key string) (couchcore.LifecycleFailureTarget, bool) {
	var target couchcore.LifecycleFailureTarget
	if strings.HasPrefix(key, "hub-start:") {
		root := strings.TrimPrefix(key, "hub-start:")
		return couchcore.LifecycleFailureTarget{StartingPath: root}, filepath.IsAbs(root) && filepath.Clean(root) == root
	}
	if !strings.HasPrefix(key, "remote\x00") {
		return target, false
	}
	address := strings.TrimPrefix(key, "remote\x00")
	separator := strings.LastIndexByte(address, ':')
	if separator < 0 {
		return target, false
	}
	primary := address[:separator]
	number, err := strconv.Atoi(address[separator+1:])
	if err != nil || number < 0 || !filepath.IsAbs(primary) {
		return target, false
	}
	matches := 0
	for _, row := range rows {
		matched := number == 0 && filepath.Clean(row.StartingPath) == primary
		if number > 0 && row.Target.Kind == couchcore.ThreadTargetSlot {
			matched = row.Target.Slot.PrimaryRoot == primary && row.Target.Slot.Number == number
		}
		if matched {
			target = couchcore.LifecycleFailureTargetForRow(row)
			matches++
		}
	}
	return target, matches == 1
}

// ownerContext is only the completion worker's lifetime, never effect authority.
func (q *lifecycleRemoteQueue) ownerContext() context.Context {
	if q.lifetime != nil {
		return q.lifetime
	}
	return context.Background()
}
