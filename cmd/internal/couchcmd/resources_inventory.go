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
}

func newResourceInventory(console *couchtty.Console, c *couchcore.Couch, service *messageService) *resourceInventory {
	r := &resourceInventory{console: console, couch: c, service: service, workspace: service.authority.workspace, seen: map[string]string{}, entries: map[string]couchcore.ResourceSlot{}, sessions: map[couchcore.ThreadAddress]string{}}
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
	seed, cancel := context.WithTimeout(ctx, 30*time.Second)
	seed, cancelOwner := r.couch.WithSessionOwnerPass(seed)
	rows, err := r.couch.ActionableThreadInventoryContext(seed, snapshot.Hosted)
	if err == nil {
		snapshot.Rows = rows
		snapshot.InventoryKnown = true
	}
	r.refresh(seed, snapshot)
	cancelOwner()
	cancel()
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

func (r *resourceInventory) refresh(ctx context.Context, snapshot couchtty.ResourceInventorySnapshot) {
	r.mu.Lock()
	old := append([]couchcore.ResourceSlot(nil), r.rows...)
	if !snapshot.InventoryKnown && snapshot.Rows == nil {
		snapshot.Rows = append([]couchcore.ActionableThreadSummary(nil), r.sourceRows...)
	}
	sessions := make(map[couchcore.ThreadAddress]string, len(r.sessions))
	for a, s := range r.sessions {
		sessions[a] = s
	}
	r.mu.Unlock()
	nextSeen := map[string]string{}
	nextEntries := map[string]couchcore.ResourceSlot{}
	var next []couchcore.ResourceSlot
	nextSessions := map[couchcore.ThreadAddress]string{}
	for _, row := range snapshot.Rows {
		if len(next) >= couchcore.ResourceMaxSlots || ctx.Err() != nil {
			break
		}
		fingerprint := resourceRowFingerprint(row, snapshot.Hosted)
		key := resourceInventoryKey(row)
		entry, exists := r.entries[key]
		if !exists || r.seen[key] != fingerprint {
			entry = couchcore.ResourceSlot{Thread: row.Address, Slot: row.Address.RepoScope + "/" + string(row.Address.Tag)}
			path := row.WorkingPath
			if row.Target.Kind == couchcore.ThreadTargetSlot {
				path = row.Target.Slot.WorktreeRoot
				entry.Slot = fmt.Sprintf("%s:%d", row.Target.Slot.Repo, row.Target.Slot.Number)
			}
			if r.workspace != nil {
				if id, err := r.workspace(ctx, path); err == nil && id.Address != nil && id.RepoIdentity != "" {
					entry.Workspace = id
					entry.Slot = *id.Address
				}
			}
			if r.probe != nil && row.Address != (couchcore.ThreadAddress{}) {
				binding, err := r.probe(ctx, row.Address)
				if err == nil {
					sessions[row.Address] = binding.Name
				} else {
					delete(sessions, row.Address)
				}
			}
		}
		nextSeen[key] = fingerprint
		nextEntries[key] = entry
		next = append(next, entry)
		nextSessions[row.Address] = sessions[row.Address]
	}
	// A canceled pass must not erase rows it did not get to inspect.
	if ctx.Err() != nil {
		return
	}
	// Retire removed proof addresses, never a second empty-address slot.
	retained := map[couchcore.ThreadAddress]bool{}
	for _, row := range next {
		retained[row.Thread] = true
	}
	for _, row := range old {
		if !retained[row.Thread] {
			r.cache.Forget(row.Thread)
		}
	}
	r.seen = nextSeen
	r.entries = nextEntries
	r.mu.Lock()
	r.rows = next
	r.sourceRows = append([]couchcore.ActionableThreadSummary(nil), snapshot.Rows...)
	r.hosted = append([]couchcore.LiveTTYObservation(nil), snapshot.Hosted...)
	r.sessions = nextSessions
	r.mu.Unlock()
}

// slots performs no process, filesystem, git or terminal IO. Wrapper admission
// can update the shared proof cache independently of the inventory worker.
func (r *resourceInventory) slots() []couchcore.ResourceSlot {
	r.mu.Lock()
	rows := append([]couchcore.ResourceSlot(nil), r.rows...)
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
			if len(rows) >= couchcore.ResourceMaxSlots {
				continue
			}
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
	sort.Slice(rows, func(i, j int) bool { return rows[i].Slot < rows[j].Slot })
	return rows
}
