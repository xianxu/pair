package couchcore

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// SlotReport is a slot's observed resources and the plan reconcile would run
// (pair#387): the plan half of plan/apply, read by couch --show.
type SlotReport struct {
	Address     string
	Observation SlotObservation
	Plan        SlotPlan
	PlanError   string
}

// ShowResult is couch --show's answer: the matching threads, and the slot's
// report when the reference names a :1+ slot (with or without a thread).
type ShowResult struct {
	Threads []ThreadSummary
	Slot    *SlotReport
}

// slotAgentEvidence is the agent evidence for a slot from its thread rows:
// the most active row wins, so a running agent is never hidden by a parked
// sibling.
func slotAgentEvidence(rows []ThreadSummary) EvidenceAgent {
	if len(rows) == 0 {
		return AgentNone
	}
	best := AgentNone
	for _, row := range rows {
		if e := agentEvidence(row.State, row.Reason); agentRank[e] > agentRank[best] {
			best = e
		}
	}
	return best
}

// slotOfShowReference finds the :1+ slot a show reference names: a workspace
// reference (repo:N), a path at or inside a slot, or the one slot all the
// matched threads start in. The identity comes from git (Discover), never from
// the <primary>/.git convention, and the path is resolved through symlinks
// first: observation compares against git's own resolved paths. A workspace
// reference git cannot resolve keeps its own error (e.g. "slot does not
// exist (existing: …)").
func (c *Couch) slotOfShowReference(ctx context.Context, ref string, matches []ThreadRecord) (SlotIdentity, bool, error) {
	if c.Slots == nil {
		return SlotIdentity{}, false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	path, recognized, err := c.WorkspaceReferencePath(ctx, ref)
	if recognized {
		if err != nil {
			return SlotIdentity{}, false, err
		}
		ref = path
	}
	candidates := []string{ref}
	if !filepath.IsAbs(ref) {
		candidates = nil
		for _, m := range matches {
			candidates = append(candidates, m.StartingPath)
		}
	}
	primary, number := "", 0
	for _, p := range candidates {
		physical, err := retainedPhysicalPath(p)
		if err != nil {
			// Never "not a slot" on a failed probe: the caller reports it.
			return SlotIdentity{}, false, fmt.Errorf("resolve %s: %w", p, err)
		}
		pr, n, ok := ParseSlotPath(physical)
		if !ok || (primary != "" && (pr != primary || n != number)) {
			return SlotIdentity{}, false, nil
		}
		primary, number = pr, n
	}
	if primary == "" {
		return SlotIdentity{}, false, nil
	}
	slot, err := c.slotIdentityFromGit(ctx, primary, number)
	if err != nil {
		return SlotIdentity{}, false, err
	}
	return slot, true, nil
}

// slotIdentityFromGit is slot n of the repository at primary as git knows it:
// Discover's candidate when there is one, else the conventional location with
// git's own common directory (a number known only from leftovers).
func (c *Couch) slotIdentityFromGit(ctx context.Context, primary string, n int) (SlotIdentity, error) {
	repository, err := c.Slots.Discover(ctx, primary)
	if err != nil {
		return SlotIdentity{}, err
	}
	for _, candidate := range repository.Slots {
		if candidate.Identity.Number == n {
			return candidate.Identity, nil
		}
	}
	slot := conventionalSlot(repository.Identity.PrimaryRoot, n)
	slot.RepoIdentity = repository.Identity.RepoIdentity
	return slot, nil
}

// ReconcileSlot converges the :1+ slot a reference names (couch --reconcile,
// pair#387). It is Ensure with the slot's own agent evidence and storage
// registration, and it ignores a remembered setup failure: the operator runs
// it after fixing a cause outside the slot.
func (c *Couch) ReconcileSlot(ctx context.Context, ref string) (ProvisionResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	slot, ok, err := c.slotOfShowReference(ctx, ref, nil)
	if err != nil {
		return ProvisionResult{}, err
	}
	if !ok {
		return ProvisionResult{}, fmt.Errorf("%s does not name a :1+ slot (repo:N, or a path in the slot)", ref)
	}
	if c.Workspaces == nil {
		return ProvisionResult{}, fmt.Errorf("workspace provisioner is unavailable")
	}
	agentNow := func(ctx context.Context) EvidenceAgent { return c.slotAgentNow(ctx, slot) }
	return c.Workspaces.Ensure(ctx, ProvisionRequest{Path: slot.PrimaryRoot, Slot: slot.Number, Progress: c.WorkspaceProgress,
		Agent: agentNow(ctx), AgentNow: agentNow, IgnoreMemo: true})
}

// SlotReportFor observes a slot and plans its reconcile. It changes nothing.
func (c *Couch) SlotReportFor(ctx context.Context, slot SlotIdentity, rows []ThreadSummary) SlotReport {
	return c.slotReportWithAgent(ctx, slot, slotAgentEvidence(rows))
}

func (c *Couch) slotReportWithAgent(ctx context.Context, slot SlotIdentity, agent EvidenceAgent) SlotReport {
	io := c.SlotIO
	if io == nil {
		io = OSProvisionIO{}
	}
	obs := ObserveSlot(ctx, SlotObserveInput{IO: io, Layout: LayoutOf(slot), Agent: agent})
	report := SlotReport{Address: WorkspaceReference{Repo: slot.Repo, Number: slot.Number}.String(), Observation: obs}
	plan, err := PlanSlot(PlanInput{Observation: obs})
	if err != nil {
		report.PlanError = err.Error()
	}
	report.Plan = plan
	return report
}

// SlotPlanSummary is the one-line text of a plan, shared by --show and the
// reconcile result.
func SlotPlanSummary(p SlotPlan, planErr string) string {
	if planErr != "" {
		return planErr
	}
	if p.Empty() {
		return "nothing to do"
	}
	var parts []string
	for _, s := range p.Steps {
		parts = append(parts, s.String())
	}
	for _, s := range p.Stops {
		text := "stops at " + string(s.Resource) + " (" + string(s.Class) + ")"
		if s.Reason != "" {
			text += ": " + s.Reason
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "; ")
}
