package storagegc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

type CollectionEntry struct {
	Source      string             `json:"source"`
	Destination string             `json:"destination"`
	Top         bool               `json:"top"`
	Identity    CollectionIdentity `json:"identity"`
}
type CollectionIdentity struct {
	Device   uint64 `json:"device"`
	Inode    uint64 `json:"inode"`
	Mode     uint32 `json:"mode"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
}

func collectionIdentity(path string) (CollectionIdentity, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return CollectionIdentity{}, err
	}
	if !st.IsDir() && !st.Mode().IsRegular() {
		return CollectionIdentity{}, fmt.Errorf("unsafe collection member %s", path)
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return CollectionIdentity{}, errors.New("file identity unavailable")
	}
	return CollectionIdentity{uint64(sys.Dev), uint64(sys.Ino), uint32(st.Mode()), st.Size(), st.ModTime().UnixNano()}, nil
}
func (i CollectionIdentity) directory() bool { return os.FileMode(i.Mode).IsDir() }
func (i CollectionIdentity) matches(other CollectionIdentity) bool {
	return i.Device == other.Device && i.Inode == other.Inode && i.Mode == other.Mode && (i.directory() || (i.Size == other.Size && i.Modified == other.Modified))
}
func safeRelative(path string) bool {
	return path != "" && path != "." && !filepath.IsAbs(path) && filepath.Clean(path) == path && path != ".." && !strings.HasPrefix(path, ".."+string(os.PathSeparator))
}
func checkCollectionParents(root, relative string) error {
	if !safeRelative(relative) {
		return errors.New("unsafe collection path")
	}
	if err := checkDirectory(root, false); err != nil {
		return err
	}
	dir := root
	for _, part := range strings.Split(filepath.Dir(relative), string(os.PathSeparator)) {
		if part == "." {
			continue
		}
		dir = filepath.Join(dir, part)
		if err := checkDirectory(dir, false); err != nil {
			return err
		}
	}
	return nil
}
func (c *Collector) transactionDir() string {
	return filepath.Join(c.Coordinator.Root, ".retention", "transactions")
}
func (c *Collector) quarantine(t CollectionTransaction) string {
	return filepath.Join(c.Coordinator.Root, ".retention", "quarantine", t.ID)
}
func (c *Collector) transactionPath(t CollectionTransaction) string {
	return filepath.Join(c.transactionDir(), t.ID+".json")
}
func (c *Collector) fault(step string) error {
	if c.Fault != nil {
		return c.Fault(step)
	}
	return nil
}
func (c *Collector) prepareCollection(held *Locked, item CollectionItem) (CollectionTransaction, error) {
	t := CollectionTransaction{Version: 1, Owner: item.Owner, Bucket: item.Bucket, Archives: append([]ArchiveReference(nil), item.Archives...), Phase: "prepared"}
	if !held.Writable(c.Coordinator.Root) {
		return t, errors.New("collection requires writable root lock")
	}
	if err := c.Coordinator.validateOwner(item.Owner); err != nil {
		return t, err
	}
	if item.Bucket != artifactpath.SessionRetention && item.Bucket != artifactpath.CaptureRetention {
		return t, errors.New("unsupported collection bucket")
	}
	if err := held.pendingOwnerTransaction(item.Owner); err != nil {
		return t, err
	}
	state, err := c.Coordinator.ReadOwner(item.Owner)
	if err != nil {
		return t, err
	}
	t.Incarnation = state.Activity.Incarnation
	t.ID, err = randomID()
	if err != nil {
		return t, err
	}
	index, err := artifactpath.NewMatchIndex([]artifactpath.StorageOwner{item.Owner}, c.Agents)
	if err != nil {
		return t, err
	}
	members := map[string]artifactpath.ArtifactMember{}
	for _, m := range item.Members {
		actual, e := index.Match(m.Path)
		if e != nil || actual != m || m.Owner != item.Owner || m.Retention != item.Bucket {
			return t, fmt.Errorf("invalid collection authority: %s", m.Path)
		}
		if _, ok := members[m.Path]; ok {
			return t, errors.New("duplicate collection member")
		}
		members[m.Path] = m
	}
	paths := make([]string, 0, len(members))
	for p := range members {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	topPaths := []string{}
	for _, path := range paths {
		if err := held.CheckContext(); err != nil {
			return t, err
		}
		rel, e := filepath.Rel(c.Coordinator.Root, path)
		if e != nil {
			return t, e
		}
		if e = checkCollectionParents(c.Coordinator.Root, rel); e != nil {
			return t, e
		}
		identity, e := collectionIdentity(path)
		if e != nil {
			return t, e
		}
		if identity.directory() != members[path].Directory {
			return t, errors.New("collection member changed type")
		}
		top := -1
		for i, p := range topPaths {
			if strings.HasPrefix(path, p+string(os.PathSeparator)) {
				top = i
				break
			}
		}
		isTop := top < 0
		if isTop {
			top = len(topPaths)
			topPaths = append(topPaths, path)
		}
		destination := strconv.Itoa(top)
		if !isTop {
			child, _ := filepath.Rel(topPaths[top], path)
			destination = filepath.Join(destination, child)
		}
		t.Entries = append(t.Entries, CollectionEntry{rel, destination, isTop, identity})
	}
	for _, path := range topPaths {
		if err := held.CheckContext(); err != nil {
			return t, err
		}
		if members[path].Directory {
			err = filepath.WalkDir(path, func(p string, d fs.DirEntry, e error) error {
				if err := held.CheckContext(); err != nil {
					return err
				}
				if e != nil {
					return e
				}
				if _, ok := members[p]; !ok {
					return fmt.Errorf("unrecorded collection descendant: %s", p)
				}
				return nil
			})
			if err != nil {
				return t, err
			}
		}
	}
	if len(t.Entries) == 0 && len(t.Archives) == 0 &&
		(t.Bucket != artifactpath.SessionRetention || item.Decision.State != Eligible) {
		return t, errors.New("empty collection requires eligible session retirement")
	}
	for _, a := range t.Archives {
		if a.Owner != t.Owner || t.Bucket != artifactpath.SessionRetention {
			return t, errors.New("archive belongs to different collection")
		}
	}
	return t, nil
}
func (c *Collector) collectItem(held *Locked, item CollectionItem) error {
	if err := held.CheckContext(); err != nil {
		return err
	}
	t, err := c.prepareCollection(held, item)
	if err != nil {
		return err
	}
	// Only the shared journal directory may precede publication. Unique
	// quarantine state must always have durable transaction authority.
	for _, dir := range []string{c.transactionDir()} {
		if err := held.CheckContext(); err != nil {
			return err
		}
		if err := checkDirectory(dir, true); err != nil {
			return err
		}
		if err := c.Coordinator.syncDirectory(filepath.Dir(dir)); err != nil {
			return err
		}
	}
	if err := held.CheckContext(); err != nil {
		return err
	}
	if err := held.atomicJSON(c.transactionPath(t), t); err != nil {
		return err
	}
	if err := c.fault("journal"); err != nil {
		return err
	}
	return c.resumeCollection(held, &t)
}
func (c *Collector) validateTransaction(t CollectionTransaction) error {
	if t.Version != 1 || len(t.ID) != 32 || strings.Trim(t.ID, "0123456789abcdef") != "" || t.Incarnation == "" {
		return errors.New("invalid collection transaction")
	}
	if err := c.Coordinator.validateOwner(t.Owner); err != nil {
		return err
	}
	if !validCollectionPhase(t.Phase) {
		return errors.New("invalid collection phase")
	}
	if t.Bucket != artifactpath.SessionRetention && t.Bucket != artifactpath.CaptureRetention {
		return errors.New("invalid collection bucket")
	}
	if t.Bucket == artifactpath.CaptureRetention && len(t.Entries) == 0 {
		return errors.New("capture collection requires payload entries")
	}
	index, err := artifactpath.NewMatchIndex([]artifactpath.StorageOwner{t.Owner}, c.Agents)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	dest := map[string]bool{}
	tops := map[string]string{}
	for _, e := range t.Entries {
		if !safeRelative(e.Source) || !safeRelative(e.Destination) || seen[e.Source] || dest[e.Destination] {
			return errors.New("invalid transaction member path")
		}
		seen[e.Source] = true
		dest[e.Destination] = true
		m, err := index.Match(filepath.Join(c.Coordinator.Root, e.Source))
		if err != nil || m.Retention != t.Bucket || m.Directory != e.Identity.directory() {
			return errors.New("transaction authority mismatch")
		}
		if e.Top {
			if filepath.Base(e.Destination) != e.Destination {
				return errors.New("invalid top destination")
			}
			tops[e.Destination] = e.Source
		}
	}
	for _, e := range t.Entries {
		head := strings.Split(e.Destination, string(os.PathSeparator))[0]
		source, ok := tops[head]
		if !ok {
			return errors.New("missing transaction top")
		}
		suffix := strings.TrimPrefix(e.Destination, head)
		if e.Source != source+suffix {
			return errors.New("transaction destination mismatch")
		}
	}
	for _, a := range t.Archives {
		if a.Owner != t.Owner || t.Bucket != artifactpath.SessionRetention {
			return errors.New("invalid transaction archive")
		}
	}
	return nil
}

// verifyTree checks the complete current destination subtree against the frozen
// inventory. During deletion recorded missing leaves are expected; new children
// and substitutions are always errors.
func (c *Collector) verifyTree(held *Locked, t CollectionTransaction, missing bool) error {
	if err := held.CheckContext(); err != nil {
		return err
	}
	root := c.quarantine(t)
	if err := checkDirectory(root, false); err != nil {
		if missing && errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	entries := map[string]CollectionEntry{}
	for _, e := range t.Entries {
		entries[e.Destination] = e
	}
	seen := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err := held.CheckContext(); err != nil {
			return err
		}
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		e, ok := entries[rel]
		seen[rel] = true
		if !ok {
			return fmt.Errorf("unexpected quarantine child: %s", path)
		}
		actual, err := collectionIdentity(path)
		if err != nil {
			return err
		}
		if !e.Identity.matches(actual) {
			return fmt.Errorf("quarantine identity changed: %s", path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !missing && len(seen) != len(entries) {
		return errors.New("quarantine member disappeared")
	}
	return nil
}
func (c *Collector) resumeCollection(held *Locked, t *CollectionTransaction) error {
	if err := held.CheckContext(); err != nil {
		return err
	}
	if !held.Writable(c.Coordinator.Root) {
		return errors.New("collection requires writable root lock")
	}
	if err := c.validateTransaction(*t); err != nil {
		return err
	}
	if t.Phase == "prepared" {
		// A killed publisher may leave an authoritative prepared journal before
		// creating any quarantine directories. Recovery creates them only after
		// validating the frozen transaction, and before touching source names.
		for _, dir := range []string{filepath.Dir(c.quarantine(*t)), c.quarantine(*t)} {
			if err := held.CheckContext(); err != nil {
				return err
			}
			if err := checkDirectory(dir, true); err != nil {
				return err
			}
			if err := c.Coordinator.syncDirectory(filepath.Dir(dir)); err != nil {
				return err
			}
		}
		if err := c.fault("quarantine"); err != nil {
			return err
		}
	} else if err := checkDirectory(filepath.Dir(c.quarantine(*t)), false); err != nil {
		return err
	}
	if t.Phase == "prepared" {
		if len(t.Archives) > 0 && c.References == nil {
			return errors.New("archive reference writer unavailable")
		}
		for i, a := range t.Archives {
			if err := held.CheckContext(); err != nil {
				return err
			}
			if err := c.References.Detach(held, t.ID, a); err != nil {
				return err
			}
			if err := c.fault("archive:" + strconv.Itoa(i)); err != nil {
				return err
			}
		}
		for i, e := range t.Entries {
			if err := held.CheckContext(); err != nil {
				return err
			}
			if !e.Top {
				continue
			}
			src := filepath.Join(c.Coordinator.Root, e.Source)
			dst := filepath.Join(c.quarantine(*t), e.Destination)
			if err := checkCollectionParents(c.quarantine(*t), e.Destination); err != nil {
				return err
			}
			existing, err := collectionIdentity(dst)
			if err == nil {
				if !e.Identity.matches(existing) {
					return errors.New("quarantine destination identity mismatch")
				}
				continue
			}
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if err := checkCollectionParents(c.Coordinator.Root, e.Source); err != nil {
				return err
			}
			actual, err := collectionIdentity(src)
			if err != nil {
				return err
			}
			if !e.Identity.matches(actual) {
				return errors.New("source identity changed before collection")
			}
			// Recheck every descendant immediately before moving its top directory.
			if e.Identity.directory() {
				if err := c.verifySourceTree(held, *t, e); err != nil {
					return err
				}
			}
			rename := c.Rename
			if rename == nil {
				rename = os.Rename
			}
			if err := held.CheckContext(); err != nil {
				return err
			}
			if err := rename(src, dst); err != nil {
				return err
			}
			if err := c.Coordinator.syncDirectory(filepath.Dir(src)); err != nil {
				return err
			}
			if err := c.Coordinator.syncDirectory(filepath.Dir(dst)); err != nil {
				return err
			}
			if err := c.fault("rename:" + strconv.Itoa(i)); err != nil {
				return err
			}
		}
		if err := c.verifyTree(held, *t, false); err != nil {
			return err
		}
		if err := c.advanceTransaction(held, t, CollectionDetachmentProved); err != nil {
			return err
		}
		if err := c.fault("detached"); err != nil {
			return err
		}
	}
	if t.Phase == "detached" {
		if t.Bucket == artifactpath.SessionRetention {
			state, err := c.Coordinator.ReadOwner(t.Owner)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			// A newer incarnation also owns the shared bindings. Recovery can
			// finish deleting the detached bytes without touching those bindings.
			if errors.Is(err, os.ErrNotExist) || state.Activity.Incarnation == t.Incarnation {
				if err := held.CheckContext(); err != nil {
					return err
				}
				if c.CleanupOwner != nil {
					if e := c.CleanupOwner(held, t.Owner); e != nil {
						return e
					}
				}
				if e := c.fault("cleanup"); e != nil {
					return e
				}
			}

			if err == nil && state.Activity.Incarnation == t.Incarnation {
				if err := held.CheckContext(); err != nil {
					return err
				}
				if err := os.Remove(c.Coordinator.statePath(t.Owner)); err != nil {
					return err
				}
				if err := c.Coordinator.syncDirectory(filepath.Dir(c.Coordinator.statePath(t.Owner))); err != nil {
					return err
				}
			}
		}
		if err := c.fault("retired"); err != nil {
			return err
		}
		if err := c.advanceTransaction(held, t, CollectionOwnerRetired); err != nil {
			return err
		}
		if err := c.fault("finalized"); err != nil {
			return err
		}
	}
	// Finalization is durable before receipts can disappear; replay must never
	// issue Detach against a newer same-address Couch archive.
	if len(t.Archives) > 0 && c.References == nil {
		return errors.New("archive receipt writer unavailable")
	}
	for i, a := range t.Archives {
		if err := held.CheckContext(); err != nil {
			return err
		}
		if err := c.References.Forget(held, t.ID, a); err != nil {
			return err
		}
		if err := c.fault("forget:" + strconv.Itoa(i)); err != nil {
			return err
		}
	}
	if err := c.verifyTree(held, *t, true); err != nil {
		return err
	}
	entries := append([]CollectionEntry(nil), t.Entries...)
	sort.Slice(entries, func(i, j int) bool { return len(entries[i].Destination) > len(entries[j].Destination) })
	for i, e := range entries {
		if err := held.CheckContext(); err != nil {
			return err
		}
		path := filepath.Join(c.quarantine(*t), e.Destination)
		if err := checkCollectionParents(c.quarantine(*t), e.Destination); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		actual, err := collectionIdentity(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !e.Identity.matches(actual) {
			return errors.New("quarantine changed during deletion")
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		if err := c.Coordinator.syncDirectory(filepath.Dir(path)); err != nil {
			return err
		}
		if err := c.fault("remove:" + strconv.Itoa(i)); err != nil {
			return err
		}
	}
	if err := held.CheckContext(); err != nil {
		return err
	}
	if err := os.Remove(c.quarantine(*t)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := c.Coordinator.syncDirectory(filepath.Dir(c.quarantine(*t))); err != nil {
		return err
	}
	if err := c.fault("forgotten"); err != nil {
		return err
	}
	if err := held.CheckContext(); err != nil {
		return err
	}
	if err := os.Remove(c.transactionPath(*t)); err != nil {
		return err
	}
	return c.Coordinator.syncDirectory(c.transactionDir())
}
func (c *Collector) verifySourceTree(held *Locked, t CollectionTransaction, top CollectionEntry) error {
	if err := held.CheckContext(); err != nil {
		return err
	}
	entries := map[string]CollectionEntry{}
	for _, e := range t.Entries {
		if e.Source == top.Source || strings.HasPrefix(e.Source, top.Source+string(os.PathSeparator)) {
			entries[e.Source] = e
		}
	}
	seen := 0
	err := filepath.WalkDir(filepath.Join(c.Coordinator.Root, top.Source), func(path string, d fs.DirEntry, err error) error {
		if err := held.CheckContext(); err != nil {
			return err
		}
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(c.Coordinator.Root, path)
		e, ok := entries[rel]
		if !ok {
			return errors.New("new source descendant")
		}
		seen++
		actual, err := collectionIdentity(path)
		if err != nil {
			return err
		}
		if !e.Identity.matches(actual) {
			return errors.New("source descendant changed")
		}
		return nil
	})
	if err != nil {
		return err
	}
	if seen != len(entries) {
		return errors.New("source descendant disappeared")
	}
	return nil
}

// ErrMaintenanceYield asks optional work to resume on its next scheduled pass.
var ErrMaintenanceYield = errors.New("maintenance work budget exhausted")

func (c *Collector) recoverTransactions(held *Locked, limit int) error {
	if !held.Writable(c.Coordinator.Root) {
		return errors.New("transaction recovery requires writable root lock")
	}
	if err := checkDirectory(c.transactionDir(), false); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	entries, err := os.ReadDir(c.transactionDir())
	if err != nil {
		return err
	}
	if limit <= 0 {
		limit = 100
	}
	for i, entry := range entries {
		if err := held.CheckContext(); err != nil {
			return err
		}
		if i >= limit {
			return ErrMaintenanceYield
		}
		if isPendingMetadata(entry.Name()) {
			if err := held.RemovePendingMetadata(held.ctx, c.transactionDir(), entry.Name()); err != nil {
				return err
			}
			continue
		}
		var t CollectionTransaction
		if err := readStateJSON(filepath.Join(c.transactionDir(), entry.Name()), &t); err != nil {
			return err
		}
		if entry.Name() != t.ID+".json" {
			return errors.New("transaction filename mismatch")
		}
		if err := c.resumeCollection(held, &t); err != nil {
			return err
		}
	}
	return nil
}
func (l *Locked) pendingOwnerTransaction(owner artifactpath.StorageOwner) error {
	if err := l.CheckContext(); err != nil {
		return err
	}
	dir := filepath.Join(l.coordinator.Root, ".retention", "transactions")
	if err := checkDirectory(dir, false); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := l.CheckContext(); err != nil {
			return err
		}
		if isPendingMetadata(entry.Name()) {
			continue
		}
		var t CollectionTransaction
		if err := readStateJSON(filepath.Join(dir, entry.Name()), &t); err != nil {
			return err
		}
		if t.Version != 1 || len(t.ID) != 32 || strings.Trim(t.ID, "0123456789abcdef") != "" || entry.Name() != t.ID+".json" || t.Incarnation == "" {
			return errors.New("malformed pending collection transaction")
		}
		if err := l.coordinator.validateOwner(t.Owner); err != nil {
			return err
		}
		if !validCollectionPhase(t.Phase) {
			return errors.New("invalid pending collection phase")
		}
		if t.Owner == owner {
			return errors.New("owner has pending collection transaction")
		}
	}
	return nil
}

// Advance only after the corresponding external effects have been proved.
// Publish before changing the in-memory state so a failed write can be retried.
func (c *Collector) advanceTransaction(held *Locked, current *CollectionTransaction, event CollectionEvent) error {
	if err := held.CheckContext(); err != nil {
		return err
	}
	next, err := ReduceTransaction(*current, event)
	if err != nil {
		return err
	}
	if err := held.atomicJSON(c.transactionPath(next), next); err != nil {
		return err
	}
	*current = next
	return nil
}
