package historyarchive_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

func assertDriveInventory(t *testing.T, got ha.DriveReport, phase string, files int) {
	t.Helper()
	if got.Inventory == nil || got.Inventory.Destination != "pair-backup:backups/pair-history" || got.Inventory.FolderID != "folder-1" || got.Inventory.Phase != phase || got.Inventory.Files != files {
		t.Fatalf("inventory=%+v, want destination, pinned folder, phase=%s files=%d", got.Inventory, phase, files)
	}
}

func TestDriveFileCountUsesExistingListingsAtNAnd2N(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			cfg, _, root := driveFixture(t, n)
			fake := &archivetest.FakeRclone{}
			mirror := ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}
			got, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
			if err != nil {
				t.Fatal(err)
			}
			assertDriveInventory(t, got, "after", 2*n)
			if fake.Lists != 2 || fake.CopyBatches != 1 || fake.Copies != 2*n || fake.Invocations != 4 {
				t.Fatalf("first pass lists=%d batches=%d copies=%d calls=%d", fake.Lists, fake.CopyBatches, fake.Copies, fake.Invocations)
			}
			again, err := mirror.Run(context.Background(), cfg, filepath.Join(root, "archive"))
			if err != nil || again.AlreadyRecorded != n {
				t.Fatalf("receipt pass=%+v err=%v", again, err)
			}
			assertDriveInventory(t, again, "after", 2*n)
			if fake.Lists != 4 || fake.CopyBatches != 1 || fake.Copies != 2*n || fake.Invocations != 7 {
				t.Fatalf("receipt pass lists=%d batches=%d copies=%d calls=%d", fake.Lists, fake.CopyBatches, fake.Copies, fake.Invocations)
			}
		})
	}
}

func TestDriveFileCountCountsDuplicateFilesNotDirectories(t *testing.T) {
	cfg, _, root := driveFixture(t, 1)
	fake := &archivetest.FakeRclone{Objects: map[string][]archivetest.FakeRemoteObject{
		"unrelated": {{ID: "one"}, {ID: "two"}},
		"directory": {{ID: "dir", IsDir: true}},
	}}
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil {
		t.Fatal(err)
	}
	assertDriveInventory(t, got, "after", 4)
	if fake.Lists != 2 || fake.Invocations != 4 {
		t.Fatalf("lists=%d calls=%d", fake.Lists, fake.Invocations)
	}
}

func TestDriveFileCountDirectoryStillConsumesDestinationCapacity(t *testing.T) {
	cfg, _, root := driveFixture(t, 1)
	fake := &archivetest.FakeRclone{Objects: map[string][]archivetest.FakeRemoteObject{
		"unrelated": make([]archivetest.FakeRemoteObject, 99998),
		"directory": {{ID: "dir", IsDir: true}},
	}}
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err != nil || got.Deferred != 1 || got.Verified != 0 || fake.Copies != 0 {
		t.Fatalf("capacity=%+v err=%v copies=%d", got, err, fake.Copies)
	}
	assertDriveInventory(t, got, "after", 99998)
	if fake.Lists != 2 || fake.Invocations != 3 {
		t.Fatalf("lists=%d calls=%d", fake.Lists, fake.Invocations)
	}
}

func TestDriveFileCountFinalObservationIncludesLateDuplicate(t *testing.T) {
	cfg, records, root := driveFixture(t, 1)
	fake := &archivetest.FakeRclone{}
	fake.Before = func(_ context.Context, operation, _ string) error {
		if operation == "lsjson" && fake.Lists == 1 {
			name := records[0].Label.ID + ".tar.zst.gpg"
			fake.Objects[name] = append(fake.Objects[name], archivetest.FakeRemoteObject{ID: "late", Data: fake.Objects[name][0].Data})
		}
		return nil
	}
	got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
	if err == nil || got.Verified != 0 {
		t.Fatalf("late conflict=%+v err=%v", got, err)
	}
	assertDriveInventory(t, got, "after", 3)
	if fake.Lists != 2 || fake.Invocations != 4 {
		t.Fatalf("lists=%d calls=%d", fake.Lists, fake.Invocations)
	}
}

