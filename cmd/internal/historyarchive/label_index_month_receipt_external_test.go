package historyarchive_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
)

func monthReceiptFixture(t *testing.T) (ha.ContinuousConfig, ha.CatalogRecord, ha.DriveDestination, ha.UploadReceipt, string) {
	t.Helper()
	cfg, records, root := driveFixture(t, 1, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	fake := monthFixtureRemote("2025", "01")
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC) }}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.Verified != 1 {
		t.Fatalf("upload %+v err=%v", got, err)
	}
	dest, err := ha.ParseDriveDestination(cfg.DriveRemote, cfg.DriveFolder)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg.Catalog+".uploads", dest.Namespace(), records[0].Label.ID+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := ha.DecodeUploadReceipt(raw)
	if err != nil {
		t.Fatal(err)
	}
	return cfg, records[0], dest, receipt, path
}

func TestMonthReceiptProjectionUsesPhysicalLocation(t *testing.T) {
	cfg, record, _, receipt, path := monthReceiptFixture(t)
	receipt.Extensions = map[string]json.RawMessage{"future_receipt": json.RawMessage(`{"keep":[1,2]}`)}
	receipt.Bundle.Extensions = map[string]json.RawMessage{"future_object": json.RawMessage(`"opaque"`)}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, sync := range []bool{false, true} {
		var view ha.LabelIndex
		if sync {
			view, err = ha.SyncLocalLabelIndex(context.Background(), cfg, indexSyncTime)
		} else {
			view, err = ha.LoadLabelView(context.Background(), cfg, indexSyncTime)
		}
		if err != nil {
			t.Fatal(err)
		}
		loc, ok := view.Records[record.Label.ID].Locations["drive\n"+receipt.Destination+"\n"+receipt.FolderID]
		if !ok || !loc.Available || loc.Evidence != "verified" || loc.Proof.Method != "upload-receipt" {
			t.Fatalf("physical location missing: %+v", view.Records[record.Label.ID])
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(raw) {
		t.Fatalf("receipt mutated: %v", err)
	}
}

func TestMonthReceiptProjectionRejectsWrongLayout(t *testing.T) {
	for _, relative := range []string{"/2025/02", "/2025/01/extra", "/../outside"} {
		t.Run(relative, func(t *testing.T) {
			cfg, _, dest, receipt, path := monthReceiptFixture(t)
			receipt.Destination = dest.String() + relative
			raw, err := json.Marshal(receipt)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err = ha.LoadLabelView(context.Background(), cfg, indexSyncTime); err == nil {
				t.Fatal("accepted receipt outside bundle layout")
			}
		})
	}
}

func TestMonthReceiptRespectsCompleteObservation(t *testing.T) {
	cfg, record, _, receipt, path := monthReceiptFixture(t)
	store, err := ha.OpenLabelIndexStore(context.Background(), cfg.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	marker := ha.DestinationObservation{Backend: "drive", Locator: receipt.Destination, FolderID: receipt.FolderID, ObservedAt: indexSyncTime}
	snapshot := ha.LabelIndex{Version: ha.Version, Records: map[string]ha.LabelIndexRecord{}, CompleteDestinations: map[string]ha.DestinationObservation{marker.Identity(): marker}}
	if err = store.Commit(snapshot); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	for _, newer := range []bool{false, true} {
		if newer {
			receipt.VerifiedAt = indexSyncTime.Add(time.Hour)
			raw, e := json.Marshal(receipt)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(path, raw, 0600); e != nil {
				t.Fatal(e)
			}
		}
		view, e := ha.LoadLabelView(context.Background(), cfg, indexSyncTime.Add(2*time.Hour))
		if e != nil {
			t.Fatal(e)
		}
		location := view.Records[record.Label.ID].Locations["drive\n"+receipt.Destination+"\n"+receipt.FolderID]
		if location.Available != newer || location.Proof.Method != "upload-receipt" {
			t.Fatalf("newer=%v location=%+v", newer, location)
		}
	}
}
