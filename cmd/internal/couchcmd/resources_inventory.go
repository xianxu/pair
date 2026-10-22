package couchcmd

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

type resourceInventory struct {
	// seedDone releases sampling after the owned initial discovery attempt, even
	// when that attempt fails. Only run closes it; later refreshes do not reset it.
	seedDone   chan struct{}
	mu         sync.Mutex
	console    *couchtty.Console
	couch      *couchcore.Couch
	service    *messageService
	cache      *couchcore.ResourceRootCache
	workspace  func(context.Context, string) (couchcore.WorkspaceIdentity, error)
	probe      func(context.Context, couchcore.ThreadAddress) (couchcore.PairSessionBinding, error)
	seen       map[string]string // worker-owned
	entries    map[string]couchcore.ResourceSlot
	rows       []couchcore.ResourceSlot
	sourceRows []couchcore.ActionableThreadSummary
	hosted     []couchcore.LiveTTYObservation
	sessions   map[couchcore.ThreadAddress]string
	complete   bool
	omitted    int
}

func newResourceInventory(console *couchtty.Console, c *couchcore.Couch, service *messageService) *resourceInventory {
	r := &resourceInventory{seedDone: make(chan struct{}), console: console, couch: c, service: service, workspace: service.authority.workspace, seen: map[string]string{}, entries: map[string]couchcore.ResourceSlot{}, sessions: map[couchcore.ThreadAddress]string{}}
	if source, ok := c.Artifacts.(interface {
		ResourceRootEvidenceCache() *couchcore.ResourceRootCache
	}); ok {
		r.cache = source.ResourceRootEvidenceCache()
	}
	if source, ok := c.Artifacts.(interface {
		PairSessionContext(context.Context, couchcore.ThreadAddress) (couchcore.PairSessionBinding, error)
	}); ok {
		r.probe = source.PairSessionContext
	}
	return r
}

// run owns all inventory/proof IO. Sampling only reads slots(). Startup and
// changed lifecycle observations are the only triggers, never the sample timer.
func (r *resourceInventory) run(ctx context.Context) {
	changes := r.console.ResourceChanges()
	snapshot := r.console.ResourceInventory()
	r.seed(ctx, snapshot)
	for {
		select {
		case <-ctx.Done():
			return
		case <-changes:
			pass, cancel := context.WithTimeout(ctx, 30*time.Second)
			pass, cancelOwner := r.couch.WithSessionOwnerPass(pass)
			r.refresh(pass, r.console.ResourceInventory())
			cancelOwner()
			cancel()
		}
	}
}

// seed owns exactly one initial discovery attempt and its completion signal.
func (r *resourceInventory) seed(ctx context.Context, snapshot couchtty.ResourceInventorySnapshot) {
	seed, cancel := context.WithTimeout(ctx, 30*time.Second)
	seed, cancelOwner := r.couch.WithSessionOwnerPass(seed)
	rows, err := r.couch.ActionableThreadInventoryContext(seed, snapshot.Hosted)
	if err == nil {
		snapshot.Rows = rows
		snapshot.InventoryKnown = true
	} else {
		// Cached console census cannot turn a failed initial attempt into proof.
		snapshot.InventoryKnown = false
	}
	r.refresh(seed, snapshot)
	cancelOwner()
	cancel()
	close(r.seedDone)
}

// runSampler shares the service lifetime with the inventory worker. Waiting for
// an attempted seed distinguishes cold startup from observed incomplete proof;
// cancellation cannot start a sampler just because both channels are ready.
func (r *resourceInventory) runSampler(ctx context.Context, sampler *resourceSampler) {
	select {
	case <-ctx.Done():
		return
	case <-r.seedDone:
	}
	if ctx.Err() != nil {
		return
	}
	sampler.run(ctx)
}

func resourceInventoryKey(row couchcore.ActionableThreadSummary) string {
	if row.Target.Kind == couchcore.ThreadTargetSlot {
		return "slot:" + row.Target.Slot.WorktreeRoot
	}
	return "thread:" + row.Address.RepoScope + "/" + string(row.Address.Tag)
}

func resourceRowFingerprint(row couchcore.ActionableThreadSummary, hosted []couchcore.LiveTTYObservation) string {
	identities := []string{}
	for _, h := range hosted {
		if h.Address == row.Address {
			identities = append(identities, fmt.Sprintf("%d@%s", h.Process.PID, h.Process.Identity))
		}
	}
	sort.Strings(identities)
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%v", row.Address.RepoScope, row.Address.Tag, row.WorkingPath, row.Target.Slot.WorktreeRoot, row.State, row.LastActiveAt.UTC().Format(time.RFC3339Nano), identities)
}

