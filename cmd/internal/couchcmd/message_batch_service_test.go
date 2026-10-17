package couchcmd

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// serviceBatchReceiver models one wrapper's prepared composer and confirmation.
// It cannot freeze while busy or complete delivery before explicit confirmation.
type serviceBatchReceiver struct {
	serviceEndpointFake
	prepared  chan couchmessage.Message
	ready     chan struct{}
	pasted    chan []couchmessage.Message
	confirmed chan struct{}
	legacy    chan couchmessage.Message
}

func newServiceBatchReceiver() *serviceBatchReceiver {
	return &serviceBatchReceiver{prepared: make(chan couchmessage.Message, 8), ready: make(chan struct{}, 8), pasted: make(chan []couchmessage.Message, 8), confirmed: make(chan struct{}, 8), legacy: make(chan couchmessage.Message, 8)}
}
func (f *serviceBatchReceiver) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	select {
	case f.legacy <- m:
	case <-ctx.Done():
		return couchmessage.Receipt{}, ctx.Err()
	}
	return couchmessage.Receipt{Message: m, Status: couchmessage.Submitted}, nil
}
func (f *serviceBatchReceiver) DeliverBatch(ctx context.Context, head couchmessage.Message, freeze func() ([]couchmessage.Message, error), progress func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	select {
	case f.prepared <- head:
	case <-ctx.Done():
		return couchmessage.Receipt{}, ctx.Err()
	}
	if progress != nil {
		progress(couchmessage.Receipt{Message: head, Status: couchmessage.Held, Detail: "waiting for recognized composer"})
	}
	select {
	case <-f.ready:
	case <-ctx.Done():
		return couchmessage.Receipt{}, ctx.Err()
	}
	group, err := freeze()
	if err != nil {
		return couchmessage.Receipt{}, err
	}
	select {
	case f.pasted <- group:
	case <-ctx.Done():
		return couchmessage.Receipt{}, ctx.Err()
	}
	select {
	case <-f.confirmed:
		return couchmessage.Receipt{Message: head, Status: couchmessage.Submitted}, nil
	case <-ctx.Done():
		return couchmessage.Receipt{}, ctx.Err()
	}
}
func serviceBatchReceive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-testwait.After(t):
		t.Fatal("receiver gate not reached")
		var zero T
		return zero
	}
}
func serviceBatchConnect(t *testing.T, r *serviceRig, n int, agent string, endpoint couchmessage.DeliveryEndpoint) couchmessage.Binding {
	t.Helper()
	b := r.world.add(n)
	b.Agent = agent
	r.world.set(b, func(slot *worldSlot) { slot.binding = b })
	if endpoint != nil {
		r.world.mu.Lock()
		r.world.endpoints[b] = endpoint
		r.world.mu.Unlock()
	}
	r.attach(b, couchmessage.PaneHandle(fmt.Sprintf("batch-pane-%d", n)))
	r.wrapper(b)
	r.waitConnected(b, true)
	return b
}
func serviceBatchSend(t *testing.T, r *serviceRig, from, to couchmessage.Binding, id string) couchmessage.Message {
	t.Helper()
	req := callerRequest(from, "send")
	req.ID, req.Target, req.Body = id, to.Slot, "landed "+id
	var response couchmessage.Response
	if err := couchmessage.Call(context.Background(), r.brokerSock, req, &response); err != nil {
		t.Fatal(err)
	}
	if response.Receipt == nil || (response.Code != "accepted" && response.Code != "ok") {
		t.Fatalf("send %+v", response)
	}
	return response.Receipt.Message
}
func serviceBatchStatus(t *testing.T, r *serviceRig, from couchmessage.Binding, id string) couchmessage.Receipt {
	t.Helper()
	req := callerRequest(from, "status")
	req.ID = id
	var response couchmessage.Response
	if err := couchmessage.Call(context.Background(), r.brokerSock, req, &response); err != nil {
		t.Fatal(err)
	}
	if response.Receipt == nil {
		t.Fatalf("status %+v", response)
	}
	return *response.Receipt
}
func serviceBatchWaitSubmitted(t *testing.T, r *serviceRig, m couchmessage.Message) {
	t.Helper()
	deadline := testwait.After(t)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		receipt := serviceBatchStatus(t, r, m.From, m.ID)
		if receipt.Status == couchmessage.Submitted {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("not submitted: %+v", receipt)
		case <-tick.C:
		}
	}
}

