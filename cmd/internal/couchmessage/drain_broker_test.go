package couchmessage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDrainNoticeBypassesHeldPeerFIFO(t *testing.T) {
	now := time.Now()
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, to := brokerBinding("pair:0"), brokerBinding("pair:1")
	f := newFakeEndpoint(now)
	if err := b.Register(from, newFakeEndpoint(now)); err != nil {
		t.Fatal(err)
	}
	if err := b.Register(to, f); err != nil {
		t.Fatal(err)
	}
	if err := b.HoldDrain("generation"); err != nil {
		t.Fatal(err)
	}
	// Accepted before the hold: a queued peer prefix must survive, in order.
	b.mu.Lock()
	for _, id := range []string{"peer1", "peer2"} {
		m := Message{ID: id, From: from, To: to, Body: id, Deadline: now.Add(time.Minute), AcceptedAt: now}
		if err := b.apply(b.actors[to], Event{Kind: Enqueue, Binding: to, Message: m, At: now}); err != nil {
			t.Fatal(err)
		}
	}
	b.mu.Unlock()
	if _, err := b.Send(context.Background(), from, "new", Route{Target: to.Slot}, "new"); !errors.Is(err, ErrDraining) {
		t.Fatalf("send while draining: %v", err)
	}
	if _, err := b.SendSystem(context.Background(), from, "ordinary-system", Route{Target: to.Slot}, "not control"); !errors.Is(err, ErrDraining) {
		t.Fatalf("system bypass: %v", err)
	}
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint"); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-f.delivered:
		if m.ID != "control" || m.Origin != OriginSystem {
			t.Fatalf("first delivery %+v", m)
		}
		f.outcomes <- Receipt{Status: Submitted}
	case <-time.After(time.Second):
		t.Fatal("control stranded behind peers")
	}
	waitStatus(t, b, from, "control", Submitted)
	if err := b.ReleaseDrain("wrong"); err == nil {
		t.Fatal("foreign owner released")
	}
	if err := b.ReleaseDrain("generation"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"peer1", "peer2"} {
		select {
		case m := <-f.delivered:
			if m.ID != want {
				t.Fatalf("peer order %s, want %s", m.ID, want)
			}
			f.outcomes <- Receipt{Status: Submitted}
		case <-time.After(2 * time.Second):
			t.Fatalf("peer %s not restored", want)
		}
	}
}

func TestDrainControlDoesNotRefillOrSpendPeerAllowance(t *testing.T) {
	now := time.Now()
	binding := brokerBinding("pair:0")
	s, _, err := Advance(ActorState{}, Event{Kind: Register, Binding: binding, At: now})
	if err != nil {
		t.Fatal(err)
	}
	s.remaining = 0
	m := Message{ID: "notice", From: binding, To: binding, Origin: OriginSystem, Body: "checkpoint", Deadline: now.Add(time.Minute)}
	next, _, err := Advance(s, Event{Kind: EnqueueDrainControl, Binding: binding, Message: m, At: now})
	if err != nil || next.Remaining() != 0 {
		t.Fatalf("control changed allowance: %d %v", next.Remaining(), err)
	}
}

type drainCancelEndpoint struct {
	*FakeDeliveryEndpoint
	reserveEntered  chan struct{}
	reserveRelease  chan struct{}
	commitGate      chan struct{}
	deliveryEntered chan struct{}
	revokeUnknown   bool
	revokeInstance  string
	loseRevokeReply bool
	releaseEntered  chan struct{}
	releaseGate     chan struct{}
	revoked         map[string]bool
	committed       map[string]Receipt
}

