package wrapcmd

import (
	"os"
	"sync"
	"testing"
)

func TestMuseComposerTrackerDetectsObservedComposer(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)

	// Empty prompt at 30;1H with prompt glyph, cursor at 30;3H.
	tr.feed([]byte(
		"\x1b[30;1H\x1b[38;2;90;160;255;49m\xe2\x9f\xa9 \x1b[?25h\x1b[30;3H",
	))

	if st := tr.state(); !st.active() {
		t.Fatalf("composer active = false, want true (state: %+v)", st)
	}
}

func TestMuseComposerTrackerDetectsNonEmptyComposer(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)

	// Non-empty prompt with BG (work on #140) at 9;1H.
	tr.feed([]byte(
		"\x1b[9;1H\x1b[38;2;90;160;255;48;2;38;56;84m\xe2\x9f\xa9 \x1b[38;2;204;211;219;48;2;38;56;84mwork on #140\x1b[?25h\x1b[9;15H",
	))

	if st := tr.state(); !st.active() {
		t.Fatalf("composer active = false for non-empty prompt, want true (state: %+v)", st)
	}
}

func TestMuseComposerTrackerRejectsHiddenCursor(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)
	tr.feed([]byte(
		"\x1b[30;1H\x1b[38;2;90;160;255;49m\xe2\x9f\xa9 \x1b[?25l\x1b[30;3H",
	))

	if st := tr.state(); st.active() {
		t.Fatalf("composer active = true with hidden cursor (state: %+v)", st)
	}
}

func TestMuseComposerTrackerRejectsVisibleCursorWithoutPrompt(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)
	tr.feed([]byte("\x1b[?25h\x1b[30;3H"))

	if st := tr.state(); st.active() {
		t.Fatalf("composer active = true for cursor without prompt (state: %+v)", st)
	}
}

func TestMuseComposerTrackerRejectsPromptAwayFromCursor(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)
	tr.feed([]byte(
		"\x1b[30;1H\x1b[38;2;90;160;255;49m\xe2\x9f\xa9 \x1b[?25h\x1b[20;3H",
	))

	if st := tr.state(); st.active() {
		t.Fatalf("composer active = true for prompt away from cursor (state: %+v)", st)
	}
}

func TestMuseComposerTrackerHandlesSplitPrompt(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)

	// Split UTF-8 prompt across chunks: e2 | 9f a9
	tr.feed([]byte("\x1b[30;1H\x1b[38;2;90;160;255;49m\xe2"))
	tr.feed([]byte("\x9f\xa9 \x1b[?25h\x1b[30;3H"))

	if st := tr.state(); !st.active() {
		t.Fatalf("composer active = false after split prompt, want true (state: %+v)", st)
	}
}

func TestMuseComposerTrackerHandlesSplitPromptTwoByte(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)

	tr.feed([]byte("\x1b[30;1H\x1b[38;2;90;160;255;49m\xe2\x9f"))
	tr.feed([]byte("\xa9 \x1b[?25h\x1b[30;3H"))

	if st := tr.state(); !st.active() {
		t.Fatalf("composer active = false after 2-byte split, want true (state: %+v)", st)
	}
}

func TestMuseComposerTrackerRejectsUnterminatedPrompt(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)
	tr.feed([]byte("\x1b[30;1H\xe2"))
	tr.feed([]byte("\x1b[?25h\x1b[30;3H"))

	if st := tr.state(); st.active() {
		t.Fatalf("composer active = true after unterminated prompt (state: %+v)", st)
	}
}

func TestMuseComposerTrackerClearsEraseDisplay(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)
	tr.feed([]byte(
		"\x1b[30;1H\x1b[38;2;90;160;255;49m\xe2\x9f\xa9 \x1b[?25h\x1b[30;3H",
	))

	tr.feed([]byte("\x1b[2J\x1b[?25h\x1b[30;3H"))

	if st := tr.state(); st.active() {
		t.Fatalf("composer active = true after erase display (state: %+v)", st)
	}
}

func TestMuseComposerTrackerConcurrentFeedAndState(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			tr.feed([]byte(
				"\x1b[30;1H\x1b[38;2;90;160;255;49m\xe2\x9f\xa9 \x1b[?25h\x1b[30;3H",
			))
			tr.feed([]byte("\x1b[2J\x1b[?25h\x1b[30;3H"))
		}
	}()

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			_ = tr.state().active()
		}
	}()

	close(start)
	wg.Wait()
}

func TestMuseComposerTrackerResizeClearsImpossibleState(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)
	tr.feed([]byte(
		"\x1b[30;1H\x1b[38;2;90;160;255;49m\xe2\x9f\xa9 \x1b[?25h\x1b[30;3H",
	))
	tr.resize(0, 0)

	if st := tr.state(); st.active() {
		t.Fatalf("composer active = true after zero resize (state: %+v)", st)
	}
}

func TestMuseComposerTrackerIgnoresDiffBG(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)
	// Diff view uses 33;58;43 BG with no prompt — should not activate.
	tr.feed([]byte(
		"\x1b[30;1H\x1b[38;2;103;108;116;48;2;33;58;43m   1 \x1b[?25h\x1b[30;3H",
	))
	if st := tr.state(); st.active() {
		t.Fatalf("composer active = true for diff BG without prompt (state: %+v)", st)
	}
}

func TestMuseComposerTrackerSnapshotDifferentialOracle(t *testing.T) {
	literal, err := os.ReadFile("testdata/tty/muse/0.1.0-R708.1/composer.raw")
	if err != nil {
		t.Fatalf("read literal Muse fixture: %v", err)
	}
	qualified := "\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ \x1b[9;1H\x1b[2m────\x1b[?25h\x1b[8;3H"
	qualifiedNonEmpty := "\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ work on #140" +
		"\x1b[9;1H\x1b[2m────\x1b[?25h\x1b[8;15H"
	tests := []struct {
		name   string
		stream []byte
		want   bool
	}{
		{"literal captured composer", literal, true},
		{"generated captured signature", []byte(qualified), true},
		{"qualified non-empty prompt", []byte(qualifiedNonEmpty), true},
		{"hidden cursor", []byte(qualified + "\x1b[?25l"), false},
		{"bare old U+203A glyph", []byte("\x1b[8;1H› \x1b[?25h\x1b[8;3H"), false},
		{"unqualified U+27E9 glyph", []byte("\x1b[8;1H⟩ \x1b[?25h\x1b[8;3H"), true},
		{"stale prompt mutation", []byte(qualified + "\x1b[8;1H\x1b[22mnot a prompt\x1b[?25h\x1b[8;3H"), true},
		{
			"one local weak glyph plus distant complete evidence",
			[]byte("\x1b[2;1H\x1b[2m────\x1b[3;1H\x1b[22m⟩ \x1b[4;1H\x1b[2m────" +
				"\x1b[20;1H\x1b[22m⟩ \x1b[?25h\x1b[20;3H"),
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tracker := newMuseComposerTracker()
			tracker.resize(38, 120)
			tracker.feed(test.stream)
			if got := tracker.state().active(); got != test.want {
				t.Fatalf("legacy active = %t, want frozen %t", got, test.want)
			}
		})
	}
}
