package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// SessionNameIndexCacheIO observes real reads separately from decoder input.
// Nil functions select the production filesystem and strict decoder.
type SessionNameIndexCacheIO struct {
	Stat     func(string) (os.FileInfo, error)
	ReadFile func(string) ([]byte, error)
	Decode   func(string) (SessionNameIndex, error)
}

type indexCacheState struct {
	generation uint64
	ready      bool
}
type indexCacheEvent struct {
	begin, commit bool
	generation    uint64
}

func advanceIndexCache(s indexCacheState, e indexCacheEvent) indexCacheState {
	if e.begin {
		return indexCacheState{generation: s.generation + 1}
	}
	if e.commit && e.generation == s.generation {
		s.ready = true
	}
	return s
}

type indexCacheEntry struct {
	state       indexCacheState
	fingerprint sessioninventory.ArtifactFingerprint
	index       SessionNameIndex
	loading     chan struct{}
	used        uint64
}

// SessionNameIndexCache retains only the latest binding per owner for each
// stable regular-file generation. It never retains live process authority.
type SessionNameIndexCache struct {
	mu      sync.Mutex
	entries map[string]*indexCacheEntry
	clock   uint64
	io      SessionNameIndexCacheIO
}

func NewSessionNameIndexCache() *SessionNameIndexCache {
	return NewSessionNameIndexCacheWithIO(SessionNameIndexCacheIO{})
}
func NewSessionNameIndexCacheWithIO(io SessionNameIndexCacheIO) *SessionNameIndexCache {
	if io.Stat == nil {
		io.Stat = os.Lstat
	}
	if io.ReadFile == nil {
		io.ReadFile = os.ReadFile
	}
	if io.Decode == nil {
		io.Decode = DecodeSessionNameIndex
	}
	return &SessionNameIndexCache{entries: make(map[string]*indexCacheEntry), io: io}
}
func compactSessionNameIndex(index SessionNameIndex) SessionNameIndex {
	type owner struct{ scope, tag string }
	seen := map[owner]bool{}
	out := SessionNameIndex{}
	for i := len(index.Entries) - 1; i >= 0; i-- {
		e := index.Entries[i]
		k := owner{e.ScopeKey, e.Tag}
		if !seen[k] {
			seen[k] = true
			out.Entries = append(out.Entries, e)
		}
	}
	for i, j := 0, len(out.Entries)-1; i < j; i, j = i+1, j-1 {
		out.Entries[i], out.Entries[j] = out.Entries[j], out.Entries[i]
	}
	return out
}
func cloneIndex(index SessionNameIndex) SessionNameIndex {
	return SessionNameIndex{Entries: append([]SessionNameEntry(nil), index.Entries...)}
}
func (c *SessionNameIndexCache) fingerprint(path string) (sessioninventory.ArtifactFingerprint, error) {
	info, err := c.io.Stat(path)
	if err != nil {
		return sessioninventory.ArtifactFingerprint{}, err
	}
	if !info.Mode().IsRegular() {
		return sessioninventory.ArtifactFingerprint{}, fmt.Errorf("not a regular session-name index: %s", path)
	}
	return sessioninventory.FingerprintFileInfo(path, info)
}
func sameIndexFingerprint(a, b sessioninventory.ArtifactFingerprint) bool {
	return a.StableFileID == b.StableFileID && a.GenerationToken == b.GenerationToken && a.MutationToken == b.MutationToken && a.Size == b.Size && indexTimeEqual(a.ModTime, b.ModTime) && indexTimeEqual(a.BirthTime, b.BirthTime)
}
func indexTimeEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

