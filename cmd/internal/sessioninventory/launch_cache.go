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
	loading map[ownerLaunchKey]*ownerLaunchLoad
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
	previous    *ResumeTarget
	ok          bool
	diagnostics []Diagnostic
	used        uint64
}

// compactOwnerLaunch keeps just the fields consumed by owner queries. The
// singular binding proof remains activity authority; launch baselines and
// conflicting binding payloads have no role after the strict reducer runs.
func compactOwnerLaunch(current sessionledger.Current) sessionledger.Current {
	launch := current.Launch
	current.Launch = sessionledger.Record{Ordinal: launch.Ordinal, Agent: launch.Agent, RequestedNativeID: launch.RequestedNativeID, RequestOrigin: launch.RequestOrigin}
	current.Bindings = nil
	if current.Binding != nil {
		binding := current.Binding
		current.Binding = &sessionledger.Record{Version: binding.Version, RootNativeID: binding.RootNativeID, AuthorizationProof: binding.AuthorizationProof}
	}
	return current
}
func cloneLaunch(current sessionledger.Current) sessionledger.Current {
	if current.Binding != nil {
		binding := *current.Binding
		current.Binding = &binding
	}
	return current
}
func (e ownerLaunchEntry) answer() (sessionledger.Current, *ResumeTarget, []Diagnostic) {
	var previous *ResumeTarget
	if e.previous != nil {
		copy := *e.previous
		previous = &copy
	}
	return cloneLaunch(e.current), previous, append([]Diagnostic(nil), e.diagnostics...)
}

type ownerLaunchLoad struct {
	done    chan struct{}
	invalid bool
}

// invalidateOwner also fences loads started before an actual incomplete source
// observation. An independently canceled waiter does not call this method.
func (c *OwnerLaunchCache) invalidateOwner(root, scope, tag string, agent Agent, fenceLoads bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	matches := func(k ownerLaunchKey) bool {
		return k.rootPath == root && k.scopeKey == scope && k.tag == tag && k.agent == agent
	}
	for k := range c.entries {
		if matches(k) {
			delete(c.entries, k)
		}
	}
	for k, load := range c.loading {
		if fenceLoads && matches(k) {
			load.invalid = true
		}
	}
}
func (c *OwnerLaunchCache) invalidateLoad(key ownerLaunchKey, load *ownerLaunchLoad) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
	if c.loading[key] == load && load != nil {
		load.invalid = true
	}
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
func (c *OwnerLaunchCache) enter(ctx context.Context, key ownerLaunchKey) (func(), *ownerLaunchLoad, error) {
	if c == nil {
		return func() {}, nil, ctx.Err()
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		c.mu.Lock()
		if c.loading == nil {
			c.loading = make(map[ownerLaunchKey]*ownerLaunchLoad)
		}
		wait, busy := c.loading[key]
		if !busy {
			load := &ownerLaunchLoad{done: make(chan struct{})}
			c.loading[key] = load
			c.mu.Unlock()
			return func() { c.mu.Lock(); delete(c.loading, key); close(load.done); c.mu.Unlock() }, load, nil
		}
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-wait.done:
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
func (c *OwnerLaunchCache) put(key ownerLaunchKey, load *ownerLaunchLoad, entry ownerLaunchEntry) bool {
	if c == nil {
		return true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if load == nil || c.loading[key] != load || load.invalid {
		return false
	}
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
	return true
}
