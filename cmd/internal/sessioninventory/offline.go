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
	rootByNative, _ := rootIdentityMaps(inventory.Forests, input.Agent)
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

	offlineRounds, diagnostics := RoundsAfterLaunch(inventory, input.ScopeKey, input.Tag, input.Agent, input.Log, launch, input.NativeEvents)
	inventory.Diagnostics = append(inventory.Diagnostics, diagnostics...)
	return ResolveBindings(inventory, []BindingInput{{
		ScopeKey: input.ScopeKey, Tag: input.Tag, Agent: input.Agent, LaunchPresent: launchPresent,
		LedgerRootNodeIDs: ledgerRoots, OfflineRounds: offlineRounds, ConfigRootNodeID: input.ConfigRootNodeID,
	}})
}

// RoundsAfterLaunch returns only exact completed rounds in the durable suffix
// delimited by one launch baseline. Live monitoring and offline recovery share
// this projection; only their evidence label differs at binding time.
func RoundsAfterLaunch(inventory Inventory, scopeKey, tag string, agent Agent, log []byte, launch sessionledger.Record, nativeEvents []NativeEventFact) ([]RoundObservation, []Diagnostic) {
	if launch.Ordinal == 0 {
		return nil, nil
	}
	parsed := ParsePairLog(log, launch.PairLogOffset)
	for i := range parsed.Facts {
		parsed.Facts[i].ScopeKey, parsed.Facts[i].Tag, parsed.Facts[i].Agent = scopeKey, tag, agent
	}
	var diagnostics []Diagnostic
	if len(parsed.MalformedOffsets) != 0 {
		for _, offset := range parsed.MalformedOffsets {
			diagnostics = append(diagnostics, diagnosticWithSource(DiagnosticPairRecordMalformed, agent, nil, fmt.Sprintf("pair-log:%d", offset), "Pair log suffix is malformed"))
		}
		return nil, diagnostics
	}
	_, nativeByRoot := rootIdentityMaps(inventory.Forests, agent)
	watermarks := map[string]uint64{}
	for _, watermark := range launch.NativeWatermarks {
		watermarks[watermark.RootNativeID] = watermark.EventPosition
	}
	filteredEvents := make([]NativeEventFact, 0, len(nativeEvents))
	for _, event := range nativeEvents {
		event.Agent = agent
		nativeID := nativeByRoot[event.RootNodeID]
		if nativeID == "" {
			continue
		}
		if watermark, existed := watermarks[nativeID]; existed && event.Position <= watermark {
			continue
		}
		filteredEvents = append(filteredEvents, event)
	}
	return QualifyTurnSequence(parsed.Facts, filteredEvents), diagnostics
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
