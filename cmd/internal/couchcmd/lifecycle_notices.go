package couchcmd

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

func lifecycleNoticesEnabled(getenv func(string) string) (bool, error) {
	switch getenv("COUCH_LIFECYCLE_NOTICES") {
	case "", "on":
		return true, nil
	case "off":
		return false, nil
	default:
		return false, errors.New("COUCH_LIFECYCLE_NOTICES must be on or off")
	}
}

type lifecycleAttempt struct {
	id       string
	hub      string
	from     couchmessage.Binding
	cancel   context.CancelFunc
	admitted bool
	retired  bool
	started  bool
}

// unresolvedFact is a fact waiting for its slot identity, and when it began
// waiting (an ordinary fact expires, see lifecycleOrdinaryWait).
type unresolvedFact struct {
	fact couchtty.LifecycleObservation
	at   time.Time
}

// lifecycleOrdinaryWait bounds how long an ordinary-target fact waits for its
// identity. The gap it covers is a launch report followed by the wrapper's
// connect and verification (seconds, admission itself is capped at 10 s); a
// fact older than this would be published as stale news (pair#693).
const lifecycleOrdinaryWait = 2 * time.Minute

type lifecycleAdmission struct {
	token   uint64
	receipt couchmessage.Receipt
	err     error
}

// lifecyclePublisher owns bounded Slots projections, not message history. Its
// callbacks reduce memory only; admission runs on owned, cancellable workers.
type lifecyclePublisher struct {
	mu          sync.Mutex
	service     *messageService
	state       couchcore.LifecycleNoticeState
	ready       map[uint64]couchcore.LifecycleNoticeEffect
	attempts    map[uint64]lifecycleAttempt
	roots       map[string]bool
	identities  map[string]couchcore.WorkspaceIdentity
	generation  uint64
	unresolved  map[string]unresolvedFact
	now         func() time.Time
	diagnostics map[string]lifecycleDiagnostic
	// Roots never retire within this publisher. Report its fixed capacity
	// failure once, even after the worker drains the transient diagnostic map.
	capacityWarned bool
	wake           chan struct{}
	results        chan lifecycleAdmission
	notice         func(string)
	warning        func(string)
}

type lifecycleDiagnostic struct {
	body       string
	actionable bool
}

func startLifecyclePublisher(s *messageService, notice func(string), warning ...func(string)) *lifecyclePublisher {
	p := &lifecyclePublisher{service: s, ready: map[uint64]couchcore.LifecycleNoticeEffect{}, attempts: map[uint64]lifecycleAttempt{}, roots: map[string]bool{}, identities: map[string]couchcore.WorkspaceIdentity{}, unresolved: map[string]unresolvedFact{}, diagnostics: map[string]lifecycleDiagnostic{}, wake: make(chan struct{}, 1), results: make(chan lifecycleAdmission, 128), notice: notice, now: s.authority.now}
	if len(warning) > 0 {
		p.warning = warning[0]
	}
	s.lifecycle.Store(p)
	s.workers.Add(1)
	go func() { defer s.workers.Done(); p.run(s.lifetime) }()
	return p
}
func (p *lifecyclePublisher) signal() {
	select {
	case p.wake <- struct{}{}:
	default:
	}
}
func lifecycleBindingKey(b couchmessage.Binding) string {
	raw, _ := json.Marshal(b)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

func (p *lifecyclePublisher) advanceLocked(e couchcore.LifecycleNoticeEvent) {
	var effects []couchcore.LifecycleNoticeEffect
	p.state, effects = couchcore.AdvanceLifecycleNotices(p.state, e)
	for _, fx := range effects {
		switch fx.Kind {
		case couchcore.LifecycleNoticePublish:
			p.ready[fx.Token] = fx
		case couchcore.LifecycleNoticeDiagnostic:
			p.diagnostics[fx.Hub] = lifecycleDiagnostic{body: fx.Body, actionable: e.Outcome != "unavailable" && e.Outcome != "cancelled" && e.Outcome != "uncertain"}
		case couchcore.LifecycleNoticeCancel:
			_, unstarted := p.ready[fx.Token]
			delete(p.ready, fx.Token)
			if unstarted {
				// This part never left memory: settle cancellation so the
				// reducer can freeze against the replacement recipient.
				p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeAdmission, Hub: fx.Hub, Token: fx.Token, Outcome: "unavailable"})
			}
			if a, ok := p.attempts[fx.Token]; ok && !a.admitted {
				a.cancel()
			}
		}
	}
}

