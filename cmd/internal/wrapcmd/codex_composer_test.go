package wrapcmd

import (
	"sync"
	"testing"
)

func TestCodexComposerTrackerDetectsObservedBottomComposer(t *testing.T) {
	tr := newCodexComposerTracker()
	tr.resize(38, 120)

	tr.feed([]byte(
		"\x1b[35;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[36;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[37;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[38;1H\x1b[49m\x1b[K" +
			"\x1b[?25h\x1b[36;3H",
	))

	if st := tr.state(); !st.active() {
		t.Fatalf("composer active = false, want true (state: %+v)", st)
	}
}

func TestCodexComposerTrackerRejectsHiddenCursor(t *testing.T) {
	tr := newCodexComposerTracker()
	tr.resize(38, 120)
	tr.feed([]byte(
		"\x1b[35;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[36;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[37;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[?25l\x1b[36;3H",
	))

	if st := tr.state(); st.active() {
		t.Fatalf("composer active = true with hidden cursor (state: %+v)", st)
	}
}

func TestCodexComposerTrackerRejectsNonBottomPaint(t *testing.T) {
	tr := newCodexComposerTracker()
	tr.resize(38, 120)
	tr.feed([]byte(
		"\x1b[10;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[11;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[?25h\x1b[11;3H",
	))

	if st := tr.state(); st.active() {
		t.Fatalf("composer active = true for non-bottom paint (state: %+v)", st)
	}
}

func TestCodexComposerTrackerHandlesSplitCSI(t *testing.T) {
	tr := newCodexComposerTracker()
	tr.resize(38, 120)

	tr.feed([]byte("\x1b[35;1H\x1b[48;2;57;"))
	tr.feed([]byte("57;57m\x1b[K\x1b[36;1H\x1b[48;2;57;57;57m\x1b[K"))
	tr.feed([]byte("\x1b[37;1H\x1b[48;2;57;57;57m\x1b[K\x1b[?25h\x1b[36;3H"))

	if st := tr.state(); !st.active() {
		t.Fatalf("composer active = false after split CSI, want true (state: %+v)", st)
	}
}

func TestCodexComposerTrackerClearsBottomRowsRepaintedWithoutComposerBG(t *testing.T) {
	tr := newCodexComposerTracker()
	tr.resize(38, 120)
	tr.feed([]byte(
		"\x1b[35;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[36;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[37;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[?25h\x1b[36;3H",
	))
	tr.feed([]byte(
		"\x1b[35;1H\x1b[49m\x1b[K" +
			"\x1b[36;1H\x1b[49m\x1b[K" +
			"\x1b[37;1H\x1b[49m\x1b[K" +
			"\x1b[?25h\x1b[36;3H",
	))

	if st := tr.state(); st.active() {
		t.Fatalf("composer active = true after bottom rows cleared (state: %+v)", st)
	}
}

func TestCodexComposerTrackerClearsEraseDisplay(t *testing.T) {
	tr := newCodexComposerTracker()
	tr.resize(38, 120)
	tr.feed([]byte(
		"\x1b[35;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[36;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[37;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[?25h\x1b[36;3H",
	))

	tr.feed([]byte("\x1b[2J\x1b[?25h\x1b[36;3H"))

	if st := tr.state(); st.active() {
		t.Fatalf("composer active = true after erase display (state: %+v)", st)
	}
}

func TestCodexComposerTrackerConcurrentFeedAndState(t *testing.T) {
	tr := newCodexComposerTracker()
	tr.resize(38, 120)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			tr.feed([]byte(
				"\x1b[35;1H\x1b[48;2;57;57;57m\x1b[K" +
					"\x1b[36;1H\x1b[48;2;57;57;57m\x1b[K" +
					"\x1b[37;1H\x1b[48;2;57;57;57m\x1b[K" +
					"\x1b[?25h\x1b[36;3H",
			))
			tr.feed([]byte("\x1b[2J\x1b[?25h\x1b[36;3H"))
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

func TestCodexComposerTrackerResizeClearsImpossibleState(t *testing.T) {
	tr := newCodexComposerTracker()
	tr.resize(38, 120)
	tr.feed([]byte(
		"\x1b[35;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[36;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[37;1H\x1b[48;2;57;57;57m\x1b[K" +
			"\x1b[?25h\x1b[36;3H",
	))
	tr.resize(0, 0)

	if st := tr.state(); st.active() {
		t.Fatalf("composer active = true after zero resize (state: %+v)", st)
	}
}
