package sessioninventory

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

type BindingStatus string

const (
	BindingProvisional BindingStatus = "provisional"
	BindingEstablished BindingStatus = "established"
	BindingAmbiguous   BindingStatus = "ambiguous"
	BindingUnbound     BindingStatus = "unbound"
)

type EvidenceKind string

const (
	EvidenceLedger       EvidenceKind = "ledger"
	EvidenceLiveRound    EvidenceKind = "live_round"
	EvidenceOfflineRound EvidenceKind = "offline_round"
	EvidenceConfig       EvidenceKind = "config"
)

type CandidateOutcome string

const (
	CandidateLocked   CandidateOutcome = "locked"
	CandidateConflict CandidateOutcome = "conflict"
	CandidateRejected CandidateOutcome = "rejected"
)

// Evidence explains one candidate edge without retaining authored text.
// pair:155-concept pure new M2
type Evidence struct {
	StableID             string       `json:"evidence_id"`
	Kind                 EvidenceKind `json:"kind"`
	Rank                 int          `json:"rank"`
	SourceRef            string       `json:"source_ref"`
	RootNodeID           string       `json:"root_node_id"`
	Positive             bool         `json:"positive"`
	SourcePositions      []uint64     `json:"source_positions"`
	DestinationPositions []uint64     `json:"destination_positions"`
	Fingerprints         []string     `json:"fingerprints"`
}

// Candidate is one scanner-authorized root considered at the selected rank.
// pair:155-concept pure new M2
type Candidate struct {
	RootNodeID  string           `json:"root_node_id"`
	Rank        int              `json:"rank"`
	Outcome     CandidateOutcome `json:"outcome"`
	EvidenceIDs []string         `json:"evidence_ids"`
}

// Ambiguity retains equal qualifying roots instead of selecting by chronology.
// pair:155-concept pure new M2
type Ambiguity struct {
	StableID    string   `json:"ambiguity_id"`
	Kind        string   `json:"kind"`
	Rank        int      `json:"rank"`
	BindingIDs  []string `json:"binding_ids"`
	RootNodeIDs []string `json:"root_node_ids"`
	EvidenceIDs []string `json:"evidence_ids"`
}

// Binding is Pair's provisional or established relation to one native root.
// pair:155-concept pure new M2
type Binding struct {
	StableID   string        `json:"binding_id"`
	ScopeKey   string        `json:"scope_key"`
	Tag        string        `json:"tag"`
	Agent      Agent         `json:"agent"`
	RootNodeID *string       `json:"root_node_id"`
	Status     BindingStatus `json:"status"`
	Candidates []Candidate   `json:"candidates"`
	Evidence   []Evidence    `json:"evidence"`
}

// BindingInput supplies already-parsed durable/live facts for one Pair owner.
type BindingInput struct {
	ScopeKey          string
	Tag               string
	Agent             Agent
	LaunchPresent     bool
	LedgerRootNodeID  string
	LedgerRootNodeIDs []string
	LiveRounds        []RoundObservation
	OfflineRounds     []RoundObservation
	ConfigRootNodeID  string
}

// NodeBinding is binding inheritance through one validated native parent edge.
type NodeBinding struct {
	BindingID      string
	RootNodeID     string
	NodeID         string
	EdgeProvenance []EdgeProvenance
}

type bindingWork struct {
	input      BindingInput
	binding    Binding
	rank       int
	authority  EvidenceKind
	active     map[string]bool
	all        map[string]*Candidate
	lockedRoot string
}

