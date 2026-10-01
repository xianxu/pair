package couchmessage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// FakeDeliveryEndpoint retains reservations and unresolved deliveries. Tests
// choose when delivery completes; it never contacts a real session.
type FakeDeliveryEndpoint struct {
	mu          sync.Mutex
	observation Observation
	reserved    string
	delivered   chan Message
	outcomes    chan Receipt
}

func newFakeEndpoint(at time.Time) *FakeDeliveryEndpoint {
	return &FakeDeliveryEndpoint{observation: Observation{LastActivity: at, Sequence: 1}, delivered: make(chan Message, 1), outcomes: make(chan Receipt, 1)}
}
func (f *FakeDeliveryEndpoint) Observe(context.Context) (Observation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.observation, nil
}
func (f *FakeDeliveryEndpoint) Reserve(_ context.Context, id string, seq uint64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reserved != "" || f.observation.Sequence != seq {
		return errors.New("changed or reserved")
	}
	f.reserved = id
	return nil
}
func (f *FakeDeliveryEndpoint) Release(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reserved == id {
		f.reserved = ""
	}
	return nil
}
func (f *FakeDeliveryEndpoint) Deliver(ctx context.Context, m Message) (Receipt, error) {
	select {
	case f.delivered <- m:
	case <-ctx.Done():
		return Receipt{}, ctx.Err()
	}
	select {
	case r := <-f.outcomes:
		return r, nil
	case <-ctx.Done():
		return Receipt{}, ctx.Err()
	}
}
func brokerBinding(slot string) Binding {
	return Binding{Slot: slot, Repository: "/repo/.git", Scope: "scope", Tag: slot, Session: "session-" + slot, Nonce: "n", PID: 1, Start: "start", Agent: "codex", Version: "test"}
}

