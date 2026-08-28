package sessioninventory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type RenderFormat string

const (
	RenderHuman RenderFormat = "human"
	RenderJSON  RenderFormat = "json"
)

type inventoryV1 struct {
	SchemaVersion int             `json:"schema_version"`
	Forests       []forestV1      `json:"forests"`
	Correlations  []correlationV1 `json:"correlations"`
	Ambiguities   []ambiguityV1   `json:"ambiguities"`
	Diagnostics   []diagnosticV1  `json:"diagnostics"`
}

type forestV1 struct {
	Agent   Agent    `json:"agent"`
	Roots   []nodeV1 `json:"roots"`
	Orphans []nodeV1 `json:"orphans"`
}

type nodeV1 struct {
	NodeID         string      `json:"node_id"`
	NativeID       string      `json:"native_id"`
	Role           Role        `json:"role"`
	ParentNativeID *string     `json:"parent_native_id"`
	Resumable      bool        `json:"resumable"`
	CreatedAt      *string     `json:"created_at"`
	TimeSource     *TimeSource `json:"time_source"`
	Artifacts      []Artifact  `json:"artifacts"`
	Children       []nodeV1    `json:"children"`
}

type diagnosticV1 struct {
	DiagnosticID string         `json:"diagnostic_id"`
	Severity     Severity       `json:"severity"`
	Code         DiagnosticCode `json:"code"`
	Agent        *Agent         `json:"agent"`
	NativeID     *string        `json:"native_id"`
	StorageRoot  *string        `json:"storage_root"`
	RelativePath *string        `json:"relative_path"`
	SourceRef    *string        `json:"source_ref"`
}

type correlationV1 struct {
	BindingID  string        `json:"binding_id"`
	ScopeKey   string        `json:"scope_key"`
	Tag        string        `json:"tag"`
	Agent      Agent         `json:"agent"`
	RootNodeID *string       `json:"root_node_id"`
	Status     BindingStatus `json:"status"`
	Candidates []Candidate   `json:"candidates"`
	Evidence   []Evidence    `json:"evidence"`
}

type ambiguityV1 struct {
	AmbiguityID string   `json:"ambiguity_id"`
	Kind        string   `json:"kind"`
	Rank        int      `json:"rank"`
	BindingIDs  []string `json:"binding_ids"`
	RootNodeIDs []string `json:"root_node_ids"`
	EvidenceIDs []string `json:"evidence_ids"`
}

// RenderV1 renders the complete canonical public inventory. It first builds a
// private DTO so internal diagnostic detail and edge implementation fields can
// never leak into the stable schema.
func RenderV1(input Inventory, format RenderFormat) ([]byte, error) {
	canonical := SortInventory(input)
	view := projectV1(canonical)
	switch format {
	case RenderJSON:
		var output bytes.Buffer
		encoder := json.NewEncoder(&output)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(view); err != nil {
			return nil, err
		}
		return output.Bytes(), nil
	case RenderHuman:
		return renderHumanV1(view), nil
	default:
		return nil, fmt.Errorf("unsupported inventory render format %q", format)
	}
}

