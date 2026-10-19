package archivecmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		return historyarchive.ContinuousReport{Archived: map[string]int{"claude-transcripts": 2}, Drive: &historyarchive.DriveReport{Verified: 1, Pending: 2, DownloadedChecksums: 1, Notes: []string{"fixture: bundle=backend-md5 label=downloaded-sha256"}}}, nil
	}
	out.Reset()
	if code := RunContext(context.Background(), []string{"continuous", "--kind", "claude-transcripts"}, &out, &errOut, nil); code != 0 || !strings.Contains(out.String(), "claude-transcripts: 2 file(s) archived") {
		t.Fatalf("configured pass: code=%d out=%q", code, out.String())
	}
	if !strings.Contains(out.String(), "Drive: 1 verified") || !strings.Contains(out.String(), "2 pending") || !strings.Contains(out.String(), "downloaded-sha256") {
		t.Fatalf("Drive result omitted: %s", out.String())
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