func TestBrokerAdmissionIsImmediateAndMailboxIsBounded(t *testing.T) {
	now := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	b := NewBroker(context.Background(), func() time.Time { return now }, func(context.Context, Binding) (bool, error) { return true, nil })
	defer b.Close()
	a, z := brokerBinding("brain:0"), brokerBinding("pair:1")
	fa, fz := newFakeEndpoint(now.Add(-time.Minute)), newFakeEndpoint(now.Add(-time.Minute))
	if err := b.Register(a, fa); err != nil {
		t.Fatal(err)
	}
	if err := b.Register(z, fz); err != nil {
		t.Fatal(err)
	}
	r, err := b.Send(context.Background(), a, "id-1", Route{Target: "pair:1"}, "hello")
	if err != nil || r.Status != Queued {
		t.Fatalf("admission: %+v %v", r, err)
	}
	select {
	case m := <-fz.delivered:
		if m.Body != "hello" {
			t.Fatal(m)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not deliver")
	}
	if _, err = b.Send(context.Background(), a, "id-2", Route{Target: "pair:1"}, "second"); !errors.Is(err, ErrRecipientBusy) {
		t.Fatalf("second send: %v", err)
	}
	duplicate, err := b.Send(context.Background(), a, "id-1", Route{Target: "pair:1"}, "hello")
	if err != nil || duplicate.Message.ID != r.Message.ID {
		t.Fatalf("dedupe: %+v %v", duplicate, err)
	}
	if _, err = b.Send(context.Background(), a, "id-1", Route{Target: "pair:1"}, "changed"); err == nil {
		t.Fatal("conflicting duplicate accepted")
	}
	if _, err = b.Status(brokerBinding("other:0"), r.Message.ID); err == nil {
		t.Fatal("unregistered status caller accepted")
	}
}

func TestBrokerFamilyObservationInvalidatedByOperatorInput(t *testing.T) {
	now := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	var nanos atomic.Int64
	nanos.Store(now.UnixNano())
	entered, release := make(chan struct{}), make(chan struct{})
	b := NewBroker(context.Background(), func() time.Time { return time.Unix(0, nanos.Load()) }, func(context.Context, Binding) (bool, error) { close(entered); <-release; return true, nil })
	defer b.Close()
	a, z := brokerBinding("brain:0"), brokerBinding("pair:1")
	if err := b.Register(a, newFakeEndpoint(now)); err != nil {
		t.Fatal(err)
	}
	if err := b.Register(z, newFakeEndpoint(now)); err != nil {
		t.Fatal(err)
	}
	nanos.Store(now.Add(time.Minute).UnixNano())
	result := make(chan error, 1)
	go func() { _, err := b.Send(context.Background(), a, "id", Route{Target: "pair"}, "hello"); result <- err }()
	<-entered
	b.OperatorInput(z)
	close(release)
	if err := <-result; err == nil {
		t.Fatal("stale quiet observation admitted")
	}
}

func TestBrokerFamilySkipsUnavailableFirstCandidate(t *testing.T) {
	base := time.Unix(1000, 0)
	var nanos atomic.Int64
	nanos.Store(base.UnixNano())
	b := NewBroker(context.Background(), func() time.Time { return time.Unix(0, nanos.Load()) }, func(_ context.Context, v Binding) (bool, error) { return v.Slot != "pair:0", nil })
	defer b.Close()
	from := brokerBinding("brain:0")
	for _, v := range []Binding{from, brokerBinding("pair:0"), brokerBinding("pair:1")} {
		if err := b.Register(v, newFakeEndpoint(base)); err != nil {
			t.Fatal(err)
		}
	}
	nanos.Store(base.Add(time.Minute).UnixNano())
	r, err := b.Send(context.Background(), from, "fallback", Route{Target: "pair"}, "work")
	if err != nil || r.Message.To.Slot != "pair:1" {
		t.Fatalf("fallback %+v %v", r, err)
	}
}
func TestBrokerPreservesAmbiguousFamilies(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from := brokerBinding("brain:0")
	a, z := brokerBinding("pair:1"), brokerBinding("pair:1")
	z.Repository = "/different/.git"
	z.Scope = "other"
	for _, v := range []Binding{from, a, z} {
		if err := b.Register(v, newFakeEndpoint(now)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := b.Send(context.Background(), from, "ambiguous", Route{Target: "pair:1"}, "work"); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("ambiguity: %v", err)
	}
}
func TestBrokerSimultaneousIDAdmittedOnce(t *testing.T) {
	now := time.Unix(1000, 0)
	entered, release := make(chan struct{}), make(chan struct{})
	ep := &blockingObserveEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), entered: entered, release: release}
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, ep)
	results := make(chan Receipt, 2)
	errs := make(chan error, 2)
	send := func() {
		r, e := b.Send(context.Background(), from, "same", Route{Target: "pair:1"}, "work")
		results <- r
		errs <- e
	}
	go send()
	<-entered
	go send()
	close(release)
	for i := 0; i < 2; i++ {
		if e := <-errs; e != nil {
			t.Fatal(e)
		}
		if r := <-results; r.Message.ID != "same" {
			t.Fatal(r)
		}
	}
	b.mu.Lock()
	remaining := -1
	for _, actor := range b.actors {
		if actor.state.Binding() == to {
			remaining = actor.state.Remaining()
		}
	}
	b.mu.Unlock()
	if remaining != 7 {
		t.Fatalf("remaining %d", remaining)
	}
}

type blockingObserveEndpoint struct {
	*FakeDeliveryEndpoint
	entered, release chan struct{}
	once             sync.Once
}

func (f *blockingObserveEndpoint) Observe(ctx context.Context) (Observation, error) {
	f.once.Do(func() { close(f.entered) })
	select {
	case <-f.release:
		return f.FakeDeliveryEndpoint.Observe(ctx)
	case <-ctx.Done():
		return Observation{}, ctx.Err()
	}
}

