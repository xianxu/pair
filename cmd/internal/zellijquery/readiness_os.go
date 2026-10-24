package zellijquery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/panebirth"
)

const MaxReadySessions = 128

type readinessStore struct {
	namespace, transport string
	paths                artifactpath.PairCachePaths
}

// ChildEnvironment uses the same last-value-wins snapshot for resolution and
// execution, including explicit environments supplied by disposable fixtures.
func ChildEnvironment(env []string) []string {
	if env == nil {
		snapshot := os.Environ()
		if os.Getenv("TMPDIR") == "" {
			snapshot = append(snapshot, "TMPDIR="+os.TempDir())
		}
		return snapshot
	}
	return append([]string(nil), env...)
}

func newReadinessStore(env []string) (readinessStore, error) {
	values := map[string]string{}
	for _, entry := range ChildEnvironment(env) {
		if k, v, ok := strings.Cut(entry, "="); ok {
			values[k] = v
		}
	}
	temp := values["TMPDIR"]
	transport, err := ResolveZellijSocketRoot(values["ZELLIJ_SOCKET_DIR"], temp, values["XDG_RUNTIME_DIR"], runtime.GOOS, os.Getuid())
	if err != nil {
		return readinessStore{}, err
	}
	namespace, err := canonicalNamespace(transport)
	if err != nil {
		return readinessStore{}, err
	}
	paths, err := artifactpath.ResolvePairCache(values["HOME"])
	return readinessStore{namespace: namespace, transport: transport, paths: paths}, err
}

// Canonicalize the existing ancestor without creating native directories. The
// transport spelling is separately preserved for Unix pathname length limits.
func canonicalNamespace(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	parent := filepath.Dir(path)
	if parent == path {
		return "", err
	}
	resolved, err = canonicalNamespace(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, filepath.Base(path)), nil
}

func ReadinessNamespace(env []string) (string, error) {
	s, err := newReadinessStore(env)
	return s.namespace, err
}

func (s readinessStore) lock(ctx context.Context) (io.Closer, error) {
	path, err := s.paths.ZellijNamespaceLease(s.namespace)
	if err != nil {
		return nil, err
	}
	return AcquireFileLease(ctx, path)
}

func boundedDirectory(path string, limit int) ([]os.DirEntry, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(limit + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > limit {
		return nil, fmt.Errorf("readiness candidate limit %d exceeded", limit)
	}
	return entries, nil
}

func (s readinessStore) marker(ctx context.Context, session string) (ReadyMarker, error) {
	path, err := s.paths.ZellijReadyMarker(s.namespace, session)
	if err != nil {
		return ReadyMarker{}, err
	}
	raw, err := osfs.ReadRegularMetadata(ctx, path, panebirth.MaxRecordBytes)
	if err != nil {
		return ReadyMarker{}, err
	}
	return DecodeReadyMarker(raw)
}

func (s readinessStore) qualify(ctx context.Context) error {
	entries, err := boundedDirectory(AppendNativeSocketPath(s.transport, "contract_version_1"), MaxReadySessions)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrReadinessUnknown, err)
	}
	markers := make(map[string]ReadyMarker, len(entries))
	candidates := make([]string, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrReadinessUnknown, err)
		}
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("%w: unexpected native endpoint %s", ErrReadinessUnknown, entry.Name())
		}
		marker, err := s.marker(ctx, entry.Name())
		if err != nil {
			return fmt.Errorf("%w: %s: %v", ErrReadinessUnknown, entry.Name(), err)
		}
		candidates = append(candidates, entry.Name())
		markers[entry.Name()] = marker
	}
	return PlanProbeAdmission(s.namespace, candidates, markers)
}

