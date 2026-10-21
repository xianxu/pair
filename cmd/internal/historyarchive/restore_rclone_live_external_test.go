package historyarchive_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
)

func TestRestoreRcloneRecordedMonthAndMixed(t *testing.T) {
	if os.Getenv("PAIR_RCLONE_LOCAL_CONFORMANCE") != "1" {
		t.Skip("installed rclone private-backend qualification")
	}
	binary, err := exec.LookPath("rclone")
	if err != nil {
		t.Fatal(err)
	}
	cfg, records, root := driveFixture(t, 2, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	cfg.DriveRemote, cfg.DriveFolder = "fixture", "backups/pair-history"
	remoteRoot := filepath.Join(root, "remote")
	folder := filepath.Join(remoteRoot, "backups", "pair-history")
	if err = os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "rclone.conf")
	if err = os.WriteFile(config, []byte("[fixture]\ntype = alias\nremote = "+remoteRoot+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	trace := &streamingDownloadRunner{runner: ha.ExecRcloneRunner{Binary: binary, Env: []string{"HOME=" + root, "RCLONE_CONFIG=" + config, "TMPDIR=" + root, "PATH=" + os.Getenv("PATH")}}}
	remote := ha.Rclone{Runner: trace, LocalConformance: true}
	for name, key := range map[string]string{records[0].Label.ID + ".tar.zst.gpg": records[0].BundleKey, records[0].Label.ID + ".json": records[0].LabelKey} {
		raw, e := os.ReadFile(filepath.Join(cfg.Sink, key))
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(folder, name), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	ctx := context.Background()
	report, err := (ha.DriveMirror{Rclone: remote}).Run(ctx, cfg, filepath.Join(root, "archive"))
	if err != nil || report.Verified != 2 {
		t.Fatalf("upload=%+v err=%v", report, err)
	}
	if _, err = ha.SyncLocalLabelIndex(ctx, cfg, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	scratch := filepath.Join(root, "scratch")
	if err = os.Mkdir(scratch, 0700); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		ids  []string
	}{{"root", []string{records[0].Label.ID}}, {"month", []string{records[1].Label.ID}}, {"mixed", []string{records[0].Label.ID, records[1].Label.ID}}} {
		t.Run(tc.name, func(t *testing.T) {
			decoder := &driveRestoreDecoder{}
			target := filepath.Join(scratch, tc.name)
			got, e := (ha.Restorer{Config: cfg, Rclone: remote, Decoder: decoder}).Run(ctx, ha.RestoreRequest{IDs: tc.ids, From: "drive", To: target})
			if e != nil || got.Bundles != len(tc.ids) || decoder.Calls != len(tc.ids) {
				t.Fatalf("report=%+v decodes=%d err=%v", got, decoder.Calls, e)
			}
			for _, id := range tc.ids {
				for _, record := range records {
					if record.Label.ID != id {
						continue
					}
					for _, member := range record.Manifest.Files {
						data, readErr := os.ReadFile(filepath.Join(target, id, "files", member.Path))
						if readErr != nil || int64(len(data)) != member.Size || fmt.Sprintf("%x", sha256.Sum256(data)) != member.SHA256 {
							t.Fatalf("restored member %s: %v", member.Path, readErr)
						}
					}
				}
			}
			assertDriveRestoreStageGone(t, ha.RestoreRequest{To: target})
		})
	}
	t.Run("listed size mismatch before copy", func(t *testing.T) {
		path := filepath.Join(folder, "2025", "02", records[1].Label.ID+".tar.zst.gpg")
		data, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(path, append(data, 'x'), 0600); e != nil {
			t.Fatal(e)
		}
		trace.calls = nil
		decoder := &driveRestoreDecoder{}
		req := ha.RestoreRequest{IDs: []string{records[1].Label.ID}, From: "drive", To: filepath.Join(scratch, "refused")}
		report, e := (ha.Restorer{Config: cfg, Rclone: remote, Decoder: decoder}).Run(ctx, req)
		assertDriveRestoreRefused(t, report, e, req, decoder)
		for _, call := range trace.calls {
			if call[0] == "copy" {
				t.Fatalf("size mismatch started copy: %v", call)
			}
		}
	})
}
