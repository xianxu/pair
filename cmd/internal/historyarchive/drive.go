package historyarchive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const maxDriveCandidates = 100000
const maxDriveCandidateBytes = 64 << 20
const drivePhaseBudget = 30 * time.Minute
const maxDriveBatchBytes int64 = 4 << 30

// DriveReport describes independent remote evidence; it grants no local authority.
type DriveReport struct {
	Inventory           *DriveInventoryObservation
	Verified            int
	AlreadyRecorded     int
	DownloadedChecksums int
	Deferred            int
	StagedBytes         int64
	LinkedObjects       int
	CopiedStageBytes    int64
	Pending             int
	Notes               []string
}

// DriveMirror copies catalogued ciphertext under the caller's continuous lock.
// It never changes the local catalog. Network work starts after its lock closes.
// pair:m5-concept integration
type DriveMirror struct {
	Rclone         Rclone
	Now            func() time.Time
	BatchByteLimit int64 // optional lower cap; never above 4 GiB
}

type driveCandidate struct {
	Label               BundleLabel
	BundleKey, LabelKey string
	Receipt             UploadReceipt
	State               UploadState
}

func driveCandidates(ctx context.Context, cfg ContinuousConfig) ([]driveCandidate, error) {
	c, err := OpenCatalog(ctx, cfg.Catalog)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	return projectDriveCandidates(ctx, c, cfg)
}

