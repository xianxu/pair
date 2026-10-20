package historyarchive

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestContinuousColdWALPublicationAndIsolation(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 unavailable")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(base, "home")
	source := filepath.Join(home, ".codex")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	cold := filepath.Join(source, "a_history.sqlite")
	script := `import sqlite3,sys
c=sqlite3.connect(sys.argv[1])
c.execute('PRAGMA journal_mode=WAL')
c.execute('CREATE TABLE opaque_future(payload BLOB, future INTEGER)')
c.execute("INSERT INTO opaque_future(rowid,payload,future) VALUES(731,X'00FF01',9007199254740993)")
c.commit()
c.close()
`
	if out, err := exec.Command("python3", "-c", script, cold).CombinedOutput(); err != nil {
		t.Fatalf("closed WAL fixture: %s %v", out, err)
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Lstat(cold + suffix); !os.IsNotExist(err) {
			t.Fatalf("cold fixture has sidecar %s: %v", suffix, err)
		}
	}
	original, err := os.ReadFile(cold)
	if err != nil {
		t.Fatal(err)
	}
	if !coldWALHeader(original) {
		t.Fatal("fixture not a closed WAL database")
	}
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	cfg := ContinuousConfig{Version: Version, Recipients: []string{strings.Repeat("A", 40)}, Sink: filepath.Join(base, "sink"), Catalog: filepath.Join(base, "catalog", "catalog.jsonl")}
	codec := &archiveTestCodec{}
	pass := ContinuousPass{Archiver: Archiver{Codec: codec, Sink: LocalSink{Root: cfg.Sink}, Now: func() time.Time { return now }}, Config: cfg, Store: filepath.Join(base, "store"), SnapshotRoot: filepath.Join(base, "staging"), Home: home, Host: "fixture", Env: map[string]string{}, Kinds: []string{"codex-database"}, Now: func() time.Time { return now }, Snapshotter: SQLiteSnapshotter{Scratch: filepath.Join(base, "scratch")}}
	records := func() []CatalogRecord {
		t.Helper()
		catalog, err := OpenCatalog(context.Background(), cfg.Catalog)
		if err != nil {
			t.Fatal(err)
		}
		defer catalog.Close()
		var rows []CatalogRecord
		if err := catalog.Each(context.Background(), func(row CatalogRecord) error { rows = append(rows, row); return nil }); err != nil {
			t.Fatal(err)
		}
		return rows
	}
	first, err := pass.Run(context.Background())
	if err != nil || first.Archived["codex-database"] != 1 || len(first.Databases) != 1 || first.Databases[0].Status != DatabaseArchived {
		t.Fatalf("first cold pass: %+v %v", first, err)
	}
	rows := records()
	if len(rows) != 1 {
		t.Fatalf("catalog: %+v", rows)
	}
	record := rows[0]
	if !record.VerifiedAt.Equal(now) || record.Manifest.Source.Repository != source || len(record.Manifest.Files) != 1 || record.Manifest.Files[0].Path != filepath.Base(cold) || first.Databases[0].Source.Repository != source {
		t.Fatalf("lost original source identity: %+v %+v", record, first.Databases)
	}
	bundle, err := os.ReadFile(filepath.Join(cfg.Sink, record.BundleKey))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(pass.SnapshotRoot, "codex-database", filepath.Base(cold))
	output, err := exec.Command("sqlite3", snapshot, "SELECT rowid,hex(payload),future FROM opaque_future;").CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "731|00FF01|9007199254740993" {
		t.Fatalf("opaque content or rowid changed: %q %v", output, err)
	}
	now = now.Add(72 * time.Hour)
	second, err := pass.Run(context.Background())
	if err != nil || second.Archived["codex-database"] != 0 || len(second.Databases) != 1 || second.Databases[0].Status != DatabaseUnchanged || !second.Databases[0].CheckedAt.Equal(now) || !second.Databases[0].LastVerified.Equal(record.VerifiedAt) || second.Databases[0].NeedsAttention(now, DefaultDatabaseMaxAge) || codec.calls != 1 || len(records()) != 1 {
		t.Fatalf("unchanged cold pass: %+v calls=%d %v", second, codec.calls, err)
	}
	if current, err := os.ReadFile(cold); err != nil || !bytes.Equal(current, original) {
		t.Fatalf("native cold database changed: %v", err)
	}
	// Retain its recognizable WAL header, but invalidate the first btree page.
	broken := append([]byte(nil), original...)
	broken[100] = 0xff
	if err := os.WriteFile(cold, broken, 0600); err != nil {
		t.Fatal(err)
	}
	active := walDatabase(t, source)
	mixed, err := pass.Run(context.Background())
	if err != nil || mixed.Archived["codex-database"] != 1 || len(mixed.Databases) != 2 {
		t.Fatalf("bad first candidate starved later active WAL: %+v %v", mixed, err)
	}
	for _, row := range mixed.Databases {
		switch row.Path {
		case filepath.Base(cold):
			if row.Status != DatabaseFailed || !row.CheckedAt.IsZero() || !row.LastVerified.Equal(record.VerifiedAt) {
				t.Fatalf("invalid cold source claimed current: %+v", row)
			}
		case filepath.Base(active):
			if row.Status != DatabaseArchived || !row.LastVerified.Equal(now) {
				t.Fatalf("later active WAL not archived: %+v", row)
			}
		default:
			t.Fatalf("unexpected source: %+v", row)
		}
	}
	rows = records()
	if len(rows) != 2 {
		t.Fatalf("failed snapshot published or previous bundle vanished: %+v", rows)
	}
	retained := false
	for _, row := range rows {
		if row.Label.ID == record.Label.ID {
			retained = sameCatalogRecord(row, record)
		}
	}
	if !retained {
		t.Fatal("prior verified catalog record changed or disappeared")
	}
	if after, err := os.ReadFile(filepath.Join(cfg.Sink, record.BundleKey)); err != nil || !bytes.Equal(after, bundle) {
		t.Fatalf("previous verified bundle changed: %v", err)
	}
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Lstat(cold + suffix); !os.IsNotExist(err) {
			t.Fatalf("source sidecar created: %s %v", suffix, err)
		}
	}
}
