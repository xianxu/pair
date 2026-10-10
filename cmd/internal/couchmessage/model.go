// Package couchmessage owns the ephemeral peer-message admission model.
package couchmessage

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	InboundAllowance = 8
	QuietInterval    = 30 * time.Second
	ReceiptLifetime  = time.Hour
	MaxBodyBytes     = 8 * 1024
	// MaxBindingBytes bounds the complete JSON-encoded identity, including escaping.
	MaxBindingBytes       = 4 * 1024
	MaxReceiptDetailBytes = 1024
)

// Binding identifies one wrapper incarnation in one verified checkout. Slot is
// only its public address: messages must match the entire binding at delivery.
type Binding struct {
	Slot, Repository, Scope, Tag, Session, Nonce, Agent, Version string
	PID                                                          int
	Start                                                        string
}

func (b Binding) Validate() error {
	if _, _, err := parseSlot(b.Slot); err != nil {
		return err
	}
	if b.Repository == "" || b.Scope == "" || b.Tag == "" || b.Session == "" || b.Nonce == "" || b.Agent == "" || b.Version == "" || b.PID <= 0 || b.Start == "" {
		return errors.New("incomplete wrapper binding")
	}
	raw, err := json.Marshal(b)
	if err != nil || len(raw) > MaxBindingBytes {
		return errors.New("wrapper binding exceeds encoded identity limit")
	}
	return nil
}

// ValidateBody admits bounded UTF-8 text, preserving intentional newlines.
func ValidateBody(body string) error {
	if !utf8.ValidString(body) || strings.TrimSpace(body) == "" || len(body) > MaxBodyBytes || strings.IndexFunc(body, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\t' }) >= 0 {
		return errors.New("message body must be nonempty UTF-8 text of at most 8192 bytes without terminal controls")
	}
	return nil
}

// Deadline is the paste-by time: the latest moment the recipient's wrapper may
// begin the paste. The delivery window (DeliveryTimeout) starts at the paste,
// so a slot still booting spends PasteTimeout, not the window (pair#427).
type Message struct {
	ID       string
	From, To Binding
	Body     string
	Deadline time.Time
}

// Horizon is the latest a delivery can still be in flight: a paste at the
// deadline plus its full window. It is the one outer bound the broker side
// reads; the wrapper bounds its own window from the actual paste.
func (m Message) Horizon() time.Time { return m.Deadline.Add(DeliveryTimeout) }

type Status string

const (
	Queued        Status = "queued"
	Delivering    Status = "delivering"
	Submitted     Status = "submitted"
	Expired       Status = "expired"
	Cancelled     Status = "cancelled"
	Indeterminate Status = "indeterminate"
)

func (s Status) Terminal() bool {
	return s == Submitted || s == Expired || s == Cancelled || s == Indeterminate
}

type Receipt struct {
	Message     Message
	Status      Status
	Detail      string
	RetainUntil time.Time
}

var (
	ErrRecipientBusy   = errors.New("recipient has a pending message")
	ErrBudgetExhausted = errors.New("recipient peer-message allowance exhausted")
	ErrUnavailable     = errors.New("recipient is not live")
	ErrUnsupported     = errors.New("recipient does not support peer delivery")
	// ErrUncertain: an outcome may exist that this request could not observe.
	ErrUncertain     = errors.New("outcome uncertain")
	ErrAmbiguous     = errors.New("recipient repository is ambiguous")
	ErrInvalidTarget = errors.New("invalid recipient target")
	ErrNoRecipient   = errors.New("no available live recipient")
)

type ActorState struct {
	binding                    Binding
	connected                  bool
	remaining                  int
	pending                    Message
	phase                      Status
	registeredAt, lastActivity time.Time
}

func (s ActorState) Binding() Binding         { return s.binding }
func (s ActorState) Connected() bool          { return s.connected }
func (s ActorState) Remaining() int           { return s.remaining }
func (s ActorState) Pending() (Message, bool) { return s.pending, s.pending.ID != "" }
func (s ActorState) LastActivity() time.Time  { return s.lastActivity }
func (s ActorState) RegisteredAt() time.Time  { return s.registeredAt }

type EventKind string

const (
	Register         EventKind = "register"
	Disconnect       EventKind = "disconnect"
	OperatorInput    EventKind = "operator-input"
	AgentOutput      EventKind = "agent-output"
	OperatorSubmit   EventKind = "operator-submit"
	Admit            EventKind = "admit"
	DeliveryStarted  EventKind = "delivery-started"
	DeliveryFinished EventKind = "delivery-finished"
	Tick             EventKind = "tick"
)

