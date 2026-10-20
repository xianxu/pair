package historyarchive_test

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/archivetestenv"
	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

func TestRestoreLiveRemoteOnlyRoundTrip(t *testing.T) {
	if os.Getenv("PAIR_ARCHIVE_LIVE") != "1" {
		t.Skip("set PAIR_ARCHIVE_LIVE=1 for isolated real GPG/zstd restore conformance")
	}
	for _, name := range []string{"gpg", "gpgconf", "zstd"} {
		if _, err := exec.LookPath(name); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	key, err := archivetestenv.NewGPGHome(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := key.Close(); err != nil {
			t.Errorf("disposable key cleanup: %v", err)
		}
	})
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "originals")
	fixtures := map[string][]byte{"empty-file": {}, "unicode/雪.txt": []byte("hello 雪\n"), "nested/binary[*]?.bin": {0, 255, 1, 2, 0, 128}}
	hashes := map[string][32]byte{}
	var names []string
	var rawBytes int64
	for name, data := range fixtures {
		path := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		hashes[name] = sha256.Sum256(data)
		rawBytes += int64(len(data))
		names = append(names, name)
	}
	sort.Strings(names)
	cfg := ha.ContinuousConfig{Version: ha.Version, Recipients: []string{key.Fingerprint}, Sink: filepath.Join(root, "sink"), Catalog: filepath.Join(root, "catalog", "history.jsonl")}
	request := ha.ArchiveRequest{Version: ha.Version, Kind: "ops-documents", Source: ha.SourceIdentity{Host: "isolated-restore-fixture", Root: source}, Files: names, Recipients: cfg.Recipients, Sink: cfg.Sink, Catalog: cfg.Catalog}
	records, err := (ha.Archiver{Codec: ha.CommandCodec{Home: key.Home}, Sink: ha.LocalSink{Root: cfg.Sink}}).Create(ctx, request)
	if err != nil || len(records) != 1 {
		t.Fatalf("create: bundles=%d err=%v", len(records), err)
	}
	id := records[0].Label.ID
	cipherBytes := records[0].Label.Bytes
	fake := &archivetest.FakeRclone{}
	remote := ha.Rclone{Runner: fake}
	store := filepath.Join(root, "archive-store")
	upload, err := (ha.DriveMirror{Rclone: remote}).Run(ctx, cfg, store)
	if err != nil || upload.Verified != 1 {
		t.Fatalf("upload: %+v %v", upload, err)
	}
	remoteBefore, err := json.Marshal(fake.Objects)
	if err != nil {
		t.Fatal(err)
	}
	remoteCipher := fake.Objects[id+".tar.zst.gpg"]
	if len(remoteCipher) != 1 {
		t.Fatal("fake remote lacks exact ciphertext")
	}
	assertLiveTarMatchesOriginals(t, ctx, key.Home, remoteCipher[0].Data, id, fixtures, hashes)
	// Lose every local archive authority/copy, including any index and receipt.
	// These three roots were exclusively created by this fixture.
	for _, owned := range []string{cfg.Sink, filepath.Dir(cfg.Catalog), store} {
		if err := os.RemoveAll(owned); err != nil {
			t.Fatal(err)
		}
	}
	fake.DownloadedLabelBytes = 0
	fake.DownloadedCipherBytes = 0
	at := time.Now().UTC()
	synced, err := (ha.DriveIndexSync{Rclone: remote, Now: func() time.Time { return at }}).Run(ctx, cfg)
	if err != nil || synced.LabelCount != 1 {
		t.Fatalf("Drive-only sync: %+v %v", synced, err)
	}
	if fake.DownloadedCipherBytes != 0 || fake.DownloadedLabelBytes == 0 {
		t.Fatalf("sync downloaded ciphertext or no labels: cipher=%d label=%d", fake.DownloadedCipherBytes, fake.DownloadedLabelBytes)
	}
	view, err := ha.LoadLabelView(ctx, cfg, at.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	found, err := ha.SelectLabels(view, ha.FindQuery{Kind: request.Kind})
	if err != nil || len(found) != 1 || found[0].Label.ID != id {
		t.Fatalf("find rebuilt label: count=%d err=%v", len(found), err)
	}
	for _, loc := range found[0].Locations {
		if loc.Backend != "drive" {
			t.Fatal("Drive reconstruction invented local evidence")
		}
	}
	scratch := filepath.Join(root, "scratch")
	if err := os.Mkdir(scratch, 0700); err != nil {
		t.Fatal(err)
	}
	decoder := &liveCountingDecoder{decoder: ha.CommandDecoder{Home: key.Home}}
	restorer := ha.Restorer{Config: cfg, Decoder: decoder, Rclone: remote, Now: func() time.Time { return at.Add(2 * time.Second) }}
	for _, run := range []struct {
		name  string
		paths []string
		files int
	}{{"all", nil, len(fixtures)}, {"filtered", []string{"nested/binary[*]?.bin"}, 1}} {
		t.Run(run.name, func(t *testing.T) {
			target := filepath.Join(scratch, run.name)
			report, err := restorer.Run(ctx, ha.RestoreRequest{IDs: []string{id}, To: target, Paths: run.paths})
			if err != nil || report.State.Phase != ha.RestorePublished || report.Bundles != 1 || report.Files != run.files || report.Target != target || report.CipherBytes != cipherBytes || report.RawBytes != rawBytes {
				t.Fatalf("restore: %+v %v", report, err)
			}
			assertLivePublishedTree(t, target, id, fixtures, hashes, run.paths)
			for name, want := range fixtures {
				got, err := os.ReadFile(filepath.Join(source, name))
				if err != nil || !bytes.Equal(got, want) || sha256.Sum256(got) != hashes[name] {
					t.Fatalf("source changed: %q err=%v", name, err)
				}
			}
			assertLiveNoLocalAuthority(t, cfg, scratch)
		})
	}
	if decoder.calls != 2 {
		t.Fatalf("production decoder calls: got=%d want=2", decoder.calls)
	}
	remoteAfter, err := json.Marshal(fake.Objects)
	if err != nil || !bytes.Equal(remoteBefore, remoteAfter) {
		t.Fatal("restore changed fake remote objects")
	}
	if fake.DownloadedCipherBytes != 2*cipherBytes {
		t.Fatalf("unexpected restore transfer bytes: got=%d want=%d", fake.DownloadedCipherBytes, 2*cipherBytes)
	}
	t.Logf("remote-only real-key restore: bundles=1 files=%d ciphertext=%d; complete and literal-filtered outputs independently match originals; source and remote bytes unchanged", len(fixtures), cipherBytes)
}