func newDrainCancelEndpoint(now time.Time) *drainCancelEndpoint {
	return &drainCancelEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), revoked: map[string]bool{}, committed: map[string]Receipt{}}
}
func (f *drainCancelEndpoint) ObserveDrain(context.Context) (DrainObservation, error) {
	return DrainObservation{}, nil
}
func (f *drainCancelEndpoint) InterruptDrain(context.Context, string, DrainObservation) (DrainInterruptResult, error) {
	return DrainInterruptResult{}, nil
}
func (f *drainCancelEndpoint) DrainInterruptStatus(context.Context, string) (DrainInterruptResult, error) {
	return DrainInterruptResult{}, nil
}
func (f *drainCancelEndpoint) Reserve(ctx context.Context, id string, seq uint64) error {
	if f.reserveEntered != nil {
		close(f.reserveEntered)
		select {
		case <-f.reserveRelease:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	f.mu.Lock()
	revoked := f.revoked[id]
	f.mu.Unlock()
	if revoked {
		return ErrUnavailable
	}
	return f.FakeDeliveryEndpoint.Reserve(ctx, id, seq)
}
func (f *drainCancelEndpoint) Deliver(ctx context.Context, m Message, progress func(Receipt)) (Receipt, error) {
	if f.deliveryEntered != nil {
		close(f.deliveryEntered)
	}
	if f.commitGate != nil {
		select {
		case <-f.commitGate:
		case <-ctx.Done():
			return Receipt{}, ctx.Err()
		}
	}
	f.mu.Lock()
	if f.revoked[m.ID] {
		f.mu.Unlock()
		return Receipt{Message: m, Status: Cancelled}, nil
	}
	f.committed[m.ID] = Receipt{Message: m, Status: Delivering}
	f.mu.Unlock()
	return f.FakeDeliveryEndpoint.Deliver(ctx, m, progress)
}
func (f *drainCancelEndpoint) RevokeDrain(_ context.Context, id string) (DrainRevokeResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.revokeUnknown {
		return DrainRevokeResult{ID: id, Instance: f.revokeInstance, Outcome: "unknown"}, nil
	}
	if r, ok := f.committed[id]; ok {
		return DrainRevokeResult{ID: id, Instance: f.revokeInstance, Outcome: "already-committed", Receipt: &r}, nil
	}
	f.revoked[id] = true
	f.reserved = ""
	if f.loseRevokeReply {
		f.loseRevokeReply = false
		return DrainRevokeResult{}, errors.New("lost revoke reply after effect")
	}
	return DrainRevokeResult{ID: id, Instance: f.revokeInstance, Outcome: "not-committed"}, nil
}
func drainCancelRig(t *testing.T, f *drainCancelEndpoint) (*Broker, Binding, Binding) {
	t.Helper()
	b := NewBroker(context.Background(), time.Now, nil)
	t.Cleanup(func() { b.Close() })
	from, to := brokerBinding("pair:0"), brokerBinding("pair:1")
	if err := b.Register(from, newFakeEndpoint(time.Now())); err != nil {
		t.Fatal(err)
	}
	if err := b.Register(to, f); err != nil {
		t.Fatal(err)
	}
	if err := b.HoldDrain("generation"); err != nil {
		t.Fatal(err)
	}
	return b, from, to
}
func TestDrainCancelRevokesNoticeBeforeReserve(t *testing.T) {
	f := newDrainCancelEndpoint(time.Now())
	b, from, to := drainCancelRig(t, f)
	// Seed without waking: cancellation precedes the first reservation.
	b.mu.Lock()
	m := Message{ID: "control", From: from, To: to, Origin: OriginSystem, Body: "checkpoint", AcceptedAt: time.Now(), Deadline: time.Now().Add(time.Minute)}
	err := b.apply(b.actors[to], Event{Kind: EnqueueDrainControl, Binding: to, Message: m, At: time.Now()})
	b.drainControls[m.ID] = "generation"
	b.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SendDrain(context.Background(), "generation", from, "new", to, "checkpoint"); !errors.Is(err, ErrDraining) {
		t.Fatal("cancel allowed new notice", err)
	}
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err != nil {
		t.Fatal("repeat", err)
	}
	if err := b.ReleaseDrain("generation"); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-f.delivered:
		t.Fatalf("cancelled notice delivered: %+v", m)
	default:
	}
}
func TestDrainCancelRevokesNoticeDuringReserve(t *testing.T) {
	f := newDrainCancelEndpoint(time.Now())
	f.reserveEntered = make(chan struct{})
	f.reserveRelease = make(chan struct{})
	b, from, to := drainCancelRig(t, f)
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-f.reserveEntered:
	case <-time.After(time.Second):
		t.Fatal("no reservation")
	}
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err != nil {
		t.Fatal(err)
	}
	close(f.reserveRelease)
	if err := b.ReleaseDrain("generation"); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, b, from, "control", Cancelled)
	select {
	case m := <-f.delivered:
		t.Fatalf("late reserve delivered: %+v", m)
	case <-time.After(20 * time.Millisecond):
	}
}
func TestDrainCancelRevokesNoticeBeforeCommitAndReconcilesLostReply(t *testing.T) {
	f := newDrainCancelEndpoint(time.Now())
	f.commitGate = make(chan struct{})
	f.deliveryEntered = make(chan struct{})
	f.loseRevokeReply = true
	b, from, to := drainCancelRig(t, f)
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-f.deliveryEntered:
	case <-time.After(time.Second):
		t.Fatal("no pending commit")
	}
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err == nil {
		t.Fatal("lost reply became confirmed")
	}
	if err := b.ReleaseDrain("generation"); err == nil {
		t.Fatal("released uncertain revoke")
	}
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err != nil {
		t.Fatal(err)
	}
	close(f.commitGate)
	if err := b.ReleaseDrain("generation"); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-f.delivered:
		t.Fatalf("cancelled commit delivered: %+v", m)
	case <-time.After(20 * time.Millisecond):
	}
}
func TestDrainCancelCommittedRequiresRetainedKnownTerminal(t *testing.T) {
	f := newDrainCancelEndpoint(time.Now())
	b, from, to := drainCancelRig(t, f)
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint"); err != nil {
		t.Fatal(err)
	}
	var m Message
	select {
	case m = <-f.delivered:
	case <-time.After(time.Second):
		t.Fatal("no committed notice")
	}
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err == nil {
		t.Fatal("pending committed receipt treated known terminal")
	}
	if err := b.ReleaseDrain("generation"); err == nil {
		t.Fatal("released pending handoff")
	}
	f.mu.Lock()
	f.committed[m.ID] = Receipt{Message: m, Status: Submitted}
	f.mu.Unlock()
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err != nil {
		t.Fatal(err)
	}
	if err := b.ReleaseDrain("generation"); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, b, from, "control", Submitted)
	if err := b.HoldDrain("new-generation"); err != nil {
		t.Fatal(err)
	}
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err == nil {
		t.Fatal("stale cancellation mutated next generation")
	}
	select {
	case <-f.delivered:
		t.Fatal("repasted notice")
	default:
	}
}

