package couchmessage

import (
	"errors"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func binding(slot string) Binding {
	return Binding{Slot: slot, Repository: "/repo/.git", Scope: "scope", Tag: slot, Session: "session", Nonce: "nonce", Agent: "codex", Version: "1", PID: 123, Start: "start"}
}
func registered(t *testing.T, b Binding, now time.Time) ActorState {
	t.Helper()
	s, _, err := Advance(ActorState{}, Event{Kind: Register, Binding: b, At: now})
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func advance(t *testing.T, s ActorState, e Event) (ActorState, []Effect) {
	t.Helper()
	n, fx, err := Advance(s, e)
	if err != nil {
		t.Fatal(err)
	}
	return n, fx
}
func TestAdmissionAndTerminalTransitions(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("pair:1")
	s := registered(t, b, now)
	m := Message{ID: "m1", From: binding("brain:0"), To: b, Body: "work", Deadline: now.Add(30 * time.Second)}
	s, fx := advance(t, s, Event{Kind: Admit, Binding: b, Message: m, At: now})
	if s.Remaining() != 7 || len(fx) != 1 || fx[0].Receipt.Status != Queued {
		t.Fatalf("admit: %#v %#v", s, fx)
	}
	original := s
	s, fx = advance(t, s, Event{Kind: Admit, Binding: b, Message: m, At: now})
	if s.Remaining() != 7 || len(fx) != 0 {
		t.Fatal("duplicate spends budget or emits work")
	}
	other := m
	other.ID = "m2"
	if _, _, err := Advance(s, Event{Kind: Admit, Binding: b, Message: other, At: now}); !errors.Is(err, ErrRecipientBusy) {
		t.Fatalf("busy: %v", err)
	}
	s, _ = advance(t, s, Event{Kind: DeliveryStarted, Binding: b, Message: m, At: now})
	s, fx = advance(t, s, Event{Kind: DeliveryFinished, Binding: b, Message: m, Status: Submitted, At: now})
	if _, ok := s.Pending(); ok || len(fx) != 1 || fx[0].Receipt.Status != Submitted {
		t.Fatal("terminal did not clear pending")
	}
	s, fx = advance(t, s, Event{Kind: DeliveryFinished, Binding: b, Message: m, Status: Cancelled, At: now})
	if len(fx) != 0 {
		t.Fatal("late event rewrote terminal receipt")
	}
	if _, ok := original.Pending(); !ok {
		t.Fatal("mutated input state")
	}
}
func TestReconnectRetainsBudgetAndReplacementRejectsStaleEvents(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("pair:1")
	s := registered(t, b, now)
	m := Message{ID: "m", From: binding("brain:0"), To: b, Body: "work", Deadline: now.Add(time.Minute)}
	s, _ = advance(t, s, Event{Kind: Admit, Binding: b, Message: m, At: now})
	s, fx := advance(t, s, Event{Kind: Disconnect, Binding: b, At: now})
	if s.Connected() || fx[0].Receipt.Status != Cancelled {
		t.Fatal("disconnect")
	}
	s, _ = advance(t, s, Event{Kind: Register, Binding: b, At: now.Add(time.Second)})
	if s.Remaining() != 7 || !s.LastActivity().Equal(now.Add(time.Second)) {
		t.Fatal("reconnect resets budget or assumes quiet")
	}
	replacement := b
	replacement.Start = "new"
	s, _ = advance(t, s, Event{Kind: Register, Binding: replacement, At: now.Add(2 * time.Second)})
	if s.Remaining() != 8 {
		t.Fatal("new incarnation did not reset")
	}
	if _, _, err := Advance(s, Event{Kind: OperatorSubmit, Binding: b, At: now}); err == nil {
		t.Fatal("stale incarnation accepted")
	}
}
func TestDeadlineAndDisconnectAfterDeliveryAreIndeterminate(t *testing.T) {
	for _, event := range []EventKind{Tick, Disconnect} {
		t.Run(string(event), func(t *testing.T) {
			now := time.Unix(100, 0)
			b := binding("pair:1")
			s := registered(t, b, now)
			m := Message{ID: "m", From: binding("brain:0"), To: b, Body: "work", Deadline: now.Add(time.Second)}
			s, _ = advance(t, s, Event{Kind: Admit, Binding: b, Message: m, At: now})
			s, _ = advance(t, s, Event{Kind: DeliveryStarted, Binding: b, Message: m, At: now})
			_, fx := advance(t, s, Event{Kind: event, Binding: b, At: m.Horizon()})
			if len(fx) != 1 || fx[0].Receipt.Status != Indeterminate {
				t.Fatalf("effects %#v", fx)
			}
		})
	}
}
func TestGeneratedBudgetAndPendingInvariants(t *testing.T) {
	rng := rand.New(rand.NewSource(353))
	now := time.Unix(100, 0)
	b := binding("pair:1")
	s := registered(t, b, now)
	for i := 0; i < 2000; i++ {
		previous := s.Remaining()
		kind := []EventKind{Admit, OperatorInput, AgentOutput, OperatorSubmit, Tick, DeliveryFinished}[rng.Intn(6)]
		m, ok := s.Pending()
		if !ok {
			m = Message{ID: strconv.Itoa(i), From: binding("brain:0"), To: b, Body: "work", Deadline: now.Add(time.Second)}
		}
		next, _, err := Advance(s, Event{Kind: kind, Binding: b, Message: m, At: now, Status: Submitted})
		if err == nil {
			s = next
		}
		if s.Remaining() < 0 || s.Remaining() > 8 {
			t.Fatal("unbounded budget")
		}
		if kind != OperatorSubmit && s.Remaining() > previous {
			t.Fatal("nonoperator event replenished budget")
		}
		now = now.Add(time.Millisecond)
	}
}

func TestBudgetCannotBeResetByReconnectOrTraffic(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("pair:1")
	s := registered(t, b, now)
	for i := 0; i < InboundAllowance; i++ {
		m := Message{ID: strconv.Itoa(i), From: binding("brain:0"), To: b, Body: "work", Deadline: now.Add(time.Minute)}
		s, _ = advance(t, s, Event{Kind: Admit, Binding: b, Message: m, At: now})
		s, _ = advance(t, s, Event{Kind: DeliveryFinished, Binding: b, Message: m, Status: Cancelled, At: now})
	}
	for _, kind := range []EventKind{OperatorInput, AgentOutput, Disconnect, Register, Tick} {
		s, _ = advance(t, s, Event{Kind: kind, Binding: b, At: now.Add(time.Second)})
	}
	m := Message{ID: "ninth", From: binding("brain:0"), To: b, Body: "work", Deadline: now.Add(time.Minute)}
	if _, _, err := Advance(s, Event{Kind: Admit, Binding: b, Message: m, At: now.Add(time.Second)}); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("ninth: %v", err)
	}
	s, _ = advance(t, s, Event{Kind: OperatorSubmit, Binding: b, At: now.Add(time.Second)})
	if s.Remaining() != 8 {
		t.Fatal("operator submission did not reset allowance")
	}
}

func TestLateCompletionCannotClaimSubmission(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("pair:1")
	s := registered(t, b, now)
	m := Message{ID: "late", From: binding("brain:0"), To: b, Body: "work", Deadline: now.Add(time.Second)}
	s, _ = advance(t, s, Event{Kind: Admit, Binding: b, Message: m, At: now})
	s, _ = advance(t, s, Event{Kind: DeliveryStarted, Binding: b, Message: m, At: now})
	_, fx := advance(t, s, Event{Kind: DeliveryFinished, Binding: b, Message: m, Status: Submitted, At: m.Horizon()})
	if len(fx) != 1 || fx[0].Receipt.Status != Indeterminate {
		t.Fatal("late completion upgraded to submitted")
	}
}

// pair#427: the deadline is paste-by; the window runs from the paste. A
// submission completing after the paste-by time but inside the horizon stands,
// and the broker's own expiry waits for the horizon.
func TestDeliveryWindowRunsPastThePasteDeadline(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("pair:1")
	s := registered(t, b, now)
	m := Message{ID: "window", From: binding("brain:0"), To: b, Body: "work", Deadline: now.Add(time.Second)}
	if m.Horizon() != m.Deadline.Add(MaxHold+DeliveryTimeout) {
		t.Fatal(m.Horizon())
	}
	s, _ = advance(t, s, Event{Kind: Admit, Binding: b, Message: m, At: now})
	s, _ = advance(t, s, Event{Kind: DeliveryStarted, Binding: b, Message: m, At: now})
	ticked, fx := advance(t, s, Event{Kind: Tick, At: m.Deadline.Add(time.Second)})
	if len(fx) != 0 {
		t.Fatalf("expired at the paste deadline: %+v", fx)
	}
	if _, fx = advance(t, ticked, Event{Kind: Tick, At: m.Horizon()}); len(fx) != 1 || fx[0].Receipt.Status != Indeterminate {
		t.Fatalf("no expiry at the horizon: %+v", fx)
	}
	if _, fx = advance(t, ticked, Event{Kind: DeliveryFinished, Binding: b, Message: m, Status: Submitted, At: m.Deadline.Add(time.Second)}); len(fx) != 1 || fx[0].Receipt.Status != Submitted {
		t.Fatalf("in-window submission downgraded: %+v", fx)
	}
}

// pair#435: the recipient's interim receipt reaches the sender's view, so a
// held delivery reads "held" with its reason; the broker's horizon covers
// the longest hold, and a late Held never rewrites a terminal receipt.
func TestDeliveryProgressReportsHeld(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("pair:1")
	s := registered(t, b, now)
	m := Message{ID: "held", From: binding("brain:0"), To: b, Body: "work", Deadline: now.Add(PasteTimeout)}
	s, _ = advance(t, s, Event{Kind: Admit, Binding: b, Message: m, At: now})
	if _, fx := advance(t, s, Event{Kind: DeliveryProgress, Binding: b, Message: m, Status: Held, At: now}); len(fx) != 0 {
		t.Fatalf("progress before the job owns the delivery: %+v", fx)
	}
	s, _ = advance(t, s, Event{Kind: DeliveryStarted, Binding: b, Message: m, At: now})
	at := m.Deadline.Add(time.Minute) // past the paste-by: the hold stopped the clock
	s, fx := advance(t, s, Event{Kind: DeliveryProgress, Binding: b, Message: m, Status: Held, Detail: "waiting for the composer to be in view", At: at})
	if len(fx) != 1 || fx[0].Receipt.Status != Held || fx[0].Receipt.Detail != "waiting for the composer to be in view" {
		t.Fatalf("held progress: %+v", fx)
	}
	if _, fx = advance(t, s, Event{Kind: Tick, At: at}); len(fx) != 0 {
		t.Fatalf("broker expired a held delivery inside the hold limit: %+v", fx)
	}
	if _, fx = advance(t, s, Event{Kind: DeliveryProgress, Binding: b, Message: m, Status: Queued, Detail: "waiting for paste mode", At: at}); len(fx) != 1 || fx[0].Receipt.Status != Delivering {
		t.Fatalf("released progress: %+v", fx)
	}
	if _, _, err := Advance(s, Event{Kind: DeliveryProgress, Binding: b, Message: m, Status: Submitted, At: at}); err == nil {
		t.Fatal("terminal status accepted as progress")
	}
	s, _ = advance(t, s, Event{Kind: DeliveryFinished, Binding: b, Message: m, Status: Submitted, At: at})
	if _, fx = advance(t, s, Event{Kind: DeliveryProgress, Binding: b, Message: m, Status: Held, At: at}); len(fx) != 0 {
		t.Fatalf("late progress after the outcome: %+v", fx)
	}
}

func TestBodyValidation(t *testing.T) {
	for _, body := range []string{"", " \n", string([]byte{0xff}), "hello\x00there", "hello\x1b[201~\r"} {
		if ValidateBody(body) == nil {
			t.Fatalf("accepted unsafe body %q", body)
		}
	}
	if err := ValidateBody("do this\nthen that\tplease"); err != nil {
		t.Fatal(err)
	}
}

// #433 Done-when 1: past the burst a message is refused; one RefillInterval
// later exactly one more is admitted. The listing's RemainingAt agrees.
func TestBucketRefillsOneMessagePerInterval(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("pair:1")
	s := drain(t, registered(t, b, now), b, now)
	late := now.Add(RefillInterval - time.Second)
	if _, _, err := Advance(s, admitEvent(b, "early", late)); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("before the interval: %v", err)
	}
	refilled := now.Add(RefillInterval)
	if got := s.RemainingAt(refilled); got != 1 {
		t.Fatalf("RemainingAt = %d, want 1", got)
	}
	s, _ = advance(t, s, admitEvent(b, "one", refilled))
	s, _ = advance(t, s, Event{Kind: DeliveryFinished, Binding: b, Message: mustPending(t, s), Status: Cancelled, At: refilled})
	if _, _, err := Advance(s, admitEvent(b, "two", refilled.Add(time.Second))); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("second after one interval: %v", err)
	}
	if got := s.RemainingAt(now.Add(time.Hour)); got != InboundAllowance {
		t.Fatalf("refill exceeded capacity: %d", got)
	}
}

