package couchschedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/osfs"

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
		if privateCacheName.MatchString(name) {
			snapshot, e := captureCacheDir(cache, filepath.Join(cache, name))
			if e != nil {
				return nil, e
			}
			entries = append(entries, cacheEntry{dir: filepath.Join(cache, name), bytes: snapshot.bytes})
			continue
		}
		if !cacheKey(name) {
			return nil, fmt.Errorf("unknown cache entry: %s", name)
		}
		dir := filepath.Join(cache, name)
		// Content validity is checked before reuse. Damaged derived cache data with
		// only owned names remains disposable/rebuildable under the same key leases.
		snapshot, e := captureCacheDir(cache, dir)
		if e != nil {
			return nil, e
		}
		st, e := os.Lstat(dir)
		if e != nil {
			return nil, e
		}
		entries = append(entries, cacheEntry{key: name, dir: dir, bytes: snapshot.bytes, used: st.ModTime().UnixNano()})
	}
	return entries, nil
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
	if e = b.recoverCacheLocked(ctx, cache); e != nil {
		return e
	}
	entries, e := inspectCache(ctx, cache)
	if e != nil {
		return e
	}
	if binary == nil {
		raw, err := osfs.ReadRegularMetadata(ctx, filepath.Join(cache, m.Key, "manifest.json"), 2<<20)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(raw, &m); err != nil {
			return err
		}
	}
	now := time.Now()
	if b.cacheNow != nil {
		now = b.cacheNow()
	}
	// Access time is directory metadata; hits never rewrite manifest bytes.
	if binary != nil {
		for _, entry := range entries {
			if entry.key == m.Key {
				raw, e := osfs.ReadRegularMetadata(ctx, filepath.Join(entry.dir, "manifest.json"), 2<<20)
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if e != nil {
					continue
				}
				var old BuildManifest
				if e = json.Unmarshal(raw, &old); e == nil {
					m = mergeBuildMetadata(m, old)
				}
			}
		}
	}
	encoded, e := encodeBuildManifest(m)
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
	ownExists := false
	for _, entry := range entries {
		total += entry.bytes
		if entry.key == m.Key {
			ownExists = true
			own = entry.bytes
		}
	}
	required := own
	if binary != nil {
		required = int64(len(binary) + len(encoded))
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
		if entry.key == "" || entry.key == m.Key {
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
		e = b.evictCacheEntry(ctx, cache, entry)
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
		if e = os.Chtimes(dir, now, now); e != nil {
			return e
		}
		return b.cacheStep("manifest-update-ready")
	}
	if ownExists {
		if e = b.evictCacheEntry(ctx, cache, cacheEntry{key: m.Key, dir: dir}); e != nil {
			return e
		}
	}
	stage, e := os.MkdirTemp(cache, fmt.Sprintf(".publication-%d-", time.Now().UnixNano()))
	if e != nil {
		return e
	}
	defer func() {
		_ = os.Remove(filepath.Join(stage, "manifest.json"))
		_ = os.Remove(filepath.Join(stage, "task"))
		_ = os.Remove(stage)
	}()
	if e = b.cacheStep("publication-created"); e != nil {
		return e
	}
	if e = writeCachePayload(filepath.Join(stage, "task"), binary, 0700); e != nil {
		return e
	}
	if e = b.cacheStep("publication-task-written"); e != nil {
		return e
	}
	if e = writeCachePayload(filepath.Join(stage, "manifest.json"), encoded, 0600); e != nil {
		return e
	}
	if e = b.cacheStep("publication-manifest-written"); e != nil {
		return e
	}
	if e = durablefile.SyncDirectory(stage); e != nil {
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
	if e = b.cacheStep("publication-renamed"); e != nil {
		return e
	}
	if e = os.Chtimes(dir, now, now); e != nil {
		return e
	}
	return durablefile.SyncDirectory(cache)
}

// Only dedicated, never-referenced cache staging names are recoverable. New
// names retain their creation instant across rename; old names use mtime.
var privateCacheName = regexp.MustCompile(`^\.(publication|trash)-([0-9]+)(?:-([0-9]+))?$`)
var legacyCacheTemp = regexp.MustCompile(`^\.(binary|manifest)-[0-9]+$`)

const cacheResidueGrace = time.Minute

type cacheFileIdentity struct {
	device, inode  uint64
	mode           os.FileMode
	size, modified int64
}

func cacheFileID(st os.FileInfo) (cacheFileIdentity, error) {
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return cacheFileIdentity{}, errors.New("cache file identity unavailable")
	}
	return cacheFileIdentity{uint64(sys.Dev), uint64(sys.Ino), st.Mode(), st.Size(), st.ModTime().UnixNano()}, nil
}
func (i cacheFileIdentity) matches(other cacheFileIdentity) bool {
	return i.device == other.device && i.inode == other.inode && i.mode == other.mode && (i.mode.IsDir() || i.size == other.size && i.modified == other.modified)
}

