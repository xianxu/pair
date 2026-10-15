package sessioninventory_test

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"os"
	"sync"
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
	cancel()
	if _, ok, err := cache.Query(ctx, rt, "scope", "work", sessioninventory.AgentCodex); err == nil || ok {
		t.Fatalf("waiter ok=%v err=%v", ok, err)
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
