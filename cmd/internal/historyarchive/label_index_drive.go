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
	"reflect"
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
			return report, errors.New("Drive index folder lacks stable identity")
		}
	}
	before, err := d.Rclone.ObserveFlatInventory(ctx, pin)
	if err != nil {
		return report, err
	}
	relevant, names, err := indexDriveInventory(before)
	if err != nil {
		return report, err
	}
	before = nil
	labels, err := childDir(store.StageDir, "labels", true)
	if err != nil {
		return report, err
	}
	defer labels.Close()
	if len(names) > 0 {
		err = d.Rclone.DownloadBatch(ctx, pin, names, filepath.Join(store.StagePath, "labels"), filepath.Join(store.StagePath, "files-from"), MaxIndexBytes, MaxMetadataBytes)
		if err != nil {
			return report, err
		}
	}
	after, err := d.Rclone.ObserveFlatInventory(ctx, pin)
	if err != nil {
		return report, err
	}
	final, _, err := indexDriveInventory(after)
	if err != nil {
		return report, err
	}
	if !reflect.DeepEqual(relevant, final) {
		return report, errors.New("Drive archive pairs changed during label observation")
	}
	after = nil
	now := time.Now().UTC()
	if d.Now != nil {
		now = d.Now().UTC()
	}
	refresh := IndexRefresh{Backend: "drive", Locator: dest.String(), FolderID: pin.FolderID, At: now, Complete: true}
	var used int64
	for _, name := range names {
		if err = ctx.Err(); err != nil {
			return report, err
		}
		f, e := openRegularAt(labels, name, unix.O_RDONLY)
		if e != nil {
			return report, e
		}
		raw, e := io.ReadAll(io.LimitReader(f, MaxMetadataBytes+1))
		e = errors.Join(e, f.Close())
		if e != nil {
			return report, e
		}
		if len(raw) > MaxMetadataBytes || int64(len(raw)) > MaxIndexBytes-used {
			return report, errors.New("Drive label download exceeds metadata bound")
		}
		used += int64(len(raw))
		remoteLabel := relevant[name]
		if int64(len(raw)) != remoteLabel.Size {
			return report, errors.New("downloaded Drive label size mismatch")
		}
		sum := sha256.Sum256(raw)
		labelSHA := hex.EncodeToString(sum[:])
		md5sum := md5.Sum(raw)
		labelMD5 := hex.EncodeToString(md5sum[:])
		hashes, e := indexDriveHashes(remoteLabel)
		if e != nil {
			return report, e
		}
		if hashes["sha256"] != "" && hashes["sha256"] != labelSHA || hashes["md5"] != "" && hashes["md5"] != labelMD5 {
			return report, errors.New("downloaded Drive label checksum conflict")
		}
		label, e := DecodeLabel(raw)
		if e != nil {
			return report, e
		}
		if e = ValidateLabel(label); e != nil {
			return report, e
		}
		if name != label.ID+".json" {
			return report, errors.New("Drive label ID does not match object name")
		}
		bundleName := label.ID + ".tar.zst.gpg"
		bundle := relevant[bundleName]
		if bundle.Size != label.Bytes {
			return report, errors.New("Drive bundle size does not match label")
		}
		hashes, e = indexDriveHashes(bundle)
		if e != nil {
			return report, e
		}
		if hashes["sha256"] != "" && hashes["sha256"] != label.SHA256 {
			return report, errors.New("Drive bundle SHA256 conflicts with label")
		}
		location := BundleLocation{Backend: "drive", Locator: dest.String(), FolderID: pin.FolderID, BundleKey: bundleName, LabelKey: name, Available: true, Evidence: "observed", FirstObserved: now, LastObserved: now}
		if hashes["sha256"] == label.SHA256 {
			location.Evidence = "verified"
			location.Proof = LocationProof{Method: "backend-sha256", SHA256: label.SHA256, VerifiedAt: now}
		} else {
			receipt, found, e := readUploadReceipt(ctx, LocalSink{Root: cfg.Catalog + ".uploads"}, dest.Namespace()+"/"+label.ID+".json")
			if e != nil {
				return report, e
			}
			if found {
				if receipt.Destination != dest.String() || receipt.FolderID != pin.FolderID || receipt.BundleID != label.ID || receipt.Bundle.Name != bundleName || receipt.Bundle.Size != label.Bytes || receipt.Bundle.SHA256 != label.SHA256 || receipt.Bundle.RemoteID != bundle.ID || receipt.Label.Name != name || receipt.Label.Size != int64(len(raw)) || receipt.Label.SHA256 != labelSHA || receipt.Label.MD5 != labelMD5 || receipt.Label.RemoteID != remoteLabel.ID || receipt.VerifiedAt.After(now) {
					return report, errors.New("Drive upload receipt conflicts with current pair")
				}
				if hashes["md5"] != "" && hashes["md5"] != receipt.Bundle.MD5 {
					return report, errors.New("Drive bundle MD5 conflicts with upload receipt")
				}
				location.Evidence = "verified"
				location.Proof = LocationProof{Method: "upload-receipt", SHA256: label.SHA256, VerifiedAt: receipt.VerifiedAt}
			}
		}
		refresh.Observations = append(refresh.Observations, IndexObservation{Label: label, Location: location})
	}
	if err = ctx.Err(); err != nil {
		return report, err
	}
	next, err := ApplyIndexRefresh(old, refresh)
	if err != nil {
		return report, err
	}
	if err = store.Commit(next); err != nil {
		return report, err
	}
	report.LabelCount = len(refresh.Observations)
	for _, ob := range refresh.Observations {
		if ob.Location.Evidence == "verified" {
			report.Verified++
		} else {
			report.Observed++
		}
	}
	return report, nil
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
