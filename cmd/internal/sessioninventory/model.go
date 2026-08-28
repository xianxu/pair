// Package sessioninventory owns Pair's deterministic model of native agent
// session forests.
package sessioninventory

import (
	"path"
	"sort"
	"strings"
	"time"
)

type Agent string

const (
	AgentClaude Agent = "claude"
	AgentCodex  Agent = "codex"
	AgentAgy    Agent = "agy"
	AgentMuse   Agent = "muse"
)

type Role string

const (
	RoleUnknown  Role = "unknown"
	RoleRoot     Role = "root"
	RoleSubagent Role = "subagent"
)

type TimeSource string

const (
	TimeSourceMetadata TimeSource = "metadata"
	TimeSourceBirth    TimeSource = "birth"
	TimeSourceMTime    TimeSource = "mtime"
)

type NativeTime struct {
	Value  time.Time  `json:"value"`
	Source TimeSource `json:"source"`
}

type ArtifactKind string

const (
	ArtifactTranscript ArtifactKind = "transcript"
	ArtifactDatabase   ArtifactKind = "database"
	ArtifactMetadata   ArtifactKind = "metadata"
)

type Artifact struct {
	StorageRoot  string       `json:"storage_root"`
	RelativePath string       `json:"relative_path"`
	Kind         ArtifactKind `json:"kind"`
}

// pair:155-concept pure new M1
type EdgeProvenance struct {
	Schema   string   `json:"schema"`
	Artifact Artifact `json:"artifact"`
}

// pair:155-concept pure new M1
type ParentEdge struct {
	StableID   string           `json:"stable_id"`
	ParentID   string           `json:"parent_id"`
	ChildID    string           `json:"child_id"`
	Provenance []EdgeProvenance `json:"provenance"`
}

// Fact is a scanner-owned assertion about one native session node. BuildForest
// is the only place facts become parent edges.
type Fact struct {
	Agent          Agent            `json:"agent"`
	NativeID       string           `json:"native_id"`
	Role           Role             `json:"role"`
	ParentID       *string          `json:"parent_id"`
	Time           *NativeTime      `json:"time"`
	Resumable      bool             `json:"resumable"`
	Disputed       bool             `json:"disputed"`
	Artifacts      []Artifact       `json:"artifacts"`
	EdgeProvenance []EdgeProvenance `json:"edge_provenance"`
}

// pair:155-concept pure new M1
type NativeRecordFact = Fact

type Node struct {
	StableID   string      `json:"stable_id"`
	Agent      Agent       `json:"agent"`
	NativeID   string      `json:"native_id"`
	Role       Role        `json:"role"`
	ParentID   *string     `json:"parent_id"`
	Time       *NativeTime `json:"time"`
	Resumable  bool        `json:"resumable"`
	Artifacts  []Artifact  `json:"artifacts"`
	ParentEdge *ParentEdge `json:"parent_edge"`
	Children   []Node      `json:"children"`
}

// pair:155-concept pure new M1
type SessionNode = Node

type Forest struct {
	Agent   Agent  `json:"agent"`
	Roots   []Node `json:"roots"`
	Orphans []Node `json:"orphans"`
}

// pair:155-concept pure new M1
type SessionForest = Forest

type DiagnosticCode string

