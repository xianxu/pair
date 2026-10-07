package couchcore

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// RecoverPlanSchemaVersion versions the recover-plan report's JSON.
const RecoverPlanSchemaVersion = 1

// RecoverPlanInput is everything DeriveRecoverPlan joins (pair#367). The shell
// (Couch.RecoverPlan) gathers it; the derivation does no IO.
type RecoverPlanInput struct {
	Fleets []FleetObservation
	Couch  CouchObservation
	// SlotCandidates are the slots of an unavailable or unsupported fleet,
	// from its enrolled primaries and EnumerateSlotCandidates.
	SlotCandidates []RecoverSlotCandidate
	// LocalGit is ProbeSlotGit's observation per candidate path.
	LocalGit map[string]RecoverLocalGit
	// SlotPlans is the slot reconciler's report per :1+ host path (pair#387);
	// a slot without one reads as converged.
	SlotPlans map[string]SlotReport
}

// Fleet observation states.
const (
	FleetObservationPresent     = "present"
	FleetObservationUnavailable = "unavailable" // sdlc failed, timed out, overflowed or was not asked
	FleetObservationUnsupported = "unsupported" // another schema, or a v1 build predating #288/#289
)

// FleetObservation is one fleet's sdlc read: its inventory when present, or
// why there is none. An absent inventory never reads as zero slots.
type FleetObservation struct {
	Root      string
	Vantage   string
	State     string
	Error     string
	Inventory FleetInventory
}

// Couch observation states.
const (
	CouchObservationOK          = "ok"
	CouchObservationUnavailable = "unavailable"
)

// CouchObservation is Couch's actionable inventory, or why it is unread.
type CouchObservation struct {
	State string
	Error string
	Rows  []ActionableThreadSummary
}

// RecoverSlotCandidate is one slot of a fleet sdlc could not describe, found
// by Couch's own conventional slot layout.
type RecoverSlotCandidate struct {
	Fleet   string
	Address string
	Path    string
	Missing bool // the host directory is absent
}

// RecoverLocalGit is ProbeSlotGit's answer for one path, or why there is none.
type RecoverLocalGit struct {
	Status SlotGitStatus
	Err    string
}

// RecoverPlan is the report: one row per slot path, plus what each source
// said and what was deliberately left out.
type RecoverPlan struct {
	SchemaVersion int            `json:"schema_version"`
	Fleets        []RecoverFleet `json:"fleets"`
	Couch         RecoverCouch   `json:"couch"`
	Rows          []RecoverRow   `json:"rows"`
	Ignored       RecoverIgnored `json:"ignored"`
}

type RecoverFleet struct {
	Root    string `json:"root"`
	Vantage string `json:"vantage,omitempty"`
	State   string `json:"state"`
	Error   string `json:"error,omitempty"`
}

type RecoverCouch struct {
	State string `json:"state"`
	Error string `json:"error,omitempty"`
}

// RecoverIgnored counts what is not a slot path: claims on non-slot
// worktrees and threads that are not a slot's row.
type RecoverIgnored struct {
	OffSlotClaims  int `json:"off_slot_claims"`
	NonSlotThreads int `json:"non_slot_threads"`
}

// RecoverRow is one slot path's evidence and suggested next step.
type RecoverRow struct {
	Address   string       `json:"address"`
	Path      string       `json:"path"`
	Fleet     string       `json:"fleet,omitempty"`
	Class     RecoverClass `json:"class"`
	Automatic bool         `json:"automatic"`
	Reason    string       `json:"reason"`
	// Evidence is the union of work evidence found in the slot.
	Evidence []string      `json:"evidence"`
	Git      RecoverGit    `json:"git"`
	Disk     RecoverDisk   `json:"disk"`
	Agent    RecoverAgent  `json:"agent"`
	Claims   RecoverClaims `json:"claims"`
	Next     RecoverNext   `json:"next"`
}

type RecoverGit struct {
	Source      string `json:"source"`
	Branch      string `json:"branch,omitempty"`
	State       string `json:"state"`
	Issue       string `json:"issue,omitempty"`
	IssueStatus string `json:"issue_status,omitempty"`
	Dirty       string `json:"dirty"`
	Unlanded    string `json:"unlanded"`
	Operation   string `json:"operation"`
	Error       string `json:"error,omitempty"`
}

type RecoverDisk struct {
	Directory string              `json:"directory"`
	Verdict   string              `json:"verdict,omitempty"`
	Members   []RecoverDiskMember `json:"members,omitempty"`
}

type RecoverDiskMember struct {
	Role    string   `json:"role"`
	Path    string   `json:"path"`
	Verdict string   `json:"verdict"`
	Reasons []string `json:"reasons"`
}

type RecoverAgent struct {
	State   string             `json:"state"`
	Threads int                `json:"threads"`
	Offered []string           `json:"offered"`
	Rows    []RecoverThreadRef `json:"rows,omitempty"`
	// Orphan is the orphaned zellij server when State is "orphaned" (#399).
	Orphan *RecoverOrphan `json:"orphan,omitempty"`
}

// RecoverOrphan names an orphaned server: what reap acts on.
type RecoverOrphan struct {
	PID     int    `json:"pid"`
	Session string `json:"session"`
}

type RecoverThreadRef struct {
	RepoScope string `json:"repo_scope"`
	Tag       string `json:"tag"`
	State     string `json:"state"`
	Reason    string `json:"reason,omitempty"`
}

// RecoverClaims are the host's claims (Active/Inactive, against the host's
// branch) and the dependency members' claims, each against its own member.
type RecoverClaims struct {
	Quality    string               `json:"quality"`
	Error      string               `json:"error,omitempty"`
	Active     string               `json:"active,omitempty"`
	Inactive   []string             `json:"inactive,omitempty"`
	Dependency []RecoverMemberClaim `json:"dependency,omitempty"`
}

// RecoverMemberClaim is one dependency claim: the checkout holding it and how
// that checkout's own branch relates to it (active, resting, other, unknown).
type RecoverMemberClaim struct {
	Ref      string `json:"ref"`
	Checkout string `json:"checkout"`
	State    string `json:"state"`
}

// RecoverNext is the suggestion. Hold and Notes are closed vocabularies
// (AllRecoverHolds, AllRecoverNotes); a parameterized code is kind:suffix.
type RecoverNext struct {
	Steps []RecoverStep `json:"steps"`
	Hold  []string      `json:"hold"`
	Notes []string      `json:"notes"`
}

// RecoverStep is one action: resume or reboot (from ActorActions), or
// ask-agent-restore with the message to send the slot's agent. Command is the
// CLI text that runs it from a live Couch slot (SlotOperationCommand or
// SendToCommand).
type RecoverStep struct {
	Action  string `json:"action"`
	Command string `json:"command,omitempty"`
	Message string `json:"message,omitempty"`
}

// RestoreWorkspaceMessage is the one text a slot's own agent receives when a
// claim sits on its checkout's resting branch. checkout names the member that
// holds the claim (the host repository or a dependency), so the agent restores
// the right repository.
func RestoreWorkspaceMessage(ref, address, checkout string) string {
	return "Recovery (" + address + "): restore this slot's " + checkout + " checkout for " + ref +
		" through sdlc (check out its issue branch); never discard files. Reply with what sdlc issue show reports."
}

// RecoverClass is a row's class; AllRecoverClasses follows the rule order.
type RecoverClass string

const (
	RecoverDirectoryMissing RecoverClass = "directory-missing"
	RecoverAgentUnknown     RecoverClass = "agent-unknown"
	// RecoverOrphanedServer: the agent's zellij server is alive but lost its
	// socket (#399). Its conversation may still be running.
	RecoverOrphanedServer    RecoverClass = "orphaned-server"
	RecoverStartUnreconciled RecoverClass = "start-unreconciled"
	RecoverAmbiguousThreads  RecoverClass = "ambiguous-threads"
	RecoverConflict          RecoverClass = "conflict"
	RecoverAmbiguousClaims   RecoverClass = "ambiguous-claims"
	// RecoverLanded: a clean, unclaimed slot on a done issue's branch, the
	// normal leftover after landing. No step: returning it to rest is not a
	// recovery action.
	RecoverLanded              RecoverClass = "landed"
	RecoverEvidenceUnavailable RecoverClass = "evidence-unavailable"
	RecoverIdle                RecoverClass = "idle"
	RecoverUnidentifiedWork    RecoverClass = "unidentified-work"
	RecoverNoCouchThread       RecoverClass = "no-couch-thread"
	RecoverRestoreWorkspace    RecoverClass = "restore-workspace"
	RecoverAgrees              RecoverClass = "agrees"
	RecoverClaimLikelyLost     RecoverClass = "claim-likely-lost"
	RecoverPartialEvidence     RecoverClass = "partial-evidence"
	RecoverNoSafeStep          RecoverClass = "no-safe-step"
	RecoverNoRule              RecoverClass = "no-rule"
	// RecoverReconcilable: the slot's workspace has converge steps reconcile
	// can run (pair#387); RecoverSlotNeedsZero: it cannot converge, and the
	// repository's :0 agent must look.
	RecoverReconcilable  RecoverClass = "reconcilable"
	RecoverSlotNeedsZero RecoverClass = "slot-needs-zero"
)