// ResolveBindings applies evidence precedence, round-set intersection, and
// global Pair-owner root exclusivity without consulting timestamps.
func ResolveBindings(inventory Inventory, inputs []BindingInput) Inventory {
	output := SortInventory(inventory)
	authorized := authorizedRoots(output.Forests)
	works := make([]bindingWork, 0, len(inputs))
	for _, input := range inputs {
		work, diagnostics := prepareBinding(input, authorized[input.Agent])
		works = append(works, work)
		output.Diagnostics = append(output.Diagnostics, diagnostics...)
	}
	slices.SortFunc(works, func(a, b bindingWork) int { return compareBindingOwner(a.binding, b.binding) })

	reserved := map[string]string{}
	for rank := 1; rank <= 4; rank++ {
		for i := range works {
			if works[i].rank != rank {
				continue
			}
			for root := range works[i].active {
				if owner, exists := reserved[root]; exists && owner != bindingOwner(works[i].binding) {
					delete(works[i].active, root)
					works[i].all[root].Outcome = CandidateRejected
				}
			}
		}
		changed := true
		for changed {
			changed = false
			for i := range works {
				work := &works[i]
				if work.rank != rank || work.lockedRoot != "" || len(work.active) != 1 {
					continue
				}
				root := onlyRoot(work.active)
				owner := bindingOwner(work.binding)
				if competingOwner(works, rank, root, owner) {
					continue
				}
				work.lockedRoot = root
				work.all[root].Outcome = CandidateLocked
				reserved[root] = owner
				changed = true
				for j := range works {
					if j == i || works[j].rank != rank || bindingOwner(works[j].binding) == owner || !works[j].active[root] {
						continue
					}
					delete(works[j].active, root)
					works[j].all[root].Outcome = CandidateRejected
				}
			}
		}
	}

	for i := range works {
		work := &works[i]
		if work.lockedRoot != "" {
			work.binding.RootNodeID = cloneString(&work.lockedRoot)
			if work.authority == EvidenceLedger || work.authority == EvidenceConfig {
				work.binding.Status = BindingEstablished
			} else {
				work.binding.Status = BindingProvisional
			}
		} else if len(work.active) != 0 {
			work.binding.Status = BindingAmbiguous
		} else if work.input.LaunchPresent {
			work.binding.Status = BindingProvisional
		} else {
			work.binding.Status = BindingUnbound
		}
		work.binding.StableID = StableID("binding", work.binding.ScopeKey, work.binding.Tag, string(work.binding.Agent), valueOrEmpty(work.binding.RootNodeID))
		for _, evidence := range work.binding.Evidence {
			if candidate := work.all[evidence.RootNodeID]; candidate != nil {
				candidate.EvidenceIDs = append(candidate.EvidenceIDs, evidence.StableID)
			}
		}
		for _, candidate := range work.all {
			slices.Sort(candidate.EvidenceIDs)
			work.binding.Candidates = append(work.binding.Candidates, *candidate)
		}
		slices.SortFunc(work.binding.Candidates, compareCandidate)
		for e := range work.binding.Evidence {
			work.binding.Evidence[e].Positive = work.lockedRoot != "" && work.binding.Evidence[e].RootNodeID == work.lockedRoot && work.binding.Evidence[e].Rank == work.rank
		}
		slices.SortFunc(work.binding.Evidence, compareEvidence)
		if work.binding.Status == BindingAmbiguous {
			for root := range work.active {
				if competingOwner(works, work.rank, root, bindingOwner(work.binding)) {
					output.Diagnostics = append(output.Diagnostics, bindingDiagnostic(DiagnosticBindingConflict, work.input, root, "competing Pair owners"))
				}
			}
			if len(work.active) > 1 {
				output.Ambiguities = append(output.Ambiguities, ambiguityFor(*work))
			}
		}
		output.Bindings = append(output.Bindings, work.binding)
	}
	output.Ambiguities = append(output.Ambiguities, competingOwnerAmbiguities(works)...)
	slices.SortFunc(output.Bindings, compareBinding)
	slices.SortFunc(output.Ambiguities, compareAmbiguity)
	return SortInventory(output)
}

func authorizedRoots(forests []Forest) map[Agent]map[string]bool {
	result := map[Agent]map[string]bool{}
	for _, forest := range forests {
		if result[forest.Agent] == nil {
			result[forest.Agent] = map[string]bool{}
		}
		for _, root := range forest.Roots {
			result[forest.Agent][root.StableID] = true
		}
	}
	return result
}

