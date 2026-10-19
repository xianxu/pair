package historyarchive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type snapshotPublicationSink struct {
	Sink
	failures []error
	cancel   context.CancelFunc
	calls    int
}

func TestContinuousSnapshotRejectsOverlapBeforeStoreOrSweep(t *testing.T) {
	for _, pointer := range []bool{false, true} {
		t.Run(fmt.Sprintf("pointer=%v", pointer), func(t *testing.T) {
			base, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			home := filepath.Join(base, "home")
			src := filepath.Join(home, ".codex")
			// This is native source data, not collector scratch, despite its name.
			marker := filepath.Join(src, "sqlite-native", "preserve")
			if err := os.MkdirAll(filepath.Dir(marker), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(marker, []byte("native"), 0600); err != nil {
				t.Fatal(err)
			}
			snapshotter := SQLiteSnapshotter{Scratch: src}
			var implementation Snapshotter = snapshotter
			if pointer {
				implementation = &snapshotter
			}
			// Even a filtered pass must not sweep another kind's native root.
			pass := ContinuousPass{Store: filepath.Join(base, "store"), SnapshotRoot: filepath.Join(base, "staging"), Home: home, Host: "fixture", Env: map[string]string{}, Kinds: []string{"claude-transcripts"}, Snapshotter: implementation}
			if _, err := pass.Run(context.Background()); err == nil {
				t.Fatal("overlapping snapshot scratch admitted")
			}
			if raw, err := os.ReadFile(marker); err != nil || string(raw) != "native" {
				t.Fatalf("sweep mutated native data before overlap refusal: %q %v", raw, err)
			}
			if _, err := os.Stat(pass.Store); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("store mutated before overlap refusal: %v", err)
			}
		})
	}
}

