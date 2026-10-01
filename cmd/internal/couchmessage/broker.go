package couchmessage

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

const AdmissionTimeout = 2 * time.Second
const DeliveryTimeout = 30 * time.Second
const MaxActors = 128
const MaxReceipts = 256

// Observation comes directly from the exact wrapper, not a display cache.
type Observation struct {
	LastActivity time.Time
	Sequence     uint64
	// Submission is the monotonic count of fully written genuine operator submissions.
	Submission uint64
}

// DeliveryEndpoint must honor cancellation and bind every operation to the
// same wrapper incarnation. Reserve conditionally checks Observe's sequence.
type DeliveryEndpoint interface {
	Observe(context.Context) (Observation, error)
	Reserve(context.Context, string, uint64) error
	Release(context.Context, string) error
	Deliver(context.Context, Message) (Receipt, error)
}
type SlotActor struct {
	binding     Binding
	state       ActorState
	endpoint    DeliveryEndpoint
	sequence    uint64
	submission  uint64
	reservation string
	inbox       chan Message
	cancel      context.CancelFunc
}
type admission struct {
	from         Binding
	target, body string
	done         chan struct{}
	receipt      Receipt
	err          error
}
type deliveryJob struct{ cancel context.CancelFunc }
type Broker struct {
	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	now          func() time.Time
	resting      func(context.Context, Binding) (bool, error)
	actors       map[Binding]*SlotActor
	receipts     map[string]Receipt
	destinations map[string]string
	admissions   map[string]*admission
	jobs         map[string]*deliveryJob
	workers      sync.WaitGroup
}

func NewBroker(parent context.Context, now func() time.Time, resting func(context.Context, Binding) (bool, error)) *Broker {
	if parent == nil {
		parent = context.Background()
	}
	if now == nil {
		now = time.Now
	}
	ctx, cancel := context.WithCancel(parent)
	return &Broker{ctx: ctx, cancel: cancel, now: now, resting: resting, actors: map[Binding]*SlotActor{}, receipts: map[string]Receipt{}, destinations: map[string]string{}, admissions: map[string]*admission{}, jobs: map[string]*deliveryJob{}}
}

// apply is called only under mu. Terminal receipts cannot be rewritten by late
// endpoint completion after disconnect cleared the actor's pending message.
func (b *Broker) apply(a *SlotActor, e Event) error {
	next, fx, err := Advance(a.state, e)
	if err != nil {
		return err
	}
	a.state = next
	for _, effect := range fx {
		r := effect.Receipt
		if old, ok := b.receipts[r.Message.ID]; ok && old.Status.Terminal() {
			continue
		}
		b.receipts[r.Message.ID] = r
	}
	return nil
}
func (b *Broker) Register(binding Binding, endpoint DeliveryEndpoint) error {
	if err := binding.Validate(); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ctx.Err(); err != nil {
		return err
	}
	old := b.actors[binding]
	if old != nil && old.state.Connected() {
		return nil
	}
	if old == nil && len(b.actors) >= MaxActors {
		return errors.New("actor capacity reached")
	}
	// Preserve disconnected incarnations as bounded tombstones so reattachment
	// cannot replenish their allowance. Different repository identities coexist.
	for identity, a := range b.actors {
		if identity != binding && identity.Repository == binding.Repository && identity.Slot == binding.Slot && a.state.Connected() {
			b.disconnectLocked(a)
		}
	}
	var state ActorState
	if old != nil {
		state = old.state
	}
	next, _, err := Advance(state, Event{Kind: Register, Binding: binding, At: b.now()})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(b.ctx)
	a := &SlotActor{binding: binding, state: next, endpoint: endpoint, inbox: make(chan Message, 1), cancel: cancel}
	if old != nil {
		a.submission = old.submission
	}
	b.actors[binding] = a
	b.workers.Add(1)
	go b.runActor(ctx, a)
	return nil
}
func (b *Broker) disconnectLocked(a *SlotActor) {
	a.cancel()
	a.sequence++
	a.reservation = ""
	binding := a.state.Binding()
	_ = b.apply(a, Event{Kind: Disconnect, Binding: binding, At: b.now()})
	// A sender disappearing also ends attempts it owns, including messages that
	// have not yet reached a destination worker.
	for _, dest := range b.actors {
		m, ok := dest.state.Pending()
		if !ok || m.From != binding {
			continue
		}
		if job := b.jobs[m.ID]; job != nil {
			job.cancel()
		}
		status := Cancelled
		if b.receipts[m.ID].Status == Delivering {
			status = Indeterminate
		}
		_ = b.apply(dest, Event{Kind: DeliveryFinished, Binding: m.To, Message: m, Status: status, Detail: "sender disconnected", At: b.now()})
	}
}
func (b *Broker) Disconnect(binding Binding) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if a := b.actors[binding]; a != nil {
		b.disconnectLocked(a)
	}
}
func (b *Broker) operatorEvent(binding Binding, kind EventKind) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if a := b.actors[binding]; a != nil && a.state.Connected() {
		a.sequence++
		_ = b.apply(a, Event{Kind: kind, Binding: binding, At: b.now()})
	}
}
func (b *Broker) OperatorInput(binding Binding) { b.operatorEvent(binding, OperatorInput) }

