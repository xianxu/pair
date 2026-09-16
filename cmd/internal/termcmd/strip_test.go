package termcmd

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/textwidth"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

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

func TestDegenerateWidthsDrawNothingRatherThanWrapping(t *testing.T) {
	for _, width := range []int{0, -1} {
		r := RenderStrip(width, StripModel{Tabs: []TabChip{{Name: "a"}, {Name: "b"}}})
		if r.Body != "" || len(r.Spans) != 0 {
			t.Fatalf("width %d drew %q with %d spans", width, r.Body, len(r.Spans))
		}
	}
}

func TestNoTabsRendersNothing(t *testing.T) {
	if r := RenderStrip(40, StripModel{}); r.Body != "" {
		t.Fatalf("an empty model drew %q", r.Body)
	}
}

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

func TestEveryPaneTitleProducerSatisfiesEveryConsumer(t *testing.T) {
	// The producer axis is the PREDICATE'S BOUNDARY, not three hand-picked
	// names. Three fixtures could not see BR-56 -- `paneTitleLocked` restated
	// `RoleForPane`'s predicate as `HasPrefix(name, "terminal")`, which agrees
	// with it on `work` and `terminal 1` and disagrees on every name that starts
	// with `terminal` and then CONTINUES. So the corpus is exactly those cases:
	// names on either side of the boundary, plus the ones that straddle it.
	names := []string{
		"terminal 1",   // the default: already classifies
		"work",         // plainly does not: must be prefixed
		"terminals",    // starts with "terminal", does NOT classify -- BR-56
		"terminal-2",   // ditto, punctuation instead of a space
		"terminalwork", // ditto, no separator at all
		"terminal",     // the bare word: classifies on the equality arm
		"Terminal 1",   // case: RoleForPane folds, so the producer must not care
		"日本語",          // wide, non-ASCII, nowhere near the boundary
		"a b c",        // spaces, but not the prefix
	}
	producers := make([]struct {
		name  string
		title func(t *testing.T) string
	}, 0, len(names)*2)
	for _, n := range names {
		n := n
		producers = append(producers,
			struct {
				name  string
				title func(t *testing.T) string
			}{"active tab named " + n, func(t *testing.T) string {
				return titleOf(t, []*terminalTab{{id: 1, name: "terminal 1"}, {id: 2, name: n}}, 1, false)
			}},
			struct {
				name  string
				title func(t *testing.T) string
			}{"rename open over " + n, func(t *testing.T) string {
				return titleOf(t, []*terminalTab{{id: 1, name: "terminal 1"}, {id: 2, name: n}}, 1, true)
			}},
		)
	}
	// The consumers, from the derivation the plan records:
	//   grep -rn "\.Title" cmd --include=*.go | grep -v _test.go
	consumers := []struct {
		name string
		ok   func(title string) bool
	}{
		{"workbenchshortcut.RoleForPane (no command)", func(title string) bool {
			return workbenchshortcut.RoleForPane(zellijpane.Pane{Title: title}) == workbenchshortcut.PaneRoleRightTerminal
		}},
		{"launcher.ClassifyLiveLayout (no command)", func(title string) bool {
			mode, ok := launcher.ClassifyLiveLayout([]zellijpane.Pane{
				{Title: "claude", TerminalCommand: "pair wrap claude"},
				{Title: "draft"},
				{Title: title},
			})
			return ok && mode == launcher.Layout3
		}},
	}

	for _, p := range producers {
		title := p.title(t)
		for _, c := range consumers {
			if !c.ok(title) {
				t.Errorf("producer %q emits %q, which %s does not classify as the right terminal",
					p.name, title, c.name)
			}
		}
	}
}

func titleOf(t *testing.T, tabs []*terminalTab, active int, renaming bool) string {
	t.Helper()
	mux := &terminalMux{rt: &fakeRuntime{}, done: make(chan struct{}), tabs: tabs, active: active}
	if renaming {
		if _, _, err := mux.beginRename(); err != nil {
			t.Fatal(err)
		}
	}
	mux.mu.Lock()
	defer mux.mu.Unlock()
	return mux.paneTitleLocked()
}

func TestTheRenameFieldIsDrawnOnTheStrip(t *testing.T) {
	m := StripModel{
		Tabs:   []TabChip{{Name: "one"}, {Name: "two"}},
		Active: 1,
		Rename: &RenameField{Tab: 1, Text: "bu│ilt"},
	}
	got := RenderStrip(40, m).Body
	if want := "one [rename: bu│ilt]"; got != want {
		t.Fatalf("RenderStrip = %q, want %q", got, want)
	}
}

func TestARenameFieldCannotInjectEscapes(t *testing.T) {
	m := StripModel{
		Tabs:   []TabChip{{Name: "one"}, {Name: "two"}},
		Active: 1,
		Rename: &RenameField{Tab: 1, Text: "a\x1b[31mred"},
	}
	if got := RenderStrip(40, m).Body; strings.Contains(got, "\x1b") {
		t.Fatalf("a rename field reached the row as an escape sequence: %q", got)
	}
}

func TestANarrowPaneKeepsTheRenameFieldVisible(t *testing.T) {
	m := StripModel{
		Tabs:   []TabChip{{Name: "aaaaaaaaaa"}, {Name: "bbbbbbbbbb"}, {Name: "cccccccccc"}},
		Active: 0,
		Rename: &RenameField{Tab: 2, Text: "zz│"},
	}
	if got := RenderStrip(18, m).Body; !strings.Contains(got, "zz│") {
		t.Fatalf("the rename field was dropped by the width budget: %q", got)
	}
}

func TestARenameWhoseTabExitedStaysOnTheRow(t *testing.T) {
	m := StripModel{
		Tabs:   []TabChip{{Name: "aaaa"}, {Name: "bbbb"}},
		Active: 1,
		Rename: &RenameField{Tab: 7, Text: "zz│"},
	}
	if got := RenderStrip(40, m).Body; got != "aaaa [bbbb] [rename: zz│]" {
		t.Fatalf("RenderStrip = %q, want the detached field after the tabs", got)
	}
	// The spans still describe TABS only: a detached field belongs to no tab, so
	// a click there must not land on one (#200 reads these).
	if spans := RenderStrip(40, m).Spans; len(spans) != 2 {
		t.Fatalf("spans = %#v, want one per tab and none for the detached field", spans)
	}
	// Narrow: the field the operator is typing into is what must survive.
	if got := RenderStrip(14, m).Body; !strings.Contains(got, "zz│") {
		t.Fatalf("RenderStrip(14) = %q, dropped the field being typed into", got)
	}
}