func projectDriveCandidates(ctx context.Context, c *Catalog, cfg ContinuousConfig) ([]driveCandidate, error) {
	var out []driveCandidate
	bytesUsed := 0
	seen := map[string]bool{}
	err := c.Each(ctx, func(r CatalogRecord) error {
		if r.Sink != cfg.Sink || seen[r.Label.ID] {
			return nil
		}
		candidate := driveCandidate{Label: r.Label, BundleKey: r.BundleKey, LabelKey: r.LabelKey}
		raw, e := json.Marshal(candidate)
		if e != nil {
			return e
		}
		if len(out) >= maxDriveCandidates || len(raw) > maxDriveCandidateBytes-bytesUsed {
			return errors.New("Drive candidate projection exceeds bound")
		}
		bytesUsed += len(raw)
		seen[r.Label.ID] = true
		out = append(out, candidate)
		return nil
	})
	return out, err
}

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
	objects, err := openBatchObjects(stage)
	if err != nil {
		return report, err
	}
	defer objects.Close()
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
	dest, err := ParseDriveDestination(cfg.DriveRemote, cfg.DriveFolder)
	if err != nil {
		return report, err
	}
	pin, err := d.Rclone.ResolveDestination(ctx, dest)
	if err != nil {
		return report, err
	}
	if !validRemoteIdentity(pin.FolderID) {
		if d.Rclone.LocalConformance {
			pin.FolderID = pin.Root
		} else {
			return report, errors.New("Drive folder has invalid identity")
		}
	}
	before, err := d.Rclone.ObserveFlatInventory(ctx, pin)
	if err != nil {
		return report, err
	}
	observedBefore, destinationEntries := observeDriveInventory(dest, pin.FolderID, "before", before)
	report.Inventory = &observedBefore
	receipts := LocalSink{Root: cfg.Catalog + ".uploads"}
	receiptCount, err := countUploadReceipts(ctx, receipts.Root, dest.Namespace())
	if err != nil {
		return report, err
	}
	destinationDeferred := 0
	source := LocalSink{Root: cfg.Sink}
	var ready []driveCandidate
	var failures []error
	var used int64
	exhausted, needsCopy := false, false
	now := time.Now().UTC()
	if d.Now != nil {
		now = d.Now().UTC()
	}
	for _, candidate := range candidates {
		if err = ctx.Err(); err != nil {
			return report, err
		}
		bname, lname, e := dest.RemoteObjectNames(candidate.Label.ID)
		if e != nil {
			return report, e
		}
		label, e := readDriveLabel(ctx, source, candidate)
		if e != nil {
			failures = append(failures, e)
			continue
		}
		expected := UploadReceipt{Version: Version, Destination: dest.String(), FolderID: pin.FolderID, BundleID: candidate.Label.ID, VerifiedAt: now,
			Bundle: UploadObject{Name: bname, Size: candidate.Label.Bytes, SHA256: candidate.Label.SHA256}, Label: UploadObject{Name: lname, Size: int64(len(label))}}
		_, expected.Label.SHA256, e = fileDigest(ctx, bytes.NewReader(label), nil, MaxMetadataBytes)
		if e != nil {
			return report, e
		}
		key := dest.Namespace() + "/" + candidate.Label.ID + ".json"
		existing, found, e := readUploadReceipt(ctx, receipts, key)
		if e != nil {
			failures = append(failures, e)
			continue
		}
		if found {
			want := existing
			want.Destination = expected.Destination
			want.FolderID = expected.FolderID
			want.BundleID = expected.BundleID
			want.Bundle.Name = expected.Bundle.Name
			want.Bundle.Size = expected.Bundle.Size
			want.Bundle.SHA256 = expected.Bundle.SHA256
			want.Label.Name = expected.Label.Name
			want.Label.Size = expected.Label.Size
			want.Label.SHA256 = expected.Label.SHA256
			if !MatchReceipt(existing, want) {
				failures = append(failures, errors.New("Drive receipt conflicts with local pair; inspect upload catalog"))
				continue
			}
			report.AlreadyRecorded++
			report.Pending--
			continue
		}
		next, admit, e := AdmitBatchPair(used, expected.Bundle.Size, expected.Label.Size, limit)
		if e != nil {
			return report, e
		}
		if exhausted || !admit || len(ready) >= maxFlatInventoryEntries/2 {
			exhausted = true
			report.Deferred++
			continue
		}
		candidate.Receipt = expected
		candidate.State = UploadState{Destination: dest.String(), BundleID: candidate.Label.ID, Phase: UploadInventoryBefore}
		valid := true
		missingEntries := 0
		for _, object := range []UploadObject{expected.Bundle, expected.Label} {
			entries := before[object.Name]
			if len(entries) == 0 {
				missingEntries++
			}
			if len(entries) > 1 || len(entries) == 1 && (entries[0].IsDir || entries[0].Size != object.Size) {
				valid = false
				failures = append(failures, fmt.Errorf("%s: Drive name duplicated or size mismatch", candidate.Label.ID))
				break
			}
		}
		if !valid {
			_, _ = advanceUpload(&candidate, UploadConflict, false)
			continue
		}
		// Reserve only missing members, atomically for the complete pair. A full
		// destination can still reconcile objects already present, including a
		// later candidate after an earlier pair could not fit.
		if missingEntries > maxFlatInventoryEntries-destinationEntries {
			report.Deferred++
			destinationDeferred++
			continue
		}
		for i, sourceKey := range []string{candidate.BundleKey, candidate.LabelKey} {
			object := &candidate.Receipt.Bundle
			if i == 1 {
				object = &candidate.Receipt.Label
			}
			md5sum, linked, e := PrepareBatchObject(ctx, source, sourceKey, *object, objects)
			if e != nil {
				return report, fmt.Errorf("prepare Drive batch %s: %w", candidate.Label.ID, e)
			}
			object.MD5 = md5sum
			if linked {
				report.LinkedObjects++
			} else {
				report.CopiedStageBytes += object.Size
			}
			if entries := before[object.Name]; len(entries) == 1 {
				if _, e = VerifyRemoteEntry(entries[0], *object); e != nil {
					valid = false
					failures = append(failures, fmt.Errorf("%s: %w", candidate.Label.ID, e))
					break
				}
			}
		}
		if !valid {
			for _, name := range []string{bname, lname} {
				if e = unix.Unlinkat(int(objects.Fd()), name, 0); e != nil && !errors.Is(e, unix.ENOENT) {
					return report, e
				}
			}
			_, _ = advanceUpload(&candidate, UploadConflict, false)
			continue
		}
		if _, e = advanceUpload(&candidate, UploadPrepared, false); e != nil {
			return report, e
		}
		used = next
		destinationEntries += missingEntries
		needsCopy = needsCopy || len(before[bname]) == 0 || len(before[lname]) == 0
		ready = append(ready, candidate)
	}
	report.StagedBytes = used
	if batchDeferred := report.Deferred - destinationDeferred; batchDeferred > 0 {
		report.Notes = append(report.Notes, fmt.Sprintf("%d complete pair(s) deferred by %d-byte batch limit", batchDeferred, limit))
	}
	if destinationDeferred > 0 {
		report.Notes = append(report.Notes, fmt.Sprintf("%d complete pair(s) deferred by %d-object destination entry limit", destinationDeferred, maxFlatInventoryEntries))
	}
	if len(ready) > 0 {
		report.Notes = append(report.Notes, fmt.Sprintf("batch staged %d logical bytes: %d hard links, %d copied bytes", report.StagedBytes, report.LinkedObjects, report.CopiedStageBytes))
	}
	if needsCopy {
		if e := d.Rclone.CopyBatch(ctx, pin, objects.Name()); e != nil {
			failures = append(failures, fmt.Errorf("Drive copy batch: %w", e))
		}
		if err = ctx.Err(); err != nil {
			return report, errors.Join(append(failures, err)...)
		}
	}
	for i := range ready {
		if _, e := advanceUpload(&ready[i], UploadBatchObserved, false); e != nil {
			return report, e
		}
	}
	before = nil
	after, err := d.Rclone.ObserveFlatInventory(ctx, pin)
	if err != nil {
		return report, errors.Join(append(failures, err)...)
	}
	observedAfter, _ := observeDriveInventory(dest, pin.FolderID, "after", after)
	report.Inventory = &observedAfter
	fallback := map[string]string{}
	for i := range ready {
		candidate := &ready[i]
		valid := true
		for _, object := range []*UploadObject{&candidate.Receipt.Bundle, &candidate.Receipt.Label} {
			entries := after[object.Name]
			if len(entries) != 1 {
				failures = append(failures, fmt.Errorf("%s: Drive object %s missing or duplicated", candidate.Label.ID, object.Name))
				valid = false
				break
			}
			entry := entries[0]
			method, e := VerifyRemoteEntry(entry, *object)
			if e != nil {
				failures = append(failures, fmt.Errorf("%s: %w", candidate.Label.ID, e))
				valid = false
				break
			}
			if !d.Rclone.LocalConformance && !validRemoteIdentity(entry.ID) {
				failures = append(failures, errors.New("Drive object lacks stable identity"))
				valid = false
				break
			}
			object.RemoteID = entry.ID
			if d.Rclone.LocalConformance && object.RemoteID == "" {
				object.RemoteID = object.Name
			}
			object.Verification = method
			if method == "downloaded-sha256" {
				fallback[object.Name] = object.SHA256
			}
		}
		if !valid {
			delete(fallback, candidate.Receipt.Bundle.Name)
			delete(fallback, candidate.Receipt.Label.Name)
			_, _ = advanceUpload(candidate, UploadConflict, false)
		}
	}
	var downloaded map[string]string
	if len(fallback) > 0 {
		listPath, e := writeDownloadList(ctx, stage, fallback)
		if e != nil {
			return report, e
		}
		downloaded, e = d.Rclone.DownloadSHA256Batch(ctx, pin, listPath)
		if e != nil {
			failures = append(failures, fmt.Errorf("Drive checksum batch: %w", e))
		}
	}
	for i := range ready {
		candidate := &ready[i]
		if candidate.State.Phase != UploadInventoryAfter {
			continue
		}
		valid := true
		for _, object := range []UploadObject{candidate.Receipt.Bundle, candidate.Receipt.Label} {
			if object.Verification == "downloaded-sha256" {
				if downloaded[object.Name] != object.SHA256 {
					valid = false
					failures = append(failures, fmt.Errorf("%s: downloaded checksum missing or mismatched", candidate.Label.ID))
					break
				}
				report.DownloadedChecksums++
			}
		}
		if !valid {
			_, _ = advanceUpload(candidate, UploadUnknown, false)
			continue
		}
		effect, e := advanceUpload(candidate, UploadFinalMatch, true)
		if e != nil {
			return report, e
		}
		if effect != UploadRecordReceipt {
			return report, errors.New("upload transition did not authorize receipt")
		}
		if receiptCount >= maxDriveCandidates {
			failures = append(failures, errors.New("Drive upload catalog reached 100000 receipt bound"))
			break
		}
		key := dest.Namespace() + "/" + candidate.Label.ID + ".json"
		if e = PublishUploadReceipt(ctx, receipts, key, candidate.Receipt); e != nil {
			failures = append(failures, e)
			_, _ = advanceUpload(candidate, UploadUnknown, false)
			continue
		}
		if _, e = advanceUpload(candidate, UploadRecorded, false); e != nil {
			return report, e
		}
		receiptCount++
		report.Verified++
		report.Pending--
		report.Notes = append(report.Notes, fmt.Sprintf("%s: bundle=%s label=%s", candidate.Label.ID, candidate.Receipt.Bundle.Verification, candidate.Receipt.Label.Verification))
	}
	return report, errors.Join(failures...)
}

