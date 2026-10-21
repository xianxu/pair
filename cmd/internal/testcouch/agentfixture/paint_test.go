package agentfixture

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestComposerCapturesRemainPinned(t *testing.T) {
	for agent, want := range map[string]string{"claude": "a0c73050e9f1398c4c5d00e88005db6157ecf2aeb49ce7f3cf55373088a24d96", "codex": "418f46453c1a50e893769d8dbb855769019b116752a65e1ecc47494f4b9bc808"} {
		raw, err := CapturedComposer(agent)
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != want {
			t.Fatalf("%s capture changed: %s", agent, got)
		}
		frame := Paint(AgentState{Agent: agent, Phase: Idle}, 120, 38)
		if !strings.Contains(frame, "\x1b[?2004h") || !strings.HasSuffix(frame, "\x1b[?25h\x1b[?2026l") {
			t.Fatal("composer lost paste/coherent cursor modes")
		}
		if strings.Contains(frame, "alpha") || strings.Contains(frame, "loading") {
			t.Fatal("occupied or startup fixture composer")
		}
	}
}
