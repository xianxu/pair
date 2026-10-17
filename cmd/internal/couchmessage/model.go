// Package couchmessage owns the ephemeral peer-message admission model.
package couchmessage

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	// The breaker is a leaky bucket (#433): a burst of InboundAllowance, one
	// message back per RefillInterval. A runaway exchange arrives seconds
	// apart and empties the burst long before it refills; reports minutes apart
	// never drain it. HubAllowance is the TL slot's burst: the slot Couch
	// started from, which every worker reports to.
	InboundAllowance = 8
	HubAllowance     = 64
	RefillInterval   = 3 * time.Minute
	QuietInterval    = 30 * time.Second
	ReceiptLifetime  = time.Hour
	MaxBodyBytes     = 8 * 1024
	// MaxBindingBytes bounds the complete JSON-encoded identity, including escaping.
	MaxBindingBytes       = 4 * 1024
	MaxReceiptDetailBytes = 1024
	// MaxHold bounds how far holds the operator causes (the composer
	// scrolled out of view, a draft in it) can push the paste-by back
	// (pair#435). The recipient takes one message at a time, so a pane left
	// scrolled up blocks other senders for at most this long.
	MaxHold = 30 * time.Minute
	// A recipient takes one delivery at a time; messages sent meanwhile wait
	// behind it in order (pair#492). MaxQueued bounds the line. QueueWait
	// bounds a wait: past a full hold and its delivery, inside ReceiptLifetime
	// so the expiry stays readable.
	MaxQueued = 32
	QueueWait = 45 * time.Minute
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
// Time the operator holds the delivery doesn't count against it, up to
// HoldLimit (pair#435).
type Message struct {
	ID       string
	From, To Binding
	Body     string
	Deadline time.Time
	// AcceptedAt is the broker's immutable admission time, including time
	// spent queued before promotion. Zero is accepted for legacy messages.
	AcceptedAt time.Time `json:",omitzero"`
}

// HoldLimit is the latest paste-by that operator holds can reach.
func (m Message) HoldLimit() time.Time { return m.Deadline.Add(MaxHold) }

// Horizon is the latest a delivery can still be in flight: a paste at the
// hold limit plus its full window. It is the one outer bound the broker side
// reads; the wrapper bounds its own paste-by and window.
func (m Message) Horizon() time.Time { return m.HoldLimit().Add(DeliveryTimeout) }

type Status string

const (
	Queued     Status = "queued"
	Delivering Status = "delivering"
	// Held: not pasted, and the paste-by clock is stopped while the operator
	// holds it (pair#435); the detail names the hold.
	Held          Status = "held"
	Submitted     Status = "submitted"
	Expired       Status = "expired"
	Cancelled     Status = "cancelled"
	Indeterminate Status = "indeterminate"
	// Refused: never admitted, so delivered nowhere; the detail says why
	// (pair#618).
	Refused Status = "refused"
)

func (s Status) Terminal() bool {
	return s == Submitted || s == Expired || s == Cancelled || s == Indeterminate || s == Refused
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
	binding             Binding
	connected           bool
	remaining, capacity int
	refilledAt          time.Time
	pending             Message
	phase               Status
	// queue waits behind pending, oldest first. A queued message's Deadline is
	// its latest promotion; Promote replaces it with the paste-by.
	queue                      []Message
	registeredAt, lastActivity time.Time
}

func (s ActorState) Binding() Binding         { return s.binding }
func (s ActorState) Connected() bool          { return s.connected }
func (s ActorState) Remaining() int           { return s.remaining }
func (s ActorState) Capacity() int            { return s.capacity }
func (s ActorState) Pending() (Message, bool) { return s.pending, s.pending.ID != "" }
func (s ActorState) Queued() int              { return len(s.queue) }

// QueueHead is the next message to promote.
func (s ActorState) QueueHead() (Message, bool) {
	if len(s.queue) == 0 {
		return Message{}, false
	}
	return s.queue[0], true
}
func (s ActorState) LastActivity() time.Time { return s.lastActivity }
func (s ActorState) RegisteredAt() time.Time { return s.registeredAt }

// RemainingAt is the allowance at now, crediting refills no event has applied.
func (s ActorState) RemainingAt(now time.Time) int { return s.refill(now).remaining }

func (s ActorState) refill(now time.Time) ActorState {
	if s.remaining >= s.capacity {
		s.refilledAt = now
		return s
	}
	if n := int(now.Sub(s.refilledAt) / RefillInterval); n > 0 {
		s.remaining = min(s.capacity, s.remaining+n)
		s.refilledAt = s.refilledAt.Add(time.Duration(n) * RefillInterval)
	}
	return s
}

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
	// DeliveryProgress carries the recipient's interim receipt (Queued, Held
	// or Delivering) to the sender's view of a delivery in flight.
	DeliveryProgress EventKind = "delivery-progress"
	Tick             EventKind = "tick"
	// Promote makes the queue's head (Message.ID names it) the delivery once
	// the recipient's wrapper reserved it (pair#492).
	Promote EventKind = "promote"
	// Cancel ends queued messages with Detail: the one Message.ID names, or
	// with no ID, every one from Message.From.
	Cancel EventKind = "cancel"
	// Enqueue is Admit that always joins the queue: the wrapper refused a
	// reservation, so the worker promotes it once the wrapper settles.
	Enqueue EventKind = "enqueue"
)