type cacheDirSnapshot struct {
	identity cacheFileIdentity
	files    map[string]cacheFileIdentity
	bytes    int64
}

func captureCacheDir(cache, path string) (cacheDirSnapshot, error) {
	var snapshot cacheDirSnapshot
	if e := buildSafePath(cache, path); e != nil {
		return snapshot, e
	}
	st, e := os.Lstat(path)
	if e != nil {
		return snapshot, e
	}
	if !st.IsDir() {
		return snapshot, errors.New("unsafe cache staging directory")
	}
	snapshot.identity, e = cacheFileID(st)
	if e != nil {
		return snapshot, e
	}
	snapshot.files = map[string]cacheFileIdentity{}
	f, e := os.Open(path)
	if e != nil {
		return snapshot, e
	}
	defer f.Close()
	names, e := f.Readdirnames(9)
	if e != nil && !errors.Is(e, io.EOF) {
		return snapshot, e
	}
	if len(names) > 8 {
		return snapshot, errors.New("cache staging member limit exceeded")
	}
	for _, name := range names {
		if name != "task" && name != "manifest.json" && !legacyCacheTemp.MatchString(name) {
			return snapshot, errors.New("unknown cache staging member")
		}
		st, e := os.Lstat(filepath.Join(path, name))
		if e != nil {
			return snapshot, e
		}
		if !st.Mode().IsRegular() || st.Size() < 0 || st.Size() > DefaultCacheBytes {
			return snapshot, errors.New("unsafe cache staging payload")
		}
		id, e := cacheFileID(st)
		if e != nil {
			return snapshot, e
		}
		snapshot.files[name] = id
		snapshot.bytes += st.Size()
	}
	return snapshot, nil
}
func (b Builder) cacheStep(step string) error {
	if b.cacheFault != nil {
		return b.cacheFault(step)
	}
	return nil
}
func verifyCacheDirName(path string, want cacheFileIdentity) error {
	st, e := os.Lstat(path)
	if e != nil {
		return e
	}
	id, e := cacheFileID(st)
	if e != nil {
		return e
	}
	if !want.matches(id) {
		return errors.New("cache staging directory replaced")
	}
	return nil
}

