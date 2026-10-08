package couchtty

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestParseOSC4Reply(t *testing.T) {
	cases := []struct {
		raw   string
		index int
		rgba  color.RGBA
		ok    bool
	}{
		{"\x1b]4;1;rgb:cccc/0000/0000\x1b\\", 1, color.RGBA{0xcc, 0, 0, 0xff}, true},
		{"\x1b]4;15;rgb:ff/ff/ff\x07", 15, color.RGBA{0xff, 0xff, 0xff, 0xff}, true},
		{"\x1b]4;0;rgb:1e1e/1f1f/2929\x1b\\", 0, color.RGBA{0x1e, 0x1f, 0x29, 0xff}, true},
		{"\x1b]4;3;rgb:f/8/0\x07", 3, color.RGBA{0xff, 0x88, 0, 0xff}, true},
		{"\x1b]4;16;rgb:ffff/ffff/ffff\x07", 0, color.RGBA{}, false}, // only 0–15
		{"\x1b]4;1;?\x07", 0, color.RGBA{}, false},
		{"\x1b]4;1;rgb:zz/00/00\x07", 0, color.RGBA{}, false},
		{"\x1b]4;1;rgb:ff/00\x07", 0, color.RGBA{}, false},
		{"\x1b]4;1;rgb:fffff/0/0\x07", 0, color.RGBA{}, false},
		{"\x1b]10;rgb:ffff/ffff/ffff\x07", 0, color.RGBA{}, false},
		{"\x1b]4;-1;rgb:ff/ff/ff\x07", 0, color.RGBA{}, false},
	}
	for _, c := range cases {
		index, rgba, ok := parseOSC4Reply(c.raw)
		if ok != c.ok || (ok && (index != c.index || rgba != c.rgba)) {
			t.Errorf("parseOSC4Reply(%q) = %d %v %v, want %d %v %v", c.raw, index, rgba, ok, c.index, c.rgba, c.ok)
		}
	}
}

func TestPaletteQueryAsksForAllSixteenColours(t *testing.T) {
	for i := range 16 {
		want := fmt.Sprintf("\x1b]4;%d;?\x1b\\", i)
		if !strings.Contains(paletteQuery, want) {
			t.Fatalf("paletteQuery lacks %q", want)
		}
	}
}

// OSC 4 replies arrive as unknown OSC events, classified as replies by the
// terminal decoder (its input_test pins that); Couch keeps them for the
// broadcast theme and never forwards them to a child.
func TestOSC4ReplyFeedsBroadcastThemeNotChild(t *testing.T) {
	f := newFixture(t, 24, 80)
	before := len(f.child.Writes())
	reply := uv.UnknownOscEvent("\x1b]4;2;rgb:00/ff/00\x1b\\")
	f.con.routeInputEvent(terminal.InputEvent{Event: reply, Reply: true, Raw: []byte(reply)})
	if theme := f.con.broadcastTheme(); theme.ANSI[2] != "#00ff00" {
		t.Fatalf("theme ANSI[2] = %q", theme.ANSI[2])
	}
	if len(f.child.Writes()) != before {
		t.Fatal("the reply reached the child")
	}
}
