package historyarchive

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func pureRequest() ArchiveRequest {
	return ArchiveRequest{Version: 1, Kind: "ops-documents", Source: SourceIdentity{Host: "test", Root: "/tmp/input"}, Files: []string{"a"}, Recipients: []string{strings.Repeat("A", 40)}, Sink: "/tmp/sink", Catalog: "/tmp/catalog"}
}
func pureManifest() Manifest {
	tm := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	return Manifest{Version: 1, Kind: "ops-documents", Source: pureRequest().Source, From: tm, To: tm, Compression: "zstd -19", Recipients: pureRequest().Recipients, Files: []FileEntry{{Path: "a", Size: 1, Mtime: tm, SHA256: strings.Repeat("a", 64)}}}
}
func TestRequestStrictAndPaths(t *testing.T) {
	r := pureRequest()
	b, _ := json.Marshal(r)
	if _, err := DecodeRequest(b); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{string(b) + " {}", strings.Replace(string(b), `"version":1`, `"version":1,"oops":2`, 1), `null`} {
		if _, err := DecodeRequest([]byte(bad)); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	for _, p := range []string{"", "/a", "../a", "a/../b", "a//b", "a/./b", `a\b`, "a\x00b"} {
		r := pureRequest()
		r.Files = []string{p}
		if ValidateRequest(r) == nil {
			t.Errorf("accepted %q", p)
		}
	}
	r.Files = []string{"a", "a"}
	if ValidateRequest(r) == nil {
		t.Fatal("duplicate accepted")
	}
	r = pureRequest()
	r.Kind = "codex-database"
	if ValidateRequest(r) == nil {
		t.Fatal("live database accepted")
	}
}
func TestManifestIdentityAndBounds(t *testing.T) {
	m := pureManifest()
	m.Files = append(m.Files, FileEntry{Path: "b", Size: 2, Mtime: m.From, SHA256: strings.Repeat("b", 64)})
	id, err := m.ID()
	if err != nil {
		t.Fatal(err)
	}
	m.Files[0], m.Files[1] = m.Files[1], m.Files[0]
	other, _ := m.ID()
	if id != other {
		t.Fatal("order changed identity")
	}
	m.Source.Host = "other"
	other, _ = m.ID()
	if id == other {
		t.Fatal("source absent from identity")
	}
	cases := []func(*Manifest){func(m *Manifest) { m.Version = 2 }, func(m *Manifest) { m.Files[0].Size = MaxBundleBytes + 1 }, func(m *Manifest) { m.Files[0].SHA256 = "bad" }, func(m *Manifest) { m.Files[0].Mtime = time.Time{} }, func(m *Manifest) { m.To = m.From.Add(-time.Second) }, func(m *Manifest) { m.Recipients = []string{"bad"} }, func(m *Manifest) { m.Files = append(m.Files, m.Files[0]) }}
	for i, f := range cases {
		m := pureManifest()
		f(&m)
		if ValidateManifest(m) == nil {
			t.Errorf("case %d accepted", i)
		}
	}
	m = pureManifest()
	bundle, label, err := m.Keys()
	if err != nil || !strings.HasPrefix(bundle, "ops-documents/2026/01/") || !strings.HasSuffix(bundle, ".tar.zst.gpg") || !strings.HasSuffix(label, ".json") {
		t.Fatalf("keys %s %s %v", bundle, label, err)
	}
}
func TestPlanPartitions(t *testing.T) {
	r := pureRequest()
	m := pureManifest()
	files := []FileEntry{m.Files[0], m.Files[0], m.Files[0]}
	files[0].Path = "a"
	files[1].Path = "b"
	files[2].Path = "c"
	files[0].Size = TargetBundleBytes
	files[1].Size = 1
	files[2].Mtime = m.From.AddDate(0, 1, 0)
	r.Files = []string{"a", "b", "c"}
	plans, err := PlanBundles(r, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 3 {
		t.Fatalf("got %d bundles", len(plans))
	}
	seen := map[string]bool{}
	for _, p := range plans {
		if err := ValidateManifest(p); err != nil {
			t.Fatal(err)
		}
		for _, f := range p.Files {
			if seen[f.Path] {
				t.Fatal("duplicate")
			}
			seen[f.Path] = true
		}
	}
	if len(seen) != 3 {
		t.Fatal("lost files")
	}
	files[0].Size = MaxBundleBytes + 1
	if _, err := PlanBundles(r, files); err == nil {
		t.Fatal("oversize accepted")
	}
}
func TestPublicationDurableOnly(t *testing.T) {
	for n := 0; n < 16; n++ {
		o := PublicationObservation{BundlePresent: n&1 != 0, LabelPresent: n&2 != 0, Matches: n&4 != 0, Durable: n&8 != 0}
		want := PublicationConflict
		if !o.BundlePresent && !o.LabelPresent {
			want = PublicationAbsent
		} else if o.BundlePresent && o.LabelPresent && o.Matches && o.Durable {
			want = PublicationVerifiedExisting
		}
		if got := DecidePublication(o); got != want {
			t.Fatalf("%+v: %s", o, got)
		}
	}
}

func TestPlanMetadataPartitions(t *testing.T) {
	r := pureRequest()
	r.Files = nil
	var files []FileEntry
	for n := 0; n < 9000; n++ {
		p := fmt.Sprintf("%05d-%s", n, strings.Repeat("x", 350))
		r.Files = append(r.Files, p)
		f := pureManifest().Files[0]
		f.Path = p
		files = append(files, f)
	}
	plans, err := PlanBundles(r, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) < 2 {
		t.Fatal("metadata not partitioned")
	}
	count := 0
	for _, m := range plans {
		count += len(m.Files)
		if err := ValidateManifest(m); err != nil {
			t.Fatal(err)
		}
	}
	if count != len(files) {
		t.Fatal("files lost")
	}
}
func TestRecordRejectsInconsistentEvidence(t *testing.T) {
	m := pureManifest()
	id, _ := m.ID()
	b, l, _ := m.Keys()
	r := CatalogRecord{Manifest: m, Label: BundleLabel{Version: 1, ID: id, Kind: m.Kind, From: m.From, To: m.To, FileCount: 1, RawBytes: 1, Bytes: 100, Compression: m.Compression, Recipients: m.Recipients, SHA256: strings.Repeat("b", 64)}, Sink: "local", BundleKey: b, LabelKey: l, VerifiedAt: m.From}
	if err := ValidateRecord(r); err != nil {
		t.Fatal(err)
	}
	cases := []func(*CatalogRecord){func(r *CatalogRecord) { r.Label.RawBytes++ }, func(r *CatalogRecord) { r.Label.ID = strings.Repeat("b", 64) }, func(r *CatalogRecord) { r.BundleKey = "other" }, func(r *CatalogRecord) { r.VerifiedAt = time.Time{} }, func(r *CatalogRecord) { r.Label.Recipients = []string{strings.Repeat("B", 40)} }, func(r *CatalogRecord) { r.Label.Bytes = MaxCiphertextBytes + 1 }}
	for n, f := range cases {
		c := r
		f(&c)
		if ValidateRecord(c) == nil {
			t.Errorf("inconsistent evidence %d accepted", n)
		}
	}
}
func TestRequestFingerprintNormalization(t *testing.T) {
	r := pureRequest()
	r.Recipients = []string{strings.Repeat("a", 40)}
	b, _ := json.Marshal(r)
	got, err := DecodeRequest(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.Recipients[0] != strings.Repeat("A", 40) {
		t.Fatal("not normalized")
	}
	r.Recipients = append(r.Recipients, strings.Repeat("A", 40))
	if ValidateRequest(r) == nil {
		t.Fatal("case-duplicate recipients accepted")
	}
}
func TestRequestRootDefaults(t *testing.T) {
	r := pureRequest()
	r.Kind = "codex-transcripts"
	r.Source.Root = ""
	b, _ := json.Marshal(r)
	if _, err := DecodeRequest(b); err == nil {
		t.Fatal("explicit decoder accepted missing root")
	}
	got, err := DecodeRequestWithRoots(b, "/home/me", map[string]string{"CODEX_HOME": "/agent"})
	if err != nil || got.Source.Root != "/agent" {
		t.Fatalf("default root %+v %v", got, err)
	}
	r.Kind = "ops-documents"
	b, _ = json.Marshal(r)
	if _, err := DecodeRequestWithRoots(b, "/home/me", nil); err == nil {
		t.Fatal("invented ops root")
	}
}
func TestPlanBoundaryAdmission(t *testing.T) {
	r := pureRequest()
	f := pureManifest().Files[0]
	f.Size = MaxBundleBytes
	p, err := PlanBundles(r, []FileEntry{f})
	if err != nil || len(p) != 1 {
		t.Fatalf("max file refused: %v", err)
	}
	f.Size = 0
	p, err = PlanBundles(r, []FileEntry{f})
	if err != nil || len(p) != 1 {
		t.Fatalf("empty file refused: %v", err)
	}
	r.Files = make([]string, MaxFiles+1)
	if ValidateRequest(r) == nil {
		t.Fatal("file count exceeded")
	}
	r = pureRequest()
	r.Source.Host = strings.Repeat("h", MaxMetadataBytes)
	if ValidateRequest(r) == nil {
		t.Fatal("metadata exceeded")
	}
	m := pureManifest()
	m.Files[0].Mtime = m.From.In(time.FixedZone("offset", 3600))
	if ValidateManifest(m) == nil {
		t.Fatal("non UTC accepted")
	}
}
