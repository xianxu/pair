package sessioninventory

import (
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
			root.Agent = agent
			found, err := visitNativeEventsForRoot(runtime, root, func(fact NativeEventFact) {
				facts = append(facts, fact)
			})
			diagnostics = append(diagnostics, found...)
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

func visitNativeEventsForRoot(runtime Runtime, root Node, visit func(NativeEventFact)) ([]Diagnostic, error) {
	artifact, err := RootTranscript(root)
	if err != nil {
		return nil, err
	}
	var diagnostics []Diagnostic
	err = visitJSONLinesAt(runtime, artifact, jsonRecordLimit, func(line []byte, lineStart uint64) bool {
		events, disposition := NormalizeNativeEvent(root.Agent, line)
		if disposition == EventNearMiss {
			source := fmt.Sprintf("%s:%d", root.StableID, lineStart)
			diagnostics = append(diagnostics, diagnosticWithSource(DiagnosticTurnUnusable, root.Agent, nil, source, "native record is not usable causal evidence"))
			return false
		}
		for index, event := range events {
			visit(NativeEventFact{
				Agent: root.Agent, RootNodeID: root.StableID,
				Position: ((lineStart + 1) << 8) | uint64(index), Event: event,
			})
		}
		return false
	})
	return diagnostics, err
}

// TextEventWindowForRoot streams one scanner-authorized root transcript and
// retains only a bounded recent text window plus the nearest older user anchor.
// pair:155-concept integration new final
func TextEventWindowForRoot(runtime Runtime, root Node, maxRecent int) ([]NativeEvent, error) {
	if maxRecent <= 0 {
		return nil, nil
	}
	var recent []NativeEvent
	var olderUser *NativeEvent
	_, err := visitNativeEventsForRoot(runtime, root, func(fact NativeEventFact) {
		if (fact.Event.Kind != EventOperator && fact.Event.Kind != EventAssistant) || fact.Event.Text == "" {
			return
		}
		recent = append(recent, fact.Event)
		if len(recent) > maxRecent {
			dropped := recent[0]
			recent = recent[1:]
			if dropped.Kind == EventOperator {
				copy := dropped
				olderUser = &copy
			}
		}
	})
	if err != nil {
		return nil, err
	}
	hasUser := false
	for _, event := range recent {
		if event.Kind == EventOperator {
			hasUser = true
			break
		}
	}
	if !hasUser && olderUser != nil {
		recent = append([]NativeEvent{*olderUser}, recent...)
	}
	return recent, nil
}
