package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// FakeFleetFailure is how FakeFleetSDLC's next runs fail.
type FakeFleetFailure uint8

const (
	FakeFleetOK      FakeFleetFailure = iota
	FakeFleetExit                     // sdlc exits non-zero
	FakeFleetHang                     // sdlc never answers; the caller's deadline ends it
	FakeFleetGarbage                  // sdlc prints bytes that are not an inventory
)

// FakeFleetSDLC is a stateful model of `sdlc fleet inventory --json` behind
// ProvisionIO (ARCH-MOCK): fleets of slots whose members carry branch, dirt,
// operation, unlanded commits, issues and claims, plus dangling claims and
// off-slot rows. Verdicts follow sdlc's JudgeCheckout precedence and Worst
// fold (ariadne cmd/sdlc/internal/fleet/slots.go); the golden fixture and
// TestFleetInventoryLiveConformance keep its vocabulary honest.
type FakeFleetSDLC struct {
	mu      sync.Mutex
	fleets  map[string]*FakeFleet
	Machine FleetMachine
	// Schema is the schema_version the fake prints.
	Schema int
	Fail   FakeFleetFailure
	// Route answers a vantage from a named fleet root, overriding the
	// root-contains-vantage rule (a test forcing two vantages onto one
	// document).
	Route map[string]string
	Calls []ProvisionCommand
}

// FakeFleet is one fleet root's state.
type FakeFleet struct {
	owner    *FakeFleetSDLC
	root     string
	slots    map[string]*fakeFleetSlot
	offSlot  map[string]*fakeFleetMember
	dangling []FleetDanglingClaim
	schema   int // 0 prints the owner's Schema
}

type fakeFleetSlot struct {
	address string
	repo    string
	number  int
	members []*fakeFleetMember // host first
}

type fakeFleetMember struct {
	role, path, resting, repoRoot string
	branch                        string
	detached, missing             bool
	dirty                         int
	operation                     string
	ahead                         int
	baseUnavailable               bool
	issueStatus                   string // declared status of the branch-prefix issue
	claims                        []FleetClaim
	claimsState, claimsError      string
}

func NewFakeFleetSDLC() *FakeFleetSDLC {
	return &FakeFleetSDLC{
		fleets:  map[string]*FakeFleet{},
		Machine: FleetMachine{State: "present", Fingerprint: "fake-machine", Name: "Fake Machine"},
		Schema:  FleetSchemaVersion,
	}
}

// Fleet returns the fleet rooted at root, creating it empty on first use.
func (f *FakeFleetSDLC) Fleet(root string) *FakeFleet {
	f.mu.Lock()
	defer f.mu.Unlock()
	root = filepath.Clean(root)
	if fleet := f.fleets[root]; fleet != nil {
		return fleet
	}
	fleet := &FakeFleet{owner: f, root: root, slots: map[string]*fakeFleetSlot{}, offSlot: map[string]*fakeFleetMember{}}
	f.fleets[root] = fleet
	return fleet
}

// SetSchema makes this fleet alone print another schema_version.
func (fl *FakeFleet) SetSchema(version int) {
	defer fl.lock()()
	fl.schema = version
}

// SlotPath is the host checkout of a slot address in this fleet: root/repo for
// :0 and root/worktree/repo-slotN/repo for :N.
func (fl *FakeFleet) SlotPath(address string) string {
	repo, n := fakeSplitAddress(address)
	if n == 0 {
		return filepath.Join(fl.root, repo)
	}
	return filepath.Join(fl.root, "worktree", repo+"-slot"+strconv.Itoa(n), repo)
}

func fakeSplitAddress(address string) (string, int) {
	repo, number, ok := strings.Cut(address, ":")
	n, err := strconv.Atoi(number)
	if !ok || err != nil || n < 0 || repo == "" {
		panic(fmt.Sprintf("fake fleet: invalid slot address %q", address))
	}
	return repo, n
}

func (fl *FakeFleet) lock() func() {
	fl.owner.mu.Lock()
	return fl.owner.mu.Unlock
}

