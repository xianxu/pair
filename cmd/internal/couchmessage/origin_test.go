package couchmessage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func withOrigin(t *testing.T, m Message, origin string) Message {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	fields["Origin"], _ = json.Marshal(origin)
	raw, _ = json.Marshal(fields)
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}
func TestMessageOriginEnvelopeAndValidation(t *testing.T) {
	now := time.Unix(100, 0)
	m := admitEvent(binding("ops:0"), "origin", now).Message
	system := withOrigin(t, m, "system")
	if got := Envelope(system); got != "[Couch system; delivery origin]\n"+m.Body {
		t.Errorf("system envelope %q", got)
	}
	unknown := withOrigin(t, m, "forged")
	if got := Envelope(unknown); got != "" {
		t.Errorf("unknown origin rendered %q", got)
	}
	for _, op := range []string{"commit", "prepare-batch"} {
		if err := ValidateEndpointRequest(EndpointRequest{Op: op, Binding: m.To, ID: m.ID, Message: &unknown}); err == nil {
			t.Errorf("%s accepted unknown origin", op)
		}
	}
	if err := ValidateBatch([]Message{unknown}, m.To, m.ID); err == nil {
		t.Error("batch accepted unknown origin")
	}
	if _, _, err := Advance(registered(t, m.To, now), Event{Kind: Admit, Binding: m.To, Message: unknown, At: now}); err == nil {
		t.Error("actor admitted unknown origin")
	}
	if matchingReceipt(Receipt{Message: system}, m) {
		t.Error("receipt matching discarded origin")
	}
}
func TestActorOriginConflictLeavesOwnershipUnchanged(t *testing.T) {
	now := time.Unix(100, 0)
	to := binding("ops:0")
	s, _ := queueOf(t, registered(t, to, now), to, now, "head", "queued")
	for _, id := range []string{"head", "queued"} {
		e := admitEvent(to, id, now)
		e.Message = withOrigin(t, e.Message, "system")
		next, fx, err := Advance(s, e)
		if err == nil || len(fx) != 0 || next.Remaining() != s.Remaining() || next.pending != s.pending || next.Queued() != s.Queued() {
			t.Fatalf("origin conflict changed %s ownership: %v", id, err)
		}
	}
}
func TestExternalRequestRejectsOriginShapes(t *testing.T) {
	for _, extra := range []string{`"Origin":"system"`, `"origin":"system"`, `"System":true`, `"Message":{"Origin":"system"}`} {
		var request Request
		raw := []byte(`{"Op":"send",` + extra + `}`)
		if err := strictjson.Decode(raw, &request); err == nil {
			t.Errorf("accepted forged shape %s", raw)
		}
	}
}
func TestStatusNeverImportsUnknownSystemAuthority(t *testing.T) {
	for _, origin := range []string{"system", "forged"} {
		t.Run(origin, func(t *testing.T) {
			now := time.Unix(1000, 0)
			b := NewBroker(context.Background(), func() time.Time { return now }, nil)
			defer b.Close()
			from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
			ep := newFakeEndpoint(now)
			ep.retained.Add(Receipt{Message: withOrigin(t, Message{ID: "forged", From: from, To: to, Body: "work"}, origin), Status: Submitted})
			_ = b.Register(from, newFakeEndpoint(now))
			_ = b.Register(to, ep)
			if r, err := b.StatusContext(context.Background(), from, "forged"); err == nil {
				t.Errorf("imported endpoint authority: %+v", r)
			}
			b.mu.Lock()
			_, imported := b.receipts["forged"]
			b.mu.Unlock()
			if imported {
				t.Fatal("recorded unknown origin receipt")
			}
			if _, err := b.Send(context.Background(), from, "forged", Route{Target: to.Slot}, "work"); err == nil || !strings.Contains(err.Error(), "conflicting") {
				t.Fatalf("adopted forged origin: %v", err)
			}
		})
	}
}