// Read preserves legacy-first, scoped-second precedence without retaining raw
// history. Errors never return a prior successful projection.
func (c *SessionNameIndexCache) Read(ctx context.Context, globalDataDir, selectedScopeDir string) (SessionNameIndex, error) {
	legacy, err := artifactpath.ResolveLegacyRoot(globalDataDir)
	if err != nil {
		return SessionNameIndex{}, err
	}
	scope, err := artifactpath.ResolveSelectedScope(selectedScopeDir)
	if err != nil {
		return SessionNameIndex{}, err
	}
	paths := []string{legacy.SessionBindings()}
	if p := scope.SessionBindings(); p != paths[0] {
		paths = append(paths, p)
	}
	var out SessionNameIndex
	for _, path := range paths {
		index, e := c.readOne(ctx, path)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return SessionNameIndex{}, fmt.Errorf("read session-name index %s: %w", path, e)
		}
		out.Entries = append(out.Entries, index.Entries...)
	}
	if err := ctx.Err(); err != nil {
		return SessionNameIndex{}, err
	}
	return compactSessionNameIndex(out), nil
}
func (c *SessionNameIndexCache) readOne(ctx context.Context, path string) (SessionNameIndex, error) {
	for {
		if err := ctx.Err(); err != nil {
			return SessionNameIndex{}, err
		}
		fp, err := c.fingerprint(path)
		if canceled := ctx.Err(); canceled != nil {
			return SessionNameIndex{}, canceled
		}
		c.mu.Lock()
		entry := c.entries[path]
		if err != nil {
			if entry != nil && entry.loading == nil {
				delete(c.entries, path)
			}
			c.mu.Unlock()
			return SessionNameIndex{}, err
		}
		if entry != nil && entry.loading != nil {
			wait := entry.loading
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return SessionNameIndex{}, ctx.Err()
			case <-wait:
				continue
			}
		}
		c.clock++
		if entry != nil && entry.state.ready && sameIndexFingerprint(entry.fingerprint, fp) {
			entry.used = c.clock
			answer := cloneIndex(entry.index)
			c.mu.Unlock()
			return answer, nil
		}
		if entry == nil {
			if len(c.entries) >= 128 {
				var oldest string
				var tick uint64 = ^uint64(0)
				for key, e := range c.entries {
					if e.loading == nil && e.used < tick {
						oldest, tick = key, e.used
					}
				}
				if oldest != "" {
					delete(c.entries, oldest)
				}
			}
			entry = &indexCacheEntry{}
			if len(c.entries) < 128 {
				c.entries[path] = entry
			}
		}
		entry.state = advanceIndexCache(entry.state, indexCacheEvent{begin: true})
		token := entry.state.generation
		entry.loading = make(chan struct{})
		entry.used = c.clock
		c.mu.Unlock()
		index, stamp, loadErr := c.load(ctx, path, fp)
		c.mu.Lock()
		if loadErr == nil && ctx.Err() == nil && c.entries[path] == entry {
			entry.state = advanceIndexCache(entry.state, indexCacheEvent{commit: true, generation: token})
			entry.fingerprint = stamp
			entry.index = index
		}
		close(entry.loading)
		entry.loading = nil
		c.mu.Unlock()
		if e := ctx.Err(); e != nil {
			return SessionNameIndex{}, e
		}
		return cloneIndex(index), loadErr
	}
}
func (c *SessionNameIndexCache) load(ctx context.Context, path string, before sessioninventory.ArtifactFingerprint) (SessionNameIndex, sessioninventory.ArtifactFingerprint, error) {
	for attempt := 0; attempt < 2; attempt++ {
		if err := ctx.Err(); err != nil {
			return SessionNameIndex{}, before, err
		}
		raw, err := c.io.ReadFile(path)
		if err != nil {
			return SessionNameIndex{}, before, err
		}
		if err := ctx.Err(); err != nil {
			return SessionNameIndex{}, before, err
		}
		index, err := c.io.Decode(string(raw))
		if err != nil {
			return SessionNameIndex{}, before, err
		}
		after, err := c.fingerprint(path)
		if err != nil {
			return SessionNameIndex{}, before, err
		}
		if int64(len(raw)) == after.Size && sameIndexFingerprint(before, after) {
			return compactSessionNameIndex(index), after, nil
		}
		before = after
	}
	return SessionNameIndex{}, before, errors.New("session-name index changed during both read attempts")
}
