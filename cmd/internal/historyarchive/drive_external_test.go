package historyarchive_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

func driveFixture(t *testing.T, count int) (ha.ContinuousConfig, []ha.CatalogRecord, string) {
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
		got, e := a.Create(context.Background(), ha.ArchiveRequest{Version: 1, Kind: "ops-documents", Source: ha.SourceIdentity{Host: "fixture", Root: src}, Files: []string{name}, Recipients: cfg.Recipients, Sink: cfg.Sink, Catalog: cfg.Catalog})
		if e != nil {
			t.Fatal(e)
		}
		records = append(records, got...)
	}
	return cfg, records, root
}

func TestDriveMirrorCountedBacklogAndRerun(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			cfg, records, root := driveFixture(t, n)
			before := treeState(t, cfg.Sink)
			fake := &archivetest.FakeRclone{}
			mirror := ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}
			got, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
			if err != nil || got.Verified != n {
				t.Fatalf("mirror = %+v, %v", got, err)
			}
			if fake.Lists != 2 || fake.Copies != 2*n {
				t.Fatalf("counts list=%d copies=%d, want2/%d", fake.Lists, fake.Copies, 2*n)
			}
			for _, r := range records {
				for _, name := range []string{r.Label.ID + ".tar.zst.gpg", r.Label.ID + ".json"} {
					if len(fake.Objects[name]) != 1 {
						t.Fatalf("missing unique flat object %s", name)
					}
				}
			}
			again, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
			if err != nil || again.AlreadyRecorded != n || fake.Copies != 2*n {
				t.Fatalf("rerun = %+v, %v copies=%d", again, err, fake.Copies)
			}
			if fake.Lists != 4 {
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
	cfg, _, root := driveFixture(t, 1)
	fake := &archivetest.FakeRclone{MissingHashes: true}
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.Verified != 1 || got.DownloadedChecksums != 2 || fake.Downloads != 2 {
		t.Fatalf("fallback = %+v %v downloads=%d", got, err, fake.Downloads)
	}
	if !strings.Contains(strings.Join(got.Notes, "\n"), "downloaded-sha256") {
		t.Fatalf("method missing: %+v", got)
	}
}

func TestDriveMirrorReleasesLocalCatalogBeforeCopy(t *testing.T) {
	cfg, _, root := driveFixture(t, 1)
	checked := false
	fake := &archivetest.FakeRclone{Before: func(ctx context.Context, operation, name string) error {
		if operation != "copyto" {
			return nil
		}
		catalog, err := ha.OpenCatalog(ctx, cfg.Catalog)
		if err != nil {
			return fmt.Errorf("local catalog held during network: %w", err)
		}
		checked = true
		return catalog.Close()
	}}
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

func TestDriveMirrorUncertainCopyAdoptsOnlyMatchingBytes(t *testing.T) {
	for _, partial := range []int{0, 3} {
		t.Run(fmt.Sprint(partial), func(t *testing.T) {
			cfg, _, root := driveFixture(t, 1)
			fake := &archivetest.FakeRclone{AfterCopyError: fmt.Errorf("lost acknowledgment"), PartialCopyBytes: partial}
			mirror := ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}
			got, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
			if err == nil || got.Verified != 0 {
				t.Fatalf("uncertain got %+v %v", got, err)
			}
			fake.AfterCopyError = nil
			fake.PartialCopyBytes = 0
			got, err = mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
			if partial == 0 {
				if err != nil || got.Verified != 1 || fake.Copies != 2 {
					t.Fatalf("matching adoption %+v %v copies=%d", got, err, fake.Copies)
				}
			} else {
				if err == nil || got.Verified != 0 || fake.Copies != 1 {
					t.Fatalf("partial conflict %+v %v copies=%d", got, err, fake.Copies)
				}
			}
		})
	}
}

func TestDriveMirrorFinalListingRejectsLateDuplicate(t *testing.T) {
	cfg, records, root := driveFixture(t, 1)
	name := records[0].Label.ID + ".tar.zst.gpg"
	fake := &archivetest.FakeRclone{}
	fake.Before = func(_ context.Context, operation, _ string) error {
		if operation == "lsjson" && fake.Lists == 1 {
			original := fake.Objects[name][0]
			fake.Objects[name] = append(fake.Objects[name], archivetest.FakeRemoteObject{ID: "duplicate", Data: original.Data})
		}
		return nil
	}
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err == nil || got.Verified != 0 || got.Pending != 1 {
		t.Fatalf("late duplicate = %+v %v", got, err)
	}
}
