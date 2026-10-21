package historyarchive

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// IndexSyncReport counts current remote observations, separately from proofs.
type IndexSyncReport struct{ LabelCount, Observed, Verified int }

// DriveIndexSync rebuilds discovery metadata from labels, without creating local
// archive authority or downloading bundle ciphertext for checksum evidence.
// pair:m5-concept integration
type DriveIndexSync struct {
	Rclone Rclone
	Now    func() time.Time
}

func (d DriveIndexSync) Run(ctx context.Context, cfg ContinuousConfig) (report IndexSyncReport, err error) {
	ctx, cancel := context.WithTimeout(ctx, drivePhaseBudget)
	defer cancel()
	store, err := OpenLabelIndexStore(ctx, cfg.Catalog)
	if err != nil {
		return report, err
	}
	defer func() { err = errors.Join(err, store.Close()) }()
	old, err := store.Read()
	if err != nil {
		return report, err
	}
	root, err := ParseDriveDestination(cfg.DriveRemote, cfg.DriveFolder)
	if err != nil {
		return report, err
	}
	folders, parents, err := d.observeIndexTree(ctx, root)
	if err != nil {
		return report, err
	}
	now := time.Now().UTC()
	if d.Now != nil {
		now = d.Now().UTC()
	}
	refreshes, err := missingIndexDriveFolders(old, root, folders, now)
	if err != nil {
		return report, err
	}
	for i, folder := range folders {
		refresh, e := d.readIndexDriveFolder(ctx, cfg, root, folder, store, i, now)
		if e != nil {
			return report, e
		}
		refreshes = append(refreshes, refresh)
	}
	if err = d.confirmIndexTree(ctx, root, folders, parents); err != nil {
		return report, err
	}
	next, err := applyIndexRefreshes(old, refreshes)
	if err != nil {
		return report, err
	}
	if err = ctx.Err(); err != nil {
		return report, err
	}
	if err = store.Commit(next); err != nil {
		return report, err
	}
	for _, refresh := range refreshes {
		for _, ob := range refresh.Observations {
			report.LabelCount++
			if ob.Location.Evidence == "verified" {
				report.Verified++
			} else {
				report.Observed++
			}
		}
	}
	return report, nil
}

func (d DriveIndexSync) readIndexDriveFolder(ctx context.Context, cfg ContinuousConfig, root DriveDestination, folder indexDriveFolder, store *LabelIndexStore, ordinal int, now time.Time) (refresh IndexRefresh, err error) {
	refresh = IndexRefresh{Backend: "drive", Locator: folder.dest.String(), FolderID: folder.pin.FolderID, At: now, Complete: true}
	if len(folder.names) == 0 {
		return refresh, nil
	}
	stageName := fmt.Sprintf("labels-%04d", ordinal)
	labelRoot, err := childDir(store.StageDir, "labels", true)
	if err != nil {
		return refresh, err
	}
	defer func() { err = errors.Join(err, labelRoot.Close()) }()
	labels, err := childDir(labelRoot, stageName, true)
	if err != nil {
		return refresh, err
	}
	defer func() { err = errors.Join(err, labels.Close()) }()
	if err = d.Rclone.DownloadBatch(ctx, folder.pin, folder.names, filepath.Join(store.StagePath, "labels", stageName), filepath.Join(store.StagePath, "files-from"), folder.labelBytes, MaxMetadataBytes); err != nil {
		return refresh, err
	}
	var used int64
	for _, name := range folder.names {
		if err = ctx.Err(); err != nil {
			return refresh, err
		}
		f, e := openRegularAt(labels, name, unix.O_RDONLY)
		if e != nil {
			return refresh, e
		}
		raw, e := io.ReadAll(io.LimitReader(f, MaxMetadataBytes+1))
		e = errors.Join(e, f.Close())
		if e != nil {
			return refresh, e
		}
		if len(raw) > MaxMetadataBytes || int64(len(raw)) > MaxIndexBytes-used {
			return refresh, errors.New("Drive label download exceeds metadata bound")
		}
		used += int64(len(raw))
		remoteLabel := folder.relevant[name]
		if int64(len(raw)) != remoteLabel.Size {
			return refresh, errors.New("downloaded Drive label size mismatch")
		}
		sum := sha256.Sum256(raw)
		labelSHA := hex.EncodeToString(sum[:])
		md5sum := md5.Sum(raw)
		labelMD5 := hex.EncodeToString(md5sum[:])
		hashes, e := indexDriveHashes(remoteLabel)
		if e != nil {
			return refresh, e
		}
		if hashes["sha256"] != "" && hashes["sha256"] != labelSHA || hashes["md5"] != "" && hashes["md5"] != labelMD5 {
			return refresh, errors.New("downloaded Drive label checksum conflict")
		}
		label, e := DecodeLabel(raw)
		if e != nil {
			return refresh, e
		}
		if e = ValidateLabel(label); e != nil {
			return refresh, e
		}
		if name != label.ID+".json" {
			return refresh, errors.New("Drive label ID does not match object name")
		}
		if _, e = (DriveArchiveLayout{Root: root}).ValidateRecordedDriveLocation(label, folder.dest.String()); e != nil {
			return refresh, e
		}
		bundleName := label.ID + ".tar.zst.gpg"
		bundle := folder.relevant[bundleName]
		if bundle.Size != label.Bytes {
			return refresh, errors.New("Drive bundle size does not match label")
		}
		hashes, e = indexDriveHashes(bundle)
		if e != nil {
			return refresh, e
		}
		if hashes["sha256"] != "" && hashes["sha256"] != label.SHA256 {
			return refresh, errors.New("Drive bundle SHA256 conflicts with label")
		}
		location := BundleLocation{Backend: "drive", Locator: folder.dest.String(), FolderID: folder.pin.FolderID, BundleKey: bundleName, LabelKey: name, Available: true, Evidence: "observed", FirstObserved: now, LastObserved: now}
		if hashes["sha256"] == label.SHA256 {
			location.Evidence = "verified"
			location.Proof = LocationProof{Method: "backend-sha256", SHA256: label.SHA256, VerifiedAt: now}
		} else {
			receipt, found, e := readUploadReceipt(ctx, LocalSink{Root: cfg.Catalog + ".uploads"}, root.Namespace()+"/"+label.ID+".json")
			if e != nil {
				return refresh, e
			}
			if found {
				if _, e = (DriveArchiveLayout{Root: root}).ValidateRecordedDriveLocation(label, receipt.Destination); e != nil {
					return refresh, e
				}
				// A root-scoped receipt attests exactly one physical copy. Other
				// supported copies still receive their own observed evidence.
				if receipt.Destination != folder.dest.String() {
					refresh.Observations = append(refresh.Observations, IndexObservation{Label: label, Location: location})
					continue
				}
				if receipt.Destination != folder.dest.String() || receipt.FolderID != folder.pin.FolderID || receipt.BundleID != label.ID || receipt.Bundle.Name != bundleName || receipt.Bundle.Size != label.Bytes || receipt.Bundle.SHA256 != label.SHA256 || receipt.Bundle.RemoteID != bundle.ID || receipt.Label.Name != name || receipt.Label.Size != int64(len(raw)) || receipt.Label.SHA256 != labelSHA || receipt.Label.MD5 != labelMD5 || receipt.Label.RemoteID != remoteLabel.ID || receipt.VerifiedAt.After(now) {
					return refresh, errors.New("Drive upload receipt conflicts with current pair")
				}
				if hashes["md5"] != "" && hashes["md5"] != receipt.Bundle.MD5 {
					return refresh, errors.New("Drive bundle MD5 conflicts with upload receipt")
				}
				location.Evidence = "verified"
				location.Proof = LocationProof{Method: "upload-receipt", SHA256: label.SHA256, VerifiedAt: receipt.VerifiedAt}
			}
		}
		refresh.Observations = append(refresh.Observations, IndexObservation{Label: label, Location: location})
	}
	// Release each folder's private stage before downloading the next folder,
	// keeping owned cleanup within its existing entry bound.
	for _, name := range folder.names {
		if err = unix.Unlinkat(int(labels.Fd()), name, 0); err != nil {
			return refresh, err
		}
	}
	if err = unix.Unlinkat(int(labelRoot.Fd()), stageName, unix.AT_REMOVEDIR); err != nil {
		return refresh, err
	}
	return refresh, nil
}

