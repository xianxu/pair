package couchmessage

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

// batchReceiverFake owns a prepared head until the test makes its composer
// ready, then freezes and writes one immutable paste before awaiting confirmation.
type batchReceiverFake struct {
	*FakeDeliveryEndpoint
	prepared chan Message
	ready    chan struct{}
	pastes   chan []Message
	confirm  chan Receipt
}

func newBatchReceiverFake(at time.Time) *batchReceiverFake {
	return &batchReceiverFake{FakeDeliveryEndpoint: newFakeEndpoint(at), prepared: make(chan Message, 1), ready: make(chan struct{}, 1), pastes: make(chan []Message, 1), confirm: make(chan Receipt, 1)}
}

func (f *batchReceiverFake) DeliverBatch(ctx context.Context, m Message, freeze func() ([]Message, error), progress func(Receipt)) (Receipt, error) {
	select {
	case f.prepared <- m:
	case <-ctx.Done():
		return Receipt{}, ctx.Err()
	}
	progress(Receipt{Message: m, Status: Held, Detail: "waiting for recognized composer"})
	select {
	case <-f.ready:
	case <-ctx.Done():
		return Receipt{}, ctx.Err()
	}
	group, err := freeze()
	if err != nil {
		return Receipt{}, err
	}
	select {
	case f.pastes <- group:
	case <-ctx.Done():
		return Receipt{}, ctx.Err()
	}
	select {
	case r := <-f.confirm:
		r.Message = m
		return r, nil
	case <-ctx.Done():
		return Receipt{}, ctx.Err()
	}
}

func batchBrokerFixture(t *testing.T, cap int) (*Broker, Binding, *batchReceiverFake, []Binding) {
	t.Helper()
	now := time.Now()
	b := NewBroker(context.Background(), time.Now, nil)
	t.Cleanup(func() { _ = b.Close() })
	hub := brokerBinding("ops:0")
	f := newBatchReceiverFake(now)
	if err := b.RegisterWithBatching(hub, f, HubAllowance, cap); err != nil {
		t.Fatal(err)
	}
	var senders []Binding
	for i := 1; i <= 5; i++ {
		s := brokerBinding(fmt.Sprintf("pair:%d", i))
		if err := b.Register(s, newFakeEndpoint(now)); err != nil {
			t.Fatal(err)
		}
		senders = append(senders, s)
	}
	return b, hub, f, senders
}

func batchSend(t *testing.T, b *Broker, from, to Binding, id, body string) Message {
	t.Helper()
	r, err := b.Send(context.Background(), from, id, Route{Target: to.Slot}, body)
	if err != nil {
		t.Fatal(err)
	}
	return r.Message
}

func batchPrepared(t *testing.T, f *batchReceiverFake) Message {
	t.Helper()
	select {
	case m := <-f.prepared:
		return m
	case <-testwait.After(t):
		t.Fatal("head never prepared")
		return Message{}
	}
}

func batchPaste(t *testing.T, f *batchReceiverFake) []Message {
	t.Helper()
	select {
	case g := <-f.pastes:
		return g
	case <-testwait.After(t):
		t.Fatal("batch never pasted")
		return nil
	}
}

func batchWaitStatus(t *testing.T, b *Broker, from Binding, id string, want Status) Receipt {
	t.Helper()
	deadline := testwait.After(t)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		r, err := b.Status(from, id)
		if err == nil && r.Status == want {
			return r
		}
		select {
		case <-deadline:
			t.Fatalf("receipt %s=%+v err=%v want=%s", id, r, err, want)
			return Receipt{}
		case <-tick.C:
		}
	}
}

func TestBrokerBatchIncludesReportsArrivingDuringBusyHead(t *testing.T) {
	b, hub, f, senders := batchBrokerFixture(t, DefaultPeerBatchMaxChars)
	originals := []Message{batchSend(t, b, senders[0], hub, "report-1", "first")}
	batchPrepared(t, f)
	for i := 1; i < 5; i++ {
		originals = append(originals, batchSend(t, b, senders[i], hub, fmt.Sprintf("report-%d", i+1), "later report"))
	}
	f.ready <- struct{}{}
	group := batchPaste(t, f)
	if len(group) != 5 {
		t.Fatalf("one ready composer got %d members, want five", len(group))
	}
	for i, m := range group {
		if m.ID != originals[i].ID || m.Body != originals[i].Body || m.From != originals[i].From || !m.AcceptedAt.Equal(originals[i].AcceptedAt) {
			t.Fatalf("member %d changed original", i)
		}
		r, err := b.Status(m.From, m.ID)
		if err != nil || r.Status.Terminal() {
			t.Fatalf("member succeeded before confirmation: %+v err=%v", r, err)
		}
	}
	if strings.Count(BatchEnvelope(group), "\n\n---\n\n") != 4 {
		t.Fatal("batch lost separators")
	}
	f.confirm <- Receipt{Status: Submitted}
	for _, m := range originals {
		batchWaitStatus(t, b, m.From, m.ID, Submitted)
	}
	select {
	case <-f.pastes:
		t.Fatal("confirmation caused another paste")
	default:
	}
}

func TestBrokerBatchCapPreservesOrderAndOverCapSingleton(t *testing.T) {
	b, hub, f, senders := batchBrokerFixture(t, 110)
	var originals []Message
	for i := 0; i < 5; i++ {
		body := "short"
		if i == 1 {
			body = strings.Repeat("界", 120)
		}
		originals = append(originals, batchSend(t, b, senders[i], hub, fmt.Sprint(i), body))
		if i == 0 {
			batchPrepared(t, f)
		}
	}
	var ids []string
	for len(ids) < len(originals) {
		f.ready <- struct{}{}
		group := batchPaste(t, f)
		for _, m := range group {
			ids = append(ids, m.ID)
			if m.ID == "1" && len(group) != 1 {
				t.Fatal("over-cap head joined other members")
			}
		}
		f.confirm <- Receipt{Status: Submitted}
		batchWaitStatus(t, b, group[0].From, group[0].ID, Submitted)
		if len(ids) < len(originals) {
			batchPrepared(t, f)
		}
	}
	if strings.Join(ids, ",") != "0,1,2,3,4" {
		t.Fatalf("FIFO=%v", ids)
	}
}