func TestDrainCancelUnknownCannotReleaseAfterJobBecomesTerminal(t *testing.T) {
	f := newDrainCancelEndpoint(time.Now())
	f.revokeUnknown = true
	b, from, to := drainCancelRig(t, f)
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint"); err != nil {
		t.Fatal(err)
	}
	var m Message
	select {
	case m = <-f.delivered:
	case <-time.After(time.Second):
		t.Fatal("no delivery")
	}
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err == nil {
		t.Fatal("unknown revoke succeeded")
	}
	// The original RPC can independently finish ambiguously. Terminal is not
	// synonymous with a known input disposition and must not free dispatch.
	f.outcomes <- Receipt{Status: Indeterminate}
	waitStatus(t, b, from, "control", Indeterminate)
	if err := b.ReleaseDrain("generation"); err == nil {
		t.Fatal("unknown revoke plus terminal job released gate")
	}
	if err := b.HoldDrain("new"); err == nil {
		t.Fatal("unknown old generation admitted replacement")
	}
	f.mu.Lock()
	f.revokeUnknown = false
	f.committed[m.ID] = Receipt{Message: m, Status: Submitted}
	f.mu.Unlock()
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, b, from, "control", Submitted)
	if err := b.ReleaseDrain("generation"); err != nil {
		t.Fatal(err)
	}
}

func TestDrainCancelRefusesForeignCommittedReceipt(t *testing.T) {
	f := newDrainCancelEndpoint(time.Now())
	b, from, to := drainCancelRig(t, f)
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint"); err != nil {
		t.Fatal(err)
	}
	var m Message
	select {
	case m = <-f.delivered:
	case <-time.After(time.Second):
		t.Fatal("no delivery")
	}
	m.To.Session = "replacement"
	f.mu.Lock()
	f.committed[m.ID] = Receipt{Message: m, Status: Submitted}
	f.mu.Unlock()
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err == nil {
		t.Fatal("foreign incarnation receipt accepted")
	}
	if err := b.ReleaseDrain("generation"); err == nil {
		t.Fatal("foreign receipt released dispatch")
	}
}

