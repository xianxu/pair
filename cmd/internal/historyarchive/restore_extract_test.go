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
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type extractionMember struct {
	name string
	data []byte
	kind byte
}

func extractionFixture(t *testing.T) (Manifest, BundleLabel, []extractionMember) {
	t.Helper()
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	m := Manifest{Version: Version, Kind: "ops-documents", Source: SourceIdentity{Host: "fixture", Root: "/original-restore-source"}, From: at, To: at, Compression: "zstd -19", Recipients: []string{strings.Repeat("A", 40)}}
	members := []extractionMember{{"files/notes/你好.txt", []byte("hello\x00\xff"), tar.TypeReg}, {"files/empty", nil, tar.TypeReg}}
	for _, member := range members {
		sum := sha256.Sum256(member.data)
		m.Files = append(m.Files, FileEntry{Path: strings.TrimPrefix(member.name, "files/"), Size: int64(len(member.data)), Mtime: at, SHA256: hex.EncodeToString(sum[:])})
	}
	id, err := m.ID()
	if err != nil {
		t.Fatal(err)
	}
	label := BundleLabel{Version: Version, ID: id, Kind: m.Kind, From: at, To: at, FileCount: len(m.Files), RawBytes: int64(len(members[0].data)), Bytes: 1, Compression: m.Compression, Recipients: m.Recipients, SHA256: strings.Repeat("a", 64)}
	return m, label, members
}
func extractionTar(t *testing.T, manifest []byte, members []extractionMember) []byte {
	t.Helper()
	var out bytes.Buffer
	tw := tar.NewWriter(&out)
	all := append([]extractionMember{{"MANIFEST.json", manifest, tar.TypeReg}}, members...)
	for _, member := range all {
		h := &tar.Header{Name: member.name, Typeflag: member.kind, Mode: 07777, Uid: 123, Gid: 123, Size: int64(len(member.data))}
		if member.kind != tar.TypeReg {
			h.Size = 0
			h.Linkname = "/outside"
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			if _, err := tw.Write(member.data); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func extractionOutput(t *testing.T) (*os.File, string) {
	t.Helper()
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "output")
	if err = os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f, parent
}
func TestExtractBundleRestoresExactBytesAndPrivateModes(t *testing.T) {
	m, label, members := extractionFixture(t)
	raw, _ := json.Marshal(m)
	output, parent := extractionOutput(t)
	budget := &RestoreExtractionBudget{Entries: 1}
	got, count, err := ExtractBundle(context.Background(), bytes.NewReader(extractionTar(t, raw, members)), label, output, parent, nil, budget)
	if err != nil || count != 2 || len(got.Files) != 2 || budget.Files != 2 || budget.Entries != 6 {
		t.Fatalf("extract count=%d budget=%+v manifest=%+v error=%v", count, budget, got, err)
	}
	for _, member := range members {
		path := filepath.Join(parent, "output", member.name)
		b, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(b, member.data) {
			t.Fatalf("bytes %s: %x %v", member.name, b, err)
		}
		st, err := os.Stat(path)
		if err != nil || st.Mode().Perm() != 0600 {
			t.Fatalf("file permissions %v %v", st, err)
		}
	}
	for _, name := range []string{"files", "files/notes"} {
		st, err := os.Stat(filepath.Join(parent, "output", name))
		if err != nil || st.Mode().Perm() != 0700 {
			t.Fatalf("dir permissions %v %v", st, err)
		}
	}
	if _, err = os.Stat(filepath.Join(parent, "output", "MANIFEST.json")); err != nil {
		t.Fatal(err)
	}
}
func TestExtractBundleLiteralFiltersValidateExcludedBytes(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(string(rune('a'+btoi(corrupt))), func(t *testing.T) {
			m, label, members := extractionFixture(t)
			raw, _ := json.Marshal(m)
			if corrupt {
				members[0].data = []byte("evil!!!")
			}
			output, parent := extractionOutput(t)
			budget := &RestoreExtractionBudget{}
			_, count, err := ExtractBundle(context.Background(), bytes.NewReader(extractionTar(t, raw, members)), label, output, parent, []string{"empty", "empty"}, budget)
			if (err != nil) != corrupt {
				t.Fatalf("corrupt=%v count=%d error=%v", corrupt, count, err)
			}
			if !corrupt && (count != 1 || budget.Files != 1) {
				t.Fatalf("filtered count=%d budget=%+v", count, budget)
			}
			if _, err = os.Stat(filepath.Join(parent, "output", "files", "notes")); !os.IsNotExist(err) {
				t.Fatal("excluded path written")
			}
		})
	}
}
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
func TestExtractBundleRejectsTarAttacks(t *testing.T) {
	for _, kind := range []string{"duplicate", "missing", "unexpected", "traversal", "symlink", "hardlink", "device", "directory", "wrong-size", "trailing", "manifest-unknown", "manifest-duplicate", "manifest-not-first", "summary-raw", "summary-count", "summary-time", "summary-recipients", "manifest-id", "overlap"} {
		t.Run(kind, func(t *testing.T) {
			m, label, members := extractionFixture(t)
			raw, _ := json.Marshal(m)
			output, parent := extractionOutput(t)
			switch kind {
			case "duplicate":
				members = append(members, members[0])
			case "missing":
				members = members[:1]
			case "unexpected":
				members = append(members, extractionMember{"files/unknown", nil, tar.TypeReg})
			case "traversal":
				members[0].name = "files/../outside"
			case "symlink":
				members[0].kind = tar.TypeSymlink
			case "hardlink":
				members[0].kind = tar.TypeLink
			case "device":
				members[0].kind = tar.TypeChar
			case "directory":
				members[0].kind = tar.TypeDir
			case "wrong-size":
				members[0].data = []byte("short")
			case "manifest-unknown":
				raw = append(raw[:len(raw)-1], []byte(`,"future":true}`)...)
			case "manifest-duplicate":
				raw = append(raw[:len(raw)-1], []byte(`,"version":1}`)...)
			case "summary-raw":
				label.RawBytes++
			case "summary-count":
				label.FileCount++
			case "summary-time":
				label.To = label.To.Add(time.Second)
			case "summary-recipients":
				label.Recipients = []string{strings.Repeat("B", 40)}
			case "manifest-id":
				label.ID = strings.Repeat("b", 64)
			case "overlap":
				m.Source.Root = filepath.Join(parent, "source")
				raw, _ = json.Marshal(m)
				label.ID, _ = m.ID()
			}
			plain := extractionTar(t, raw, members)
			if kind == "trailing" {
				plain = append(plain, []byte("hidden data")...)
			}
			if kind == "manifest-not-first" {
				plain = extractionTar(t, raw, members)
				copy(plain[:len("MANIFEST.json")], []byte("not-manifest!"))
			}
			if _, _, err := ExtractBundle(context.Background(), bytes.NewReader(plain), label, output, parent, nil, &RestoreExtractionBudget{}); err == nil {
				t.Fatal("unsafe tar accepted")
			}
			if kind == "overlap" {
				entries, err := os.ReadDir(output.Name())
				if err != nil || len(entries) != 0 {
					t.Fatalf("overlap wrote output: %v %v", entries, err)
				}
			}
		})
	}
}
func TestExtractBundleCanceledAndBounded(t *testing.T) {
	m, label, members := extractionFixture(t)
	raw, _ := json.Marshal(m)
	for _, kind := range []string{"cancel", "entry-limit", "file-limit", "entry-near-limit"} {
		t.Run(kind, func(t *testing.T) {
			output, parent := extractionOutput(t)
			budget := &RestoreExtractionBudget{}
			ctx := context.Background()
			switch kind {
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "entry-limit":
				budget.Entries = MaxRestoreEntries
			case "entry-near-limit":
				budget.Entries = MaxRestoreEntries - 1
			case "file-limit":
				budget.Files = MaxRestoreFiles
			}
			if _, _, err := ExtractBundle(ctx, bytes.NewReader(extractionTar(t, raw, members)), label, output, parent, nil, budget); err == nil {
				t.Fatal("limit/cancellation accepted")
			}
			if budget.Entries > MaxRestoreEntries || budget.Files > MaxRestoreFiles {
				t.Fatalf("budget exceeded: %+v", budget)
			}
		})
	}
}
func TestRestoreCreationHelpersRefuseSymlinksAndOverwrite(t *testing.T) {
	output, parent := extractionOutput(t)
	budget := &RestoreExtractionBudget{}
	f, err := CreateRestoreFile(output, "a/binary", budget)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if budget.Entries != 2 || budget.Files != 0 {
		t.Fatalf("helper budget %+v", budget)
	}
	if f, err = CreateRestoreFile(output, "a/binary", budget); err == nil {
		f.Close()
		t.Fatal("overwrite accepted")
	}
	if err = os.Symlink(t.TempDir(), filepath.Join(parent, "output", "link")); err != nil {
		t.Fatal(err)
	}
	if f, err = CreateRestoreFile(output, "link/escape", budget); err == nil {
		f.Close()
		t.Fatal("symlink followed")
	}
	if f, err = CreateRestoreFile(output, "../escape", budget); err == nil {
		f.Close()
		t.Fatal("traversal accepted")
	}
	budget.Entries = MaxRestoreEntries
	if f, err = CreateRestoreDir(output, "too-many", budget); err == nil {
		f.Close()
		t.Fatal("directory bound exceeded")
	}
	if _, err = os.Stat(filepath.Join(parent, "output", "too-many")); !os.IsNotExist(err) {
		t.Fatal("bound refused after mkdir")
	}
}

func TestExtractBundleRejectsDecompressedPaddingOverflow(t *testing.T) {
	for _, kind := range []string{"ops-documents", "codex-database"} {
		for _, extra := range []int64{0, 1} {
			name := kind + "/exact"
			if extra != 0 {
				name = kind + "/one-byte-over"
			}
			t.Run(name, func(t *testing.T) {
				m, label, members := extractionFixture(t)
				m.Kind, label.Kind = kind, kind
				var err error
				label.ID, err = m.ID()
				if err != nil {
					t.Fatal(err)
				}
				raw, err := json.Marshal(m)
				if err != nil {
					t.Fatal(err)
				}
				base := extractionTar(t, raw, members)
				limit := label.RawBytes + MaxRestoreTarOverhead
				if int64(len(base)) > limit {
					t.Fatal("invalid padding fixture")
				}
				plain := io.MultiReader(bytes.NewReader(base), io.LimitReader(zeroExtractionReader{}, limit-int64(len(base))+extra))
				output, parent := extractionOutput(t)
				_, _, err = ExtractBundle(context.Background(), plain, label, output, parent, nil, &RestoreExtractionBudget{})
				if extra == 0 && err != nil {
					t.Fatalf("exact declared payload plus overhead refused: %v", err)
				}
				if extra != 0 && (err == nil || !strings.Contains(err.Error(), "decompressed restore stream exceeds bound")) {
					t.Fatalf("declared payload plus overhead exceeded without refusal: %v", err)
				}
			})
		}
	}
}

type zeroExtractionReader struct{}

func (zeroExtractionReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestRestorePlainReaderCountsHiddenPAXRecords(t *testing.T) {
	var plain bytes.Buffer
	tw := tar.NewWriter(&plain)
	if err := tw.WriteHeader(&tar.Header{Name: "MANIFEST.json", Typeflag: tar.TypeReg, Size: 2, Mode: 0600, Format: tar.FormatPAX, PAXRecords: map[string]string{"comment": strings.Repeat("x", 4096)}}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("{}")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	// archive/tar hides the extension header from callers, but its bytes still
	// consume the same lower-level budget as visible member data.
	header, err := tar.NewReader(bytes.NewReader(plain.Bytes())).Next()
	if err != nil || header.Name != "MANIFEST.json" {
		t.Fatalf("invalid PAX fixture: %+v %v", header, err)
	}
	bounded := &restorePlainReader{ctx: context.Background(), reader: bytes.NewReader(plain.Bytes()), remaining: 1024}
	if _, err = tar.NewReader(bounded).Next(); err == nil || !strings.Contains(err.Error(), "decompressed restore stream exceeds bound") {
		t.Fatalf("hidden PAX escaped budget: %v", err)
	}
}

func TestExtractBundleFilterMayMissThisBundle(t *testing.T) {
	m, label, members := extractionFixture(t)
	raw, _ := json.Marshal(m)
	output, parent := extractionOutput(t)
	budget := &RestoreExtractionBudget{}
	_, selected, err := ExtractBundle(context.Background(), bytes.NewReader(extractionTar(t, raw, members)), label, output, parent, []string{"present-in-another-bundle"}, budget)
	if err != nil || selected != 0 || budget.Files != 0 {
		t.Fatalf("bundle-local filter miss: %d %+v %v", selected, budget, err)
	}
	if _, err = os.Stat(filepath.Join(parent, "output", "MANIFEST.json")); err != nil {
		t.Fatal(err)
	}
}

func TestExtractBundleQuotesUntrustedMemberDiagnostics(t *testing.T) {
	m, label, members := extractionFixture(t)
	m.Files[0].Path = "untrusted-\x1b]52;c;ZmFrZQ==\a\x7f\u009b"
	members[0].name = "files/" + m.Files[0].Path
	var err error
	label.ID, err = m.ID()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	members[0].data[0] ^= 1
	output, parent := extractionOutput(t)
	_, _, err = ExtractBundle(context.Background(), bytes.NewReader(extractionTar(t, raw, members)), label, output, parent, nil, &RestoreExtractionBudget{Entries: 1})
	if err == nil || !strings.Contains(err.Error(), "checksum/size mismatch") {
		t.Fatalf("corruption not exercised: %v", err)
	}
	if strings.ContainsAny(err.Error(), "\x1b\a\x7f\u009b") {
		t.Fatalf("raw untrusted member in diagnostic: %q", err.Error())
	}
	if !strings.Contains(err.Error(), `untrusted-\x1b`) {
		t.Fatalf("quoted member identity missing: %q", err.Error())
	}
}
