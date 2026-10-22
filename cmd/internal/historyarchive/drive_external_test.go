package historyarchive_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

func driveFixture(t *testing.T, count int, mtimes ...time.Time) (ha.ContinuousConfig, []ha.CatalogRecord, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "source")
	if err = os.Mkdir(src, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := ha.ContinuousConfig{Version: 1, Recipients: []string{strings.Repeat("A", 40)}, Sink: filepath.Join(root, "sink"), Catalog: filepath.Join(root, "archive", "catalog.jsonl")}
	var records []ha.CatalogRecord
	a := ha.Archiver{Codec: &archivetest.TestCodec{KnownRecipients: map[string]bool{strings.Repeat("A", 40): true}}, Sink: ha.LocalSink{Root: cfg.Sink}}
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("record-%03d.md", i)
		if err = os.WriteFile(filepath.Join(src, name), []byte(fmt.Sprintf("private source %d", i)), 0600); err != nil {
			t.Fatal(err)
		}
		if i < len(mtimes) {
			if err = os.Chtimes(filepath.Join(src, name), mtimes[i], mtimes[i]); err != nil {
				t.Fatal(err)
			}
		}
		got, e := a.Create(context.Background(), ha.ArchiveRequest{Version: 1, Kind: "ops-documents", Source: ha.SourceIdentity{Host: "fixture", Root: src}, Files: []string{name}, Recipients: cfg.Recipients, Sink: cfg.Sink, Catalog: cfg.Catalog})
		if e != nil {
			t.Fatal(e)
		}
		records = append(records, got...)
	}
	return cfg, records, root
}

// monthDriveFake pre-populates the canonical folder of a fixture's pairs. Tests
// can then isolate pair reconciliation from the separate folder-creation flow.
func monthDriveFake(t *testing.T, records []ha.CatalogRecord) *archivetest.FakeRclone {
	t.Helper()
	if len(records) == 0 {
		t.Fatal("month fixture needs a record")
	}
	stamp := records[0].Label.From.UTC()
	for _, record := range records {
		if record.Label.From.UTC().Format("2006/01") != stamp.Format("2006/01") {
			t.Fatal("month fixture spans months")
		}
	}
	return &archivetest.FakeRclone{
		FolderObjects: map[string]map[string][]archivetest.FakeRemoteObject{"month": {}},
		ChildFolders: map[string]map[string][]archivetest.FakeRemoteObject{
			"folder-1": {stamp.Format("2006"): {{ID: "year", IsDir: true}}},
			"year":     {stamp.Format("01"): {{ID: "month", IsDir: true}}},
		},
	}
}

func TestDriveMirrorCountedBacklogAndRerun(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			cfg, records, root := driveFixture(t, n)
			before := treeState(t, cfg.Sink)
			fake := monthDriveFake(t, records)
			mirror := ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}
			got, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
			if err != nil || got.Verified != n {
				t.Fatalf("mirror = %+v, %v", got, err)
			}
			if fake.Lists != 4 || fake.Copies != 2*n || fake.CopyBatches != 1 || fake.Invocations != 6 {
				t.Fatalf("counts list=%d copies=%d, want4/%d", fake.Lists, fake.Copies, 2*n)
			}
			for _, r := range records {
				for _, name := range []string{r.Label.ID + ".tar.zst.gpg", r.Label.ID + ".json"} {
					if len(fake.FolderObjects["month"][name]) != 1 {
						t.Fatalf("missing unique month object %s", name)
					}
				}
			}
			again, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
			if err != nil || again.AlreadyRecorded != n || fake.Copies != 2*n {
				t.Fatalf("rerun = %+v, %v copies=%d", again, err, fake.Copies)
			}
			if fake.Lists != 7 || fake.CopyBatches != 1 || fake.Invocations != 10 {
				t.Fatalf("rerun content listings=%d", fake.Lists)
			}
			if !sameKeys(before, treeState(t, cfg.Sink)) {
				t.Fatal("local copies changed")
			}
		})
	}
}