// The opened directory and captured inode set fence every unlink. A replaced
// name or unknown child is retained, including a replacement after inspection.
func (b Builder) deleteCacheDir(ctx context.Context, cache, path string, want cacheDirSnapshot) error {
	targets := make([]string, 0, len(want.files))
	for name := range want.files {
		targets = append(targets, name)
	}
	sort.Strings(targets)
	return b.unlinkCacheSnapshot(ctx, cache, path, want, targets, true)
}
func (b Builder) unlinkCacheSnapshot(ctx context.Context, cache, path string, want cacheDirSnapshot, targets []string, removeDirectory bool) error {
	if e := b.cacheStep("cleanup-observed"); e != nil {
		return e
	}
	if e := verifyCacheDirName(path, want.identity); e != nil {
		return e
	}
	fd, e := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if e != nil {
		return e
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return e
	}
	id, e := cacheFileID(st)
	if e != nil || !want.identity.matches(id) {
		return errors.New("cache staging directory replaced")
	}
	ns, e := f.Readdirnames(9)
	if e != nil && !errors.Is(e, io.EOF) {
		return e
	}
	if len(ns) != len(want.files) {
		return errors.New("cache staging members changed")
	}
	for _, name := range ns {
		if _, ok := want.files[name]; !ok {
			return errors.New("cache staging members changed")
		}
	}
	for _, name := range targets {
		expected, ok := want.files[name]
		if !ok {
			return errors.New("unknown cache staging member")
		}
		if e = ctx.Err(); e != nil {
			return e
		}
		if e = verifyCacheDirName(path, want.identity); e != nil {
			return e
		}
		var current unix.Stat_t
		if e = unix.Fstatat(fd, name, &current, unix.AT_SYMLINK_NOFOLLOW); e != nil {
			return e
		}
		info, e := os.Lstat(filepath.Join(path, name))
		if e != nil {
			return e
		}
		actual, e := cacheFileID(info)
		if e != nil || !expected.matches(actual) || uint64(current.Ino) != expected.inode || uint64(current.Dev) != expected.device {
			return errors.New("cache staging payload replaced")
		}
		if e = unix.Unlinkat(fd, name, 0); e != nil {
			return e
		}
		if e = b.cacheStep("cleanup-" + name); e != nil {
			return e
		}
	}
	if e = f.Sync(); e != nil {
		return e
	}
	if !removeDirectory {
		return nil
	}
	if e = verifyCacheDirName(path, want.identity); e != nil {
		return e
	}
	if e = os.Remove(path); e != nil {
		return e
	}
	if e = b.cacheStep("cleanup-removed"); e != nil {
		return e
	}
	return durablefile.SyncDirectory(cache)
}
func (b Builder) evictCacheEntry(ctx context.Context, cache string, entry cacheEntry) error {
	snapshot, e := captureCacheDir(cache, entry.dir)
	if e != nil {
		return e
	}
	trash, e := os.MkdirTemp(cache, fmt.Sprintf(".trash-%d-", time.Now().UnixNano()))
	if e != nil {
		return e
	}
	if e = b.cacheStep("eviction-reserved"); e != nil {
		return e
	}
	if e = os.Remove(trash); e != nil {
		return e
	}
	if e = verifyCacheDirName(entry.dir, snapshot.identity); e != nil {
		return e
	}
	if e = os.Rename(entry.dir, trash); e != nil {
		return e
	}
	if e = durablefile.SyncDirectory(cache); e != nil {
		return e
	}
	if e = b.cacheStep("eviction-renamed"); e != nil {
		return e
	}
	return b.deleteCacheDir(ctx, cache, trash, snapshot)
}
func (b Builder) recoverCacheLocked(ctx context.Context, cache string) error {
	ns, e := cacheNames(cache)
	if e != nil {
		return e
	}
	now := time.Now()
	if b.cacheRecoveryNow != nil {
		now = b.cacheRecoveryNow()
	}
	for _, name := range ns {
		parts := privateCacheName.FindStringSubmatch(name)
		if parts == nil {
			if cacheKey(name) {
				if e = b.recoverLegacyCacheTemps(ctx, cache, name, now); e != nil {
					return e
				}
			}
			continue
		}
		if e = ctx.Err(); e != nil {
			return e
		}
		path := filepath.Join(cache, name)
		snapshot, e := captureCacheDir(cache, path)
		if e != nil {
			return e
		}
		st, e := os.Lstat(path)
		if e != nil {
			return e
		}
		created := st.ModTime()
		if parts[3] != "" {
			n, e := strconv.ParseInt(parts[2], 10, 64)
			if e != nil {
				return e
			}
			created = time.Unix(0, n)
		}
		if created.After(now) || now.Sub(created) < cacheResidueGrace {
			continue
		}
		if e = b.deleteCacheDir(ctx, cache, path, snapshot); e != nil {
			return e
		}
	}
	return nil
}
func (b Builder) recoverCache(ctx context.Context, cache string) error {
	unlock, e := buildLock(ctx, filepath.Join(cache, ".maintenance.lock"))
	if e != nil {
		return e
	}
	defer unlock()
	return b.recoverCacheLocked(ctx, cache)
}
func writeCachePayload(path string, raw []byte, perm os.FileMode) error {
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if e != nil {
		return e
	}
	defer f.Close()
	if _, e = f.Write(raw); e != nil {
		return e
	}
	return f.Sync()
}

// Earlier builds used atomic file replacement inside the key directory. Those
// private numeric temp names were never returned as artifacts; cleanup shares
// the publication lock and cannot alter the referenced task or manifest names.
func (b Builder) recoverLegacyCacheTemps(ctx context.Context, cache, key string, now time.Time) error {
	pub, e := cacheFileLease(filepath.Join(cache, key+".lock"), unix.LOCK_EX)
	if errors.Is(e, errCacheLive) {
		return nil
	}
	if e != nil {
		return e
	}
	defer pub.Close()
	dir := filepath.Join(cache, key)
	snapshot, e := captureCacheDir(cache, dir)
	if e != nil {
		return e
	}
	var targets []string
	for name, id := range snapshot.files {
		if legacyCacheTemp.MatchString(name) && !time.Unix(0, id.modified).After(now) && now.Sub(time.Unix(0, id.modified)) >= cacheResidueGrace {
			targets = append(targets, name)
		}
	}
	if len(targets) == 0 {
		return nil
	}
	sort.Strings(targets)
	return b.unlinkCacheSnapshot(ctx, cache, dir, snapshot, targets, false)
}