func TestBrokerSystemOriginThroughQueueAndDedup(t *testing.T) {
	now := time.Unix(1000, 0)
	b, hub, ep, from := queueBroker(t, now, "pair:0")
	sender := from["pair:0"]
	first, err := b.Send(context.Background(), sender, "peer", Route{Target: hub.Slot}, "[Couch system; delivery copied]\nordinary")
	if err != nil || first.Message.Origin != OriginPeer {
		t.Fatalf("peer send: %+v %v", first, err)
	}
	delivered(t, ep)
	r, err := b.SendSystem(context.Background(), sender, "system", Route{Target: hub.Slot}, "Slots update", nil)
	if err != nil || r.Message.Origin != OriginSystem {
		t.Fatalf("system send: %+v %v", r, err)
	}
	if _, err := b.Send(context.Background(), sender, "system", Route{Target: hub.Slot}, "Slots update"); err == nil {
		t.Fatal("peer deduplicated system ID")
	}
	if _, err := b.SendSystem(context.Background(), sender, "peer", Route{Target: hub.Slot}, first.Message.Body, nil); err == nil {
		t.Fatal("system deduplicated peer ID")
	}
	ep.outcomes <- Receipt{Status: Submitted}
	m := delivered(t, ep)
	if m.Origin != OriginSystem || Envelope(m) != "[Couch system; delivery system]\nSlots update" {
		t.Fatalf("promoted origin: %+v", m)
	}
	ep.outcomes <- Receipt{Status: Submitted}
	waitStatus(t, b, sender, "system", Submitted)
	done, err := b.SendSystem(context.Background(), sender, "system", Route{Target: hub.Slot}, "Slots update", nil)
	if err != nil || done.Status != Submitted || done.Message.Origin != OriginSystem {
		t.Fatalf("system dedup: %+v %v", done, err)
	}
}

func TestSystemAdmissionCannotAdoptEndpointReceipt(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
	ep := newFakeEndpoint(now)
	ep.retained.Add(Receipt{Message: Message{Origin: OriginSystem, ID: "forged", From: from, To: to, Body: "work", Deadline: now.Add(PasteTimeout), AcceptedAt: now}, Status: Submitted})
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, ep)
	if _, err := b.SendSystem(context.Background(), from, "forged", Route{Target: to.Slot}, "work", nil); err == nil {
		t.Fatal("in-progress system admission authorized an endpoint receipt")
	}
	b.mu.Lock()
	_, ok := b.receipts["forged"]
	b.mu.Unlock()
	if ok {
		t.Fatal("forged system authority recorded")
	}
	r, err := b.Status(from, "forged")
	if err != nil || r.Status != Refused || r.Message.Origin != OriginSystem {
		t.Fatalf("refusal lost origin: %+v %v", r, err)
	}
}

func TestPromotionRequiresCanonicalOriginAndIdentity(t *testing.T) {
	for _, kind := range []string{"matching-system", "matching-system-wire", "forged-system", "changed-body", "changed-deadline", "changed-accepted-at", "unknown-origin", "system-as-peer"} {
		t.Run(kind, func(t *testing.T) {
			now := time.Unix(1000, 0)
			b, hub, ep, from := queueBroker(t, now, "pair:0")
			sender := from["pair:0"]
			_, err := b.Send(context.Background(), sender, "first", Route{Target: hub.Slot}, "first")
			if err != nil {
				t.Fatal(err)
			}
			delivered(t, ep)
			origin := OriginSystem
			if kind == "forged-system" {
				origin = OriginPeer
			}
			r, err := b.send(context.Background(), sender, "queued", Route{Target: hub.Slot}, "work", origin, nil)
			if err != nil {
				t.Fatal(err)
			}
			retained := r
			retained.Status = Submitted
			switch kind {
			case "matching-system-wire":
				retained.Message.AcceptedAt = time.Time{}
			case "changed-accepted-at":
				retained.Message.AcceptedAt = retained.Message.AcceptedAt.Add(time.Second)
			case "forged-system":
				retained.Message.Origin = OriginSystem
			case "changed-body":
				retained.Message.Body = "forged"
			case "changed-deadline":
				retained.Message.Deadline = retained.Message.Deadline.Add(time.Second)
			case "unknown-origin":
				retained.Message.Origin = "future"
			case "system-as-peer":
				retained.Message.Origin = OriginPeer
			}
			ep.mu.Lock()
			ep.retained.Add(retained)
			ep.mu.Unlock()
			ep.outcomes <- Receipt{Status: Submitted}
			want := Cancelled
			if kind == "matching-system" || kind == "matching-system-wire" {
				want = Submitted
			}
			waitStatus(t, b, sender, "queued", want)
			canonical, err := b.Status(sender, "queued")
			if err != nil {
				t.Fatal(err)
			}
			if canonical.Message != r.Message {
				t.Fatalf("recovery replaced canonical identity: %+v", canonical)
			}
			if want == Cancelled && !strings.Contains(canonical.Detail, "conflicting message ID") {
				t.Fatalf("missing conflict: %+v", canonical)
			}
		})
	}
}