func TestDriveMirrorDuplicatesRefuseWithoutRepair(t *testing.T) {
	cfg, records, root := driveFixture(t, 1)
	name := records[0].Label.ID + ".tar.zst.gpg"
	fake := &archivetest.FakeRclone{Objects: map[string][]archivetest.FakeRemoteObject{name: {{ID: "one", Data: []byte("wrong")}, {ID: "two", Data: []byte("wrong")}}}}
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err == nil || got.Verified != 0 || fake.Copies != 0 {
		t.Fatalf("duplicate = %+v %v copies=%d", got, err, fake.Copies)
	}
	if len(fake.Objects[name]) != 2 {
		t.Fatal("duplicate repaired destructively")
	}
}

func TestDriveMirrorDownloadFallbackIsReported(t *testing.T) {
	cfg, records, root := driveFixture(t, 1)
	fake := monthDriveFake(t, records)
	fake.MissingHashes = true
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.Verified != 1 || got.DownloadedChecksums != 2 || fake.Downloads != 2 || fake.HashBatches != 1 || fake.Invocations != 7 {
		t.Fatalf("fallback = %+v %v downloads=%d", got, err, fake.Downloads)
	}
	if !strings.Contains(strings.Join(got.Notes, "\n"), "downloaded-sha256") {
		t.Fatalf("method missing: %+v", got)
	}
}

func TestDriveMirrorReleasesLocalCatalogBeforeCopy(t *testing.T) {
	cfg, records, root := driveFixture(t, 1)
	checked := false
	fake := monthDriveFake(t, records)
	fake.Before = func(ctx context.Context, operation, name string) error {
		if operation != "copy" {
			return nil
		}
		catalog, err := ha.OpenCatalog(ctx, cfg.Catalog)
		if err != nil {
			return fmt.Errorf("local catalog held during network: %w", err)
		}
		checked = true
		return catalog.Close()
	}
	_, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || !checked {
		t.Fatalf("catalog check=%v err=%v", checked, err)
	}
}

func TestContinuousDriveUnavailablePreservesLocalMarker(t *testing.T) {
	cfg, _, root := driveFixture(t, 1)
	home := filepath.Join(root, "home")
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal(err)
	}
	fake := &archivetest.FakeRclone{MissingRemote: true}
	mirror := &ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}
	store := filepath.Join(root, "archive")
	pass := ha.ContinuousPass{Config: cfg, Store: store, SnapshotRoot: filepath.Join(root, "snapshots"), Home: home, Host: "fixture", Env: map[string]string{}, Archiver: ha.Archiver{Codec: &archivetest.TestCodec{KnownRecipients: map[string]bool{strings.Repeat("A", 40): true}}, Sink: ha.LocalSink{Root: cfg.Sink}}, Drive: mirror}
	before := treeState(t, cfg.Sink)
	report, err := pass.Run(context.Background())
	if err != nil {
		t.Fatalf("remote failure became local failure: %v", err)
	}
	if report.Drive == nil || report.Drive.Pending != 1 || len(report.Drive.Notes) == 0 {
		t.Fatalf("unavailable not visible: %+v", report)
	}
	if _, err := os.Stat(filepath.Join(store, ha.LastRunName)); err != nil {
		t.Fatalf("local marker missing: %v", err)
	}
	if !sameKeys(before, treeState(t, cfg.Sink)) {
		t.Fatal("local encrypted copies changed")
	}
}

func TestDriveMirrorPartialBatchRecordsOnlyVerifiedPairs(t *testing.T) {
	cfg, records, root := driveFixture(t, 2)
	fake := monthDriveFake(t, records)
	fake.AfterCopyError, fake.FailAfterObjects = fmt.Errorf("batch interrupted"), 2
	mirror := ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}
	got, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err == nil || got.Verified != 1 || got.Pending != 1 || fake.CopyBatches != 1 {
		t.Fatalf("partial batch %+v %v batches=%d", got, err, fake.CopyBatches)
	}
	assertDriveStageEmpty(t, root)
	fake.AfterCopyError = nil
	fake.FailAfterObjects = 0
	got, err = mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.Verified != 1 || got.AlreadyRecorded != 1 || fake.CopyBatches != 2 {
		t.Fatalf("batch retry %+v %v batches=%d", got, err, fake.CopyBatches)
	}
	assertDriveStageEmpty(t, root)
}

