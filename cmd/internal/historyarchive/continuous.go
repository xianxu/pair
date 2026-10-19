package historyarchive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

// ContinuousQuiet is how long a file stays unmodified before a continuous pass
// archives it. Snapshotted databases are taken daily regardless.
const ContinuousQuiet = 72 * time.Hour

// ErrPassRunning means another continuous pass holds the lock.
var ErrPassRunning = errors.New("a continuous archive pass is already running")

// Snapshotter takes a coherent private copy of one live database. Sweep drops
// private working copies a killed pass left behind.
type Snapshotter interface {
	Snapshot(ctx context.Context, root, rel, stagingRoot string) (ok bool, note string, err error)
	Sweep() error
}

// ContinuousPass archives every continuous source once (pair#592): quiet
// files, plus daily database snapshots, each file deduplicated by its source
// identity, path and content hash against the catalog. It never deletes and
// writes only under Store, SnapshotRoot and the configured sink and catalog.
type ContinuousPass struct {
	Archiver     Archiver
	Config       ContinuousConfig
	Store        string // lock and last-run marker
	SnapshotRoot string // private database snapshots, outside the catalog directory
	Host, Home   string
	Env          map[string]string
	Kinds        []string // empty: every continuous kind
	Snapshotter  Snapshotter
	Now          func() time.Time
	Quiet        time.Duration
	Log          io.Writer
	Drive        *DriveMirror // optional independent replication; production command injects it
}

// ContinuousReport counts what one pass did per kind.
type ContinuousReport struct {
	Archived map[string]int
	Skipped  []string
	Drive    *DriveReport
}

// LastRunName records the completion time of the last full pass.
const LastRunName = "continuous.last"

func (p ContinuousPass) Run(ctx context.Context) (ContinuousReport, error) {
	report := ContinuousReport{Archived: map[string]int{}}
	if err := os.MkdirAll(p.Store, 0o700); err != nil {
		return report, err
	}
	lock, err := os.OpenFile(filepath.Join(p.Store, "continuous.lock"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return report, err
	}
	defer lock.Close()
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return report, ErrPassRunning
	}
	// Private copies a killed pass left behind can be gigabytes; drop them
	// while holding the lock.
	if p.Snapshotter != nil {
		if err := p.Snapshotter.Sweep(); err != nil {
			return report, err
		}
	}
	now := p.now()
	quiet := p.Quiet
	if quiet == 0 {
		quiet = ContinuousQuiet
	}
	inv, err := Defaults()
	if err != nil {
		return report, err
	}
	roots, err := ContinuousRoots(ctx, inv, p.Config, p.Host, p.Home, p.Env)
	if err != nil {
		return report, err
	}
	index, err := p.archivedIndex(ctx)
	if err != nil {
		return report, err
	}
	// One failing source does not starve the others; the pass reports every
	// failure and is not marked complete.
	var failures []error
	for _, root := range roots {
		if !p.wants(root.Kind) {
			continue
		}
		if err := ctx.Err(); err != nil {
			return report, err
		}
		n, skipped, err := p.archiveRoot(ctx, root, index, now, quiet)
		report.Archived[root.Kind] += n
		report.Skipped = append(report.Skipped, skipped...)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s at %s: %w", root.Kind, root.Identity.Root, err))
		}
	}
	if len(failures) > 0 {
		return report, errors.Join(failures...)
	}
	if len(p.Kinds) == 0 {
		if err := os.WriteFile(filepath.Join(p.Store, LastRunName), []byte(now.UTC().Format(time.RFC3339)+"\n"), 0o600); err != nil {
			return report, err
		}
	}
	if p.Drive != nil {
		remote, remoteErr := p.Drive.Run(ctx, p.Config, p.Store)
		if remoteErr != nil {
			remote.Notes = append(remote.Notes, "Drive pending/unavailable: "+remoteErr.Error())
		}
		report.Drive = &remote
	}
	return report, nil
}

func (p ContinuousPass) archiveRoot(ctx context.Context, root SourceRoot, index *ArchivedIndex, now time.Time, quiet time.Duration) (int, []string, error) {
	var skipped []string
	identity := root.Identity
	found, err := Discover(ctx, identity.Root, root.Patterns)
	if err != nil {
		return 0, nil, err
	}
	collected := false
	if root.Snapshot {
		// Databases change continuously: snapshot daily instead of waiting
		// for quiet, and archive the private snapshot, keyed by its content.
		staging := filepath.Join(p.SnapshotRoot, root.Kind)
		var taken []string
		for _, c := range found {
			ok, note, err := p.Snapshotter.Snapshot(ctx, identity.Root, c.Path, staging)
			if err != nil {
				return 0, skipped, err
			}
			if !ok {
				skipped = append(skipped, note)
				continue
			}
			taken = append(taken, c.Path)
		}
		identity = SourceIdentity{Host: identity.Host, Root: staging, Repository: identity.Root}
		if found, err = Discover(ctx, staging, taken); err != nil {
			return 0, skipped, err
		}
		collected = true
	} else {
		found = SelectQuiet(found, now, quiet)
	}
	var fresh []string
	for _, c := range found {
		if index.Seen(identity, c) {
			continue
		}
		described, err := (&SelectedFiles{Root: identity.Root, Limit: rawBundleLimit(root.Kind)}).Describe(ctx, []string{c.Path})
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s/%s: %v", root.Kind, c.Path, err))
			continue
		}
		if index.SeenHash(identity, c.Path, described[0].SHA256) {
			continue
		}
		fresh = append(fresh, c.Path)
	}
	archived := 0
	for len(fresh) > 0 {
		n := min(len(fresh), MaxFiles)
		if collected {
			// One request per snapshot: an oversized one is skipped alone,
			// never taking the rest of the batch with it.
			n = 1
		}
		req := ArchiveRequest{Version: Version, Kind: root.Kind, Source: identity, Files: fresh[:n], Recipients: p.Config.Recipients, Sink: p.Config.Sink, Catalog: p.Config.Catalog}
		records, err := p.Archiver.create(ctx, req, collected)
		for _, r := range records {
			index.Add(r.Manifest)
			archived += len(r.Manifest.Files)
		}
		if collected && errors.Is(err, ErrCiphertextLimit) {
			// A snapshot too large even compressed is skipped, not a failed
			// pass: it would fail the same way every day (pair#616).
			skipped = append(skipped, fmt.Sprintf("%s/%s: %v; skipped", root.Kind, fresh[0], err))
			fresh = fresh[n:]
			continue
		}
		if err != nil {
			return archived, skipped, err
		}
		fresh = fresh[n:]
	}
	return archived, skipped, nil
}

func (p ContinuousPass) archivedIndex(ctx context.Context) (*ArchivedIndex, error) {
	index := NewArchivedIndex()
	catalog, err := OpenCatalog(ctx, p.Config.Catalog)
	if err != nil {
		return nil, err
	}
	defer catalog.Close()
	return index, catalog.Each(ctx, func(r CatalogRecord) error { index.Add(r.Manifest); return nil })
}

func (p ContinuousPass) wants(kind string) bool {
	if len(p.Kinds) == 0 {
		return true
	}
	for _, k := range p.Kinds {
		if k == kind {
			return true
		}
	}
	return false
}

func (p ContinuousPass) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}
