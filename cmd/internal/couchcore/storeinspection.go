package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type inspectedStore struct {
	lock      *threadStoreLock
	directory os.FileInfo
}

// StoreInspection proves ownership of the exact global and slot transaction
// locks retained for one callback. Its authority expires before locks release.
type StoreInspection struct {
	stores     map[string]inspectedStore
	namespaces map[CouchNamespace]bool
	active     bool
	ctx        context.Context
}

// WithStoreInspectionLocks reads enrollment under the global locks, then fences
// every discovered slot before calling fn. check runs before external slot
// directory discovery; nil permits the production fleet's external worktrees.
// Missing backends remain observations and are never initialized.
func WithStoreInspectionLocks(ctx context.Context, namespaces []CouchNamespace, check func(string) error, fn func(*StoreInspection) error) (err error) {
	if ctx == nil {
		return errors.New("inspection context is required")
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	inspection := &StoreInspection{ctx: ctx, stores: map[string]inspectedStore{}, namespaces: map[CouchNamespace]bool{}, active: true}
	var order []string
	defer func() {
		inspection.active = false
		for i := len(order) - 1; i >= 0; i-- {
			if lock := inspection.stores[order[i]].lock; lock != nil {
				err = errors.Join(err, lock.Close())
			}
		}
	}()
	retain := func(s *ThreadStore) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if _, ok := inspection.stores[s.root]; ok {
			return nil
		}
		if len(inspection.stores) >= 65536 {
			return errors.New("too many adoption storage backends")
		}
		if check != nil {
			if e := check(s.root); e != nil {
				return e
			}
		}
		if e := provisionSafePath(s.root); e != nil {
			return e
		}
		if e := s.validateBackendPath(); e != nil {
			return e
		}
		info, e := os.Lstat(s.root)
		if errors.Is(e, os.ErrNotExist) {
			inspection.stores[s.root] = inspectedStore{}
			order = append(order, s.root)
			return nil
		}
		if e != nil {
			return e
		}
		// Adoption holds the singleton's authority across every store it
		// inspects, so a busy store yields at once rather than wait under it.
		lock, e := s.retentionReadLock(context.Background(), 0)
		if e != nil {
			return e
		}
		inspection.stores[s.root] = inspectedStore{lock: lock, directory: info}
		order = append(order, s.root)
		return nil
	}
	nsList := append([]CouchNamespace(nil), namespaces...)
	sort.Slice(nsList, func(i, j int) bool { return nsList[i].Dir() < nsList[j].Dir() })
	for _, ns := range nsList {
		inspection.namespaces[ns] = true
		if e := retain(NewThreadStore(ns)); e != nil {
			return e
		}
	}
	var locals []*ThreadStore
	for _, ns := range nsList {
		if e := ctx.Err(); e != nil {
			return e
		}
		store := NewThreadStore(ns)
		store.inspection = inspection
		if inspection.stores[store.root].lock == nil {
			continue
		}
		var manifest threadManifest
		e := inspection.withRoot(store, func() error { var e error; manifest, _, _, e = store.loadManifestLocked(); return e })
		if e != nil {
			return e
		}
		if check != nil {
			for _, root := range manifest.SlotRepositories {
				for _, path := range []string{root, WorktreesRoot(filepath.Dir(root))} {
					if e := check(path); e != nil {
						return e
					}
				}
			}
		}
		backends, e := store.discoveredBackendsFromManifest(manifest)
		if e != nil {
			return e
		}
		if len(locals)+len(backends) > 65536 {
			return errors.New("too many adoption storage backends")
		}
		locals = append(locals, backends...)
	}
	sort.Slice(locals, func(i, j int) bool { return locals[i].root < locals[j].root })
	for _, store := range locals {
		if e := retain(store); e != nil {
			return e
		}
	}
	return fn(inspection)
}
func (i *StoreInspection) validateRoot(root string) error {
	if i == nil || !i.active {
		return errors.New("expired store inspection")
	}
	if e := i.ctx.Err(); e != nil {
		return e
	}
	held, ok := i.stores[root]
	if !ok {
		return fmt.Errorf("store appeared after inspection: %s", root)
	}
	current, e := os.Lstat(root)
	if held.lock == nil {
		if errors.Is(e, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("previously missing store changed: %s", root)
	}
	if e != nil {
		return e
	}
	if !os.SameFile(current, held.directory) {
		return fmt.Errorf("store directory changed: %s", root)
	}
	current, e = os.Lstat(filepath.Join(root, "store.lock"))
	if e != nil {
		return e
	}
	info, e := held.lock.file.Stat()
	if e != nil {
		return e
	}
	if !os.SameFile(current, info) {
		return fmt.Errorf("store lock changed: %s", root)
	}
	return nil
}
func (i *StoreInspection) withRoot(s *ThreadStore, fn func() error) error {
	if e := i.validateRoot(s.root); e != nil {
		return e
	}
	if i.stores[s.root].lock == nil {
		return nil
	}
	if e := s.validateBackendPath(); e != nil {
		return e
	}
	if _, e := os.Lstat(s.journalPath()); e == nil {
		return fmt.Errorf("store recovery pending: %s", s.root)
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	return fn()
}

// Snapshot reuses the normal strict decoders while retaining all inspection
// locks. Neither global nor local previews reacquire or recover storage.
func (i *StoreInspection) Snapshot(ns CouchNamespace) (ThreadSnapshot, error) {
	if i == nil || !i.active || !i.namespaces[ns] {
		return ThreadSnapshot{}, errors.New("namespace outside active store inspection")
	}
	for root := range i.stores {
		if e := i.validateRoot(root); e != nil {
			return ThreadSnapshot{}, e
		}
	}
	store := NewThreadStore(ns)
	store.inspection = i
	return store.PreviewSnapshot()
}

// ObserveMigrationProcesses includes every incarnation and in-flight start
// owner. Incomplete identity is unknown, never absence; recycled PIDs are dead.
func ObserveMigrationProcesses(ctx context.Context, proc ProcOps, records []ThreadRecord) ([]RecordedProcessObservation, error) {
	if ctx == nil {
		return nil, errors.New("process observation context is required")
	}
	if proc == nil {
		proc = OSProcOps{}
	}
	var out []RecordedProcessObservation
	for _, record := range records {
		for _, incarnation := range record.Incarnations {
			if e := ctx.Err(); e != nil {
				return nil, e
			}
			processes := []ProcessIdentity{{PID: incarnation.PID, Identity: incarnation.Identity}}
			if incarnation.Start != nil {
				processes = append(processes, ProcessIdentity{PID: incarnation.Start.OwnerPID, Identity: incarnation.Start.OwnerIdentity})
				if incarnation.PID == 0 && incarnation.Identity == "" {
					processes = processes[1:]
				}
			}
			for _, process := range processes {
				live := Unknown
				if process.PID > 0 && process.Identity != "" {
					live = observeExactProcessOrUnknown(proc, process)
				}
				out = append(out, RecordedProcessObservation{Address: record.Address, Process: process, Liveness: live})
			}
		}
	}
	return out, nil
}

func (s *ThreadStore) readInspectionPayload(path string) ([]byte, error) {
	file, e := openSupervisorObservation(path)
	if e != nil {
		return nil, e
	}
	defer file.Close()
	raw, e := io.ReadAll(io.LimitReader(file, localPayloadLimit+1))
	if e == nil && int64(len(raw)) > localPayloadLimit {
		return nil, fmt.Errorf("inspection payload exceeds %d bytes: %s", localPayloadLimit, path)
	}
	return raw, e
}
