package sessioninventory

import (
	"fmt"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

type OfflineRecoveryInput struct {
	ScopeKey         string
	Tag              string
	Agent            Agent
	Log              []byte
	Current          sessionledger.Current
	NativeEvents     []NativeEventFact
	ConfigRootNodeID string
}

// OfflineRecovery reconstructs only the Pair/native suffix delimited by the
// latest durable launch generation. It never consults wall-clock chronology.
func OfflineRecovery(inventory Inventory, input OfflineRecoveryInput) Inventory {
	launch := input.Current.Launch
	launchPresent := launch.Ordinal != 0
	if launchPresent && (launch.ScopeKey != input.ScopeKey || launch.Tag != input.Tag || launch.Agent != string(input.Agent)) {
		inventory.Diagnostics = append(inventory.Diagnostics, diagnosticWithSource(DiagnosticScopeRejected, input.Agent, nil, "ledger launch owner", "latest launch belongs to another Pair owner"))
		return ResolveBindings(inventory, []BindingInput{{ScopeKey: input.ScopeKey, Tag: input.Tag, Agent: input.Agent}})
	}
	rootByNative, nativeByRoot := rootIdentityMaps(inventory.Forests, input.Agent)
	ledgerRoots := make([]string, 0, len(input.Current.Bindings))
	if input.Current.Conflict {
		inventory.Diagnostics = append(inventory.Diagnostics, diagnosticWithSource(DiagnosticBindingConflict, input.Agent, nil, "ledger binding", "latest launch has conflicting binding records"))
	}
	for _, binding := range input.Current.Bindings {
		if root := rootByNative[binding.RootNativeID]; root != "" {
			ledgerRoots = append(ledgerRoots, root)
		}
	}
	if len(ledgerRoots) == 0 && input.Current.Binding != nil {
		if root := rootByNative[input.Current.Binding.RootNativeID]; root != "" {
			ledgerRoots = append(ledgerRoots, root)
		}
	}
	if !launchPresent {
		return ResolveBindings(inventory, []BindingInput{{
			ScopeKey: input.ScopeKey, Tag: input.Tag, Agent: input.Agent,
			ConfigRootNodeID: input.ConfigRootNodeID,
		}})
	}

	parsed := ParsePairLog(input.Log, launch.PairLogOffset)
	for i := range parsed.Facts {
		parsed.Facts[i].ScopeKey, parsed.Facts[i].Tag, parsed.Facts[i].Agent = input.ScopeKey, input.Tag, input.Agent
	}
	if len(parsed.MalformedOffsets) != 0 {
		for _, offset := range parsed.MalformedOffsets {
			inventory.Diagnostics = append(inventory.Diagnostics, diagnosticWithSource(DiagnosticPairRecordMalformed, input.Agent, nil, fmt.Sprintf("pair-log:%d", offset), "Pair log suffix is malformed"))
		}
		parsed.Facts = nil
	}
	watermarks := map[string]uint64{}
	for _, watermark := range launch.NativeWatermarks {
		watermarks[watermark.RootNativeID] = watermark.EventPosition
	}
	filteredEvents := make([]NativeEventFact, 0, len(input.NativeEvents))
	for _, event := range input.NativeEvents {
		event.Agent = input.Agent
		nativeID := nativeByRoot[event.RootNodeID]
		if nativeID == "" {
			continue
		}
		if watermark, existed := watermarks[nativeID]; existed && event.Position <= watermark {
			continue
		}
		filteredEvents = append(filteredEvents, event)
	}
	offlineRounds := QualifyTurnSequence(parsed.Facts, filteredEvents)
	return ResolveBindings(inventory, []BindingInput{{
		ScopeKey: input.ScopeKey, Tag: input.Tag, Agent: input.Agent, LaunchPresent: launchPresent,
		LedgerRootNodeIDs: ledgerRoots, OfflineRounds: offlineRounds, ConfigRootNodeID: input.ConfigRootNodeID,
	}})
}

func rootIdentityMaps(forests []Forest, agent Agent) (map[string]string, map[string]string) {
	byNative, byNode := map[string]string{}, map[string]string{}
	for _, forest := range forests {
		if forest.Agent != agent {
			continue
		}
		for _, root := range forest.Roots {
			byNative[root.NativeID] = root.StableID
			byNode[root.StableID] = root.NativeID
		}
	}
	return byNative, byNode
}
