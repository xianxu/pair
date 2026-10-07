package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchsingleton"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// accountHome is deliberately independent of HOME and XDG. Explicit isolation
// bypasses account lookup, so subprocess fixtures cannot take production's lease.
func localAccountHome() (string, error) {
	account, err := user.LookupId(strconv.Itoa(os.Getuid()))
	if err != nil {
		return "", fmt.Errorf("resolve local Couch account: %w", err)
	}
	return account.HomeDir, nil
}

// canonicalFuture resolves existing ancestors without creating missing roots.
func canonicalFuture(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("Couch root must be absolute: %q", path)
	}
	path = filepath.Clean(path)
	physical, err := filepath.EvalSymlinks(path)
	if err == nil {
		return physical, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if info, statErr := os.Lstat(path); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("unresolved Couch root symlink %q: %w", path, err)
	}
	parent := filepath.Dir(path)
	if parent == path {
		return "", err
	}
	physical, err = canonicalFuture(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(physical, filepath.Base(path)), nil
}

func confined(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// isolatedChildRoots retains the physical destinations validated before any
// selection or child effects. Fallbacks obey the same rule as explicit roots.
type isolatedChildRoots struct{ home, data, temporary string }

func confinedDirectory(root, path, name string) (string, error) {
	physical, err := canonicalFuture(path)
	if err != nil {
		return "", fmt.Errorf("%s isolation root: %w", name, err)
	}
	if !confined(root, physical) {
		return "", fmt.Errorf("%s must remain inside COUCH_ISOLATED_ROOT: %q", name, physical)
	}
	if info, err := os.Stat(physical); err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("%s isolation root is not a directory: %q", name, physical)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return physical, nil
}

func resolveIsolatedChildRoots(root, requestedHome string) (isolatedChildRoots, error) {
	var paths isolatedChildRoots
	home, err := canonicalFuture(requestedHome)
	if err != nil || !confined(root, home) {
		home = filepath.Join(root, "home")
	}
	for _, field := range []struct {
		name, path string
		result     *string
	}{
		{"HOME", home, &paths.home},
		{"XDG_DATA_HOME", filepath.Join(root, "data"), &paths.data},
		{"TMPDIR", filepath.Join(root, "tmp"), &paths.temporary},
	} {
		*field.result, err = confinedDirectory(root, field.path, field.name)
		if err != nil {
			return paths, err
		}
	}
	return paths, nil
}

func (r OSRuntime) singletonManager() (couchsingleton.Manager, couchsingleton.Request, string, error) {
	var m couchsingleton.Manager
	var q couchsingleton.Request
	if _, err := capturePath(r.Getenv); err != nil {
		return m, q, "", err
	}
	isolated := r.Getenv("COUCH_ISOLATED_ROOT")
	var home string
	var err error
	if isolated != "" {
		isolated, err = canonicalFuture(isolated)
		if err != nil {
			return m, q, "", err
		}
		childRoots, e := resolveIsolatedChildRoots(isolated, r.Getenv("HOME"))
		if e != nil {
			return m, q, "", e
		}
		home = childRoots.home
		m.AuthorityDir = filepath.Join(isolated, "singleton")
		m.IsolationRoot = isolated
		m.Defaults = couchsingleton.Roots{StoreDir: filepath.Join(childRoots.data, "pair", "couch"), PairDataDir: filepath.Join(childRoots.data, "pair"), IdentityDir: filepath.Join(isolated, "pair-host")}
		for _, field := range []*string{&m.AuthorityDir, &m.Defaults.StoreDir, &m.Defaults.PairDataDir, &m.Defaults.IdentityDir} {
			*field, e = confinedDirectory(isolated, *field, "Couch")
			if e != nil {
				return m, q, "", e
			}
		}
	} else {
		lookup := r.accountHome
		if lookup == nil {
			lookup = localAccountHome
		}
		home, err = lookup()
		if err != nil {
			return m, q, "", err
		}
		home, err = canonicalFuture(home)
		if err != nil {
			return m, q, "", err
		}
		m.AuthorityDir = filepath.Join(home, ".local", "share", "pair-host", "singleton")
		data := launcher.ResolveDataDir(home, "")
		m.Defaults = couchsingleton.Roots{StoreDir: filepath.Join(data, "couch"), PairDataDir: data, IdentityDir: filepath.Join(home, ".local", "share", "pair-host")}
	}
	q.Roots.StoreDir = r.Getenv("COUCH_STORE_DIR")
	q.Roots.IdentityDir = r.Getenv("COUCH_IDENTITY_DIR")
	q.Roots.PairDataDir = r.Getenv("COUCH_PAIR_DATA_DIR")
	if q.Roots.PairDataDir == "" {
		q.Roots.PairDataDir = r.Getenv("PAIR_DATA_DIR")
		// Hosted Pair exports a repo-scoped data directory. Convert only the exact
		// documented scope suffix; arbitrary ancestors are not evidence of a root.
		scope := r.Getenv("COUCH_THREAD_SCOPE")
		if scope != "" && filepath.Base(scope) == scope && q.Roots.PairDataDir != "" && filepath.Base(q.Roots.PairDataDir) == scope && filepath.Base(filepath.Dir(q.Roots.PairDataDir)) == "repos" {
			q.Roots.PairDataDir = filepath.Dir(filepath.Dir(q.Roots.PairDataDir))
		}
	}
	if q.Roots.PairDataDir == "" && r.Getenv("XDG_DATA_HOME") != "" {
		q.Roots.PairDataDir = launcher.ResolveDataDir(home, r.Getenv("XDG_DATA_HOME"))
	}
	for _, p := range []*string{&q.Roots.StoreDir, &q.Roots.PairDataDir, &q.Roots.IdentityDir} {
		if *p == "" {
			continue
		}
		*p, err = canonicalFuture(*p)
		if err != nil {
			return m, q, "", err
		}
		if isolated != "" && !confined(isolated, *p) {
			return m, q, "", fmt.Errorf("Couch root %q escapes COUCH_ISOLATED_ROOT %q; clear the inherited override", *p, isolated)
		}
	}
	// A first launch with an explicit Pair root keeps the historical default
	// store beside that root. The account-wide authority remains unchanged.
	if q.Roots.StoreDir == "" && q.Roots.PairDataDir != "" {
		m.Defaults.StoreDir = filepath.Join(q.Roots.PairDataDir, "couch")
	}
	if isolated != "" {
		for _, key := range []string{"COUCH_TRACE", "COUCH_INPUT_TRACE", "COUCH_MOUSE_TRACE"} {
			if path := r.Getenv(key); path != "" {
				canonical, e := canonicalFuture(path)
				if e != nil || !confined(isolated, canonical) {
					return m, q, "", fmt.Errorf("%s must remain inside COUCH_ISOLATED_ROOT", key)
				}
			}
		}
	}
	return m, q, isolated, nil
}

type leaseResource struct{ io.Closer }

// runtimeOwnership has two states: a live acquired resource, or released (nil).
// Copies of a prepared runtime share this handle, so Close revokes all copies.
type runtimeOwnership struct{ active atomic.Pointer[leaseResource] }

func (o *runtimeOwnership) Close() error {
	if resource := o.active.Swap(nil); resource != nil {
		return resource.Close()
	}
	return nil
}
func (o *runtimeOwnership) held() bool { return o != nil && o.active.Load() != nil }

type emptyLease struct{}

func (emptyLease) Close() error { return nil }

func (r OSRuntime) prepareSingleton(owner bool) (OSRuntime, io.Closer, error) {
	if r.selection != nil {
		if owner {
			return r, nil, errors.New("cannot reuse a resolved Couch runtime for a new owner invocation")
		}
		return r, emptyLease{}, nil
	}
	m, q, isolated, err := r.singletonManager()
	if err != nil {
		return r, nil, err
	}
	if isolated != "" {
		r.isolatedPaths, err = resolveIsolatedChildRoots(isolated, r.Getenv("HOME"))
		if err != nil {
			return r, nil, err
		}
	}
	var selection couchsingleton.Selection
	var lease io.Closer = emptyLease{}
	if owner {
		selection, lease, err = m.Acquire(q)
	} else {
		selection, err = m.Read(q)
	}
	if err != nil {
		return r, nil, err
	}
	r.selection = &selection
	if owner {
		r.ownership = &runtimeOwnership{}
		r.ownership.active.Store(&leaseResource{lease})
		lease = r.ownership
	}
	r.isolatedRoot = isolated
	if isolated != "" && owner {
		for _, path := range []string{r.isolatedPaths.home, r.isolatedPaths.temporary} {
			if err := os.MkdirAll(path, 0700); err != nil {
				lease.Close()
				return r, nil, err
			}
		}
	}
	return r, lease, nil
}

func prepareRuntime(rt Runtime, owner bool) (Runtime, io.Closer, error) {
	if production, ok := rt.(OSRuntime); ok {
		return production.prepareSingleton(owner)
	}
	return rt, emptyLease{}, nil
}

func runtimePairDataDir(rt Runtime) string {
	if root := rt.Getenv("COUCH_PAIR_DATA_DIR"); root != "" {
		return root
	}
	return launcher.ResolveDataDir(rt.Getenv("HOME"), rt.Getenv("XDG_DATA_HOME"))
}

// configuredRunner carries runtime root provenance through ordinary creation
// and blocked launch alike. Actor-specific env still overrides inherited paths.
type configuredRunner struct {
	underlying  couchcore.Runner
	environment []string
}

func (r configuredRunner) Start(dir string, argv, env []string) (couchcore.Handle, error) {
	return r.underlying.Start(dir, argv, r.childEnv(env))
}
func (r configuredRunner) StartBlocked(ctx context.Context, dir string, argv, env []string, timeout time.Duration) (couchcore.BlockedHandle, error) {
	return r.underlying.StartBlocked(ctx, dir, argv, r.childEnv(env), timeout)
}
func (r configuredRunner) childEnv(env []string) []string {
	// Clear inherited per-session destinations before supplying the new actor's
	// explicit paths. Pair computes its own paths after selecting that actor.
	var clean []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "PAIR_") && strings.HasSuffix(key, "_PATH") {
			clean = append(clean, key+"=")
		}
	}
	clean = append(clean, env...)
	return append(clean, r.environment...)
}
func (r OSRuntime) runtimeRunner(runner couchcore.Runner) couchcore.Runner {
	if r.selection == nil {
		return runner
	}
	roots := r.selection.Roots
	// Pair's PAIR_DATA_DIR is already repository-scoped. Clear the inherited
	// actor destination and pass the selected global root separately so the
	// launcher computes this actor's scope once, from its own checkout.
	env := []string{"COUCH_CAPTURE_DIR=", "PAIR_DATA_DIR=", "COUCH_PAIR_DATA_DIR=" + roots.PairDataDir, "COUCH_STORE_DIR=" + roots.StoreDir, "COUCH_IDENTITY_DIR=" + roots.IdentityDir, "COUCH_ISOLATED_ROOT=" + r.isolatedRoot}
	if r.isolatedRoot != "" {
		env = append(env, "HOME="+r.isolatedPaths.home, "XDG_DATA_HOME="+r.isolatedPaths.data, "TMPDIR="+r.isolatedPaths.temporary)
	}
	return configuredRunner{underlying: runner, environment: env}
}
