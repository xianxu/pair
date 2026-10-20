package historyarchive_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

// Fake test ciphertext is a marker followed by an ordinary tar stream. This
// decoder deliberately performs no cryptography, but exercises Restorer's real
// ciphertext acquisition, hashing, pipe ownership and extraction code paths.
type driveRestoreDecoder struct{ Calls int }

func (d *driveRestoreDecoder) Decode(ctx context.Context, input io.Reader, output io.Writer) error {
	d.Calls++
	marker := make([]byte, len(archivetest.TestMarker))
	if _, err := io.ReadFull(input, marker); err != nil {
		return err
	}
	if !bytes.Equal(marker, []byte(archivetest.TestMarker)) {
		return errors.New("not fake test ciphertext")
	}
	_, err := io.Copy(output, input)
	return errors.Join(err, ctx.Err())
}

func driveRestoreFixture(t *testing.T, count int) (ha.Restorer, ha.RestoreRequest, []ha.CatalogRecord, *archivetest.FakeRclone, *driveRestoreDecoder) {
	t.Helper()
	cfg, records, fake := uploadedIndexFixture(t, count, false)
	if _, err := runIndexSync(context.Background(), cfg, fake); err != nil {
		t.Fatal(err)
	}
	// Remove original sources too: only the remote bytes and index remain.
	for _, record := range records {
		if err := os.RemoveAll(record.Manifest.Source.Root); err != nil {
			t.Fatal(err)
		}
	}
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(parent, 0700); err != nil {
		t.Fatal(err)
	}
	request := ha.RestoreRequest{To: filepath.Join(parent, "restored"), From: "drive"}
	for _, record := range records {
		request.IDs = append(request.IDs, record.Label.ID)
	}
	decoder := &driveRestoreDecoder{}
	restorer := ha.Restorer{Config: cfg, Decoder: decoder, Rclone: ha.Rclone{Runner: fake}, Now: func() time.Time { return indexSyncTime.Add(time.Hour) }}
	fake.Invocations, fake.Lists, fake.DownloadBatches, fake.Downloads, fake.HashBatches = 0, 0, 0, 0, 0
	fake.DownloadedLabelBytes, fake.DownloadedCipherBytes = 0, 0
	return restorer, request, records, fake, decoder
}

func assertDriveRestoreStageGone(t *testing.T, request ha.RestoreRequest) {
	t.Helper()
	stage := filepath.Join(filepath.Dir(request.To), ".pair-restore-staging")
	if _, err := os.Lstat(stage); !os.IsNotExist(err) {
		t.Fatalf("restore stage remains: %v", err)
	}
}

func assertDriveRestoreRefused(t *testing.T, report ha.RestoreReport, err error, request ha.RestoreRequest, decoder *driveRestoreDecoder) {
	t.Helper()
	if err == nil || decoder.Calls != 0 || report.State.Phase == ha.RestorePublished {
		t.Fatalf("unsafe acquisition reached decoder/publication: %+v calls=%d err=%v", report, decoder.Calls, err)
	}
	if _, err = os.Lstat(request.To); !os.IsNotExist(err) {
		t.Fatalf("failed acquisition published target: %v", err)
	}
	assertDriveRestoreStageGone(t, request)
}

func TestRestorerDriveRemoteOnlyCountedNAnd2N(t *testing.T) {
	for _, count := range []int{8, 16} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			restorer, request, records, fake, decoder := driveRestoreFixture(t, count)
			indexBefore, err := os.ReadFile(restorer.Config.Catalog + ".index.json")
			if err != nil {
				t.Fatal(err)
			}
			report, err := restorer.Run(context.Background(), request)
			if err != nil || report.Bundles != count || report.Files != count || report.State.Phase != ha.RestorePublished || decoder.Calls != count {
				t.Fatalf("remote restore: %+v calls=%d error=%v", report, decoder.Calls, err)
			}
			var cipherBytes int64
			for n, record := range records {
				cipherBytes += record.Label.Bytes
				path := filepath.Join(request.To, record.Label.ID, "files", record.Manifest.Files[0].Path)
				raw, err := os.ReadFile(path)
				if err != nil || string(raw) != fmt.Sprintf("private source %d", n) {
					t.Fatalf("restored %s = %q, %v", path, raw, err)
				}
				if _, err = os.Stat(filepath.Join(request.To, record.Label.ID, "MANIFEST.json")); err != nil {
					t.Fatal(err)
				}
			}
			if fake.Invocations != 3 || fake.Lists != 1 || fake.DownloadBatches != 1 || fake.Downloads != count || fake.HashBatches != 0 || fake.DownloadedLabelBytes != 0 || fake.DownloadedCipherBytes != cipherBytes || report.CipherBytes != cipherBytes {
				t.Fatalf("unbounded Drive acquisition: calls=%d lists=%d batches=%d downloads=%d hashes=%d label=%d cipher=%d want=%d", fake.Invocations, fake.Lists, fake.DownloadBatches, fake.Downloads, fake.HashBatches, fake.DownloadedLabelBytes, fake.DownloadedCipherBytes, cipherBytes)
			}
			assertDriveRestoreStageGone(t, request)
			for _, path := range []string{restorer.Config.Catalog, restorer.Config.Sink, restorer.Config.Catalog + ".uploads"} {
				if _, err = os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("restore manufactured local authority %s: %v", path, err)
				}
			}
			indexAfter, err := os.ReadFile(restorer.Config.Catalog + ".index.json")
			if err != nil || !bytes.Equal(indexBefore, indexAfter) {
				t.Fatalf("restore mutated discovery index: %v", err)
			}
		})
	}
}

