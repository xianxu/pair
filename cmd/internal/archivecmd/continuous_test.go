package archivecmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
)

func TestContinuousCLIUnconfiguredBadArgsAndRunning(t *testing.T) {
	data := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", data)
	var out, errOut bytes.Buffer
	if code := RunContext(context.Background(), []string{"continuous"}, &out, &errOut, nil); code != 0 || !strings.Contains(out.String(), "not configured") {
		t.Fatalf("unconfigured: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
	if code := RunContext(context.Background(), []string{"continuous", "--bogus"}, &out, &errOut, nil); code != 1 {
		t.Fatalf("bad args: code=%d", code)
	}
	store := filepath.Join(data, "archive")
	os.MkdirAll(store, 0o700)
	cfg := `{"version":1,"recipients":["` + strings.Repeat("A", 40) + `"],"sink":"` + filepath.Join(data, "sink") + `","catalog":"` + filepath.Join(store, "catalog.jsonl") + `"}`
	if err := os.WriteFile(filepath.Join(store, historyarchive.ContinuousConfigName), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	saved := ContinuousPassFunc
	defer func() { ContinuousPassFunc = saved }()
	var got historyarchive.ContinuousPass
	ContinuousPassFunc = func(_ context.Context, p historyarchive.ContinuousPass) (historyarchive.ContinuousReport, error) {
		got = p
		return historyarchive.ContinuousReport{Archived: map[string]int{"claude-transcripts": 2}, Databases: []historyarchive.DatabaseBackupSummary{{Kind: "codex-database", Path: "state_5.sqlite", Status: historyarchive.DatabaseDeferred, Detail: "deadline"}}, Drive: &historyarchive.DriveReport{Verified: 1, Pending: 2, DownloadedChecksums: 1, Notes: []string{"fixture: bundle=backend-md5 label=downloaded-sha256"}}}, nil
	}
	out.Reset()
	if code := RunContext(context.Background(), []string{"continuous", "--kind", "claude-transcripts"}, &out, &errOut, nil); code != 0 || !strings.Contains(out.String(), "claude-transcripts: 2 file(s) archived") {
		t.Fatalf("configured pass: code=%d out=%q", code, out.String())
	}
	if !strings.Contains(out.String(), "Drive: 1 verified") || !strings.Contains(out.String(), "2 pending") || !strings.Contains(out.String(), "downloaded-sha256") {
		t.Fatalf("Drive result omitted: %s", out.String())
	}
	if !strings.Contains(out.String(), "state_5.sqlite: deferred; last verified=never; deadline") {
		t.Fatalf("database freshness omitted: %s", out.String())
	}
	if got.Drive == nil || got.Store != store || got.SnapshotRoot != filepath.Join(data, "archive-snapshots") || len(got.Kinds) != 1 {
		t.Fatalf("pass wiring %+v", got)
	}
	ContinuousPassFunc = func(context.Context, historyarchive.ContinuousPass) (historyarchive.ContinuousReport, error) {
		return historyarchive.ContinuousReport{}, historyarchive.ErrPassRunning
	}
	out.Reset()
	if code := RunContext(context.Background(), []string{"continuous"}, &out, &errOut, nil); code != 0 || !strings.Contains(out.String(), "already running") {
		t.Fatalf("running: code=%d out=%q", code, out.String())
	}
}

func TestContinuousCLIJSONUnconfigured(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	var out, diag bytes.Buffer
	code := RunContext(context.Background(), []string{"continuous", "--json"}, &out, &diag, nil)
	if code != 0 || !json.Valid(out.Bytes()) || !bytes.Contains(out.Bytes(), []byte(`"not-configured"`)) {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, out.String(), diag.String())
	}
}
func TestContinuousCLIInvalidFreshnessFailsBeforeWork(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	t.Setenv("PAIR_ARCHIVE_DATABASE_MAX_AGE", "-1h")
	var out, diag bytes.Buffer
	if code := RunContext(context.Background(), []string{"continuous"}, &out, &diag, nil); code != 1 || !strings.Contains(diag.String(), "PAIR_ARCHIVE_DATABASE_MAX_AGE") {
		t.Fatalf("stdout=%s stderr=%s", out.String(), diag.String())
	}
}

func TestContinuousCLIJSONPreservesPartialFailure(t *testing.T) {
	data := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", data)
	t.Setenv("PAIR_ARCHIVE_DATABASE_MAX_AGE", "72h")
	store := filepath.Join(data, "archive")
	os.MkdirAll(store, 0700)
	cfg := `{"version":1,"recipients":["` + strings.Repeat("A", 40) + `"],"sink":"` + filepath.Join(data, "sink") + `","catalog":"` + filepath.Join(store, "catalog.jsonl") + `"}`
	os.WriteFile(filepath.Join(store, historyarchive.ContinuousConfigName), []byte(cfg), 0600)
	saved := ContinuousPassFunc
	t.Cleanup(func() { ContinuousPassFunc = saved })
	ContinuousPassFunc = func(_ context.Context, p historyarchive.ContinuousPass) (historyarchive.ContinuousReport, error) {
		if len(p.Kinds) != 1 || p.Kinds[0] != "codex-database" {
			t.Fatalf("kinds=%v", p.Kinds)
		}
		return historyarchive.ContinuousReport{Databases: []historyarchive.DatabaseBackupSummary{{Source: historyarchive.SourceIdentity{Host: "fixture", Root: "/fixture"}, Kind: "codex-database", Path: "never.sqlite", Status: historyarchive.DatabaseFailed, Detail: "fixture failure"}}}, errors.New("fixture failure")
	}
	var out, diag bytes.Buffer
	if code := RunContext(context.Background(), []string{"continuous", "--json", "--kind", "codex-database"}, &out, &diag, nil); code != 1 {
		t.Fatalf("code=%d", code)
	}
	report, err := historyarchive.DecodeContinuousReport(out.Bytes())
	if err != nil || report.Outcome != historyarchive.ContinuousFailed || report.MaxDatabaseAge != 72*time.Hour || len(report.Report.Databases) != 1 {
		t.Fatalf("report=%+v err=%v stdout=%s", report, err, out.String())
	}
	if !strings.Contains(diag.String(), "fixture failure") || bytes.Contains(out.Bytes(), []byte("pair archive:")) {
		t.Fatalf("stdout=%s stderr=%s", out.String(), diag.String())
	}
}