func TestContinuousSnapshotRejectsCrossKindStagingOverlap(t *testing.T) {
	for _, alias := range []bool{false, true} {
		t.Run(fmt.Sprintf("symlink=%v", alias), func(t *testing.T) {
			base, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			home, staging := filepath.Join(base, "home"), filepath.Join(base, "snapshots")
			codex := filepath.Join(home, ".codex")
			foreign := filepath.Join(staging, "codex-database")
			if alias {
				foreign = filepath.Join(home, ".claude")
			}
			for _, dir := range []string{codex, foreign, staging} {
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if alias {
				if err := os.Symlink(foreign, filepath.Join(staging, "codex-database")); err != nil {
					t.Fatal(err)
				}
			}
			for path, text := range map[string]string{filepath.Join(codex, "state.sqlite"): "codex source", filepath.Join(foreign, "state.sqlite"): "foreign source"} {
				if err := os.WriteFile(path, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			fake := &fakeSQLiteRunner{}
			cfg := ContinuousConfig{Version: Version, Recipients: []string{strings.Repeat("A", 40)}, Sink: filepath.Join(base, "sink"), Catalog: filepath.Join(base, "catalog", "catalog.jsonl")}
			pass := ContinuousPass{Archiver: Archiver{Codec: &archiveTestCodec{}}, Config: cfg, Store: filepath.Join(base, "store"), SnapshotRoot: staging, Home: home, Host: "fixture", Env: map[string]string{"CLAUDE_CONFIG_DIR": foreign}, Kinds: []string{"codex-database"}, Snapshotter: SQLiteSnapshotter{Scratch: filepath.Join(base, "scratch"), Runner: fake}}
			_, err = pass.Run(context.Background())
			if err == nil || !strings.Contains(err.Error(), "overlap") || fake.reads != 0 {
				t.Errorf("cross-kind overlap admitted: reads=%d err=%v", fake.reads, err)
			}
			for path, text := range map[string]string{filepath.Join(codex, "state.sqlite"): "codex source", filepath.Join(foreign, "state.sqlite"): "foreign source"} {
				if raw, err := os.ReadFile(path); err != nil || string(raw) != text {
					t.Errorf("source changed: %q: %q %v", path, raw, err)
				}
			}
			if _, err := os.Stat(pass.Store); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("store created before refusal: %v", err)
			}
		})
	}
}

func (s *snapshotPublicationSink) Put(ctx context.Context, key string, r io.Reader, limit int64) error {
	s.calls++
	if s.calls <= len(s.failures) {
		if s.cancel != nil {
			s.cancel()
		}
		return s.failures[s.calls-1]
	}
	return s.Sink.Put(ctx, key, r, limit)
}

func TestContinuousSnapshotPublicationFailuresDoNotStarveLaterDatabases(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancelled=%v", cancelled), func(t *testing.T) {
			base, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			home := filepath.Join(base, "home")
			src := filepath.Join(home, ".codex")
			if err := os.MkdirAll(src, 0700); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"a.sqlite", "b.sqlite", "c.sqlite"} {
				if err := os.WriteFile(filepath.Join(src, name), []byte("source"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
			cfg := ContinuousConfig{Version: Version, Recipients: []string{strings.Repeat("A", 40)}, Sink: filepath.Join(base, "sink"), Catalog: filepath.Join(base, "catalog", "catalog.jsonl")}
			first, second := errors.New("first publication failed"), errors.New("second publication failed")
			sink := &snapshotPublicationSink{Sink: LocalSink{Root: cfg.Sink}, failures: []error{first, second}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if cancelled {
				sink.cancel = cancel
			}
			pass := ContinuousPass{Archiver: Archiver{Codec: &archiveTestCodec{}, Sink: sink, Now: func() time.Time { return now }}, Config: cfg, Store: filepath.Join(base, "store"), SnapshotRoot: filepath.Join(base, "staging"), Home: home, Host: "fixture", Env: map[string]string{}, Now: func() time.Time { return now }, Snapshotter: SQLiteSnapshotter{Scratch: filepath.Join(base, "scratch"), Runner: &fakeSQLiteRunner{}}}
			report, err := pass.Run(ctx)
			if cancelled {
				if !errors.Is(err, context.Canceled) || sink.calls != 1 || report.Archived["codex-database"] != 0 {
					t.Fatalf("cancellation must stop publication immediately: calls=%d report=%+v err=%v", sink.calls, report, err)
				}
				for _, row := range report.Databases[1:] {
					if row.Status != DatabaseFailed || !strings.Contains(row.Detail, "cancel") {
						t.Fatalf("unattempted publication must report cancellation: %+v", row)
					}
				}
				return
			}
			if !errors.Is(err, first) || !errors.Is(err, second) || report.Archived["codex-database"] != 1 {
				t.Fatalf("publication failures stranded a later database or lost an error: report=%+v err=%v", report, err)
			}
			if len(report.Databases) != 3 {
				t.Fatalf("missing database summaries: %+v", report.Databases)
			}
			for i, row := range report.Databases {
				if row.CheckedAt.IsZero() {
					t.Fatalf("validated snapshot lost observation: %+v", row)
				}
				if i < 2 {
					if row.Status != DatabaseFailed || !row.LastVerified.IsZero() || !strings.Contains(row.Detail, sink.failures[i].Error()) {
						t.Fatalf("failed publication claimed success: %+v", row)
					}
				} else if row.Status != DatabaseArchived || !row.LastVerified.Equal(now) || row.Detail != "" {
					t.Fatalf("later successful publication not reported: %+v", row)
				}
			}
			if _, err := os.Stat(filepath.Join(pass.Store, LastRunName)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed pass marked complete: %v", err)
			}
		})
	}
}

func TestContinuousSnapshotCountedNAnd2NAndFreshness(t *testing.T) {
	for _, n := range []int{3, 6} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			base, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			home := filepath.Join(base, "home")
			src := filepath.Join(home, ".codex")
			if err := os.MkdirAll(src, 0700); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < n; i++ {
				if err := os.WriteFile(filepath.Join(src, fmt.Sprintf("db_%d.sqlite", i)), []byte("source"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
			cfg := ContinuousConfig{Version: Version, Recipients: []string{strings.Repeat("A", 40)}, Sink: filepath.Join(base, "sink"), Catalog: filepath.Join(base, "catalog", "catalog.jsonl")}
			fake := &fakeSQLiteRunner{}
			pass := ContinuousPass{Archiver: Archiver{Codec: &archiveTestCodec{}, Now: func() time.Time { return now }}, Config: cfg, Store: filepath.Join(base, "store"), SnapshotRoot: filepath.Join(base, "staging"), Home: home, Host: "fixture", Env: map[string]string{}, Kinds: []string{"codex-database"}, Now: func() time.Time { return now }, Snapshotter: SQLiteSnapshotter{Scratch: filepath.Join(base, "scratch"), Runner: fake}}
			first, err := pass.Run(context.Background())
			if err != nil || first.Archived["codex-database"] != n || len(first.Databases) != n || fake.reads != n || fake.validations != n {
				t.Fatalf("cold pass: %+v fake=%+v err=%v", first, fake, err)
			}
			for _, row := range first.Databases {
				if row.Status != DatabaseArchived || !row.LastVerified.Equal(now) || row.CheckedAt.IsZero() || row.Source.Host != pass.Host || row.Source.Repository != src || row.NeedsAttention(now, DefaultDatabaseMaxAge) {
					t.Fatalf("unverified archive projected: %+v", row)
				}
			}
			verified := now
			now = now.Add(72 * time.Hour)
			fake.generation, fake.reads, fake.validations = 0, 0, 0
			second, err := pass.Run(context.Background())
			if err != nil || second.Archived["codex-database"] != 0 || fake.reads != n || fake.validations != n {
				t.Fatalf("warm pass: %+v fake=%+v err=%v", second, fake, err)
			}
			for _, row := range second.Databases {
				if row.Status != DatabaseUnchanged || !row.LastVerified.Equal(verified) || !row.CheckedAt.Equal(now) || row.NeedsAttention(now, DefaultDatabaseMaxAge) {
					t.Fatalf("unchanged confused with stale: %+v", row)
				}
			}
			fake.fail = ErrSQLiteDeadline
			fake.reads, fake.validations = 0, 0
			deferred, err := pass.Run(context.Background())
			if err != nil || deferred.Archived["codex-database"] != 0 || fake.reads != n || fake.validations != 0 {
				t.Fatalf("deferred pass retried or archived staging: %+v fake=%+v err=%v", deferred, fake, err)
			}
			for _, row := range deferred.Databases {
				if row.Status != DatabaseDeferred || !row.LastVerified.Equal(verified) || !row.CheckedAt.IsZero() || !row.NeedsAttention(now, DefaultDatabaseMaxAge) {
					t.Fatalf("unobserved source claimed fresh: %+v", row)
				}
			}
		})
	}
}
