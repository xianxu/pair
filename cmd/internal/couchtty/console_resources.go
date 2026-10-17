package couchtty

import (
	"maps"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// ResourceInventorySnapshot contains only cached inventory and hosted processes.
// Generation changes on inventory replacement, successful attach and child exit.
type ResourceInventorySnapshot struct {
	Generation     uint64
	InventoryKnown bool
	Rows           []couchcore.ActionableThreadSummary
	Hosted         []couchcore.LiveTTYObservation
}

// ResourceChanges coalesces notifications. The consumer owns its worker and
// cancellation; this channel is never closed, so disposal cannot race a sender.
// Probes belong to that worker, never to the Console mutex or event loop.
func (c *Console) ResourceChanges() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.resourceChanges == nil {
		c.resourceChanges = make(chan struct{}, 1)
	}
	return c.resourceChanges
}
func (c *Console) resourcesChangedLocked() {
	c.resourceGeneration++
	if c.resourceChanges != nil {
		select {
		case c.resourceChanges <- struct{}{}:
		default:
		}
	}
}
func (c *Console) ResourceInventory() ResourceInventorySnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return ResourceInventorySnapshot{Generation: c.resourceGeneration, InventoryKnown: c.resourceRowsKnown, Rows: cloneResourceRows(c.resourceRows), Hosted: c.snapshotMenuObservationsLocked()}
}

// HostedIngestSnapshot copies cumulative byte totals for the current hosted
// children. It deliberately avoids the endpoint parser mutex: diagnostics must
// remain available while a hidden client is producing output.
func (c *Console) HostedIngestSnapshot() couchcore.HostedIngestSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	snapshot := couchcore.HostedIngestSnapshot{
		SampledAt: time.Now(),
		Clients:   make([]couchcore.HostedIngestClient, 0, min(len(c.order), couchcore.ResourceMaxSlots)),
	}
	for _, id := range c.order {
		p := c.panes[id]
		if p == nil || p.child == nil {
			continue
		}
		select {
		case <-p.child.Exited():
			continue
		default:
		}
		if len(snapshot.Clients) == couchcore.ResourceMaxSlots {
			snapshot.Omitted++
			continue
		}
		snapshot.Clients = append(snapshot.Clients, couchcore.HostedIngestClient{
			EndpointID:  p.child.Endpoint().ID(),
			Thread:      p.thread,
			Process:     couchcore.ResourceProcessKey{PID: p.process.PID, Identity: p.process.Identity},
			IngestBytes: p.child.IngestBytes(),
		})
	}
	return snapshot
}

// Summary contains pointer-bearing display metadata. Copy that too, so no
// consumer can mutate the console's projection through a snapshot.
func cloneResourceRows(rows []couchcore.ActionableThreadSummary) []couchcore.ActionableThreadSummary {
	out := append([]couchcore.ActionableThreadSummary(nil), rows...)
	for i := range out {
		if rows[i].TitleTarget != nil {
			v := *rows[i].TitleTarget
			v.Sessions = maps.Clone(v.Sessions)
			out[i].TitleTarget = &v
		}
		if rows[i].Recovery != nil {
			v := *rows[i].Recovery
			out[i].Recovery = &v
		}
		if rows[i].Continuation != nil {
			v := *rows[i].Continuation
			out[i].Continuation = &v
		}
		if rows[i].ContinuationMissed != nil {
			v := *rows[i].ContinuationMissed
			out[i].ContinuationMissed = &v
		}
		if rows[i].Orphan != nil {
			v := *rows[i].Orphan
			out[i].Orphan = &v
		}
	}
	return out
}
