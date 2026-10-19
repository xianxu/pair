package historyarchive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalLabelsRecoverWithoutCatalogAndNeverCreateIt(t *testing.T) {
	req, codec := archiveFixture(t)
	records, err := (Archiver{Codec: codec}).Create(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	cfg := ContinuousConfig{Sink: req.Sink, Catalog: req.Catalog}
	at := time.Now().UTC()
	r, err := ObserveLocalLabels(context.Background(), cfg, at)
	if err != nil || len(r.Observations) != 1 {
		t.Fatal(r, err)
	}
	if err := os.Remove(req.Catalog); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(req.Catalog + ".lock"); err != nil {
		t.Fatal(err)
	}
	r, err = ObserveLocalLabels(context.Background(), cfg, at)
	if err != nil || len(r.Observations) != 1 || r.Observations[0].Label.ID != records[0].Label.ID {
		t.Fatal(r, err)
	}
	if _, err := os.Stat(req.Catalog); !os.IsNotExist(err) {
		t.Fatal("catalog recreated", err)
	}
	if _, err := os.Stat(req.Catalog + ".lock"); !os.IsNotExist(err) {
		t.Fatal("catalog lock recreated", err)
	}
	if r.Observations[0].Location.Evidence != "observed" || r.Observations[0].Location.Proof.Method != "" {
		t.Fatal("label-only recovery invented verification", r)
	}
}

func TestLocalLabelsCatalogKeepsAbsentCopyDiscoverable(t *testing.T) {
	req, codec := archiveFixture(t)
	records, err := (Archiver{Codec: codec}).Create(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(req.Sink, records[0].BundleKey)); err != nil {
		t.Fatal(err)
	}
	r, err := ObserveLocalLabels(context.Background(), ContinuousConfig{Sink: req.Sink, Catalog: req.Catalog}, time.Now().UTC())
	if err != nil || len(r.Observations) != 1 {
		t.Fatal(r, err)
	}
	i, err := ApplyIndexRefresh(LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, r)
	if err != nil {
		t.Fatal(err)
	}
	for _, loc := range i.Records[records[0].Label.ID].Locations {
		if loc.Available || loc.Proof.Method != "local-catalog" {
			t.Fatal("absence erased metadata or claimed availability", loc)
		}
	}
}

func TestLocalLabelsEmptyStoreDoesNotCreatePaths(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := ContinuousConfig{Sink: filepath.Join(base, "missing-sink"), Catalog: filepath.Join(base, "missing-catalog", "history.jsonl")}
	r, err := ObserveLocalLabels(context.Background(), cfg, time.Now().UTC())
	if err != nil || len(r.Observations) != 0 {
		t.Fatal(r, err)
	}
	entries, err := os.ReadDir(base)
	if err != nil || len(entries) != 0 {
		t.Fatal("read changed empty store", entries, err)
	}
}

func TestLocalLabelObservationScansRemainLinearAtNAnd2N(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			cfg := ContinuousConfig{Catalog: catalogPath(t)}
			cfg.Sink = filepath.Join(filepath.Dir(cfg.Catalog), "sink")
			var catalog bytes.Buffer
			for k := 0; k < n; k++ {
				record := catalogFixture(t, fmt.Sprintf("row-%d", k))
				record.Sink = cfg.Sink
				raw, err := json.Marshal(record)
				if err != nil {
					t.Fatal(err)
				}
				catalog.Write(raw)
				catalog.WriteByte('\n')
				label, err := json.Marshal(record.Label)
				if err != nil {
					t.Fatal(err)
				}
				labelPath := filepath.Join(cfg.Sink, record.LabelKey)
				if err := os.MkdirAll(filepath.Dir(labelPath), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(labelPath, label, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(cfg.Sink, record.BundleKey), bytes.Repeat([]byte{'x'}, int(record.Label.Bytes)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(cfg.Catalog, catalog.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(cfg.Catalog+".lock", nil, 0600); err != nil {
				t.Fatal(err)
			}
			refresh, report, err := observeLocalLabels(context.Background(), cfg, time.Now().UTC())
			if err != nil || len(refresh.Observations) != n {
				t.Fatalf("observe: count=%d err=%v", len(refresh.Observations), err)
			}
			// One validation pass at open and one streaming projection. Looking up
			// each record separately would increase scans and visits quadratically.
			if report.Catalog.ShardScans != 2 || report.Catalog.RecordVisits != 2*n || report.Catalog.DirectoryScans != 1 || report.LabelReads != n || report.DirectoryScans != 0 {
				t.Fatalf("catalog-backed inventory rescanned: %+v", report)
			}
			if err := os.Remove(cfg.Catalog); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(cfg.Catalog + ".lock"); err != nil {
				t.Fatal(err)
			}
			refresh, report, err = observeLocalLabels(context.Background(), cfg, time.Now().UTC())
			if err != nil || len(refresh.Observations) != n {
				t.Fatalf("fallback: count=%d err=%v", len(refresh.Observations), err)
			}
			// Fallback enumerates kind/year/month once each. Discovery and identity
			// matching each read one label; neither enumerates directories per record.
			if report.Catalog.ShardScans != 0 || report.Catalog.RecordVisits != 0 || report.Catalog.DirectoryScans != 0 || report.LabelReads != 2*n || report.DirectoryScans != 3 {
				t.Fatalf("fallback inventory rescanned: %+v", report)
			}
		})
	}
}
