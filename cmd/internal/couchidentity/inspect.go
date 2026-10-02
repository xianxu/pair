package couchidentity

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
)

// Inspection describes existing allocation authority without enrollment or writes.
type Inspection struct {
	Registrations           []StoreRegistration
	Local                   AllocationState
	HostExists, LocalExists bool
}

// Inspect shares the allocator's strict decoders and validators, but never creates
// roots, locks, reservations or counters. Callers revalidate before publication.
func (s IdentityStore) Inspect() (Inspection, error) {
	var out Inspection
	if physical, e := filepath.EvalSymlinks(s.StoreDir); e == nil {
		s.StoreDir = physical
	}
	var h hostRegistry
	hp, lp := filepath.Join(s.HostDir, "couch-identities.json"), filepath.Join(s.StoreDir, "identities.json")
	var err error
	out.HostExists, err = readState(hp, 4<<20, &h)
	if err != nil {
		return out, err
	}
	if out.HostExists {
		if err = h.validate(); err != nil {
			return out, authorityError(hp, err)
		}
	}
	out.Registrations = h.Stores
	out.LocalExists, err = readState(lp, 4<<10, &out.Local)
	if err != nil {
		return out, err
	}
	if out.LocalExists {
		if err = out.Local.Validate(); err != nil {
			return out, authorityError(lp, err)
		}
		if !out.HostExists {
			return out, authorityError(hp, errors.New("missing host authority for existing local C"))
		}
		proven := false
		for _, r := range h.Stores {
			if r.C == out.Local.C && r.StorePath == out.Local.StorePath && out.Local.LastN <= r.LastN && out.Local.LastM <= r.LastM {
				proven = true
			}
		}
		if !proven {
			return out, authorityError(hp, errors.New("host authority regressed or does not prove the local allocation"))
		}
		if out.Local.StorePath != s.StoreDir {
			return out, authorityError(lp, errors.New("local store identity moved; adoption cannot re-enroll"))
		}
	}
	for _, r := range h.Stores {
		if r.StorePath == s.StoreDir {
			if !out.LocalExists && (r.LastN != 0 || r.LastM != 0) {
				return out, authorityError(lp, errors.New("missing consumed local state"))
			}
			if out.LocalExists && out.Local.C != r.C {
				return out, authorityError(lp, errors.New("local C conflicts with canonical store registration"))
			}
		}
	}
	return out, nil
}

// WithInspectionLocks fences an adoption recheck with the allocator's existing
// host-then-local transaction locks, without allocating or publishing counters.
func (s IdentityStore) WithInspectionLocks(ctx context.Context, stores []string, fn func() error) (err error) {
	host, e := acquireLock(ctx, filepath.Join(s.HostDir, "couch-identities.lock"))
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, host.Close()) }()
	paths := append([]string(nil), stores...)
	slices.Sort(paths)
	paths = slices.Compact(paths)
	var locks []*allocationLock
	defer func() {
		for i := len(locks) - 1; i >= 0; i-- {
			err = errors.Join(err, locks[i].Close())
		}
	}()
	for _, p := range paths {
		if _, e := os.Lstat(p); errors.Is(e, os.ErrNotExist) {
			continue
		} else if e != nil {
			return e
		}
		l, e := acquireLock(ctx, filepath.Join(p, "identities.lock"))
		if e != nil {
			return e
		}
		locks = append(locks, l)
	}
	return fn()
}
