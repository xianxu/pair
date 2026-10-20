package historyarchive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Run routes the bounded catalog projection to physical folders. Receipts stay
// in the configured root's namespace even when their evidence names a month.
func (d DriveMirror) Run(ctx context.Context, cfg ContinuousConfig, store string) (report DriveReport, err error) {
	ctx, cancel := context.WithTimeout(ctx, drivePhaseBudget)
	defer cancel()
	stage, err := openUploadStaging(store)
	if err != nil {
		return report, err
	}
	defer stage.Close()
	if err = SweepUploadStaging(context.WithoutCancel(ctx), stage); err != nil {
		return report, err
	}
	defer func() { err = errors.Join(err, SweepUploadStaging(context.WithoutCancel(ctx), stage)) }()
	candidates, err := driveCandidates(ctx, cfg)
	if err != nil {
		return report, err
	}
	report.Pending = len(candidates)
	if len(candidates) == 0 {
		return report, nil
	}
	limit := d.BatchByteLimit
	if limit == 0 {
		limit = maxDriveBatchBytes
	}
	if limit < 0 || limit > maxDriveBatchBytes {
		return report, errors.New("invalid Drive batch byte budget")
	}
	root, err := ParseDriveDestination(cfg.DriveRemote, cfg.DriveFolder)
	if err != nil {
		return report, err
	}
	layout := DriveArchiveLayout{Root: root}
	receipts := LocalSink{Root: cfg.Catalog + ".uploads"}
	receiptCount, err := countUploadReceipts(ctx, receipts.Root, root.Namespace())
	if err != nil {
		return report, err
	}
	pin, err := d.Rclone.ResolveDestination(ctx, root)
	if err != nil {
		return report, err
	}
	if !validRemoteIdentity(pin.FolderID) {
		if !d.Rclone.LocalConformance {
			return report, errors.New("Drive folder has invalid identity")
		}
		pin.FolderID = pin.Root
	}
	before, err := d.Rclone.ObserveFlatInventory(ctx, pin)
	if err != nil {
		return report, err
	}
	observation, _ := observeDriveInventory(root, pin.FolderID, "before", before)
	report.Inventory = &observation
	report.Inventories = append(report.Inventories, observation)
	now := time.Now().UTC()
	if d.Now != nil {
		now = d.Now().UTC()
	}
	groups := map[string][]driveCandidate{}
	var legacy []driveCandidate
	var failures []error
	for _, candidate := range candidates {
		if err = ctx.Err(); err != nil {
			return report, err
		}
		month, e := layout.MonthDestination(candidate.Label)
		if e != nil {
			return report, e
		}
		existing, found, e := readUploadReceipt(ctx, receipts, root.Namespace()+"/"+candidate.Label.ID+".json")
		if e != nil {
			failures = append(failures, e)
			continue
		}
		if found {
			location, e := layout.ValidateRecordedDriveLocation(candidate.Label, existing.Destination)
			if e != nil {
				failures = append(failures, e)
				continue
			}
			folderID := existing.FolderID
			if location == root {
				folderID = pin.FolderID
			}
			expected, e := expectedDriveReceipt(ctx, cfg, candidate, location, folderID, now)
			if e != nil {
				failures = append(failures, e)
				continue
			}
			want := existing
			want.Destination, want.FolderID, want.BundleID = expected.Destination, expected.FolderID, expected.BundleID
			want.Bundle.Name, want.Bundle.Size, want.Bundle.SHA256 = expected.Bundle.Name, expected.Bundle.Size, expected.Bundle.SHA256
			want.Label.Name, want.Label.Size, want.Label.SHA256 = expected.Label.Name, expected.Label.Size, expected.Label.SHA256
			if !MatchReceipt(existing, want) {
				failures = append(failures, fmt.Errorf("%s: Drive receipt conflicts with local pair; inspect upload catalog", candidate.Label.ID))
				continue
			}
			report.AlreadyRecorded++
			report.Pending--
			continue
		}
		// Prepare only metadata here. Admission happens before any folder creation.
		candidate.Receipt, e = expectedDriveReceipt(ctx, cfg, candidate, month, pin.FolderID, now)
		if e != nil {
			failures = append(failures, e)
			continue
		}
		b, l := candidate.Receipt.Bundle.Name, candidate.Receipt.Label.Name
		if len(before[b]) != 0 || len(before[l]) != 0 {
			if len(before[b]) != 1 || len(before[l]) != 1 || before[b][0].IsDir || before[l][0].IsDir {
				failures = append(failures, fmt.Errorf("%s: legacy Drive pair is partial or ambiguous; inspect remote objects", candidate.Label.ID))
				continue
			}
			legacy = append(legacy, candidate)
			continue
		}
		groups[month.Folder] = append(groups[month.Folder], candidate)
	}
	// This bound applies before creating folders, even when every month is new.
	if len(groups) >= MaxIndexDestinations {
		return report, errors.New("Drive month projection exceeds destination bound")
	}
	run := func(dest DriveDestination, folder PinnedDrive, inventory map[string][]RemoteEntry, batch []driveCandidate) {
		got, e := d.runFolder(ctx, cfg, store, batch, dest, folder, inventory, root.Namespace(), &receiptCount, limit-report.StagedBytes)
		report.Verified += got.Verified
		report.Pending -= got.Verified
		report.Deferred += got.Deferred
		report.DownloadedChecksums += got.DownloadedChecksums
		report.StagedBytes += got.StagedBytes
		report.LinkedObjects += got.LinkedObjects
		report.CopiedStageBytes += got.CopiedStageBytes
		report.Notes = append(report.Notes, got.Notes...)
		if got.Inventory != nil {
			report.Inventories = append(report.Inventories, *got.Inventory)
			if dest == root {
				report.Inventory = got.Inventory
			}
		}
		if e != nil {
			failures = append(failures, fmt.Errorf("%s: %w", dest.String(), e))
		}
	}
	if len(legacy) > 0 {
		run(root, pin, before, legacy)
	}
	// Inspection alone never creates an empty month. Historical receipt-only
	// folders are deliberately absent from this work list.
	current := root.Folder + "/" + now.Format("2006/01")
	if _, ok := groups[current]; !ok {
		groups[current] = nil
	}
	months := make([]string, 0, len(groups))
	for month := range groups {
		months = append(months, month)
	}
	sort.Strings(months)
	type yearObservation struct {
		pin       PinnedDrive
		inventory map[string][]RemoteEntry
		err       error
		found     bool
	}
	years := map[string]yearObservation{}
	for _, month := range months {
		if err = ctx.Err(); err != nil {
			return report, errors.Join(append(failures, err)...)
		}
		batch := groups[month]
		// Defer before mkdir if the first pair cannot enter the remaining budget.
		// The verifier performs complete-pair admission for the rest of the batch.
		if len(batch) > 0 {
			_, admit, e := AdmitBatchPair(0, batch[0].Receipt.Bundle.Size, batch[0].Receipt.Label.Size, limit-report.StagedBytes)
			if e != nil {
				return report, e
			}
			if !admit {
				report.Deferred += len(batch)
				continue
			}
		}
		relative := strings.TrimPrefix(month, root.Folder+"/")
		year, leaf, _ := strings.Cut(relative, "/")
		observed, cached := years[year]
		if !cached || !observed.found && observed.err == nil && len(batch) > 0 {
			observed.pin, observed.found, observed.err = d.Rclone.ResolveChild(ctx, pin, year, before)
			if observed.err == nil && !observed.found && len(batch) > 0 {
				observed.pin, observed.err = d.Rclone.EnsureChild(ctx, pin, year, before)
				observed.found = observed.err == nil
			}
			if observed.err == nil && observed.found {
				observed.inventory, observed.err = d.Rclone.ObserveFlatInventory(ctx, observed.pin)
			}
			years[year] = observed
		}
		if observed.err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", month, observed.err))
			continue
		}
		if !observed.found {
			continue
		}
		child, found, e := d.Rclone.ResolveChild(ctx, observed.pin, leaf, observed.inventory)
		if e == nil && !found && len(batch) > 0 {
			child, e = d.Rclone.EnsureChild(ctx, observed.pin, leaf, observed.inventory)
			found = e == nil
			if found {
				observed.inventory[leaf] = []RemoteEntry{{Name: leaf, IsDir: true, ID: child.FolderID}}
				years[year] = observed
			}
		}
		if e != nil {
			failures = append(failures, fmt.Errorf("%s: %w", month, e))
			continue
		}
		if !found {
			continue
		}
		inventory, e := d.Rclone.ObserveFlatInventory(ctx, child)
		if e != nil {
			failures = append(failures, fmt.Errorf("%s: %w", month, e))
			continue
		}
		dest, e := ParseDriveDestination(root.Remote, month)
		if e != nil {
			return report, e
		}
		if len(batch) == 0 {
			observed, _ := observeDriveInventory(dest, child.FolderID, "before", inventory)
			report.Inventories = append(report.Inventories, observed)
			continue
		}
		run(dest, child, inventory, batch)
	}
	return report, errors.Join(failures...)
}

func expectedDriveReceipt(ctx context.Context, cfg ContinuousConfig, candidate driveCandidate, dest DriveDestination, folderID string, now time.Time) (UploadReceipt, error) {
	b, l, err := dest.RemoteObjectNames(candidate.Label.ID)
	if err != nil {
		return UploadReceipt{}, err
	}
	raw, err := readDriveLabel(ctx, LocalSink{Root: cfg.Sink}, candidate)
	if err != nil {
		return UploadReceipt{}, err
	}
	expected := UploadReceipt{Version: Version, Destination: dest.String(), FolderID: folderID, BundleID: candidate.Label.ID, VerifiedAt: now,
		Bundle: UploadObject{Name: b, Size: candidate.Label.Bytes, SHA256: candidate.Label.SHA256}, Label: UploadObject{Name: l, Size: int64(len(raw))}}
	_, expected.Label.SHA256, err = fileDigest(ctx, bytes.NewReader(raw), nil, MaxMetadataBytes)
	return expected, err
}
