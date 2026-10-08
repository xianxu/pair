package terminal

import (
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ttyio"
)

// rowDiffFrames feeds base, then edit, to one endpoint and composes each
// publication under a one-row chrome, as the presenter does.
func rowDiffFrames(t *testing.T, cols, rows int, base, edit string) (Publication, Frame, Publication, Frame) {
	t.Helper()
	e, err := NewEndpoint("rowdiff", Geometry{cols, rows}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	chrome, _ := StyledRows("CHR", cols, 1)
	publish := func(wire string) (Publication, Frame) {
		e.Feed([]byte(wire), time.Time{})
		p, err := e.Publication(time.Time{})
		if err != nil {
			t.Fatal(err)
		}
		f, err := Compose(p.Frame, Geometry{cols, rows + 1}, chrome)
		if err != nil {
			t.Fatal(err)
		}
		return p, f
	}
	p1, f1 := publish(base)
	p2, f2 := publish(edit)
	return p1, f1, p2, f2
}

// rowDiffText is random line content: plain, wide and styled runs, sometimes
// longer than the screen is wide.
func rowDiffText(r *rand.Rand, cols int) string {
	var b strings.Builder
	n := r.Intn(cols*2 + 1)
	for width := 0; width < n; {
		switch r.Intn(8) {
		case 0:
			b.WriteString("界")
			width += 2
		case 1:
			b.WriteString("\x1b[1;41mS\x1b[0m")
			width++
		case 2:
			b.WriteString(" ")
			width++
		default:
			b.WriteByte(byte('a' + r.Intn(26)))
			width++
		}
	}
	return b.String()
}

// TestHistoryRowDiffEqualsFullRebuild: over seeded random screens and edits
// that add no history, the parent state after the incremental wire (the row
// diff where it applies, the full rebuild where its gate refuses) equals a
// full rebuild of the new frame, under the xterm oracle (pair#409).
func TestHistoryRowDiffEqualsFullRebuild(t *testing.T) {
	const cols, rows, seeds = 10, 6, 80
	diffed, refused := 0, 0
	for seed := int64(1); seed <= seeds; seed++ {
		r := rand.New(rand.NewSource(seed))
		var base strings.Builder
		for i := 0; i < rows+r.Intn(4); i++ {
			if i > 0 {
				base.WriteString("\r\n")
			}
			base.WriteString(rowDiffText(r, cols))
		}
		// Edits stay above the bottom row and never scroll: CUP, then rewrite,
		// recolour or clear part of a row, or only move the cursor.
		var edit strings.Builder
		for i := 0; i < 1+r.Intn(3); i++ {
			y, x := 1+r.Intn(rows-1), 1+r.Intn(cols)
			fmt.Fprintf(&edit, "\x1b[%d;1H", y)
			switch r.Intn(5) {
			case 0:
				edit.WriteString("\x1b[2K")
				text := rowDiffText(r, cols)
				if y == rows-1 && len(text) > cols { // the row above the bottom may wrap into it, not past it
					text = text[:cols]
				}
				edit.WriteString(text)
			case 1:
				fmt.Fprintf(&edit, "\x1b[%d;%dH\x1b[44m\x1b[K\x1b[0m", y, x)
			case 2:
				fmt.Fprintf(&edit, "\x1b[%d;%dH\x1b[K", y, x)
			case 3:
				fmt.Fprintf(&edit, "\x1b[%d;%dH%c", y, x, 'A'+r.Intn(26))
			default:
				fmt.Fprintf(&edit, "\x1b[%d;%dH", y, x)
			}
		}
		p1, f1, p2, f2 := rowDiffFrames(t, cols, rows, base.String(), edit.String())
		if p2.History.Cursor != p1.History.Cursor {
			continue // an edit that pushed history is the full path's case, not this one's
		}
		first, state := renderHistoryBytes(t, Frame{}, f1, p1.History, HistoryState{})
		plan, err := RenderWithHistory(f1, f2, p2.History, state)
		if err != nil {
			t.Fatal(err)
		}
		if plan.diff {
			diffed++
		} else {
			refused++
		}
		incremental, _ := renderHistoryBytes(t, f1, f2, p2.History, state)
		// The reference is the full rebuild of the new frame issued after the
		// same first paint (an empty previous frame forces the reset), so both
		// sides share the parent's mode history and differ only in the paint.
		rebuilt, _ := renderHistoryBytes(t, Frame{}, f2, p2.History, state)
		got := settledLinks(runHistoryOracle(t, cols, rows+1, string(first), string(incremental))[1])
		want := settledLinks(runHistoryOracle(t, cols, rows+1, string(first), string(rebuilt))[1])
		if plan.diff && os.Getenv("PAIR_TERMINAL_NATIVE") == "1" {
			// Zellij keeps row provenance xterm does not show; it is the parent
			// of pair's own presenter, so a row diff must agree there too.
			if native, rebuiltNative := nativeHistoryDump(t, cols, rows+1, string(first)+string(incremental)), nativeHistoryDump(t, cols, rows+1, string(first)+string(rebuilt)); native != rebuiltNative {
				t.Fatalf("seed %d: zellij state differs from the full rebuild\nbase %q\nedit %q\ngot  %q\nwant %q", seed, base.String(), edit.String(), native, rebuiltNative)
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("seed %d (diff=%t): parent state differs from the full rebuild in %v\nbase %q\nedit %q\nincremental %q",
				seed, plan.diff, oracleDifferences(got, want), base.String(), edit.String(), incremental)
		}
	}
	t.Logf("%d row-diffed, %d refused (full rebuild)", diffed, refused)
	if diffed < seeds/4 || refused == 0 {
		t.Fatalf("generator exercised %d diffs and %d refusals; it must cover both", diffed, refused)
	}
}

// oracleDifferences names the fields two oracle screens differ in, with the
// differing rows, so a failing seed is readable.
func oracleDifferences(got, want historyOracleScreen) []string {
	var out []string
	if got.X != want.X || got.Y != want.Y {
		out = append(out, fmt.Sprintf("cursor %d,%d want %d,%d", got.X, got.Y, want.X, want.Y))
	}
	for y := range want.Lines {
		if y >= len(got.Lines) || got.Lines[y] != want.Lines[y] {
			out = append(out, fmt.Sprintf("line %d", y))
		} else if y < len(got.Cells) && y < len(want.Cells) && !reflect.DeepEqual(got.Cells[y], want.Cells[y]) {
			for x := range want.Cells[y] {
				if x < len(got.Cells[y]) && got.Cells[y][x] != want.Cells[y][x] {
					out = append(out, fmt.Sprintf("cell %d,%d %+v want %+v", x, y, got.Cells[y][x], want.Cells[y][x]))
					break
				}
			}
		}
	}
	if !reflect.DeepEqual(got.Wraps, want.Wraps) {
		out = append(out, fmt.Sprintf("wraps %v want %v", got.Wraps, want.Wraps))
	}
	if !reflect.DeepEqual(got.History, want.History) {
		out = append(out, fmt.Sprintf("history (%d rows, want %d)", len(got.History), len(want.History)))
	}
	if !reflect.DeepEqual(got.Modes, want.Modes) || got.CursorStyle != want.CursorStyle || !reflect.DeepEqual(got.Links, want.Links) {
		out = append(out, fmt.Sprintf("modes %v want %v; cursor style %q/%t want %q/%t; links %v want %v", got.Modes, want.Modes, got.CursorStyle, got.CursorBlink, want.CursorStyle, want.CursorBlink, got.Links, want.Links))
	}
	return out
}

// settledLinks keeps only the hyperlinks that are set. The oracle lists one
// entry per buffer line, and a rebuild's ED3 leaves the buffer a line longer
// than an incremental paint does, with no link on it: a count, not state.
func settledLinks(s historyOracleScreen) historyOracleScreen {
	var links []string
	for _, l := range s.Links {
		if l != "" && l != ";" { // ";" is an empty params;url pair
			links = append(links, l)
		}
	}
	s.Links = links
	return s
}

// TestChangedPlainRowsGate: what the row diff takes and what it refuses. The
// generative test proves equality where it applies; this pins how wide that is.
func TestChangedPlainRowsGate(t *testing.T) {
	base := "one\r\ntwo\r\nlong line wraps here\r\nfour"
	_, f, _, cursorOnly := rowDiffFrames(t, 10, 6, base, "\x1b[1;1H")
	if rows, ok := changedPlainRows(f, cursorOnly); !ok || len(rows) != 0 {
		t.Fatalf("cursor move: %v %t", rows, ok)
	}
	_, f, _, plainEdit := rowDiffFrames(t, 10, 6, base, "\x1b[2;1H\x1b[2KTWO!")
	if rows, ok := changedPlainRows(f, plainEdit); !ok || !reflect.DeepEqual(rows, []int{1}) {
		t.Fatalf("plain row edit: %v %t", rows, ok)
	}
	// Row 3 continues row 2's long line: editing it touches a wrapped row.
	_, f, _, wrappedEdit := rowDiffFrames(t, 10, 6, base, "\x1b[4;3HX")
	if _, ok := changedPlainRows(f, wrappedEdit); ok {
		t.Fatal("a wrapped row was row-diffed")
	}
	// Rewriting a short row long enough to wrap flips the next row's flag.
	_, f, _, wraps := rowDiffFrames(t, 10, 6, base, "\x1b[2;1H\x1b[2Ktwo is now long")
	if _, ok := changedPlainRows(f, wraps); ok {
		t.Fatal("a wrap-flag change was row-diffed")
	}
	// Repainting the head of a soft-wrapped line leaves the continuation's own
	// flag alone, so it is row-diffed (both oracles agree; see the
	// generative test).
	_, f, _, head := rowDiffFrames(t, 10, 6, base, "\x1b[3;1HL")
	if rows, ok := changedPlainRows(f, head); !ok || !reflect.DeepEqual(rows, []int{2}) {
		t.Fatalf("head of a wrapped line: %v %t", rows, ok)
	}
	other := plainEdit
	other.EndpointID = "other"
	if _, ok := changedPlainRows(f, other); ok {
		t.Fatal("an endpoint switch was row-diffed")
	}
	resized := plainEdit
	resized.Geometry.Cols++
	if _, ok := changedPlainRows(f, resized); ok {
		t.Fatal("a resize was row-diffed")
	}
}

// TestHistoryEmitSpinnerTickWritesOnlyTheChangedRow: the measured case in
// pair#409. An agent's elapsed-time counter ticks on a 191×53 screen and nothing
// else changes. The full rebuild repaints the whole screen; the row diff
// writes one row.
func TestHistoryEmitSpinnerTickWritesOnlyTheChangedRow(t *testing.T) {
	const cols, rows = 191, 52
	var base strings.Builder
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&base, "\x1b[2mline %d of the agent's earlier output, long enough to look like a real transcript row\x1b[0m\r\n", i)
	}
	base.WriteString("✽ Generating… (53s · ↓ 1.3k tokens)")
	p1, f1, p2, f2 := rowDiffFrames(t, cols, rows, base.String(), "\r\x1b[2K✽ Generating… (54s · ↓ 1.3k tokens)")
	_, state := renderHistoryBytes(t, Frame{}, f1, p1.History, HistoryState{})
	tick, _ := renderHistoryBytes(t, f1, f2, p2.History, state)
	full, _ := renderHistoryBytes(t, Frame{}, f2, p2.History, state)
	t.Logf("one spinner tick: %d bytes (full rebuild: %d bytes)", len(tick), len(full))
	if len(tick) > 1024 || len(full) < 10*len(tick) {
		t.Fatalf("tick wrote %d bytes, full rebuild %d", len(tick), len(full))
	}
}
