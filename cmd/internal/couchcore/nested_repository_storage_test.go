package couchcore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func recordAtCheckout(t *testing.T, root, path, tag string) ThreadRecord {
	t.Helper()
	r := validThreadRecord(t)
	scope, err := launcher.ResolveRepoScope(root)
	if err != nil {
		t.Fatal(err)
	}
	r.Address = ThreadAddress{RepoScope: scope.Key, Tag: ThreadTag(tag)}
	r.StartingPath, r.WorkingPath = path, path
	return r
}

func TestNestedRepositoryStorageRemainsIndependentOfOuterSlot(t *testing.T) {
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	nested := filepath.Join(f.host(1), "nested")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	f.git(nested, "init")
	repo, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := newTestThreadStore(t)
	outer := recordAtCheckout(t, f.host(1), f.host(1), "outer")
	inner := recordAtCheckout(t, nested, nested, "inner")
	for _, r := range []ThreadRecord{outer, inner} {
		archived := r
		archived.Address.Tag += "-archived"
		if _, err := s.CreateThread(archived); err != nil {
			t.Fatal(err)
		}
		if err := s.ArchiveThread(archived.Address); err != nil {
			t.Fatal(err)
		}
		if _, err := s.CreateThread(r); err != nil {
			t.Fatal(err)
		}
	}
	for _, pref := range []PathLaunchPreference{
		{SchemaVersion: PathLaunchPreferenceSchemaVersion, RepoIdentity: repo.Identity.RepoIdentity, PhysicalPath: outer.StartingPath, LastAgent: "codex", ArgvByAgent: map[string][]string{"codex": {"--outer"}}, Revision: 1},
		{SchemaVersion: PathLaunchPreferenceSchemaVersion, RepoIdentity: filepath.Join(nested, ".git"), PhysicalPath: inner.StartingPath, LastAgent: "claude", ArgvByAgent: map[string][]string{"claude": {"--inner"}}, Revision: 1},
	} {
		if err := writePathLaunchPreferenceForTest(s, pref); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.EnrollSlotRepository(context.Background(), repo); err != nil {
		t.Fatalf("independent nested repository blocked outer enrollment: %v", err)
	}
	outerStore, err := s.storeForAddress(outer.Address)
	if err != nil || outerStore == s {
		t.Fatalf("outer not local: %v", err)
	}
	innerStore, err := s.storeForAddress(inner.Address)
	if err != nil || innerStore != s {
		t.Fatalf("inner stole outer slot: %v", err)
	}
	for _, r := range []ThreadRecord{outer, inner} {
		if _, err := s.GetThread(r.Address); err != nil {
			t.Fatal(err)
		}
		archived := r.Address
		archived.Tag += "-archived"
		backend, err := s.storeForAddress(archived)
		if err != nil {
			t.Fatal(err)
		}
		if (backend == s) != (r.Address == inner.Address) {
			t.Fatal("archive crossed repository boundary")
		}
	}
	archivedRecords, err := s.ArchivedThreads()
	if err != nil || len(archivedRecords) != 2 {
		t.Fatalf("archive inventory hid nested repository: %+v %v", archivedRecords, err)
	}
	if _, err := outerStore.CreateThread(inner); err == nil {
		t.Fatal("local origin accepted nested repository scope")
	}
	for _, item := range []struct{ identity, path, agent string }{{repo.Identity.RepoIdentity, outer.StartingPath, "codex"}, {filepath.Join(nested, ".git"), inner.StartingPath, "claude"}} {
		pref, found, err := s.GetPathLaunchPreference(item.identity, item.path)
		if err != nil || !found || pref.LastAgent != item.agent {
			t.Fatalf("preference crossed repository boundary: %+v %v %v", pref, found, err)
		}
	}
	snap, err := s.Snapshot()
	if err != nil || len(snap.Records) != 2 {
		t.Fatalf("snapshot hid repository: %+v %v", snap, err)
	}
	rows := ProjectActionableThreads(ThreadProjectionInput{Records: snap.Records, Slots: snap.Slots})
	if len(rows) != 2 {
		t.Fatalf("inventory hid repository: %+v", rows)
	}
	for _, row := range rows {
		if row.Address == inner.Address && row.Target.Kind != ThreadTargetOrdinary {
			t.Fatalf("nested row adopted outer slot: %+v", row)
		}
	}
	fresh := recordAtCheckout(t, nested, nested, "second-inner")
	if _, err := s.CreateThread(fresh); err != nil {
		t.Fatalf("nested create routed into occupied outer slot: %v", err)
	}
}
