package historyarchive

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type Sink interface {
	Put(context.Context, string, io.Reader, int64) error
	Get(context.Context, string) (io.ReadCloser, error)
	List(context.Context, string) ([]string, error)
	Verify(context.Context, string, string, int64) error
}

// LocalSink serializes publication in each destination directory with a stable
// advisory lock. The next lock holder removes abandoned private staging files.
// Locks are never removed, so concurrent writers always contend on one inode.
type LocalSink struct {
	Root  string
	Fault func(stage string) error
}

func (s LocalSink) fault(stage string) error {
	if s.Fault != nil {
		return s.Fault(stage)
	}
	return nil
}
func (s LocalSink) Put(ctx context.Context, key string, r io.Reader, maxBytes int64) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if e := safeSinkKey(key); e != nil {
		return e
	}
	if maxBytes < 0 || maxBytes > MaxCiphertextBytes {
		return errors.New("invalid sink byte limit")
	}
	root, e := openSafeDir(s.Root, true)
	if e != nil {
		return e
	}
	defer root.Close()
	dir, name, e := safeParent(root, key, true)
	if e != nil {
		return e
	}
	defer dir.Close()
	lock, staging, e := lockSinkParent(ctx, dir)
	if e != nil {
		return e
	}
	defer lock.Close() // close releases flock only after staging cleanup
	defer staging.Close()
	var nonce [16]byte
	if _, e = rand.Read(nonce[:]); e != nil {
		return e
	}
	temp := ".partial-" + hex.EncodeToString(nonce[:])
	fd, e := unix.Openat(int(staging.Fd()), temp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if e != nil {
		return e
	}
	f := os.NewFile(uintptr(fd), temp)
	defer f.Close()
	defer unix.Unlinkat(int(staging.Fd()), temp, 0)
	if _, _, e = fileDigest(ctx, r, f, maxBytes); e != nil {
		return e
	}
	if e = s.fault("sync-file"); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	if e = s.fault("publish"); e != nil {
		return e
	}
	if e = unix.Linkat(int(staging.Fd()), temp, int(dir.Fd()), name, 0); e != nil {
		return e
	}
	uncertain := func(e error) error { return commitoutcome.Wrap(commitoutcome.Indeterminate, e) }
	if e = s.fault("after-publish"); e != nil {
		return uncertain(e)
	}
	if e = unix.Unlinkat(int(staging.Fd()), temp, 0); e != nil {
		return uncertain(e)
	}
	if e = staging.Sync(); e != nil {
		return uncertain(e)
	}
	if e = s.syncDirectories(ctx, root, key); e != nil {
		return uncertain(e)
	}
	return nil
}
func (s LocalSink) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if e := safeSinkKey(key); e != nil {
		return nil, e
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	root, e := openSafeDir(s.Root, false)
	if e != nil {
		return nil, e
	}
	defer root.Close()
	f, e := openRegularAt(root, key, unix.O_RDONLY)
	if e != nil {
		return nil, e
	}
	return &sinkReader{ctx: ctx, f: f}, nil
}

type sinkReader struct {
	ctx context.Context
	f   *os.File
}

func (r *sinkReader) Read(p []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	return r.f.Read(p)
}
func (r *sinkReader) Close() error { return r.f.Close() }

