package couchmessage

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const AdmissionTimeout = 2 * time.Second

// PasteTimeout bounds the initial reservation/commit exchange. Accepted
// composer waits follow receiver lifetime; DeliveryTimeout starts at paste.
const PasteTimeout = 90 * time.Second
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
// Deliver hands progress each interim receipt it observes (pair#435).
type DeliveryEndpoint interface {
	Observe(context.Context) (Observation, error)
	Reserve(context.Context, string, uint64) error
	Release(context.Context, string) error
	Deliver(ctx context.Context, m Message, progress func(Receipt)) (Receipt, error)
}
type SlotActor struct {
	binding     Binding
	state       ActorState
	endpoint    DeliveryEndpoint
	sequence    uint64
	submission  uint64
	reservation string
	inbox       chan Message
	// wake tells the worker a message joined the queue (pair#492).
	wake chan struct{}
	// blockedPromotions counts the worker finding the queue blocked by
	// another attempt's reservation: the interleaving the close review
	// required a test of, observable without timing.
	blockedPromotions int
	batchMaxChars     int // nonzero only for explicitly enabled hub receivers
	cancel            context.CancelFunc
	// The latest observation the wrapper pushed (#365), kept so an actor
	// listing reads memory instead of probing every slot. It stays current
	// while the actor is connected: the wrapper pushes every change.
	observed   Observation
	observedAt time.Time
	// disconnectedAt orders tombstones for eviction.
	disconnectedAt time.Time
}
type admission struct {
	origin  Origin
	from    Binding
	route   Route
	body    string
	done    chan struct{}
	receipt Receipt
	err     error
}
type deliveryJob struct {
	cancel context.CancelFunc
	done   chan struct{}
}
type Broker struct {
	familyEligible     func(Binding, Binding) bool
	submissionObserver func(Binding, uint64)
	mu                 sync.Mutex
	ctx                context.Context
	cancel             context.CancelFunc
	now                func() time.Time
	resting            func(context.Context, Binding) (bool, error)
	restingView        func(Binding) (resting, known bool)
	niceView           func(Binding) (NiceFacts, bool)
	families           func(context.Context) (map[string]string, error)
	actors             map[Binding]*SlotActor
	receipts           map[string]Receipt
	refusals           map[string]refusal
	destinations       map[string]Route
	admissions         map[string]*admission
	jobs               map[string]*deliveryJob
	workers            sync.WaitGroup
	drainGeneration    string
	drainCancelling    bool
	drainControls      map[string]string
	drainRevoked       map[string]bool
}

// SetFamilyEligibility installs the supervisor's trusted ownership projection.
// It filters implicit selection only, after global repository identity checks.
// Install before serving; the callback must read memory only.
func (b *Broker) SetFamilyEligibility(eligible func(Binding, Binding) bool) {
	b.familyEligible = eligible
}
func (b *Broker) eligible(from, to Binding) bool {
	return b.familyEligible == nil || b.familyEligible(from, to)
}

// SetFamilies installs the enrolled-repository source: every enrolled family
// (directory name) mapped to its alias, "" when none. Routing resolves names
// against it as well as live bindings. It must be called before the broker
// serves requests; nil means live bindings only.
func (b *Broker) SetFamilies(families func(context.Context) (map[string]string, error)) {
	b.families = families
}

// SetRestingView installs the listing's in-memory resting-branch source (the
// Console's slot-git cache). Listings never run git; nil leaves every row's
// resting state unknown. Call before the broker serves requests.
func (b *Broker) SetRestingView(view func(Binding) (resting, known bool)) {
	b.restingView = view
}

// SetNiceView installs the slot renicer's facts (#502), read by listings
// from memory; nil leaves them out. It may come after the broker serves:
// listings read it under the broker's lock.
func (b *Broker) SetNiceView(view func(Binding) (NiceFacts, bool)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.niceView = view
}

// Connected is every connected slot's binding: who the renicer looks at.
func (b *Broker) Connected() []Binding {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []Binding
	for _, a := range b.actors {
		if a.state.Connected() {
			out = append(out, a.binding)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slot < out[j].Slot })
	return out
}

