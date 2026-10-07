package couchcore

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// planFixture builds a RecoverPlanInput from the stateful sdlc fake plus
// Couch rows, so every class fixture is a world rather than a hand-written
// evidence tuple.
type planFixture struct {
	t          *testing.T
	fake       *FakeFleetSDLC
	fleet      *FakeFleet
	root       string
	fleetState string // "" = decode the fake's document
	rows       []ActionableThreadSummary
	couchErr   string
	candidates []RecoverSlotCandidate
	localGit   map[string]RecoverLocalGit
	slotPlans  map[string]SlotReport
}

// workspaceReport is a slot reconciler report whose checkout is usable, with
// the given plan.
func workspaceReport(plan SlotPlan) SlotReport {
	return SlotReport{Address: "pair:1", Plan: plan, Observation: SlotObservation{Resources: []ResourceObservation{
		{ID: ResourceHost, State: StatePresent}, {ID: ResourceSetup, State: StatePresent}}}}
}

// plan records the slot reconciler's report for an address.
func (p *planFixture) plan(address string, report SlotReport) {
	if p.slotPlans == nil {
		p.slotPlans = map[string]SlotReport{}
	}
	p.slotPlans[filepath.Clean(p.path(address))] = report
}

func newPlanFixture(t *testing.T) *planFixture {
	fake := NewFakeFleetSDLC()
	return &planFixture{t: t, fake: fake, fleet: fake.Fleet("/fleet"), root: "/fleet", localGit: map[string]RecoverLocalGit{}}
}

func (p *planFixture) path(address string) string { return p.fleet.SlotPath(address) }

// thread adds one Couch row standing for the slot address.
// orphanedThread is a thread whose zellij server is alive with its socket
// gone (#399): unusable/orphaned-server, carrying the server.
func (p *planFixture) orphanedThread(address string, pid int) {
	p.thread(address, ThreadUnusable, ReasonOrphanedServer)
	p.rows[len(p.rows)-1].Orphan = &launcher.SessionServerIdentity{PID: pid, Identity: "t", Session: "📁" + address}
}

func (p *planFixture) thread(address string, state ActionableThreadState, reason ThreadReason) {
	repo, n := fakeSplitAddress(address)
	path := p.path(address)
	scope, err := launcher.ResolveRepoScope(path)
	if err != nil {
		p.t.Fatal(err)
	}
	row := ActionableThreadSummary{
		Address: ThreadAddress{RepoScope: scope.Key, Tag: ThreadTag(fmt.Sprintf("t%d%d", len(p.rows), n))},
		State:   state, Reason: reason, StartingPath: path, WorkingPath: path,
		Target: ThreadTarget{Kind: ThreadTargetOrdinary},
	}
	if n > 0 {
		row.Target = ThreadTarget{Kind: ThreadTargetSlot, Slot: SlotIdentity{Repo: repo, Number: n, PrimaryRoot: filepath.Join(p.root, repo), EnvironmentRoot: filepath.Dir(path), WorktreeRoot: path}}
	}
	row.Target.Address = row.Address
	p.rows = append(p.rows, row)
}

func (p *planFixture) input() RecoverPlanInput {
	p.t.Helper()
	obs := FleetObservation{Root: p.root, Vantage: filepath.Join(p.root, "pair"), State: p.fleetState}
	if p.fleetState == "" {
		obs.State = FleetObservationPresent
		obs.Inventory = p.fleet.Inventory()
	}
	couch := CouchObservation{State: CouchObservationOK, Rows: p.rows}
	if p.couchErr != "" {
		couch = CouchObservation{State: CouchObservationUnavailable, Error: p.couchErr}
	}
	return RecoverPlanInput{Fleets: []FleetObservation{obs}, Couch: couch, SlotCandidates: p.candidates, LocalGit: p.localGit, SlotPlans: p.slotPlans}
}

type recoverPlanCase struct {
	name    string
	setup   func(*planFixture)
	address string
	want    RecoverClass
	steps   []string
	hold    []string
	notes   []string
	// restore names what ask-agent-restore's message must name: the claim
	// and the checkout holding it.
	restore [2]string
}

func stepActions(row RecoverRow) []string {
	var out []string
	for _, s := range row.Next.Steps {
		out = append(out, s.Action)
	}
	return out
}

func findRow(t *testing.T, plan RecoverPlan, address string) RecoverRow {
	t.Helper()
	for _, row := range plan.Rows {
		if row.Address == address {
			return row
		}
	}
	t.Fatalf("no row %s in %+v", address, plan.Rows)
	return RecoverRow{}
}

func codeKind(code string) string {
	kind, _, _ := strings.Cut(code, ":")
	return kind
}

// activeDependency: pair:1's host clean (resting, or landed on a done issue's
// branch) and its ariadne dependency on its own claimed issue branch.
func activeDependency(p *planFixture, address string, landed bool) {
	p.fleet.AddSlot(address)
	if landed {
		p.fleet.SetBranch(address, "000016-x")
		p.fleet.SetIssueStatus(address, "done")
	}
	dep := p.fleet.AddDependency(address, "ariadne")
	p.fleet.SetMemberBranch(dep, "000290-y")
	p.fleet.ClaimAt(dep, "ariadne#000290", "")
	p.thread(address, ThreadParked, "")
}

func dependencyPath(p *planFixture) string { return "/fleet/worktree/pair-slot1/ariadne" }

// claimedOnBranch is the common world: pair:1 on its claimed issue branch.
func claimedOnBranch(p *planFixture) {
	p.fleet.AddSlot("pair:1")
	p.fleet.SetBranch("pair:1", "000011-x")
	p.fleet.Claim("pair:1", "pair#000011")
}

// primaryRebootOnly is a :0 whose conversation is gone: reboot is offered,
// resume is not (no recovery verdict, no unfinished request).
func primaryRebootOnly(p *planFixture) {
	p.fleet.AddSlot("pair:0")
	p.thread("pair:0", ThreadUnusable, ReasonSessionGone)
}

