package agentdrive

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestParseKeySequence(t *testing.T) {
	specs := []string{"text:/model:界 🙂", "enter", "submit", "up", "down", "esc", "clear-line", "wait:0", "wait:1000"}
	want := KeySequence{{Kind: KeyText, Text: "/model:界 🙂"}, {Kind: KeyEnter}, {Kind: KeySubmit}, {Kind: KeyUp}, {Kind: KeyDown}, {Kind: KeyEsc}, {Kind: KeyClearLine}, {Kind: KeyWait}, {Kind: KeyWait, WaitMS: 1000}}
	got, err := ParseKeySequence(specs)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, %v; want %#v", got, err, want)
	}
}

func TestParseKeySequenceRejectsMalformed(t *testing.T) {
	for _, spec := range []string{"", "return", "ENTER", " enter", "enter ", "enter:x", "escape", "text:", "text:a\nb", "text:a\rb", "text:a\tb", "text:\x1b", "text:\x7f", "text:\u0085", "text:\u200b", "text:\u2028", "text:\xff", "wait:", "wait:-1", "wait:+1", "wait:1.0", "wait: 1", "wait:1 ", "wait:1001", "wait:99999999999999999999999999999"} {
		t.Run(spec, func(t *testing.T) {
			if got, err := ParseKeySequence([]string{spec}); err == nil || got != nil {
				t.Fatalf("accepted %q: %#v, %v", spec, got, err)
			}
		})
	}
}

func TestKeySequenceBounds(t *testing.T) {
	for _, tc := range []struct {
		name  string
		specs []string
		valid bool
	}{
		{"empty", nil, false},
		{"steps limit", repeatSpec("enter", 64), true},
		{"steps overflow", repeatSpec("enter", 65), false},
		{"text limit", []string{"text:" + strings.Repeat("x", 8192)}, true},
		{"text aggregate overflow", []string{"text:" + strings.Repeat("x", 8192), "text:x"}, false},
		{"text byte overflow", []string{"text:" + strings.Repeat("界", 2731)}, false},
		{"wait aggregate limit", repeatSpec("wait:1000", 5), true},
		{"wait aggregate overflow", append(repeatSpec("wait:1000", 5), "wait:1"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseKeySequence(tc.specs)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, err=%v", tc.valid, err)
			}
		})
	}
}

func repeatSpec(spec string, n int) []string {
	s := make([]string, n)
	for i := range s {
		s[i] = spec
	}
	return s
}

func TestKeySequenceValidateClosedShape(t *testing.T) {
	for _, step := range []KeyStep{
		{Kind: "unknown"}, {Kind: KeyText}, {Kind: KeyText, Text: "x", WaitMS: 1}, {Kind: KeyWait, Text: "x"}, {Kind: KeyWait, WaitMS: -1}, {Kind: KeyWait, WaitMS: 1001},
		{Kind: KeyEnter, Text: "x"}, {Kind: KeySubmit, WaitMS: 1}, {Kind: KeyUp, Text: "x"}, {Kind: KeyDown, WaitMS: -1}, {Kind: KeyEsc, Text: "x"}, {Kind: KeyClearLine, WaitMS: 1},
	} {
		if err := (KeySequence{step}).Validate(); err == nil {
			t.Errorf("accepted %#v", step)
		}
	}
}

func TestKeyStepInputBytes(t *testing.T) {
	for _, tc := range []struct {
		step KeyStep
		want []byte
	}{
		{KeyStep{Kind: KeyText, Text: "hello界"}, []byte("hello界")},
		{KeyStep{Kind: KeyEnter}, []byte("\r")}, {KeyStep{Kind: KeySubmit}, []byte("\x1b\r")},
		{KeyStep{Kind: KeyUp}, []byte("\x1b[A")}, {KeyStep{Kind: KeyDown}, []byte("\x1b[B")},
		{KeyStep{Kind: KeyEsc}, []byte("\x1b")}, {KeyStep{Kind: KeyClearLine}, []byte("\x15")}, {KeyStep{Kind: KeyWait}, nil}, {KeyStep{Kind: "unknown"}, nil},
	} {
		if got := tc.step.InputBytes(); !bytes.Equal(got, tc.want) {
			t.Errorf("%#v bytes=%q; want %q", tc.step, got, tc.want)
		}
	}
	seq, err := ParseKeySequence([]string{"esc", "enter"})
	if err != nil {
		t.Fatal(err)
	}
	if len(seq) != 2 || !bytes.Equal(seq[0].InputBytes(), []byte{27}) || !bytes.Equal(seq[1].InputBytes(), []byte{13}) {
		t.Fatalf("ESC boundary lost: %#v", seq)
	}
}

// The recipe's directions are the caller's arrows now: one name, one byte
// sequence (pair#717).
func TestRecipeDirectionsAreCallerArrows(t *testing.T) {
	for kind, bytes := range map[KeyKind]string{KeyLeft: "\x1b[D", KeyRight: "\x1b[C"} {
		if string((KeyStep{Kind: kind}).InputBytes()) != bytes {
			t.Fatalf("%s bytes", kind)
		}
		if _, err := ParseKeySequence([]string{string(kind)}); err != nil {
			t.Fatalf("%s refused: %v", kind, err)
		}
	}
}

