package sessioninventory

import (
	"strings"
	"testing"
)

func TestRecoveryTailScoreNormalizationAndClipping(t *testing.T) {
	messages := []RecoveryMessage{{Role: EventAssistant, Text: recoveryTestText, Complete: true}}
	full := ScoreRecoveryTail([]string{"• **one two** `three` four five six", "seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen"}, messages)
	if full.Matched != 11 || full.Total != 11 || full.Percent != 100 {
		t.Fatalf("score=%+v", full)
	}
	clipped := ScoreRecoveryTail([]string{"nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen"}, messages)
	if clipped.Matched != 3 || clipped.Total != 11 {
		t.Fatalf("score=%+v", clipped)
	}
	if score := ScoreRecoveryTail([]string{recoveryTestText}, []RecoveryMessage{{Role: EventOperator, Text: recoveryTestText, Complete: true}, {Role: EventAssistant, Text: recoveryTestText}}); score.Total != 0 {
		t.Fatal("noncomplete/nonassistant scored")
	}
}
func TestRecoveryTailLastThreeAndDistinctWindows(t *testing.T) {
	messages := []RecoveryMessage{{Role: EventAssistant, Text: recoveryTestText, Complete: true}}
	for i := 0; i < 3; i++ {
		messages = append(messages, RecoveryMessage{Role: EventAssistant, Text: strings.Repeat("different ", 16), Complete: true})
	}
	if score := ScoreRecoveryTail([]string{recoveryTestText}, messages); score.Matched != 0 {
		t.Fatal("old fourth message scored")
	}
	score := ScoreRecoveryTail([]string{strings.Repeat("same ", 18)}, []RecoveryMessage{{Role: EventAssistant, Text: strings.Repeat("same ", 18), Complete: true}})
	if score.Total != 1 || score.Matched != 1 {
		t.Fatalf("repetition inflated support: %+v", score)
	}
}

func TestRecoveryNativeOnlyRankingStableTiesAndSnapshotOwnership(t *testing.T) {
	c := RecoveryCandidate{Agent: AgentCodex, NativeID: "a", CWD: "/repo", RootEligible: true, Interactive: InteractiveEvidence{State: InteractiveYes}, Messages: []RecoveryMessage{{Role: EventAssistant, Text: recoveryTestText, Complete: true}}}
	d := c
	d.NativeID = "b"
	wrongCWD := c
	wrongCWD.CWD = "/elsewhere"
	child := c
	child.RootEligible = false
	exec := c
	exec.Interactive.State = InteractiveNo
	for _, input := range [][]RecoveryCandidate{{d, c, child, wrongCWD, exec}, {c, d, wrongCWD, exec, child}} {
		ranked := rankRecoveryCandidates(input, []string{recoveryTestText}, "/repo", nil, true)
		if len(ranked) != 2 || ranked[0].Candidate.NativeID != "a" || ranked[1].Candidate.NativeID != "b" {
			t.Fatalf("unstable ties or ineligible root: %+v", ranked)
		}
		if ranked[0].Score != (RecoveryTailScore{}) || !ranked[0].RequiresExplicitSelection {
			t.Fatalf("screen score or automatic authority: %+v", ranked[0])
		}
		ranked[0].Candidate.Messages[0].Text = "changed"
		if c.Messages[0].Text != recoveryTestText {
			t.Fatal("native ranking aliases input message snapshot")
		}
	}
}
