package sessioninventory_test

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestActivityCacheReusesValidatedContentAndInvalidatesRewrite(t *testing.T) {
	rt, transcript, _ := proofBackedCodexFixture(t, "019d1111-1111-7111-8111-111111111111", nil)
	at := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	// Give the bound transcript a mtime using the real stateful fixture API.
	entry, err := rt.StatFile(transcript)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := rt.ReadFile(transcript, -1)
	if err != nil {
		t.Fatal(err)
	}
	entry.ModTime = &at
	rt.PutFile(entry, raw)
	cache := sessioninventory.NewActivityCache()
	first, ok, err := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || !ok {
		t.Fatalf("cold activity=%v,%v,%v", first, ok, err)
	}
	before := rt.OperationCount("read_at", "")
	second, ok, err := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || !ok || first != second {
		t.Fatalf("warm activity=%v,%v,%v", second, ok, err)
	}
	if got := rt.OperationCount("read_at", ""); got != before {
		t.Fatalf("warm repeated reads %d -> %d", before, got)
	}
	rt.ReplaceFile(entry, []byte("{invalid}\n"), "replacement", "changed")
	if got, ok, err := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); err == nil && ok {
		t.Fatalf("rewritten transcript returned cached authority: %v", got)
	}
}

func TestActivityCacheCancellationCannotReuseSuccess(t *testing.T) {
	rt, _, _ := proofBackedCodexFixture(t, "019d1111-1111-7111-8111-111111111111", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok, err := sessioninventory.NewActivityCache().Query(ctx, rt, "scope", "work", sessioninventory.AgentCodex); err == nil || ok {
		t.Fatalf("canceled activity ok=%v err=%v", ok, err)
	}
}

type catalogMetadataFailureRuntime struct {
	*sessioninventorytest.FakeRuntime
}

func (catalogMetadataFailureRuntime) ActivityCatalogFingerprint() (sessioninventory.ArtifactFingerprint, error) {
	return sessioninventory.ArtifactFingerprint{}, os.ErrPermission
}
func TestActivityCacheCatalogMetadataFailureUsesUncachedValidation(t *testing.T) {
	at := time.Now()
	rt, _, _ := proofBackedCodexFixture(t, "019d1111-1111-7111-8111-111111111111", &at)
	q, err := sessioninventory.QuerySession(rt, "scope", "work", sessioninventory.AgentCodex)
	if err != nil {
		t.Fatal(err)
	}
	expected, ok, err := sessioninventory.ActivityForSession(rt, q)
	if err != nil || !ok {
		t.Fatalf("uncached control=%v,%v", ok, err)
	}
	got, ok, err := sessioninventory.NewActivityCache().Query(context.Background(), catalogMetadataFailureRuntime{rt}, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || !ok || got != expected {
		t.Fatalf("disposable catalog metadata failure changed activity: %v,%v,%v", got, ok, err)
	}
}

type blockingActivityCatalog struct {
	*sessioninventorytest.FakeRuntime
	once             sync.Once
	entered, release chan struct{}
}

func (r *blockingActivityCatalog) LoadSessionInventoryCatalog() (sessioninventory.Catalog, error) {
	r.once.Do(func() { close(r.entered); <-r.release })
	return r.FakeRuntime.LoadSessionInventoryCatalog()
}
func TestActivityCacheCanceledWaiterDoesNotCancelLoader(t *testing.T) {
	at := time.Now()
	fake, _, _ := proofBackedCodexFixture(t, "019d1111-1111-7111-8111-111111111111", &at)
	rt := &blockingActivityCatalog{FakeRuntime: fake, entered: make(chan struct{}), release: make(chan struct{})}
	cache := sessioninventory.NewActivityCache()
	loaded := make(chan error, 1)
	go func() {
		_, ok, e := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex)
		if e == nil && !ok {
			e = os.ErrInvalid
		}
		loaded <- e
	}()
	<-rt.entered
	ctx, cancel := context.WithCancel(context.Background())
	waiter := &activityWaitContext{Context: ctx, observed: make(chan struct{})}
	waited := make(chan error, 1)
	go func() {
		_, ok, err := cache.Query(waiter, rt, "scope", "work", sessioninventory.AgentCodex)
		if err == nil || ok {
			waited <- os.ErrInvalid
			return
		}
		waited <- err
	}()
	<-waiter.observed
	cancel()
	if err := <-waited; !errors.Is(err, context.Canceled) {
		t.Fatalf("queued waiter err=%v", err)
	}
	close(rt.release)
	if err := <-loaded; err != nil {
		t.Fatal(err)
	}
	if _, ok, err := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); err != nil || !ok {
		t.Fatalf("loader cache lost ok=%v err=%v", ok, err)
	}
}

