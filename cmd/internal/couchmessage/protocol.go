package couchmessage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Request has closed per-operation shapes. Wrapper lifecycle operations use a
// full binding and must pass the supervisor adapter's live identity check.
type Request struct {
	Op, Scope, Tag, Session, Nonce, ID, Target, Body string
	Binding                                          *Binding
}

type Response struct {
	Code, Error string
	Receipt     *Receipt
	Actors      []Candidate
}

func validMessageID(id string) bool {
	return id != "" && len(id) <= 128 && utf8.ValidString(id) && strings.IndexFunc(id, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) < 0
}

func ValidateRequest(r Request) error {
	// Caller identity and target fields cannot exceed a complete binding's budget.
	for _, field := range []string{r.Op, r.Scope, r.Tag, r.Session, r.Nonce, r.Target} {
		if len(field) > MaxBindingBytes {
			return errors.New("request identity exceeds limit")
		}
	}

	if r.Op == "register" || r.Op == "operator-submit" {
		if r.Binding == nil || r.Scope != "" || r.Tag != "" || r.Session != "" || r.Nonce != "" || r.ID != "" || r.Target != "" || r.Body != "" {
			return errors.New("wrapper operation requires only an exact binding")
		}
		return r.Binding.Validate()
	}
	if r.Binding != nil || r.Scope == "" || r.Tag == "" || r.Session == "" || r.Nonce == "" {
		return errors.New("operation requires the calling conversation identity")
	}
	switch r.Op {
	case "actors":
		if r.ID != "" || r.Target != "" || r.Body != "" {
			return errors.New("actors takes no message fields")
		}
	case "status":
		if !validMessageID(r.ID) || r.Target != "" || r.Body != "" {
			return errors.New("status requires only a message ID")
		}
	case "send":
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

// Handle is the broker protocol adapter. verifyRegister must validate a wrapper
// against committed live panes/workspace identity and register its endpoint;
// the transport's private socket or claimed Binding alone is not that proof.
func Handle(ctx context.Context, b *Broker, r Request, verifyRegister func(context.Context, Binding) error) Response {
	if err := ValidateRequest(r); err != nil {
		return Response{Code: "invalid-request", Error: err.Error()}
	}
	if err := ctx.Err(); err != nil {
		return protocolError(err, false)
	}
	if r.Op == "register" || r.Op == "operator-submit" {
		if verifyRegister == nil {
			return Response{Code: "unavailable", Error: "wrapper verification is unavailable"}
		}
		if err := verifyRegister(ctx, *r.Binding); err != nil {
			return protocolError(err, false)
		}
		if r.Op == "operator-submit" {
			if err := b.RefreshSubmission(ctx, *r.Binding); err != nil {
				return protocolError(err, false)
			}
		}
		return Response{Code: "ok"}
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
		receipt, err := b.Status(caller, r.ID)
		if err != nil {
			return protocolError(err, false)
		}
		return Response{Code: "ok", Receipt: &receipt}
	case "send":
		receipt, err := b.Send(ctx, caller, r.ID, r.Target, r.Body)
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
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		code = "uncertain"
	}
	return Response{Code: code, Error: err.Error()}
}
