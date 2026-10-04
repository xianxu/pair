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
}

type RecoverThreadRef struct {
	RepoScope string `json:"repo_scope"`
	Tag       string `json:"tag"`
	State     string `json:"state"`
	Reason    string `json:"reason,omitempty"`
}

type RecoverClaims struct {
	Quality  string   `json:"quality"`
	Error    string   `json:"error,omitempty"`
	Active   string   `json:"active,omitempty"`
	Inactive []string `json:"inactive,omitempty"`
}

// RecoverNext is the suggestion. Hold and Notes are closed vocabularies
// (AllRecoverHolds, AllRecoverNotes); a parameterized code is kind:suffix.
type RecoverNext struct {
	Steps []RecoverStep `json:"steps"`
	Hold  []string      `json:"hold"`
	Notes []string      `json:"notes"`
}

// RecoverStep is one action: resume or reboot (from ActorActions), or
// ask-agent-restore with the message to send the slot's agent. Command stays
// empty until the socket primitives exist (M2).
type RecoverStep struct {
	Action  string `json:"action"`
	Command string `json:"command,omitempty"`
	Message string `json:"message,omitempty"`
}

// RestoreWorkspaceMessage is the one text a slot's own agent receives when its
// claim sits on the slot's resting branch.
func RestoreWorkspaceMessage(ref, address string) string {
	return "Recovery (" + address + "): restore the workspace of this slot for " + ref +
		" through sdlc (check out its issue branch); never discard files. Reply with what sdlc issue show reports."
}

// RecoverClass is a row's class; AllRecoverClasses follows the rule order.
type RecoverClass string

const (
	RecoverDirectoryMissing    RecoverClass = "directory-missing"
	RecoverAgentUnknown        RecoverClass = "agent-unknown"
	RecoverStartUnreconciled   RecoverClass = "start-unreconciled"
	RecoverAmbiguousThreads    RecoverClass = "ambiguous-threads"
	RecoverConflict            RecoverClass = "conflict"
	RecoverAmbiguousClaims     RecoverClass = "ambiguous-claims"
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
)

func AllRecoverClasses() []RecoverClass {
	return []RecoverClass{RecoverDirectoryMissing, RecoverAgentUnknown, RecoverStartUnreconciled, RecoverAmbiguousThreads,
		RecoverConflict, RecoverAmbiguousClaims, RecoverEvidenceUnavailable, RecoverIdle, RecoverUnidentifiedWork,
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
)

func AllRecoverHolds() []RecoverHold {
	return []RecoverHold{HoldDirectoryMissing, HoldCouchUnavailable, HoldStartUnreconciled, HoldAgentUnknown, HoldThreads,
		HoldConflict, HoldAmbiguousClaims, HoldGitUnknown, HoldUnidentifiedWork, HoldNoCouchThread,
		HoldRebootUnsafeOperation, HoldRebootUnsafeGit, HoldNoActorAction, HoldResumeOnly, HoldNoRule}
}

// Conflict facts, in the order a conflict hold lists them.
var recoverConflictFacts = []string{"claim-elsewhere", "issue-terminal", "claim-workspace", "claim-off-branch", "claim-branch-mismatch"}

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
)

func AllRecoverNotes() []RecoverNote {
	return []RecoverNote{NoteInactiveClaims, NoteDetachedHead, NoteRestingBranchDirty, NoteInspectUncommittedFirst,
		NoteClaimRepair, NoteClaimsStale, NoteClaimsPartial, NoteClaimsUnknown, NoteClaimsAbsent, NoteClaimsUnsupported,
		NoteGitLocalProbe, NoteGitUnknown}
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

	ThreadsZero EvidenceThreads = "0"
	ThreadsOne  EvidenceThreads = "1"
	ThreadsMany EvidenceThreads = "many"

	// Offer is what ActorActions offers the single joined row.
	OfferNone         EvidenceOffer = "none"
	OfferReboot       EvidenceOffer = "reboot"
	OfferResumeReboot EvidenceOffer = "resume-reboot"

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

	TriYes     TriState = "yes"
	TriNo      TriState = "no"
	TriUnknown TriState = "unknown"
)

