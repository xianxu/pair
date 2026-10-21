package wrapcmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// Native fixtures must satisfy the same input authority as the real wrapper.
// Checking only the prompt glyph or a fixture's own state misses footer and
// cursor geometry that make the production composer correctly decline input.
func TestDisposableFixtureComposerConformance(t *testing.T) {
	for _, size := range []struct{ cols, rows int }{{120, 32}, {132, 36}, {20, 10}} {
		for _, agent := range []string{"claude", "codex"} {
			for _, tc := range []struct {
				name  string
				phase agentfixture.Phase
				draft string
				text  string
				want  PeerComposerState
			}{
				{"idle empty", agentfixture.Idle, "", "", PeerComposerEmpty},
				{"idle occupied", agentfixture.Idle, "ordinary input", "ordinary input", PeerComposerOccupied},
				{"idle multiline", agentfixture.Idle, "one\ntwo", "one\ntwo", PeerComposerOccupied},
				{"idle wrapped", agentfixture.Idle, strings.Repeat("x", size.cols+3), strings.Repeat("x", size.cols-3) + "\n" + strings.Repeat("x", 6), PeerComposerOccupied},
				// Peer input does not require execution idle: a running agent
				// may retain an available empty composer below its progress.
				{"working empty", agentfixture.Working, "", "", PeerComposerEmpty},
				{"working occupied", agentfixture.Working, "ordinary input", "ordinary input", PeerComposerOccupied},
				{"picker", agentfixture.Picker, "", "", PeerComposerUnknown},
			} {
				t.Run(fmt.Sprintf("%s/%dx%d/%s", agent, size.cols, size.rows, tc.name), func(t *testing.T) {
					model, err := newTerminalModel(size.cols, size.rows)
					if err != nil {
						t.Fatal(err)
					}
					defer model.Close()
					paint := agentfixture.Paint(agentfixture.AgentState{Agent: agent, Phase: tc.phase, Draft: tc.draft}, size.cols, size.rows)
					if err := model.Feed([]byte(paint)); err != nil {
						t.Fatal(err)
					}
					snapshot := model.Snapshot()
					if got := peerComposerState(agent, snapshot); got != tc.want {
						t.Fatalf("fixture state=%v want=%v cursor=%+v", got, tc.want, snapshot.Cursor)
					}
					text, ok := qualifiedComposerText(agent, snapshot)
					if ok != (tc.want != PeerComposerUnknown) || text != tc.text {
						t.Fatalf("fixture text=%q recognized=%t want=%q recognized=%t", text, ok, tc.text, tc.want != PeerComposerUnknown)
					}
				})
			}
		}
	}
}
