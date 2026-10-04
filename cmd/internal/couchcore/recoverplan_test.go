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
}

func newPlanFixture(t *testing.T) *planFixture {
	fake := NewFakeFleetSDLC()
	return &planFixture{t: t, fake: fake, fleet: fake.Fleet("/fleet"), root: "/fleet", localGit: map[string]RecoverLocalGit{}}
}

func (p *planFixture) path(address string) string { return p.fleet.SlotPath(address) }

// thread adds one Couch row standing for the slot address.
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
	return RecoverPlanInput{Fleets: []FleetObservation{obs}, Couch: couch, SlotCandidates: p.candidates, LocalGit: p.localGit}
}

type recoverPlanCase struct {
	name    string
	setup   func(*planFixture)
	address string
	want    RecoverClass
	steps   []string
	hold    []string
	notes   []string
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
		{name: "dirty resting branch, no claim", address: "pair:1", want: RecoverUnidentifiedWork, hold: []string{"unidentified-work"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.SetDirty("pair:1", 2)
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "claimed slot with no thread", address: "pair:1", want: RecoverNoCouchThread, hold: []string{"no-couch-thread"},
			setup: claimedOnBranch},
		{name: "one claim on a clean resting branch", address: "pair:1", want: RecoverRestoreWorkspace, steps: []string{"resume", "ask-agent-restore"},
			setup: func(p *planFixture) {
				p.fleet.AddSlot("pair:1")
				p.fleet.Claim("pair:1", "pair#000014")
				p.thread("pair:1", ThreadParked, "")
			}},
		{name: "one claim on a clean resting branch, live", address: "pair:1", want: RecoverRestoreWorkspace, steps: []string{"ask-agent-restore"},
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
			if row.Reason == "" {
				t.Fatal("row has no reason")
			}
			for _, step := range row.Next.Steps {
				if step.Command != "" {
					t.Fatalf("M1 step carries command text %q", step.Command)
				}
				if step.Action == "ask-agent-restore" && step.Message != RestoreWorkspaceMessage(row.Claims.Inactive[0], row.Address) {
					t.Fatalf("restore message = %q", step.Message)
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

// evidenceDomain crosses every value of every SlotEvidence dimension, each
// from its own All* list, keeping only what consistent admits.
func evidenceDomain() []SlotEvidence {
	var out []SlotEvidence
	for _, dir := range AllEvidenceDirs() {
		for _, couch := range AllEvidenceCouch() {
			for _, agent := range AllEvidenceAgents() {
				for _, threads := range AllEvidenceThreads() {
					for _, offer := range AllEvidenceOffers() {
						for _, branch := range AllEvidenceBranches() {
							for _, claims := range AllEvidenceClaims() {
								for _, quality := range AllEvidenceQualities() {
									for _, source := range AllEvidenceGitSources() {
										for _, dirty := range AllEvidenceTriStates() {
											for _, unlanded := range AllEvidenceTriStates() {
												for _, operation := range AllEvidenceTriStates() {
													for _, workspace := range []bool{false, true} {
														for _, elsewhere := range []bool{false, true} {
															e := SlotEvidence{Dir: dir, Couch: couch, Agent: agent, Threads: threads, Offer: offer, Branch: branch, Claims: claims, Quality: quality,
																GitSource: source, Dirty: dirty, Unlanded: unlanded, Operation: operation, Workspace: workspace, Elsewhere: elsewhere}
															if consistent(e) {
																out = append(out, e)
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
	return out
}

func TestDeriveRecoverPlanIsTotalOverTheEvidenceDomain(t *testing.T) {
	start := time.Now()
	domain := evidenceDomain()
	if len(domain) < 1000 {
		t.Fatalf("evidence domain has %d points; consistent() admits too little", len(domain))
	}
	for _, e := range domain {
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
			case "ask-agent-restore":
			default:
				fail("unknown step " + step)
			}
			if step == "reboot" && (e.Operation != TriNo || e.Branch == BranchDetached || e.Branch == BranchUnknown || e.gitUnknown() || e.Agent == AgentBusy || e.Agent == AgentUnusableUnknown) {
				fail("unsafe reboot")
			}
		}
		if d.Class == RecoverIdle && (e.Dirty != TriNo || e.Unlanded != TriNo || e.Operation != TriNo) {
			fail("unknown or present work read as idle")
		}
		clean := e.Dirty == TriNo && e.Unlanded == TriNo && e.Operation == TriNo && e.Claims == ClaimsNone
		if d.Class == RecoverLanded && (!clean || e.Branch != BranchTerminalIssue || len(d.Steps) > 0) {
			fail("landed without a clean, unclaimed done-issue branch")
		}
		if slices.Contains(d.Hold, "conflict:issue-terminal") && !(e.Claims != ClaimsNone || e.Dirty == TriYes || e.Unlanded == TriYes || e.Operation == TriYes) {
			fail("a done-issue branch without dirt, unlanded commits or a claim read as a conflict")
		}
	}
	t.Logf("%d consistent evidence points in %s", len(domain), time.Since(start))
}

// consistent's exclusions are physical facts, each pinned here so the domain
// cannot shrink silently.
func TestConsistentExcludesOnlyImpossibleEvidence(t *testing.T) {
	base := SlotEvidence{Dir: DirPresent, Couch: CouchOK, Agent: AgentParked, Threads: ThreadsOne, Offer: OfferResumeReboot, Branch: BranchOpenIssue,
		Claims: ClaimsOneActive, Quality: QualityPresent, GitSource: GitSourceSDLC, Dirty: TriNo, Unlanded: TriNo, Operation: TriNo}
	if !consistent(base) {
		t.Fatal("the ordinary claimed, parked slot is excluded")
	}
	for name, mutate := range map[string]func(*SlotEvidence){
		"couch unavailable with a thread":    func(e *SlotEvidence) { e.Couch = CouchUnavailable },
		"no agent but a thread":              func(e *SlotEvidence) { e.Agent = AgentNone },
		"parked agent offering reboot only":  func(e *SlotEvidence) { e.Offer = OfferReboot },
		"unknown git source, known branch":   func(e *SlotEvidence) { e.GitSource = GitSourceUnknown },
		"local probe with claims":            func(e *SlotEvidence) { e.GitSource = GitSourceLocalProbe },
		"unread claims listing one":          func(e *SlotEvidence) { e.Quality = QualityUnknown },
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
	if one.Class != RecoverAgrees || !slices.Contains(one.Next.Notes, "inactive-claims") || !slices.Contains(one.Claims.Inactive, "ariadne#000290") {
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
		(e.Quality == QualityUnknown || e.Quality == QualityUnsupported || e.Quality == QualityAbsent) && e.Claims != ClaimsNone,
		e.Claims == ClaimsNone && e.Workspace,
		(e.Claims == ClaimsOneActive || e.Claims == ClaimsManyWithActive) && (e.Branch == BranchResting || e.Branch == BranchOther || e.Branch == BranchDetached),
		e.Elsewhere && (e.Claims == ClaimsOneActive || e.Claims == ClaimsManyWithActive || e.Branch == BranchResting || e.Branch == BranchOther || e.Branch == BranchDetached),
		e.Dir == DirMissing && e.GitSource != GitSourceUnknown:
		return false
	}
	return true
}
