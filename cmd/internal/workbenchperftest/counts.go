// Package workbenchperftest provides counted-work fixtures for consumer tests.
package workbenchperftest

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// Key keeps the artifact family, owner and source identity independently visible.
type Key struct{ Family, Owner, Path string }

// Counter records filesystem bytes and decoder input bytes separately.
// Its zero value is ready for use, including concurrent callbacks.
type Counter struct {
	mu            sync.Mutex
	read, decoded map[Key]int64
}

func (c *Counter) RecordRead(family, owner, path string, n int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.read == nil {
		c.read = map[Key]int64{}
	}
	c.read[Key{family, owner, path}] += n
}
func (c *Counter) RecordDecode(family, owner, path string, n int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.decoded == nil {
		c.decoded = map[Key]int64{}
	}
	c.decoded[Key{family, owner, path}] += n
}

// Snapshot owns independent maps; callers may retain or modify it freely.
type Snapshot struct{ Read, Decoded map[Key]int64 }

func (c *Counter) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := Snapshot{Read: map[Key]int64{}, Decoded: map[Key]int64{}}
	for k, v := range c.read {
		s.Read[k] = v
	}
	for k, v := range c.decoded {
		s.Decoded[k] = v
	}
	return s
}

// Delta preserves disappeared keys and negative differences to expose resets.
func (s Snapshot) Delta(before Snapshot) Snapshot {
	d := Snapshot{Read: map[Key]int64{}, Decoded: map[Key]int64{}}
	for k, v := range s.Read {
		d.Read[k] = v
	}
	for k, v := range before.Read {
		d.Read[k] -= v
	}
	for k, v := range s.Decoded {
		d.Decoded[k] = v
	}
	for k, v := range before.Decoded {
		d.Decoded[k] -= v
	}
	return d
}
func (s Snapshot) ReadBytes() int64 {
	var n int64
	for _, v := range s.Read {
		n += v
	}
	return n
}
func (s Snapshot) DecodeBytes() int64 {
	var n int64
	for _, v := range s.Decoded {
		n += v
	}
	return n
}
func (s Snapshot) ReadFamily(family string) int64 {
	var n int64
	for k, v := range s.Read {
		if k.Family == family {
			n += v
		}
	}
	return n
}
func (s Snapshot) DecodeFamily(family string) int64 {
	var n int64
	for k, v := range s.Decoded {
		if k.Family == family {
			n += v
		}
	}
	return n
}
func (s Snapshot) RequireZero(t testing.TB) {
	t.Helper()
	for k, n := range s.Read {
		if n != 0 {
			t.Errorf("unchanged work read %+v bytes=%d", k, n)
		}
	}
	for k, n := range s.Decoded {
		if n != 0 {
			t.Errorf("unchanged work decoded %+v bytes=%d", k, n)
		}
	}
}

type queryObserver struct {
	counter *Counter
	owner   string
}

func (c *Counter) Observer(owner string) sessioninventory.QueryWorkObserver {
	return queryObserver{c, owner}
}
func artifactFamily(a sessioninventory.Artifact) string {
	if a.StorageRoot != "pair-data" {
		return "native"
	}
	if artifactpath.IsLedgerHistorySidecar(filepath.Base(a.RelativePath)) {
		return "ledger"
	}
	return "catalog"
}
func (o queryObserver) ReadWork(a sessioninventory.Artifact, n int64) {
	o.counter.RecordRead(artifactFamily(a), o.owner, a.StorageRoot+"/"+a.RelativePath, n)
}
func (o queryObserver) DecodeWork(a sessioninventory.Artifact, n int64) {
	o.counter.RecordDecode(artifactFamily(a), o.owner, a.StorageRoot+"/"+a.RelativePath, n)
}
