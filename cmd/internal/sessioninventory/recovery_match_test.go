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
