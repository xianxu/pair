package storagegc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
)

// StoreRegistry is the explicit inventory of Couch namespaces whose references
// protect this Pair root. MigrationComplete acknowledges the legacy inventory;
// subsequent managed stores register under coordination before publishing refs.
type StoreRegistry struct {
	Version           int      `json:"version"`
	Stores            []string `json:"stores"`
	MigrationComplete bool     `json:"migration_complete"`
}

func (c *Coordinator) registryPath() string {
	return filepath.Join(c.Root, ".retention", "stores.json")
}

func canonicalStore(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("store path must be absolute")
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if err := checkDirectory(canonical, false); err != nil {
		return "", err
	}
	// ReadDir verifies that references can be enumerated, not only that the
	// directory exists. This is metadata only, never a payload scan.
	dir, err := os.Open(canonical)
	if err != nil {
		return "", err
	}
	_, readErr := dir.Readdirnames(1)
	closeErr := dir.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return "", readErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return canonical, nil
}

// ValidateStructure checks persisted inventory without assuming all namespaces
// are mounted. Availability is required by collection, not registration.
func (r StoreRegistry) ValidateStructure() error {
	if r.Version != 1 || r.Stores == nil {
		return errors.New("invalid store registry schema")
	}
	for i, path := range r.Stores {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return errors.New("registry store path must be clean and absolute")
		}
		if i > 0 && r.Stores[i-1] >= path {
			return errors.New("registry stores must be unique and sorted")
		}
	}
	return nil
}

func (r StoreRegistry) validate() error {
	if err := r.ValidateStructure(); err != nil {
		return err
	}
	for _, path := range r.Stores {
		canonical, err := canonicalStore(path)
		if err != nil {
			return fmt.Errorf("registered store %q unavailable: %w", path, err)
		}
		if canonical != path {
			return fmt.Errorf("registered store %q is not canonical", path)
		}
	}
	return nil
}

func (c *Coordinator) readRegistryFile() (StoreRegistry, error) {
	var r StoreRegistry
	if err := checkDirectory(filepath.Join(c.Root, ".retention"), false); err != nil {
		return r, err
	}
	err := readStateJSON(c.registryPath(), &r)
	return r, err
}

// ReadRegistry is read-only and fails closed if any registered namespace is
// missing, aliased or unreadable. A missing registry is not an empty inventory.
func (c *Coordinator) ReadRegistry() (StoreRegistry, error) {
	r, err := c.readRegistryFile()
	if err != nil {
		return r, err
	}
	return r, r.validate()
}

func (l *Locked) loadRegistry() (StoreRegistry, error) {
	r, err := l.coordinator.readRegistryFile()
	// Only a missing metadata file initializes an inventory. Missing STORE paths
	// from validation must never turn an existing inventory into an empty one.
	if errors.Is(err, os.ErrNotExist) {
		return StoreRegistry{Version: 1, Stores: []string{}}, nil
	}
	if err != nil {
		return r, err
	}
	return r, r.ValidateStructure()
}

// RegisterStore resolves user-configured aliases once, stores the canonical
// namespace and deduplicates registrations. Existing completed migration stays
// complete because this registration precedes the new store's references.
func (c *Coordinator) RegisterStore(ctx context.Context, path string) error {
	return c.WithLock(ctx, func(l *Locked) error { return l.RegisterStore(path) })
}

// RegisterStore publishes inventory within an existing root-lock scope, so a
// namespace adapter can publish its references without an unregister gap.
func (l *Locked) RegisterStore(path string) error {
	if l == nil || !l.active {
		return errors.New("expired retention lock")
	}
	canonical, err := canonicalStore(path)
	if err != nil {
		return err
	}
	r, err := l.loadRegistry()
	if err != nil {
		return err
	}
	if slices.Contains(r.Stores, canonical) {
		return nil
	}
	r.Stores = append(r.Stores, canonical)
	slices.Sort(r.Stores)
	return l.atomicJSON(l.coordinator.registryPath(), r)
}