func TestMessageBatchServiceSelectsOnlyVerifiedClaudeHub(t *testing.T) {
	for _, tc := range []struct {
		name, agent, hub string
		batch            bool
	}{
		{"codex hub", "codex", "/repo0", false},
		{"claude hub", "claude", "/repo0", true},
		{"codex worker", "codex", "/repo99", false},
		{"claude worker", "claude", "/repo99", false},
		{"qoder hub", "qoder", "/repo0", false},
		{"unconfigured hub", "codex", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newServiceRig(t, func(a messageAuthority) messageAuthority { a.hub = tc.hub; return a })
			fake := newServiceBatchReceiver()
			to := serviceBatchConnect(t, r, 0, tc.agent, fake)
			from := r.connect(1)
			original := serviceBatchSend(t, r, from, to, "policy")
			if tc.batch {
				select {
				case <-fake.prepared:
				case m := <-fake.legacy:
					t.Fatalf("verified %s hub used singleton: %+v", tc.agent, m)
				case <-testwait.After(t):
					t.Fatal("hub never prepared")
				}
				fake.ready <- struct{}{}
				group := serviceBatchReceive(t, fake.pasted)
				if len(group) != 1 || group[0].ID != original.ID {
					t.Fatalf("group %+v", group)
				}
				fake.confirmed <- struct{}{}
			} else {
				select {
				case <-fake.legacy:
				case m := <-fake.prepared:
					t.Fatalf("nonqualified receiver prepared batch %+v", m)
				case <-testwait.After(t):
					t.Fatal("singleton never delivered")
				}
			}
			serviceBatchWaitSubmitted(t, r, original)
		})
	}
}

func TestMessageBatchServiceFreezesFiveAndAccountsEachAfterConfirmation(t *testing.T) {
	sink := &factSink{}
	log := testFactLog(t, sink)
	r := newServiceRig(t, func(a messageAuthority) messageAuthority { a.hub = "/repo0"; return a })
	fake := newServiceBatchReceiver()
	hub := serviceBatchConnect(t, r, 0, "claude", fake)
	senders := make([]couchmessage.Binding, 6)
	for i := range senders {
		senders[i] = r.connect(i + 1)
	}
	store := t.TempDir()
	t.Cleanup(r.s.Close) // stop accounting before its temporary ledger is removed
	r.s.recordDeliveries(store)
	r.s.facts.Store(log)
	ledger := &sessionsync.Ledger{Dir: sessionsync.LedgerDir(store)}
	originals := []couchmessage.Message{serviceBatchSend(t, r, senders[0], hub, "report-1")}
	select {
	case <-fake.prepared:
	case <-fake.legacy:
		t.Fatal("hub did not use batch adapter")
	case <-testwait.After(t):
		t.Fatal("no prepare")
	}
	for i := 1; i < 5; i++ {
		originals = append(originals, serviceBatchSend(t, r, senders[i], hub, fmt.Sprintf("report-%d", i+1)))
	}
	fake.ready <- struct{}{}
	group := serviceBatchReceive(t, fake.pasted)
	if len(group) != 5 {
		t.Fatalf("one ready composer froze %d reports, want5", len(group))
	}
	for i, m := range group {
		if m.ID != originals[i].ID || m.From != originals[i].From || m.Body != originals[i].Body || !m.AcceptedAt.Equal(originals[i].AcceptedAt) {
			t.Fatalf("FIFO original %d changed %+v", i, m)
		}
		if receipt := serviceBatchStatus(t, r, m.From, m.ID); receipt.Status.Terminal() {
			t.Fatalf("success before confirmation: %+v", receipt)
		}
	}
	if rows, err := ledger.Read(hub.Slot); err != nil || len(rows) != 0 {
		t.Fatalf("recorded before confirmation: %+v %v", rows, err)
	}
	for _, fact := range sink.rows(t) {
		if fact.Event == "hub_delivery" {
			t.Fatal("delivery fact recorded before confirmation")
		}
	}
	late := serviceBatchSend(t, r, senders[5], hub, "report-6")
	if receipt := serviceBatchStatus(t, r, late.From, late.ID); receipt.Status.Terminal() {
		t.Fatalf("late report completed early %+v", receipt)
	}
	fake.confirmed <- struct{}{}
	for _, m := range originals {
		serviceBatchWaitSubmitted(t, r, m)
	}
	next := serviceBatchReceive(t, fake.prepared)
	if next.ID != late.ID {
		t.Fatalf("next prepared %+v", next)
	}
	rows, err := ledger.Read(hub.Slot)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatalf("one confirmation recorded %d original reports, want5: %+v", len(rows), rows)
	}
	for i, row := range rows {
		m := originals[i]
		if row.ID != m.ID || row.From != m.From.Slot || row.Head != m.Body || row.Uncertain || !row.AddressedAt.Equal(m.AcceptedAt) {
			t.Fatalf("per-message accounting changed %+v", row)
		}
	}
	deadline := testwait.After(t)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		facts := map[string]supervisionFact{}
		for _, fact := range sink.rows(t) {
			if fact.Event == "hub_delivery" {
				facts[fact.Message] = fact
			}
		}
		if len(facts) == 5 {
			for _, m := range originals {
				fact, ok := facts[log.hash("message", m.ID)]
				if !ok || fact.Sender != log.subject(m.From) || fact.Subject != log.subject(hub) || fact.Outcome != "confirmed" || fact.Generated == nil || *fact.Generated {
					t.Fatalf("per-message fact missing or changed: %+v", fact)
				}
			}
			break
		}
		select {
		case <-deadline:
			t.Fatalf("batch emitted %d distinct delivery facts, want5", len(facts))
		case <-tick.C:
		}
	}
	fake.ready <- struct{}{}
	second := serviceBatchReceive(t, fake.pasted)
	if len(second) != 1 || second[0].ID != late.ID {
		t.Fatalf("late arrival joined old group %+v", second)
	}
	fake.confirmed <- struct{}{}
	serviceBatchWaitSubmitted(t, r, late)
}

