package termcmd

import (
	"bytes"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

type RenameDecoderState struct {
	Pending []byte
	InPaste bool
}

var renameControlSequences = []struct {
	sequence string
	event    RenameEventKind
}{
	{"\x1b[D", RenameMoveLeft},
	{"\x1bOD", RenameMoveLeft},
	{"\x1b[C", RenameMoveRight},
	{"\x1bOC", RenameMoveRight},
	{"\x1b[H", RenameHome},
	{"\x1bOH", RenameHome},
	{"\x1b[1~", RenameHome},
	{"\x1b[F", RenameEnd},
	{"\x1bOF", RenameEnd},
	{"\x1b[4~", RenameEnd},
	{"\x1b[3~", RenameDelete},
	{"\x1b[127;9u", RenameDeleteToStart},
}

var (
	bracketedPasteStart = []byte("\x1b[200~")
	bracketedPasteEnd   = []byte("\x1b[201~")
)

func DecodeRenameInput(state RenameDecoderState, data []byte, flushEscape, eof bool) (RenameDecoderState, []RenameEvent, bool) {
	input := append(append([]byte(nil), state.Pending...), data...)
	state.Pending = nil
	var events []RenameEvent

	for len(input) > 0 {
		if state.InPaste {
			if end := bytes.Index(input, bracketedPasteEnd); end >= 0 {
				input = input[end+len(bracketedPasteEnd):]
				state.InPaste = false
				events = append(events, RenameEvent{Kind: RenameConsume})
				continue
			}
			keep := longestSuffixPrefix(input, bracketedPasteEnd)
			state.Pending = append(state.Pending, input[len(input)-keep:]...)
			input = nil
			break
		}

		switch input[0] {
		case '\r', '\n':
			events = append(events, RenameEvent{Kind: RenameCommit})
			state.Pending = nil
			return state, events, true
		case 0x7f, '\b':
			events = append(events, RenameEvent{Kind: RenameBackspace})
			input = input[1:]
			continue
		case 0x1b:
			if len(input) == 1 {
				if flushEscape || eof {
					events = append(events, RenameEvent{Kind: RenameCancel})
					return RenameDecoderState{}, events, true
				}
				state.Pending = append(state.Pending, input...)
				return state, events, false
			}
			if bytes.HasPrefix(input, bracketedPasteStart) {
				events = append(events, RenameEvent{Kind: RenameConsume})
				state.InPaste = true
				input = input[len(bracketedPasteStart):]
				continue
			}
			if !eof && bytes.HasPrefix(bracketedPasteStart, input) {
				state.Pending = append(state.Pending, input...)
				return state, events, false
			}
			if event, size, ok := completeRenameControl(input); ok {
				events = append(events, RenameEvent{Kind: event})
				input = input[size:]
				continue
			}
			if !eof && renameControlPrefix(input) {
				state.Pending = append(state.Pending, input...)
				return state, events, false
			}
			if _, rest, ok := workbenchshortcut.DecodeChordPrefix(input); ok {
				size := len(input) - len(rest)
				events = append(events, RenameEvent{Kind: RenameConsume})
				input = input[size:]
				continue
			}
			if !eof && workbenchshortcut.IsChordPrefix(input) {
				state.Pending = append(state.Pending, input...)
				return state, events, false
			}
			if size, complete := sgrMouseSize(input); complete {
				events = append(events, RenameEvent{Kind: RenameConsume})
				input = input[size:]
				continue
			}
			if !eof && escapeSequenceIncomplete(input) {
				state.Pending = append(state.Pending, input...)
				return state, events, false
			}
			if flushEscape {
				events = append(events, RenameEvent{Kind: RenameCancel})
				return RenameDecoderState{}, events, true
			}
			size := malformedEscapeSize(input)
			events = append(events, RenameEvent{Kind: RenameConsume})
			input = input[size:]
			continue
		}

		if input[0] < utf8.RuneSelf {
			if input[0] >= 0x20 {
				events = append(events, RenameEvent{Kind: RenameInsert, Rune: rune(input[0])})
			} else {
				events = append(events, RenameEvent{Kind: RenameConsume})
			}
			input = input[1:]
			continue
		}
		if !utf8.FullRune(input) {
			state.Pending = append(state.Pending, input...)
			input = nil
			break
		}
		r, size := utf8.DecodeRune(input)
		if r == utf8.RuneError && size == 1 {
			events = append(events, RenameEvent{Kind: RenameConsume})
			input = input[1:]
			continue
		}
		events = append(events, RenameEvent{Kind: RenameInsert, Rune: r})
		input = input[size:]
	}

	if eof {
		state.Pending = nil
		state.InPaste = false
		events = append(events, RenameEvent{Kind: RenameCancel})
		return state, events, true
	}
	return state, events, false
}

func completeRenameControl(input []byte) (RenameEventKind, int, bool) {
	for _, candidate := range renameControlSequences {
		if bytes.HasPrefix(input, []byte(candidate.sequence)) {
			return candidate.event, len(candidate.sequence), true
		}
	}
	return RenameConsume, 0, false
}

func renameControlPrefix(input []byte) bool {
	for _, candidate := range renameControlSequences {
		if len(input) < len(candidate.sequence) && bytes.HasPrefix([]byte(candidate.sequence), input) {
			return true
		}
	}
	return false
}

func sgrMouseSize(input []byte) (int, bool) {
	if !bytes.HasPrefix(input, []byte("\x1b[<")) {
		return 0, false
	}
	// Same 'M' press / 'm' release terminator pair the pump uses — driven by the
	// one constant so the sites can't drift apart (they did: #127).
	idx := bytes.IndexAny(input[3:], sgrMouseTerminators)
	if idx < 0 {
		return 0, false
	}
	return idx + 3 + 1, true
}

func escapeSequenceIncomplete(input []byte) bool {
	if len(input) < 2 {
		return bytes.Equal(input, []byte{0x1b})
	}
	switch input[1] {
	case '[':
		return csiEnd(input) < 0
	case 'O':
		return len(input) < 3
	default:
		return false
	}
}

func malformedEscapeSize(input []byte) int {
	if len(input) < 2 {
		return len(input)
	}
	if input[1] != '[' && input[1] != 'O' {
		return 2
	}
	if end := csiEnd(input); end >= 0 {
		return end
	}
	return len(input)
}

func isTerminalFinalByte(c byte) bool {
	return c >= 0x40 && c <= 0x7e
}

func longestSuffixPrefix(input, prefix []byte) int {
	max := len(input)
	if max >= len(prefix) {
		max = len(prefix) - 1
	}
	for size := max; size > 0; size-- {
		if bytes.Equal(input[len(input)-size:], prefix[:size]) {
			return size
		}
	}
	return 0
}