func TestActivityCacheMissingLedgerCreationInvalidatesAbsence(t *testing.T) {
	at := time.Now()
	rt, _, ledger := proofBackedCodexFixture(t, "019d1111-1111-7111-8111-111111111111", &at)
	entry, err := rt.StatFile(ledger)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := rt.ReadFile(ledger, -1)
	if err != nil {
		t.Fatal(err)
	}
	rt.DeleteFile(ledger)
	cache := sessioninventory.NewActivityCache()
	if _, ok, err := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); err != nil || ok {
		t.Fatalf("absent=%v,%v", ok, err)
	}
	rt.PutFile(entry, raw)
	if _, ok, err := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); err != nil || !ok {
		t.Fatalf("new ledger failed invalidation=%v,%v", ok, err)
	}
}

// The cache's gate select is the first consumer of Done, so this barrier
// establishes an actually queued waiter instead of a pre-canceled query.
type activityWaitContext struct {
	context.Context
	once     sync.Once
	observed chan struct{}
}

func (c *activityWaitContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.observed) })
	return c.Context.Done()
}

func TestActivityCacheActiveLoaderCancellationDoesNotPublish(t *testing.T) {
	at := time.Now()
	fake, _, _ := proofBackedCodexFixture(t, "019d1111-1111-7111-8111-111111111111", &at)
	rt := &blockingActivityCatalog{FakeRuntime: fake, entered: make(chan struct{}), release: make(chan struct{})}
	cache := sessioninventory.NewActivityCache()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, ok, e := cache.Query(ctx, rt, "scope", "work", sessioninventory.AgentCodex)
		if e == nil && ok {
			done <- os.ErrInvalid
			return
		}
		done <- e
	}()
	<-rt.entered
	cancel()
	close(rt.release)
	if e := <-done; !errors.Is(e, context.Canceled) {
		t.Fatalf("active loader cancellation=%v", e)
	}
	reads := fake.OperationCount("read_at", "")
	if _, ok, e := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); e != nil || !ok {
		t.Fatalf("fresh query=%v,%v", ok, e)
	}
	if fake.OperationCount("read_at", "") <= reads {
		t.Fatal("canceled loader published reusable evidence")
	}
	reads = fake.OperationCount("read_at", "")
	if _, ok, e := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); e != nil || !ok {
		t.Fatalf("warm query=%v,%v", ok, e)
	}
	if fake.OperationCount("read_at", "") != reads {
		t.Fatal("fresh recovery did not become warm")
	}
}