func prepareBinding(input BindingInput, authorized map[string]bool) (bindingWork, []Diagnostic) {
	binding := Binding{ScopeKey: input.ScopeKey, Tag: input.Tag, Agent: input.Agent}
	work := bindingWork{input: input, binding: binding, active: map[string]bool{}, all: map[string]*Candidate{}}
	var diagnostics []Diagnostic
	addDirect := func(kind EvidenceKind, rank int, root, source string) bool {
		if root == "" {
			return false
		}
		if !authorized[root] {
			diagnostics = append(diagnostics, bindingDiagnostic(DiagnosticBindingStale, input, root, source))
			return false
		}
		work.binding.Evidence = append(work.binding.Evidence, makeEvidence(input, kind, rank, root, source, nil))
		return true
	}
	ledgerRoots := append([]string(nil), input.LedgerRootNodeIDs...)
	if input.LedgerRootNodeID != "" {
		ledgerRoots = append(ledgerRoots, input.LedgerRootNodeID)
	}
	slices.Sort(ledgerRoots)
	ledgerRoots = slices.Compact(ledgerRoots)
	validLedgerRoots := map[string]bool{}
	for _, root := range ledgerRoots {
		if addDirect(EvidenceLedger, 1, root, "ledger") {
			validLedgerRoots[root] = true
		}
	}
	liveSets := addRoundEvidence(&work, input, EvidenceLiveRound, 2, input.LiveRounds, authorized)
	offlineSets := addRoundEvidence(&work, input, EvidenceOfflineRound, 3, input.OfflineRounds, authorized)
	configValid := addDirect(EvidenceConfig, 4, input.ConfigRootNodeID, "config")

	var selected map[string]bool
	switch {
	case len(validLedgerRoots) != 0:
		work.rank, work.authority, selected = 1, EvidenceLedger, validLedgerRoots
	case len(liveSets) != 0:
		work.rank, work.authority, selected = 2, EvidenceLiveRound, intersectRootSets(liveSets)
	case len(offlineSets) != 0:
		work.rank, work.authority, selected = 3, EvidenceOfflineRound, intersectRootSets(offlineSets)
	case configValid && !input.LaunchPresent:
		work.rank, work.authority, selected = 4, EvidenceConfig, map[string]bool{input.ConfigRootNodeID: true}
	}
	for root := range selected {
		work.active[root] = true
		work.all[root] = &Candidate{RootNodeID: root, Rank: work.rank, Outcome: CandidateConflict}
	}
	for _, evidence := range work.binding.Evidence {
		if work.all[evidence.RootNodeID] == nil {
			work.all[evidence.RootNodeID] = &Candidate{RootNodeID: evidence.RootNodeID, Rank: evidence.Rank, Outcome: CandidateRejected}
		}
	}
	if len(validLedgerRoots) != 0 && input.ConfigRootNodeID != "" && !validLedgerRoots[input.ConfigRootNodeID] {
		diagnostics = append(diagnostics, bindingDiagnostic(DiagnosticBindingStale, input, input.ConfigRootNodeID, "config disagrees with ledger"))
	}
	return work, diagnostics
}