func advanceUpload(c *driveCandidate, kind UploadEventKind, final bool) (UploadEffect, error) {
	next, effect, err := NextUpload(c.State, UploadEvent{Destination: c.State.Destination, BundleID: c.State.BundleID, Kind: kind, BundleMatched: final, LabelMatched: final})
	if err == nil {
		c.State = next
	}
	return effect, err
}

// VerifyRemoteEntry returns the required verification path, never size-only success.
func VerifyRemoteEntry(entry RemoteEntry, want UploadObject) (string, error) {
	if entry.Name != want.Name || entry.IsDir || entry.Size != want.Size {
		return "", errors.New("Drive object name/size conflict")
	}
	normalized := map[string]bool{}
	for key := range entry.Hashes {
		key = strings.ReplaceAll(strings.ToLower(key), "-", "")
		if normalized[key] {
			return "", errors.New("Drive checksum aliases are ambiguous")
		}
		normalized[key] = true
	}
	for _, method := range []struct{ key, want, path string }{{"sha256", want.SHA256, "backend-sha256"}, {"md5", want.MD5, "backend-md5"}} {
		for key, hash := range entry.Hashes {
			if strings.ReplaceAll(strings.ToLower(key), "-", "") == method.key && hash != "" {
				if !strings.EqualFold(hash, method.want) {
					return "", errors.New("Drive object checksum conflict")
				}
				return method.path, nil
			}
		}
	}
	return "downloaded-sha256", nil
}