// Counts executions while retaining the actual GPG/zstd decoder and its lifecycle.
type liveCountingDecoder struct {
	decoder ha.CommandDecoder
	calls   int
}

func (d *liveCountingDecoder) Decode(ctx context.Context, in io.Reader, out io.Writer) error {
	d.calls++
	return d.decoder.Decode(ctx, in, out)
}

func assertLiveManifest(t *testing.T, raw []byte, id string, fixtures map[string][]byte, hashes map[string][32]byte) {
	t.Helper()
	var manifest ha.Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	gotID, err := manifest.ID()
	if err != nil || gotID != id || len(manifest.Files) != len(fixtures) {
		t.Fatalf("manifest identity/file set: %v", err)
	}
	seen := map[string]bool{}
	for _, entry := range manifest.Files {
		data, ok := fixtures[entry.Path]
		hash := hashes[entry.Path]
		if !ok || seen[entry.Path] || entry.Size != int64(len(data)) || entry.SHA256 != hex.EncodeToString(hash[:]) {
			t.Fatalf("manifest differs from independent original: %q", entry.Path)
		}
		seen[entry.Path] = true
	}
}

func assertLiveTarMatchesOriginals(t *testing.T, ctx context.Context, home string, ciphertext []byte, id string, fixtures map[string][]byte, hashes map[string][32]byte) {
	t.Helper()
	// Use independently invoked real tools to audit every member, including files
	// excluded by the later literal-filtered production restore. Never spool tar.
	decrypt := exec.CommandContext(ctx, "gpg", "--no-options", "--homedir", home, "--batch", "--pinentry-mode", "loopback", "--passphrase", "", "--decrypt")
	decrypt.Stdin = bytes.NewReader(ciphertext)
	compressed, err := decrypt.Output()
	if err != nil {
		t.Fatal(err)
	}
	unzip := exec.CommandContext(ctx, "zstd", "--decompress", "--stdout", "--quiet")
	unzip.Stdin = bytes.NewReader(compressed)
	plain, err := unzip.Output()
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(bytes.NewReader(plain))
	manifestSeen := false
	seen := map[string]bool{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		if header.Name == "MANIFEST.json" {
			if manifestSeen || len(seen) != 0 {
				t.Fatal("manifest missing first/unique invariant")
			}
			manifestSeen = true
			assertLiveManifest(t, raw, id, fixtures, hashes)
			continue
		}
		name := strings.TrimPrefix(header.Name, "files/")
		want, ok := fixtures[name]
		if !manifestSeen || header.Name != "files/"+name || !ok || seen[name] || !bytes.Equal(raw, want) || sha256.Sum256(raw) != hashes[name] {
			t.Fatalf("independent decrypted member mismatch: %q", header.Name)
		}
		seen[name] = true
	}
	if !manifestSeen || len(seen) != len(fixtures) {
		t.Fatal("independent recovery omitted a member")
	}
}

