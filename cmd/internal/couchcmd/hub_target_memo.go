package couchcmd

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// hubTargetMemoAge bounds how old a remembered target slot may be when
// admission uses it. Slot topology changes rarely, and queue prepare and the
// effect gate re-prove authority on fresh discovery before any effect.
const hubTargetMemoAge = 10 * time.Minute

// errSlotCatalogWarming is admission's refusal when a target's first fresh
// resolution has not finished inside the admission budget.
var errSlotCatalogWarming = errors.New("slot catalog is warming; nothing was queued; retry in a few seconds")

type hubTargetEntry struct {
	slot couchcore.HubSlot
	at   time.Time
}

// hubTargetMemo remembers each target reference's HubSlot for admission
// (pair#726). Only the fresh resolver fills it: every fresh resolution, at
// admission, queue prepare or the effect gate, refreshes the entry. A cold or
// expired entry starts one single-flight fresh resolution under the
// supervisor's lifetime; admission waits for it only within its own budget.
type hubTargetMemo struct {
	resolve  func(context.Context, string) (couchcore.HubSlot, error)
	now      func() time.Time
	maxAge   time.Duration
	lifetime context.Context

	mu       sync.Mutex
	entries  map[string]hubTargetEntry
	inflight map[string]*hubTargetFlight
	wg       sync.WaitGroup
}

type hubTargetFlight struct {
	done chan struct{}
	slot couchcore.HubSlot
	err  error
}

func newHubTargetMemo(lifetime context.Context, resolve func(context.Context, string) (couchcore.HubSlot, error), now func() time.Time) *hubTargetMemo {
	if now == nil {
		now = time.Now
	}
	if lifetime == nil {
		lifetime = context.Background()
	}
	return &hubTargetMemo{resolve: resolve, now: now, maxAge: hubTargetMemoAge, lifetime: lifetime, entries: map[string]hubTargetEntry{}, inflight: map[string]*hubTargetFlight{}}
}

// Fresh resolves now, on the caller's context, and remembers a success.
func (m *hubTargetMemo) Fresh(ctx context.Context, ref string) (couchcore.HubSlot, error) {
	slot, err := m.resolve(ctx, ref)
	if err == nil {
		m.remember(ref, slot)
	}
	return slot, err
}

// Warm answers from a young entry with no IO. Otherwise it joins or starts the
// reference's single-flight resolution and waits until ctx ends; an
// unfinished resolution keeps running and fills the memo for the next caller.
func (m *hubTargetMemo) Warm(ctx context.Context, ref string) (couchcore.HubSlot, error) {
	m.mu.Lock()
	if e, ok := m.entries[ref]; ok && m.now().Sub(e.at) < m.maxAge {
		m.mu.Unlock()
		return e.slot, nil
	}
	flight, ok := m.inflight[ref]
	if !ok {
		flight = &hubTargetFlight{done: make(chan struct{})}
		m.inflight[ref] = flight
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			flight.slot, flight.err = m.Fresh(m.lifetime, ref)
			m.mu.Lock()
			delete(m.inflight, ref)
			m.mu.Unlock()
			close(flight.done)
		}()
	}
	m.mu.Unlock()
	select {
	case <-flight.done:
		return flight.slot, flight.err
	case <-ctx.Done():
		return couchcore.HubSlot{}, errSlotCatalogWarming
	}
}

func (m *hubTargetMemo) remember(ref string, slot couchcore.HubSlot) {
	m.mu.Lock()
	m.entries[ref] = hubTargetEntry{slot: slot, at: m.now()}
	m.mu.Unlock()
}

// wait joins outstanding resolutions; the lifetime's end cancels them.
func (m *hubTargetMemo) wait() { m.wg.Wait() }
