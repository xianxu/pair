package historyarchive

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// copyCodec "encodes" by copying: enough to exercise the ciphertext bound.
type copyCodec struct{}

func (copyCodec) CheckRecipients(context.Context, []string) error { return nil }
func (copyCodec) Encode(_ context.Context, r io.Reader, w io.Writer, _ []string) error {
	_, err := io.Copy(w, r)
	return err
}

func TestRawBundleLimitIsKindBased(t *testing.T) {
	if rawBundleLimit("codex-database") != MaxSnapshotRawBytes || rawBundleLimit("claude-transcripts") != MaxBundleBytes {
		t.Fatal("snapshot kinds get the snapshot bound; others the general bound")
	}
	mt := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	big := int64(MaxBundleBytes) + 512<<20
	manifest := func(kind string) Manifest {
		return Manifest{Version: Version, Kind: kind, Source: SourceIdentity{Host: "h", Root: "/r"}, From: mt, To: mt, Compression: "zstd -19",
			Recipients: []string{strings.Repeat("A", 40)}, Files: []FileEntry{{Path: "thread_history_1.sqlite", Size: big, Mtime: mt, SHA256: strings.Repeat("a", 64)}}}
	}
	if err := ValidateManifest(manifest("codex-database")); err != nil {
		t.Fatalf("a 1.5 GiB snapshot manifest must be valid (catalog reads re-validate it): %v", err)
	}
	if err := ValidateManifest(manifest("claude-transcripts")); err == nil {
		t.Fatal("a 1.5 GiB transcript bundle must still be refused")
	}
	label := BundleLabel{Version: Version, ID: strings.Repeat("a", 64), Kind: "codex-database", From: mt, To: mt, FileCount: 1, RawBytes: big, Bytes: 300 << 20,
		Compression: "zstd -19", Recipients: []string{strings.Repeat("A", 40)}, SHA256: strings.Repeat("b", 64)}
	if err := ValidateLabel(label); err != nil {
		t.Fatalf("snapshot label: %v", err)
	}
	label.Kind = "claude-transcripts"
	if err := ValidateLabel(label); err == nil {
		t.Fatal("transcript label over the general bound accepted")
	}
}

func TestCiphertextOverflowIsTheSentinelAndASnapshotSkip(t *testing.T) {
	saved := maxCiphertext
	maxCiphertext = 64
	defer func() { maxCiphertext = saved }()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(base, "src")
	os.MkdirAll(src, 0o700)
	os.WriteFile(filepath.Join(src, "one"), []byte(strings.Repeat("x", 4096)), 0o600)
	req := ArchiveRequest{Version: Version, Kind: "ops-documents", Source: SourceIdentity{Host: "h", Root: src}, Files: []string{"one"},
		Recipients: []string{strings.Repeat("A", 40)}, Sink: filepath.Join(base, "sink"), Catalog: filepath.Join(base, "catalog", "c.jsonl")}
	a := Archiver{Codec: copyCodec{}, Sink: &LocalSink{Root: req.Sink}}
	if _, err := a.Create(context.Background(), req); !errors.Is(err, ErrCiphertextLimit) {
		t.Fatalf("overflow error %v, want ErrCiphertextLimit", err)
	}
	// In a continuous pass, an oversized snapshot is a skip, not a failure.
	home := filepath.Join(base, "home")
	os.MkdirAll(filepath.Join(home, ".codex"), 0o700)
	os.WriteFile(filepath.Join(home, ".codex", "big.sqlite"), []byte(strings.Repeat("d", 16384)), 0o600)
	os.WriteFile(filepath.Join(home, ".codex", "small.sqlite"), []byte("s"), 0o600)
	maxCiphertext = 8192 // the small snapshot's bundle fits; the big one's does not
	cfg := ContinuousConfig{Version: Version, Recipients: req.Recipients, Sink: req.Sink, Catalog: req.Catalog}
	pass := ContinuousPass{Archiver: a, Config: cfg, Store: filepath.Join(base, "store"), SnapshotRoot: filepath.Join(base, "snaps"),
		Host: "h", Home: home, Env: map[string]string{}, Kinds: []string{"codex-database"}, Snapshotter: copySnapshotter{}}
	report, err := pass.Run(context.Background())
	if err != nil || len(report.Skipped) != 1 || !strings.Contains(report.Skipped[0], "ciphertext bound") {
		t.Fatalf("oversized snapshot: err=%v skipped=%v, want one skip note and no failure", err, report.Skipped)
	}
	if report.Archived["codex-database"] != 1 {
		t.Fatalf("archived %+v; the oversized snapshot must not take the other one with it", report.Archived)
	}
}

type copySnapshotter struct{}

func (copySnapshotter) Snapshot(_ context.Context, root, rel, staging string) (SnapshotResult, error) {
	os.MkdirAll(staging, 0o700)
	return SnapshotResult{Status: SnapshotReady}, copyRegularFile(filepath.Join(root, rel), filepath.Join(staging, rel))
}
func (copySnapshotter) Sweep() error { return nil }
