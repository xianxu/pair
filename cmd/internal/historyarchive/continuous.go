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
	Snapshot(ctx context.Context, root, rel, stagingRoot string) (SnapshotResult, error)
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
	Archived  map[string]int
	Skipped   []string
	Drive     *DriveReport
	Databases []DatabaseBackupSummary
}

// LastRunName records the completion time of the last full pass.
const LastRunName = "continuous.last"

func (p ContinuousPass) Run(ctx context.Context) (ContinuousReport, error) {
	report := ContinuousReport{Archived: map[string]int{}}
	inv, err := Defaults()
	if err != nil {
		return report, err
	}
	roots, err := ContinuousRoots(ctx, inv, p.Config, p.Host, p.Home, p.Env)
	if err != nil {
		return report, err
	}
	// Every snapshot output must be disjoint from every native source, not
	// just its own kind: a destination can alias another agent's home.
	var staging []string
	for _, root := range roots {
		if root.Snapshot {
			staging = append(staging, filepath.Join(p.SnapshotRoot, root.Kind))
		}
	}
	// Sweep can remove whole scratch directories. Check every native root,
	// including kinds excluded from this pass, before any store mutation.
	for _, root := range roots {
		switch snapshotter := p.Snapshotter.(type) {
		case SQLiteSnapshotter:
			err = snapshotter.validateRoots(root.Identity.Root, staging...)
		case *SQLiteSnapshotter:
			err = snapshotter.validateRoots(root.Identity.Root, staging...)
		}
		if err != nil {
			return report, err
		}
	}
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
	if p.Drive != nil {
		// Collect an interrupted pass's links even if local collection fails
		// before the optional Drive phase. Cleanup failure is remote-only.
		stage, cleanupErr := openUploadStaging(p.Store)
		if cleanupErr == nil {
			cleanupErr = errors.Join(SweepUploadStaging(context.WithoutCancel(ctx), stage), stage.Close())
		}
		if cleanupErr != nil {
			report.Drive = &DriveReport{Notes: []string{"staging cleanup: " + cleanupErr.Error()}}
		}
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
		n, skipped, err := p.archiveRoot(ctx, root, index, now, quiet, &report.Databases)
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
			remote.Notes = append(remote.Notes, "operation report: "+remoteErr.Error())
		}
		report.Drive = &remote
	}
	return report, nil
}