// TestStepBytes pins every caller key's bytes to the xterm encodings the
// drive sends (pair#717). A table change must change this test too.
func TestStepBytes(t *testing.T) {
	want := map[string]string{
		"enter": "\r", "submit": "\x1b\r", "up": "\x1b[A", "down": "\x1b[B", "left": "\x1b[D", "right": "\x1b[C",
		"tab": "\t", "shift-tab": "\x1b[Z", "backspace": "\x7f", "delete": "\x1b[3~", "home": "\x1b[H", "end": "\x1b[F",
		"pageup": "\x1b[5~", "pagedown": "\x1b[6~", "esc": "\x1b", "clear-line": "\x15",
		"f1": "\x1bOP", "f2": "\x1bOQ", "f3": "\x1bOR", "f4": "\x1bOS", "f5": "\x1b[15~", "f6": "\x1b[17~",
		"f7": "\x1b[18~", "f8": "\x1b[19~", "f9": "\x1b[20~", "f10": "\x1b[21~", "f11": "\x1b[23~", "f12": "\x1b[24~",
		"ctrl-a": "\x01", "ctrl-o": "\x0f", "ctrl-r": "\x12", "ctrl-y": "\x19", "alt-b": "\x1bb", "alt-z": "\x1bz",
	}
	for name, bytes := range want {
		seq, err := ParseKeySequence([]string{name})
		if err != nil {
			t.Errorf("%s refused: %v", name, err)
			continue
		}
		if got := string(seq[0].InputBytes()); got != bytes {
			t.Errorf("%s = %q, want %q", name, got, bytes)
		}
	}
	named := 0
	for _, step := range Steps {
		if step.Kind == KeyText || step.Kind == KeyWait {
			continue
		}
		named++
		if step.Bytes == "" {
			t.Errorf("%s has no bytes", step.Syntax)
		}
	}
	if named != 16+12+(26-len(RefusedSteps))+26 {
		t.Errorf("named keys = %d; the families or exclusions changed", named)
	}
}

func TestRefusedStepsStateTheirReason(t *testing.T) {
	for name, reason := range RefusedSteps {
		_, err := ParseKeySequence([]string{"tab", string(name)})
		if err == nil || !strings.Contains(err.Error(), reason) {
			t.Errorf("%s: %v", name, err)
		}
		if isStep(name) {
			t.Errorf("%s is both refused and a step", name)
		}
	}
	if !strings.Contains(RefusedSteps["ctrl-c"], "couch --interrupt") {
		t.Error("ctrl-c refusal does not point at the interrupt")
	}
}

func TestUnknownStepListsTheNames(t *testing.T) {
	for _, name := range []string{"f13", "ctrl-1", "alt-[", "shift-up", "Tab"} {
		_, err := ParseKeySequence([]string{"enter", name})
		if err == nil || !strings.Contains(err.Error(), StepSyntax()) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func FuzzParseKeySequence(f *testing.F) {
	for _, seed := range []string{"text:hello\x00enter", "esc\x00enter", "wait:0", "wait:1000", "wait:-1", "wait:99999999999999999999", "text:\xff", "text:a\nb", "text:\x1b", "wait:1000\x00wait:1000\x00wait:1000\x00wait:1000\x00wait:1000\x00wait:1", strings.Join(repeatSpec("enter", 65), "\x00"), "text:" + strings.Repeat("x", 8193)} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		seq, err := ParseKeySequence(strings.Split(raw, "\x00"))
		if err != nil {
			if seq != nil {
				t.Fatal("invalid parse returned actionable sequence")
			}
			return
		}
		if len(seq) < 1 || len(seq) > 64 {
			t.Fatalf("step count=%d", len(seq))
		}
		textBytes, totalWait := 0, 0
		for _, step := range seq {
			switch step.Kind {
			case KeyText:
				if len(step.Text) == 0 || !utf8.ValidString(step.Text) || step.WaitMS != 0 {
					t.Fatalf("bad text: %#v", step)
				}
				for _, r := range step.Text {
					if !unicode.IsPrint(r) {
						t.Fatalf("nonprintable rune %U", r)
					}
				}
				textBytes += len(step.Text)
			case KeyWait:
				if step.WaitMS < 0 || step.WaitMS > 1000 || step.Text != "" || step.InputBytes() != nil {
					t.Fatalf("bad wait: %#v", step)
				}
				totalWait += step.WaitMS
			case KeyEnter, KeySubmit, KeyUp, KeyDown, KeyEsc, KeyClearLine:
				if step.Text != "" || step.WaitMS != 0 {
					t.Fatalf("bad shape: %#v", step)
				}
			default:
				t.Fatalf("unknown kind: %#v", step)
			}
		}
		if textBytes > 8192 || totalWait > 5000 {
			t.Fatalf("text bytes=%d wait ms=%d", textBytes, totalWait)
		}
	})
}