func assertDriveStageEmpty(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "archive", ".drive-staging"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("stage not cleaned: %v %v", entries, err)
	}
}

func TestDriveMirrorBatchBudgetDefersCompletePairs(t *testing.T) {
	// Fractional mtimes change both tar PAX and label lengths: the pairs need
	// not have equal sizes even when their source text has the same length.
	cfg, records, root := driveFixture(t, 2,
		time.Unix(1700000000, 100000000).UTC(),
		time.Unix(1700000000, 123456789).UTC())
	pairBytes := make([]int64, len(records))
	var budget, total int64
	for i, record := range records {
		label, err := os.Stat(filepath.Join(cfg.Sink, record.LabelKey))
		if err != nil {
			t.Fatal(err)
		}
		pairBytes[i] = record.Label.Bytes + label.Size()
		budget = max(budget, pairBytes[i])
		total += pairBytes[i]
	}
	if pairBytes[0] == pairBytes[1] || budget >= total {
		t.Fatalf("want unequal pairs that fit separately but not together: sizes=%v budget=%d", pairBytes, budget)
	}
	fake := monthDriveFake(t, records)
	mirror := ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}, BatchByteLimit: budget}
	got, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.Verified != 1 || got.Pending != 1 || got.Deferred != 1 || fake.Copies != 2 {
		t.Fatalf("budget %+v %v copies=%d", got, err, fake.Copies)
	}
	assertDriveStageEmpty(t, root)
	got, err = mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.Verified != 1 || got.Pending != 0 || got.Deferred != 0 || got.AlreadyRecorded != 1 || fake.Copies != 4 || fake.CopyBatches != 2 {
		t.Fatalf("deferred retry %+v %v copies=%d batches=%d", got, err, fake.Copies, fake.CopyBatches)
	}
	assertDriveStageEmpty(t, root)
}

func TestDriveMirrorDestinationCapacityDefersWithoutPoisoningInventory(t *testing.T) {
	cfg, records, root := driveFixture(t, 2)
	// Duplicate, unrelated names consume real destination entries too.
	fake := monthDriveFake(t, records)
	fake.FolderObjects["month"]["unrelated"] = make([]archivetest.FakeRemoteObject, 99998)
	mirror := ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}
	got, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.Verified != 1 || got.Deferred != 1 || got.Pending != 1 || fake.Copies != 2 {
		t.Fatalf("near-full destination = %+v, %v, copies=%d", got, err, fake.Copies)
	}
	if !strings.Contains(strings.Join(got.Notes, "\n"), "destination entry limit") {
		t.Fatalf("capacity deferral not visible: %+v", got)
	}
	// The pass must not have made the reader's own 100000-entry bound unusable.
	got, err = mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.AlreadyRecorded != 1 || got.Deferred != 1 || got.Pending != 1 || fake.Copies != 2 {
		t.Fatalf("full destination retry = %+v, %v, copies=%d", got, err, fake.Copies)
	}
	// An operator-created vacancy admits the deferred complete pair next pass.
	fake.FolderObjects["month"]["unrelated"] = fake.FolderObjects["month"]["unrelated"][:99996]
	got, err = mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.AlreadyRecorded != 1 || got.Verified != 1 || got.Pending != 0 || fake.Copies != 4 {
		t.Fatalf("capacity recovery = %+v, %v, copies=%d", got, err, fake.Copies)
	}
	assertDriveStageEmpty(t, root)
}

