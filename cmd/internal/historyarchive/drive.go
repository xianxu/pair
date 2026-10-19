package historyarchive

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
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

// DriveReport describes independent remote evidence; it grants no local authority.
type DriveReport struct {
	Verified            int
	AlreadyRecorded     int
	DownloadedChecksums int
	Pending             int
	Notes               []string
}

// DriveMirror copies catalogued ciphertext under the caller's continuous lock.
// It never changes the local catalog. Network work starts after its lock closes.
// pair:m5-concept integration
type DriveMirror struct {
	Rclone Rclone
	Now    func() time.Time
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
	candidates, err := driveCandidates(ctx, cfg)
	if err != nil {
		return report, err
	}
	report.Pending = len(candidates)
	if len(candidates) == 0 {
		return report, nil
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
	receiptRoot := cfg.Catalog + ".uploads"
	receipts := LocalSink{Root: receiptRoot}
	receiptCount, err := countUploadReceipts(ctx, receiptRoot, dest.Namespace())
	if err != nil {
		return report, err
	}
	stage, err := openUploadStaging(store)
	if err != nil {
		return report, err
	}
	defer stage.Close()
	if err = SweepUploadStaging(ctx, stage); err != nil {
		return report, err
	}
	source := LocalSink{Root: cfg.Sink}
	var ready []driveCandidate
	var failures []error
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
		candidate.Receipt = expected
		candidate.State = UploadState{Destination: dest.String(), BundleID: candidate.Label.ID, Phase: UploadInventoryBefore}
		if _, e = advanceUpload(&candidate, UploadMatch, false); e != nil {
			return report, e
		}
		for i, key := range []string{candidate.BundleKey, candidate.LabelKey} {
			object := &candidate.Receipt.Bundle
			if i == 1 {
				object = &candidate.Receipt.Label
			}
			staged, md5sum, e := StageVerifiedObject(ctx, source, key, *object, stage)
			if e != nil {
				failures = append(failures, e)
				_, _ = advanceUpload(&candidate, UploadUnknown, false)
				break
			}
			object.MD5 = md5sum
			entries := before[object.Name]
			if len(entries) > 1 {
				e = errors.New("Drive name is duplicated; inspect remote")
			} else if len(entries) == 1 {
				_, e = VerifyRemoteEntry(entries[0], *object)
				if e == nil {
					_, e = advanceUpload(&candidate, UploadMatch, false)
				}
			} else {
				var effect UploadEffect
				effect, e = advanceUpload(&candidate, UploadAbsent, false)
				if e == nil && (effect == UploadCopyBundle || effect == UploadCopyLabel) {
					e = d.Rclone.Copy(ctx, pin, staged, object.Name)
					if e == nil {
						_, e = advanceUpload(&candidate, UploadCopied, false)
					}
				}
			}
			cleanupErr := os.Remove(staged)
			e = errors.Join(e, cleanupErr)
			if e != nil {
				failures = append(failures, fmt.Errorf("%s: %w", candidate.Label.ID, e))
				_, _ = advanceUpload(&candidate, UploadConflict, false)
				break
			}
		}
		if candidate.State.Phase == UploadInventoryAfter {
			ready = append(ready, candidate)
		}
	}
	before = nil
	after, err := d.Rclone.ObserveFlatInventory(ctx, pin)
	if err != nil {
		return report, err
	}
	for _, candidate := range ready {
		ok := true
		for _, object := range []*UploadObject{&candidate.Receipt.Bundle, &candidate.Receipt.Label} {
			entries := after[object.Name]
			if len(entries) != 1 {
				failures = append(failures, fmt.Errorf("%s: Drive name missing or duplicated", candidate.Label.ID))
				ok = false
				break
			}
			entry := entries[0]
			method, e := VerifyRemoteEntry(entry, *object)
			if e != nil {
				failures = append(failures, e)
				ok = false
				break
			}
			if method == "downloaded-sha256" {
				digest, e := d.Rclone.DownloadSHA256(ctx, pin, object.Name)
				if e != nil || digest != object.SHA256 {
					failures = append(failures, errors.New("Drive downloaded checksum did not verify"))
					ok = false
					break
				}
				report.DownloadedChecksums++
			}
			if !d.Rclone.LocalConformance && !validRemoteIdentity(entry.ID) {
				failures = append(failures, errors.New("Drive object lacks stable identity"))
				ok = false
				break
			}
			object.RemoteID = entry.ID
			if d.Rclone.LocalConformance && object.RemoteID == "" {
				object.RemoteID = object.Name
			}
			object.Verification = method
		}
		if !ok {
			_, _ = advanceUpload(&candidate, UploadConflict, false)
			continue
		}
		effect, e := advanceUpload(&candidate, UploadFinalMatch, true)
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
			_, _ = advanceUpload(&candidate, UploadUnknown, false)
			continue
		}
		if _, e = advanceUpload(&candidate, UploadRecorded, false); e != nil {
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
	if err = decodeJSONNoDuplicateKeys(raw, &label, true); err != nil {
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
func openUploadStaging(store string) (*os.File, error) {
	dir, err := openSafeDir(store, true)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	child, err := childDir(dir, ".drive-staging", true)
	if err != nil {
		return nil, err
	}
	defer child.Close()
	fd, err := unix.Dup(int(child.Fd()))
	if err != nil {
		return nil, err
	}
	unix.CloseOnExec(fd)
	return os.NewFile(uintptr(fd), filepath.Join(store, ".drive-staging")), nil
}
func SweepUploadStaging(ctx context.Context, dir *os.File) error {
	// Open a fresh descriptor so each sweep starts at the beginning.
	scan, err := childDir(dir, ".", false)
	if err != nil {
		return err
	}
	defer scan.Close()
	count := 0
	for {
		if err = ctx.Err(); err != nil {
			return err
		}
		entries, e := scan.ReadDir(128)
		for _, entry := range entries {
			count++
			if count > MaxFiles {
				return errors.New("Drive staging cleanup exceeds bound")
			}
			if !strings.HasPrefix(entry.Name(), ".object-") {
				return errors.New("unrecognized Drive staging entry")
			}
			f, err := openRegularAt(scan, entry.Name(), unix.O_RDONLY)
			if err != nil {
				return err
			}
			f.Close()
			if err = unix.Unlinkat(int(scan.Fd()), entry.Name(), 0); err != nil {
				return err
			}
		}
		if e == io.EOF {
			return nil
		}
		if e != nil {
			return e
		}
	}
}
func StageVerifiedObject(ctx context.Context, source Sink, key string, want UploadObject, dir *os.File) (staged, md5sum string, err error) {
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return "", "", err
	}
	name := ".object-" + hex.EncodeToString(nonce[:])
	f, err := openRegularAt(dir, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL)
	if err != nil {
		return "", "", err
	}
	staged = filepath.Join(dir.Name(), name)
	defer func() {
		f.Close()
		if err != nil {
			unix.Unlinkat(int(dir.Fd()), name, 0)
		}
	}()
	r, err := source.Get(ctx, key)
	if err != nil {
		return "", "", err
	}
	defer r.Close()
	md5hash := md5.New()
	n, hash, err := fileDigest(ctx, io.TeeReader(r, md5hash), f, want.Size)
	if err != nil {
		return "", "", err
	}
	if n != want.Size || hash != want.SHA256 {
		return "", "", errors.New("local encrypted object checksum mismatch")
	}
	if err = f.Sync(); err != nil {
		return "", "", err
	}
	if err = f.Close(); err != nil {
		return "", "", err
	}
	return staged, hex.EncodeToString(md5hash.Sum(nil)), nil
}
