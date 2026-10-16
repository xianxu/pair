package couchcore

import (
	"github.com/xianxu/pair/cmd/internal/launcher"
	"sort"
	"sync"
)

// ResourceRootEvidence is diagnostic attribution, never permission for effects.
// Unknown evidence carries no server. Revision orders initiation, not completion.
type ResourceRootEvidence struct {
	Address        ThreadAddress
	Session        string
	Server         launcher.SessionServerIdentity
	Revision       uint64
	Known, Present bool
}

const resourceRootCacheLimit = 4096

// ResourceRootCache shares root observations across checker copies. It is
// bounded for the supervisor lifetime; Forget releases a removed address.
type ResourceRootCache struct {
	mu       sync.Mutex
	sequence uint64
	rows     map[ThreadAddress]ResourceRootEvidence
}

func NewResourceRootCache() *ResourceRootCache {
	return &ResourceRootCache{rows: map[ThreadAddress]ResourceRootEvidence{}}
}
func (c *ResourceRootCache) Begin(address ThreadAddress, session string) uint64 {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rows == nil {
		c.rows = map[ThreadAddress]ResourceRootEvidence{}
	}
	if _, exists := c.rows[address]; !exists && len(c.rows) >= resourceRootCacheLimit {
		return 0
	}
	c.sequence++
	c.rows[address] = ResourceRootEvidence{Address: address, Session: session, Revision: c.sequence}
	return c.sequence
}
func (c *ResourceRootCache) Complete(address ThreadAddress, revision uint64, owner launcher.SessionOwnerObservation, err error) {
	if c == nil || revision == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	row, ok := c.rows[address]
	if !ok || row.Revision != revision {
		return
	}
	if err == nil {
		switch owner.State {
		case launcher.SessionOwnerOwned:
			if owner.Name == row.Session && owner.Server.Session == row.Session && owner.Server.PID > 0 && owner.Server.Identity != "" {
				row.Known, row.Present, row.Server = true, true, owner.Server
			}
		case launcher.SessionOwnerAbsent, launcher.SessionOwnerForeign:
			row.Known = true
		}
	}
	c.rows[address] = row
}
func (c *ResourceRootCache) Forget(address ThreadAddress) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.rows, address)
}
func (c *ResourceRootCache) Snapshot() []ResourceRootEvidence {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]ResourceRootEvidence, 0, len(c.rows))
	for _, r := range c.rows {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Address.RepoScope != out[j].Address.RepoScope {
			return out[i].Address.RepoScope < out[j].Address.RepoScope
		}
		return out[i].Address.Tag < out[j].Address.Tag
	})
	return out
}
func (c ScopedThreadArtifactCollisionChecker) ResourceRootEvidenceCache() *ResourceRootCache {
	return c.ResourceRoots
}