// CompleteMigration acknowledges exactly the currently registered inventory.
// Inventory changes between preview and acknowledgment require a fresh review.
func (c *Coordinator) CompleteMigration(ctx context.Context, expectedPaths []string) error {
	return c.WithLock(ctx, func(l *Locked) error {
		r, err := l.loadRegistry()
		if err != nil {
			return err
		}
		if err := r.validate(); err != nil {
			return err
		}
		expected := make([]string, 0, len(expectedPaths))
		for _, path := range expectedPaths {
			canonical, err := canonicalStore(path)
			if err != nil {
				return err
			}
			if slices.Contains(expected, canonical) {
				return errors.New("duplicate expected store")
			}
			expected = append(expected, canonical)
		}
		slices.Sort(expected)
		if !slices.Equal(expected, r.Stores) {
			return errors.New("registered inventory differs from acknowledged stores")
		}
		r.MigrationComplete = true
		return l.atomicJSON(c.registryPath(), r)
	})
}

// UnregisterStore calls the namespace owner's empty proof under coordination.
// The proof must take its store lock in that order and cover all references;
// all membership publishers must register before adding new references. Unlike
// ordinary registration, removal conservatively requires full inventory
// availability; use explicit abandonment for a permanently missing store.
func (c *Coordinator) UnregisterStore(ctx context.Context, path string, empty func(string) (bool, error)) error {
	if empty == nil {
		return errors.New("unregister requires explicit empty-store proof")
	}
	return c.WithLock(ctx, func(l *Locked) error {
		r, err := c.ReadRegistry()
		if err != nil {
			return err
		}
		canonical, err := canonicalStore(path)
		if err != nil {
			return err
		}
		index := slices.Index(r.Stores, canonical)
		if index < 0 {
			return errors.New("store is not registered")
		}
		proven, err := empty(canonical)
		if err != nil {
			return err
		}
		if !proven {
			return errors.New("store still has references")
		}
		r.Stores = slices.Delete(r.Stores, index, index+1)
		return l.atomicJSON(c.registryPath(), r)
	})
}

// ForgetMissingStore explicitly abandons an unavailable namespace. It makes no
// empty-store claim: collection stays disabled until the operator acknowledges
// the remaining complete inventory in a separate operation.
func (c *Coordinator) ForgetMissingStore(ctx context.Context, path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("store path must be clean and absolute")
	}
	return c.WithLock(ctx, func(l *Locked) error {
		r, err := l.loadRegistry()
		if err != nil {
			return err
		}
		index := slices.Index(r.Stores, path)
		if index < 0 {
			return errors.New("store is not registered")
		}
		if err := requireMissingStore(path); err != nil {
			return err
		}
		r.Stores = slices.Delete(r.Stores, index, index+1)
		r.MigrationComplete = false
		return l.atomicJSON(c.registryPath(), r)
	})
}

// Inspect each existing component so a dangling/parent symlink cannot turn an
// alias into proof that the registered physical directory has disappeared.
func requireMissingStore(path string) error {
	parent := filepath.Dir(path)
	if parent != path {
		if err := requireDirectoryOrMissing(parent); err != nil {
			return err
		}
	}
	_, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("registered store still exists; restore it or use empty-store removal")
}

func requireDirectoryOrMissing(path string) error {
	parent := filepath.Dir(path)
	if parent != path {
		if err := requireDirectoryOrMissing(parent); err != nil {
			return err
		}
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("unsafe registered store ancestor %q", path)
	}
	return nil
}

// RegistryEntry is an inspection result, never authority to collect references.
type RegistryEntry struct {
	Path        string
	Unavailable string
}

// InspectRegistry reports the structurally valid inventory even during an
// outage. GC must continue to use ReadRegistry's complete availability check.
func (c *Coordinator) InspectRegistry() ([]RegistryEntry, error) {
	r, err := c.readRegistryFile()
	if err != nil {
		return nil, err
	}
	if err := r.ValidateStructure(); err != nil {
		return nil, err
	}
	entries := make([]RegistryEntry, 0, len(r.Stores))
	for _, path := range r.Stores {
		entry := RegistryEntry{Path: path}
		canonical, err := canonicalStore(path)
		if err != nil {
			entry.Unavailable = err.Error()
		} else if canonical != path {
			entry.Unavailable = "registered path is no longer canonical"
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