// The snapshot has already observed the old ledger before Load blocks. A
// replacement here must discard that in-flight generation and retry it.
func TestActivityCacheReplacementDuringBlockedLoadRetriesBeforePublication(t *testing.T) {
	at := time.Now()
	fake, _, ledger := proofBackedCodexFixture(t, "019d1111-1111-7111-8111-111111111111", &at)
	entry, e := fake.StatFile(ledger)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := fake.ReadFile(ledger, -1)
	if e != nil {
		t.Fatal(e)
	}
	rt := &blockingActivityCatalog{FakeRuntime: fake, entered: make(chan struct{}), release: make(chan struct{})}
	cache := sessioninventory.NewActivityCache()
	done := make(chan error, 1)
	go func() {
		_, ok, e := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex)
		if e == nil && !ok {
			e = os.ErrInvalid
		}
		done <- e
	}()
	<-rt.entered
	fake.ReplaceFile(entry, raw, "replacement-ledger", "mutation-replaced")
	close(rt.release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	// The read count distinguishes a fresh replacement validation from merely
	// publishing the pre-replacement result with its old stamp.
	if reads := fake.OperationCount("read_at", ledger.StorageRoot+":"+ledger.RelativePath); reads < 2 {
		t.Fatalf("replacement was not retried: ledger reads=%d", reads)
	}
	reads := fake.OperationCount("read_at", "")
	if _, ok, e := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); e != nil || !ok {
		t.Fatalf("stable replacement=%v,%v", ok, e)
	}
	if fake.OperationCount("read_at", "") != reads {
		t.Fatal("replacement generation was not cached")
	}
}

type activityDriftCatalog struct {
	*sessioninventorytest.FakeRuntime
	stamps   atomic.Int32
	loads    atomic.Int32
	barriers chan int32
	releases chan struct{}
	drift    atomic.Bool
}

func (r *activityDriftCatalog) LoadSessionInventoryCatalog() (sessioninventory.Catalog, error) {
	r.loads.Add(1)
	return r.FakeRuntime.LoadSessionInventoryCatalog()
}
func (r *activityDriftCatalog) ActivityCatalogFingerprint() (sessioninventory.ArtifactFingerprint, error) {
	n := r.stamps.Add(1)
	if r.drift.Load() && (n == 2 || n == 4) {
		r.barriers <- n
		<-r.releases
	}
	return r.FakeRuntime.ActivityCatalogFingerprint()
}
func TestActivityCacheRepeatedDriftHasBoundedRetryAndNoPublication(t *testing.T) {
	at := time.Now()
	fake, _, _ := proofBackedCodexFixture(t, "019d1111-1111-7111-8111-111111111111", &at)
	rt := &activityDriftCatalog{FakeRuntime: fake, barriers: make(chan int32), releases: make(chan struct{})}
	rt.drift.Store(true)
	cache := sessioninventory.NewActivityCache()
	done := make(chan error, 1)
	go func() {
		_, ok, e := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex)
		if e == nil || ok {
			done <- os.ErrInvalid
			return
		}
		done <- e
	}()
	for _, want := range []int32{2, 4} {
		select {
		case n := <-rt.barriers:
			if n != want {
				t.Fatalf("drift observation=%d want%d", n, want)
			}
		case err := <-done:
			t.Fatalf("load completed before drift observation %d: %v", want, err)
		}
		if e := fake.PublishSessionInventoryValidations(nil); e != nil {
			t.Fatal(e)
		}
		rt.releases <- struct{}{}
	}
	if e := <-done; !errors.Is(e, sessioninventory.ErrArtifactChanged) {
		t.Fatalf("repeated drift=%v", e)
	}
	if n := rt.stamps.Load(); n != 4 {
		t.Fatalf("unbounded/reused attempts: stamps=%d", n)
	}
	if n := rt.loads.Load(); n != 2 {
		t.Fatalf("attempts=%d want2", n)
	}
	rt.drift.Store(false)
	reads := fake.OperationCount("read_at", "")
	if _, ok, e := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); e != nil || !ok {
		t.Fatalf("unchanged recovery=%v,%v", ok, e)
	}
	if fake.OperationCount("read_at", "") <= reads {
		t.Fatal("drifting result was published")
	}
	reads = fake.OperationCount("read_at", "")
	if _, ok, e := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); e != nil || !ok {
		t.Fatalf("warm=%v,%v", ok, e)
	}
	if fake.OperationCount("read_at", "") != reads {
		t.Fatal("recovered result not cached")
	}
}
