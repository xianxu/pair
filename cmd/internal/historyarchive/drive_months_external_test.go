package historyarchive_test

import (
	"context"
	"fmt"
	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDriveMirrorWritesMonthWithoutListingUnrelatedHistory(t *testing.T) {
	stamp := time.Date(2025, 12, 31, 23, 59, 0, 0, time.UTC)
	cfg, records, root := driveFixture(t, 1, stamp)
	fake := &archivetest.FakeRclone{ChildFolders: map[string]map[string][]archivetest.FakeRemoteObject{
		"folder-1": {"2024": {{ID: "old-year", IsDir: true}}},
		"old-year": {"01": {{ID: "old-month", IsDir: true}}},
	}}
	mirror := ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC) }}
	got, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.Verified != 1 {
		t.Fatalf("report=%+v err=%v", got, err)
	}
	years := fake.ChildFolders["folder-1"]["2025"]
	if len(years) != 1 {
		t.Fatalf("new year missing: %+v", fake.ChildFolders)
	}
	months := fake.ChildFolders[years[0].ID]["12"]
	if len(months) != 1 {
		t.Fatalf("new month missing: %+v", fake.ChildFolders)
	}
	for _, name := range []string{records[0].Label.ID + ".tar.zst.gpg", records[0].Label.ID + ".json"} {
		if len(fake.FolderObjects[months[0].ID][name]) != 1 || len(fake.Objects[name]) != 0 {
			t.Fatalf("object not uniquely under month: %s", name)
		}
	}
	if fake.FolderLists["old-year"] != 0 || fake.FolderLists["old-month"] != 0 {
		t.Fatalf("enumerated old history: %+v", fake.FolderLists)
	}
	copies := fake.Copies
	again, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || again.AlreadyRecorded != 1 || fake.Copies != copies {
		t.Fatalf("rerun %+v err=%v copies=%d", again, err, fake.Copies)
	}
	if fake.FolderLists[months[0].ID] != 2 {
		t.Fatalf("receipt caused historical month rescan: %+v", fake.FolderLists)
	}
}

func TestDriveMirrorAdoptsLegacyPairWithoutMigration(t *testing.T) {
	cfg, records, root := driveFixture(t, 1, time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))
	record := records[0]
	fake := &archivetest.FakeRclone{Objects: map[string][]archivetest.FakeRemoteObject{}, ChildFolders: map[string]map[string][]archivetest.FakeRemoteObject{}}
	for name, key := range map[string]string{record.Label.ID + ".tar.zst.gpg": record.BundleKey, record.Label.ID + ".json": record.LabelKey} {
		raw, err := os.ReadFile(filepath.Join(cfg.Sink, key))
		if err != nil {
			t.Fatal(err)
		}
		fake.Objects[name] = []archivetest.FakeRemoteObject{{ID: name, Data: raw}}
	}
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.Verified != 1 || fake.Mkdirs != 0 || fake.Copies != 0 {
		t.Fatalf("legacy %+v err=%v mkdir=%d copies=%d", got, err, fake.Mkdirs, fake.Copies)
	}
}

func TestDriveMirrorDoesNotDuplicatePartialLegacyPair(t *testing.T) {
	cfg, records, root := driveFixture(t, 1)
	raw, err := os.ReadFile(filepath.Join(cfg.Sink, records[0].BundleKey))
	if err != nil {
		t.Fatal(err)
	}
	fake := &archivetest.FakeRclone{Objects: map[string][]archivetest.FakeRemoteObject{records[0].Label.ID + ".tar.zst.gpg": {{ID: "legacy", Data: raw}}}, ChildFolders: map[string]map[string][]archivetest.FakeRemoteObject{}}
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err == nil || got.Verified != 0 || got.Pending != 1 || fake.Copies != 0 || fake.Mkdirs != 0 {
		t.Fatalf("partial legacy %+v err=%v copies=%d", got, err, fake.Copies)
	}
}

func monthFixtureRemote(year string, months ...string) *archivetest.FakeRclone {
	fake := &archivetest.FakeRclone{ChildFolders: map[string]map[string][]archivetest.FakeRemoteObject{
		"folder-1":     {year: {{ID: "year-" + year, IsDir: true}}},
		"year-" + year: {},
	}, FolderObjects: map[string]map[string][]archivetest.FakeRemoteObject{}}
	for _, month := range months {
		fake.ChildFolders["year-"+year][month] = []archivetest.FakeRemoteObject{{ID: "month-" + month, IsDir: true}}
	}
	return fake
}

func TestDriveMirrorMonthFailurePreservesOtherProgress(t *testing.T) {
	cfg, _, root := driveFixture(t, 2, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	fake := monthFixtureRemote("2025", "01", "02")
	fake.BeforeFolder = func(_ context.Context, op, folder string) error {
		if op == "lsjson" && folder == "month-01" {
			return fmt.Errorf("January unavailable")
		}
		return nil
	}
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC) }}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err == nil || got.Verified != 1 || got.Pending != 1 || fake.CopyBatches != 1 {
		t.Fatalf("report=%+v err=%v copies=%d", got, err, fake.CopyBatches)
	}
	if fake.FolderLists["year-2025"] != 1 {
		t.Fatalf("year rescanned: %+v", fake.FolderLists)
	}
}

