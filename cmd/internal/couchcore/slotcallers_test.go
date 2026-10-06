package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestDiscoverUnionsNumberSources: slot numbers come from environment
// directories, worktree registrations and resting branches. A number known
// only from leftovers is a candidate marked ErrSlotNeedsReconcile, never
// missing and never a repository-wide failure.
func TestDiscoverUnionsNumberSources(t *testing.T) {
	f := newProvisionFixture(t)
	p := NewWorkspaceProvisioner(f)
	for _, n := range []int{1, 2} {
		if _, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: n}); err != nil {
			t.Fatal(err)
		}
	}
	// Slot 1: its directory deleted, the registration left behind.
	os.RemoveAll(filepath.Dir(f.host(1)))
	// Slot 3: only a resting branch.
	f.git(f.Primary, "branch", "main-slot3", f.Base)
	repo, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatalf("one broken slot failed the repository's inventory: %v", err)
	}
	got := map[int]SlotCandidate{}
	for _, c := range repo.Slots {
		got[c.Identity.Number] = c
	}
	if c := got[1]; !errors.Is(c.Err, ErrSlotNeedsReconcile) {
		t.Errorf("slot 1 (registration only) = %+v, want needs-reconcile", c)
	}
	if c := got[2]; !c.Verified || c.Err != nil {
		t.Errorf("slot 2 (healthy) = %+v", c)
	}
	if c := got[3]; !errors.Is(c.Err, ErrSlotNeedsReconcile) {
		t.Errorf("slot 3 (branch only) = %+v, want needs-reconcile", c)
	}
}

// TestOpenReconcilesAVerifiedSlotWhoseSetupIsMissing: verifyHost proves only
// identity, so an open of a verified host whose marker is gone must still
// converge setup before the agent starts (red before pair#387 Task 2.5).
func TestOpenReconcilesAVerifiedSlotWhoseSetupIsMissing(t *testing.T) {
	f := newProvisionFixture(t)
	if _, err := NewWorkspaceProvisioner(f).Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1}); err != nil {
		t.Fatal(err)
	}
	admin := f.git(f.host(1), "rev-parse", "--absolute-git-dir")
	os.Remove(SetupMarkerPath(admin))
	calls := f.WeaveCalls
	env := newTestEnv(t, f.Primary)
	env.Couch.Git = ExecGit{}
	env.Couch.Path = OSPathOps{}
	env.Couch.Slots = NewOSSlotCatalog(f)
	env.Couch.Workspaces = NewWorkspaceProvisioner(f)
	if _, _, err := env.Couch.selectedSlot(context.Background(), f.host(1)); err != nil {
		t.Fatal(err)
	}
	if f.WeaveCalls != calls+1 {
		t.Fatalf("weave calls %d, want the missing setup compiled on open", f.WeaveCalls-calls)
	}
	if _, err := os.Stat(SetupMarkerPath(admin)); err != nil {
		t.Fatalf("marker not restored: %v", err)
	}
}

// TestOpenRefusesWithTheHandoffWhenSetupCannotConverge: the tools:1 shape (a
// dependency with no clone source) refuses with the :0 hand-off and starts no
// agent.
func TestOpenRefusesWithTheHandoffWhenSetupCannotConverge(t *testing.T) {
	f := newProvisionFixture(t)
	if _, err := NewWorkspaceProvisioner(f).Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(f.host(1), "construct", "deps"), "substrate ../ariadne\n")
	os.Remove(SetupMarkerPath(f.git(f.host(1), "rev-parse", "--absolute-git-dir")))
	env := newTestEnv(t, f.Primary)
	env.Couch.Git = ExecGit{}
	env.Couch.Path = OSPathOps{}
	env.Couch.Slots = NewOSSlotCatalog(f)
	env.Couch.Workspaces = NewWorkspaceProvisioner(f)
	_, _, err := env.Couch.selectedSlot(context.Background(), f.host(1))
	var blocked *SlotReconcileError
	if !errors.As(err, &blocked) || blocked.Failure.Class != FailureHandoff || blocked.Failure.Resource != ResourceSetup {
		t.Fatalf("err = %v, want the setup hand-off", err)
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("an agent started: %v", env.Runner.Ops)
	}
}
