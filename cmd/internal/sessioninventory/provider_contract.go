package sessioninventory

// ProviderContract names one reviewed append-only producer/storage contract.
// Unknown values never authorize suffix reuse.
// pair:156-concept pure new final
type ProviderContract string

const (
	ProviderClaudeJSONLV1        ProviderContract = "claude-jsonl-v1"
	ProviderCodexJSONLV1         ProviderContract = "codex-jsonl-v1"
	ProviderMuseJSONLV1          ProviderContract = "muse-jsonl-v1"
	ProviderAgyTranscriptJSONLV1 ProviderContract = "agy-transcript-jsonl-v1"
)

func ProviderContractFor(agent Agent, storageRoot, scannerSchema string) (ProviderContract, bool) {
	var contract ProviderContract
	switch {
	case agent == AgentClaude && storageRoot == "claude-projects" && scannerSchema == "claude-v1":
		contract = ProviderClaudeJSONLV1
	case agent == AgentCodex && storageRoot == "codex-sessions" && scannerSchema == "codex-v1":
		contract = ProviderCodexJSONLV1
	case agent == AgentMuse && storageRoot == "muse-sessions" && scannerSchema == "muse-v1":
		contract = ProviderMuseJSONLV1
	case agent == AgentAgy && storageRoot == "agy-brain" && scannerSchema == "agy-transcript-v1":
		contract = ProviderAgyTranscriptJSONLV1
	}
	return contract, contract != ""
}
