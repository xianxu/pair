package historyarchive

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
)

// LoadLabelView composes existing discovery evidence without publishing metadata,
// repairing catalogs, reestablishing receipt durability or contacting a remote.
func LoadLabelView(ctx context.Context, cfg ContinuousConfig, at time.Time) (LabelIndex, error) {
	if err := ctx.Err(); err != nil {
		return LabelIndex{}, err
	}
	old, err := ReadLabelIndex(cfg.Catalog + ".index.json")
	if errors.Is(err, os.ErrNotExist) {
		old, err = LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, nil
	}
	if err != nil {
		return LabelIndex{}, err
	}
	return composeLocalLabelView(ctx, cfg, at, old)
}

// SyncLocalLabelIndex explicitly publishes a complete local observation while
// holding the index mutation lock across observation, merge and publication.
func SyncLocalLabelIndex(ctx context.Context, cfg ContinuousConfig, at time.Time) (i LabelIndex, err error) {
	store, err := OpenLabelIndexStore(ctx, cfg.Catalog)
	if err != nil {
		return LabelIndex{}, err
	}
	outcome := commitoutcome.NotAuthoritative
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			err = commitoutcome.Join(outcome, err, closeErr)
		}
	}()
	old, err := store.Read()
	if err != nil {
		return LabelIndex{}, err
	}
	i, err = composeLocalLabelView(ctx, cfg, at, old)
	if err != nil {
		return LabelIndex{}, err
	}
	err = store.Commit(i)
	outcome = commitoutcome.Of(err)
	if err != nil {
		return LabelIndex{}, err
	}
	return i, nil
}

func composeLocalLabelView(ctx context.Context, cfg ContinuousConfig, at time.Time, old LabelIndex) (LabelIndex, error) {
	refresh, err := ObserveLocalLabels(ctx, cfg, at)
	if err != nil {
		return LabelIndex{}, err
	}
	view, err := ApplyIndexRefresh(old, refresh)
	if err != nil {
		return LabelIndex{}, err
	}
	// Optional Drive configuration cannot disable local discovery or local sync.
	// Indexed remote history is retained; receipt projection needs a valid identity.
	if dest, destinationErr := ParseDriveDestination(cfg.DriveRemote, cfg.DriveFolder); destinationErr == nil {
		if err = addExistingReceiptLocations(ctx, cfg, dest, &view); err != nil {
			return LabelIndex{}, err
		}
	}
	if err = ctx.Err(); err != nil {
		return LabelIndex{}, err
	}
	return view, nil
}

func readViewUploadReceipt(ctx context.Context, sink LocalSink, key string) (UploadReceipt, bool, error) {
	f, err := sink.Get(ctx, key)
	if errors.Is(err, os.ErrNotExist) {
		return UploadReceipt{}, false, nil
	}
	if err != nil {
		return UploadReceipt{}, false, err
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxUploadReceiptBytes+1))
	err = errors.Join(err, f.Close())
	if err != nil {
		return UploadReceipt{}, false, err
	}
	receipt, err := DecodeUploadReceipt(raw)
	return receipt, true, err
}

func addExistingReceiptLocations(ctx context.Context, cfg ContinuousConfig, dest DriveDestination, view *LabelIndex) error {
	// Track the serialized size incrementally: adding N receipts never rescans
	// the entire index N times or permits the candidate snapshot to exceed bounds.
	raw, err := json.Marshal(view)
	if err != nil {
		return err
	}
	bytesUsed := len(raw)
	receipts := LocalSink{Root: cfg.Catalog + ".uploads"}
	labels := LocalSink{Root: cfg.Sink}
	namespace := dest.Namespace() + "/"
	for id, record := range view.Records {
		if err := ctx.Err(); err != nil {
			return err
		}
		receipt, found, err := readViewUploadReceipt(ctx, receipts, namespace+id+".json")
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		physical, err := (DriveArchiveLayout{Root: dest}).ValidateRecordedDriveLocation(record.Label, receipt.Destination)
		if err != nil {
			return err
		}
		b, l, err := physical.RemoteObjectNames(id)
		if err != nil {
			return err
		}
		if receipt.BundleID != id || receipt.Bundle.Name != b || receipt.Label.Name != l || receipt.Bundle.SHA256 != record.Label.SHA256 || receipt.Bundle.Size != record.Label.Bytes {
			return errors.New("upload receipt conflicts with configured destination or bundle label")
		}
		_, labelKey, err := BundleKeys(record.Label.Kind, record.Label.From, record.Label.To, id)
		if err != nil {
			return err
		}
		// Receipt hashes refer to immutable raw sidecar bytes, including future fields.
		// A missing local sidecar does not erase the historical upload attestation.
		f, err := labels.Get(ctx, labelKey)
		if err == nil {
			sidecar, readErr := io.ReadAll(io.LimitReader(f, MaxMetadataBytes+1))
			readErr = errors.Join(readErr, f.Close())
			if readErr != nil {
				return readErr
			}
			sum := sha256.Sum256(sidecar)
			if len(sidecar) > MaxMetadataBytes || int64(len(sidecar)) != receipt.Label.Size || hex.EncodeToString(sum[:]) != receipt.Label.SHA256 {
				return errors.New("upload receipt conflicts with local label bytes")
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		loc := BundleLocation{Backend: "drive", Locator: physical.String(), FolderID: receipt.FolderID, BundleKey: b, LabelKey: l, Available: true, Evidence: "verified", FirstObserved: receipt.VerifiedAt, LastObserved: receipt.VerifiedAt,
			Proof: LocationProof{Method: "upload-receipt", SHA256: record.Label.SHA256, VerifiedAt: receipt.VerifiedAt}}
		prior, exists := record.Locations[loc.Identity()]
		marker, complete := view.CompleteDestinations[loc.Backend+"\n"+loc.Locator]
		loc, err = reconcileReceiptLocation(prior, exists, marker, complete, loc)
		if err != nil {
			return err
		}
		if !exists && len(record.Locations) >= MaxIndexLocations {
			return errors.New("receipt location exceeds index location bound")
		}
		before, err := json.Marshal(record)
		if err != nil {
			return err
		}
		record.Locations[loc.Identity()] = loc
		after, err := json.Marshal(record)
		if err != nil {
			return err
		}
		if len(after) > MaxMetadataBytes || len(after)-len(before) > MaxIndexBytes-bytesUsed {
			return errIndexBound
		}
		bytesUsed += len(after) - len(before)
		view.Records[id] = record
	}
	return ValidateLabelIndex(*view)
}