func (p ContinuousPass) archiveRoot(ctx context.Context, root SourceRoot, index *ArchivedIndex, now time.Time, quiet time.Duration, summaries *[]DatabaseBackupSummary) (int, []string, error) {
	var skipped []string
	identity := root.Identity
	found, err := Discover(ctx, identity.Root, root.Patterns)
	if err != nil {
		return 0, nil, err
	}
	collected := false
	rows := map[string]int{}
	var failures []error
	if root.Snapshot {
		staging := filepath.Join(p.SnapshotRoot, root.Kind)
		snapshotIdentity := SourceIdentity{Host: identity.Host, Root: staging, Repository: identity.Root}
		var taken []string
		for _, c := range found {
			row := DatabaseBackupSummary{Kind: root.Kind, Path: c.Path, Status: DatabaseFailed, LastVerified: index.LatestVerified(snapshotIdentity, c.Path)}
			result, err := p.Snapshotter.Snapshot(ctx, identity.Root, c.Path, staging)
			row.Detail = result.Detail
			if err != nil {
				row.Detail = err.Error()
				failures = append(failures, err)
			}
			switch {
			case err != nil:
			case result.Status == SnapshotReady:
				row.CheckedAt = p.now()
				row.Detail = "snapshot validated; publication pending"
				taken = append(taken, c.Path)
			case result.Status == SnapshotDeferred:
				row.Status = DatabaseDeferred
			}
			rows[c.Path] = len(*summaries)
			*summaries = append(*summaries, row)
			if ctx.Err() != nil {
				return 0, skipped, ctx.Err()
			}
			if err != nil || result.Status != SnapshotReady {
				skipped = append(skipped, c.Path+": "+row.Detail)
			}
		}
		identity = snapshotIdentity
		if found, err = Discover(ctx, staging, taken); err != nil {
			return 0, skipped, err
		}
		collected = true
	} else {
		found = SelectQuiet(found, now, quiet)
	}
	setRow := func(path string, status DatabaseBackupStatus, detail string) {
		if i, ok := rows[path]; ok {
			row := &(*summaries)[i]
			row.Status, row.Detail = status, detail
			row.LastVerified = index.LatestVerified(identity, path)
		}
	}
	var fresh []string
	for _, c := range found {
		if index.Seen(identity, c) {
			setRow(c.Path, DatabaseUnchanged, "")
			continue
		}
		described, err := (&SelectedFiles{Root: identity.Root, Limit: rawBundleLimit(root.Kind)}).Describe(ctx, []string{c.Path})
		if err != nil {
			setRow(c.Path, DatabaseFailed, err.Error())
			skipped = append(skipped, fmt.Sprintf("%s/%s: %v", root.Kind, c.Path, err))
			continue
		}
		if index.SeenHash(identity, c.Path, described[0].SHA256) {
			setRow(c.Path, DatabaseUnchanged, "")
			continue
		}
		fresh = append(fresh, c.Path)
	}
	archived := 0
	for len(fresh) > 0 {
		if collected && ctx.Err() != nil {
			for _, path := range fresh {
				setRow(path, DatabaseFailed, "publication cancelled before attempt: "+ctx.Err().Error())
			}
			return archived, skipped, errors.Join(append(failures, ctx.Err())...)
		}
		n := min(len(fresh), MaxFiles)
		if collected {
			// One request per snapshot: an oversized one is skipped alone,
			// never taking the rest of the batch with it.
			n = 1
		}
		req := ArchiveRequest{Version: Version, Kind: root.Kind, Source: identity, Files: fresh[:n], Recipients: p.Config.Recipients, Sink: p.Config.Sink, Catalog: p.Config.Catalog}
		records, err := p.Archiver.create(ctx, req, collected)
		for _, r := range records {
			index.AddRecord(r)
			for _, f := range r.Manifest.Files {
				setRow(f.Path, DatabaseArchived, "")
			}
			archived += len(r.Manifest.Files)
		}
		if collected && errors.Is(err, ErrCiphertextLimit) && ctx.Err() == nil {
			setRow(fresh[0], DatabaseFailed, err.Error())
			// A snapshot too large even compressed is skipped, not a failed
			// pass: it would fail the same way every day (pair#616).
			skipped = append(skipped, fmt.Sprintf("%s/%s: %v; skipped", root.Kind, fresh[0], err))
			fresh = fresh[n:]
			continue
		}
		if err != nil {
			for _, path := range fresh[:n] {
				setRow(path, DatabaseFailed, err.Error())
			}
			if !collected {
				return archived, skipped, err
			}
			if ctx.Err() != nil {
				for _, path := range fresh[n:] {
					setRow(path, DatabaseFailed, "publication cancelled before attempt: "+ctx.Err().Error())
				}
				return archived, skipped, errors.Join(append(failures, ctx.Err())...)
			}
			failures = append(failures, fmt.Errorf("%s: %w", fresh[0], err))
		}
		fresh = fresh[n:]
	}
	return archived, skipped, errors.Join(failures...)
}

func (p ContinuousPass) archivedIndex(ctx context.Context) (*ArchivedIndex, error) {
	index := NewArchivedIndex()
	catalog, err := OpenCatalog(ctx, p.Config.Catalog)
	if err != nil {
		return nil, err
	}
	defer catalog.Close()
	return index, catalog.Each(ctx, func(r CatalogRecord) error { index.AddRecord(r); return nil })
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
