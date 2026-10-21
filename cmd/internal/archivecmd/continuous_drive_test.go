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
)

func TestContinuousCLIInvalidDriveThresholdFailsBeforeWork(t *testing.T) {
	for _, raw := range []string{"0", "-1", "oops", "999999999999999999999999999999"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("PAIR_DATA_DIR", t.TempDir())
			t.Setenv("PAIR_ARCHIVE_DRIVE_MAX_FILES", raw)
			store := filepath.Join(os.Getenv("PAIR_DATA_DIR"), "archive")
			if err := os.MkdirAll(store, 0700); err != nil {
				t.Fatal(err)
			}
			cfg := `{"version":1,"recipients":["` + strings.Repeat("A", 40) + `"],"sink":"` + filepath.Join(store, "sink") + `","catalog":"` + filepath.Join(store, "catalog.jsonl") + `"}`
			if err := os.WriteFile(filepath.Join(store, historyarchive.ContinuousConfigName), []byte(cfg), 0600); err != nil {
				t.Fatal(err)
			}
			saved, calls := ContinuousPassFunc, 0
			t.Cleanup(func() { ContinuousPassFunc = saved })
			ContinuousPassFunc = func(context.Context, historyarchive.ContinuousPass) (historyarchive.ContinuousReport, error) {
				calls++
				return historyarchive.ContinuousReport{}, nil
			}
			var out, diag bytes.Buffer
			if code := RunContext(context.Background(), []string{"continuous", "--json"}, &out, &diag, nil); code != 1 || out.Len() != 0 || !strings.Contains(diag.String(), "PAIR_ARCHIVE_DRIVE_MAX_FILES") {
				t.Fatalf("code=%d out=%s diag=%s", code, &out, &diag)
			}
			if calls != 0 {
				t.Fatalf("invalid threshold performed %d passes", calls)
			}
		})
	}
}

func TestContinuousCLIDriveCountProjection(t *testing.T) {
	data := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", data)
	t.Setenv("PAIR_ARCHIVE_DRIVE_MAX_FILES", "7")
	store := filepath.Join(data, "archive")
	if err := os.MkdirAll(store, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := `{"version":1,"recipients":["` + strings.Repeat("A", 40) + `"],"sink":"` + filepath.Join(data, "sink") + `","catalog":"` + filepath.Join(store, "catalog.jsonl") + `"}`
	if err := os.WriteFile(filepath.Join(store, historyarchive.ContinuousConfigName), []byte(cfg), 0600); err != nil {
		t.Fatal(err)
	}
	saved := ContinuousPassFunc
	t.Cleanup(func() { ContinuousPassFunc = saved })
	observation := &historyarchive.DriveInventoryObservation{Destination: "pair-backup:backups/pair-history", FolderID: "folder-1", Phase: "after", Files: 8}
	var observations []historyarchive.DriveInventoryObservation
	ContinuousPassFunc = func(context.Context, historyarchive.ContinuousPass) (historyarchive.ContinuousReport, error) {
		return historyarchive.ContinuousReport{Drive: &historyarchive.DriveReport{Inventory: observation, Inventories: observations}}, nil
	}
	var out, diag bytes.Buffer
	if code := RunContext(context.Background(), []string{"continuous", "--json"}, &out, &diag, nil); code != 0 {
		t.Fatalf("code=%d diag=%s", code, &diag)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if string(envelope["max_drive_files"]) != "7" {
		t.Fatalf("threshold missing: %s", &out)
	}
	out.Reset()
	if code := RunContext(context.Background(), []string{"continuous"}, &out, &diag, nil); code != 0 || !strings.Contains(out.String(), observation.Line()) || !strings.Contains(out.String(), observation.Warning(7)) {
		t.Fatalf("code=%d out=%s diag=%s", code, &out, &diag)
	}
	before := *observation
	before.Phase = "before"
	before.Files = 99
	leaf := *observation
	leaf.Destination += "/2026/10"
	leaf.FolderID = "month-id"
	observations = []historyarchive.DriveInventoryObservation{*observation, before, leaf, leaf}
	out.Reset()
	if code := RunContext(context.Background(), []string{"continuous"}, &out, &diag, nil); code != 0 || strings.Count(out.String(), leaf.Line()) != 2 || strings.Contains(out.String(), before.Line()) || strings.Contains(out.String(), "total") {
		t.Fatalf("folder projection: code=%d out=%s diag=%s", code, &out, &diag)
	}
	observations = nil
	observation = nil
	out.Reset()
	if code := RunContext(context.Background(), []string{"continuous"}, &out, &diag, nil); code != 0 || !strings.Contains(out.String(), observation.Line()) || strings.Contains(out.String(), "threshold") {
		t.Fatalf("unknown: code=%d out=%s diag=%s", code, &out, &diag)
	}
}

func TestContinuousCLIUnconfiguredDefaultDriveThreshold(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	t.Setenv("PAIR_ARCHIVE_DRIVE_MAX_FILES", "")
	var out, diag bytes.Buffer
	if code := RunContext(context.Background(), []string{"continuous", "--json"}, &out, &diag, nil); code != 0 {
		t.Fatalf("code=%d diag=%s", code, &diag)
	}
	var r map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if string(r["max_drive_files"]) != "20000" {
		t.Fatalf("default missing: %s", &out)
	}
}
