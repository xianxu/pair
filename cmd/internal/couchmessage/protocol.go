package couchmessage

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Request has closed per-operation shapes. Wrapper lifecycle operations use a
// full binding and must pass the supervisor adapter's live identity check.
type Request struct {
	Op, Scope, Tag, Session, Nonce, ID, Target, Body string
	// Agent narrows a send to slots running that agent; send only.
	Agent string `json:",omitempty"`
	// Confirmed declares the operator's confirmation for an operation whose
	// declaration requires one (reboot); resume and reboot only.
	Confirmed bool `json:",omitempty"`
	// SameBinary and ForceUnknown are relaunch/reload-context overrides
	// (pair#421); each is recorded in the receipt when it changed the outcome.
	SameBinary   bool `json:",omitempty"`
	ForceUnknown bool `json:",omitempty"`
	// TailScope and TailTag name the thread whose in-memory terminal tail
	// reads, Lines how much of it (pair#425); tail only. A tail may instead
	// name its slot in Target (pair#429), resolved as a send's exact slot is.
	TailScope string `json:",omitempty"`
	TailTag   string `json:",omitempty"`
	Lines     int    `json:",omitempty"`
	Binding   *Binding
}

type Response struct {
	Code, Error string
	Receipt     *Receipt
	Actors      []Candidate
	// Operation is a resume/reboot request's receipt (pair#367 M2).
	Operation *OperationReceipt `json:",omitempty"`
	// Broadcast answers broadcast-status (pair#413); nil means no broadcast.
	Broadcast *BroadcastStatus `json:",omitempty"`
	// Tail answers tail (pair#425); TailThread names whose it is (pair#429).
	Tail       *Tail       `json:",omitempty"`
	TailThread *TailThread `json:",omitempty"`
}

// BroadcastStatus is the running console's broadcast, as `couch
// --broadcast-list` prints it (pair#413). It has no field for the link or its
// token -- the token is the broadcast's only credential, so a listing can
// never print it (operator decision).
type BroadcastStatus struct {
	State     string    `json:"state"`
	StartedAt time.Time `json:"started_at,omitzero"`
	Mode      string    `json:"mode,omitempty"`
	Viewers   int       `json:"viewers"`
}

func validMessageID(id string) bool {
	return id != "" && len(id) <= 128 && utf8.ValidString(id) && strings.IndexFunc(id, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) < 0
}

func ValidateRequest(r Request) error {
	// Caller identity and target fields cannot exceed a complete binding's budget.
	for _, field := range []string{r.Op, r.Scope, r.Tag, r.Session, r.Nonce, r.Target, r.Agent, r.TailScope, r.TailTag} {
		if len(field) > MaxBindingBytes {
			return errors.New("request identity exceeds limit")
		}
	}

	if r.Op == "register" || r.Op == "operator-submit" {
		if r.Binding == nil || r.Scope != "" || r.Tag != "" || r.Session != "" || r.Nonce != "" || r.ID != "" || r.Target != "" || r.Body != "" || r.Agent != "" {
			return errors.New("wrapper operation requires only an exact binding")
		}
		return r.Binding.Validate()
	}
	if r.Op == "tail" || r.TailScope != "" || r.TailTag != "" || r.Lines != 0 {
		// The operator's peek from any shell (pair#425): like
		// broadcast-status, no conversation identity, and a read-only answer.
		if r.Op != "tail" {
			return errors.New("a tail target applies only to tail")
		}
		if r.Binding != nil || r.Scope != "" || r.Tag != "" || r.Session != "" || r.Nonce != "" || r.ID != "" || r.Body != "" || r.Agent != "" || r.Confirmed || r.SameBinary || r.ForceUnknown {
			return errors.New("tail takes only a thread and a line count")
		}
		byThread := r.TailScope != "" && r.TailTag != ""
		if r.Target != "" {
			if r.TailScope != "" || r.TailTag != "" {
				return errors.New("tail names its thread by scope and tag or by slot, not both")
			}
			if _, _, err := parseSlot(r.Target); err != nil {
				return errors.New("tail requires an exact slot (repo:N)")
			}
		} else if !byThread {
			return errors.New("tail requires a thread scope and tag")
		}
		return ValidTailLines(r.Lines)
	}
	if r.Op == "broadcast-status" {
		// The operator's query from any shell (pair#413): no conversation
		// identity, no message fields. The socket's owner-only store is the
		// access control, and the answer is read-only.
		if r.Binding != nil || r.Scope != "" || r.Tag != "" || r.Session != "" || r.Nonce != "" || r.ID != "" || r.Target != "" || r.Body != "" || r.Agent != "" || r.Confirmed {
			return errors.New("broadcast-status takes no fields")
		}
		return nil
	}
	if r.Binding != nil || r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "" {
		return errors.New("operation requires the calling conversation identity")
	}
	if r.Agent != "" && (r.Op != "send" || !validFamily(r.Agent)) {
		return errors.New("an agent filter applies only to send and must be a plain agent name")
	}
	if r.Confirmed && !couchcore.IsSlotOperation(r.Op) {
		return errors.New("a confirmation applies only to slot operations")
	}
	if r.SameBinary && !couchcore.SlotOperationTakesSameBinary(r.Op) {
		return errors.New("--same-binary applies only to relaunch")
	}
	if r.ForceUnknown && !couchcore.SlotOperationTakesForceUnknown(r.Op) {
		return errors.New("--force-unknown applies only to relaunch and reload-context")
	}
	switch {
	case couchcore.IsSlotOperation(r.Op):
		// One exact slot: a primitive acts on one slot, never a family.
		if !validMessageID(r.ID) || r.Body != "" {
			return errors.New(r.Op + " requires only a request ID and an exact slot")
		}
		// parseSlot refuses a family ("pair") as well as a malformed slot.
		if _, _, err := parseSlot(r.Target); err != nil {
			return err
		}
	case r.Op == "operation-status":
		if !validMessageID(r.ID) || r.Target != "" || r.Body != "" {
			return errors.New("operation-status requires only a request ID")
		}
	case r.Op == "actors":
		if r.ID != "" || r.Target != "" || r.Body != "" {
			return errors.New("actors takes no message fields")
		}
	case r.Op == "status":
		if !validMessageID(r.ID) || r.Target != "" || r.Body != "" {
			return errors.New("status requires only a message ID")
		}
	case r.Op == "send":
		if !validMessageID(r.ID) {
			return errors.New("invalid message ID")
		}
		if strings.Contains(r.Target, ":") {
			if _, _, err := parseSlot(r.Target); err != nil {
				return err
			}
		} else if !validFamily(r.Target) {
			return ErrInvalidTarget
		}
		return ValidateBody(r.Body)
	default:
		return fmt.Errorf("unknown message operation %q", r.Op)
	}
	return nil
}

