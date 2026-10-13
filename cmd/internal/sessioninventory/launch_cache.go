package sessioninventory

import (
	"sync"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// OwnerLaunchCache keeps each owner's current launch, as its ledger last
// said, between queries (#507). Ledgers are append-only and change only at a
// launch or a binding, but run to megabytes, and Couch's idle fading asks of
// every live thread each pass. An entry stands while its ledger's
// fingerprint does; any change re-parses. In memory only: one entry per
// owner a long-lived caller asked about, gone with the caller.
type OwnerLaunchCache struct {
	mu      sync.Mutex
	entries map[ownerLaunchKey]ownerLaunchEntry
}

// OwnerLaunchCacher is a Runtime that keeps an OwnerLaunchCache.
type OwnerLaunchCacher interface {
	OwnerLaunchCache() *OwnerLaunchCache
}

type ownerLaunchKey struct {
	// rootPath tells scopes apart: every scope's Pair root has one name.
	rootPath string
	ledger   Artifact
	scopeKey string
	tag      string
	agent    Agent
}

type ownerLaunchEntry struct {
	fingerprint ArtifactFingerprint
	current     sessionledger.Current
	ok          bool
	diagnostics []Diagnostic
}

// answer is the entry for one caller: its own Current (the binding record and
// the bindings slice copied, so a caller's change or a pointer into them never
// reaches the cache) and diagnostics. A Record's nested slices stay shared and
// are read-only, as every reader of a parsed ledger already treats them.
func (e ownerLaunchEntry) answer() (sessionledger.Current, []Diagnostic) {
	current := e.current
	if current.Binding != nil {
		binding := *current.Binding
		current.Binding = &binding
	}
	current.Bindings = append([]sessionledger.Record(nil), current.Bindings...)
	return current, append([]Diagnostic(nil), e.diagnostics...)
}

func NewOwnerLaunchCache() *OwnerLaunchCache {
	return &OwnerLaunchCache{entries: map[ownerLaunchKey]ownerLaunchEntry{}}
}

func ownerLaunchCacheOf(runtime Runtime) *OwnerLaunchCache {
	if cacher, ok := runtime.(OwnerLaunchCacher); ok {
		return cacher.OwnerLaunchCache()
	}
	return nil
}

// get and put are no-ops on a nil cache, the uncached runtime's.
func (c *OwnerLaunchCache) get(key ownerLaunchKey, fingerprint ArtifactFingerprint) (ownerLaunchEntry, bool) {
	if c == nil {
		return ownerLaunchEntry{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || !equalFingerprint(entry.fingerprint, fingerprint) {
		return ownerLaunchEntry{}, false
	}
	return entry, true
}

func (c *OwnerLaunchCache) put(key ownerLaunchKey, entry ownerLaunchEntry) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = entry
}
