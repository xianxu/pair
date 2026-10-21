package historyarchive_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

func indexMonthsFixture(t *testing.T, count int) (ha.ContinuousConfig, []ha.CatalogRecord, *archivetest.FakeRclone) {
	t.Helper()
	stamps := make([]time.Time, count)
	for i := range stamps {
		stamps[i] = time.Date(2025, time.Month(1+i%2), 1, 0, 0, 0, 0, time.UTC)
	}
	cfg, records, _ := driveFixture(t, count, stamps...)
	fake := legacyIndexDriveFake(t, cfg, records)
	fake.ChildFolders = map[string]map[string][]archivetest.FakeRemoteObject{
		"folder-1": {"2025": {{ID: "year", IsDir: true}}},
		"year":     {"01": {{ID: "jan", IsDir: true}}, "02": {{ID: "feb", IsDir: true}}},
	}
	fake.FolderObjects = map[string]map[string][]archivetest.FakeRemoteObject{"jan": {}, "feb": {}}
	for i, record := range records {
		folder := []string{"jan", "feb"}[i%2]
		for _, suffix := range []string{".json", ".tar.zst.gpg"} {
			name := record.Label.ID + suffix
			fake.FolderObjects[folder][name] = fake.Objects[name]
			delete(fake.Objects, name)
		}
	}
	return cfg, records, fake
}

func TestDriveIndexMonthsMixedAndCounted(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			cfg, records, fake := indexMonthsFixture(t, n)
			// The same immutable ID can have independently observed physical copies.
			id := records[0].Label.ID
			for _, suffix := range []string{".json", ".tar.zst.gpg"} {
				fake.Objects[id+suffix] = fake.FolderObjects["jan"][id+suffix]
			}
			fake.ChildFolders["year"]["03"] = []archivetest.FakeRemoteObject{{ID: "empty", IsDir: true}}
			fake.ChildFolders["folder-1"]["misc"] = []archivetest.FakeRemoteObject{{ID: "ignored", IsDir: true}}
			fake.ChildFolders["year"]["13"] = []archivetest.FakeRemoteObject{{ID: "ignored-month", IsDir: true}}
			fake.ChildFolders["year"]["0a"] = []archivetest.FakeRemoteObject{{ID: "ignored-lexical-month", IsDir: true}}
			report, err := runIndexSync(context.Background(), cfg, fake)
			if err != nil || report.LabelCount != n+1 || report.Verified != n+1 {
				t.Fatalf("mixed sync: %+v %v", report, err)
			}
			idx, err := ha.ReadLabelIndex(cfg.Catalog + ".index.json")
			if err != nil || len(idx.Records) != n || len(idx.Records[id].Locations) != 2 || len(idx.CompleteDestinations) != 4 {
				t.Fatalf("mixed snapshot: records=%d complete=%d err=%v", len(idx.Records), len(idx.CompleteDestinations), err)
			}
			if fake.FolderLists["ignored"] != 0 || fake.FolderLists["ignored-month"] != 0 || fake.FolderLists["ignored-lexical-month"] != 0 || fake.Mkdirs != 0 || fake.HashBatches != 0 || fake.DownloadedCipherBytes != 0 || fake.DownloadBatches != 3 {
				t.Fatalf("unexpected remote effects: %+v", fake)
			}
			// Two snapshots of each root/year/month plus two root resolutions.
			if fake.Invocations != 15 {
				t.Fatalf("N-dependent operations: %d", fake.Invocations)
			}
		})
	}
}

