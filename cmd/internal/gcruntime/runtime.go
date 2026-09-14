package gcruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

type Service struct {
	Collector         *storagegc.Collector
	DiagnosticOptions diagnosticlog.Options
}
type Report struct {
	Storage                  storagegc.CollectionReport `json:"storage"`
	Diagnostics              []diagnosticlog.Segment    `json:"diagnostics"`
	DiagnosticCollectedBytes int64                      `json:"diagnostic_collected_bytes"`
}

func New(root string) (*Service, error) {
	c, err := storagegc.NewCoordinator(root)
	if err != nil {
		return nil, err
	}
	collector := &storagegc.Collector{Coordinator: c, Agents: launcher.AgentInventory(), References: CouchReferences{c}, CleanupOwner: cleanupOwner}
	collector.LegacyRoot = legacyEvidence(c, diagnosticlog.OSInspection{})
	opts := diagnosticlog.EnvironmentOptions(func(key string) string {
		if key == "PAIR_DATA_DIR" {
			return c.Root
		}
		return ""
	})
	return &Service{Collector: collector, DiagnosticOptions: opts}, nil
}

func legacyEvidence(c *storagegc.Coordinator, inspector diagnosticlog.Inspection) func(context.Context, []storagegc.ProcessIdentity) (storagegc.Liveness, error) {
	return func(ctx context.Context, managed []storagegc.ProcessIdentity) (storagegc.Liveness, error) {
		if err := ctx.Err(); err != nil {
			return storagegc.ProcessUnknown, err
		}
		entries, err := c.ReadRuntimes()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return storagegc.ProcessUnknown, err
		}
		for _, r := range entries {
			managed = append(managed, r.Process)
		}
		runtimes, err := inspector.Runtimes()
		if err != nil {
			return storagegc.ProcessUnknown, err
		}
		for _, r := range runtimes {
			if r.Process.PID == os.Getpid() || r.Protocol {
				continue
			}
			if r.Process.Root != "" && r.Process.Root != c.Root {
				continue
			}
			known := false
			for _, p := range managed {
				if p.PID == r.Process.PID && p.Birth == r.Process.Birth && p.Birth != "" {
					known = true
					break
				}
			}
			if !known {
				return storagegc.ProcessUnknown, nil
			}
		}
		return storagegc.ProcessDead, nil
	}
}
func cleanupOwner(held *storagegc.Locked, owner artifactpath.StorageOwner) error {
	if err := launcher.RemoveOwnerSessionBindings(held, owner); err != nil {
		return err
	}
	p, err := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
	if err != nil {
		return err
	}
	path := p.SessionInventoryCatalog()
	st, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return errors.New("unsafe derived catalog")
	}
	return (sessioninventory.CatalogStore{Runtime: sessioninventory.CatalogOSRuntime{}}).Invalidate(path)
}
func (s *Service) registry() ([]diagnosticlog.RegistryEntry, error) {
	var entries []diagnosticlog.RegistryEntry
	for offset := 0; offset < 100000; offset += 100 {
		page, err := diagnosticlog.EnumerateRoot(s.Collector.Coordinator.Root, offset, 100)
		if err != nil {
			return nil, err
		}
		entries = append(entries, page...)
		if len(page) < 100 {
			return entries, nil
		}
	}
	return nil, errors.New("diagnostic registry exceeds inventory budget")
}
func (s *Service) prepare() ([]diagnosticlog.RegistryEntry, error) {
	entries, err := s.registry()
	if err != nil {
		return nil, err
	}
	s.Collector.ExcludedPaths = nil
	for _, e := range entries {
		s.Collector.ExcludedPaths = append(s.Collector.ExcludedPaths, e.Directory, e.Lock)
	}
	return entries, nil
}
func diagnosticPaths(r *storagegc.CollectionReport, entries []diagnosticlog.RegistryEntry) []string {
	paths := map[string]bool{}
	for _, e := range entries {
		paths[e.Path] = true
	}
	kept := r.Items[:0]
	for _, item := range r.Items {
		if item.Bucket == artifactpath.DebugRetention {
			for _, m := range item.Members {
				paths[m.Path] = true
			}
			continue
		}
		kept = append(kept, item)
	}
	r.Items = kept
	out := make([]string, 0, len(paths))
	for p := range paths {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
func (s *Service) Preview(ctx context.Context) (r Report, err error) {
	entries, err := s.prepare()
	if err != nil {
		return r, err
	}
	r.Storage, err = s.Collector.Preview(ctx)
	if err != nil {
		return r, err
	}
	for _, path := range diagnosticPaths(&r.Storage, entries) {
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
		rows, e := s.diagnosticPages(ctx, path, false)
		if e != nil {
			r.Diagnostics = append(r.Diagnostics, diagnosticlog.Segment{Path: path, Reason: e.Error()})
			continue
		}
		r.Diagnostics = append(r.Diagnostics, rows...)
	}
	return r, nil
}
func (s *Service) Apply(ctx context.Context, limit int) (r Report, err error) {
	entries, err := s.prepare()
	if err != nil {
		return r, err
	}
	r.Storage, err = s.Collector.Apply(ctx, limit)
	if err != nil {
		return r, err
	}
	paths := diagnosticPaths(&r.Storage, entries)
	if !r.Storage.MigrationComplete {
		return r, nil
	}
	for i, path := range paths {
		if i >= limit {
			break
		}
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
		rows, e := s.diagnosticPages(ctx, path, true)
		if e != nil {
			r.Diagnostics = append(r.Diagnostics, diagnosticlog.Segment{Path: path, Reason: e.Error()})
			continue
		}
		for _, row := range rows {
			if row.Eligible {
				r.DiagnosticCollectedBytes += row.Bytes
			}
		}
		r.Diagnostics = append(r.Diagnostics, rows...)
	}
	return r, nil
}

// RootFromSelected preserves the already selected scoped/flat namespace. The
// public CLI provides an explicit fallback; this helper never consults HOME.
func RootFromSelected(path string) (string, error) {
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if filepath.Base(filepath.Dir(p)) == "repos" {
		p = filepath.Dir(filepath.Dir(p))
	}
	return p, nil
}

// Explicit commands traverse all bounded pages; the scheduled worker persists
// one page cursor per batch instead of holding the UI or a daemon alive.
func (s *Service) diagnosticPages(ctx context.Context, path string, apply bool) ([]diagnosticlog.Segment, error) {
	var result []diagnosticlog.Segment
	cursor := ""
	for page := 0; page < 10000; page++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		var rows []diagnosticlog.Segment
		var next string
		var complete bool
		var err error
		if apply {
			err = s.Collector.Coordinator.WithLock(ctx, func(*storagegc.Locked) error {
				registry, err := s.Collector.Coordinator.ReadRegistry()
				if err != nil {
					return err
				}
				if !registry.MigrationComplete {
					return errors.New("Couch inventory not acknowledged")
				}
				rows, next, complete, err = diagnosticlog.CollectLegacyPage(path, s.DiagnosticOptions, cursor, 100)
				return err
			})
		} else {
			rows, next, complete, err = diagnosticlog.PreviewLegacyPage(path, s.DiagnosticOptions, cursor, 100)
		}
		if err != nil {
			return result, err
		}
		result = append(result, rows...)
		if complete {
			return result, nil
		}
		if next == cursor {
			return result, errors.New("diagnostic cursor did not advance")
		}
		cursor = next
	}
	return result, errors.New("diagnostic traversal budget exceeded")
}
