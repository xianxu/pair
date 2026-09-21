package sessionwatch

import (
	"bytes"
	"encoding/json"

	"github.com/xianxu/pair/cmd/internal/resumeform"
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
	Proofs        map[string]sessionledger.AuthorizationProof
	RequireProof  bool
	Args          []string
}

type ObserveInput = WatcherInventory

func SupportsAgent(agent string) bool {
	switch agent {
	case "claude", "codex", "agy", "muse", "qoder":
		return true
	default:
		return false
	}
}

// StripResumeArgs removes every resume spelling the shared resume-form table
// defines (resumeform.Forms — the same table the launcher's extractor and
// validator read) plus the codex/muse leading `resume <id>` subcommand from
// args before they are persisted; the session_id field is the canonical store
// for that binding.
func StripResumeArgs(agent string, args []string) []string {
	if (agent == "codex" || agent == "muse") && len(args) >= 2 && args[0] == "resume" {
		args = args[2:]
	}
	return resumeform.Strip(args)
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