func TestRestorerDriveHashesActualCiphertextBeforeDecoder(t *testing.T) {
	restorer, request, records, fake, decoder := driveRestoreFixture(t, 1)
	name := records[0].Label.ID + ".tar.zst.gpg"
	fake.Objects[name][0].Data[0] ^= 1 // Keep size and advertised SHA256 plausible.
	restorer.Rclone.Runner = indexInventoryRunner{fake: fake, alter: func(entries []ha.RemoteEntry) []ha.RemoteEntry {
		for n := range entries {
			if entries[n].Name == name {
				entries[n].Hashes = map[string]string{"SHA-256": records[0].Label.SHA256}
			}
		}
		return entries
	}}
	report, err := restorer.Run(context.Background(), request)
	assertDriveRestoreRefused(t, report, err, request, decoder)
	if fake.DownloadBatches != 1 || fake.DownloadedCipherBytes != records[0].Label.Bytes {
		t.Fatal("corruption test did not acquire actual ciphertext")
	}
}

func TestRestorerDrivePartialAndCanceledBatchNeverPublishes(t *testing.T) {
	for _, phase := range []string{"partial", "cancel"} {
		t.Run(phase, func(t *testing.T) {
			restorer, request, _, fake, decoder := driveRestoreFixture(t, 2)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if phase == "partial" {
				fake.FailAfterObjects = 1
			} else {
				fake.Before = func(_ context.Context, operation, _ string) error {
					if operation == "copy" {
						cancel()
						return ctx.Err()
					}
					return nil
				}
			}
			report, err := restorer.Run(ctx, request)
			assertDriveRestoreRefused(t, report, err, request, decoder)
			if phase == "partial" && (fake.DownloadBatches != 1 || fake.DownloadedCipherBytes == 0) {
				t.Fatal("partial transfer left no exercised partial bytes")
			}
			if phase == "cancel" && (!errors.Is(err, context.Canceled) || report.State.Phase != ha.RestoreCancelled) {
				t.Fatalf("cancellation state: %+v %v", report, err)
			}
		})
	}
}

func TestRestorerDriveDuplicateNamesRefusedBeforeDownload(t *testing.T) {
	restorer, request, records, fake, decoder := driveRestoreFixture(t, 1)
	name := records[0].Label.ID + ".tar.zst.gpg"
	duplicate := fake.Objects[name][0]
	duplicate.ID = "duplicate-object"
	fake.Objects[name] = append(fake.Objects[name], duplicate)
	report, err := restorer.Run(context.Background(), request)
	assertDriveRestoreRefused(t, report, err, request, decoder)
	if fake.Invocations != 2 || fake.Lists != 1 || fake.DownloadBatches != 0 || fake.DownloadedCipherBytes != 0 {
		t.Fatalf("duplicate reached download: %+v", fake)
	}
	if len(fake.Objects[name]) != 2 {
		t.Fatal("duplicate repaired destructively")
	}
}

func TestRestorerDriveBoundsPostListingGrowthBeforeTransfer(t *testing.T) {
	restorer, request, records, fake, decoder := driveRestoreFixture(t, 1)
	name := records[0].Label.ID + ".tar.zst.gpg"
	fake.Before = func(_ context.Context, operation, _ string) error {
		if operation == "copy" {
			fake.Objects[name][0].Data = append(fake.Objects[name][0].Data, 0)
		}
		return nil
	}
	report, err := restorer.Run(context.Background(), request)
	assertDriveRestoreRefused(t, report, err, request, decoder)
	if fake.DownloadBatches != 1 || fake.DownloadedCipherBytes != 0 {
		t.Fatalf("post-listing growth exceeded admitted transfer envelope: batches=%d bytes=%d", fake.DownloadBatches, fake.DownloadedCipherBytes)
	}
}
