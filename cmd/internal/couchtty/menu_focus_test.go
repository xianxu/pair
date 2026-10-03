package couchtty

import (
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

func focusSpace(s MenuState) MenuState {
	s, _ = reduceKey(s, PanelKey{Kind: KeyRune, Rune: ' '})
	return s
}

func TestMenuFocusMembership(t *testing.T) {
	for _, tc := range []struct {
		name, description, published string
		live, want                   bool
	}{
		// The operator description is stored, not displayed (pair#363).
		{"operator", " work ", "", true, false},
		{"published", "", "progress", true, true},
		{"precedence", "work", " \x1b[2J\n\x0e\u0085 ", true, false},
		{"controls", "\x1b[31m\x1b[0m\r\n\x7f", "", true, false},
		{"whitespace", " \t\u2003 ", "", true, false},
		{"dead", "", "work", false, false},
		{"empty", "", "", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := menuThreads()[0]
			row.Description, row.PublishedSummary = tc.description, tc.published
			if !tc.live {
				row.State = couchcore.ThreadParked
			}
			state := focusSpace(NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address))
			if state.CurrentFrame().Filter != "" {
				t.Fatalf("space became filter %q", state.CurrentFrame().Filter)
			}
			if got := len(VisibleMenuThreads(state)) == 1; got != tc.want {
				t.Fatalf("membership = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMenuFocusOrderSelectionAndRouting(t *testing.T) {
	rows := menuThreads()
	rows[1].State = couchcore.ThreadLive
	rows[1].PublishedSummary = "review task"
	third := rows[0]
	third.Address = menuAddress("couch-three")
	third.Name = "third"
	third.PublishedSummary = "third task"
	rows = append(rows, third)
	state := NewMenuState(rows, rows[0].Address)
	normal := VisibleMenuThreads(state)
	var want []couchcore.ThreadAddress
	for _, r := range normal {
		if r.PublishedSummary != "" {
			want = append(want, r.Address)
		}
	}
	state = focusSpace(state)
	var got []couchcore.ThreadAddress
	for _, r := range VisibleMenuThreads(state) {
		got = append(got, r.Address)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("focus order = %v, want %v", got, want)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	selected := state.CurrentFrame().SelectedAddress
	state = focusSpace(focusSpace(state))
	if state.CurrentFrame().SelectedAddress != selected {
		t.Fatal("toggle lost visible identity")
	}
	_, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 1 || effects[0].Operation != "switch" || effects[0].Args["tag"] != string(selected.Tag) {
		t.Fatalf("effects = %+v", effects)
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state = focusSpace(state)
	if state.CurrentFrame().Kind != MenuFrameActions || state.CurrentFrame().Filter != " " {
		t.Fatalf("child space = %+v", state.CurrentFrame())
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEscape})
	if len(VisibleMenuThreads(state)) != 2 {
		t.Fatal("child return lost focus view")
	}
}

func TestMenuFocusNonemptyFilterKeepsSpaceLiteralInBothViews(t *testing.T) {
	for name, view := range map[string]MenuRootView{"normal": MenuViewNormal, "focus": MenuViewFocus} {
		t.Run(name, func(t *testing.T) {
			state := NewMenuState(menuThreads(), menuAddress("couch-one"))
			if view == MenuViewFocus {
				state = focusSpace(state)
			}
			state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: 'r'})
			next, effects := reduceKey(state, PanelKey{Kind: KeyRune, Rune: ' '})
			if frame := next.CurrentFrame(); frame.Filter != "r " || frame.View != view || len(effects) != 0 {
				t.Fatalf("literal space: frame=%+v effects=%+v", frame, effects)
			}
		})
	}
}

func TestMenuFocusRefreshAndOverlay(t *testing.T) {
	rows := menuThreads()
	rows[0].PublishedSummary = "first"
	rows[1].PublishedSummary = "second"
	rows[1].State = couchcore.ThreadDetached
	state := NewMenuState(rows, rows[0].Address)
	state.Reattach.Attached = map[couchcore.ThreadAddress]uint64{rows[1].Address: 0}
	state = focusSpace(state)
	if len(VisibleMenuThreads(state)) != 2 {
		t.Fatal("focus ignored attached overlay")
	}
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	rows[1].State = couchcore.ThreadLive
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: rows})
	if state.CurrentFrame().SelectedAddress != rows[1].Address {
		t.Fatal("refresh lost still visible selection")
	}
	rows[1].PublishedSummary = ""
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: rows})
	if got := VisibleMenuThreads(state); len(got) != 1 || state.CurrentFrame().SelectedAddress != rows[0].Address {
		t.Fatalf("refresh membership/selection = %+v %+v", got, state.CurrentFrame())
	}
	rows[0].State = couchcore.ThreadParked
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: rows})
	_, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 {
		t.Fatalf("hidden row routed: %+v", effects)
	}
}

