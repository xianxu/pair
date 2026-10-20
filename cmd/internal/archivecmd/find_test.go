package archivecmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

func TestFindCLIValidatesBeforeReadingConfiguration(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	for _, args := range [][]string{{"find", "--bogus"}, {"find", "--since", "yesterday"}, {"find", "--since", "2026-01-02", "--until", "2026-01-01"}, {"find", "--kind", "unsupported"}} {
		var out, diag bytes.Buffer
		if code := RunContext(context.Background(), args, &out, &diag, nil); code != 1 || diag.Len() == 0 {
			t.Fatalf("%v: code%d out%q err%q", args, code, out.String(), diag.String())
		}
	}
	var out, diag bytes.Buffer
	if code := RunContext(context.Background(), []string{"find", "--help"}, &out, &diag, nil); code != 0 || !strings.Contains(out.String(), "--since") {
		t.Fatalf("help: %d %q %q", code, out.String(), diag.String())
	}
}

func TestFindCLIExistingBundleAndLocalSync(t *testing.T) {
	data, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_DATA_DIR", data)
	store, source := filepath.Join(data, "archive"), filepath.Join(data, "source")
	for _, dir := range []string{store, source} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "private-transcript"), []byte("private fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := historyarchive.ContinuousConfig{Version: 1, Recipients: []string{strings.Repeat("A", 40)}, Sink: filepath.Join(data, "sink"), Catalog: filepath.Join(store, "history.jsonl")}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store, "continuous.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	req := historyarchive.ArchiveRequest{Version: 1, Kind: "ops-documents", Source: historyarchive.SourceIdentity{Host: "fixture", Root: source}, Files: []string{"private-transcript"}, Recipients: cfg.Recipients, Sink: cfg.Sink, Catalog: cfg.Catalog}
	records, err := (historyarchive.Archiver{Codec: &archivetest.TestCodec{KnownRecipients: map[string]bool{cfg.Recipients[0]: true}}}).Create(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	var out, diag bytes.Buffer
	if code := RunContext(context.Background(), []string{"find"}, &out, &diag, nil); code != 0 {
		t.Fatalf("find: %d %q", code, diag.String())
	}
	var found historyarchive.LabelIndexRecord
	if err := json.Unmarshal(out.Bytes(), &found); err != nil {
		t.Fatal(err)
	}
	if found.Label.ID != records[0].Label.ID || len(found.Locations) != 1 || strings.Contains(out.String(), "private-transcript") || strings.Contains(out.String(), source) {
		t.Fatal("missing label/location or private manifest exposed", out.String())
	}
	if _, err := os.Stat(cfg.Catalog + ".index.json"); !os.IsNotExist(err) {
		t.Fatal("find published index", err)
	}
	out.Reset()
	diag.Reset()
	if code := RunContext(context.Background(), []string{"sync-index"}, &out, &diag, nil); code != 0 || !strings.Contains(out.String(), "1 label(s)") {
		t.Fatalf("sync: %d %q %q", code, out.String(), diag.String())
	}
	if _, err := historyarchive.ReadLabelIndex(cfg.Catalog + ".index.json"); err != nil {
		t.Fatal(err)
	}
}

func TestFindCLIEmptyConfiguredStorePerformsNoWrites(t *testing.T) {
	data, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_DATA_DIR", data)
	store := filepath.Join(data, "archive")
	if err := os.Mkdir(store, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := `{"version":1,"recipients":["` + strings.Repeat("A", 40) + `"],"sink":"` + filepath.Join(data, "sink") + `","catalog":"` + filepath.Join(store, "history.jsonl") + `"}`
	if err := os.WriteFile(filepath.Join(store, "continuous.json"), []byte(cfg), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diag bytes.Buffer
	if code := RunContext(context.Background(), []string{"find"}, &out, &diag, nil); code != 0 || out.Len() != 0 {
		t.Fatalf("find: %d %q %q", code, out.String(), diag.String())
	}
	entries, err := os.ReadDir(store)
	if err != nil || len(entries) != 1 {
		t.Fatal("find wrote archive metadata", entries, err)
	}
	if _, err := os.Stat(filepath.Join(data, "sink")); !os.IsNotExist(err) {
		t.Fatal("find created sink", err)
	}
}

func TestSyncIndexCLIRejectsArgumentsAndHelpsWithoutConfiguration(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	for _, args := range [][]string{{"sync-index", "--from", "other"}, {"sync-index", "--bogus"}} {
		var out, diag bytes.Buffer
		if code := RunContext(context.Background(), args, &out, &diag, nil); code != 1 {
			t.Fatal(args, code)
		}
	}
	var out, diag bytes.Buffer
	if code := RunContext(context.Background(), []string{"sync-index", "--help"}, &out, &diag, nil); code != 0 || !strings.Contains(out.String(), "--from") {
		t.Fatalf("help: %d %q %q", code, out.String(), diag.String())
	}
}
