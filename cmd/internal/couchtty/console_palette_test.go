package couchtty

import (
	"bytes"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

const (
	replyWhiteBG = "\x1b]11;rgb:ffff/ffff/ffff\x1b\\"
	replyBlackFG = "\x1b]10;rgb:0000/0000/0000\x1b\\"
)

// The query goes out exactly once, and before anything else couch writes: no
// frame can be split by it.
func TestRunAsksForTheTerminalPaletteBeforeTheFirstFrame(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "a first frame", func() bool { return len(f.host.Written()) > len(paletteQuery) })
	written := f.host.Written()
	if !strings.HasPrefix(written, paletteQuery) {
		t.Fatalf("first bytes written = %q, want the palette query first", written[:min(len(written), 40)])
	}
	if n := strings.Count(written, paletteQuery); n != 1 {
		t.Fatalf("palette query sent %d times, want 1", n)
	}
}

func TestPaletteRepliesMakeThePaletteKnownAndNeverReachTheChild(t *testing.T) {
	f := newFixture(t, 24, 80)
	_, _ = f.stdin.Write([]byte(replyWhiteBG))
	waitFor(t, "the background reply", func() bool { return f.con.menuSnapshot().Palette.BG == color.RGBA{0xff, 0xff, 0xff, 0xff} })
	if f.con.menuSnapshot().Palette.Known {
		t.Fatal("palette known after only the background reply")
	}
	_, _ = f.stdin.Write([]byte(replyBlackFG))
	waitFor(t, "both replies", func() bool { return f.con.menuSnapshot().Palette.Known })
	if got := f.con.menuSnapshot().Palette; got.FG != (color.RGBA{0, 0, 0, 0xff}) {
		t.Fatalf("foreground = %v, want black", got.FG)
	}
	// Light scheme: an idle chip now fades toward white.
	f.con.mu.Lock()
	model := f.con.statusModelLocked()
	f.con.mu.Unlock()
	if got := FadeStyle(model.Palette, IdleDay, baseDefault); got == "\x1b[90m" || got == "" {
		t.Fatalf("a known light palette still falls back: %q", got)
	}
	for _, write := range f.child.Writes() {
		if bytes.Contains(write, []byte("]1")) {
			t.Fatalf("a colour reply reached the child: %q", write)
		}
	}
}

func TestAMalformedPaletteReplyLeavesThePaletteUnknown(t *testing.T) {
	f := newFixture(t, 24, 80)
	_, _ = f.stdin.Write([]byte("\x1b]11;not-a-colour\x1b\\" + replyBlackFG))
	waitFor(t, "the good reply", func() bool { return f.con.menuSnapshot().Palette.FG == color.RGBA{0, 0, 0, 0xff} })
	time.Sleep(20 * time.Millisecond)
	if f.con.menuSnapshot().Palette.Known {
		t.Fatal("a malformed background reply counted as an answer")
	}
}

// The menu state is built lazily on first attach; a palette recorded before
// that must survive it.
func TestEnsureMenuKeepsAnEarlierPalette(t *testing.T) {
	c := &Console{}
	c.SetColorModes(true, false)
	c.menu.Palette.FG, c.menu.Palette.Known = color.RGBA{1, 2, 3, 0xff}, true
	c.ensureMenuLocked(couchcore.ThreadAddress{})
	if got := c.menu.Palette; !got.Known || !got.TrueColor || got.FG != (color.RGBA{1, 2, 3, 0xff}) {
		t.Fatalf("palette after the lazy build = %+v", got)
	}
	if !c.menuReady || c.menu.Notice.Text == "" {
		t.Fatal("ensureMenuLocked did not build the menu")
	}
}
