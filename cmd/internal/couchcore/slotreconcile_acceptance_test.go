package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Real-git acceptance for pair#387 M2 (Done-when bullet 3), through the
// production paths: Ensure (add slot's and open's provisioning) and
// Couch.ReconcileSlot (couch --reconcile).

func reconcileCouch(t *testing.T, f *ProvisionFixture) *Couch {
	t.Helper()
	env := newTestEnv(t, f.Primary)
	env.Couch.Git = ExecGit{}
	env.Couch.Path = OSPathOps{}
	env.Couch.Slots = NewOSSlotCatalog(f)
	env.Couch.Workspaces = NewWorkspaceProvisioner(f)
	env.Couch.SlotIO = f
	return env.Couch
}

// The issue's original case: a deleted slot directory leaves its registration
// and main-slotN behind. It no longer blocks the repository: allocation
// reuses the number, provisioning brings the slot back on its own branch, and
// a second reconcile is a no-op.
func TestAcceptanceDeletedSlotDirectory(t *testing.T) {
	f := newProvisionFixture(t)
	p := NewWorkspaceProvisioner(f)
	if _, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1, Agent: AgentNone}); err != nil {
		t.Fatal(err)
	}
	f.git(f.host(1), "switch", "-q", "-c", "issue-work")
	os.RemoveAll(filepath.Dir(f.host(1)))
	repo, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatalf("a deleted slot blocked the repository's inventory: %v", err)
	}
	alloc, err := SelectStartSlot(repo.Slots, map[int]bool{0: true})
	if err != nil || alloc.Number != 1 || !alloc.Exists {
		t.Fatalf("allocation %+v %v, want slot 1 reused", alloc, err)
	}
	r, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1, Agent: AgentNone})
	if err != nil {
		t.Fatal(err)
	}
	if got := f.git(f.host(1), "branch", "--show-current"); got != "issue-work" || r.Disposition != "created" {
		t.Fatalf("re-added on %q (%s), want issue-work", got, r.Disposition)
	}
	c := reconcileCouch(t, f)
	again, err := c.ReconcileSlot(context.Background(), f.host(1))
	if err != nil || len(again.Report.Plan.Steps) != 0 || again.Disposition != "reused" {
		t.Fatalf("second reconcile: %+v %v", again, err)
	}
}

// The tools:1 shape: an interrupted setup (no clone, no marker). With the
// clone source known it converges; without one it hands off to :0 with
// weave's own line, and the same again on a second run.
func TestAcceptanceInterruptedSetup(t *testing.T) {
	for _, sourceKnown := range []bool{true, false} {
		f := newProvisionFixture(t, "tools")
		p := NewWorkspaceProvisioner(f)
		if _, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1, Agent: AgentNone}); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(f.host(1), "construct", "deps"), "substrate ../ariadne\n")
		os.Remove(SetupMarkerPath(f.git(f.host(1), "rev-parse", "--absolute-git-dir")))
		f.DepSources = map[string]bool{"ariadne": sourceKnown}
		c := reconcileCouch(t, f)
		r, err := c.ReconcileSlot(context.Background(), f.host(1))
		if sourceKnown {
			if err != nil || len(r.Report.Plan.Steps) != 0 || r.Warning != "" {
				t.Fatalf("known source: %+v %v", r, err)
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(f.host(1)), "ariadne", "construct", "base.manifest")); err != nil {
				t.Fatal("dependency not cloned")
			}
			continue
		}
		var blocked *SlotReconcileError
		if !errors.As(err, &blocked) || blocked.Failure.Class != FailureHandoff ||
			!strings.Contains(err.Error(), "Error: missing substrate") || !strings.Contains(err.Error(), "ask the tools:0 agent") || strings.Contains(strings.ToLower(err.Error()), "retry") {
			t.Fatalf("sourceless: %v, want weave's line and the :0 hand-off", err)
		}
		calls := f.WeaveCalls
		_, err2 := c.ReconcileSlot(context.Background(), f.host(1))
		if err2 == nil || err2.Error() != err.Error() {
			t.Fatalf("second run: %v, want the same hand-off", err2)
		}
		// A plain open does not recompile the known failure (R5).
		if _, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1, Agent: AgentNone}); err == nil || f.WeaveCalls != calls+1 {
			t.Fatalf("plain open after the known failure: %v, weave calls %d (want no new compile beyond --reconcile's)", err, f.WeaveCalls-calls)
		}
	}
}

// A missing dependency clone under a valid marker converges by recompiling;
// with no source the slot stays usable with the warning.
func TestAcceptanceMissingDependencyUnderAValidMarker(t *testing.T) {
	for _, sourceKnown := range []bool{true, false} {
		f := newProvisionFixture(t)
		f.DepSources = map[string]bool{"ariadne": true}
		write(t, filepath.Join(f.Primary, "construct", "deps"), "substrate ../ariadne\n")
		f.git(f.Primary, "add", "construct/deps")
		f.git(f.Primary, "commit", "-q", "-m", "declare ariadne")
		f.git(f.Primary, "push", "-q", "upstream", "main")
		p := NewWorkspaceProvisioner(f)
		if _, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1, Agent: AgentNone}); err != nil {
			t.Fatal(err)
		}
		dep := filepath.Join(filepath.Dir(f.host(1)), "ariadne")
		os.RemoveAll(dep)
		f.DepSources = map[string]bool{"ariadne": sourceKnown}
		r, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1, Agent: AgentNone})
		if err != nil {
			t.Fatalf("source %v: %v", sourceKnown, err)
		}
		_, statErr := os.Stat(dep)
		if sourceKnown && (statErr != nil || r.Warning != "") {
			t.Fatalf("known source: dep %v warning %q", statErr, r.Warning)
		}
		if !sourceKnown && (r.Warning == "" || !strings.Contains(r.Warning, "missing substrate")) {
			t.Fatalf("no source: warning %q, want weave's line with the slot usable", r.Warning)
		}
	}
}

// The healthy-slot observation budget (ARCH-CONSTRAINTS): today's reuse path
// plus a handful of git calls. Logged, and failed above 500 ms.
func TestAcceptanceObserveBudget(t *testing.T) {
	s := newObservedSlot(t)
	os.MkdirAll(s.layout.Store(), 0o700)
	start := time.Now()
	obs := ObserveSlot(context.Background(), SlotObserveInput{IO: OSProvisionIO{Env: s.f.IO.Env}, Layout: s.layout, Agent: AgentNone})
	elapsed := time.Since(start)
	t.Logf("healthy-slot observe: %v", elapsed)
	if plan, _ := PlanSlot(PlanInput{Observation: obs}); !plan.Empty() {
		t.Fatalf("healthy slot plans %q", SlotPlanSummary(plan, ""))
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("healthy-slot observe took %v (budget 500 ms)", elapsed)
	}
}
