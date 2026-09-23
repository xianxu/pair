package storagegc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

type ArchiveReference struct {
	SlotEnvironment string                    `json:"slot_environment,omitempty"`
	Store           string                    `json:"store"`
	Owner           artifactpath.StorageOwner `json:"owner"`
	RecordHash      string                    `json:"record_hash"`
	ArchivedAt      time.Time                 `json:"archived_at"`
	ClockError      string                    `json:"clock_error,omitempty"`
}
type References struct {
	Visible  []artifactpath.StorageOwner
	Archives []ArchiveReference
}

// ReferenceStore is implemented by the Couch-owned adapter at composition time.
// It retains its own cross-filesystem journal and exact operation receipts.
type ReferenceStore interface {
	Snapshot(context.Context, *Locked, []string) (References, error)
	Detach(*Locked, string, ArchiveReference) error
	Forget(*Locked, string, ArchiveReference) error
}
type CollectionItem struct {
	Owner    artifactpath.StorageOwner     `json:"owner"`
	Bucket   artifactpath.RetentionClass   `json:"bucket"`
	Decision RetentionDecision             `json:"decision"`
	Bytes    int64                         `json:"bytes"`
	Members  []artifactpath.ArtifactMember `json:"members"`
	Archives []ArchiveReference            `json:"archives,omitempty"`
}
type CollectionReport struct {
	Items             []CollectionItem `json:"items"`
	Unknown           []string         `json:"unknown,omitempty"`
	Complete          bool             `json:"complete"`
	MigrationComplete bool             `json:"migration_complete"`
	BlockReason       string           `json:"block_reason,omitempty"`
	CollectedBytes    int64            `json:"collected_bytes"`
	Collected         int              `json:"collected"`
	NextOwner         string           `json:"next_owner,omitempty"`
	BatchComplete     bool             `json:"batch_complete"`
	OwnersProcessed   int              `json:"owners_processed"`
}
type Collector struct {
	Coordinator   *Coordinator
	Agents        []string
	References    ReferenceStore
	Legacy        func(context.Context, artifactpath.StorageOwner) (Liveness, error)
	LegacyRoot    func(context.Context, []ProcessIdentity) (Liveness, error)
	MaxEntries    int
	ExcludedPaths []string
	Fault         func(string) error
	Rename        func(string, string) error
	CleanupOwner  func(*Locked, artifactpath.StorageOwner) error
}

func (c *Collector) Preview(ctx context.Context) (report CollectionReport, err error) {
	if c.Coordinator == nil {
		return report, errors.New("collector requires coordinator")
	}
	err = c.Coordinator.WithReadLock(ctx, func(l *Locked) error {
		var e error
		report, e = c.snapshot(ctx, l)
		return e
	})
	return
}

// DiagnosticInventory discovers exact debugging paths without evaluating
// session clocks, leases or process liveness. Each diagnostic page proves its
// own writer and generation safety at collection time.
func (c *Collector) DiagnosticInventory(ctx context.Context) (report CollectionReport, err error) {
	if c.Coordinator == nil {
		return report, errors.New("collector requires coordinator")
	}
	err = c.Coordinator.TryWithReadLock(ctx, func(l *Locked) error {
		var e error
		report, e = c.snapshotWithInventory(ctx, l, "", 0, nil, diagnosticPathsOnly)
		return e
	})
	return
}
func (c *Collector) snapshot(ctx context.Context, held *Locked) (CollectionReport, error) {
	return c.snapshotPage(ctx, held, "", 0)
}

type snapshotPurpose uint8

const (
	sessionDecisions snapshotPurpose = iota
	diagnosticPathsOnly
)

// inventorySnapshot exists only within one uninterrupted root-lock callback.
// Metadata onboarding changes clocks, never the discovered payload paths.
type inventorySnapshot struct {
	ready     bool
	inventory RootInventory
	managed   []ProcessIdentity
}

