package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/osfs"
)

// Store persists couch's registry, naming table and policy.
//
// The location is UNSCOPED -- the registry spans every worktree, which is the
// whole point of `couch list`, so a per-tree ScopedLaunchDataDir would mean
// spawning in /a and listing from /b read different files.
//
// It is a snapshot, not an append-only log: osfs.FS.WriteAtomic is temp+rename
// and cannot append. If append semantics are ever wanted the house idiom is
// sessionwatch.appendSessionLedger (sessionwatch/run.go:184-198).
//
// The directory is passed in, so a test points it at t.TempDir() and nothing
// depends on production configuration.
type Store struct {
	dir string
	fs  osfs.FS
}

func NewStore(dir string) Store { return Store{dir: dir} }

// Dir is where couch keeps its state. Exported so a spawned child can be told
// where to publish its description.
func (s Store) Dir() string { return s.dir }

func (s Store) registryPath() string { return filepath.Join(s.dir, "registry.json") }

// DescDir is where an agent writes its own one-line description.
func (s Store) DescDir() string { return filepath.Join(s.dir, "desc") }

type snapshot struct {
	Actors []ActorRecord        `json:"actors"`
	Names  map[string]NameEntry `json:"names"`
}

func (s Store) Save(reg Registry, names NamingTable) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("store dir: %w", err)
	}
	data, err := json.MarshalIndent(snapshot{Actors: reg.Records(), Names: names.All()}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}
	if err := s.fs.WriteAtomic(s.registryPath(), string(data)+"\n"); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}
	return nil
}

// Load returns empty state and no error when nothing has been written yet:
// a first run is not a failure.
func (s Store) Load() (Registry, NamingTable, error) {
	reg, names := NewRegistry(), NewNamingTable()

	raw, err := s.fs.ReadFile(s.registryPath())
	switch {
	case err == nil:
		var snap snapshot
		if err := json.Unmarshal([]byte(raw), &snap); err != nil {
			return reg, names, fmt.Errorf("decode snapshot: %w", err)
		}
		for _, a := range snap.Actors {
			// Insert, not Register: replay must reproduce what was persisted
			// exactly, including inert legacy fields awaiting migration.
			reg = reg.Insert(a)
		}
		for _, e := range snap.Names {
			if e.Name != "" {
				names = names.SetName(e.Tree, e.Name)
			}
			if e.Description != "" {
				names = names.SetDescription(e.Tree, e.Description)
			}
		}
	case errors.Is(err, fs.ErrNotExist):
		// A first run is not a failure.
	default:
		// Anything else -- permissions, IO -- must NOT read as an empty
		// registry, or the next Save silently overwrites a snapshot we simply
		// failed to read.
		return reg, names, fmt.Errorf("read snapshot: %w", err)
	}
	return reg, names, nil
}

// ReadDescription reads the one-line description an agent wrote for its tree.
func (s Store) ReadDescription(w Worktree) (string, error) {
	raw, err := s.fs.ReadFile(filepath.Join(s.DescDir(), sanitizeKey(w.Key())))
	if err != nil {
		return "", err
	}
	return trimTrailingNewline(raw), nil
}

// WriteDescription is what an agent calls to publish its own one-liner.
func (s Store) WriteDescription(w Worktree, desc string) error {
	if err := os.MkdirAll(s.DescDir(), 0o755); err != nil {
		return err
	}
	return s.fs.WriteAtomic(filepath.Join(s.DescDir(), sanitizeKey(w.Key())), desc+"\n")
}