func AllRecoverClasses() []RecoverClass {
	return []RecoverClass{RecoverDirectoryMissing, RecoverAgentUnknown, RecoverOrphanedServer, RecoverStartUnreconciled, RecoverAmbiguousThreads,
		RecoverSlotNeedsZero, RecoverReconcilable, RecoverConflict, RecoverAmbiguousClaims, RecoverLanded, RecoverEvidenceUnavailable, RecoverIdle, RecoverUnidentifiedWork,
		RecoverNoCouchThread, RecoverRestoreWorkspace, RecoverAgrees, RecoverClaimLikelyLost, RecoverPartialEvidence,
		RecoverNoSafeStep, RecoverNoRule}
}

// RecoverHold is a hold-code kind: why a row has no automatic step.
type RecoverHold string

const (
	HoldDirectoryMissing      RecoverHold = "directory-missing"
	HoldCouchUnavailable      RecoverHold = "couch-unavailable"
	HoldStartUnreconciled     RecoverHold = "start-unreconciled"
	HoldAgentUnknown          RecoverHold = "agent-unknown"
	HoldOrphanedServer        RecoverHold = "orphaned-server"
	HoldThreads               RecoverHold = "threads"  // threads:<n>
	HoldConflict              RecoverHold = "conflict" // conflict:<fact>
	HoldAmbiguousClaims       RecoverHold = "ambiguous-claims"
	HoldGitUnknown            RecoverHold = "git-unknown"
	HoldUnidentifiedWork      RecoverHold = "unidentified-work"
	HoldNoCouchThread         RecoverHold = "no-couch-thread"
	HoldRebootUnsafeOperation RecoverHold = "reboot-unsafe-operation"
	HoldRebootUnsafeGit       RecoverHold = "reboot-unsafe-git"
	HoldNoActorAction         RecoverHold = "no-actor-action"
	// HoldResumeOnly: the evidence supports resume only, and resume is not
	// offered, so reboot is not suggested in its place.
	HoldResumeOnly RecoverHold = "resume-only"
	HoldNoRule     RecoverHold = "no-rule"
	// HoldWorkspaceHandoff: the slot reconciler cannot converge the workspace
	// (pair#387); the repository's :0 agent investigates.
	HoldWorkspaceHandoff RecoverHold = "workspace-handoff"
)

func AllRecoverHolds() []RecoverHold {
	return []RecoverHold{HoldDirectoryMissing, HoldCouchUnavailable, HoldStartUnreconciled, HoldAgentUnknown, HoldOrphanedServer, HoldThreads,
		HoldConflict, HoldAmbiguousClaims, HoldGitUnknown, HoldUnidentifiedWork, HoldNoCouchThread,
		HoldRebootUnsafeOperation, HoldRebootUnsafeGit, HoldNoActorAction, HoldResumeOnly, HoldWorkspaceHandoff, HoldNoRule}
}

// Conflict facts, in the order a conflict hold lists them.
var recoverConflictFacts = []string{"claim-elsewhere", "issue-terminal", "claim-workspace", "claim-off-branch", "claim-branch-mismatch", "dependency-claim"}

// RecoverNote qualifies a row without holding it.
type RecoverNote string

const (
	NoteInactiveClaims          RecoverNote = "inactive-claims"
	NoteDetachedHead            RecoverNote = "detached-head"
	NoteRestingBranchDirty      RecoverNote = "resting-branch-dirty"
	NoteInspectUncommittedFirst RecoverNote = "inspect-uncommitted-first"
	NoteClaimRepair             RecoverNote = "claim-repair"
	NoteClaimsStale             RecoverNote = "claims-stale"
	NoteClaimsPartial           RecoverNote = "claims-partial"
	NoteClaimsUnknown           RecoverNote = "claims-unknown"
	NoteClaimsAbsent            RecoverNote = "claims-absent"
	NoteClaimsUnsupported       RecoverNote = "claims-unsupported"
	NoteGitLocalProbe           RecoverNote = "git-local-probe"
	NoteGitUnknown              RecoverNote = "git-unknown"
	NoteIssueDoneBranch         RecoverNote = "issue-done-branch"
	// NoteDependencyUnread: a dependency checkout, or a claim on one, could
	// not be read (a missing member holding a claim, or unread facts); it is
	// evidence, not absence.
	NoteDependencyUnread RecoverNote = "dependency-unread"
	// NoteDependencyWork: a dependency checkout carries work of its own
	// (dirt, unlanded commits, an operation, another branch) no claim names.
	NoteDependencyWork RecoverNote = "dependency-work"
	// NoteWorkspaceHeld: the slot's workspace needs repair that waits on its
	// agent (pair#387); NoteWorkspaceUnknown: part of it could not be observed.
	NoteWorkspaceHeld     RecoverNote = "workspace-held"
	NoteWorkspaceUnknown  RecoverNote = "workspace-unknown"
	NoteWorkspaceDegraded RecoverNote = "workspace-degraded" // usable, but something did not converge
)

func AllRecoverNotes() []RecoverNote {
	return []RecoverNote{NoteInactiveClaims, NoteDetachedHead, NoteRestingBranchDirty, NoteInspectUncommittedFirst,
		NoteClaimRepair, NoteClaimsStale, NoteClaimsPartial, NoteClaimsUnknown, NoteClaimsAbsent, NoteClaimsUnsupported,
		NoteGitLocalProbe, NoteGitUnknown, NoteIssueDoneBranch, NoteDependencyUnread, NoteDependencyWork,
		NoteWorkspaceHeld, NoteWorkspaceUnknown, NoteWorkspaceDegraded}
}

// SlotEvidence dimensions. Every dimension is closed; unknown is a value.
type (
	EvidenceDir       string
	EvidenceCouch     string
	EvidenceAgent     string
	EvidenceThreads   string
	EvidenceOffer     string
	EvidenceBranch    string
	EvidenceClaims    string
	EvidenceQuality   string
	EvidenceGitSource string
	EvidenceDepClaims string
	EvidenceDepTree   string
	TriState          string
)

const (
	DirPresent EvidenceDir = "present"
	DirMissing EvidenceDir = "missing"

	CouchOK          EvidenceCouch = "ok"
	CouchUnavailable EvidenceCouch = "unavailable"

	AgentNone            EvidenceAgent = "none"
	AgentLive            EvidenceAgent = "live"
	AgentDetached        EvidenceAgent = "detached"
	AgentParked          EvidenceAgent = "parked"
	AgentBusy            EvidenceAgent = "busy"
	AgentUnusable        EvidenceAgent = "unusable"
	AgentUnusableUnknown EvidenceAgent = "unusable-unknown"
	// AgentOrphaned: the thread's zellij server is alive, its socket gone (#399).
	AgentOrphaned EvidenceAgent = "orphaned"

	ThreadsZero EvidenceThreads = "0"
	ThreadsOne  EvidenceThreads = "1"
	ThreadsMany EvidenceThreads = "many"

	// Offer is what ActorActions offers the single joined row.
	OfferNone         EvidenceOffer = "none"
	OfferReboot       EvidenceOffer = "reboot"
	OfferResumeReboot EvidenceOffer = "resume-reboot"
	// OfferReap is an orphaned row: reap, and nothing else, is offered (#399).
	OfferReap EvidenceOffer = "reap"

	BranchResting       EvidenceBranch = "resting"
	BranchOpenIssue     EvidenceBranch = "open-issue"
	BranchTerminalIssue EvidenceBranch = "terminal-issue"
	BranchOther         EvidenceBranch = "other"
	BranchDetached      EvidenceBranch = "detached"
	BranchUnknown       EvidenceBranch = "unknown"

	ClaimsNone              EvidenceClaims = "none"
	ClaimsOneActive         EvidenceClaims = "one-active" // its issue is the checked-out branch
	ClaimsOneInactive       EvidenceClaims = "one-inactive"
	ClaimsManyWithActive    EvidenceClaims = "many-with-active"
	ClaimsManyWithoutActive EvidenceClaims = "many-without-active"

	QualityPresent     EvidenceQuality = "present"
	QualityStale       EvidenceQuality = "stale"
	QualityPartial     EvidenceQuality = "partial"
	QualityUnknown     EvidenceQuality = "unknown"
	QualityAbsent      EvidenceQuality = "absent"
	QualityUnsupported EvidenceQuality = "unsupported"

	GitSourceSDLC       EvidenceGitSource = "sdlc"
	GitSourceLocalProbe EvidenceGitSource = "local-probe"
	GitSourceUnknown    EvidenceGitSource = "unknown"

	// DepClaims: the dependency members, each judged on its OWN facts (its
	// branch, dirt, unlanded commits, operation and claims; pair#367 BR-4,
	// BR-14), never the host's, then folded. Host dimensions never read
	// dependency facts.
	DepNone         EvidenceDepClaims = "none"          // clean, unclaimed (or absent without a claim)
	DepActive       EvidenceDepClaims = "active"        // every claim on its own member's issue branch
	DepResting      EvidenceDepClaims = "resting"       // one claim on its clean member's resting branch
	DepRestingDirty EvidenceDepClaims = "resting-dirty" // the same, its member dirty
	DepConflict     EvidenceDepClaims = "conflict"      // a claim on another branch, or several resting
	DepWork         EvidenceDepClaims = "work"          // a member's own work no claim names
	DepUnknown      EvidenceDepClaims = "unknown"       // a member, or the claim on it, unread or gone

	// DepTree folds every dependency checkout's working tree into one
	// slot-level fact, worst first: an operation in progress, then anything
	// unread, then dirt. Only slot-level judgments read it (rule A's reboot
	// guard and its inspect-uncommitted note: a reboot replaces the whole
	// slot's agent), never a judgment about one member (M1 review round 4).
	DepTreeClean     EvidenceDepTree = "clean"
	DepTreeDirty     EvidenceDepTree = "dirty"
	DepTreeUnknown   EvidenceDepTree = "unknown"
	DepTreeOperation EvidenceDepTree = "operation"

	TriYes     TriState = "yes"
	TriNo      TriState = "no"
	TriUnknown TriState = "unknown"
)

