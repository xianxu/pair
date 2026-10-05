package couchcore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/storagegc"
	"golang.org/x/sys/unix"
)

// holdStoreLock takes store.lock from another descriptor, as a concurrent
// writer (inventory refresh, slot probe, activity write) does.
func holdStoreLock(t *testing.T, s *ThreadStore) func() {
	t.Helper()
	file, err := os.OpenFile(filepath.Join(s.root, "store.lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	return func() { _ = unix.Flock(int(file.Fd()), unix.LOCK_UN); _ = file.Close() }
}

func storeReadLockFixture(t *testing.T) *testEnv {
	t.Helper()
	env := newTestEnv(t, "/repo")
	// A write creates the store and its lock file.
	if _, err := env.Couch.Threads.CreateThread(validThreadRecord(t)); err != nil {
		t.Fatal(err)
	}
	return env
}

// A store read waits briefly for a lock that is only ever held briefly: the
// smoke test's reboot failed once with "read launch preference: resource
// temporarily unavailable" while something held store.lock for a moment.
func TestStoreReadWaitsForABrieflyHeldLock(t *testing.T) {
	env := storeReadLockFixture(t)
	release := holdStoreLock(t, env.Couch.Threads)
	go func() { time.Sleep(50 * time.Millisecond); release() }()
	if _, _, err := env.Couch.Threads.PreviewPathLaunchPreference("/repo/.git", "/repo", "scope"); err != nil {
		t.Fatalf("preference read while the store was briefly busy: %v", err)
	}
}

// Past the bound the read refuses with a typed, worded error, never a raw
// errno.
func TestStoreReadRefusesTypedWhenTheStoreStaysBusy(t *testing.T) {
	env := storeReadLockFixture(t)
	release := holdStoreLock(t, env.Couch.Threads)
	defer release()
	defer func(saved time.Duration) { storeReadLockWait = saved }(storeReadLockWait)
	storeReadLockWait = 60 * time.Millisecond
	start := time.Now()
	_, _, err := env.Couch.Threads.PreviewPathLaunchPreference("/repo/.git", "/repo", "scope")
	if !errors.Is(err, ErrThreadStoreBusy) || strings.Contains(err.Error(), "resource temporarily unavailable") {
		t.Fatalf("busy store: %v", err)
	}
	if waited := time.Since(start); waited < storeReadLockWait {
		t.Fatalf("refused after %s, before the %s bound", waited, storeReadLockWait)
	}
	// Retention callers still yield through the maintenance contract.
	if !errors.Is(retentionLockError(err), storagegc.ErrCoordinatorBusy) {
		t.Fatalf("retention mapping of %v lost the busy yield", err)
	}
}
