package sessioninventory

import (
	"context"
	"sync"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// OwnerLaunchCache retains only current and previous-established launch evidence,
// not parsed ledger history. It belongs to a long-lived caller, not the process.
// Current metadata remains authoritative; file IO never holds its map mutex.
type OwnerLaunchCache struct {
	mu      sync.Mutex
	entries map[ownerLaunchKey]ownerLaunchEntry
	loading map[ownerLaunchKey]chan struct{}
	clock   uint64
}

// OwnerLaunchCacher is a Runtime that keeps an OwnerLaunchCache.
type OwnerLaunchCacher interface{ OwnerLaunchCache() *OwnerLaunchCache }

type ownerLaunchKey struct {
	rootPath string
	ledger   Artifact
	scopeKey string
	tag      string
	agent    Agent
}

type ownerLaunchEntry struct {
	fingerprint ArtifactFingerprint
	current     sessionledger.Current
	previous    *sessionledger.Current
	ok          bool
	diagnostics []Diagnostic
	used        uint64
}

func cloneLaunch(current sessionledger.Current) sessionledger.Current {
	if current.Binding != nil {
		binding := *current.Binding
		current.Binding = &binding
	}
	current.Bindings = append([]sessionledger.Record(nil), current.Bindings...)
	return current
}

// A Record's nested evidence is immutable, as it is for parsed ledger readers.
// Copy the mutable projection containers so answers cannot mutate cache state.
func (e ownerLaunchEntry) answer() (sessionledger.Current, *sessionledger.Current, []Diagnostic) {
	var previous *sessionledger.Current
	if e.previous != nil {
		copy := cloneLaunch(*e.previous)
		previous = &copy
	}
	return cloneLaunch(e.current), previous, append([]Diagnostic(nil), e.diagnostics...)
}

func NewOwnerLaunchCache() *OwnerLaunchCache { return &OwnerLaunchCache{} }
func ownerLaunchCacheOf(runtime Runtime) *OwnerLaunchCache {
	if cacher, ok := runtime.(OwnerLaunchCacher); ok {
		return cacher.OwnerLaunchCache()
	}
	return nil
}

// enter serializes just one owner's cold reads. Waiters can cancel and unrelated
// owners run independently. Rechecking metadata after entry prevents a waiter
// from using the fingerprint it observed before waiting.
func (c *OwnerLaunchCache) enter(ctx context.Context, key ownerLaunchKey) (func(), error) {
	if c == nil {
		return func() {}, ctx.Err()
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		c.mu.Lock()
		if c.loading == nil {
			c.loading = make(map[ownerLaunchKey]chan struct{})
		}
		wait, busy := c.loading[key]
		if !busy {
			done := make(chan struct{})
			c.loading[key] = done
			c.mu.Unlock()
			return func() { c.mu.Lock(); delete(c.loading, key); close(done); c.mu.Unlock() }, nil
		}
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-wait:
		}
	}
}

func (c *OwnerLaunchCache) get(key ownerLaunchKey, fingerprint ArtifactFingerprint) (ownerLaunchEntry, bool) {
	if c == nil {
		return ownerLaunchEntry{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return ownerLaunchEntry{}, false
	}
	if !equalFingerprint(entry.fingerprint, fingerprint) {
		delete(c.entries, key)
		return ownerLaunchEntry{}, false
	}
	c.clock++
	entry.used = c.clock
	c.entries[key] = entry
	return entry, true
}
func (c *OwnerLaunchCache) put(key ownerLaunchKey, entry ownerLaunchEntry) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[ownerLaunchKey]ownerLaunchEntry)
	}
	if _, exists := c.entries[key]; !exists && len(c.entries) >= 128 {
		var oldest ownerLaunchKey
		var used uint64
		first := true
		for k, e := range c.entries {
			if first || e.used < used {
				oldest = k
				used = e.used
				first = false
			}
		}
		delete(c.entries, oldest)
	}
	c.clock++
	entry.used = c.clock
	c.entries[key] = entry
}