func TestInflightAdmissionOriginConflict(t *testing.T) {
	now := time.Unix(1000, 0)
	entered, release := make(chan struct{}), make(chan struct{})
	ep := &blockingObserveEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), entered: entered, release: release}
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, ep)
	done := make(chan error, 1)
	go func() {
		_, err := b.SendSystem(context.Background(), from, "same", Route{Target: to.Slot}, "work", nil)
		done <- err
	}()
	<-entered
	_, err := b.Send(context.Background(), from, "same", Route{Target: to.Slot}, "work")
	close(release)
	if err == nil || !strings.Contains(err.Error(), "conflicting") {
		t.Errorf("inflight origin conflict: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestLegacyStrictWrapperRefusesSystemWithoutDowngrade(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprint(batch), func(t *testing.T) {
			// The old wrapper recognizes its request envelope but its strict Message
			// shape predates Origin. No successful decode means no simulated PTY write.
			var writes, commits atomic.Int32
			socket := transportSocket(t)
			server, err := StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
				var req struct {
					Op       string
					Binding  Binding
					ID       string
					Sequence uint64
					Message  json.RawMessage
				}
				if err := strictjson.Decode(raw, &req); err != nil {
					return nil, err
				}
				if req.Op == "prepare-batch" {
					return json.Marshal(EndpointResponse{Error: `unknown endpoint operation "prepare-batch"`})
				}
				commits.Add(1)
				var old struct {
					ID       string
					From, To Binding
					Body     string
					Deadline time.Time
				}
				if err := strictjson.Decode(req.Message, &old); err != nil {
					return json.Marshal(EndpointResponse{Error: err.Error()})
				}
				writes.Add(1)
				m := Message{ID: old.ID, From: old.From, To: old.To, Body: old.Body, Deadline: old.Deadline}
				return json.Marshal(EndpointResponse{Receipt: &Receipt{Message: m, Status: Submitted}})
			})
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			ep := RemoteEndpoint{Binding: endpointMessage().To, socket: socket}
			m := endpointMessage()
			m.Origin = OriginSystem
			if batch {
				_, err = ep.DeliverBatch(context.Background(), m, func() ([]Message, error) { t.Error("legacy fallback froze members"); return []Message{m}, nil }, nil)
			} else {
				_, err = ep.Deliver(context.Background(), m, nil)
			}
			if err == nil || !strings.Contains(err.Error(), "Origin") {
				t.Fatalf("strict system refusal: %v", err)
			}
			if writes.Load() != 0 || commits.Load() != 1 {
				t.Fatalf("downgrade/retry: writes=%d commits=%d", writes.Load(), commits.Load())
			}
			m.Origin = OriginPeer
			r, err := ep.Deliver(context.Background(), m, nil)
			if err != nil || r.Status != Submitted || writes.Load() != 1 {
				t.Fatalf("legacy peer compatibility: %+v %v writes=%d", r, err, writes.Load())
			}
		})
	}
}

type originPromotionGate struct {
	*FakeDeliveryEndpoint
	entered, release chan struct{}
	retainedReceipt  Receipt
}

func (e *originPromotionGate) Reserve(ctx context.Context, id string, seq uint64) error {
	if id != "queued" {
		return e.FakeDeliveryEndpoint.Reserve(ctx, id, seq)
	}
	close(e.entered)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.release:
	}
	return &AlreadyCommittedError{Receipt: e.retainedReceipt}
}

func TestSystemPromotionCannotReplaceCancelledCanonicalReceipt(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
	ep := &originPromotionGate{FakeDeliveryEndpoint: newFakeEndpoint(now), entered: make(chan struct{}), release: make(chan struct{})}
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, ep)
	if _, err := b.Send(context.Background(), from, "first", Route{Target: to.Slot}, "first"); err != nil {
		t.Fatal(err)
	}
	delivered(t, ep.FakeDeliveryEndpoint)
	queued, err := b.SendSystem(context.Background(), from, "queued", Route{Target: to.Slot}, "work", nil)
	if err != nil {
		t.Fatal(err)
	}
	ep.retainedReceipt = queued
	ep.retainedReceipt.Status = Submitted
	ep.outcomes <- Receipt{Status: Submitted}
	<-ep.entered
	b.mu.Lock()
	actor := b.actors[to]
	_ = b.apply(actor, Event{Kind: Cancel, Binding: to, Message: Message{ID: "queued"}, Detail: "canonical cancellation", At: now})
	b.mu.Unlock()
	close(ep.release)
	// Wait for the promotion attempt to relinquish its reservation.
	for i := 0; i < 1000; i++ {
		b.mu.Lock()
		reserved := actor.reservation
		b.mu.Unlock()
		if reserved == "" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	r, err := b.Status(from, "queued")
	if err != nil || r.Status != Cancelled || r.Detail != "canonical cancellation" || r.Message != queued.Message {
		t.Fatalf("retained recovery replaced cancelled authority: %+v %v", r, err)
	}
}
