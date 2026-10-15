package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

func cacheIndexLine(name string) string {
	s, _ := BuildSessionNameIndexLine(SessionNameEntry{SessionName: "pair-" + name, ScopeKey: "aaaaaaaaaaaaaaaa", Tag: "work", RepoRoot: "/tmp/project", RepoName: "project"})
	return s + "\n"
}
func TestSessionNameIndexCacheUnchangedAndMutations(t *testing.T) {
	dir := t.TempDir()
	scope := filepath.Join(dir, "repos", "aaaaaaaaaaaaaaaa")
	if err := os.MkdirAll(scope, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(scope, "session-names.jsonl")
	// Resolve the real contract path independently of the cache.
	rt := NewScopedOSRuntime(dir, scope, "")
	_ = rt
	var reads, decodes int
	c := NewSessionNameIndexCacheWithIO(SessionNameIndexCacheIO{ReadFile: func(p string) ([]byte, error) { b, e := os.ReadFile(p); reads += len(b); return b, e }, Decode: func(s string) (SessionNameIndex, error) { decodes += len(s); return DecodeSessionNameIndex(s) }})
	write := func(s string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(cacheIndexLine("first"))
	first, err := c.Read(context.Background(), dir, scope)
	if err != nil || len(first.Entries) != 1 {
		t.Fatalf("cold: %#v %v", first, err)
	}
	if reads == 0 || decodes == 0 {
		t.Fatal("disconnected counters")
	}
	reads, decodes = 0, 0
	for range 3 {
		got, e := c.Read(context.Background(), dir, scope)
		if e != nil || !reflect.DeepEqual(got, first) {
			t.Fatalf("warm %v %v", got, e)
		}
	}
	if reads != 0 || decodes != 0 {
		t.Fatalf("unchanged read=%d decode=%d", reads, decodes)
	}
	info, _ := os.Stat(path)
	write(cacheIndexLine("other"))
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	got, err := c.Read(context.Background(), dir, scope)
	if err != nil || got.Entries[0].SessionName != "pair-other" {
		t.Fatalf("same-size rewrite: %v %v", got, err)
	}
	tmp := path + ".new"
	if err := os.WriteFile(tmp, []byte(cacheIndexLine("third")), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatal(err)
	}
	got, err = c.Read(context.Background(), dir, scope)
	if err != nil || got.Entries[0].SessionName != "pair-third" {
		t.Fatalf("replace: %v %v", got, err)
	}
	write("broken\n")
	if _, e := c.Read(context.Background(), dir, scope); e == nil {
		t.Fatal("corrupt accepted")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	got, err = c.Read(context.Background(), dir, scope)
	if err != nil || len(got.Entries) != 0 {
		t.Fatalf("delete %v %v", got, err)
	}
}

func TestSessionNameIndexCacheConcurrentCancellationAndReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session-names.jsonl")
	if e := os.WriteFile(path, []byte(cacheIndexLine("first")), 0600); e != nil {
		t.Fatal(e)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	var calls atomic.Int32
	c := NewSessionNameIndexCacheWithIO(SessionNameIndexCacheIO{ReadFile: func(p string) ([]byte, error) {
		b, e := os.ReadFile(p)
		calls.Add(1)
		once.Do(func() { close(entered); <-release })
		return b, e
	}})
	done := make(chan SessionNameIndex, 1)
	go func() {
		v, e := c.Read(context.Background(), dir, dir)
		if e != nil {
			done <- SessionNameIndex{}
			return
		}
		done <- v
	}()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := c.Read(ctx, dir, dir); !errors.Is(e, context.Canceled) {
		t.Fatalf("cancel %v", e)
	}
	temp := path + ".new"
	if e := os.WriteFile(temp, []byte(cacheIndexLine("other")), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Rename(temp, path); e != nil {
		t.Fatal(e)
	}
	close(release)
	got := <-done
	if len(got.Entries) != 1 || got.Entries[0].SessionName != "pair-other" {
		t.Fatalf("stale late read: %v", got)
	}
	if calls.Load() != 2 {
		t.Fatalf("loads=%d want bounded retry 2", calls.Load())
	}
	before := calls.Load()
	if _, e := c.Read(context.Background(), dir, dir); e != nil {
		t.Fatal(e)
	}
	if calls.Load() != before {
		t.Fatal("stable replacement not cached")
	}
}
func TestSessionNameIndexCacheScopePrecedenceAndCallerIsolation(t *testing.T) {
	dir := t.TempDir()
	scope := filepath.Join(dir, "repos", "aaaaaaaaaaaaaaaa")
	if e := os.MkdirAll(scope, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, "session-names.jsonl"), []byte(cacheIndexLine("first")+cacheIndexLine("other")), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(scope, "session-names.jsonl"), []byte(cacheIndexLine("third")), 0600); e != nil {
		t.Fatal(e)
	}
	c := NewSessionNameIndexCache()
	got, e := c.Read(context.Background(), dir, scope)
	if e != nil || len(got.Entries) != 1 || got.Entries[0].SessionName != "pair-third" {
		t.Fatalf("scope precedence %v %v", got, e)
	}
	got.Entries[0].SessionName = "pair-mutated"
	got, e = c.Read(context.Background(), dir, scope)
	if e != nil || got.Entries[0].SessionName != "pair-third" {
		t.Fatal("caller mutated cache")
	}
	got, e = c.Read(context.Background(), dir, dir)
	if e != nil || len(got.Entries) != 1 || got.Entries[0].SessionName != "pair-other" {
		t.Fatalf("legacy projection %v %v", got, e)
	}
}
func TestAdvanceIndexCacheRejectsLateCommit(t *testing.T) {
	one := advanceIndexCache(indexCacheState{}, indexCacheEvent{begin: true})
	two := advanceIndexCache(one, indexCacheEvent{begin: true})
	if advanceIndexCache(two, indexCacheEvent{commit: true, generation: one.generation}).ready {
		t.Fatal("late commit granted authority")
	}
	if !advanceIndexCache(two, indexCacheEvent{commit: true, generation: two.generation}).ready {
		t.Fatal("current load not published")
	}
}

func TestSessionNameIndexCacheWaiterCancellationAndSingleFlight(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session-names.jsonl")
	if e := os.WriteFile(path, []byte(cacheIndexLine("first")), 0600); e != nil {
		t.Fatal(e)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	waiterObserved := make(chan struct{}, 1)
	var stats atomic.Int32
	c := NewSessionNameIndexCacheWithIO(SessionNameIndexCacheIO{Stat: func(p string) (os.FileInfo, error) {
		if stats.Add(1) == 2 {
			waiterObserved <- struct{}{}
		}
		return os.Lstat(p)
	}, ReadFile: func(p string) ([]byte, error) { calls.Add(1); close(entered); <-release; return os.ReadFile(p) }})
	first := make(chan error, 1)
	go func() { _, e := c.Read(context.Background(), dir, dir); first <- e }()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	second := make(chan error, 1)
	go func() { _, e := c.Read(ctx, dir, dir); second <- e }()
	<-waiterObserved
	cancel()
	if e := <-second; !errors.Is(e, context.Canceled) {
		t.Fatalf("waiter cancel %v", e)
	}
	close(release)
	if e := <-first; e != nil {
		t.Fatal(e)
	}
	if _, e := c.Read(context.Background(), dir, dir); e != nil {
		t.Fatal(e)
	}
	if calls.Load() != 1 {
		t.Fatalf("duplicate loads: %d", calls.Load())
	}
}
func TestSessionNameIndexCacheEvictsLeastRecentEntry(t *testing.T) {
	c := NewSessionNameIndexCache()
	ctx := context.Background()
	base := t.TempDir()
	for i := range 129 {
		dir := filepath.Join(base, fmt.Sprint(i))
		if e := os.Mkdir(dir, 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(dir, "session-names.jsonl"), []byte(cacheIndexLine("first")), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e := c.Read(ctx, dir, dir); e != nil {
			t.Fatal(e)
		}
	}
	if len(c.entries) != 128 {
		t.Fatalf("entries=%d", len(c.entries))
	}
	if _, ok := c.entries[filepath.Join(base, "0", "session-names.jsonl")]; ok {
		t.Fatal("oldest retained")
	}
}

func TestSessionNameIndexCacheMovingFileHasOneRetry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session-names.jsonl")
	if e := os.WriteFile(path, []byte(cacheIndexLine("first")), 0600); e != nil {
		t.Fatal(e)
	}
	calls := 0
	c := NewSessionNameIndexCacheWithIO(SessionNameIndexCacheIO{ReadFile: func(p string) ([]byte, error) {
		raw, e := os.ReadFile(p)
		calls++
		tmp := p + ".new"
		if err := os.WriteFile(tmp, []byte(cacheIndexLine(fmt.Sprint(calls))), 0600); err != nil {
			return nil, err
		}
		if err := os.Rename(tmp, p); err != nil {
			return nil, err
		}
		return raw, e
	}})
	if _, e := c.Read(context.Background(), dir, dir); e == nil {
		t.Fatal("moving file published")
	}
	if calls != 2 {
		t.Fatalf("attempts=%d", calls)
	}
}
func TestSessionNameIndexCacheCanceledLoadDoesNotPublish(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session-names.jsonl")
	if e := os.WriteFile(path, []byte(cacheIndexLine("first")), 0600); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	c := NewSessionNameIndexCacheWithIO(SessionNameIndexCacheIO{ReadFile: func(p string) ([]byte, error) {
		calls++
		if calls == 1 {
			cancel()
		}
		return os.ReadFile(p)
	}})
	if _, e := c.Read(ctx, dir, dir); !errors.Is(e, context.Canceled) {
		t.Fatalf("cancel load %v", e)
	}
	if _, e := c.Read(context.Background(), dir, dir); e != nil {
		t.Fatal(e)
	}
	if calls != 2 {
		t.Fatal("canceled read cached")
	}
}