// indexDriveInventory validates every relevant name before selecting labels.
// A lone archive member is an incomplete observation, never a missing location.
func indexDriveInventory(inventory map[string][]RemoteEntry) (map[string]RemoteEntry, []string, error) {
	relevant := make(map[string]RemoteEntry)
	var names []string
	var used int64
	for name, entries := range inventory {
		id, label, relevantName := indexDriveName(name)
		if !relevantName {
			continue
		}
		if strings.ToLower(id) != id || len(entries) != 1 {
			return nil, nil, errors.New("Drive archive name is noncanonical or duplicated")
		}
		entry := entries[0]
		if entry.Name != name || entry.IsDir || !validRemoteIdentity(entry.ID) || entry.Size <= 0 {
			return nil, nil, errors.New("Drive archive object lacks regular stable identity")
		}
		if _, err := indexDriveHashes(entry); err != nil {
			return nil, nil, err
		}
		if label {
			if len(names) >= MaxIndexRecords || entry.Size > MaxMetadataBytes || entry.Size > MaxIndexBytes-used {
				return nil, nil, errors.New("Drive label inventory exceeds metadata bound")
			}
			used += entry.Size
			names = append(names, name)
		} else if entry.Size > MaxCiphertextBytes {
			return nil, nil, errors.New("Drive bundle exceeds ciphertext bound")
		}
		relevant[name] = entry
	}
	for name := range relevant {
		id, label, _ := indexDriveName(name)
		peer := id + ".json"
		if label {
			peer = id + ".tar.zst.gpg"
		}
		if _, ok := relevant[peer]; !ok {
			return nil, nil, fmt.Errorf("Drive archive pair is incomplete: %s", name)
		}
	}
	sort.Strings(names)
	return relevant, names, nil
}

func indexDriveName(name string) (id string, label, relevant bool) {
	for _, suffix := range []string{".tar.zst.gpg", ".json"} {
		if strings.HasSuffix(name, suffix) {
			id = strings.TrimSuffix(name, suffix)
			if validHex(id, 64) {
				return id, suffix == ".json", true
			}
		}
	}
	return "", false, false
}

func indexDriveHashes(entry RemoteEntry) (map[string]string, error) {
	out := make(map[string]string, len(entry.Hashes))
	for key, value := range entry.Hashes {
		key = strings.ReplaceAll(strings.ToLower(key), "-", "")
		if _, ok := out[key]; ok {
			return nil, errors.New("ambiguous Drive checksum aliases")
		}
		value = strings.ToLower(value)
		if value != "" && (key == "sha256" && !validHex(value, 64) || key == "md5" && !validHex(value, 32)) {
			return nil, errors.New("invalid Drive object checksum")
		}
		out[key] = value
	}
	return out, nil
}
