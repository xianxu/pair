package termcmd

import (
	"io"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// activeChildOwnsScreen is the #196 tri-state read through RepaintModes: a child
// on the alt screen (observed) owns it; off, or never-observed, does not.
func TestActiveChildOwnsScreenTriState(t *testing.T) {
	for _, tt := range []struct {
		name string
		feed []byte
		want bool
	}{
		{"on alt screen", []byte("\x1b[?1049h"), true},
		{"returned to primary", []byte("\x1b[?1049h\x1b[?1049l"), false},
		{"never spoke about the buffer", []byte("plain shell output"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mux := &terminalMux{tabs: []*terminalTab{{id: 1, child: ptychild.NewFakeChild(tt.feed)}}, active: 0}
			if got := mux.activeChildOwnsScreen(); got != tt.want {
				t.Fatalf("activeChildOwnsScreen() = %v, want %v", got, tt.want)
			}
		})
	}
	if (&terminalMux{active: -1}).activeChildOwnsScreen() {
		t.Fatal("no active tab must not own the screen")
	}
	if (&terminalMux{tabs: []*terminalTab{{id: 1}}, active: 0}).activeChildOwnsScreen() {
		t.Fatal("a nil child must not own the screen")
	}
}

// #234's residual: ESC,j typed inside the deadline decodes as ChordAltJ. Under a
// full-screen child its raw bytes \x1bj now reach the child = ESC then j, which
// is what the user typed. At a shell Alt+j is the focus chord and is swallowed.
func TestEscThenJReachesAFullScreenChildAsTwoKeys(t *testing.T) {
	mux := &fakeMux{ownsScreen: true}
	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1bj")}}, mux, &fakeRuntime{}, io.Discard)
	if got := strings.Join(mux.ops, ","); got != "write:\x1bj" {
		t.Fatalf("ops = %q, want the Alt+j bytes forwarded to the full-screen child", got)
	}
	shell := &fakeMux{ownsScreen: false}
	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1bj")}}, shell, &fakeRuntime{}, io.Discard)
	for _, op := range shell.ops {
		if strings.HasPrefix(op, "write:") {
			t.Fatalf("shell ops = %v: Alt+j must not reach the child at a shell", shell.ops)
		}
	}
}

// M-k (focus-left) is the keyboard escape and must NOT pass through even under a
// full-screen child, or the operator with nvim focused on the right has no
// keyboard path back to the agent pane (PQ-1).
func TestFocusLeftNeverPassesThroughToAFullScreenChild(t *testing.T) {
	mux := &fakeMux{ownsScreen: true}
	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1b[107;3u")}}, mux, &fakeRuntime{}, io.Discard) // Alt+k
	for _, op := range mux.ops {
		if strings.HasPrefix(op, "write:") {
			t.Fatalf("ops = %v: M-k must fire focus-left, never reach the full-screen child", mux.ops)
		}
	}
}

// A global fires even under a full-screen child.
func TestGlobalChordFiresUnderAFullScreenChild(t *testing.T) {
	mux := &fakeMux{ownsScreen: true}
	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1b[110;3u")}}, mux, &fakeRuntime{}, io.Discard) // Alt+n restart (global)
	for _, op := range mux.ops {
		if strings.HasPrefix(op, "write:") {
			t.Fatalf("ops = %v: a global must not pass through to the child", mux.ops)
		}
	}
}

// Every chord in the table, both alt-screen states. Under a full-screen child a
// pass-through chord's raw bytes reach the child and nothing else does; a chord
// that does NOT pass through (global, or M-k) never reaches the child. At a
// shell (ownsScreen=false), only draft-only chords reach the child, with their
// raw bytes unchanged.
func TestEveryChordAgainstBothAltScreenStates(t *testing.T) {
	for _, seq := range workbenchshortcut.ChordSequences() {
		chord, ok := workbenchshortcut.DecodeChord([]byte(seq))
		if !ok {
			t.Fatalf("%q did not decode to a chord", seq)
		}
		draftOnly := workbenchshortcut.IsDraftChord(chord)
		pass := draftOnly || workbenchshortcut.RightTerminalChordPassesThrough(chord)

		t.Run("fullscreen/"+workbenchshortcut.ChordName(chord)+"/"+seq, func(t *testing.T) {
			mux := &fakeMux{ownsScreen: true, activeName: "work"}
			pumpStdin(&splitReader{chunks: [][]byte{[]byte(seq)}}, mux, &fakeRuntime{}, io.Discard)
			got := strings.Join(mux.ops, ",")
			if pass {
				if got != "write:"+seq {
					t.Fatalf("pass-through chord %q: ops = %q, want the raw bytes forwarded", seq, got)
				}
			} else {
				for _, op := range mux.ops {
					if strings.HasPrefix(op, "write:") {
						t.Fatalf("non-passthrough chord %q reached the child: ops = %v", seq, mux.ops)
					}
				}
			}
		})

		t.Run("shell/"+workbenchshortcut.ChordName(chord)+"/"+seq, func(t *testing.T) {
			mux := &fakeMux{ownsScreen: false, activeName: "work"}
			pumpStdin(&splitReader{chunks: [][]byte{[]byte(seq)}}, mux, &fakeRuntime{}, io.Discard)
			if draftOnly {
				if got := strings.Join(mux.ops, ","); got != "write:"+seq {
					t.Fatalf("draft-only chord %q: shell ops = %q, want the raw bytes forwarded", seq, got)
				}
				return
			}
			for _, op := range mux.ops {
				if strings.HasPrefix(op, "write:") {
					t.Fatalf("at a shell, recognised chord %q must not reach the child: ops = %v", seq, mux.ops)
				}
			}
		})
	}
}

// The from-anywhere set (#243) delivers GLOBAL chords to the right terminal, so
// they drive its tabs even when it shows a full-screen app — the #227
// regression that role-scoped Alt+Left/Right delivery caused.
func TestFromAnywhereChordsDriveTheRightTerminalUnderFullScreen(t *testing.T) {
	for _, tt := range []struct{ seq, want string }{
		{"\x1b[1;4D", "prev-tab"}, // Alt+Shift+Left
		{"\x1b[1;4C", "next-tab"}, // Alt+Shift+Right
		{"\x1b[84;4u", "new-tab"}, // Alt+Shift+t
	} {
		mux := &fakeMux{ownsScreen: true}
		pumpStdin(&splitReader{chunks: [][]byte{[]byte(tt.seq)}}, mux, &fakeRuntime{}, io.Discard)
		if got := strings.Join(mux.ops, ","); got != tt.want {
			t.Errorf("%q under fullscreen -> ops=%q, want %q (must NOT pass through)", tt.seq, got, tt.want)
		}
	}
}