// syncDirectories repeats all publication edges, including the root's parent,
// even on adoption. A readable object is not itself durability evidence.
func (s LocalSink) syncDirectories(ctx context.Context, root *os.File, key string) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if e := s.fault("sync-directory"); e != nil {
		return e
	}
	dir, e := childDir(root, ".", false)
	if e != nil {
		return e
	}
	defer func() { dir.Close() }()
	if e = dir.Sync(); e != nil {
		return e
	}
	for _, part := range strings.Split(path.Dir(key), "/") {
		if part == "." {
			continue
		}
		next, e := childDir(dir, part, false)
		if e != nil {
			return e
		}
		dir.Close()
		dir = next
		if e = ctx.Err(); e != nil {
			return e
		}
		if e = dir.Sync(); e != nil {
			return e
		}
	}
	// Sync every configured-root ancestor so retries also repair a failed initial
	// directory creation, without relying on remembered creation flags.
	absolute, e := filepath.Abs(s.Root)
	if e != nil {
		return e
	}
	for parent := filepath.Dir(absolute); ; parent = filepath.Dir(parent) {
		f, e := openSafeDir(parent, false)
		if e != nil {
			return e
		}
		e = f.Sync()
		f.Close()
		if e != nil {
			return e
		}
		if parent == "/" {
			break
		}
	}
	return nil
}
func (s LocalSink) Verify(ctx context.Context, key, wantSHA256 string, wantBytes int64) error {
	if e := safeSinkKey(key); e != nil {
		return e
	}
	if wantBytes < 0 || wantBytes > MaxCiphertextBytes {
		return errors.New("invalid verification byte count")
	}
	root, e := openSafeDir(s.Root, false)
	if e != nil {
		return e
	}
	defer root.Close()
	f, e := openRegularAt(root, key, unix.O_RDONLY)
	if e != nil {
		return e
	}
	defer f.Close()
	before, e := f.Stat()
	if e != nil {
		return e
	}
	if before.Size() != wantBytes {
		return errors.New("sink object size mismatch")
	}
	n, digest, e := fileDigest(ctx, f, nil, wantBytes)
	if e != nil {
		return e
	}
	after, e := f.Stat()
	if e != nil {
		return e
	}
	if !sameSource(before, after) || n != wantBytes || digest != wantSHA256 {
		return errors.New("sink object checksum/size mismatch")
	}
	if e = s.fault("verify"); e != nil {
		return e
	}
	if e = s.fault("sync-file"); e != nil {
		return commitoutcome.Wrap(commitoutcome.Indeterminate, e)
	}
	if e = f.Sync(); e != nil {
		return commitoutcome.Wrap(commitoutcome.Indeterminate, e)
	}
	if e = s.syncDirectories(ctx, root, key); e != nil {
		return commitoutcome.Wrap(commitoutcome.Indeterminate, e)
	}
	return nil
}
func (s LocalSink) List(ctx context.Context, prefix string) ([]string, error) {
	if prefix != "" {
		if e := safeRelative(strings.TrimSuffix(prefix, "/")); e != nil {
			return nil, e
		}
	}
	root, e := openSafeDir(s.Root, false)
	if e != nil {
		return nil, e
	}
	defer root.Close()
	var out []string
	count := 0
	var walk func(*os.File, string) error
	walk = func(dir *os.File, base string) error {
		for {
			if e := ctx.Err(); e != nil {
				return e
			}
			entries, e := dir.ReadDir(128)
			for _, entry := range entries {
				if reservedSinkName(entry.Name()) {
					continue
				}
				count++
				if count > MaxFiles {
					return errors.New("sink listing exceeds 10000 entries; use a smaller sink root")
				}
				key := path.Join(base, entry.Name())
				if entry.Type()&os.ModeSymlink != 0 {
					return fmt.Errorf("sink contains symlink %q", key)
				}
				if entry.IsDir() {
					next, e := childDir(dir, entry.Name(), false)
					if e != nil {
						return e
					}
					e = walk(next, key)
					next.Close()
					if e != nil {
						return e
					}
					continue
				}
				f, e := openRegularAt(dir, entry.Name(), unix.O_RDONLY)
				if e != nil {
					return e
				}
				f.Close()
				if !reservedSinkName(entry.Name()) && strings.HasPrefix(key, prefix) {
					out = append(out, key)
				}
			}
			if errors.Is(e, io.EOF) {
				return nil
			}
			if e != nil {
				return e
			}
		}
	}
	if e = walk(root, ""); e != nil {
		return nil, e
	}
	sort.Strings(out)
	return out, nil
}

func reservedSinkName(name string) bool {
	return name == ".archive-put.lock" || name == ".archive-staging" || strings.HasPrefix(name, ".partial-")
}
func safeSinkKey(key string) error {
	if e := safeRelative(key); e != nil {
		return e
	}
	for _, part := range strings.Split(key, "/") {
		if reservedSinkName(part) {
			return errors.New("reserved sink internal path")
		}
	}
	return nil
}

// The stable lock and partial namespace are private to this adapter. A killed
// writer releases flock automatically; cleanup therefore cannot remove a live
// writer's staging file. Different destination directories remain independent.
func lockSinkParent(ctx context.Context, dir *os.File) (*os.File, *os.File, error) {
	lock, e := openRegularAt(dir, ".archive-put.lock", unix.O_RDWR|unix.O_CREAT)
	if e != nil {
		return nil, nil, e
	}
	var scan *os.File
	fail := func(e error) (*os.File, *os.File, error) {
		if scan != nil {
			scan.Close()
		}
		lock.Close()
		return nil, nil, e
	}
	if e = unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); e != nil {
		return fail(fmt.Errorf("sink directory busy; retry publication: %w", e))
	}
	scan, e = childDir(dir, ".archive-staging", true)
	if e != nil {
		return fail(e)
	}
	count := 0
	for {
		if e = ctx.Err(); e != nil {
			return fail(e)
		}
		entries, readErr := scan.ReadDir(128)
		for _, entry := range entries {
			count++
			if count > MaxFiles {
				return fail(errors.New("sink staging cleanup exceeds 10000 entries"))
			}
			if !strings.HasPrefix(entry.Name(), ".partial-") {
				continue
			}
			f, e := openRegularAt(scan, entry.Name(), unix.O_RDONLY)
			if e != nil {
				return fail(e)
			}
			f.Close()
			if e = unix.Unlinkat(int(scan.Fd()), entry.Name(), 0); e != nil {
				return fail(e)
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return fail(readErr)
		}
	}
	if e = scan.Sync(); e != nil {
		return fail(e)
	}
	if e = dir.Sync(); e != nil {
		return fail(e)
	}
	return lock, scan, nil
}
