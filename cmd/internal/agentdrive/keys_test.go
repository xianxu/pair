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
