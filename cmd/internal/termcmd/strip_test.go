package termcmd

import (
	"io"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/textwidth"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// EVERY test here drives at least TWO tabs with a non-active one present.
//
// That is a standing requirement, not a style note: M2's Critical (BR-35)
// shipped because every test in that milestone drove a single active tab, so
// the whole active/inactive distinction was unobserved by the suite.

func TestActiveTabIsDistinguishableWithoutColour(t *testing.T) {
	r := RenderStrip(40, StripModel{
		Tabs:   []TabChip{{Name: "one"}, {Name: "two"}, {Name: "three"}},
		Active: 1,
	})
	plain := string(ansi.Strip([]byte(r.Body)))
	if !strings.Contains(plain, "[two]") {
		t.Fatalf("the active tab is not marked in PLAIN text: %q", plain)
	}
	for _, inactive := range []string{"[one]", "[three]"} {
		if strings.Contains(plain, inactive) {
			t.Fatalf("an inactive tab is marked like the active one: %q", plain)
		}
	}
	// Colour may decorate, but must not be the only signal: a colour-blind
	// operator, a mono terminal, and `ansi.Strip`ped logs all lose it.
	if plain == r.Body && strings.Count(plain, "[") != 1 {
		t.Fatalf("active marking is ambiguous in plain text: %q", plain)
	}
}

// Carried from #172's BR-32: spans are DISPLAY COLUMNS. A rune count puts every
// span after a wide character one column left of what was drawn -- and an
// all-ASCII suite stays green while it does, which is why this fixture is not
// ASCII.
func TestSpansAreDisplayColumnsNotRuneCounts(t *testing.T) {
	r := RenderStrip(40, StripModel{
		Tabs:   []TabChip{{Name: "日本語"}, {Name: "build"}},
		Active: 0,
	})
	if len(r.Spans) != 2 {
		t.Fatalf("got %d spans for 2 tabs: %+v", len(r.Spans), r.Spans)
	}
	plain := string(ansi.Strip([]byte(r.Body)))
	at := strings.Index(plain, "build")
	if at < 0 {
		t.Fatalf("the second tab was not drawn: %q", plain)
	}
	wantCol := textwidth.Width(plain[:at])
	if r.Spans[1].Start != wantCol {
		t.Fatalf("span start %d is a rune count; the column is %d (%q)",
			r.Spans[1].Start, wantCol, plain)
	}
	// And the wide tab's own span must be as wide as it drew.
	if got := r.Spans[0].End - r.Spans[0].Start; got != textwidth.Width("[日本語]") {
		t.Fatalf("wide tab span is %d columns, drew %d", got, textwidth.Width("[日本語]"))
	}
}

// Tab names come from the operator. An escape in one must not become an escape
// in our row -- the same policy as couch's status row, via the same package.
func TestATabNameCannotInjectEscapesOrControls(t *testing.T) {
	r := RenderStrip(40, StripModel{
		Tabs:   []TabChip{{Name: "a\x1b[31mred"}, {Name: "b\x0egarble"}},
		Active: 0,
	})
	for _, bad := range []string{"\x1b", "\x0e"} {
		if strings.Contains(r.Body, bad) {
			t.Fatalf("control byte %q from a tab name reached the row: %q", bad, r.Body)
		}
	}
	if !strings.Contains(r.Body, "red") || !strings.Contains(r.Body, "garble") {
		t.Fatalf("sanitising mangled the readable text: %q", r.Body)
	}
}

// A narrow pane must not silently drop the tab the operator is looking at.
func TestANarrowPaneKeepsTheActiveTabVisible(t *testing.T) {
	m := StripModel{
		Tabs:   []TabChip{{Name: "alpha"}, {Name: "bravo"}, {Name: "charlie"}},
		Active: 2,
	}
	for _, width := range []int{40, 20, 12, 9} {
		r := RenderStrip(width, m)
		plain := string(ansi.Strip([]byte(r.Body)))
		if textwidth.Width(plain) > width {
			t.Fatalf("width %d: drew %d columns: %q", width, textwidth.Width(plain), plain)
		}
		if !strings.Contains(plain, "charlie") && !strings.Contains(plain, "charli") &&
			!strings.Contains(plain, "char") {
			t.Fatalf("width %d dropped the ACTIVE tab entirely: %q", width, plain)
		}
	}
}

// Degenerate widths must not panic or produce a row that wraps.
func TestDegenerateWidthsDrawNothingRatherThanWrapping(t *testing.T) {
	for _, width := range []int{0, -1} {
		r := RenderStrip(width, StripModel{Tabs: []TabChip{{Name: "a"}, {Name: "b"}}})
		if r.Body != "" || len(r.Spans) != 0 {
			t.Fatalf("width %d drew %q with %d spans", width, r.Body, len(r.Spans))
		}
	}
}

// No tabs is a real state (the last one just closed) and must not panic.
func TestNoTabsRendersNothing(t *testing.T) {
	if r := RenderStrip(40, StripModel{}); r.Body != "" {
		t.Fatalf("an empty model drew %q", r.Body)
	}
}

// An out-of-range Active index must not panic or mark the wrong tab: it comes
// from a mutable mux whose tabs can close between render and paint.
func TestAnOutOfRangeActiveIndexMarksNothing(t *testing.T) {
	for _, active := range []int{-1, 2, 99} {
		r := RenderStrip(40, StripModel{
			Tabs:   []TabChip{{Name: "one"}, {Name: "two"}},
			Active: active,
		})
		plain := string(ansi.Strip([]byte(r.Body)))
		if strings.Contains(plain, "[") {
			t.Fatalf("active=%d marked a tab: %q", active, plain)
		}
		if !strings.Contains(plain, "one") || !strings.Contains(plain, "two") {
			t.Fatalf("active=%d dropped tabs: %q", active, plain)
		}
	}
}

// --- wiring: the strip on a live mux ----------------------------------------
//
// Every case below drives at least TWO tabs with a non-active one present.

func stripMux(t *testing.T) (*terminalMux, *writerRecorder) {
	t.Helper()
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	m.rows, m.cols = 24, 80
	m.tabs = append(m.tabs,
		&terminalTab{id: 1, name: "one"},
		&terminalTab{id: 2, name: "two"})
	m.active = 1
	go m.copyActiveOutput()
	return m, rec
}

// M3.5, ARCH-ORDER's most-likely-mishandled event: on a row-dirty batch the
// region is re-Reserved BEFORE the row is repainted.
//
// Not belt-and-braces. A child that reset margins dropped the region a moment
// ago, and painting into an unreserved screen puts the row where the child's
// content belongs -- so a repaint that emits only the row is worse than none.
func TestARowDirtyBatchReReservesBeforeRepainting(t *testing.T) {
	m, rec := stripMux(t)
	defer close(m.done)

	// A child emitting a margin reset: ptychild.Screen counts that as row-dirty.
	m.output <- ptyChunk{id: 2, data: []byte("\x1b[r"), rowDirty: true}
	m.drainForTest()

	got := rec.String()
	region := strings.Index(got, "\x1b[1;23r") // the region, rows 1..23 of 24
	row := strings.Index(got, "\x1b[24;1H")    // the reserved row
	if region < 0 {
		t.Fatalf("the region was not re-asserted after the child reset margins: %q", got)
	}
	if row < 0 {
		t.Fatalf("the row was not repainted: %q", got)
	}
	if region > row {
		t.Fatalf("the row was painted BEFORE the region was re-asserted: %q", got)
	}
	if !strings.Contains(got, "[two]") {
		t.Fatalf("the repaint did not carry the tab state: %q", got)
	}
}

// A takeover clears the screen; the row it cleared is ours to put back.
func TestATakeoverRepaintsTheStrip(t *testing.T) {
	m, rec := stripMux(t)
	defer close(m.done)

	m.redrawTab([]byte("replayed"))
	m.drainForTest()
	if !strings.Contains(rec.String(), "[two]") {
		t.Fatalf("the strip was not restored after a takeover: %q", rec.String())
	}
}

// The child is sized to the pane MINUS the row, so it cannot scroll onto it.
func TestTheChildIsSizedBelowTheStrip(t *testing.T) {
	m, _ := stripMux(t)
	defer close(m.done)
	m.mu.Lock()
	got := m.childSizeLocked()
	m.mu.Unlock()
	if got.Rows != 23 {
		t.Fatalf("child rows = %d; the pane is 24 and the strip owns one", got.Rows)
	}
}

// The strip reflects the ACTIVE tab, which is the one thing it exists to say.
func TestSwitchingTabsChangesWhichTabIsMarked(t *testing.T) {
	m, rec := stripMux(t)
	defer close(m.done)

	m.paintStrip()
	m.drainForTest()
	if !strings.Contains(rec.String(), "[two]") {
		t.Fatalf("initial strip does not mark the active tab: %q", rec.String())
	}

	m.previousTab()
	m.drainForTest()
	tail := rec.String()
	last := strings.LastIndex(tail, "[one]")
	prev := strings.LastIndex(tail, "[two]")
	if last < 0 {
		t.Fatalf("after switching, the new active tab is not marked: %q", tail)
	}
	if last < prev {
		t.Fatalf("the strip still marks the old tab most recently: %q", tail)
	}
}

// M3.6: the DERIVED consumer set still classifies the pane once the title is
// degraded to the active tab's name.
//
// Asserted against the real consumers rather than a remembered list (finding 9),
// including the TerminalCommand == "" case zellijpane.paneFrom admits, where the
// command fallback is unavailable and the title is all there is.
func TestTheDegradedTitleStillClassifiesThePane(t *testing.T) {
	mux := &terminalMux{
		tabs: []*terminalTab{
			{id: 1, name: "terminal 1"},
			{id: 2, name: "work"},
		},
		active: 1,
	}
	title := mux.paneTitleLocked()

	t.Run("with the command present, the fallback carries it", func(t *testing.T) {
		got := workbenchshortcut.RoleForPane(zellijpane.Pane{
			Title: title, TerminalCommand: "/usr/local/bin/pair term",
		})
		if got != workbenchshortcut.PaneRoleRightTerminal {
			t.Fatalf("RoleForPane = %v, want RightTerminal", got)
		}
	})

	// THE CASE THAT FOUND A REAL DEFECT. With no command the title is the only
	// signal, and RoleForPane's classification routes the operator's global
	// shortcuts -- so a title that stops matching costs that pane its
	// keybindings, silently.
	//
	// A first version of M3.6 degraded to the bare tab name and this failed:
	// `work` matches neither arm, while the packed `terminal 1 [work]` DID match
	// `HasPrefix "terminal "` because it began with the first tab's default
	// name. That also disproved a claim in the plan's finding 9 (that the packed
	// form never matched shortcut.go's arm), now corrected. Hence the prefix.
	t.Run("with no command, the degraded title still classifies", func(t *testing.T) {
		if got := workbenchshortcut.RoleForPane(zellijpane.Pane{Title: title}); got != workbenchshortcut.PaneRoleRightTerminal {
			t.Fatalf("RoleForPane(%q) = %v with no command; a renamed tab would "+
				"silently cost this pane its global shortcuts", title, got)
		}
	})

	// The default tab name IS the classifier-friendly one, which is why the
	// common case keeps working.
	t.Run("a default tab name still matches by title alone", func(t *testing.T) {
		got := workbenchshortcut.RoleForPane(zellijpane.Pane{Title: "terminal 1"})
		if got != workbenchshortcut.PaneRoleRightTerminal {
			t.Fatalf("RoleForPane(%q) = %v, want RightTerminal", "terminal 1", got)
		}
	})
}

// --- (f): never write inside the child's cursor save ------------------------

// THE BUG THIS CLOSES. The cursor save slot is shared, one per terminal. A
// paint that saves and restores inside the child's DECSC..DECRC pair leaves the
// slot holding OUR position, and the child's restore lands there -- the
// operator's `l` finishing with the cursor in the tab strip, and zsh's
// right-prompt drawn on the strip's row (zsh uses terminfo sc/rc, which ARE
// these bytes).
//
// Option (a), moving our paint to CSI s/u, is dead: probes/cursorsaveslots
// measured that there is no usable second slot. So the fix is not to write at
// all while the child holds one.
func TestNoPaintLandsInsideTheChildsCursorSave(t *testing.T) {
	m, rec := stripMux(t)
	defer close(m.done)

	// The child saves the cursor and keeps writing -- zsh drawing a prompt.
	m.output <- ptyChunk{id: 2, data: []byte("prompt\x1b7")}
	m.drainForTest()

	m.paintStrip()
	m.drainForTest()
	if strings.Contains(rec.String(), "[two]") {
		t.Fatalf("a paint landed inside the child's cursor save; its restore will "+
			"now recover OUR position: %q", rec.String())
	}

	// The child restores. The debt is paid on that very chunk.
	m.output <- ptyChunk{id: 2, data: []byte("\x1b8rest")}
	m.drainForTest()
	if !strings.Contains(rec.String(), "[two]") {
		t.Fatalf("the paint was dropped rather than deferred: %q", rec.String())
	}
}

// A row-dirty batch records a DEBT and does not paint immediately -- couch's
// pattern (couchtty/console.go:1147). It matters far more here than it did
// there: a shell emits erases on every prompt redraw, so painting per row-dirty
// batch means painting constantly, and constantly at the moment the child is
// mid-prompt with a save outstanding.
func TestARowDirtyBatchDefersWhileTheChildHoldsASave(t *testing.T) {
	m, rec := stripMux(t)
	defer close(m.done)

	// Row-dirty AND a held save in one batch: the row is owed, but paying it
	// now is exactly the collision.
	m.output <- ptyChunk{id: 2, data: []byte("\x1b[2J\x1b7"), rowDirty: true}
	m.drainForTest()
	if strings.Contains(rec.String(), "[two]") {
		t.Fatalf("row-dirty painted while the child held a save: %q", rec.String())
	}
	if !m.stripOwedForTest() {
		t.Fatal("the row-dirty debt was dropped rather than recorded")
	}

	m.output <- ptyChunk{id: 2, data: []byte("\x1b8")}
	m.drainForTest()
	if !strings.Contains(rec.String(), "[two]") {
		t.Fatalf("the owed repaint never landed after the child restored: %q", rec.String())
	}
	if m.stripOwedForTest() {
		t.Fatal("the debt survived being paid")
	}
}

// Stale beats wrong. A child holding a save indefinitely must leave the row
// UNCHANGED, never repainted at the cost of the child's cursor.
func TestAHeldSaveLeavesTheRowStaleRatherThanCorruptingTheChild(t *testing.T) {
	m, rec := stripMux(t)
	defer close(m.done)

	m.output <- ptyChunk{id: 2, data: []byte("\x1b7")}
	m.drainForTest()
	for i := 0; i < 5; i++ {
		m.paintStrip()
		m.output <- ptyChunk{id: 2, data: []byte("more output")}
		m.drainForTest()
	}
	if strings.Contains(rec.String(), "[two]") {
		t.Fatalf("a paint escaped while the child held a save: %q", rec.String())
	}
}