func AllEvidenceDirs() []EvidenceDir    { return []EvidenceDir{DirPresent, DirMissing} }
func AllEvidenceCouch() []EvidenceCouch { return []EvidenceCouch{CouchOK, CouchUnavailable} }
func AllEvidenceAgents() []EvidenceAgent {
	return []EvidenceAgent{AgentNone, AgentLive, AgentDetached, AgentParked, AgentBusy, AgentUnusable, AgentUnusableUnknown, AgentOrphaned}
}
func AllEvidenceThreads() []EvidenceThreads {
	return []EvidenceThreads{ThreadsZero, ThreadsOne, ThreadsMany}
}
func AllEvidenceOffers() []EvidenceOffer {
	return []EvidenceOffer{OfferNone, OfferReboot, OfferResumeReboot, OfferReap}
}
func AllEvidenceBranches() []EvidenceBranch {
	return []EvidenceBranch{BranchResting, BranchOpenIssue, BranchTerminalIssue, BranchOther, BranchDetached, BranchUnknown}
}
func AllEvidenceClaims() []EvidenceClaims {
	return []EvidenceClaims{ClaimsNone, ClaimsOneActive, ClaimsOneInactive, ClaimsManyWithActive, ClaimsManyWithoutActive}
}
func AllEvidenceQualities() []EvidenceQuality {
	return []EvidenceQuality{QualityPresent, QualityStale, QualityPartial, QualityUnknown, QualityAbsent, QualityUnsupported}
}
func AllEvidenceGitSources() []EvidenceGitSource {
	return []EvidenceGitSource{GitSourceSDLC, GitSourceLocalProbe, GitSourceUnknown}
}
func AllEvidenceTriStates() []TriState { return []TriState{TriYes, TriNo, TriUnknown} }
func AllEvidenceDepClaims() []EvidenceDepClaims {
	return []EvidenceDepClaims{DepNone, DepActive, DepResting, DepRestingDirty, DepConflict, DepWork, DepUnknown}
}
func AllEvidenceDepTrees() []EvidenceDepTree {
	return []EvidenceDepTree{DepTreeClean, DepTreeDirty, DepTreeUnknown, DepTreeOperation}
}

func (o EvidenceOffer) actions() []string {
	switch o {
	case OfferReboot:
		return []string{"reboot"}
	case OfferResumeReboot:
		return []string{"resume", "reboot"}
	case OfferReap:
		return []string{"reap"}
	}
	return nil
}

func offerOf(actions []string) EvidenceOffer {
	switch {
	case slices.Contains(actions, "reap"):
		return OfferReap
	case slices.Contains(actions, "resume"):
		return OfferResumeReboot
	case slices.Contains(actions, "reboot"):
		return OfferReboot
	}
	return OfferNone
}

// SlotEvidence is one slot's facts over closed dimensions: the whole input of
// classifyRecover.
type SlotEvidence struct {
	Dir     EvidenceDir
	Couch   EvidenceCouch
	Agent   EvidenceAgent
	Threads EvidenceThreads
	Offer   EvidenceOffer
	Branch  EvidenceBranch
	// Claims are the host's claims, judged against the host's branch.
	Claims    EvidenceClaims
	DepClaims EvidenceDepClaims
	// DepTree is the dependency checkouts' working trees folded into one
	// slot-level fact, read only through the slot* helpers below.
	DepTree EvidenceDepTree
	// Quality, Dirty, Unlanded and Operation are the HOST's own facts.
	Quality   EvidenceQuality
	GitSource EvidenceGitSource
	Dirty     TriState
	Unlanded  TriState
	Operation TriState
	// Workspace: a host claim records a non-empty workspace other than this
	// slot's address (a dependency or plain clone records an empty one).
	Workspace bool
	// Elsewhere: the branch issue's claim sits on another path.
	Elsewhere bool
	// Reconcile is the slot reconciler's reading of the workspace (pair#387).
	Reconcile EvidenceReconcile
}

// EvidenceReconcile is RecoverSlotClass's reading of a slot plan.
type EvidenceReconcile string

const (
	ReconcileConverged    EvidenceReconcile = "converged"
	ReconcileReconcilable EvidenceReconcile = "reconcilable"
	ReconcileHeld         EvidenceReconcile = "held"
	ReconcileNeedsZero    EvidenceReconcile = "needs-zero"
	ReconcileDegraded     EvidenceReconcile = "degraded"
	ReconcileUnknown      EvidenceReconcile = "unknown"
)

func AllEvidenceReconciles() []EvidenceReconcile {
	return []EvidenceReconcile{ReconcileConverged, ReconcileReconcilable, ReconcileHeld, ReconcileNeedsZero, ReconcileDegraded, ReconcileUnknown}
}

// RecoverSlotClass reads a slot reconciler report for the recovery report
// through the callers' own decision (SlotOutcome over the report's plan), so
// the report never holds a slot its callers would open: a hold waits on the
// agent; converge steps are reconcilable (their blocking state is what
// reconcile fixes); a hand-off on a slot no agent could work in is :0's; one
// on a usable slot is degraded (noted); an unobservable resource is noted.
func RecoverSlotClass(r SlotReport) EvidenceReconcile {
	if r.PlanError != "" {
		return ReconcileConverged // :0 or not a slot: nothing to reconcile
	}
	stops := map[StopClass]bool{}
	for _, s := range r.Plan.Stops {
		stops[s.Class] = true
	}
	repo, _, _ := strings.Cut(r.Address, ":")
	blocking, _ := SlotOutcome(r.Address, repo, ReconcileResult{Observation: r.Observation, Plan: r.Plan}, nil)
	switch {
	case stops[StopHold]:
		return ReconcileHeld
	case len(r.Plan.Steps) > 0:
		return ReconcileReconcilable
	case blocking != nil && (stops[StopHandoff] || len(r.Plan.Stops) == 0):
		return ReconcileNeedsZero
	case stops[StopHandoff]:
		return ReconcileDegraded
	case stops[StopUnknown] || stops[StopRetryable]:
		return ReconcileUnknown
	}
	return ReconcileConverged
}

func (e SlotEvidence) gitUnknown() bool {
	return e.Branch == BranchUnknown || e.Dirty == TriUnknown || e.Unlanded == TriUnknown || e.Operation == TriUnknown
}

// Reboot safety is a SLOT-level property: a reboot replaces the agent of the
// whole slot, so it folds every member's tree, the host's and each
// dependency's, where host-only judgments (hostAtRest, the rule table's host
// dimensions) read the host's facts alone. These three are the only readers
// of DepTree.
func (e SlotEvidence) slotOperation() bool {
	return e.Operation == TriYes || e.DepTree == DepTreeOperation
}

func (e SlotEvidence) slotGitUnknown() bool {
	return e.Branch == BranchDetached || e.gitUnknown() || e.DepTree == DepTreeUnknown || e.DepClaims == DepUnknown
}

func (e SlotEvidence) slotDirty() bool {
	return e.Dirty == TriYes || e.DepTree == DepTreeDirty
}

// recoverDecision is classifyRecover's answer, before row text is attached.
type recoverDecision struct {
	Class RecoverClass
	Steps []string
	Hold  []string // kind, or kind:suffix for conflicts; threads is suffixed by the caller
	Notes []RecoverNote
}

