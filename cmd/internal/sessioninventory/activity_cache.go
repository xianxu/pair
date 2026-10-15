package sessioninventory

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"os"
	"path/filepath"
	"sync"
)

// ActivityCatalogMetadata observes the disposable catalog without decoding it.
// A runtime without this seam remains correct through the uncached query.
type ActivityCatalogMetadata interface {
	ActivityCatalogFingerprint() (ArtifactFingerprint, error)
}

type activityDependency struct {
	artifact    Artifact
	fingerprint ArtifactFingerprint
	present     bool
}
type activityListing struct {
	root  StorageRoot
	files map[Artifact]ArtifactFingerprint
}
type activityEvidence struct {
	files          map[Artifact]activityDependency
	listings       []activityListing
	catalog        ArtifactFingerprint
	catalogPresent bool
	activity       SessionActivity
	present        bool
}
type activityKey struct {
	root, scope, tag string
	agent            Agent
}
type activityPhase uint8

const (
	activityAbsent activityPhase = iota
	activityLoading
	activityReady
)

type activityEvent uint8

const (
	activityBegin activityEvent = iota
	activityCommit
	activityDiscard
)

func advanceActivityCache(state activityPhase, event activityEvent) activityPhase {
	switch event {
	case activityBegin:
		return activityLoading
	case activityCommit:
		if state == activityLoading {
			return activityReady
		}
	case activityDiscard:
		return activityAbsent
	}
	return state
}

type activitySlot struct {
	gate     chan struct{}
	refs     int
	used     uint64
	phase    activityPhase
	evidence activityEvidence
}

// ActivityCache retains compact timestamps and the exact observed input stamps,
// not transcript trees. Each long-lived consumer owns one; no cache is global.
type ActivityCache struct {
	mu    sync.Mutex
	slots map[activityKey]*activitySlot
	clock uint64
}

func NewActivityCache() *ActivityCache {
	return &ActivityCache{slots: make(map[activityKey]*activitySlot)}
}
func (c *ActivityCache) acquire(ctx context.Context, key activityKey) (*activitySlot, func(), error) {
	c.mu.Lock()
	c.clock++
	if c.slots == nil {
		c.slots = make(map[activityKey]*activitySlot)
	}
	slot := c.slots[key]
	if slot == nil {
		if len(c.slots) >= 128 {
			var oldest activityKey
			var candidate *activitySlot
			for k, v := range c.slots {
				if v.refs == 0 && (candidate == nil || v.used < candidate.used) {
					oldest, candidate = k, v
				}
			}
			if candidate != nil {
				delete(c.slots, oldest)
			} else {
				c.mu.Unlock()
				return nil, func() {}, nil
			}
		}
		slot = &activitySlot{gate: make(chan struct{}, 1)}
		c.slots[key] = slot
	}
	slot.refs++
	slot.used = c.clock
	c.mu.Unlock()
	releaseRef := func() { c.mu.Lock(); slot.refs--; c.mu.Unlock() }
	select {
	case slot.gate <- struct{}{}:
		return slot, func() { <-slot.gate; releaseRef() }, nil
	case <-ctx.Done():
		releaseRef()
		return nil, func() {}, ctx.Err()
	}
}