func TestBrokerDisconnectCancelsSenderWorkAndReconnectRetainsBudget(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
	endpoint := newFakeEndpoint(now)
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, endpoint)
	if _, err := b.Send(context.Background(), from, "disconnect", Route{Target: "pair:1"}, "work"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-endpoint.delivered:
	case <-time.After(time.Second):
		t.Fatal("delivery not started")
	}
	b.Disconnect(from)
	r, err := b.Status(to, "disconnect")
	if err != nil || r.Status != Indeterminate {
		t.Fatalf("sender disconnect %+v %v", r, err)
	}
	b.Disconnect(to)
	_ = b.Register(to, newFakeEndpoint(now))
	b.mu.Lock()
	remaining := b.actors[to].state.Remaining()
	b.mu.Unlock()
	if remaining != 7 {
		t.Fatalf("reconnect replenished %d", remaining)
	}
	if _, err = b.Caller(from.Scope, from.Tag, from.Session, from.Nonce); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("disconnected caller %v", err)
	}
	if _, err = b.Status(from, "disconnect"); err == nil {
		t.Fatal("disconnected caller read receipt")
	}
}
func TestBrokerActorsRecordedObservationsAndInputInvalidation(t *testing.T) {
	base := time.Unix(1000, 0)
	var nanos atomic.Int64
	nanos.Store(base.UnixNano())
	b := NewBroker(context.Background(), func() time.Time { return time.Unix(0, nanos.Load()) }, func(_ context.Context, v Binding) (bool, error) {
		if v.Slot == "pair:2" {
			return false, errors.New("unknown branch")
		}
		return true, nil
	})
	defer b.Close()
	b.SetRestingView(func(v Binding) (bool, bool) { return true, v.Slot != "pair:2" })
	from, to, unknown := brokerBinding("brain:0"), brokerBinding("pair:1"), brokerBinding("pair:2")
	for _, v := range []Binding{from, to, unknown} {
		_ = b.Register(v, newFakeEndpoint(base))
	}
	nanos.Store(base.Add(time.Minute).UnixNano())
	// The listing reports what the wrappers pushed and the resting view.
	for _, v := range []Binding{to, unknown} {
		if err := b.ReconcileObservation(v, Observation{LastActivity: base, Sequence: 1}); err != nil {
			t.Fatal(err)
		}
	}
	b.ObserveInputThread(to.Scope, to.Tag)
	rows, err := b.Actors(context.Background(), from)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		switch r.Binding {
		case to:
			if !r.Known || !r.Resting || !r.LastActivity.Equal(base.Add(time.Minute)) {
				t.Fatalf("input row %+v", r)
			}
		case unknown:
			if r.Known {
				t.Fatal("failed branch became known")
			}
		}
	}
	got, err := b.Caller(from.Scope, from.Tag, from.Session, from.Nonce)
	if err != nil || got != from {
		t.Fatalf("caller %v %v", got, err)
	}
}
// BR-11: tombstones are bounded by eviction, not by refusing new launches.
// Only a table of connected actors refuses; a full table of tombstones gives
// up its longest-disconnected entry, and a new launch of a slot retires that
// slot's old incarnations at once.
func TestBrokerBoundsDisconnectedActors(t *testing.T) {
	var nanos atomic.Int64
	nanos.Store(time.Unix(1000, 0).UnixNano())
	now := func() time.Time { return time.Unix(0, nanos.Add(1)) }
	b := NewBroker(context.Background(), now, nil)
	defer b.Close()
	for i := 0; i < MaxActors; i++ {
		v := brokerBinding(fmt.Sprintf("pair:%d", i))
		if err := b.Register(v, newFakeEndpoint(now())); err != nil {
			t.Fatal(err)
		}
		b.Disconnect(v)
	}
	if err := b.Register(brokerBinding("pair:999"), newFakeEndpoint(now())); err != nil {
		t.Fatalf("full table of tombstones refused a new launch: %v", err)
	}
	b.mu.Lock()
	_, oldest := b.actors[brokerBinding("pair:0")]
	size := len(b.actors)
	b.mu.Unlock()
	if oldest || size != MaxActors {
		t.Fatalf("evicted the wrong tombstone: pair:0 kept=%v size=%d", oldest, size)
	}
	// A new launch in a slot retires that slot's tombstone.
	relaunch := brokerBinding("pair:5")
	relaunch.Nonce = "next-launch"
	if err := b.Register(relaunch, newFakeEndpoint(now())); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	_, stale := b.actors[brokerBinding("pair:5")]
	b.mu.Unlock()
	if stale {
		t.Fatal("relaunched slot kept its old incarnation")
	}
	// A table of connected actors still refuses.
	full := NewBroker(context.Background(), now, nil)
	defer full.Close()
	for i := 0; i < MaxActors; i++ {
		if err := full.Register(brokerBinding(fmt.Sprintf("pair:%d", i)), newFakeEndpoint(now())); err != nil {
			t.Fatal(err)
		}
	}
	if err := full.Register(brokerBinding("pair:999"), newFakeEndpoint(now())); err == nil {
		t.Fatal("registered past a table of connected actors")
	}
}
func TestBrokerSimultaneousFamilyReservationsChooseDistinctSlots(t *testing.T) {
	base := time.Unix(1000, 0)
	var nanos atomic.Int64
	nanos.Store(base.UnixNano())
	b := NewBroker(context.Background(), func() time.Time { return time.Unix(0, nanos.Load()) }, func(context.Context, Binding) (bool, error) { return true, nil })
	defer b.Close()
	from := brokerBinding("brain:0")
	for _, v := range []Binding{from, brokerBinding("pair:1"), brokerBinding("pair:2")} {
		_ = b.Register(v, newFakeEndpoint(base))
	}
	nanos.Store(base.Add(time.Minute).UnixNano())
	result := make(chan Receipt, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func(i int) {
			r, e := b.Send(context.Background(), from, fmt.Sprint(i), Route{Target: "pair"}, "work")
			result <- r
			errs <- e
		}(i)
	}
	for i := 0; i < 2; i++ {
		if e := <-errs; e != nil {
			t.Fatal(e)
		}
	}
	a, z := <-result, <-result
	if a.Message.To == z.Message.To {
		t.Fatal("reserved same recipient twice")
	}
}

