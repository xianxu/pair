package pairlifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func TestConsumeAttemptCriticalSection(t *testing.T) {
	t.Parallel()
	paths := lifecyclePaths(t)
	runtime := newMemoryRuntime()
	store := Store{Runtime: runtime}
	request := validQuitRequest()
	if _, err := store.ConsumeAttempt(context.Background(), paths, 1, func(context.Context, *LockedAttempt, QuitRequest) CleanupResult {
		t.Fatal("cleanup ran without committed request")
		return CleanupResult{}
	}); err == nil {
		t.Fatal("missing request was consumed")
	}
	if err := store.PublishRequest(paths, request); err != nil {
		t.Fatal(err)
	}

	calls := 0
	wantTime := time.Date(2026, 8, 30, 1, 2, 3, 0, time.UTC)
	completion, err := store.ConsumeAttempt(context.Background(), paths, 1, func(_ context.Context, locked *LockedAttempt, got QuitRequest) CleanupResult {
		calls++
		if got != request || locked.Request() != request {
			t.Fatalf("callback request=%#v locked=%#v", got, locked.Request())
		}
		return CleanupResult{Outcome: CompletionSuccess, CompletedAt: wantTime}
	})
	if err != nil {
		t.Fatal(err)
	}
	if completion.Outcome != CompletionSuccess || !completion.CompletedAt.Equal(wantTime) || calls != 1 {
		t.Fatalf("completion=%#v calls=%d", completion, calls)
	}
	second, err := store.ConsumeAttempt(context.Background(), paths, 1, func(context.Context, *LockedAttempt, QuitRequest) CleanupResult {
		calls++
		return CleanupResult{}
	})
	if err != nil || second != completion || calls != 1 {
		t.Fatalf("dedupe completion=%#v err=%v calls=%d", second, err, calls)
	}
}

func TestConsumeAttemptSerializesDifferentAttempts(t *testing.T) {
	paths := lifecyclePaths(t)
	runtime := newMemoryRuntime()
	store := Store{Runtime: runtime}
	first := validQuitRequest()
	second := validQuitRequest()
	second.Attempt = 2
	second.CompletionKey = "quit-completion-2"
	if err := store.PublishRequest(paths, first); err != nil {
		t.Fatal(err)
	}
	if err := store.PublishRequest(paths, second); err != nil {
		t.Fatal(err)
	}
	lockAttempts := make(chan struct{}, 2)
	runtime.beforeLock = func() { lockAttempts <- struct{}{} }
	enteredFirst := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		_, err := store.ConsumeAttempt(context.Background(), paths, 1, func(context.Context, *LockedAttempt, QuitRequest) CleanupResult {
			close(enteredFirst)
			<-releaseFirst
			return CleanupResult{Outcome: CompletionSuccess, CompletedAt: time.Now()}
		})
		firstDone <- err
	}()
	<-lockAttempts
	<-enteredFirst
	enteredSecond := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		_, err := store.ConsumeAttempt(context.Background(), paths, 2, func(context.Context, *LockedAttempt, QuitRequest) CleanupResult {
			close(enteredSecond)
			return CleanupResult{Outcome: CompletionSuccess, CompletedAt: time.Now()}
		})
		secondDone <- err
	}()
	<-lockAttempts
	select {
	case <-enteredSecond:
		t.Fatal("different attempt entered while transaction lock was held")
	default:
	}
	close(releaseFirst)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if err := <-secondDone; err != nil {
		t.Fatal(err)
	}
}

func lifecyclePaths(t *testing.T) artifactpath.LifecyclePaths {
	t.Helper()
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: "/data", RepoScope: "scope", Tag: "work"})
	if err != nil {
		t.Fatal(err)
	}
	lifecycle, err := paths.Lifecycle("nonce-1")
	if err != nil {
		t.Fatal(err)
	}
	return lifecycle
}

func TestLifecycleStorePublishFailureMatrix(t *testing.T) {
	t.Parallel()

	for _, recordKind := range []RecordKind{RecordRequest, RecordCompletion} {
		for _, test := range []struct {
			stage       string
			wantOutcome PublicationOutcome
			wantFinal   bool
		}{
			{stage: "mkdir", wantOutcome: NotCommitted},
			{stage: "lock", wantOutcome: NotCommitted},
			{stage: "create", wantOutcome: NotCommitted},
			{stage: "write", wantOutcome: NotCommitted},
			{stage: "file-sync", wantOutcome: NotCommitted},
			{stage: "close", wantOutcome: NotCommitted},
			{stage: "rename", wantOutcome: NotCommitted},
			{stage: "directory-sync", wantOutcome: Indeterminate, wantFinal: true},
			{stage: "unlock", wantOutcome: Committed, wantFinal: true},
		} {
			recordKind, test := recordKind, test
			t.Run(string(recordKind)+"/"+test.stage, func(t *testing.T) {
				t.Parallel()
				runtime := newMemoryRuntime()
				runtime.fail[test.stage] = 1
				paths := lifecyclePaths(t)
				store := Store{Runtime: runtime}
				err := publishTestRecord(store, paths, recordKind)
				if err == nil {
					t.Fatal("injected publication failure returned nil")
				}
				if got := PublicationOutcomeOf(err); got != test.wantOutcome {
					t.Fatalf("outcome=%v, want %v (err=%v)", got, test.wantOutcome, err)
				}
				final := testRecordPath(t, paths, recordKind)
				if _, ok := runtime.files[final]; ok != test.wantFinal {
					t.Fatalf("final exists=%v, want %v", ok, test.wantFinal)
				}
				for path := range runtime.files {
					if filepath.Dir(path) == paths.Dir() && filepath.Base(path) != filepath.Base(final) && path != paths.Lock() {
						t.Fatalf("temporary artifact survived failure: %s", path)
					}
				}
			})
		}
	}
}

