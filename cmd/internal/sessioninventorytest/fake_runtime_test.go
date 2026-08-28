package sessioninventorytest

import (
	"errors"
	"slices"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestFakeRuntimeModelsPersistentStorageAndFailures(t *testing.T) {
	t.Parallel()

	runtime := NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentAgy, Name: "agy", Path: "/native/agy"}
	first := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "first.db"}, Size: 3}
	second := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "second.db"}, Size: 3}
	runtime.AddRoot(root)
	runtime.PutFile(first, []byte("one"))
	runtime.PutFile(second, []byte("two"))
	runtime.SetListingOrder(root.Name, []string{"second.db", "first.db"})

	if got := runtime.NativeRoots(sessioninventory.AgentAgy); !slices.Equal(got, []sessioninventory.StorageRoot{root}) {
		t.Fatalf("roots = %#v, want %#v", got, []sessioninventory.StorageRoot{root})
	}
	files, err := runtime.ListFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{files[0].Artifact.RelativePath, files[1].Artifact.RelativePath}; !slices.Equal(got, []string{"second.db", "first.db"}) {
		t.Fatalf("listing order = %#v", got)
	}

	runtime.PutFile(first, []byte("updated"))
	if got, err := runtime.ReadFile(first.Artifact, 32); err != nil || string(got) != "updated" {
		t.Fatalf("updated read = %q, %v", got, err)
	}
	if _, err := runtime.ReadFile(first.Artifact, 3); !errors.Is(err, sessioninventory.ErrReadLimit) {
		t.Fatalf("bounded read error = %v, want ErrReadLimit", err)
	}
	if got, eof, err := runtime.ReadAt(first.Artifact, 1, 2); err != nil || eof || string(got) != "pd" {
		t.Fatalf("range read = %q, eof=%v, err=%v", got, eof, err)
	}
	if got, eof, err := runtime.ReadAt(first.Artifact, 6, 4); err != nil || !eof || string(got) != "d" {
		t.Fatalf("final range read = %q, eof=%v, err=%v", got, eof, err)
	}

	wantErr := errors.New("unreadable")
	runtime.SetError(OperationReadFile, first.Artifact.StorageRoot+":"+first.Artifact.RelativePath, wantErr)
	if _, err := runtime.ReadFile(first.Artifact, 32); !errors.Is(err, wantErr) {
		t.Fatalf("injected read error = %v, want %v", err, wantErr)
	}
}

func TestFakeRuntimeModelsSQLiteAndProcessMutation(t *testing.T) {
	t.Parallel()

	runtime := NewFakeRuntime()
	database := sessioninventory.Artifact{StorageRoot: "agy", RelativePath: "conversation.db"}
	query := "select id, parent from trajectory"
	wantRows := sessioninventory.SQLiteResult{Columns: []string{"id", "parent"}, Rows: [][]string{{"root", ""}}}
	runtime.PutSQLite(database, query, wantRows)
	if got, err := runtime.QuerySQLite(database, query, 4096); err != nil || !slices.Equal(got.Columns, wantRows.Columns) || len(got.Rows) != 1 || !slices.Equal(got.Rows[0], wantRows.Rows[0]) {
		t.Fatalf("sqlite result = %#v, %v", got, err)
	}

	runtime.SetProcess("10", "start-a", []string{"11"}, []string{"/native/agy/conversation.db"})
	if got := runtime.ProcessIdentity("10"); got != "start-a" {
		t.Fatalf("identity = %q, want start-a", got)
	}
	runtime.SetProcess("10", "start-b", []string{"12"}, nil)
	if got := runtime.ProcessIdentity("10"); got != "start-b" {
		t.Fatalf("mutated identity = %q, want start-b", got)
	}
	if got := runtime.ProcessChildren()["10"]; !slices.Equal(got, []string{"12"}) {
		t.Fatalf("mutated children = %#v", got)
	}
}
