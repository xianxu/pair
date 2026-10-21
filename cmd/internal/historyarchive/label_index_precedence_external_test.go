package historyarchive_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

func precedenceFixture(t *testing.T, count int) (ha.ContinuousConfig, []ha.CatalogRecord, *archivetest.FakeRclone, ha.DriveDestination, []ha.UploadReceipt) {
	t.Helper()
	cfg, records, root := driveFixture(t, count)
	fake := legacyIndexDriveFake(t, cfg, records)
	if report, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive")); err != nil || report.Verified != count {
		t.Fatalf("upload: %+v %v", report, err)
	}
	dest, err := ha.ParseDriveDestination(cfg.DriveRemote, cfg.DriveFolder)
	if err != nil {
		t.Fatal(err)
	}
	var receipts []ha.UploadReceipt
	for _, record := range records {
		raw, err := os.ReadFile(filepath.Join(cfg.Catalog+".uploads", dest.Namespace(), record.Label.ID+".json"))
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := ha.DecodeUploadReceipt(raw)
		if err != nil {
			t.Fatal(err)
		}
		receipts = append(receipts, receipt)
	}
	return cfg, records, fake, dest, receipts
}

func assertPrecedenceView(t *testing.T, view ha.LabelIndex, cfg ha.ContinuousConfig, records []ha.CatalogRecord, receipts []ha.UploadReceipt, availability []bool) {
	t.Helper()
	if len(view.Records) != len(records) {
		t.Fatalf("local records lost: %d", len(view.Records))
	}
	for n, record := range records {
		got, ok := view.Records[record.Label.ID]
		if !ok {
			t.Fatalf("local record missing: %s", record.Label.ID)
		}
		local, ok := got.Locations["local\n"+cfg.Sink+"\n"]
		if !ok || !local.Available || local.Proof.Method != "local-catalog" {
			t.Fatalf("local evidence lost: %+v", local)
		}
		receipt := receipts[n]
		drive, ok := got.Locations["drive\n"+receipt.Destination+"\n"+receipt.FolderID]
		if !ok || drive.Available != availability[n] || drive.Evidence != "verified" || drive.Proof.Method != "upload-receipt" || drive.Proof.SHA256 != record.Label.SHA256 || !drive.Proof.VerifiedAt.Equal(receipt.VerifiedAt) {
			t.Fatalf("receipt %s availability=%v: %+v", record.Label.ID, availability[n], drive)
		}
	}
}

func assertPrecedenceBothViews(t *testing.T, cfg ha.ContinuousConfig, records []ha.CatalogRecord, receipts []ha.UploadReceipt, availability []bool, at time.Time) {
	t.Helper()
	view, err := ha.LoadLabelView(context.Background(), cfg, at)
	if err != nil {
		t.Fatal(err)
	}
	assertPrecedenceView(t, view, cfg, records, receipts, availability)
	view, err = ha.SyncLocalLabelIndex(context.Background(), cfg, at)
	if err != nil {
		t.Fatal(err)
	}
	assertPrecedenceView(t, view, cfg, records, receipts, availability)
}

func TestDriveIndexCompleteEmptySuppressesUnseenOldReceipt(t *testing.T) {
	cfg, records, fake, _, receipts := precedenceFixture(t, 1)
	fake.Objects = map[string][]archivetest.FakeRemoteObject{}
	if report, err := runIndexSync(context.Background(), cfg, fake); err != nil || report.LabelCount != 0 {
		t.Fatalf("initial empty sync: %+v %v", report, err)
	}
	assertPrecedenceBothViews(t, cfg, records, receipts, []bool{false}, indexSyncTime.Add(time.Hour))
}

func TestDriveIndexCompleteFolderRebindSuppressesUnseenOldReceipt(t *testing.T) {
	cfg, records, fake, _, receipts := precedenceFixture(t, 1)
	fake.FolderID = "new-folder"
	fake.Objects = map[string][]archivetest.FakeRemoteObject{}
	if report, err := runIndexSync(context.Background(), cfg, fake); err != nil || report.LabelCount != 0 {
		t.Fatalf("rebound empty sync: %+v %v", report, err)
	}
	assertPrecedenceBothViews(t, cfg, records, receipts, []bool{false}, indexSyncTime.Add(time.Hour))
}

func TestDriveIndexNewerSameBindingReceiptPromotesOnlyOwnID(t *testing.T) {
	cfg, records, fake, dest, receipts := precedenceFixture(t, 2)
	fake.Objects = map[string][]archivetest.FakeRemoteObject{}
	if _, err := runIndexSync(context.Background(), cfg, fake); err != nil {
		t.Fatal(err)
	}
	// Preserve every attested object field; only the verification time advances
	// for this one receipt, modeling a later confirmed upload at the same binding.
	receipts[0].VerifiedAt = indexSyncTime.Add(time.Hour)
	if err := ha.ValidateUploadReceipt(receipts[0]); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(receipts[0])
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg.Catalog+".uploads", dest.Namespace(), records[0].Label.ID+".json")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	assertPrecedenceBothViews(t, cfg, records, receipts, []bool{true, false}, indexSyncTime.Add(2*time.Hour))
}

func TestDriveIndexPartialRefreshPreservesCompleteDestination(t *testing.T) {
	cfg, _, fake, dest, _ := precedenceFixture(t, 1)
	objects := fake.Objects
	fake.Objects = map[string][]archivetest.FakeRemoteObject{}
	if _, err := runIndexSync(context.Background(), cfg, fake); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(cfg.Catalog + ".index.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot map[string]json.RawMessage
	if err = json.Unmarshal(before, &snapshot); err != nil {
		t.Fatal(err)
	}
	var complete map[string]json.RawMessage
	if err = json.Unmarshal(snapshot["complete_destinations"], &complete); err != nil {
		t.Fatalf("missing complete-destination marker: %v", err)
	}
	if _, ok := complete["drive\n"+dest.String()]; !ok {
		t.Fatal("configured Drive completion not recorded")
	}
	fake.Objects = objects
	fake.FailAfterObjects = 1
	if _, err = (ha.DriveIndexSync{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return indexSyncTime.Add(time.Hour) }}).Run(context.Background(), cfg); err == nil {
		t.Fatal("partial refresh succeeded")
	}
	after, err := os.ReadFile(cfg.Catalog + ".index.json")
	if err != nil || string(after) != string(before) {
		t.Fatalf("partial refresh changed complete marker/snapshot: %v", err)
	}
}