func TestDriveIndexMonthsFailureIsAtomic(t *testing.T) {
	for _, failure := range []string{"late-folder", "second-download", "corrupt-label", "rebound-year", "rebound-root", "duplicate-month", "file-year", "changed-pair", "new-month"} {
		t.Run(failure, func(t *testing.T) {
			cfg, records, fake := indexMonthsFixture(t, 2)
			if _, err := runIndexSync(context.Background(), cfg, fake); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(cfg.Catalog + ".index.json")
			if err != nil {
				t.Fatal(err)
			}
			fake.FolderLists = map[string]int{}
			fake.BeforeFolder = func(_ context.Context, op, folder string) error {
				if failure == "second-download" && op == "copy" && folder == "feb" {
					return errors.New("second folder download failed")
				}
				if op != "lsjson" {
					return nil
				}
				switch failure {
				case "late-folder":
					if folder == "feb" {
						return errors.New("late listing failure")
					}
				case "rebound-year":
					if folder == "folder-1" && fake.FolderLists[folder] == 1 {
						fake.ChildFolders[folder]["2025"][0].ID = "replacement"
					}
				case "rebound-root":
					if folder == "feb" {
						fake.FolderID = "replacement-root"
					}
				case "duplicate-month":
					fake.ChildFolders["year"]["01"] = []archivetest.FakeRemoteObject{{ID: "jan", IsDir: true}, {ID: "duplicate", IsDir: true}}
				case "file-year":
					fake.ChildFolders["folder-1"]["2025"][0].IsDir = false
				case "changed-pair":
					if folder == "jan" && fake.FolderLists[folder] == 1 {
						fake.FolderObjects[folder][records[0].Label.ID+".json"][0].ID = "replacement-label"
					}
				case "new-month":
					if folder == "year" && fake.FolderLists[folder] == 1 {
						fake.ChildFolders[folder]["03"] = []archivetest.FakeRemoteObject{{ID: "late-month", IsDir: true}}
					}
				case "corrupt-label":
					fake.FolderObjects["feb"][records[1].Label.ID+".json"][0].Data = []byte("invalid JSON")
				}
				return nil
			}
			if _, err = runIndexSync(context.Background(), cfg, fake); err == nil {
				t.Fatal("incomplete/unstable tree accepted")
			}
			after, err := os.ReadFile(cfg.Catalog + ".index.json")
			if err != nil || string(after) != string(before) {
				t.Fatalf("failed tree changed snapshot: %v", err)
			}
			if _, err = os.Stat(cfg.Catalog + ".index-staging"); !os.IsNotExist(err) {
				t.Fatalf("failed tree retained owned stage: %v", err)
			}
		})
	}
}

func TestDriveIndexMonthsDisappearanceRetainsHistory(t *testing.T) {
	cfg, records, fake := indexMonthsFixture(t, 2)
	if _, err := runIndexSync(context.Background(), cfg, fake); err != nil {
		t.Fatal(err)
	}
	initial, err := ha.ReadLabelIndex(cfg.Catalog + ".index.json")
	if err != nil || len(initial.Records[records[0].Label.ID].Locations) != 1 {
		t.Fatalf("missing initial month observation: %v", err)
	}
	delete(fake.ChildFolders["year"], "01")
	later := indexSyncTime.Add(time.Hour)
	if _, err := (ha.DriveIndexSync{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return later }}).Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	idx, err := ha.ReadLabelIndex(cfg.Catalog + ".index.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, loc := range idx.Records[records[0].Label.ID].Locations {
		if loc.Available || loc.Proof.Method != "backend-sha256" {
			t.Fatalf("disappearance lost proof or availability: %+v", loc)
		}
		if !idx.CompleteDestinations["drive\n"+loc.Locator].ObservedAt.Equal(later) {
			t.Fatal("absent month marker not advanced")
		}
	}
}

func TestDriveIndexMonthsCompleteRootRebindInvalidatesChildHistory(t *testing.T) {
	cfg, records, fake := indexMonthsFixture(t, 1)
	if _, err := runIndexSync(context.Background(), cfg, fake); err != nil {
		t.Fatal(err)
	}
	fake.FolderID = "rebound-root"
	fake.ChildFolders = map[string]map[string][]archivetest.FakeRemoteObject{}
	later := indexSyncTime.Add(time.Hour)
	if _, err := (ha.DriveIndexSync{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return later }}).Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	idx, err := ha.ReadLabelIndex(cfg.Catalog + ".index.json")
	if err != nil {
		t.Fatal(err)
	}
	root, _ := ha.ParseDriveDestination(cfg.DriveRemote, cfg.DriveFolder)
	if idx.CompleteDestinations["drive\n"+root.String()].FolderID != "rebound-root" {
		t.Fatal("new root binding not recorded")
	}
	if len(idx.Records[records[0].Label.ID].Locations) != 1 {
		t.Fatal("child history lost")
	}
	for _, loc := range idx.Records[records[0].Label.ID].Locations {
		if loc.Available || loc.Proof.Method != "backend-sha256" || !idx.CompleteDestinations["drive\n"+loc.Locator].ObservedAt.Equal(later) {
			t.Fatalf("old child remains current or lost proof: %+v", loc)
		}
	}
}