func TestLifecycleStoreRejectsInvalidBeforeIO(t *testing.T) {
	t.Parallel()
	for _, mutate := range []func(*QuitRequest){
		func(request *QuitRequest) { request.Attempt = 0 },
		func(request *QuitRequest) { request.CompletionKey = "other-key" },
	} {
		runtime := newMemoryRuntime()
		request := validQuitRequest()
		mutate(&request)
		err := (Store{Runtime: runtime}).PublishRequest(lifecyclePaths(t), request)
		if err == nil || PublicationOutcomeOf(err) != NotCommitted {
			t.Fatalf("err=%v outcome=%v", err, PublicationOutcomeOf(err))
		}
		if len(runtime.calls) != 0 {
			t.Fatalf("invalid record reached IO: %v", runtime.calls)
		}
	}
}

func TestLifecycleStoreImmutableFinal(t *testing.T) {
	t.Parallel()
	paths := lifecyclePaths(t)

	t.Run("identical request is idempotent", func(t *testing.T) {
		runtime := newMemoryRuntime()
		store := Store{Runtime: runtime}
		if err := store.PublishRequest(paths, validQuitRequest()); err != nil {
			t.Fatal(err)
		}
		renames := runtime.callCount("rename")
		if err := store.PublishRequest(paths, validQuitRequest()); err != nil {
			t.Fatal(err)
		}
		if runtime.callCount("rename") != renames {
			t.Fatal("idempotent publication renamed over final")
		}
	})

	t.Run("different valid request conflicts", func(t *testing.T) {
		runtime := newMemoryRuntime()
		store := Store{Runtime: runtime}
		if err := store.PublishRequest(paths, validQuitRequest()); err != nil {
			t.Fatal(err)
		}
		other := validQuitRequest()
		other.Session = "pair-other"
		err := store.PublishRequest(paths, other)
		if PublicationOutcomeOf(err) != Conflict {
			t.Fatalf("outcome=%v err=%v", PublicationOutcomeOf(err), err)
		}
	})

	t.Run("invalid final refuses", func(t *testing.T) {
		runtime := newMemoryRuntime()
		final := testRecordPath(t, paths, RecordRequest)
		runtime.files[final] = []byte("not json")
		err := (Store{Runtime: runtime}).PublishRequest(paths, validQuitRequest())
		if PublicationOutcomeOf(err) != Conflict {
			t.Fatalf("outcome=%v err=%v", PublicationOutcomeOf(err), err)
		}
		if !bytes.Equal(runtime.files[final], []byte("not json")) {
			t.Fatal("invalid final was overwritten")
		}
	})

	t.Run("prepared final reconciles", func(t *testing.T) {
		runtime := newMemoryRuntime()
		runtime.fail["directory-sync"] = 1
		store := Store{Runtime: runtime}
		if err := store.PublishCompletion(paths, validQuitCompletion()); PublicationOutcomeOf(err) != Indeterminate {
			t.Fatalf("publish outcome=%v err=%v", PublicationOutcomeOf(err), err)
		}
		if err := store.Reconcile(paths, RecordCompletion, 1); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("invalid prepared final does not reconcile", func(t *testing.T) {
		runtime := newMemoryRuntime()
		final := testRecordPath(t, paths, RecordCompletion)
		completion := validQuitCompletion()
		completion.CompletionKey = "safe-but-wrong"
		raw, err := json.Marshal(completion)
		if err != nil {
			t.Fatal(err)
		}
		runtime.files[final] = append(raw, '\n')
		err = (Store{Runtime: runtime}).Reconcile(paths, RecordCompletion, 1)
		if PublicationOutcomeOf(err) != Conflict {
			t.Fatalf("outcome=%v err=%v", PublicationOutcomeOf(err), err)
		}
	})
}

func TestLifecycleStoreConcurrentPublishers(t *testing.T) {
	t.Parallel()
	for _, different := range []bool{false, true} {
		name := "identical"
		if different {
			name = "conflicting"
		}
		t.Run(name, func(t *testing.T) {
			lifecycle := lifecyclePaths(t)
			runtime := newMemoryRuntime()
			requests := []QuitRequest{validQuitRequest(), validQuitRequest()}
			if different {
				requests[1].Session = "pair-other"
			}
			start := make(chan struct{})
			results := make(chan error, 2)
			for _, request := range requests {
				request := request
				go func() {
					<-start
					results <- (Store{Runtime: runtime}).PublishRequest(lifecycle, request)
				}()
			}
			close(start)
			outcomes := []PublicationOutcome{PublicationOutcomeOf(<-results), PublicationOutcomeOf(<-results)}
			if different {
				if !containsOutcomes(outcomes, Committed, Conflict) {
					t.Fatalf("outcomes=%v, want committed+conflict", outcomes)
				}
			} else if outcomes[0] != Committed || outcomes[1] != Committed {
				t.Fatalf("outcomes=%v, want both committed", outcomes)
			}
		})
	}
}

func publishTestRecord(store Store, paths artifactpath.LifecyclePaths, kind RecordKind) error {
	if kind == RecordRequest {
		return store.PublishRequest(paths, validQuitRequest())
	}
	return store.PublishCompletion(paths, validQuitCompletion())
}

func testRecordPath(t *testing.T, paths artifactpath.LifecyclePaths, kind RecordKind) string {
	t.Helper()
	var path string
	var err error
	if kind == RecordRequest {
		path, err = paths.Request(1)
	} else {
		path, err = paths.Completion(1)
	}
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func containsOutcomes(got []PublicationOutcome, first, second PublicationOutcome) bool {
	return (got[0] == first && got[1] == second) || (got[0] == second && got[1] == first)
}

type memoryRuntime struct {
	mu         sync.Mutex
	lockMu     sync.Mutex
	files      map[string][]byte
	fail       map[string]int
	calls      []string
	nextTmp    int
	locked     bool
	beforeLock func()
}

func newMemoryRuntime() *memoryRuntime {
	return &memoryRuntime{files: map[string][]byte{}, fail: map[string]int{}}
}

func (r *memoryRuntime) shouldFail(stage string) error {
	r.calls = append(r.calls, stage)
	if r.fail[stage] > 0 {
		r.fail[stage]--
		return fmt.Errorf("injected %s failure", stage)
	}
	return nil
}

func (r *memoryRuntime) MkdirAll(string, os.FileMode) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.shouldFail("mkdir")
}
func (r *memoryRuntime) Lock(string) (Unlocker, error) {
	r.mu.Lock()
	if err := r.shouldFail("lock"); err != nil {
		r.mu.Unlock()
		return nil, err
	}
	r.mu.Unlock()
	if r.beforeLock != nil {
		r.beforeLock()
	}
	r.lockMu.Lock()
	r.mu.Lock()
	r.locked = true
	r.mu.Unlock()
	return &memoryUnlocker{runtime: r}, nil
}
func (r *memoryRuntime) ReadFile(path string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, "read")
	raw, ok := r.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), raw...), nil
}
func (r *memoryRuntime) CreateTemp(dir, _ string) (StoreFile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.shouldFail("create"); err != nil {
		return nil, err
	}
	r.nextTmp++
	return &memoryFile{runtime: r, name: filepath.Join(dir, fmt.Sprintf(".temp-%d", r.nextTmp))}, nil
}
func (r *memoryRuntime) Remove(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, "remove")
	delete(r.files, path)
	return nil
}
func (r *memoryRuntime) Rename(oldPath, newPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.shouldFail("rename"); err != nil {
		return err
	}
	r.files[newPath] = append([]byte(nil), r.files[oldPath]...)
	delete(r.files, oldPath)
	return nil
}
func (r *memoryRuntime) SyncDirectory(string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.shouldFail("directory-sync")
}
func (r *memoryRuntime) callCount(stage string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, call := range r.calls {
		if call == stage {
			count++
		}
	}
	return count
}