func (c *Collector) snapshotPage(ctx context.Context, held *Locked, after string, ownerLimit int) (CollectionReport, error) {
	return c.snapshotWithInventory(ctx, held, after, ownerLimit, nil, sessionDecisions)
}

func (c *Collector) snapshotWithInventory(ctx context.Context, held *Locked, after string, ownerLimit int, cache *inventorySnapshot, purpose snapshotPurpose) (CollectionReport, error) {
	r := CollectionReport{BatchComplete: true}
	registry, err := c.Coordinator.ReadRegistry()
	referencesComplete := err == nil
	if err != nil {
		r.BlockReason = "Couch store inventory unavailable: " + err.Error()
	}
	r.MigrationComplete = err == nil && registry.MigrationComplete
	refs := References{}
	if err == nil && len(registry.Stores) > 0 {
		if c.References == nil {
			referencesComplete = false
			r.BlockReason = "Couch reference reader unavailable"
		} else {
			refs, err = c.References.Snapshot(ctx, held, registry.Stores)
			if err != nil {
				referencesComplete = false
				r.BlockReason = "Couch references unavailable: " + err.Error()
			}
		}
	}
	var inventory RootInventory
	var managed []ProcessIdentity
	if cache != nil && cache.ready {
		inventory, managed = cache.inventory, cache.managed
	} else {
		known := append([]artifactpath.StorageOwner(nil), refs.Visible...)
		for _, a := range refs.Archives {
			known = append(known, a.Owner)
		}
		pendingLeft := ownerLimit
		if pendingLeft <= 0 {
			pendingLeft = 100
		}
		var recoverPending func(string, string) error
		if held.Writable(c.Coordinator.Root) {
			recoverPending = func(dir, name string) error {
				if pendingLeft == 0 {
					return ErrMaintenanceYield
				}
				pendingLeft--
				return held.RemovePendingMetadata(ctx, dir, name)
			}
		}
		// Metadata owners keep otherwise anchorless stored artifacts discoverable.
		dir := filepath.Join(c.Coordinator.Root, ".retention", "owners")
		limit := c.MaxEntries
		if limit == 0 {
			limit = 100000
		}
		entries, e := boundedOwnerEntries(dir, limit)
		if errors.Is(e, errMetadataBudget) {
			r.BlockReason = e.Error()
			return r, nil
		}
		if e != nil {
			return r, e
		}
		for _, entry := range entries {
			if ctx.Err() != nil {
				return r, ctx.Err()
			}
			if isPendingMetadata(entry.Name()) {
				if recoverPending != nil {
					if err := recoverPending(dir, entry.Name()); err != nil {
						return r, err
					}
				}
				continue
			}
			if entry.IsDir() {
				return r, errors.New("unexpected activity directory")
			}
			var s OwnerState
			if e := readStateJSON(filepath.Join(dir, entry.Name()), &s); e != nil {
				return r, e
			}
			if _, e := c.Coordinator.ReadOwner(s.Activity.Owner); e != nil {
				return r, e
			}
			if c.Coordinator.statePath(s.Activity.Owner) != filepath.Join(dir, entry.Name()) {
				return r, errors.New("activity filename does not match owner")
			}
			known = append(known, s.Activity.Owner)
			for _, p := range s.Processes {
				managed = append(managed, p.Process)
			}
		}
		excluded := append(append([]string(nil), registry.Stores...), c.ExcludedPaths...)
		inventory, e = inventoryRootContext(ctx, c.Coordinator.Root, known, c.Agents, limit, excluded, recoverPending)
		if e != nil {
			return r, e
		}

		if cache != nil {
			cache.ready = true
			cache.inventory = inventory
			cache.managed = managed
		}
	}
	r.Complete = inventory.Complete
	r.Unknown = inventory.Unknown
	if purpose == diagnosticPathsOnly {
		for _, group := range inventory.Groups {
			item := CollectionItem{Owner: group.Owner, Bucket: artifactpath.DebugRetention}
			for _, member := range group.Members {
				if err := ctx.Err(); err != nil {
					return r, err
				}
				if member.Retention == artifactpath.DebugRetention {
					item.Members = append(item.Members, member)
				}
			}
			if len(item.Members) > 0 {
				r.Items = append(r.Items, item)
			}
		}
		return r, nil
	}
	rootLegacy := ProcessUnknown
	if c.LegacyRoot != nil {
		var err error
		rootLegacy, err = c.LegacyRoot(ctx, managed)
		if err != nil {
			rootLegacy = ProcessUnknown
		}
	}
	for _, group := range inventory.Groups {
		if group.Owner.Key() <= after {
			continue
		}
		if ownerLimit > 0 && r.OwnersProcessed >= ownerLimit {
			r.BatchComplete = false
			break
		}
		r.OwnersProcessed++
		r.NextOwner = group.Owner.Key()
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
		owner := group.Owner
		evidence := Evidence{Owner: owner, Complete: inventory.Complete && referencesComplete, Live: ProcessUnknown}
		if len(group.Blockers) > 0 {
			evidence.BlockReason = group.Blockers[0]
		}
		if err := held.pendingOwnerTransaction(owner); err != nil {
			evidence.BlockReason = err.Error()
		}
		state, e := c.Coordinator.ReadOwner(owner)
		if e == nil {
			evidence.Activity = &state.Activity
		} else if !errors.Is(e, os.ErrNotExist) {
			evidence.BlockReason = e.Error()
		}
		if len(state.Intents) > 0 || len(state.Starts) > 0 {
			evidence.BlockReason = "pending use or launch handoff"
		}
		for _, v := range refs.Visible {
			if v == owner {
				evidence.Protected = true
			}
		}
		var archives []ArchiveReference
		for _, a := range refs.Archives {
			if a.Owner == owner {
				archives = append(archives, a)
				evidence.ArchiveTimes = append(evidence.ArchiveTimes, a.ArchivedAt)
				if a.ClockError != "" {
					evidence.BlockReason = a.ClockError
				}
			}
		}
		legacy := rootLegacy
		if c.Legacy != nil {
			var e error
			legacy, e = c.Legacy(ctx, owner)
			if e != nil {
				legacy = ProcessUnknown
			}
		}
		evidence.Live = legacy
		for _, p := range state.Processes {
			live := ProcessUnknown
			if c.Coordinator.Probe != nil {
				live = c.Coordinator.Probe.Inspect(p.Process)
			}
			if live == ProcessAlive {
				evidence.Live = ProcessAlive
				break
			}
			if live == ProcessUnknown {
				evidence.Live = ProcessUnknown
			}
		}
		session := CollectionItem{Owner: owner, Bucket: artifactpath.SessionRetention, Archives: archives}
		captures := map[string][]artifactpath.ArtifactMember{}
		for _, m := range group.Members {
			if err := ctx.Err(); err != nil {
				return r, err
			}
			switch m.Retention {
			case artifactpath.CaptureRetention:
				cap, e := artifactpath.ParseParkedCapture(m, time.FixedZone("legacy-latest", -14*60*60))
				if e != nil {
					r.Items = append(r.Items, CollectionItem{Owner: owner, Bucket: m.Retention, Members: []artifactpath.ArtifactMember{m}, Decision: RetentionDecision{State: Blocked, Reason: e.Error()}})
					continue
				}
				captures[cap.Raw] = append(captures[cap.Raw], m)
			case artifactpath.DebugRetention:
				// Managed rotated diagnostics use diagnosticlog's per-generation collector.
				// This row reports the historical current file without authorizing unlink.
				item := CollectionItem{Owner: owner, Bucket: m.Retention, Members: []artifactpath.ArtifactMember{m}, Decision: RetentionDecision{State: Blocked, Reason: "debugging generation requires coordinated writer collection"}}
				item.Bytes, _ = memberBytesContext(ctx, item.Members)
				r.Items = append(r.Items, item)
			default:
				session.Members = append(session.Members, m)
			}
		}
		session.Decision = Decide(c.Coordinator.Now(), evidence)
		session.Bytes, e = memberBytesContext(ctx, session.Members)
		if e != nil {
			session.Decision = RetentionDecision{State: Blocked, Reason: e.Error()}
		}
		if len(session.Members) > 0 || len(session.Archives) > 0 || evidence.Activity != nil {
			r.Items = append(r.Items, session)
		}
		for raw, members := range captures {
			if err := ctx.Err(); err != nil {
				return r, err
			}
			ce := CaptureEvidence{Complete: evidence.Complete, Reader: legacy, BlockReason: evidence.BlockReason}
			hasRaw := false
			for _, m := range members {
				cap, e := artifactpath.ParseParkedCapture(m, time.FixedZone("legacy-latest", -14*60*60))
				if e != nil {
					ce.BlockReason = e.Error()
					continue
				}
				if cap.CapturedAt.After(ce.CapturedAt) {
					ce.CapturedAt = cap.CapturedAt
				}
				st, e := os.Lstat(m.Path)
				if e != nil || !st.Mode().IsRegular() {
					ce.BlockReason = "capture file unavailable or unsafe"
					continue
				}
				if st.ModTime().After(ce.CapturedAt) {
					ce.CapturedAt = st.ModTime()
				}
				if m.Path == raw {
					hasRaw = true
				}
			}
			if !hasRaw {
				ce.BlockReason = "capture events have no raw file"
			}
			capture, parseErr := artifactpath.ParseParkedCapture(members[0], time.UTC)
			if parseErr == nil {
				at, present, metadataErr := capturePublicationTime(capture)
				if metadataErr != nil {
					ce.BlockReason = metadataErr.Error()
				} else if present {
					ce.CapturedAt = at
				}
			}
			for _, p := range state.Processes {
				if p.Target != "" && p.Target != raw {
					continue
				}
				if p.Target == "" && knownNonCaptureRole(p.Role) {
					continue
				}
				live := ProcessUnknown
				if c.Coordinator.Probe != nil {
					live = c.Coordinator.Probe.Inspect(p.Process)
				}
				if live == ProcessAlive {
					ce.Reader = ProcessAlive
					break
				}
				if live == ProcessUnknown {
					ce.Reader = ProcessUnknown
				}
			}
			item := CollectionItem{Owner: owner, Bucket: artifactpath.CaptureRetention, Members: members, Decision: DecideCapture(c.Coordinator.Now(), ce)}
			item.Bytes, e = memberBytesContext(ctx, members)
			if e != nil {
				item.Decision = RetentionDecision{State: Blocked, Reason: e.Error()}
			}
			r.Items = append(r.Items, item)
		}
	}
	sort.SliceStable(r.Items, func(i, j int) bool {
		a, b := r.Items[i], r.Items[j]
		if a.Owner.Key() != b.Owner.Key() {
			return a.Owner.Key() < b.Owner.Key()
		}
		if a.Bucket != b.Bucket {
			return a.Bucket < b.Bucket
		}
		return firstPath(a) < firstPath(b)
	})
	return r, nil
}
func firstPath(i CollectionItem) string {
	if len(i.Members) == 0 {
		return ""
	}
	return i.Members[0].Path
}
func memberBytesContext(ctx context.Context, members []artifactpath.ArtifactMember) (int64, error) {
	var total int64
	for _, m := range members {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		st, e := os.Lstat(m.Path)
		if e != nil {
			return 0, e
		}
		if st.Mode()&os.ModeSymlink != 0 || (!st.IsDir() && !st.Mode().IsRegular()) {
			return 0, fmt.Errorf("unsafe artifact %s", m.Path)
		}
		if st.Mode().IsRegular() {
			total += st.Size()
		}
	}
	return total, nil
}
func knownNonCaptureRole(role string) bool {
	switch role {
	case "wrapper", "draft-editor", "title-poller", "session-watch", "launcher", "prompt-log-writer", "scrollback-opener", "changelog-opener", "changelog-render", "scrollback-reader", "writer":
		return true
	}
	return false
}

