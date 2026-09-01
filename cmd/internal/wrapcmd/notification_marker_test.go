package wrapcmd

import (
	"testing"
	"unicode"
)

func TestClaudeEndOfTurnGrammar(t *testing.T) {
	re := endOfTurnByAgent["claude"]
	tests := []struct {
		name   string
		marker string
		want   bool
	}{
		{"ASCII verb", "✻ Churned for 21s", true},
		{"precomposed Unicode letter", "✻ Sautéed for 34s · done 1:39 PM", true},
		{"letters from another script", "✻ Сделано for 1m 2s", true},
		{"empty verb", "✻ for 34s", false},
		{"combining mark", "✻ Saute\u0301ed for 34s", false},
		{"digit", "✻ Done2 for 34s", false},
		{"punctuation", "✻ Re-done for 34s", false},
		{"whitespace inside verb", "✻ All done for 34s", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := re.MatchString(tt.marker); got != tt.want {
				t.Fatalf("MatchString(%q) = %v, want %v", tt.marker, got, tt.want)
			}
		})
	}
}

func FuzzClaudeEndOfTurnGrammar(f *testing.F) {
	for _, r := range []rune{'A', 'é', 'Я', '\u0301', '2', '-', ' '} {
		f.Add(int32(r))
	}
	f.Fuzz(func(t *testing.T, value int32) {
		r := rune(value)
		marker := "✻ X" + string(r) + "Y for 1s"
		want := unicode.IsLetter(r)
		if got := endOfTurnByAgent["claude"].MatchString(marker); got != want {
			t.Fatalf("inserted rune %U: match = %v, want %v", r, got, want)
		}
	})
}
