package termcmd

import (
	"fmt"
	"reflect"
	"testing"
)

func decodeRenameChunks(chunks ...[]byte) ([]RenameEvent, RenameDecoderState, bool) {
	var state RenameDecoderState
	var events []RenameEvent
	var exited bool
	for _, chunk := range chunks {
		var got []RenameEvent
		state, got, exited = DecodeRenameInput(state, chunk, false, false)
		events = append(events, got...)
		if exited {
			break
		}
	}
	return events, state, exited
}

func TestDecodeRenameInputControlsAtEverySplit(t *testing.T) {
	tests := []struct {
		name string
		seq  string
		want RenameEventKind
	}{
		{"enter cr", "\r", RenameCommit},
		{"enter lf", "\n", RenameCommit},
		{"backspace del", "\x7f", RenameBackspace},
		{"backspace bs", "\b", RenameBackspace},
		{"left csi", "\x1b[D", RenameMoveLeft},
		{"left ss3", "\x1bOD", RenameMoveLeft},
		{"right csi", "\x1b[C", RenameMoveRight},
		{"right ss3", "\x1bOC", RenameMoveRight},
		{"home csi", "\x1b[H", RenameHome},
		{"home ss3", "\x1bOH", RenameHome},
		{"home tilde", "\x1b[1~", RenameHome},
		{"end csi", "\x1b[F", RenameEnd},
		{"end ss3", "\x1bOF", RenameEnd},
		{"end tilde", "\x1b[4~", RenameEnd},
		{"delete", "\x1b[3~", RenameDelete},
		{"super backspace", "\x1b[127;9u", RenameDeleteToStart},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for split := 0; split <= len(tt.seq); split++ {
				events, state, exited := decodeRenameChunks([]byte(tt.seq[:split]), []byte(tt.seq[split:]))
				if exited != (tt.want == RenameCommit) {
					t.Fatalf("split %d exited=%v", split, exited)
				}
				want := []RenameEvent{{Kind: tt.want}}
				if !reflect.DeepEqual(events, want) {
					t.Fatalf("split %d events=%#v, want %#v (pending %q)", split, events, want, state.Pending)
				}
				if len(state.Pending) != 0 {
					t.Fatalf("split %d pending=%q, want empty", split, state.Pending)
				}
			}
		})
	}
}

func TestDecodeRenameInputUTF8AtEverySplit(t *testing.T) {
	for _, text := range []string{"é", "界", "🙂"} {
		t.Run(text, func(t *testing.T) {
			for split := 0; split <= len(text); split++ {
				events, state, exited := decodeRenameChunks([]byte(text[:split]), []byte(text[split:]))
				want := []RenameEvent{{Kind: RenameInsert, Rune: []rune(text)[0]}}
				if exited || !reflect.DeepEqual(events, want) || len(state.Pending) != 0 {
					t.Fatalf("split %d = events %#v pending %q exited %v; want %#v", split, events, state.Pending, exited, want)
				}
			}
		})
	}
}

func TestDecodeRenameInputConsumesWorkbenchShortcutsAndMouse(t *testing.T) {
	for _, seq := range []string{
		"\x1br",
		"\x1b[114;3u",
		"\x1b[110;7u",
		"\x1b[1;3A",
		"\x1b[13;4u",
		"\x1b[<0;12;4M",
		"\x1b[<0;12;4m",
	} {
		t.Run(fmt.Sprintf("%q", seq), func(t *testing.T) {
			for split := 0; split <= len(seq); split++ {
				events, state, exited := decodeRenameChunks([]byte(seq[:split]), []byte(seq[split:]))
				if exited || !reflect.DeepEqual(events, []RenameEvent{{Kind: RenameConsume}}) || len(state.Pending) != 0 {
					t.Fatalf("%q split %d = %#v pending %q exited=%v", seq, split, events, state.Pending, exited)
				}
			}
		})
	}
}

func TestDecodeRenameInputConsumesBracketedPastePayload(t *testing.T) {
	seq := "\x1b[200~hello🙂\x1b[201~x"
	want := []RenameEvent{{Kind: RenameConsume}, {Kind: RenameConsume}, {Kind: RenameInsert, Rune: 'x'}}
	for split := 0; split <= len(seq); split++ {
		events, state, exited := decodeRenameChunks([]byte(seq[:split]), []byte(seq[split:]))
		if exited || state.InPaste || len(state.Pending) != 0 {
			t.Fatalf("split %d state=%#v exited=%v", split, state, exited)
		}
		if !reflect.DeepEqual(events, want) {
			t.Fatalf("split %d events=%#v, want %#v", split, events, want)
		}
	}
}

func TestDecodeRenameInputEscapeTimeoutAndEOF(t *testing.T) {
	state, events, exited := DecodeRenameInput(RenameDecoderState{}, []byte{0x1b}, false, false)
	if exited || len(events) != 0 || string(state.Pending) != "\x1b" {
		t.Fatalf("held escape = %#v %#v %v", state, events, exited)
	}
	state, events, exited = DecodeRenameInput(state, nil, true, false)
	if !exited || !reflect.DeepEqual(events, []RenameEvent{{Kind: RenameCancel}}) || len(state.Pending) != 0 {
		t.Fatalf("flushed escape = %#v %#v %v", state, events, exited)
	}

	state, _, _ = DecodeRenameInput(RenameDecoderState{}, []byte("\x1b["), false, false)
	state, events, exited = DecodeRenameInput(state, nil, false, true)
	if !exited || !reflect.DeepEqual(events, []RenameEvent{{Kind: RenameConsume}, {Kind: RenameCancel}}) || len(state.Pending) != 0 {
		t.Fatalf("EOF incomplete = %#v %#v %v", state, events, exited)
	}
}

func TestDecodeRenameInputConsumesInvalidAndPreservesFollowingRune(t *testing.T) {
	events, state, exited := decodeRenameChunks([]byte{0xff, 'x'})
	want := []RenameEvent{{Kind: RenameConsume}, {Kind: RenameInsert, Rune: 'x'}}
	if exited || !reflect.DeepEqual(events, want) || len(state.Pending) != 0 {
		t.Fatalf("invalid = %#v pending=%q exited=%v, want %#v", events, state.Pending, exited, want)
	}

	events, state, exited = decodeRenameChunks([]byte("\x1b[?;x"))
	want = []RenameEvent{{Kind: RenameConsume}, {Kind: RenameInsert, Rune: 'x'}}
	if exited || !reflect.DeepEqual(events, want) || len(state.Pending) != 0 {
		t.Fatalf("malformed = %#v pending=%q exited=%v, want %#v", events, state.Pending, exited, want)
	}
}

func TestDecodeRenameInputExitConsumesSameReadSuffix(t *testing.T) {
	for _, seq := range []string{"x\ry", "x\x1by"} {
		state, events, exited := DecodeRenameInput(RenameDecoderState{}, []byte(seq), true, false)
		if !exited || len(state.Pending) != 0 {
			t.Fatalf("%q state=%#v exited=%v", seq, state, exited)
		}
		if len(events) != 2 || events[0] != (RenameEvent{Kind: RenameInsert, Rune: 'x'}) {
			t.Fatalf("%q events=%#v", seq, events)
		}
		if events[1].Kind != RenameCommit && events[1].Kind != RenameCancel {
			t.Fatalf("%q exit event=%#v", seq, events[1])
		}
	}
}