func TestDrainCancelUnknownHoldsDespiteTerminalSubmitted(t *testing.T) {
	f := newDrainCancelEndpoint(time.Now())
	f.revokeUnknown = true
	b, from, to := drainCancelRig(t, f)
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-f.delivered:
	case <-time.After(time.Second):
		t.Fatal("no delivery")
	}
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err == nil {
		t.Fatal("unknown revoke succeeded")
	}
	f.outcomes <- Receipt{Status: Submitted}
	waitStatus(t, b, from, "control", Submitted)
	if err := b.ReleaseDrain("generation"); err == nil {
		t.Fatal("late terminal receipt bypassed cancellation reconciliation")
	}
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err != nil {
		t.Fatal(err)
	}
	if err := b.ReleaseDrain("generation"); err != nil {
		t.Fatal(err)
	}
}

func (f *drainCancelEndpoint) Release(ctx context.Context, id string) error {
	if f.releaseGate != nil {
		close(f.releaseEntered)
		select {
		case <-f.releaseGate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return f.FakeDeliveryEndpoint.Release(ctx, id)
}
func TestDrainCancelJoinsDeliveryCleanupBeforeRelease(t *testing.T) {
	f := newDrainCancelEndpoint(time.Now())
	f.commitGate = make(chan struct{})
	f.deliveryEntered = make(chan struct{})
	f.releaseGate = make(chan struct{})
	f.releaseEntered = make(chan struct{})
	b, from, to := drainCancelRig(t, f)
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-f.deliveryEntered:
	case <-time.After(time.Second):
		t.Fatal("no pending delivery")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- b.CancelDrainControl(ctx, "generation", "control") }()
	select {
	case <-f.releaseEntered:
	case <-time.After(time.Second):
		t.Fatal("no cleanup")
	}
	if err := <-done; !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("reported cancellation before held cleanup joined: %v", err)
	}
	if err := b.ReleaseDrain("generation"); err == nil {
		t.Fatal("released while delivery cleanup running")
	}
	close(f.releaseGate)
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err != nil {
		t.Fatal(err)
	}
	if err := b.ReleaseDrain("generation"); err != nil {
		t.Fatal(err)
	}
}
func TestDrainBeginCancelStopsFirstNotice(t *testing.T) {
	f := newDrainCancelEndpoint(time.Now())
	b, from, to := drainCancelRig(t, f)
	if err := b.BeginDrainCancel("other"); err == nil {
		t.Fatal("foreign generation cancelled")
	}
	if err := b.BeginDrainCancel("generation"); err != nil {
		t.Fatal(err)
	}
	if err := b.BeginDrainCancel("generation"); err != nil {
		t.Fatal("duplicate", err)
	}
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint"); !errors.Is(err, ErrDraining) {
		t.Fatalf("notice admitted after cancel: %v", err)
	}
	if err := b.ReleaseDrain("generation"); err != nil {
		t.Fatal(err)
	}
}

func TestDrainControlReceiptSurvivesActiveGenerationDeadline(t *testing.T) {
	f := newDrainCancelEndpoint(time.Now())
	b, from, to := drainCancelRig(t, f)
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-f.delivered:
		f.outcomes <- Receipt{Status: Submitted}
	case <-time.After(time.Second):
		t.Fatal("no delivery")
	}
	waitStatus(t, b, from, "control", Submitted)
	b.mu.Lock()
	future := time.Now().Add(25 * time.Hour)
	b.now = func() time.Time { return future }
	b.sweep()
	_, exists := b.receipts["control"]
	b.mu.Unlock()
	if !exists {
		t.Fatal("active drain receipt expired before max drain deadline")
	}
	if err := b.CancelDrainControl(context.Background(), "generation", "control"); err != nil {
		t.Fatal(err)
	}
	if err := b.ReleaseDrain("generation"); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	b.sweep()
	_, exists = b.receipts["control"]
	b.mu.Unlock()
	if exists {
		t.Fatal("released drain retained receipt beyond ordinary lifetime")
	}
}