// Refresh publishes the bounded census before slow proofs. During a refresh,
// readers retain slot names but never infer absence from unfinished discovery.
func (r *resourceInventory) refresh(ctx context.Context, snapshot couchtty.ResourceInventorySnapshot) {
	r.mu.Lock()
	old := append([]couchcore.ResourceSlot(nil), r.rows...)
	inheritedOmitted := 0
	if !snapshot.InventoryKnown && snapshot.Rows == nil {
		snapshot.Rows = append([]couchcore.ActionableThreadSummary(nil), r.sourceRows...)
		inheritedOmitted = r.omitted
	}
	sessions := make(map[couchcore.ThreadAddress]string, len(r.sessions))
	for a, s := range r.sessions {
		sessions[a] = s
	}
	r.complete = false
	r.mu.Unlock()
	// Prefer hosted/admitted current threads, then other live rows, before parked
	// records. Stable keys make cap selection reproducible across menu ordering.
	active := map[couchcore.ThreadAddress]bool{}
	for _, h := range snapshot.Hosted {
		active[h.Address] = true
	}
	if r.service != nil && r.service.broker != nil {
		for _, b := range r.service.broker.Connected() {
			active[couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}] = true
		}
	}
	rows := append([]couchcore.ActionableThreadSummary(nil), snapshot.Rows...)
	priority := func(row couchcore.ActionableThreadSummary) int {
		if active[row.Address] {
			return 0
		}
		if row.State == couchcore.ThreadLive {
			return 1
		}
		if row.State == couchcore.ThreadDetached {
			return 2
		}
		return 3
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := priority(rows[i]), priority(rows[j])
		if a != b {
			return a < b
		}
		return resourceInventoryKey(rows[i]) < resourceInventoryKey(rows[j])
	})
	omitted := inheritedOmitted
	if len(rows) > couchcore.ResourceMaxSlots {
		omitted += len(rows) - couchcore.ResourceMaxSlots
		rows = rows[:couchcore.ResourceMaxSlots]
	}
	next := make([]couchcore.ResourceSlot, len(rows))
	nextSeen := map[string]string{}
	nextEntries := map[string]couchcore.ResourceSlot{}
	nextSessions := map[couchcore.ThreadAddress]string{}
	changed := make([]bool, len(rows))
	for i, row := range rows {
		key := resourceInventoryKey(row)
		fingerprint := resourceRowFingerprint(row, snapshot.Hosted)
		entry, exists := r.entries[key]
		changed[i] = !exists || r.seen[key] != fingerprint
		if changed[i] {
			entry = couchcore.ResourceSlot{Thread: row.Address, Slot: row.Address.RepoScope + "/" + string(row.Address.Tag), Reason: "ownership discovery pending"}
			if row.Target.Kind == couchcore.ThreadTargetSlot {
				entry.Slot = fmt.Sprintf("%s:%d", row.Target.Slot.Repo, row.Target.Slot.Number)
			}
			delete(sessions, row.Address)
		}
		next[i] = entry
		nextSeen[key] = fingerprint
		nextEntries[key] = entry
		nextSessions[row.Address] = sessions[row.Address]
	}
	publish := func(complete bool) {
		r.mu.Lock()
		r.rows = append([]couchcore.ResourceSlot(nil), next...)
		r.sourceRows = append([]couchcore.ActionableThreadSummary(nil), rows...)
		r.hosted = append([]couchcore.LiveTTYObservation(nil), snapshot.Hosted...)
		r.sessions = make(map[couchcore.ThreadAddress]string, len(nextSessions))
		for a, s := range nextSessions {
			r.sessions[a] = s
		}
		r.complete = complete
		r.omitted = omitted
		r.mu.Unlock()
	}
	publish(false)
	workspaceComplete := true
	for i, row := range rows {
		if ctx.Err() != nil {
			return
		}
		if changed[i] {
			path := row.WorkingPath
			if row.Target.Kind == couchcore.ThreadTargetSlot {
				path = row.Target.Slot.WorktreeRoot
			}
			if r.workspace != nil {
				if id, err := r.workspace(ctx, path); err == nil && id.Address != nil && id.RepoIdentity != "" {
					next[i].Workspace = id
					next[i].Slot = *id.Address
				}
			}
			// Workspace identity is useful even while terminal ownership is slow.
			publish(false)
			if r.probe != nil && row.Address != (couchcore.ThreadAddress{}) {
				binding, err := r.probe(ctx, row.Address)
				if err == nil {
					nextSessions[row.Address] = binding.Name
				}
			}
		}
		if next[i].Workspace.RepoIdentity == "" || next[i].Workspace.WorktreeRoot == "" {
			workspaceComplete = false
		}
		nextEntries[resourceInventoryKey(row)] = next[i]
		publish(false)
	}
	if ctx.Err() != nil {
		return
	}
	complete := snapshot.InventoryKnown && omitted == 0 && workspaceComplete
	// Unknown/omitted rows are not proof of removal; keep their exact cache proof.
	if complete {
		retained := map[couchcore.ThreadAddress]bool{}
		for _, row := range next {
			retained[row.Thread] = true
		}
		for _, row := range old {
			if !retained[row.Thread] {
				r.cache.Forget(row.Thread)
			}
		}
	}
	r.seen = nextSeen
	r.entries = nextEntries
	publish(complete)
}

