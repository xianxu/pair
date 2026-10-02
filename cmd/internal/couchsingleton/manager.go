package couchsingleton

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

type Manager struct {
	AuthorityDir  string
	IsolationRoot string
	Defaults      Roots
	Proc          couchcore.ProcOps
	// Optional synchronous IO failure/order seams; nil in production.
	AfterInspect  func() error
	BeforePublish func() error
	Publish       func(string, []byte) error
}

func (m Manager) selectionPath() string { return filepath.Join(m.AuthorityDir, "selection.json") }
func (m Manager) Read(q Request) (Selection, error) {
	var s Selection
	if e := m.validateRequest(q); e != nil {
		return s, e
	}
	raw, e := readBounded(m.selectionPath(), maxSelectionBytes)
	if e != nil {
		if errors.Is(e, os.ErrNotExist) {
			return s, fmt.Errorf("Couch is UNMIGRATED; preview with couch --adopt-store %q, then repeat with --apply <digest>: %w", resolved(m.Defaults, q).StoreDir, e)
		}
		return s, fmt.Errorf("Couch selection unavailable; restore %q: %w", m.selectionPath(), e)
	}
	if e = strictjson.Decode(raw, &s); e != nil {
		return s, fmt.Errorf("invalid Couch selection; restore %q: %w", m.selectionPath(), e)
	}
	if e := m.validatePaths(s.Roots, s.Excluded); e != nil {
		return s, e
	}
	if s.Version != 1 {
		return s, errors.New("unsupported Couch selection version")
	}
	if e = validateRoots(s.Roots); e != nil {
		return s, e
	}
	for _, p := range []string{s.Roots.StoreDir, s.Roots.PairDataDir, s.Roots.IdentityDir} {
		if _, e = couchcore.ExistingCouchNamespace(p); e != nil {
			return s, fmt.Errorf("selected root unavailable; restore %s: %w", p, e)
		}
	}
	for i, p := range s.Excluded {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || p == s.Roots.StoreDir || i > 0 && s.Excluded[i-1] >= p {
			return s, errors.New("invalid selection exclusions")
		}
	}
	for _, pair := range [][2]string{{q.Roots.StoreDir, s.Roots.StoreDir}, {q.Roots.PairDataDir, s.Roots.PairDataDir}, {q.Roots.IdentityDir, s.Roots.IdentityDir}} {
		if pair[0] != "" && pair[0] != pair[1] {
			return s, fmt.Errorf("configured root %q conflicts with selected %q; unset the override", pair[0], pair[1])
		}
	}
	if len(q.Exclude) > 0 && !sameStrings(q.Exclude, s.Excluded) {
		return s, errors.New("exclusions conflict with immutable selection")
	}
	return s, nil
}
func (m Manager) Preview(q Request) (Report, error) {
	if e := m.validateRequest(q); e != nil {
		return Report{}, e
	}
	if _, e := os.Lstat(m.selectionPath()); e == nil {
		s, e := m.Read(q)
		return Report{Selection: s, Status: "SELECTED"}, e
	} else if !errors.Is(e, os.ErrNotExist) {
		return Report{}, e
	}
	return m.inspect(q, nil)
}
func (m Manager) Adopt(q Request, expect string) (Selection, error) {
	s, l, e := m.acquire(q, expect, true)
	if l != nil {
		e = errors.Join(e, l.Close())
	}
	return s, e
}
func (m Manager) Acquire(q Request) (Selection, io.Closer, error) { return m.acquire(q, "", false) }

type leases struct{ host, store *couchcore.SupervisorLease }

