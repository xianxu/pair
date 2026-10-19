package historyarchive

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
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/archivetestenv"
)

func TestCodecLiveRoundTrip(t *testing.T) {
	if os.Getenv("PAIR_ARCHIVE_LIVE") != "1" {
		t.Skip("set PAIR_ARCHIVE_LIVE=1 for isolated real GPG/zstd conformance")
	}
	testStart := time.Now()
	for _, name := range []string{"gpg", "gpgconf", "zstd"} {
		if _, err := exec.LookPath(name); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	home, fp := key.Home, key.Fingerprint
	c := CommandCodec{Home: home}
	if err := c.CheckRecipients(ctx, []string{fp}); err != nil {
		t.Fatal(err)
	}
	var plain bytes.Buffer
	tw := tar.NewWriter(&plain)
	for _, f := range []struct {
		name string
		data []byte
	}{{"empty", nil}, {"unicode-雪", []byte("hello")}, {"binary", []byte{0, 255, 1, 2}}} {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0600, Size: int64(len(f.data))}); err != nil {
			t.Fatal(err)
		}
		tw.Write(f.data)
	}
	tw.Close()
	for _, descriptor := range []string{"zstd -9", "zstd -19", "zstd -20", "zstd -22"} {
		var cipher bytes.Buffer
		start := time.Now()
		if err := c.Encode(ctx, bytes.NewReader(plain.Bytes()), &cipher, []string{fp}, descriptor); err != nil {
			t.Fatal(err)
		}
		dec := exec.CommandContext(ctx, "gpg", "--no-options", "--homedir", home, "--batch", "--pinentry-mode", "loopback", "--passphrase", "", "--decrypt")
		dec.Stdin = bytes.NewReader(cipher.Bytes())
		compressed, err := dec.Output()
		if err != nil {
			t.Fatal(err)
		}
		unzip := exec.CommandContext(ctx, "zstd", "--decompress", "--stdout", "--quiet")
		unzip.Stdin = bytes.NewReader(compressed)
		got, err := unzip.Output()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, plain.Bytes()) {
			t.Fatal("independent decrypt/decompress differs")
		}
		t.Logf("round trip: tar=%d ciphertext=%d elapsed=%s (%s -T0; no strict RSS claim)", plain.Len(), cipher.Len(), time.Since(start), descriptor)

	}

	// Exercise the production coordinator and filesystem sink with this same
	// disposable key. Canonical paths avoid macOS /tmp symlink traversal.
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	fixtures := map[string][]byte{"empty-file": {}, "unicode-雪.txt": []byte("hello 雪\n"), "binary-file": {0, 255, 1, 2, 0, 128}}
	hashes := map[string][32]byte{}
	var names []string
	for name, data := range fixtures {
		if err := os.WriteFile(filepath.Join(source, name), data, 0600); err != nil {
			t.Fatal(err)
		}
		hashes[name] = sha256.Sum256(data)
		names = append(names, name)
	}
	req := ArchiveRequest{Version: 1, Kind: "ops-documents", Source: SourceIdentity{Host: "isolated-live-fixture", Root: source}, Files: names, Recipients: []string{fp}, Sink: filepath.Join(root, "sink"), Catalog: filepath.Join(root, "catalog", "history.jsonl")}
	sink := LocalSink{Root: req.Sink}
	pipelineStart := time.Now()
	records, err := (Archiver{Codec: c, Sink: sink}).Create(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("want one bundle, got %d", len(records))
	}
	record := records[0]
	catalog, err := OpenCatalog(ctx, req.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	persisted, found, findErr := catalog.Find(ctx, record.Label.ID)
	catalogCloseErr := catalog.Close()
	if findErr != nil {
		t.Fatal(findErr)
	}
	if catalogCloseErr != nil {
		t.Fatal(catalogCloseErr)
	}
	if !found || !reflect.DeepEqual(persisted, record) {
		t.Fatal("reopened catalog does not match returned verified record")
	}
	if err := sink.Verify(ctx, record.BundleKey, record.Label.SHA256, record.Label.Bytes); err != nil {
		t.Fatal(err)
	}
	reader, err := sink.Get(ctx, record.BundleKey)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := io.ReadAll(reader)
	closeErr := reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	digest := sha256.Sum256(ciphertext)
	if hex.EncodeToString(digest[:]) != record.Label.SHA256 || int64(len(ciphertext)) != record.Label.Bytes {
		t.Fatal("independent ciphertext digest/size mismatch")
	}
	// Labels can be read without decryption; they must not expose source names.
	labelReader, err := sink.Get(ctx, record.LabelKey)
	if err != nil {
		t.Fatal(err)
	}
	labelBytes, err := io.ReadAll(labelReader)
	labelReader.Close()
	if err != nil {
		t.Fatal(err)
	}
	var label BundleLabel
	if err := json.Unmarshal(labelBytes, &label); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(label, record.Label) {
		t.Fatal("published label differs")
	}
	for name := range fixtures {
		if bytes.Contains(labelBytes, []byte(name)) {
			t.Fatal("label exposed a source filename")
		}
	}
	if bytes.Contains(labelBytes, []byte(source)) {
		t.Fatal("label exposed source root")
	}
	// Independently invoke recovery tools; no production decode helper is used.
	decrypt := exec.CommandContext(ctx, "gpg", "--no-options", "--homedir", home, "--batch", "--pinentry-mode", "loopback", "--passphrase", "", "--decrypt")
	decrypt.Stdin = bytes.NewReader(ciphertext)
	packed, err := decrypt.Output()
	if err != nil {
		t.Fatal(err)
	}
	decompress := exec.CommandContext(ctx, "zstd", "--decompress", "--stdout", "--quiet")
	decompress.Stdin = bytes.NewReader(packed)
	recoveredTar, err := decompress.Output()
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(bytes.NewReader(recoveredTar))
	restored := map[string]bool{}
	var restoredManifest Manifest
	manifestSeen := false
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		payload, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		if h.Name == "MANIFEST.json" {
			if manifestSeen {
				t.Fatal("duplicate manifest")
			}
			manifestSeen = true
			if err := json.Unmarshal(payload, &restoredManifest); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if !strings.HasPrefix(h.Name, "files/") {
			t.Fatalf("unexpected tar entry %q", h.Name)
		}
		name := strings.TrimPrefix(h.Name, "files/")
		expected, ok := fixtures[name]
		if !ok || restored[name] || !bytes.Equal(payload, expected) {
			t.Fatalf("unexpected/duplicate/changed recovered file %q", name)
		}
		if sha256.Sum256(payload) != hashes[name] {
			t.Fatalf("recovered digest differs: %q", name)
		}
		restored[name] = true
	}
	if !manifestSeen || len(restored) != len(fixtures) {
		t.Fatal("missing manifest or fixture")
	}
	if !reflect.DeepEqual(restoredManifest, record.Manifest) {
		t.Fatal("recovered manifest differs from private catalog record")
	}
	for _, entry := range restoredManifest.Files {
		hash, ok := hashes[entry.Path]
		if !ok || entry.SHA256 != hex.EncodeToString(hash[:]) || entry.Size != int64(len(fixtures[entry.Path])) {
			t.Fatal("manifest file digest/size differs from independent fixture")
		}
	}
	// Independent reads prove the original sources were preserved.
	for name, hash := range hashes {
		data, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		if sha256.Sum256(data) != hash {
			t.Fatalf("source changed: %q", name)
		}
	}
	pipelineElapsed := time.Since(pipelineStart)
	// Corrupt only the disposable published bundle, then demand verification fail.
	ciphertext[0] ^= 0xff
	if err := os.WriteFile(filepath.Join(req.Sink, record.BundleKey), ciphertext, 0600); err != nil {
		t.Fatal(err)
	}
	if err := sink.Verify(ctx, record.BundleKey, record.Label.SHA256, record.Label.Bytes); err == nil {
		t.Fatal("verification accepted corrupt ciphertext")
	}
	t.Logf("full archive round trip: files=%d tar=%d ciphertext=%d pipeline=%s total-with-key-generation=%s; corrupted sink rejected, source hashes unchanged", len(fixtures), len(recoveredTar), record.Label.Bytes, pipelineElapsed, time.Since(testStart))

}