func projectV1(inventory Inventory) inventoryV1 {
	result := inventoryV1{
		SchemaVersion: 1,
		Forests:       make([]forestV1, 0, len(inventory.Forests)),
		Correlations:  make([]correlationV1, 0, len(inventory.Bindings)),
		Ambiguities:   make([]ambiguityV1, 0, len(inventory.Ambiguities)),
		Diagnostics:   make([]diagnosticV1, 0, len(inventory.Diagnostics)),
	}
	for _, binding := range inventory.Bindings {
		correlation := correlationV1{
			BindingID: binding.StableID, ScopeKey: binding.ScopeKey, Tag: binding.Tag, Agent: binding.Agent,
			RootNodeID: cloneString(binding.RootNodeID), Status: binding.Status,
			Candidates: make([]Candidate, len(binding.Candidates)), Evidence: make([]Evidence, len(binding.Evidence)),
		}
		copy(correlation.Candidates, binding.Candidates)
		copy(correlation.Evidence, binding.Evidence)
		result.Correlations = append(result.Correlations, correlation)
	}
	for _, ambiguity := range inventory.Ambiguities {
		entry := ambiguityV1{
			AmbiguityID: ambiguity.StableID, Kind: ambiguity.Kind, Rank: ambiguity.Rank,
			BindingIDs: make([]string, len(ambiguity.BindingIDs)), RootNodeIDs: make([]string, len(ambiguity.RootNodeIDs)), EvidenceIDs: make([]string, len(ambiguity.EvidenceIDs)),
		}
		copy(entry.BindingIDs, ambiguity.BindingIDs)
		copy(entry.RootNodeIDs, ambiguity.RootNodeIDs)
		copy(entry.EvidenceIDs, ambiguity.EvidenceIDs)
		result.Ambiguities = append(result.Ambiguities, entry)
	}
	for _, forest := range inventory.Forests {
		result.Forests = append(result.Forests, forestV1{
			Agent: forest.Agent, Roots: projectNodes(forest.Roots, false), Orphans: projectNodes(forest.Orphans, true),
		})
	}
	for _, diagnostic := range inventory.Diagnostics {
		entry := diagnosticV1{
			DiagnosticID: diagnostic.StableID, Severity: diagnostic.Severity, Code: diagnostic.Code,
			NativeID: cloneString(diagnostic.NativeID), SourceRef: cloneString(diagnostic.SourceRef),
		}
		if diagnostic.Agent != "" {
			agent := diagnostic.Agent
			entry.Agent = &agent
		}
		if diagnostic.Path != nil {
			storageRoot, relativePath := diagnostic.Path.StorageRoot, diagnostic.Path.RelativePath
			entry.StorageRoot, entry.RelativePath = &storageRoot, &relativePath
		}
		result.Diagnostics = append(result.Diagnostics, entry)
	}
	return result
}

func projectNodes(nodes []Node, orphan bool) []nodeV1 {
	result := make([]nodeV1, 0, len(nodes))
	for _, node := range nodes {
		role := node.Role
		if orphan {
			role = Role("orphan")
		}
		entry := nodeV1{
			NodeID: node.StableID, NativeID: node.NativeID, Role: role,
			ParentNativeID: cloneString(node.ParentID), Resumable: node.Resumable,
			Artifacts: append([]Artifact(nil), node.Artifacts...), Children: projectNodes(node.Children, false),
		}
		if entry.Artifacts == nil {
			entry.Artifacts = []Artifact{}
		}
		if node.Time != nil {
			created := node.Time.Value.UTC().Format(time.RFC3339Nano)
			source := node.Time.Source
			entry.CreatedAt, entry.TimeSource = &created, &source
		}
		result = append(result, entry)
	}
	return result
}

func renderHumanV1(inventory inventoryV1) []byte {
	var output strings.Builder
	fmt.Fprintf(&output, "session inventory schema=%d\n", inventory.SchemaVersion)
	for _, forest := range inventory.Forests {
		fmt.Fprintf(&output, "%s roots=%d orphans=%d\n", forest.Agent, len(forest.Roots), len(forest.Orphans))
		for _, root := range forest.Roots {
			renderHumanNode(&output, root, "  ")
		}
		for _, orphan := range forest.Orphans {
			renderHumanNode(&output, orphan, "  ")
		}
	}
	for _, binding := range inventory.Correlations {
		root := "-"
		if binding.RootNodeID != nil {
			root = *binding.RootNodeID
		}
		fmt.Fprintf(&output, "binding %s/%s %s status=%s root=%s id=%s\n", binding.ScopeKey, binding.Tag, binding.Agent, binding.Status, root, binding.BindingID)
	}
	for _, ambiguity := range inventory.Ambiguities {
		fmt.Fprintf(&output, "ambiguity %s rank=%d roots=%s id=%s\n", ambiguity.Kind, ambiguity.Rank, strings.Join(ambiguity.RootNodeIDs, ","), ambiguity.AmbiguityID)
	}
	for _, diagnostic := range inventory.Diagnostics {
		fmt.Fprintf(&output, "diagnostic %s %s id=%s\n", diagnostic.Severity, diagnostic.Code, diagnostic.DiagnosticID)
	}
	return []byte(output.String())
}

func renderHumanNode(output *strings.Builder, node nodeV1, indent string) {
	fmt.Fprintf(output, "%s%s %s node=%s resumable=%t\n", indent, node.Role, node.NativeID, node.NodeID, node.Resumable)
	for _, child := range node.Children {
		renderHumanNode(output, child, indent+"  ")
	}
}
