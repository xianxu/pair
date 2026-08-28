package sessioninventory

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sort"
	"strings"
)

// StableID hashes a kind and unambiguous length-prefixed tuple fields. It is
// independent of JSON encoding and therefore remains stable as rendering grows.
func StableID(kind string, parts ...string) string {
	hash := sha256.New()
	var length [8]byte
	for _, part := range append([]string{kind}, parts...) {
		binary.BigEndian.PutUint64(length[:], uint64(len(part)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(part))
	}
	return kind + "-" + hex.EncodeToString(hash.Sum(nil))[:24]
}

// SortInventory returns a deeply cloned, canonically ordered inventory.
func SortInventory(input Inventory) Inventory {
	coalescedDiagnostics := make(map[string]Diagnostic, len(input.Diagnostics))
	for _, diagnostic := range input.Diagnostics {
		if severity, ok := diagnosticSeverity(diagnostic.Code); ok {
			diagnostic.Severity = severity
		}
		diagnostic.StableID = diagnosticID(diagnostic)
		previous, exists := coalescedDiagnostics[diagnostic.StableID]
		if !exists || diagnostic.Detail < previous.Detail {
			coalescedDiagnostics[diagnostic.StableID] = diagnostic
		}
	}
	output := Inventory{
		Forests:     make([]Forest, len(input.Forests)),
		Bindings:    make([]Binding, len(input.Bindings)),
		Ambiguities: make([]Ambiguity, len(input.Ambiguities)),
		Diagnostics: make([]Diagnostic, 0, len(coalescedDiagnostics)),
	}
	for i, forest := range input.Forests {
		output.Forests[i] = Forest{
			Agent:   forest.Agent,
			Roots:   cloneNodes(forest.Roots),
			Orphans: cloneNodes(forest.Orphans),
		}
		sortNodes(output.Forests[i].Roots)
		sortNodes(output.Forests[i].Orphans)
	}
	sort.Slice(output.Forests, func(i, j int) bool { return output.Forests[i].Agent < output.Forests[j].Agent })
	for i, binding := range input.Bindings {
		output.Bindings[i] = binding
		output.Bindings[i].RootNodeID = cloneString(binding.RootNodeID)
		output.Bindings[i].Candidates = append([]Candidate(nil), binding.Candidates...)
		for candidate := range output.Bindings[i].Candidates {
			output.Bindings[i].Candidates[candidate].EvidenceIDs = append([]string(nil), binding.Candidates[candidate].EvidenceIDs...)
			sort.Strings(output.Bindings[i].Candidates[candidate].EvidenceIDs)
		}
		output.Bindings[i].Evidence = append([]Evidence(nil), binding.Evidence...)
		for evidence := range output.Bindings[i].Evidence {
			output.Bindings[i].Evidence[evidence].SourcePositions = append([]uint64(nil), binding.Evidence[evidence].SourcePositions...)
			output.Bindings[i].Evidence[evidence].DestinationPositions = append([]uint64(nil), binding.Evidence[evidence].DestinationPositions...)
			output.Bindings[i].Evidence[evidence].Fingerprints = append([]string(nil), binding.Evidence[evidence].Fingerprints...)
			sort.Slice(output.Bindings[i].Evidence[evidence].SourcePositions, func(a, b int) bool {
				return output.Bindings[i].Evidence[evidence].SourcePositions[a] < output.Bindings[i].Evidence[evidence].SourcePositions[b]
			})
			sort.Slice(output.Bindings[i].Evidence[evidence].DestinationPositions, func(a, b int) bool {
				return output.Bindings[i].Evidence[evidence].DestinationPositions[a] < output.Bindings[i].Evidence[evidence].DestinationPositions[b]
			})
			sort.Strings(output.Bindings[i].Evidence[evidence].Fingerprints)
		}
		sort.Slice(output.Bindings[i].Candidates, func(a, b int) bool {
			return compareCandidate(output.Bindings[i].Candidates[a], output.Bindings[i].Candidates[b]) < 0
		})
		sort.Slice(output.Bindings[i].Evidence, func(a, b int) bool {
			return compareEvidence(output.Bindings[i].Evidence[a], output.Bindings[i].Evidence[b]) < 0
		})
	}
	sort.Slice(output.Bindings, func(i, j int) bool { return compareBinding(output.Bindings[i], output.Bindings[j]) < 0 })
	for i, ambiguity := range input.Ambiguities {
		output.Ambiguities[i] = ambiguity
		output.Ambiguities[i].BindingIDs = append([]string(nil), ambiguity.BindingIDs...)
		output.Ambiguities[i].RootNodeIDs = append([]string(nil), ambiguity.RootNodeIDs...)
		output.Ambiguities[i].EvidenceIDs = append([]string(nil), ambiguity.EvidenceIDs...)
		sort.Strings(output.Ambiguities[i].BindingIDs)
		sort.Strings(output.Ambiguities[i].RootNodeIDs)
		sort.Strings(output.Ambiguities[i].EvidenceIDs)
	}
	sort.Slice(output.Ambiguities, func(i, j int) bool { return compareAmbiguity(output.Ambiguities[i], output.Ambiguities[j]) < 0 })

	for _, diagnostic := range coalescedDiagnostics {
		cloned := diagnostic
		cloned.NativeID = cloneString(diagnostic.NativeID)
		cloned.SourceRef = cloneString(diagnostic.SourceRef)
		if diagnostic.Path != nil {
			pathCopy := *diagnostic.Path
			cloned.Path = &pathCopy
		}
		output.Diagnostics = append(output.Diagnostics, cloned)
	}
	sort.Slice(output.Diagnostics, func(i, j int) bool {
		return compareDiagnostic(output.Diagnostics[i], output.Diagnostics[j]) < 0
	})
	return output
}

func cloneNodes(input []Node) []Node {
	result := make([]Node, len(input))
	for i, node := range input {
		result[i] = cloneNode(node)
		result[i].Children = cloneNodes(node.Children)
	}
	return result
}

func sortNodes(nodes []Node) {
	for i := range nodes {
		sort.Slice(nodes[i].Artifacts, func(a, b int) bool {
			return compareArtifact(nodes[i].Artifacts[a], nodes[i].Artifacts[b]) < 0
		})
		if nodes[i].ParentEdge != nil {
			sort.Slice(nodes[i].ParentEdge.Provenance, func(a, b int) bool {
				left, right := nodes[i].ParentEdge.Provenance[a], nodes[i].ParentEdge.Provenance[b]
				if left.Schema != right.Schema {
					return left.Schema < right.Schema
				}
				return compareArtifact(left.Artifact, right.Artifact) < 0
			})
		}
		sortNodes(nodes[i].Children)
	}
	sort.Slice(nodes, func(i, j int) bool { return compareNode(nodes[i], nodes[j]) < 0 })
}

func compareNode(left, right Node) int {
	if compared := compareNativeTime(left.Time, right.Time); compared != 0 {
		return compared
	}
	if compared := strings.Compare(left.NativeID, right.NativeID); compared != 0 {
		return compared
	}
	leftArtifact, rightArtifact := firstArtifact(left.Artifacts), firstArtifact(right.Artifacts)
	if compared := compareArtifact(leftArtifact, rightArtifact); compared != 0 {
		return compared
	}
	return strings.Compare(left.StableID, right.StableID)
}

func firstArtifact(artifacts []Artifact) Artifact {
	if len(artifacts) == 0 {
		return Artifact{StorageRoot: "\uffff", RelativePath: "\uffff"}
	}
	return artifacts[0]
}

func compareArtifact(left, right Artifact) int {
	if compared := strings.Compare(left.StorageRoot, right.StorageRoot); compared != 0 {
		return compared
	}
	if compared := strings.Compare(left.RelativePath, right.RelativePath); compared != 0 {
		return compared
	}
	return strings.Compare(string(left.Kind), string(right.Kind))
}

func compareDiagnostic(left, right Diagnostic) int {
	leftNative, rightNative := nullableString(left.NativeID), nullableString(right.NativeID)
	leftPath, rightPath := nullableArtifact(left.Path), nullableArtifact(right.Path)
	parts := [][2]string{
		{severityOrder(left.Severity), severityOrder(right.Severity)},
		{string(left.Code), string(right.Code)},
		{nullableAgent(left.Agent), nullableAgent(right.Agent)},
		{leftNative, rightNative},
		{leftPath, rightPath},
		{nullableString(left.SourceRef), nullableString(right.SourceRef)},
		{left.StableID, right.StableID},
	}
	for _, part := range parts {
		if compared := strings.Compare(part[0], part[1]); compared != 0 {
			return compared
		}
	}
	return 0
}

func nullableAgent(value Agent) string {
	if value == "" {
		return "\uffff"
	}
	return string(value)
}

func severityOrder(severity Severity) string {
	switch severity {
	case SeverityError:
		return "0"
	case SeverityWarning:
		return "1"
	case SeverityInfo:
		return "2"
	default:
		return "3"
	}
}

func nullableString(value *string) string {
	if value == nil {
		return "\uffff"
	}
	return *value
}

func nullableArtifact(value *Artifact) string {
	if value == nil {
		return "\uffff"
	}
	return value.StorageRoot + "\x00" + value.RelativePath + "\x00" + string(value.Kind)
}