func TestBrokerCloseReleasesAcceptedReservations(t *testing.T) {
	for i := 0; i < 30; i++ {
		now := time.Unix(1000, 0)
		b := NewBroker(context.Background(), func() time.Time { return now }, nil)
		from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
		ep := newFakeEndpoint(now)
		_ = b.Register(from, newFakeEndpoint(now))
		_ = b.Register(to, ep)
		if _, err := b.Send(context.Background(), from, "close", Route{Target: "pair:1"}, "work"); err != nil {
			t.Fatal(err)
		}
		_ = b.Close()
		ep.mu.Lock()
		reserved := ep.reserved
		ep.mu.Unlock()
		if reserved != "" {
			t.Fatal("shutdown left accepted reservation")
		}
	}
}

func TestCancelledQueuedMessageStillOccupiesWorkerInbox(t *testing.T) {
	// Cancellation clears the model pending slot before its worker necessarily
	// consumes the queued item. Admission must not block on that stale inbox.
	a := &SlotActor{state: registered(t, brokerBinding("pair:1"), time.Unix(1000, 0)), inbox: make(chan Message, 1), endpoint: newFakeEndpoint(time.Unix(1000, 0))}
	a.inbox <- Message{ID: "cancelled-before-worker-start"}
	if !actorCandidate(a).Pending {
		t.Fatal("cancelled queued item could block next admission under broker mutex")
	}
}

func TestReceiptCapacityIncludesInflightAdmission(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, one, two := brokerBinding("brain:0"), brokerBinding("pair:1"), brokerBinding("pair:2")
	entered, release := make(chan struct{}), make(chan struct{})
	ep := &blockingObserveEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), entered: entered, release: release}
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(one, ep)
	_ = b.Register(two, newFakeEndpoint(now))
	b.mu.Lock()
	for i := 0; i < MaxReceipts-1; i++ {
		b.receipts[fmt.Sprint(i)] = Receipt{Status: Submitted, RetainUntil: now.Add(time.Hour)}
	}
	b.mu.Unlock()
	first := make(chan error, 1)
	go func() {
		_, err := b.Send(context.Background(), from, "last-capacity", Route{Target: "pair:1"}, "work")
		first <- err
	}()
	<-entered
	_, err := b.Send(context.Background(), from, "over-capacity", Route{Target: "pair:2"}, "work")
	close(release)
	if err == nil {
		t.Fatal("inflight admission did not reserve receipt capacity")
	}
	if err := <-first; err != nil {
		t.Fatal(err)
	}
}

