package couchcore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// converger is a slot converger over the observed slot's fixture, holding the
// host creation lease as the reconcile loop does.
func (s *observedSlot) converger(t *testing.T) *slotConverger {
	t.Helper()
	lease, err := AcquireHostCreationLease(s.layout.common)
	if err != nil {
		t.Fatal(err)
	}
	cv := &slotConverger{p: NewWorkspaceProvisioner(s.f), layout: s.layout, lease: lease}
	t.Cleanup(func() { cv.lease.Close() })
	return cv
}

// convergeTwice runs a step twice: the second run must change nothing.
func convergeTwice(t *testing.T, s *observedSlot, step PlannedStep) {
	t.Helper()
	cv := s.converger(t)
	defer cv.lease.Close()
	if err := cv.converge(context.Background(), step); err != nil {
		t.Fatalf("%s: %v", step, err)
	}
	before := s.observe(t)
	if err := cv.converge(context.Background(), step); err != nil {
		t.Fatalf("%s repeated: %v", step, err)
	}
	if after := s.observe(t); !after.Equal(before) {
		t.Fatalf("%s repeated changed the slot:\nbefore %+v\nafter  %+v", step, before.Resources, after.Resources)
	}
}

func resource(t *testing.T, s *observedSlot, id SlotResourceID) ResourceObservation {
	t.Helper()
	r, _ := s.observe(t).Get(id)
	return r
}

func TestConvergeMkdirEnv(t *testing.T) {
	s := newObservedSlot(t)
	os.RemoveAll(s.layout.Env())
	convergeTwice(t, s, PlannedStep{Step: StepMkdirEnv, Resource: ResourceEnv})
	if r := resource(t, s, ResourceEnv); r.State != StatePresent {
		t.Fatalf("env %+v", r)
	}
}

func TestConvergeRemoveIntent(t *testing.T) {
	s := newObservedSlot(t)
	write(t, s.layout.Intent(), "{}")
	convergeTwice(t, s, PlannedStep{Step: StepRemoveIntent, Resource: ResourceIntent})
	if r := resource(t, s, ResourceIntent); r.State != StateAbsent {
		t.Fatalf("intent %+v", r)
	}
}

func TestConvergeCreateBranch(t *testing.T) {
	s := newObservedSlot(t)
	s.f.git(s.layout.Host(), "switch", "-q", "-c", "issue-work")
	s.f.git(s.f.Primary, "branch", "-D", s.layout.RestingBranch())
	convergeTwice(t, s, PlannedStep{Step: StepCreateBranch, Resource: ResourceBranch})
	if got := s.f.git(s.f.Primary, "rev-parse", s.layout.RestingRef()); got != s.f.Base {
		t.Fatalf("main-slot1 at %s, want remote main %s", got, s.f.Base)
	}
}

func TestConvergeCreateBranchAdoptsAConcurrentCreator(t *testing.T) {
	s := newObservedSlot(t)
	s.f.git(s.layout.Host(), "switch", "-q", "-c", "issue-work")
	s.f.git(s.f.Primary, "branch", "-D", s.layout.RestingBranch())
	other := s.f.git(s.f.Primary, "commit-tree", "-m", "someone else's", s.f.git(s.f.Primary, "rev-parse", "HEAD^{tree}"))
	s.f.AfterGit = func(c ProvisionCommand, _ []byte) error {
		if len(c.Args) > 0 && c.Args[0] == "fetch" {
			s.f.AfterGit = nil
			s.f.git(s.f.Primary, "update-ref", s.layout.RestingRef(), other)
		}
		return nil
	}
	if err := s.converger(t).converge(context.Background(), PlannedStep{Step: StepCreateBranch, Resource: ResourceBranch}); err != nil {
		t.Fatalf("a concurrently created branch must be adopted, not an error: %v", err)
	}
	if got := s.f.git(s.f.Primary, "rev-parse", s.layout.RestingRef()); got != other {
		t.Fatalf("main-slot1 at %s; the zero-OID compare and swap must not overwrite the creator's %s", got, other)
	}
}

func TestConvergeSetUpstream(t *testing.T) {
	s := newObservedSlot(t)
	rest := s.layout.RestingBranch()
	s.f.git(s.f.Primary, "config", "--unset", "branch."+rest+".merge")
	convergeTwice(t, s, PlannedStep{Step: StepSetUpstream, Resource: ResourceUpstream})
	if r := resource(t, s, ResourceUpstream); r.State != StatePresent {
		t.Fatalf("upstream %+v", r)
	}
	if got := s.f.git(s.f.Primary, "config", "branch."+rest+".remote"); got != "upstream" {
		t.Fatalf("existing remote overwritten: %s", got)
	}
}