const (
	DiagnosticStorageUnreadable           DiagnosticCode = "storage_unreadable"
	DiagnosticStorageAbsent               DiagnosticCode = "storage_absent"
	DiagnosticNodeMalformed               DiagnosticCode = "node_malformed"
	DiagnosticSchemaNearMiss              DiagnosticCode = "schema_near_miss"
	DiagnosticParentMissing               DiagnosticCode = "parent_missing"
	DiagnosticParentConflict              DiagnosticCode = "parent_conflict"
	DiagnosticDuplicateConflict           DiagnosticCode = "duplicate_conflict"
	DiagnosticBindingStale                DiagnosticCode = "binding_stale"
	DiagnosticBindingConflict             DiagnosticCode = "binding_conflict"
	DiagnosticProcessChanged              DiagnosticCode = "process_changed"
	DiagnosticTurnUnusable                DiagnosticCode = "turn_unusable"
	DiagnosticConformanceNoSample         DiagnosticCode = "conformance_no_sample"
	DiagnosticSendIncomplete              DiagnosticCode = "send_incomplete"
	DiagnosticSendAborted                 DiagnosticCode = "send_aborted"
	DiagnosticArtifactPathInvalid         DiagnosticCode = "artifact_path_invalid"
	DiagnosticPairRecordMalformed         DiagnosticCode = "pair_record_malformed"
	DiagnosticScopeRejected               DiagnosticCode = "scope_rejected"
	DiagnosticConformancePrivacyViolation DiagnosticCode = "conformance_privacy_violation"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// pair:155-concept pure new M1
type Diagnostic struct {
	StableID  string         `json:"stable_id"`
	Code      DiagnosticCode `json:"code"`
	Severity  Severity       `json:"severity"`
	Agent     Agent          `json:"agent"`
	NativeID  *string        `json:"native_id"`
	Path      *Artifact      `json:"path"`
	SourceRef *string        `json:"source_ref"`
	Detail    string         `json:"detail"`
}

type factKey struct {
	agent    Agent
	nativeID string
}

type canonicalNode struct {
	node       Node
	conflicted bool
	provenance []EdgeProvenance
}

// BuildForest coalesces scanner facts and creates only parent edges that all
// facts agree on. Conflicting, missing, and cyclic edges fail closed as orphans.
func BuildForest(facts []Fact) Inventory {
	groups := make(map[factKey][]Fact)
	var diagnostics []Diagnostic
	for _, fact := range facts {
		if !validAgent(fact.Agent) || fact.NativeID == "" || (!validRole(fact.Role) && !fact.Disputed) {
			diagnostics = append(diagnostics, diagnostic(DiagnosticNodeMalformed, fact.Agent, optionalString(fact.NativeID), "invalid agent, native ID, or role"))
			continue
		}
		cleaned := fact
		cleaned.ParentID = cloneString(fact.ParentID)
		cleaned.Time = cloneTime(fact.Time)
		cleaned.Artifacts = nil
		for _, artifact := range fact.Artifacts {
			if !validArtifact(artifact) {
				d := diagnostic(DiagnosticNodeMalformed, fact.Agent, &fact.NativeID, "artifact path is not canonical and relative")
				d.Path = &Artifact{StorageRoot: artifact.StorageRoot, RelativePath: artifact.RelativePath}
				d.StableID = diagnosticID(d)
				diagnostics = append(diagnostics, d)
				continue
			}
			cleaned.Artifacts = append(cleaned.Artifacts, artifact)
		}
		key := factKey{agent: fact.Agent, nativeID: fact.NativeID}
		groups[key] = append(groups[key], cleaned)
	}

	keys := make([]factKey, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].agent != keys[j].agent {
			return keys[i].agent < keys[j].agent
		}
		return keys[i].nativeID < keys[j].nativeID
	})

	byAgent := make(map[Agent]map[string]canonicalNode)
	for _, key := range keys {
		canonical, found := canonicalizeFacts(key, groups[key])
		diagnostics = append(diagnostics, found...)
		if byAgent[key.agent] == nil {
			byAgent[key.agent] = make(map[string]canonicalNode)
		}
		byAgent[key.agent][key.nativeID] = canonical
	}

	forests := make([]Forest, 0, len(byAgent))
	for agent, nodes := range byAgent {
		forest, found := buildAgentForest(agent, nodes)
		forests = append(forests, forest)
		diagnostics = append(diagnostics, found...)
	}

	return SortInventory(Inventory{Forests: forests, Diagnostics: diagnostics})
}