func TestMenuFocusRenderAndExtents(t *testing.T) {
	rows := menuThreads()
	rows[0].Description = "operator fallback"
	rows[0].PublishedSummary = "\x1b[2J  最新\n任务\u0085 " + strings.Repeat("界", 100)
	state := focusSpace(NewMenuState(rows, rows[0].Address))
	state.Attention = map[couchcore.ThreadAddress][]AttentionMessage{rows[0].Address: {{Text: "attention"}}}
	for _, color := range []bool{false, true} {
		rendered := RenderMenuView(state, 40, 12, time.Time{}, color)
		plain := string(ansi.Strip([]byte(rendered.Body)))
		if !strings.Contains(plain, "threads › focus") || !strings.Contains(plain, "compiler ◆ 最新任务") || strings.Contains(plain, "/repo/") || strings.Contains(plain, "fallback") || strings.Contains(rendered.Body, "\x1b[2J") {
			t.Fatalf("focus render = %q", rendered.Body)
		}
		for _, line := range strings.Split(plain, "\r\n") {
			if !utf8.ValidString(line) || textwidth.Width(line) > 40 || strings.ContainsAny(line, "\u0085\x0e") {
				t.Fatalf("unsafe/unbounded line %q", line)
			}
		}
		if len(rendered.Extents) != 1 || rendered.Extents[0].End-rendered.Extents[0].Start != 2 {
			t.Fatalf("extents = %+v", rendered.Extents)
		}
		for y := rendered.Extents[0].Start; y < rendered.Extents[0].End; y++ {
			if a, ok := rendered.PointToActor(y, 1); !ok || a != rows[0].Address {
				t.Fatal("wrong hit target")
			}
		}
	}
}

func TestMenuFocusEmptyAndNoMatch(t *testing.T) {
	state := focusSpace(NewMenuState(menuThreads(), menuAddress("couch-one")))
	if got := RenderMenu(state, 100, 12, time.Time{}, false); !strings.Contains(got, "no tagged live threads") {
		t.Fatalf("missing placeholder: %q", got)
	}
	rows := menuThreads()
	rows[0].PublishedSummary = "task"
	state = focusSpace(NewMenuState(rows, rows[0].Address))
	state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: 'z'})
	if got := RenderMenu(state, 100, 12, time.Time{}, false); !strings.Contains(got, "no match") || strings.Contains(got, "no tagged live threads") {
		t.Fatalf("wrong no match: %q", got)
	}
}

func TestMenuFocusPreservesNormalExactSearchAndLabels(t *testing.T) {
	rows := menuThreads()
	rows[0].Name = "target"
	rows[0].Address = menuAddress("target") // Exact tag but untagged: focus must not expose fuzzy matches.
	rows[1].Name = "target longer"
	rows[1].State = couchcore.ThreadLive
	rows[1].PublishedSummary = "task"
	state := focusSpace(NewMenuState(rows, rows[0].Address))
	for _, r := range "target" {
		state, _ = reduceKey(state, PanelKey{Kind: KeyRune, Rune: r})
	}
	if got := VisibleMenuThreads(state); len(got) != 0 {
		t.Fatalf("focus changed exact search to fuzzy: %+v", got)
	}
	rows[1].Name = "target"
	state = focusSpace(NewMenuState(rows, rows[0].Address))
	var label string
	for _, entry := range PresentThreads(rows, nil) {
		if entry.Row.Address == rows[1].Address {
			label = entry.Label
		}
	}
	if label == "target" {
		t.Fatal("fixture must require disambiguation")
	}
	if got := string(ansi.Strip([]byte(RenderMenu(state, 120, 12, time.Time{}, false)))); !strings.Contains(got, label+" ◆ task") {
		t.Fatalf("focus changed full inventory label: %q", got)
	}
}

// pair#372: the focus row is `name ◆ description ◆ slug` on one line.
func TestMenuFocusRendersSlugInline(t *testing.T) {
	render := func(row couchcore.ActionableThreadSummary, width int) string {
		state := focusSpace(NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address))
		return string(ansi.Strip([]byte(RenderMenu(state, width, 12, time.Time{}, false))))
	}
	row := menuThreads()[0]
	row.PublishedSummary = "task"
	row.Slug = "main-slot3 | couch \x1b[2Jswitcher\n click-select"
	if got := render(row, 100); !strings.Contains(got, "compiler ◆ task ◆ main-slot3 | couch switcher click-select") || strings.Contains(got, "\x1b[2J") {
		t.Fatalf("slug render = %q", got)
	}
	row.Slug = " \t "
	if got := render(row, 100); !strings.Contains(got, "compiler ◆ task") || strings.Contains(got, "task ◆") {
		t.Fatalf("blank slug left a trailing diamond: %q", got)
	}
	row.Slug = strings.Repeat("界", 100)
	for _, line := range strings.Split(render(row, 40), "\r\n") {
		if textwidth.Width(line) > 40 {
			t.Fatalf("long slug wrapped or overflowed: %q", line)
		}
	}
	row.PublishedSummary, row.Slug = "", "main | busy"
	state := focusSpace(NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address))
	if len(VisibleMenuThreads(state)) != 0 {
		t.Fatal("a slug without a description joined the focus view")
	}
}

// A slug change is display only: it is not a notification.
func TestMenuFocusSlugChangeIsNotAttention(t *testing.T) {
	rows := menuThreads()
	rows[0].PublishedSummary, rows[0].Slug = "task", "main | one"
	state := focusSpace(NewMenuState(rows, rows[0].Address))
	rows[0].Slug = "main | two"
	state, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: rows})
	if len(effects) != 0 || len(state.Attention) != 0 || state.Notice.Text != "" {
		t.Fatalf("slug change produced effects=%+v attention=%+v notice=%+v", effects, state.Attention, state.Notice)
	}
	if got := RenderMenu(state, 100, 12, time.Time{}, false); !strings.Contains(got, "main | two") {
		t.Fatalf("refreshed slug not rendered: %q", got)
	}
}