func (fl *FakeFleet) slot(address string) *fakeFleetSlot {
	s := fl.slots[address]
	if s == nil {
		panic(fmt.Sprintf("fake fleet: no slot %s", address))
	}
	return s
}

// AddSlot adds a clean slot resting on its resting branch, claims read.
func (fl *FakeFleet) AddSlot(address string) string {
	defer fl.lock()()
	repo, n := fakeSplitAddress(address)
	path := fl.SlotPath(address)
	host := &fakeFleetMember{role: "host", path: path, resting: RestingBranch(n), repoRoot: filepath.Join(fl.root, repo), branch: RestingBranch(n), claimsState: FleetClaimsPresent}
	fl.slots[address] = &fakeFleetSlot{address: address, repo: repo, number: n, members: []*fakeFleetMember{host}}
	return path
}

// AddDependency adds a present dependency checkout to a numbered slot.
func (fl *FakeFleet) AddDependency(address, name string) string {
	defer fl.lock()()
	s := fl.slot(address)
	path := filepath.Join(filepath.Dir(fl.SlotPath(address)), name)
	s.members = append(s.members, &fakeFleetMember{role: "dependency", path: path, resting: RestingBranch(0), repoRoot: path, branch: RestingBranch(0), claimsState: FleetClaimsPresent})
	return path
}

// MissingMember declares a dependency the slot lacks on disk.
func (fl *FakeFleet) MissingMember(address, name string) {
	defer fl.lock()()
	s := fl.slot(address)
	path := filepath.Join(filepath.Dir(fl.SlotPath(address)), name)
	s.members = append(s.members, &fakeFleetMember{role: "dependency", path: path, resting: RestingBranch(0), missing: true})
}

// AddOffSlotRow adds a worktree that is no slot (a scratch worktree).
func (fl *FakeFleet) AddOffSlotRow(path, repoRoot, branch string) {
	defer fl.lock()()
	fl.offSlot[path] = &fakeFleetMember{role: "worktree", path: path, repoRoot: repoRoot, branch: branch, claimsState: FleetClaimsPresent}
}

func (fl *FakeFleet) host(address string) *fakeFleetMember { return fl.slot(address).members[0] }

func (fl *FakeFleet) member(path string) *fakeFleetMember {
	for _, s := range fl.slots {
		for _, m := range s.members {
			if m.path == path && !m.missing {
				return m
			}
		}
	}
	if m := fl.offSlot[path]; m != nil {
		return m
	}
	panic(fmt.Sprintf("fake fleet: no checkout %s", path))
}

func (fl *FakeFleet) SetDirty(address string, n int) {
	defer fl.lock()()
	fl.host(address).dirty = n
}

func (fl *FakeFleet) SetOperation(address, operation string) {
	defer fl.lock()()
	fl.host(address).operation = operation
}

func (fl *FakeFleet) SetAhead(address string, n int) {
	defer fl.lock()()
	fl.host(address).ahead = n
}

// SetBaseUnavailable makes the host's base comparison fail (probe:base).
func (fl *FakeFleet) SetBaseUnavailable(address string) {
	defer fl.lock()()
	fl.host(address).baseUnavailable = true
}

var fakeIssueBranch = regexp.MustCompile(`^([0-9]{6})-`)

// SetBranch checks out branch on the host. An issue-prefixed branch names an
// open ("working") issue until SetIssueStatus says otherwise.
func (fl *FakeFleet) SetBranch(address, branch string) {
	defer fl.lock()()
	m := fl.host(address)
	m.branch, m.detached, m.issueStatus = branch, false, ""
	if fakeIssueBranch.MatchString(branch) {
		m.issueStatus = "working"
	}
}

// SetMemberBranch checks out branch on any present checkout of the fleet
// (a dependency member), with the same issue rule as SetBranch.
func (fl *FakeFleet) SetMemberBranch(path, branch string) {
	defer fl.lock()()
	m := fl.member(path)
	m.branch, m.detached, m.issueStatus = branch, false, ""
	if fakeIssueBranch.MatchString(branch) {
		m.issueStatus = "working"
	}
}