// #433 Done-when 2: a self-addressed message (the operator's session-sync
// submit) refills the bucket and spends nothing.
func TestSelfAddressedMessageRefillsFully(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("pair:1")
	s := drain(t, registered(t, b, now), b, now)
	self := Message{ID: "sync", From: b, To: b, Body: "submitted: x.md", Deadline: now.Add(time.Minute)}
	s, fx := advance(t, s, Event{Kind: Admit, Binding: b, Message: self, At: now})
	if s.Remaining() != InboundAllowance || len(fx) != 1 || fx[0].Receipt.Status != Queued {
		t.Fatalf("self-addressed admit: remaining %d, %+v", s.Remaining(), fx)
	}
}

// #433 Done-when 3: the hub's bucket admits well past the ordinary burst with
// no operator keystroke.
func TestHubAllowanceAdmitsPastTheOrdinaryBurst(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("ops:0")
	s, _, err := Advance(ActorState{}, Event{Kind: Register, Binding: b, Allowance: HubAllowance, At: now})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4*InboundAllowance; i++ {
		s, _ = advance(t, s, admitEvent(b, strconv.Itoa(i), now))
		s, _ = advance(t, s, Event{Kind: DeliveryFinished, Binding: b, Message: mustPending(t, s), Status: Cancelled, At: now})
	}
	if s.Capacity() != HubAllowance || s.Remaining() != HubAllowance-4*InboundAllowance {
		t.Fatalf("hub %d/%d", s.Remaining(), s.Capacity())
	}
}

