package sessioninventory

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"
)

// NativeEventsWithRuntime projects allowlisted causal events from each
// scanner-authorized root transcript. Descendant transcripts are deliberately
// excluded: parentage propagates an established root but cannot create root
// correlation evidence.
func NativeEventsWithRuntime(runtime Runtime, inventory Inventory, agent Agent) ([]NativeEventFact, []Diagnostic) {
	var facts []NativeEventFact
	var diagnostics []Diagnostic
	for _, forest := range inventory.Forests {
		if forest.Agent != agent {
			continue
		}
		for _, root := range forest.Roots {
			var transcripts []Artifact
			for _, artifact := range root.Artifacts {
				if artifact.Kind == ArtifactTranscript {
					transcripts = append(transcripts, artifact)
				}
			}
			if len(transcripts) != 1 {
				detail := fmt.Sprintf("root has %d transcript artifacts; exactly one is required for causal ordering", len(transcripts))
				diagnostics = append(diagnostics, diagnosticWithSource(DiagnosticTurnUnusable, agent, &root.NativeID, root.StableID, detail))
				continue
			}
			err := visitJSONLinesAt(runtime, transcripts[0], jsonRecordLimit, true, func(line []byte, lineStart uint64) bool {
				events, disposition := NormalizeNativeEvent(agent, line)
				if disposition == EventNearMiss {
					source := fmt.Sprintf("%s:%d", root.StableID, lineStart)
					diagnostics = append(diagnostics, diagnosticWithSource(DiagnosticTurnUnusable, agent, nil, source, "native record is not usable causal evidence"))
					return false
				}
				for _, event := range events {
					facts = append(facts, NativeEventFact{RootNodeID: root.StableID, Position: lineStart, Event: event})
				}
				return false
			})
			if err != nil {
				diagnostics = append(diagnostics, diagnosticWithSource(DiagnosticTurnUnusable, agent, &root.NativeID, root.StableID, "root transcript is unreadable for causal matching"))
			}
		}
	}
	slices.SortFunc(facts, func(a, b NativeEventFact) int {
		if result := cmp.Compare(a.RootNodeID, b.RootNodeID); result != 0 {
			return result
		}
		if result := cmp.Compare(a.Position, b.Position); result != 0 {
			return result
		}
		return cmp.Compare(a.Event.Kind, b.Event.Kind)
	})
	return facts, diagnostics
}

func nativeEventsFromJSONL(agent Agent, rootNodeID string, raw []byte, diagnostics *[]Diagnostic) []NativeEventFact {
	var facts []NativeEventFact
	offset := uint64(0)
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		lineStart := offset
		offset += uint64(len(line)) + 1
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		events, disposition := NormalizeNativeEvent(agent, line)
		if disposition == EventNearMiss {
			source := fmt.Sprintf("%s:%d", rootNodeID, lineStart)
			*diagnostics = append(*diagnostics, diagnosticWithSource(DiagnosticTurnUnusable, agent, nil, source, "native record is not an allowlisted causal event"))
			continue
		}
		if disposition != EventAccepted {
			continue
		}
		for index, event := range events {
			facts = append(facts, NativeEventFact{
				Agent: agent, RootNodeID: rootNodeID,
				Position: ((lineStart + 1) << 8) | uint64(index), Event: event,
			})
		}
	}
	return facts
}