// SetMemberDirty, SetMemberAhead and SetMemberOperation set one checkout's
// own facts (a dependency member's), as SetDirty and friends do the host's.
func (fl *FakeFleet) SetMemberDirty(path string, n int) {
	defer fl.lock()()
	fl.member(path).dirty = n
}

func (fl *FakeFleet) SetMemberAhead(path string, n int) {
	defer fl.lock()()
	fl.member(path).ahead = n
}

// SetMemberBaseUnavailable makes one checkout's base comparison fail
// (probe:base): its unlanded commits are unread.
func (fl *FakeFleet) SetMemberBaseUnavailable(path string) {
	defer fl.lock()()
	fl.member(path).baseUnavailable = true
}

func (fl *FakeFleet) SetMemberOperation(path, operation string) {
	defer fl.lock()()
	fl.member(path).operation = operation
}

// SetDetached detaches the host's HEAD.
func (fl *FakeFleet) SetDetached(address string) {
	defer fl.lock()()
	m := fl.host(address)
	m.branch, m.detached, m.issueStatus = "", true, ""
}

// SetIssueStatus sets the declared status of the host branch's issue.
func (fl *FakeFleet) SetIssueStatus(address, status string) {
	defer fl.lock()()
	fl.host(address).issueStatus = status
}

// Claim records this machine's claim of ref on the slot's host, made from the
// slot itself (claimant workspace = address).
func (fl *FakeFleet) Claim(address, ref string) {
	path := fl.SlotPath(address)
	fl.ClaimAt(path, ref, address)
}

// ClaimAt records a claim on any present checkout; workspace "" is the shape a
// dependency or plain clone records.
func (fl *FakeFleet) ClaimAt(path, ref, workspace string) {
	defer fl.lock()()
	m := fl.member(path)
	m.claims = append(m.claims, FleetClaim{Ref: ref, Status: "working", Revision: strings.Repeat("a", 40), Claimant: FleetClaimant{
		Operator: "Fake", Machine: fl.owner.Machine.Fingerprint, MachineName: fl.owner.Machine.Name, Workspace: workspace, Worktree: path, Repository: "example.com/" + filepath.Base(m.repoRoot),
	}})
}

// Release drops every claim on the slot's members.
func (fl *FakeFleet) Release(address string) {
	defer fl.lock()()
	for _, m := range fl.slot(address).members {
		m.claims = nil
	}
}

// SetClaimsState sets the claim read quality of every member of the slot.
// Unknown and absent reads carry no claims, as sdlc prints them.
func (fl *FakeFleet) SetClaimsState(address, state, reason string) {
	defer fl.lock()()
	for _, m := range fl.slot(address).members {
		m.claimsState, m.claimsError = state, reason
	}
}

// RemoveSlot deletes the slot's checkouts; their claims become dangling.
func (fl *FakeFleet) RemoveSlot(address string) {
	defer fl.lock()()
	s := fl.slot(address)
	for _, m := range s.members {
		for _, c := range m.claims {
			fl.dangling = append(fl.dangling, FleetDanglingClaim{RepoIdentity: filepath.Join(m.repoRoot, ".git"), RepoRoot: m.repoRoot, FleetClaim: c})
		}
	}
	delete(fl.slots, address)
}

// AddDanglingClaim records a claim whose worktree is no row at all.
func (fl *FakeFleet) AddDanglingClaim(repoRoot, worktree, ref, workspace string) {
	defer fl.lock()()
	fl.dangling = append(fl.dangling, FleetDanglingClaim{RepoIdentity: filepath.Join(repoRoot, ".git"), RepoRoot: repoRoot, FleetClaim: FleetClaim{
		Ref: ref, Status: "working", Revision: strings.Repeat("b", 40), Claimant: FleetClaimant{
			Operator: "Fake", Machine: fl.owner.Machine.Fingerprint, MachineName: fl.owner.Machine.Name, Workspace: workspace, Worktree: worktree, Repository: "example.com/" + filepath.Base(repoRoot)},
	}})
}