// slots performs no process, filesystem, git or terminal IO. Wrapper admission
// can update the shared proof cache independently of the inventory worker.
func (r *resourceInventory) slots() []couchcore.ResourceSlot { return r.snapshot().Slots }

func (r *resourceInventory) snapshot() couchcore.ResourceSlotInventory {
	r.mu.Lock()
	rows := append([]couchcore.ResourceSlot(nil), r.rows...)
	complete, omitted := r.complete, r.omitted
	hosted := append([]couchcore.LiveTTYObservation(nil), r.hosted...)
	sessions := make(map[couchcore.ThreadAddress]string, len(r.sessions))
	for a, s := range r.sessions {
		sessions[a] = s
	}
	r.mu.Unlock()
	roots := map[couchcore.ThreadAddress]couchcore.ResourceRootEvidence{}
	for _, proof := range r.cache.Snapshot() {
		roots[proof.Address] = proof
	}
	indexes := map[couchcore.ThreadAddress]int{}
	for i, row := range rows {
		indexes[row.Thread] = i
	}
	bindings := []couchmessage.Binding{}
	workspaces := map[couchmessage.Binding]couchcore.WorkspaceIdentity{}
	if r.service != nil && r.service.broker != nil {
		bindings = r.service.broker.Connected()
		r.service.mu.Lock()
		for _, b := range bindings {
			workspaces[b] = r.service.workspaces[b]
		}
		r.service.mu.Unlock()
	}
	for _, b := range bindings {
		a := couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}
		i, exists := indexes[a]
		if !exists {
			i = len(rows)
			indexes[a] = i
			rows = append(rows, couchcore.ResourceSlot{Slot: b.Slot, Thread: a})
		}
		rows[i].Workspace = workspaces[b]
		rows[i].Slot = b.Slot
		sessions[a] = b.Session
	}
	for i := range rows {
		row := &rows[i]
		proof := roots[row.Thread]
		workspaceKnown := row.Workspace.RepoIdentity != "" && row.Workspace.WorktreeRoot != ""
		row.OwnershipKnown = workspaceKnown && proof.Known && proof.Session == sessions[row.Thread]
		if !workspaceKnown {
			row.Reason = "workspace identity unavailable"
		} else if !row.OwnershipKnown {
			row.Reason = "terminal ownership unconfirmed"
		} else {
			row.Reason = ""
		}
		add := func(pid int, identity string) {
			if pid > 0 && identity != "" {
				row.Roots = append(row.Roots, couchcore.ResourceRoot{Key: couchcore.ResourceProcessKey{PID: pid, Identity: identity}, Slot: row.Slot, Thread: row.Thread, Workspace: row.Workspace})
			}
		}
		if row.OwnershipKnown && proof.Present {
			add(proof.Server.PID, proof.Server.Identity)
		}
		for _, h := range hosted {
			if h.Address == row.Thread {
				add(h.Process.PID, h.Process.Identity)
			}
		}
		for _, b := range bindings {
			if b.Scope != row.Thread.RepoScope || couchcore.ThreadTag(b.Tag) != row.Thread.Tag {
				continue
			}
			add(b.PID, b.Start)
			if live := r.service.liveness.Load(); live != nil {
				if l, ok := (*live)[b.Slot]; ok && l.Binding == b && l.Settled != nil {
					pending, known := r.service.broker.ResourceIdle(b)
					row.IdleKnown = known
					row.Idle = known && *l.Settled && !pending && (l.Picker == nil || !*l.Picker)
				}
			}
		}
	}

	live := map[string]bool{}
	for _, b := range bindings {
		live[b.Slot] = true
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if live[rows[i].Slot] != live[rows[j].Slot] {
			return live[rows[i].Slot]
		}
		return rows[i].Slot < rows[j].Slot
	})
	if len(rows) > couchcore.ResourceMaxSlots {
		omitted += len(rows) - couchcore.ResourceMaxSlots
		rows = rows[:couchcore.ResourceMaxSlots]
		complete = false
	}
	return couchcore.ResourceSlotInventory{Slots: rows, Complete: complete && omitted == 0, Omitted: omitted}
}
