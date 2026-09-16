package vt

import (
	"fmt"
	"io"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func assertCoherentCells(t *testing.T, e *Emulator) {
	t.Helper()
	for y := 0; y < e.Height(); y++ {
		for x := 0; x < e.Width(); x++ {
			c := e.CellAt(x, y)
			if c.Content != "" {
				first, width := ansi.FirstGraphemeCluster(c.Content, ansi.GraphemeWidth)
				n := len(first)
				if c.Width == 0 || n != len(c.Content) || width != c.Width {
					t.Fatalf("invalid cell (%d,%d): %#v decoded width=%d bytes=%d", x, y, c, width, n)
				}
			}
			if c.Width == 2 && (x+1 >= e.Width() || e.CellAt(x+1, y).Width != 0 || e.CellAt(x+1, y).Content != "") {
				t.Fatalf("broken wide cell (%d,%d): %#v", x, y, c)
			}
		}
	}
}

func TestPairOrphanZeroWidthAfterBoundary(t *testing.T) {
	for _, tc := range []struct{ name, prefix, after string }{
		{"initial", "", "Z"},
		{"SGR", "A\x1b[31m", "AZ"},
		{"column", "A\x1b[2G", "AZ"},
		{"wide-SGR", "界\x1b[31m", "界Z"},
		{"wide-interior", "界\x1b[2G", " Z"},
		{"variation-base", "♥\x1b[31m", "♥Z"},
		{"carriage-return", "A\r", "Z"},
		{"erase", "A\x1b[2J", " Z"},
	} {
		for _, mark := range []string{"\u0301", "\u200d", "\ufe0f", "\u0301\u200d\ufe0f"} {
			wire := tc.prefix + mark
			t.Run(fmt.Sprintf("%s/%U", tc.name, []rune(mark)), func(t *testing.T) {
				for split := 0; split <= len(wire); split++ {
					e := NewEmulator(8, 2)
					e.SetReplyWriter(io.Discard)
					e.WriteString(wire[:split])
					assertCoherentCells(t, e)
					e.WriteString(wire[split:])
					assertCoherentCells(t, e)
					e.WriteString("Z")
					assertCoherentCells(t, e)
					if got := e.String(); got != tc.after+"\n" {
						t.Fatalf("split %d got %q want %q", split, got, tc.after)
					}
					e.Close()
				}
			})
		}
	}
}

func TestPairZeroWidthDoesNotTriggerWrap(t *testing.T) {
	for _, mark := range []string{"\u0301", "\u200d", "\ufe0f"} {
		wire := "ABCDEFGH\x1b[31m" + mark
		for split := 0; split <= len(wire); split++ {
			e := NewEmulator(4, 2)
			e.SetReplyWriter(io.Discard)
			e.WriteString(wire[:split])
			e.WriteString(wire[split:])
			assertCoherentCells(t, e)
			if got := e.String(); got != "ABCD\nEFGH" {
				t.Fatalf("mark wrapped at split %d: %q", split, got)
			}
			e.WriteString("Z")
			assertCoherentCells(t, e)
			if got := e.String(); got != "EFGH\nZ" {
				t.Fatalf("continued output split %d: %q", split, got)
			}
			e.Close()
		}
	}
}

func TestPairZeroWidthStillExtendsOpenCluster(t *testing.T) {
	for _, cluster := range []string{"A\u0301", "👩\u200d💻", "❤\ufe0f"} {
		for split := 0; split <= len(cluster); split++ {
			e := NewEmulator(8, 2)
			e.SetReplyWriter(io.Discard)
			e.WriteString(cluster[:split])
			assertCoherentCells(t, e)
			e.WriteString(cluster[split:])
			assertCoherentCells(t, e)
			e.WriteString("Z")
			if got := e.CellAt(0, 0).Content; got != cluster {
				t.Fatalf("split %d got %q want %q", split, got, cluster)
			}
			assertCoherentCells(t, e)
			e.Close()
		}
	}
}

func TestPairZeroWidthControlsSplitJoiner(t *testing.T) {
	for _, control := range []string{"\x1b[31m", "\x1b[3G"} {
		wire := "👩" + control + "\u200d💻Z"
		for split := 0; split <= len(wire); split++ {
			e := NewEmulator(8, 2)
			e.SetReplyWriter(io.Discard)
			e.WriteString(wire[:split])
			assertCoherentCells(t, e)
			e.WriteString(wire[split:])
			assertCoherentCells(t, e)
			if e.CellAt(0, 0).Content != "👩" || e.CellAt(2, 0).Content != "💻" || e.CellAt(4, 0).Content != "Z" {
				t.Fatalf("split %d unexpectedly joined across control: %q", split, e.String())
			}
			e.Close()
		}
	}
}