// #433 Done-when 4: two agents answering each other a second apart are cut
// off within seconds, even when one of them is the hub.
func TestPingPongIsCutOffWithinSeconds(t *testing.T) {
	for _, hub := range []bool{false, true} {
		now := time.Unix(100, 0)
		a, w := binding("ops:0"), binding("pair:1")
		allowance := 0
		if hub {
			allowance = HubAllowance
		}
		sa, _, _ := Advance(ActorState{}, Event{Kind: Register, Binding: a, Allowance: allowance, At: now})
		sw := registered(t, w, now)
		states := map[Binding]*ActorState{a: &sa, w: &sw}
		from, to := a, w
		var err error
		for i := 0; err == nil; i++ {
			at := now.Add(time.Duration(i) * time.Second)
			if at.Sub(now) > 20*time.Second {
				t.Fatalf("hub=%v: ping-pong still running after %v", hub, at.Sub(now))
			}
			m := Message{ID: strconv.Itoa(i), From: from, To: to, Body: "ping", Deadline: at.Add(time.Minute)}
			var next ActorState
			if next, _, err = Advance(*states[to], Event{Kind: Admit, Binding: to, Message: m, At: at}); err == nil {
				next, _ = advance(t, next, Event{Kind: DeliveryFinished, Binding: to, Message: m, Status: Cancelled, At: at})
				*states[to] = next
			}
			from, to = to, from
		}
		if !errors.Is(err, ErrBudgetExhausted) {
			t.Fatalf("hub=%v: %v", hub, err)
		}
	}
}

func admitEvent(b Binding, id string, at time.Time) Event {
	return Event{Kind: Admit, Binding: b, Message: Message{ID: id, From: binding("brain:0"), To: b, Body: "work", Deadline: at.Add(time.Minute)}, At: at}
}

func mustPending(t *testing.T, s ActorState) Message {
	t.Helper()
	m, ok := s.Pending()
	if !ok {
		t.Fatal("no pending message")
	}
	return m
}

// drain spends b's whole bucket at now and confirms the next one is refused.
func drain(t *testing.T, s ActorState, b Binding, now time.Time) ActorState {
	t.Helper()
	for i := 0; i < s.Capacity(); i++ {
		s, _ = advance(t, s, admitEvent(b, "drain"+strconv.Itoa(i), now))
		s, _ = advance(t, s, Event{Kind: DeliveryFinished, Binding: b, Message: mustPending(t, s), Status: Cancelled, At: now})
	}
	if _, _, err := Advance(s, admitEvent(b, "over", now)); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("burst past allowance: %v", err)
	}
	return s
}
