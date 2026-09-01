package couchtty

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
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

type closeErrorDirectory struct {
	directoryCursor
	err error
}

func (d closeErrorDirectory) Close() error {
	return errors.Join(d.directoryCursor.Close(), d.err)
}

func TestOSDirectoryBatchReaderReturnsCloseFailure(t *testing.T) {
	want := errors.New("close failed")
	reader := OSDirectoryBatchReader{Open: func(path string) (directoryCursor, error) {
		file, err := os.Open(path)
		return closeErrorDirectory{directoryCursor: file, err: want}, err
	}}
	if err := reader.ReadDirectoryBatches(context.Background(), t.TempDir(), 2, func([]CompletionEntry) bool { return true }); !errors.Is(err, want) {
		t.Fatalf("close error = %v, want joined %v", err, want)
	}
}

type fakeDirectoryBatchReader struct {
	started chan string
	release chan struct{}
	entries map[string][]CompletionEntry
	batches chan int
	errors  map[string]error
}

func (f *fakeDirectoryBatchReader) ReadDirectoryBatches(ctx context.Context, directory string, batchSize int, yield func([]CompletionEntry) bool) error {
	if f.started != nil {
		f.started <- directory
	}
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	entries := f.entries[directory]
	for len(entries) > 0 {
		count := min(batchSize, len(entries))
		if f.batches != nil {
			f.batches <- count
		}
		if !yield(append([]CompletionEntry(nil), entries[:count]...)) {
			return context.Canceled
		}
		entries = entries[count:]
	}
	return f.errors[directory]
}

func TestFakeDirectoryBatchReaderModelsBoundedBatchesAndErrors(t *testing.T) {
	want := errors.New("unreadable")
	entries := make([]CompletionEntry, completionBatchSize+3)
	reader := &fakeDirectoryBatchReader{entries: map[string][]CompletionEntry{".": entries}, batches: make(chan int, 2), errors: map[string]error{".": want}}
	var total int
	err := reader.ReadDirectoryBatches(context.Background(), ".", completionBatchSize, func(batch []CompletionEntry) bool {
		total += len(batch)
		return true
	})
	if !errors.Is(err, want) || total != len(entries) || <-reader.batches != completionBatchSize || <-reader.batches != 3 {
		t.Fatalf("fake contract: err=%v total=%d", err, total)
	}
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

func TestConsoleCompletionReaderErrorStaysLocalToCurrentPath(t *testing.T) {
	f := newFixture(t, 24, 80)
	want := errors.New("directory unreadable")
	f.con.SetDirectoryBatchReader(&fakeDirectoryBatchReader{errors: map[string]error{".": want}})
	state := NewMenuState(nil, couchcore.ThreadAddress{})
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: 'x'})
	state, effects := reduceKey(state, PanelKey{Kind: KeyTab})
	f.con.mu.Lock()
	f.con.menu, f.con.menuReady = state, true
	f.con.mu.Unlock()
	f.con.dispatchMenuEffects(effects)
	waitUpTo(t, 250*time.Millisecond, "completion error", func() bool {
		got := f.con.menuSnapshot()
		return got.CurrentFrame().Path == "x" && got.Notice.Level == MenuNoticeError && strings.Contains(got.Notice.Text, want.Error())
	})
}

func TestConsoleCompletionCancellationPreservesJoinedTerminalFailure(t *testing.T) {
	f := newFixture(t, 24, 80)
	want := errors.New("close failed after cancellation")
	f.con.SetDirectoryBatchReader(&fakeDirectoryBatchReader{errors: map[string]error{".": errors.Join(context.Canceled, want)}})
	state := NewMenuState(nil, couchcore.ThreadAddress{})
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state, effects := reduceKey(state, PanelKey{Kind: KeyTab})
	f.con.mu.Lock()
	f.con.menu, f.con.menuReady = state, true
	f.con.mu.Unlock()
	f.con.dispatchMenuEffects(effects)
	waitUpTo(t, 250*time.Millisecond, "joined terminal error", func() bool {
		return strings.Contains(f.con.menuSnapshot().Notice.Text, want.Error())
	})
}