func TestSubmissionGenerationIsReconciledBeforeAdmissionAndOnlyOnce(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
	ep := newFakeEndpoint(now)
	_ = b.Register(from, newFakeEndpoint(now))
	_ = b.Register(to, ep)
	send := func(id string) {
		t.Helper()
		ep.outcomes <- Receipt{Status: Submitted}
		r, err := b.Send(context.Background(), from, id, Route{Target: to.Slot}, "work")
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-ep.delivered:
		case <-time.After(time.Second):
			t.Fatal("delivery missing")
		}
		deadline := time.Now().Add(time.Second)
		for {
			r, err = b.Status(from, r.Message.ID)
			if err == nil && r.Status == Submitted {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("receipt %+v %v", r, err)
			}
			time.Sleep(time.Millisecond)
		}
	}
	for i := 0; i < InboundAllowance; i++ {
		send(fmt.Sprintf("before-%d", i))
	}
	ep.mu.Lock()
	ep.observation.Submission = 1
	ep.observation.Sequence++
	ep.mu.Unlock()
	// No notification was received: the next admission must observe the unseen
	// human generation even though the cached allowance is exhausted.
	send("after-0")
	for i := 1; i < 4; i++ {
		send(fmt.Sprintf("after-%d", i))
	}
	// A delayed submit frame and its duplicates must not reset the four spent
	// admissions belonging to the same human submission.
	ep.mu.Lock()
	seen := ep.observation
	ep.mu.Unlock()
	for i := 0; i < 3; i++ {
		if err := b.ReconcileObservation(to, seen); err != nil {
			t.Fatalf("notification %v", err)
		}
	}
	// Nor may a reconnect, which resends the latest observation.
	b.Disconnect(to)
	if err := b.Register(to, ep); err != nil {
		t.Fatal(err)
	}
	if err := b.ReconcileObservation(to, seen); err != nil {
		t.Fatal(err)
	}
	for i := 4; i < InboundAllowance; i++ {
		send(fmt.Sprintf("after-%d", i))
	}
	if _, err := b.Send(context.Background(), from, "ninth-after-human", Route{Target: to.Slot}, "work"); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("ninth admission after one human submission: %v", err)
	}
}

func TestFamilyObservesHumanGenerationWithoutInventingRecentActivity(t *testing.T) {
	base := time.Unix(1000, 0)
	var nanos atomic.Int64
	nanos.Store(base.UnixNano())
	b := NewBroker(context.Background(), func() time.Time { return time.Unix(0, nanos.Load()) }, func(context.Context, Binding) (bool, error) { return true, nil })
	defer b.Close()
	from, to := brokerBinding("brain:0"), brokerBinding("pair:1")
	ep := newFakeEndpoint(base)
	_ = b.Register(from, newFakeEndpoint(base))
	_ = b.Register(to, ep)
	// Exhaust through the same pure admission transition without starting a
	// receiver worker: these are retained historical, already-cancelled attempts.
	b.mu.Lock()
	a := b.actors[to]
	for i := 0; i < InboundAllowance; i++ {
		m := Message{ID: fmt.Sprint("old-", i), From: from, To: to, Body: "work", Deadline: base.Add(time.Second)}
		_ = b.apply(a, Event{Kind: Admit, Binding: to, Message: m, At: base})
		_ = b.apply(a, Event{Kind: DeliveryFinished, Binding: to, Message: m, Status: Cancelled, At: base})
	}
	b.mu.Unlock()
	ep.mu.Lock()
	ep.observation.Submission = 1
	ep.observation.Sequence++
	ep.mu.Unlock()
	nanos.Store(base.Add(QuietInterval).UnixNano())
	r, err := b.Send(context.Background(), from, "fresh-family", Route{Target: "pair"}, "work")
	if err != nil || r.Message.To != to {
		t.Fatalf("fresh human generation excluded from family routing: %+v %v", r, err)
	}
}