func TestBrokerBatchDisconnectAnyFrozenMemberCancelsWholeGroup(t *testing.T) {
	for leaving := 0; leaving < 5; leaving++ {
		t.Run(fmt.Sprint(leaving), func(t *testing.T) {
			b, hub, f, senders := batchBrokerFixture(t, DefaultPeerBatchMaxChars)
			var originals []Message
			for i := range senders {
				originals = append(originals, batchSend(t, b, senders[i], hub, fmt.Sprint(i), "report"))
				if i == 0 {
					batchPrepared(t, f)
				}
			}
			f.ready <- struct{}{}
			if g := batchPaste(t, f); len(g) != 5 {
				t.Fatalf("group=%d", len(g))
			}
			b.Disconnect(senders[leaving])
			for _, m := range originals {
				// Receiver can see every receipt even after that sender disconnects.
				batchWaitStatus(t, b, hub, m.ID, Indeterminate)
			}
		})
	}
}

func TestBrokerBatchLateArrivalWaitsForNextComposer(t *testing.T) {
	b, hub, f, senders := batchBrokerFixture(t, DefaultPeerBatchMaxChars)
	first := batchSend(t, b, senders[0], hub, "first", "head")
	batchPrepared(t, f)
	f.ready <- struct{}{}
	if g := batchPaste(t, f); len(g) != 1 {
		t.Fatal(g)
	}
	late := batchSend(t, b, senders[1], hub, "late", "next turn")
	f.confirm <- Receipt{Status: Indeterminate, Detail: "paste incomplete"}
	batchWaitStatus(t, b, hub, first.ID, Indeterminate)
	batchPrepared(t, f)
	select {
	case <-f.pastes:
		t.Fatal("late arrival pasted before next readiness")
	default:
	}
	f.ready <- struct{}{}
	if g := batchPaste(t, f); len(g) != 1 || g[0].ID != late.ID {
		t.Fatalf("next group=%v", g)
	}
	f.confirm <- Receipt{Status: Submitted}
	batchWaitStatus(t, b, hub, late.ID, Submitted)
}

func TestBrokerBatchFailedPasteCannotSubmitAnyMember(t *testing.T) {
	for _, outcome := range []Status{Cancelled, Indeterminate, Expired} {
		t.Run(string(outcome), func(t *testing.T) {
			b, hub, f, senders := batchBrokerFixture(t, DefaultPeerBatchMaxChars)
			for i, from := range senders {
				batchSend(t, b, from, hub, fmt.Sprint(i), "report")
				if i == 0 {
					batchPrepared(t, f)
				}
			}
			f.ready <- struct{}{}
			group := batchPaste(t, f)
			if len(group) != 5 {
				t.Fatalf("group=%d", len(group))
			}
			f.confirm <- Receipt{Status: outcome, Detail: "failed batch paste"}
			for _, m := range group {
				r := batchWaitStatus(t, b, hub, m.ID, outcome)
				if r.Detail != "failed batch paste" {
					t.Fatalf("member lost group evidence: %+v", r)
				}
			}
		})
	}
}

func TestBrokerBatchQueuedSenderDisconnectDoesNotCancelOtherReports(t *testing.T) {
	b, hub, f, senders := batchBrokerFixture(t, DefaultPeerBatchMaxChars)
	for i, from := range senders {
		batchSend(t, b, from, hub, fmt.Sprint(i), "report")
		if i == 0 {
			batchPrepared(t, f)
		}
	}
	b.Disconnect(senders[2])
	batchWaitStatus(t, b, hub, "2", Cancelled)
	f.ready <- struct{}{}
	group := batchPaste(t, f)
	var ids []string
	for _, m := range group {
		ids = append(ids, m.ID)
	}
	if strings.Join(ids, ",") != "0,1,3,4" {
		t.Fatalf("surviving queue=%v", ids)
	}
	f.confirm <- Receipt{Status: Submitted}
	for _, m := range group {
		batchWaitStatus(t, b, hub, m.ID, Submitted)
	}
}

func TestBrokerBatchHeldHubDoesNotBlockAnotherReceiver(t *testing.T) {
	b, hub, f, senders := batchBrokerFixture(t, DefaultPeerBatchMaxChars)
	batchSend(t, b, senders[0], hub, "held", "long turn")
	batchPrepared(t, f)
	other := brokerBinding("bench:0")
	endpoint := newBatchReceiverFake(time.Now())
	if err := b.RegisterWithBatching(other, endpoint, HubAllowance, DefaultPeerBatchMaxChars); err != nil {
		t.Fatal(err)
	}
	m := batchSend(t, b, senders[1], other, "independent", "report")
	batchPrepared(t, endpoint)
	endpoint.ready <- struct{}{}
	if group := batchPaste(t, endpoint); len(group) != 1 || group[0].ID != m.ID {
		t.Fatal(group)
	}
	endpoint.confirm <- Receipt{Status: Submitted}
	batchWaitStatus(t, b, other, m.ID, Submitted)
	if r, err := b.Status(hub, "held"); err != nil || r.Status.Terminal() {
		t.Fatalf("other receiver changed held head: %+v err=%v", r, err)
	}
}
