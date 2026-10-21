package historyarchive_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

// Keep the old root objects meaningful while enabling the fake's pinned graph.
func monthRestoreFixture(t *testing.T, count int) (ha.Restorer, ha.RestoreRequest, []ha.CatalogRecord, *archivetest.FakeRclone, *driveRestoreDecoder) {
	t.Helper()
	r, req, records, fake, decoder := driveRestoreFixture(t, count)
	fake.ChildFolders = map[string]map[string][]archivetest.FakeRemoteObject{}
	fake.FolderObjects = map[string]map[string][]archivetest.FakeRemoteObject{}
	index, err := ha.ReadLabelIndex(r.Config.Catalog + ".index.json")
	if err != nil {
		t.Fatal(err)
	}
	root, err := ha.ParseDriveDestination(r.Config.DriveRemote, r.Config.DriveFolder)
	if err != nil {
		t.Fatal(err)
	}
	for n, record := range records {
		if n%2 != 0 {
			continue
		}
		month, err := (ha.DriveArchiveLayout{Root: root}).MonthDestination(record.Label)
		if err != nil {
			t.Fatal(err)
		}
		parts := strings.Split(strings.TrimPrefix(month.Folder, root.Folder+"/"), "/")
		fake.ChildFolders["folder-1"] = map[string][]archivetest.FakeRemoteObject{parts[0]: {{ID: "year", IsDir: true}}}
		fake.ChildFolders["year"] = map[string][]archivetest.FakeRemoteObject{parts[1]: {{ID: "month", IsDir: true}}}
		if fake.FolderObjects["month"] == nil {
			fake.FolderObjects["month"] = map[string][]archivetest.FakeRemoteObject{}
		}
		for _, name := range []string{record.Label.ID + ".json", record.Label.ID + ".tar.zst.gpg"} {
			fake.FolderObjects["month"][name] = fake.Objects[name]
			delete(fake.Objects, name)
		}
		entry := index.Records[record.Label.ID]
		locations := map[string]ha.BundleLocation{}
		for _, loc := range entry.Locations {
			loc.Locator = month.String()
			loc.FolderID = "month"
			locations[loc.Identity()] = loc
		}
		entry.Locations = locations
		index.Records[record.Label.ID] = entry
	}
	index.CompleteDestinations = nil
	writeRestoreMonthIndex(t, r.Config, index)
	fake.FolderLists = nil
	return r, req, records, fake, decoder
}
func writeRestoreMonthIndex(t *testing.T, cfg ha.ContinuousConfig, index ha.LabelIndex) {
	t.Helper()
	raw, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(cfg.Catalog+".index.json", raw, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestRestorerRecordedMonthAndLegacyUsePinnedGroups(t *testing.T) {
	r, req, _, fake, decoder := monthRestoreFixture(t, 8)
	fake.ChildFolders["folder-1"]["1900"] = []archivetest.FakeRemoteObject{{ID: "unrelated-year", IsDir: true}}
	unrelatedMonth := "12"
	if _, ok := fake.ChildFolders["year"][unrelatedMonth]; ok {
		unrelatedMonth = "11"
	}
	fake.ChildFolders["year"][unrelatedMonth] = []archivetest.FakeRemoteObject{{ID: "unrelated-month", IsDir: true}}
	fake.FolderObjects["month"][strings.Repeat("f", 64)+".tar.zst.gpg"] = []archivetest.FakeRemoteObject{{ID: "unselected", Data: []byte("unselected")}}
	report, err := r.Run(context.Background(), req)
	if fake.Downloads != 8 {
		t.Fatalf("unselected objects downloaded: %d", fake.Downloads)
	}
	if err != nil || report.State.Phase != ha.RestorePublished || decoder.Calls != 8 {
		t.Fatalf("mixed restore: %+v calls=%d err=%v", report, decoder.Calls, err)
	}
	if fake.DownloadBatches != 2 || fake.Lists != 3 || fake.FolderLists["folder-1"] != 1 || fake.FolderLists["year"] != 1 || fake.FolderLists["month"] != 1 || fake.Mkdirs != 0 {
		t.Fatalf("unexpected traversal: lists=%v batches=%d mkdir=%d", fake.FolderLists, fake.DownloadBatches, fake.Mkdirs)
	}
	assertDriveRestoreStageGone(t, req)
}
func TestRestorerRecordedMonthRefusesUnsafeSources(t *testing.T) {
	for _, kind := range []string{"missing-record", "outside", "wrong-month", "rebound", "duplicate-month", "duplicate-object", "corrupt"} {
		t.Run(kind, func(t *testing.T) {
			r, req, records, fake, decoder := monthRestoreFixture(t, 1)
			index, err := ha.ReadLabelIndex(r.Config.Catalog + ".index.json")
			if err != nil {
				t.Fatal(err)
			}
			record := index.Records[records[0].Label.ID]
			switch kind {
			case "missing-record":
				record.Locations = map[string]ha.BundleLocation{}
			case "outside", "wrong-month":
				changed := map[string]ha.BundleLocation{}
				for _, loc := range record.Locations {
					loc.Locator = "backup:unrelated"
					if kind == "wrong-month" {
						root, _ := ha.ParseDriveDestination(r.Config.DriveRemote, r.Config.DriveFolder)
						loc.Locator = root.String() + "/1900/01"
					}
					changed[loc.Identity()] = loc
				}
				record.Locations = changed
			case "rebound":
				for name, children := range fake.ChildFolders["year"] {
					children[0].ID = "other"
					fake.ChildFolders["year"][name] = children
				}
			case "duplicate-month":
				for name, children := range fake.ChildFolders["year"] {
					fake.ChildFolders["year"][name] = append(children, archivetest.FakeRemoteObject{ID: "duplicate", IsDir: true})
				}
			case "duplicate-object":
				name := records[0].Label.ID + ".tar.zst.gpg"
				fake.FolderObjects["month"][name] = append(fake.FolderObjects["month"][name], fake.FolderObjects["month"][name][0])
			case "corrupt":
				// A good older legacy copy must not hide a corrupt selected month copy.
				name := records[0].Label.ID + ".tar.zst.gpg"
				object := fake.FolderObjects["month"][name][0]
				object.Data = append([]byte(nil), object.Data...)
				fake.Objects[name] = []archivetest.FakeRemoteObject{object}
				for _, loc := range record.Locations {
					loc.LastObserved = loc.LastObserved.Add(-time.Second)
					loc.FirstObserved = loc.LastObserved
					root, _ := ha.ParseDriveDestination(r.Config.DriveRemote, r.Config.DriveFolder)
					loc.Locator = root.String()
					loc.FolderID = "folder-1"
					loc.Proof = ha.LocationProof{}
					loc.Evidence = "observed"
					record.Locations[loc.Identity()] = loc
					break
				}
				fake.FolderObjects["month"][records[0].Label.ID+".tar.zst.gpg"][0].Data[0] ^= 1
			}
			index.Records[records[0].Label.ID] = record
			writeRestoreMonthIndex(t, r.Config, index)
			report, err := r.Run(context.Background(), req)
			assertDriveRestoreRefused(t, report, err, req, decoder)
			if (kind == "outside" || kind == "wrong-month" || kind == "missing-record") && fake.Invocations != 0 {
				t.Fatalf("invalid metadata reached remote: %d", fake.Invocations)
			}
			if kind == "rebound" && fake.FolderLists["other"] != 0 {
				t.Fatal("listed rebound leaf folder before validating its ID")
			}
			if kind == "corrupt" && fake.DownloadBatches != 1 {
				t.Fatalf("corrupt selected source triggered fallback: %d downloads: %v", fake.DownloadBatches, err)
			}
			if kind != "corrupt" && fake.DownloadBatches != 0 {
				t.Fatal("unsafe source downloaded")
			}
			if fake.Mkdirs != 0 {
				t.Fatal("restore created remote folders")
			}
		})
	}
}

// The native local backend omits directory IDs during root resolution. Keep
// ordinary object inventory identities intact so this tests only that seam.
type restoreLocalRootIDRunner struct{ fake *archivetest.FakeRclone }

func (r restoreLocalRootIDRunner) Run(ctx context.Context, args []string, limit int) ([]byte, error) {
	raw, err := r.fake.Run(ctx, args, limit)
	if err != nil || args[0] != "lsjson" {
		return raw, err
	}
	resolution := false
	for _, arg := range args {
		if arg == "--dirs-only" {
			resolution = true
		}
	}
	if !resolution {
		return raw, nil
	}
	var entries []ha.RemoteEntry
	if err = json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	for n := range entries {
		entries[n].ID = ""
	}
	return json.Marshal(entries)
}
func TestRestorerLocalConformanceRootWithoutDirectoryID(t *testing.T) {
	r, req, records, fake, decoder := driveRestoreFixture(t, 1)
	root, err := ha.ParseDriveDestination(r.Config.DriveRemote, r.Config.DriveFolder)
	if err != nil {
		t.Fatal(err)
	}
	index, err := ha.ReadLabelIndex(r.Config.Catalog + ".index.json")
	if err != nil {
		t.Fatal(err)
	}
	record := index.Records[records[0].Label.ID]
	locations := map[string]ha.BundleLocation{}
	for _, loc := range record.Locations {
		loc.FolderID = root.String()
		locations[loc.Identity()] = loc
	}
	record.Locations = locations
	index.Records[records[0].Label.ID] = record
	index.CompleteDestinations = nil
	writeRestoreMonthIndex(t, r.Config, index)
	r.Rclone = ha.Rclone{Runner: restoreLocalRootIDRunner{fake: fake}, LocalConformance: true}
	report, err := r.Run(context.Background(), req)
	if err != nil || report.State.Phase != ha.RestorePublished || decoder.Calls != 1 {
		t.Fatalf("local backend root fallback: %+v decoder=%d err=%v", report, decoder.Calls, err)
	}
	if fake.DownloadBatches != 1 {
		t.Fatalf("download batches=%d", fake.DownloadBatches)
	}
	assertDriveRestoreStageGone(t, req)
}
