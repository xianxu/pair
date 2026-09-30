package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSeparateGitDirectoryEnrollmentPreservesStorageAuthority(t *testing.T) {
	f := newProvisionFixture(t)
	common := filepath.Join(filepath.Dir(f.Primary), "external-git")
	f.git(f.Primary, "init", "--separate-git-dir", common)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	repository, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	if repository.Identity.RepoIdentity != common {
		t.Fatalf("catalog did not preserve common Git directory: %+v", repository.Identity)
	}
	s, _ := newTestThreadStore(t)
	record := recordAtCheckout(t, f.host(1), f.host(1), "separate")
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{PID: 123, Identity: "saved-owner", State: IncarnationLive, RepoIdentity: common, LaunchProfile: record.LatestLaunchProfile}}
	if _, err := s.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	preference := PathLaunchPreference{SchemaVersion: PathLaunchPreferenceSchemaVersion, RepoIdentity: common, PhysicalPath: record.StartingPath, LastAgent: "claude", ArgvByAgent: map[string][]string{"claude": {"--saved"}}, Revision: 1}
	if err := writePathLaunchPreferenceForTest(s, preference); err != nil {
		t.Fatal(err)
	}
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatalf("enrollment: %v", err)
	}
	for _, store := range []*ThreadStore{s, NewThreadStore(s.namespace)} {
		got, err := store.GetThread(record.Address)
		if err != nil || got.Address != record.Address {
			t.Fatalf("enrolled record lost: %+v %v", got, err)
		}
		local, err := store.storeForPath(record.StartingPath, "", common)
		if err != nil || local == store || local.slot.RepoIdentity != common {
			t.Fatalf("path routing invented common identity: %+v %v", local, err)
		}
		pref, found, err := store.GetPathLaunchPreference(common, record.StartingPath)
		if err != nil || !found || pref.RepoIdentity != common {
			t.Fatalf("preference lost: %+v %v %v", pref, found, err)
		}
		snapshot, err := store.Snapshot()
		if err != nil || len(snapshot.Records) != 1 || len(snapshot.Slots) != 1 || snapshot.Slots[0].Identity.RepoIdentity != common {
			t.Fatalf("inventory lost authority: %+v %v", snapshot, err)
		}
	}
	// Simulate a pre-upgrade root listing: retained checkout scope remains usable
	// without inventing a common-directory identity, then catalog enrollment repairs it.
	forgetIdentity := func() {
		t.Helper()
		if err := s.withLock(func() error {
			manifest, _, _, err := s.loadManifestLocked()
			if err != nil {
				return err
			}
			manifest.SlotRepositoryIdentities = nil
			raw, err := json.Marshal(manifest)
			if err != nil {
				return err
			}
			return os.WriteFile(s.manifestPath(), raw, 0600)
		}); err != nil {
			t.Fatal(err)
		}
	}
	forgetIdentity()
	reopened := NewThreadStore(s.namespace)
	if _, err := reopened.GetThread(record.Address); err != nil {
		t.Fatalf("legacy scope readback lost: %v", err)
	}
	legacy, err := reopened.Snapshot()
	if err != nil || len(legacy.Records) != 1 || len(legacy.Slots) != 1 {
		t.Fatalf("legacy inventory lost: %+v %v", legacy, err)
	}
	unknown := legacy.Slots[0].Identity
	if unknown.RepoIdentity != "" {
		t.Fatalf("invented common directory: %+v", unknown)
	}
	if unknown.Validate() == nil {
		t.Fatal("passive legacy location became action authority")
	}
	if err := (ThreadTarget{Kind: ThreadTargetSlot, Slot: unknown}).Validate(); err != nil {
		t.Fatalf("legacy location vanished from UI: %v", err)
	}
	if _, _, err := reopened.GetPathLaunchPreference(common, record.StartingPath); err == nil || !strings.Contains(err.Error(), "verify its enrollment") {
		t.Fatalf("unknown identity used for preference routing: %v", err)
	}
	env := newTestEnv(t, record.StartingPath)
	env.Couch.Threads = reopened
	env.Couch.Namespace = s.namespace
	env.Couch.Slots = NewOSSlotCatalog(f)
	env.Couch.RepoAgentDefault = func(root, agent string) (LaunchProfile, bool, error) {
		if root != f.Primary {
			t.Fatalf("default lookup lost primary: %s", root)
		}
		return LaunchProfile{}, false, nil
	}
	env.Git.replies[GitCall{Dir: record.StartingPath, Args: "rev-parse --git-common-dir"}] = common
	env.Artifacts.SetPairSession(record.Address, "pair-separate-git", true)
	env.Artifacts.SetDetachedSession(record.Address, "pair-separate-git")
	beforePreview, err := os.ReadFile(s.manifestPath())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.PrepareStart(context.Background(), StartArgs{Action: StartOpen, Worktree: Worktree(record.StartingPath)}); err != nil {
		t.Fatalf("legacy identity repair preview deadlocked: %v", err)
	}
	afterPreview, err := os.ReadFile(s.manifestPath())
	if err != nil || !bytes.Equal(beforePreview, afterPreview) {
		t.Fatalf("preview wrote identity authority: %v", err)
	}
	if _, err := env.Couch.OpenSlot(context.Background(), record.StartingPath, "claude"); err != nil {
		t.Fatalf("legacy identity repair open deadlocked: %v", err)
	}
	if _, found, err := reopened.GetPathLaunchPreference(common, record.StartingPath); err != nil || !found {
		t.Fatalf("verified backfill did not restore preferences: %v", err)
	}
	if _, err := reopened.ReserveRepositoryFamily(context.Background(), repository, RepositoryFamily{RepoIdentity: common, PrimaryRoot: f.Primary, RelativeStart: "."}); err != nil {
		t.Fatal(err)
	}
	forgetIdentity()
	if _, found, err := NewThreadStore(s.namespace).GetPathLaunchPreference(common, record.StartingPath); err != nil || !found {
		t.Fatalf("saved family authority was ignored: %v", err)
	}

}

func TestLegacyUnknownCommonDirectoryUsesOnlyExactCheckoutScope(t *testing.T) {
	record := recordAtCheckout(t, "/primary/worktree/repo-slot1/repo", "/primary/worktree/repo-slot1/repo/sub", "retained")
	record.Incarnations = []ThreadIncarnation{{RepoIdentity: "/separate/common"}}
	if _, belongs, err := RecordCheckoutMembership(record, "", "/primary/worktree/repo-slot1/repo"); err != nil || !belongs {
		t.Fatalf("exact scope lost: %v %v", belongs, err)
	}
	if _, belongs, err := CheckoutMembership("", "/primary/worktree/repo-slot1/repo", record.StartingPath, "", "/separate/common"); err != nil || belongs {
		t.Fatalf("unknown expected common directory became authority: %v %v", belongs, err)
	}
	nested := recordAtCheckout(t, "/primary/worktree/repo-slot1/repo/sub", record.StartingPath, "nested")
	if _, belongs, err := RecordCheckoutMembership(nested, "", "/primary/worktree/repo-slot1/repo"); err != nil || belongs {
		t.Fatalf("nested scope adopted by unknown enrollment: %v %v", belongs, err)
	}
}