// classifyRecover is the rule table of pair#367 Task 1.5: the first matching
// rule decides the class; steps come only from rule A over ActorActions' offer.
func classifyRecover(e SlotEvidence) recoverDecision {
	claims := e.Claims != ClaimsNone
	// work: a host claim, or anything a dependency member's own judgment
	// found (a claim, its own unclaimed work, or an unread member).
	work := claims || e.DepClaims != DepNone
	read := e.Quality == QualityPresent || e.Quality == QualityStale
	var d recoverDecision
	switch {
	case e.Dir == DirMissing && e.Reconcile == ReconcileReconcilable:
		// A slot whose directory is gone but whose reconciler has a plan
		// (its leftovers are there) is re-created by reconcile.
		d = recoverDecision{Class: RecoverReconcilable, Steps: []string{"reconcile"}}
	case e.Dir == DirMissing:
		d = recoverDecision{Class: RecoverDirectoryMissing, Hold: []string{string(HoldDirectoryMissing)}}
	case e.Couch == CouchUnavailable:
		d = recoverDecision{Class: RecoverAgentUnknown, Hold: []string{string(HoldCouchUnavailable)}}
	case e.Agent == AgentBusy:
		d = recoverDecision{Class: RecoverStartUnreconciled, Hold: []string{string(HoldStartUnreconciled)}}
	case e.Agent == AgentUnusableUnknown:
		d = recoverDecision{Class: RecoverAgentUnknown, Hold: []string{string(HoldAgentUnknown)}}
	case e.Agent == AgentOrphaned && e.Offer == OfferReap:
		// The confirmed reap ends the orphaned tree; the thread then reads like
		// any thread whose session ended, so resume follows (#399). Never
		// reboot: that would archive a conversation still being written.
		d = recoverDecision{Class: RecoverOrphanedServer, Steps: []string{"reap", "resume"}}
	case e.Agent == AgentOrphaned:
		d = recoverDecision{Class: RecoverOrphanedServer, Hold: []string{string(HoldOrphanedServer)}}
	case e.Threads == ThreadsMany:
		d = recoverDecision{Class: RecoverAmbiguousThreads, Hold: []string{string(HoldThreads)}}
	case e.Reconcile == ReconcileNeedsZero:
		// Reconcile cannot converge the workspace, so resume or reboot would
		// fail at it too: the :0 agent looks first.
		d = recoverDecision{Class: RecoverSlotNeedsZero, Hold: []string{string(HoldWorkspaceHandoff)}}
	case len(conflictFacts(e, claims, read)) > 0:
		d = recoverDecision{Class: RecoverConflict}
		for _, fact := range conflictFacts(e, claims, read) {
			d.Hold = append(d.Hold, string(HoldConflict)+":"+fact)
		}
	case e.Claims == ClaimsManyWithoutActive:
		d = recoverDecision{Class: RecoverAmbiguousClaims, Hold: []string{string(HoldAmbiguousClaims)}}
	case !work && e.Branch == BranchTerminalIssue && hostAtRest(e):
		d = recoverDecision{Class: RecoverLanded, Notes: []RecoverNote{NoteIssueDoneBranch}}
	case !work && e.Branch != BranchOpenIssue && e.gitUnknown():
		d = recoverDecision{Class: RecoverEvidenceUnavailable, Hold: []string{string(HoldGitUnknown)}}
	case !work && e.Branch == BranchResting && hostAtRest(e):
		d = recoverDecision{Class: RecoverIdle}
	case !work && e.Branch != BranchOpenIssue:
		// resting with work, another branch or a detached HEAD
		d = recoverDecision{Class: RecoverUnidentifiedWork, Hold: []string{string(HoldUnidentifiedWork)}}
	case e.Agent == AgentNone:
		d = recoverDecision{Class: RecoverNoCouchThread, Hold: []string{string(HoldNoCouchThread)}}
	case e.Claims == ClaimsOneInactive && e.Branch == BranchResting:
		// The host's claim is restored; a dependency claim beside it is noted.
		d = restoreWorkspaceDecision(e, e.Dirty, depNotes(e))
	case e.Claims == ClaimsOneActive || e.Claims == ClaimsManyWithActive || e.Claims == ClaimsOneInactive && e.Branch == BranchDetached:
		notes := depNotes(e)
		if e.Claims == ClaimsManyWithActive {
			notes = append(notes, NoteInactiveClaims)
		}
		if e.Branch == BranchDetached {
			notes = append(notes, NoteDetachedHead)
		}
		if e.gitUnknown() {
			notes = append(notes, NoteGitUnknown)
		}
		d = withRuleA(RecoverAgrees, e, false, notes)
	case e.Branch == BranchOpenIssue && !claims && read && e.GitSource == GitSourceSDLC && !e.gitUnknown():
		d = withRuleA(RecoverClaimLikelyLost, e, false, append(depNotes(e), NoteClaimRepair))
	case e.Branch == BranchOpenIssue || claims && e.Branch == BranchUnknown:
		notes := depNotes(e)
		switch e.Quality {
		case QualityPartial:
			notes = append(notes, NoteClaimsPartial)
		case QualityUnknown:
			notes = append(notes, NoteClaimsUnknown)
		case QualityAbsent:
			notes = append(notes, NoteClaimsAbsent)
		case QualityUnsupported:
			notes = append(notes, NoteClaimsUnsupported)
		}
		if e.GitSource == GitSourceLocalProbe {
			notes = append(notes, NoteGitLocalProbe)
		}
		if e.gitUnknown() {
			notes = append(notes, NoteGitUnknown)
		}
		d = withRuleA(RecoverPartialEvidence, e, true, notes)
	case e.DepClaims != DepNone:
		// Only dependency claims, the host off any issue branch: each claim was
		// judged against its own member, so the host's branch decides only
		// whether the host itself is clean and at rest.
		d = dependencyClaimDecision(e)
	default:
		d = recoverDecision{Class: RecoverNoRule, Hold: []string{string(HoldNoRule)}}
	}
	if e.Quality == QualityStale {
		d.Notes = append(d.Notes, NoteClaimsStale)
	}
	d.Notes = sortedNotes(d.Notes)
	// Open, resume and reboot reconcile the workspace first, so a fixable
	// workspace needs its own step only where the row has none (idle); a
	// repair waiting on a live agent, or a resource not observed, is noted.
	switch e.Reconcile {
	case ReconcileReconcilable:
		if d.Class == RecoverIdle {
			d = recoverDecision{Class: RecoverReconcilable, Steps: []string{"reconcile"}, Notes: d.Notes}
		}
	case ReconcileHeld:
		d.Notes = append(d.Notes, NoteWorkspaceHeld)
	case ReconcileDegraded:
		d.Notes = append(d.Notes, NoteWorkspaceDegraded)
	case ReconcileUnknown:
		d.Notes = append(d.Notes, NoteWorkspaceUnknown)
	}
	return d
}

func conflictFacts(e SlotEvidence, claims, read bool) []string {
	applies := map[string]bool{
		"claim-elsewhere": e.Elsewhere,
		// A done issue's branch conflicts only when work is still on it; a
		// clean, unclaimed one is the leftover of landing (RecoverLanded).
		"issue-terminal":        e.Branch == BranchTerminalIssue && (claims || hostCarriesWork(e)),
		"claim-workspace":       e.Workspace,
		"claim-off-branch":      claims && e.Branch == BranchOther,
		"claim-branch-mismatch": claims && e.Branch == BranchOpenIssue && (e.Claims == ClaimsOneInactive || e.Claims == ClaimsManyWithoutActive) && read,
		"dependency-claim":      e.DepClaims == DepConflict,
	}
	var facts []string
	for _, fact := range recoverConflictFacts {
		if applies[fact] {
			facts = append(facts, fact)
		}
	}
	return facts
}

// hostAtRest is the one reading of "the host carries no work of its own": on
// its resting branch or on a done issue's branch (landed), with no dirt,
// nothing unlanded and no operation in progress. Every fact it reads is the
// host's own; a dependency's work is judged on that dependency (DepClaims),
// never folded into the host (BR-10, BR-14). Rules idle and landed, the
// issue-terminal conflict fact and dependencyClaimDecision all read it.
func hostAtRest(e SlotEvidence) bool {
	return (e.Branch == BranchResting || e.Branch == BranchTerminalIssue) && e.Dirty == TriNo && e.Unlanded == TriNo && e.Operation == TriNo
}

// hostCarriesWork: the host's own facts show work (dirt, unlanded commits or
// an operation), read for the issue-terminal conflict.
func hostCarriesWork(e SlotEvidence) bool {
	return e.Dirty == TriYes || e.Unlanded == TriYes || e.Operation == TriYes
}

// depNotes qualifies a host-decided row by its dependency members: a claim
// resting on its member's resting branch is listed as inactive (no step of
// its own); an unread member and a member's unclaimed work are named.
func depNotes(e SlotEvidence) []RecoverNote {
	switch e.DepClaims {
	case DepResting, DepRestingDirty:
		return []RecoverNote{NoteInactiveClaims}
	case DepUnknown:
		return []RecoverNote{NoteDependencyUnread}
	case DepWork:
		return []RecoverNote{NoteDependencyWork}
	}
	return nil
}

// dependencyClaimDecision classifies a slot whose only claims are its
// dependencies', with the host not on an issue branch.
func dependencyClaimDecision(e SlotEvidence) recoverDecision {
	switch {
	case e.DepClaims == DepWork:
		// A dependency's own work that no claim names.
		return recoverDecision{Class: RecoverUnidentifiedWork, Hold: []string{string(HoldUnidentifiedWork)}, Notes: []RecoverNote{NoteDependencyWork}}
	case hostAtRest(e):
		var landed []RecoverNote
		if e.Branch == BranchTerminalIssue {
			landed = []RecoverNote{NoteIssueDoneBranch}
		}
		switch e.DepClaims {
		case DepActive:
			return withRuleA(RecoverAgrees, e, false, landed)
		case DepResting:
			// The dirt that decides a restore is the holding member's own.
			return restoreWorkspaceDecision(e, TriNo, landed)
		case DepRestingDirty:
			return restoreWorkspaceDecision(e, TriYes, landed)
		}
		return withRuleA(RecoverPartialEvidence, e, true, append(landed, NoteDependencyUnread))
	case e.gitUnknown():
		return withRuleA(RecoverPartialEvidence, e, true, append(depNotes(e), NoteGitUnknown))
	}
	// The host carries work of its own (another branch, a detached HEAD,
	// unlanded commits or an operation) that no claim names.
	return recoverDecision{Class: RecoverUnidentifiedWork, Hold: []string{string(HoldUnidentifiedWork)}}
}