func AllEvidenceDirs() []EvidenceDir    { return []EvidenceDir{DirPresent, DirMissing} }
func AllEvidenceCouch() []EvidenceCouch { return []EvidenceCouch{CouchOK, CouchUnavailable} }
func AllEvidenceAgents() []EvidenceAgent {
	return []EvidenceAgent{AgentNone, AgentLive, AgentDetached, AgentParked, AgentBusy, AgentUnusable, AgentUnusableUnknown}
}
func AllEvidenceThreads() []EvidenceThreads {
	return []EvidenceThreads{ThreadsZero, ThreadsOne, ThreadsMany}
}
func AllEvidenceOffers() []EvidenceOffer {
	return []EvidenceOffer{OfferNone, OfferReboot, OfferResumeReboot}
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

func (o EvidenceOffer) actions() []string {
	switch o {
	case OfferReboot:
		return []string{"reboot"}
	case OfferResumeReboot:
		return []string{"resume", "reboot"}
	}
	return nil
}

func offerOf(actions []string) EvidenceOffer {
	switch {
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
	Dir       EvidenceDir
	Couch     EvidenceCouch
	Agent     EvidenceAgent
	Threads   EvidenceThreads
	Offer     EvidenceOffer
	Branch    EvidenceBranch
	Claims    EvidenceClaims
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
}

func (e SlotEvidence) gitUnknown() bool {
	return e.Branch == BranchUnknown || e.Dirty == TriUnknown || e.Unlanded == TriUnknown || e.Operation == TriUnknown
}

// consistent excludes evidence that cannot be observed together. It only
// prunes the totality test's domain; production evidence is derived, not
// checked against it.
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
	read := e.Quality == QualityPresent || e.Quality == QualityStale
	var d recoverDecision
	switch {
	case e.Dir == DirMissing:
		d = recoverDecision{Class: RecoverDirectoryMissing, Hold: []string{string(HoldDirectoryMissing)}}
	case e.Couch == CouchUnavailable:
		d = recoverDecision{Class: RecoverAgentUnknown, Hold: []string{string(HoldCouchUnavailable)}}
	case e.Agent == AgentBusy:
		d = recoverDecision{Class: RecoverStartUnreconciled, Hold: []string{string(HoldStartUnreconciled)}}
	case e.Agent == AgentUnusableUnknown:
		d = recoverDecision{Class: RecoverAgentUnknown, Hold: []string{string(HoldAgentUnknown)}}
	case e.Threads == ThreadsMany:
		d = recoverDecision{Class: RecoverAmbiguousThreads, Hold: []string{string(HoldThreads)}}
	case len(conflictFacts(e, claims, read)) > 0:
		d = recoverDecision{Class: RecoverConflict}
		for _, fact := range conflictFacts(e, claims, read) {
			d.Hold = append(d.Hold, string(HoldConflict)+":"+fact)
		}
	case e.Claims == ClaimsManyWithoutActive:
		d = recoverDecision{Class: RecoverAmbiguousClaims, Hold: []string{string(HoldAmbiguousClaims)}}
	case !claims && e.Branch != BranchOpenIssue && e.gitUnknown():
		d = recoverDecision{Class: RecoverEvidenceUnavailable, Hold: []string{string(HoldGitUnknown)}}
	case !claims && e.Branch == BranchResting && e.Dirty == TriNo && e.Unlanded == TriNo && e.Operation == TriNo:
		d = recoverDecision{Class: RecoverIdle}
	case !claims && e.Branch != BranchOpenIssue:
		// resting with work, another branch or a detached HEAD
		d = recoverDecision{Class: RecoverUnidentifiedWork, Hold: []string{string(HoldUnidentifiedWork)}}
	case e.Agent == AgentNone:
		d = recoverDecision{Class: RecoverNoCouchThread, Hold: []string{string(HoldNoCouchThread)}}
	case e.Claims == ClaimsOneInactive && e.Branch == BranchResting:
		d = restoreWorkspaceDecision(e)
	case e.Claims == ClaimsOneActive || e.Claims == ClaimsManyWithActive || e.Claims == ClaimsOneInactive && e.Branch == BranchDetached:
		var notes []RecoverNote
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
		d = withRuleA(RecoverClaimLikelyLost, e, false, []RecoverNote{NoteClaimRepair})
	case e.Branch == BranchOpenIssue || claims && e.Branch == BranchUnknown:
		var notes []RecoverNote
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
	default:
		d = recoverDecision{Class: RecoverNoRule, Hold: []string{string(HoldNoRule)}}
	}
	if e.Quality == QualityStale {
		d.Notes = append(d.Notes, NoteClaimsStale)
	}
	d.Notes = sortedNotes(d.Notes)
	return d
}

func conflictFacts(e SlotEvidence, claims, read bool) []string {
	applies := map[string]bool{
		"claim-elsewhere":       e.Elsewhere,
		"issue-terminal":        e.Branch == BranchTerminalIssue,
		"claim-workspace":       e.Workspace,
		"claim-off-branch":      claims && e.Branch == BranchOther,
		"claim-branch-mismatch": claims && e.Branch == BranchOpenIssue && (e.Claims == ClaimsOneInactive || e.Claims == ClaimsManyWithoutActive) && read,
	}
	var facts []string
	for _, fact := range recoverConflictFacts {
		if applies[fact] {
			facts = append(facts, fact)
		}
	}
	return facts
}

// restoreWorkspaceDecision: one claim on the resting branch. The slot's agent
// is asked to restore its issue branch, unless uncommitted (or unread) files
// could make the switch fail or carry edits across: then resume only.
func restoreWorkspaceDecision(e SlotEvidence) recoverDecision {
	if e.Dirty != TriNo {
		notes := []RecoverNote{NoteRestingBranchDirty}
		if e.Dirty == TriUnknown {
			notes = []RecoverNote{NoteGitUnknown}
		}
		return withRuleA(RecoverRestoreWorkspace, e, true, notes)
	}
	d := withRuleA(RecoverRestoreWorkspace, e, false, nil)
	if d.Class == RecoverRestoreWorkspace {
		d.Steps = append(d.Steps, "ask-agent-restore")
	}
	return d
}

// withRuleA attaches rule A's actor step to a class, or turns the row into
// no-safe-step when no step is safe. resumeOnly forbids reboot.
func withRuleA(class RecoverClass, e SlotEvidence, resumeOnly bool, notes []RecoverNote) recoverDecision {
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
	case e.Operation == TriYes:
		hold = HoldRebootUnsafeOperation
	case e.Branch == BranchDetached || e.gitUnknown():
		hold = HoldRebootUnsafeGit
	default:
		d.Steps = []string{"reboot"}
		if e.Dirty == TriYes {
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
// (construct/vocabulary/issue.cue).
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
	if slot, ok := conventionalSlotFromPath(path); ok {
		return slot, true
	}
	env := filepath.Dir(path)
	repo, _, ok := strings.Cut(filepath.Base(env), "-slot")
	if !ok || repo == "" {
		return SlotIdentity{}, false
	}
	return conventionalSlotFromPath(filepath.Join(env, repo))
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
}

// slotEvidenceOf reads one slot's sources into closed evidence (pure).
func slotEvidenceOf(s *recoverSlot, in RecoverPlanInput) slotFacts {
	var f slotFacts
	e := &f.evidence
	e.Dir = DirPresent
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
	resting := RestingBranch(s.number)
	activeRef := ""
	var claimRefs []string
	var hostClaims []FleetClaim
	e.Quality = QualityUnknown
	claimsError := ""
	switch {
	case s.slot != nil:
		f.disk.Verdict = s.slot.Verdict
		if !knownFleetVerdicts[s.slot.Verdict] {
			f.disk.Verdict = FleetVerdictUnknown
		}
		dirty, unlanded, operation := []TriState{}, []TriState{}, []TriState{}
		for i, m := range s.slot.Members {
			f.disk.Members = append(f.disk.Members, RecoverDiskMember{Role: m.Role, Path: m.Path, Verdict: m.Verdict, Reasons: append([]string{}, m.Reasons...)})
			if m.Verdict == FleetVerdictMissing {
				continue
			}
			row, read := s.rows[filepath.Clean(m.Path)]
			md, mu, mo := memberGit(m, row, read)
			dirty, unlanded, operation = append(dirty, md), append(unlanded, mu), append(operation, mo)
			if read {
				for _, c := range row.Claims {
					claimRefs = append(claimRefs, c.Ref)
				}
				q := qualityOf(row.ClaimsState)
				if i == 0 || qualityRank(q) > qualityRank(e.Quality) {
					e.Quality, claimsError = q, row.ClaimsError
				}
			} else if i == 0 {
				e.Quality = QualityUnknown
			}
			if i != 0 {
				continue
			}
			if read {
				hostClaims = row.Claims
				resting = m.RestingBranch
				f.git.Branch = row.Branch
				e.Branch, activeRef, f.git.Issue, f.git.IssueStatus = sdlcBranch(m, row, s.repo)
			}
		}
		if len(s.slot.Members) > 0 && s.slot.Members[0].Verdict != FleetVerdictMissing {
			e.GitSource = GitSourceSDLC
			e.Dirty, e.Unlanded, e.Operation = foldTri(dirty), foldTri(unlanded), foldTri(operation)
		}
		if _, read := s.rows[filepath.Clean(s.path)]; !read {
			e.GitSource, e.Branch, e.Dirty, e.Unlanded, e.Operation = GitSourceUnknown, BranchUnknown, TriUnknown, TriUnknown, TriUnknown
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
		for _, c := range s.dangling {
			claimRefs = append(claimRefs, c.Ref)
		}
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
	f.claims = RecoverClaims{Quality: string(e.Quality), Error: claimsError}
	for _, ref := range claimRefs {
		if active && ref == activeRef {
			f.claims.Active = ref
		} else {
			f.claims.Inactive = append(f.claims.Inactive, ref)
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
			if a := agentOf(row); a == AgentBusy || a == AgentUnusableUnknown && e.Agent != AgentBusy {
				e.Agent = a
			}
		}
	}
	if e.Couch == CouchOK {
		f.agent.State = string(e.Agent)
	}
	f.agent.Threads = len(s.threads)

	// The union of work evidence.
	if len(claimRefs) > 0 {
		f.work = append(f.work, "claim")
	}
	if e.Branch == BranchOpenIssue {
		f.work = append(f.work, "issue-branch")
	}
	if e.Unlanded == TriYes {
		f.work = append(f.work, "unlanded-commits")
	}
	if e.Dirty == TriYes {
		f.work = append(f.work, "dirty")
	}
	if e.Operation == TriYes {
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

func agentOf(row ActionableThreadSummary) EvidenceAgent {
	switch row.State {
	case ThreadLive:
		return AgentLive
	case ThreadDetached:
		return AgentDetached
	case ThreadParked:
		return AgentParked
	case ThreadBusy:
		return AgentBusy
	case ThreadUnusable:
		if row.Reason == ReasonUnknown {
			return AgentUnusableUnknown
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
	}
	return QualityUnknown
}

// qualityRank orders claim-read weakness; a dependency degrades the host's
// read only when its own is weaker. absent (no tracker) is a complete read.
func qualityRank(q EvidenceQuality) int {
	return map[EvidenceQuality]int{QualityPresent: 0, QualityAbsent: 0, QualityStale: 1, QualityPartial: 2, QualityUnknown: 3, QualityUnsupported: 4}[q]
}

func triOf(b bool) TriState {
	if b {
		return TriYes
	}
	return TriNo
}

func foldTri(values []TriState) TriState {
	out := TriNo
	for _, v := range values {
		if v == TriYes {
			return TriYes
		}
		if v == TriUnknown {
			out = TriUnknown
		}
	}
	return out
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
		step := RecoverStep{Action: action}
		if action == "ask-agent-restore" {
			step.Message = RestoreWorkspaceMessage(f.claims.Inactive[0], s.address)
		}
		row.Next.Steps = append(row.Next.Steps, step)
	}
	row.Automatic = len(row.Next.Steps) > 0 && len(row.Next.Hold) == 0
	row.Reason = recoverReason(row, in)
	return row
}

// recoverReason is the one author of every row's reason text.
func recoverReason(row RecoverRow, in RecoverPlanInput) string {
	var text string
	switch row.Class {
	case RecoverDirectoryMissing:
		text = "the slot directory is gone; repairing it is pair#387"
	case RecoverAgentUnknown:
		if slices.Contains(row.Next.Hold, string(HoldCouchUnavailable)) {
			text = "Couch's store could not be read: " + in.Couch.Error
		} else {
			text = "the agent's state could not be checked; read the report again"
		}
	case RecoverStartUnreconciled:
		text = "a start was claimed and not reconciled; read the report again later, never reboot (it could duplicate a live agent)"
	case RecoverAmbiguousThreads:
		text = fmt.Sprintf("%d Couch threads stand for this slot", row.Agent.Threads)
	case RecoverConflict:
		text = "the evidence conflicts (" + strings.Join(row.Next.Hold, ", ") + "); inspect before acting"
	case RecoverAmbiguousClaims:
		text = "several claims and none is checked out (" + strings.Join(row.Claims.Inactive, ", ") + ")"
	case RecoverEvidenceUnavailable:
		text = "no claim, and the slot's git state could not be fully read"
	case RecoverIdle:
		text = "clean resting branch with no claim; nothing to recover"
	case RecoverUnidentifiedWork:
		text = "work with no claim or issue branch (branch " + orUnknown(row.Git.Branch) + ")"
	case RecoverNoCouchThread:
		text = "work evidence but no Couch thread stands for this slot"
	case RecoverRestoreWorkspace:
		text = "claimed " + strings.Join(row.Claims.Inactive, ", ") + " on the resting branch"
		if slices.Contains(row.Next.Notes, string(NoteRestingBranchDirty)) {
			text += "; no restore request: switching branches with uncommitted files can fail or carry the edits across"
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
