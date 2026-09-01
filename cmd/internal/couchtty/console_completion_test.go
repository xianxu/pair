package couchtty

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestOSDirectoryBatchReaderReturnsOnlyNavigableDirectoriesInBatches(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"alpha", "beta", ".hidden"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "alpha"), filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	var batches [][]CompletionEntry
	err := (OSDirectoryBatchReader{}).ReadDirectoryBatches(context.Background(), root, 2, func(batch []CompletionEntry) bool {
		batches = append(batches, append([]CompletionEntry(nil), batch...))
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	var got []CompletionEntry
	for _, batch := range batches {
		if len(batch) > 2 {
			t.Fatalf("batch exceeded bound: %+v", batch)
		}
		got = append(got, batch...)
	}
	sort.Slice(got, func(i, j int) bool { return got[i].Name < got[j].Name })
	want := []CompletionEntry{{Name: ".hidden", Directory: true}, {Name: "alias", Directory: true}, {Name: "alpha", Directory: true}, {Name: "beta", Directory: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entries = %+v, want %+v", got, want)
	}
}

type fakeDirectoryBatchReader struct {
	started chan string
	release chan struct{}
	entries map[string][]CompletionEntry
}

func (f *fakeDirectoryBatchReader) ReadDirectoryBatches(ctx context.Context, directory string, _ int, yield func([]CompletionEntry) bool) error {
	f.started <- directory
	select {
	case <-f.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	yield(f.entries[directory])
	return nil
}

func TestConsoleCompletionRunsLatestPendingRequest(t *testing.T) {
	f := newFixture(t, 24, 80)
	reader := &fakeDirectoryBatchReader{started: make(chan string, 2), release: make(chan struct{}), entries: map[string][]CompletionEntry{"two/": {{Name: "target", Directory: true}}}}
	f.con.SetDirectoryBatchReader(reader)
	f.con.dispatchMenuEffects([]MenuEffect{{Completion: &CompletionRequest{Identity: CompletionIdentity{FrameInstance: 1, Generation: 1}, Path: "one/"}}})
	if got := <-reader.started; got != "one/" {
		t.Fatalf("first directory = %q", got)
	}
	f.con.dispatchMenuEffects([]MenuEffect{{Completion: &CompletionRequest{Identity: CompletionIdentity{FrameInstance: 1, Generation: 2}, Path: "two/"}}})
	if got := <-reader.started; got != "two/" {
		t.Fatalf("latest directory = %q", got)
	}
	close(reader.release)
	f.con.Stop()
}
