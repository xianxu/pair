package wrapcmd

import (
	"bytes"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// This is assembled synthetic limit text, not the operator's original screen.
// Its provenance is recorded with the shared fixture. It proves conservative
// no-input admission; the operator's M3 live check qualifies real force behavior.
func TestSyntheticClaudeUsageLimitDoesNotAuthorizeInterrupt(t *testing.T) {
	f, d := workingTurn(t)
	f.output(agentfixture.Paint(agentfixture.AgentState{Agent: "claude", Phase: agentfixture.SyntheticClaudeLimit}, 120, 32))
	snapshot := f.proxy.terminal.Snapshot()
	if peerComposerState("claude", snapshot) != PeerComposerUnknown {
		t.Fatal("synthetic limit manufactured a composer")
	}
	if text, ok := qualifiedComposerText("claude", snapshot); ok || text != "" {
		t.Fatalf("synthetic limit admitted input: %q %v", text, ok)
	}
	screen := f.proxy.agentDriveScreen()
	if screen.Interruptible || screen.Idle {
		t.Fatalf("limit invented idle/working proof: %+v", screen)
	}
	if result, err := driveStart(t, d, interruptRequest()); err != nil || result.Outcome != agentdrive.OutcomeRefused || result.Detail != agentdrive.InterruptRefusal {
		t.Fatalf("synthetic limit admitted interrupt: %+v %v", result, err)
	}
	var out bytes.Buffer
	f.proxy.dispatchAgentDrive(&out)
	if out.Len() != 0 {
		t.Fatalf("synthetic limit sent input: %q", out.String())
	}
}
