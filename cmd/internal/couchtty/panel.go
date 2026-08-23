package couchtty

import (
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// PanelRow is one line of couch's own screen.
type PanelRow struct {
	Tree  couchcore.Worktree
	Label string
	Desc  string
	Live  bool
	// Bell is the point of the panel being a place to LOOK: an actor that
	// wants attention says so here, not only on the status row where it
	// competes for one line.
	Bell bool
}

// PanelModel is the panel as DATA: what to show, filtered, in a stable order.
//
// Pure. The console renders it and #148's advisor can read the same rows, which
// is the "no state the operator can see that an LLM cannot" property stated in
// the project.
type PanelModel struct {
	all []PanelRow

	// shown is the last filtered result, and it is what Pick indexes.
	// Numbered selection has to mean "the Nth thing on screen"; indexing the
	// underlying set instead is how an operator types 2 and lands somewhere
	// else.
	shown []PanelRow

	// cursor is the highlighted row, 0-based into shown. A list with no
	// highlight is a list you cannot navigate -- the operator has no way to
	// tell what Enter will do.
	cursor int
}

// Cursor is the highlighted row index.
func (m *PanelModel) Cursor() int { return m.cursor }

// Move steps the highlight, clamping rather than wrapping. Wrapping in a short
// list makes "press down twice" unpredictable.
func (m *PanelModel) Move(delta int) {
	if len(m.shown) == 0 {
		m.cursor = 0
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.shown) {
		m.cursor = len(m.shown) - 1
	}
}

// Selected is the highlighted row.
func (m *PanelModel) Selected() (PanelRow, bool) {
	if m.cursor < 0 || m.cursor >= len(m.shown) {
		return PanelRow{}, false
	}
	return m.shown[m.cursor], true
}

// NewPanelModel builds the rows from couch's own summaries, so a tree that is
// PARKED -- named, no live actor -- is listed exactly as `couch list` lists it.
// That thread is the one this project exists to stop losing, so it is not
// filtered out for being idle.
func NewPanelModel(trees []couchcore.TreeSummary) *PanelModel {
	m := &PanelModel{all: make([]PanelRow, 0, len(trees))}
	for _, t := range trees {
		label := t.Name
		if label == "" {
			// An unnamed tree still has to be identifiable; an empty chip is
			// unusable. Same fallback `couch list` renders.
			label = t.Tree.Repo()
		}
		m.all = append(m.all, PanelRow{
			Tree:  t.Tree,
			Label: label,
			Desc:  t.Desc,
			Live:  t.Live(),
		})
	}
	m.shown = m.all
	return m
}

// Rows is everything the panel knows about, unfiltered.
func (m *PanelModel) Rows() []PanelRow { return m.all }

// Shown is the current filtered view -- what the operator is looking at.
func (m *PanelModel) Shown() []PanelRow { return m.shown }

// Filter narrows the rows by INJECTING the match rule rather than restating it.
//
// resolve is `couch.LookupTrees` in production: one rule serving the CLI, the
// panel, and #148's advisor. Restating it here is the drift Decision 12 exists
// to prevent -- and the earlier plan text got the rule's own field list wrong,
// which is what a second copy does.
//
// An empty query is not a resolution: it means "show everything", and asking
// the resolver would make the panel's DEFAULT view depend on a match rule.
func (m *PanelModel) Filter(query string, resolve func(string) []couchcore.Worktree) []PanelRow {
	if query == "" || resolve == nil {
		m.shown = m.all
		m.clampCursor()
		return m.shown
	}
	want := map[string]bool{}
	for _, w := range resolve(query) {
		want[w.Key()] = true
	}
	// Filtered in the ORIGINAL order rather than the resolver's: numbered
	// selection is only safe if rows do not move under the operator's fingers,
	// and a resolver is free to return whatever order it likes.
	out := make([]PanelRow, 0, len(want))
	for _, r := range m.all {
		if want[r.Tree.Key()] {
			out = append(out, r)
		}
	}
	m.shown = out
	m.clampCursor()
	return out
}

// clampCursor keeps the highlight on a row that exists: filtering can shrink
// the list under it, and a cursor past the end selects nothing.
func (m *PanelModel) clampCursor() {
	if m.cursor >= len(m.shown) {
		m.cursor = len(m.shown) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// Pick resolves a 1-based keystroke to a row the operator can currently SEE.
func (m *PanelModel) Pick(n int) (PanelRow, bool) {
	if n < 1 || n > len(m.shown) {
		return PanelRow{}, false
	}
	return m.shown[n-1], true
}

// RenderPanel draws the panel for the operator.
//
// Deliberately plain -- a list to read, not chrome. But it MUST show three
// things or it is not usable: which row is selected, which actors want
// attention, and what the keys are. The first cut showed a bare list and the
// operator had no way to tell that arrows, Enter or Escape did anything.
func RenderPanel(rows []PanelRow, cursor int) string {
	var b strings.Builder
	b.WriteString("couch — actors\r\n\r\n")
	if len(rows) == 0 {
		b.WriteString("  (no match)\r\n")
		return b.String()
	}
	for i, r := range rows {
		marker := "  "
		if i == cursor {
			marker = "▸ "
		}
		state := " "
		if !r.Live {
			// A parked thread stays listed: it is exactly the one an operator
			// loses track of.
			state = "·"
		}
		bell := " "
		if r.Bell {
			bell = "*"
		}
		fmt.Fprintf(&b, "%s%d%s%s %s", marker, i+1, state, bell, sanitize(r.Label))
		if r.Desc != "" {
			fmt.Fprintf(&b, "  — %s", sanitize(r.Desc))
		}
		b.WriteString("\r\n")
	}
	return b.String()
}

// RenderPanelWithQuery draws the panel plus the typeahead buffer and the keys,
// so the operator can see why the list narrowed and what to press.
func RenderPanelWithQuery(query string, rows []PanelRow, cursor int) string {
	var b strings.Builder
	b.WriteString(RenderPanel(rows, cursor))
	b.WriteString("\r\n")
	if query != "" {
		fmt.Fprintf(&b, "  filter: %s\r\n", sanitize(query))
	}
	b.WriteString("  ↑↓ select · 1-9 jump · enter switch · s start · x stop · esc back\r\n")
	return b.String()
}

// PanelActions is what the operator can do from the panel.
//
// Names only, and every one must be a name in couchcore.Operations(): the panel
// DISPATCHES through that table rather than implementing anything, so there is
// no operator action the advisor cannot also perform (#148's design test).
//
// It lists what is WIRED, not what is planned. The first version returned four
// names with nothing behind them and the audit passed anyway -- a subset check
// is satisfied by a list that does nothing, which is why the audit now also
// requires each name to be reachable from a keystroke.
func PanelActions() []string {
	return []string{"start", "stop", "name", "describe"}
}

// PanelActionKeys maps each action to the key that invokes it, so the audit can
// check the action is reachable rather than merely declared.
func PanelActionKeys() map[string]byte {
	return map[string]byte{
		"start":    's',
		"stop":     'x',
		"name":     'n',
		"describe": 'd',
	}
}