func canonicalizeFacts(key factKey, facts []Fact) (canonicalNode, []Diagnostic) {
	roles := make(map[Role]struct{})
	parents := make(map[string]*string)
	artifacts := make(map[string]Artifact)
	provenance := make(map[string]EdgeProvenance)
	var earliest *NativeTime
	for _, fact := range facts {
		roles[fact.Role] = struct{}{}
		parentKey := "<nil>"
		if fact.ParentID != nil {
			parentKey = *fact.ParentID
		}
		parents[parentKey] = cloneString(fact.ParentID)
		for _, artifact := range fact.Artifacts {
			artifacts[artifact.StorageRoot+"\x00"+artifact.RelativePath+"\x00"+string(artifact.Kind)] = artifact
		}
		for _, item := range fact.EdgeProvenance {
			if item.Schema == "" || !validArtifact(item.Artifact) {
				continue
			}
			itemKey := item.Schema + "\x00" + item.Artifact.StorageRoot + "\x00" + item.Artifact.RelativePath + "\x00" + string(item.Artifact.Kind)
			provenance[itemKey] = item
		}
		if earliest == nil || compareNativeTime(fact.Time, earliest) < 0 {
			earliest = cloneTime(fact.Time)
		}
	}

	role := firstRole(roles)
	parent := firstParent(parents)
	conflicted := false
	var diagnostics []Diagnostic
	if len(roles) != 1 {
		role = RoleUnknown
		parent = nil
		conflicted = true
		diagnostics = append(diagnostics, diagnostic(DiagnosticDuplicateConflict, key.agent, &key.nativeID, "duplicate facts disagree on role"))
	}
	if len(parents) != 1 {
		parent = nil
		conflicted = true
		diagnostics = append(diagnostics, diagnostic(DiagnosticParentConflict, key.agent, &key.nativeID, "duplicate facts disagree on parent"))
	}
	if role == RoleRoot && parent != nil {
		parent = nil
		conflicted = true
		diagnostics = append(diagnostics, diagnostic(DiagnosticParentConflict, key.agent, &key.nativeID, "root fact names a parent"))
	}
	if role == RoleSubagent && parent == nil && !conflicted {
		diagnostics = append(diagnostics, diagnostic(DiagnosticParentMissing, key.agent, &key.nativeID, "subagent fact has no parent"))
	}
	for _, fact := range facts {
		if fact.Disputed {
			role = RoleUnknown
			parent = nil
			conflicted = true
			break
		}
	}

	artifactList := make([]Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactList = append(artifactList, artifact)
	}
	provenanceList := make([]EdgeProvenance, 0, len(provenance))
	for _, item := range provenance {
		provenanceList = append(provenanceList, item)
	}
	sort.Slice(provenanceList, func(i, j int) bool {
		if provenanceList[i].Schema != provenanceList[j].Schema {
			return provenanceList[i].Schema < provenanceList[j].Schema
		}
		return compareArtifact(provenanceList[i].Artifact, provenanceList[j].Artifact) < 0
	})
	node := Node{
		StableID:  StableID("node", string(key.agent), key.nativeID),
		Agent:     key.agent,
		NativeID:  key.nativeID,
		Role:      role,
		ParentID:  parent,
		Time:      earliest,
		Resumable: firstResumable(facts) && !conflicted,
		Artifacts: artifactList,
	}
	return canonicalNode{node: node, conflicted: conflicted, provenance: provenanceList}, diagnostics
}

func buildAgentForest(agent Agent, canonical map[string]canonicalNode) (Forest, []Diagnostic) {
	links := make(map[string]string)
	var diagnostics []Diagnostic
	for nativeID, entry := range canonical {
		node := entry.node
		if entry.conflicted || node.Role != RoleSubagent || node.ParentID == nil {
			continue
		}
		parentID := *node.ParentID
		parent, ok := canonical[parentID]
		switch {
		case parentID == nativeID:
			diagnostics = append(diagnostics, diagnostic(DiagnosticParentConflict, agent, &nativeID, "node names itself as parent"))
		case !ok || parent.conflicted || parent.node.Role == RoleUnknown:
			diagnostics = append(diagnostics, diagnostic(DiagnosticParentMissing, agent, &nativeID, "parent is absent or disputed"))
		default:
			links[nativeID] = parentID
		}
	}

	for _, cycleID := range cyclicNodes(links) {
		delete(links, cycleID)
		diagnostics = append(diagnostics, diagnostic(DiagnosticParentConflict, agent, &cycleID, "parent edge participates in a cycle"))
	}

	children := make(map[string][]string)
	for child, parent := range links {
		children[parent] = append(children[parent], child)
	}
	for parent := range children {
		sort.Strings(children[parent])
	}
	var materialize func(string) Node
	materialize = func(nativeID string) Node {
		node := cloneNode(canonical[nativeID].node)
		if parentID, attached := links[nativeID]; attached {
			node.ParentEdge = &ParentEdge{
				StableID:   StableID("edge", string(agent), parentID, nativeID),
				ParentID:   parentID,
				ChildID:    nativeID,
				Provenance: append([]EdgeProvenance(nil), canonical[nativeID].provenance...),
			}
		}
		node.Children = make([]Node, 0, len(children[nativeID]))
		for _, child := range children[nativeID] {
			node.Children = append(node.Children, materialize(child))
		}
		return node
	}

	forest := Forest{Agent: agent}
	ids := make([]string, 0, len(canonical))
	for nativeID := range canonical {
		ids = append(ids, nativeID)
	}
	sort.Strings(ids)
	for _, nativeID := range ids {
		entry := canonical[nativeID]
		if _, attached := links[nativeID]; attached {
			continue
		}
		if !entry.conflicted && entry.node.Role == RoleRoot && entry.node.ParentID == nil {
			forest.Roots = append(forest.Roots, materialize(nativeID))
		} else {
			forest.Orphans = append(forest.Orphans, materialize(nativeID))
		}
	}
	return forest, diagnostics
}