// restoreWorkspaceDecision: one claim on the resting branch. The slot's agent
// is asked to restore its issue branch, unless uncommitted (or unread) files
// could make the switch fail or carry edits across: then resume only.
// extra carries the notes of claims the restore does not ask about, so every
// claim in the slot shows in the row's steps or notes.
// dirt is the restoring member's own: the host's for a host claim, the
// holding dependency's for a dependency claim.
func restoreWorkspaceDecision(e SlotEvidence, dirt TriState, extra []RecoverNote) recoverDecision {
	// Built fresh (slices.Concat never shares extra's backing array): dirt
	// names why there is no restore request, unread names the unread tree.
	switch dirt {
	case TriYes:
		return withRuleA(RecoverRestoreWorkspace, e, true, slices.Concat(extra, []RecoverNote{NoteRestingBranchDirty}))
	case TriUnknown:
		return withRuleA(RecoverRestoreWorkspace, e, true, slices.Concat(extra, []RecoverNote{NoteGitUnknown}))
	}
	d := withRuleA(RecoverRestoreWorkspace, e, false, extra)
	if d.Class == RecoverRestoreWorkspace {
		d.Steps = append(d.Steps, "ask-agent-restore")
	}
	return d
}

// withRuleA attaches rule A's actor step to a class, or turns the row into
// no-safe-step when no step is safe. resumeOnly forbids reboot.
func withRuleA(class RecoverClass, e SlotEvidence, resumeOnly bool, notes []RecoverNote) recoverDecision {
	notes = slices.Clone(notes) // the decision owns its notes; never the caller's array
	d := recoverDecision{Class: class, Notes: notes}
	if e.Agent == AgentLive {
		return d
	}
	offered := e.Offer.actions()
	var hold RecoverHold
	switch {
	case slices.Contains(offered, "resume"):
		d.Steps = []string{"resume"}
		return d
	case !slices.Contains(offered, "reboot"):
		hold = HoldNoActorAction
	case resumeOnly:
		hold = HoldResumeOnly
	case e.slotOperation():
		hold = HoldRebootUnsafeOperation
	case e.slotGitUnknown():
		hold = HoldRebootUnsafeGit
	default:
		d.Steps = []string{"reboot"}
		if e.slotDirty() {
			d.Notes = append(d.Notes, NoteInspectUncommittedFirst)
		}
		return d
	}
	return recoverDecision{Class: RecoverNoSafeStep, Hold: []string{string(hold)}, Notes: notes}
}

func sortedNotes(notes []RecoverNote) []RecoverNote {
	order := AllRecoverNotes()
	sort.SliceStable(notes, func(i, j int) bool { return slices.Index(order, notes[i]) < slices.Index(order, notes[j]) })
	return slices.Compact(notes)
}

// recoverSlot is one address of the slot universe and every source's view of it.
type recoverSlot struct {
	address    string
	repo       string
	number     int
	path       string
	fleet      string
	fleetState string
	slot       *FleetSlot
	inv        *FleetInventory
	rows       map[string]FleetRow
	candidate  *RecoverSlotCandidate
	dangling   []FleetDanglingClaim
	threads    []ActionableThreadSummary
}

var issueBranchPattern = regexp.MustCompile(`^([0-9]{6})-`)

// issueTerminalStatuses mirrors ariadne's issue vocabulary terminal set
// (construct/vocabulary/issue.cue). Deferred (#367 M1 review): sdlc should
// emit terminality on FleetIssue so Couch stops restating it.
var issueTerminalStatuses = []string{"done", "wontfix", "punt"}

// DeriveRecoverPlan joins the fleet and Couch observations into one row per
// slot path (pair#367). Pure: no IO.
func DeriveRecoverPlan(in RecoverPlanInput) RecoverPlan {
	plan := RecoverPlan{SchemaVersion: RecoverPlanSchemaVersion, Rows: []RecoverRow{}, Couch: RecoverCouch{State: in.Couch.State, Error: in.Couch.Error}}
	universe := map[string]*recoverSlot{}
	byPath := map[string]string{}
	add := func(s *recoverSlot) {
		universe[s.address] = s
		byPath[filepath.Clean(s.path)] = s.address
	}
	for i := range in.Fleets {
		obs := &in.Fleets[i]
		fleet := RecoverFleet{Root: obs.Root, Vantage: obs.Vantage, State: obs.State, Error: obs.Error}
		if obs.State == FleetObservationPresent {
			inv := &obs.Inventory
			rows := map[string]FleetRow{}
			for _, row := range inv.Rows {
				rows[filepath.Clean(row.TreePath)] = row
			}
			members := map[string]bool{}
			var duplicates []string
			for j := range inv.Slots {
				slot := &inv.Slots[j]
				if _, seen := universe[slot.Address]; seen {
					duplicates = append(duplicates, slot.Address)
					continue
				}
				host := ""
				if len(slot.Members) > 0 {
					host = slot.Members[0].Path
				}
				for _, m := range slot.Members {
					members[filepath.Clean(m.Path)] = true
				}
				add(&recoverSlot{address: slot.Address, repo: slot.Repo, number: slot.Slot, path: filepath.Clean(host), fleet: obs.Root, fleetState: obs.State, slot: slot, inv: inv, rows: rows})
			}
			for _, row := range inv.Rows {
				if !members[filepath.Clean(row.TreePath)] {
					plan.Ignored.OffSlotClaims += len(row.Claims)
				}
			}
			for _, claim := range inv.DanglingClaims {
				slot, ok := conventionalSlotOfMember(filepath.Clean(claim.Claimant.Worktree))
				if !ok {
					plan.Ignored.OffSlotClaims++
					continue
				}
				address := WorkspaceReference{Repo: slot.Repo, Number: slot.Number}.String()
				s := universe[address]
				if s == nil {
					s = &recoverSlot{address: address, repo: slot.Repo, number: slot.Number, path: slot.WorktreeRoot, fleet: obs.Root, fleetState: obs.State, inv: inv}
					add(s)
				}
				s.dangling = append(s.dangling, claim)
			}
			if len(duplicates) > 0 {
				msg := "duplicate slots (first fleet wins): " + strings.Join(duplicates, ", ")
				fleet.Error = strings.TrimPrefix(fleet.Error+"; "+msg, "; ")
			}
		}
		plan.Fleets = append(plan.Fleets, fleet)
	}
	for i := range in.SlotCandidates {
		c := &in.SlotCandidates[i]
		if _, seen := universe[c.Address]; seen {
			continue
		}
		repo, number := splitRecoverAddress(c.Address)
		state := FleetObservationUnavailable
		for _, obs := range in.Fleets {
			if obs.Root == c.Fleet {
				state = obs.State
			}
		}
		add(&recoverSlot{address: c.Address, repo: repo, number: number, path: filepath.Clean(c.Path), fleet: c.Fleet, fleetState: state, candidate: c})
	}
	if in.Couch.State == CouchObservationOK {
		for _, row := range in.Couch.Rows {
			if row.Target.Kind == ThreadTargetSlot {
				slot := row.Target.Slot
				address, ok := byPath[filepath.Clean(slot.WorktreeRoot)]
				if !ok {
					address = WorkspaceReference{Repo: slot.Repo, Number: slot.Number}.String()
					if universe[address] == nil {
						add(&recoverSlot{address: address, repo: slot.Repo, number: slot.Number, path: filepath.Clean(slot.WorktreeRoot)})
					}
				}
				universe[address].threads = append(universe[address].threads, row)
				continue
			}
			joined := false
			for _, s := range universe {
				if s.number != 0 {
					continue
				}
				scope, err := launcher.ResolveRepoScope(s.path)
				if err == nil && IsPrimaryRow(row, s.path, scope.Key) {
					s.threads = append(s.threads, row)
					joined = true
					break
				}
			}
			if !joined {
				plan.Ignored.NonSlotThreads++
			}
		}
	}
	for _, s := range universe {
		plan.Rows = append(plan.Rows, recoverRowOf(s, in))
	}
	sort.Slice(plan.Rows, func(i, j int) bool {
		ri, ni := splitRecoverAddress(plan.Rows[i].Address)
		rj, nj := splitRecoverAddress(plan.Rows[j].Address)
		if ri != rj {
			return ri < rj
		}
		return ni < nj
	})
	return plan
}

// conventionalSlotOfMember recognizes a slot host path or a dependency
// checkout beside one (worktree/<repo>-slotN/<dep>).
func conventionalSlotOfMember(path string) (SlotIdentity, bool) {
	primary, n, ok := ParseSlotPath(path)
	if !ok {
		return SlotIdentity{}, false
	}
	slot := conventionalSlot(primary, n)
	// The host itself, or a checkout directly beside it in the environment.
	if path != slot.WorktreeRoot && filepath.Dir(path) != slot.EnvironmentRoot {
		return SlotIdentity{}, false
	}
	return slot, slot.Validate() == nil
}

func splitRecoverAddress(address string) (string, int) {
	repo, number, _ := strings.Cut(address, ":")
	n, _ := strconv.Atoi(number)
	return repo, n
}

// slotFacts is slotEvidenceOf's full reading: the evidence plus the values
// the row displays.
type slotFacts struct {
	evidence    SlotEvidence
	git         RecoverGit
	disk        RecoverDisk
	agent       RecoverAgent
	claims      RecoverClaims
	work        []string
	threadCount int
	// restoreRef/restoreCheckout name the claim ask-agent-restore asks about
	// and the checkout holding it.
	restoreRef, restoreCheckout string
}

