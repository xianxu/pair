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

// TestRebootGetsPastAHoldItsAdviceNames (BR-10): the hold advice for a slot
// whose repair waits on a live agent says "reboot the slot to repair it", so
// reboot's first pass must get past exactly that hold, stop the agent, and let
// its post-stop pass repair. A hand-off still refuses before anything stops.
func TestRebootGetsPastAHoldItsAdviceNames(t *testing.T) {
	for _, tc := range []struct {
		name     string
		first    ReconcileFailure
		proceeds bool
	}{
		{"live-agent hold", ReconcileFailure{Resource: ResourceHost, Class: FailureHold, Cause: StopReasonAgentLive}, true},
		{"hand-off", ReconcileFailure{Resource: ResourceSetup, Class: FailureHandoff, Cause: "Error: missing substrate"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, local := slotRecoveryOperationFixture(t)
			old := slotRecordFixture(t, env, local)
			env.Artifacts.SetPairSession(old.Address, "pair-slot-detached", true)
			env.Artifacts.SetDetachedSession(old.Address, "pair-slot-detached")
			calls := 0
			env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
				calls++
				if calls == 1 {
					return ProvisionResult{}, &SlotReconcileError{Address: "repo-name:1", Repo: "repo-name", Failure: tc.first}
				}
				return slotReadyResult(local), nil
			})
			result, err := dispatchReboot(env, map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": old.Address.RepoScope})
			if !tc.proceeds {
				if err == nil || len(env.Artifacts.Quiesces()) != 0 {
					t.Fatalf("a hand-off must refuse with nothing stopped: err=%v quiesced=%v", err, env.Artifacts.Quiesces())
				}
				return
			}
			if err != nil {
				t.Fatalf("reboot refused the hold its own advice names: %v", err)
			}
			if got := env.Artifacts.Quiesces(); len(got) != 1 || got[0] != old.Address {
				t.Fatalf("quiesced %v, want the slot's agent stopped", got)
			}
			if _, ok := result.Started(); !ok || calls < 2 {
				t.Fatalf("result %+v, reconcile calls %d: want the post-stop pass and a fresh start", result, calls)
			}
		})
	}
}