type memoryUnlocker struct{ runtime *memoryRuntime }

func (u *memoryUnlocker) Close() error {
	u.runtime.mu.Lock()
	u.runtime.locked = false
	err := u.runtime.shouldFail("unlock")
	u.runtime.mu.Unlock()
	u.runtime.lockMu.Unlock()
	return err
}

type memoryFile struct {
	runtime *memoryRuntime
	name    string
	buffer  bytes.Buffer
}

func (f *memoryFile) Write(raw []byte) (int, error) {
	f.runtime.mu.Lock()
	defer f.runtime.mu.Unlock()
	if err := f.runtime.shouldFail("write"); err != nil {
		if len(raw) > 1 {
			_, _ = f.buffer.Write(raw[:1])
			f.runtime.files[f.name] = append([]byte(nil), f.buffer.Bytes()...)
			return 1, err
		}
		return 0, err
	}
	n, err := f.buffer.Write(raw)
	f.runtime.files[f.name] = append([]byte(nil), f.buffer.Bytes()...)
	return n, err
}
func (f *memoryFile) Sync() error {
	f.runtime.mu.Lock()
	defer f.runtime.mu.Unlock()
	return f.runtime.shouldFail("file-sync")
}
func (f *memoryFile) Close() error {
	f.runtime.mu.Lock()
	defer f.runtime.mu.Unlock()
	return f.runtime.shouldFail("close")
}
func (f *memoryFile) Name() string { return f.name }

var _ io.Writer = (*memoryFile)(nil)