// workspaceSnapshot copies verified cached identities only. It performs no
// discovery, and may also be used at the drain boundary before freezing facts.
func (p *lifecyclePublisher) workspaceSnapshot() map[string]couchcore.WorkspaceIdentity {
	p.service.mu.Lock()
	defer p.service.mu.Unlock()
	identities := make(map[string]couchcore.WorkspaceIdentity, len(p.service.workspaces))
	for _, id := range p.service.workspaces {
		identities[id.WorktreeRoot] = id
	}
	return identities
}
func (p *lifecyclePublisher) cacheIdentitiesLocked(identities map[string]couchcore.WorkspaceIdentity) {
	for path, id := range identities {
		if len(p.identities) < 128 || p.identities[path].WorktreeRoot != "" {
			p.identities[path] = id
		}
	}
}

// ownershipChanged is called by the successful move producer while it holds
// the existing hub effect gate. Drop history; keep only admitted receipts.
func (p *lifecyclePublisher) ownershipChanged(newOwner string) {
	p.mu.Lock()
	p.generation++
	clear(p.ready)
	clear(p.unresolved)
	for token, a := range p.attempts {
		a.retired = true
		if !a.started {
			a.cancel()
		}
		p.attempts[token] = a
	}
	p.roots[newOwner] = true // current configured hub; bounded by HubOwnership
	p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeDiscard, Hubs: []string{newOwner}})
	p.mu.Unlock()
	p.signal()
}
func (p *lifecyclePublisher) observe(e couchtty.LifecycleObservation) {
	var identities map[string]couchcore.WorkspaceIdentity
	if e.Kind == "restart-drained" {
		identities = p.workspaceSnapshot()
	}
	p.mu.Lock()
	switch e.Kind {
	case "restart-armed":
		p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeArm})
	case "restart-drained":
		p.cacheIdentitiesLocked(identities)
		p.reducePendingLocked()
		// Missing cached metadata cannot establish slot ownership. Ordinary or
		// unrouteable paths must not hold the completed reattach pass indefinitely.
		// Only untargeted facts are reported: an ordinary row may simply not be
		// a slot (pair#693), which is no missing metadata.
		missing := 0
		for _, u := range p.unresolved {
			if u.fact.Target.Kind == "" {
				missing++
			}
		}
		if missing > 0 {
			p.diagnostics[""] = lifecycleDiagnostic{body: fmt.Sprintf("Lifecycle: %d slots' metadata unverified at restart drain", missing), actionable: true}
		}
		clear(p.unresolved)
		p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeDrain, Reason: fmt.Sprintf("restart reattachments complete; %d failed", e.Failures)})
	default:
		p.observeFactLocked(e)
	}
	p.mu.Unlock()
	p.signal()
}
func (p *lifecyclePublisher) observeFactLocked(e couchtty.LifecycleObservation) {
	if e.Path == "" || e.Path == p.service.authority.hub {
		return
	}
	// A typed Console target already has the canonical slot location. Prefer it
	// over a cache entry that may still describe the preceding incarnation.
	id, ok := p.identities[e.Path]
	if e.Target.Kind == couchcore.ThreadTargetSlot {
		slot := e.Target.Slot
		n := slot.Number
		id = couchcore.WorkspaceIdentity{Repo: slot.Repo, PrimaryRoot: slot.PrimaryRoot, WorktreeRoot: slot.WorktreeRoot, Slot: &n}
		ok = true
	}
	// No cached identity yet, ordinary target or none: the fact usually
	// arrives first, since the Console reports a launch before the wrapper's
	// session connects and its workspace is verified (pair#693). It waits for
	// the next refresh rather than being lost.
	if !ok {
		p.retainUnresolvedLocked(e)
		return
	}
	hub := p.service.authority.owner(id)
	if hub == "" || hub == id.WorktreeRoot {
		return
	}
	slot := ""
	if id.Address != nil {
		slot = *id.Address
	}
	if slot == "" && id.Slot != nil {
		slot = fmt.Sprintf("%s:%d", id.Repo, *id.Slot)
	}
	if slot == "" {
		return
	}
	if e.Kind == "needs-relaunch" {
		if live := p.service.liveness.Load(); live != nil {
			if current, exists := (*live)[slot]; exists && e.Address != (couchcore.ThreadAddress{RepoScope: current.Binding.Scope, Tag: couchcore.ThreadTag(current.Binding.Tag)}) {
				return
			}
		}
	}
	if !p.roots[hub] && len(p.roots) >= 128 {
		if !p.capacityWarned {
			p.capacityWarned = true
			p.diagnostics[""] = lifecycleDiagnostic{body: "Lifecycle: hub projection full; inspect Slots", actionable: true}
		}
		return
	}
	p.roots[hub] = true
	p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeConfirmed, Fact: couchcore.LifecycleNoticeFact{
		Hub: hub, Slot: slot, Address: e.Address, Incarnation: e.Incarnation, Observation: e.Observation, Kind: couchcore.LifecycleNoticeKind(e.Kind), Operation: e.Operation, Agent: e.Agent, Model: e.Model, Branch: e.Branch, Issue: e.Issue, Initiation: e.Initiation, Mode: e.Mode,
	}})
}