func TestDriveIndexMonthsAggregateBoundsBeforeDownload(t *testing.T) {
	for _, bound := range []string{"labels", "destinations"} {
		t.Run(bound, func(t *testing.T) {
			cfg, _, fake := indexMonthsFixture(t, 2)
			if bound == "destinations" {
				fake.ChildFolders = map[string]map[string][]archivetest.FakeRemoteObject{"folder-1": {}}
				for y := 2000; y < 2086; y++ {
					year := fmt.Sprint(y)
					fake.ChildFolders["folder-1"][year] = []archivetest.FakeRemoteObject{{ID: year, IsDir: true}}
					fake.ChildFolders[year] = map[string][]archivetest.FakeRemoteObject{}
					for m := 1; m <= 12; m++ {
						mon := fmt.Sprintf("%02d", m)
						fake.ChildFolders[year][mon] = []archivetest.FakeRemoteObject{{ID: year + mon, IsDir: true}}
					}
				}
			}
			runner := indexInventoryRunner{fake: fake, alter: func(entries []ha.RemoteEntry) []ha.RemoteEntry {
				if bound != "labels" {
					return entries
				}
				var label, bundle ha.RemoteEntry
				for _, e := range entries {
					if strings.HasSuffix(e.Name, ".json") {
						label = e
					}
					if strings.HasSuffix(e.Name, ".gpg") {
						bundle = e
					}
				}
				if label.Name == "" {
					return entries
				}
				var out []ha.RemoteEntry
				for i := 0; i < 9; i++ {
					l, b := label, bundle
					id := fmt.Sprintf("%064x", i)
					l.Name = id + ".json"
					l.ID = "l" + id
					l.Size = ha.MaxMetadataBytes
					b.Name = id + ".tar.zst.gpg"
					b.ID = "b" + id
					out = append(out, l, b)
				}
				return out
			}}
			if _, err := (ha.DriveIndexSync{Rclone: ha.Rclone{Runner: runner}}).Run(context.Background(), cfg); err == nil || fake.DownloadBatches != 0 {
				t.Fatalf("global %s bound: err=%v downloads=%d", bound, err, fake.DownloadBatches)
			}
		})
	}
}

func TestDriveIndexMonthsReceiptRootNamespace(t *testing.T) {
	cfg, records, fake := indexMonthsFixture(t, 1)
	// Adopt the preexisting month pair to produce a real root-scoped receipt.
	if _, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return indexSyncTime.Add(-time.Hour) }}).Run(context.Background(), cfg, cfg.Catalog+"-stage"); err != nil {
		t.Fatal(err)
	}
	fake.MissingHashes = true
	report, err := runIndexSync(context.Background(), cfg, fake)
	if err != nil || report.Verified != 1 {
		t.Fatalf("month receipt: %+v %v", report, err)
	}
	idx, err := ha.ReadLabelIndex(cfg.Catalog + ".index.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, loc := range idx.Records[records[0].Label.ID].Locations {
		if loc.Proof.Method != "upload-receipt" {
			t.Fatalf("receipt not retained: %+v", loc)
		}
	}
}

func TestDriveIndexMonthsReceiptDoesNotAttestOtherCopy(t *testing.T) {
	cfg, records, fake := indexMonthsFixture(t, 1)
	if _, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return indexSyncTime.Add(-time.Hour) }}).Run(context.Background(), cfg, cfg.Catalog+"-stage"); err != nil {
		t.Fatal(err)
	}
	id := records[0].Label.ID
	for _, suffix := range []string{".json", ".tar.zst.gpg"} {
		fake.Objects[id+suffix] = fake.FolderObjects["jan"][id+suffix]
	}
	fake.MissingHashes = true
	report, err := runIndexSync(context.Background(), cfg, fake)
	if err != nil || report.Verified != 1 || report.Observed != 1 {
		t.Fatalf("physical receipt leaked or rejected independent copy: %+v %v", report, err)
	}
}

func TestDriveIndexMonthsUnknownFieldsSurviveCompleteRefresh(t *testing.T) {
	cfg, records, fake := indexMonthsFixture(t, 2)
	name := records[0].Label.ID + ".json"
	var label map[string]json.RawMessage
	if err := json.Unmarshal(fake.FolderObjects["jan"][name][0].Data, &label); err != nil {
		t.Fatal(err)
	}
	label["future_month_label"] = json.RawMessage(`{"number":12345678901234567890}`)
	raw, err := json.Marshal(label)
	if err != nil {
		t.Fatal(err)
	}
	fake.FolderObjects["jan"][name][0].Data = raw
	if _, err = runIndexSync(context.Background(), cfg, fake); err != nil {
		t.Fatal(err)
	}
	store, err := ha.OpenLabelIndexStore(context.Background(), cfg.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	idx.Extensions = map[string]json.RawMessage{"future_index": json.RawMessage(`true`)}
	for key, marker := range idx.CompleteDestinations {
		marker.Extensions = map[string]json.RawMessage{"future_marker": json.RawMessage(`42`)}
		idx.CompleteDestinations[key] = marker
	}
	if err = store.Commit(idx); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = (ha.DriveIndexSync{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return indexSyncTime.Add(time.Hour) }}).Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	idx, err = ha.ReadLabelIndex(cfg.Catalog + ".index.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(idx.Extensions["future_index"]) != "true" || string(idx.Records[records[0].Label.ID].Label.Extensions["future_month_label"]) != `{"number":12345678901234567890}` {
		t.Fatal("unknown index/label data lost")
	}
	for _, marker := range idx.CompleteDestinations {
		if string(marker.Extensions["future_marker"]) != "42" {
			t.Fatal("unknown complete marker lost")
		}
	}
}
