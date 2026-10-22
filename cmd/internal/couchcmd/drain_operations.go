package couchcmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// drainNoticeReference names the latest notice without owning another mutable
// copy of its result. Only drainMu changes the per-slot reference.
type drainNoticeReference struct {
	key receiptKey
	id  string
}

func drainIntentDigest(intent couchmessage.DrainRequest, children []couchmessage.DrainChild) string {
	targets := make([]string, len(children))
	for i, child := range children {
		targets[i] = child.Workspace
	}
	raw, _ := json.Marshal(struct {
		Intent  couchmessage.DrainRequest
		Targets []string
	}{intent, targets})
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

// registerDrainLocked registers the complete operation before publishing flags.
// Lock order is drainMu then receipt mu. Workers cross the publication lock
// before broker admission, but the handler never waits for endpoint IO.
func (s *messageService) registerDrainLocked(ctx context.Context, caller couchmessage.Binding, r couchmessage.Request, targets []string, body string) (couchmessage.Response, bool) {
	ops := s.drainOps
	key := receiptKey{caller.Scope, caller.Tag, r.ID}
	ops.mu.Lock()
	defer ops.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return admissionRefusal("drain registration", err), false
	}
	sort.Strings(targets)
	children := make([]couchmessage.DrainChild, len(targets))
	for i, target := range targets {
		id, err := newRequestID()
		if err != nil {
			return couchmessage.Response{Code: "refused", Error: err.Error()}, false
		}
		children[i] = couchmessage.DrainChild{Workspace: target, ID: id}
	}
	op, err := couchmessage.NewDrainOperation(r.Drain.Action, children)
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}, false
	}
	event := couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptAdmit, ID: r.ID, Op: "drain", IntentDigest: drainIntentDigest(*r.Drain, children), Drain: &op, At: ops.now()}
	receipt, created, err := ops.reserveLocked(caller, event)
	if err != nil || !created {
		return receiptAdmissionResponse(receipt, err), false
	}
	tasks := make([]drainAdmissionTask, len(children))
	for i, child := range children {
		tasks[i] = drainAdmissionTask{target: child.Workspace, run: func(ctx context.Context) {
			// Registration, flags and console publication precede endpoint work.
			s.drainMu.Lock()
			s.drainMu.Unlock()
			s.admitDrainChild(ctx, caller, key, child, body)
		}}
	}
	if err := s.drainExecutor.enqueue(tasks); err != nil {
		delete(ops.receipts, key)
		return couchmessage.Response{Code: "refused", Error: err.Error()}, false
	}
	// No callbacks can take receipt mu before registration returns. Preserve the
	// existing operation reducer as the only parent phase-transition authority.
	receipt, _ = couchmessage.ApplyReceiptEvent(receipt, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptStart, ID: r.ID})
	receipt = s.refreshDrainOperation(receipt)
	ops.receipts[key] = receipt
	for _, child := range children {
		s.drainNotices[child.Workspace] = drainNoticeReference{key: key, id: child.ID}
	}
	return receiptAdmissionResponse(receipt, nil), true
}

func (s *messageService) admitDrainChild(ctx context.Context, caller couchmessage.Binding, key receiptKey, child couchmessage.DrainChild, body string) {
	var receipt couchmessage.Receipt
	err := ctx.Err()
	if err == nil {
		receipt, err = s.broker.SendSystem(ctx, caller, child.ID, couchmessage.Route{Target: child.Workspace}, body)
	}
	event := couchmessage.DrainChildEvent{ID: child.ID, Admission: true}
	if err != nil {
		event.Error = err.Error()
	} else {
		event.Receipt = &receipt
	}
	ops := s.drainOps
	ops.mu.Lock()
	defer ops.mu.Unlock()
	parent, held := ops.receipts[key]
	if !held || parent.Drain == nil {
		return
	}
	if next, err := couchmessage.ApplyDrainChild(*parent.Drain, event); err == nil {
		parent.Drain = &next
		ops.receipts[key] = s.refreshDrainOperation(parent)
	}
}