func TestDriveMirrorDestinationCapacityChargesOnlyMissingMembers(t *testing.T) {
	for _, existingMembers := range []int{1, 2} {
		t.Run(fmt.Sprint(existingMembers), func(t *testing.T) {
			cfg, records, root := driveFixture(t, 2)
			fake := monthDriveFake(t, records)
			fake.FolderObjects["month"]["unrelated"] = make([]archivetest.FakeRemoteObject, 99998)
			// The first pair cannot fit; it must not block the second pair.
			record := records[1]
			keys := []string{record.BundleKey, record.LabelKey}
			names := []string{record.Label.ID + ".tar.zst.gpg", record.Label.ID + ".json"}
			for i := 0; i < existingMembers; i++ {
				data, err := os.ReadFile(filepath.Join(cfg.Sink, keys[i]))
				if err != nil {
					t.Fatal(err)
				}
				fake.FolderObjects["month"][names[i]] = []archivetest.FakeRemoteObject{{ID: fmt.Sprintf("retained-%d", i), Data: data}}
			}
			mirror := ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}
			got, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
			if err != nil || got.Verified != 1 || got.Pending != 1 || got.Deferred != 1 || fake.Copies != 2-existingMembers {
				t.Fatalf("existing=%d reconciliation = %+v, %v, copies=%d", existingMembers, got, err, fake.Copies)
			}
			if fake.CopyBatches != 2-existingMembers {
				t.Fatalf("existing=%d batches=%d", existingMembers, fake.CopyBatches)
			}
			assertDriveStageEmpty(t, root)
		})
	}
}

func TestDriveMirrorFinalListingRejectsLateDuplicate(t *testing.T) {
	cfg, records, root := driveFixture(t, 1)
	name := records[0].Label.ID + ".tar.zst.gpg"
	fake := monthDriveFake(t, records)
	fake.BeforeFolder = func(_ context.Context, operation, folderID string) error {
		if operation == "lsjson" && folderID == "month" && fake.FolderLists[folderID] == 1 {
			original := fake.FolderObjects["month"][name][0]
			fake.FolderObjects["month"][name] = append(fake.FolderObjects["month"][name], archivetest.FakeRemoteObject{ID: "duplicate", Data: original.Data})
		}
		return nil
	}
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err == nil || got.Verified != 0 || got.Pending != 1 {
		t.Fatalf("late duplicate = %+v %v", got, err)
	}
}

func TestDriveMirrorBatchStageCleanedOnFailureAndCancel(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			cfg, records, root := driveFixture(t, 1)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fake := monthDriveFake(t, records)
			fake.Before = func(_ context.Context, operation, _ string) error {
				if operation != "copy" {
					return nil
				}
				if cancelled {
					cancel()
					return context.Canceled
				}
				return fmt.Errorf("batch failed")
			}
			got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(ctx, cfg, filepath.Join(root, "archive"))
			if err == nil || got.Verified != 0 {
				t.Fatalf("failure result %+v %v", got, err)
			}
			assertDriveStageEmpty(t, root)
		})
	}
}

func TestContinuousSweepsInterruptedDriveStageEvenWhenLocalBackupFails(t *testing.T) {
	cfg, records, root := driveFixture(t, 1)
	store := filepath.Join(root, "archive")
	objects := filepath.Join(store, ".drive-staging", "objects")
	if err := os.MkdirAll(objects, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(cfg.Sink, records[0].BundleKey), filepath.Join(objects, records[0].Label.ID+".tar.zst.gpg")); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".codex", "state_5.sqlite"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	fake := &archivetest.FakeRclone{}
	pass := ha.ContinuousPass{Config: cfg, Store: store, SnapshotRoot: filepath.Join(root, "snapshots"), Home: home, Host: "fixture", Env: map[string]string{}, Snapshotter: failingSnapshotter{}, Archiver: ha.Archiver{Codec: &archivetest.TestCodec{KnownRecipients: map[string]bool{strings.Repeat("A", 40): true}}, Sink: ha.LocalSink{Root: cfg.Sink}}, Drive: &ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}}
	if _, err := pass.Run(context.Background()); err == nil {
		t.Fatal("expected local snapshot failure")
	}
	assertDriveStageEmpty(t, root)
	if fake.Invocations != 0 {
		t.Fatal("local failure started remote work")
	}
}
