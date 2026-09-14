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

func (r StoreRegistry) validate() error {
	if r.Version != 1 || r.Stores == nil {
		return errors.New("invalid store registry schema")
	}
	for i, path := range r.Stores {
		canonical, err := canonicalStore(path)
		if err != nil {
			return fmt.Errorf("registered store %q unavailable: %w", path, err)
		}
		if canonical != path {
			return fmt.Errorf("registered store %q is not canonical", path)
		}
		if i > 0 && r.Stores[i-1] >= path {
			return errors.New("registry stores must be unique and sorted")
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
	return r, r.validate()
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
// all membership publishers must register before adding new references.
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