// ReconcileObservation consumes trusted wrapper evidence once per submission
// generation. A delayed notification cannot replenish an already-spent epoch.
// Registration adapters may supply their just-verified handshake observation.
func (b *Broker) ReconcileObservation(binding Binding, observed Observation) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	a := b.actors[binding]
	if a == nil || !b.caller(binding) {
		return ErrUnavailable
	}
	return b.reconcileObservation(a, observed)
}
func (b *Broker) reconcileObservation(a *SlotActor, observed Observation) error {
	if observed.Submission <= a.submission {
		return nil
	}
	at := observed.LastActivity
	if at.IsZero() {
		return errors.New("submission observation has no activity timestamp")
	}
	if err := b.apply(a, Event{Kind: OperatorSubmit, Binding: a.binding, At: at}); err != nil {
		return err
	}
	a.submission = observed.Submission
	return nil
}

// RefreshSubmission treats an operator notification as a wakeup, not authority
// to reset a counter. The receiver supplies its latest monotonic generation.
func (b *Broker) RefreshSubmission(parent context.Context, binding Binding) error {
	ctx, cancel := context.WithTimeout(parent, AdmissionTimeout)
	defer cancel()
	b.mu.Lock()
	a := b.actors[binding]
	valid := a != nil && b.caller(binding)
	b.mu.Unlock()
	if !valid {
		return ErrUnavailable
	}
	if a.endpoint == nil {
		return ErrUnsupported
	}
	observed, err := a.endpoint.Observe(ctx)
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.actors[binding] != a || !b.caller(binding) {
		return ErrUnavailable
	}
	return b.reconcileObservation(a, observed)
}

// ObserveInputThread is called before Console forwards real operator input.
func (b *Broker) ObserveInputThread(scope, tag string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for binding, a := range b.actors {
		if binding.Scope == scope && binding.Tag == tag && a.state.Connected() {
			a.sequence++
			_ = b.apply(a, Event{Kind: OperatorInput, Binding: binding, At: b.now()})
		}
	}
}

