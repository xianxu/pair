package historyarchive

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func catalogFixture(t *testing.T, id string) CatalogRecord {
	t.Helper()
	stamp := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	m := Manifest{Version: 1, Kind: "ops-documents", Source: SourceIdentity{Host: "test", Root: "/fixture"}, From: stamp, To: stamp, Compression: "zstd -19", Recipients: []string{strings.Repeat("A", 40)}, Files: []FileEntry{{Path: id, Size: 1, Mtime: stamp, SHA256: strings.Repeat("b", 64)}}}
	mid, err := m.ID()
	if err != nil {
		t.Fatal(err)
	}
	b, l, err := m.Keys()
	if err != nil {
		t.Fatal(err)
	}
	return CatalogRecord{Manifest: m, Label: BundleLabel{Version: 1, ID: mid, Kind: m.Kind, From: stamp, To: stamp, FileCount: 1, RawBytes: 1, Bytes: 10, Compression: m.Compression, Recipients: m.Recipients, SHA256: strings.Repeat("c", 64)}, Sink: "/sink", BundleKey: b, LabelKey: l, VerifiedAt: stamp}
}
func catalogPath(t *testing.T) string {
	t.Helper()
	d, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	return filepath.Join(d, "catalog.jsonl")
}
func TestCatalogRepairsTornTailAndAdoptsOnce(t *testing.T) {
	ctx := context.Background()
	p := catalogPath(t)
	r := catalogFixture(t, "one")
	c, e := OpenCatalog(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	if e = c.Append(ctx, r); e != nil {
		t.Fatal(e)
	}
	c.Close()
	f, e := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	f.WriteString(`{"torn":`)
	f.Close()
	c, e = OpenCatalog(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	got, ok, e := c.Find(ctx, r.Label.ID)
	if e != nil || !ok || got.Label.ID != r.Label.ID {
		t.Fatalf("find: %v %v", ok, e)
	}
	if e = c.Append(ctx, r); e != nil {
		t.Fatal(e)
	}
	raw, _ := os.ReadFile(p)
	if strings.Count(string(raw), "\n") != 1 || strings.Contains(string(raw), "torn") {
		t.Fatalf("tail not repaired: %s", raw)
	}
}
func TestCatalogRollsAndSearchesEarlierShards(t *testing.T) {
	ctx := context.Background()
	p := catalogPath(t)
	r := catalogFixture(t, "one")
	raw, _ := json.Marshal(r)
	c, e := openCatalog(ctx, p, int64(len(raw)+1))
	if e != nil {
		t.Fatal(e)
	}
	if e = c.Append(ctx, r); e != nil {
		t.Fatal(e)
	}
	second := catalogFixture(t, "two")
	if e = c.Append(ctx, second); e != nil {
		t.Fatal(e)
	}
	c.Close()
	if _, e = os.Stat(p + ".000001"); e != nil {
		t.Fatal(e)
	}
	c, e = openCatalog(ctx, p, int64(len(raw)+1))
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	if _, ok, e := c.Find(ctx, r.Label.ID); e != nil || !ok {
		t.Fatalf("earlier shard lost: %v", e)
	}
	if e = c.Append(ctx, r); e != nil {
		t.Fatal(e)
	}
}
func TestCatalogRefusesCompleteCorruptionAndConcurrentWriter(t *testing.T) {
	ctx := context.Background()
	p := catalogPath(t)
	c, e := OpenCatalog(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = OpenCatalog(ctx, p); e == nil {
		t.Fatal("concurrent writer admitted")
	}
	c.Close()
	if e = os.WriteFile(p, []byte("{broken}\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if c, e = OpenCatalog(ctx, p); e == nil {
		c.Close()
		t.Fatal("complete corruption accepted")
	}
}
func TestCatalogRepeatedSyncFailureNeverReportsSuccess(t *testing.T) {
	ctx := context.Background()
	p := catalogPath(t)
	c, e := OpenCatalog(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	r := catalogFixture(t, "one")
	syncs := 0
	c.fault = func(stage string) error {
		syncs++
		if syncs > 1 {
			return os.ErrPermission
		}
		return nil
	}
	if e = c.Append(ctx, r); e == nil {
		t.Fatal("failed sync reported success")
	}
	c.Close()
	c, e = OpenCatalog(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	c.fault = func(stage string) error { return os.ErrPermission }
	if _, _, e = c.Find(ctx, r.Label.ID); e == nil {
		t.Fatal("retry did not reconfirm durability")
	}
	c.fault = nil
	if e = c.Append(ctx, r); e != nil {
		t.Fatal(e)
	}
	raw, _ := os.ReadFile(p)
	if strings.Count(string(raw), "\n") != 1 {
		t.Fatal("duplicate append")
	}
}

func TestCatalogFailedOpenReleasesLock(t *testing.T) {
	p := catalogPath(t)
	ctx := context.Background()
	if e := os.WriteFile(p, []byte("invalid\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if c, e := OpenCatalog(ctx, p); e == nil {
		c.Close()
		t.Fatal("accepted corrupt record")
	}
	if e := os.WriteFile(p, nil, 0600); e != nil {
		t.Fatal(e)
	}
	c, e := OpenCatalog(ctx, p)
	if e != nil {
		t.Fatalf("failed open leaked catalog lock: %v", e)
	}
	c.Close()
}
