package historyarchive_test

import (
	"context"
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