func TestDriveMirrorMonthGroupsShareByteBudget(t *testing.T) {
	cfg, records, root := driveFixture(t, 2, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	var limit int64
	for _, record := range records {
		label, err := os.ReadFile(filepath.Join(cfg.Sink, record.LabelKey))
		if err != nil {
			t.Fatal(err)
		}
		size := record.Label.Bytes + int64(len(label))
		if size > limit {
			limit = size
		}
	}
	fake := monthFixtureRemote("2025", "01", "02")
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}, BatchByteLimit: limit}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.Verified != 1 || got.Pending != 1 || got.Deferred != 1 || got.StagedBytes > limit || fake.CopyBatches != 1 {
		t.Fatalf("report=%+v err=%v limit=%d batches=%d", got, err, limit, fake.CopyBatches)
	}
	if fake.FolderLists["month-02"] != 0 {
		t.Fatalf("deferred month visited: %+v", fake.FolderLists)
	}
}

func TestDriveMirrorMonthReceiptUsesConfiguredNamespace(t *testing.T) {
	cfg, records, root := driveFixture(t, 1, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	fake := monthFixtureRemote("2025", "01")
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.Verified != 1 {
		t.Fatalf("report=%+v err=%v", got, err)
	}
	dest, err := ha.ParseDriveDestination(cfg.DriveRemote, cfg.DriveFolder)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(cfg.Catalog+".uploads", dest.Namespace(), records[0].Label.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := ha.DecodeUploadReceipt(raw)
	if err != nil || receipt.Destination != dest.String()+"/2025/01" || receipt.FolderID != "month-01" {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}

func TestDriveMirrorMonthCommandGrowth(t *testing.T) {
	for _, months := range []int{1, 2} {
		t.Run(fmt.Sprint(months), func(t *testing.T) {
			stamps := []time.Time{time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
			if months == 2 {
				stamps = append(stamps, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
			}
			cfg, _, root := driveFixture(t, months, stamps...)
			fake := monthFixtureRemote("2025", "01", "02")
			fake.MissingHashes = true
			got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return stamps[0] }}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
			// Root resolution + root/year inventories + two inventories, one copy
			// and one downloaded-checksum batch per active physical month.
			if err != nil || got.Verified != months || fake.CopyBatches != months || fake.HashBatches != months || fake.Invocations != 3+4*months {
				t.Fatalf("months=%d report=%+v err=%v calls=%d", months, got, err, fake.Invocations)
			}
		})
	}
}

func TestDriveMirrorParentCapacityAcrossCreates(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		t.Run(fmt.Sprintf("uncertain=%v", uncertain), func(t *testing.T) {
			cfg, _, root := driveFixture(t, 2, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
			fake := &archivetest.FakeRclone{Objects: map[string][]archivetest.FakeRemoteObject{"occupied": make([]archivetest.FakeRemoteObject, 99999)}, ChildFolders: map[string]map[string][]archivetest.FakeRemoteObject{}}
			for i := range fake.Objects["occupied"] {
				fake.Objects["occupied"][i] = archivetest.FakeRemoteObject{ID: fmt.Sprint(i), IsDir: true}
			}
			rootReads := 0
			fake.BeforeFolder = func(_ context.Context, op, id string) error {
				if op == "lsjson" && id == "folder-1" {
					rootReads++
					if uncertain && rootReads == 3 {
						return fmt.Errorf("lost year confirmation")
					}
				}
				return nil
			}
			got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC) }}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
			if err == nil || got.Pending == 0 {
				t.Fatalf("report=%+v err=%v", got, err)
			}
			if len(fake.ChildFolders["folder-1"]["2025"]) != 0 {
				t.Fatal("second year created past confirmed or uncertain capacity")
			}
			if fake.Mkdirs > 2 {
				t.Fatalf("extra creates: %d", fake.Mkdirs)
			}
			// The retained root must remain readable at the production decoder limit.
			pin, e := (ha.Rclone{Runner: fake}).ResolveDestination(context.Background(), ha.DriveDestination{Remote: "pair-backup", Folder: "backups/pair-history"})
			if e != nil {
				t.Fatal(e)
			}
			if _, e = (ha.Rclone{Runner: fake}).ObserveFlatInventory(context.Background(), pin); e != nil {
				t.Fatalf("root poisoned: %v", e)
			}
		})
	}
}

func TestDriveMirrorMonthCapacityReservesUncertainCreate(t *testing.T) {
	cfg, _, root := driveFixture(t, 2, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	fake := monthFixtureRemote("2025")
	fake.ChildFolders["year-2025"]["occupied"] = make([]archivetest.FakeRemoteObject, 99999)
	for i := range fake.ChildFolders["year-2025"]["occupied"] {
		fake.ChildFolders["year-2025"]["occupied"][i] = archivetest.FakeRemoteObject{ID: fmt.Sprint(i), IsDir: true}
	}
	yearReads := 0
	fake.BeforeFolder = func(_ context.Context, op, id string) error {
		if op == "lsjson" && id == "year-2025" {
			yearReads++
			if yearReads == 2 {
				return fmt.Errorf("lost month confirmation")
			}
		}
		return nil
	}
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC) }}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err == nil || got.Pending != 2 || fake.Mkdirs != 1 || len(fake.ChildFolders["year-2025"]["02"]) != 0 {
		t.Fatalf("report=%+v err=%v mkdirs=%d", got, err, fake.Mkdirs)
	}
}
