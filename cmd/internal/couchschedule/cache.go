package couchschedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"golang.org/x/sys/unix"
)

const DefaultCacheBytes int64 = 256 << 20
const maxCacheNames = 16384

var ErrCacheCapacity = errors.New("scheduled build cache capacity exhausted; live artifacts remain protected")
var errCacheLive = errors.New("scheduled build artifact is in use")

type artifactLease struct {
	file *os.File
	once sync.Once
	err  error
}

func (l *artifactLease) Close() error { l.once.Do(func() { l.err = l.file.Close() }); return l.err }
func cacheKey(key string) bool        { return len(key) == 64 && strings.Trim(key, "0123456789abcdef") == "" }
func cacheLease(cache, key string, mode int) (*artifactLease, error) {
	return cacheFileLease(filepath.Join(cache, key+".lease"), mode)
}
func cacheFileLease(path string, mode int) (*artifactLease, error) {
	fd, e := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0600)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), path)
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		f.Close()
		return nil, errors.New("unsafe cache lease")
	}
	if e = unix.Flock(fd, mode|unix.LOCK_NB); e != nil {
		f.Close()
		if errors.Is(e, unix.EWOULDBLOCK) || errors.Is(e, unix.EAGAIN) {
			return nil, errCacheLive
		}
		return nil, e
	}
	return &artifactLease{file: f}, nil
}
func cacheNames(cache string) ([]string, error) {
	if e := buildSafePath(cache, cache); e != nil {
		return nil, e
	}
	f, e := os.Open(cache)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	ns, e := f.Readdirnames(maxCacheNames + 1)
	if e != nil && !errors.Is(e, io.EOF) {
		return nil, e
	}
	if len(ns) >= maxCacheNames {
		return nil, errors.New("scheduled cache inventory limit reached; permanent lock inodes require offline maintenance")
	}
	sort.Strings(ns)
	return ns, nil
}
func validateCacheShape(cache, dir string) error {
	if e := buildSafePath(cache, dir); e != nil {
		return e
	}
	f, e := os.Open(dir)
	if e != nil {
		return e
	}
	defer f.Close()
	ns, e := f.Readdirnames(3)
	if e != nil && !errors.Is(e, io.EOF) {
		return e
	}
	sort.Strings(ns)
	if len(ns) != 2 || ns[0] != "manifest.json" || ns[1] != "task" {
		return errors.New("cache entry has unknown or missing files")
	}
	for _, name := range ns {
		p := filepath.Join(dir, name)
		if e = buildSafePath(cache, p); e != nil {
			return e
		}
		st, e := os.Lstat(p)
		if e != nil {
			return e
		}
		if !st.Mode().IsRegular() {
			return errors.New("cache payload is not regular")
		}
	}
	return nil
}

type cacheEntry struct {
	key, dir string
	bytes    int64
	used     int64
}

func inspectCache(ctx context.Context, cache string) ([]cacheEntry, error) {
	ns, e := cacheNames(cache)
	if e != nil {
		return nil, e
	}
	var entries []cacheEntry
	for _, name := range ns {
		if e = ctx.Err(); e != nil {
			return nil, e
		}
		if name == ".maintenance.lock" || strings.HasSuffix(name, ".lock") && cacheKey(strings.TrimSuffix(name, ".lock")) || strings.HasSuffix(name, ".lease") && cacheKey(strings.TrimSuffix(name, ".lease")) {
			st, e := os.Lstat(filepath.Join(cache, name))
			if e != nil {
				return nil, e
			}
			if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
				return nil, errors.New("unsafe permanent cache lock")
			}
			continue
		}
		if !cacheKey(name) {
			return nil, fmt.Errorf("unknown cache entry: %s", name)
		}
		dir := filepath.Join(cache, name)
		if e = validateCacheShape(cache, dir); e != nil {
			return nil, e
		}
		raw, e := osfs.ReadRegularMetadata(ctx, filepath.Join(dir, "manifest.json"), 2<<20)
		if e != nil {
			return nil, e
		}
		var m BuildManifest
		if e = strictjson.Decode(raw, &m); e != nil {
			return nil, e
		}
		if m.Version != 1 || m.Key != name || !cacheKey(m.BinarySHA256) {
			return nil, errors.New("invalid cache manifest")
		}
		key := m.Key
		used := m.LastUsed
		m.Key = ""
		m.BinarySHA256 = ""
		m.LastUsed = 0
		canonical, e := json.Marshal(m)
		if e != nil || digestBuild(canonical) != key {
			return nil, errors.New("cache manifest key mismatch")
		}
		st, e := os.Lstat(filepath.Join(dir, "task"))
		if e != nil {
			return nil, e
		}
		if st.Size() < 0 || st.Size() > DefaultCacheBytes || st.Mode().Perm() != 0700 {
			return nil, errors.New("invalid cached binary")
		}
		entries = append(entries, cacheEntry{name, dir, st.Size() + int64(len(raw)), used})
	}
	return entries, nil
}
func removeCacheEntry(cache string, e cacheEntry) error {
	if err := validateCacheShape(cache, e.dir); err != nil {
		return err
	}
	for _, name := range []string{"manifest.json", "task"} {
		if err := os.Remove(filepath.Join(e.dir, name)); err != nil {
			return err
		}
	}
	if err := os.Remove(e.dir); err != nil {
		return err
	}
	return durablefile.SyncDirectory(cache)
}

