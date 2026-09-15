package workbenchshortcut

import (
	"bytes"
	"testing"
)

func TestFindChordOutsidePastePartitions(t *testing.T) {
	for chord := Chord(1); chord < ChordMax(); chord++ {
		for _, raw := range ChordEncodings(chord) {
			for _, pasted := range []bool{false, true} {
				prefix := []byte("text\x1b[200~")
				if pasted {
					prefix = []byte("text")
				}
				prefix = append(prefix, raw...)
				prefix = append(prefix, []byte("\x1b[201~")...)
				input := append(append([]byte(nil), prefix...), raw...)
				before, got, candidate, rest, ok := FindChordOutsidePaste(input, pasted)
				if !ok || got != chord || !bytes.Equal(before, prefix) || !bytes.Equal(candidate, raw) || len(rest) != 0 {
					t.Fatalf("paste=%v input=%q before=%q chord=%v candidate=%q rest=%q", pasted, input, before, got, candidate, rest)
				}
			}
		}
	}
}

func TestPendingInputSuffixBound(t *testing.T) {
	patterns := [][]byte{[]byte("\x1b[200~"), []byte("\x1b[201~")}
	for chord := Chord(1); chord < ChordMax(); chord++ {
		patterns = append(patterns, ChordEncodings(chord)...)
	}
	for _, pattern := range patterns {
		for n := 1; n < len(pattern); n++ {
			input := append([]byte("ordinary"), pattern[:n]...)
			if got := PendingInputSuffix(input); got != n {
				t.Fatalf("input=%q suffix=%d want=%d", input, got, n)
			}
		}
	}
	if got := PendingInputSuffix([]byte("ordinary\x1b[999999x")); got != 0 {
		t.Fatalf("unrelated sequence held %d", got)
	}
}
