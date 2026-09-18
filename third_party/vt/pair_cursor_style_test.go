package vt

import "testing"

// DECSCUSR 0 or an absent parameter hands the cursor back to the host
// terminal's configured default; RIS and a fresh screen start there (#283).
func TestPairCursorStyleCarriesHostDefault(t *testing.T) {
	cases := []struct {
		name, stream string
		style        CursorStyle
		steady       bool
	}{
		{"fresh", "", CursorDefault, false},
		{"zero after bar", "\x1b[6 q\x1b[0 q", CursorDefault, false},
		{"absent after underline", "\x1b[4 q\x1b[ q", CursorDefault, false},
		{"zero after blinking block", "\x1b[1 q\x1b[0 q", CursorDefault, false},
		{"reset", "\x1b[6 q\x1bc", CursorDefault, false},
		{"blinking block", "\x1b[1 q", CursorBlock, false},
		{"steady block", "\x1b[2 q", CursorBlock, true},
		{"blinking underline", "\x1b[3 q", CursorUnderline, false},
		{"steady underline", "\x1b[4 q", CursorUnderline, true},
		{"blinking bar", "\x1b[5 q", CursorBar, false},
		{"steady bar", "\x1b[6 q", CursorBar, true},
		{"unknown ignored", "\x1b[3 q\x1b[7 q", CursorUnderline, false},
	}
	for _, tc := range cases {
		e := NewEmulator(4, 2)
		e.WriteString(tc.stream)
		if c := e.Cursor(); c.Style != tc.style || c.Steady != tc.steady {
			t.Errorf("%s: style=%d steady=%v want %d/%v", tc.name, c.Style, c.Steady, tc.style, tc.steady)
		}
		e.Close()
	}
}

// The callback fires on every change of state, including an explicit blinking
// block returning to the default, which the old mapping made indistinguishable.
func TestPairCursorStyleCallbackReportsDefault(t *testing.T) {
	e := NewEmulator(4, 2)
	defer e.Close()
	var got []CursorStyle
	e.SetCallbacks(Callbacks{CursorStyle: func(s CursorStyle, _ bool) { got = append(got, s) }})
	e.WriteString("\x1b[1 q\x1b[0 q")
	if len(got) != 2 || got[0] != CursorBlock || got[1] != CursorDefault {
		t.Fatalf("callbacks %v", got)
	}
}