func cyclicNodes(links map[string]string) []string {
	cyclic := make(map[string]struct{})
	starts := make([]string, 0, len(links))
	for child := range links {
		starts = append(starts, child)
	}
	sort.Strings(starts)
	for _, start := range starts {
		positions := make(map[string]int)
		var chain []string
		for current := start; ; {
			if index, seen := positions[current]; seen {
				for _, nativeID := range chain[index:] {
					cyclic[nativeID] = struct{}{}
				}
				break
			}
			positions[current] = len(chain)
			chain = append(chain, current)
			parent, ok := links[current]
			if !ok {
				break
			}
			current = parent
		}
	}
	result := make([]string, 0, len(cyclic))
	for nativeID := range cyclic {
		result = append(result, nativeID)
	}
	sort.Strings(result)
	return result
}

func diagnostic(code DiagnosticCode, agent Agent, nativeID *string, detail string) Diagnostic {
	severity, ok := diagnosticSeverity(code)
	if !ok {
		severity = SeverityError
	}
	d := Diagnostic{Code: code, Severity: severity, Agent: agent, NativeID: cloneString(nativeID), Detail: detail}
	d.StableID = diagnosticID(d)
	return d
}

func diagnosticID(d Diagnostic) string {
	nativeID := ""
	if d.NativeID != nil {
		nativeID = *d.NativeID
	}
	storageRoot := ""
	relativePath := ""
	if d.Path != nil {
		storageRoot = d.Path.StorageRoot
		relativePath = d.Path.RelativePath
	}
	sourceRef := ""
	if d.SourceRef != nil {
		sourceRef = *d.SourceRef
	}
	return StableID("diagnostic", string(d.Severity), string(d.Code), string(d.Agent), nativeID, storageRoot, relativePath, sourceRef)
}

func validAgent(agent Agent) bool {
	switch agent {
	case AgentClaude, AgentCodex, AgentAgy, AgentMuse:
		return true
	default:
		return false
	}
}

func validRole(role Role) bool { return role == RoleRoot || role == RoleSubagent }

func validArtifact(artifact Artifact) bool {
	if artifact.StorageRoot == "" || artifact.RelativePath == "" || strings.Contains(artifact.RelativePath, "\\") {
		return false
	}
	clean := path.Clean(artifact.RelativePath)
	return clean == artifact.RelativePath && clean != "." && !path.IsAbs(clean) && clean != ".." && !strings.HasPrefix(clean, "../")
}

func firstRole(values map[Role]struct{}) Role {
	roles := make([]Role, 0, len(values))
	for role := range values {
		roles = append(roles, role)
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i] < roles[j] })
	if len(roles) == 0 {
		return RoleUnknown
	}
	return roles[0]
}

func firstParent(values map[string]*string) *string {
	parents := make([]string, 0, len(values))
	for parent := range values {
		parents = append(parents, parent)
	}
	sort.Strings(parents)
	if len(parents) == 0 {
		return nil
	}
	return cloneString(values[parents[0]])
}

func firstResumable(facts []Fact) bool {
	for _, fact := range facts {
		if fact.Resumable {
			return true
		}
	}
	return false
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTime(value *NativeTime) *NativeTime {
	if value == nil {
		return nil
	}
	return &NativeTime{Value: value.Value, Source: value.Source}
}

func cloneNode(value Node) Node {
	cloned := value
	cloned.ParentID = cloneString(value.ParentID)
	cloned.Time = cloneTime(value.Time)
	cloned.Artifacts = append([]Artifact(nil), value.Artifacts...)
	if value.ParentEdge != nil {
		edge := *value.ParentEdge
		edge.Provenance = append([]EdgeProvenance(nil), value.ParentEdge.Provenance...)
		cloned.ParentEdge = &edge
	}
	cloned.Children = nil
	return cloned
}

func compareNativeTime(left, right *NativeTime) int {
	switch {
	case left == nil && right == nil:
		return 0
	case left == nil:
		return 1
	case right == nil:
		return -1
	}
	if compared := left.Value.UTC().Compare(right.Value.UTC()); compared != 0 {
		return compared
	}
	return 0
}
