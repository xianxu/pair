package sessioninventory

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

const pairArtifactReadLimit = int64(64 << 20)

type pairOwner struct {
	scope string
	tag   string
	agent Agent
}

type pairConfig struct {
	Agent     string `json:"agent"`
	SessionID string `json:"session_id"`
}

// RecoverPairBindings projects typed current generations and the narrow
// launch-delimited offline round window from scanner-authorized Pair artifacts.
func RecoverPairBindings(runtime Runtime, inventory Inventory, scopeMode, currentScopeKey string, agents []Agent) (Inventory, error) {
	pairRoot := runtime.PairDataRoot()
	files, err := runtime.ListFiles(pairRoot)
	if errors.Is(err, ErrStorageAbsent) {
		return inventory, nil
	}
	if err != nil {
		return Inventory{}, errors.New("Pair artifact inventory unavailable")
	}
	allowed := map[Agent]bool{}
	for _, agent := range agents {
		allowed[agent] = true
	}
	records := map[pairOwner][]sessionledger.Record{}
	logs := map[pairOwner][]byte{}
	configs := map[pairOwner]string{}
	var diagnostics []Diagnostic

	for _, file := range files {
		scope, name, ok := selectedPairArtifact(file.Artifact.RelativePath, scopeMode, currentScopeKey)
		if !ok {
			continue
		}
		historyTag, historyArtifact := artifactpath.TagFromHistorySidecar(name)
		tag, configAgent, configArtifact := artifactpath.TagAgentFromConfigSidecar(name, agentNames(agents))
		switch {
		case historyArtifact && artifactpath.IsLedgerHistorySidecar(name):
			tag := historyTag
			raw, readErr := runtime.ReadFile(file.Artifact, pairArtifactReadLimit)
			if readErr != nil {
				diagnostics = append(diagnostics, diagnosticWithSource(DiagnosticStorageUnreadable, "", nil, "ledger:"+tag, "Pair ledger is unreadable"))
				continue
			}
			parsed := sessionledger.ParseLedger(raw)
			for _, ordinal := range parsed.MalformedOrdinals {
				diagnostics = append(diagnostics, diagnosticWithSource(DiagnosticPairRecordMalformed, "", nil, fmt.Sprintf("ledger:%s:%d", tag, ordinal), "typed Pair ledger row is malformed"))
			}
			for _, record := range parsed.Records {
				agent := Agent(record.Agent)
				if !allowed[agent] || record.Tag != tag || (scope != "" && record.ScopeKey != scope) {
					continue
				}
				owner := pairOwner{scope: record.ScopeKey, tag: record.Tag, agent: agent}
				records[owner] = append(records[owner], record)
			}
		case historyArtifact && artifactpath.IsLogHistorySidecar(name):
			tag := historyTag
			raw, readErr := runtime.ReadFile(file.Artifact, pairArtifactReadLimit)
			if readErr != nil {
				diagnostics = append(diagnostics, diagnosticWithSource(DiagnosticStorageUnreadable, "", nil, "log:"+tag, "Pair log is unreadable"))
				continue
			}
			for _, agent := range agents {
				logs[pairOwner{scope: scope, tag: tag, agent: agent}] = raw
			}
		case configArtifact:
			agent := Agent(configAgent)
			if !allowed[agent] {
				continue
			}
			raw, readErr := runtime.ReadFile(file.Artifact, pairArtifactReadLimit)
			if readErr != nil {
				diagnostics = append(diagnostics, diagnosticWithSource(DiagnosticStorageUnreadable, agent, nil, "config:"+tag, "Pair config is unreadable"))
				continue
			}
			var config pairConfig
			if json.Unmarshal(raw, &config) != nil || config.Agent != string(agent) || config.SessionID == "" {
				diagnostics = append(diagnostics, diagnosticWithSource(DiagnosticPairRecordMalformed, agent, nil, "config:"+tag, "Pair config is malformed"))
				continue
			}
			configs[pairOwner{scope: scope, tag: tag, agent: agent}] = config.SessionID
		}
	}

	owners := map[pairOwner]bool{}
	for owner := range records {
		owners[owner] = true
	}
	for owner := range configs {
		owners[owner] = true
	}
	orderedOwners := make([]pairOwner, 0, len(owners))
	for owner := range owners {
		orderedOwners = append(orderedOwners, owner)
	}
	sort.Slice(orderedOwners, func(i, j int) bool {
		left, right := orderedOwners[i], orderedOwners[j]
		if left.scope != right.scope {
			return left.scope < right.scope
		}
		if left.tag != right.tag {
			return left.tag < right.tag
		}
		return left.agent < right.agent
	})

	eventsByAgent := map[Agent][]NativeEventFact{}
	var inputs []BindingInput
	for _, owner := range orderedOwners {
		rootByNative, _ := rootIdentityMaps(inventory.Forests, owner.agent)
		input := BindingInput{ScopeKey: owner.scope, Tag: owner.tag, Agent: owner.agent}
		if nativeID := configs[owner]; nativeID != "" {
			input.ConfigRootNodeID = rootByNative[nativeID]
		}
		current, present := sessionledger.CurrentLaunch(records[owner], sessionledger.Owner{ScopeKey: owner.scope, Tag: owner.tag, Agent: string(owner.agent)})
		input.LaunchPresent = present
		if present {
			if current.Conflict {
				diagnostics = append(diagnostics, diagnosticWithSource(DiagnosticBindingConflict, owner.agent, nil, "ledger binding", "latest launch has conflicting binding records"))
			}
			for _, binding := range current.Bindings {
				if root := rootByNative[binding.RootNativeID]; root != "" {
					input.LedgerRootNodeIDs = append(input.LedgerRootNodeIDs, root)
				}
			}
			if len(input.LedgerRootNodeIDs) == 0 {
				if _, loaded := eventsByAgent[owner.agent]; !loaded {
					events, found := NativeEventsWithRuntime(runtime, inventory, owner.agent)
					eventsByAgent[owner.agent] = events
					diagnostics = append(diagnostics, found...)
				}
				rounds, found := RoundsAfterLaunch(inventory, owner.scope, owner.tag, owner.agent, logs[owner], current.Launch, eventsByAgent[owner.agent])
				input.OfflineRounds = rounds
				diagnostics = append(diagnostics, found...)
			}
		}
		inputs = append(inputs, input)
	}
	inventory.Diagnostics = append(inventory.Diagnostics, diagnostics...)
	return ResolveBindings(inventory, inputs), nil
}

func selectedPairArtifact(relativePath, scopeMode, currentScopeKey string) (string, string, bool) {
	clean := path.Clean(relativePath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", "", false
	}
	parts := strings.Split(clean, "/")
	if len(parts) == 3 && parts[0] == "repos" {
		if scopeMode == "current" && parts[1] != currentScopeKey {
			return "", "", false
		}
		return parts[1], parts[2], true
	}
	if len(parts) == 1 && currentScopeKey != "" {
		return currentScopeKey, parts[0], true
	}
	return "", "", false
}

func agentNames(agents []Agent) []string {
	result := make([]string, len(agents))
	for i, agent := range agents {
		result[i] = string(agent)
	}
	return result
}
