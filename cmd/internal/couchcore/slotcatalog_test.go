package couchcore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSlotIdentityFromWorkspace(t *testing.T) {
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	p := NewWorkspaceProvisioner(f)
	id, err := p.identity(context.Background(), f.host(1))
	if err != nil {
		t.Fatal(err)
	}
	got, err := SlotIdentityFromWorkspace(id)
	if err != nil || got.Number != 1 || got.EnvironmentRoot != filepath.Dir(f.host(1)) {
		t.Fatalf("%+v %v", got, err)
	}
	id.Kind = "dependency"
	if _, err := SlotIdentityFromWorkspace(id); err == nil {
		t.Fatal("accepted dependency")
	}
}

func TestSlotCatalogPreservesIncompleteAndRegisteredMissing(t *testing.T) {
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	f.git(f.Primary, "worktree", "add", "-b", "main-slot3", f.host(3), "main")
	if err := os.RemoveAll(filepath.Dir(f.host(3))); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(f.host(2)), 0700); err != nil {
		t.Fatal(err)
	}
	got, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Slots) != 3 {
		t.Fatalf("slots %+v", got.Slots)
	}
	for i, c := range got.Slots {
		if c.Identity.Number != i+1 {
			t.Fatalf("order %+v", got.Slots)
		}
		if c.Verified != (i == 0) || (c.Err == nil) != (i == 0) {
			t.Fatalf("candidate %+v", c)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(f.host(1)), ".couch")); !os.IsNotExist(err) {
		t.Fatalf("discovery wrote metadata: %v", err)
	}
	if f.WeaveCalls != 0 {
		t.Fatal("discovery compiled")
	}
}

func TestEnumerateSlotCandidatesSafetyAndBound(t *testing.T) {
	f := newProvisionFixture(t)
	if err := os.MkdirAll(f.host(2), 0700); err != nil {
		t.Fatal(err)
	}
	rows, err := EnumerateSlotCandidates(f.Primary)
	if err != nil || len(rows) != 1 || rows[0].Identity.Number != 2 {
		t.Fatalf("%+v %v", rows, err)
	}
	if err := os.Symlink(f.Primary, filepath.Join(filepath.Dir(f.host(2)), ".couch")); err != nil {
		t.Fatal(err)
	}
	if _, err := EnumerateSlotCandidates(f.Primary); err == nil {
		t.Fatal("accepted metadata symlink")
	}
	if err := os.Remove(filepath.Join(filepath.Dir(f.host(2)), ".couch")); err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 129; n++ {
		if err := os.MkdirAll(filepath.Join(filepath.Dir(f.Primary), "worktree", fmt.Sprintf("repo-name-slot%d", n)), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := EnumerateSlotCandidates(f.Primary); err == nil {
		t.Fatal("accepted excessive candidates")
	}
}

func TestSlotCatalogFakeMutableSnapshot(t *testing.T) {
	f := &SlotCatalogFake{Repositories: map[string]SlotRepository{"/repo": {Slots: []SlotCandidate{{Identity: SlotIdentity{Number: 1}}}}}}
	got, err := f.Discover(context.Background(), "/repo")
	if err != nil {
		t.Fatal(err)
	}
	got.Slots[0].Identity.Number = 9
	next, err := f.Discover(context.Background(), "/repo")
	if err != nil || next.Slots[0].Identity.Number != 1 {
		t.Fatalf("snapshot mutated fake: %+v %v", next, err)
	}
	f.Errors = map[string]error{"/repo": os.ErrPermission}
	if _, err := f.Discover(context.Background(), "/repo"); err == nil {
		t.Fatal("ignored error")
	}
}

func TestSlotCatalogRefusesForeignAndReplacedHost(t *testing.T) {
	t.Run("foreign", func(t *testing.T) {
		f := newProvisionFixture(t)
		if err := os.MkdirAll(f.host(1), 0700); err != nil {
			t.Fatal(err)
		}
		f.git(f.host(1), "init", "-b", "main-slot1")
		f.git(f.host(1), "commit", "--allow-empty", "-m", "foreign")
		got, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
		if err != nil || len(got.Slots) != 1 || got.Slots[0].Verified || got.Slots[0].Err == nil {
			t.Fatalf("%+v %v", got, err)
		}
	})
	t.Run("removed-during-probe", func(t *testing.T) {
		f := newProvisionFixture(t)
		f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
		f.AfterGit = func(c ProvisionCommand, _ []byte) error {
			if c.Dir == f.host(1) && len(c.Args) > 2 && c.Args[2] == "--git-common-dir" {
				return os.RemoveAll(f.host(1))
			}
			return nil
		}
		got, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
		if err == nil && (len(got.Slots) != 1 || got.Slots[0].Verified || got.Slots[0].Err == nil) {
			t.Fatalf("disappeared host became verified: %+v", got)
		}
	})
}

func TestEnumerateSlotCandidatesRequiresPrimary(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EnumerateSlotCandidates(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing enrolled root looked empty")
	}
}

func TestSlotCatalogCancellationDuringLastProbe(t *testing.T) {
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.AfterGit = func(c ProvisionCommand, _ []byte) error {
		if c.Dir == f.host(1) && len(c.Args) > 2 && c.Args[2] == "--git-common-dir" {
			cancel()
		}
		return nil
	}
	if _, err := NewOSSlotCatalog(f).Discover(ctx, f.Primary); err == nil {
		t.Fatal("ignored cancellation in last probe")
	}
}
