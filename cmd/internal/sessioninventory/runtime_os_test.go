package sessioninventory

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

	outside := filepath.Join(t.TempDir(), "outside.jsonl")
	if err := os.WriteFile(outside, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(roots[0].Path, "escape.jsonl")); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.ListFiles(roots[0]); !errors.Is(err, ErrPathEscape) {
		t.Fatalf("symlink listing error = %v, want ErrPathEscape", err)
	}
	if _, err := runtime.ReadFile(Artifact{StorageRoot: roots[0].Name, RelativePath: "escape.jsonl"}, 32); !errors.Is(err, ErrPathEscape) {
		t.Fatalf("symlink read error = %v, want ErrPathEscape", err)
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
}
