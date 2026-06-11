package main

import (
	"image/color"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

// trimReset strips the always-emitted trailing reset so each test's
// expected output can focus on the substantive payload.
func trimReset(s string) string {
	return strings.TrimSuffix(s, "\x1b[0m")
}

// cell is a tiny convenience constructor for tests; the real callers
// build cells via the emulator, never literal struct values.
func cell(content string, width int) uv.Cell {
	return uv.Cell{Content: content, Width: width}
}

// styledCell adds a fg color so we can test style-transition handling
// without dragging in the full attribute matrix.
func styledCell(content string, width int, fg color.Color) uv.Cell {
	c := cell(content, width)
	c.Style.Fg = fg
	return c
}

// styledCellBg sets a non-default background — an inverse-video / box-fill
// cell. A blank with a bg is "visible" in colored mode but invisible in plain
// mode (no bg is emitted there).
func styledCellBg(content string, width int, bg color.Color) uv.Cell {
	c := cell(content, width)
	c.Style.Bg = bg
	return c
}

func TestSerializeRow_Plain_NoSGR(t *testing.T) {
	line := uv.Line{styledCell("hi", 1, color.RGBA{R: 255, A: 255}), cell(" ", 1)}
	got := serializeRow(line, true)
	if got != "hi" {
		t.Fatalf("plain: want %q got %q", "hi", got)
	}
	if strings.Contains(got, "\x1b") {
		t.Fatalf("plain row must contain no escape: %q", got)
	}
}

func TestSerializeRow_Plain_TrimsTrailingDefaultBlanks(t *testing.T) {
	line := uv.Line{cell("a", 1), cell(" ", 1), cell(" ", 1)}
	if got := serializeRow(line, true); got != "a" {
		t.Fatalf("want %q got %q", "a", got)
	}
}

func TestSerializeRow_Plain_TrimsTrailingBgBlank(t *testing.T) {
	// A trailing blank visible ONLY via background. Plain emits no bg, so it's
	// just padding to trim; colored keeps it (it's visible).
	line := uv.Line{cell("a", 1), styledCellBg(" ", 1, color.RGBA{B: 255, A: 255})}
	if got := serializeRow(line, true); got != "a" {
		t.Fatalf("plain should trim bg-only blank: got %q", got)
	}
	if got := stripSGR(serializeRow(line, false)); got != "a " {
		t.Fatalf("colored should keep the visible bg-blank (want %q): got %q", "a ", got)
	}
}

func TestSerializeRow_Colored_StillResets(t *testing.T) {
	line := uv.Line{styledCell("hi", 1, color.RGBA{R: 255, A: 255})}
	if got := serializeRow(line, false); !strings.HasSuffix(got, "\x1b[0m") {
		t.Fatalf("colored row should still terminate with reset: %q", got)
	}
}

func TestSerializeRow_PlainAscii(t *testing.T) {
	line := uv.Line{cell("h", 1), cell("i", 1)}
	got := trimReset(serializeRow(line, false))
	if got != "hi" {
		t.Errorf("got %q, want %q", got, "hi")
	}
}

func TestSerializeRow_Empty(t *testing.T) {
	// All-blank cells (default-style spaces) should serialize to "".
	line := uv.Line{cell(" ", 1), cell(" ", 1), cell(" ", 1)}
	got := serializeRow(line, false)
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestSerializeRow_TrimsTrailingBlanks(t *testing.T) {
	// "hi" followed by trailing default-blank padding. The padding
	// should not survive into the rendered line — it's just terminal
	// pad, the viewer doesn't want to scroll past it.
	line := uv.Line{
		cell("h", 1), cell("i", 1),
		cell(" ", 1), cell(" ", 1), cell(" ", 1),
	}
	got := trimReset(serializeRow(line, false))
	if got != "hi" {
		t.Errorf("got %q, want %q", got, "hi")
	}
}

func TestSerializeRow_PreservesNonDefaultBgBlank(t *testing.T) {
	// A blank cell with a non-default background represents inverse-
	// video padding (e.g. status bars). It's visible content and must
	// not be trimmed.
	bgCell := cell(" ", 1)
	bgCell.Style.Bg = color.RGBA{R: 255, A: 255}
	line := uv.Line{cell("x", 1), bgCell, cell(" ", 1)}
	got := serializeRow(line, false)
	// The bg-blank cell is at index 1; trailing default-blank at index
	// 2 should still be trimmed. The output must contain a space for
	// the bg-blank cell.
	if !strings.Contains(got, " ") {
		t.Errorf("expected output to contain a space (for bg-blank cell); got %q", got)
	}
	if strings.Count(got, " ") != 1 {
		t.Errorf("expected exactly one space (bg-blank kept, trailing trimmed); got %q", got)
	}
}

func TestSerializeRow_WideGraphemeNoPhantomSpace(t *testing.T) {
	// The bug: ultraviolet stores 🔴 as a Width=2 cell *followed by*
	// a zero-value Cell{} placeholder (Width=0, Content=""). The
	// previous loop turned that placeholder into a literal ' ' write,
	// rendering 🔴 as "🔴 " with a stray column of whitespace after
	// every emoji. serializeRow must skip IsZero cells.
	line := uv.Line{
		cell("X", 1),
		cell("🔴", 2),
		{}, // zero-value continuation cell
		cell("Y", 1),
	}
	got := trimReset(serializeRow(line, false))
	want := "X🔴Y"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSerializeRow_WideGraphemeAtEnd(t *testing.T) {
	// Wide grapheme as the last visible glyph: the continuation cell
	// is the literal last index of the line, but it must not show up
	// in the output nor extend `last` such that a stray space gets
	// written.
	line := uv.Line{
		cell("X", 1),
		cell("🔴", 2),
		{}, // continuation
	}
	got := trimReset(serializeRow(line, false))
	want := "X🔴"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSerializeRow_StyleDiffEmittedOnTransition(t *testing.T) {
	// First two cells share a style (red), next two share a different
	// style (blue). The renderer should emit one SGR sequence on
	// entering red, one on switching to blue, and the final reset.
	red := color.RGBA{R: 255, A: 255}
	blue := color.RGBA{B: 255, A: 255}
	line := uv.Line{
		styledCell("a", 1, red), styledCell("b", 1, red),
		styledCell("c", 1, blue), styledCell("d", 1, blue),
	}
	got := serializeRow(line, false)
	if !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("expected trailing reset, got %q", got)
	}
	// At minimum we expect: SGR-into-red, "ab", SGR-into-blue, "cd",
	// reset. Count escape introducers.
	escs := strings.Count(got, "\x1b[")
	if escs < 3 {
		t.Errorf("expected ≥3 SGR sequences (red→blue→reset), got %d in %q", escs, got)
	}
	// The non-SGR payload should be exactly "abcd" in order.
	plain := stripSGR(got)
	if plain != "abcd" {
		t.Errorf("expected payload %q after stripping SGR, got %q (from %q)", "abcd", plain, got)
	}
}

func TestSerializeRow_TerminatingReset(t *testing.T) {
	// Every non-empty line ends with the full reset so subsequent
	// concatenated lines start from default style.
	line := uv.Line{cell("z", 1)}
	got := serializeRow(line, false)
	if !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("expected trailing \\x1b[0m, got %q", got)
	}
}

// stripSGR removes SGR (CSI … m) sequences only. Mirrors the regex used
// by pair-scrollback-open's overlay step.
func stripSGR(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) {
				c := s[j]
				if (c >= '0' && c <= '9') || c == ';' {
					j++
					continue
				}
				if c == 'm' {
					i = j + 1
					goto next
				}
				// non-SGR final byte; bail out and write through.
				break
			}
			// malformed — write through the ESC and move on
		}
		b.WriteByte(s[i])
		i++
	next:
	}
	return b.String()
}