func (l *leases) Close() error { return errors.Join(l.store.Close(), l.host.Close()) }
func (m Manager) acquire(q Request, expect string, explicit bool) (s Selection, result io.Closer, err error) {
	if e := m.validateRequest(q); e != nil {
		return s, nil, e
	}
	if m.Proc == nil {
		m.Proc = couchcore.OSProcOps{}
	}
	host, e := couchcore.ResolveCouchNamespace(m.AuthorityDir, "")
	if e != nil {
		return s, nil, e
	}
	if _, probeErr := couchcore.ObserveSupervisor(host, m.Proc, nil); probeErr != nil {
		return s, nil, fmt.Errorf("Couch singleton ownership unavailable: %w", probeErr)
	}
	hl, e := couchcore.AcquireSupervisorLease(host, m.Proc)
	if e != nil {
		if selected, readErr := m.Read(Request{}); readErr == nil {
			return s, nil, fmt.Errorf("Couch singleton for selected store %q already running or unavailable: %w", selected.Roots.StoreDir, e)
		}
		return s, nil, fmt.Errorf("Couch singleton already running or unavailable: %w", e)
	}
	l := &leases{host: hl}
	defer func() {
		if err != nil {
			err = errors.Join(err, l.Close())
		}
	}()
	_, e = os.Lstat(m.selectionPath())
	existing := e == nil
	var report Report
	if existing {
		s, e = m.Read(q)
	} else if errors.Is(e, os.ErrNotExist) {
		report, e = m.inspect(q, nil)
		if e == nil && len(report.Blockers) > 0 {
			e = errors.New(report.Blockers[0])
		}
		if e == nil && explicit && (expect == "" || expect != report.Digest) {
			e = errors.New("adoption evidence changed; run a fresh preview")
		}
		s = report.Selection
	}
	if e != nil {
		return s, nil, e
	}
	if m.AfterInspect != nil {
		if e = m.AfterInspect(); e != nil {
			return s, nil, e
		}
	}
	for _, p := range []string{s.Roots.PairDataDir, s.Roots.IdentityDir, s.Roots.StoreDir} {
		if !existing {
			if e = os.MkdirAll(p, 0700); e != nil {
				return s, nil, e
			}
		}
		if _, e = couchcore.ExistingCouchNamespace(p); e != nil {
			return s, nil, e
		}
	}
	ns, e := couchcore.ExistingCouchNamespace(s.Roots.StoreDir)
	if e != nil {
		return s, nil, e
	}
	if _, probeErr := couchcore.ObserveSupervisor(ns, m.Proc, nil); probeErr != nil {
		return s, nil, fmt.Errorf("selected store %q ownership unavailable: %w", ns.Dir(), probeErr)
	}
	l.store, e = couchcore.AcquireSupervisorLease(ns, m.Proc)
	if e != nil {
		return s, nil, e
	}
	if !existing {
		if m.BeforePublish != nil {
			if e = m.BeforePublish(); e != nil {
				return s, nil, e
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		paths := make([]string, 0, len(report.Candidates))
		var namespaces []couchcore.CouchNamespace
		for _, c := range report.Candidates {
			paths = append(paths, c.Roots.StoreDir)
			ns, e := couchcore.ExistingCouchNamespace(c.Roots.StoreDir)
			if e == nil {
				namespaces = append(namespaces, ns)
			} else if !errors.Is(e, os.ErrNotExist) {
				return s, nil, e
			}
		}
		e = (&storagegc.Coordinator{Root: s.Roots.PairDataDir}).TryWithLock(ctx, func(_ *storagegc.Locked) error {
			return (couchidentity.IdentityStore{HostDir: s.Roots.IdentityDir}).WithInspectionLocks(ctx, paths, func() error {
				return couchcore.WithStoreInspectionLocks(ctx, namespaces, func(path string) error { return m.validatePaths(s.Roots, []string{path}) }, func(inspection *couchcore.StoreInspection) error {
					fresh, e := m.inspectSources(q, l.store, inspection)
					if e != nil {
						return e
					}
					if len(fresh.Blockers) > 0 {
						return errors.New(fresh.Blockers[0])
					}
					if fresh.Digest != report.Digest || !reflect.DeepEqual(s, fresh.Selection) {
						return errors.New("adoption source changed before publication; preview again")
					}
					raw, e := encodeSelection(s)
					if e != nil {
						return e
					}
					publish := m.Publish
					if publish == nil {
						publish = func(p string, b []byte) error { return durablefile.WriteAtomicStaged(p, b, p+".publication") }
					}
					return publish(m.selectionPath(), raw)
				})
			})
		})
		if e != nil {
			return s, nil, e
		}
	}
	return s, l, nil
}
