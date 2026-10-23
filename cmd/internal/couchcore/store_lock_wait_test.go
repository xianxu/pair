package couchcore

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/storagegc"
	"golang.org/x/sys/unix"
)

// coordinatedLockFixture is a coordinated store, as production builds it
// (couch.go), with its store.lock created.
func coordinatedLockFixture(t *testing.T) *ThreadStore {
	t.Helper()
	store, _ := retentionStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = store.namespace.Dir(), store.namespace.Dir()
	if _, err := store.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	return store
}

func observeStoreLockContention(t *testing.T, observe func()) {
	t.Helper()
	onStoreLockContended = observe
	t.Cleanup(func() { onStoreLockContended = nil })
}

// #827: a preview read holds store.lock outside the coordinator. A foreground
// park write that meets it waits it out instead of failing at the first
// EWOULDBLOCK -- the failure that left full-quit parks behind.
func TestForegroundStoreWriteWaitsOutABriefPreviewHold(t *testing.T) {
	store := coordinatedLockFixture(t)
	release := holdStoreLock(t, store)
	var once sync.Once
	observeStoreLockContention(t, func() { once.Do(release) })
	ran := false
	err := store.withUpdateLockContext(context.Background(), func() error { ran = true; return nil })
	if err != nil || !ran {
		t.Fatalf("foreground write refused a briefly held store: ran=%v %v", ran, err)
	}
}

// Past the bound the write refuses with the typed busy error, never a raw
// errno, and never after waiting less than the bound.
func TestForegroundStoreWriteRefusesTypedPastTheBound(t *testing.T) {
	store := coordinatedLockFixture(t)
	release := holdStoreLock(t, store)
	defer release()
	defer func(saved time.Duration) { storeReadLockWait = saved }(storeReadLockWait)
	storeReadLockWait = 60 * time.Millisecond
	start := time.Now()
	err := store.withUpdateLockContext(context.Background(), func() error { return nil })
	if !errors.Is(err, ErrThreadStoreBusy) || errors.Is(err, unix.EWOULDBLOCK) {
		t.Fatalf("held store: %v", err)
	}
	if waited := time.Since(start); waited < storeReadLockWait {
		t.Fatalf("refused after %s, before the %s bound", waited, storeReadLockWait)
	}
}

// Cancellation still ends the wait at once.
func TestForegroundStoreWriteWaitStopsOnCancellation(t *testing.T) {
	store := coordinatedLockFixture(t)
	release := holdStoreLock(t, store)
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	observeStoreLockContention(t, cancel)
	if err := store.withUpdateLockContext(ctx, func() error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled wait: %v", err)
	}
}

// Retention maintenance keeps its yield: at the first contention it returns,
// without polling, and maps to the coordinator's busy signal.
func TestMaintenanceStoreLockStillYieldsAtOnce(t *testing.T) {
	store := coordinatedLockFixture(t)
	release := holdStoreLock(t, store)
	defer release()
	observeStoreLockContention(t, func() { t.Error("maintenance polled a busy store") })
	err := store.coordinator.WithLock(context.Background(), func(held *storagegc.Locked) error {
		return store.withStoreLockChecked(func() error { return nil }, held.CheckContext, 0)
	})
	if !errors.Is(err, unix.EWOULDBLOCK) || !errors.Is(retentionLockError(err), storagegc.ErrCoordinatorBusy) {
		t.Fatalf("maintenance on a busy store: %v", err)
	}
}