// refreshDrainOperation reads only broker-owned snapshots. Called with receipt
// mu held; neither callbacks nor wrapper queries acquire the publication lock.
func (s *messageService) refreshDrainOperation(parent couchmessage.OperationReceipt) couchmessage.OperationReceipt {
	if parent.Drain == nil || parent.Status.Terminal() {
		return parent
	}
	op := *parent.Drain
	events := make([]couchmessage.DrainChildEvent, 0, len(op.Children))
	for _, child := range op.Children {
		if child.Phase != couchmessage.DrainChildAdmitted {
			continue
		}
		receipt, known, active := s.broker.ReceiptSnapshot(child.ID)
		if active && receipt.Status != couchmessage.Submitted {
			continue
		}
		event := couchmessage.DrainChildEvent{ID: child.ID}
		if known {
			event.Receipt = &receipt
		} else {
			event.Error = "retained child receipt unavailable; inspect the recipient before re-notifying"
		}
		events = append(events, event)
	}
	if len(events) > 0 {
		next, err := couchmessage.ApplyDrainChildren(op, events)
		if err != nil {
			return parent
		}
		op = next
	}
	parent.Drain = &op
	status := couchmessage.DrainOperationStatus(op)
	if status.Terminal() {
		next, err := couchmessage.ApplyReceiptEvent(parent, couchmessage.ReceiptEvent{Kind: couchmessage.ReceiptFinish, ID: parent.ID, Outcome: couchmessage.ReceiptOutcome{Status: status}, At: s.drainOps.now()})
		if err == nil {
			return next
		}
	}
	return parent
}

// drainNoticeView carries only public correlation and diagnostic summaries.
type drainNoticeView struct{ state, id, operation, detail string }

// drainNoticeSnapshotLocked reads every referenced parent once. The caller
// holds drainMu; snapshotting does not perform endpoint IO or repair receipts.
func (s *messageService) drainNoticeSnapshotLocked() map[string]drainNoticeView {
	ops := s.drainOps
	result := make(map[string]drainNoticeView, len(s.drainFlags))
	if ops == nil {
		return result
	}
	ops.mu.Lock()
	defer ops.mu.Unlock()
	ops.sweepLocked()
	indexed := map[receiptKey]map[string]couchmessage.DrainChild{}
	for workspace := range s.drainFlags {
		ref, ok := s.drainNotices[workspace]
		view := drainNoticeView{state: "failed", id: ref.id, operation: ref.key.id, detail: "notice outcome unconfirmed: operation no longer retained"}
		if ok {
			children, cached := indexed[ref.key]
			if !cached {
				children = map[string]couchmessage.DrainChild{}
				if parent, held := ops.receipts[ref.key]; held && parent.Drain != nil {
					for _, child := range parent.Drain.Children {
						children[child.ID] = child
					}
				}
				indexed[ref.key] = children
			}
			if child, found := children[ref.id]; found {
				view.state, view.detail = couchmessage.DrainNoticeState(child), child.Detail
			}
		}
		result[workspace] = view
	}
	return result
}

// drainReplayLocked checks retained intent before rediscovering/capping a new
// cohort. A replay after membership changes cannot apply flags a second time.
func (s *messageService) drainReplayLocked(caller couchmessage.Binding, request couchmessage.Request) (couchmessage.Response, bool) {
	ops := s.drainOps
	ops.mu.Lock()
	defer ops.mu.Unlock()
	ops.sweepLocked()
	old, held := ops.receipts[receiptKey{caller.Scope, caller.Tag, request.ID}]
	if !held {
		return couchmessage.Response{}, false
	}
	if old.Drain == nil || old.IntentDigest != drainIntentDigest(*request.Drain, old.Drain.Children) {
		return receiptAdmissionResponse(old, couchmessage.ErrReceiptIDConflict), true
	}
	return receiptAdmissionResponse(old, nil), true
}