// readFamilies runs outside the broker lock: the source may do store IO.
func (b *Broker) readFamilies(ctx context.Context) (map[string]string, error) {
	if b.families == nil {
		return nil, nil
	}
	families, err := b.families(ctx)
	if err != nil {
		return nil, fmt.Errorf("read enrolled repositories for routing (retry; if it persists, check the couch store and its repository-aliases.json, which can be removed to reset aliases): %w", err)
	}
	return families, nil
}

func NewBroker(parent context.Context, now func() time.Time, resting func(context.Context, Binding) (bool, error)) *Broker {
	if parent == nil {
		parent = context.Background()
	}
	if now == nil {
		now = time.Now
	}
	ctx, cancel := context.WithCancel(parent)
	return &Broker{ctx: ctx, cancel: cancel, now: now, resting: resting, actors: map[Binding]*SlotActor{}, receipts: map[string]Receipt{}, refusals: map[string]refusal{}, destinations: map[string]Route{}, admissions: map[string]*admission{}, jobs: map[string]*deliveryJob{}}
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
	return b.RegisterWithAllowance(binding, endpoint, InboundAllowance)
}

// RegisterWithAllowance sizes a new incarnation's bucket; a reconnecting one
// keeps its own (#433).
func (b *Broker) RegisterWithAllowance(binding Binding, endpoint DeliveryEndpoint, allowance int) error {
	return b.register(binding, endpoint, allowance, 0)
}

// RegisterWithBatching is selected by the verified hub adapter, never by senders.
func (b *Broker) RegisterWithBatching(binding Binding, endpoint DeliveryEndpoint, allowance, maxChars int) error {
	if maxChars < 1 || maxChars > MaxPeerBatchChars || binding.Agent != "claude" {
		return errors.New("invalid peer batch receiver or character cap")
	}
	return b.register(binding, endpoint, allowance, maxChars)
}

func (b *Broker) register(binding Binding, endpoint DeliveryEndpoint, allowance, batchMaxChars int) error {
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
	// A disconnected incarnation stays as a tombstone so the same binding
	// reconnecting (an exec, a reattach) cannot replenish its allowance. A
	// different binding for the slot is a new launch that starts fresh, so it
	// retires the slot's other incarnations outright; their receipts live on
	// in b.receipts. Different repository identities coexist (#365 BR-11).
	for identity, a := range b.actors {
		if identity != binding && identity.Repository == binding.Repository && identity.Slot == binding.Slot {
			if a.state.Connected() {
				b.disconnectLocked(a)
			}
			delete(b.actors, identity)
		}
	}
	if old == nil && len(b.actors) >= MaxActors && !b.evictOldestTombstoneLocked() {
		return errors.New("actor capacity reached")
	}
	var state ActorState
	if old != nil {
		state = old.state
	}
	next, _, err := Advance(state, Event{Kind: Register, Binding: binding, Allowance: allowance, At: b.now()})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(b.ctx)
	a := &SlotActor{binding: binding, state: next, endpoint: endpoint, inbox: make(chan Message, 1), wake: make(chan struct{}, 1), cancel: cancel, batchMaxChars: batchMaxChars}
	if old != nil {
		a.submission = old.submission
	}
	b.actors[binding] = a
	b.workers.Add(1)
	go b.runActor(ctx, a)
	return nil
}

// evictOldestTombstoneLocked bounds tombstones by capacity rather than by
// time: when the table is full, the longest-disconnected actor goes. Only a
// table of connected actors refuses a new registration.
func (b *Broker) evictOldestTombstoneLocked() bool {
	var oldest Binding
	var at time.Time
	found := false
	for identity, a := range b.actors {
		if !a.state.Connected() && (!found || a.disconnectedAt.Before(at)) {
			oldest, at, found = identity, a.disconnectedAt, true
		}
	}
	if !found {
		return false
	}
	delete(b.actors, oldest)
	return true
}