// AcquireProbe owns exclusion through the joined native invocation. A caller
// holding a creator's explicit permit must not recursively call this function.
func AcquireProbe(ctx context.Context, env []string) (io.Closer, error) {
	s, err := newReadinessStore(env)
	if err != nil {
		return nil, err
	}
	if permit, ok := ctx.Value(creatorPermitKey{}).(*creatorPermit); ok {
		if !permit.active.Load() || permit.namespace != s.namespace {
			return nil, fmt.Errorf("%w: expired or foreign creator preflight permit", ErrReadinessUnknown)
		}
		if err := s.qualify(ctx); err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return borrowedCreatorLease{}, nil
	}
	lease, err := s.lock(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.qualify(ctx); err != nil {
		return nil, errors.Join(err, lease.Close())
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, lease.Close())
	}
	return lease, nil
}

// CreatorGuard's kernel lease belongs to the live creator, not its daemon or
// interactive client. Owner death releases it; it is deliberately not a journal.
type CreatorGuard struct {
	mu       sync.Mutex
	store    readinessStore
	identity ReadyIdentity
	lease    io.Closer
	prepared bool
}

type creatorPermitKey struct{}
type creatorPermit struct {
	namespace string
	active    atomic.Bool
}
type borrowedCreatorLease struct{}

func (borrowedCreatorLease) Close() error { return nil }

// Preflight lends the already-held exclusion only to synchronous, joined reads
// before Prepare. The permit cannot be serialized or reused after the callback.
func (c *CreatorGuard) Preflight(ctx context.Context, fn func(context.Context) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lease == nil || c.prepared {
		return errors.New("creator preflight phase has ended")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	permit := &creatorPermit{namespace: c.store.namespace}
	permit.active.Store(true)
	defer permit.active.Store(false)
	return fn(context.WithValue(ctx, creatorPermitKey{}, permit))
}

func AcquireCreator(ctx context.Context, env []string, identity ReadyIdentity) (*CreatorGuard, error) {
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	s, err := newReadinessStore(env)
	if err != nil {
		return nil, err
	}
	if identity.Namespace != s.namespace {
		return nil, errors.New("creator namespace differs from effective child namespace")
	}
	lease, err := s.lock(ctx)
	if err != nil {
		return nil, err
	}
	return &CreatorGuard{store: s, identity: identity, lease: lease}, nil
}

// Prepare runs only after the existing handoff's absence/authority checks, just
// before native spawn. It never deletes another session's positive evidence.
func (c *CreatorGuard) Prepare(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lease == nil {
		return errors.New("creator exclusion has ended")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	dir, err := c.store.paths.ZellijReadyDirectory(c.store.namespace)
	if err != nil {
		return err
	}
	entries, err := boundedDirectory(dir, MaxReadySessions)
	if err != nil {
		return err
	}
	path, err := c.store.paths.ZellijReadyMarker(c.store.namespace, c.identity.Session)
	if err != nil {
		return err
	}
	_, statErr := os.Lstat(path)
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	if len(entries) >= MaxReadySessions && errors.Is(statErr, os.ErrNotExist) {
		return errors.New("ready marker capacity reached; use existing explicit retirement")
	}
	if statErr == nil {
		if _, err := c.store.marker(ctx, c.identity.Session); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	c.prepared = true
	return nil
}

func (c *CreatorGuard) PublishReady(ctx context.Context, observed ReadyIdentity) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lease == nil || !c.prepared {
		return errors.New("creator is not awaiting first-pane publication")
	}
	if !c.identity.Equal(observed) {
		return errors.New("pane readiness names another launch")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := c.store.paths.ZellijReadyMarker(c.store.namespace, c.identity.Session)
	if err != nil {
		return err
	}
	if old, err := c.store.marker(ctx, c.identity.Session); err == nil {
		if !old.Identity.Equal(observed) {
			return errors.New("ready marker changed generation before publication")
		}
		// Preserve the compatible existing record byte-for-byte.
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	} else {
		raw, err := EncodeReadyMarker(ReadyMarker{Identity: observed})
		if err != nil {
			return err
		}
		if err := (osfs.FS{}).WriteAtomic(path, string(raw)); err != nil {
			return err
		}
	}
	lease := c.lease
	c.lease = nil
	return lease.Close()
}

func (c *CreatorGuard) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lease == nil {
		return nil
	}
	lease := c.lease
	c.lease = nil
	return lease.Close()
}
