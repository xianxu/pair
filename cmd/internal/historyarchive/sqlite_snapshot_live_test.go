package historyarchive

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type measuredSQLiteRunner struct {
	t    *testing.T
	real ExecSQLiteRunner
}

func (r measuredSQLiteRunner) Run(ctx context.Context, req SQLiteRequest) (SQLiteRunResult, error) {
	start := time.Now()
	out, err := r.real.Run(ctx, req)
	if req.Purpose == SQLiteAcquire {
		var size int64
		if info, statErr := os.Stat(filepath.Join(req.Work, "snapshot.sqlite")); statErr == nil {
			size = info.Size()
		}
		r.t.Logf("source=%s acquisition=%s private_bytes=%d error=%v", filepath.Base(req.Database), time.Since(start), size, err)
	}
	return out, err
}

// TestSQLiteSnapshotLiveContinuous is explicitly opt-in: it writes verified
// encrypted bundles to the existing configured local sink/catalog, not Drive.
// The hub coordinates this acceptance; the normal suite never opens native DBs.
func TestSQLiteSnapshotLiveContinuous(t *testing.T) {
	if os.Getenv("PAIR_SQLITE_LIVE_ACCEPTANCE") != "1" {
		t.Skip("hub-coordinated native SQLite acceptance only")
	}
	launcher := os.Getenv("PAIR_SQLITE_LIVE_LAUNCHER")
	if !filepath.IsAbs(launcher) {
		t.Fatal("PAIR_SQLITE_LIVE_LAUNCHER must name the tested pair-go binary")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{}
	for _, key := range []string{"PAIR_DATA_DIR", "XDG_DATA_HOME", "CODEX_HOME", "CLAUDE_CONFIG_DIR", "QODER_HOME", "COUCH_STORE_DIR"} {
		env[key] = os.Getenv(key)
	}
	inv, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	data, err := inv.ResolveRoot("couch-sessions", "", home, env)
	if err != nil {
		t.Fatal(err)
	}
	store, snapshots := filepath.Join(data, "archive"), filepath.Join(data, "archive-snapshots")
	cfg, err := LoadContinuousConfig(filepath.Join(store, ContinuousConfigName))
	if err != nil || cfg == nil {
		t.Fatalf("existing continuous config: %v", err)
	}
	host, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pass := ContinuousPass{Archiver: Archiver{Codec: CommandCodec{}, Sink: LocalSink{Root: cfg.Sink}}, Config: *cfg, Store: store, SnapshotRoot: snapshots, Host: host, Home: home, Env: env, Kinds: []string{"codex-database"},
		Snapshotter: SQLiteSnapshotter{Scratch: filepath.Join(snapshots, ".scratch"), Runner: measuredSQLiteRunner{t: t, real: ExecSQLiteRunner{SQLite: "sqlite3", Launcher: launcher}}}}
	report, err := pass.Run(ctx)
	seen := map[string]DatabaseBackupStatus{}
	for _, row := range report.Databases {
		t.Log(row.Line(time.Now()))
		seen[row.Path] = row.Status
	}
	if err != nil {
		t.Fatalf("local continuous pass: %v", err)
	}
	for _, name := range []string{"logs_2.sqlite", "state_5.sqlite", "thread_history_1.sqlite"} {
		if seen[name] != DatabaseArchived && seen[name] != DatabaseUnchanged {
			t.Errorf("%s did not produce a coherent verified backup: %q; no automatic retry", name, seen[name])
		}
	}
}
