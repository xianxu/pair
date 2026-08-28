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
	output := Inventory{
		Forests:     make([]Forest, len(input.Forests)),
		Diagnostics: make([]Diagnostic, len(input.Diagnostics)),
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

	for i, diagnostic := range input.Diagnostics {
		output.Diagnostics[i] = diagnostic
		output.Diagnostics[i].NativeID = cloneString(diagnostic.NativeID)
		if diagnostic.Path != nil {
			pathCopy := *diagnostic.Path
			output.Diagnostics[i].Path = &pathCopy
		}
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
		{string(left.Code), string(right.Code)},
		{string(left.Severity), string(right.Severity)},
		{string(left.Agent), string(right.Agent)},
		{leftNative, rightNative},
		{leftPath, rightPath},
		{left.StableID, right.StableID},
	}
	for _, part := range parts {
		if compared := strings.Compare(part[0], part[1]); compared != 0 {
			return compared
		}
	}
	return 0
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
