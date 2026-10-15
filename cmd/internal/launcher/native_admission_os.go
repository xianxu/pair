package launcher

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type NativeAdmissionStore struct {
	Home string
	io   *nativeAdmissionIO
}
type nativeAdmissionIO struct {
	self    func() (storagegc.ProcessIdentity, error)
	child   func(int) (storagegc.ProcessIdentity, int, error)
	inspect func(storagegc.ProcessIdentity) storagegc.Liveness
	acquire func(context.Context, string) (io.Closer, error)
	read    func(context.Context, string, int64) ([]byte, error)
	write   func(string, []byte) error
}
type nativeAdmissionFile struct {
	Version int
	Entries []NativeAdmissionEntry
}

func (s NativeAdmissionStore) ops() nativeAdmissionIO {
	ops := nativeAdmissionIO{self: func() (storagegc.ProcessIdentity, error) { return storagegc.CurrentProcessIdentity(os.Getpid()) }, child: strictAdmissionProcess, inspect: (storagegc.OSProcessProbe{}).Inspect, acquire: AcquireTerminalLease, read: osfs.ReadRegularMetadata, write: func(path string, raw []byte) error { return durablefile.WriteAtomic(path, raw, ".native-admission-*") }}
	if s.io != nil {
		if s.io.self != nil {
			ops.self = s.io.self
		}
		if s.io.child != nil {
			ops.child = s.io.child
		}
		if s.io.inspect != nil {
			ops.inspect = s.io.inspect
		}
		if s.io.acquire != nil {
			ops.acquire = s.io.acquire
		}
		if s.io.read != nil {
			ops.read = s.io.read
		}
		if s.io.write != nil {
			ops.write = s.io.write
		}
	}
	return ops
}
func strictAdmissionProcess(pid int) (storagegc.ProcessIdentity, int, error) {
	before, err := storagegc.CurrentProcessIdentity(pid)
	if err != nil {
		return before, 0, err
	}
	rows, err := procutil.Table()
	if err != nil {
		return before, 0, err
	}
	for _, p := range rows {
		if p.PID != pid {
			continue
		}
		after, err := storagegc.CurrentProcessIdentity(pid)
		if err != nil || before != after {
			return before, 0, errors.New("native process generation changed")
		}
		if p.Zombie {
			return before, 0, errors.New("native process already exited")
		}
		return before, p.PPID, nil
	}
	return before, 0, errors.New("native process parent unavailable")
}
func (s NativeAdmissionStore) withStore(ctx context.Context, fn func(context.Context, *nativeAdmissionFile, nativeAdmissionIO) error) (err error) {
	if ctx == nil {
		return errors.New("native admission requires context")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, zjTimeout)
	defer cancel()
	cache, err := artifactpath.ResolvePairCache(s.Home)
	if err != nil {
		return err
	}
	path, err := cache.NativeAdmissions()
	if err != nil {
		return err
	}
	lock, err := cache.NativeAdmissionLease()
	if err != nil {
		return err
	}
	ops := s.ops()
	lease, err := ops.acquire(ctx, lock)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	state := nativeAdmissionFile{Version: 1}
	raw, err := ops.read(ctx, path, maxNativeAdmissionBytes)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil {
		if err = strictjson.Decode(raw, &state); err != nil {
			return err
		}
	}
	if state.Version != 1 || len(state.Entries) > maxNativeAdmissions {
		return errors.New("unsupported or oversized native admission store")
	}
	seen := make(map[[2]string]bool, len(state.Entries))
	for _, e := range state.Entries {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = validateAdmissionEntry(e); err != nil {
			return err
		}
		key := [2]string{e.Request.Agent, e.Request.NativeID}
		if seen[key] {
			return errors.New("duplicate native admission key")
		}
		seen[key] = true
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	before, _ := json.Marshal(state)
	if err = fn(ctx, &state, ops); err != nil {
		return err
	}
	after, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if len(after) > maxNativeAdmissionBytes || len(state.Entries) > maxNativeAdmissions {
		return errors.New("native admission capacity exhausted; use Fresh")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if !slices.Equal(before, after) {
		return ops.write(path, after)
	}
	return nil
}
func newAdmissionToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
func encodeAdmissionGrant(g nativeAdmissionGrant) (string, error) {
	raw, err := json.Marshal(g)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
func decodeAdmissionGrant(raw string) (nativeAdmissionGrant, error) {
	var g nativeAdmissionGrant
	if len(raw) > 16384 {
		return g, errors.New("native admission grant too large")
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return g, err
	}
	if err = strictjson.Decode(b, &g); err != nil {
		return g, err
	}
	if g.Version != 1 {
		return g, errors.New("unsupported native admission grant")
	}
	e := NativeAdmissionEntry{Request: g.Request, Token: g.Token, Helper: g.Helper, Phase: NativeAdmissionReserved}
	return g, validateAdmissionEntry(e)
}
func admissionIndex(state *nativeAdmissionFile, g nativeAdmissionGrant) int {
	for i, e := range state.Entries {
		if sameAdmissionKey(e.Request, g.Request) {
			return i
		}
	}
	return -1
}
func applyAdmission(state *nativeAdmissionFile, g nativeAdmissionGrant, c NativeAdmissionChange) error {
	i := admissionIndex(state, g)
	var current *NativeAdmissionEntry
	if i >= 0 {
		current = &state.Entries[i]
	}
	next, err := AdvanceNativeAdmission(current, c)
	if err != nil {
		return err
	}
	if next == nil {
		state.Entries = slices.Delete(state.Entries, i, i+1)
	} else if i >= 0 {
		state.Entries[i] = *next
	} else {
		state.Entries = append(state.Entries, *next)
	}
	return nil
}
func (s NativeAdmissionStore) Reserve(ctx context.Context, r NativeAdmissionRequest, check func(context.Context, NativeAdmissionOwners) error) (string, error) {
	if err := validateAdmissionRequest(r); err != nil {
		return "", err
	}
	if check == nil {
		return "", errors.New("native admission requires legacy owner observation")
	}
	var grant nativeAdmissionGrant
	err := s.withStore(ctx, func(ctx context.Context, state *nativeAdmissionFile, ops nativeAdmissionIO) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		self, err := ops.self()
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !validAdmissionProcess(self) {
			return errors.New("native helper identity unavailable")
		}
		// Only both-dead running entries are collectable. Unknown spawning effects
		// and dead helpers without a completed handoff remain deliberately retained.
		for i := len(state.Entries) - 1; i >= 0; i-- {
			if err := ctx.Err(); err != nil {
				return err
			}
			e := state.Entries[i]
			if e.Phase != NativeAdmissionRunning {
				continue
			}
			// ProcessProbe is synchronous and has no context API. A single
			// probe must return before cancellation can be observed; never
			// start the next probe or transition after that cancellation.
			w := ops.inspect(e.Wrapper)
			if err := ctx.Err(); err != nil {
				return err
			}
			c := ops.inspect(e.Child)
			if err := ctx.Err(); err != nil {
				return err
			}
			if w == storagegc.ProcessDead && c == storagegc.ProcessDead {
				if err := applyAdmission(state, nativeAdmissionGrant{Request: e.Request}, NativeAdmissionChange{Transition: NativeAdmissionCollect, WrapperLiveness: w, ChildLiveness: c}); err != nil {
					return err
				}
			}
		}
		for _, e := range state.Entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if sameAdmissionKey(e.Request, r) {
				return errors.New("native conversation already has an active or unresolved writer; use Fresh")
			}
		}
		if len(state.Entries) >= maxNativeAdmissions {
			return errors.New("native admission capacity exhausted; use Fresh")
		}
		if err := withNativeAdmissionOwners(ctx, state, ops, check); err != nil {
			return err
		}
		token, err := newAdmissionToken()
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		grant = nativeAdmissionGrant{Version: 1, Request: r, Helper: self, Token: token}
		return applyAdmission(state, grant, admissionChange(grant, NativeAdmissionReserve, storagegc.ProcessIdentity{}))
	})
	if err != nil {
		return "", err
	}
	return encodeAdmissionGrant(grant)
}
func (s NativeAdmissionStore) Claim(ctx context.Context, raw, agent, digest string) (claim NativeAdmissionClaim, err error) {
	grant, err := decodeAdmissionGrant(raw)
	if err != nil {
		return claim, err
	}
	if agent != grant.Request.Agent || digest != grant.Request.CommandDigest {
		return claim, errors.New("native admission invocation changed")
	}
	err = s.withStore(ctx, func(ctx context.Context, state *nativeAdmissionFile, ops nativeAdmissionIO) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		self, e := ops.self()
		if e != nil {
			return e
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if e = applyAdmission(state, grant, admissionChange(grant, NativeAdmissionSpawn, self)); e != nil {
			return e
		}
		claim = NativeAdmissionClaim{grant: grant, wrapper: self, home: filepath.Clean(s.Home)}
		return nil
	})
	if err != nil {
		return NativeAdmissionClaim{}, err
	}
	return claim, nil
}
func (s NativeAdmissionStore) PublishChild(ctx context.Context, claim NativeAdmissionClaim, pid int) error {
	if claim.home == "" || claim.home != filepath.Clean(s.Home) || !validAdmissionProcess(claim.wrapper) {
		return errors.New("invalid native admission claim")
	}
	return s.withStore(ctx, func(ctx context.Context, state *nativeAdmissionFile, ops nativeAdmissionIO) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		self, e := ops.self()
		if e != nil {
			return e
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if self != claim.wrapper {
			return errors.New("native wrapper generation changed")
		}
		child, parent, e := ops.child(pid)
		if e != nil {
			return e
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if parent != self.PID {
			return errors.New("native child does not belong to claimed wrapper")
		}
		i := admissionIndex(state, claim.grant)
		if i < 0 || state.Entries[i].Wrapper != self {
			return errors.New("native wrapper claim changed")
		}
		return applyAdmission(state, claim.grant, admissionChange(claim.grant, NativeAdmissionPublish, child))
	})
}

// CancelBeforeHandoff is valid only before the caller exposes the grant to any
// spawning side effect. It is never deferred across a terminal handoff.
func (s NativeAdmissionStore) CancelBeforeHandoff(ctx context.Context, raw string) error {
	g, err := decodeAdmissionGrant(raw)
	if err != nil {
		return err
	}
	return s.withStore(ctx, func(ctx context.Context, state *nativeAdmissionFile, ops nativeAdmissionIO) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		self, e := ops.self()
		if e != nil {
			return e
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return applyAdmission(state, g, admissionChange(g, NativeAdmissionCancel, self))
	})
}

// withNativeAdmissionOwners lends redacted immutable facts only while the
// caller holds the store lock. It neither collects nor publishes authority.
func withNativeAdmissionOwners(ctx context.Context, state *nativeAdmissionFile, ops nativeAdmissionIO, check func(context.Context, NativeAdmissionOwners) error) error {
	facts := make([]NativeAdmissionEntry, 0, len(state.Entries))
	for _, e := range state.Entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if e.Phase == NativeAdmissionSpawning || e.Phase == NativeAdmissionRunning {
			e.Token = ""
			facts = append(facts, e)
		}
	}
	var active atomic.Bool
	active.Store(true)
	defer active.Store(false)
	if err := ctx.Err(); err != nil {
		return err
	}
	process := func(pid int) (storagegc.ProcessIdentity, int, error) {
		if err := ctx.Err(); err != nil {
			return storagegc.ProcessIdentity{}, 0, err
		}
		identity, parent, err := ops.child(pid)
		if err != nil {
			return identity, parent, err
		}
		if err := ctx.Err(); err != nil {
			return storagegc.ProcessIdentity{}, 0, err
		}
		return identity, parent, nil
	}
	if err := check(ctx, NativeAdmissionOwners{entries: facts, process: process, active: &active}); err != nil {
		return err
	}
	return ctx.Err()
}
