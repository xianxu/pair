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

// hubTargetFailureAge is how long a failed resolution answers admission with
// its own error, so retries learn the cause instead of "warming" forever.
const hubTargetFailureAge = 30 * time.Second

// hubTargetMaxFlights caps concurrent background resolutions.
const hubTargetMaxFlights = 8

// errSlotCatalogWarming is admission's refusal when a target's first fresh
// resolution has not finished inside the admission budget.
var errSlotCatalogWarming = errors.New("slot catalog is warming; nothing was queued; retry in a few seconds")

type hubTargetEntry struct {
	slot couchcore.HubSlot
	err  error
	at   time.Time
}

// hubTargetKey canonicalizes a repo:N reference; anything else keys as given.
func hubTargetKey(ref string) string {
	if parsed, ok, err := couchcore.ParseWorkspaceReference(ref); ok && err == nil {
		return parsed.String()
	}
	return ref
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

// Fresh resolves now, on the caller's context, and remembers the outcome. A
// cancelled resolution proves nothing about the target, so it is not kept.
func (m *hubTargetMemo) Fresh(ctx context.Context, ref string) (couchcore.HubSlot, error) {
	slot, err := m.resolve(ctx, ref)
	if err == nil || ctx.Err() == nil {
		m.remember(ref, slot, err)
	}
	return slot, err
}

// Warm answers from a young entry with no IO. Otherwise it joins or starts the
// reference's single-flight resolution and waits until ctx ends; an
// unfinished resolution keeps running and fills the memo for the next caller.
func (m *hubTargetMemo) Warm(ctx context.Context, ref string) (couchcore.HubSlot, error) {
	key := hubTargetKey(ref)
	m.mu.Lock()
	if e, ok := m.entries[key]; ok {
		age := m.now().Sub(e.at)
		if e.err == nil && age < m.maxAge {
			m.mu.Unlock()
			return e.slot, nil
		}
		if e.err != nil && age < hubTargetFailureAge {
			m.mu.Unlock()
			return couchcore.HubSlot{}, e.err
		}
	}
	flight, ok := m.inflight[key]
	if !ok {
		if len(m.inflight) >= hubTargetMaxFlights {
			m.mu.Unlock()
			return couchcore.HubSlot{}, errSlotCatalogWarming
		}
		flight = &hubTargetFlight{done: make(chan struct{})}
		m.inflight[key] = flight
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			flight.slot, flight.err = m.Fresh(m.lifetime, ref)
			m.mu.Lock()
			delete(m.inflight, key)
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

func (m *hubTargetMemo) remember(ref string, slot couchcore.HubSlot, err error) {
	m.mu.Lock()
	m.entries[hubTargetKey(ref)] = hubTargetEntry{slot: slot, err: err, at: m.now()}
	m.mu.Unlock()
}

// wait joins outstanding resolutions; the lifetime's end cancels them.
func (m *hubTargetMemo) wait() { m.wg.Wait() }