// retainUnresolvedLocked keeps the newest observation per path, at most 128
// paths. At the cap a new path evicts the oldest observation, so paths that
// never verify (no slot workspace) cannot crowd out one that will. Drain and
// ownership change clear the set; refresh also drops an ordinary fact older
// than lifecycleOrdinaryWait (reducePendingLocked).
func (p *lifecyclePublisher) retainUnresolvedLocked(e couchtty.LifecycleObservation) {
	if len(e.Path) > 4096 {
		return
	}
	u := unresolvedFact{fact: e, at: p.clock()}
	if previous, exists := p.unresolved[e.Path]; exists {
		if e.Observation > previous.fact.Observation {
			p.unresolved[e.Path] = u
		}
		return
	}
	if len(p.unresolved) >= 128 {
		oldest := ""
		for path, w := range p.unresolved {
			if oldest == "" || w.fact.Observation < p.unresolved[oldest].fact.Observation {
				oldest = path
			}
		}
		delete(p.unresolved, oldest)
	}
	p.unresolved[e.Path] = u
}

func (p *lifecyclePublisher) clock() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

func (p *lifecyclePublisher) reducePendingLocked() {
	// Reduce the complete resolvable set while holding the collector lock. Drain
	// must not race a batch removed from unresolved but not yet reduced.
	now := p.clock()
	for path, u := range p.unresolved {
		if _, ok := p.identities[path]; ok {
			delete(p.unresolved, path)
			p.observeFactLocked(u.fact)
		} else if u.fact.Target.Kind == couchcore.ThreadTargetOrdinary && now.Sub(u.at) > lifecycleOrdinaryWait {
			delete(p.unresolved, path)
		}
	}
}

// refresh reads only already verified registry/workspace snapshots; it never
// discovers a slot or invokes a native/Git subprocess.
func (p *lifecyclePublisher) refresh() {
	identities := p.workspaceSnapshot()
	p.mu.Lock()
	p.cacheIdentitiesLocked(identities)
	p.reducePendingLocked()
	var roots []string
	for root := range p.roots {
		roots = append(roots, root)
	}
	p.mu.Unlock()
	for _, root := range roots {
		b, ok := p.service.hubBindingAt(root)
		p.mu.Lock()
		p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeHubReady, Hub: root, Ready: ok, Incarnation: lifecycleBindingKey(b)})
		p.mu.Unlock()
	}
}