// Caller resolves launch context to the exact current wrapper. Ambiguity never
// authorizes a guess, even if a malformed registration duplicated a context.
func (b *Broker) Caller(scope, tag, session, nonce string) (Binding, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var found Binding
	if scope == "" || tag == "" || session == "" || nonce == "" {
		return found, ErrUnavailable
	}
	for binding, a := range b.actors {
		if a.state.Connected() && binding.Scope == scope && binding.Tag == tag && binding.Session == session && binding.Nonce == nonce {
			if found != (Binding{}) {
				return Binding{}, ErrAmbiguous
			}
			found = binding
		}
	}
	if found == (Binding{}) {
		return found, ErrUnavailable
	}
	return found, nil
}
func (b *Broker) sweep() {
	now := b.now()
	for id, r := range b.receipts {
		if r.Status.Terminal() && !r.RetainUntil.IsZero() && !now.Before(r.RetainUntil) {
			delete(b.receipts, id)
			delete(b.destinations, id)
		}
	}
}
func (b *Broker) caller(binding Binding) bool {
	a := b.actors[binding]
	return b.ctx.Err() == nil && a != nil && a.state.Connected()
}
func (b *Broker) Status(caller Binding, id string) (Receipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sweep()
	r, ok := b.receipts[id]
	if !b.caller(caller) || !ok || (r.Message.From != caller && r.Message.To != caller) {
		return Receipt{}, errors.New("receipt unavailable for this actor")
	}
	return r, nil
}
func actorCandidate(a *SlotActor) Candidate {
	_, pending := a.state.Pending()
	return Candidate{Binding: a.state.Binding(), LastActivity: a.state.LastActivity(), Pending: pending || a.reservation != "" || len(a.inbox) != 0, Remaining: a.state.Remaining(), Supported: a.endpoint != nil}
}
func (b *Broker) observe(ctx context.Context, a *SlotActor) (Candidate, Observation, error) {
	b.mu.Lock()
	c := actorCandidate(a)
	sequence := a.sequence
	b.mu.Unlock()
	if !c.Supported {
		return c, Observation{}, ErrUnsupported
	}
	resting := false
	var err error
	if b.resting == nil {
		return c, Observation{}, errors.New("branch observation unavailable")
	}
	resting, err = b.resting(ctx, c.Binding)
	if err != nil {
		return c, Observation{}, err
	}
	obs, err := a.endpoint.Observe(ctx)
	if err != nil {
		return c, obs, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.caller(c.Binding) || b.actors[c.Binding] != a || a.sequence != sequence {
		return c, obs, ErrUnavailable
	}
	if err := b.reconcileObservation(a, obs); err != nil {
		return c, obs, err
	}
	c = actorCandidate(a)
	c.Resting = resting
	c.Known = !obs.LastActivity.IsZero()
	if obs.LastActivity.After(c.LastActivity) {
		c.LastActivity = obs.LastActivity
	}
	return c, obs, nil
}

// Actors uses fresh branch/wrapper observations outside the broker lock. Failed
// observations are visible as Known=false, never as an available recipient.
func (b *Broker) Actors(parent context.Context, caller Binding) ([]Candidate, error) {
	ctx, cancel := context.WithTimeout(parent, AdmissionTimeout)
	defer cancel()
	b.mu.Lock()
	if !b.caller(caller) {
		b.mu.Unlock()
		return nil, ErrUnavailable
	}
	var actors []*SlotActor
	for _, a := range b.actors {
		if a.state.Connected() {
			actors = append(actors, a)
		}
	}
	b.mu.Unlock()
	sort.Slice(actors, func(i, j int) bool { return actors[i].binding.Slot < actors[j].binding.Slot })
	var rows []Candidate
	for _, a := range actors {
		c, _, err := b.observe(ctx, a)
		if err != nil {
			c.Known = false
		}
		b.mu.Lock()
		live := b.caller(c.Binding) && b.actors[c.Binding] == a
		b.mu.Unlock()
		if live {
			rows = append(rows, c)
		}
	}
	b.mu.Lock()
	valid := b.caller(caller)
	b.mu.Unlock()
	if !valid {
		return nil, ErrUnavailable
	}
	return rows, nil
}

// Send acknowledges mailbox admission, never model or task completion.
func (b *Broker) Send(parent context.Context, from Binding, id, target, body string) (Receipt, error) {
	ctx, cancel := context.WithTimeout(parent, AdmissionTimeout)
	defer cancel()
	stop := context.AfterFunc(b.ctx, cancel)
	defer stop()
	if id == "" || len(id) > 128 {
		return Receipt{}, errors.New("invalid message ID")
	}
	if err := ValidateBody(body); err != nil {
		return Receipt{}, err
	}
	b.mu.Lock()
	b.sweep()
	if !b.caller(from) {
		b.mu.Unlock()
		return Receipt{}, ErrUnavailable
	}
	if old, ok := b.receipts[id]; ok {
		same := old.Message.From == from && old.Message.Body == body && b.destinations[id] == target
		b.mu.Unlock()
		if !same {
			return Receipt{}, errors.New("conflicting message ID")
		}
		return old, nil
	}
	if existing := b.admissions[id]; existing != nil {
		same := existing.from == from && existing.body == body && existing.target == target
		b.mu.Unlock()
		if !same {
			return Receipt{}, errors.New("conflicting message ID")
		}
		select {
		case <-existing.done:
			return existing.receipt, existing.err
		case <-ctx.Done():
			return Receipt{}, ctx.Err()
		}
	}
	if len(b.receipts)+len(b.admissions) >= MaxReceipts {
		b.mu.Unlock()
		return Receipt{}, errors.New("receipt capacity reached")
	}
	pending := &admission{from: from, target: target, body: body, done: make(chan struct{})}
	b.admissions[id] = pending
	b.mu.Unlock()
	receipt, err := b.admit(ctx, from, id, target, body)
	b.mu.Lock()
	pending.receipt, pending.err = receipt, err
	delete(b.admissions, id)
	close(pending.done)
	b.mu.Unlock()
	return receipt, err
}
func (b *Broker) admit(ctx context.Context, from Binding, id, target, body string) (Receipt, error) {
	family := !strings.Contains(target, ":")
	tried := map[Binding]bool{}
	for {
		if err := ctx.Err(); err != nil {
			return Receipt{}, err
		}
		b.mu.Lock()
		if !b.caller(from) {
			b.mu.Unlock()
			return Receipt{}, ErrUnavailable
		}
		var candidates []Candidate
		for binding, a := range b.actors {
			if !a.state.Connected() || (family && binding == from) {
				continue
			}
			c := actorCandidate(a)
			// A fresh wrapper observation may contain an unseen human submission.
			// Probe exhausted actors before applying the authoritative budget check.
			c.Remaining = InboundAllowance
			c.Known = true
			c.Resting = true
			if tried[binding] {
				c.Known = false
			}
			candidates = append(candidates, c)
		}
		to, err := ResolveRecipient(target, candidates, b.now())
		if err != nil {
			b.mu.Unlock()
			return Receipt{}, err
		}
		a := b.actors[to]
		a.reservation = id
		seq := a.sequence
		b.mu.Unlock()
		receipt, err := b.tryAdmission(ctx, a, seq, from, id, target, body, family)
		if err == nil {
			return receipt, nil
		}
		b.mu.Lock()
		if a.reservation == id {
			a.reservation = ""
		}
		b.mu.Unlock()
		releaseCtx, releaseCancel := context.WithTimeout(ctx, AdmissionTimeout)
		_ = a.endpoint.Release(releaseCtx, id)
		releaseCancel()
		if !family {
			return Receipt{}, err
		}
		tried[to] = true
	}
}
func (b *Broker) tryAdmission(ctx context.Context, a *SlotActor, seq uint64, from Binding, id, target, body string, family bool) (Receipt, error) {
	// The binding itself never changes on a SlotActor; reconnect installs a new
	// actor retaining the old actor's model state.
	to := a.binding
	var obs Observation
	var err error
	if family {
		var c Candidate
		c, obs, err = b.observe(ctx, a)
		if err == nil && (!c.Known || !c.Resting || b.now().Before(c.LastActivity.Add(QuietInterval))) {
			err = ErrNoRecipient
		}
	} else {
		obs, err = a.endpoint.Observe(ctx)
	}
	if err != nil {
		return Receipt{}, err
	}
	if err = a.endpoint.Reserve(ctx, id, obs.Sequence); err != nil {
		return Receipt{}, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err = ctx.Err(); err != nil {
		return Receipt{}, err
	}
	if b.actors[to] != a || a.reservation != id || !b.caller(to) || a.sequence != seq || !b.caller(from) {
		return Receipt{}, ErrUnavailable
	}
	if err = b.reconcileObservation(a, obs); err != nil {
		return Receipt{}, err
	}
	if len(a.inbox) != 0 {
		return Receipt{}, ErrRecipientBusy
	}
	now := b.now()
	m := Message{ID: id, From: from, To: to, Body: body, Deadline: now.Add(DeliveryTimeout)}
	if err = b.apply(a, Event{Kind: Admit, Binding: to, Message: m, At: now}); err != nil {
		return Receipt{}, err
	}
	b.destinations[id] = target
	a.reservation = ""
	a.inbox <- m
	return b.receipts[id], nil
}
func (b *Broker) runActor(ctx context.Context, a *SlotActor) {
	defer b.workers.Done()
	defer func() {
		for {
			select {
			case m := <-a.inbox:
				ctx, cancel := context.WithTimeout(context.Background(), AdmissionTimeout)
				_ = a.endpoint.Release(ctx, m.ID)
				cancel()
			default:
				return
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case m := <-a.inbox:
			deliveryCtx, cancel := context.WithTimeout(ctx, DeliveryTimeout+ReceiptTimeout)
			job := &deliveryJob{cancel: cancel}
			b.mu.Lock()
			pending, ok := a.state.Pending()
			valid := ok && pending == m && b.actors[m.To] == a && b.caller(m.To) && b.caller(m.From) && ctx.Err() == nil
			if valid {
				b.jobs[m.ID] = job
				_ = b.apply(a, Event{Kind: DeliveryStarted, Binding: m.To, Message: m, At: b.now()})
				pending, ok = a.state.Pending()
				valid = ok && pending == m
			}
			b.mu.Unlock()
			var result Receipt
			var err error
			if valid {
				result, err = a.endpoint.Deliver(deliveryCtx, m)
			} else {
				result.Status = Cancelled
				result.Detail = "delivery binding unavailable"
			}
			cancel()
			status, detail := result.Status, result.Detail
			if err != nil {
				status = Indeterminate
				detail = err.Error()
			}
			if !status.Terminal() {
				status = Indeterminate
				detail = "receiver returned no terminal delivery outcome"
			}
			releaseCtx, releaseCancel := context.WithTimeout(context.Background(), AdmissionTimeout)
			_ = a.endpoint.Release(releaseCtx, m.ID)
			releaseCancel()
			b.mu.Lock()
			if b.jobs[m.ID] == job {
				delete(b.jobs, m.ID)
			}
			_ = b.apply(a, Event{Kind: DeliveryFinished, Binding: m.To, Message: m, Status: status, Detail: detail, At: b.now()})
			b.mu.Unlock()
		}
	}
}
func (b *Broker) Close() error {
	b.mu.Lock()
	b.cancel()
	for _, a := range b.actors {
		b.disconnectLocked(a)
	}
	b.mu.Unlock()
	b.workers.Wait()
	return nil
}
