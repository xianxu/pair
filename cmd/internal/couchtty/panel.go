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
	return out
}

// Pick resolves a 1-based keystroke to a row the operator can currently SEE.
func (m *PanelModel) Pick(n int) (PanelRow, bool) {
	if n < 1 || n > len(m.shown) {
		return PanelRow{}, false
	}
	return m.shown[n-1], true
}

// RenderPanel draws the panel's rows for the operator.
//
// Pure, and deliberately plain: this is a list to read, not a UI. The Spec's
// "richer navigation lives inside couch's TUI with typeahead" is about having a
// screen to look at, not about chrome.
func RenderPanel(rows []PanelRow) string {
	var b strings.Builder
	b.WriteString("couch — actors\r\n\r\n")
	if len(rows) == 0 {
		b.WriteString("  (nothing running)\r\n")
		return b.String()
	}
	for i, r := range rows {
		state := " "
		if !r.Live {
			// A parked thread stays listed: it is exactly the one an operator
			// loses track of.
			state = "·"
		}
		fmt.Fprintf(&b, "  %d%s %s", i+1, state, r.Label)
		if r.Desc != "" {
			fmt.Fprintf(&b, "  — %s", sanitize(r.Desc))
		}
		b.WriteString("\r\n")
	}
	return b.String()
}

// PanelActions is what the operator can do from the panel.
//
// Names only, and every one must be a name in couchcore.Operations(): the panel
// dispatches through that table rather than implementing anything, so there is
// no operator action the advisor cannot also perform (#148's design test). A
// private verb here would be exactly the drift the ops table exists to stop,
// which is why an audit asserts this set is a subset of the declared one.
func PanelActions() []string {
	return []string{"start", "stop", "name", "describe"}
}

// RenderPanelWithQuery draws the panel with the typeahead buffer visible, so
// the operator can see why the list narrowed.
func RenderPanelWithQuery(query string, rows []PanelRow) string {
	body := RenderPanel(rows)
	if query == "" {
		return body
	}
	return body + "\r\n  /" + sanitize(query) + "\r\n"
}