func addRoundEvidence(work *bindingWork, input BindingInput, kind EvidenceKind, rank int, observations []RoundObservation, authorized map[string]bool) []map[string]bool {
	groups := map[string]map[string]bool{}
	for _, observation := range observations {
		if (observation.ScopeKey != "" && observation.ScopeKey != input.ScopeKey) ||
			(observation.Tag != "" && observation.Tag != input.Tag) ||
			(observation.Agent != "" && observation.Agent != input.Agent) {
			continue
		}
		if !authorized[observation.RootNodeID] {
			continue
		}
		key := uint64sKey(observation.PairPositions)
		if groups[key] == nil {
			groups[key] = map[string]bool{}
		}
		groups[key][observation.RootNodeID] = true
		sourceRef := key + "->" + uint64sKey(observation.NativePositions)
		work.binding.Evidence = append(work.binding.Evidence, makeEvidence(input, kind, rank, observation.RootNodeID, sourceRef, &observation))
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	sets := make([]map[string]bool, 0, len(keys))
	for _, key := range keys {
		sets = append(sets, groups[key])
	}
	return sets
}

func makeEvidence(input BindingInput, kind EvidenceKind, rank int, root, source string, observation *RoundObservation) Evidence {
	evidence := Evidence{Kind: kind, Rank: rank, RootNodeID: root, SourceRef: source}
	if observation != nil {
		evidence.SourcePositions = slices.Clone(observation.PairPositions)
		evidence.DestinationPositions = slices.Clone(observation.NativePositions)
		evidence.Fingerprints = slices.Clone(observation.Fingerprints)
	}
	evidence.StableID = StableID("evidence", string(kind), source, input.ScopeKey, input.Tag, string(input.Agent), root)
	return evidence
}

func intersectRootSets(sets []map[string]bool) map[string]bool {
	if len(sets) == 0 {
		return nil
	}
	result := map[string]bool{}
	for root := range sets[0] {
		result[root] = true
	}
	for _, set := range sets[1:] {
		for root := range result {
			if !set[root] {
				delete(result, root)
			}
		}
	}
	return result
}

func competingOwner(works []bindingWork, rank int, root, owner string) bool {
	for _, work := range works {
		if work.rank == rank && work.lockedRoot == "" && work.active[root] && bindingOwner(work.binding) != owner {
			return true
		}
	}
	return false
}

func onlyRoot(roots map[string]bool) string {
	for root := range roots {
		return root
	}
	return ""
}

func bindingOwner(binding Binding) string {
	return binding.ScopeKey + "\x00" + binding.Tag + "\x00" + string(binding.Agent)
}

func bindingDiagnostic(code DiagnosticCode, input BindingInput, root, source string) Diagnostic {
	sourceRef := source + ":" + root
	return diagnosticWithSource(code, input.Agent, nil, sourceRef, "binding evidence does not authorize a unique current root")
}

func ambiguityFor(work bindingWork) Ambiguity {
	ambiguity := Ambiguity{Kind: "root", Rank: work.rank, BindingIDs: []string{work.binding.StableID}}
	for root := range work.active {
		ambiguity.RootNodeIDs = append(ambiguity.RootNodeIDs, root)
		if candidate := work.all[root]; candidate != nil {
			ambiguity.EvidenceIDs = append(ambiguity.EvidenceIDs, candidate.EvidenceIDs...)
		}
	}
	slices.Sort(ambiguity.RootNodeIDs)
	slices.Sort(ambiguity.EvidenceIDs)
	ambiguity.StableID = StableID("ambiguity", ambiguity.Kind, fmt.Sprint(ambiguity.Rank), strings.Join(ambiguity.BindingIDs, "\x00"), strings.Join(ambiguity.RootNodeIDs, "\x00"))
	return ambiguity
}

func competingOwnerAmbiguities(works []bindingWork) []Ambiguity {
	type conflictKey struct {
		rank int
		root string
	}
	groups := map[conflictKey]map[int]bool{}
	for i, work := range works {
		if work.binding.Status != BindingAmbiguous {
			continue
		}
		for root := range work.active {
			if competingOwner(works, work.rank, root, bindingOwner(work.binding)) {
				key := conflictKey{rank: work.rank, root: root}
				if groups[key] == nil {
					groups[key] = map[int]bool{}
				}
				groups[key][i] = true
			}
		}
	}
	result := make([]Ambiguity, 0, len(groups))
	for key, indexes := range groups {
		ambiguity := Ambiguity{Kind: "root_owner", Rank: key.rank, RootNodeIDs: []string{key.root}}
		for index := range indexes {
			work := works[index]
			ambiguity.BindingIDs = append(ambiguity.BindingIDs, work.binding.StableID)
			if candidate := work.all[key.root]; candidate != nil {
				ambiguity.EvidenceIDs = append(ambiguity.EvidenceIDs, candidate.EvidenceIDs...)
			}
		}
		slices.Sort(ambiguity.BindingIDs)
		slices.Sort(ambiguity.EvidenceIDs)
		ambiguity.EvidenceIDs = slices.Compact(ambiguity.EvidenceIDs)
		ambiguity.StableID = StableID("ambiguity", ambiguity.Kind, fmt.Sprint(ambiguity.Rank), strings.Join(ambiguity.BindingIDs, "\x00"), key.root)
		result = append(result, ambiguity)
	}
	slices.SortFunc(result, compareAmbiguity)
	return result
}

func compareBindingOwner(a, b Binding) int {
	for _, values := range [][2]string{{a.ScopeKey, b.ScopeKey}, {a.Tag, b.Tag}, {string(a.Agent), string(b.Agent)}} {
		if result := cmp.Compare(values[0], values[1]); result != 0 {
			return result
		}
	}
	return 0
}

func compareBinding(a, b Binding) int {
	if result := compareBindingOwner(a, b); result != 0 {
		return result
	}
	if result := cmp.Compare(nullableString(a.RootNodeID), nullableString(b.RootNodeID)); result != 0 {
		return result
	}
	return cmp.Compare(a.StableID, b.StableID)
}

func compareCandidate(a, b Candidate) int {
	if result := cmp.Compare(a.Rank, b.Rank); result != 0 {
		return result
	}
	if result := cmp.Compare(a.RootNodeID, b.RootNodeID); result != 0 {
		return result
	}
	if result := cmp.Compare(a.Outcome, b.Outcome); result != 0 {
		return result
	}
	return cmp.Compare(strings.Join(a.EvidenceIDs, "\x00"), strings.Join(b.EvidenceIDs, "\x00"))
}

func compareEvidence(a, b Evidence) int {
	if result := cmp.Compare(a.Rank, b.Rank); result != 0 {
		return result
	}
	for _, values := range [][2]string{{string(a.Kind), string(b.Kind)}, {a.SourceRef, b.SourceRef}, {a.StableID, b.StableID}} {
		if result := cmp.Compare(values[0], values[1]); result != 0 {
			return result
		}
	}
	return 0
}

func compareAmbiguity(a, b Ambiguity) int {
	if result := cmp.Compare(a.Kind, b.Kind); result != 0 {
		return result
	}
	if result := cmp.Compare(a.Rank, b.Rank); result != 0 {
		return result
	}
	for _, values := range [][2]string{
		{strings.Join(a.BindingIDs, "\x00"), strings.Join(b.BindingIDs, "\x00")},
		{strings.Join(a.RootNodeIDs, "\x00"), strings.Join(b.RootNodeIDs, "\x00")},
		{a.StableID, b.StableID},
	} {
		if result := cmp.Compare(values[0], values[1]); result != 0 {
			return result
		}
	}
	return 0
}

func uint64sKey(values []uint64) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprint(value)
	}
	return strings.Join(parts, ",")
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// PropagateBinding attaches descendants only through validated forest edges.
func PropagateBinding(binding Binding, forest Forest) []NodeBinding {
	if binding.Status != BindingEstablished || binding.RootNodeID == nil {
		return nil
	}
	var result []NodeBinding
	for _, root := range forest.Roots {
		if root.StableID != *binding.RootNodeID {
			continue
		}
		var visit func(Node)
		visit = func(parent Node) {
			for _, child := range parent.Children {
				if child.ParentEdge == nil || child.ParentEdge.ParentID != parent.NativeID || child.ParentEdge.ChildID != child.NativeID {
					continue
				}
				result = append(result, NodeBinding{BindingID: binding.StableID, RootNodeID: *binding.RootNodeID, NodeID: child.StableID, EdgeProvenance: slices.Clone(child.ParentEdge.Provenance)})
				visit(child)
			}
		}
		visit(root)
		break
	}
	return result
}
