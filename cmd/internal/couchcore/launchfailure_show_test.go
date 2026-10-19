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
		if err := store.Save(t.Context(), LaunchFailure{Address: ThreadAddress{RepoScope: scope, Tag: "removed"}, Nonce: "attempt", FailedAt: time.Now(), LaunchError: "exec refused"}); err != nil {
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
	if err := store.Save(t.Context(), LaunchFailure{Address: ThreadAddress{RepoScope: "different-current-scope", Tag: "gone"}, Nonce: "attempt", FailedAt: time.Now(), StartingPath: real}); err != nil {
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
	if err := store.Save(t.Context(), LaunchFailure{Address: ThreadAddress{RepoScope: "slot-scope", Tag: "rolled-back"}, Nonce: "attempt", StartingPath: s.layout.Host(), FailedAt: time.Now()}); err != nil {
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

func TestShowLifecycleFailureLatestUnresolvedAndEmptySlot(t *testing.T) {
	env := newTestEnv(t, "/repo")
	store := NewFakeLaunchFailureStore()
	env.Couch.LaunchFailures = store
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	at := time.Unix(100, 0)
	for i, nonce := range []string{"old", "latest"} {
		f := LaunchFailure{StartingPath: root, Slot: "pair:8", Operation: "recover-park", Nonce: nonce, FailedAt: at.Add(time.Duration(i) * time.Second)}
		if err := store.Save(t.Context(), f); err != nil {
			t.Fatal(err)
		}
	}
	got, detail := env.Couch.launchFailuresForShow(t.Context(), "scope", root, nil, SlotIdentity{})
	if detail != "" || len(got) != 1 || got[0].Nonce != "latest" {
		t.Fatalf("show=%+v detail=%s", got, detail)
	}
}

func TestShowLifecycleFailureUsesRecordedSuccessOnly(t *testing.T) {
	at := time.Unix(100, 0)
	address := ThreadAddress{RepoScope: "scope", Tag: "tag"}
	for _, tc := range []struct {
		name   string
		record ThreadRecord
		hidden bool
	}{
		{"retire after failure", ThreadRecord{LastActiveAt: at.Add(time.Second)}, false},
		{"verified park", ThreadRecord{VerifiedPark: &VerifiedPark{ParkedAt: at.Add(time.Second)}}, true},
		{"committed live", ThreadRecord{Incarnations: []ThreadIncarnation{{State: IncarnationLive, StartedAt: at.Add(time.Second)}}}, true},
		{"equal time", ThreadRecord{VerifiedPark: &VerifiedPark{ParkedAt: at}}, false},
		{"creating", ThreadRecord{Incarnations: []ThreadIncarnation{{State: IncarnationCreating, StartedAt: at.Add(time.Second)}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			store := NewFakeLaunchFailureStore()
			env.Couch.LaunchFailures = store
			if err := store.Save(t.Context(), LaunchFailure{Address: address, Nonce: "failure", FailedAt: at}); err != nil {
				t.Fatal(err)
			}
			tc.record.Address = address
			got, detail := env.Couch.launchFailuresForShow(t.Context(), "scope", "tag", []ThreadRecord{tc.record}, SlotIdentity{})
			if detail != "" || (len(got) == 0) != tc.hidden {
				t.Fatalf("failures=%+v detail=%s hidden=%v", got, detail, tc.hidden)
			}
		})
	}
}

func TestShowLifecycleFailureExactUnresolvedTarget(t *testing.T) {
	env := newTestEnv(t, "/repo")
	store := NewFakeLaunchFailureStore()
	env.Couch.LaunchFailures = store
	if err := store.Save(t.Context(), LaunchFailure{Target: "pair:8", Nonce: "failure", FailedAt: time.Unix(100, 0)}); err != nil {
		t.Fatal(err)
	}
	got, detail := env.Couch.launchFailuresForShow(t.Context(), "scope", "pair:8", nil, SlotIdentity{})
	if detail != "" || len(got) != 1 {
		t.Fatalf("failures=%+v detail=%s", got, detail)
	}
	absent, _ := env.Couch.launchFailuresForShow(t.Context(), "scope", "pair:9", nil, SlotIdentity{})
	if len(absent) != 0 {
		t.Fatalf("cross-slot diagnostics=%+v", absent)
	}
}

func TestShowLifecycleFailureNewestAcrossExactAndSlotIdentity(t *testing.T) {
	env := newTestEnv(t, "/repo")
	store := NewFakeLaunchFailureStore()
	env.Couch.LaunchFailures = store
	address := ThreadAddress{RepoScope: "scope", Tag: "tag"}
	at := time.Unix(100, 0)
	for _, f := range []LaunchFailure{{Address: address, StartingPath: "/a", Nonce: "old slot", FailedAt: at}, {Address: address, Nonce: "new exact", FailedAt: at.Add(time.Second)}} {
		if err := store.Save(t.Context(), f); err != nil {
			t.Fatal(err)
		}
	}
	got, detail := env.Couch.launchFailuresForShow(t.Context(), "scope", "tag", []ThreadRecord{{Address: address, StartingPath: "/a"}}, SlotIdentity{})
	if detail != "" || len(got) != 1 || got[0].Nonce != "new exact" {
		t.Fatalf("failures=%+v detail=%s", got, detail)
	}
}

func TestShowTargetOnlyLifecycleFailureUsesResolvedSlotSelection(t *testing.T) {
	at := time.Unix(100, 0)
	slot := SlotIdentity{Repo: "pair", PrimaryRoot: "/workspace/pair", WorktreeRoot: "/workspace/worktree/pair-slot8/pair", Number: 8}
	address := ThreadAddress{RepoScope: "scope", Tag: "tag"}
	for _, target := range []string{"pair:8", "/workspace/pair:8"} {
		for _, success := range []time.Time{{}, at, at.Add(time.Second)} {
			for _, ref := range []string{"pair:8", "/workspace/pair:8", slot.WorktreeRoot, "tag"} {
				t.Run(target+"/"+ref+"/"+success.String(), func(t *testing.T) {
					env := newTestEnv(t, "/repo")
					store := NewFakeLaunchFailureStore()
					env.Couch.LaunchFailures = store
					if err := store.Save(t.Context(), LaunchFailure{Target: target, Nonce: "failure", FailedAt: at}); err != nil {
						t.Fatal(err)
					}
					record := ThreadRecord{Address: address, StartingPath: slot.WorktreeRoot}
					if !success.IsZero() {
						record.VerifiedPark = &VerifiedPark{ParkedAt: success}
					}
					got, detail := env.Couch.launchFailuresForShow(t.Context(), "scope", ref, []ThreadRecord{record}, slot)
					row := ActionableThreadSummary{Address: address, StartingPath: slot.WorktreeRoot, Target: ThreadTarget{Kind: ThreadTargetSlot, Slot: slot}, LastLifecycleSuccessAt: LastLifecycleSuccessAt(record)}
					projected := ProjectLifecycleFailures([]ActionableThreadSummary{row}, []LaunchFailure{{Target: target, Nonce: "failure", FailedAt: at}}, "")
					hidden := success.After(at)
					if detail != "" || (len(got) == 0) != hidden || (projected[0].Failure == nil) != hidden {
						t.Fatalf("show=%+v row=%+v hidden=%v detail=%s", got, projected[0].Failure, hidden, detail)
					}
				})
			}
		}
	}
}

func TestShowTargetOnlyLifecycleFailureResolvedEmptySlot(t *testing.T) {
	env := newTestEnv(t, "/repo")
	store := NewFakeLaunchFailureStore()
	env.Couch.LaunchFailures = store
	slot := SlotIdentity{Repo: "pair", PrimaryRoot: "/workspace/pair", WorktreeRoot: "/workspace/worktree/pair-slot8/pair", Number: 8}
	if err := store.Save(t.Context(), LaunchFailure{Target: "pair:8", Nonce: "failure", FailedAt: time.Unix(100, 0)}); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"/workspace/pair:8", slot.WorktreeRoot} {
		got, detail := env.Couch.launchFailuresForShow(t.Context(), "scope", ref, nil, slot)
		if detail != "" || len(got) != 1 {
			t.Fatalf("ref=%s got=%+v detail=%s", ref, got, detail)
		}
	}
	other := slot
	other.Number = 9
	other.WorktreeRoot = "/workspace/worktree/pair-slot9/pair"
	got, detail := env.Couch.launchFailuresForShow(t.Context(), "scope", other.WorktreeRoot, nil, other)
	if detail != "" || len(got) != 0 {
		t.Fatalf("cross-slot=%+v detail=%s", got, detail)
	}
}
