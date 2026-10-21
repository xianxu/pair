package couchtty

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	qrcode "github.com/skip2/go-qrcode"
)

func TestRemoteLinkQRPreservesEncodedModulesAndQuietZone(t *testing.T) {
	link := "https://remote.example.com/remote/0123456789abcdefghijklmnopqrstuvwxyzABCDEFG/"
	code, err := qrcode.New(link, qrcode.Medium)
	if err != nil {
		t.Fatal(err)
	}
	want := code.Bitmap()
	size := len(want)
	rows := remoteLinkQR(link, size, (size+1)/2)
	if len(rows) != (size+1)/2 {
		t.Fatalf("row count %d, want %d", len(rows), (size+1)/2)
	}
	for y, row := range rows {
		if !strings.HasPrefix(row, "\x1b[38;2;0;0;0;48;2;255;255;255m") || !strings.HasSuffix(row, "\x1b[0m") {
			t.Fatalf("row %d does not explicitly set and restore black/white colors", y)
		}
		if got := ansi.StringWidth(row); got != size {
			t.Fatalf("row %d width %d, want %d", y, got, size)
		}
		cells := []rune(ansi.Strip(row))
		for x, cell := range cells {
			var top, bottom bool
			switch cell {
			case ' ':
			case '▀':
				top = true
			case '▄':
				bottom = true
			case '█':
				top, bottom = true, true
			default:
				t.Fatalf("unexpected module glyph %q", cell)
			}
			if top != want[2*y][x] {
				t.Fatalf("top module (%d,%d) changed", x, 2*y)
			}
			lower := 2*y+1 < size && want[2*y+1][x]
			if bottom != lower {
				t.Fatalf("bottom module (%d,%d) changed (including odd final row)", x, 2*y+1)
			}
		}
	}
	// The encoder's default four-module quiet zone remains white on every side.
	for y, row := range want {
		for x, black := range row {
			if (x < 4 || y < 4 || x >= size-4 || y >= size-4) && black {
				t.Fatalf("quiet zone lost at (%d,%d)", x, y)
			}
		}
	}
	if got := remoteLinkQR(link, size-1, (size+1)/2); got != nil {
		t.Fatal("must fall back to text when one column short")
	}
	if got := remoteLinkQR(link, size, (size+1)/2-1); got != nil {
		t.Fatal("must fall back to text when one row short")
	}
}

func TestRemoteLinkQRRejectsInvalidOrUnboundedInputs(t *testing.T) {
	for _, link := range []string{"", "http://remote.example.com/", "https://user:secret@remote.example.com/", "https:///missing", "https://remote.example.com/#fragment", "https://remote.example.com/\n", strings.Repeat("x", 1025)} {
		if got := remoteLinkQR(link, 1000, 1000); got != nil {
			t.Fatalf("accepted invalid link %q", link)
		}
	}
	link := "https://remote.example.com/"
	for _, geometry := range [][2]int{{0, 100}, {100, 0}, {-1, 100}, {100, -1}, {28, 100}, {100, 14}} {
		if got := remoteLinkQR(link, geometry[0], geometry[1]); got != nil {
			t.Fatalf("accepted insufficient geometry %v", geometry)
		}
	}
	oversized := strings.Repeat("x", 1<<20)
	if n := testing.AllocsPerRun(100, func() { remoteLinkQR(oversized, 1000, 1000) }); n != 0 {
		t.Fatalf("oversized input allocated %g times", n)
	}
	if n := testing.AllocsPerRun(100, func() { remoteLinkQR(link, 1, 1) }); n != 0 {
		t.Fatalf("impossible geometry allocated %g times", n)
	}
}