// Build holds its publication lock before entering this short maintenance
// section. Eviction never waits for a publication or execution lease, so this
// order cannot deadlock another builder waiting for maintenance.
func (b Builder) maintainCache(ctx context.Context, cache string, m BuildManifest, binary []byte) error {
	unlock, e := buildLock(ctx, filepath.Join(cache, ".maintenance.lock"))
	if e != nil {
		return e
	}
	defer unlock()
	entries, e := inspectCache(ctx, cache)
	if e != nil {
		return e
	}
	if binary == nil {
		raw, err := osfs.ReadRegularMetadata(ctx, filepath.Join(cache, m.Key, "manifest.json"), 2<<20)
		if err != nil {
			return err
		}
		if err = strictjson.Decode(raw, &m); err != nil {
			return err
		}
	}
	now := time.Now()
	if b.cacheNow != nil {
		now = b.cacheNow()
	}
	m.LastUsed = now.UnixNano()
	encoded, e := json.Marshal(m)
	if e != nil {
		return e
	}
	budget := b.CacheBytes
	if budget == 0 {
		budget = DefaultCacheBytes
	}
	if budget < 1 || budget > DefaultCacheBytes {
		return errors.New("invalid scheduled cache byte budget")
	}
	dir := filepath.Join(cache, m.Key)
	var total, own int64
	for _, entry := range entries {
		total += entry.bytes
		if entry.key == m.Key {
			own = entry.bytes
		}
	}
	required := own
	if binary != nil {
		required = int64(len(binary) + len(encoded))
	} else {
		st, e := os.Lstat(filepath.Join(dir, "task"))
		if e != nil {
			return e
		}
		required = st.Size() + int64(len(encoded))
	}
	if required > budget {
		return ErrCacheCapacity
	}
	total = total - own + required
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].used == entries[j].used {
			return entries[i].key < entries[j].key
		}
		return entries[i].used < entries[j].used
	})
	for _, entry := range entries {
		if total <= budget {
			break
		}
		if entry.key == m.Key {
			continue
		}
		if e = ctx.Err(); e != nil {
			return e
		}
		pub, e := cacheFileLease(filepath.Join(cache, entry.key+".lock"), unix.LOCK_EX)
		if errors.Is(e, errCacheLive) {
			continue
		}
		if e != nil {
			return e
		}
		live, e := cacheLease(cache, entry.key, unix.LOCK_EX)
		if errors.Is(e, errCacheLive) {
			pub.Close()
			continue
		}
		if e != nil {
			pub.Close()
			return e
		}
		e = removeCacheEntry(cache, entry)
		live.Close()
		pub.Close()
		if e != nil {
			return e
		}
		total -= entry.bytes
	}
	if total > budget {
		return ErrCacheCapacity
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	if binary == nil {
		return durablefile.WriteAtomic(filepath.Join(dir, "manifest.json"), encoded, ".manifest-*")
	}
	if own > 0 {
		if e = removeCacheEntry(cache, cacheEntry{key: m.Key, dir: dir}); e != nil {
			return e
		}
	}
	stage, e := os.MkdirTemp(cache, ".publication-")
	if e != nil {
		return e
	}
	defer func() {
		_ = os.Remove(filepath.Join(stage, "manifest.json"))
		_ = os.Remove(filepath.Join(stage, "task"))
		_ = os.Remove(stage)
	}()
	if e = durablefile.WriteAtomicMode(filepath.Join(stage, "task"), binary, ".binary-*", 0700); e != nil {
		return e
	}
	if e = durablefile.WriteAtomic(filepath.Join(stage, "manifest.json"), encoded, ".manifest-*"); e != nil {
		return e
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	if b.cacheFault != nil {
		if e = b.cacheFault("publication-ready"); e != nil {
			return e
		}
	}
	if e = os.Rename(stage, dir); e != nil {
		return e
	}
	return durablefile.SyncDirectory(cache)
}