func recoverPlanCases() []recoverPlanCase {
	return []recoverPlanCase{
		{name: "dangling claim on a vanished slot", address: "pair:5", want: RecoverDirectoryMissing, hold: []string{"directory-missing"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:0")
				p.fleet.AddDanglingClaim("/fleet/pair", "/fleet/worktree/pair-slot5/pair", "pair#000015", "pair:5")
			}},
		{name: "couch store unreadable", address: "pair:1", want: RecoverAgentUnknown, hold: []string{"couch-unavailable"},
			setup: func(p *planFixture) { claimedOnBranch(p); p.couchErr = "store unreadable" }},
		{name: "busy start", address: "pair:1", want: RecoverStartUnreconciled, hold: []string{"start-unreconciled"},
			setup: func(p *planFixture) { claimedOnBranch(p); p.thread("pair:1", ThreadBusy, "") }},
		{name: "unusable unknown", address: "pair:1", want: RecoverAgentUnknown, hold: []string{"agent-unknown"},
			setup: func(p *planFixture) { claimedOnBranch(p); p.thread("pair:1", ThreadUnusable, ReasonUnknown) }},
		{name: "orphaned server", address: "pair:1", want: RecoverOrphanedServer, hold: []string{"orphaned-server"},
			setup: func(p *planFixture) { claimedOnBranch(p); p.orphanedThread("pair:1", 4242) }},
		{name: "two threads", address: "pair:1", want: RecoverAmbiguousThreads, hold: []string{"threads:2"},
			setup: func(p *planFixture) {
				claimedOnBranch(p)
				p.thread("pair:1", ThreadParked, "")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "claim elsewhere", address: "pair:2", want: RecoverConflict, hold: []string{"conflict:claim-elsewhere"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:2")
				p.fleet.SetBranch("pair:2", "000012-x")
				p.fleet.AddSlot("pair:3")
				p.fleet.Claim("pair:3", "pair#000012")
				p.thread("pair:2", ThreadParked, "")
			}},
		{name: "terminal issue branch", address: "pair:1", want: RecoverConflict, hold: []string{"conflict:issue-terminal"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "000013-x")
				p.fleet.SetIssueStatus("pair:1", "done")
				p.fleet.SetDirty("pair:1", 1)
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "clean unclaimed done-issue branch", address: "pair:1", want: RecoverLanded, notes: []string{"issue-done-branch"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "000013-x")
				p.fleet.SetIssueStatus("pair:1", "done")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "done-issue branch with unread base", address: "pair:1", want: RecoverEvidenceUnavailable, hold: []string{"git-unknown"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "000013-x")
				p.fleet.SetIssueStatus("pair:1", "wontfix")
				p.fleet.SetBaseUnavailable("pair:1")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "host claim workspace mismatch", address: "pair:1", want: RecoverConflict, hold: []string{"conflict:claim-workspace"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "000011-x")
				p.fleet.ClaimAt(p.path("pair:1"), "pair#000011", "pair:9")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "claim on a non-issue branch", address: "pair:1", want: RecoverConflict, hold: []string{"conflict:claim-off-branch"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "topic")
				p.fleet.Claim("pair:1", "pair#000011")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "claim for A with B checked out", address: "pair:1", want: RecoverConflict, hold: []string{"conflict:claim-branch-mismatch"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "000022-x")
				p.fleet.Claim("pair:1", "pair#000021")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "two claims on the resting branch", address: "pair:1", want: RecoverAmbiguousClaims, hold: []string{"ambiguous-claims"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.Claim("pair:1", "pair#000031")
				p.fleet.Claim("pair:1", "pair#000032")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "unclaimed resting branch with unread base", address: "pair:1", want: RecoverEvidenceUnavailable, hold: []string{"git-unknown"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBaseUnavailable("pair:1")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "conversation alone", address: "pair:1", want: RecoverIdle,
			setup: func(p *planFixture) { p.fleet.AddSlot("pair:1"); p.thread("pair:1", ThreadParked, "") }},
		{name: "workspace reconcilable", address: "pair:1", want: RecoverReconcilable, steps: []string{"reconcile"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.thread("pair:1", ThreadParked, "")
				p.plan("pair:1", workspaceReport(SlotPlan{Steps: []PlannedStep{{Step: StepCompile, Resource: ResourceSetup}}}))
			}},
		{name: "workspace needs :0", address: "pair:1", want: RecoverSlotNeedsZero, hold: []string{"workspace-handoff"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.thread("pair:1", ThreadParked, "")
				// Setup never completed and cannot: no agent could work here.
				r := workspaceReport(SlotPlan{Stops: []PlanStop{{Resource: ResourceSetup, Class: StopHandoff, Reason: "setup-failed-known: Error: missing substrate"}}})
				r.Observation.Resources[1] = ResourceObservation{ID: ResourceSetup, State: StateAbsent, Sub: SubFailedKnown}
				p.plan("pair:1", r)
			}},
		{name: "workspace degraded but usable", address: "pair:1", want: RecoverIdle, notes: []string{"workspace-degraded"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.thread("pair:1", ThreadParked, "")
				// The resting branch is checked out elsewhere, but the host works:
				// the callers open it with a warning, so the report does not hold it.
				p.plan("pair:1", workspaceReport(SlotPlan{Stops: []PlanStop{{Resource: ResourceBranch, Class: StopHandoff, Reason: "main-slot1 is checked out elsewhere"}}}))
			}},
		{name: "workspace held by a live agent", address: "pair:1", want: RecoverIdle, notes: []string{"workspace-held"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.thread("pair:1", ThreadLive, "")
				p.plan("pair:1", workspaceReport(SlotPlan{Stops: []PlanStop{{Resource: DepResource("ariadne"), Class: StopHold, Reason: StopReasonAgentLive}}}))
			}},
		{name: "workspace partly unobservable", address: "pair:1", want: RecoverIdle, notes: []string{"workspace-unknown"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.thread("pair:1", ThreadParked, "")
				p.plan("pair:1", workspaceReport(SlotPlan{Stops: []PlanStop{{Resource: ResourceDeps, Class: StopUnknown, Reason: "construct/deps unreadable"}}}))
			}},
		{name: "deleted slot directory with leftovers", address: "pair:5", want: RecoverReconcilable, steps: []string{"reconcile"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:0")
				p.fleet.AddDanglingClaim("/fleet/pair", "/fleet/worktree/pair-slot5/pair", "pair#000015", "pair:5")
				p.slotPlans = map[string]SlotReport{"/fleet/worktree/pair-slot5/pair": {Address: "pair:5", Plan: SlotPlan{Steps: []PlannedStep{{Step: StepMkdirEnv, Resource: ResourceEnv}}},
					Observation: SlotObservation{Resources: []ResourceObservation{{ID: ResourceEnv, State: StateAbsent}}}}}
			}},
		{name: "dirty resting branch, no claim", address: "pair:1", want: RecoverUnidentifiedWork, hold: []string{"unidentified-work"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetDirty("pair:1", 2)
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "claimed slot with no thread", address: "pair:1", want: RecoverNoCouchThread, hold: []string{"no-couch-thread"},
			setup: claimedOnBranch},
		{name: "one claim on a clean resting branch", address: "pair:1", want: RecoverRestoreWorkspace, steps: []string{"resume", "ask-agent-restore"}, restore: [2]string{"pair#000014", "pair"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.Claim("pair:1", "pair#000014")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "one claim on a clean resting branch, live", address: "pair:1", want: RecoverRestoreWorkspace, steps: []string{"ask-agent-restore"}, restore: [2]string{"pair#000014", "pair"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.Claim("pair:1", "pair#000014")
				p.thread("pair:1", ThreadLive, "")
			}},
		{name: "one claim on a dirty resting branch", address: "pair:1", want: RecoverRestoreWorkspace, steps: []string{"resume"}, notes: []string{"resting-branch-dirty"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.Claim("pair:1", "pair#000014")
				p.fleet.SetDirty("pair:1", 1)
				p.thread("pair:1", ThreadParked, "")
			}},
		// BR-4: a dependency's claim is judged against the dependency's own
		// branch, never the host's (the live golden's ariadne#000290 shape).
		{name: "dependency claim on the dependency's resting branch", address: "pair:1", want: RecoverRestoreWorkspace, steps: []string{"resume", "ask-agent-restore"}, restore: [2]string{"ariadne#000290", "ariadne"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.ClaimAt(p.fleet.AddDependency("pair:1", "ariadne"), "ariadne#000290", "")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "dependency claim on its own issue branch", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				dep := p.fleet.AddDependency("pair:1", "ariadne")
				p.fleet.SetMemberBranch(dep, "000290-y")
				p.fleet.ClaimAt(dep, "ariadne#000290", "")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "dependency claim on another branch", address: "pair:1", want: RecoverConflict, hold: []string{"conflict:dependency-claim"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				dep := p.fleet.AddDependency("pair:1", "ariadne")
				p.fleet.SetMemberBranch(dep, "topic")
				p.fleet.ClaimAt(dep, "ariadne#000290", "")
				p.thread("pair:1", ThreadParked, "")
			}},
		// Round 2: a landed host is at rest for a dependency claim too.
		{name: "landed host beside an active dependency claim", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"}, notes: []string{"issue-done-branch"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "000016-x")
				p.fleet.SetIssueStatus("pair:1", "done")
				dep := p.fleet.AddDependency("pair:1", "ariadne")
				p.fleet.SetMemberBranch(dep, "000290-y")
				p.fleet.ClaimAt(dep, "ariadne#000290", "")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "host and dependency claims both resting", address: "pair:1", want: RecoverRestoreWorkspace, steps: []string{"resume", "ask-agent-restore"}, notes: []string{"inactive-claims"}, restore: [2]string{"pair#000014", "pair"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.Claim("pair:1", "pair#000014")
				p.fleet.ClaimAt(p.fleet.AddDependency("pair:1", "ariadne"), "ariadne#000290", "")
				p.thread("pair:1", ThreadParked, "")
			}},
		// BR-14: a dependency's git facts are the dependency's, never the host's.
		{name: "unlanded commits on a claimed dependency beside a resting host", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"},
			setup: func(p *planFixture) {
				activeDependency(p, "pair:1", false)
				p.fleet.SetMemberAhead(dependencyPath(p), 2)
			}},
		{name: "dirt on a claimed dependency beside a resting host", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"},
			setup: func(p *planFixture) {
				activeDependency(p, "pair:1", false)
				p.fleet.SetMemberDirty(dependencyPath(p), 2)
			}},
		{name: "unlanded commits on a claimed dependency beside a landed host", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"}, notes: []string{"issue-done-branch"},
			setup: func(p *planFixture) {
				activeDependency(p, "pair:1", true)
				p.fleet.SetMemberAhead(dependencyPath(p), 2)
			}},
		{name: "dirt on a claimed dependency beside a landed host", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"}, notes: []string{"issue-done-branch"},
			setup: func(p *planFixture) {
				activeDependency(p, "pair:1", true)
				p.fleet.SetMemberDirty(dependencyPath(p), 2)
			}},
		{name: "unclaimed dirt on a dependency", address: "pair:1", want: RecoverUnidentifiedWork, hold: []string{"unidentified-work"}, notes: []string{"dependency-work"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetMemberDirty(p.fleet.AddDependency("pair:1", "ariadne"), 1)
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "claimed host beside unclaimed dependency work", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"}, notes: []string{"dependency-work"},
			setup: func(p *planFixture) {
				claimedOnBranch(p)
				p.fleet.SetMemberOperation(p.fleet.AddDependency("pair:1", "ariadne"), "")
				p.fleet.SetMemberAhead(dependencyPath(p), 1)
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "dependency claim on its dirty member's resting branch", address: "pair:1", want: RecoverRestoreWorkspace, steps: []string{"resume"}, notes: []string{"resting-branch-dirty"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				dep := p.fleet.AddDependency("pair:1", "ariadne")
				p.fleet.ClaimAt(dep, "ariadne#000290", "")
				p.fleet.SetMemberDirty(dep, 1)
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "lost host claim beside a dependency claim", address: "pair:1", want: RecoverClaimLikelyLost, steps: []string{"resume"}, notes: []string{"inactive-claims", "claim-repair"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "000016-x")
				p.fleet.ClaimAt(p.fleet.AddDependency("pair:1", "ariadne"), "ariadne#000290", "")
				p.thread("pair:1", ThreadParked, "")
			}},
		// BR-5: a dangling claim on a present slot's missing dependency is
		// unread evidence, never absence.
		{name: "dangling claim on a present slot's missing dependency", address: "pair:1", want: RecoverPartialEvidence, steps: []string{"resume"}, notes: []string{"dependency-unread"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.MissingMember("pair:1", "ariadne")
				p.fleet.AddDanglingClaim("/fleet/worktree/pair-slot1/ariadne", "/fleet/worktree/pair-slot1/ariadne", "ariadne#000290", "")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "claimed branch, parked", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"},
			setup: func(p *planFixture) { claimedOnBranch(p); p.thread("pair:1", ThreadParked, "") }},
		{name: "claimed branch, detached agent", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"},
			setup: func(p *planFixture) { claimedOnBranch(p); p.thread("pair:1", ThreadDetached, "") }},
		{name: "claimed branch, live", address: "pair:1", want: RecoverAgrees,
			setup: func(p *planFixture) { claimedOnBranch(p); p.thread("pair:1", ThreadLive, "") }},
		{name: "claimed dirty branch, parked", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"},
			setup: func(p *planFixture) {
				claimedOnBranch(p)
				p.fleet.SetDirty("pair:1", 3)
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "active plus inactive claim", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"}, notes: []string{"inactive-claims"},
			setup: func(p *planFixture) {
				claimedOnBranch(p)
				p.fleet.Claim("pair:1", "pair#000099")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "claimed slot with a detached HEAD", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"}, notes: []string{"detached-head"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetDetached("pair:1")
				p.fleet.Claim("pair:1", "pair#000011")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "claimed branch with unread base", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"}, notes: []string{"git-unknown"},
			setup: func(p *planFixture) {
				claimedOnBranch(p)
				p.fleet.SetBaseUnavailable("pair:1")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "stale claims still suggest", address: "pair:1", want: RecoverAgrees, steps: []string{"resume"}, notes: []string{"claims-stale"},
			setup: func(p *planFixture) {
				claimedOnBranch(p)
				p.fleet.SetClaimsState("pair:1", FleetClaimsStale, "tracker unreachable")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "primary reboot only, dirty claimed branch", address: "pair:0", want: RecoverAgrees, steps: []string{"reboot"}, notes: []string{"inspect-uncommitted-first"},
			setup: func(p *planFixture) {
				primaryRebootOnly(p)
				p.fleet.SetBranch("pair:0", "000010-x")
				p.fleet.Claim("pair:0", "pair#000010")
				p.fleet.SetDirty("pair:0", 1)
			}},
		{name: "primary reboot only, operation in progress", address: "pair:0", want: RecoverNoSafeStep, hold: []string{"reboot-unsafe-operation"},
			setup: func(p *planFixture) {
				primaryRebootOnly(p)
				p.fleet.SetBranch("pair:0", "000010-x")
				p.fleet.Claim("pair:0", "pair#000010")
				p.fleet.SetOperation("pair:0", "rebase-merge")
			}},
		{name: "primary reboot only, detached HEAD", address: "pair:0", want: RecoverNoSafeStep, hold: []string{"reboot-unsafe-git"}, notes: []string{"detached-head"},
			setup: func(p *planFixture) {
				primaryRebootOnly(p)
				p.fleet.SetDetached("pair:0")
				p.fleet.Claim("pair:0", "pair#000010")
			}},
		// Reboot safety is a slot-level property: a reboot replaces the whole
		// slot's agent, so every member's dirt, unread git and operation
		// guard it, not only the host's (M1 review round 4).
		{name: "primary reboot only, claimed dependency with unread base", address: "pair:0", want: RecoverNoSafeStep, hold: []string{"reboot-unsafe-git"},
			setup: func(p *planFixture) {
				primaryRebootOnly(p)
				p.fleet.SetBranch("pair:0", "000010-x")
				p.fleet.Claim("pair:0", "pair#000010")
				dep := p.fleet.AddDependency("pair:0", "ariadne")
				p.fleet.SetMemberBranch(dep, "000290-y")
				p.fleet.ClaimAt(dep, "ariadne#000290", "")
				p.fleet.SetMemberBaseUnavailable(dep)
			}},
		{name: "primary reboot only, dirty claimed dependency", address: "pair:0", want: RecoverAgrees, steps: []string{"reboot"}, notes: []string{"inspect-uncommitted-first"},
			setup: func(p *planFixture) {
				primaryRebootOnly(p)
				p.fleet.SetBranch("pair:0", "000010-x")
				p.fleet.Claim("pair:0", "pair#000010")
				dep := p.fleet.AddDependency("pair:0", "ariadne")
				p.fleet.SetMemberBranch(dep, "000290-y")
				p.fleet.ClaimAt(dep, "ariadne#000290", "")
				p.fleet.SetMemberDirty(dep, 1)
			}},
		// A parked slot whose conversation cannot be resolved (its agent never
		// took a turn): no resume to suggest, reboot is (pair#367 smoke test).
		{name: "claimed slot whose parked conversation is lost", address: "pair:1", want: RecoverAgrees, steps: []string{"reboot"},
			setup: func(p *planFixture) { claimedOnBranch(p); p.thread("pair:1", ThreadUnusable, ReasonBindingLost) }},
		{name: "slot record whose path cannot be read", address: "pair:1", want: RecoverNoSafeStep, hold: []string{"no-actor-action"},
			setup: func(p *planFixture) { claimedOnBranch(p); p.thread("pair:1", ThreadUnusable, ReasonPathMissing) }},
		{name: "work but no visible claim", address: "pair:1", want: RecoverClaimLikelyLost, steps: []string{"resume"}, notes: []string{"claim-repair"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "000016-x")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "issue branch, partial claims", address: "pair:1", want: RecoverPartialEvidence, steps: []string{"resume"}, notes: []string{"claims-partial"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "000017-x")
				p.fleet.SetClaimsState("pair:1", FleetClaimsPartial, "card unreadable")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "issue branch, unknown claims", address: "pair:1", want: RecoverPartialEvidence, steps: []string{"resume"}, notes: []string{"claims-unknown"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "000017-x")
				p.fleet.SetClaimsState("pair:1", FleetClaimsUnknown, "no answer")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "issue branch, no tracker", address: "pair:1", want: RecoverPartialEvidence, steps: []string{"resume"}, notes: []string{"claims-absent"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "000017-x")
				p.fleet.SetClaimsState("pair:1", FleetClaimsAbsent, "")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "issue branch with unread base", address: "pair:1", want: RecoverPartialEvidence, steps: []string{"resume"}, notes: []string{"git-unknown"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetBranch("pair:1", "000018-x")
				p.fleet.SetBaseUnavailable("pair:1")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "unsupported fleet, local probe on an issue branch", address: "pair:1", want: RecoverPartialEvidence, steps: []string{"resume"}, notes: []string{"claims-unsupported", "git-local-probe", "git-unknown"},
			setup: func(p *planFixture) {
				p.fleetState = FleetObservationUnsupported
				p.candidates = []RecoverSlotCandidate{{Fleet: "/fleet", Address: "pair:1", Path: p.path("pair:1")}}
				p.localGit[p.path("pair:1")] = RecoverLocalGit{Status: SlotGitStatus{Branch: "000012-x", Dirty: true}}
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "partial claims, primary reboot only", address: "pair:0", want: RecoverNoSafeStep, hold: []string{"resume-only"}, notes: []string{"claims-partial"},
			setup: func(p *planFixture) {
				primaryRebootOnly(p)
				p.fleet.SetBranch("pair:0", "000019-x")
				p.fleet.SetClaimsState("pair:0", FleetClaimsPartial, "card unreadable")
			}},
	}
}

func TestDeriveRecoverPlanCoversEveryClassHoldAndNote(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range recoverPlanCases() {
		t.Run(c.name, func(t *testing.T) {
			p := newPlanFixture(t)
			c.setup(p)
			row := findRow(t, DeriveRecoverPlan(p.input()), c.address)
			if row.Class != c.want || !slices.Equal(stepActions(row), c.steps) || !slices.Equal(row.Next.Hold, c.hold) || !slices.Equal(row.Next.Notes, c.notes) {
				t.Fatalf("got %s %v %v %v, want %s %v %v %v (reason %q)", row.Class, stepActions(row), row.Next.Hold, row.Next.Notes, c.want, c.steps, c.hold, c.notes, row.Reason)
			}
			if row.Automatic != (len(row.Next.Steps) > 0 && len(row.Next.Hold) == 0) {
				t.Fatalf("automatic = %v with steps %v hold %v", row.Automatic, row.Next.Steps, row.Next.Hold)
			}
			assertReasonMatchesDecision(t, row)
			// recoverReason is total over the classes (BR-15): no fixture's
			// row may read the default text.
			if row.Class != RecoverNoRule && row.Reason == "no rule matched" {
				t.Fatalf("class %s has no reason text", row.Class)
			}
			for _, step := range row.Next.Steps {
				// Every step names the CLI text that runs it (M2): the slot
				// operation's own command, or a send to the slot's agent.
				want := SlotOperationCommand(step.Action, row.Address)
				if step.Action == "ask-agent-restore" {
					want = SendToCommand(row.Address, step.Message)
				}
				if step.Command != want {
					t.Fatalf("step %s command %q, want %q", step.Action, step.Command, want)
				}
				if step.Action == "ask-agent-restore" && step.Message != RestoreWorkspaceMessage(c.restore[0], row.Address, c.restore[1]) {
					t.Fatalf("restore message = %q, want one naming %s in the %s checkout", step.Message, c.restore[0], c.restore[1])
				}
			}
			seen["class:"+string(row.Class)] = true
			for _, h := range row.Next.Hold {
				seen["hold:"+codeKind(h)] = true
			}
			for _, n := range row.Next.Notes {
				seen["note:"+codeKind(n)] = true
			}
		})
	}
	for _, k := range derivedRecoverVocabulary() {
		if !seen[k] {
			t.Errorf("no fixture yields %s", k)
		}
	}
}

// derivedRecoverVocabulary is every class, hold kind and note kind the
// derivation can emit, read from the vocabularies themselves, minus the
// defensive no-rule the totality test proves unreachable.
func derivedRecoverVocabulary() []string {
	var out []string
	for _, c := range AllRecoverClasses() {
		if c != RecoverNoRule {
			out = append(out, "class:"+string(c))
		}
	}
	for _, h := range AllRecoverHolds() {
		if h != HoldNoRule {
			out = append(out, "hold:"+string(h))
		}
	}
	for _, n := range AllRecoverNotes() {
		out = append(out, "note:"+string(n))
	}
	return out
}

// forEachEvidence crosses every value of every SlotEvidence dimension, each
// from its own All* list, and calls visit for what consistent admits. It does
// not materialize the domain (millions of points).
func forEachEvidence(visit func(SlotEvidence)) {
	tri := AllEvidenceTriStates()
	for _, dir := range AllEvidenceDirs() {
		for _, couch := range AllEvidenceCouch() {
			for _, agent := range AllEvidenceAgents() {
				for _, threads := range AllEvidenceThreads() {
					for _, offer := range AllEvidenceOffers() {
						for _, branch := range AllEvidenceBranches() {
							for _, claims := range AllEvidenceClaims() {
								for _, quality := range AllEvidenceQualities() {
									for _, source := range AllEvidenceGitSources() {
										for _, dirty := range tri {
											for _, unlanded := range tri {
												for _, operation := range tri {
													for _, workspace := range []bool{false, true} {
														for _, elsewhere := range []bool{false, true} {
															for _, dep := range AllEvidenceDepClaims() {
																for _, depTree := range AllEvidenceDepTrees() {
																	e := SlotEvidence{DepClaims: dep, DepTree: depTree, Dir: dir, Couch: couch, Agent: agent, Threads: threads, Offer: offer,
																		Branch: branch, Claims: claims, Quality: quality, GitSource: source, Dirty: dirty, Unlanded: unlanded, Operation: operation,
																		Workspace: workspace, Elsewhere: elsewhere}
																	if consistent(e) {
																		visit(e)
																	}
																}
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// withDependencyFacts is e with only its dependency dimensions replaced.
func withDependencyFacts(e SlotEvidence, dep EvidenceDepClaims, tree EvidenceDepTree) SlotEvidence {
	e.DepClaims, e.DepTree = dep, tree
	return e
}

func TestDeriveRecoverPlanIsTotalOverTheEvidenceDomain(t *testing.T) {
	start := time.Now()
	points := 0
	forEachEvidence(func(e SlotEvidence) {
		points++
		d := classifyRecover(e)
		fail := func(why string) { t.Fatalf("%s: %+v → %+v", why, e, d) }
		if d.Class == RecoverNoRule {
			fail("no rule matches")
		}
		if len(d.Hold) > 0 && len(d.Steps) > 0 {
			fail("a hold with steps")
		}
		offered := e.Offer.actions()
		for _, step := range d.Steps {
			switch step {
			case "resume", "reboot":
				if !slices.Contains(offered, step) {
					fail(step + " is not offered by ActorActions")
				}
			case "ask-agent-restore", "reconcile":
			default:
				fail("unknown step " + step)
			}
			if step == "reboot" && (e.Operation != TriNo || e.DepTree == DepTreeOperation || e.DepTree == DepTreeUnknown || e.Branch == BranchDetached || e.Branch == BranchUnknown || e.gitUnknown() || e.Agent == AgentBusy || e.Agent == AgentUnusableUnknown) {
				fail("unsafe reboot")
			}
		}
		// Reboot safety is slot-level: dirt in ANY member of the slot never
		// yields a reboot without the inspect-uncommitted-first note, and an
		// operation or unread tree in any member never yields one at all.
		if slices.Contains(d.Steps, "reboot") {
			if (e.Dirty == TriYes || e.DepTree == DepTreeDirty) && !slices.Contains(d.Notes, NoteInspectUncommittedFirst) {
				fail("reboot over a dirty member without inspect-uncommitted-first")
			}
		}
		if (e.Operation == TriYes || e.DepTree == DepTreeOperation) && slices.Contains(d.Steps, "reboot") {
			fail("reboot over an operation in some member")
		}
		if d.Class == RecoverIdle && (e.Dirty != TriNo || e.Unlanded != TriNo || e.Operation != TriNo) {
			fail("unknown or present work read as idle")
		}
		if d.Class == RecoverRestoreWorkspace && !(e.Claims == ClaimsOneInactive && e.Branch == BranchResting ||
			e.Claims == ClaimsNone && (e.DepClaims == DepResting || e.DepClaims == DepRestingDirty) && hostAtRest(e)) {
			fail("restore-workspace without one claim on its own member's resting branch")
		}
		if slices.Contains(d.Hold, "conflict:claim-branch-mismatch") && e.Claims != ClaimsOneInactive && e.Claims != ClaimsManyWithoutActive {
			fail("a branch mismatch not made of the host's own claims")
		}
		if (d.Class == RecoverIdle || d.Class == RecoverLanded) && e.DepClaims != DepNone {
			fail("a dependency claim read as nothing to recover")
		}
		// A host at rest (resting or landed, nothing unlanded, no operation)
		// beside dependency claims is never unidentified work (BR-10).
		if d.Class == RecoverUnidentifiedWork && e.Claims == ClaimsNone && e.DepClaims != DepNone && e.DepClaims != DepWork && e.Dirty == TriNo &&
			(e.Branch == BranchResting || e.Branch == BranchTerminalIssue) && e.Unlanded == TriNo && e.Operation == TriNo {
			fail("a host at rest beside a dependency claim read as unidentified work")
		}
		clean := e.Dirty == TriNo && e.Unlanded == TriNo && e.Operation == TriNo && e.Claims == ClaimsNone
		if d.Class == RecoverLanded && (!clean || e.Branch != BranchTerminalIssue || len(d.Steps) > 0) {
			fail("landed without a clean, unclaimed done-issue branch")
		}
		if slices.Contains(d.Hold, "conflict:issue-terminal") && !(e.Claims != ClaimsNone || e.Dirty == TriYes || e.Unlanded == TriYes || e.Operation == TriYes) {
			fail("a done-issue branch without dirt, unlanded commits or a claim read as a conflict")
		}
	})
	if points < 1000 {
		t.Fatalf("evidence domain has %d points; consistent() admits too little", points)
	}
	t.Logf("%d consistent evidence points in %s", points, time.Since(start))
}

// consistent's exclusions are physical facts, each pinned here so the domain
// cannot shrink silently.
func TestConsistentExcludesOnlyImpossibleEvidence(t *testing.T) {
	base := SlotEvidence{Dir: DirPresent, Couch: CouchOK, Agent: AgentParked, Threads: ThreadsOne, Offer: OfferResumeReboot, Branch: BranchOpenIssue,
		Claims: ClaimsOneActive, DepClaims: DepNone, DepTree: DepTreeClean, Quality: QualityPresent, GitSource: GitSourceSDLC, Dirty: TriNo, Unlanded: TriNo, Operation: TriNo}
	if !consistent(base) {
		t.Fatal("the ordinary claimed, parked slot is excluded")
	}
	for name, mutate := range map[string]func(*SlotEvidence){
		"couch unavailable with a thread":    func(e *SlotEvidence) { e.Couch = CouchUnavailable },
		"no agent but a thread":              func(e *SlotEvidence) { e.Agent = AgentNone },
		"parked agent offering reboot only":  func(e *SlotEvidence) { e.Offer = OfferReboot },
		"unknown git source, known branch":   func(e *SlotEvidence) { e.GitSource = GitSourceUnknown },
		"local probe with claims":            func(e *SlotEvidence) { e.GitSource = GitSourceLocalProbe },
		"no tracker listing a host claim":    func(e *SlotEvidence) { e.Quality = QualityAbsent },
		"unread host read listing a claim":   func(e *SlotEvidence) { e.Quality = QualityUnknown },
		"dependency operation, no finding":   func(e *SlotEvidence) { e.DepTree = DepTreeOperation },
		"dependency dirt, no finding":        func(e *SlotEvidence) { e.DepTree = DepTreeDirty },
		"resting-dirty member, clean trees":  func(e *SlotEvidence) { e.Branch, e.Claims, e.DepClaims = BranchResting, ClaimsNone, DepRestingDirty },
		"active claim on the resting branch": func(e *SlotEvidence) { e.Branch = BranchResting },
		"workspace mismatch without claims":  func(e *SlotEvidence) { e.Claims, e.Workspace = ClaimsNone, true },
		"claim both here and elsewhere":      func(e *SlotEvidence) { e.Elsewhere = true },
	} {
		e := base
		mutate(&e)
		if consistent(e) {
			t.Errorf("%s: admitted %+v", name, e)
		}
	}
}

func TestOneWeakSourceNeverHoldsARepo(t *testing.T) {
	p := newPlanFixture(t)
	claimedOnBranch(p)
	p.thread("pair:1", ThreadParked, "")
	p.fleet.AddSlot("pair:2")
	p.fleet.SetBranch("pair:2", "000020-x")
	p.thread("pair:2", ThreadParked, "")
	p.fleet.AddSlot("pair:3")
	p.fleet.SetBranch("pair:3", "000030-x")
	p.fleet.SetClaimsState("pair:3", FleetClaimsPartial, "card 30 unreadable")
	p.thread("pair:3", ThreadParked, "")
	plan := DeriveRecoverPlan(p.input())
	for address, want := range map[string]RecoverClass{"pair:1": RecoverAgrees, "pair:2": RecoverClaimLikelyLost, "pair:3": RecoverPartialEvidence} {
		row := findRow(t, plan, address)
		if row.Class != want || !slices.Equal(stepActions(row), []string{"resume"}) {
			t.Errorf("%s = %s %v, want %s [resume]", address, row.Class, stepActions(row), want)
		}
	}
}

func TestUnsupportedFleetUsesLocalProbe(t *testing.T) {
	p := newPlanFixture(t)
	p.fleetState = FleetObservationUnsupported
	p.candidates = []RecoverSlotCandidate{
		{Fleet: "/fleet", Address: "pair:1", Path: p.path("pair:1")},
		{Fleet: "/fleet", Address: "pair:2", Path: p.path("pair:2")},
	}
	p.localGit[p.path("pair:1")] = RecoverLocalGit{Status: SlotGitStatus{Branch: "000012-x", Dirty: true}}
	p.localGit[p.path("pair:2")] = RecoverLocalGit{Status: SlotGitStatus{Branch: "main-slot2"}}
	p.thread("pair:1", ThreadParked, "")
	p.thread("pair:2", ThreadParked, "")
	plan := DeriveRecoverPlan(p.input())
	one := findRow(t, plan, "pair:1")
	if one.Class != RecoverPartialEvidence || !slices.Equal(stepActions(one), []string{"resume"}) || !slices.Contains(one.Next.Notes, "git-local-probe") || one.Git.Source != "local-probe" {
		t.Fatalf("pair:1 = %+v", one)
	}
	two := findRow(t, plan, "pair:2")
	if two.Class != RecoverEvidenceUnavailable || len(two.Next.Steps) != 0 {
		t.Fatalf("pair:2 = %+v", two)
	}
	if plan.Fleets[0].State != FleetObservationUnsupported {
		t.Fatalf("fleet = %+v", plan.Fleets[0])
	}
	// A probe that failed is unknown, never a clean slot.
	p.localGit[p.path("pair:2")] = RecoverLocalGit{Err: "git status: timeout"}
	if row := findRow(t, DeriveRecoverPlan(p.input()), "pair:2"); row.Class != RecoverEvidenceUnavailable || row.Git.Source != "unknown" {
		t.Fatalf("failed probe = %+v", row)
	}
}

func TestJoinUsesThePrimaryRowDefinition(t *testing.T) {
	p := newPlanFixture(t)
	p.fleet.AddSlot("pair:0")
	p.thread("pair:0", ThreadParked, "")
	sub := p.rows[0]
	sub.Address.Tag = "subdir"
	sub.Target.Address = sub.Address
	sub.StartingPath = filepath.Join(sub.StartingPath, "cmd")
	p.rows = append(p.rows, sub)
	plan := DeriveRecoverPlan(p.input())
	if row := findRow(t, plan, "pair:0"); row.Agent.Threads != 1 || row.Class != RecoverIdle {
		t.Fatalf(":0 = %+v", row)
	}
	if plan.Ignored.NonSlotThreads != 1 {
		t.Fatalf("ignored = %+v", plan.Ignored)
	}
	aliased := ApplyRepositoryAliases(append([]ActionableThreadSummary(nil), p.rows...), []RepositoryName{{Key: "/fleet/pair", Alias: "pp"}})
	if aliased[0].RepositoryAlias != "pp" || aliased[1].RepositoryAlias != "" {
		t.Fatalf("aliases = %q %q", aliased[0].RepositoryAlias, aliased[1].RepositoryAlias)
	}
}

func TestClaimsAttachToTheirSlotPath(t *testing.T) {
	p := newPlanFixture(t)
	p.fleet.AddSlot("pair:1")
	p.fleet.SetBranch("pair:1", "000011-x")
	p.fleet.Claim("pair:1", "pair#000011")
	dep := p.fleet.AddDependency("pair:1", "ariadne")
	p.fleet.ClaimAt(dep, "ariadne#000290", "")
	p.thread("pair:1", ThreadParked, "")
	p.fleet.AddSlot("pair:2")
	p.fleet.SetBranch("pair:2", "000021-x")
	p.fleet.ClaimAt(p.path("pair:2"), "pair#000021", "pair:9")
	p.thread("pair:2", ThreadParked, "")
	p.fleet.AddDanglingClaim("/fleet/pair", "/fleet/worktree/pair-slot5/pair", "pair#000015", "pair:5")
	p.fleet.AddOffSlotRow("/scratch/wt", "/fleet/pair", "")
	p.fleet.ClaimAt("/scratch/wt", "pair#000077", "")
	plan := DeriveRecoverPlan(p.input())
	one := findRow(t, plan, "pair:1")
	if one.Class != RecoverAgrees || !slices.Contains(one.Next.Notes, "inactive-claims") || !slices.Equal(one.Claims.Dependency, []RecoverMemberClaim{{Ref: "ariadne#000290", Checkout: "ariadne", State: "resting"}}) || len(one.Claims.Inactive) != 0 {
		t.Fatalf("pair:1 = %+v", one)
	}
	if two := findRow(t, plan, "pair:2"); two.Class != RecoverConflict || !slices.Equal(two.Next.Hold, []string{"conflict:claim-workspace"}) {
		t.Fatalf("pair:2 = %+v", two)
	}
	if five := findRow(t, plan, "pair:5"); five.Class != RecoverDirectoryMissing || five.Disk.Directory != "missing" {
		t.Fatalf("pair:5 = %+v", five)
	}
	if plan.Ignored.OffSlotClaims != 1 {
		t.Fatalf("ignored = %+v", plan.Ignored)
	}
}

func TestDeriveRecoverPlanDedupesSlotsAcrossFleets(t *testing.T) {
	p := newPlanFixture(t)
	claimedOnBranch(p)
	in := p.input()
	second := in.Fleets[0]
	second.Root = "/fleet-copy"
	in.Fleets = append(in.Fleets, second)
	plan := DeriveRecoverPlan(in)
	count := 0
	for _, row := range plan.Rows {
		if row.Address == "pair:1" {
			count++
		}
	}
	if count != 1 || !strings.Contains(plan.Fleets[1].Error, "pair:1") {
		t.Fatalf("rows %d, fleets %+v", count, plan.Fleets)
	}
}

// consistent excludes evidence that cannot be observed together. It only
// prunes the totality test's domain (so it lives with the test); production
// evidence is derived, not checked against it.
func consistent(e SlotEvidence) bool {
	switch {
	case e.Couch == CouchUnavailable && e.Threads != ThreadsZero,
		(e.Agent == AgentNone) != (e.Threads == ThreadsZero),
		e.Threads != ThreadsOne && e.Offer != OfferNone,
		(e.Agent == AgentParked || e.Agent == AgentDetached) && e.Threads == ThreadsOne && e.Offer != OfferResumeReboot,
		e.Agent != AgentUnusable && e.Agent != AgentParked && e.Agent != AgentDetached && e.Offer != OfferNone,
		e.GitSource == GitSourceUnknown && !(e.Branch == BranchUnknown && e.Dirty == TriUnknown && e.Unlanded == TriUnknown && e.Operation == TriUnknown),
		e.GitSource == GitSourceLocalProbe && (e.Unlanded != TriUnknown || e.Operation != TriUnknown || e.Dirty == TriUnknown ||
			e.Branch == BranchTerminalIssue || e.Branch == BranchUnknown || (e.Quality != QualityUnknown && e.Quality != QualityUnsupported)),
		e.GitSource == GitSourceSDLC && e.Quality == QualityUnsupported,
		// An unread host read lists no host claims, but a dependency's weaker
		// read can lower Quality while the host's claims stand: only absent
		// (no tracker) and unsupported (no fleet read) exclude claims.
		// Quality is the host's own read (BR-14): an unread, unsupported or
		// trackerless host lists no host claims.
		(e.Quality == QualityUnknown || e.Quality == QualityUnsupported || e.Quality == QualityAbsent) && e.Claims != ClaimsNone,
		(e.Quality == QualityUnsupported || e.GitSource == GitSourceLocalProbe) && (e.DepClaims != DepNone || e.DepTree != DepTreeClean),
		e.GitSource == GitSourceUnknown && (e.DepClaims != DepNone && e.DepClaims != DepUnknown || e.DepTree != DepTreeClean),
		// A dependency tree with dirt, an operation or unread facts is that
		// member's own work or unread state: its judgment cannot be "nothing".
		e.DepTree != DepTreeClean && e.DepClaims == DepNone,
		// The resting-dirty member is itself a dirty tree.
		e.DepClaims == DepRestingDirty && e.DepTree == DepTreeClean,
		e.Claims == ClaimsNone && e.Workspace,
		(e.Claims == ClaimsOneActive || e.Claims == ClaimsManyWithActive) && (e.Branch == BranchResting || e.Branch == BranchOther || e.Branch == BranchDetached),
		e.Elsewhere && (e.Claims == ClaimsOneActive || e.Claims == ClaimsManyWithActive || e.Branch == BranchResting || e.Branch == BranchOther || e.Branch == BranchDetached),
		e.Dir == DirMissing && e.GitSource != GitSourceUnknown:
		return false
	}
	return true
}

// assertReasonMatchesDecision checks a row's text against its own decision:
// the reason may not deny a claim the row holds, offer a restore request the
// steps lack, or both offer and refuse one; and every dependency claim the
// row does not act on is named in its notes (round 2 minors).
func assertReasonMatchesDecision(t *testing.T, row RecoverRow) {
	t.Helper()
	if row.Reason == "" {
		t.Fatal("row has no reason")
	}
	asks := slices.ContainsFunc(row.Next.Steps, func(s RecoverStep) bool { return s.Action == "ask-agent-restore" })
	claimed := row.Claims.Active != "" || len(row.Claims.Inactive) > 0 || len(row.Claims.Dependency) > 0
	switch {
	case claimed && strings.Contains(row.Reason, "no claim"):
		t.Fatalf("reason %q denies the row's claims %+v", row.Reason, row.Claims)
	case strings.Contains(row.Reason, "ask the slot's agent") != asks:
		t.Fatalf("reason %q disagrees with the steps %+v", row.Reason, row.Next.Steps)
	case asks && strings.Contains(row.Reason, "no restore request"):
		t.Fatalf("reason %q both asks for and refuses a restore", row.Reason)
	}
	if len(row.Next.Hold) > 0 {
		return
	}
	for _, c := range row.Claims.Dependency {
		named := strings.Contains(row.Reason, c.Ref) || slices.ContainsFunc(row.Next.Steps, func(s RecoverStep) bool { return strings.Contains(s.Message, c.Ref) })
		if c.State != "active" && !named && !slices.Contains(row.Next.Notes, "inactive-claims") && !slices.Contains(row.Next.Notes, "dependency-unread") {
			t.Fatalf("dependency claim %+v is in neither the steps nor the notes: %+v", c, row.Next)
		}
	}
}

// TestHostJudgmentsReadOnlyHostFacts is BR-14's rule as a property over the
// whole domain: changing ONLY the dependency facts (their judgment and their
// operation) never changes a judgment about the host -- hostAtRest, the
// issue-terminal conflict fact, or whether a host-only row is idle or landed.
func TestHostJudgmentsReadOnlyHostFacts(t *testing.T) {
	checked := 0
	forEachEvidence(func(e SlotEvidence) {
		if e.DepClaims != DepNone || e.DepTree != DepTreeClean {
			return // each base point once; its dependency variants below
		}
		base := classifyRecover(e)
		claims := e.Claims != ClaimsNone
		read := e.Quality == QualityPresent || e.Quality == QualityStale
		terminal := slices.Contains(conflictFacts(e, claims, read), "issue-terminal")
		for _, dep := range AllEvidenceDepClaims() {
			for _, op := range AllEvidenceDepTrees() {
				v := withDependencyFacts(e, dep, op)
				if !consistent(v) {
					continue
				}
				checked++
				if hostAtRest(v) != hostAtRest(e) {
					t.Fatalf("hostAtRest changed with dependency facts %s/%s: %+v", dep, op, e)
				}
				if slices.Contains(conflictFacts(v, claims, read), "issue-terminal") != terminal {
					t.Fatalf("issue-terminal changed with dependency facts %s/%s: %+v", dep, op, e)
				}
				// Idle and landed are host-only answers: with no dependency
				// finding they are decided by host facts alone, and a
				// dependency operation alone never changes them.
				if dep == DepNone && (base.Class == RecoverIdle || base.Class == RecoverLanded) && classifyRecover(v).Class != base.Class {
					t.Fatalf("%s changed with a dependency operation %s: %+v", base.Class, op, e)
				}
			}
		}
	})
	if checked == 0 {
		t.Fatal("no dependency variants checked")
	}
}

// The host's git facts in the row are the host's own; the dependency's dirt
// shows only in the evidence union and the disk members (BR-14).
func TestDependencyFactsStayOnTheDependency(t *testing.T) {
	p := newPlanFixture(t)
	activeDependency(p, "pair:1", false)
	p.fleet.SetMemberDirty(dependencyPath(p), 2)
	row := findRow(t, DeriveRecoverPlan(p.input()), "pair:1")
	if row.Git.Dirty != "no" || !slices.Contains(row.Evidence, "dirty") {
		t.Fatalf("host dirty %q, evidence %v; want the host clean and the union dirty", row.Git.Dirty, row.Evidence)
	}
}

// restoreWorkspaceDecision's notes never share a backing array with the
// caller's extra notes: a caller slice with spare capacity must survive a
// later append untouched in the decision (M1 review round 4, slice alias).
func TestRestoreWorkspaceNotesDoNotAliasExtra(t *testing.T) {
	e := SlotEvidence{Dir: DirPresent, Couch: CouchOK, Agent: AgentParked, Threads: ThreadsOne, Offer: OfferResumeReboot, Branch: BranchResting,
		Claims: ClaimsOneInactive, DepClaims: DepResting, Quality: QualityPresent, GitSource: GitSourceSDLC, Dirty: TriYes, Unlanded: TriNo, Operation: TriNo}
	for dirt, want := range map[TriState][]RecoverNote{
		TriNo:      {NoteInactiveClaims},
		TriYes:     {NoteInactiveClaims, NoteRestingBranchDirty},
		TriUnknown: {NoteInactiveClaims, NoteGitUnknown},
	} {
		extra := make([]RecoverNote, 1, 8)
		extra[0] = NoteInactiveClaims
		d := restoreWorkspaceDecision(e, dirt, extra)
		_ = append(extra, NoteClaimRepair, NoteClaimRepair)
		if !slices.Equal(d.Notes, want) {
			t.Errorf("dirt %s: notes %v, want %v", dirt, d.Notes, want)
		}
	}
}

// TestRecoverReconcileReadingIsMetamorphic (pair#387): over a deterministic
// stride of the full evidence domain, each slot-reconciler reading changes the
// decision exactly as stated, relative to the same evidence read as converged:
// needs-zero holds the row unless an earlier rule decided it; reconcilable
// turns idle or a missing directory (and nothing else) into the reconcile step; held and unknown add
// their note and change nothing else; reconcile is never stepped otherwise.
func TestRecoverReconcileReadingIsMetamorphic(t *testing.T) {
	earlier := map[RecoverClass]bool{RecoverDirectoryMissing: true, RecoverAgentUnknown: true, RecoverOrphanedServer: true, RecoverStartUnreconciled: true, RecoverAmbiguousThreads: true}
	points := 0
	index := 0
	forEachEvidence(func(e SlotEvidence) {
		index++
		if index%23 != 0 {
			return
		}
		points++
		e.Reconcile = ReconcileConverged
		base := classifyRecover(e)
		if slices.Contains(base.Steps, "reconcile") {
			t.Fatalf("converged workspace stepped reconcile: %+v → %+v", e, base)
		}
		for _, r := range AllEvidenceReconciles() {
			e.Reconcile = r
			d := classifyRecover(e)
			fail := func(why string) { t.Fatalf("%s: %+v (%s) → %+v, converged → %+v", why, e, r, d, base) }
			switch r {
			case ReconcileNeedsZero:
				if earlier[base.Class] {
					if d.Class != base.Class {
						fail("an earlier rule was overridden")
					}
				} else if d.Class != RecoverSlotNeedsZero || !slices.Equal(d.Hold, []string{string(HoldWorkspaceHandoff)}) || len(d.Steps) != 0 {
					fail("needs-zero did not hold the row")
				}
			case ReconcileReconcilable:
				// Idle, or a missing directory the reconciler can re-create.
				if base.Class == RecoverIdle || base.Class == RecoverDirectoryMissing {
					if d.Class != RecoverReconcilable || !slices.Equal(d.Steps, []string{"reconcile"}) {
						fail("a reconcilable workspace with no other step is not stepped")
					}
				} else if d.Class != base.Class || !slices.Equal(d.Steps, base.Steps) {
					fail("reconcilable changed a non-idle decision")
				}
			case ReconcileHeld, ReconcileUnknown, ReconcileDegraded:
				note := map[EvidenceReconcile]RecoverNote{ReconcileHeld: NoteWorkspaceHeld, ReconcileUnknown: NoteWorkspaceUnknown, ReconcileDegraded: NoteWorkspaceDegraded}[r]
				if d.Class != base.Class || !slices.Equal(d.Steps, base.Steps) || !slices.Contains(d.Notes, note) {
					fail("held/unknown must only add its note")
				}
			}
		}
	})
	if points < 50000 {
		t.Fatalf("only %d sampled points; the stride is not covering the domain", points)
	}
}

// TestRecoverSlotClassAgreesWithTheCallers (BR-16): over the planner's
// derived perturbation domain and every agent state, the report holds a slot
// for :0 only when the callers' own SlotOutcome would refuse it, and never
// holds one they would open.
func TestRecoverSlotClassAgreesWithTheCallers(t *testing.T) {
	for _, agent := range AllEvidenceAgents() {
		for _, p := range slotPerturbations() {
			o := perturb(healthyObservation(), p)
			o.Agent = agent
			plan, err := PlanSlot(PlanInput{Observation: o})
			if err != nil {
				t.Fatal(err)
			}
			r := SlotReport{Address: "pair:1", Observation: o, Plan: plan}
			blocking, _ := SlotOutcome("pair:1", "pair", ReconcileResult{Observation: o, Plan: plan}, nil)
			class := RecoverSlotClass(r)
			if class == ReconcileNeedsZero && blocking == nil {
				t.Errorf("agent=%s %s: the report holds a slot the callers would open", agent, p)
			}
			if blocking != nil && len(plan.Steps) == 0 && class == ReconcileConverged {
				t.Errorf("agent=%s %s: the callers refuse a slot the report calls converged", agent, p)
			}
		}
	}
}

// The report names an orphan as such, with its server, and suggests nothing a
// running conversation can't survive: no resume, no reboot (#399 M1).
func TestRecoverPlanNamesAnOrphanedAgent(t *testing.T) {
	p := newPlanFixture(t)
	claimedOnBranch(p)
	p.orphanedThread("pair:1", 4242)
	row := findRow(t, DeriveRecoverPlan(p.input()), "pair:1")
	if row.Agent.State != string(AgentOrphaned) || row.Agent.Orphan == nil || row.Agent.Orphan.PID != 4242 {
		t.Fatalf("agent = %+v", row.Agent)
	}
	if len(row.Next.Steps) != 0 {
		t.Fatalf("steps = %+v, want none until reap exists", row.Next.Steps)
	}
	if !strings.Contains(row.Reason, "server PID 4242 lost its socket") {
		t.Fatalf("reason = %q", row.Reason)
	}
}

// A many-thread slot keeps its most urgent attention state by the one
// agentRank: an orphan outranks an unknown row (#399 M1 review).
func TestManyThreadsWithAnOrphanReadOrphaned(t *testing.T) {
	p := newPlanFixture(t)
	claimedOnBranch(p)
	p.thread("pair:1", ThreadUnusable, ReasonUnknown)
	p.orphanedThread("pair:1", 4242)
	row := findRow(t, DeriveRecoverPlan(p.input()), "pair:1")
	if row.Class != RecoverOrphanedServer || row.Agent.State != string(AgentOrphaned) {
		t.Fatalf("class %q agent %q", row.Class, row.Agent.State)
	}
}