func TestConvergeRepairHost(t *testing.T) {
	s := newObservedSlot(t)
	admin := s.admin(t)
	write(t, filepath.Join(s.layout.Host(), ".git"), "gitdir: /nonexistent/admin\n")
	convergeTwice(t, s, PlannedStep{Step: StepRepairHost, Resource: ResourceHost})
	if r := resource(t, s, ResourceHost); r.State != StatePresent {
		t.Fatalf("host %+v after repair (admin %s)", r, admin)
	}
}

// TestConvergeWorktreeAddOverAStaleRegistration: the host of a slot whose
// directory was deleted comes back on the branch its registration names, and
// another slot's stale registration is left alone (no repository-wide prune).
func TestConvergeWorktreeAddOverAStaleRegistration(t *testing.T) {
	s := newObservedSlot(t)
	if _, err := NewWorkspaceProvisioner(s.f).Ensure(context.Background(), ProvisionRequest{Path: s.f.Primary, Slot: 2}); err != nil {
		t.Fatal(err)
	}
	slot2 := NewSlotLayout(s.f.Primary, s.layout.common, 2)
	s.f.git(s.layout.Host(), "switch", "-q", "-c", "issue-work")
	write(t, filepath.Join(s.layout.Host(), "dirty.txt"), "lost with the directory")
	os.RemoveAll(s.layout.Env())
	os.RemoveAll(slot2.Env())
	convergeTwice(t, s, PlannedStep{Step: StepMkdirEnv, Resource: ResourceEnv})
	convergeTwice(t, s, PlannedStep{Step: StepWorktreeAdd, Resource: ResourceHost, Branch: "issue-work", Force: true})
	if got := s.f.git(s.layout.Host(), "branch", "--show-current"); got != "issue-work" {
		t.Fatalf("re-added on %q, want the registration's issue-work", got)
	}
	var slot2Stale bool
	for _, e := range parseWorktreeList(s.f.git(s.f.Primary, "worktree", "list", "--porcelain", "-z")) {
		if filepath.Clean(e.Path) == slot2.Host() && e.Prunable {
			slot2Stale = true
		}
	}
	if !slot2Stale {
		t.Fatal("slot 2's stale registration was removed; only this slot's may be touched")
	}
}

func TestConvergeWorktreeAddFallsBackWhenTheBranchIsElsewhere(t *testing.T) {
	s := newObservedSlot(t)
	s.f.git(s.layout.Host(), "switch", "-q", "-c", "issue-work")
	os.RemoveAll(s.layout.Env())
	elsewhere := filepath.Join(t.TempDir(), "elsewhere")
	s.f.git(s.f.Primary, "worktree", "prune")
	s.f.git(s.f.Primary, "worktree", "add", "-q", elsewhere, "issue-work")
	cv := s.converger(t)
	if err := cv.converge(context.Background(), PlannedStep{Step: StepMkdirEnv, Resource: ResourceEnv}); err != nil {
		t.Fatal(err)
	}
	if err := cv.converge(context.Background(), PlannedStep{Step: StepWorktreeAdd, Resource: ResourceHost, Branch: "issue-work"}); err != nil {
		t.Fatal(err)
	}
	if got := s.f.git(s.layout.Host(), "branch", "--show-current"); got != s.layout.RestingBranch() {
		t.Fatalf("re-added on %q, want %s: issue-work is live in another worktree", got, s.layout.RestingBranch())
	}
}

func TestConvergeCompileWritesTheMarker(t *testing.T) {
	s := newObservedSlot(t)
	os.Remove(SetupMarkerPath(s.admin(t)))
	calls := s.f.WeaveCalls
	cv := s.converger(t)
	relock := func() (*HostCreationLease, error) { return AcquireHostCreationLease(s.layout.common) }
	cv.lease.Close()
	if err := cv.compileSetup(context.Background(), relock); err != nil {
		t.Fatal(err)
	}
	defer cv.lease.Close()
	if s.f.WeaveCalls != calls+1 {
		t.Fatalf("weave calls %d, want one compile", s.f.WeaveCalls-calls)
	}
	if r := resource(t, s, ResourceSetup); r.State != StatePresent {
		t.Fatalf("setup %+v after compile", r)
	}
	s.f.FailWeave = true
	os.Remove(SetupMarkerPath(s.admin(t)))
	cv.lease.Close()
	if err := cv.compileSetup(context.Background(), relock); err == nil || !strings.Contains(err.Error(), "fixture setup failed") {
		t.Fatalf("failed compile: %v", err)
	}
	if r := resource(t, s, ResourceSetup); r.State != StateAbsent {
		t.Fatalf("a failed compile wrote a marker: %+v", r)
	}
}
