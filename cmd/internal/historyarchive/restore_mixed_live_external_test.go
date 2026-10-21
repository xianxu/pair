package historyarchive_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/archivetestenv"
	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

func TestRestoreLiveMixedLayoutsAfterMetadataLoss(t *testing.T) {
	if os.Getenv("PAIR_ARCHIVE_LIVE") != "1" {
		t.Skip("opt-in isolated throwaway GPG/zstd mixed-layout conformance")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	key, err := archivetestenv.NewGPGHome(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if e := key.Close(); e != nil {
			t.Errorf("throwaway key cleanup: %v", e)
		}
	})
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "originals")
	if err = os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := ha.ContinuousConfig{Version: ha.Version, Recipients: []string{key.Fingerprint}, Sink: filepath.Join(root, "sink"), Catalog: filepath.Join(root, "metadata", "catalog.jsonl")}
	fixture := map[string][]byte{"legacy.txt": []byte("legacy recovery\n"), "month-雪.bin": {0, 255, 1, 2, 0}}
	var records []ha.CatalogRecord
	var cipherBytes, rawBytes int64
	names := []string{"legacy.txt", "month-雪.bin"}
	for i, name := range names {
		if err = os.WriteFile(filepath.Join(source, name), fixture[name], 0600); err != nil {
			t.Fatal(err)
		}
		stamp := time.Date(2025, time.Month(i+1), 1, 0, 0, 0, 0, time.UTC)
		if err = os.Chtimes(filepath.Join(source, name), stamp, stamp); err != nil {
			t.Fatal(err)
		}
		made, e := (ha.Archiver{Codec: ha.CommandCodec{Home: key.Home}, Sink: ha.LocalSink{Root: cfg.Sink}}).Create(ctx, ha.ArchiveRequest{Version: ha.Version, Kind: "ops-documents", Source: ha.SourceIdentity{Host: "mixed-recovery-fixture", Root: source}, Files: []string{name}, Recipients: cfg.Recipients, Sink: cfg.Sink, Catalog: cfg.Catalog})
		if e != nil || len(made) != 1 {
			t.Fatalf("create %s count=%d err=%v", name, len(made), e)
		}
		records = append(records, made[0])
		cipherBytes += made[0].Label.Bytes
		rawBytes += made[0].Label.RawBytes
	}
	fake := legacyIndexDriveFake(t, cfg, records[:1])
	fake.ChildFolders = map[string]map[string][]archivetest.FakeRemoteObject{}
	remote := ha.Rclone{Runner: fake}
	store := filepath.Join(root, "archive-store")
	mirror := ha.DriveMirror{Rclone: remote}
	uploaded, err := mirror.Run(ctx, cfg, store)
	if err != nil || uploaded.Verified != 2 || fake.CopyBatches != 1 {
		t.Fatalf("upload=%+v err=%v batches=%d", uploaded, err, fake.CopyBatches)
	}
	again, err := mirror.Run(ctx, cfg, store)
	if err != nil || again.AlreadyRecorded != 2 || fake.CopyBatches != 1 {
		t.Fatalf("retry=%+v err=%v batches=%d", again, err, fake.CopyBatches)
	}
	remoteState := func() []byte {
		raw, e := json.Marshal([]any{fake.Objects, fake.ChildFolders, fake.FolderObjects})
		if e != nil {
			t.Fatal(e)
		}
		return raw
	}
	before := remoteState()
	for i, record := range records {
		raw, e := os.ReadFile(filepath.Join(cfg.Sink, record.BundleKey))
		if e != nil {
			t.Fatal(e)
		}
		name := names[i]
		assertLiveTarMatchesOriginals(t, ctx, key.Home, raw, record.Label.ID, map[string][]byte{name: fixture[name]}, map[string][32]byte{name: sha256.Sum256(fixture[name])})
	}
	// Only fixture-owned roots disappear; no local archive authority survives.
	for _, owned := range []string{cfg.Sink, filepath.Dir(cfg.Catalog), store} {
		if err = os.RemoveAll(owned); err != nil {
			t.Fatal(err)
		}
	}
	fake.DownloadedLabelBytes, fake.DownloadedCipherBytes = 0, 0
	at := time.Now().UTC()
	sync, err := (ha.DriveIndexSync{Rclone: remote, Now: func() time.Time { return at }}).Run(ctx, cfg)
	if err != nil || sync.LabelCount != 2 || fake.DownloadedCipherBytes != 0 || fake.DownloadedLabelBytes == 0 {
		t.Fatalf("sync=%+v err=%v cipher=%d labels=%d", sync, err, fake.DownloadedCipherBytes, fake.DownloadedLabelBytes)
	}
	view, err := ha.LoadLabelView(ctx, cfg, at.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, record := range records {
		ids = append(ids, record.Label.ID)
		locations := view.Records[record.Label.ID].Locations
		if len(locations) != 1 {
			t.Fatalf("unexpected location set: %+v", locations)
		}
		for _, location := range locations {
			if location.Backend != "drive" {
				t.Fatal("invented local authority")
			}
		}
	}
	scratch := filepath.Join(root, "scratch")
	if err = os.Mkdir(scratch, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(scratch, "restored")
	decoder := &liveCountingDecoder{decoder: ha.CommandDecoder{Home: key.Home}}
	report, err := (ha.Restorer{Config: cfg, Decoder: decoder, Rclone: remote, Now: func() time.Time { return at.Add(2 * time.Second) }}).Run(ctx, ha.RestoreRequest{IDs: ids, To: target, From: "drive"})
	if err != nil || report.State.Phase != ha.RestorePublished || report.Bundles != 2 || report.Files != 2 || report.CipherBytes != cipherBytes || report.RawBytes != rawBytes || decoder.calls != 2 {
		t.Fatalf("restore=%+v err=%v decodes=%d", report, err, decoder.calls)
	}
	children, err := os.ReadDir(target)
	if err != nil || len(children) != 2 {
		t.Fatalf("publication count=%d err=%v", len(children), err)
	}
	for i, record := range records {
		name := names[i]
		raw, e := os.ReadFile(filepath.Join(target, record.Label.ID, "files", name))
		if e != nil || !bytes.Equal(raw, fixture[name]) || sha256.Sum256(raw) != sha256.Sum256(fixture[name]) {
			t.Fatalf("restored %s err=%v", name, e)
		}
		manifest, e := os.ReadFile(filepath.Join(target, record.Label.ID, "MANIFEST.json"))
		if e != nil {
			t.Fatal(e)
		}
		assertLiveManifest(t, manifest, record.Label.ID, map[string][]byte{name: fixture[name]}, map[string][32]byte{name: sha256.Sum256(fixture[name])})
		original, e := os.ReadFile(filepath.Join(source, name))
		if e != nil || !bytes.Equal(original, fixture[name]) {
			t.Fatalf("original changed: %s %v", name, e)
		}
	}
	assertLiveNoLocalAuthority(t, cfg, scratch)
	if !bytes.Equal(before, remoteState()) || fake.DownloadedCipherBytes != cipherBytes {
		t.Fatalf("remote changed or unexpected transfer: %d want=%d", fake.DownloadedCipherBytes, cipherBytes)
	}
	t.Logf("mixed legacy/month real-key recovery: 2 bundles, %d ciphertext bytes, %d raw bytes; original hashes agree, stage absent, no retry copy", cipherBytes, rawBytes)
}
