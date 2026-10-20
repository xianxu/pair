package historyarchive

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func labelViewFixture(t *testing.T) (ContinuousConfig, CatalogRecord, UploadReceipt, time.Time) {
	t.Helper()
	req, codec := archiveFixture(t)
	records, err := (Archiver{Codec: codec}).Create(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	cfg := ContinuousConfig{Sink: req.Sink, Catalog: req.Catalog, DriveRemote: "fixture", DriveFolder: "archive/history"}
	record := records[0]
	raw, err := os.ReadFile(filepath.Join(cfg.Sink, record.LabelKey))
	if err != nil {
		t.Fatal(err)
	}
	dest, err := ParseDriveDestination(cfg.DriveRemote, cfg.DriveFolder)
	if err != nil {
		t.Fatal(err)
	}
	sum, md := sha256.Sum256(raw), md5.Sum(raw)
	receipt := UploadReceipt{Version: Version, Destination: dest.String(), FolderID: "observed-folder", BundleID: record.Label.ID, VerifiedAt: record.VerifiedAt,
		Bundle: UploadObject{Name: record.Label.ID + ".tar.zst.gpg", Size: record.Label.Bytes, SHA256: record.Label.SHA256, MD5: strings.Repeat("a", 32), RemoteID: "bundle-object", Verification: "backend-sha256"},
		Label:  UploadObject{Name: record.Label.ID + ".json", Size: int64(len(raw)), SHA256: hex.EncodeToString(sum[:]), MD5: hex.EncodeToString(md[:]), RemoteID: "label-object", Verification: "backend-sha256"}}
	writeViewReceipt(t, cfg, receipt)
	return cfg, record, receipt, record.VerifiedAt.Add(time.Hour)
}
func writeViewReceipt(t *testing.T, cfg ContinuousConfig, receipt UploadReceipt) {
	t.Helper()
	dest, err := ParseDriveDestination(cfg.DriveRemote, cfg.DriveFolder)
	if err != nil {
		t.Fatal(err)
	}
	key := dest.Namespace() + "/" + receipt.BundleID + ".json"
	if err := PublishUploadReceipt(context.Background(), LocalSink{Root: cfg.Catalog + ".uploads"}, key, receipt); err != nil {
		t.Fatal(err)
	}
}
func writeViewIndex(t *testing.T, cfg ContinuousConfig, i LabelIndex) {
	t.Helper()
	store, err := OpenLabelIndexStore(context.Background(), cfg.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Commit(i); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLabelViewReadsExistingArchiveAndUploadReceiptOffline(t *testing.T) {
	cfg, record, receipt, at := labelViewFixture(t)
	root := filepath.Dir(filepath.Dir(cfg.Catalog))
	before := catalogReadSnapshot(t, root)
	view, err := LoadLabelView(context.Background(), cfg, at)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := view.Records[record.Label.ID]
	if !ok || len(got.Locations) != 2 {
		t.Fatalf("locations: %+v", got)
	}
	local := got.Locations["local\n"+cfg.Sink+"\n"]
	drive := got.Locations["drive\n"+receipt.Destination+"\n"+receipt.FolderID]
	if !local.Available || local.Proof.Method != "local-catalog" {
		t.Fatalf("local evidence: %+v", local)
	}
	if !drive.Available || drive.Proof.Method != "upload-receipt" || drive.Evidence != "verified" || !drive.LastObserved.Equal(receipt.VerifiedAt) || !drive.FirstObserved.Equal(receipt.VerifiedAt) {
		t.Fatalf("Drive receipt evidence: %+v", drive)
	}
	if after := catalogReadSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("offline view wrote or mutated metadata")
	}
	if _, err := os.Stat(cfg.Catalog + ".index.json"); !os.IsNotExist(err) {
		t.Fatal("view created index")
	}
}

func TestLoadLabelViewLocalDiscoverySurvivesInvalidOptionalDriveConfiguration(t *testing.T) {
	cfg, record, _, at := labelViewFixture(t)
	cfg.DriveRemote = "invalid:remote"
	view, err := LoadLabelView(context.Background(), cfg, at)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := view.Records[record.Label.ID]; !ok || len(got.Locations) != 1 {
		t.Fatal("invalid optional Drive configuration blocked local labels", got)
	}
}

func TestLoadLabelViewMissingMetadataCreatesNothing(t *testing.T) {
	root := filepath.Dir(catalogPath(t))
	cfg := ContinuousConfig{Catalog: filepath.Join(root, "missing", "history"), Sink: filepath.Join(root, "missing-sink")}
	before := catalogReadSnapshot(t, root)
	got, err := LoadLabelView(context.Background(), cfg, time.Now().UTC())
	if err != nil || got.Version != Version || got.Records == nil || len(got.Records) != 0 {
		t.Fatalf("empty view: %+v %v", got, err)
	}
	if after := catalogReadSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("empty view created paths")
	}
}

func TestLoadLabelViewRecoversSidecarsWithoutCatalog(t *testing.T) {
	cfg, record, _, at := labelViewFixture(t)
	if err := os.Remove(cfg.Catalog); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(cfg.Catalog + ".lock"); err != nil {
		t.Fatal(err)
	}
	before := catalogReadSnapshot(t, filepath.Dir(filepath.Dir(cfg.Catalog)))
	got, err := LoadLabelView(context.Background(), cfg, at)
	if err != nil {
		t.Fatal(err)
	}
	r := got.Records[record.Label.ID]
	loc := r.Locations["local\n"+cfg.Sink+"\n"]
	if !loc.Available || loc.Evidence != "observed" || loc.Proof.Method != "" || len(r.Locations) != 2 {
		t.Fatalf("sidecar evidence: %+v", r)
	}
	if after := catalogReadSnapshot(t, filepath.Dir(filepath.Dir(cfg.Catalog))); !reflect.DeepEqual(before, after) {
		t.Fatal("sidecar view created catalog")
	}
}

func TestLoadLabelViewPreservesUnknownAndUnavailableDriveHistory(t *testing.T) {
	cfg, record, receipt, at := labelViewFixture(t)
	old := LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}, Extensions: map[string]json.RawMessage{"future": json.RawMessage(`{"keep":[1,2]}`)}}
	loc := BundleLocation{Backend: "drive", Locator: receipt.Destination, FolderID: receipt.FolderID, BundleKey: receipt.Bundle.Name, LabelKey: receipt.Label.Name, Available: false, Evidence: "verified", FirstObserved: receipt.VerifiedAt, LastObserved: at, Proof: LocationProof{Method: "backend-sha256", SHA256: record.Label.SHA256, VerifiedAt: receipt.VerifiedAt}, Extensions: map[string]json.RawMessage{"new_fact": json.RawMessage(`true`)}}
	old.Records[record.Label.ID] = LabelIndexRecord{Label: record.Label, Locations: map[string]BundleLocation{loc.Identity(): loc}, Extensions: map[string]json.RawMessage{"record_future": json.RawMessage(`"yes"`)}}
	writeViewIndex(t, cfg, old)
	got, err := LoadLabelView(context.Background(), cfg, at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	drive := got.Records[record.Label.ID].Locations[loc.Identity()]
	if drive.Available || drive.Proof.Method != "backend-sha256" || !drive.LastObserved.Equal(at) {
		t.Fatalf("old receipt resurrected Drive availability: %+v", drive)
	}
	if string(got.Extensions["future"]) != `{"keep":[1,2]}` || string(drive.Extensions["new_fact"]) != `true` || string(got.Records[record.Label.ID].Extensions["record_future"]) != `"yes"` {
		t.Fatal("lost snapshot extensions")
	}
}

