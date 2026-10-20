package couchcmd

import (
	"context"
	"errors"
	"fmt"
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

// hubTargetMaxEntries caps remembered references; each write first sweeps
// entries too old to answer, then drops the oldest beyond the cap.
const hubTargetMaxEntries = 256

// errSlotCatalogWarming is admission's refusal when a target's first fresh
// resolution has not finished inside the admission budget.
var errSlotCatalogWarming = errors.New("slot catalog is warming; nothing was queued; retry in a few seconds")

// hubTargetEntry keeps when its resolution started, which orders results, and
// when it finished, which ages it: a slow failed resolution still answers
// for the full hubTargetFailureAge after it ends.
type hubTargetEntry struct {
	slot              couchcore.HubSlot
	err               error
	started, finished time.Time
}

func (e hubTargetEntry) live(now time.Time, maxAge time.Duration) bool {
	if e.err != nil {
		return now.Sub(e.finished) < hubTargetFailureAge
	}
	return now.Sub(e.finished) < maxAge
}

// hubTargetKey canonicalizes a repo:N reference; anything else keys as given.
func hubTargetKey(ref string) string {
	if parsed, ok, err := couchcore.ParseWorkspaceReference(ref); ok && err == nil {
		return parsed.String()
	}
	return ref
}

// hubTargetMemo remembers each target reference's HubSlot for admission
// (pair#726). Repository discovery fills the admission snapshot in one flight;
// individual fresh resolutions at queue prepare and the effect gate refresh
// their targets. All flights belong to the supervisor lifetime; admission waits
// only within its own budget.
type hubTargetMemo struct {
	resolve  func(context.Context, string) (couchcore.HubSlot, error)
	names    func(context.Context) ([]couchcore.RepositoryName, error)
	discover func(context.Context, string) (map[int]couchcore.HubSlot, error)
	now      func() time.Time
	maxAge   time.Duration
	lifetime context.Context

	mu       sync.Mutex
	entries  map[string]hubTargetEntry
	inflight map[string]*hubTargetFlight
	wg       sync.WaitGroup
}

type hubTargetFlight struct {
	done    chan struct{}
	slot    couchcore.HubSlot
	err     error
	slots   map[int]couchcore.HubSlot
	started time.Time
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
	started := m.now()
	slot, err := m.resolve(ctx, ref)
	if err == nil || ctx.Err() == nil {
		m.remember(ref, slot, err, started)
	}
	return slot, err
}

// Warm answers from a young entry with no IO. Cold references share one
// repository discovery; missing/unverified slots retain the fresh target path.
// Neither warm source grants effect authority: Fresh always resolves again.
func (m *hubTargetMemo) Warm(ctx context.Context, ref string) (couchcore.HubSlot, error) {
	ctx, cancel := context.WithTimeout(ctx, slotAdmissionBudget)
	defer cancel()
	if e, ok := m.entry(hubTargetKey(ref)); ok {
		return e.slot, e.err
	}
	if parsed, ok, err := couchcore.ParseWorkspaceReference(ref); ok && err == nil && m.names != nil && m.discover != nil {
		names, err := m.names(ctx)
		if err != nil {
			return couchcore.HubSlot{}, err
		}
		repository, _, err := couchcore.ResolveRepositoryName(parsed.Repo, names)
		if err == nil {
			key := repositoryFlightKey(repository.Key)
			if e, ok := m.entry(key); ok {
				if e.err != nil {
					return couchcore.HubSlot{}, e.err
				}
			} else {
				flight := m.repositoryFlight(repository.Key)
				if err := waitHubTargetFlight(ctx, flight); err != nil {
					return couchcore.HubSlot{}, err
				}
				if flight.err != nil {
					return couchcore.HubSlot{}, flight.err
				}
				// Resolve the requested prefix too; it need not be an exact name/alias.
				if slot, ok := flight.slots[parsed.Number]; ok {
					m.remember(ref, slot, nil, flight.started)
				}
			}
			if e, ok := m.entry(hubTargetKey(ref)); ok {
				return e.slot, e.err
			}
			// A completed repository snapshot may have been prewarmed before this
			// prefix was requested. Canonical names use the same validated identity.
			canonical := couchcore.WorkspaceReference{Repo: repository.Dir, Number: parsed.Number}.String()
			if e, ok := m.entry(canonical); ok {
				return e.slot, e.err
			}
		}
	}
	flight := m.startFlight(hubTargetKey(ref), func(f *hubTargetFlight) {
		f.slot, f.err = m.Fresh(m.lifetime, ref)
	})
	if err := waitHubTargetFlight(ctx, flight); err != nil {
		return couchcore.HubSlot{}, err
	}
	return flight.slot, flight.err
}

func repositoryFlightKey(primary string) string { return "\x00repository:" + primary }

func (m *hubTargetMemo) entry(key string) (hubTargetEntry, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[key]
	return e, ok && e.live(m.now(), m.maxAge)
}

// All kinds of flight share one capacity and the supervisor's lifetime.
func (m *hubTargetMemo) startFlight(key string, run func(*hubTargetFlight)) *hubTargetFlight {
	m.mu.Lock()
	defer m.mu.Unlock()
	if f, ok := m.inflight[key]; ok {
		return f
	}
	// Recheck under the flight lock: another flight may have completed since
	// the caller's cache read. Reuse its outcome rather than discovering twice.
	if e, ok := m.entries[key]; ok && e.live(m.now(), m.maxAge) {
		f := &hubTargetFlight{done: make(chan struct{}), slot: e.slot, err: e.err, started: e.started}
		close(f.done)
		return f
	}
	if m.lifetime.Err() != nil || len(m.inflight) >= hubTargetMaxFlights {
		return nil
	}
	f := &hubTargetFlight{done: make(chan struct{}), started: m.now()}
	m.inflight[key] = f
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		run(f)
		m.mu.Lock()
		delete(m.inflight, key)
		close(f.done)
		m.mu.Unlock()
	}()
	return f
}

