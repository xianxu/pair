package historyarchive

import (
	"context"
	"golang.org/x/sys/unix"
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
	walBytes := func() int64 {
		if info, err := os.Stat(req.Database + "-wal"); err == nil {
			return info.Size()
		}
		return 0
	}
	beforeWAL := walBytes()
	start := time.Now()
	out, err := r.real.Run(ctx, req)
	if req.Purpose == SQLiteAcquire {
		var size int64
		if info, statErr := os.Stat(filepath.Join(req.Work, "snapshot.sqlite")); statErr == nil {
			size = info.Size()
		}
		afterWAL := walBytes()
		r.t.Logf("source=%s budget_seconds=%d acquisition=%s private_bytes=%d wal_before=%d wal_after=%d wal_delta=%d error=%v", filepath.Base(req.Database), req.ReadSeconds, time.Since(start), size, beforeWAL, afterWAL, afterWAL-beforeWAL, err)
	}
	return out, err
}

type liveArchiveConfig struct {
	home, host, store, snapshots string
	env                          map[string]string
	cfg                          *ContinuousConfig
	inv                          SourceInventory
}

func loadLiveArchiveConfig(t *testing.T) liveArchiveConfig {
	t.Helper()
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
	return liveArchiveConfig{home: home, host: host, store: store, snapshots: snapshots, env: env, cfg: cfg, inv: inv}
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
	settings := loadLiveArchiveConfig(t)
	home, host, store, snapshots, env, cfg := settings.home, settings.host, settings.store, settings.snapshots, settings.env, settings.cfg
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

// TestSQLiteSnapshotPublishStagedHistory publishes only the already validated
// private stage. It never discovers/opens a native database or invokes SQLite.
func TestSQLiteSnapshotPublishStagedHistory(t *testing.T) {
	if os.Getenv("PAIR_SQLITE_PUBLISH_STAGED") != "1" {
		t.Skip("hub-coordinated staged publication only")
	}
	settings := loadLiveArchiveConfig(t)
	lock, err := os.OpenFile(filepath.Join(settings.store, "continuous.lock"), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(settings.snapshots, "codex-database")
	nativeRoot, err := settings.inv.ResolveRoot("codex-database", "", settings.home, settings.env)
	if err != nil {
		t.Fatal(err)
	}
	req := ArchiveRequest{Version: Version, Kind: "codex-database", Source: SourceIdentity{Host: settings.host, Root: root, Repository: nativeRoot}, Files: []string{"thread_history_1.sqlite"}, Recipients: settings.cfg.Recipients, Sink: settings.cfg.Sink, Catalog: settings.cfg.Catalog}
	info, err := os.Lstat(filepath.Join(root, req.Files[0]))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("validated stage unavailable: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	codec := CommandCodec{Zstd: os.Getenv("PAIR_STAGE_ZSTD"), GPG: os.Getenv("PAIR_STAGE_GPG")}
	records, err := (Archiver{Codec: codec, Sink: LocalSink{Root: req.Sink}}).create(ctx, req, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("staged publication returned %d records", len(records))
	}
	record := records[0]
	t.Logf("verified=%s raw_bytes=%d encrypted_bytes=%d bundle=%s", record.VerifiedAt.UTC().Format(time.RFC3339), info.Size(), record.Label.Bytes, record.Label.ID)
}