type serviceBatchLegacyEndpoint struct {
	serviceEndpointFake
	delivered chan couchmessage.Message
}

func (f serviceBatchLegacyEndpoint) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	select {
	case f.delivered <- m:
		return couchmessage.Receipt{Message: m, Status: couchmessage.Submitted}, nil
	case <-ctx.Done():
		return couchmessage.Receipt{}, ctx.Err()
	}
}
func TestMessageBatchServiceLegacyUnderlyingEndpointRemainsSingleton(t *testing.T) {
	r := newServiceRig(t, func(a messageAuthority) messageAuthority { a.hub = "/repo0"; return a })
	legacy := serviceBatchLegacyEndpoint{delivered: make(chan couchmessage.Message, 1)}
	to := serviceBatchConnect(t, r, 0, "claude", legacy)
	from := r.connect(1)
	m := serviceBatchSend(t, r, from, to, "old-endpoint")
	got := serviceBatchReceive(t, legacy.delivered)
	if got.ID != m.ID {
		t.Fatalf("singleton changed %+v", got)
	}
	serviceBatchWaitSubmitted(t, r, m)
}

func TestMessageBatchAdapterChecksLiveIdentityBeforePreparation(t *testing.T) {
	r := newServiceRig(t)
	b := r.connect(1)
	fake := newServiceBatchReceiver()
	adapter, ok := any(messageEndpoint{service: r.s, binding: b, endpoint: fake}).(couchmessage.BatchDeliveryEndpoint)
	if !ok {
		t.Fatal("message adapter lacks batch delivery")
	}
	for _, tc := range []struct {
		name   string
		change func(*worldSlot)
	}{
		{"pid replaced", func(s *worldSlot) { s.pidFile++ }},
		{"session replaced", func(s *worldSlot) { s.recorded = false }},
		{"terminal gone", func(s *worldSlot) { s.live = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r.world.set(b, tc.change)
			called := false
			receipt, err := adapter.DeliverBatch(context.Background(), couchmessage.Message{ID: "obsolete", To: b}, func() ([]couchmessage.Message, error) { called = true; return nil, nil }, nil)
			if err != nil || receipt.Status != couchmessage.Cancelled || called {
				t.Fatalf("obsolete identity reached freeze: %+v %v called=%v", receipt, err, called)
			}
			select {
			case <-fake.prepared:
				t.Fatal("obsolete identity reached endpoint prepare")
			default:
			}
			r.world.set(b, func(s *worldSlot) { s.pidFile, s.recorded, s.live = b.PID, true, true })
		})
	}
}

func TestMessageBatchServiceUsesConfiguredCap(t *testing.T) {
	r := newServiceRig(t, func(a messageAuthority) messageAuthority { a.hub = "/repo0"; a.batchMaxChars = 1; return a })
	fake := newServiceBatchReceiver()
	hub := serviceBatchConnect(t, r, 0, "claude", fake)
	from := r.connect(1)
	first := serviceBatchSend(t, r, from, hub, "cap-first")
	select {
	case <-fake.prepared:
	case <-fake.legacy:
		t.Fatal("configured hub did not prepare batch")
	case <-testwait.After(t):
		t.Fatal("no prepare")
	}
	second := serviceBatchSend(t, r, from, hub, "cap-second")
	fake.ready <- struct{}{}
	group := serviceBatchReceive(t, fake.pasted)
	if len(group) != 1 || group[0].ID != first.ID {
		t.Fatalf("configured cap ignored: %+v", group)
	}
	fake.confirmed <- struct{}{}
	serviceBatchWaitSubmitted(t, r, first)
	if head := serviceBatchReceive(t, fake.prepared); head.ID != second.ID {
		t.Fatalf("wrong remaining head %+v", head)
	}
	fake.ready <- struct{}{}
	group = serviceBatchReceive(t, fake.pasted)
	if len(group) != 1 || group[0].ID != second.ID {
		t.Fatalf("over-cap singleton not delivered %+v", group)
	}
	fake.confirmed <- struct{}{}
	serviceBatchWaitSubmitted(t, r, second)
}