// fakeTerminalIssueStatuses mirrors ariadne's issue vocabulary terminal set.
var fakeTerminalIssueStatuses = []string{"done", "wontfix", "punt"}

func (m *fakeFleetMember) row() FleetRow {
	zero := 0
	dirty, ahead := m.dirty, m.ahead
	row := FleetRow{
		RepoIdentity: filepath.Join(m.repoRoot, ".git"), RepoRoot: m.repoRoot, TreePath: m.path,
		Branch: m.branch, Detached: m.detached,
		Facts:       FleetFacts{Available: true, BaseAvailable: !m.baseUnavailable, Behind: &zero, DirtyCount: &dirty},
		Issues:      []FleetIssue{},
		Claims:      []FleetClaim{},
		ClaimsState: m.claimsState, ClaimsError: m.claimsError,
	}
	if !m.baseUnavailable {
		row.Facts.Ahead = &ahead
	} else {
		row.Facts.Behind = nil
	}
	if match := fakeIssueBranch.FindStringSubmatch(m.branch); match != nil && m.issueStatus != "" {
		row.Issues = append(row.Issues, FleetIssue{Ref: filepath.Base(m.repoRoot) + "#" + match[1], DeclaredStatus: m.issueStatus, Provenance: "branch-prefix"})
	}
	if m.claimsState == FleetClaimsPresent || m.claimsState == FleetClaimsStale || m.claimsState == FleetClaimsPartial {
		row.Claims = append(row.Claims, m.claims...)
	}
	return row
}

// fakeJudge is sdlc's JudgeCheckout over the fields the fake models. The
// operation is not a JSON fact (sdlc prints it only as a reason), so it is
// passed beside the row it judges.
func fakeJudge(row FleetRow, operation, resting string) (string, []string) {
	var recovery, holds, probes []string
	f := row.Facts
	if f.DirtyCount != nil && *f.DirtyCount > 0 {
		recovery = append(recovery, "dirty")
	}
	if operation != "" {
		recovery = append(recovery, "operation:"+operation)
	}
	if !f.BaseAvailable || f.Ahead == nil {
		probes = append(probes, "probe:base")
	} else if *f.Ahead > 0 {
		holds = append(holds, "unlanded-commits")
	}
	if row.Detached {
		recovery = append(recovery, "detached")
	}
	if row.Branch != "" && row.Branch != resting {
		for _, issue := range row.Issues {
			if !slices.Contains(fakeTerminalIssueStatuses, issue.DeclaredStatus) {
				holds = append(holds, "open-issue:"+issue.Ref)
			}
		}
	}
	switch row.ClaimsState {
	case FleetClaimsPresent, FleetClaimsStale, FleetClaimsPartial:
		for _, c := range row.Claims {
			holds = append(holds, "claimed:"+c.Ref)
		}
	case FleetClaimsAbsent:
	default:
		probes = append(probes, "probe:claims")
	}
	switch {
	case len(recovery) > 0:
		return FleetVerdictNeedsRecovery, append(recovery, probes...)
	case len(holds) > 0 && len(probes) == 1 && probes[0] == "probe:claims":
		return FleetVerdictHoldsWork, holds
	case len(probes) > 0:
		return FleetVerdictUnknown, probes
	case len(holds) > 0:
		return FleetVerdictHoldsWork, holds
	}
	return FleetVerdictReady, []string{}
}

var fleetVerdictRank = map[string]int{FleetVerdictReady: 0, FleetVerdictHoldsWork: 1, FleetVerdictUnknown: 2, FleetVerdictMissing: 3, FleetVerdictNeedsRecovery: 4}

func fakeWorst(verdicts []string) string {
	worst := FleetVerdictReady
	for _, v := range verdicts {
		if fleetVerdictRank[v] > fleetVerdictRank[worst] {
			worst = v
		}
	}
	return worst
}