type Event struct {
	Kind    EventKind
	Binding Binding
	// Allowance is a Register's bucket size; zero means InboundAllowance.
	Allowance int
	Message   Message
	At        time.Time
	Status    Status
	Detail    string
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
	emitFor := func(m Message, status Status, detail string) {
		effects = append(effects, Effect{Receipt: Receipt{Message: m, Status: status, Detail: boundedReceiptDetail(detail), RetainUntil: e.At.Add(ReceiptLifetime)}})
	}
	emit := func(status Status, detail string) { emitFor(s.pending, status, detail) }
	// positions re-reports each queued message's place from index from on.
	positions := func(from int) {
		for i := from; i < len(s.queue); i++ {
			emitFor(s.queue[i], Queued, "position "+strconv.Itoa(i+1))
		}
	}
	// dequeue ends queued messages for which end returns true.
	dequeue := func(end func(Message) bool, status Status, detail string) {
		kept := s.queue[:0:0]
		first := -1
		for _, m := range s.queue {
			if end(m) {
				emitFor(m, status, detail)
				if first < 0 {
					first = len(kept)
				}
				continue
			}
			kept = append(kept, m)
		}
		s.queue = kept
		if first >= 0 {
			positions(first)
		}
	}
	// A recipient going away ends its line in words a sender can act on.
	leave := func() {
		dequeue(func(Message) bool { return true }, Cancelled, s.binding.Slot+" disconnected (relaunched or exited) before delivery; resend")
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
			leave()
			allowance := e.Allowance
			if allowance <= 0 {
				allowance = InboundAllowance
			}
			s = ActorState{binding: e.Binding, remaining: allowance, capacity: allowance, refilledAt: e.At}
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
		leave()
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
		dequeue(func(m Message) bool { return !e.At.Before(m.Deadline) }, Expired, "waited in the queue past "+QueueWait.String()+"; resend if still needed")
		return s, effects, nil
	}
	if !s.connected {
		return fail(ErrUnavailable)
	}
	if e.At.After(s.refilledAt) {
		s = s.refill(e.At)
	}
	switch e.Kind {
	case OperatorInput, AgentOutput:
		activity()
	case OperatorSubmit:
		s.remaining, s.refilledAt = s.capacity, e.At
		activity()
	case Admit, Enqueue:
		m := e.Message
		if !validMessageID(m.ID) || m.To != s.binding || m.From.Validate() != nil || ValidateBody(m.Body) != nil || !m.Deadline.After(e.At) {
			return fail(errors.New("invalid message admission"))
		}
		if m.ID == s.pending.ID {
			return s, nil, nil
		}
		for _, q := range s.queue {
			if q.ID == m.ID {
				return s, nil, nil
			}
		}
		busy := s.pending.ID != "" || len(s.queue) > 0 || e.Kind == Enqueue
		if busy && len(s.queue) >= MaxQueued {
			return fail(fmt.Errorf("%w: %d messages already wait", ErrRecipientBusy, MaxQueued))
		}
		// A self-addressed message comes from inside the recipient's own
		// session: the operator's session-sync submit (parley.nvim#313). It
		// is an operator submit, not a peer message (#433).
		if m.From == s.binding {
			s.remaining, s.refilledAt = s.capacity, e.At
		} else if s.remaining <= 0 {
			return fail(ErrBudgetExhausted)
		} else {
			s.remaining--
		}
		if busy {
			m.Deadline = e.At.Add(QueueWait)
			s.queue = append(slices.Clip(s.queue), m)
			positions(len(s.queue) - 1)
			break
		}
		s.pending = m
		s.phase = Queued
		emit(Queued, "")
	case Promote:
		head, ok := s.QueueHead()
		if s.pending.ID != "" || !ok || head.ID != e.Message.ID {
			return fail(errors.New("promotion is not of the queue's head behind an idle recipient"))
		}
		if !e.At.Before(head.Deadline) {
			return fail(errors.New("queued message expired before promotion"))
		}
		s.queue = s.queue[1:]
		head.Deadline = e.At.Add(PasteTimeout)
		s.pending = head
		s.phase = Queued
		emit(Queued, "")
		positions(0)
	case Cancel:
		dequeue(func(m Message) bool {
			if e.Message.ID != "" {
				return m.ID == e.Message.ID
			}
			return e.Message.From != (Binding{}) && m.From == e.Message.From
		}, Cancelled, e.Detail)
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
	case DeliveryProgress:
		if s.pending.ID == "" || e.Message != s.pending || s.phase != Delivering {
			return s, nil, nil
		}
		status := Delivering
		switch e.Status {
		case Held:
			status = Held
		case Queued, Delivering:
		default:
			return fail(fmt.Errorf("invalid interim delivery status %q", e.Status))
		}
		emit(status, e.Detail)
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
