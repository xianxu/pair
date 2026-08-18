package wrapcmd

import (
	"os"
	"testing"
)

type composerDifferentialCase struct {
	name              string
	stream            []byte
	want              bool
	allowedCorrection string
}

func TestCodexComposerActiveSnapshotDifferential(t *testing.T) {
	composer := "\x1b[19;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[20;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[21;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[?25h\x1b[20;3H"
	cases := []composerDifferentialCase{
		{name: "generated composer", stream: []byte(composer), want: true},
		{name: "hidden cursor", stream: []byte(composer + "\x1b[?25l")},
		{name: "erased composer", stream: []byte(composer + "\x1b[2J\x1b[?25h\x1b[20;3H")},
		{name: "composer away from cursor", stream: []byte(composer + "\x1b[?25h\x1b[30;3H")},
		{
			name: "one local painted row plus distant complete evidence",
			stream: []byte("\x1b[9;1H\x1b[48;2;57;57;57m\x1b[K" +
				"\x1b[10;1H\x1b[48;2;57;57;57m\x1b[K" +
				"\x1b[20;1H\x1b[48;2;57;57;57m\x1b[K" +
				"\x1b[?25h\x1b[20;3H"),
		},
	}

	runComposerSnapshotDifferential(t, cases, func(stream []byte) bool {
		tracker := newCodexComposerTracker()
		tracker.resize(38, 120)
		tracker.feed(stream)
		return tracker.state().active()
	}, codexComposerActive)
}

func TestMuseComposerActiveSnapshotDifferential(t *testing.T) {
	literal, err := os.ReadFile("testdata/tty/muse/0.1.0-R708.1/composer.raw")
	if err != nil {
		t.Fatalf("read literal Muse fixture: %v", err)
	}
	qualified := "\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ \x1b[9;1H\x1b[2m────\x1b[?25h\x1b[8;3H"
	cases := []composerDifferentialCase{
		{name: "literal captured composer", stream: literal, want: true},
		{name: "generated captured signature", stream: []byte(qualified), want: true},
		{name: "hidden cursor", stream: []byte(qualified + "\x1b[?25l")},
		{name: "bare old U+203A glyph", stream: []byte("\x1b[8;1H› \x1b[?25h\x1b[8;3H")},
		{
			name:              "unqualified U+27E9 glyph",
			stream:            []byte("\x1b[8;1H⟩ \x1b[?25h\x1b[8;3H"),
			allowedCorrection: "unqualified glyph",
		},
		{
			name:              "stale prompt mutation",
			stream:            []byte(qualified + "\x1b[8;1H\x1b[22mnot a prompt\x1b[?25h\x1b[8;3H"),
			allowedCorrection: "stale mutation",
		},
		{
			name: "one local weak glyph plus distant complete evidence",
			stream: []byte("\x1b[2;1H\x1b[2m────\x1b[3;1H\x1b[22m⟩ \x1b[4;1H\x1b[2m────" +
				"\x1b[20;1H\x1b[22m⟩ \x1b[?25h\x1b[20;3H"),
			allowedCorrection: "unqualified glyph",
		},
	}

	runComposerSnapshotDifferential(t, cases, func(stream []byte) bool {
		tracker := newMuseComposerTracker()
		tracker.resize(38, 120)
		tracker.feed(stream)
		return tracker.state().active()
	}, museComposerActive)
}

func runComposerSnapshotDifferential(
	t *testing.T,
	cases []composerDifferentialCase,
	legacy func([]byte) bool,
	recognize composerRecognizer,
) {
	t.Helper()
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			old := legacy(test.stream)
			model := newTerminalModelForTest(t, 120, 38)
			if err := model.Feed(test.stream); err != nil {
				t.Fatalf("feed terminal model: %v", err)
			}
			new := recognize(model.Snapshot())
			if !old && new {
				t.Fatalf("forbidden differential: old=false new=true")
			}
			if test.allowedCorrection == "" && new != old {
				t.Fatalf("unapproved differential: old=%t new=%t", old, new)
			}
			if test.allowedCorrection != "" && (!old || new) {
				t.Fatalf("%s correction = old:%t new:%t, want old:true new:false", test.allowedCorrection, old, new)
			}
			if new != test.want {
				t.Fatalf("snapshot active = %t, want %t", new, test.want)
			}
		})
	}
}
