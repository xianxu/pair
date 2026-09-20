package sessioninventory

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
)

func TestOSRuntimeBoundaries(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	pairData := filepath.Join(t.TempDir(), "pair-data")
	runtime := NewOSRuntime(home, pairData)
	roots := runtime.NativeRoots(AgentClaude)
	if len(roots) != 1 || roots[0].Name != "claude-projects" || roots[0].Path != filepath.Join(home, ".claude", "projects") {
		t.Fatalf("claude roots = %#v", roots)
	}
	if got := runtime.PairDataRoot(); got.Name != "pair-data" || got.Path != pairData {
		t.Fatalf("pair data root = %#v", got)
	}

	if err := os.MkdirAll(roots[0].Path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(roots[0].Path, "root.jsonl"), []byte("root"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := runtime.ListFiles(roots[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Artifact != (Artifact{StorageRoot: roots[0].Name, RelativePath: "root.jsonl"}) || files[0].Size != 4 || files[0].ModTime == nil {
		t.Fatalf("files = %#v", files)
	}
	if got, err := runtime.ReadFile(files[0].Artifact, 4); err != nil || string(got) != "root" {
		t.Fatalf("read = %q, %v", got, err)
	}
	if _, err := runtime.ReadFile(files[0].Artifact, 3); !errors.Is(err, ErrReadLimit) {
		t.Fatalf("bounded read error = %v, want ErrReadLimit", err)
	}
	if got, err := runtime.ReadFile(files[0].Artifact, -1); err != nil || string(got) != "root" {
		t.Fatalf("unlimited read = %q, %v", got, err)
	}
	for _, limit := range []int64{-2, 0} {
		if _, err := runtime.ReadFile(files[0].Artifact, limit); !errors.Is(err, ErrReadLimit) {
			t.Fatalf("limit %d: %v", limit, err)
		}
	}
	if got, eof, err := runtime.ReadAt(files[0].Artifact, 1, 2); err != nil || eof || string(got) != "oo" {
		t.Fatalf("range read = %q, eof=%v, err=%v", got, eof, err)
	}
	if got, eof, err := runtime.ReadAt(files[0].Artifact, 3, 2); err != nil || !eof || string(got) != "t" {
		t.Fatalf("final range read = %q, eof=%v, err=%v", got, eof, err)
	}

	outside := filepath.Join(t.TempDir(), "outside.jsonl")
	if err := os.WriteFile(outside, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(roots[0].Path, "escape.jsonl")); err != nil {
		t.Fatal(err)
	}
	partial, err := runtime.ListFiles(roots[0])
	if !errors.Is(err, ErrPathEscape) {
		t.Fatalf("symlink listing error = %v, want ErrPathEscape", err)
	}
	if len(partial) != 1 || partial[0].Artifact.RelativePath != "root.jsonl" {
		t.Fatalf("partial listing = %#v, want valid regular files preserved", partial)
	}
	if _, err := runtime.ReadFile(Artifact{StorageRoot: roots[0].Name, RelativePath: "escape.jsonl"}, 32); !errors.Is(err, ErrPathEscape) {
		t.Fatalf("symlink read error = %v, want ErrPathEscape", err)
	}
	if _, err := runtime.ReadFile(Artifact{StorageRoot: roots[0].Name, RelativePath: "escape.jsonl"}, -1); !errors.Is(err, ErrPathEscape) {
		t.Fatalf("unlimited escaped read: %v", err)
	}
	if err := syscall.Mkfifo(filepath.Join(roots[0].Path, "blocking.jsonl"), 0o600); err != nil {
		t.Fatal(err)
	}
	partial, err = runtime.ListFiles(roots[0])
	if !errors.Is(err, ErrPathEscape) || len(partial) != 1 || partial[0].Artifact.RelativePath != "root.jsonl" {
		t.Fatalf("special-file listing = %#v, %v, want valid file plus rejection", partial, err)
	}
}

func TestOSRuntimeListFilesFingerprint(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	runtimeIO := NewOSRuntime(home, t.TempDir())
	root := runtimeIO.NativeRoots(AgentClaude)[0]
	if err := os.MkdirAll(root.Path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root.Path, "root.jsonl"), []byte("root\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	files, err := runtimeIO.ListFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("files = %#v", files)
	}
	entry := files[0]
	if entry.StableFileID == "" || entry.MutationToken == "" {
		t.Fatalf("fingerprint = %#v, want stable identity and mutation token", entry)
	}
	if runtime.GOOS == "darwin" && entry.GenerationToken == "" {
		t.Skip("filesystem does not expose a nonzero Darwin file generation")
	}
	if runtime.GOOS == "linux" && entry.GenerationToken != "" {
		t.Fatalf("Linux generation = %q, want unavailable (statx birth time is not a generation)", entry.GenerationToken)
	}
}

func TestOSRuntimeSQLiteReadOnlyAdapter(t *testing.T) {
	t.Parallel()

	sqlite, err := exec.LookPath("sqlite3")
	if err != nil {
		t.Skip("sqlite3 is not installed")
	}
	home := t.TempDir()
	runtime := NewOSRuntime(home, t.TempDir())
	root := runtime.NativeRoots(AgentAgy)[0]
	if err := os.MkdirAll(root.Path, 0o755); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(root.Path, "conversation.db")
	if output, err := exec.Command(sqlite, databasePath, "create table trajectory(id text, parent text); insert into trajectory values ('root', null), ('child', 'root');").CombinedOutput(); err != nil {
		t.Fatalf("create sqlite fixture: %v: %s", err, output)
	}
	database := Artifact{StorageRoot: root.Name, RelativePath: "conversation.db"}
	got, err := runtime.QuerySQLite(database, "select id, coalesce(parent, '') as parent from trajectory order by id", 4096)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Columns, []string{"id", "parent"}) || len(got.Rows) != 2 || !slices.Equal(got.Rows[0], []string{"child", "root"}) || !slices.Equal(got.Rows[1], []string{"root", ""}) {
		t.Fatalf("sqlite result = %#v", got)
	}
	if _, err := runtime.QuerySQLite(database, "create table forbidden(value text)", 4096); err == nil {
		t.Fatal("read-only adapter accepted a write query")
	}
	const largeQuery = "select hex(zeroblob(600000)) as payload"
	if _, err := runtime.QuerySQLite(database, largeQuery, 1<<20); !errors.Is(err, ErrReadLimit) {
		t.Fatalf("bounded query: %v", err)
	}
	large, err := runtime.QuerySQLite(database, largeQuery, -1)
	if err != nil || len(large.Rows) != 1 || len(large.Rows[0][0]) != 1200000 {
		t.Fatalf("unlimited query: rows=%d err=%v", len(large.Rows), err)
	}
	if _, err := runtime.QuerySQLite(database, "create table still_forbidden(value text)", -1); err == nil {
		t.Fatal("unlimited query allowed write")
	}
	if _, err := runtime.QuerySQLite(database, largeQuery, -2); !errors.Is(err, ErrReadLimit) {
		t.Fatalf("invalid query limit: %v", err)
	}
	if _, err := runtime.QuerySQLite(database, "select "+strings.Repeat("missing_column", 1000), -1); !errors.Is(err, ErrReadLimit) {
		t.Fatalf("unlimited stdout must retain bounded error diagnostics: %v", err)
	}
}
