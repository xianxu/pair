package historyarchive_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
)

type fixtureSnapshotter struct {
	calls   int
	scratch string
}

func (f *fixtureSnapshotter) Sweep() error {
	matches, _ := filepath.Glob(filepath.Join(f.scratch, "sqlite-*"))
	for _, m := range matches {
		if err := os.RemoveAll(m); err != nil {
			return err
		}
	}
	return nil
}

func (f *fixtureSnapshotter) Snapshot(_ context.Context, root, rel, staging string) (ha.SnapshotResult, error) {
	f.calls++
	raw, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return ha.SnapshotResult{Status: ha.SnapshotRejected, Detail: err.Error()}, nil
	}
	if err := os.MkdirAll(staging, 0o700); err != nil {
		return ha.SnapshotResult{}, err
	}
	return ha.SnapshotResult{Status: ha.SnapshotReady}, os.WriteFile(filepath.Join(staging, rel), raw, 0o600)
}

func treeState(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			raw, _ := os.ReadFile(p)
			out[p] = info.ModTime().String() + string(raw)
		}
		return nil
	})
	return out
}

func TestContinuousPassArchivesQuietFilesOnceAndNewVersions(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(base, "home")
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	old := now.Add(-100 * time.Hour)
	write := func(rel, body string, mtime time.Time) string {
		p := filepath.Join(home, rel)
		os.MkdirAll(filepath.Dir(p), 0o700)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		os.Chtimes(p, mtime, mtime)
		return p
	}
	write(".claude/projects/p/a.jsonl", "quiet", old)
	fresh := write(".claude/projects/p/b.jsonl", "still writing", now.Add(-time.Hour))
	write(".codex/state_5.sqlite", "db v1", now.Add(-time.Minute))
	cfg := ha.ContinuousConfig{Version: 1, Recipients: []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}, Sink: filepath.Join(base, "sink"), Catalog: filepath.Join(base, "archive", "catalog.jsonl")}
	snap := &fixtureSnapshotter{scratch: filepath.Join(base, "scratch")}
	// A killed pass left a private copy behind: the next pass sweeps it.
	leftover := filepath.Join(snap.scratch, "sqlite-killed")
	os.MkdirAll(leftover, 0o700)
	pass := ha.ContinuousPass{Archiver: ha.Archiver{Codec: testCodec(), Sink: &ha.LocalSink{Root: cfg.Sink}}, Config: cfg,
		Store: filepath.Join(base, "archive"), SnapshotRoot: filepath.Join(base, "archive-snapshots"), Host: "h", Home: home,
		Env: map[string]string{}, Snapshotter: snap, Now: func() time.Time { return now }}
	sources := treeState(t, home)
	r, err := pass.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.Archived["claude-transcripts"] != 1 || r.Archived["codex-database"] != 1 {
		t.Fatalf("first pass %+v, want the quiet transcript and the database snapshot", r.Archived)
	}
	if _, err := os.Stat(leftover); !os.IsNotExist(err) {
		t.Fatalf("the pass did not sweep a leftover private copy: %v", err)
	}
	if again, err := pass.Run(context.Background()); err != nil || sum(again.Archived) != 0 {
		t.Fatalf("second pass archived %+v (%v); unchanged files must not be archived twice", again.Archived, err)
	}
	// The fresh transcript settles; the database changes: both archive once.
	os.Chtimes(fresh, old, old)
	write(".codex/state_5.sqlite", "db v2", now.Add(-time.Minute))
	sources = treeState(t, home)
	third, err := pass.Run(context.Background())
	if err != nil || third.Archived["claude-transcripts"] != 1 || third.Archived["codex-database"] != 1 {
		t.Fatalf("third pass %+v (%v), want the settled file and the new database version", third.Archived, err)
	}
	// A quiet transcript whose content changed is archived again, once.
	write(".claude/projects/p/a.jsonl", "quiet, then appended", old.Add(time.Minute))
	sources = treeState(t, home)
	fourth, err := pass.Run(context.Background())
	if err != nil || fourth.Archived["claude-transcripts"] != 1 {
		t.Fatalf("fourth pass %+v (%v), want the changed quiet transcript as a new version", fourth.Archived, err)
	}
	if after := treeState(t, home); !sameKeys(sources, after) {
		t.Fatalf("a pass changed the sources")
	}
	if _, err := os.Stat(filepath.Join(pass.Store, ha.LastRunName)); err != nil {
		t.Fatalf("no last-run marker: %v", err)
	}
}

func TestContinuousPassLockRefusesAConcurrentPass(t *testing.T) {
	base := t.TempDir()
	store := filepath.Join(base, "archive")
	os.MkdirAll(store, 0o700)
	f, err := os.OpenFile(filepath.Join(store, "continuous.lock"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	holdLock(t, f)
	pass := ha.ContinuousPass{Store: store}
	if _, err := pass.Run(context.Background()); err != ha.ErrPassRunning {
		t.Fatalf("concurrent pass: %v, want ErrPassRunning", err)
	}
}

func sum(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func sameKeys(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	ka, kb := []string{}, []string{}
	for k, v := range a {
		ka = append(ka, k+v)
	}
	for k, v := range b {
		kb = append(kb, k+v)
	}
	sort.Strings(ka)
	sort.Strings(kb)
	for i := range ka {
		if ka[i] != kb[i] {
			return false
		}
	}
	return true
}

type failingSnapshotter struct{}

func (failingSnapshotter) Snapshot(context.Context, string, string, string) (ha.SnapshotResult, error) {
	return ha.SnapshotResult{}, errors.New("snapshot store unavailable")
}
func (failingSnapshotter) Sweep() error { return nil }

// One failing source does not starve the others, and the pass is not marked
// complete.
func TestContinuousPassKeepsGoingPastAFailingSource(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(base, "home")
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	old := now.Add(-100 * time.Hour)
	// The database fails; ops documents come after it in pass order.
	for rel, body := range map[string]string{".claude/projects/p/a.jsonl": "quiet", ".codex/state_5.sqlite": "db", "ops/teams/devinfra/qa/q.md": "question"} {
		p := filepath.Join(home, rel)
		os.MkdirAll(filepath.Dir(p), 0o700)
		os.WriteFile(p, []byte(body), 0o600)
		os.Chtimes(p, old, old)
	}
	cfg := ha.ContinuousConfig{Version: 1, Recipients: []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}, Sink: filepath.Join(base, "sink"), Catalog: filepath.Join(base, "archive", "catalog.jsonl"), OpsRoot: filepath.Join(home, "ops")}
	pass := ha.ContinuousPass{Archiver: ha.Archiver{Codec: testCodec(), Sink: &ha.LocalSink{Root: cfg.Sink}}, Config: cfg,
		Store: filepath.Join(base, "archive"), SnapshotRoot: filepath.Join(base, "archive-snapshots"), Host: "h", Home: home,
		Env: map[string]string{}, Snapshotter: failingSnapshotter{}, Now: func() time.Time { return now }}
	r, err := pass.Run(context.Background())
	if err == nil || r.Archived["claude-transcripts"] != 1 || r.Archived["ops-documents"] != 1 {
		t.Fatalf("pass err=%v archived=%+v, want the failure reported and the transcript archived", err, r.Archived)
	}
	if _, err := os.Stat(filepath.Join(pass.Store, ha.LastRunName)); !os.IsNotExist(err) {
		t.Fatal("a failed pass was marked complete")
	}
}
