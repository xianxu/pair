package historyarchive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
				if row.Status != DatabaseArchived || !row.LastVerified.Equal(now) || row.CheckedAt.IsZero() {
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
				if row.Status != DatabaseUnchanged || !row.LastVerified.Equal(verified) || !row.CheckedAt.Equal(now) {
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
				if row.Status != DatabaseDeferred || !row.LastVerified.Equal(verified) || !row.CheckedAt.IsZero() {
					t.Fatalf("unobserved source claimed fresh: %+v", row)
				}
			}
		})
	}
}