// Query keeps QuerySessionContext's scanner-authorized, current-launch-only
// semantics. File changes invalidate success before any result is reused.
func (c *ActivityCache) Query(ctx context.Context, rt Runtime, scope, tag string, agent Agent) (SessionActivity, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return SessionActivity{}, false, err
	}
	metadata, canCache := rt.(ActivityCatalogMetadata)
	if c == nil || !canCache {
		q, e := QuerySessionContext(ctx, rt, scope, tag, agent)
		if e != nil {
			return SessionActivity{}, false, e
		}
		return ActivityForSession(rt, q)
	}
	slot, release, err := c.acquire(ctx, activityKey{filepath.Clean(rt.PairDataRoot().Path), scope, tag, agent})
	if err != nil {
		return SessionActivity{}, false, err
	}
	defer release()
	if slot != nil && slot.phase == activityReady && activityDependenciesCurrent(ctx, rt, metadata, slot.evidence) {
		return slot.evidence.activity, slot.evidence.present, nil
	}
	if err := ctx.Err(); err != nil {
		if slot != nil {
			slot.phase = advanceActivityCache(slot.phase, activityDiscard)
		}
		return SessionActivity{}, false, err
	}
	if slot != nil {
		slot.phase = advanceActivityCache(slot.phase, activityBegin)
	}
	for attempt := 0; attempt < 2; attempt++ {
		tracker := &activityObservedRuntime{Runtime: rt, files: map[Artifact]activityDependency{}}
		paths, pathErr := artifactpath.ResolveScoped(rt.PairDataRoot().Path, tag)
		if pathErr != nil {
			if slot != nil {
				slot.phase = advanceActivityCache(slot.phase, activityDiscard)
			}
			return SessionActivity{}, false, pathErr
		}
		_, ledgerErr := tracker.StatFile(Artifact{StorageRoot: rt.PairDataRoot().Name, RelativePath: filepath.Base(paths.Ledger())})
		if ledgerErr != nil && !errors.Is(ledgerErr, os.ErrNotExist) {
			if slot != nil {
				slot.phase = advanceActivityCache(slot.phase, activityDiscard)
			}
			return SessionActivity{}, false, ledgerErr
		}
		catalog, present, err := activityCatalogStamp(metadata)
		if err != nil {
			if slot != nil {
				slot.phase = advanceActivityCache(slot.phase, activityDiscard)
			}
			// Catalogs are disposable acceleration, not required authority.
			// Preserve QuerySessionContext's proof/native fallback and keep
			// this unobservable input out of cache publication.
			q, queryErr := QuerySessionContext(ctx, rt, scope, tag, agent)
			if queryErr != nil {
				return SessionActivity{}, false, queryErr
			}
			return ActivityForSession(rt, q)
		}
		var queryRuntime Runtime = tracker
		if advancer, ok := rt.(SessionCatalogAdvancer); ok {
			queryRuntime = &activityCatalogRuntime{activityObservedRuntime: tracker, advancer: advancer, catalogPresent: present}
		}
		q, queryErr := QuerySessionContext(ctx, queryRuntime, scope, tag, agent)
		if queryErr != nil {
			if slot != nil {
				slot.phase = advanceActivityCache(slot.phase, activityDiscard)
			}
			return SessionActivity{}, false, queryErr
		}
		activity, found, activityErr := ActivityForSession(queryRuntime, q)
		if activityErr != nil || ctx.Err() != nil {
			if slot != nil {
				slot.phase = advanceActivityCache(slot.phase, activityDiscard)
			}
			if ctx.Err() != nil {
				activityErr = ctx.Err()
			}
			return SessionActivity{}, false, activityErr
		}
		// Optional catalog/native failures may still leave an independently valid
		// fallback. Return that result, but never retain incomplete observations.
		if tracker.incomplete {
			if slot != nil {
				slot.phase = advanceActivityCache(slot.phase, activityDiscard)
			}
			return activity, found, nil
		}
		evidence := activityEvidence{files: tracker.files, listings: tracker.listings, catalog: catalog, catalogPresent: present, activity: activity, present: found}
		// A successful query may publish catalog advancement itself. Retry with that
		// generation rather than claiming the old snapshot was stable.
		stable := !tracker.changed && activityDependenciesCurrent(ctx, rt, metadata, evidence)
		if stable {
			if slot != nil && (found || len(q.Diagnostics) == 0) {
				slot.evidence = evidence
				slot.phase = advanceActivityCache(slot.phase, activityCommit)
			} else if slot != nil {
				slot.phase = advanceActivityCache(slot.phase, activityDiscard)
			}
			return activity, found, nil
		}
	}
	if slot != nil {
		slot.phase = advanceActivityCache(slot.phase, activityDiscard)
	}
	if err := ctx.Err(); err != nil {
		return SessionActivity{}, false, err
	}
	return SessionActivity{}, false, ErrArtifactChanged
}
func activityCatalogStamp(rt ActivityCatalogMetadata) (ArtifactFingerprint, bool, error) {
	f, e := rt.ActivityCatalogFingerprint()
	if errors.Is(e, os.ErrNotExist) {
		return ArtifactFingerprint{}, false, nil
	}
	return f, e == nil, e
}

// activityDependenciesCurrent is the thin observation shell. Equality and
// legal reuse are decided by activityDependenciesMatch without filesystem IO.
func activityDependenciesCurrent(ctx context.Context, rt Runtime, catalog ActivityCatalogMetadata, want activityEvidence) bool {
	if ctx.Err() != nil {
		return false
	}
	stamp, present, err := activityCatalogStamp(catalog)
	if err != nil {
		return false
	}
	observed := activityEvidence{catalog: stamp, catalogPresent: present, files: make(map[Artifact]activityDependency, len(want.files))}
	for artifact := range want.files {
		if ctx.Err() != nil {
			return false
		}
		entry, err := rt.StatFile(artifact)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return false
		}
		d := activityDependency{artifact: artifact, present: err == nil}
		if d.present {
			d.fingerprint = fingerprintFromEntry(entry)
		}
		observed.files[artifact] = d
	}
	for _, prior := range want.listings {
		entries, err := rt.ListFiles(prior.root)
		if err != nil {
			return false
		}
		listing := activityListing{root: prior.root, files: make(map[Artifact]ArtifactFingerprint, len(entries))}
		for _, entry := range entries {
			listing.files[activityArtifactKey(entry.Artifact)] = fingerprintFromEntry(entry)
		}
		observed.listings = append(observed.listings, listing)
	}
	return ctx.Err() == nil && activityDependenciesMatch(want, observed)
}
func activityDependenciesMatch(want, observed activityEvidence) bool {
	if want.catalogPresent != observed.catalogPresent || (want.catalogPresent && !equalFingerprint(want.catalog, observed.catalog)) || len(want.files) != len(observed.files) || len(want.listings) != len(observed.listings) {
		return false
	}
	for a, w := range want.files {
		got, ok := observed.files[a]
		if !ok || got.present != w.present || (w.present && !equalFingerprint(w.fingerprint, got.fingerprint)) {
			return false
		}
	}
	for i, w := range want.listings {
		got := observed.listings[i]
		if w.root != got.root || len(w.files) != len(got.files) {
			return false
		}
		for a, f := range w.files {
			g, ok := got.files[a]
			if !ok || !equalFingerprint(f, g) {
				return false
			}
		}
	}
	return true
}