func TestDriveFileCountFailedFinalListingRetainsBefore(t *testing.T) {
	for _, initial := range []int{0, 3} {
		t.Run(fmt.Sprint(initial), func(t *testing.T) {
			cfg, _, root := driveFixture(t, 1)
			fake := &archivetest.FakeRclone{Objects: map[string][]archivetest.FakeRemoteObject{"unrelated": make([]archivetest.FakeRemoteObject, initial)}}
			fake.Before = func(_ context.Context, operation, _ string) error {
				if operation == "lsjson" && fake.Lists == 1 {
					return errors.New("final listing unavailable")
				}
				return nil
			}
			got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(context.Background(), cfg, filepath.Join(root, "archive"))
			if err == nil {
				t.Fatal("wanted listing failure")
			}
			assertDriveInventory(t, got, "before", initial)
			if fake.Lists != 1 || fake.Invocations != 4 {
				t.Fatalf("lists=%d calls=%d", fake.Lists, fake.Invocations)
			}
		})
	}
}

func TestDriveFileCountNoSuccessfulListingIsUnknown(t *testing.T) {
	for _, mode := range []string{"resolve", "first-list", "cancel-before-list", "no-candidates"} {
		t.Run(mode, func(t *testing.T) {
			n := 1
			if mode == "no-candidates" {
				n = 0
			}
			cfg, _, root := driveFixture(t, n)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fake := &archivetest.FakeRclone{}
			calls := 0
			fake.Before = func(_ context.Context, operation, _ string) error {
				if operation == "lsjson" {
					calls++
					if mode == "resolve" || calls == 2 && mode == "first-list" {
						return errors.New("listing unavailable")
					}
					if calls == 2 && mode == "cancel-before-list" {
						cancel()
						return context.Canceled
					}
				}
				return nil
			}
			got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(ctx, cfg, filepath.Join(root, "archive"))
			if got.Inventory != nil {
				t.Fatalf("invented observation=%+v", got.Inventory)
			}
			if mode == "no-candidates" {
				if err != nil || fake.Invocations != 0 {
					t.Fatalf("empty candidates err=%v calls=%d", err, fake.Invocations)
				}
			} else if err == nil || fake.Lists != 0 || fake.Invocations != calls {
				t.Fatalf("err=%v calls=%d lists=%d hooks=%d", err, fake.Invocations, fake.Lists, calls)
			}
		})
	}
}

func TestDriveFileCountPartialCopyAndCancellation(t *testing.T) {
	for _, mode := range []string{"partial-copy", "copy-failure", "cancel-copy"} {
		t.Run(mode, func(t *testing.T) {
			cfg, _, root := driveFixture(t, 2)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fake := &archivetest.FakeRclone{}
			if mode == "partial-copy" {
				fake.AfterCopyError = errors.New("interrupted after one pair")
				fake.FailAfterObjects = 2
			} else {
				fake.Before = func(_ context.Context, operation, _ string) error {
					if operation == "copy" {
						if mode == "cancel-copy" {
							cancel()
							return context.Canceled
						}
						return errors.New("copy unavailable")
					}
					return nil
				}
			}
			got, err := (ha.DriveMirror{Rclone: ha.Rclone{Runner: fake}}).Run(ctx, cfg, filepath.Join(root, "archive"))
			if err == nil {
				t.Fatal("wanted failed/cancelled copy")
			}
			phase, files, lists, calls := "after", 0, 2, 4
			if mode == "partial-copy" {
				files = 2
			}
			if mode == "cancel-copy" {
				phase, lists, calls = "before", 1, 3
			}
			assertDriveInventory(t, got, phase, files)
			if fake.Lists != lists || fake.Invocations != calls {
				t.Fatalf("calls=%d lists=%d", fake.Invocations, fake.Lists)
			}
		})
	}
}