// slotEvidenceOf reads one slot's sources into closed evidence (pure).
func slotEvidenceOf(s *recoverSlot, in RecoverPlanInput) slotFacts {
	var f slotFacts
	e := &f.evidence
	e.Dir = DirPresent
	e.Reconcile = ReconcileConverged
	if report, ok := in.SlotPlans[filepath.Clean(s.path)]; ok && s.number > 0 {
		e.Reconcile = RecoverSlotClass(report)
	}
	switch {
	case s.slot != nil:
	case s.candidate != nil:
		if s.candidate.Missing {
			e.Dir = DirMissing
		}
	case len(s.dangling) > 0:
		e.Dir = DirMissing
	default:
		for _, row := range s.threads {
			if row.State == ThreadUnusable && row.Reason == ReasonPathMissing {
				e.Dir = DirMissing
			}
		}
	}
	f.disk.Directory = string(e.Dir)

	// Git, from sdlc's slot members, else the local probe, else unknown.
	e.GitSource, e.Branch, e.Dirty, e.Unlanded, e.Operation = GitSourceUnknown, BranchUnknown, TriUnknown, TriUnknown, TriUnknown
	e.DepTree = DepTreeClean
	resting := RestingBranch(s.number)
	activeRef := ""
	var claimRefs []string // the host's
	var depClaims []RecoverMemberClaim
	var deps []dependencyJudgment
	var hostClaims []FleetClaim
	e.Quality = QualityUnknown
	claimsError := ""
	switch {
	case s.slot != nil:
		f.disk.Verdict = s.slot.Verdict
		if !knownFleetVerdicts[s.slot.Verdict] {
			f.disk.Verdict = FleetVerdictUnknown
		}
		// Every member is judged on its own facts (BR-14): the host's fill the
		// host dimensions, each dependency's its own judgment. The union
		// across members is built only for the row's evidence list.
		for i, m := range s.slot.Members {
			f.disk.Members = append(f.disk.Members, RecoverDiskMember{Role: m.Role, Path: m.Path, Verdict: m.Verdict, Reasons: append([]string{}, m.Reasons...)})
			row, read := s.rows[filepath.Clean(m.Path)]
			if i != 0 {
				dep := judgeDependency(m, row, read, danglingOn(s.dangling, m.Path))
				deps = append(deps, dep)
				depClaims = append(depClaims, dep.claims...)
				continue
			}
			if m.Verdict == FleetVerdictMissing || !read {
				continue
			}
			e.GitSource = GitSourceSDLC
			e.Dirty, e.Unlanded, e.Operation = memberGit(m, row, read)
			for _, c := range row.Claims {
				claimRefs = append(claimRefs, c.Ref)
			}
			e.Quality, claimsError = qualityOf(row.ClaimsState), row.ClaimsError
			hostClaims = row.Claims
			resting = m.RestingBranch
			f.git.Branch = row.Branch
			e.Branch, activeRef, f.git.Issue, f.git.IssueStatus = sdlcBranch(m, row, s.repo)
		}
	case s.candidate != nil:
		e.Quality = QualityUnknown
		if s.fleetState == FleetObservationUnsupported {
			e.Quality = QualityUnsupported
		}
		if probe, ok := in.LocalGit[s.path]; ok && probe.Err == "" && e.Dir == DirPresent {
			st := probe.Status
			e.GitSource, e.Dirty = GitSourceLocalProbe, triOf(st.Dirty)
			f.git.Branch = st.Branch
			switch {
			case st.Detached:
				e.Branch = BranchDetached
			case st.Branch == resting:
				e.Branch = BranchResting
			case issueBranchPattern.MatchString(st.Branch):
				e.Branch = BranchOpenIssue
				activeRef = s.repo + "#" + issueBranchPattern.FindStringSubmatch(st.Branch)[1]
				f.git.Issue = activeRef
			default:
				e.Branch = BranchOther
			}
		} else if ok {
			f.git.Error = probe.Err
		}
	case len(s.dangling) > 0:
		e.Quality = QualityPresent
	}
	// Dangling claims (their checkout is no inventory row) always count toward
	// their slot, present or not: on the host path a host claim, otherwise a
	// dependency claim whose member's branch is unread (pair#367 BR-5). A
	// listed member already took its own above.
	for _, c := range s.dangling {
		path := filepath.Clean(c.Claimant.Worktree)
		if path == s.path {
			claimRefs = append(claimRefs, c.Ref)
			continue
		}
		if s.slot != nil && slices.ContainsFunc(s.slot.Members, func(m FleetMember) bool { return filepath.Clean(m.Path) == path }) {
			continue
		}
		// Not a listed member: its tree is unread, never clean.
		dep := dependencyJudgment{verdict: DepUnknown, tree: DepTreeUnknown, dirty: TriUnknown, unlanded: TriUnknown, operation: TriUnknown,
			claims: []RecoverMemberClaim{{Ref: c.Ref, Checkout: filepath.Base(path), State: memberClaimUnknown}}}
		deps = append(deps, dep)
		depClaims = append(depClaims, dep.claims...)
	}
	f.git.Source, f.git.State = string(e.GitSource), string(e.Branch)
	f.git.Dirty, f.git.Unlanded, f.git.Operation = string(e.Dirty), string(e.Unlanded), string(e.Operation)

	// Claims, against the checked-out branch's issue.
	claimRefs = uniqueStrings(claimRefs)
	active := activeRef != "" && slices.Contains(claimRefs, activeRef)
	switch {
	case len(claimRefs) == 0:
		e.Claims = ClaimsNone
	case len(claimRefs) == 1 && active:
		e.Claims = ClaimsOneActive
	case len(claimRefs) == 1:
		e.Claims = ClaimsOneInactive
	case active:
		e.Claims = ClaimsManyWithActive
	default:
		e.Claims = ClaimsManyWithoutActive
	}
	f.claims = RecoverClaims{Quality: string(e.Quality), Error: claimsError, Dependency: depClaims}
	for _, ref := range claimRefs {
		if active && ref == activeRef {
			f.claims.Active = ref
		} else {
			f.claims.Inactive = append(f.claims.Inactive, ref)
		}
	}
	e.DepClaims, e.DepTree = foldDependencies(deps)
	switch {
	case e.Claims == ClaimsOneInactive:
		f.restoreRef, f.restoreCheckout = f.claims.Inactive[0], filepath.Base(s.path)
	case e.DepClaims == DepResting || e.DepClaims == DepRestingDirty:
		for _, c := range depClaims {
			if c.State == memberClaimResting {
				f.restoreRef, f.restoreCheckout = c.Ref, c.Checkout
			}
		}
	}
	for _, c := range hostClaims {
		if c.Claimant.Workspace != "" && c.Claimant.Workspace != s.address {
			e.Workspace = true
		}
	}
	if activeRef != "" && !active && s.inv != nil {
		e.Elsewhere = claimedElsewhere(s, activeRef)
	}

	// Agent.
	e.Couch, e.Agent, e.Threads, e.Offer = CouchOK, AgentNone, ThreadsZero, OfferNone
	f.agent = RecoverAgent{State: string(AgentNone), Offered: []string{}}
	if in.Couch.State != CouchObservationOK {
		e.Couch = CouchUnavailable
		f.agent.State = "unknown"
	}
	f.threadCount = len(s.threads)
	for _, row := range s.threads {
		f.agent.Rows = append(f.agent.Rows, RecoverThreadRef{RepoScope: row.Address.RepoScope, Tag: string(row.Address.Tag), State: string(row.State), Reason: string(row.Reason)})
	}
	switch len(s.threads) {
	case 0:
	case 1:
		e.Threads, e.Agent = ThreadsOne, agentOf(s.threads[0])
		offered := ActorActions(ActorRowFactsOf(s.threads[0]))
		e.Offer = offerOf(offered)
		f.agent.Offered = append(f.agent.Offered, offered...)
	default:
		e.Threads, e.Agent = ThreadsMany, agentOf(s.threads[0])
		for _, row := range s.threads {
			if a := agentOf(row); needsAttention(a) && (!needsAttention(e.Agent) || agentRank[a] > agentRank[e.Agent]) {
				e.Agent = a
			}
		}
	}
	if e.Couch == CouchOK {
		f.agent.State = string(e.Agent)
	}
	f.agent.Threads = len(s.threads)
	for _, row := range s.threads {
		if row.Orphan != nil {
			f.agent.Orphan = &RecoverOrphan{PID: row.Orphan.PID, Session: row.Orphan.Session}
			break
		}
	}

	// The union of work evidence.
	if len(claimRefs) > 0 || len(depClaims) > 0 {
		f.work = append(f.work, "claim")
	}
	if e.Branch == BranchOpenIssue {
		f.work = append(f.work, "issue-branch")
	}
	// The one place facts cross members: the row's evidence list is the
	// union of what any checkout of the slot carries (display only).
	anyMember := func(host TriState, of func(dependencyJudgment) TriState) bool {
		if host == TriYes {
			return true
		}
		return slices.ContainsFunc(deps, func(d dependencyJudgment) bool { return of(d) == TriYes })
	}
	if anyMember(e.Unlanded, func(d dependencyJudgment) TriState { return d.unlanded }) {
		f.work = append(f.work, "unlanded-commits")
	}
	if anyMember(e.Dirty, func(d dependencyJudgment) TriState { return d.dirty }) {
		f.work = append(f.work, "dirty")
	}
	if anyMember(e.Operation, func(d dependencyJudgment) TriState { return d.operation }) {
		f.work = append(f.work, "operation")
	}
	for _, row := range s.threads {
		if !(row.State == ThreadUnusable && row.Reason == ReasonNeverStarted) {
			f.work = append(f.work, "conversation")
			break
		}
	}
	return f
}