// Apply initializes missing session clocks even before inventory migration is
// acknowledged. Payload deletion requires a complete registered-store inventory.
func (c *Collector) Apply(ctx context.Context, limit int) (CollectionReport, error) {
	return c.applyPage(ctx, "", limit, false)
}

// ApplyPage advances by visited owner, including retained and newly initialized
// owners. Optional maintenance never waits for the foreground coordinator.
func (c *Collector) ApplyPage(ctx context.Context, after string, limit int) (CollectionReport, error) {
	return c.applyPage(ctx, after, limit, true)
}

func (c *Collector) applyPage(ctx context.Context, after string, limit int, optional bool) (report CollectionReport, err error) {
	if c.Coordinator == nil || limit <= 0 {
		return report, errors.New("collector and positive work limit required")
	}
	ownerLimit := 0 // Explicit apply scans all owners; limit bounds collections.
	lock := c.Coordinator.WithLock
	if optional {
		ownerLimit = limit
		lock = c.Coordinator.TryWithLock
	}
	err = lock(ctx, func(held *Locked) error {
		if _, err := held.RecoverPendingMetadata(ctx, c.Coordinator.PendingMetadataDir(), limit); err != nil {
			return err
		}
		registry, registryErr := c.Coordinator.ReadRegistry()
		if registryErr == nil {
			if recovery, ok := c.References.(interface {
				Recover(context.Context, *Locked, []string) error
			}); ok {
				if err := recovery.Recover(ctx, held, registry.Stores); err != nil {
					return err
				}
			}
		}
		if err := c.recoverTransactions(held, limit); err != nil {
			return err
		}
		cache := &inventorySnapshot{}
		initial, err := c.snapshotWithInventory(ctx, held, after, ownerLimit, cache, sessionDecisions)
		if err != nil {
			return err
		}
		seen := map[artifactpath.StorageOwner]bool{}
		var owners []artifactpath.StorageOwner
		for _, item := range initial.Items {
			if !seen[item.Owner] {
				seen[item.Owner] = true
				owners = append(owners, item.Owner)
			}
		}
		if onboarding, ok := c.References.(interface {
			Onboard(context.Context, *Locked, []string, []artifactpath.StorageOwner) error
		}); ok && registryErr == nil {
			if err := onboarding.Onboard(ctx, held, registry.Stores, owners); err != nil {
				return err
			}
		}
		seen = map[artifactpath.StorageOwner]bool{}
		for _, item := range initial.Items {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if seen[item.Owner] {
				continue
			}
			seen[item.Owner] = true
			state, err := c.Coordinator.ReadOwner(item.Owner)
			if errors.Is(err, os.ErrNotExist) {
				state, err = held.load(item.Owner)
				if err == nil {
					err = held.save(state)
				}
			} else if err == nil {
				err = held.recoverUse(item.Owner)
			}
			if err != nil {
				return err
			}
		}
		report, err = c.snapshotWithInventory(ctx, held, after, ownerLimit, cache, sessionDecisions)
		if err != nil {
			return err
		}
		if !report.MigrationComplete || !report.Complete {
			return nil
		}
		for _, item := range report.Items {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if report.Collected >= limit {
				break
			}
			if item.Decision.State != Eligible {
				continue
			}
			if err := c.collectItem(held, item); err != nil {
				return err
			}
			report.Collected++
			report.CollectedBytes += item.Bytes
		}
		return nil
	})
	return
}

var errMetadataBudget = errors.New("activity metadata exceeds inventory budget; collection retained")

func boundedOwnerEntries(dir string, limit int) ([]os.DirEntry, error) {
	f, err := os.Open(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(limit + 1)
	if len(entries) > limit {
		return nil, errMetadataBudget
	}
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return entries, err
}
