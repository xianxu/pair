package couchidentity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"io"
	"math"
	"os"
	"path/filepath"
)

const maxStores = 4096

type IdentityStore struct {
	HostDir, StoreDir string
	beforePublish     func(string) error
}
type hostRegistry struct {
	SchemaVersion int                 `json:"schema_version"`
	NextC         uint64              `json:"next_c"`
	Stores        []StoreRegistration `json:"stores"`
}

func authorityError(path string, e error) error {
	return fmt.Errorf("identity authority %s: %w; restore non-regressed host authority and valid local state (coordinated rollback or automatic re-enrollment is unsupported)", path, e)
}
func readState(path string, limit int64, out any) (bool, error) {
	f, e := openRegular(path, os.O_RDONLY)
	if errors.Is(e, os.ErrNotExist) {
		return false, nil
	}
	if e != nil {
		return false, authorityError(path, e)
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, limit+1))
	if e != nil {
		return false, authorityError(path, e)
	}
	if int64(len(raw)) > limit {
		return false, authorityError(path, errors.New("size limit exceeded"))
	}
	if e = strictjson.Decode(raw, out); e != nil {
		return false, authorityError(path, e)
	}
	return true, nil
}
func (h hostRegistry) validate() error {
	if h.SchemaVersion != 1 || h.NextC == 0 || len(h.Stores) > maxStores {
		return errors.New("invalid host registry")
	}
	paths := map[string]bool{}
	ids := map[uint64]bool{}
	for _, r := range h.Stores {
		if !filepath.IsAbs(r.StorePath) || filepath.Clean(r.StorePath) != r.StorePath || r.C == 0 || r.C >= h.NextC || paths[r.StorePath] || ids[r.C] {
			return errors.New("invalid or duplicate host registration")
		}
		paths[r.StorePath] = true
		ids[r.C] = true
	}
	return nil
}
func (s IdentityStore) Allocate(ctx context.Context, q AllocationRequest) (AllocationResult, error) {
	var zero AllocationResult
	if ctx == nil {
		return zero, errors.New("allocation context is required")
	}
	if e := ctx.Err(); e != nil {
		return zero, e
	}
	if s.HostDir == "" || s.StoreDir == "" {
		return zero, errors.New("host and store directories are required")
	}
	for _, p := range []string{s.HostDir, s.StoreDir} {
		if e := durablefile.EnsureDirectory(p); e != nil {
			return zero, e
		}
	}
	root, e := filepath.Abs(s.StoreDir)
	if e != nil {
		return zero, e
	}
	root, e = filepath.EvalSymlinks(root)
	if e != nil {
		return zero, e
	}
	host, e := filepath.Abs(s.HostDir)
	if e != nil {
		return zero, e
	}
	host, e = filepath.EvalSymlinks(host)
	if e != nil {
		return zero, e
	}
	hp, lp := filepath.Join(host, "couch-identities.json"), filepath.Join(root, "identities.json")
	hl, e := acquireLock(ctx, filepath.Join(host, "couch-identities.lock"))
	if e != nil {
		return zero, e
	}
	defer hl.Close()
	ll, e := acquireLock(ctx, filepath.Join(root, "identities.lock"))
	if e != nil {
		return zero, e
	}
	defer ll.Close()
	var h hostRegistry
	var state AllocationState
	he, e := readState(hp, 4<<20, &h)
	if e != nil {
		return zero, e
	}
	le, e := readState(lp, 4<<10, &state)
	if e != nil {
		return zero, e
	}
	if le {
		if e = state.Validate(); e != nil {
			return zero, authorityError(lp, e)
		}
	}
	if !he {
		if le {
			return zero, authorityError(hp, errors.New("missing host authority for existing local C"))
		}
		h = hostRegistry{SchemaVersion: 1, NextC: 1}
	} else if e = h.validate(); e != nil {
		return zero, authorityError(hp, e)
	}
	idx := -1
	old := -1
	for i, r := range h.Stores {
		if r.StorePath == root {
			idx = i
		}
		if le && r.C == state.C && r.StorePath == state.StorePath {
			old = i
		}
	}
	if le {
		if old < 0 || state.LastN > h.Stores[old].LastN || state.LastM > h.Stores[old].LastM {
			return zero, authorityError(hp, errors.New("host authority regressed or does not prove the local allocation"))
		}
		if idx >= 0 && (state.C != h.Stores[idx].C || state.StorePath != root) && (h.Stores[idx].LastN != 0 || h.Stores[idx].LastM != 0) {
			return zero, authorityError(lp, errors.New("local C conflicts with canonical store registration"))
		}
	}
	if idx < 0 {
		if len(h.Stores) == maxStores {
			return zero, authorityError(hp, errors.New("4096 permanent stores reserved; migrate to an independently administered namespace"))
		}
		if h.NextC == math.MaxUint64 {
			return zero, authorityError(hp, errors.New("store identity counter exhausted"))
		}
		h.Stores = append(h.Stores, StoreRegistration{StorePath: root, C: h.NextC})
		h.NextC++
		idx = len(h.Stores) - 1
		state = AllocationState{SchemaVersion: 1, C: h.Stores[idx].C, StorePath: root}
	} else if !le || old != idx {
		r := h.Stores[idx]
		if r.LastN != 0 || r.LastM != 0 {
			return zero, authorityError(lp, errors.New("missing consumed local state"))
		}
		state = AllocationState{SchemaVersion: 1, C: r.C, StorePath: root}
	}
	r := h.Stores[idx]
	state.LastN = max(state.LastN, r.LastN)
	state.LastM = max(state.LastM, r.LastM)
	next, result, e := AdvanceAllocation(state, q)
	if e != nil {
		return zero, e
	}
	// Enrollment is independently durable so a first local-write failure remains retryable.
	if !le || old != idx {
		if e = s.publish(ctx, hp, h); e != nil {
			return zero, e
		}
		if e = s.publish(ctx, lp, state); e != nil {
			return zero, e
		}
	}
	h.Stores[idx].LastN = next.LastN
	h.Stores[idx].LastM = next.LastM
	if e = s.publish(ctx, hp, h); e != nil {
		return zero, e
	}
	if e = s.publish(ctx, lp, next); e != nil {
		return zero, e
	}
	return result, nil
}
func (s IdentityStore) publish(ctx context.Context, path string, value any) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if s.beforePublish != nil {
		if e := s.beforePublish(path); e != nil {
			return e
		}
	}
	raw, e := json.Marshal(value)
	if e != nil {
		return e
	}
	limit := 4 << 20
	if filepath.Base(path) == "identities.json" {
		limit = 4 << 10
	}
	if len(raw) > limit {
		return authorityError(path, errors.New("size limit exceeded"))
	}
	return durablefile.WriteAtomicStaged(path, raw, path+".publication")
}