func TestDrainNoticeExpectedInstanceFence(t *testing.T) {
	f := newDrainCancelEndpoint(time.Now())
	b, from, to := drainCancelRig(t, f)
	for _, instances := range [][]string{{""}, {"one", "two"}, {strings.Repeat("x", 129)}, {"bad\ninstance"}} {
		if _, err := b.SendDrain(context.Background(), "generation", from, "invalid", to, "checkpoint", instances...); err == nil {
			t.Fatalf("accepted invalid instance %q", instances)
		}
	}
	r, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint", "expected-instance")
	if err != nil {
		t.Fatal(err)
	}
	if r.Message.DrainInstance != "expected-instance" {
		t.Fatal("admission lost instance fence")
	}
	select {
	case m := <-f.delivered:
		if m.DrainInstance != "expected-instance" {
			t.Fatal("delivery lost instance fence")
		}
		f.outcomes <- Receipt{Status: Submitted}
	case <-time.After(time.Second):
		t.Fatal("no delivery")
	}
	waitStatus(t, b, from, "control", Submitted)
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint", "different-instance"); err == nil {
		t.Fatal("retained control adopted different instance")
	}
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint", "expected-instance"); err != nil {
		t.Fatal("matching retained control refused", err)
	}
	other := r.Message
	other.DrainInstance = "different-instance"
	if drainReceiptMatches(r.Message, other) {
		t.Fatal("retained receipt match ignored instance")
	}
}

func TestDrainInstancePureAdmissionRejectsPeerAndMalformedFence(t *testing.T) {
	now := time.Now()
	binding := brokerBinding("pair:0")
	s, _, err := Advance(ActorState{}, Event{Kind: Register, Binding: binding, At: now})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []Message{
		{ID: "peer", Origin: OriginPeer, From: binding, To: binding, Body: "x", Deadline: now.Add(time.Minute), DrainInstance: "instance"},
		{ID: "system", Origin: OriginSystem, From: binding, To: binding, Body: "x", Deadline: now.Add(time.Minute), DrainInstance: "invalid\ninstance"},
	} {
		if _, _, err := Advance(s, Event{Kind: Enqueue, Binding: binding, Message: m, At: now}); err == nil {
			t.Fatalf("admitted invalid pure fence %+v", m)
		}
	}
}

func TestDrainPromotionRejectsRetainedDifferentInstance(t *testing.T) {
	now := time.Now()
	f := newDrainCancelEndpoint(now)
	b, from, to := drainCancelRig(t, f)
	b.mu.Lock()
	b.now = func() time.Time { return now }
	b.mu.Unlock()
	f.mu.Lock()
	f.retained.Add(Receipt{Message: Message{ID: "control", Origin: OriginSystem, From: from, To: to, Body: "checkpoint", Deadline: now.Add(PasteTimeout), AcceptedAt: now, DrainInstance: "other-instance"}, Status: Submitted})
	f.mu.Unlock()
	if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint", "expected-instance"); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, b, from, "control", Cancelled)
	select {
	case <-f.delivered:
		t.Fatal("retained conflict repasted")
	default:
	}
}

func TestDrainCancelRejectsReplacementWrapperRevoke(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprint(committed), func(t *testing.T) {
			f := newDrainCancelEndpoint(time.Now())
			f.revokeInstance = "replacement-instance"
			if !committed {
				f.commitGate = make(chan struct{})
				f.deliveryEntered = make(chan struct{})
			}
			b, from, to := drainCancelRig(t, f)
			if _, err := b.SendDrain(context.Background(), "generation", from, "control", to, "checkpoint", "original-instance"); err != nil {
				t.Fatal(err)
			}
			if committed {
				var m Message
				select {
				case m = <-f.delivered:
				case <-time.After(time.Second):
					t.Fatal("no delivery")
				}
				f.mu.Lock()
				f.committed[m.ID] = Receipt{Message: m, Status: Submitted}
				f.mu.Unlock()
			} else {
				select {
				case <-f.deliveryEntered:
				case <-time.After(time.Second):
					t.Fatal("no pending commit")
				}
			}
			if err := b.CancelDrainControl(context.Background(), "generation", "control"); err == nil {
				t.Fatal("replacement wrapper resolved old drain notice")
			}
			if err := b.ReleaseDrain("generation"); err == nil {
				t.Fatal("replacement revocation released old generation")
			}
			if !committed {
				close(f.commitGate)
			}
		})
	}
}