func activityArtifactKey(a Artifact) Artifact {
	return Artifact{StorageRoot: a.StorageRoot, RelativePath: a.RelativePath}
}

// This adapter records metadata the real query actually observed. Native
// listings remain dependencies too: a newly appearing candidate must invalidate
// a proofless filename query. Pair-root listings select one exact ledger, whose
// read supplies its dependency, and do not make unrelated logs invalidate it.
type activityObservedRuntime struct {
	Runtime
	files      map[Artifact]activityDependency
	listings   []activityListing
	changed    bool
	incomplete bool
}

func (r *activityObservedRuntime) observe(a Artifact, entry FileEntry, err error) {
	a = activityArtifactKey(a)
	d := activityDependency{artifact: a, present: err == nil}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		r.incomplete = true
		return
	}
	if err == nil {
		d.fingerprint = fingerprintFromEntry(entry)
	}
	if prior, ok := r.files[a]; ok && (prior.present != d.present || (d.present && !equalFingerprint(prior.fingerprint, d.fingerprint))) {
		r.changed = true
	}
	r.files[a] = d
}
func (r *activityObservedRuntime) StatFile(a Artifact) (FileEntry, error) {
	e, err := r.Runtime.StatFile(a)
	r.observe(a, e, err)
	return e, err
}
func (r *activityObservedRuntime) ReadAt(a Artifact, offset, limit int64) ([]byte, bool, error) {
	_, _ = r.StatFile(a)
	b, e, err := r.Runtime.ReadAt(a, offset, limit)
	if err != nil {
		r.incomplete = true
	}
	_, _ = r.StatFile(a)
	return b, e, err
}
func (r *activityObservedRuntime) ReadFile(a Artifact, limit int64) ([]byte, error) {
	_, _ = r.StatFile(a)
	b, err := r.Runtime.ReadFile(a, limit)
	if err != nil {
		r.incomplete = true
	}
	_, _ = r.StatFile(a)
	return b, err
}
func (r *activityObservedRuntime) ListFiles(root StorageRoot) ([]FileEntry, error) {
	entries, err := r.Runtime.ListFiles(root)
	if err != nil {
		r.incomplete = true
	}
	if root.Name != r.PairDataRoot().Name {
		if err != nil {
			r.changed = true
			return entries, err
		}
		listing := activityListing{root: root, files: make(map[Artifact]ArtifactFingerprint, len(entries))}
		for _, e := range entries {
			listing.files[activityArtifactKey(e.Artifact)] = fingerprintFromEntry(e)
		}
		r.listings = append(r.listings, listing)
	}
	return entries, err
}
func (r *activityObservedRuntime) OwnerLaunchCache() *OwnerLaunchCache {
	return ownerLaunchCacheOf(r.Runtime)
}
func (r *activityObservedRuntime) DecodeWork(a Artifact, n int64) {
	if o, ok := r.Runtime.(interface{ DecodeWork(Artifact, int64) }); ok {
		o.DecodeWork(a, n)
	}
}

type activityCatalogRuntime struct {
	*activityObservedRuntime
	advancer       SessionCatalogAdvancer
	catalogPresent bool
}

func (r *activityCatalogRuntime) LoadSessionInventoryCatalog() (Catalog, error) {
	catalog, err := r.advancer.LoadSessionInventoryCatalog()
	if err != nil && (r.catalogPresent || !errors.Is(err, os.ErrNotExist)) {
		r.incomplete = true
	}
	return catalog, err
}
func (r *activityCatalogRuntime) PublishSessionInventoryValidations(v []TargetValidation) error {
	err := r.advancer.PublishSessionInventoryValidations(v)
	if err != nil {
		r.incomplete = true
	}
	return err
}
