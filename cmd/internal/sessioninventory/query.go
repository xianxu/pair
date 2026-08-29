package sessioninventory

import (
	"errors"
	"fmt"
)

var ErrRootTranscript = errors.New("established root must have exactly one transcript artifact")

// SessionQuery preserves the owner's binding state. Root is populated only for
// an established binding whose scanner-authorized root is present.
type SessionQuery struct {
	Status      BindingStatus
	Root        *Node
	Diagnostics []Diagnostic
}

// SessionForOwner is the pure owner lookup shared by native-session consumers.
// Ambiguous, provisional, and unbound owners never receive a root fallback.
func SessionForOwner(inventory Inventory, scopeKey, tag string, agent Agent) SessionQuery {
	query := SessionQuery{Status: BindingUnbound, Diagnostics: append([]Diagnostic(nil), inventory.Diagnostics...)}
	for _, binding := range inventory.Bindings {
		if binding.ScopeKey != scopeKey || binding.Tag != tag || binding.Agent != agent {
			continue
		}
		query.Status = binding.Status
		if binding.Status != BindingEstablished || binding.RootNodeID == nil {
			return query
		}
		for _, forest := range inventory.Forests {
			if forest.Agent != agent {
				continue
			}
			for _, root := range forest.Roots {
				if root.StableID == *binding.RootNodeID {
					value := cloneNode(root)
					query.Root = &value
					return query
				}
			}
		}
		return query
	}
	return query
}

// QuerySession scans one agent and projects the current Pair owner through the
// same binding recovery used by the public inventory command.
func QuerySession(runtime Runtime, scopeKey, tag string, agent Agent) (SessionQuery, error) {
	inventory := InventoryWithRuntime(runtime, ScannerForAgent(agent))
	resolved, err := RecoverPairBindings(runtime, inventory, "current", scopeKey, []Agent{agent})
	if err != nil {
		return SessionQuery{}, err
	}
	return SessionForOwner(resolved, scopeKey, tag, agent), nil
}

// RootTranscript returns the one scanner-authorized transcript for a root.
func RootTranscript(root Node) (Artifact, error) {
	var transcript Artifact
	count := 0
	for _, artifact := range root.Artifacts {
		if artifact.Kind == ArtifactTranscript {
			transcript = artifact
			count++
		}
	}
	if count != 1 {
		return Artifact{}, fmt.Errorf("%w: got %d", ErrRootTranscript, count)
	}
	return transcript, nil
}

// ReadRootTranscript reads only the scanner-authorized root transcript.
func ReadRootTranscript(runtime Runtime, root Node) ([]byte, error) {
	artifact, err := RootTranscript(root)
	if err != nil {
		return nil, err
	}
	return readJSONLines(runtime, artifact, jsonRecordLimit)
}
