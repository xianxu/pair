package historyarchive

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalSinkRoundTripCollisionAndCorruption(t *testing.T) {
	s := LocalSink{Root: filepath.Join(archiveTemp(t), "sink")}
	ctx := context.Background()
	if e := s.Put(ctx, "kind/2026/data", bytes.NewBufferString("abc"), 3); e != nil {
		t.Fatal(e)
	}
	if e := s.Put(ctx, "kind/2026/data", bytes.NewBufferString("other"), 8); !errors.Is(e, os.ErrExist) {
		t.Fatalf("collision %v", e)
	}
	r, e := s.Get(ctx, "kind/2026/data")
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := io.ReadAll(r)
	r.Close()
	if string(raw) != "abc" {
		t.Fatal(string(raw))
	}
	keys, e := s.List(ctx, "kind/")
	if e != nil || len(keys) != 1 || keys[0] != "kind/2026/data" {
		t.Fatalf("list %v %v", keys, e)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte("abc")))
	if e = s.Verify(ctx, keys[0], hash, 3); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(s.Root, keys[0]), []byte("bad"), 0600)
	if e = s.Verify(ctx, keys[0], hash, 3); e == nil {
		t.Fatal("corruption accepted")
	}
}
func TestLocalSinkRepeatedSyncFailure(t *testing.T) {
	d := archiveTemp(t)
	ctx := context.Background()
	fail := func(stage string) error {
		if stage == "sync-directory" {
			return errors.New("sync fault")
		}
		return nil
	}
	s := LocalSink{Root: d, Fault: fail}
	e := s.Put(ctx, "a/object", bytes.NewBufferString("abc"), 3)
	if commitoutcome.Of(e) != commitoutcome.Indeterminate {
		t.Fatalf("outcome %v", e)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte("abc")))
	retry := LocalSink{Root: d, Fault: fail}
	if e = retry.Verify(ctx, "a/object", hash, 3); e == nil {
		t.Fatal("retry bypassed sync")
	}
	retry.Fault = nil
	if e = retry.Verify(ctx, "a/object", hash, 3); e != nil {
		t.Fatal(e)
	}
}
func TestLocalSinkBoundsAndUnsafePaths(t *testing.T) {
	d := archiveTemp(t)
	s := LocalSink{Root: d}
	ctx := context.Background()
	for _, key := range []string{"../x", "/x", "a/../x"} {
		if e := s.Put(ctx, key, bytes.NewReader(nil), 1); e == nil {
			t.Fatalf("unsafe %s", key)
		}
	}
	os.Symlink(d, filepath.Join(d, "link"))
	if e := s.Put(ctx, "link/x", bytes.NewReader(nil), 1); e == nil {
		t.Fatal("parent symlink")
	}
	if e := s.Put(ctx, "large", bytes.NewBufferString("ab"), 1); e == nil {
		t.Fatal("oversize")
	}
	if _, e := os.Stat(filepath.Join(d, "large")); !os.IsNotExist(e) {
		t.Fatal("published oversize")
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if e := s.Put(ctx, "cancel", bytes.NewReader(nil), 1); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}

type failingArchiveReader struct{}

func (failingArchiveReader) Read([]byte) (int, error) { return 0, errors.New("source read fault") }
func TestLocalSinkFailedStreamCleansStaleAndOwnedTemps(t *testing.T) {
	d := archiveTemp(t)
	s := LocalSink{Root: d}
	os.WriteFile(filepath.Join(d, ".partial-other"), []byte("retained"), 0600)
	if e := s.Put(context.Background(), "object", failingArchiveReader{}, 20); e == nil {
		t.Fatal("read error ignored")
	}
	entries, e := os.ReadDir(d)
	if e != nil {
		t.Fatal(e)
	}
	if len(entries) != 1 || entries[0].Name() != ".archive-put.lock" {
		t.Fatalf("unexpected cleanup: %v", entries)
	}
}

func TestLocalSinkLockProtectsActiveStage(t *testing.T) {
	d := archiveTemp(t)
	ctx := context.Background()
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	s := LocalSink{Root: d, Fault: func(stage string) error {
		if stage == "sync-file" {
			close(entered)
			<-release
		}
		return nil
	}}
	go func() { done <- s.Put(ctx, "first", bytes.NewBufferString("abc"), 3) }()
	<-entered
	second := LocalSink{Root: d}
	err := second.Put(ctx, "second", bytes.NewReader(nil), 0)
	entries, _ := os.ReadDir(d)
	partials := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".partial-") {
			partials++
		}
	}
	close(release)
	firstErr := <-done
	if err == nil {
		t.Fatal("competing writer was not refused")
	}
	if partials != 1 {
		t.Fatalf("active stage removed: %v", entries)
	}
	if firstErr != nil {
		t.Fatal(firstErr)
	}
	keys, e := second.List(ctx, "")
	if e != nil || len(keys) != 1 || keys[0] != "first" {
		t.Fatalf("internal lock leaked: %v %v", keys, e)
	}
}
func TestLocalSinkReservedNamesRefused(t *testing.T) {
	s := LocalSink{Root: archiveTemp(t)}
	ctx := context.Background()
	for _, key := range []string{".archive-put.lock", ".partial-dead", "a/.partial-dead/file"} {
		if e := s.Put(ctx, key, bytes.NewReader(nil), 0); e == nil {
			t.Fatalf("put reserved %s", key)
		}
		if f, e := s.Get(ctx, key); e == nil {
			f.Close()
			t.Fatalf("get reserved %s", key)
		}
	}
}