func TestLoadLabelViewRefusesReceiptConflicts(t *testing.T) {
	for _, field := range []string{"destination", "bundle-id", "bundle-sha", "bundle-size", "label-sha", "label-size", "name"} {
		t.Run(field, func(t *testing.T) {
			cfg, _, receipt, at := labelViewFixture(t)
			dest, _ := ParseDriveDestination(cfg.DriveRemote, cfg.DriveFolder)
			originalID := receipt.BundleID
			switch field {
			case "destination":
				receipt.Destination = "other:folder"
			case "bundle-id":
				receipt.BundleID = strings.Repeat("b", 64)
				receipt.Bundle.Name = receipt.BundleID + ".tar.zst.gpg"
				receipt.Label.Name = receipt.BundleID + ".json"
			case "bundle-sha":
				receipt.Bundle.SHA256 = strings.Repeat("e", 64)
			case "bundle-size":
				receipt.Bundle.Size++
			case "label-sha":
				receipt.Label.SHA256 = strings.Repeat("f", 64)
			case "label-size":
				receipt.Label.Size++
			case "name":
				receipt.Bundle.Name = "wrong.tar.zst.gpg"
			}
			// Write a conflicting persisted receipt directly, including a malformed name.
			raw, err := json.Marshal(receipt)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(cfg.Catalog+".uploads", dest.Namespace(), originalID+".json")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			before := catalogReadSnapshot(t, filepath.Dir(filepath.Dir(cfg.Catalog)))
			if _, err := LoadLabelView(context.Background(), cfg, at); err == nil {
				t.Fatal("accepted conflicting receipt")
			}
			if after := catalogReadSnapshot(t, filepath.Dir(filepath.Dir(cfg.Catalog))); !reflect.DeepEqual(before, after) {
				t.Fatal("failed view mutated metadata")
			}
		})
	}
}