func assertLivePublishedTree(t *testing.T, target, id string, fixtures map[string][]byte, hashes map[string][32]byte, paths []string) {
	t.Helper()
	selected := map[string]bool{}
	for name := range fixtures {
		if len(paths) == 0 {
			selected[name] = true
		}
	}
	for _, name := range paths {
		selected[name] = true
	}
	entries, err := os.ReadDir(target)
	if err != nil || len(entries) != 1 || entries[0].Name() != id {
		t.Fatalf("published bundle set: %+v %v", entries, err)
	}
	raw, err := os.ReadFile(filepath.Join(target, id, "MANIFEST.json"))
	if err != nil {
		t.Fatal(err)
	}
	assertLiveManifest(t, raw, id, fixtures, hashes)
	seen := map[string]bool{}
	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Mode().Perm() != 0700 {
				t.Errorf("nonprivate restored directory: %q", path)
			}
			return nil
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
			t.Errorf("unsafe restored member mode: %q", path)
			return nil
		}
		rel, err := filepath.Rel(filepath.Join(target, id), path)
		if err != nil {
			return err
		}
		if rel == "MANIFEST.json" {
			return nil
		}
		name := strings.TrimPrefix(filepath.ToSlash(rel), "files/")
		got, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if filepath.ToSlash(rel) != "files/"+name || !selected[name] || seen[name] || !bytes.Equal(got, fixtures[name]) || sha256.Sum256(got) != hashes[name] {
			t.Errorf("unexpected/changed restored file: %q", name)
		}
		seen[name] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(seen, selected) {
		t.Fatalf("restored selection: got=%v want=%v", seen, selected)
	}
}

func assertLiveNoLocalAuthority(t *testing.T, cfg ha.ContinuousConfig, scratch string) {
	t.Helper()
	for _, path := range []string{cfg.Catalog, cfg.Catalog + ".lock", cfg.Catalog + ".uploads", cfg.Sink, cfg.Catalog + ".index-staging", filepath.Join(scratch, ".pair-restore-staging")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("unexpected local authority or stage: %q err=%v", path, err)
		}
	}
}