func waitHubTargetFlight(ctx context.Context, f *hubTargetFlight) error {
	if f == nil {
		return errSlotCatalogWarming
	}
	select {
	case <-f.done:
		return nil
	case <-ctx.Done():
		return errSlotCatalogWarming
	}
}

func (m *hubTargetMemo) repositoryFlight(primary string) *hubTargetFlight {
	return m.startFlight(repositoryFlightKey(primary), func(f *hubTargetFlight) {
		names, err := m.names(m.lifetime)
		if err == nil {
			f.slots, err = m.discover(m.lifetime, primary)
		}
		f.err = err
		if m.lifetime.Err() != nil {
			return
		}
		// Only unambiguous names are populated. Admission may use this snapshot,
		// but the operation still re-proves its exact target before effects.
		if err == nil {
			for _, name := range names {
				if name.Key != primary {
					continue
				}
				for _, raw := range []string{name.Dir, name.Alias} {
					resolved, _, err := couchcore.ResolveRepositoryName(raw, names)
					if err != nil || resolved.Key != primary {
						continue
					}
					for number, slot := range f.slots {
						m.remember(fmt.Sprintf("%s:%d", raw, number), slot, nil, f.started)
					}
				}
			}
		}
		m.remember(repositoryFlightKey(primary), couchcore.HubSlot{}, err, f.started)
	})
}

// Prewarm starts one flight for each distinct hub repository without delaying
// startup. Cancel the memo lifetime, then wait, to join all background work.
func (m *hubTargetMemo) Prewarm(primaries []string) {
	if m.names == nil || m.discover == nil {
		return
	}
	seen := map[string]bool{}
	for _, primary := range primaries {
		if primary == "" || seen[primary] {
			continue
		}
		seen[primary] = true
		if _, ok := m.entry(repositoryFlightKey(primary)); !ok {
			m.repositoryFlight(primary)
		}
	}
}

// remember keeps a resolution stamped with the time it started, so an older
// resolution finishing late never replaces a newer one.
func (m *hubTargetMemo) remember(ref string, slot couchcore.HubSlot, err error, started time.Time) {
	key := hubTargetKey(ref)
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.entries[key]; ok && old.started.After(started) {
		return
	}
	now := m.now()
	for k, e := range m.entries {
		if !e.live(now, m.maxAge) {
			delete(m.entries, k)
		}
	}
	for len(m.entries) >= hubTargetMaxEntries {
		oldest := ""
		for k, e := range m.entries {
			if oldest == "" || e.finished.Before(m.entries[oldest].finished) {
				oldest = k
			}
		}
		delete(m.entries, oldest)
	}
	m.entries[key] = hubTargetEntry{slot: slot, err: err, started: started, finished: now}
}

// wait joins outstanding resolutions; the lifetime's end cancels them.
func (m *hubTargetMemo) wait() { m.wg.Wait() }
