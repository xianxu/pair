package couchcore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

// contendedPreferenceFixture publishes once, so the global store and its
// store.lock exist, then shortens the foreground wait for the test.
func contendedPreferenceFixture(t *testing.T) (*ThreadStore, *ThreadStore, []byte) {
	t.Helper()
	store, _ := repoPreferenceStoreFixture(t)
	if err := store.rememberRepoConfiguration("/first/.git", LaunchProfile{Agent: "codex", Argv: []string{"first"}}, nil); err != nil {
		t.Fatal(err)
	}
	global, err := store.repoPreferencesBackend()
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(global.repoPreferencesPath())
	if err != nil {
		t.Fatal(err)
	}
	saved := storeReadLockWait
	storeReadLockWait = 30 * time.Millisecond
	t.Cleanup(func() { storeReadLockWait = saved })
	return store, global, before
}

func publishSecond(store *ThreadStore, ctx context.Context) error {
	return store.rememberRepoConfigurationContext(ctx, "/second/.git", LaunchProfile{Agent: "claude", Argv: []string{"second"}}, nil)
}

func assertPreferencesUnchanged(t *testing.T, global *ThreadStore, before []byte) {
	t.Helper()
	after, err := os.ReadFile(global.repoPreferencesPath())
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("preferences changed: %v\n%s", err, after)
	}
}

// #877: a holder that outlasts one bounded wait is contention, not refusal.
// The typed busy result retries within the caller's budget (#728's contract,
// which #827's typed wait had hidden from the loop).
func TestRepoPreferencePublicationRetriesTypedContention(t *testing.T) {
	store, global, _ := contendedPreferenceFixture(t)
	release := holdStoreLock(t, global)
	var once sync.Once
	defer once.Do(release)
	time.AfterFunc(5*storeReadLockWait, func() { once.Do(release) })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := publishSecond(store, ctx); err != nil {
		t.Fatalf("contended publication refused: %v", err)
	}
	doc, err := store.previewRepoPreferences()
	if err != nil || len(doc.Repositories) != 2 {
		t.Fatalf("other repositories not preserved: %v %+v", err, doc.Repositories)
	}
}

// A store that stays busy to the caller's deadline refuses with both causes,
// and writes nothing.
func TestRepoPreferencePublicationRefusesBusyAtTheDeadline(t *testing.T) {
	store, global, before := contendedPreferenceFixture(t)
	defer holdStoreLock(t, global)()
	ctx, cancel := context.WithTimeout(context.Background(), 8*storeReadLockWait)
	defer cancel()
	err := publishSecond(store, ctx)
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, ErrThreadStoreBusy) {
		t.Fatalf("busy to the deadline: %v", err)
	}
	assertPreferencesUnchanged(t, global, before)
}

func TestRepoPreferencePublicationStopsOnCancellation(t *testing.T) {
	store, global, before := contendedPreferenceFixture(t)
	defer holdStoreLock(t, global)()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	observeStoreLockContention(t, cancel)
	if err := publishSecond(store, ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled publication: %v", err)
	}
	assertPreferencesUnchanged(t, global, before)
}

// The retry is bounded only by the caller, so a caller without a deadline is
// refused before it can spin.
func TestRepoPreferencePublicationRefusesAnUnboundedCaller(t *testing.T) {
	store, global, before := contendedPreferenceFixture(t)
	defer holdStoreLock(t, global)()
	observeStoreLockContention(t, func() { t.Error("an unbounded caller waited on the store") })
	if err := publishSecond(store, context.Background()); !errors.Is(err, errUnboundedPublication) {
		t.Fatalf("unbounded caller: %v", err)
	}
	assertPreferencesUnchanged(t, global, before)
}