func (p *lifecyclePublisher) run(ctx context.Context) {
	var workers sync.WaitGroup
	defer workers.Wait()
	// This ticker observes bounded in-memory receipt state only. Registry
	// changes and new facts also wake immediately; no extra discovery is polled.
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		p.refresh()
		p.mu.Lock()
		for token, a := range p.attempts {
			if !a.admitted {
				continue
			}
			r, known, delivering := p.service.broker.ReceiptSnapshot(a.id)
			if known && r.Status.Terminal() && !delivering {
				if !a.retired {
					kind := couchcore.LifecycleNoticeReceipt
					if r.Status == couchmessage.Refused {
						kind = couchcore.LifecycleNoticeAdmission
					}
					p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: kind, Hub: a.hub, Token: token, Outcome: string(r.Status)})
				}
				a.cancel()
				delete(p.attempts, token)
			}
		}
		jobs := p.ready
		generation := p.generation
		p.ready = map[uint64]couchcore.LifecycleNoticeEffect{}
		diagnostics := p.diagnostics
		p.diagnostics = map[string]lifecycleDiagnostic{}
		p.mu.Unlock()
		for _, message := range diagnostics {
			if message.actionable && p.warning != nil {
				p.warning(message.body)
			} else if p.notice != nil {
				p.notice(message.body)
			}
		}
		for token, fx := range jobs {
			p.mu.Lock()
			if generation != p.generation {
				p.mu.Unlock()
				continue
			}
			full := len(p.attempts) >= 128
			if full {
				p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeAdmission, Hub: fx.Hub, Token: token, Outcome: "capacity"})
			}
			p.mu.Unlock()
			if full {
				continue
			}
			b, connected := p.service.hubBindingAt(fx.Hub)
			if !connected || lifecycleBindingKey(b) != fx.Incarnation {
				p.mu.Lock()
				p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeAdmission, Hub: fx.Hub, Token: token, Outcome: "unavailable"})
				p.mu.Unlock()
				continue
			}
			id, err := newRequestID()
			if err != nil {
				p.mu.Lock()
				p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeAdmission, Hub: fx.Hub, Token: token, Outcome: "refused"})
				p.mu.Unlock()
				continue
			}
			admissionCtx, cancel := context.WithCancel(ctx)
			p.mu.Lock()
			if generation != p.generation {
				p.mu.Unlock()
				cancel()
				continue
			}
			p.attempts[token] = lifecycleAttempt{id: id, hub: fx.Hub, from: b, cancel: cancel}
			p.mu.Unlock()
			workers.Add(1)
			go func(token uint64, body string, b couchmessage.Binding, id string) {
				defer workers.Done()
				var r couchmessage.Receipt
				release, err := holdHubEffect(admissionCtx, p.service.authority.hubs)
				if err == nil {
					p.mu.Lock()
					a := p.attempts[token]
					valid := !a.retired && admissionCtx.Err() == nil
					if valid {
						a.started = true
						p.attempts[token] = a
					}
					p.mu.Unlock()
					if valid {
						r, err = p.service.broker.SendSystem(admissionCtx, b, id, couchmessage.Route{Target: b.Slot}, body)
					} else {
						err = couchmessage.ErrUnavailable // proven not admitted
					}
					release()
				} else {
					err = couchmessage.ErrUnavailable // gate never acquired; no write
				}
				select {
				case p.results <- lifecycleAdmission{token: token, receipt: r, err: err}:
				case <-ctx.Done():
				}
			}(token, fx.Body, b, id)
		}
		select {
		case <-ctx.Done():
			p.mu.Lock()
			p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeShutdown})
			for _, a := range p.attempts {
				a.cancel()
			}
			p.mu.Unlock()
			return
		case result := <-p.results:
			p.mu.Lock()
			if a, ok := p.attempts[result.token]; ok {
				outcome := "accepted"
				if result.err != nil {
					outcome = "refused"
					if errors.Is(result.err, couchmessage.ErrUnsupported) {
						outcome = "unsupported"
					}
					if errors.Is(result.err, couchmessage.ErrUnavailable) {
						outcome = "unavailable"
					}
					if errors.Is(result.err, context.DeadlineExceeded) || errors.Is(result.err, context.Canceled) || errors.Is(result.err, couchmessage.ErrUncertain) {
						outcome = "uncertain"
					}
					if outcome == "uncertain" {
						// Retire uncertain input once, allowing genuinely newer
						// projections. Keep this old ID for read-only polling;
						// its eventual receipt cannot change the collector again.
						if !a.retired {
							p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeAdmission, Hub: a.hub, Token: result.token, Outcome: "uncertain"})
						}
						a.admitted = true
						a.retired = true
						p.attempts[result.token] = a
						p.diagnostics[a.hub] = lifecycleDiagnostic{body: "Couch lifecycle admission uncertain; polling canonical receipt without retry"}
					} else {
						a.cancel()
						delete(p.attempts, result.token)
					}
				} else {
					a.admitted = true
					p.attempts[result.token] = a
				}
				if outcome != "uncertain" {
					p.advanceLocked(couchcore.LifecycleNoticeEvent{Kind: couchcore.LifecycleNoticeAdmission, Hub: a.hub, Token: result.token, Outcome: outcome})
				}
			}
			p.mu.Unlock()
		case <-p.wake:
		case <-tick.C:
		}
	}
}
