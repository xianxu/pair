package assets

import (
	"bytes"
	"context"
	"sync"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// MemStore is the stateful fake Store: bodies in memory by hash, URL
// bindings, no resampling (every body is exact).
type MemStore struct {
	mu     sync.Mutex
	bodies map[string][]byte
	urls   map[string]*recording.Asset
}

func NewMemStore() *MemStore {
	return &MemStore{bodies: map[string][]byte{}, urls: map[string]*recording.Asset{}}
}

func (m *MemStore) Put(ctx context.Context, c Capture) (*recording.Asset, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	hash := recording.HashOf(c.Body)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.bodies[hash]; !ok {
		m.bodies[hash] = bytes.Clone(c.Body)
	}
	a := &recording.Asset{URL: c.URL, Hash: hash, MIME: c.MIME, Size: int64(len(c.Body))}
	m.urls[c.URL] = a
	return a, nil
}

func (m *MemStore) Get(hash string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bodies[hash]
	if !ok {
		return nil, ErrNotFound
	}
	return bytes.Clone(b), nil
}

func (m *MemStore) Lookup(url string) (*recording.Asset, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.urls[url]
	return a, ok
}
