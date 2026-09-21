package launcher

import (
	"bytes"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"github.com/xianxu/pair/cmd/internal/sessionwatch"
)

// sessionInventoryKnownGaps names inventory agents whose session-side wiring
// (scanner, event adapter, provider contract, watcher/ledger membership) is
// still missing. The parity test asserts each gap is still exactly this shape,
// so the milestone that closes it must delete the entry.
var sessionInventoryKnownGaps = map[string]string{
	"qoder": "pair#300 M2: no sessioninventory scanner/event adapter/provider contract; not watchable and ledger-rejected (all four probed below)",
}

func TestAgentInventoryParityWithSessionTables(t *testing.T) {
	for _, agent := range AgentInventory() {
		t.Run(agent, func(t *testing.T) {
			accepted := sessionInventoryAcceptsAgent(agent)
			scansReal := !scannerIsUnsupportedFallback(sessioninventory.Agent(agent))
			watchable := sessionwatch.SupportsAgent(agent)
			ledgerRejects := ledgerRejectsAgent(agent)
			if gap, knownGap := sessionInventoryKnownGaps[agent]; knownGap {
				if !accepted || scansReal || watchable || !ledgerRejects {
					t.Fatalf("known gap is closed or changed shape; delete its entry (%s)", gap)
				}
				return
			}
			if !accepted || !scansReal || !watchable || ledgerRejects {
				t.Fatalf("accepted=%v scanner=%v watchable=%v ledgerRejects=%v — wire the session-inventory tables for %q", accepted, scansReal, watchable, ledgerRejects, agent)
			}
		})
	}
}

// scannerIsUnsupportedFallback reports whether ScannerForAgent resolved to the
// near-miss placeholder ("unsupported agent") rather than a real scanner.
func scannerIsUnsupportedFallback(agent sessioninventory.Agent) bool {
	result := sessioninventory.ScannerForAgent(agent)(sessioninventorytest.NewFakeRuntime())
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == sessioninventory.DiagnosticSchemaNearMiss && diagnostic.Agent == agent && diagnostic.Detail == "unsupported agent" {
			return true
		}
	}
	return false
}

func sessionInventoryAcceptsAgent(agent string) bool {
	var stdout, stderr bytes.Buffer
	return sessioninventory.RunCLIWithRuntime([]string{"--agent", agent, "--json"}, func(string) string { return "" }, sessioninventorytest.NewFakeRuntime(), &stdout, &stderr) == 0
}

// ledgerRejectsAgent reports whether sessionledger's typed decoder refuses a
// launch record for the agent (the isSupportedAgent dispatch in record.go).
// A gap agent's records must fail closed as malformed, not parse as
// compatibility rows.
func ledgerRejectsAgent(agent string) bool {
	line := []byte(`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"` + agent + `","pair_log_offset":0,"native_watermarks":[]}` + "\n")
	result := sessionledger.ParseLedger(line)
	return len(result.Records) == 0 && len(result.MalformedOrdinals) == 1
}
