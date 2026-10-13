package couchcore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShowLaunchFailureWithoutThread(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t, "/repo")
	store := NewFakeLaunchFailureStore()
	env.Couch.LaunchFailures = store
	for _, scope := range []string{"scope", "other"} {
		if err := store.Save(LaunchFailure{Address: ThreadAddress{RepoScope: scope, Tag: "removed"}, Nonce: "attempt", FailedAt: time.Now(), LaunchError: "exec refused"}); err != nil {
			t.Fatal(err)
		}
	}
	result, err := dispatchTestOperation(env.Couch, "show", map[string]string{"ref": "removed", "repo-scope": "scope"})
	if err != nil {
		t.Fatal(err)
	}
	show := result.(ShowResult)
	if len(show.Threads) != 0 || len(show.LaunchFailures) != 1 || show.LaunchFailures[0].Address.RepoScope != "scope" {
		t.Fatalf("show = %+v", show)
	}
	if _, err := dispatchTestOperation(env.Couch, "show", map[string]string{"ref": "remove", "repo-scope": "scope"}); !errors.Is(err, ErrThreadReferenceNotFound) {
		t.Fatalf("fuzzy diagnostics lookup: %v", err)
	}
}

func TestShowLaunchFailureCanonicalPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Fatal(err)
	}
	env := newTestEnv(t, "/repo")
	store := NewFakeLaunchFailureStore()
	env.Couch.LaunchFailures = store
	if err := store.Save(LaunchFailure{Address: ThreadAddress{RepoScope: "different-current-scope", Tag: "gone"}, Nonce: "attempt", FailedAt: time.Now(), StartingPath: real}); err != nil {
		t.Fatal(err)
	}
	result, err := dispatchTestOperation(env.Couch, "show", map[string]string{"ref": alias, "repo-scope": "scope"})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.(ShowResult); len(got.Threads) != 0 || len(got.LaunchFailures) != 1 {
		t.Fatalf("show = %+v", got)
	}
}

func TestShowLaunchFailureReadErrorIsVisible(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t, "/repo")
	store := NewFakeLaunchFailureStore()
	store.ReadError = errors.New("corrupt diagnostics")
	env.Couch.LaunchFailures = store
	result, err := dispatchTestOperation(env.Couch, "show", map[string]string{"ref": "removed", "repo-scope": "scope"})
	if err != nil {
		t.Fatal(err)
	}
	got := result.(ShowResult)
	if len(got.Threads) != 0 || len(got.LaunchFailures) != 0 || !strings.Contains(got.LaunchFailureError, "corrupt diagnostics") {
		t.Fatalf("show = %+v", got)
	}
}

func TestShowLaunchFailureSlotAfterRollback(t *testing.T) {
	t.Parallel()
	s := newObservedSlot(t)
	env := newTestEnv(t, "/repo")
	env.Couch.SlotIO = s.f
	env.Couch.Slots = NewOSSlotCatalog(s.f)
	store := NewFakeLaunchFailureStore()
	env.Couch.LaunchFailures = store
	if err := store.Save(LaunchFailure{Address: ThreadAddress{RepoScope: "slot-scope", Tag: "rolled-back"}, Nonce: "attempt", StartingPath: s.layout.Host(), FailedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(s.layout.Env()); err != nil {
		t.Fatal(err)
	}
	result, err := dispatchTestOperation(env.Couch, "show", map[string]string{"ref": s.layout.Host(), "repo-scope": "scope"})
	if err != nil {
		t.Fatal(err)
	}
	got := result.(ShowResult)
	if got.Slot == nil || len(got.Threads) != 0 || len(got.LaunchFailures) != 1 {
		t.Fatalf("show = %+v", got)
	}
}