// Member-claim states: a dependency claim judged against its own member.
const (
	memberClaimActive  = "active"  // the member is on the claim's issue branch
	memberClaimResting = "resting" // the member rests on its resting branch
	memberClaimOther   = "other"   // another branch, or a detached HEAD
	memberClaimUnknown = "unknown" // the member's branch is unread or gone
)

// judgeMemberClaim judges one claim against the branch of the member it sits
// on (pair#367 BR-4): never against the host's.
func judgeMemberClaim(ref string, m FleetMember, row FleetRow) RecoverMemberClaim {
	c := RecoverMemberClaim{Ref: ref, Checkout: filepath.Base(m.Path)}
	state, issueRef, _, _ := sdlcBranch(m, row, filepath.Base(m.Path))
	switch {
	case issueRef != "" && issueRef == ref && state != BranchUnknown:
		c.State = memberClaimActive
	case state == BranchResting:
		c.State = memberClaimResting
	case state == BranchUnknown || !knownFleetVerdicts[m.Verdict]:
		c.State = memberClaimUnknown
	default:
		c.State = memberClaimOther
	}
	return c
}

// dependencyJudgment is one dependency member judged on its own facts.
type dependencyJudgment struct {
	verdict                    EvidenceDepClaims
	claims                     []RecoverMemberClaim
	dirty, unlanded, operation TriState
	// tree is this member's working tree for the slot-level reboot guard.
	tree EvidenceDepTree
}

// memberTree reads one member's tree, worst first: an operation, anything
// unread (dirt, unlanded commits, operation or branch), then dirt.
func memberTree(dirty, unlanded, operation TriState, branch EvidenceBranch) EvidenceDepTree {
	switch {
	case operation == TriYes:
		return DepTreeOperation
	case dirty == TriUnknown || unlanded == TriUnknown || operation == TriUnknown || branch == BranchUnknown:
		return DepTreeUnknown
	case dirty == TriYes:
		return DepTreeDirty
	}
	return DepTreeClean
}

// danglingOn lists the dangling claims whose checkout is path.
func danglingOn(dangling []FleetDanglingClaim, path string) []string {
	var refs []string
	for _, c := range dangling {
		if filepath.Clean(c.Claimant.Worktree) == filepath.Clean(path) {
			refs = append(refs, c.Ref)
		}
	}
	return refs
}

// judgeDependency judges one dependency member reading only its own branch,
// dirt, unlanded commits, operation and claims (BR-4, BR-14).
func judgeDependency(m FleetMember, row FleetRow, read bool, dangling []string) dependencyJudgment {
	d := dependencyJudgment{dirty: TriNo, unlanded: TriNo, operation: TriNo, tree: DepTreeClean}
	for _, ref := range dangling {
		d.claims = append(d.claims, RecoverMemberClaim{Ref: ref, Checkout: filepath.Base(m.Path), State: memberClaimUnknown})
	}
	switch {
	case m.Verdict == FleetVerdictMissing:
		// Absent: a claim on it is unread evidence; without one, the disk
		// verdict shows it and there is nothing of its own to judge. Its tree
		// is gone, so nothing in it can be lost (clean for the reboot guard;
		// a claim on it still holds reboot through DepUnknown).
		if len(d.claims) > 0 {
			d.verdict = DepUnknown
		} else {
			d.verdict = DepNone
		}
		return d
	case !read:
		d.verdict, d.dirty, d.unlanded, d.operation, d.tree = DepUnknown, TriUnknown, TriUnknown, TriUnknown, DepTreeUnknown
		return d
	}
	d.dirty, d.unlanded, d.operation = memberGit(m, row, read)
	for _, c := range row.Claims {
		d.claims = append(d.claims, judgeMemberClaim(c.Ref, m, row))
	}
	states := map[string]int{}
	for _, c := range d.claims {
		states[c.State]++
	}
	branch, _, _, _ := sdlcBranch(m, row, filepath.Base(m.Path))
	d.tree = memberTree(d.dirty, d.unlanded, d.operation, branch)
	switch {
	case states[memberClaimOther] > 0 || states[memberClaimResting] > 1:
		d.verdict = DepConflict
	case states[memberClaimUnknown] > 0:
		d.verdict = DepUnknown
	case states[memberClaimActive] > 0:
		d.verdict = DepActive
	case states[memberClaimResting] == 1 && d.dirty == TriNo:
		d.verdict = DepResting
	case states[memberClaimResting] == 1 && d.dirty == TriYes:
		d.verdict = DepRestingDirty
	case states[memberClaimResting] == 1:
		d.verdict = DepUnknown
	case d.dirty == TriYes || d.unlanded == TriYes || d.operation == TriYes || branch == BranchOther || branch == BranchDetached || branch == BranchOpenIssue:
		// Unclaimed work of its own, or a branch no claim names.
		d.verdict = DepWork
	case d.dirty == TriUnknown || d.unlanded == TriUnknown || d.operation == TriUnknown || branch == BranchUnknown:
		d.verdict = DepUnknown
	default:
		d.verdict = DepNone
	}
	return d
}

// foldDependencies reduces the per-member judgments to the slot's dependency
// dimension, plus the slot-level tree fact rule A's reboot guard reads: the
// worst dependency tree (AllEvidenceDepTrees is ordered best to worst).
func foldDependencies(deps []dependencyJudgment) (EvidenceDepClaims, EvidenceDepTree) {
	rank := map[EvidenceDepClaims]int{DepNone: 0, DepActive: 1, DepResting: 2, DepRestingDirty: 3, DepUnknown: 4, DepWork: 5, DepConflict: 6}
	trees := AllEvidenceDepTrees()
	out, tree, resting := DepNone, DepTreeClean, 0
	for _, d := range deps {
		if rank[d.verdict] > rank[out] {
			out = d.verdict
		}
		if d.verdict == DepResting || d.verdict == DepRestingDirty {
			resting++
		}
		if slices.Index(trees, d.tree) > slices.Index(trees, tree) {
			tree = d.tree
		}
	}
	if resting > 1 {
		out = DepConflict
	}
	return out, tree
}

// memberGit reads one present member's dirt, unlanded commits and operation.
// A fact sdlc could not read is unknown, never absent.
func memberGit(m FleetMember, row FleetRow, read bool) (TriState, TriState, TriState) {
	if !read || !knownFleetVerdicts[m.Verdict] || !row.Facts.Available {
		return TriUnknown, TriUnknown, TriUnknown
	}
	for _, r := range m.Reasons {
		if !knownFleetReason(r) {
			return TriUnknown, TriUnknown, TriUnknown
		}
	}
	dirty := TriUnknown
	if row.Facts.DirtyCount != nil {
		dirty = triOf(*row.Facts.DirtyCount > 0)
	}
	unlanded := TriUnknown
	if row.Facts.BaseAvailable && row.Facts.Ahead != nil {
		unlanded = triOf(*row.Facts.Ahead > 0)
	}
	operation := TriNo
	for _, r := range m.Reasons {
		switch {
		case strings.HasPrefix(r, "operation:"):
			return dirty, unlanded, TriYes
		case r == "probe:operation" || r == "probe:facts" || r == "probe:checkout":
			operation = TriUnknown
		}
	}
	return dirty, unlanded, operation
}

// sdlcBranch reads the host's branch state and the issue it names.
func sdlcBranch(m FleetMember, row FleetRow, repo string) (EvidenceBranch, string, string, string) {
	switch {
	case row.Detached:
		return BranchDetached, "", "", ""
	case row.Branch == "":
		return BranchUnknown, "", "", ""
	case row.Branch == m.RestingBranch:
		return BranchResting, "", "", ""
	}
	match := issueBranchPattern.FindStringSubmatch(row.Branch)
	if match == nil {
		return BranchOther, "", "", ""
	}
	if slices.Contains(m.Reasons, "probe:issue") {
		return BranchUnknown, repo + "#" + match[1], repo + "#" + match[1], ""
	}
	for _, issue := range row.Issues {
		if issue.Provenance != "branch-prefix" {
			continue
		}
		if slices.Contains(issueTerminalStatuses, issue.DeclaredStatus) {
			return BranchTerminalIssue, issue.Ref, issue.Ref, issue.DeclaredStatus
		}
		return BranchOpenIssue, issue.Ref, issue.Ref, issue.DeclaredStatus
	}
	// The tracker has no such issue: an ordinary branch.
	return BranchOther, "", "", ""
}

func claimedElsewhere(s *recoverSlot, ref string) bool {
	mine := map[string]bool{}
	if s.slot != nil {
		for _, m := range s.slot.Members {
			mine[filepath.Clean(m.Path)] = true
		}
	}
	for _, row := range s.inv.Rows {
		if mine[filepath.Clean(row.TreePath)] {
			continue
		}
		for _, c := range row.Claims {
			if c.Ref == ref {
				return true
			}
		}
	}
	for _, c := range s.inv.DanglingClaims {
		if c.Ref == ref && !mine[filepath.Clean(c.Claimant.Worktree)] {
			return true
		}
	}
	return false
}