type Event struct {
	Kind    EventKind
	Binding Binding
	Message Message
	At      time.Time
	Status  Status
	Detail  string
}
type Effect struct{ Receipt Receipt }

// Advance is pure. The broker retains receipts and deduplicates completed IDs;
// the actor also deduplicates its current pending admission without spending.
func Advance(s ActorState, e Event) (ActorState, []Effect, error) {
	original := s
	fail := func(err error) (ActorState, []Effect, error) { return original, nil, err }
	if e.At.IsZero() {
		return fail(errors.New("event observation time is required"))
	}
	var effects []Effect
	emit := func(status Status, detail string) {
		effects = append(effects, Effect{Receipt: Receipt{Message: s.pending, Status: status, Detail: boundedReceiptDetail(detail), RetainUntil: e.At.Add(ReceiptLifetime)}})
	}
	finish := func(status Status, detail string) {
		if s.pending.ID != "" {
			emit(status, detail)
			s.pending = Message{}
			s.phase = ""
		}
	}
	interrupt := func(detail string) {
		status := Cancelled
		if s.phase == Delivering {
			status = Indeterminate
		}
		finish(status, detail)
	}
	activity := func() {
		if e.At.After(s.lastActivity) {
			s.lastActivity = e.At
		}
	}
	if e.Kind == Register {
		if err := e.Binding.Validate(); err != nil {
			return fail(err)
		}
		if s.binding == e.Binding && s.connected {
			return s, nil, nil
		}
		if s.binding != e.Binding {
			interrupt("wrapper incarnation replaced")
			s = ActorState{binding: e.Binding, remaining: InboundAllowance}
		}
		s.connected = true
		s.registeredAt = e.At
		activity()
		return s, effects, nil
	}
	if e.Kind != Tick || e.Binding != (Binding{}) {
		if s.binding != (e.Binding) || s.binding == (Binding{}) {
			return fail(errors.New("stale or unknown wrapper binding"))
		}
	}
	if e.Kind == Disconnect {
		interrupt("wrapper disconnected")
		s.connected = false
		return s, effects, nil
	}
	if e.Kind == Tick {
		if s.pending.ID != "" && !e.At.Before(s.pending.Horizon()) {
			status := Expired
			if s.phase == Delivering {
				status = Indeterminate
			}
			finish(status, "delivery deadline elapsed")
		}
		return s, effects, nil
	}
	if !s.connected {
		return fail(ErrUnavailable)
	}
	switch e.Kind {
	case OperatorInput, AgentOutput:
		activity()
	case OperatorSubmit:
		s.remaining = InboundAllowance
		activity()
	case Admit:
		m := e.Message
		if !validMessageID(m.ID) || m.To != s.binding || m.From.Validate() != nil || ValidateBody(m.Body) != nil || !m.Deadline.After(e.At) {
			return fail(errors.New("invalid message admission"))
		}
		if s.pending.ID != "" {
			if m == s.pending {
				return s, nil, nil
			}
			return fail(ErrRecipientBusy)
		}
		if s.remaining <= 0 {
			return fail(ErrBudgetExhausted)
		}
		s.pending = m
		s.phase = Queued
		s.remaining--
		emit(Queued, "")
	case DeliveryStarted:
		if s.pending.ID == "" || e.Message != s.pending {
			return s, nil, nil
		}
		if !e.At.Before(s.pending.Deadline) {
			finish(Expired, "delivery deadline elapsed")
			break
		}
		if s.phase == Queued {
			s.phase = Delivering
			emit(Delivering, "")
		}
	case DeliveryFinished:
		if s.pending.ID == "" || e.Message != s.pending {
			return s, nil, nil
		}
		if !e.Status.Terminal() {
			return fail(fmt.Errorf("invalid terminal delivery status %q", e.Status))
		}
		if e.Status == Submitted && s.phase != Delivering {
			return fail(errors.New("submission requires delivery ownership"))
		}
		if !e.At.Before(s.pending.Horizon()) && e.Status == Submitted {
			finish(Indeterminate, "delivery completion arrived after deadline")
			break
		}
		finish(e.Status, e.Detail)
	default:
		return fail(fmt.Errorf("unknown actor event %q", e.Kind))
	}
	return s, effects, nil
}

// Keep diagnostic text bounded without changing delivery status or message bytes.
func boundedReceiptDetail(detail string) string {
	detail = strings.ToValidUTF8(detail, "�")
	if len(detail) <= MaxReceiptDetailBytes {
		return detail
	}
	end := MaxReceiptDetailBytes - len("…")
	for !utf8.RuneStart(detail[end]) {
		end--
	}
	return detail[:end] + "…"
}