func TestSyncLocalLabelIndexPublishesOnlyIndexAndUsesLifetimeLock(t *testing.T) {
	cfg, record, _, at := labelViewFixture(t)
	root := filepath.Dir(filepath.Dir(cfg.Catalog))
	before := catalogReadSnapshot(t, root)
	got, err := SyncLocalLabelIndex(context.Background(), cfg, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Records) != 1 || len(got.Records[record.Label.ID].Locations) != 2 {
		t.Fatalf("sync evidence: %+v", got)
	}
	persisted, err := ReadLabelIndex(cfg.Catalog + ".index.json")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(got)
	b, _ := json.Marshal(persisted)
	if string(a) != string(b) {
		t.Fatal("sync returned different snapshot")
	}
	after := catalogReadSnapshot(t, root)
	for path, state := range before {
		if state.Mode.IsDir() && path == filepath.Dir(cfg.Catalog) {
			continue
		}
		if !reflect.DeepEqual(state, after[path]) {
			t.Fatalf("sync changed existing path %s", path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok && path != cfg.Catalog+".index.json" && path != cfg.Catalog+".index.lock" {
			t.Fatalf("sync created unrelated path %s", path)
		}
	}
	store, err := OpenLabelIndexStore(context.Background(), cfg.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := SyncLocalLabelIndex(context.Background(), cfg, at.Add(time.Hour)); err == nil {
		t.Fatal("sync bypassed lifetime lock")
	}
}

func TestLoadLabelViewRefusesBusyCatalogAndConflictingSnapshot(t *testing.T) {
	cfg, record, _, at := labelViewFixture(t)
	writer, err := OpenCatalog(context.Background(), cfg.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLabelView(context.Background(), cfg, at); err == nil {
		t.Fatal("returned partial view while catalog busy")
	}
	writer.Close()
	label := record.Label
	label.SHA256 = strings.Repeat("f", 64)
	writeViewIndex(t, cfg, LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{label.ID: {Label: label, Locations: map[string]BundleLocation{}}}})
	if _, err := LoadLabelView(context.Background(), cfg, at); err == nil {
		t.Fatal("accepted conflicting ciphertext for same bundle ID")
	}
}

func TestLoadLabelViewUsesHistoricalReceiptWithoutLocalCopies(t *testing.T) {
	cfg, record, receipt, at := labelViewFixture(t)
	old := LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{record.Label.ID: {Label: record.Label, Locations: map[string]BundleLocation{}}}}
	writeViewIndex(t, cfg, old)
	for _, path := range []string{cfg.Catalog, cfg.Catalog + ".lock", filepath.Join(cfg.Sink, record.LabelKey), filepath.Join(cfg.Sink, record.BundleKey)} {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
	before := catalogReadSnapshot(t, filepath.Dir(filepath.Dir(cfg.Catalog)))
	got, err := LoadLabelView(context.Background(), cfg, at)
	if err != nil {
		t.Fatal(err)
	}
	locations := got.Records[record.Label.ID].Locations
	if len(locations) != 1 || locations["drive\n"+receipt.Destination+"\n"+receipt.FolderID].Proof.Method != "upload-receipt" {
		t.Fatalf("historical receipt: %+v", locations)
	}
	if after := catalogReadSnapshot(t, filepath.Dir(filepath.Dir(cfg.Catalog))); !reflect.DeepEqual(before, after) {
		t.Fatal("historical read manufactured local metadata")
	}
}

func TestSyncLocalLabelIndexFailurePreservesPriorSnapshot(t *testing.T) {
	cfg, _, receipt, at := labelViewFixture(t)
	if _, err := SyncLocalLabelIndex(context.Background(), cfg, at); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(cfg.Catalog + ".index.json")
	if err != nil {
		t.Fatal(err)
	}
	dest, _ := ParseDriveDestination(cfg.DriveRemote, cfg.DriveFolder)
	receipt.Bundle.SHA256 = strings.Repeat("e", 64)
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.Catalog+".uploads", dest.Namespace(), receipt.BundleID+".json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncLocalLabelIndex(context.Background(), cfg, at.Add(time.Hour)); err == nil {
		t.Fatal("conflicting observation published")
	}
	after, err := os.ReadFile(cfg.Catalog + ".index.json")
	if err != nil || string(before) != string(after) {
		t.Fatal("failed sync changed prior snapshot")
	}
	if _, err := os.Stat(cfg.Catalog + ".index-staging"); !os.IsNotExist(err) {
		t.Fatal("failed sync leaked staging")
	}
}