func (b *Broker) disconnectLocked(a *SlotActor) {
	a.disconnectedAt = b.now()
	a.cancel()
	a.sequence++
	a.reservation = ""
	binding := a.state.Binding()
	_ = b.apply(a, Event{Kind: Disconnect, Binding: binding, At: b.now()})
	// A sender disappearing also ends attempts it owns, including messages that
	// have not yet reached a destination worker.
	for _, dest := range b.actors {
		m, ok := dest.state.Pending()
		if !ok {
			continue
		}
		owns := false
		for _, member := range dest.state.PendingMessages() {
			owns = owns || member.From == binding
		}
		if !owns {
			continue
		}
		status := Cancelled
		if job := b.jobs[m.ID]; job != nil {
			job.cancel()
			status = Indeterminate
		}
		_ = b.apply(dest, Event{Kind: DeliveryFinished, Binding: m.To, Message: m, Status: status, Detail: "sender disconnected", At: b.now()})
	}
	for _, dest := range b.actors {
		if dest != a && dest.state.Queued() > 0 {
			_ = b.apply(dest, Event{Kind: Cancel, Binding: dest.state.Binding(), Message: Message{From: binding}, Detail: "sender disconnected before delivery", At: b.now()})
		}
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

// Submission reports the accepted genuine operator-submission generation for
// a connected binding. It reads memory only; stale or rejected observations
// never advance this counter.
func (b *Broker) Submission(binding Binding) (uint64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	a := b.actors[binding]
	if a == nil || !a.state.Connected() {
		return 0, false
	}
	return a.submission, true
}

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
	a.observed, a.observedAt = observed, b.now()
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
	if b.submissionObserver != nil {
		b.submissionObserver(a.binding, a.submission)
	}
	return nil
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
	b.sweepRefusalsLocked(now)
	for id, r := range b.receipts {
		if b.drainControls[id] == "" && r.Status.Terminal() && !r.RetainUntil.IsZero() && !now.Before(r.RetainUntil) {
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
	r, ok := b.statusLocked(caller, id)
	if !b.caller(caller) || !ok {
		return Receipt{}, errors.New("receipt unavailable for this actor")
	}
	return r, nil
}

// statusLocked shares receipt visibility and current waiting projections
// between exact-ID reads and status-prefix resolution.
func (b *Broker) statusLocked(caller Binding, id string) (Receipt, bool) {
	r, ok := b.receiptLocked(id)
	if !ok {
		if refused, found := b.refusals[id]; found {
			r, ok = refused.receipt, true
		}
	}
	return r, ok && (r.Message.From == caller || r.Message.To == caller)
}

// ReceiptSnapshot reads the supervisor's retained receipt without a live
// caller. It performs no endpoint IO and keeps the normal terminal receipt retention
// rules, including when the sender has disconnected. The final bool reports an
// active delivery callback: disconnection can make its receipt terminal before
// endpoint accounting finishes. Consumers must retain provenance until it ends.
func (b *Broker) ReceiptSnapshot(id string) (Receipt, bool, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sweep()
	r, known := b.receiptLocked(id)
	return r, known, b.jobs[id] != nil
}

// receiptLocked adds the current waiting-line position and the receiver's
// standing composer wait to nonterminal pre-paste receipts.
func (b *Broker) receiptLocked(id string) (Receipt, bool) {
	r, ok := b.receipts[id]
	a := b.actors[r.Message.To]
	if ok && a != nil {
		pending, _ := a.state.Pending()
		r = a.state.waitingReceipt(r, b.receipts[pending.ID])
	}
	return r, ok
}
func actorCandidate(a *SlotActor, now time.Time) Candidate {
	_, pending := a.state.Pending()
	return Candidate{Binding: a.state.Binding(), LastActivity: a.state.LastActivity(), Pending: pending || a.reservation != "" || len(a.inbox) != 0 || a.state.Queued() > 0, Remaining: a.state.RemainingAt(now), Capacity: a.state.Capacity(), Supported: a.endpoint != nil}
}
func (b *Broker) observe(ctx context.Context, a *SlotActor) (Candidate, Observation, error) {
	b.mu.Lock()
	c := actorCandidate(a, b.now())
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
	c = actorCandidate(a, b.now())
	c.Resting = resting
	c.Known = !obs.LastActivity.IsZero()
	if obs.LastActivity.After(c.LastActivity) {
		c.LastActivity = obs.LastActivity
	}
	return c, obs, nil
}

// Actors reports the broker's memory: each connected actor's last pushed
// observation and the resting view. It does no IO, so its cost does not grow
// with slot count or probe latency. A missing observation or unknown resting
// state is Known=false, never an available recipient; admission still
// observes fresh before it reserves.
func (b *Broker) Actors(parent context.Context, caller Binding) ([]Candidate, error) {
	b.mu.Lock()
	if !b.caller(caller) {
		b.mu.Unlock()
		return nil, ErrUnavailable
	}
	var rows []Candidate
	for _, a := range b.actors {
		if !a.state.Connected() {
			continue
		}
		c := actorCandidate(a, b.now())
		if a.observed.LastActivity.After(c.LastActivity) {
			c.LastActivity = a.observed.LastActivity
		}
		restingKnown := false
		if b.restingView != nil {
			c.Resting, restingKnown = b.restingView(a.binding)
		}
		c.Known = !c.LastActivity.IsZero() && !a.observedAt.IsZero() && restingKnown
		if b.niceView != nil {
			if facts, ok := b.niceView(a.binding); ok {
				c.Nice = &facts
			}
		}
		rows = append(rows, c)
	}
	b.mu.Unlock()
	sort.Slice(rows, func(i, j int) bool { return rows[i].Binding.Slot < rows[j].Binding.Slot })
	// Aliases are display-only here: a store that stays busy leaves them out
	// rather than failing the listing.
	ctx, cancel := context.WithTimeout(parent, 250*time.Millisecond)
	defer cancel()
	if families, err := b.readFamilies(ctx); err == nil {
		for i := range rows {
			if family, _, err := parseSlot(rows[i].Binding.Slot); err == nil {
				rows[i].Alias = families[family]
			}
		}
	}
	return rows, nil
}

// Send acknowledges mailbox admission, never model or task completion.
func (b *Broker) Send(parent context.Context, from Binding, id string, route Route, body string) (Receipt, error) {
	return b.send(parent, from, id, route, body, OriginPeer)
}

// SendSystem admits a supervisor-generated notice through the ordinary mailbox.
// It is deliberately absent from the external request protocol.
func (b *Broker) SendSystem(parent context.Context, from Binding, id string, route Route, body string) (Receipt, error) {
	return b.send(parent, from, id, route, body, OriginSystem)
}

func (b *Broker) send(parent context.Context, from Binding, id string, route Route, body string, origin Origin) (Receipt, error) {
	if !origin.Valid() {
		return Receipt{}, errors.New("unknown message origin")
	}

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
	if b.drainGeneration != "" {
		b.mu.Unlock()
		return Receipt{}, ErrDraining
	}
	if !b.caller(from) {
		b.mu.Unlock()
		return Receipt{}, ErrUnavailable
	}
	if old, ok := b.receiptLocked(id); ok {
		same := old.Message.Origin == origin && old.Message.From == from && old.Message.Body == body && b.destinations[id] == route
		b.mu.Unlock()
		if !same {
			return Receipt{}, errors.New("conflicting message ID")
		}
		return old, nil
	}
	if existing := b.admissions[id]; existing != nil {
		same := existing.origin == origin && existing.from == from && existing.body == body && existing.route == route
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
	pending := &admission{origin: origin, from: from, route: route, body: body, done: make(chan struct{})}
	b.admissions[id] = pending
	b.mu.Unlock()
	receipt, families, err := b.admit(ctx, from, id, route, body, origin)
	b.mu.Lock()
	if _, stored := b.receipts[id]; err != nil && !stored {
		err = b.refuseLocked(from, id, route, body, err, families, origin)
	} else if err == nil {
		delete(b.refusals, id)
	}
	pending.receipt, pending.err = receipt, err
	delete(b.admissions, id)
	close(pending.done)
	b.mu.Unlock()
	return receipt, err
}
func (b *Broker) admit(ctx context.Context, from Binding, id string, route Route, body string, origin Origin) (Receipt, map[string]string, error) {
	family := !strings.Contains(route.Target, ":")
	families, err := b.readFamilies(ctx)
	if err != nil {
		return Receipt{}, families, err
	}
	tried := map[Binding]bool{}
	for {
		if err := ctx.Err(); err != nil {
			return Receipt{}, families, err
		}
		b.mu.Lock()
		if b.drainGeneration != "" {
			b.mu.Unlock()
			return Receipt{}, families, ErrDraining
		}
		if !b.caller(from) {
			b.mu.Unlock()
			return Receipt{}, families, ErrUnavailable
		}
		var candidates []Candidate
		for binding, a := range b.actors {
			if !a.state.Connected() {
				continue
			}
			c := actorCandidate(a, b.now())
			// A fresh wrapper observation may contain an unseen human submission.
			// Probe exhausted actors before applying the authoritative budget check.
			c.Remaining = InboundAllowance
			c.Known = true
			c.Resting = true
			// Keep the sender in family identity checks, but never select it.
			// As with tried candidates, it is ineligible for a fresh observation.
			if tried[binding] || (family && binding == from) {
				c.Known = false
			}
			candidates = append(candidates, c)
		}
		to, err := ResolveRecipient(route, candidates, families, b.now(), func(to Binding) bool { return b.eligible(from, to) })
		if err != nil {
			b.mu.Unlock()
			return Receipt{}, families, err
		}
		a := b.actors[to]
		if !family && (a.reservation != "" || len(a.inbox) != 0 || actorCandidate(a, b.now()).Pending) {
			receipt, err := b.enqueueLocked(a, from, id, route, body, origin)
			b.mu.Unlock()
			return receipt, families, err
		}
		a.reservation = id
		seq := a.sequence
		b.mu.Unlock()
		receipt, err := b.tryAdmission(ctx, a, seq, from, id, route, body, family, origin)
		if err == nil {
			return receipt, families, nil
		}
		b.mu.Lock()
		a.releaseReservationLocked(id)
		b.mu.Unlock()
		// Free any wrapper reservation this attempt holds before the message
		// can be queued: the worker reserves the same ID when it promotes it.
		releaseCtx, releaseCancel := context.WithTimeout(ctx, AdmissionTimeout)
		_ = a.endpoint.Release(releaseCtx, id)
		releaseCancel()
		// A busy wrapper (the operator typing, a delivery releasing) holds an
		// exact message in the queue; the worker promotes it when it settles.
		if !family && errors.Is(err, ErrRecipientBusy) {
			b.mu.Lock()
			defer b.mu.Unlock()
			if b.actors[to] != a || !b.caller(to) || !b.caller(from) {
				return Receipt{}, families, ErrUnavailable
			}
			receipt, err := b.enqueueLocked(a, from, id, route, body, origin)
			return receipt, families, err
		}
		if !family {
			return Receipt{}, families, err
		}
		tried[to] = true
	}
}
func (b *Broker) tryAdmission(ctx context.Context, a *SlotActor, seq uint64, from Binding, id string, route Route, body string, family bool, origin Origin) (Receipt, error) {
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
		var committed *AlreadyCommittedError
		if errors.As(err, &committed) {
			return b.adoptRetained(a, from, id, route, body, committed.Receipt, origin)
		}
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
	if b.drainGeneration != "" {
		return Receipt{}, ErrDraining
	}
	if family && !b.eligible(from, to) {
		return Receipt{}, ErrNoRecipient
	}
	if err = b.reconcileObservation(a, obs); err != nil {
		return Receipt{}, err
	}
	if len(a.inbox) != 0 || a.state.Queued() > 0 {
		return Receipt{}, ErrRecipientBusy
	}
	now := b.now()
	m := Message{Origin: origin, ID: id, From: from, To: to, Body: body, Deadline: now.Add(PasteTimeout), AcceptedAt: now}
	if err = b.apply(a, Event{Kind: Admit, Binding: to, Message: m, At: now}); err != nil {
		return Receipt{}, err
	}
	b.destinations[id] = route
	a.reservation = ""
	a.inbox <- m
	receipt, _ := b.receiptLocked(id)
	return receipt, nil
}

// enqueueLocked lines an exact message up behind the recipient's delivery
// (pair#492) and wakes the worker that promotes it.
func (b *Broker) enqueueLocked(a *SlotActor, from Binding, id string, route Route, body string, origin Origin) (Receipt, error) {
	now := b.now()
	m := Message{Origin: origin, ID: id, From: from, To: a.binding, Body: body, Deadline: now.Add(PasteTimeout), AcceptedAt: now}
	if err := b.apply(a, Event{Kind: Enqueue, Binding: a.binding, Message: m, At: now}); err != nil {
		return Receipt{}, err
	}
	b.destinations[id] = route
	a.wakeLocked()
	receipt, _ := b.receiptLocked(id)
	return receipt, nil
}

// wakeLocked tells the worker to look at the queue again.
func (a *SlotActor) wakeLocked() {
	select {
	case a.wake <- struct{}{}:
	default:
	}
}

// releaseReservationLocked ends an admission attempt's hold on the actor.
// Every path other than the worker's own that frees the actor wakes the
// worker when messages wait: an attempt that blocked a promotion and then
// failed would otherwise strand the queue (pair#492 close review).
func (a *SlotActor) releaseReservationLocked(id string) {
	if a.reservation != id || id == "" {
		return
	}
	a.reservation = ""
	if a.state.Queued() > 0 {
		a.wakeLocked()
	}
}

// PromoteRetry is how long a queued message waits before the worker asks a
// busy wrapper again: holding, never refusing.
const PromoteRetry = time.Second

// promote reserves the queue's head on the wrapper and makes it the delivery.
// It reports whether to try again later: the wrapper was busy or unreachable.
func (b *Broker) promote(ctx context.Context, a *SlotActor) bool {
	b.mu.Lock()
	_ = b.apply(a, Event{Kind: Tick, At: b.now()})
	head, ok := b.eligibleDrainHead(a)
	for ok && !b.caller(head.From) {
		_ = b.apply(a, Event{Kind: Cancel, Binding: a.binding, Message: Message{ID: head.ID}, Detail: "sender disconnected before delivery", At: b.now()})
		head, ok = b.eligibleDrainHead(a)
	}
	_, pending := a.state.Pending()
	if ok && a.reservation != "" {
		a.blockedPromotions++
	}
	if !ok || pending || a.reservation != "" || len(a.inbox) != 0 || b.actors[a.binding] != a || !b.caller(a.binding) {
		b.mu.Unlock()
		return false
	}
	a.reservation = head.ID
	seq := a.sequence
	b.mu.Unlock()
	obs, err := a.endpoint.Observe(ctx)
	if err == nil {
		err = a.endpoint.Reserve(ctx, head.ID, obs.Sequence)
	}
	b.mu.Lock()
	// The worker frees its own attempt without a wake: it schedules its own
	// retry (PromoteRetry), and a self-wake would spin on the wrapper. Retain
	// the reservation through cleanup IO, even while other actors can progress.
	defer func() {
		if a.reservation == head.ID {
			a.reservation = ""
		}
		b.mu.Unlock()
	}()
	var committed *AlreadyCommittedError
	if errors.As(err, &committed) {
		// Network IO cannot restore authority after cancellation, replacement,
		// or removal of the exact admitted queue head.
		current, queued := b.eligibleDrainHead(a)
		if !queued || current != head || ctx.Err() != nil || a.sequence != seq ||
			b.actors[a.binding] != a || !b.caller(a.binding) || !b.caller(head.From) {
			return false
		}
		// The wrapper already holds this ID: answer with its outcome.
		retained := committed.Receipt
		// System authority comes only from the already admitted queue head.
		// Peer recovery remains compatible with legacy wrapper deadlines.
		match := retained.Message.Origin.Valid() && retained.Message.Origin == head.Origin &&
			retained.Message.ID == head.ID && retained.Message.From == head.From &&
			retained.Message.To == head.To && retained.Message.Body == head.Body && retained.Message.DrainInstance == head.DrainInstance
		if head.Origin == OriginSystem {
			match = match && retained.Message.Deadline.Equal(head.Deadline) &&
				(retained.Message.AcceptedAt.IsZero() || retained.Message.AcceptedAt.Equal(head.AcceptedAt))
		}
		if !match {
			_ = b.apply(a, Event{Kind: Cancel, Binding: a.binding, Message: Message{ID: head.ID}, Detail: "conflicting message ID retained by recipient", At: b.now()})
			a.wakeLocked()
			return false
		}
		route := b.destinations[head.ID]
		_ = b.apply(a, Event{Kind: Cancel, Binding: a.binding, Message: Message{ID: head.ID}, At: b.now()})
		delete(b.receipts, head.ID)
		// Keep the broker's exact immutable identity, including admission time.
		retained.Message = head
		b.recordRecoveredLocked(retained, route)
		a.wakeLocked()
		return false
	}
	if err == nil && (ctx.Err() != nil || a.sequence != seq || b.actors[a.binding] != a) {
		err = ErrUnavailable
	}
	if current, allowed := b.eligibleDrainHead(a); err == nil && (!allowed || current.ID != head.ID) {
		err = ErrDraining
	}
	if err == nil {
		err = b.reconcileObservation(a, obs)
	}
	if err == nil {
		err = b.apply(a, Event{Kind: Promote, Binding: a.binding, Message: Message{ID: head.ID}, At: b.now()})
	}
	if err != nil {
		b.mu.Unlock()
		releaseCtx, cancel := context.WithTimeout(context.Background(), AdmissionTimeout)
		_ = a.endpoint.Release(releaseCtx, head.ID)
		cancel()
		b.mu.Lock()
		return ctx.Err() == nil
	}
	m, _ := a.state.Pending()
	a.inbox <- m
	return false
}

// adoptRetained answers a reused ID with the recipient's own outcome instead
// of delivering again: the wrapper already holds this delivery. Only the same
// sender and body may adopt it; anything else is a conflicting ID.
func (b *Broker) adoptRetained(a *SlotActor, from Binding, id string, route Route, body string, r Receipt, origin Origin) (Receipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	a.releaseReservationLocked(id)
	if origin != OriginPeer || r.Message.Origin != OriginPeer || r.Message.ID != id || r.Message.From != from || r.Message.To != a.binding || r.Message.Body != body {
		return Receipt{}, errors.New("conflicting message ID")
	}
	return b.recordRecoveredLocked(r, route), nil
}

// recordRecoveredLocked keeps a receipt learned from a recipient. One still
// in flight there has no broker job watching it any more, so it is recorded
// as Indeterminate rather than left Delivering forever.
func (b *Broker) recordRecoveredLocked(r Receipt, route Route) Receipt {
	if old, ok := b.receiptLocked(r.Message.ID); ok {
		return old
	}
	if !r.Status.Terminal() {
		r.Status, r.Detail = Indeterminate, "delivery was in flight at the recipient when this broker learned of it; check the recipient's transcript"
	}
	r.RetainUntil = b.now().Add(ReceiptLifetime)
	b.receipts[r.Message.ID] = r
	b.destinations[r.Message.ID] = route
	return r
}

// StatusContext resolves exact IDs or unique caller-visible receipt prefixes,
// plus — for an ID this broker does not remember,
// as after a Couch restart — a query of every connected recipient's retained
// receipts. A recipient that does not answer makes the result uncertain, not
// absent. It runs only on this request path, never in the background.
func (b *Broker) StatusContext(parent context.Context, caller Binding, id string) (Receipt, error) {
	b.mu.Lock()
	b.sweep()
	if !b.caller(caller) {
		b.mu.Unlock()
		return Receipt{}, ErrUnavailable
	}
	if r, ok := b.statusLocked(caller, id); ok {
		b.mu.Unlock()
		return r, nil
	}
	matches := make(map[string]Receipt)
	match := func(fullID string) {
		if strings.HasPrefix(fullID, id) {
			if r, ok := b.statusLocked(caller, fullID); ok {
				matches[fullID] = r
			}
		}
	}
	for fullID := range b.receipts {
		match(fullID)
	}
	for fullID := range b.refusals {
		match(fullID)
	}
	if len(matches) == 1 {
		for _, r := range matches {
			b.mu.Unlock()
			return r, nil
		}
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for fullID := range matches {
			ids = append(ids, fullID)
		}
		sort.Strings(ids)
		b.mu.Unlock()
		return Receipt{}, fmt.Errorf("ambiguous message ID prefix %q: %s", id, strings.Join(ids, ", "))
	}
	if _, known := b.receipts[id]; known {
		b.mu.Unlock()
		return Receipt{}, errors.New("receipt unavailable for this actor")
	}
	type holder struct {
		binding Binding
		h       ReceiptHolder
	}
	var holders []holder
	for binding, a := range b.actors {
		if h, ok := a.endpoint.(ReceiptHolder); ok && a.state.Connected() {
			holders = append(holders, holder{binding, h})
		}
	}
	b.mu.Unlock()
	ctx, cancel := context.WithTimeout(parent, AdmissionTimeout/2)
	defer cancel()
	type answer struct {
		from Binding
		r    Receipt
		err  error
	}
	answers := make(chan answer, len(holders))
	limit := make(chan struct{}, 4)
	for _, h := range holders {
		go func(h holder) {
			limit <- struct{}{}
			defer func() { <-limit }()
			r, err := h.h.Retained(ctx, id)
			answers <- answer{h.binding, r, err}
		}(h)
	}
	var found *Receipt
	unanswered := 0
	var decodeFailures []string
	for range holders {
		a := <-answers
		switch {
		case a.err == nil:
			// A wrapper vouches only for deliveries addressed to itself; a
			// receipt naming another recipient is untrusted input (#365 BR-13).
			if a.r.Message.Origin == OriginPeer && a.r.Message.ID == id && a.r.Message.To == a.from && (a.r.Message.From == caller || a.r.Message.To == caller) {
				r := a.r
				found = &r
			}
		case !errors.Is(a.err, ErrUnknownDelivery):
			unanswered++
			var decode *replyDecodeError
			if errors.As(a.err, &decode) {
				decodeFailures = append(decodeFailures, a.err.Error())
			}
		}
	}
	if found != nil {
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.recordRecoveredLocked(*found, Route{Target: found.Message.To.Slot}), nil
	}
	if unanswered > 0 {
		if len(decodeFailures) > 0 {
			sort.Strings(decodeFailures)
			return Receipt{}, fmt.Errorf("%w: %s", ErrUncertain, strings.Join(decodeFailures, "; "))
		}
		return Receipt{}, fmt.Errorf("%w: %d connected recipient(s) did not answer for message %s", ErrUncertain, unanswered, id)
	}
	return Receipt{}, fmt.Errorf("unknown message ID or prefix %q: no matching receipt found", id)
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
	var retry <-chan time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.wake:
		case <-retry:
		case m := <-a.inbox:
			b.deliver(ctx, a, m)
		}
		retry = nil
		if b.promote(ctx, a) {
			retry = time.After(PromoteRetry)
		}
	}
}

// deliver runs one delivery to its terminal outcome.
func (b *Broker) deliver(ctx context.Context, a *SlotActor, m Message) {
	// Accepted composer waits end with this exact receiver's lifetime.
	deliveryCtx, cancel := context.WithCancel(ctx)
	job := &deliveryJob{cancel: cancel, done: make(chan struct{})}
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
	progress := func(r Receipt) {
		b.mu.Lock()
		defer b.mu.Unlock()
		_ = b.apply(a, Event{Kind: DeliveryProgress, Binding: m.To, Message: m, Status: r.Status, Detail: r.Detail, At: b.now()})
	}
	if valid {
		if endpoint, ok := a.endpoint.(BatchDeliveryEndpoint); ok && a.batchMaxChars > 0 && !b.isDrainControl(m.ID) {
			result, err = endpoint.DeliverBatch(deliveryCtx, m, func() ([]Message, error) {
				return b.freezeBatch(deliveryCtx, a, m, job)
			}, progress)
		} else {
			result, err = a.endpoint.Deliver(deliveryCtx, m, progress)
		}
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
	for id, active := range b.jobs {
		if active == job {
			delete(b.jobs, id)
		}
	}
	_ = b.apply(a, Event{Kind: DeliveryFinished, Binding: m.To, Message: m, Status: status, Detail: detail, At: b.now()})
	close(job.done)
	b.mu.Unlock()
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

// SetSubmissionObserver receives accepted counter advances under the broker
// lock. The observer must only offer bounded memory data, never IO or broker calls.
func (b *Broker) SetSubmissionObserver(observe func(Binding, uint64)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.submissionObserver = observe
}

// ResourceIdle returns pending-work evidence without observation or alias IO.
// known is false for disconnected generations; the caller combines this with
// the exact wrapper's settled evidence rather than inferring idle from silence.
func (b *Broker) ResourceIdle(binding Binding) (pending, known bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	a := b.actors[binding]
	if a == nil || !a.state.Connected() {
		return false, false
	}
	return actorCandidate(a, b.now()).Pending, true
}