// Inventory is the document the fake prints for one fleet.
func (fl *FakeFleet) Inventory() FleetInventory {
	defer fl.lock()()
	return fl.inventoryLocked()
}

func (fl *FakeFleet) inventoryLocked() FleetInventory {
	schema := fl.owner.Schema
	if fl.schema != 0 {
		schema = fl.schema
	}
	inv := FleetInventory{SchemaVersion: schema, Machine: fl.owner.Machine, Rows: []FleetRow{}, Slots: []FleetSlot{}, DanglingClaims: append([]FleetDanglingClaim{}, fl.dangling...), Diagnostics: []FleetDiagnostic{}}
	addresses := make([]string, 0, len(fl.slots))
	for address := range fl.slots {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	for _, address := range addresses {
		s := fl.slots[address]
		slot := FleetSlot{Address: address, Repo: s.repo, Slot: s.number, RestingBranch: RestingBranch(s.number)}
		if s.number > 0 {
			slot.EnvironmentRoot = filepath.Dir(fl.SlotPath(address))
		}
		var verdicts []string
		for _, m := range s.members {
			member := FleetMember{Role: m.role, Path: m.path, RestingBranch: m.resting}
			if m.missing {
				member.Verdict, member.Reasons = FleetVerdictMissing, []string{"missing"}
			} else {
				row := m.row()
				inv.Rows = append(inv.Rows, row)
				member.Branch = m.branch
				member.Verdict, member.Reasons = fakeJudge(row, m.operation, m.resting)
			}
			verdicts = append(verdicts, member.Verdict)
			slot.Members = append(slot.Members, member)
		}
		slot.Verdict = fakeWorst(verdicts)
		inv.Slots = append(inv.Slots, slot)
	}
	paths := make([]string, 0, len(fl.offSlot))
	for path := range fl.offSlot {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		inv.Rows = append(inv.Rows, fl.offSlot[path].row())
	}
	return inv
}

// Run is ProvisionIO. It answers only `sdlc fleet inventory --json --path V`
// run from V, and refuses any other program or argv.
func (f *FakeFleetSDLC) Run(ctx context.Context, request ProvisionCommand) ([]byte, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, ProvisionCommand{Dir: request.Dir, Program: request.Program, Args: append([]string(nil), request.Args...), Timeout: request.Timeout})
	f.mu.Unlock()
	if ctx == nil {
		return nil, errors.New("fake sdlc: nil context")
	}
	if request.Program != "sdlc" {
		return nil, fmt.Errorf("fake sdlc: refuses program %q", request.Program)
	}
	vantage := request.Dir
	if want := []string{"fleet", "inventory", "--json", "--path", vantage}; !slices.Equal(request.Args, want) {
		return nil, fmt.Errorf("fake sdlc: refuses argv %q", request.Args)
	}
	switch f.Fail {
	case FakeFleetExit:
		return nil, errors.New("fake sdlc: exit status 1")
	case FakeFleetHang:
		<-ctx.Done()
		return nil, ctx.Err()
	case FakeFleetGarbage:
		return []byte("sdlc: not an inventory\n"), nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	fleet := f.fleetFor(vantage)
	if fleet == nil {
		return nil, fmt.Errorf("fake sdlc: %s is in no fleet", vantage)
	}
	inv := fleet.inventoryLocked()
	raw, err := json.Marshal(inv)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// fleetFor resolves a vantage to its fleet: an explicit Route, else the
// longest fleet root containing it. Caller holds f.mu.
func (f *FakeFleetSDLC) fleetFor(vantage string) *FakeFleet {
	if root, ok := f.Route[vantage]; ok {
		return f.fleets[filepath.Clean(root)]
	}
	var best *FakeFleet
	for root, fleet := range f.fleets {
		if (vantage == root || strings.HasPrefix(vantage, root+string(filepath.Separator))) && (best == nil || len(root) > len(best.root)) {
			best = fleet
		}
	}
	return best
}