func readDriveLabel(ctx context.Context, source Sink, c driveCandidate) ([]byte, error) {
	r, err := source.Get(ctx, c.LabelKey)
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(r, MaxMetadataBytes+1))
	err = errors.Join(err, r.Close())
	if err != nil {
		return nil, err
	}
	var label BundleLabel
	if len(raw) > MaxMetadataBytes {
		return nil, errors.New("Drive label exceeds metadata bound")
	}
	if label, err = DecodeLabel(raw); err != nil {
		return nil, err
	}
	if err = ValidateLabel(label); err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(label, c.Label) {
		return nil, errors.New("local label conflicts with catalog")
	}
	return raw, nil
}

func readUploadReceipt(ctx context.Context, sink Sink, key string) (UploadReceipt, bool, error) {
	r, err := sink.Get(ctx, key)
	if errors.Is(err, os.ErrNotExist) {
		return UploadReceipt{}, false, nil
	}
	if err != nil {
		return UploadReceipt{}, false, err
	}
	raw, err := io.ReadAll(io.LimitReader(r, MaxUploadReceiptBytes+1))
	err = errors.Join(err, r.Close())
	if err != nil {
		return UploadReceipt{}, false, err
	}
	receipt, err := DecodeUploadReceipt(raw)
	if err != nil {
		return receipt, true, err
	}
	_, hash, err := fileDigest(ctx, bytes.NewReader(raw), nil, MaxUploadReceiptBytes)
	if err == nil {
		err = sink.Verify(ctx, key, hash, int64(len(raw)))
	}
	return receipt, true, err
}
func PublishUploadReceipt(ctx context.Context, sink Sink, key string, receipt UploadReceipt) error {
	if err := ValidateUploadReceipt(receipt); err != nil {
		return err
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	if err = sink.Put(ctx, key, bytes.NewReader(raw), MaxUploadReceiptBytes); err != nil {
		return err
	}
	_, hash, err := fileDigest(ctx, bytes.NewReader(raw), nil, MaxUploadReceiptBytes)
	if err != nil {
		return err
	}
	return sink.Verify(ctx, key, hash, int64(len(raw)))
}

func countUploadReceipts(ctx context.Context, root, namespace string) (int, error) {
	dir, err := openSafeDir(filepath.Join(root, namespace), false)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer dir.Close()
	count := 0
	for {
		if err = ctx.Err(); err != nil {
			return 0, err
		}
		entries, e := dir.ReadDir(128)
		for _, entry := range entries {
			if reservedSinkName(entry.Name()) {
				continue
			}
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return 0, errors.New("invalid upload catalog entry")
			}
			if !strings.HasSuffix(entry.Name(), ".json") || !validHex(strings.TrimSuffix(entry.Name(), ".json"), 64) {
				return 0, errors.New("invalid upload receipt filename")
			}
			count++
			if count > maxDriveCandidates {
				return 0, errors.New("upload catalog exceeds receipt bound")
			}
		}
		if e == io.EOF {
			return count, nil
		}
		if e != nil {
			return 0, e
		}
	}
}