// Handle is the broker protocol adapter for sender requests. Wrappers
// register through their lifecycle session (#365); the old per-second
// register and operator-submit requests are answered unsupported without any
// verification work, so a wrapper from an older binary costs nothing.
func Handle(ctx context.Context, b *Broker, r Request) Response {
	if err := ValidateRequest(r); err != nil {
		return Response{Code: "invalid-request", Error: err.Error()}
	}
	if err := ctx.Err(); err != nil {
		return protocolError(err, false)
	}
	if r.Op == "register" || r.Op == "operator-submit" {
		return Response{Code: "unsupported", Error: "this wrapper predates lifecycle sessions; relaunch the slot to receive messages"}
	}
	caller, err := b.Caller(r.Scope, r.Tag, r.Session, r.Nonce)
	if err != nil {
		return protocolError(err, false)
	}
	switch r.Op {
	case "actors":
		actors, err := b.Actors(ctx, caller)
		if err != nil {
			return protocolError(err, false)
		}
		return Response{Code: "ok", Actors: actors}
	case "status":
		receipt, err := b.StatusContext(ctx, caller, r.ID)
		if err != nil {
			return protocolError(err, false)
		}
		return Response{Code: "ok", Receipt: &receipt}
	case "send":
		receipt, err := b.Send(ctx, caller, r.ID, Route{Target: r.Target, Agent: r.Agent}, r.Body)
		if err != nil {
			return protocolError(err, !strings.Contains(r.Target, ":"))
		}
		return Response{Code: "accepted", Receipt: &receipt}
	}
	return Response{Code: "invalid-request", Error: "unknown message operation"}
}

func protocolError(err error, family bool) Response {
	code := "refused"
	switch {
	case errors.Is(err, ErrNoRecipient) || (family && errors.Is(err, ErrUnavailable)):
		code = "not-dispatched"
	case errors.Is(err, ErrUnavailable):
		code = "unavailable"
	case errors.Is(err, ErrRecipientBusy):
		code = "busy"
	case errors.Is(err, ErrBudgetExhausted):
		code = "breaker-exhausted"
	case errors.Is(err, ErrUnsupported):
		code = "unsupported"
	case errors.Is(err, ErrAmbiguous):
		code = "ambiguous"
	case errors.Is(err, ErrInvalidTarget):
		code = "invalid-target"
	case errors.Is(err, ErrUncertain) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		code = "uncertain"
	}
	return Response{Code: code, Error: err.Error()}
}