func agentOf(row ActionableThreadSummary) EvidenceAgent { return agentEvidence(row.State, row.Reason) }

// agentRank is the ONE ordering of agent evidence (#399 M1 review): the slot
// report keeps its most active row by it, and a many-thread slot keeps its most
// urgent attention state by it. An orphan is a running agent nothing can reach:
// above every row that isn't running, below the reachable ones.
var agentRank = map[EvidenceAgent]int{
	AgentBusy: 7, AgentLive: 6, AgentDetached: 5, AgentOrphaned: 4,
	AgentUnusableUnknown: 3, AgentUnusable: 2, AgentParked: 1, AgentNone: 0,
}

// needsAttention is agent evidence that holds a many-thread slot's row.
func needsAttention(a EvidenceAgent) bool {
	return a == AgentBusy || a == AgentOrphaned || a == AgentUnusableUnknown
}

// agentEvidence is the one reading of a classified thread row as agent
// evidence; the recovery report and the slot reconciler share it.
func agentEvidence(state ActionableThreadState, reason ThreadReason) EvidenceAgent {
	switch state {
	case ThreadLive:
		return AgentLive
	case ThreadDetached:
		return AgentDetached
	case ThreadParked:
		return AgentParked
	case ThreadBusy:
		return AgentBusy
	case ThreadUnusable:
		switch reason {
		case ReasonUnknown:
			return AgentUnusableUnknown
		case ReasonOrphanedServer:
			return AgentOrphaned
		}
	}
	return AgentUnusable
}

func qualityOf(state string) EvidenceQuality {
	switch state {
	case FleetClaimsPresent:
		return QualityPresent
	case FleetClaimsStale:
		return QualityStale
	case FleetClaimsPartial:
		return QualityPartial
	case FleetClaimsAbsent:
		return QualityAbsent
	case FleetClaimsUnknown:
		return QualityUnknown
	}
	// A state this build does not know is no read at all.
	return QualityUnknown
}

func triOf(b bool) TriState {
	if b {
		return TriYes
	}
	return TriNo
}

func uniqueStrings(values []string) []string {
	out := []string{}
	for _, v := range values {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

// recoverRowOf builds one row: evidence, then the rule table, then text.
func recoverRowOf(s *recoverSlot, in RecoverPlanInput) RecoverRow {
	f := slotEvidenceOf(s, in)
	d := classifyRecover(f.evidence)
	row := RecoverRow{Address: s.address, Path: s.path, Fleet: s.fleet, Class: d.Class, Evidence: f.work, Git: f.git, Disk: f.disk, Agent: f.agent, Claims: f.claims,
		Next: RecoverNext{Steps: []RecoverStep{}, Hold: []string{}, Notes: []string{}}}
	if row.Evidence == nil {
		row.Evidence = []string{}
	}
	for _, h := range d.Hold {
		if h == string(HoldThreads) {
			h += ":" + strconv.Itoa(f.threadCount)
		}
		row.Next.Hold = append(row.Next.Hold, h)
	}
	for _, n := range d.Notes {
		row.Next.Notes = append(row.Next.Notes, string(n))
	}
	for _, action := range d.Steps {
		step := RecoverStep{Action: action, Command: SlotOperationCommand(action, s.address)}
		if action == "ask-agent-restore" {
			step.Message = RestoreWorkspaceMessage(f.restoreRef, s.address, f.restoreCheckout)
			step.Command = SendToCommand(s.address, step.Message)
		}
		row.Next.Steps = append(row.Next.Steps, step)
	}
	row.Automatic = len(row.Next.Steps) > 0 && len(row.Next.Hold) == 0
	row.Reason = recoverReason(row, f, in)
	return row
}

// recoverWorkspacePlan is the slot reconciler's plan for a row, as text.
func recoverWorkspacePlan(row RecoverRow, in RecoverPlanInput) string {
	r, ok := in.SlotPlans[filepath.Clean(row.Path)]
	if !ok {
		return "no plan observed"
	}
	return SlotPlanSummary(r.Plan, r.PlanError)
}

// recoverWorkspaceAdvice is the reconciler's own advice for a row whose
// workspace cannot converge (the text callers print).
func recoverWorkspaceAdvice(row RecoverRow, in RecoverPlanInput) string {
	r, ok := in.SlotPlans[filepath.Clean(row.Path)]
	if !ok {
		return "the slot's workspace cannot converge"
	}
	repo, _, _ := strings.Cut(row.Address, ":")
	if blocking, _ := SlotOutcome(row.Address, repo, ReconcileResult{Observation: r.Observation, Plan: r.Plan}, nil); blocking != nil {
		return blocking.Error()
	}
	return "the slot's workspace cannot converge: " + SlotPlanSummary(r.Plan, r.PlanError)
}

// recoverReason is the one author of every row's reason text.
func recoverReason(row RecoverRow, f slotFacts, in RecoverPlanInput) string {
	asks := slices.ContainsFunc(row.Next.Steps, func(s RecoverStep) bool { return s.Action == "ask-agent-restore" })
	var text string
	switch row.Class {
	case RecoverDirectoryMissing:
		text = "the slot directory is gone and nothing is left to re-create it from"
	case RecoverReconcilable:
		text = "the slot's workspace needs repair (" + recoverWorkspacePlan(row, in) + "); couch --reconcile " + row.Address + " runs it, as open, resume and reboot do first"
	case RecoverSlotNeedsZero:
		text = recoverWorkspaceAdvice(row, in)
	case RecoverAgentUnknown:
		if slices.Contains(row.Next.Hold, string(HoldCouchUnavailable)) {
			text = "Couch's store could not be read: " + in.Couch.Error
		} else {
			text = "the agent's state could not be checked; read the report again"
		}
	case RecoverOrphanedServer:
		text = "the agent's zellij server is running but lost its socket; its conversation may still be writing, so reap it (confirmed) before resuming, and never reboot"
		if o := row.Agent.Orphan; o != nil {
			text = launcher.OrphanDiagnostic(o.Session, o.PID) + "; its conversation may still be writing, so reap it (confirmed) before resuming, and never reboot"
		}
	case RecoverStartUnreconciled:
		text = "a start was claimed and not reconciled; read the report again later, never reboot (it could duplicate a live agent)"
	case RecoverAmbiguousThreads:
		text = fmt.Sprintf("%d Couch threads stand for this slot", row.Agent.Threads)
	case RecoverConflict:
		text = "the evidence conflicts (" + strings.Join(row.Next.Hold, ", ") + "); inspect before acting"
	case RecoverAmbiguousClaims:
		text = "several claims and none is checked out (" + strings.Join(row.Claims.Inactive, ", ") + ")"
	case RecoverLanded:
		text = "issue done; the slot can return to its resting branch"
	case RecoverEvidenceUnavailable:
		text = "no claim, and the slot's git state could not be fully read"
	case RecoverIdle:
		text = "clean resting branch with no claim; nothing to recover"
	case RecoverUnidentifiedWork:
		text = "the host carries work no claim or issue branch names (branch " + orUnknown(row.Git.Branch) + ")"
		if !slices.Contains(f.work, "claim") {
			text = "work with no claim or issue branch (branch " + orUnknown(row.Git.Branch) + ")"
		}
	case RecoverNoCouchThread:
		text = "work evidence but no Couch thread stands for this slot"
	case RecoverRestoreWorkspace:
		text = "claimed " + f.restoreRef + " on the " + f.restoreCheckout + " checkout's resting branch"
		switch {
		case asks:
			text += "; ask the slot's agent to restore it"
		case slices.Contains(row.Next.Notes, string(NoteRestingBranchDirty)):
			text += "; no restore request: switching branches with uncommitted files can fail or carry the edits across"
		case slices.Contains(row.Next.Notes, string(NoteGitUnknown)):
			text += "; no restore request: the working tree could not be read"
		}
	case RecoverAgrees:
		text = "claim and checkout agree"
	case RecoverClaimLikelyLost:
		text = "issue branch " + row.Git.Issue + " with no visible claim; first run `sdlc issue show " + issueNumber(row.Git.Issue) +
			" --json`; if a repair is needed, the slot's own agent runs `sdlc claim --issue N --adopt` or `sdlc reclaim` under operator direction"
	case RecoverPartialEvidence:
		text = "some evidence is missing; resume only"
	case RecoverNoSafeStep:
		text = "no step is safe (" + strings.Join(row.Next.Hold, ", ") + ")"
	default:
		text = "no rule matched"
	}
	if slices.Contains(row.Next.Notes, string(NoteClaimsStale)) {
		text += "; claims stale: " + row.Claims.Error
	}
	// A workspace note carries the reconciler's own reading.
	for _, note := range []RecoverNote{NoteWorkspaceHeld, NoteWorkspaceDegraded, NoteWorkspaceUnknown} {
		if slices.Contains(row.Next.Notes, string(note)) {
			text += "; workspace (" + string(note) + "): " + recoverWorkspacePlan(row, in)
		}
	}
	return text
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

func issueNumber(ref string) string {
	_, n, ok := strings.Cut(ref, "#")
	if !ok {
		return ref
	}
	return strings.TrimLeft(n, "0")
}
