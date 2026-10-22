package couchmessage

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// DrainChildPhase separates broker admission from confirmed composer delivery.
type DrainChildPhase string

const (
	DrainChildAwaitingAdmission DrainChildPhase = "awaiting-admission"
	DrainChildAdmitted          DrainChildPhase = "admitted"
	DrainChildDelivered         DrainChildPhase = "delivered"
	DrainChildFailed            DrainChildPhase = "failed"
)

// DrainChild records one preallocated notice identity. Detail is diagnostic only.
type DrainChild struct {
	Workspace     string
	ID            string
	Phase         DrainChildPhase
	ReceiptStatus Status
	Detail        string
}

// DrainOperation holds a bounded, immutable-by-reduction fan-out snapshot.
type DrainOperation struct {
	Action   string
	Children []DrainChild
}

// DrainChildEvent carries either the admission outcome or a later receipt read.
// Error and Receipt are mutually exclusive. Missing receipts explicitly fail
// unconfirmed rather than being mistaken for successful delivery.
type DrainChildEvent struct {
	ID        string
	Receipt   *Receipt
	Error     string
	Admission bool
}

// NewDrainOperation allocates every child before any notice can be admitted.
// It rejects overflow rather than silently omitting a member of the fleet.
func NewDrainOperation(action string, children []DrainChild) (DrainOperation, error) {
	if action != "start" && action != "undrain" {
		return DrainOperation{}, errors.New("invalid drain action")
	}
	if len(children) > MaxActors {
		return DrainOperation{}, errors.New("drain child count exceeds actor limit")
	}
	ids, workspaces := make(map[string]bool, len(children)), make(map[string]bool, len(children))
	for _, child := range children {
		raw, err := json.Marshal(child.Workspace)
		if !validMessageID(child.ID) || child.Workspace == "" || !utf8.ValidString(child.Workspace) || err != nil || len(raw) > MaxBindingBytes {
			return DrainOperation{}, errors.New("invalid drain child identity")
		}
		if ids[child.ID] || workspaces[child.Workspace] {
			return DrainOperation{}, errors.New("duplicate drain child identity")
		}
		if child.Phase != "" && child.Phase != DrainChildAwaitingAdmission || child.ReceiptStatus != "" || child.Detail != "" {
			return DrainOperation{}, errors.New("drain child is not fresh")
		}
		ids[child.ID], workspaces[child.Workspace] = true, true
	}
	op := DrainOperation{Action: action, Children: append([]DrainChild(nil), children...)}
	for i := range op.Children {
		op.Children[i].Phase = DrainChildAwaitingAdmission
	}
	return op, nil
}

// ApplyDrainChild is the single-event convenience for ApplyDrainChildren.
func ApplyDrainChild(op DrainOperation, event DrainChildEvent) (DrainOperation, error) {
	return ApplyDrainChildren(op, []DrainChildEvent{event})
}

// ApplyDrainChildren reduces one receipt snapshot in linear time, cloning the
// children once. Each child may appear at most once. Any invalid event restores
// the entire original snapshot in that detached allocation.
func ApplyDrainChildren(op DrainOperation, events []DrainChildEvent) (DrainOperation, error) {
	next := DrainOperation{Action: op.Action, Children: append([]DrainChild(nil), op.Children...)}
	indices := make(map[string]int, len(next.Children))
	for i, child := range next.Children {
		if _, exists := indices[child.ID]; exists {
			return next, errors.New("duplicate drain child identity")
		}
		indices[child.ID] = i
	}
	seen := make(map[string]bool, len(events))
	for _, event := range events {
		index, exists := indices[event.ID]
		var err error
		switch {
		case !exists:
			err = errors.New("unknown drain child")
		case seen[event.ID]:
			err = errors.New("duplicate drain child event")
		default:
			seen[event.ID] = true
			next.Children[index], err = applyDrainChildEvent(next.Children[index], event)
		}
		if err != nil {
			copy(next.Children, op.Children)
			return next, err
		}
	}
	return next, nil
}

func applyDrainChildEvent(child DrainChild, event DrainChildEvent) (DrainChild, error) {
	if event.Admission && child.Phase != DrainChildAwaitingAdmission || !event.Admission && child.Phase != DrainChildAdmitted {
		return child, errors.New("stale drain child event")
	}
	if event.Receipt != nil {
		if event.Error != "" {
			return child, errors.New("contradictory drain receipt and error")
		}
		if event.Receipt.Message.ID != event.ID {
			return child, errors.New("drain receipt identity mismatch")
		}
		switch event.Receipt.Status {
		case Queued, Delivering, Held, Submitted, Expired, Cancelled, Indeterminate, Refused:
		default:
			return child, errors.New("invalid drain receipt status")
		}
	}
	if event.Error != "" || event.Receipt == nil {
		child.Phase = DrainChildFailed
		child.Detail = drainDetail("notice delivery unconfirmed: " + event.Error)
		if event.Error == "" {
			child.Detail = "notice delivery unconfirmed: receipt unavailable"
		}
		return child, nil
	}
	receipt := event.Receipt
	child.ReceiptStatus = receipt.Status
	child.Detail = drainDetail(receipt.Detail)
	switch {
	case receipt.Status == Submitted:
		child.Phase = DrainChildDelivered
	case receipt.Status.Terminal():
		child.Phase = DrainChildFailed
		if receipt.Status == Indeterminate {
			child.Detail = drainDetail("notice delivery unconfirmed: " + receipt.Detail)
		} else if child.Detail == "" {
			child.Detail = fmt.Sprintf("notice delivery failed: %s", receipt.Status)
		}
	default:
		child.Phase = DrainChildAdmitted
	}
	return child, nil
}

// DrainOperationStatus stays running while any child lacks a terminal outcome.
func DrainOperationStatus(op DrainOperation) ReceiptStatus {
	failed := false
	for _, child := range op.Children {
		switch child.Phase {
		case DrainChildFailed:
			failed = true
		case DrainChildDelivered:
		default:
			return ReceiptRunning
		}
	}
	if failed {
		return ReceiptFailed
	}
	return ReceiptSucceeded
}

// DrainNoticeState is the compact public delivery projection, never a flag.
func DrainNoticeState(child DrainChild) string {
	switch child.Phase {
	case DrainChildDelivered:
		return "delivered"
	case DrainChildFailed:
		return "failed"
	default:
		return "pending"
	}
}

// drainDetail bounds the actual JSON string, including escapes and quotes.
// Diagnostic replacement and explicit truncation never alter child identities.
func drainDetail(detail string) string {
	detail = strings.ToValidUTF8(detail, "�")
	raw, _ := json.Marshal(detail)
	if len(raw) <= 256 {
		return detail
	}
	// Each rune contributes at least one encoded byte. Only the bounded prefix
	// can fit, so avoid repeatedly marshaling an arbitrarily large diagnostic.
	var prefix strings.Builder
	used := 2 + len("…")
	for _, r := range detail {
		encoded, _ := json.Marshal(string(r))
		size := len(encoded) - 2
		if used+size > 256 {
			break
		}
		prefix.WriteRune(r)
		used += size
	}
	return prefix.String() + "…"
}
