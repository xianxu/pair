package sessionwatch

import (
	"bytes"
	"encoding/json"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

type ConfigPayload struct {
	Agent     string   `json:"agent"`
	Args      []string `json:"args"`
	SessionID string   `json:"session_id"`
}

// WatcherInventory is one launch generation's pure inventory/round input to
// the persistence boundary.
// pair:155-concept integration modified M2
type WatcherInventory struct {
	Owner         sessionledger.Owner
	LedgerPath    string
	LaunchOrdinal uint64
	Inventory     sessioninventory.Inventory
	LiveRounds    []sessioninventory.RoundObservation
	Args          []string
}

type ObserveInput = WatcherInventory

func SupportsAgent(agent string) bool {
	switch agent {
	case "claude", "codex", "agy", "muse":
		return true
	default:
		return false
	}
}

// StripResumeArgs removes resume bindings from args before they are persisted;
// the session_id field is the canonical store for that binding.
func StripResumeArgs(agent string, args []string) []string {
	stripped := make([]string, 0, len(args))
	i := 0
	if (agent == "codex" || agent == "muse") && len(args) >= 2 && args[0] == "resume" {
		i = 2
	}
	for i < len(args) {
		if args[i] == "--resume" {
			i += 2
			continue
		}
		stripped = append(stripped, args[i])
		i++
	}
	return stripped
}

func ConfigJSON(payload ConfigPayload) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
