package workbenchshortcut

import "bytes"

const PasteStart = "\x1b[200~"
const PasteEnd = "\x1b[201~"

// FindChordOutsidePaste finds the next actionable chord. The caller retains
// paste state by processing before; this lookahead owns no state between reads.
func FindChordOutsidePaste(data []byte, inPaste bool) ([]byte, Chord, []byte, []byte, bool) {
	for offset := 0; offset < len(data); {
		tail := data[offset:]
		if inPaste {
			if bytes.HasPrefix(tail, []byte(PasteEnd)) {
				inPaste = false
				offset += len(PasteEnd)
			} else {
				offset++
			}
			continue
		}
		if bytes.HasPrefix(tail, []byte(PasteStart)) {
			inPaste = true
			offset += len(PasteStart)
			continue
		}
		if chord, rest, ok := DecodeChordPrefix(tail); ok {
			end := len(data) - len(rest)
			return data[:offset], chord, data[offset:end], rest, true
		}
		offset++
	}
	return data, ChordUnknown, nil, nil, false
}

// PendingInputSuffix retains only a proper prefix of a finite input encoding.
// Ordinary input before it can be emitted immediately. The return value is
// strictly smaller than the longest encoding, independent of input size.
func PendingInputSuffix(data []byte) int {
	longest := 0
	consider := func(pattern string) {
		limit := min(len(data), len(pattern)-1)
		for n := limit; n > longest; n-- {
			if bytes.Equal(data[len(data)-n:], []byte(pattern[:n])) {
				longest = n
				break
			}
		}
	}
	consider(PasteStart)
	consider(PasteEnd)
	for _, candidate := range chordSequences {
		consider(candidate.sequence)
	}
	return longest
}
